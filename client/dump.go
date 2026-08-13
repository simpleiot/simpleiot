package client

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/simpleiot/simpleiot/data"
)

// DumpOptions controls what a dump reports.
type DumpOptions struct {
	// NodeID limits the tree to one subtree. Empty means the instance root.
	NodeID string
	// Points includes every point of every node, with its origin and time.
	Points bool
	// Streams includes the JetStream boundary-origin stream inventory.
	Streams bool
}

// DumpInstance describes an instance as it actually is, for troubleshooting. Where
// [ExportNodes] writes the configuration a file would need to recreate an
// instance, a dump writes the identifiers and structure that explain how one
// is behaving: node and edge IDs, every parent of every node, tombstones,
// point origins and timestamps, and the replication streams. It is meant to be
// read rather than applied, so it deliberately shares no format with export.
func DumpInstance(nc *nats.Conn, o DumpOptions) (string, error) {
	var b strings.Builder

	root, err := GetRootNode(nc)
	if err != nil {
		return "", fmt.Errorf("error getting root node: %w", err)
	}

	desc, _ := root.Points.Text(data.PointTypeDescription, "")
	fmt.Fprintf(&b, "instance\n  root %v  %v  %q\n", root.ID, root.Type, desc)

	if o.Streams {
		if err := dumpStreams(&b, nc, root.ID); err != nil {
			fmt.Fprintf(&b, "  error: %v\n", err)
		}
	}

	start := o.NodeID
	if start == "" || start == "root" {
		start = root.ID
	}

	// strays collects nodes carrying the virtual "root" parent that are not
	// this instance's root. Each one is a second root, which makes the
	// instance serve a tree that is not its own.
	strays := map[string]bool{}

	fmt.Fprintf(&b, "\ntree from %v\n", start)
	visited := map[string]bool{start: true}
	if err := dumpNode(&b, nc, start, "  ", visited, strays, root.ID, o); err != nil {
		return "", err
	}

	if len(strays) > 0 {
		fmt.Fprintf(&b, "\nanomalies\n")
		ids := make([]string, 0, len(strays))
		for id := range strays {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			fmt.Fprintf(&b, "  %v has the virtual \"root\" parent but is not this "+
				"instance's root (%v)\n", id, root.ID)
		}
	}

	return b.String(), nil
}

// dumpStreams lists the boundary-origin streams backing this instance, which
// is the quickest way to see which instances it replicates with.
func dumpStreams(b *strings.Builder, nc *nats.Conn, rootID string) error {
	js, err := jetstream.New(nc)
	if err != nil {
		return fmt.Errorf("error creating JetStream context: %w", err)
	}

	ctx := context.Background()
	var lines []string
	lister := js.ListStreams(ctx, jetstream.WithStreamListSubject("inst.>"))
	for si := range lister.Info() {
		// inst_<boundary>_<origin>; node IDs are UUIDs, so "_" only
		// separates the two fields
		tok := strings.Split(si.Config.Name, "_")
		if len(tok) != 3 || tok[0] != "inst" {
			continue
		}
		boundary, origin := tok[1], tok[2]

		role := "replica"
		switch {
		case boundary == rootID && origin == rootID:
			role = "own"
		case origin == rootID:
			role = "written by this instance"
		}

		lines = append(lines, fmt.Sprintf(
			"  boundary %v  origin %v  %v msgs  (%v)",
			boundary, origin, si.State.Msgs, role))
	}
	if err := lister.Err(); err != nil {
		return fmt.Errorf("error listing streams: %w", err)
	}

	sort.Strings(lines)
	fmt.Fprintf(b, "\nstreams\n")
	for _, l := range lines {
		fmt.Fprintln(b, l)
	}

	return nil
}

// dumpNode writes one node and recurses into its children. Deleted nodes are
// included, since a node that should be gone and is not explains as much as a
// missing one.
func dumpNode(b *strings.Builder, nc *nats.Conn, id, indent string,
	visited, strays map[string]bool, rootID string, o DumpOptions) error {

	children, err := GetNodes(nc, id, "all", "", true)
	if err != nil {
		return fmt.Errorf("error getting children of %v: %w", id, err)
	}

	for _, c := range children {
		desc, _ := c.Points.Text(data.PointTypeDescription, "")

		var notes []string
		if ts, _ := c.IsTombstone(); ts {
			notes = append(notes, "deleted")
		}

		// every parent of this node, so mirrors and stray root edges show
		parents, err := GetNodes(nc, "all", c.ID, "", true)
		if err != nil {
			return fmt.Errorf("error getting parents of %v: %w", c.ID, err)
		}
		var ups []string
		for _, p := range parents {
			if p.Parent == "root" && p.ID != rootID {
				strays[p.ID] = true
			}
			if p.Parent != id {
				ups = append(ups, p.Parent)
			}
		}
		if len(ups) > 0 {
			sort.Strings(ups)
			notes = append(notes, "also under "+strings.Join(ups, ", "))
		}

		note := ""
		if len(notes) > 0 {
			note = "  [" + strings.Join(notes, "; ") + "]"
		}

		fmt.Fprintf(b, "%v%v  %v  %q%v\n", indent, c.Type, c.ID, desc, note)

		if o.Points {
			dumpPoints(b, indent+"    ", c)
		}

		if visited[c.ID] {
			fmt.Fprintf(b, "%v    (already shown above)\n", indent)
			continue
		}
		visited[c.ID] = true

		if err := dumpNode(b, nc, c.ID, indent+"    ", visited, strays,
			rootID, o); err != nil {
			return err
		}
	}

	return nil
}

// dumpPoints writes a node's points and edge points with the metadata export
// leaves out: which instance or user wrote each one, and when.
func dumpPoints(b *strings.Builder, indent string, n data.NodeEdge) {
	write := func(kind string, pts data.Points) {
		sorted := make(data.Points, len(pts))
		copy(sorted, pts)
		sort.Slice(sorted, func(i, j int) bool {
			if sorted[i].Type != sorted[j].Type {
				return sorted[i].Type < sorted[j].Type
			}
			return sorted[i].Key < sorted[j].Key
		})

		for _, p := range sorted {
			val := fmt.Sprintf("%v", p.Val())
			if p.Txt() != "" {
				val = fmt.Sprintf("%q", p.Txt())
			}

			origin := p.Origin
			if origin == "" {
				origin = "-"
			}

			key := ""
			if p.Key != "" && p.Key != "0" {
				key = "[" + p.Key + "]"
			}

			fmt.Fprintf(b, "%v%v %v%v = %v  origin %v  %v\n", indent, kind,
				p.Type, key, val, origin,
				p.Time.UTC().Format("2006-01-02T15:04:05Z"))
		}
	}

	write("point", n.Points)
	write("edge ", n.EdgePoints)
}

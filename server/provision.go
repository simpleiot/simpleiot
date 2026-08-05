package server

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	yaml "github.com/goccy/go-yaml"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/client"
	"github.com/simpleiot/simpleiot/data"
)

// provisioner applies node files from a directory and from file nodes in the
// tree. Both are the same operation siot import performs; provisioning only
// decides when to run it and records what happened.
type provisioner struct {
	nc  *nats.Conn
	dir string
}

// provisionSource is one file to apply, from either the directory or the tree.
type provisionSource struct {
	// Name labels the source, appears in the Origin of every point it
	// applies, and orders the files on disk.
	Name string
	// Created orders file nodes, oldest first. It is zero for a file on disk.
	Created time.Time
	// NodeID is set when the source is a file node in the tree.
	NodeID   string
	Contents []byte
	Hash     string
}

// fromTree reports whether a source is a file node rather than a file on disk.
func (s provisionSource) fromTree() bool {
	return s.NodeID != ""
}

// origin is what a point applied from this source is stamped with.
func (s provisionSource) origin() string {
	return "provision:" + s.Name
}

// provisionState is what was recorded the last time a source was applied. For
// a file on disk it lives on a provisioningFile node; for a file node it lives
// on that node.
type provisionState struct {
	Hash    string
	Applied time.Time
	Error   string
}

// run applies every source whose contents have changed since the last pass.
func (p *provisioner) run() error {
	sources, err := p.sources()
	if err != nil {
		return err
	}

	states, err := p.states()
	if err != nil {
		return err
	}

	seen := map[string]bool{}

	for _, s := range sources {
		if seen[s.Name] {
			// two sources with one name: apply the first and say so, rather
			// than applying one of them and leaving the operator to work out
			// which
			err := fmt.Errorf("another provisioning file is already called %q, so this one was not applied", s.Name)
			log.Println("Provision:", err)

			if err := p.record(s, provisionState{Error: err.Error()}); err != nil {
				log.Println("Provision: error recording state:", err)
			}

			continue
		}

		seen[s.Name] = true

		prev := states[s.Name]

		// a file that applied cleanly is left alone until it changes. One that
		// failed is retried, since what it was waiting for -- a parent another
		// file creates, a description someone fixes -- may have arrived.
		if prev.Hash == s.Hash && prev.Error == "" {
			continue
		}

		state := provisionState{Hash: s.Hash, Applied: time.Now()}

		if err := p.apply(s); err != nil {
			state.Error = err.Error()

			// a file that keeps failing the same way says so once rather than
			// every time the rescan comes around
			if state.Error != prev.Error {
				log.Printf("Provision: %v: %v\n", s.Name, err)
			}
		} else {
			log.Printf("Provision: applied %v\n", s.Name)
		}

		if err := p.record(s, state); err != nil {
			log.Println("Provision: error recording state:", err)
		}
	}

	// a file that is gone takes its state node with it. File nodes need no
	// equivalent, since their state goes when they do.
	return p.pruneStateNodes(seen)
}

// apply parses one source and hands it to the engine siot import uses.
func (p *provisioner) apply(s provisionSource) error {
	var f data.NodeFile

	if err := yaml.Unmarshal(s.Contents, &f); err != nil {
		return fmt.Errorf("error parsing: %w", err)
	}

	plan, err := client.Apply(p.nc, f, client.ApplyOptions{Origin: s.origin()})
	if err != nil {
		return err
	}

	if len(plan.Errors) > 0 {
		msgs := make([]string, len(plan.Errors))
		for i, e := range plan.Errors {
			msgs[i] = e.Error()
		}

		return fmt.Errorf("%v", strings.Join(msgs, "; "))
	}

	return nil
}

// sources collects what to apply this pass: files in the directory in lexical
// order, then file nodes oldest first, which is the order an image's baseline
// then anything uploaded on top of it should apply in.
func (p *provisioner) sources() ([]provisionSource, error) {
	sources, err := p.dirSources()
	if err != nil {
		return nil, err
	}

	tree, err := p.treeSources()
	if err != nil {
		return nil, err
	}

	return append(sources, tree...), nil
}

// dirSources reads the provisioning directory, if there is one.
func (p *provisioner) dirSources() ([]provisionSource, error) {
	if p.dir == "" {
		return nil, nil
	}

	entries, err := os.ReadDir(p.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("error reading provisioning directory: %w", err)
	}

	var out []provisionSource

	for _, e := range entries {
		if e.IsDir() || !provisionFileName(e.Name()) {
			continue
		}

		contents, err := os.ReadFile(filepath.Join(p.dir, e.Name()))
		if err != nil {
			log.Printf("Provision: error reading %v: %v\n", e.Name(), err)
			continue
		}

		out = append(out, provisionSource{
			Name:     e.Name(),
			Contents: contents,
			Hash:     hashContents(contents),
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })

	return out, nil
}

// treeSources reads the file nodes under the provisioning node.
func (p *provisioner) treeSources() ([]provisionSource, error) {
	node, err := p.provisioningNode(false)
	if err != nil || node == "" {
		return nil, err
	}

	files, err := client.GetNodesType[client.File](p.nc, node, "all")
	if err != nil {
		return nil, fmt.Errorf("error getting provisioning files: %w", err)
	}

	var out []provisionSource

	for _, f := range files {
		contents, err := f.GetContents()
		if err != nil {
			log.Printf("Provision: error reading file node %v: %v\n", f.ID, err)
			continue
		}

		// a file node exists from the moment it is added, and its contents
		// arrive when someone uploads them. Until then there is nothing to
		// apply, and recording a pass against an empty file would tell an
		// operator it had been provisioned before they had chosen a file.
		if len(contents) == 0 {
			continue
		}

		name := f.Description
		if name == "" {
			name = f.Name
		}

		out = append(out, provisionSource{
			Name:     name,
			Created:  time.Unix(int64(f.Created), 0),
			NodeID:   f.ID,
			Contents: contents,
			Hash:     hashContents(contents),
		})
	}

	// oldest first, so uploads apply in the order they were added. A node with
	// no created point sorts last, by name, so the order stays predictable.
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]

		switch {
		case a.Created.IsZero() && b.Created.IsZero():
			return a.Name < b.Name
		case a.Created.IsZero():
			return false
		case b.Created.IsZero():
			return true
		case a.Created.Equal(b.Created):
			return a.Name < b.Name
		default:
			return a.Created.Before(b.Created)
		}
	})

	return out, nil
}

// provisionFileName reports whether a directory entry looks like a
// provisioning file, so that editor backups and the like are left alone.
func provisionFileName(name string) bool {
	if strings.HasPrefix(name, ".") {
		return false
	}

	switch filepath.Ext(name) {
	case ".yaml", ".yml":
		return true
	default:
		return false
	}
}

func hashContents(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// provisioningNode returns the ID of the provisioning node under the root,
// creating it when create is set.
func (p *provisioner) provisioningNode(create bool) (string, error) {
	root, err := client.GetRootNode(p.nc)
	if err != nil {
		return "", fmt.Errorf("error getting root node: %w", err)
	}

	nodes, err := client.GetNodes(p.nc, root.ID, "all", data.NodeTypeProvisioning, false)
	if err != nil {
		return "", fmt.Errorf("error getting provisioning node: %w", err)
	}

	if len(nodes) > 0 {
		return nodes[0].ID, nil
	}

	if !create {
		return "", nil
	}

	id := uuid.New().String()

	err = client.SendNode(p.nc, data.NodeEdge{
		ID:     id,
		Type:   data.NodeTypeProvisioning,
		Parent: root.ID,
		Points: data.Points{
			data.NewPointString(data.PointTypeDescription, "", "Provisioning"),
		},
	}, "provision")

	if err != nil {
		return "", fmt.Errorf("error creating provisioning node: %w", err)
	}

	return id, nil
}

// states reads what was recorded for each source last time, keyed by name.
func (p *provisioner) states() (map[string]provisionState, error) {
	out := map[string]provisionState{}

	node, err := p.provisioningNode(false)
	if err != nil {
		return nil, err
	}

	if node == "" {
		return out, nil
	}

	children, err := client.GetNodes(p.nc, node, "all", "", false)
	if err != nil {
		return nil, fmt.Errorf("error getting provisioning state: %w", err)
	}

	for _, c := range children {
		switch c.Type {
		case data.NodeTypeProvisioningFile:
			name, _ := c.Points.Text(data.PointTypeDescription, "")
			hash, _ := c.Points.Text(data.PointTypeHash, "")
			errText, _ := c.Points.Text(data.PointTypeError, "")

			applied := time.Time{}
			if p, ok := c.Points.Find(data.PointTypeHash, ""); ok {
				applied = p.Time
			}

			out[name] = provisionState{Hash: hash, Applied: applied, Error: errText}

		case data.NodeTypeFile:
			name, _ := c.Points.Text(data.PointTypeDescription, "")
			if name == "" {
				name, _ = c.Points.Text(data.PointTypeName, "")
			}

			hash, _ := c.Points.Text(data.PointTypeProvisionHash, "")
			errText, _ := c.Points.Text(data.PointTypeError, "")

			applied := time.Time{}
			if p, ok := c.Points.Find(data.PointTypeProvisionHash, ""); ok {
				applied = p.Time
			}

			out[name] = provisionState{Hash: hash, Applied: applied, Error: errText}
		}
	}

	return out, nil
}

// record writes what happened to a source. A file node keeps its state on
// itself, so that an operator sees one node with its status rather than a file
// and a status node beside it; a file on disk has nowhere else to keep it, so
// it gets a provisioningFile node.
func (p *provisioner) record(s provisionSource, state provisionState) error {
	node, err := p.provisioningNode(true)
	if err != nil {
		return err
	}

	points := data.Points{
		data.NewPointString(data.PointTypeError, "", state.Error),
	}

	if s.fromTree() {
		points = append(points, data.NewPointString(data.PointTypeProvisionHash, "", state.Hash))
		return client.SendNodePoints(p.nc, s.NodeID, points, false)
	}

	id, err := p.stateNode(node, s.Name)
	if err != nil {
		return err
	}

	points = append(points,
		data.NewPointString(data.PointTypeHash, "", state.Hash),
		data.NewPointString(data.PointTypeDescription, "", s.Name),
	)

	return client.SendNode(p.nc, data.NodeEdge{
		ID:     id,
		Type:   data.NodeTypeProvisioningFile,
		Parent: node,
		Points: points,
	}, "provision")
}

// stateNode finds the provisioningFile node for a path, or makes up an ID for
// a new one.
func (p *provisioner) stateNode(parent, name string) (string, error) {
	children, err := client.GetNodes(p.nc, parent, "all", data.NodeTypeProvisioningFile, false)
	if err != nil {
		return "", fmt.Errorf("error getting provisioning state: %w", err)
	}

	for _, c := range children {
		if desc, _ := c.Points.Text(data.PointTypeDescription, ""); desc == name {
			return c.ID, nil
		}
	}

	return uuid.New().String(), nil
}

// pruneStateNodes removes the state of files that are no longer there. The
// nodes those files provisioned stay, since provisioning does not own them.
func (p *provisioner) pruneStateNodes(seen map[string]bool) error {
	node, err := p.provisioningNode(false)
	if err != nil || node == "" {
		return err
	}

	children, err := client.GetNodes(p.nc, node, "all", data.NodeTypeProvisioningFile, false)
	if err != nil {
		return fmt.Errorf("error getting provisioning state: %w", err)
	}

	for _, c := range children {
		name, _ := c.Points.Text(data.PointTypeDescription, "")

		if seen[name] {
			continue
		}

		log.Printf("Provision: %v is gone, removing its state\n", name)

		if err := client.DeleteNode(p.nc, c.ID, node, "provision"); err != nil {
			return fmt.Errorf("error removing provisioning state: %w", err)
		}
	}

	return nil
}

// defaultProvisionInterval is how often a pass runs when nothing has told us
// to. The watch and the subscription do the work; this catches what they miss,
// such as a network filesystem fsnotify says nothing about, and costs one hash
// per source when nothing has changed.
const defaultProvisionInterval = time.Minute

// provisionDebounce is how long to wait for a directory to settle. A package
// manager or an editor writing several files produces a burst of events, and
// applying once at the end of it is both cheaper and less surprising than
// applying per file.
const provisionDebounce = time.Second

// watch runs a pass at start-up and then whenever a file changes, a file node
// changes, or the interval comes around, until cancel is closed.
func (p *provisioner) watch(interval time.Duration, cancel chan struct{}) error {
	if interval <= 0 {
		interval = defaultProvisionInterval
	}

	// a file node changing is an ordinary point message, so provisioning
	// learns about an upload the way everything else in SIOT learns about a
	// change
	changed := make(chan struct{}, 1)

	notify := func() {
		select {
		case changed <- struct{}{}:
		default:
			// a pass is already pending
		}
	}

	// A node point is published on p.<node>.<type>.<key>, so the subject says
	// which point changed and there is nothing to decode.
	//
	// Only the points an upload writes count. A description would be the
	// obvious one to include, but provisioning writes descriptions itself when
	// it records state, and reacting to those would have every pass schedule
	// another. Renaming a file node is picked up by the rescan instead.
	for _, typ := range []string{data.PointTypeData, data.PointTypeName, data.PointTypeCreated} {
		sub, err := p.nc.Subscribe(client.SubjectNodePoints("*")+"."+typ+".*", func(_ *nats.Msg) {
			notify()
		})

		if err != nil {
			return fmt.Errorf("provision: error subscribing: %w", err)
		}

		defer func() {
			_ = sub.Unsubscribe()
		}()
	}

	watcher, err := p.watchDir()
	if err != nil {
		log.Println("Provision:", err)
	}

	if watcher != nil {
		defer watcher.Close()
	}

	if err := p.run(); err != nil {
		log.Println("Provision:", err)
	}

	// the debounce timer is only running while events are arriving
	settle := time.NewTimer(provisionDebounce)
	if !settle.Stop() {
		<-settle.C
	}

	periodic := time.NewTicker(interval)
	defer periodic.Stop()

	var events <-chan fsnotify.Event
	var watchErrors <-chan error

	if watcher != nil {
		events = watcher.Events
		watchErrors = watcher.Errors
	}

	for {
		select {
		case <-cancel:
			return nil

		case e := <-events:
			if provisionFileName(filepath.Base(e.Name)) {
				settle.Reset(provisionDebounce)
			}

			continue

		case err := <-watchErrors:
			log.Println("Provision: watch error:", err)
			continue

		case <-changed:
			// wait for the tree to settle rather than reading it now. Sending
			// a node writes its points before the edge that attaches it, so a
			// pass run the instant a point arrives can look for the file node
			// before it is anywhere to be found.
			settle.Reset(provisionDebounce)
			continue

		case <-settle.C:
		case <-periodic.C:
		}

		if err := p.run(); err != nil {
			log.Println("Provision:", err)
		}
	}
}

// watchDir watches the provisioning directory. A directory that does not exist
// yet is not an error: the parent is watched instead, so that creating it later
// starts provisioning without a restart.
func (p *provisioner) watchDir() (*fsnotify.Watcher, error) {
	if p.dir == "" {
		return nil, nil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("error creating a watcher: %w", err)
	}

	if err := watcher.Add(p.dir); err == nil {
		return watcher, nil
	}

	parent := filepath.Dir(p.dir)

	if err := watcher.Add(parent); err != nil {
		watcher.Close()
		return nil, fmt.Errorf("provisioning directory %v does not exist, and neither does %v, "+
			"so changes to it will only be seen by the periodic rescan", p.dir, parent)
	}

	log.Printf("Provision: %v does not exist yet, watching %v for it\n", p.dir, parent)

	return watcher, nil
}

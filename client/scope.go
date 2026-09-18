package client

import (
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// maxScopeDepth bounds the walk from a target node up toward the root. The
// tree is never this deep in practice, so reaching the limit means a cycle or
// a runaway, and the target is refused.
const maxScopeDepth = 64

// checkWriteTarget returns an error when a client may not write to target.
// self is the client's own node and parent is the node it sits under. A
// client may write to its own node, to its parent, and to any node below its
// parent; a target anywhere else in the tree is refused.
//
// Rules, signal generators, and serial devices take the target from a point,
// and clients publish on the server's full-access connection, so without this
// check a user scoped to one group could write any point on any node whose
// ID they knew. A rule that needs to reach across groups belongs at a level
// that contains both.
//
// The target is also checked as a subject token before it is placed in one:
// an ID with a period in it would address a different subject.
//
// The walk goes up from the target, following every parent of every node it
// reaches, since a node can sit under more than one parent. It is bounded by
// maxScopeDepth and by a visited set. A target that does not exist has no
// parents to walk and is refused as well.
func checkWriteTarget(nc *nats.Conn, self, parent, target string) error {
	if target == "" {
		return fmt.Errorf("target node ID must be set")
	}

	if err := data.CheckSubjectToken(target); err != nil {
		return fmt.Errorf("target node ID %w", err)
	}

	if target == self || target == parent {
		return nil
	}

	visited := map[string]bool{target: true}
	frontier := []string{target}

	for depth := 0; depth < maxScopeDepth && len(frontier) > 0; depth++ {
		var next []string

		for _, id := range frontier {
			edges, err := GetNodes(nc, "all", id, "", false)
			if err != nil {
				return fmt.Errorf("error looking up node %v: %w", id, err)
			}

			for _, e := range edges {
				if e.Parent == parent {
					return nil
				}

				if e.Parent == "" || e.Parent == "root" || visited[e.Parent] {
					continue
				}

				visited[e.Parent] = true
				next = append(next, e.Parent)
			}
		}

		frontier = next
	}

	return fmt.Errorf("node %v is not under %v; a client may only write to its parent or nodes below it",
		target, parent)
}

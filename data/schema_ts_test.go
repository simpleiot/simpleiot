package data

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strconv"
	"testing"
)

// TestSchemaTS checks the point and node types the web UI declares in
// ui/src/lib/siot/schema.ts against the constants in schema.go, so a type
// renamed here fails until the UI is changed with it.
func TestSchemaTS(t *testing.T) {
	ts, err := os.ReadFile("../ui/src/lib/siot/schema.ts")
	if err != nil {
		t.Fatal(err)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "schema.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	goValues := map[string]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			if v, err := strconv.Unquote(lit.Value); err == nil {
				goValues[v] = true
			}
		}
		return true
	})

	entries := regexp.MustCompile(`(?m)^\t(\w+): "([^"]*)",$`).FindAllSubmatch(ts, -1)
	if len(entries) < 100 {
		t.Fatalf("found %v entries in schema.ts; has its layout changed?", len(entries))
	}
	for _, e := range entries {
		if !goValues[string(e[2])] {
			t.Errorf("schema.ts %s: %q is not a constant in schema.go", e[1], e[2])
		}
	}
}

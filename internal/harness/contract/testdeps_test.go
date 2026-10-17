package contract

import (
	"fmt"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// testOnlyModules are modules production code must not import. Rapid is a direct dependency
// for property-based tests only (phase-2-shacl-compilation, task 2.2); importing it from a
// non-test file would build it into every consumer's binary.
var testOnlyModules = []string{"pgregory.net/rapid"}

// testOnlyImportViolations reports every non-test .go file among files that imports a
// test-only module, and a go.mod that does not require rapid v1.3.0 directly.
func testOnlyImportViolations(t *testing.T, root string, files []string) []string {
	t.Helper()
	var bad []string
	fset := token.NewFileSet()
	for _, name := range files {
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(root, filepath.FromSlash(name)), nil, parser.ImportsOnly)
		if err != nil {
			bad = append(bad, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		for _, imp := range f.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				continue
			}
			for _, m := range testOnlyModules {
				if path == m || strings.HasPrefix(path, m+"/") {
					bad = append(bad, fmt.Sprintf("%s:%d: production file imports test-only module %s",
						name, fset.Position(imp.Pos()).Line, path))
				}
			}
		}
	}
	direct := false
	for _, line := range readLines(t, root, "go.mod") {
		fields := strings.Fields(strings.TrimPrefix(strings.TrimSpace(line), "require "))
		if len(fields) >= 2 && fields[0] == "pgregory.net/rapid" && fields[1] == "v1.3.0" &&
			!strings.Contains(line, "// indirect") {
			direct = true
		}
	}
	if !direct {
		bad = append(bad, "go.mod: pgregory.net/rapid v1.3.0 is not a direct requirement")
	}
	return bad
}

func TestTestOnlyImports(t *testing.T) {
	root := repoRoot(t)
	requireNoViolations(t, "test-only import", testOnlyImportViolations(t, root, repoFiles(t, root)))
}

func TestTestOnlyImportsSensitivity(t *testing.T) {
	const goMod = "module example\n\ngo 1.26\n\nrequire pgregory.net/rapid v1.3.0\n"
	const rapidUser = "package p\n\nimport \"pgregory.net/rapid\"\n\nvar _ = rapid.Int\n"
	t.Run("clean", func(t *testing.T) {
		root, files := writeTree(t, map[string]string{"go.mod": goMod, "p/p_test.go": rapidUser})
		requireNoViolations(t, "clean tree", testOnlyImportViolations(t, root, files))
	})
	t.Run("production import", func(t *testing.T) {
		root, files := writeTree(t, map[string]string{"go.mod": goMod, "p/p.go": rapidUser})
		requireViolation(t, testOnlyImportViolations(t, root, files), "p/p.go:3", "pgregory.net/rapid")
	})
	t.Run("indirect requirement", func(t *testing.T) {
		root, files := writeTree(t, map[string]string{
			"go.mod": strings.Replace(goMod, "v1.3.0", "v1.3.0 // indirect", 1), "p/p_test.go": rapidUser,
		})
		requireViolation(t, testOnlyImportViolations(t, root, files), "not a direct requirement")
	})
	t.Run("other version", func(t *testing.T) {
		root, files := writeTree(t, map[string]string{
			"go.mod": strings.Replace(goMod, "v1.3.0", "v1.2.0", 1), "p/p_test.go": rapidUser,
		})
		requireViolation(t, testOnlyImportViolations(t, root, files), "not a direct requirement")
	})
}

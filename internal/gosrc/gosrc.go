// Package gosrc reads Go source for tests and fixture generators: the exported
// identifiers of a package and the values of its exported string constants,
// evaluated by go/types. Catalog tests use it so the checked set is what the
// package actually exports, not a hand-written copy. Production code must not
// import this package.
package gosrc

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Package is the exported surface of one Go package.
type Package struct {
	// Exported lists every exported package-level identifier, sorted.
	Exported []string
	// Consts maps each exported string constant to its value.
	Consts map[string]string
}

// Load type-checks the non-test .go files of dir. Imports are resolved from
// source, so the package may import the standard library.
func Load(dir string) (Package, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return Package{}, fmt.Errorf("gosrc: %w", err)
	}
	var paths []string
	for _, e := range entries {
		n := e.Name()
		if strings.HasSuffix(n, ".go") && !strings.HasSuffix(n, "_test.go") {
			paths = append(paths, filepath.Join(dir, n))
		}
	}
	return LoadFiles(paths...)
}

// LoadFiles type-checks the given files as one package.
func LoadFiles(paths ...string) (Package, error) {
	if len(paths) == 0 {
		return Package{}, fmt.Errorf("gosrc: no files")
	}
	fset := token.NewFileSet()
	var files []*ast.File
	for _, p := range paths {
		f, err := parser.ParseFile(fset, p, nil, parser.SkipObjectResolution)
		if err != nil {
			return Package{}, fmt.Errorf("gosrc: %w", err)
		}
		files = append(files, f)
	}
	conf := types.Config{Importer: importer.ForCompiler(fset, "source", nil)}
	pkg, err := conf.Check(files[0].Name.Name, fset, files, nil)
	if err != nil {
		return Package{}, fmt.Errorf("gosrc: type-check: %w", err)
	}
	out := Package{Consts: map[string]string{}}
	scope := pkg.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if !obj.Exported() {
			continue
		}
		out.Exported = append(out.Exported, name)
		if c, ok := obj.(*types.Const); ok && c.Val().Kind() == constant.String {
			out.Consts[name] = constant.StringVal(c.Val())
		}
	}
	sort.Strings(out.Exported)
	return out, nil
}

// MapLiteralIdents returns the identifier key/value pairs of the composite
// literal assigned to the package-level variable name in file, in source order.
// Non-identifier keys or values fail.
func MapLiteralIdents(path, name string) ([][2]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("gosrc: %w", err)
	}
	for _, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			vs := spec.(*ast.ValueSpec)
			for i, id := range vs.Names {
				if id.Name != name || i >= len(vs.Values) {
					continue
				}
				lit, ok := vs.Values[i].(*ast.CompositeLit)
				if !ok {
					return nil, fmt.Errorf("gosrc: %s is not a composite literal", name)
				}
				var out [][2]string
				for _, elt := range lit.Elts {
					kv, ok := elt.(*ast.KeyValueExpr)
					if !ok {
						return nil, fmt.Errorf("gosrc: %s: element at %s is not key: value", name, fset.Position(elt.Pos()))
					}
					k, kok := kv.Key.(*ast.Ident)
					v, vok := kv.Value.(*ast.Ident)
					if !kok || !vok {
						return nil, fmt.Errorf("gosrc: %s: non-identifier entry at %s", name, fset.Position(elt.Pos()))
					}
					out = append(out, [2]string{k.Name, v.Name})
				}
				return out, nil
			}
		}
	}
	return nil, fmt.Errorf("gosrc: no variable %s in %s", name, path)
}

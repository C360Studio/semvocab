package shacl

// Issue #10, ruling Q32: an IRI that is a node shape (ruling Q27) and also
// the object of a node shape's sh:property is built in neither role, and
// Compile reports exactly one issue for it, naming both roles.

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

// compileOrders parses the statements in their given order and reversed,
// and compiles each set against testPack and testMapping.
func compileOrders(t *testing.T, stmts []string) (sets []ShapeSet, errs []error) {
	t.Helper()
	rev := slices.Clone(stmts)
	slices.Reverse(rev)
	for _, order := range [][]string{stmts, rev} {
		set := parseTestTurtle(t, strings.Join(order, "\n"))
		_, err := Compile(testPack(), set, testMapping())
		sets = append(sets, set)
		errs = append(errs, err)
	}
	return sets, errs
}

// issuesOn returns the issues err reports on shape.
func issuesOn(t *testing.T, err error, shape string) []Issue {
	t.Helper()
	var ce *CompileError
	if !errors.As(err, &ce) {
		t.Fatalf("err = %v, want a *CompileError", err)
	}
	var out []Issue
	for _, is := range ce.Issues {
		if is.Shape == shape {
			out = append(out, is)
		}
	}
	return out
}

// requireDualRoleIssue checks that err reports exactly one issue on ex:P,
// and that it is the dual-role issue naming each of want.
func requireDualRoleIssue(t *testing.T, err error, want ...string) {
	t.Helper()
	got := issuesOn(t, err, ex+"P")
	if len(got) != 1 {
		t.Fatalf("want exactly one issue on ex:P, got %d:\n%v", len(got), err)
	}
	is := got[0]
	if is.Construct != sh+"property" || is.Path != "" {
		t.Errorf("issue = %+v, want construct sh:property and no path", is)
	}
	for _, w := range append([]string{"shape <" + ex + "P> is used as a node shape", "and as a property shape"},
		want...) {
		if !strings.Contains(is.Message, w) {
			t.Errorf("issue message %q does not contain %q", is.Message, w)
		}
	}
}

// requireNotBuilt checks that ex:P is neither a node shape nor a property
// shape in set.
func requireNotBuilt(t *testing.T, set ShapeSet) {
	t.Helper()
	for _, ns := range set.NodeShapes {
		if ns.ID == ex+"P" {
			t.Errorf("ex:P was built as a node shape: %+v", ns)
		}
		for _, ps := range ns.Properties {
			if ps.ID == ex+"P" {
				t.Errorf("ex:P was built as a property shape of %s: %+v", ns.ID, ps)
			}
		}
	}
}

// TestDualRoleShapeIssue10 is issue #10's example, verbatim. Before the fix
// ex:P was built in both roles and drew four unrelated records
// (sh:minCount, sh:path, rdf:type, sh:targetClass).
func TestDualRoleShapeIssue10(t *testing.T) {
	sets, errs := compileOrders(t, []string{
		`ex:P a sh:NodeShape ; sh:targetClass ex:C ; sh:path ex:p ; sh:minCount 1 .`,
		`ex:A a sh:NodeShape ; sh:targetClass ex:D ; sh:property ex:P .`,
	})
	for i := range sets {
		requireDualRoleIssue(t, errs[i], "(it has sh:targetClass)", "(sh:property from <"+ex+"A>)",
			"a property shape with its own target is outside the profile")
		requireNotBuilt(t, sets[i])
	}
}

// TestDualRoleShapeTwoReferrers: both referring node shapes are named, the
// IRI's own blank nodes and list cells draw no records, and the referring
// shapes keep their other properties. The issue is the only one, so the
// rest of the document would compile; Compile still fails and returns no
// bundle.
func TestDualRoleShapeTwoReferrers(t *testing.T) {
	stmts := []string{
		`ex:P sh:targetClass ex:C ; sh:path ex:p ; sh:in ( "a" ) ; sh:property [ sh:path ex:r ] .`,
		`ex:A sh:targetClass ex:C ; sh:property ex:P , [ sh:path ex:q ; sh:minCount 1 ] .`,
		`ex:B sh:targetClass ex:C ; sh:property ex:P .`,
	}
	sets, errs := compileOrders(t, stmts)
	for i, set := range sets {
		requireDualRoleIssue(t, errs[i], "(sh:property from <"+ex+"A>, <"+ex+"B>)")
		var ce *CompileError
		errors.As(errs[i], &ce)
		if len(ce.Issues) != 1 {
			t.Errorf("want the dual-role issue only, got:\n%v", errs[i])
		}
		requireNotBuilt(t, set)
		want := []NodeShape{
			{ID: ex + "A", TargetClasses: []string{ex + "C"}, Properties: []PropertyShape{{Path: ex + "q", MinCount: 1}}},
			{ID: ex + "B", TargetClasses: []string{ex + "C"}},
		}
		if !nodeShapesEqual(set.NodeShapes, want) {
			t.Errorf("node shapes = %+v, want %+v", set.NodeShapes, want)
		}
	}
	b, err := Compile(testPack(), sets[0], testMapping())
	if b != nil || err == nil {
		t.Errorf("Compile = %v, %v; want no bundle and an error", b, err)
	}
}

// TestDualRoleShapeEachNodeShapeMarker covers every way Q27 makes a subject
// a node shape: each target predicate, and the sh:NodeShape type alone. A
// self-reference is a dual role too. The unsupported-target records the
// IRI would otherwise draw are replaced by the one issue.
func TestDualRoleShapeEachNodeShapeMarker(t *testing.T) {
	for _, c := range []struct {
		name, p, refs string
		want          []string
	}{
		{"targetNode", `ex:P sh:targetNode ex:n ; sh:path ex:p .`, `ex:A sh:targetClass ex:C ; sh:property ex:P .`,
			[]string{"(it has sh:targetNode)", "own target"}},
		{"targetSubjectsOf", `ex:P sh:targetSubjectsOf ex:p ; sh:path ex:p .`,
			`ex:A sh:targetClass ex:C ; sh:property ex:P .`, []string{"(it has sh:targetSubjectsOf)"}},
		{"targetObjectsOf", `ex:P sh:targetObjectsOf ex:p ; sh:path ex:p .`,
			`ex:A sh:targetClass ex:C ; sh:property ex:P .`, []string{"(it has sh:targetObjectsOf)"}},
		{"target", `ex:P sh:target [ a ex:T ; ex:x ex:y ] ; sh:path ex:p .`,
			`ex:A sh:targetClass ex:C ; sh:property ex:P .`, []string{"(it has sh:target)"}},
		{"several targets", `ex:P sh:targetNode ex:n ; sh:targetClass ex:C ; sh:path ex:p .`,
			`ex:A sh:targetClass ex:C ; sh:property ex:P .`, []string{"(it has sh:targetClass, sh:targetNode)"}},
		{"type only", `ex:P a sh:NodeShape ; sh:path ex:p .`, `ex:A sh:targetClass ex:C ; sh:property ex:P .`,
			[]string{"(it is typed sh:NodeShape)", "a property shape typed sh:NodeShape is outside the profile"}},
		{"self", `ex:P sh:targetClass ex:C ; sh:path ex:p .`, `ex:P sh:property ex:P .`,
			[]string{"(sh:property from <" + ex + "P>)"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			sets, errs := compileOrders(t, []string{c.p, c.refs})
			for i, set := range sets {
				requireDualRoleIssue(t, errs[i], c.want...)
				var ce *CompileError
				errors.As(errs[i], &ce)
				if len(ce.Issues) != 1 {
					t.Errorf("want the dual-role issue only, got:\n%v", errs[i])
				}
				requireNotBuilt(t, set)
			}
		})
	}
}

func nodeShapesEqual(a, b []NodeShape) bool {
	return slices.EqualFunc(a, b, func(x, y NodeShape) bool {
		return x.ID == y.ID && slices.Equal(x.TargetClasses, y.TargetClasses) &&
			slices.EqualFunc(x.Properties, y.Properties, func(p, q PropertyShape) bool {
				return p.ID == q.ID && p.Path == q.Path && p.MinCount == q.MinCount && p.MaxCount == nil &&
					q.MaxCount == nil && p.In == nil && q.In == nil
			})
	})
}

package shacl

// Failures the task 7 generated checks found, kept as named tests
// (notes/7-generated.md).

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

// TestBlankCycleRecordsIndependentOfOrder: a blank-node cycle outside every
// shape is reported the same whichever of its nodes comes first. Found by
// TestRapidShapesStatementPermutation (-rapid.seed=1209462990388893792):
// the node seen first recorded its triples and took the other as owned, so
// _:c2's "w" triple was reported in one order and not in the other.
func TestBlankCycleRecordsIndependentOfOrder(t *testing.T) {
	const ns = "<http://example.org/gen/"
	lines := []string{
		"_:c1 " + ns + "r> _:c2 .",
		"_:c2 " + ns + "q> \"w\" .",
		"_:c2 " + ns + "r> _:c1 .",
	}
	first := parseTestTurtle(t, strings.Join(lines, "\n"))
	second := parseTestTurtle(t, strings.Join([]string{lines[1], lines[2], lines[0]}, "\n"))
	if !reflect.DeepEqual(first.rejections, second.rejections) {
		t.Fatalf("records depend on triple order:\n%s\nvs\n%s", formatRejections(first.rejections),
			formatRejections(second.rejections))
	}
	requireRejection(t, first, rejection{Construct: "http://example.org/gen/q"})
}

// TestConflictingMappingRowsIndependentOfOrder: conflicting mapping rows
// give the same issues in either order. Found by TestRapidMappingRowOrder
// (-rapid.seed=69982396324603329): with two rows for one class, the first
// row read was kept, so the issues about the other row's types depended on
// which came first. The same held for a path mapped twice, and for a group
// declared twice with different modes.
func TestConflictingMappingRowsIndependentOfOrder(t *testing.T) {
	set := parseTestTurtle(t, `ex:S a sh:NodeShape ; sh:targetClass ex:C ; sh:property [ sh:path ex:p ] .`)
	m := testMapping()
	m.EntityTypes = append(m.EntityTypes, EntityTypeBinding{ClassIRI: ex + "C", DomainTypes: []string{"test.other"}})
	m.Predicates = append(m.Predicates, PredicateBinding{PathIRI: ex + "p", Name: "test.prop.zz"})
	m.Contracts[0].Groups = append(m.Contracts[0].Groups,
		GroupBinding{Name: "late", Mode: "append", Predicates: []string{"test.prop.p"}},
		GroupBinding{Name: "late", Mode: "reconcile", Predicates: []string{"test.prop.p"}})

	reversed := cloneMapping(m)
	slices.Reverse(reversed.EntityTypes)
	slices.Reverse(reversed.Predicates)
	slices.Reverse(reversed.Contracts[0].Groups)

	_, err1 := Compile(testPack(), set, m)
	_, err2 := Compile(testPack(), set, reversed)
	if err1 == nil || err2 == nil {
		t.Fatalf("conflicting rows compiled: %v, %v", err1, err2)
	}
	if err1.Error() != err2.Error() {
		t.Fatalf("issues depend on row order:\n%v\nvs\n%v", err1, err2)
	}
}

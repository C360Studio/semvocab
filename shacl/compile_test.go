package shacl

import (
	"strings"
	"testing"

	"github.com/c360studio/semengine/pkg/projection/contract"
	"github.com/c360studio/semengine/vocabulary"
)

const ex = "http://example.org/"

// testPack is a small base pack: entity type test.thing (class ex:C) and
// predicates test.prop.p, .q, and .r (paths ex:p, ex:q, ex:r) with no
// data_type, so the shapes fill it.
func testPack() vocabulary.PackSpec {
	return vocabulary.PackSpec{
		Name: "test",
		EntityTypes: []vocabulary.EntityTypeSpec{
			{DomainType: "test.thing", ClassIRI: ex + "C"},
		},
		Predicates: []vocabulary.PredicateSpec{
			{Name: "test.prop.p", IRI: ex + "p"},
			{Name: "test.prop.q", IRI: ex + "q"},
			{Name: "test.prop.r", IRI: ex + "r"},
		},
	}
}

// testMapping binds testPack: ex:C to test.thing, each path to its
// predicate, and one reconcile group holding all three.
func testMapping() Mapping {
	return Mapping{
		EntityTypes: []EntityTypeBinding{{ClassIRI: ex + "C", DomainTypes: []string{"test.thing"}}},
		Predicates: []PredicateBinding{
			{PathIRI: ex + "p", Name: "test.prop.p"},
			{PathIRI: ex + "q", Name: "test.prop.q"},
			{PathIRI: ex + "r", Name: "test.prop.r"},
		},
		Contracts: []ContractBinding{{
			DomainType: "test.thing",
			Groups: []GroupBinding{{Mode: contract.ModeReconcile,
				Predicates: []string{"test.prop.p", "test.prop.q", "test.prop.r"}}},
		}},
	}
}

func parseTestTurtle(t *testing.T, src string) ShapeSet {
	t.Helper()
	set, err := ParseTurtle(strings.NewReader(prefix + src))
	if err != nil {
		t.Fatalf("parse: %v\n%s", err, src)
	}
	return set
}

// compileTurtle parses src (after the shared prefixes) and compiles it
// against testPack and testMapping.
func compileTurtle(t *testing.T, src string) (*Bundle, error) {
	t.Helper()
	return Compile(testPack(), parseTestTurtle(t, src), testMapping())
}

// mustCompile compiles src and fails the test on error.
func mustCompile(t *testing.T, src string) *Bundle {
	t.Helper()
	b, err := compileTurtle(t, src)
	if err != nil {
		t.Fatalf("compile: %v\n%s", err, src)
	}
	return b
}

// onlyProperty returns the single compiled property of test.thing's single
// shape.
func onlyProperty(t *testing.T, b *Bundle) compiledProperty {
	t.Helper()
	shapes := b.shapes["test.thing"]
	if len(shapes) != 1 || len(shapes[0].Props) != 1 {
		t.Fatalf("want one shape with one property, got %+v", shapes)
	}
	return shapes[0].Props[0]
}

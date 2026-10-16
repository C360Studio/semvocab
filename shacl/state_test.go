package shacl

import (
	"slices"
	"testing"

	"github.com/c360studio/semengine/message"
	"github.com/c360studio/semengine/pkg/projection/contract"
)

// These are issue #1's complete-candidate-state cases (task 6.4, design
// D13). The candidate is built by a test-only model of SemEngine's write
// modes (pkg/projection/contract/contract.go:22-31 at v0.1.0-alpha.2): a
// reconcile replaces its group's set, and an append adds to it. The mode is
// read from the compiled contract, never from sh:maxCount.

// state is one entity's triples.
type state []message.Triple

// write applies one group write to stored state, by the group's mode in the
// bundle's contract for the entity's type.
func write(t *testing.T, b *Bundle, stored state, group string, triples ...message.Triple) state {
	t.Helper()
	for _, c := range b.Contracts() {
		for _, g := range c.Groups {
			if g.Name != group {
				continue
			}
			out := slices.Clone(stored)
			if g.Mode == contract.ModeReconcile {
				out = slices.DeleteFunc(out, func(tr message.Triple) bool { return slices.Contains(g.Predicates, tr.Predicate) })
			}
			for _, tr := range triples {
				if !slices.Contains(g.Predicates, tr.Predicate) {
					t.Fatalf("group %s does not hold %s", group, tr.Predicate)
				}
			}
			return append(out, triples...)
		}
	}
	t.Fatalf("no group %s in %+v", group, b.Contracts())
	return nil
}

// stateBundle compiles one shape over ex:p (maxCount 2, minCount from
// minCount) and ex:q (unbounded), with the given groups.
func stateBundle(t *testing.T, minCount string, groups ...GroupBinding) *Bundle {
	t.Helper()
	m := testMapping()
	m.Contracts = []ContractBinding{{DomainType: "test.thing", Groups: groups}}
	b, err := Compile(testPack(), parseTestTurtle(t, `ex:S sh:targetClass ex:C ;
		sh:property [ sh:path ex:p ; sh:datatype xsd:string ; sh:minCount `+minCount+` ; sh:maxCount 2 ] ;
		sh:property [ sh:path ex:q ; sh:datatype xsd:integer ] .`), m)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func group(name string, mode contract.WriteMode, preds ...string) GroupBinding {
	return GroupBinding{Name: name, Mode: mode, Predicates: preds}
}

func values(s state, pred string) []any {
	var out []any
	for _, tr := range s {
		if tr.Predicate == pred {
			out = append(out, tr.Object)
		}
	}
	return out
}

var maxCountP = res(ex+"S", ex+"p", "test.prop.p", "MaxCountConstraintComponent", "")

// TestReconcileVersusAppend is the spec scenario: stored {A,B} take a
// reconcile {B,C} or an append {C}; the candidates are {B,C} and {A,B,C},
// and each is checked against sh:maxCount 2.
func TestReconcileVersusAppend(t *testing.T) {
	stored := state{tr("test.prop.p", "A"), tr("test.prop.p", "B")}

	rec := stateBundle(t, "0", group("tags", contract.ModeReconcile, "test.prop.p", "test.prop.q"))
	candidate := write(t, rec, stored, "tags", tr("test.prop.p", "B"), tr("test.prop.p", "C"))
	if got := values(candidate, "test.prop.p"); !slices.Equal(got, []any{"B", "C"}) {
		t.Fatalf("reconcile candidate %v, want [B C]", got)
	}
	requireResults(t, validate(t, rec, candidate...))

	app := stateBundle(t, "0", group("tags", contract.ModeAppend, "test.prop.p", "test.prop.q"))
	candidate = write(t, app, stored, "tags", tr("test.prop.p", "C"))
	if got := values(candidate, "test.prop.p"); !slices.Equal(got, []any{"A", "B", "C"}) {
		t.Fatalf("append candidate %v, want [A B C]", got)
	}
	requireResults(t, validate(t, app, candidate...), maxCountP)
}

// TestMixedCardinalitiesDoNotChooseTheMode: one group holds ex:p
// (sh:maxCount 2) and ex:q (no maximum). The compiled group's mode is the
// mapping's either way (design D7), and validation reports the maximum
// whatever the mode.
func TestMixedCardinalitiesDoNotChooseTheMode(t *testing.T) {
	for _, mode := range []contract.WriteMode{contract.ModeReconcile, contract.ModeAppend} {
		b := stateBundle(t, "0", group("mixed", mode, "test.prop.p", "test.prop.q"))
		cs := b.Contracts()
		if len(cs) != 1 || len(cs[0].Groups) != 1 || cs[0].Groups[0].Mode != mode {
			t.Fatalf("mapping mode %s compiled to %+v", mode, cs)
		}
		candidate := write(t, b, nil, "mixed", tr("test.prop.p", "A"), tr("test.prop.p", "B"),
			tr("test.prop.p", "C"), tr("test.prop.q", 1), tr("test.prop.q", 2), tr("test.prop.q", 3))
		requireResults(t, validate(t, b, candidate...), maxCountP)
	}
}

// TestBirthMissingRequiredValue is the spec scenario: a birth candidate (no
// stored state) that lacks ex:p, which has sh:minCount 1, has an
// sh:MinCountConstraintComponent result.
func TestBirthMissingRequiredValue(t *testing.T) {
	m := testMapping()
	m.Contracts = []ContractBinding{{DomainType: "test.thing", Birth: []string{"test.prop.p"},
		Groups: []GroupBinding{group("counts", contract.ModeReconcile, "test.prop.q")}}}
	b, err := Compile(testPack(), parseTestTurtle(t, `ex:S sh:targetClass ex:C ;
		sh:property [ sh:path ex:p ; sh:datatype xsd:string ; sh:minCount 1 ] ;
		sh:property [ sh:path ex:q ; sh:datatype xsd:integer ] .`), m)
	if err != nil {
		t.Fatal(err)
	}
	if cs := b.Contracts(); len(cs) != 1 || !slices.Equal(cs[0].BirthPredicates, []string{"test.prop.p"}) {
		t.Fatalf("contracts %+v, want birth predicate test.prop.p", cs)
	}
	requireResults(t, validate(t, b, tr("test.prop.q", 1)),
		res(ex+"S", ex+"p", "test.prop.p", "MinCountConstraintComponent", ""))
	requireResults(t, validate(t, b, tr("test.prop.p", "born"), tr("test.prop.q", 1)))
}

// TestRebuiltPartialUpdate: a write that changes only ex:q passes once the
// candidate is rebuilt with the stored required ex:p. The patch alone, which
// D13 says the validator does not take, would fail sh:minCount.
func TestRebuiltPartialUpdate(t *testing.T) {
	b := stateBundle(t, "1", group("tags", contract.ModeReconcile, "test.prop.p"),
		group("counts", contract.ModeReconcile, "test.prop.q"))
	stored := state{tr("test.prop.p", "A"), tr("test.prop.q", 1)}
	patch := []message.Triple{tr("test.prop.q", 2)}

	candidate := write(t, b, stored, "counts", patch...)
	if got := values(candidate, "test.prop.q"); !slices.Equal(got, []any{2}) {
		t.Fatalf("candidate q %v, want [2]", got)
	}
	requireResults(t, validate(t, b, candidate...))
	requireResults(t, validate(t, b, patch...), res(ex+"S", ex+"p", "test.prop.p", "MinCountConstraintComponent", ""))
}

// TestAppendPastStoredMaximum: stored {A,B} is at sh:maxCount 2, so an
// append of one more value fails, though the appended value alone would not.
func TestAppendPastStoredMaximum(t *testing.T) {
	b := stateBundle(t, "0", group("tags", contract.ModeAppend, "test.prop.p", "test.prop.q"))
	stored := state{tr("test.prop.p", "A"), tr("test.prop.p", "B")}
	requireResults(t, validate(t, b, stored...))
	requireResults(t, validate(t, b, write(t, b, stored, "tags", tr("test.prop.p", "C"))...), maxCountP)
	requireResults(t, validate(t, b, tr("test.prop.p", "C")))
	// Appending a value already stored adds no term, so it stays at the maximum.
	requireResults(t, validate(t, b, write(t, b, stored, "tags", tr("test.prop.p", "B"))...))
}

// TestStaleAppendRevisionBoundary is the spec scenario on the revision
// boundary (design D13). Two writers read stored {A} and each validates one
// append: {A,B} and {A,C} each conform to sh:maxCount 2. An append carries
// no revision at the pinned SemEngine (pkg/projection/mutation_types.go:61-67
// at v0.1.0-alpha.2; only a reconcile has ExpectedRevision, :46-58), so both
// can land, and the persisted {A,B,C} fails. A validated candidate is a
// snapshot, not a persisted guarantee.
func TestStaleAppendRevisionBoundary(t *testing.T) {
	b := stateBundle(t, "0", group("tags", contract.ModeAppend, "test.prop.p", "test.prop.q"))
	read := state{tr("test.prop.p", "A")}

	first := write(t, b, read, "tags", tr("test.prop.p", "B"))
	second := write(t, b, read, "tags", tr("test.prop.p", "C"))
	requireResults(t, validate(t, b, first...))
	requireResults(t, validate(t, b, second...))

	persisted := write(t, b, write(t, b, read, "tags", tr("test.prop.p", "B")), "tags", tr("test.prop.p", "C"))
	requireResults(t, validate(t, b, persisted...), maxCountP)

	// The rule the boundary implies: on re-reading the persisted state, the
	// second writer's candidate is rebuilt from it and fails.
	reread := write(t, b, first, "tags", tr("test.prop.p", "C"))
	requireResults(t, validate(t, b, reread...), maxCountP)
}

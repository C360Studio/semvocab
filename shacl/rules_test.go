package shacl

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/c360studio/semengine/pkg/projection/contract"
	semtypes "github.com/c360studio/semengine/pkg/types"
	"github.com/c360studio/semengine/vocabulary"
)

// TestClosedShapes is task 5.5 (design D12): a closed shape must ignore
// rdf:type; it allows its own mapped paths, its mapped ignored properties,
// and the mapping's closed_ignored_predicates; sh:closed false has no effect.
func TestClosedShapes(t *testing.T) {
	_, err := compileTurtle(t, `ex:S sh:targetClass ex:C ; sh:closed true ; sh:property [ sh:path ex:p ] .`)
	if is := requireIssue(t, err, sh+"closed", "rdf:type", "closed-001"); is.Shape != ex+"S" {
		t.Errorf("issue = %+v, want shape ex:S", is)
	}
	_, err = compileTurtle(t, `ex:S sh:targetClass ex:C ; sh:closed true ; sh:ignoredProperties ( ex:q ) ; sh:property [ sh:path ex:p ] .`)
	requireIssue(t, err, sh+"closed", "rdf:type")

	m := testMapping()
	m.ClosedIgnoredPredicates = []string{"test.meta.note", "test.meta.note"}
	b, err := Compile(testPack(), parseTestTurtle(t,
		`ex:S sh:targetClass ex:C ; sh:closed true ; sh:ignoredProperties ( rdf:type ex:q ex:unmapped ) ; sh:property [ sh:path ex:p ] .`), m)
	if err != nil {
		t.Fatal(err)
	}
	s := b.shapes["test.thing"][0]
	if want := []string{"test.meta.note", "test.prop.p", "test.prop.q"}; !s.Closed || !slices.Equal(s.Allowed, want) {
		t.Errorf("closed %v allowed %v, want closed with %v", s.Closed, s.Allowed, want)
	}

	open := mustCompile(t, `ex:S sh:targetClass ex:C ; sh:closed false ; sh:property [ sh:path ex:p ] .`)
	plain := mustCompile(t, `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ] .`)
	if !reflect.DeepEqual(open, plain) {
		t.Error("sh:closed false changed the bundle")
	}

	m = testMapping()
	m.ClosedIgnoredPredicates = []string{"not-dotted"}
	_, err = Compile(testPack(), parseTestTurtle(t, `ex:S sh:targetClass ex:C .`), m)
	requireIssue(t, err, "closed_ignored_predicates", `"not-dotted"`)

	// The W3C case D12 follows: core/node/closed-001 lists no rdf:type in
	// sh:ignoredProperties, so the rule rejects it with no adapter.
	_, err = Compile(vocabulary.PackSpec{Name: "w3c"}, loadW3C(t, "node/closed-001.ttl"), Mapping{})
	requireIssue(t, err, sh+"closed", "rdf:type")
}

// subclassPack adds test.sub (class ex:D, a subclass of ex:C) and test.other
// (class ex:E) to testPack.
func subclassPack() vocabulary.PackSpec {
	p := testPack()
	p.EntityTypes = append(p.EntityTypes,
		vocabulary.EntityTypeSpec{DomainType: "test.sub", ClassIRI: ex + "D", Superclasses: []string{ex + "C"}},
		vocabulary.EntityTypeSpec{DomainType: "test.other", ClassIRI: ex + "E"},
	)
	return p
}

// TestMappingRules is task 5.6 (design D5, D6, rulings Q13, Q14, Q24).
func TestMappingRules(t *testing.T) {
	shapes := `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:datatype xsd:integer ] .`

	t.Run("explicit bindings", func(t *testing.T) {
		m := testMapping()
		m.Predicates = m.Predicates[1:] // drop ex:p
		_, err := Compile(testPack(), parseTestTurtle(t, shapes), m)
		if is := requireIssue(t, err, sh+"path", "no predicates mapping row"); is.Shape != ex+"S" || is.Path != ex+"p" {
			t.Errorf("issue = %+v, want shape ex:S and path ex:p", is)
		}
		m = testMapping()
		m.EntityTypes = nil
		_, err = Compile(testPack(), parseTestTurtle(t, shapes), m)
		if is := requireIssue(t, err, sh+"targetClass", ex+"C"); is.Shape != ex+"S" {
			t.Errorf("issue = %+v, want shape ex:S", is)
		}
	})

	t.Run("iri agreement", func(t *testing.T) {
		p := testPack()
		p.Predicates[0].IRI = ex + "other"
		_, err := Compile(p, parseTestTurtle(t, shapes), testMapping())
		requireIssue(t, err, "predicates", "export writes the pack iri")
		m := testMapping()
		m.Predicates[0].Name = "test.prop.missing"
		_, err = Compile(testPack(), parseTestTurtle(t, shapes), m)
		requireIssue(t, err, "predicates", "test.prop.missing, which the pack does not declare")
	})

	t.Run("conflicting mappings", func(t *testing.T) {
		m := testMapping()
		m.Predicates = append(m.Predicates, PredicateBinding{PathIRI: ex + "p", Name: "test.prop.q"})
		_, err := Compile(testPack(), parseTestTurtle(t, shapes), m)
		requireIssue(t, err, "predicates", "mapped twice (test.prop.p and test.prop.q)")
		m = testMapping()
		m.EntityTypes = append(m.EntityTypes, EntityTypeBinding{ClassIRI: ex + "C", DomainTypes: []string{"test.thing"}})
		_, err = Compile(testPack(), parseTestTurtle(t, shapes), m)
		requireIssue(t, err, "entity_types", "more than one row")
	})

	t.Run("closure-checked targeting", func(t *testing.T) {
		m := testMapping()
		m.Contracts = append(m.Contracts, ContractBinding{DomainType: "test.sub", Groups: m.Contracts[0].Groups})
		_, err := Compile(subclassPack(), parseTestTurtle(t, shapes), m)
		requireIssue(t, err, "entity_types", "does not list test.sub")

		m.EntityTypes[0].DomainTypes = []string{"test.thing", "test.sub"}
		b, err := Compile(subclassPack(), parseTestTurtle(t, shapes), m)
		if err != nil {
			t.Fatal(err)
		}
		if len(b.shapes["test.sub"]) != 1 || len(b.shapes["test.thing"]) != 1 {
			t.Errorf("shapes per type = %v, want the shape on test.thing and test.sub", b.shapes)
		}

		m.EntityTypes[0].DomainTypes = []string{"test.thing", "test.sub", "test.other"}
		_, err = Compile(subclassPack(), parseTestTurtle(t, shapes), m)
		requireIssue(t, err, "entity_types", "lists test.other, whose class_iri and superclasses do not contain it")
		m.EntityTypes[0].DomainTypes = []string{"test.thing", "test.sub", "test.none"}
		_, err = Compile(subclassPack(), parseTestTurtle(t, shapes), m)
		requireIssue(t, err, "entity_types", "lists test.none, which the pack does not declare")
	})

	t.Run("datatype fill and conflict", func(t *testing.T) {
		p := testPack()
		p.Predicates[0].DataType = vocabulary.DataTypeFloat
		_, err := Compile(p, parseTestTurtle(t, shapes), testMapping())
		is := requireIssue(t, err, sh+"datatype", `test.prop.p has pack data_type "float"`)
		if is.Shape != ex+"S" || is.Path != ex+"p" {
			t.Errorf("issue = %+v, want shape ex:S and path ex:p", is)
		}
		p.Predicates[0].DataType = vocabulary.DataTypeInt
		if _, err := Compile(p, parseTestTurtle(t, shapes), testMapping()); err != nil {
			t.Errorf("agreeing data_type: %v", err)
		}

		// The spec's scenario: two shapes give one predicate xsd:double and
		// xsd:integer; the issue names the predicate and both shapes.
		_, err = compileTurtle(t, `
ex:S1 sh:targetClass ex:C ; sh:property ex:P1 .
ex:P1 sh:path ex:p ; sh:datatype xsd:double .
ex:S2 sh:targetClass ex:C ; sh:property ex:P2 .
ex:P2 sh:path ex:p ; sh:datatype xsd:integer .`)
		is = requireIssue(t, err, sh+"datatype", "test.prop.p", ex+"P1")
		if is.Shape != ex+"P2" {
			t.Errorf("issue = %+v, want it on ex:P2 naming ex:P1", is)
		}
	})

	t.Run("several shapes on one class", func(t *testing.T) {
		b := mustCompile(t, `
ex:S1 sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:minCount 1 ] .
ex:S2 sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:maxCount 3 ] ; sh:property [ sh:path ex:q ; sh:maxCount 1 ] .`)
		got := b.shapes["test.thing"]
		if len(got) != 2 || got[0].ID != ex+"S1" || got[1].ID != ex+"S2" || len(got[0].Props) != 1 || len(got[1].Props) != 2 {
			t.Fatalf("shapes = %+v, want S1 with one property and S2 with two", got)
		}
		if got[0].Props[0].MinCount != 1 || *got[1].Props[0].MaxCount != 3 || got[1].Props[1].Predicate != "test.prop.q" {
			t.Errorf("constraints lost: %+v", got)
		}
	})

	t.Run("pack copy keeps order", func(t *testing.T) {
		p := testPack()
		p.Predicates[1].Projection = &vocabulary.ProjectionSpec{Steps: []vocabulary.ProjectionStep{
			{PredicateIRI: ex + "step2", NodeClassIRI: ex + "N"}, {PredicateIRI: ex + "step1", ValuePredicateIRI: ex + "v"},
		}}
		b, err := Compile(p, parseTestTurtle(t, shapes), testMapping())
		if err != nil {
			t.Fatal(err)
		}
		got := b.Pack()
		want := testPack()
		want.Predicates[0].DataType = vocabulary.DataTypeInt
		want.Predicates[1].Projection = p.Predicates[1].Projection
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Pack() = %+v, want the base pack with test.prop.p filled", got)
		}
		got.Predicates[1].Projection.Steps[0].PredicateIRI = "changed"
		got.Predicates[0].Name = "changed"
		if again := b.Pack(); again.Predicates[1].Projection.Steps[0].PredicateIRI != ex+"step2" || again.Predicates[0].Name != "test.prop.p" {
			t.Error("Pack() returned shared state")
		}
		if p.Predicates[0].DataType != "" {
			t.Error("Compile changed the caller's pack")
		}
	})
}

// contractPack has two entity types with two predicate categories each.
func contractPack() vocabulary.PackSpec {
	return vocabulary.PackSpec{
		Name: "test",
		EntityTypes: []vocabulary.EntityTypeSpec{
			{DomainType: "test.thing", ClassIRI: ex + "C"},
			{DomainType: "test.other", ClassIRI: ex + "E"},
		},
		Predicates: []vocabulary.PredicateSpec{
			{Name: "test.prop.p", IRI: ex + "p"},
			{Name: "test.prop.q", IRI: ex + "q"},
			{Name: "test.meta.r", IRI: ex + "r"},
			{Name: "test.meta.s", IRI: ex + "s"},
		},
	}
}

func contractMapping(contracts ...ContractBinding) Mapping {
	return Mapping{
		EntityTypes: []EntityTypeBinding{
			{ClassIRI: ex + "C", DomainTypes: []string{"test.thing"}},
			{ClassIRI: ex + "E", DomainTypes: []string{"test.other"}},
		},
		Predicates: []PredicateBinding{
			{PathIRI: ex + "p", Name: "test.prop.p"}, {PathIRI: ex + "q", Name: "test.prop.q"},
			{PathIRI: ex + "r", Name: "test.meta.r"}, {PathIRI: ex + "s", Name: "test.meta.s"},
		},
		Contracts: contracts,
	}
}

const contractShapes = `
ex:S sh:targetClass ex:C ;
  sh:property [ sh:path ex:p ; sh:minCount 1 ; sh:maxCount 1 ] ;
  sh:property [ sh:path ex:q ] ;
  sh:property [ sh:path ex:r ] .
ex:T sh:targetClass ex:E .`

// TestContracts is task 5.7 (design D7, rulings Q18, Q19): the pattern,
// explicit modes and birth, conflicting modes, every constrained predicate in
// the contract, sorted names, and Unwritable(). Every contract passes
// SemEngine's ValidateShape.
func TestContracts(t *testing.T) {
	compile := func(t *testing.T, cs ...ContractBinding) (*Bundle, error) {
		t.Helper()
		return Compile(contractPack(), parseTestTurtle(t, contractShapes), contractMapping(cs...))
	}

	b, err := compile(t, ContractBinding{
		DomainType: "test.thing",
		Birth:      []string{"test.meta.r"},
		Groups: []GroupBinding{
			// Unnamed: the name defaults to the category. Two declarations
			// of one name with one mode merge.
			{Mode: contract.ModeAppend, Predicates: []string{"test.prop.q"}},
			{Name: "prop", Mode: contract.ModeAppend, Predicates: []string{"test.prop.p"}},
			{Name: "extra", Mode: contract.ModeReconcile, Predicates: []string{"test.meta.s"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []contract.Contract{{
		Name:          "test.thing",
		EntityPattern: "*.*.*.test.thing.*",
		Groups: []contract.PredicateGroup{
			{Name: "extra", Mode: contract.ModeReconcile, Predicates: []string{"test.meta.s"}},
			// sh:maxCount 1 on test.prop.p does not make the group
			// reconcile, and sh:minCount 1 does not make it birth.
			{Name: "prop", Mode: contract.ModeAppend, Predicates: []string{"test.prop.p", "test.prop.q"}},
		},
		BirthPredicates: []string{"test.meta.r"},
	}}
	if got := b.Contracts(); !reflect.DeepEqual(got, want) {
		t.Errorf("Contracts() =\n%+v\nwant\n%+v", got, want)
	}
	for _, c := range b.Contracts() {
		if err := c.ValidateShape(); err != nil {
			t.Errorf("ValidateShape: %v", err)
		}
	}
	if got := b.Unwritable(); !slices.Equal(got, []string{"test.other"}) {
		t.Errorf("Unwritable() = %v, want [test.other]", got)
	}
	// The pattern selects the type's positions only, by SemEngine's matcher.
	for id, want := range map[string]bool{
		"acme.edge.sensor.test.thing.t1": true, "acme.edge.test.other.thing.t1": false, "acme.edge.sensor.test.other.t1": false,
	} {
		if got, err := semtypes.MatchEntityIDPattern(b.Contracts()[0].EntityPattern, id); err != nil || got != want {
			t.Errorf("match %s = %v, %v; want %v", id, got, err, want)
		}
	}
	// Copies: changing a returned contract changes nothing in the bundle.
	got := b.Contracts()
	got[0].Groups[0].Predicates[0] = "changed"
	got[0].BirthPredicates[0] = "changed"
	if !reflect.DeepEqual(b.Contracts(), want) {
		t.Error("Contracts() returned shared state")
	}

	// Names given, sorted by name; an empty binding leaves its type
	// unwritable.
	b, err = compile(t,
		ContractBinding{DomainType: "test.thing", Name: "zeta",
			Groups: []GroupBinding{{Mode: contract.ModeReconcile, Predicates: []string{"test.prop.q", "test.prop.p", "test.meta.r"}, Name: "all"}}},
		ContractBinding{DomainType: "test.other", Name: "alpha",
			Groups: []GroupBinding{{Mode: contract.ModeAppend, Predicates: []string{"test.meta.s"}}}},
	)
	if err != nil {
		t.Fatal(err)
	}
	cs := b.Contracts()
	if len(cs) != 2 || cs[0].Name != "alpha" || cs[1].Name != "zeta" ||
		!slices.Equal(cs[1].Groups[0].Predicates, []string{"test.meta.r", "test.prop.p", "test.prop.q"}) {
		t.Errorf("contracts = %+v, want alpha then zeta with sorted predicates", cs)
	}
	if len(b.Unwritable()) != 0 {
		t.Errorf("Unwritable() = %v, want none", b.Unwritable())
	}
	b, err = compile(t, ContractBinding{DomainType: "test.thing",
		Groups: []GroupBinding{{Name: "all", Mode: contract.ModeReconcile, Predicates: []string{"test.prop.q", "test.prop.p", "test.meta.r"}}}},
		ContractBinding{DomainType: "test.other"})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(b.Unwritable(), []string{"test.other"}) || len(b.Contracts()) != 1 {
		t.Errorf("an empty binding: contracts %+v, unwritable %v", b.Contracts(), b.Unwritable())
	}

	all := []string{"test.prop.p", "test.prop.q", "test.meta.r"}
	for _, tt := range []struct {
		name      string
		contracts []ContractBinding
		msg       string
	}{
		{"conflicting modes", []ContractBinding{{DomainType: "test.thing", Groups: []GroupBinding{
			{Name: "g", Mode: contract.ModeReconcile, Predicates: all[:2]},
			{Name: "g", Mode: contract.ModeAppend, Predicates: all[2:]},
		}}}, "group g is declared with modes reconcile and append"},
		{"mode missing", []ContractBinding{{DomainType: "test.thing", Groups: []GroupBinding{
			{Name: "g", Predicates: all},
		}}}, `mode "" must be reconcile or append`},
		{"mode unknown", []ContractBinding{{DomainType: "test.thing", Groups: []GroupBinding{
			{Name: "g", Mode: "replace", Predicates: all},
		}}}, `mode "replace" must be reconcile or append`},
		{"default name across categories", []ContractBinding{{DomainType: "test.thing", Groups: []GroupBinding{
			{Mode: contract.ModeReconcile, Predicates: all},
		}}}, "share one category"},
		{"birth and group", []ContractBinding{{DomainType: "test.thing", Birth: []string{"test.prop.p"}, Groups: []GroupBinding{
			{Name: "g", Mode: contract.ModeReconcile, Predicates: all},
		}}}, "test.prop.p is a birth predicate and in group g"},
		{"two groups", []ContractBinding{{DomainType: "test.thing", Groups: []GroupBinding{
			{Name: "g", Mode: contract.ModeReconcile, Predicates: all},
			{Name: "h", Mode: contract.ModeReconcile, Predicates: all[:1]},
		}}}, "test.prop.p is in groups g and h"},
		{"undeclared predicate", []ContractBinding{{DomainType: "test.thing", Groups: []GroupBinding{
			{Name: "g", Mode: contract.ModeReconcile, Predicates: append([]string{"test.prop.none"}, all...)},
		}}}, "test.prop.none, which the compiled pack does not declare"},
		{"constrained predicate outside", []ContractBinding{{DomainType: "test.thing", Groups: []GroupBinding{
			{Name: "g", Mode: contract.ModeReconcile, Predicates: all[:2]},
		}}}, "constrains test.meta.r for test.thing"},
		{"no contract for a constrained type", nil, "constrains test.prop.p for test.thing"},
		{"unmapped type", []ContractBinding{{DomainType: "test.none", Birth: all}}, "no entity_types row lists"},
		{"two bindings", []ContractBinding{{DomainType: "test.thing", Birth: all}, {DomainType: "test.thing", Birth: all}},
			"test.thing has more than one contract"},
		{"same name", []ContractBinding{{DomainType: "test.thing", Name: "x", Birth: all},
			{DomainType: "test.other", Name: "x", Birth: []string{"test.meta.s"}}}, `both named "x"`},
		{"invalid group name", []ContractBinding{{DomainType: "test.thing", Groups: []GroupBinding{
			{Name: "a.b", Mode: contract.ModeReconcile, Predicates: all},
		}}}, "ValidateShape"},
		{"empty group", []ContractBinding{{DomainType: "test.thing", Birth: all, Groups: []GroupBinding{
			{Name: "g", Mode: contract.ModeReconcile},
		}}}, `group "g" has no predicates`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := compile(t, tt.contracts...)
			requireIssue(t, err, "contracts", tt.msg)
		})
	}
}

// entityIDPack declares test.link.target (path ex:t) as entity_id, the
// predicate kind ruling Q17 restricts to cardinality constraints.
func entityIDPack() (vocabulary.PackSpec, Mapping) {
	p := testPack()
	p.Predicates = append(p.Predicates, vocabulary.PredicateSpec{Name: "test.link.target", IRI: ex + "t",
		DataType: vocabulary.DataTypeEntityID})
	m := testMapping()
	m.Predicates = append(m.Predicates, PredicateBinding{PathIRI: ex + "t", Name: "test.link.target"})
	m.Contracts[0].Birth = []string{"test.link.target"}
	return p, m
}

// TestEntityIDCardinalityOnly is issue #13 (ruling Q17, design D8): on a
// path mapped to an entity_id predicate, every constraint other than
// sh:minCount and sh:maxCount fails, one issue per construct naming shape
// and path, whether or not sh:datatype is given. Cardinality alone, with
// severity and messages, compiles.
func TestEntityIDCardinalityOnly(t *testing.T) {
	pack, m := entityIDPack()
	compile := func(constraints string) (*Bundle, error) {
		set, err := ParseTurtle(strings.NewReader(prefix +
			`ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:t ; ` + constraints + ` ] .`))
		if err != nil {
			t.Fatal(err)
		}
		return Compile(pack, set, m)
	}
	if _, err := compile(`sh:minCount 1 ; sh:maxCount 2 ; sh:severity sh:Warning ; sh:message "m"`); err != nil {
		t.Fatalf("cardinality only: %v", err)
	}
	for construct, constraints := range map[string]string{
		"pattern":      `sh:pattern "^urn:"`,
		"in":           `sh:in ( "a" )`,
		"minInclusive": `sh:minInclusive 1`,
		"maxInclusive": `sh:maxInclusive 1`,
		"minExclusive": `sh:minExclusive 1`,
		"maxExclusive": `sh:maxExclusive 1`,
		"datatype":     `sh:datatype xsd:string`,
	} {
		t.Run(construct, func(t *testing.T) {
			b, err := compile(`sh:minCount 1 ; ` + constraints)
			if b != nil {
				t.Error("a bundle was returned")
			}
			is := requireIssue(t, err, sh+construct, "entity_id", "cardinality", "Q17")
			if is.Shape != ex+"S" || is.Path != ex+"t" {
				t.Errorf("issue = %+v, want shape ex:S and path ex:t", is)
			}
			var ce *CompileError
			errors.As(err, &ce)
			if len(ce.Issues) != 1 {
				t.Errorf("want exactly one issue, got:\n%v", err)
			}
		})
	}
}

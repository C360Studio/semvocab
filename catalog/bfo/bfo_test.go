package bfo_test

import (
	"bufio"
	"fmt"
	"maps"
	"os"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/c360studio/semvocab/catalog/bfo"
	"github.com/c360studio/semvocab/catalog/cco"
	"github.com/c360studio/semvocab/internal/gosrc"
	"github.com/c360studio/semvocab/internal/ledger"
	"github.com/c360studio/semvocab/internal/oracle"
)

const semStreamsPin = "8b99efe9c66a4faa4fa509f9f62cc6bad8392128"

// hierarchyAPI is the non-constant exported surface of the package.
var hierarchyAPI = []string{"IsA", "Parents", "SubClassOf"}

// fixture is the SemStreams-pin parity fixture (testdata/semstreams-bfo.txt).
type fixture struct {
	pin    string
	consts map[string]string
	edges  [][2]string // child name, parent name, in source order
}

func loadFixture(t *testing.T) fixture {
	t.Helper()
	f, err := os.Open("testdata/semstreams-bfo.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Error(err)
		}
	}()
	fx := fixture{consts: map[string]string{}}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		switch {
		case fields[0] == "pin" && len(fields) == 2:
			fx.pin = fields[1]
		case fields[0] == "const" && len(fields) == 3:
			fx.consts[fields[1]] = fields[2]
		case fields[0] == "subclass" && len(fields) == 3:
			fx.edges = append(fx.edges, [2]string{fields[1], fields[2]})
		default:
			t.Fatalf("fixture: malformed line %q", line)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if fx.pin != semStreamsPin {
		t.Fatalf("fixture pin %q, want %s", fx.pin, semStreamsPin)
	}
	return fx
}

type inputs struct {
	src     ledger.BFOSource
	entry   ledger.Entry
	fx      fixture
	release *oracle.Graph
	pkg     gosrc.Package
}

func loadInputs(t *testing.T) inputs {
	t.Helper()
	l, err := ledger.Load()
	if err != nil {
		t.Fatal(err)
	}
	src, err := l.BFO()
	if err != nil {
		t.Fatal(err)
	}
	entry, err := l.Entry("bfo")
	if err != nil {
		t.Fatal(err)
	}
	release, err := oracle.Load(oracle.BFO2020)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := gosrc.Load(".")
	if err != nil {
		t.Fatal(err)
	}
	return inputs{src: src, entry: entry, fx: loadFixture(t), release: release, pkg: pkg}
}

// externalAdditions maps each ledger external addition ("BFO_0000141 immaterial
// entity") to its IRI and expected release label.
func externalAdditions(t *testing.T, src ledger.BFOSource) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, term := range src.ExternalAdditions.Terms {
		id, label, ok := strings.Cut(term, " ")
		if !ok {
			t.Fatalf("ledger external addition %q lacks a label", term)
		}
		out[bfo.Namespace+id] = label
	}
	return out
}

// parityProblems checks the package's exported surface against the ledger, the
// SemStreams fixture, and the release file.
func parityProblems(in inputs, additions map[string]string) []string {
	var bad []string
	add := func(format string, args ...any) { bad = append(bad, fmt.Sprintf(format, args...)) }

	classes := slices.Clone(in.src.SymbolsAdmittedClasses)
	relations := in.src.SymbolsAdmittedRelations
	admitted := append(slices.Clone(classes), relations...)

	// Exactly the admitted symbols, the external additions, and the hierarchy API.
	byValue := map[string][]string{}
	for name, v := range in.pkg.Consts {
		byValue[v] = append(byValue[v], name)
	}
	want := append(slices.Clone(admitted), hierarchyAPI...)
	for name := range in.src.ExternalAdditions.Relations {
		want = append(want, name)
	}
	for iri := range additions {
		if names := byValue[iri]; len(names) == 1 {
			want = append(want, names[0])
		} else {
			add("external addition %s: %d exported constants, want 1", iri, len(names))
		}
	}
	sort.Strings(want)
	if !slices.Equal(in.pkg.Exported, want) {
		add("exported identifiers %v, want %v", in.pkg.Exported, want)
	}
	for _, x := range in.entry.Excluded() {
		if slices.Contains(in.pkg.Exported, x) {
			add("excluded source symbol %s is exported", x)
		}
	}

	// IRI parity with the SemStreams pin.
	for _, name := range admitted {
		got, ok := in.pkg.Consts[name]
		if !ok {
			add("admitted symbol %s is not an exported string constant", name)
			continue
		}
		if want, ok := in.fx.consts[name]; !ok || got != want {
			add("%s = %q, SemStreams pin has %q", name, got, want)
		}
	}

	// Declared in the release with the expected rdf:type.
	check := func(name, typ string) {
		iri := in.pkg.Consts[name]
		if !in.release.HasType(iri, typ) {
			add("%s (%s) is not declared as %s in bfo-core.ttl", name, iri, typ)
		}
	}
	for _, name := range classes {
		if name != "Namespace" {
			check(name, oracle.OWLClass)
		}
	}
	for _, name := range relations {
		check(name, oracle.OWLObjectProp)
	}
	for iri, label := range additions {
		if !in.release.HasType(iri, oracle.OWLClass) || !slices.Contains(in.release.Labels(iri), label) {
			add("external addition %s is not an owl:Class labelled %q in bfo-core.ttl", iri, label)
		}
	}
	return append(bad, relationProblems(in)...)
}

// relationIRIs maps every admitted relation (SemStreams-ported and external
// addition) to its exported constant value; names with no exported constant
// are left out.
func relationIRIs(in inputs) map[string]string {
	out := map[string]string{}
	names := slices.Clone(in.src.SymbolsAdmittedRelations)
	for name := range in.src.ExternalAdditions.Relations {
		names = append(names, name)
	}
	for _, name := range names {
		if iri, ok := in.pkg.Consts[name]; ok {
			out[name] = iri
		}
	}
	return out
}

// relationProblems checks each ledger external relation against the exported
// constant and the release, every relation constant for owl:deprecated, and
// every owl:inverseOf an external relation declares for admission.
func relationProblems(in inputs) []string {
	var bad []string
	add := func(format string, args ...any) { bad = append(bad, fmt.Sprintf(format, args...)) }
	admitted := map[string]bool{}
	byName := relationIRIs(in)
	for _, iri := range byName {
		admitted[iri] = true
	}
	for _, name := range slices.Sorted(maps.Keys(byName)) {
		if iri := byName[name]; in.release.Deprecated(iri) {
			add("%s (%s) is owl:deprecated in bfo-core.ttl", name, iri)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(in.src.ExternalAdditions.Relations)) {
		rec := in.src.ExternalAdditions.Relations[name]
		iri, ok := in.pkg.Consts[name]
		if !ok {
			add("external relation %s is not an exported string constant", name)
			continue
		}
		if iri != bfo.Namespace+rec[0] {
			add("%s = %q, ledger records %s", name, iri, rec[0])
		}
		if !in.release.HasType(iri, oracle.OWLObjectProp) {
			add("%s (%s) is not declared as %s in bfo-core.ttl", name, iri, oracle.OWLObjectProp)
		}
		if labels := in.release.Labels(iri); !slices.Equal(labels, []string{rec[1]}) {
			add("%s (%s) has rdfs:label %q in bfo-core.ttl, ledger records %q", name, iri, labels, rec[1])
		}
		for _, inv := range in.release.IRIObjects(iri, oracle.OWLInverseOf) {
			if !admitted[inv] {
				add("%s (%s) declares owl:inverseOf %s, which is not admitted", name, iri, inv)
			}
		}
	}
	return bad
}

// TestCatalogPortParity checks the exported surface against the ledger, the
// SemStreams fixture, and bfo-2020, and restates the reused-name and
// precedence scenarios from ruling R1 on #31.
//
// spec: openspec/changes/admit-bfo-bridge-relations/specs/catalogs-and-packs/spec.md#Requirement: BFO relations are declared in the release
func TestCatalogPortParity(t *testing.T) {
	in := loadInputs(t)
	for _, p := range parityProblems(in, externalAdditions(t, in.src)) {
		t.Error(p)
	}
	if n := len(in.src.ExternalAdditions.Relations); n != 15 {
		t.Errorf("ledger records %d external relations, ruling R1 admits 15", n)
	}
	// Spec scenarios: SemStreams names at their BFO 2020 IRIs, and the
	// direction of temporal precedence.
	for _, c := range []struct{ name, got, local, label string }{
		{"BearerOf", bfo.BearerOf, "BFO_0000196", "bearer of"},
		{"InheresIn", bfo.InheresIn, "BFO_0000197", "inheres in"},
		{"OccupiesTemporalRegion", bfo.OccupiesTemporalRegion, "BFO_0000199", "occupies temporal region"},
		{"Precedes", bfo.Precedes, "BFO_0000063", "precedes"},
		{"PrecededBy", bfo.PrecededBy, "BFO_0000062", "preceded by"},
	} {
		if labels := in.release.Labels(c.got); c.got != bfo.Namespace+c.local || !slices.Equal(labels, []string{c.label}) {
			t.Errorf("bfo.%s = %s labelled %q, want %s labelled %q", c.name, c.got, labels, c.local, c.label)
		}
	}
}

// compiledRelations is the compiled value of each ruling R1 constant. gosrc
// reads bfo.go from disk, so only these values reflect a go test -overlay
// mutant of bfo.go (task mutate:check).
var compiledRelations = map[string]string{
	"BearerOf": bfo.BearerOf, "InheresIn": bfo.InheresIn, "GenericallyDependsOn": bfo.GenericallyDependsOn,
	"IsCarrierOf": bfo.IsCarrierOf, "IsConcretizedBy": bfo.IsConcretizedBy, "Concretizes": bfo.Concretizes,
	"OccupiesTemporalRegion": bfo.OccupiesTemporalRegion, "HasOccurrentPart": bfo.HasOccurrentPart,
	"OccurrentPartOf": bfo.OccurrentPartOf, "ContinuantPartOf": bfo.ContinuantPartOf,
	"HasContinuantPart": bfo.HasContinuantPart, "HasMemberPart": bfo.HasMemberPart, "MemberPartOf": bfo.MemberPartOf,
	"PrecededBy": bfo.PrecededBy, "Precedes": bfo.Precedes,
}

// TestRelationConstantsMatchLedger requires the compiled value of every
// ruling R1 constant to be the ledger's bfo-2020 IRI, declared in the release
// with the ledger's label, so a mutant of bfo.go is seen.
//
// spec: openspec/changes/admit-bfo-bridge-relations/specs/catalogs-and-packs/spec.md#Requirement: BFO relations are declared in the release
func TestRelationConstantsMatchLedger(t *testing.T) {
	in := loadInputs(t)
	rels := in.src.ExternalAdditions.Relations
	if got, want := slices.Sorted(maps.Keys(compiledRelations)), slices.Sorted(maps.Keys(rels)); !slices.Equal(got, want) {
		t.Fatalf("compiled R1 constants %v, ledger external relations %v", got, want)
	}
	for _, name := range slices.Sorted(maps.Keys(rels)) {
		iri, rec := compiledRelations[name], rels[name]
		if iri != bfo.Namespace+rec[0] {
			t.Errorf("bfo.%s = %s, ledger records %s", name, iri, rec[0])
		}
		if !in.release.HasType(iri, oracle.OWLObjectProp) || !slices.Equal(in.release.Labels(iri), []string{rec[1]}) {
			t.Errorf("bfo.%s (%s) is not an owl:ObjectProperty labelled %q in bfo-core.ttl", name, iri, rec[1])
		}
	}
}

// TestCatalogPortParitySensitivity seeds faults into the exported surface, the
// ledger, and the release graph, and requires parityProblems to name each.
//
// spec: openspec/changes/admit-bfo-bridge-relations/specs/catalogs-and-packs/spec.md#Requirement: BFO relations are declared in the release
func TestCatalogPortParitySensitivity(t *testing.T) {
	drop := func(p *gosrc.Package, name string) {
		delete(p.Consts, name)
		p.Exported = slices.DeleteFunc(p.Exported, func(s string) bool { return s == name })
	}
	tests := []struct {
		name   string
		mutate func(*testing.T, *inputs)
		want   string
	}{
		{"altered IRI", func(_ *testing.T, in *inputs) { in.pkg.Consts["Site"] = bfo.Namespace + "BFO_0000030" }, "Site = "},
		{"excluded symbol exported", func(_ *testing.T, in *inputs) {
			in.pkg.Exported = append(in.pkg.Exported, "PrecedesTemporally")
			sort.Strings(in.pkg.Exported)
		}, "excluded source symbol PrecedesTemporally"},
		{"relation typed as class", func(_ *testing.T, in *inputs) { in.pkg.Consts["OccursIn"] = bfo.Process }, "OccursIn ("},
		{"BearerOf at BFO 2.0 IRI", func(_ *testing.T, in *inputs) {
			in.pkg.Consts["BearerOf"] = bfo.Namespace + "BFO_0000053"
		}, `BearerOf = "` + bfo.Namespace + `BFO_0000053", ledger records BFO_0000196`},
		{"Precedes and PrecededBy swapped", func(_ *testing.T, in *inputs) {
			in.pkg.Consts["Precedes"], in.pkg.Consts["PrecededBy"] = in.pkg.Consts["PrecededBy"], in.pkg.Consts["Precedes"]
		}, `Precedes = "` + bfo.Namespace + `BFO_0000062", ledger records BFO_0000063`},
		{"wrong label", func(_ *testing.T, in *inputs) {
			in.src.ExternalAdditions.Relations["IsCarrierOf"] = [2]string{"BFO_0000101", "carrier of"}
		}, `ledger records "carrier of"`},
		{"deprecated term", func(t *testing.T, in *inputs) {
			merged, err := oracle.LoadMerged(oracle.BFO2020, oracle.CCO22)
			if err != nil {
				t.Fatal(err)
			}
			in.release = merged
			// ont00001769 carries owl:deprecated true in CCO v2.2.
			in.pkg.Consts["Concretizes"] = cco.Namespace + "ont00001769"
		}, "Concretizes (" + cco.Namespace + "ont00001769) is owl:deprecated"},
		{"InheresIn dropped", func(_ *testing.T, in *inputs) {
			drop(&in.pkg, "InheresIn")
			delete(in.src.ExternalAdditions.Relations, "InheresIn")
		}, "BearerOf (" + bfo.Namespace + "BFO_0000196) declares owl:inverseOf " + bfo.Namespace + "BFO_0000197, which is not admitted"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := loadInputs(t)
			in.src.ExternalAdditions.Relations = maps.Clone(in.src.ExternalAdditions.Relations)
			tt.mutate(t, &in)
			got := strings.Join(parityProblems(in, externalAdditions(t, in.src)), "\n")
			if !strings.Contains(got, tt.want) {
				t.Fatalf("seeded fault not named; want %q in:\n%s", tt.want, got)
			}
		})
	}
}

// classIRIs returns every class constant: admitted classes and additions.
func classIRIs(t *testing.T, in inputs) []string {
	t.Helper()
	var out []string
	for _, name := range in.src.SymbolsAdmittedClasses {
		if name != "Namespace" {
			out = append(out, in.pkg.Consts[name])
		}
	}
	for iri := range externalAdditions(t, in.src) {
		out = append(out, iri)
	}
	sort.Strings(out)
	return out
}

// hierarchyProblems compares a Parents/IsA implementation with the release.
func hierarchyProblems(classes []string, release *oracle.Graph, parents func(string) []string,
	isA func(string, string) bool) []string {
	var bad []string
	for _, c := range classes {
		want := release.Closure(c)
		if got := parents(c); !slices.Equal(got, want) {
			bad = append(bad, fmt.Sprintf("Parents(%s) = %v, release closure %v", c, got, want))
		}
		if !isA(c, c) {
			bad = append(bad, fmt.Sprintf("IsA(%s, itself) = false", c))
		}
		for _, other := range classes {
			if other == c {
				continue
			}
			if got, want := isA(c, other), slices.Contains(want, other); got != want {
				bad = append(bad, fmt.Sprintf("IsA(%s, %s) = %v, release says %v", c, other, got, want))
			}
		}
	}
	return bad
}

func TestCatalogHierarchyMatchesRelease(t *testing.T) {
	in := loadInputs(t)
	classes := classIRIs(t, in)
	for _, p := range hierarchyProblems(classes, in.release, bfo.Parents, bfo.IsA) {
		t.Error(p)
	}
	for _, c := range classes {
		direct := in.release.DirectSuperclasses(c)
		got, ok := bfo.SubClassOf(c)
		switch {
		case len(direct) == 0 && ok:
			t.Errorf("SubClassOf(%s) = %s, release has no parent", c, got)
		case len(direct) == 1 && (!ok || got != direct[0]):
			t.Errorf("SubClassOf(%s) = %s, %v; release parent %s", c, got, ok, direct[0])
		case len(direct) > 1:
			t.Errorf("release asserts %d named parents for %s; the single-parent API cannot hold them", len(direct), c)
		}
	}

	// Spec scenario: Site lies under immaterial entity, as BFO 2020 asserts.
	if !slices.Contains(bfo.Parents(bfo.Site), bfo.Namespace+"BFO_0000141") {
		t.Errorf("Parents(Site) = %v, want BFO_0000141 included", bfo.Parents(bfo.Site))
	}

	// The ledger's recorded SemStreams divergences, and only those, differ in
	// their direct parent; the SemStreams parent remains an ancestor.
	var divergent []string
	for _, e := range in.fx.edges {
		child, parent := in.fx.consts[e[0]], in.fx.consts[e[1]]
		if got, _ := bfo.SubClassOf(child); got != parent {
			divergent = append(divergent, e[0])
			if !bfo.IsA(child, parent) {
				t.Errorf("%s: SemStreams parent %s is not an ancestor in the release", e[0], e[1])
			}
		}
	}
	if want := []string{"Site", "SpatialRegion"}; !slices.Equal(divergent, want) {
		t.Errorf("SemStreams hierarchy divergences %v, want %v", divergent, want)
	}
	for _, name := range divergent {
		if !strings.Contains(in.entry.HierarchyDivergenceFromSource, name) {
			t.Errorf("ledger hierarchy_divergence_from_source does not record %s", name)
		}
	}
}

// TestCatalogHierarchySensitivity runs the hierarchy check against SemStreams'
// own map, which the release contradicts for Site and SpatialRegion, and
// requires the check to name the divergence.
func TestCatalogHierarchySensitivity(t *testing.T) {
	in := loadInputs(t)
	ss := map[string]string{}
	for _, e := range in.fx.edges {
		ss[in.fx.consts[e[0]]] = in.fx.consts[e[1]]
	}
	parents := func(iri string) []string {
		var out []string
		for p, ok := ss[iri]; ok; p, ok = ss[p] {
			out = append(out, p)
		}
		return out
	}
	isA := func(c, p string) bool { return c == p || slices.Contains(parents(c), p) }
	got := strings.Join(hierarchyProblems(classIRIs(t, in), in.release, parents, isA), "\n")
	if !strings.Contains(got, "Parents("+bfo.Site+")") {
		t.Fatalf("SemStreams hierarchy not flagged for Site:\n%s", got)
	}
}

// inverseKey names an owl:inverseOf pair independently of its direction.
func inverseKey(a, b string) string {
	if b < a {
		a, b = b, a
	}
	return a + " " + b
}

// axiomProblems compares the ledger's relation_axioms, both ways, with the
// owl:inverseOf and rdfs:subPropertyOf triples bfo-core.ttl asserts between
// two admitted relations. Axioms reaching outside the admitted set are not
// expected in the ledger.
func axiomProblems(in inputs) []string {
	var bad []string
	add := func(format string, args ...any) { bad = append(bad, fmt.Sprintf(format, args...)) }
	byName := relationIRIs(in)
	byIRI := map[string]string{}
	for name, iri := range byName {
		byIRI[iri] = name
	}

	releaseInv := map[string]bool{}
	releaseSub := map[string]string{}
	for _, name := range slices.Sorted(maps.Keys(byName)) {
		for _, o := range in.release.IRIObjects(byName[name], oracle.OWLInverseOf) {
			if other, ok := byIRI[o]; ok {
				releaseInv[inverseKey(name, other)] = true
			}
		}
		for _, o := range in.release.IRIObjects(byName[name], oracle.RDFSSubPropertyOf) {
			if super, ok := byIRI[o]; ok {
				releaseSub[name] = super
			}
		}
	}

	ledgerInv := map[string]bool{}
	for _, p := range in.src.RelationAxioms.InversePairs {
		for _, n := range p {
			if _, ok := byName[n]; !ok {
				add("ledger inverse pair %v names %s, which is not an admitted relation", p, n)
			}
		}
		ledgerInv[inverseKey(p[0], p[1])] = true
	}
	for _, k := range slices.Sorted(maps.Keys(ledgerInv)) {
		if !releaseInv[k] {
			add("ledger inverse pair %s is not asserted by owl:inverseOf in bfo-core.ttl", k)
		}
	}
	for _, k := range slices.Sorted(maps.Keys(releaseInv)) {
		if !ledgerInv[k] {
			add("bfo-core.ttl asserts owl:inverseOf between %s; ledger relation_axioms.inverse_pairs omits it", k)
		}
	}

	ledgerSub := in.src.RelationAxioms.SubPropertyOf
	for _, sub := range slices.Sorted(maps.Keys(ledgerSub)) {
		if got, ok := releaseSub[sub]; !ok || got != ledgerSub[sub] {
			add("ledger records %s rdfs:subPropertyOf %s; bfo-core.ttl does not assert it", sub, ledgerSub[sub])
		}
	}
	for _, sub := range slices.Sorted(maps.Keys(releaseSub)) {
		if got, ok := ledgerSub[sub]; !ok || got != releaseSub[sub] {
			add("bfo-core.ttl asserts %s rdfs:subPropertyOf %s; ledger relation_axioms.sub_property_of omits it",
				sub, releaseSub[sub])
		}
	}
	return bad
}

// TestRelationAxiomsMatchRelease requires the ledger's relation_axioms to equal
// the release's axioms between admitted relations, and restates the spec
// scenarios from bfo-core.ttl lines 284, 308, 377, and 531.
//
// spec: openspec/changes/admit-bfo-bridge-relations/specs/catalogs-and-packs/spec.md#Requirement: BFO relation axioms follow the release
func TestRelationAxiomsMatchRelease(t *testing.T) {
	in := loadInputs(t)
	for _, p := range axiomProblems(in) {
		t.Error(p)
	}
	ax := in.src.RelationAxioms
	for sub, super := range map[string]string{
		"HasMemberPart": "HasContinuantPart", "HasTemporalPart": "HasOccurrentPart", "TemporalPartOf": "OccurrentPartOf",
	} {
		if ax.SubPropertyOf[sub] != super {
			t.Errorf("relation_axioms.sub_property_of[%s] = %q, want %s", sub, ax.SubPropertyOf[sub], super)
		}
	}
	if !slices.ContainsFunc(ax.InversePairs, func(p [2]string) bool { return inverseKey(p[0], p[1]) == "BearerOf InheresIn" }) {
		t.Errorf("relation_axioms.inverse_pairs = %v, want BearerOf paired with InheresIn", ax.InversePairs)
	}
	if got := in.release.IRIObjects(bfo.InheresIn, oracle.OWLInverseOf); len(got) != 0 {
		t.Errorf("bfo-core.ttl declares owl:inverseOf %v on InheresIn; the scenario assumes one side only", got)
	}
}

// TestRelationAxiomsMatchReleaseSensitivity seeds faults into a copy of the
// ledger's relation_axioms and requires axiomProblems to name each, and a
// pair recorded in the reverse of the release's direction to pass.
//
// spec: openspec/changes/admit-bfo-bridge-relations/specs/catalogs-and-packs/spec.md#Requirement: BFO relation axioms follow the release
func TestRelationAxiomsMatchReleaseSensitivity(t *testing.T) {
	isBearer := func(p [2]string) bool { return p[0] == "BearerOf" }
	tests := []struct {
		name   string
		mutate func(*inputs)
		want   string // "" requires no problem
	}{
		{"dropped pair", func(in *inputs) {
			in.src.RelationAxioms.InversePairs = slices.DeleteFunc(in.src.RelationAxioms.InversePairs, isBearer)
		}, "asserts owl:inverseOf between BearerOf InheresIn; ledger relation_axioms.inverse_pairs omits it"},
		{"invented pair", func(in *inputs) {
			in.src.RelationAxioms.InversePairs = append(in.src.RelationAxioms.InversePairs, [2]string{"Precedes", "OccursIn"})
		}, "ledger inverse pair OccursIn Precedes is not asserted"},
		{"reversed subproperty edge", func(in *inputs) {
			delete(in.src.RelationAxioms.SubPropertyOf, "HasMemberPart")
			in.src.RelationAxioms.SubPropertyOf["HasContinuantPart"] = "HasMemberPart"
		}, "ledger records HasContinuantPart rdfs:subPropertyOf HasMemberPart; bfo-core.ttl does not assert it"},
		{"pair recorded in reverse direction", func(in *inputs) {
			i := slices.IndexFunc(in.src.RelationAxioms.InversePairs, isBearer)
			in.src.RelationAxioms.InversePairs[i] = [2]string{"InheresIn", "BearerOf"}
		}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := loadInputs(t)
			in.src.RelationAxioms.InversePairs = slices.Clone(in.src.RelationAxioms.InversePairs)
			in.src.RelationAxioms.SubPropertyOf = maps.Clone(in.src.RelationAxioms.SubPropertyOf)
			tt.mutate(&in)
			problems := axiomProblems(in)
			got := strings.Join(problems, "\n")
			if tt.want == "" {
				if len(problems) != 0 {
					t.Fatalf("want no problem, got:\n%s", got)
				}
				return
			}
			if !strings.Contains(got, tt.want) {
				t.Fatalf("seeded fault not named; want %q in:\n%s", tt.want, got)
			}
		})
	}
}

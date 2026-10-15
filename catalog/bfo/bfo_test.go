package bfo_test

import (
	"bufio"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/c360studio/semvocab/catalog/bfo"
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
	return bad
}

func TestCatalogPortParity(t *testing.T) {
	in := loadInputs(t)
	for _, p := range parityProblems(in, externalAdditions(t, in.src)) {
		t.Error(p)
	}
}

// TestCatalogPortParitySensitivity seeds one wrong IRI, one excluded symbol,
// and one undeclared term, and requires parityProblems to name each.
func TestCatalogPortParitySensitivity(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*gosrc.Package)
		want   string
	}{
		{"altered IRI", func(p *gosrc.Package) { p.Consts["Site"] = bfo.Namespace + "BFO_0000030" }, "Site = "},
		{"excluded symbol exported", func(p *gosrc.Package) {
			p.Exported = append(p.Exported, "PrecedesTemporally")
			sort.Strings(p.Exported)
		}, "excluded source symbol PrecedesTemporally"},
		{"relation typed as class", func(p *gosrc.Package) { p.Consts["OccursIn"] = bfo.Process }, "OccursIn ("},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := loadInputs(t)
			tt.mutate(&in.pkg)
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

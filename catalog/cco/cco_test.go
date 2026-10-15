package cco_test

import (
	"fmt"
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

// cco1xNamespace is the CCO 1.x namespace SemStreams used; no IRI under it is
// admitted (ledger, cco-2.2 divergence).
const cco1xNamespace = "http://www.ontologyrepository.com/CommonCoreOntologies/"

type inputs struct {
	src       ledger.CCOSource
	entry     ledger.Entry
	namespace string
	release   *oracle.Graph
	pkg       gosrc.Package
}

func loadInputs(t *testing.T) inputs {
	t.Helper()
	l, err := ledger.Load()
	if err != nil {
		t.Fatal(err)
	}
	src, err := l.CCO()
	if err != nil {
		t.Fatal(err)
	}
	entry, err := l.Entry("cco")
	if err != nil {
		t.Fatal(err)
	}
	release, err := oracle.Load(oracle.CCO22)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := gosrc.Load(".")
	if err != nil {
		t.Fatal(err)
	}
	return inputs{src: src, entry: entry, namespace: l.Pins.Ontologies["cco-2.2"].Namespace, release: release, pkg: pkg}
}

func parityProblems(in inputs) []string {
	var bad []string
	add := func(format string, args ...any) { bad = append(bad, fmt.Sprintf(format, args...)) }

	want := []string{"IsA", "Namespace", "Parents", "SubClassOf"}
	for name := range in.src.AdmittedClasses {
		want = append(want, name)
	}
	for name := range in.src.AdmittedProperties {
		want = append(want, name)
	}
	sort.Strings(want)
	if !slices.Equal(in.pkg.Exported, want) {
		add("exported identifiers %v, want %v", in.pkg.Exported, want)
	}
	excluded := in.entry.Excluded()
	for old := range in.src.RenamedFromSemStreams {
		excluded = append(excluded, old)
	}
	for _, x := range excluded {
		if slices.Contains(in.pkg.Exported, x) {
			add("excluded or renamed source symbol %s is exported", x)
		}
	}
	if got := in.pkg.Consts["Namespace"]; got != in.namespace {
		add("Namespace = %q, ledger pin namespace %q", got, in.namespace)
	}
	for name, v := range in.pkg.Consts {
		if strings.HasPrefix(v, cco1xNamespace) {
			add("%s carries a CCO 1.x IRI %s", name, v)
		}
	}

	check := func(name, local, label, typ string) {
		iri, ok := in.pkg.Consts[name]
		switch {
		case !ok:
			add("admitted term %s is not an exported string constant", name)
			return
		case iri != in.namespace+local:
			add("%s = %q, ledger records %s%s", name, iri, in.namespace, local)
		}
		if !in.release.HasType(iri, typ) {
			add("%s (%s) is not declared as %s in the CCO v2.2 file", name, iri, typ)
		}
		if in.release.Deprecated(iri) {
			add("%s (%s) is owl:deprecated in the CCO v2.2 file", name, iri)
		}
		if labels := in.release.Labels(iri); !slices.Equal(labels, []string{label}) {
			add("%s (%s) has rdfs:label %q in the CCO v2.2 file, ledger records %q", name, iri, labels, label)
		}
	}
	for name, v := range in.src.AdmittedClasses {
		check(name, v[0], v[1], oracle.OWLClass)
	}
	for name, v := range in.src.AdmittedProperties {
		check(name, v[0], v[1], oracle.OWLObjectProp)
	}
	return bad
}

func TestCatalogPortParity(t *testing.T) {
	for _, p := range parityProblems(loadInputs(t)) {
		t.Error(p)
	}
}

// TestCatalogPortParitySensitivity seeds faults into the exported surface and
// the ledger table and requires parityProblems to name each.
func TestCatalogPortParitySensitivity(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*inputs)
		want   string
	}{
		{"altered IRI", func(in *inputs) { in.pkg.Consts["Vehicle"] = cco.Namespace + "ont00000714" }, "Vehicle = "},
		{"deprecated term", func(in *inputs) {
			in.pkg.Consts["IsAbout"] = cco.Namespace + "ont00001769"
			in.src.AdmittedProperties["IsAbout"] = [2]string{"ont00001769", "has decimal value"}
		}, "is owl:deprecated"},
		{"wrong label", func(in *inputs) {
			in.src.AdmittedClasses["MaterialArtifact"] = [2]string{"ont00000995", "Artifact"}
		}, `ledger records "Artifact"`},
		{"renamed symbol kept", func(in *inputs) {
			in.pkg.Exported = append(in.pkg.Exported, "Document")
			sort.Strings(in.pkg.Exported)
		}, "renamed source symbol Document"},
		{"CCO 1.x IRI", func(in *inputs) { in.pkg.Consts["Vehicle"] = cco1xNamespace + "Vehicle" }, "CCO 1.x IRI"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := loadInputs(t)
			// The ledger maps are shared with no other test; copy before mutating.
			in.src.AdmittedClasses = cloneMap(in.src.AdmittedClasses)
			in.src.AdmittedProperties = cloneMap(in.src.AdmittedProperties)
			tt.mutate(&in)
			got := strings.Join(parityProblems(in), "\n")
			if !strings.Contains(got, tt.want) {
				t.Fatalf("seeded fault not named; want %q in:\n%s", tt.want, got)
			}
		})
	}
}

func cloneMap(m map[string][2]string) map[string][2]string {
	out := make(map[string][2]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func classIRIs(in inputs) []string {
	var out []string
	for _, v := range in.src.AdmittedClasses {
		out = append(out, in.namespace+v[0])
	}
	sort.Strings(out)
	return out
}

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
		for _, other := range append(slices.Clone(classes), want...) {
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
	for _, p := range hierarchyProblems(classIRIs(in), in.release, cco.Parents, cco.IsA) {
		t.Error(p)
	}
	for _, c := range classIRIs(in) {
		direct := in.release.DirectSuperclasses(c)
		got, ok := cco.SubClassOf(c)
		if len(direct) != 1 || !ok || got != direct[0] {
			t.Errorf("SubClassOf(%s) = %s, %v; release direct parents %v", c, got, ok, direct)
		}
	}

	// Design D5's closures, as written in the Phase 0 design.
	o := func(id string) string { return bfo.Namespace + "BFO_" + id }
	c := func(id string) string { return cco.Namespace + id }
	d5 := map[string][]string{
		cco.Vehicle:          {c("ont00000995"), o("0000040"), o("0000004"), o("0000002"), o("0000001")},
		cco.ActOfObservation: {c("ont00000228"), c("ont00000005"), o("0000015"), o("0000003"), o("0000001")},
		cco.Altitude: {c("ont00000967"), c("ont00001367"), c("ont00001202"), o("0000019"), o("0000020"),
			o("0000002"), o("0000001")},
		cco.MeasurementInformationContentEntity: {c("ont00000853"), c("ont00000958"), o("0000031"), o("0000002"),
			o("0000001")},
		cco.InformationBearingEntity: {o("0000030"), o("0000040"), o("0000004"), o("0000002"), o("0000001")},
	}
	for class, want := range d5 {
		if got := cco.Parents(class); !slices.Equal(got, want) {
			t.Errorf("Parents(%s) = %v, design D5 %v", class, got, want)
		}
	}
	if cco.IsA(cco.Vehicle, bfo.Object) {
		t.Error("IsA(Vehicle, BFO_0000030) = true; CCO v2.2 does not entail it (ruling Q2 makes it DIEKM-local)")
	}
}

// TestCatalogHierarchySensitivity drops one ancestor from Parents and
// requires the hierarchy check to name it.
func TestCatalogHierarchySensitivity(t *testing.T) {
	in := loadInputs(t)
	short := func(iri string) []string {
		p := cco.Parents(iri)
		if iri == cco.Vehicle {
			p = slices.DeleteFunc(slices.Clone(p), func(s string) bool { return s == bfo.IndependentContinuant })
		}
		return p
	}
	got := strings.Join(hierarchyProblems(classIRIs(in), in.release, short, cco.IsA), "\n")
	if !strings.Contains(got, "Parents("+cco.Vehicle+")") {
		t.Fatalf("missing ancestor not flagged:\n%s", got)
	}
}

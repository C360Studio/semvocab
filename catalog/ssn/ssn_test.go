package ssn_test

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/c360studio/semvocab/catalog/sosa"
	"github.com/c360studio/semvocab/catalog/ssn"
	"github.com/c360studio/semvocab/internal/gosrc"
	"github.com/c360studio/semvocab/internal/ledger"
	"github.com/c360studio/semvocab/internal/oracle"
)

type inputs struct {
	src         ledger.SSNSource
	sosaSrc     ledger.SOSASource
	entry       ledger.Entry
	sosaEntry   ledger.Entry
	sosaNS      string
	ssnNS       string
	systemsNS   string
	ssnGraph    *oracle.Graph
	systemGraph *oracle.Graph
	merged      *oracle.Graph
	pkg         gosrc.Package
}

func loadInputs(t *testing.T) inputs {
	t.Helper()
	l, err := ledger.Load()
	if err != nil {
		t.Fatal(err)
	}
	src, err := l.SSN()
	if err != nil {
		t.Fatal(err)
	}
	sosaSrc, err := l.SOSA()
	if err != nil {
		t.Fatal(err)
	}
	entry, err := l.Entry("ssn")
	if err != nil {
		t.Fatal(err)
	}
	sosaEntry, err := l.Entry("sosa")
	if err != nil {
		t.Fatal(err)
	}
	ssnGraph, err := oracle.Load(oracle.SSN2017)
	if err != nil {
		t.Fatal(err)
	}
	systemGraph, err := oracle.Load(oracle.SSNSystem2017)
	if err != nil {
		t.Fatal(err)
	}
	merged, err := oracle.LoadMerged(oracle.SOSA2017, oracle.SSN2017, oracle.SSNSystem2017)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := gosrc.Load(".")
	if err != nil {
		t.Fatal(err)
	}
	return inputs{
		src:         src,
		sosaSrc:     sosaSrc,
		entry:       entry,
		sosaEntry:   sosaEntry,
		sosaNS:      l.Pins.Ontologies["sosa-2017"].Namespace,
		ssnNS:       l.Pins.Ontologies["ssn-2017"].Namespace,
		systemsNS:   l.Pins.Ontologies["ssn-system-2017"].Namespace,
		ssnGraph:    ssnGraph,
		systemGraph: systemGraph,
		merged:      merged,
		pkg:         pkg,
	}
}

func parityProblems(in inputs) []string {
	var bad []string
	add := func(format string, args ...any) { bad = append(bad, fmt.Sprintf(format, args...)) }

	want := []string{"DirectParents", "IsA", "Namespace", "Parents", "SystemsNamespace"}
	for name := range in.src.AdmittedClasses {
		want = append(want, name)
	}
	for name := range in.src.AdmittedObjectProperties {
		want = append(want, name)
	}
	for name := range in.src.AdmittedSystemClasses {
		want = append(want, name)
	}
	for name := range in.src.AdmittedSystemObjectProperties {
		want = append(want, name)
	}
	sort.Strings(want)
	if len(want) != 57 {
		add("expected 57 admitted identifiers in ssn, got %d", len(want))
	}
	if !slices.Equal(in.pkg.Exported, want) {
		add("exported identifiers %v, want %v", in.pkg.Exported, want)
	}

	excluded := append(in.entry.Excluded(), in.sosaEntry.Excluded()...)
	excluded = append(excluded, "HasResultingSample", "IsSamplingResultOf", "SubClassOf")
	for _, x := range excluded {
		if slices.Contains(in.pkg.Exported, x) {
			add("excluded symbol %s is exported", x)
		}
	}

	if got := in.pkg.Consts["Namespace"]; got != in.ssnNS {
		add("Namespace = %q, ledger pin namespace %q", got, in.ssnNS)
	}
	if got := in.pkg.Consts["SystemsNamespace"]; got != in.systemsNS {
		add("SystemsNamespace = %q, ledger pin namespace %q", got, in.systemsNS)
	}

	check := func(name, ns, local, label, typ string, g *oracle.Graph, fileLabel string) {
		iri, ok := in.pkg.Consts[name]
		switch {
		case !ok:
			add("admitted term %s is not an exported string constant", name)
			return
		case iri != ns+local:
			add("%s = %q, ledger records %s%s", name, iri, ns, local)
		}
		if !g.HasType(iri, typ) {
			add("%s (%s) is not declared as %s in %s", name, iri, typ, fileLabel)
		}
		if g.Deprecated(iri) {
			add("%s (%s) is owl:deprecated in %s", name, iri, fileLabel)
		}
		if labels := g.Labels(iri); !slices.Equal(labels, []string{label}) {
			add("%s (%s) has rdfs:label %q in %s, ledger records %q", name, iri, labels, fileLabel, label)
		}
	}
	for name, v := range in.src.AdmittedClasses {
		check(name, in.ssnNS, v[0], v[1], oracle.OWLClass, in.ssnGraph, "ssn.ttl")
	}
	for name, v := range in.src.AdmittedObjectProperties {
		check(name, in.ssnNS, v[0], v[1], oracle.OWLObjectProp, in.ssnGraph, "ssn.ttl")
	}
	for name, v := range in.src.AdmittedSystemClasses {
		check(name, in.systemsNS, v[0], v[1], oracle.OWLClass, in.systemGraph, "ssn-system.ttl")
	}
	for name, v := range in.src.AdmittedSystemObjectProperties {
		check(name, in.systemsNS, v[0], v[1], oracle.OWLObjectProp, in.systemGraph, "ssn-system.ttl")
	}
	return bad
}

func TestCatalogParity(t *testing.T) {
	for _, p := range parityProblems(loadInputs(t)) {
		t.Error(p)
	}
}

// TestCatalogParitySensitivity seeds faults into the exported surface and the
// ledger table and requires parityProblems to name each.
func TestCatalogParitySensitivity(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*inputs)
		want   string
	}{
		{"altered ssn IRI", func(in *inputs) {
			in.pkg.Consts["System"] = ssn.Namespace + "Device"
		}, "System = "},
		{"altered ssn-system IRI", func(in *inputs) {
			in.pkg.Consts["Accuracy"] = ssn.Namespace + "Accuracy"
		}, "Accuracy = "},
		{"wrong label", func(in *inputs) {
			in.src.AdmittedSystemClasses["OperatingPowerRange"] = [2]string{"OperatingPowerRange", "Power Range"}
		}, `ledger records "Power Range"`},
		{"excluded symbol", func(in *inputs) {
			in.pkg.Exported = append(in.pkg.Exported, "SamplingFeature")
			sort.Strings(in.pkg.Exported)
		}, "excluded symbol SamplingFeature"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := loadInputs(t)
			in.src.AdmittedClasses = cloneMap(in.src.AdmittedClasses)
			in.src.AdmittedObjectProperties = cloneMap(in.src.AdmittedObjectProperties)
			in.src.AdmittedSystemClasses = cloneMap(in.src.AdmittedSystemClasses)
			in.src.AdmittedSystemObjectProperties = cloneMap(in.src.AdmittedSystemObjectProperties)
			in.pkg.Consts = cloneStringMap(in.pkg.Consts)
			in.pkg.Exported = slices.Clone(in.pkg.Exported)
			tt.mutate(&in)
			got := strings.Join(parityProblems(in), "\n")
			if !strings.Contains(got, tt.want) {
				t.Fatalf("seeded fault not named; want %q in:\n%s", tt.want, got)
			}
		})
	}
}

func allClassIRIs(in inputs) []string {
	var out []string
	for _, v := range in.sosaSrc.AdmittedClasses {
		out = append(out, in.sosaNS+v[0])
	}
	for _, v := range in.src.AdmittedClasses {
		out = append(out, in.ssnNS+v[0])
	}
	for _, v := range in.src.AdmittedSystemClasses {
		out = append(out, in.systemsNS+v[0])
	}
	sort.Strings(out)
	return out
}

func hierarchyProblems(
	classes []string,
	merged *oracle.Graph,
	directParents func(string) []string,
	parents func(string) []string,
	isA func(string, string) bool,
) []string {
	var bad []string
	for _, c := range classes {
		wantDirect := merged.DirectSuperclasses(c)
		if got := directParents(c); !slices.Equal(got, wantDirect) {
			bad = append(bad, fmt.Sprintf("DirectParents(%s) = %v, merged direct superclasses %v", c, got, wantDirect))
		}
		wantClosure := merged.Closure(c)
		if got := parents(c); !slices.Equal(got, wantClosure) {
			bad = append(bad, fmt.Sprintf("Parents(%s) = %v, merged closure %v", c, got, wantClosure))
		}
		if !isA(c, c) {
			bad = append(bad, fmt.Sprintf("IsA(%s, itself) = false", c))
		}
		for _, other := range append(slices.Clone(classes), wantClosure...) {
			if other == c {
				continue
			}
			if got, want := isA(c, other), slices.Contains(wantClosure, other); got != want {
				bad = append(bad, fmt.Sprintf("IsA(%s, %s) = %v, merged closure says %v", c, other, got, want))
			}
		}
	}
	return bad
}

func TestCatalogHierarchyMatchesRelease(t *testing.T) {
	in := loadInputs(t)
	classes := allClassIRIs(in)
	if len(classes) != 42 {
		t.Fatalf("expected 42 classes across sosa + ssn + ssn-system, got %d", len(classes))
	}
	for _, p := range hierarchyProblems(classes, in.merged, ssn.DirectParents, ssn.Parents, ssn.IsA) {
		t.Error(p)
	}

	// Spec scenario: multi-parent SOSA/SSN class sosa:Sample -> [sosa:FeatureOfInterest, sosa:Result].
	gotSample := ssn.DirectParents(sosa.Sample)
	wantSample := []string{sosa.FeatureOfInterest, sosa.Result}
	if !slices.Equal(gotSample, wantSample) {
		t.Fatalf("DirectParents(sosa.Sample) = %v, want %v", gotSample, wantSample)
	}
	if !ssn.IsA(sosa.Sample, sosa.FeatureOfInterest) || !ssn.IsA(sosa.Sample, sosa.Result) {
		t.Error("IsA(sosa.Sample, FeatureOfInterest/Result) = false, want true")
	}

	// Defensive copy on DirectParents so caller mutation never corrupts internal state.
	gotSample[0] = "http://example.org/corrupted"
	if after := ssn.DirectParents(sosa.Sample); !slices.Equal(after, wantSample) {
		t.Errorf("caller slice mutation leaked into DirectParents: %v", after)
	}
}

// TestCatalogHierarchySensitivity drops one parent from DirectParents and one
// ancestor from Parents and requires hierarchyProblems to name each.
func TestCatalogHierarchySensitivity(t *testing.T) {
	in := loadInputs(t)
	classes := allClassIRIs(in)

	t.Run("missing direct parent", func(t *testing.T) {
		shortDirect := func(iri string) []string {
			p := ssn.DirectParents(iri)
			if iri == sosa.Sample {
				return p[:1]
			}
			return p
		}
		got := strings.Join(hierarchyProblems(classes, in.merged, shortDirect, ssn.Parents, ssn.IsA), "\n")
		if !strings.Contains(got, "DirectParents("+sosa.Sample+")") {
			t.Fatalf("missing direct parent not flagged:\n%s", got)
		}
	})

	t.Run("missing closure ancestor", func(t *testing.T) {
		shortParents := func(iri string) []string {
			p := ssn.Parents(iri)
			if iri == ssn.Accuracy {
				return slices.DeleteFunc(slices.Clone(p), func(s string) bool { return s == ssn.Property })
			}
			return p
		}
		got := strings.Join(hierarchyProblems(classes, in.merged, ssn.DirectParents, shortParents, ssn.IsA), "\n")
		if !strings.Contains(got, "Parents("+ssn.Accuracy+")") {
			t.Fatalf("missing ancestor not flagged:\n%s", got)
		}
	})
}

func cloneMap(m map[string][2]string) map[string][2]string {
	out := make(map[string][2]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func cloneStringMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

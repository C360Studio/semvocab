package sosa_test

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/c360studio/semvocab/catalog/sosa"
	"github.com/c360studio/semvocab/internal/gosrc"
	"github.com/c360studio/semvocab/internal/ledger"
	"github.com/c360studio/semvocab/internal/oracle"
)

type inputs struct {
	src       ledger.SOSASource
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
	src, err := l.SOSA()
	if err != nil {
		t.Fatal(err)
	}
	entry, err := l.Entry("sosa")
	if err != nil {
		t.Fatal(err)
	}
	release, err := oracle.Load(oracle.SOSA2017)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := gosrc.Load(".")
	if err != nil {
		t.Fatal(err)
	}
	return inputs{
		src:       src,
		entry:     entry,
		namespace: l.Pins.Ontologies["sosa-2017"].Namespace,
		release:   release,
		pkg:       pkg,
	}
}

func parityProblems(in inputs) []string {
	var bad []string
	add := func(format string, args ...any) { bad = append(bad, fmt.Sprintf(format, args...)) }

	want := []string{"Namespace"}
	for name := range in.src.AdmittedClasses {
		want = append(want, name)
	}
	for name := range in.src.AdmittedObjectProperties {
		want = append(want, name)
	}
	for name := range in.src.AdmittedDatatypeProperties {
		want = append(want, name)
	}
	sort.Strings(want)
	if len(want) != 37 {
		add("expected 37 admitted identifiers in sosa, got %d", len(want))
	}
	if !slices.Equal(in.pkg.Exported, want) {
		add("exported identifiers %v, want %v", in.pkg.Exported, want)
	}

	excluded := append(in.entry.Excluded(), "HasResultingSample", "IsSamplingResultOf")
	for _, x := range excluded {
		if slices.Contains(in.pkg.Exported, x) {
			add("excluded source symbol %s is exported", x)
		}
	}
	for _, fn := range []string{"SubClassOf", "DirectParents", "Parents", "IsA"} {
		if slices.Contains(in.pkg.Exported, fn) {
			add("hierarchy function %s exported; sosa.ttl asserts zero rdfs:subClassOf triples", fn)
		}
	}
	if got := in.pkg.Consts["Namespace"]; got != in.namespace {
		add("Namespace = %q, ledger pin namespace %q", got, in.namespace)
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
			add("%s (%s) is not declared as %s in sosa.ttl", name, iri, typ)
		}
		if in.release.Deprecated(iri) {
			add("%s (%s) is owl:deprecated in sosa.ttl", name, iri)
		}
		if labels := in.release.Labels(iri); !slices.Equal(labels, []string{label}) {
			add("%s (%s) has rdfs:label %q in sosa.ttl, ledger records %q", name, iri, labels, label)
		}
	}
	admittedIRIs := make(map[string]bool, 36)
	for name, v := range in.src.AdmittedClasses {
		check(name, v[0], v[1], oracle.OWLClass)
		iri := in.namespace + v[0]
		admittedIRIs[iri] = true
		if supers := in.release.DirectSuperclasses(iri); len(supers) != 0 {
			add("sosa.ttl asserts unexpected rdfs:subClassOf on %s: %v", iri, supers)
		}
	}
	for name, v := range in.src.AdmittedObjectProperties {
		check(name, v[0], v[1], oracle.OWLObjectProp)
		admittedIRIs[in.namespace+v[0]] = true
	}
	for name, v := range in.src.AdmittedDatatypeProperties {
		check(name, v[0], v[1], oracle.OWLDataProp)
		admittedIRIs[in.namespace+v[0]] = true
	}
	for _, s := range in.release.SubjectsWithPrefix(in.namespace) {
		if !admittedIRIs[s] {
			add("unadmitted sosa.ttl subject %s", s)
		}
	}
	return bad
}

func TestCatalogParity(t *testing.T) {
	in := loadInputs(t)
	for _, p := range parityProblems(in) {
		t.Error(p)
	}
	if sosa.Sensor != "http://www.w3.org/ns/sosa/Sensor" {
		t.Errorf("sosa.Sensor = %q", sosa.Sensor)
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
		{"altered IRI", func(in *inputs) {
			in.pkg.Consts["Sensor"] = sosa.Namespace + "Detector"
		}, "Sensor = "},
		{"wrong label", func(in *inputs) {
			in.src.AdmittedClasses["FeatureOfInterest"] = [2]string{"FeatureOfInterest", "Feature of Interest"}
		}, `ledger records "Feature of Interest"`},
		{"wrong property type", func(in *inputs) {
			delete(in.src.AdmittedDatatypeProperties, "ResultTime")
			in.src.AdmittedObjectProperties["ResultTime"] = [2]string{"resultTime", "result time"}
		}, "ResultTime (http://www.w3.org/ns/sosa/resultTime) is not declared as http://www.w3.org/2002/07/owl#ObjectProperty"},
		{"excluded semconnect symbol", func(in *inputs) {
			in.pkg.Exported = append(in.pkg.Exported, "SamplingFeature")
			sort.Strings(in.pkg.Exported)
		}, "excluded source symbol SamplingFeature"},
		{"commented-out sosa symbol", func(in *inputs) {
			in.pkg.Exported = append(in.pkg.Exported, "HasResultingSample")
			sort.Strings(in.pkg.Exported)
		}, "excluded source symbol HasResultingSample"},
		{"unexpected hierarchy function", func(in *inputs) {
			in.pkg.Exported = append(in.pkg.Exported, "Parents")
			sort.Strings(in.pkg.Exported)
		}, "hierarchy function Parents exported"},
		{"unadmitted sosa.ttl subject", func(in *inputs) {
			delete(in.src.AdmittedClasses, "Sensor")
		}, "unadmitted sosa.ttl subject http://www.w3.org/ns/sosa/Sensor"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := loadInputs(t)
			in.src.AdmittedClasses = cloneMap(in.src.AdmittedClasses)
			in.src.AdmittedObjectProperties = cloneMap(in.src.AdmittedObjectProperties)
			in.src.AdmittedDatatypeProperties = cloneMap(in.src.AdmittedDatatypeProperties)
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

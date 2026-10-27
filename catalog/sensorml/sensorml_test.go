package sensorml_test

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/c360studio/semvocab/catalog/sensorml"
	"github.com/c360studio/semvocab/catalog/swe"
	"github.com/c360studio/semvocab/internal/gosrc"
	"github.com/c360studio/semvocab/internal/ledger"
	"github.com/c360studio/semvocab/internal/oracle"
)

type inputs struct {
	src         ledger.SensorMLSource
	entry       ledger.Entry
	specBaseURI string
	namespace   string
	schemas     *oracle.SchemaCatalog
	pkg         gosrc.Package
}

func loadInputs(t *testing.T) inputs {
	t.Helper()
	l, err := ledger.Load()
	if err != nil {
		t.Fatal(err)
	}
	src, err := l.SensorML()
	if err != nil {
		t.Fatal(err)
	}
	entry, err := l.Entry("sensorml")
	if err != nil {
		t.Fatal(err)
	}
	schemas, err := oracle.LoadSchemaCatalog(oracle.SensorML30)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := gosrc.Load(".")
	if err != nil {
		t.Fatal(err)
	}
	return inputs{
		src:         src,
		entry:       entry,
		specBaseURI: l.Pins.Schemas["sensorml-3.0"].SpecBaseURI,
		namespace:   entry.Namespace.IRI,
		schemas:     schemas,
		pkg:         pkg,
	}
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
	if len(want) != 133 {
		add("expected 133 admitted identifiers in sensorml, got %d", len(want))
	}
	if !slices.Equal(in.pkg.Exported, want) {
		add("exported identifiers %v, want %v", in.pkg.Exported, want)
	}

	excluded := append(
		in.entry.Excluded(),
		"Type", "PropType",
		"Href", "PropHref", "Rel", "PropRel", "Hreflang", "PropHreflang",
		"Title", "PropTitle", "UID", "PropUID", "RT", "PropRT", "If", "PropIf",
	)
	for _, x := range excluded {
		if slices.Contains(in.pkg.Exported, x) {
			add("excluded source symbol %s is exported", x)
		}
	}
	if in.namespace != in.specBaseURI+"/" {
		add("ledger namespace %q does not equal spec_base_uri+/ (%q)", in.namespace, in.specBaseURI+"/")
	}
	if got := in.pkg.Consts["Namespace"]; got != in.namespace {
		add("Namespace = %q, ledger namespace %q", got, in.namespace)
	}

	for name, v := range in.src.AdmittedClasses {
		local, loc, wantParent := v[0], v[1], v[2]
		iri, ok := in.pkg.Consts[name]
		switch {
		case !ok:
			add("admitted class %s is not an exported string constant", name)
			continue
		case iri != in.namespace+local:
			add("%s = %q, ledger records %s%s", name, iri, in.namespace, local)
		}
		if !in.schemas.HasObjectClass(local, loc) {
			add("%s (%s) does not resolve to an object schema at %s", name, local, loc)
			continue
		}
		gotParent, err := in.schemas.DirectParent(local, loc)
		if err != nil {
			add("%s DirectParent(%s): %v", name, loc, err)
		} else if gotParent != wantParent {
			add("%s at %s has allOf parent %q in schema, ledger records %q", name, loc, gotParent, wantParent)
		}
	}

	propSet := map[string]bool{"type": true}
	for name, key := range in.src.AdmittedProperties {
		if !strings.HasPrefix(name, "Prop") {
			add("property constant %s lacks Prop prefix", name)
		}
		iri, ok := in.pkg.Consts[name]
		switch {
		case !ok:
			add("admitted property %s is not an exported string constant", name)
		case iri != in.namespace+key:
			add("%s = %q, ledger records %s%s", name, iri, in.namespace, key)
		}
		propSet[key] = true
	}
	var wantProps []string
	for k := range propSet {
		wantProps = append(wantProps, k)
	}
	slices.Sort(wantProps)
	if gotProps := in.schemas.DeclaredProperties(); !slices.Equal(gotProps, wantProps) {
		add("admitted properties U {type} = %v, schema declared properties = %v", wantProps, gotProps)
	}
	return bad
}

func TestCatalogParity(t *testing.T) {
	for _, p := range parityProblems(loadInputs(t)) {
		t.Error(p)
	}
	if sensorml.PropLink != "http://www.opengis.net/spec/sensorML/3.0/link" {
		t.Errorf("sensorml.PropLink = %q", sensorml.PropLink)
	}
	if sensorml.Link != "http://www.opengis.net/spec/sensorML/3.0/Link" {
		t.Errorf("sensorml.Link = %q", sensorml.Link)
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
		{"altered class IRI", func(in *inputs) {
			in.pkg.Consts["PhysicalSystem"] = sensorml.Namespace + "System"
		}, "PhysicalSystem = "},
		{"non-object union location", func(in *inputs) {
			in.src.AdmittedClasses["GeoPose"] = [3]string{"Pose", "sensorml/schemas/json/Pose.json", ""}
			in.pkg.Consts["GeoPose"] = sensorml.Namespace + "Pose"
		}, "does not resolve to an object schema"},
		{"pure ref alias location", func(in *inputs) {
			in.src.AdmittedClasses["Link"] = [3]string{"XLink", "sensorml/schemas/json/commonDefs.json#/$defs/XLink", ""}
			in.pkg.Consts["Link"] = sensorml.Namespace + "XLink"
		}, "does not resolve to an object schema"},
		{"wrong cross-catalog parent", func(in *inputs) {
			in.src.AdmittedClasses["CharacteristicList"] = [3]string{
				"CharacteristicList",
				"sensorml/schemas/json/DescribedObject.json#/$defs/CharacteristicList",
				"DescribedObject",
			}
		}, `ledger records "DescribedObject"`},
		{"missing schema property", func(in *inputs) {
			delete(in.src.AdmittedProperties, "PropOrganisationName")
			delete(in.pkg.Consts, "PropOrganisationName")
			in.pkg.Exported = slices.DeleteFunc(in.pkg.Exported, func(s string) bool { return s == "PropOrganisationName" })
		}, "schema declared properties"},
		{"excluded RFC 5988 link property", func(in *inputs) {
			in.pkg.Exported = append(in.pkg.Exported, "PropHref")
			sort.Strings(in.pkg.Exported)
		}, "excluded source symbol PropHref"},
		{"excluded sweCommonDefs alias", func(in *inputs) {
			in.pkg.Exported = append(in.pkg.Exported, "AbstractSWEIdentifiable")
			sort.Strings(in.pkg.Exported)
		}, "excluded source symbol AbstractSWEIdentifiable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := loadInputs(t)
			in.src.AdmittedClasses = cloneClassMap(in.src.AdmittedClasses)
			in.src.AdmittedProperties = cloneStringMap(in.src.AdmittedProperties)
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

func classIRIs(in inputs) []string {
	var out []string
	for _, v := range in.src.AdmittedClasses {
		out = append(out, in.namespace+v[0])
	}
	sort.Strings(out)
	return out
}

func hierarchyProblems(
	classes []string,
	schemas *oracle.SchemaCatalog,
	subClassOf func(string) (string, bool),
	parents func(string) []string,
	isA func(string, string) bool,
) []string {
	var bad []string
	for _, c := range classes {
		want := schemas.Closure(c)
		if got := parents(c); !slices.Equal(got, want) {
			bad = append(bad, fmt.Sprintf("Parents(%s) = %v, schema closure %v", c, got, want))
		}
		gotDirect, ok := subClassOf(c)
		switch {
		case len(want) == 0 && (ok || gotDirect != ""):
			bad = append(bad, fmt.Sprintf("SubClassOf(%s) = %q, %v; want empty, false", c, gotDirect, ok))
		case len(want) > 0 && (!ok || gotDirect != want[0]):
			bad = append(bad, fmt.Sprintf("SubClassOf(%s) = %q, %v; want %q, true", c, gotDirect, ok, want[0]))
		}
		if !isA(c, c) {
			bad = append(bad, fmt.Sprintf("IsA(%s, itself) = false", c))
		}
		for _, other := range append(slices.Clone(classes), want...) {
			if other == c {
				continue
			}
			if got, wantIsA := isA(c, other), slices.Contains(want, other); got != wantIsA {
				bad = append(bad, fmt.Sprintf("IsA(%s, %s) = %v, schema closure says %v", c, other, got, wantIsA))
			}
		}
	}
	return bad
}

func TestCatalogHierarchyMatchesSchema(t *testing.T) {
	in := loadInputs(t)
	for _, p := range hierarchyProblems(classIRIs(in), in.schemas, sensorml.SubClassOf, sensorml.Parents, sensorml.IsA) {
		t.Error(p)
	}

	// Spec scenario: cross-catalog JSON Schema hierarchy into catalog/swe.
	wantCharList := []string{swe.AbstractSweIdentifiable, swe.AbstractSWE}
	if got := sensorml.Parents(sensorml.CharacteristicList); !slices.Equal(got, wantCharList) {
		t.Errorf("Parents(CharacteristicList) = %v, want %v", got, wantCharList)
	}
	if !sensorml.IsA(sensorml.CharacteristicList, swe.AbstractSWE) {
		t.Error("IsA(CharacteristicList, swe.AbstractSWE) = false, want true")
	}

	wantPhysSys := []string{
		sensorml.AbstractPhysicalProcess,
		sensorml.AbstractProcess,
		sensorml.DescribedObject,
	}
	if got := sensorml.Parents(sensorml.PhysicalSystem); !slices.Equal(got, wantPhysSys) {
		t.Errorf("Parents(PhysicalSystem) = %v, want %v", got, wantPhysSys)
	}
}

// TestCatalogHierarchySensitivity drops swe.AbstractSWE from Parents and
// requires hierarchyProblems to name the cross-catalog break.
func TestCatalogHierarchySensitivity(t *testing.T) {
	in := loadInputs(t)
	short := func(iri string) []string {
		p := sensorml.Parents(iri)
		if iri == sensorml.CharacteristicList {
			p = slices.DeleteFunc(slices.Clone(p), func(s string) bool { return s == swe.AbstractSWE })
		}
		return p
	}
	got := strings.Join(hierarchyProblems(classIRIs(in), in.schemas, sensorml.SubClassOf, short, sensorml.IsA), "\n")
	if !strings.Contains(got, "Parents("+sensorml.CharacteristicList+")") {
		t.Fatalf("missing cross-catalog ancestor not flagged:\n%s", got)
	}
}

func cloneClassMap(m map[string][3]string) map[string][3]string {
	out := make(map[string][3]string, len(m))
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

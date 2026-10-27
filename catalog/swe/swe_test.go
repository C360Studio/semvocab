package swe_test

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/c360studio/semvocab/catalog/swe"
	"github.com/c360studio/semvocab/internal/gosrc"
	"github.com/c360studio/semvocab/internal/ledger"
	"github.com/c360studio/semvocab/internal/oracle"
)

const swe20Namespace = "http://www.opengis.net/swe/2.0/"

type inputs struct {
	src         ledger.SWESource
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
	src, err := l.SWE()
	if err != nil {
		t.Fatal(err)
	}
	entry, err := l.Entry("swe")
	if err != nil {
		t.Fatal(err)
	}
	schemas, err := oracle.LoadSchemaCatalog(oracle.SWE30)
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
		specBaseURI: l.Pins.Schemas["swe-3.0"].SpecBaseURI,
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
	if len(want) != 94 {
		add("expected 94 admitted identifiers in swe, got %d", len(want))
	}
	if !slices.Equal(in.pkg.Exported, want) {
		add("exported identifiers %v, want %v", in.pkg.Exported, want)
	}

	excluded := append(in.entry.Excluded(), "PropNilValue", "Type", "PropType")
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
	for name, v := range in.pkg.Consts {
		if strings.HasPrefix(v, swe20Namespace) {
			add("%s carries SWE 2.0 IRI %s", name, v)
		}
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
	if swe.PropNamespace != "http://www.opengis.net/spec/SWE/3.0/namespace" {
		t.Errorf("swe.PropNamespace = %q", swe.PropNamespace)
	}
	if swe.Namespace != "http://www.opengis.net/spec/SWE/3.0/" {
		t.Errorf("swe.Namespace = %q", swe.Namespace)
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
			in.pkg.Consts["Quantity"] = swe.Namespace + "ScalarQuantity"
		}, "Quantity = "},
		{"SWE 2.0 IRI", func(in *inputs) {
			in.pkg.Consts["Quantity"] = swe20Namespace + "Quantity"
		}, "SWE 2.0 IRI"},
		{"non-object schema location", func(in *inputs) {
			in.src.AdmittedClasses["Quantity"] = [3]string{"AnyComponent", "swecommon/schemas/json/sweCommon.json#/$defs/AnyComponent", ""}
			in.pkg.Consts["Quantity"] = swe.Namespace + "AnyComponent"
		}, "does not resolve to an object schema"},
		{"wrong allOf parent", func(in *inputs) {
			in.src.AdmittedClasses["Quantity"] = [3]string{"Quantity", "swecommon/schemas/json/Quantity.json", "AbstractDataComponent"}
		}, `ledger records "AbstractDataComponent"`},
		{"missing schema property", func(in *inputs) {
			delete(in.src.AdmittedProperties, "PropUOM")
			delete(in.pkg.Consts, "PropUOM")
			in.pkg.Exported = slices.DeleteFunc(in.pkg.Exported, func(s string) bool { return s == "PropUOM" })
		}, "schema declared properties"},
		{"excluded symbol PropNilValue", func(in *inputs) {
			in.pkg.Exported = append(in.pkg.Exported, "PropNilValue")
			sort.Strings(in.pkg.Exported)
		}, "excluded source symbol PropNilValue"},
		{"excluded symbol ElementCount", func(in *inputs) {
			in.pkg.Exported = append(in.pkg.Exported, "ElementCount")
			sort.Strings(in.pkg.Exported)
		}, "excluded source symbol ElementCount"},
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
	for _, p := range hierarchyProblems(classIRIs(in), in.schemas, swe.SubClassOf, swe.Parents, swe.IsA) {
		t.Error(p)
	}
	wantQuantity := []string{
		swe.AbstractSimpleComponent,
		swe.AbstractDataComponent,
		swe.AbstractSweIdentifiable,
		swe.AbstractSWE,
	}
	if got := swe.Parents(swe.Quantity); !slices.Equal(got, wantQuantity) {
		t.Errorf("Parents(swe.Quantity) = %v, want %v", got, wantQuantity)
	}
}

// TestCatalogHierarchySensitivity drops one ancestor from Parents and requires
// hierarchyProblems to name it.
func TestCatalogHierarchySensitivity(t *testing.T) {
	in := loadInputs(t)
	short := func(iri string) []string {
		p := swe.Parents(iri)
		if iri == swe.Quantity {
			p = slices.DeleteFunc(slices.Clone(p), func(s string) bool { return s == swe.AbstractSweIdentifiable })
		}
		return p
	}
	got := strings.Join(hierarchyProblems(classIRIs(in), in.schemas, swe.SubClassOf, short, swe.IsA), "\n")
	if !strings.Contains(got, "Parents("+swe.Quantity+")") {
		t.Fatalf("missing ancestor not flagged:\n%s", got)
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

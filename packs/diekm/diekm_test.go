package diekm_test

import (
	"bytes"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/c360studio/semengine/vocabulary"
	"github.com/c360studio/semvocab/internal/turtle"
	"github.com/c360studio/semvocab/packs/diekm"
)

const altitudePredicate = "intel.measurement.altitude_m"

// TestDIEKMSpecIsTheEmbeddedYAML loads the committed diekm.yaml through
// SemEngine's loader and requires Spec to return the same value.
func TestDIEKMSpecIsTheEmbeddedYAML(t *testing.T) {
	data, err := os.ReadFile("diekm.yaml")
	if err != nil {
		t.Fatal(err)
	}
	want, err := vocabulary.LoadPackYAML(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	got, err := diekm.Spec()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Spec() = %+v, want %+v", got, want)
	}
}

// TestDIEKMPackJSONYAMLEquivalence decodes the hand-written JSON encoding of
// the pack (testdata/diekm.json) with LoadPackJSON and requires the same
// PackSpec as LoadPackYAML gives for diekm.yaml.
func TestDIEKMPackJSONYAMLEquivalence(t *testing.T) {
	yamlData, err := os.ReadFile("diekm.yaml")
	if err != nil {
		t.Fatal(err)
	}
	jsonData, err := os.ReadFile("testdata/diekm.json")
	if err != nil {
		t.Fatal(err)
	}
	fromYAML, err := vocabulary.LoadPackYAML(bytes.NewReader(yamlData))
	if err != nil {
		t.Fatal(err)
	}
	fromJSON, err := vocabulary.LoadPackJSON(bytes.NewReader(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fromJSON, fromYAML) {
		t.Errorf("JSON and YAML encodings differ:\nJSON %+v\nYAML %+v", fromJSON, fromYAML)
	}
	// Sensitivity: one changed superclass makes them differ.
	altered := strings.Replace(string(jsonData), "BFO_0000004", "BFO_0000141", 1)
	fromAltered, err := vocabulary.LoadPackJSON(strings.NewReader(altered))
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(fromAltered, fromYAML) {
		t.Error("an altered JSON superclass still compares equal")
	}
}

// TestDIEKMSpecValidationBlockedOnPredicateName records the declared degrade
// in the package comment: SemEngine's pure validator rejects the pack for the
// altitude predicate's name alone. When the name is ruled on and changed, this
// test fails and is replaced by TestDIEKMPackLoadsAndRegisters.
func TestDIEKMSpecValidationBlockedOnPredicateName(t *testing.T) {
	spec, err := diekm.Spec()
	if err != nil {
		t.Fatal(err)
	}
	err = spec.Validate()
	if err == nil || !strings.Contains(err.Error(), altitudePredicate) ||
		!strings.Contains(err.Error(), "segment_character") {
		t.Fatalf("Validate() = %v, want the segment_character rejection of %s", err, altitudePredicate)
	}
	i := slices.IndexFunc(spec.Predicates, func(p vocabulary.PredicateSpec) bool { return p.Name == altitudePredicate })
	spec.Predicates[i].Name = strings.ReplaceAll(altitudePredicate, "_", "-")
	if err := spec.Validate(); err != nil {
		t.Errorf("with only the name made lower-kebab, Validate() = %v; another defect blocks the pack", err)
	}
}

// TestDIEKMShapes parses the embedded SHACL file and checks that it stays in
// the initial profile and names only the pack's classes and predicate IRIs.
func TestDIEKMShapes(t *testing.T) {
	const sh = "http://www.w3.org/ns/shacl#"
	triples, err := turtle.Parse(bytes.NewReader(diekm.Shapes()))
	if err != nil {
		t.Fatal(err)
	}
	spec, err := diekm.Spec()
	if err != nil {
		t.Fatal(err)
	}
	var classes, predicates []string
	for _, et := range spec.EntityTypes {
		classes = append(classes, et.ClassIRI)
	}
	for _, p := range spec.Predicates {
		predicates = append(predicates, p.IRI)
	}
	allowed := []string{sh + "targetClass", sh + "property", sh + "path", sh + "datatype", sh + "maxCount",
		"http://www.w3.org/1999/02/22-rdf-syntax-ns#type"}
	var targets []string
	for _, tr := range triples {
		if !slices.Contains(allowed, tr.P.Value) {
			t.Errorf("predicate %s is outside the initial shapes profile", tr.P.Value)
		}
		switch tr.P.Value {
		case sh + "targetClass":
			targets = append(targets, tr.O.Value)
			if !slices.Contains(classes, tr.O.Value) {
				t.Errorf("sh:targetClass %s is not a pack class", tr.O.Value)
			}
		case sh + "path":
			if !slices.Contains(predicates, tr.O.Value) {
				t.Errorf("sh:path %s is not a pack predicate IRI", tr.O.Value)
			}
		}
	}
	slices.Sort(targets)
	slices.Sort(classes)
	if !slices.Equal(targets, classes) {
		t.Errorf("shapes target %v, want every pack class %v", targets, classes)
	}

	// Shapes hands out a copy; a caller's write does not reach the next call.
	b := diekm.Shapes()
	b[0] ^= 0xff
	if bytes.Equal(diekm.Shapes(), b) {
		t.Error("Shapes returned the embedded slice itself")
	}
}

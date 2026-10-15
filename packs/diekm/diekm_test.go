package diekm_test

import (
	"bytes"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/c360studio/semengine/message"
	"github.com/c360studio/semengine/vocabulary"
	"github.com/c360studio/semengine/vocabulary/export"
	"github.com/c360studio/semvocab/internal/turtle"
	"github.com/c360studio/semvocab/packs/diekm"
)

const (
	altitudePredicate = "intel.measurement.altitude-m"
	diekmNS           = "https://w3id.org/c360/diekm#"
	platformID        = "acme.edge.sensor.intel.platform.p1"
)

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

// TestDIEKMPackLoadsAndRegisters takes the exact committed diekm.yaml through
// SemEngine's LoadPackYAML and RegisterPack with
// WithPrefixRegistrar(export.RegisterPrefixes), reads every entry back through
// the registry getters, and shows the diekm prefix compacting in Turtle. The
// expected values are written out here from the Phase 0 design (D5, Q2-Q4b)
// and ruling Q11, independently of the YAML.
func TestDIEKMPackLoadsAndRegisters(t *testing.T) {
	data, err := os.ReadFile("diekm.yaml")
	if err != nil {
		t.Fatal(err)
	}
	spec, err := vocabulary.LoadPackYAML(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if err := vocabulary.RegisterPack(spec, vocabulary.WithPrefixRegistrar(export.RegisterPrefixes)); err != nil {
		t.Fatalf("RegisterPack: %v", err)
	}
	if err := diekm.Register(); err != nil {
		t.Fatalf("diekm.Register after RegisterPack: %v", err)
	}

	const (
		obo = "http://purl.obolibrary.org/obo/"
		cco = "https://www.commoncoreontologies.org/"
	)
	entityTypes := map[string]vocabulary.EntityTypeMetadata{
		"intel.platform": {
			DomainType: "intel.platform", Domain: "intel", Type: "platform",
			Description: "Defense or intelligence collection platform",
			ClassIRI:    diekmNS + "IntelligencePlatform",
			Superclasses: []string{cco + "ont00000713", obo + "BFO_0000030", cco + "ont00000995",
				obo + "BFO_0000040", obo + "BFO_0000004", obo + "BFO_0000002", obo + "BFO_0000001"},
		},
		"intel.observation": {
			DomainType: "intel.observation", Domain: "intel", Type: "observation",
			Description: "Observation act recorded by an intelligence platform",
			ClassIRI:    diekmNS + "IntelligenceObservation",
			Superclasses: []string{cco + "ont00000037", cco + "ont00000228", cco + "ont00000005",
				obo + "BFO_0000015", obo + "BFO_0000003", obo + "BFO_0000001"},
		},
	}
	for key, want := range entityTypes {
		got := vocabulary.GetEntityTypeMetadata(key)
		if got == nil || !reflect.DeepEqual(*got, want) {
			t.Errorf("GetEntityTypeMetadata(%s) = %+v, want %+v", key, got, want)
		}
		for _, iri := range append([]string{want.ClassIRI}, want.Superclasses...) {
			if !slices.Contains(vocabulary.LookupEntityTypesByClassIRI(iri), key) {
				t.Errorf("LookupEntityTypesByClassIRI(%s) = %v, want it to include %s",
					iri, vocabulary.LookupEntityTypesByClassIRI(iri), key)
			}
		}
	}
	if got := vocabulary.LookupEntityTypesByClassIRI(diekmNS + "IntelligencePlatform"); !slices.Equal(got, []string{"intel.platform"}) {
		t.Errorf("LookupEntityTypesByClassIRI(IntelligencePlatform) = %v", got)
	}

	predicates := map[string]struct {
		iri, dataType, units, description string
		steps                             []vocabulary.ProjectionStep
	}{
		"intel.role.assigned": {
			iri: diekmNS + "assignedRole", dataType: "string",
			description: "Assigned role, projected as bearer of (BFO_0000196) a role (BFO_0000023) whose rdfs:label is the value",
			steps: []vocabulary.ProjectionStep{{PredicateIRI: obo + "BFO_0000196", NodeClassIRI: obo + "BFO_0000023",
				ValuePredicateIRI: "http://www.w3.org/2000/01/rdf-schema#label"}},
		},
		altitudePredicate: {
			iri: diekmNS + "altitudeMeters", dataType: "float", units: "m",
			description: "Altitude measurement in meters",
			steps: []vocabulary.ProjectionStep{
				{PredicateIRI: obo + "BFO_0000196", NodeClassIRI: cco + "ont00000787"},
				{PredicateIRI: cco + "ont00001904", NodeClassIRI: cco + "ont00001163"},
				{PredicateIRI: obo + "BFO_0000084", NodeClassIRI: cco + "ont00000253",
					ValuePredicateIRI: diekmNS + "altitudeMetersValue"},
			},
		},
	}
	for name, want := range predicates {
		got := vocabulary.GetPredicateMetadata(name)
		if got == nil {
			t.Errorf("GetPredicateMetadata(%s) = nil", name)
			continue
		}
		if got.StandardIRI != want.iri || got.DataType != want.dataType || got.Units != want.units ||
			got.Description != want.description || got.Domain != "intel" {
			t.Errorf("GetPredicateMetadata(%s) = %+v, want IRI %s, data type %s, units %q", name, got, want.iri,
				want.dataType, want.units)
		}
		if got.Projection == nil || !reflect.DeepEqual(got.Projection.Steps, want.steps) {
			t.Errorf("GetPredicateMetadata(%s).Projection = %+v, want steps %+v", name, got.Projection, want.steps)
		}
		if lookup := vocabulary.LookupByIRI(want.iri); !slices.Equal(lookup, []string{name}) {
			t.Errorf("LookupByIRI(%s) = %v, want [%s]", want.iri, lookup, name)
		}
	}

	out, err := export.SerializeToString([]message.Triple{
		{Subject: platformID, Predicate: "intel.role.assigned", Object: "pilot"},
		{Subject: platformID, Predicate: altitudePredicate, Object: 1200.5},
	}, export.Turtle)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"@prefix diekm: <" + diekmNS + "> .", "diekm:assignedRole", "diekm:altitudeMeters"} {
		if !strings.Contains(out, s) {
			t.Errorf("Turtle output lacks %q:\n%s", s, out)
		}
	}
}

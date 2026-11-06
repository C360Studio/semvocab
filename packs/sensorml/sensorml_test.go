package sensorml_test

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
	"github.com/c360studio/semvocab/catalog/sensorml"
	"github.com/c360studio/semvocab/catalog/sosa"
	"github.com/c360studio/semvocab/catalog/ssn"
	"github.com/c360studio/semvocab/catalog/swe"
	"github.com/c360studio/semvocab/internal/oracle"
	"github.com/c360studio/semvocab/internal/turtle"
	smlpack "github.com/c360studio/semvocab/packs/sensorml"
)

const (
	dcDescription   = "http://purl.org/dc/terms/description"
	rdfsIsDefinedBy = "http://www.w3.org/2000/01/rdf-schema#isDefinedBy"
	systemID        = "c360.semconnect.systems.csapi.system.s1"
	subSystemID     = "c360.semconnect.systems.csapi.system.s2"
)

type wantPred struct {
	iri, dataType, inverseOf, description string
	steps                                 []vocabulary.ProjectionStep
}

var wantEntityTypes = map[string]vocabulary.EntityTypeMetadata{
	"csapi.system": {
		DomainType:  "csapi.system",
		Domain:      "csapi",
		Type:        "system",
		Description: "OGC API — Connected Systems Part 1 physical or virtual system",
		ClassIRI:    sensorml.PhysicalSystem,
		Superclasses: []string{
			sensorml.AbstractPhysicalProcess,
			ssn.System,
			sensorml.AbstractProcess,
			sensorml.DescribedObject,
		},
	},
	"csapi.procedure": {
		DomainType:  "csapi.procedure",
		Domain:      "csapi",
		Type:        "procedure",
		Description: "OGC API — Connected Systems Part 1 observation, sampling, or actuation procedure",
		ClassIRI:    sensorml.SimpleProcess,
		Superclasses: []string{
			sensorml.AbstractProcess,
			sosa.Procedure,
			sensorml.DescribedObject,
		},
	},
	"csapi.deployment": {
		DomainType:  "csapi.deployment",
		Domain:      "csapi",
		Type:        "deployment",
		Description: "OGC API — Connected Systems Part 1 system deployment",
		ClassIRI:    sensorml.Deployment,
		Superclasses: []string{
			sensorml.DescribedObject,
			ssn.Deployment,
		},
	},
	"csapi.samplingfeature": {
		DomainType:  "csapi.samplingfeature",
		Domain:      "csapi",
		Type:        "samplingfeature",
		Description: "OGC API — Connected Systems Part 1 sampling feature",
		ClassIRI:    sosa.Sample,
		Superclasses: []string{
			sosa.FeatureOfInterest,
			sosa.Result,
		},
	},
	"csapi.property": {
		DomainType:  "csapi.property",
		Domain:      "csapi",
		Type:        "property",
		Description: "OGC API — Connected Systems Part 1 derived observable or controllable property",
		ClassIRI:    sensorml.DerivedProperty,
		Superclasses: []string{
			swe.AbstractSweIdentifiable,
			sosa.ObservableProperty,
			swe.AbstractSWE,
			ssn.Property,
		},
	},
}

var wantPredicates = map[string]wantPred{
	"sensorml.process.type": {
		iri:         oracle.RDFType,
		dataType:    vocabulary.DataTypeString,
		description: "SensorML or SOSA resource type discriminator or class URI",
	},
	"sensorml.process.uid": {
		iri:         vocabulary.DcIdentifier,
		dataType:    vocabulary.DataTypeString,
		description: "Globally unique URI identifier (uniqueId) of the resource",
	},
	"sensorml.process.label": {
		iri:         vocabulary.DcTitle,
		dataType:    vocabulary.DataTypeString,
		description: "Human-readable label of the resource",
	},
	"sensorml.process.description": {
		iri:         dcDescription,
		dataType:    vocabulary.DataTypeString,
		description: "Human-readable textual description of the resource",
	},
	"sensorml.process.definition": {
		iri:         rdfsIsDefinedBy,
		dataType:    vocabulary.DataTypeString,
		description: "Semantic definition URI classifying the process, deployment, or sampling feature",
	},
	"sensorml.system.hosts": {
		iri:         sosa.Hosts,
		dataType:    vocabulary.DataTypeEntityID,
		inverseOf:   "sensorml.component.is-hosted-by",
		description: "Hosted subsystem, component, or sampling feature entity",
	},
	"sensorml.component.is-hosted-by": {
		iri:         sosa.IsHostedBy,
		dataType:    vocabulary.DataTypeEntityID,
		inverseOf:   "sensorml.system.hosts",
		description: "Hosting parent system entity",
	},
	"sensorml.process.has-sub-system": {
		iri:         ssn.HasSubSystem,
		dataType:    vocabulary.DataTypeEntityID,
		description: "Component subsystem entity of a composite system",
	},
	"sensorml.process.used-procedure": {
		iri:         sosa.UsedProcedure,
		dataType:    vocabulary.DataTypeString,
		description: "Procedure URI implemented or used by a system",
	},
	"sensorml.process.attached-to": {
		iri:         sensorml.PropAttachedTo,
		dataType:    vocabulary.DataTypeString,
		description: "External platform or host URI to which a physical process is attached",
	},
	"sensorml.process.position": {
		iri:         sensorml.PropPosition,
		dataType:    vocabulary.DataTypeJSON,
		description: "GeoJSON geometry, SensorML Pose, or SWE position document",
	},
	"sensorml.identifier.value": {
		iri:         vocabulary.SkosNotation,
		description: "Additional discovery identifier or notation value",
	},
	"sensorml.capability.value": {
		iri:         ssn.HasSystemCapability,
		description: "System capability value, projected via ssn-system:hasSystemCapability to sml:CapabilityList and swe:value",
		steps: []vocabulary.ProjectionStep{{
			PredicateIRI:      ssn.HasSystemCapability,
			NodeClassIRI:      sensorml.CapabilityList,
			ValuePredicateIRI: swe.PropValue,
		}},
	},
	"sensorml.characteristic.value": {
		iri:         ssn.HasProperty,
		description: "System characteristic value, projected via ssn:hasProperty to sml:CharacteristicList and swe:value",
		steps: []vocabulary.ProjectionStep{{
			PredicateIRI:      ssn.HasProperty,
			NodeClassIRI:      sensorml.CharacteristicList,
			ValuePredicateIRI: swe.PropValue,
		}},
	},
	"sensorml.deployment.deployed-system": {
		iri:         ssn.DeployedSystem,
		dataType:    vocabulary.DataTypeString,
		description: "System URI deployed during a deployment",
	},
	"sensorml.deployment.parent": {
		iri:         vocabulary.SkosBroader,
		dataType:    vocabulary.DataTypeEntityID,
		description: "Parent deployment entity containing a sub-deployment",
	},
	"sensorml.sampling.sampled-feature": {
		iri:         sosa.IsSampleOf,
		dataType:    vocabulary.DataTypeString,
		description: "Domain feature of interest URI sampled by a sampling feature",
	},
	"sensorml.sampling.hosted-procedure": {
		iri:         ssn.Implements,
		dataType:    vocabulary.DataTypeString,
		description: "Observation or sampling procedure URI hosted on a sampling feature",
	},
	"sensorml.property.definition": {
		iri:         sensorml.PropDefinition,
		dataType:    vocabulary.DataTypeString,
		description: "Semantic property definition URI of a derived property",
	},
	"sensorml.property.base-property": {
		iri:         sensorml.PropBaseProperty,
		dataType:    vocabulary.DataTypeString,
		description: "Base observable or actuatable property URI from which a derived property is derived",
	},
}

// TestSensorMLSpecIsTheEmbeddedYAML loads the committed sensorml.yaml through
// SemEngine's loader and requires Spec to return the same value.
func TestSensorMLSpecIsTheEmbeddedYAML(t *testing.T) {
	data, err := os.ReadFile("sensorml.yaml")
	if err != nil {
		t.Fatal(err)
	}
	want, err := vocabulary.LoadPackYAML(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	got, err := smlpack.Spec()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Spec() = %+v, want %+v", got, want)
	}
}

// TestSensorMLPackJSONYAMLEquivalence decodes the hand-written JSON encoding of
// the pack (testdata/sensorml.json) with LoadPackJSON and requires the same
// PackSpec as LoadPackYAML gives for sensorml.yaml.
func TestSensorMLPackJSONYAMLEquivalence(t *testing.T) {
	yamlData, err := os.ReadFile("sensorml.yaml")
	if err != nil {
		t.Fatal(err)
	}
	jsonData, err := os.ReadFile("testdata/sensorml.json")
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
	altered := strings.Replace(string(jsonData), ssn.System, sosa.Sensor, 1)
	fromAltered, err := vocabulary.LoadPackJSON(strings.NewReader(altered))
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(fromAltered, fromYAML) {
		t.Error("an altered JSON superclass still compares equal")
	}
}

// TestSensorMLShapesAndMapping parses the embedded SHACL file, checks that it
// stays within SemVocab's documented profile with open shapes, targets all 5
// pack classes, constrains all 20 pack predicate IRIs, and verifies defensive
// copies on Shapes and Mapping.
func TestSensorMLShapesAndMapping(t *testing.T) {
	const sh = "http://www.w3.org/ns/shacl#"
	shapesBytes := smlpack.Shapes()
	if len(shapesBytes) == 0 {
		t.Fatal("Shapes() returned empty slice")
	}
	triples, err := turtle.Parse(bytes.NewReader(shapesBytes))
	if err != nil {
		t.Fatal(err)
	}
	spec, err := smlpack.Spec()
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
	allowed := []string{
		oracle.RDFType,
		sh + "targetClass",
		sh + "property",
		sh + "path",
		sh + "datatype",
		sh + "minCount",
		sh + "maxCount",
		sh + "pattern",
	}
	var targets, paths []string
	for _, tr := range triples {
		if !slices.Contains(allowed, tr.P.Value) {
			t.Errorf("predicate %s is outside the sensorml shapes profile", tr.P.Value)
		}
		switch tr.P.Value {
		case sh + "targetClass":
			targets = append(targets, tr.O.Value)
			if !slices.Contains(classes, tr.O.Value) {
				t.Errorf("sh:targetClass %s is not a pack class", tr.O.Value)
			}
		case sh + "path":
			paths = append(paths, tr.O.Value)
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
	slices.Sort(paths)
	paths = slices.Compact(paths)
	slices.Sort(predicates)
	if !slices.Equal(paths, predicates) {
		t.Errorf("shapes constrain %v, want all 20 pack predicates %v", paths, predicates)
	}

	b := smlpack.Shapes()
	b[0] ^= 0xff
	if bytes.Equal(smlpack.Shapes(), b) {
		t.Error("Shapes returned the embedded slice itself")
	}

	wantMapping, err := os.ReadFile("sensorml.mapping.yaml")
	if err != nil {
		t.Fatal(err)
	}
	m := smlpack.Mapping()
	if !bytes.Equal(m, wantMapping) {
		t.Error("Mapping() does not equal sensorml.mapping.yaml")
	}
	m[0] ^= 0xff
	if bytes.Equal(smlpack.Mapping(), m) {
		t.Error("Mapping returned the embedded slice itself")
	}
}

// TestSensorMLPackRegistersEntityTypes registers sensorml and verifies all 5
// csapi.* entity types and their SensorML 3.0 + SOSA/SSN superclass lookups.
func TestSensorMLPackRegistersEntityTypes(t *testing.T) {
	data, err := os.ReadFile("sensorml.yaml")
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
	if err := smlpack.Register(); err != nil {
		t.Fatalf("sensorml.Register after RegisterPack: %v", err)
	}

	for key, want := range wantEntityTypes {
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

	if got := vocabulary.LookupEntityTypesByClassIRI(ssn.System); !slices.Contains(got, "csapi.system") {
		t.Errorf("LookupEntityTypesByClassIRI(ssn.System) = %v, want csapi.system", got)
	}
	if got := vocabulary.LookupEntityTypesByClassIRI(sosa.Procedure); !slices.Contains(got, "csapi.procedure") ||
		slices.Contains(got, "csapi.system") {
		t.Errorf("LookupEntityTypesByClassIRI(sosa.Procedure) = %v, want csapi.procedure and not csapi.system", got)
	}
	if got := vocabulary.LookupEntityTypesByClassIRI(ssn.Property); !slices.Contains(got, "csapi.property") {
		t.Errorf("LookupEntityTypesByClassIRI(ssn.Property) = %v, want csapi.property", got)
	}
	if got := vocabulary.LookupEntityTypesByClassIRI(sensorml.DescribedObject); !slices.Contains(got, "csapi.system") ||
		!slices.Contains(got, "csapi.procedure") || !slices.Contains(got, "csapi.deployment") ||
		slices.Contains(got, "csapi.samplingfeature") || slices.Contains(got, "csapi.property") {
		t.Errorf("LookupEntityTypesByClassIRI(sml:DescribedObject) = %v, want system/procedure/deployment only", got)
	}
}

// TestSensorMLPackRegistersPredicatesAndExports registers sensorml, checks all
// 20 sensorml.* predicates, inverses, projections, and IRI reverse lookups, and
// verifies Turtle prefix compaction across all domain namespaces.
func TestSensorMLPackRegistersPredicatesAndExports(t *testing.T) {
	if err := smlpack.Register(); err != nil {
		t.Fatalf("sensorml.Register: %v", err)
	}

	for name, want := range wantPredicates {
		got := vocabulary.GetPredicateMetadata(name)
		if got == nil {
			t.Errorf("GetPredicateMetadata(%s) = nil", name)
			continue
		}
		if got.StandardIRI != want.iri || got.DataType != want.dataType || got.InverseOf != want.inverseOf ||
			got.Description != want.description || got.Domain != "sensorml" {
			t.Errorf("GetPredicateMetadata(%s) = %+v, want IRI %s, dataType %q, inverseOf %q, description %q",
				name, got, want.iri, want.dataType, want.inverseOf, want.description)
		}
		if len(want.steps) == 0 {
			if got.Projection != nil {
				t.Errorf("GetPredicateMetadata(%s).Projection = %+v, want nil", name, got.Projection)
			}
		} else if got.Projection == nil || !reflect.DeepEqual(got.Projection.Steps, want.steps) {
			t.Errorf("GetPredicateMetadata(%s).Projection = %+v, want steps %+v", name, got.Projection, want.steps)
		}
		if lookup := vocabulary.LookupByIRI(want.iri); !slices.Contains(lookup, name) {
			t.Errorf("LookupByIRI(%s) = %v, want it to include %s", want.iri, lookup, name)
		}
	}

	out, err := export.SerializeToString([]message.Triple{
		{Subject: systemID, Predicate: "sensorml.process.uid", Object: "urn:x-ogc:systems:001"},
		{Subject: systemID, Predicate: "sensorml.process.label", Object: "Weather Station Alpha"},
		{Subject: systemID, Predicate: "sensorml.system.hosts", Object: subSystemID},
		{Subject: systemID, Predicate: "sensorml.process.has-sub-system", Object: subSystemID},
		{Subject: systemID, Predicate: "sensorml.process.position", Object: `{"type":"Point","coordinates":[-117.16,32.71]}`},
		{Subject: systemID, Predicate: "sensorml.identifier.value", Object: "WS-001"},
		{Subject: systemID, Predicate: "sensorml.capability.value", Object: 10.5},
		{Subject: systemID, Predicate: swe.PropValue, Object: 42.0},
	}, export.Turtle)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{
		"@prefix dc: <http://purl.org/dc/terms/> .",
		"@prefix skos: <http://www.w3.org/2004/02/skos/core#> .",
		"@prefix sosa: <" + sosa.Namespace + "> .",
		"@prefix ssn: <" + ssn.Namespace + "> .",
		"@prefix ssn-system: <" + ssn.SystemsNamespace + "> .",
		"@prefix swe: <" + swe.Namespace + "> .",
		"@prefix sml: <" + sensorml.Namespace + "> .",
		"dc:identifier",
		"dc:title",
		"sosa:hosts",
		"ssn:hasSubSystem",
		"sml:position",
		"skos:notation",
		"ssn-system:hasSystemCapability",
		"swe:value",
	} {
		if !strings.Contains(out, s) {
			t.Errorf("Turtle output lacks %q:\n%s", s, out)
		}
	}
}

package shacl_test

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/c360studio/semengine/message"
	"github.com/c360studio/semengine/pkg/projection/contract"
	semtypes "github.com/c360studio/semengine/pkg/types"
	"github.com/c360studio/semengine/vocabulary"
	"github.com/c360studio/semengine/vocabulary/export"
	"github.com/c360studio/semvocab/catalog/sensorml"
	"github.com/c360studio/semvocab/catalog/sosa"
	"github.com/c360studio/semvocab/catalog/ssn"
	"github.com/c360studio/semvocab/catalog/swe"
	smlpack "github.com/c360studio/semvocab/packs/sensorml"
	"github.com/c360studio/semvocab/shacl"
)

const shaclNS = "http://www.w3.org/ns/shacl#"

var wantSensorMLContracts = []contract.Contract{
	{
		Name:            "csapi.deployment",
		EntityPattern:   "*.*.*.csapi.deployment.*",
		BirthPredicates: []string{"sensorml.process.type"},
		Groups: []contract.PredicateGroup{{
			Name: "representation",
			Mode: contract.ModeReconcile,
			Predicates: []string{
				"sensorml.capability.value",
				"sensorml.characteristic.value",
				"sensorml.deployment.deployed-system",
				"sensorml.deployment.parent",
				"sensorml.identifier.value",
				"sensorml.process.definition",
				"sensorml.process.description",
				"sensorml.process.label",
				"sensorml.process.position",
				"sensorml.process.uid",
			},
		}},
	},
	{
		Name:            "csapi.procedure",
		EntityPattern:   "*.*.*.csapi.procedure.*",
		BirthPredicates: []string{"sensorml.process.type"},
		Groups: []contract.PredicateGroup{{
			Name: "representation",
			Mode: contract.ModeReconcile,
			Predicates: []string{
				"sensorml.capability.value",
				"sensorml.characteristic.value",
				"sensorml.identifier.value",
				"sensorml.process.definition",
				"sensorml.process.description",
				"sensorml.process.label",
				"sensorml.process.position",
				"sensorml.process.uid",
			},
		}},
	},
	{
		Name:            "csapi.property",
		EntityPattern:   "*.*.*.csapi.property.*",
		BirthPredicates: []string{"sensorml.process.type"},
		Groups: []contract.PredicateGroup{{
			Name: "representation",
			Mode: contract.ModeReconcile,
			Predicates: []string{
				"sensorml.process.description",
				"sensorml.process.label",
				"sensorml.process.uid",
				"sensorml.property.base-property",
				"sensorml.property.definition",
			},
		}},
	},
	{
		Name:            "csapi.samplingfeature",
		EntityPattern:   "*.*.*.csapi.samplingfeature.*",
		BirthPredicates: []string{"sensorml.process.type"},
		Groups: []contract.PredicateGroup{{
			Name: "representation",
			Mode: contract.ModeReconcile,
			Predicates: []string{
				"sensorml.component.is-hosted-by",
				"sensorml.process.definition",
				"sensorml.process.description",
				"sensorml.process.label",
				"sensorml.process.position",
				"sensorml.process.uid",
				"sensorml.sampling.hosted-procedure",
				"sensorml.sampling.sampled-feature",
			},
		}},
	},
	{
		Name:            "csapi.system",
		EntityPattern:   "*.*.*.csapi.system.*",
		BirthPredicates: []string{"sensorml.process.type"},
		Groups: []contract.PredicateGroup{{
			Name: "representation",
			Mode: contract.ModeReconcile,
			Predicates: []string{
				"sensorml.capability.value",
				"sensorml.characteristic.value",
				"sensorml.component.is-hosted-by",
				"sensorml.identifier.value",
				"sensorml.process.attached-to",
				"sensorml.process.definition",
				"sensorml.process.description",
				"sensorml.process.has-sub-system",
				"sensorml.process.label",
				"sensorml.process.position",
				"sensorml.process.uid",
				"sensorml.process.used-procedure",
				"sensorml.system.hosts",
			},
		}},
	},
}

func sensormlInputs(t *testing.T) (vocabulary.PackSpec, shacl.ShapeSet, shacl.Mapping) {
	t.Helper()
	spec, err := smlpack.Spec()
	if err != nil {
		t.Fatal(err)
	}
	shapesBytes := smlpack.Shapes()
	if len(shapesBytes) == 0 {
		t.Fatal("sensorml.Shapes() is empty")
	}
	shapes, err := shacl.ParseTurtle(bytes.NewReader(shapesBytes))
	if err != nil {
		t.Fatal(err)
	}
	m, err := shacl.DecodeMappingYAML(bytes.NewReader(smlpack.Mapping()))
	if err != nil {
		t.Fatal(err)
	}
	return spec, shapes, m
}

func compileSensorML(t *testing.T) *shacl.Bundle {
	t.Helper()
	spec, shapes, m := sensormlInputs(t)
	b, err := shacl.Compile(spec, shapes, m)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func observeSensorML(t *testing.T) registryState {
	t.Helper()
	s := observe(t)
	out, err := export.SerializeToString([]message.Triple{
		{Subject: "c360.semconnect.systems.csapi.system.s1", Predicate: sensorml.Namespace + "probe", Object: "x"},
		{Subject: "c360.semconnect.systems.csapi.system.s1", Predicate: swe.Namespace + "probe", Object: "x"},
		{Subject: "c360.semconnect.systems.csapi.system.s1", Predicate: ssn.SystemsNamespace + "probe", Object: "x"},
		{Subject: "c360.semconnect.systems.csapi.system.s1", Predicate: sosa.Namespace + "probe", Object: "x"},
	}, export.Turtle)
	if err != nil {
		t.Fatal(err)
	}
	s.turtle += "\n" + out
	return s
}

// TestSensorMLBundle verifies that packs/sensorml compiles cleanly from Turtle,
// JSON, and YAML, matches Spec() identically, emits 5 valid reconcile contracts
// with zero unwritable types, and matches semconnect EntityIDs.
func TestSensorMLBundle(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "packs", "sensorml", "sensorml.mapping.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, smlpack.Mapping()) {
		t.Fatal("sensorml.Mapping() is not the committed sensorml.mapping.yaml")
	}

	spec, _, m := sensormlInputs(t)
	b := compileSensorML(t)
	if got := b.Pack(); !reflect.DeepEqual(got, spec) {
		t.Errorf("b.Pack() differs from Spec():\n got %+v\nwant %+v", got, spec)
	}
	pack := b.Pack()
	if err := pack.Validate(); err != nil {
		t.Fatalf("compiled pack fails Validate: %v", err)
	}
	bad := b.Pack()
	bad.Predicates[0].Name = "sensorml.process.bad_type"
	if err := bad.Validate(); err == nil {
		t.Error("Validate accepted invalid predicate name sensorml.process.bad_type")
	}

	if got := b.Contracts(); !reflect.DeepEqual(got, wantSensorMLContracts) {
		t.Fatalf("Contracts() =\n%+v\nwant\n%+v", got, wantSensorMLContracts)
	}
	for _, c := range b.Contracts() {
		if err := c.ValidateShape(); err != nil {
			t.Errorf("contract %s fails ValidateShape: %v", c.Name, err)
		}
		_, typ, _ := strings.Cut(c.Name, ".")
		matchID := "c360.semconnect.systems.csapi." + typ + ".x1"
		missID := "c360.semconnect.systems.other." + typ + ".x1"
		if ok, err := semtypes.MatchEntityIDPattern(c.EntityPattern, matchID); err != nil || !ok {
			t.Errorf("MatchEntityIDPattern(%s, %s) = %v, %v; want true", c.EntityPattern, matchID, ok, err)
		}
		if ok, err := semtypes.MatchEntityIDPattern(c.EntityPattern, missID); err != nil || ok {
			t.Errorf("MatchEntityIDPattern(%s, %s) = %v, %v; want false", c.EntityPattern, missID, ok, err)
		}
	}
	if got := b.Unwritable(); len(got) != 0 {
		t.Errorf("Unwritable() = %v, want empty", got)
	}

	for _, name := range []string{"sensorml.shapes.json", "sensorml.shapes.yaml"} {
		src, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		decode := shacl.DecodeShapesJSON
		if strings.HasSuffix(name, ".yaml") {
			decode = shacl.DecodeShapesYAML
		}
		shapes, err := decode(bytes.NewReader(src))
		if err != nil {
			t.Fatalf("%s decode: %v", name, err)
		}
		other, err := shacl.Compile(spec, shapes, m)
		if err != nil {
			t.Fatalf("%s compile: %v", name, err)
		}
		if !reflect.DeepEqual(other, b) {
			t.Errorf("%s compiles to a different bundle from the Turtle", name)
		}
	}
}

// TestSensorMLCompileAndValidateSideEffectFree verifies the spec scenario
// "Compiling packs/sensorml is side-effect free and matches BasePack".
func TestSensorMLCompileAndValidateSideEffectFree(t *testing.T) {
	before := observeSensorML(t)

	b := compileSensorML(t)
	spec, shapes, m := sensormlInputs(t)
	m.Predicates = m.Predicates[1:]
	if _, err := shacl.Compile(spec, shapes, m); err == nil {
		t.Fatal("a compile with an unmapped sensorml path succeeded")
	}

	const sysID = "c360.semconnect.systems.csapi.system.s1"
	rep, err := b.ValidateCandidateState(sysID, []message.Triple{
		{Subject: sysID, Predicate: "sensorml.process.type", Object: "PhysicalSystem"},
		{Subject: sysID, Predicate: "sensorml.process.uid", Object: "urn:x-ogc:systems:001"},
		{Subject: sysID, Predicate: "sensorml.process.label", Object: "Station 1"},
	})
	if err != nil || !rep.Targeted || !rep.Conforms {
		t.Fatalf("ValidateCandidateState = %+v, %v; want targeted and conforming", rep, err)
	}

	after := observeSensorML(t)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("compiling or validating sensorml mutated SemEngine global registry or prefix state")
	}
}

func conformantSensorMLCandidates() []struct {
	name     string
	entityID string
	triples  []message.Triple
} {
	const (
		sysID  = "c360.semconnect.systems.csapi.system.s1"
		procID = "c360.semconnect.systems.csapi.procedure.p1"
		depID  = "c360.semconnect.systems.csapi.deployment.d1"
		sfID   = "c360.semconnect.systems.csapi.samplingfeature.sf1"
		propID = "c360.semconnect.systems.csapi.property.pr1"
	)
	return []struct {
		name     string
		entityID string
		triples  []message.Triple
	}{
		{
			name:     "csapi.system minimal",
			entityID: sysID,
			triples: []message.Triple{
				{Subject: sysID, Predicate: "sensorml.process.type", Object: "PhysicalSystem"},
				{Subject: sysID, Predicate: "sensorml.process.uid", Object: "urn:x-ogc:systems:001"},
				{Subject: sysID, Predicate: "sensorml.process.label", Object: "Weather Station Alpha"},
			},
		},
		{
			name:     "csapi.system rich with polymorphic values and open framework predicates",
			entityID: sysID,
			triples: []message.Triple{
				{Subject: sysID, Predicate: "sensorml.process.type", Object: sensorml.PhysicalSystem},
				{Subject: sysID, Predicate: "sensorml.process.uid", Object: "urn:x-ogc:systems:001"},
				{Subject: sysID, Predicate: "sensorml.process.label", Object: "Weather Station Alpha"},
				{Subject: sysID, Predicate: "sensorml.process.description", Object: "Primary station\nPier 4"},
				{Subject: sysID, Predicate: "sensorml.process.definition", Object: "http://www.opengis.net/def/system/weather-station"},
				{Subject: sysID, Predicate: "sensorml.process.position", Object: `{"type":"Point","coordinates":[-117.16,32.71]}`},
				{Subject: sysID, Predicate: "sensorml.system.hosts", Object: "c360.semconnect.systems.csapi.system.sub1"},
				{Subject: sysID, Predicate: "sensorml.component.is-hosted-by", Object: "c360.semconnect.systems.csapi.system.parent1"},
				{Subject: sysID, Predicate: "sensorml.process.has-sub-system", Object: "c360.semconnect.systems.csapi.system.sub2"},
				{Subject: sysID, Predicate: "sensorml.process.used-procedure", Object: "urn:x-ogc:procedures:001"},
				{Subject: sysID, Predicate: "sensorml.process.attached-to", Object: "urn:x-ogc:platforms:buoy-7"},
				{Subject: sysID, Predicate: "sensorml.identifier.value", Object: "SN-42"},
				{Subject: sysID, Predicate: "sensorml.identifier.value", Object: int64(42)},
				{Subject: sysID, Predicate: "sensorml.capability.value", Object: 12.5},
				{Subject: sysID, Predicate: "sensorml.capability.value", Object: int64(100)},
				{Subject: sysID, Predicate: "sensorml.capability.value", Object: true},
				{Subject: sysID, Predicate: "sensorml.capability.value", Object: "nominal"},
				{Subject: sysID, Predicate: "sensorml.characteristic.value", Object: 2.75},
				{Subject: sysID, Predicate: "sensorml.characteristic.value", Object: "IP67"},
				{Subject: sysID, Predicate: vocabulary.EntityIndexingProfile, Object: vocabulary.IndexingProfileControl},
				{Subject: sysID, Predicate: vocabulary.GeoLocationLatitude, Object: 32.71},
			},
		},
		{
			name:     "csapi.procedure without position",
			entityID: procID,
			triples: []message.Triple{
				{Subject: procID, Predicate: "sensorml.process.type", Object: "SimpleProcess"},
				{Subject: procID, Predicate: "sensorml.process.uid", Object: "urn:x-ogc:procedures:001"},
				{Subject: procID, Predicate: "sensorml.process.label", Object: "QAQC Procedure"},
				{Subject: procID, Predicate: "sensorml.capability.value", Object: 0.01},
				{Subject: procID, Predicate: "sensorml.characteristic.value", Object: "v1"},
			},
		},
		{
			name:     "csapi.deployment with position and parent",
			entityID: depID,
			triples: []message.Triple{
				{Subject: depID, Predicate: "sensorml.process.type", Object: "Deployment"},
				{Subject: depID, Predicate: "sensorml.process.uid", Object: "urn:x-ogc:deployments:001"},
				{Subject: depID, Predicate: "sensorml.process.label", Object: "Harbor Deployment"},
				{Subject: depID, Predicate: "sensorml.process.position", Object: `{"type":"Point","coordinates":[-117.2,32.7]}`},
				{Subject: depID, Predicate: "sensorml.deployment.deployed-system", Object: "urn:x-ogc:systems:001"},
				{Subject: depID, Predicate: "sensorml.deployment.parent", Object: "c360.semconnect.systems.csapi.deployment.root"},
			},
		},
		{
			name:     "csapi.samplingfeature omitting optional label",
			entityID: sfID,
			triples: []message.Triple{
				{Subject: sfID, Predicate: "sensorml.process.type", Object: sosa.Sample},
				{Subject: sfID, Predicate: "sensorml.process.uid", Object: "urn:x-ogc:sf:station-01"},
				{Subject: sfID, Predicate: "sensorml.sampling.sampled-feature", Object: "http://example.org/features/san-diego-bay"},
				{Subject: sfID, Predicate: "sensorml.sampling.hosted-procedure", Object: "urn:x-ogc:procedures:001"},
				{Subject: sfID, Predicate: "sensorml.component.is-hosted-by", Object: sysID},
				{Subject: sfID, Predicate: "sensorml.process.position", Object: `{"type":"Point","coordinates":[-117.18,32.72]}`},
			},
		},
		{
			name:     "csapi.property omitting optional uid",
			entityID: propID,
			triples: []message.Triple{
				{Subject: propID, Predicate: "sensorml.process.type", Object: "DerivedProperty"},
				{Subject: propID, Predicate: "sensorml.process.label", Object: "Hourly Mean Sea Surface Temperature"},
				{Subject: propID, Predicate: "sensorml.property.base-property", Object: "https://qudt.org/vocab/quantitykind/Temperature"},
				{Subject: propID, Predicate: "sensorml.property.definition", Object: "http://sensorml.com/ont/x-stats/HourlyMean"},
			},
		},
	}
}

func TestSensorMLValidateCandidateStateConformant(t *testing.T) {
	b := compileSensorML(t)
	for _, tc := range conformantSensorMLCandidates() {
		t.Run(tc.name, func(t *testing.T) {
			rep, err := b.ValidateCandidateState(tc.entityID, tc.triples)
			if err != nil {
				t.Fatalf("ValidateCandidateState error: %v", err)
			}
			if !rep.Targeted || !rep.Conforms || len(rep.Results) != 0 {
				t.Fatalf("report = %+v, want Targeted=true Conforms=true Results=[]", rep)
			}
		})
	}
}

type violationCase struct {
	name      string
	entityID  string
	triples   []message.Triple
	component string
	predicate string
	path      string
}

func nonConformantSensorMLCandidates() []violationCase {
	const (
		sysID  = "c360.semconnect.systems.csapi.system.s1"
		procID = "c360.semconnect.systems.csapi.procedure.p1"
		depID  = "c360.semconnect.systems.csapi.deployment.d1"
		sfID   = "c360.semconnect.systems.csapi.samplingfeature.sf1"
		propID = "c360.semconnect.systems.csapi.property.pr1"
	)
	return []violationCase{
		{
			name:     "csapi.procedure prohibits sml:position via sh:maxCount 0",
			entityID: procID,
			triples: []message.Triple{
				{Subject: procID, Predicate: "sensorml.process.type", Object: "SimpleProcess"},
				{Subject: procID, Predicate: "sensorml.process.uid", Object: "urn:x-ogc:procedures:001"},
				{Subject: procID, Predicate: "sensorml.process.label", Object: "QAQC Procedure"},
				{Subject: procID, Predicate: "sensorml.process.position", Object: `{"type":"Point","coordinates":[0,0]}`},
			},
			component: shaclNS + "MaxCountConstraintComponent",
			predicate: "sensorml.process.position",
			path:      sensorml.PropPosition,
		},
		{
			name:     "csapi.system rejects malformed JSON on sml:position",
			entityID: sysID,
			triples: []message.Triple{
				{Subject: sysID, Predicate: "sensorml.process.type", Object: "PhysicalSystem"},
				{Subject: sysID, Predicate: "sensorml.process.uid", Object: "urn:x-ogc:systems:001"},
				{Subject: sysID, Predicate: "sensorml.process.label", Object: "Weather Station Alpha"},
				{Subject: sysID, Predicate: "sensorml.process.position", Object: "not-json{"},
			},
			component: shaclNS + "DatatypeConstraintComponent",
			predicate: "sensorml.process.position",
			path:      sensorml.PropPosition,
		},
		{
			name:     "csapi.system missing required uid",
			entityID: sysID,
			triples: []message.Triple{
				{Subject: sysID, Predicate: "sensorml.process.type", Object: "PhysicalSystem"},
				{Subject: sysID, Predicate: "sensorml.process.label", Object: "Weather Station Alpha"},
			},
			component: shaclNS + "MinCountConstraintComponent",
			predicate: "sensorml.process.uid",
			path:      vocabulary.DcIdentifier,
		},
		{
			name:     "csapi.system missing required label",
			entityID: sysID,
			triples: []message.Triple{
				{Subject: sysID, Predicate: "sensorml.process.type", Object: "PhysicalSystem"},
				{Subject: sysID, Predicate: "sensorml.process.uid", Object: "urn:x-ogc:systems:001"},
			},
			component: shaclNS + "MinCountConstraintComponent",
			predicate: "sensorml.process.label",
			path:      vocabulary.DcTitle,
		},
		{
			name:     "csapi.samplingfeature missing sampled-feature",
			entityID: sfID,
			triples: []message.Triple{
				{Subject: sfID, Predicate: "sensorml.process.type", Object: sosa.Sample},
				{Subject: sfID, Predicate: "sensorml.process.uid", Object: "urn:x-ogc:sf:001"},
			},
			component: shaclNS + "MinCountConstraintComponent",
			predicate: "sensorml.sampling.sampled-feature",
			path:      sosa.IsSampleOf,
		},
		{
			name:     "csapi.property missing base-property",
			entityID: propID,
			triples: []message.Triple{
				{Subject: propID, Predicate: "sensorml.process.type", Object: "DerivedProperty"},
				{Subject: propID, Predicate: "sensorml.process.label", Object: "Derived Temp"},
			},
			component: shaclNS + "MinCountConstraintComponent",
			predicate: "sensorml.property.base-property",
			path:      sensorml.PropBaseProperty,
		},
		{
			name:     "empty string on uid fails pattern",
			entityID: sysID,
			triples: []message.Triple{
				{Subject: sysID, Predicate: "sensorml.process.type", Object: "PhysicalSystem"},
				{Subject: sysID, Predicate: "sensorml.process.uid", Object: ""},
				{Subject: sysID, Predicate: "sensorml.process.label", Object: "Weather Station Alpha"},
			},
			component: shaclNS + "PatternConstraintComponent",
			predicate: "sensorml.process.uid",
			path:      vocabulary.DcIdentifier,
		},
		{
			name:     "whitespace in uid fails anchored pattern",
			entityID: sysID,
			triples: []message.Triple{
				{Subject: sysID, Predicate: "sensorml.process.type", Object: "PhysicalSystem"},
				{Subject: sysID, Predicate: "sensorml.process.uid", Object: "urn:bad id"},
				{Subject: sysID, Predicate: "sensorml.process.label", Object: "Weather Station Alpha"},
			},
			component: shaclNS + "PatternConstraintComponent",
			predicate: "sensorml.process.uid",
			path:      vocabulary.DcIdentifier,
		},
		{
			name:     "empty string on label fails pattern",
			entityID: sysID,
			triples: []message.Triple{
				{Subject: sysID, Predicate: "sensorml.process.type", Object: "PhysicalSystem"},
				{Subject: sysID, Predicate: "sensorml.process.uid", Object: "urn:x-ogc:systems:001"},
				{Subject: sysID, Predicate: "sensorml.process.label", Object: ""},
			},
			component: shaclNS + "PatternConstraintComponent",
			predicate: "sensorml.process.label",
			path:      vocabulary.DcTitle,
		},
		{
			name:     "duplicate deployment parent exceeds maxCount 1",
			entityID: depID,
			triples: []message.Triple{
				{Subject: depID, Predicate: "sensorml.process.type", Object: "Deployment"},
				{Subject: depID, Predicate: "sensorml.process.uid", Object: "urn:x-ogc:deployments:001"},
				{Subject: depID, Predicate: "sensorml.process.label", Object: "Sub Deployment"},
				{Subject: depID, Predicate: "sensorml.deployment.parent", Object: "c360.semconnect.systems.csapi.deployment.p1"},
				{Subject: depID, Predicate: "sensorml.deployment.parent", Object: "c360.semconnect.systems.csapi.deployment.p2"},
			},
			component: shaclNS + "MaxCountConstraintComponent",
			predicate: "sensorml.deployment.parent",
			path:      vocabulary.SkosBroader,
		},
	}
}

func TestSensorMLValidateCandidateStateViolations(t *testing.T) {
	b := compileSensorML(t)
	for _, tc := range nonConformantSensorMLCandidates() {
		t.Run(tc.name, func(t *testing.T) {
			rep, err := b.ValidateCandidateState(tc.entityID, tc.triples)
			if err != nil {
				t.Fatalf("ValidateCandidateState error: %v", err)
			}
			if !rep.Targeted || rep.Conforms {
				t.Fatalf("report = %+v, want Targeted=true Conforms=false", rep)
			}
			if !slices.ContainsFunc(rep.Results, func(r shacl.Result) bool {
				return r.Component == tc.component && r.Predicate == tc.predicate && r.Path == tc.path
			}) {
				t.Fatalf("results %+v lack Component=%s Predicate=%s Path=%s",
					rep.Results, tc.component, tc.predicate, tc.path)
			}
		})
	}
}

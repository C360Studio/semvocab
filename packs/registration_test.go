package packs_test

// SemEngine's vocabulary and export registries are process-global, and
// task test:repeat runs these tests five times in one process. Each test
// therefore asserts the registry state it causes rather than restoring it.
// Test-local entries use a key unique to the invocation (uniqueKey), so a
// repeated run never sees an earlier run's entries; the embedded bfo-cco-core,
// swe-core, diekm, and sensorml packs are idempotent to re-register, and no
// test leaves a DIEKM or SensorML entry different from its pack once it
// finishes.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/c360studio/semengine/message"
	"github.com/c360studio/semengine/vocabulary"
	"github.com/c360studio/semengine/vocabulary/export"
	"github.com/c360studio/semvocab/catalog/bfo"
	"github.com/c360studio/semvocab/catalog/cco"
	"github.com/c360studio/semvocab/catalog/sensorml"
	"github.com/c360studio/semvocab/catalog/sosa"
	"github.com/c360studio/semvocab/catalog/ssn"
	"github.com/c360studio/semvocab/catalog/swe"
	"github.com/c360studio/semvocab/internal/turtle"
	"github.com/c360studio/semvocab/packs"
	"github.com/c360studio/semvocab/packs/diekm"
	smlpack "github.com/c360studio/semvocab/packs/sensorml"
)

const (
	testNS     = "http://example.org/semvocab-test#"
	diekmNS    = "https://w3id.org/c360/diekm#"
	xsdNS      = "http://www.w3.org/2001/XMLSchema#"
	platformID = "acme.edge.sensor.intel.platform.p1"
	observeID  = "acme.edge.sensor.intel.observation.o1"
)

var invocation atomic.Int64

// uniqueKey returns a lower-kebab segment unique to this process's call.
func uniqueKey(base string) string {
	return fmt.Sprintf("%s%d", base, invocation.Add(1))
}

// registryState is what a caller can observe of SemEngine's global state:
// every registered predicate and entity type, and the serializer's prefix
// output for probe IRIs in each namespace of interest.
type registryState struct {
	predicates  map[string]vocabulary.PredicateMetadata
	entityTypes map[string]vocabulary.EntityTypeMetadata
	prefixes    string
}

func observe(t *testing.T, namespaces ...string) registryState {
	t.Helper()
	s := registryState{
		predicates:  map[string]vocabulary.PredicateMetadata{},
		entityTypes: map[string]vocabulary.EntityTypeMetadata{},
	}
	for _, n := range vocabulary.ListRegisteredPredicates() {
		s.predicates[n] = *vocabulary.GetPredicateMetadata(n)
	}
	for _, k := range vocabulary.ListRegisteredEntityTypes() {
		s.entityTypes[k] = *vocabulary.GetEntityTypeMetadata(k)
	}
	var probe []message.Triple
	defaultProbeNS := []string{
		bfo.Namespace,
		cco.Namespace,
		diekmNS,
		sosa.Namespace,
		ssn.Namespace,
		ssn.SystemsNamespace,
		swe.Namespace,
		sensorml.Namespace,
	}
	for _, ns := range append(defaultProbeNS, namespaces...) {
		probe = append(probe, message.Triple{Subject: platformID, Predicate: ns + "probe", Object: "x"})
	}
	out, err := export.SerializeToString(probe, export.Turtle)
	if err != nil {
		t.Fatal(err)
	}
	s.prefixes = out
	return s
}

func assertUnchanged(t *testing.T, before, after registryState) {
	t.Helper()
	if !reflect.DeepEqual(before.predicates, after.predicates) {
		t.Error("predicate registry changed")
	}
	if !reflect.DeepEqual(before.entityTypes, after.entityTypes) {
		t.Error("entity-type registry changed")
	}
	if before.prefixes != after.prefixes {
		t.Errorf("serializer prefixes changed:\nbefore:\n%s\nafter:\n%s", before.prefixes, after.prefixes)
	}
}

func diekmSpec(t *testing.T) vocabulary.PackSpec {
	t.Helper()
	spec, err := diekm.Spec()
	if err != nil {
		t.Fatal(err)
	}
	if spec.EntityTypes[0].DomainType != "intel.platform" {
		t.Fatalf("seed expects intel.platform first, got %s", spec.EntityTypes[0].DomainType)
	}
	return spec
}

func sensormlSpec(t *testing.T) vocabulary.PackSpec {
	t.Helper()
	spec, err := smlpack.Spec()
	if err != nil {
		t.Fatal(err)
	}
	if spec.EntityTypes[0].DomainType != "csapi.system" {
		t.Fatalf("seed expects csapi.system first, got %s", spec.EntityTypes[0].DomainType)
	}
	return spec
}

func registerWithPrefixes(spec vocabulary.PackSpec) error {
	return vocabulary.RegisterPack(spec, vocabulary.WithPrefixRegistrar(export.RegisterPrefixes))
}

func TestBFOCCOCorePackLoadsAndRegisters(t *testing.T) {
	data, err := os.ReadFile("bfo-cco-core.yaml")
	if err != nil {
		t.Fatal(err)
	}
	spec, err := vocabulary.LoadPackYAML(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if embedded, err := packs.BFOCCOCoreSpec(); err != nil || !reflect.DeepEqual(embedded, spec) {
		t.Fatalf("BFOCCOCoreSpec() = %+v, %v; want the committed YAML %+v", embedded, err, spec)
	}
	want := []vocabulary.PrefixSpec{{Prefix: "obo", Namespace: bfo.Namespace}, {Prefix: "cco", Namespace: cco.Namespace}}
	if !slices.Equal(spec.Prefixes, want) || len(spec.Predicates)+len(spec.EntityTypes) != 0 {
		t.Fatalf("bfo-cco-core = %+v, want only prefixes %v (design P2, P5)", spec, want)
	}
	if err := registerWithPrefixes(spec); err != nil {
		t.Fatal(err)
	}
	if err := packs.RegisterBFOCCOCore(); err != nil {
		t.Fatal(err)
	}
	out, err := export.SerializeToString([]message.Triple{
		{Subject: platformID, Predicate: bfo.ParticipatesIn, Object: observeID, Datatype: message.EntityReferenceDatatype},
		{Subject: platformID, Predicate: cco.IsAbout, Object: observeID, Datatype: message.EntityReferenceDatatype},
	}, export.Turtle)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"@prefix obo: <" + bfo.Namespace + "> .", "@prefix cco: <" + cco.Namespace + "> .",
		"obo:BFO_0000056", "cco:ont00001808"} {
		if !strings.Contains(out, s) {
			t.Errorf("Turtle output lacks %q:\n%s", s, out)
		}
	}
}

func TestSWECorePackLoadsAndRegisters(t *testing.T) {
	data, err := os.ReadFile("swe-core.yaml")
	if err != nil {
		t.Fatal(err)
	}
	spec, err := vocabulary.LoadPackYAML(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if embedded, err := packs.SWECoreSpec(); err != nil || !reflect.DeepEqual(embedded, spec) {
		t.Fatalf("SWECoreSpec() = %+v, %v; want the committed YAML %+v", embedded, err, spec)
	}
	want := []vocabulary.PrefixSpec{
		{Prefix: "sosa", Namespace: sosa.Namespace},
		{Prefix: "ssn", Namespace: ssn.Namespace},
		{Prefix: "ssn-system", Namespace: ssn.SystemsNamespace},
		{Prefix: "swe", Namespace: swe.Namespace},
		{Prefix: "sml", Namespace: sensorml.Namespace},
	}
	if !slices.Equal(spec.Prefixes, want) || len(spec.Predicates)+len(spec.EntityTypes) != 0 {
		t.Fatalf("swe-core = %+v, want only prefixes %v", spec, want)
	}
	if err := registerWithPrefixes(spec); err != nil {
		t.Fatal(err)
	}
	if err := packs.RegisterSWECore(); err != nil {
		t.Fatal(err)
	}
	triples := []message.Triple{
		{Subject: platformID, Predicate: sosa.Observes, Object: sosa.ObservableProperty},
		{Subject: platformID, Predicate: ssn.HasSubSystem, Object: observeID, Datatype: message.EntityReferenceDatatype},
		{Subject: platformID, Predicate: ssn.HasSystemCapability, Object: ssn.Accuracy},
		{Subject: platformID, Predicate: swe.PropUOM, Object: swe.Quantity},
		{Subject: platformID, Predicate: sensorml.PropLink, Object: sensorml.PhysicalSystem},
	}
	turtleOut, err := export.SerializeToString(triples, export.Turtle)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{
		"@prefix sosa: <" + sosa.Namespace + "> .",
		"@prefix ssn: <" + ssn.Namespace + "> .",
		"@prefix ssn-system: <" + ssn.SystemsNamespace + "> .",
		"@prefix swe: <" + swe.Namespace + "> .",
		"@prefix sml: <" + sensorml.Namespace + "> .",
		"sosa:observes sosa:ObservableProperty",
		"ssn:hasSubSystem",
		"ssn-system:hasSystemCapability ssn-system:Accuracy",
		"swe:uom swe:Quantity",
		"sml:link sml:PhysicalSystem",
	} {
		if !strings.Contains(turtleOut, s) {
			t.Errorf("Turtle output lacks %q:\n%s", s, turtleOut)
		}
	}
	if strings.Contains(turtleOut, "ssn:systems/") || strings.Contains(turtleOut, "<"+ssn.SystemsNamespace+"Accuracy>") {
		t.Errorf("ssn-system IRI was not compacted under ssn-system:\n%s", turtleOut)
	}

	jsonldOut, err := export.SerializeToString(triples, export.JSONLD)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{
		`"sosa": "` + sosa.Namespace + `"`,
		`"ssn": "` + ssn.Namespace + `"`,
		`"ssn-system": "` + ssn.SystemsNamespace + `"`,
		`"swe": "` + swe.Namespace + `"`,
		`"sml": "` + sensorml.Namespace + `"`,
		`"sosa:observes"`,
		`"ssn:hasSubSystem"`,
		`"ssn-system:hasSystemCapability"`,
		`"swe:uom"`,
		`"sml:link"`,
	} {
		if !strings.Contains(jsonldOut, s) {
			t.Errorf("JSON-LD output lacks %q:\n%s", s, jsonldOut)
		}
	}
	if strings.Contains(jsonldOut, `"ssn:systems/`) {
		t.Errorf("JSON-LD compacted ssn-system predicate under ssn:\n%s", jsonldOut)
	}
	if got, wantGraph := graphFromJSONLD(t, jsonldOut), graphFromTurtle(t, turtleOut); !slices.Equal(got, wantGraph) {
		t.Errorf("JSON-LD graph differs from Turtle graph:\n got: %q\nwant: %q", got, wantGraph)
	}
}

// localPack is a test-only pack with one prefix, entity type, and predicate,
// all under keys unique to the call.
func localPack(key, classIRI string, superclasses ...string) vocabulary.PackSpec {
	return vocabulary.PackSpec{
		Name:     key,
		Prefixes: []vocabulary.PrefixSpec{{Prefix: "semvocabtest", Namespace: testNS}},
		EntityTypes: []vocabulary.EntityTypeSpec{{
			DomainType: "semvocabtest." + key, ClassIRI: classIRI, Superclasses: superclasses,
			Description: "class " + classIRI,
		}},
		Predicates: []vocabulary.PredicateSpec{{
			Name: "semvocabtest." + key + ".value", DataType: "float", IRI: classIRI + "-value",
		}},
	}
}

// TestPackRepeatedRegistration: registering the same pack again leaves every
// observable registry and prefix state equal (design D8).
func TestPackRepeatedRegistration(t *testing.T) {
	key := uniqueKey("repeat")
	for _, register := range []func() error{
		func() error { return packs.RegisterBFOCCOCore() },
		func() error { return packs.RegisterSWECore() },
		func() error { return diekm.Register() },
		func() error { return smlpack.Register() },
		func() error { return registerWithPrefixes(localPack(key, testNS+key, cco.Vehicle)) },
	} {
		if err := register(); err != nil {
			t.Fatal(err)
		}
		before := observe(t, testNS)
		if err := register(); err != nil {
			t.Fatal(err)
		}
		assertUnchanged(t, before, observe(t, testNS))
	}
	if got := vocabulary.GetEntityTypeMetadata("semvocabtest." + key); got == nil ||
		!slices.Equal(got.Superclasses, []string{cco.Vehicle}) {
		t.Errorf("repeated registration changed superclasses: %+v", got)
	}
}

// TestSWECoreAndSensorMLCoexist verifies the catalogs-and-packs scenario
// "swe-core and sensorml coexist without prefix conflict" across both
// registration orders.
func TestSWECoreAndSensorMLCoexist(t *testing.T) {
	if err := packs.RegisterSWECore(); err != nil {
		t.Fatalf("RegisterSWECore first: %v", err)
	}
	if err := smlpack.Register(); err != nil {
		t.Fatalf("sensorml.Register after RegisterSWECore: %v", err)
	}
	afterForward := observe(t)

	if err := smlpack.Register(); err != nil {
		t.Fatalf("sensorml.Register first: %v", err)
	}
	if err := packs.RegisterSWECore(); err != nil {
		t.Fatalf("RegisterSWECore after sensorml.Register: %v", err)
	}
	assertUnchanged(t, afterForward, observe(t))

	triples := []message.Triple{
		{Subject: platformID, Predicate: "sensorml.process.label", Object: "Station 1"},
		{Subject: platformID, Predicate: "sensorml.process.position", Object: `{"type":"Point","coordinates":[1,2]}`},
		{Subject: platformID, Predicate: "sensorml.capability.value", Object: 5.0},
		{Subject: platformID, Predicate: swe.PropValue, Object: 9.5},
	}
	turtleOut, err := export.SerializeToString(triples, export.Turtle)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{
		"@prefix dc: <http://purl.org/dc/terms/> .",
		"@prefix ssn-system: <" + ssn.SystemsNamespace + "> .",
		"@prefix swe: <" + swe.Namespace + "> .",
		"@prefix sml: <" + sensorml.Namespace + "> .",
		"dc:title",
		"sml:position",
		"ssn-system:hasSystemCapability",
		"swe:value",
	} {
		if !strings.Contains(turtleOut, s) {
			t.Errorf("Turtle output lacks %q:\n%s", s, turtleOut)
		}
	}
	if strings.Contains(turtleOut, "dcterms:") || strings.Contains(turtleOut, "ssn:systems/") {
		t.Errorf("Turtle output contains ambiguous prefix:\n%s", turtleOut)
	}
}

// TestPackConflictingRegistration records SemEngine's re-registration
// semantics at the pin (design D8): a conflicting prefix binding fails and
// changes nothing; a conflicting class IRI or predicate IRI is
// last-writer-wins, and superclasses are unioned in registration order.
func TestPackConflictingRegistration(t *testing.T) {
	if err := packs.RegisterBFOCCOCore(); err != nil {
		t.Fatal(err)
	}
	if err := packs.RegisterSWECore(); err != nil {
		t.Fatal(err)
	}

	t.Run("prefix rebinding fails atomically", func(t *testing.T) {
		key := uniqueKey("rebind")
		pack := localPack(key, testNS+key)
		pack.Prefixes = append(pack.Prefixes, vocabulary.PrefixSpec{Prefix: "obo", Namespace: "http://example.org/not-obo/"})
		before := observe(t, testNS, "http://example.org/not-obo/")
		err := registerWithPrefixes(pack)
		if err == nil || !strings.Contains(err.Error(), `prefix "obo" already bound`) {
			t.Fatalf("RegisterPack = %v, want the obo rebinding refused", err)
		}
		assertUnchanged(t, before, observe(t, testNS, "http://example.org/not-obo/"))
	})

	t.Run("swe-core prefix rebinding fails atomically", func(t *testing.T) {
		key := uniqueKey("rebindswe")
		pack := localPack(key, testNS+key)
		pack.Prefixes = append(pack.Prefixes, vocabulary.PrefixSpec{Prefix: "ssn-system", Namespace: "http://example.org/not-ssn-system/"})
		before := observe(t, testNS, "http://example.org/not-ssn-system/")
		err := registerWithPrefixes(pack)
		if err == nil || !strings.Contains(err.Error(), `prefix "ssn-system" already bound`) {
			t.Fatalf("RegisterPack = %v, want the ssn-system rebinding refused", err)
		}
		assertUnchanged(t, before, observe(t, testNS, "http://example.org/not-ssn-system/"))
	})

	t.Run("diekm prefix rebinding fails atomically", func(t *testing.T) {
		if err := diekm.Register(); err != nil {
			t.Fatal(err)
		}
		spec := diekmSpec(t)
		spec.Prefixes[slices.IndexFunc(spec.Prefixes, func(p vocabulary.PrefixSpec) bool { return p.Prefix == "diekm" })].Namespace = "https://example.org/other-diekm#"
		spec.EntityTypes[0].Description = "rebound"
		before := observe(t, "https://example.org/other-diekm#")
		err := registerWithPrefixes(spec)
		if err == nil || !strings.Contains(err.Error(), `prefix "diekm" already bound`) {
			t.Fatalf("RegisterPack = %v, want the diekm rebinding refused", err)
		}
		assertUnchanged(t, before, observe(t, "https://example.org/other-diekm#"))
	})

	t.Run("diekm scalar last writer wins", func(t *testing.T) {
		if err := diekm.Register(); err != nil {
			t.Fatal(err)
		}
		original := vocabulary.GetEntityTypeMetadata("intel.platform")
		spec := diekmSpec(t)
		spec.EntityTypes[0].Description = "overwritten"
		if err := registerWithPrefixes(spec); err != nil {
			t.Fatalf("conflicting description: RegisterPack = %v, want success (D8)", err)
		}
		if got := vocabulary.GetEntityTypeMetadata("intel.platform"); got.Description != "overwritten" {
			t.Errorf("description after conflicting registration = %q, want the later value", got.Description)
		}
		// Re-registering the pack is again last writer: the DIEKM value returns.
		if err := diekm.Register(); err != nil {
			t.Fatal(err)
		}
		if got := vocabulary.GetEntityTypeMetadata("intel.platform"); !reflect.DeepEqual(got, original) {
			t.Errorf("after re-registering diekm: %+v, want %+v", got, original)
		}
	})

	t.Run("class IRI last writer wins, superclasses union", func(t *testing.T) {
		key := uniqueKey("conflict")
		first, second := testNS+key+"-a", testNS+key+"-b"
		if err := registerWithPrefixes(localPack(key, first, cco.Vehicle)); err != nil {
			t.Fatal(err)
		}
		if err := registerWithPrefixes(localPack(key, second, bfo.Object)); err != nil {
			t.Fatalf("conflicting class IRI: RegisterPack = %v, want success (D8)", err)
		}
		got := vocabulary.GetEntityTypeMetadata("semvocabtest." + key)
		if got == nil || got.ClassIRI != second || got.Description != "class "+second ||
			!slices.Equal(got.Superclasses, []string{cco.Vehicle, bfo.Object}) {
			t.Fatalf("after conflicting registration: %+v; want class %s and superclasses [Vehicle Object]", got, second)
		}
		if slices.Contains(vocabulary.LookupEntityTypesByClassIRI(first), "semvocabtest."+key) {
			t.Error("the overwritten class IRI still finds the entity type")
		}
		pred := vocabulary.GetPredicateMetadata("semvocabtest." + key + ".value")
		if pred == nil || pred.StandardIRI != second+"-value" {
			t.Errorf("predicate IRI after conflicting registration: %+v, want %s-value", pred, second)
		}
		if got := vocabulary.LookupByIRI(first + "-value"); got != nil {
			t.Errorf("LookupByIRI(old IRI) = %v, want nil", got)
		}
	})
}

// TestPackAtomicFailure: a pack that fails to load or register leaves the
// registries and serializer prefixes as they were. The tests observe state
// before and after; they restore nothing.
func TestPackAtomicFailure(t *testing.T) {
	const atomicNS = "http://example.org/semvocab-atomic/"
	yamlData, err := os.ReadFile("bfo-cco-core.yaml")
	if err != nil {
		t.Fatal(err)
	}
	sweCoreYAML, err := os.ReadFile("swe-core.yaml")
	if err != nil {
		t.Fatal(err)
	}
	diekmYAML, err := os.ReadFile("diekm/diekm.yaml")
	if err != nil {
		t.Fatal(err)
	}
	sensormlYAML, err := os.ReadFile("sensorml/sensorml.yaml")
	if err != nil {
		t.Fatal(err)
	}
	// The DIEKM and SensorML cases change an entity type's description in a pack
	// that then fails; with both packs registered first, a leak would show as a
	// changed entry.
	if err := diekm.Register(); err != nil {
		t.Fatal(err)
	}
	if err := smlpack.Register(); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		attempt func(key string) error
		wantErr string
	}{
		{"unknown YAML field", func(string) error {
			_, err := vocabulary.LoadPackYAML(bytes.NewReader(append(yamlData, []byte("unexpected_field: 1\n")...)))
			return err
		}, "unexpected_field"},
		{"swe-core: unknown YAML field", func(string) error {
			_, err := vocabulary.LoadPackYAML(bytes.NewReader(append(sweCoreYAML, []byte("unexpected_field: 1\n")...)))
			return err
		}, "unexpected_field"},
		{"diekm: unknown YAML field", func(string) error {
			_, err := vocabulary.LoadPackYAML(bytes.NewReader(append(diekmYAML, []byte("unexpected_field: 1\n")...)))
			return err
		}, "unexpected_field"},
		{"diekm: invalid predicate after changed entries", func(key string) error {
			spec := diekmSpec(t)
			spec.EntityTypes[0].Description = "leaked " + key
			spec.Prefixes = append(spec.Prefixes, vocabulary.PrefixSpec{Prefix: "semvocabatomic", Namespace: atomicNS})
			spec.Predicates = append(spec.Predicates, vocabulary.PredicateSpec{Name: "intel.measurement.altitude_m"})
			return registerWithPrefixes(spec)
		}, "segment_character"},
		{"diekm: projection without a terminal value predicate", func(key string) error {
			spec := diekmSpec(t)
			spec.EntityTypes[0].Description = "leaked " + key
			steps := spec.Predicates[1].Projection.Steps
			steps[len(steps)-1].ValuePredicateIRI = ""
			return registerWithPrefixes(spec)
		}, "requires value_predicate_iri"},
		{"sensorml: unknown YAML field", func(string) error {
			_, err := vocabulary.LoadPackYAML(bytes.NewReader(append(sensormlYAML, []byte("unexpected_field: 1\n")...)))
			return err
		}, "unexpected_field"},
		{"sensorml: invalid predicate after changed entries", func(key string) error {
			spec := sensormlSpec(t)
			spec.EntityTypes[0].Description = "leaked " + key
			spec.Prefixes = append(spec.Prefixes, vocabulary.PrefixSpec{Prefix: "semvocabatomic", Namespace: atomicNS})
			spec.Predicates = append(spec.Predicates, vocabulary.PredicateSpec{Name: "sensorml.capability.bad_name"})
			return registerWithPrefixes(spec)
		}, "segment_character"},
		{"sensorml: projection without a terminal value predicate", func(key string) error {
			spec := sensormlSpec(t)
			spec.EntityTypes[0].Description = "leaked " + key
			idx := slices.IndexFunc(spec.Predicates, func(p vocabulary.PredicateSpec) bool {
				return p.Name == "sensorml.capability.value"
			})
			spec.Predicates[idx].Projection.Steps[0].ValuePredicateIRI = ""
			return registerWithPrefixes(spec)
		}, "requires value_predicate_iri"},
		{"sensorml: conflicting sml prefix rebinding", func(key string) error {
			spec := sensormlSpec(t)
			spec.EntityTypes[0].Description = "leaked " + key
			idx := slices.IndexFunc(spec.Prefixes, func(p vocabulary.PrefixSpec) bool { return p.Prefix == "sml" })
			spec.Prefixes[idx].Namespace = "http://example.org/not-sml/"
			return registerWithPrefixes(spec)
		}, `prefix "sml" already bound`},
		{"sensorml: conflicting dc prefix rebinding", func(key string) error {
			spec := sensormlSpec(t)
			spec.EntityTypes[0].Description = "leaked " + key
			idx := slices.IndexFunc(spec.Prefixes, func(p vocabulary.PrefixSpec) bool { return p.Prefix == "dc" })
			spec.Prefixes[idx].Namespace = "http://purl.org/dc/elements/1.1/"
			return registerWithPrefixes(spec)
		}, `prefix "dc" already bound`},
		{"unknown JSON field", func(string) error {
			_, err := vocabulary.LoadPackJSON(strings.NewReader(`{"name": "x", "unexpected_field": 1}`))
			return err
		}, "unexpected_field"},
		{"invalid predicate after valid entries", func(key string) error {
			pack := localPack(key, atomicNS+key)
			pack.Prefixes = []vocabulary.PrefixSpec{{Prefix: "semvocabatomic", Namespace: atomicNS}}
			pack.Predicates = append(pack.Predicates, vocabulary.PredicateSpec{Name: "invalid"})
			return registerWithPrefixes(pack)
		}, `"invalid"`},
		{"invalid superclass IRI", func(key string) error {
			pack := localPack(key, atomicNS+key, "not an iri")
			pack.Prefixes = []vocabulary.PrefixSpec{{Prefix: "semvocabatomic", Namespace: atomicNS}}
			return registerWithPrefixes(pack)
		}, "superclasses[0]"},
		{"prefix registrar refuses", func(key string) error {
			pack := localPack(key, atomicNS+key)
			pack.Prefixes = []vocabulary.PrefixSpec{
				{Prefix: "semvocabatomic", Namespace: atomicNS},
				{Prefix: "rdfs", Namespace: "http://example.org/not-rdfs#"},
			}
			return registerWithPrefixes(pack)
		}, `prefix "rdfs" already bound`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := uniqueKey("atomic")
			before := observe(t, atomicNS, testNS)
			err := tt.attempt(key)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want one naming %s", err, tt.wantErr)
			}
			after := observe(t, atomicNS, testNS)
			assertUnchanged(t, before, after)
			if strings.Contains(after.prefixes, "semvocabatomic") {
				t.Error("the failed pack's prefix reached the serializer")
			}
			if vocabulary.GetEntityTypeMetadata("semvocabtest."+key) != nil {
				t.Error("the failed pack's entity type was registered")
			}
		})
	}
}

// term is a parsed RDF term in comparable form: doubles and integers are
// compared by value, since Turtle, N-Triples, and JSON-LD may write different
// lexical forms of one number.
func term(kind turtle.Kind, value, datatype string) string {
	switch datatype {
	case xsdNS + "double":
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			value = strconv.FormatFloat(f, 'g', -1, 64)
		}
	case xsdNS + "integer":
		if n, err := strconv.ParseInt(value, 10, 64); err == nil {
			value = strconv.FormatInt(n, 10)
		}
	}
	return fmt.Sprintf("%d|%s|%s", kind, value, datatype)
}

func graphFromTurtle(t *testing.T, src string) []string {
	t.Helper()
	triples, err := turtle.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse: %v\n%s", err, src)
	}
	var out []string
	for _, tr := range triples {
		out = append(out, tr.S.Value+" "+tr.P.Value+" "+term(tr.O.Kind, tr.O.Value, tr.O.Datatype))
	}
	slices.Sort(out)
	return out
}

// graphFromJSONLD converts the compacted JSON-LD SemEngine writes (one node
// object or an @graph array; prefix terms in @context) to triples, following
// JSON-LD 1.1 Processing Algorithms §6.2 (IRI expansion of compact IRIs) and
// §8.6 (object to RDF: native numbers with a fraction are xsd:double, whole
// numbers xsd:integer, booleans xsd:boolean). Anything else fails the test.
func graphFromJSONLD(t *testing.T, src string) []string {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(src))
	dec.UseNumber()
	var doc map[string]any
	if err := dec.Decode(&doc); err != nil {
		t.Fatalf("JSON-LD: %v", err)
	}
	ctx := map[string]string{}
	if c, ok := doc["@context"].(map[string]any); ok {
		for p, v := range c {
			switch d := v.(type) {
			case string:
				ctx[p] = d
			case map[string]any:
				ctx[p], _ = d["@id"].(string)
			}
		}
	}
	expand := func(s string) string {
		if p, local, ok := strings.Cut(s, ":"); ok && !strings.HasPrefix(local, "//") {
			if ns, ok := ctx[p]; ok {
				return ns + local
			}
		}
		return s
	}
	nodes := []any{doc}
	if g, ok := doc["@graph"].([]any); ok {
		nodes = g
	}
	var out []string
	for _, n := range nodes {
		node := n.(map[string]any)
		subj := expand(node["@id"].(string))
		for k, v := range node {
			if strings.HasPrefix(k, "@") {
				continue
			}
			values, ok := v.([]any)
			if !ok {
				values = []any{v}
			}
			for _, val := range values {
				var obj string
				switch o := val.(type) {
				case string:
					obj = term(turtle.Literal, o, xsdNS+"string")
				case bool:
					obj = term(turtle.Literal, strconv.FormatBool(o), xsdNS+"boolean")
				case json.Number:
					if strings.ContainsAny(o.String(), ".eE") {
						obj = term(turtle.Literal, o.String(), xsdNS+"double")
					} else {
						obj = term(turtle.Literal, o.String(), xsdNS+"integer")
					}
				case map[string]any:
					if id, ok := o["@id"].(string); ok {
						obj = term(turtle.IRI, expand(id), "")
					} else {
						typ, _ := o["@type"].(string)
						obj = term(turtle.Literal, fmt.Sprint(o["@value"]), expand(typ))
					}
				default:
					t.Fatalf("JSON-LD value %T outside the supported subset", val)
				}
				out = append(out, subj+" "+expand(k)+" "+obj)
			}
		}
	}
	slices.Sort(out)
	return out
}

// TestDirectSerialization serializes direct triples (no projection expansion)
// with SemEngine's exporter after registering bfo-cco-core and diekm, parses each
// format independently, and compares the graphs. It is not a projection or
// SHACL conformance test.
func TestDirectSerialization(t *testing.T) {
	if err := packs.RegisterBFOCCOCore(); err != nil {
		t.Fatal(err)
	}
	if err := diekm.Register(); err != nil {
		t.Fatal(err)
	}
	triples := []message.Triple{
		{Subject: platformID, Predicate: bfo.ParticipatesIn, Object: observeID, Datatype: message.EntityReferenceDatatype},
		{Subject: platformID, Predicate: "http://www.w3.org/2000/01/rdf-schema#label", Object: `Platform "one"`},
		{Subject: platformID, Predicate: testNS + "altitude", Object: 1200.5},
		{Subject: platformID, Predicate: testNS + "count", Object: 42},
		{Subject: platformID, Predicate: testNS + "active", Object: true},
		{Subject: observeID, Predicate: cco.IsAbout, Object: platformID, Datatype: message.EntityReferenceDatatype},
		// DIEKM's dotted predicates resolve to their registered IRIs and datatypes.
		{Subject: platformID, Predicate: "intel.role.assigned", Object: "pilot"},
		{Subject: platformID, Predicate: "intel.measurement.altitude-m", Object: 980.25},
	}
	s1, s2 := export.SubjectIRI(platformID, ""), export.SubjectIRI(observeID, "")
	want := []string{
		s1 + " " + bfo.ParticipatesIn + " " + term(turtle.IRI, s2, ""),
		s1 + " http://www.w3.org/2000/01/rdf-schema#label " + term(turtle.Literal, `Platform "one"`, xsdNS+"string"),
		s1 + " " + testNS + "altitude " + term(turtle.Literal, "1200.5", xsdNS+"double"),
		s1 + " " + testNS + "count " + term(turtle.Literal, "42", xsdNS+"integer"),
		s1 + " " + testNS + "active " + term(turtle.Literal, "true", xsdNS+"boolean"),
		s2 + " " + cco.IsAbout + " " + term(turtle.IRI, s1, ""),
		s1 + " " + diekmNS + "assignedRole " + term(turtle.Literal, "pilot", xsdNS+"string"),
		s1 + " " + diekmNS + "altitudeMeters " + term(turtle.Literal, "980.25", xsdNS+"double"),
	}
	slices.Sort(want)

	for _, tc := range []struct {
		format export.Format
		parse  func(*testing.T, string) []string
	}{
		{export.Turtle, graphFromTurtle},
		{export.NTriples, graphFromTurtle},
		{export.JSONLD, graphFromJSONLD},
	} {
		t.Run(tc.format.String(), func(t *testing.T) {
			out, err := export.SerializeToString(triples, tc.format)
			if err != nil {
				t.Fatal(err)
			}
			if got := tc.parse(t, out); !slices.Equal(got, want) {
				t.Errorf("parsed graph differs\n got: %q\nwant: %q\noutput:\n%s", got, want, out)
			}
			if tc.format == export.Turtle && (!strings.Contains(out, "obo:BFO_0000056") ||
				!strings.Contains(out, "cco:ont00001808") || !strings.Contains(out, "diekm:altitudeMeters")) {
				t.Errorf("Turtle does not compact obo/cco/diekm IRIs:\n%s", out)
			}
		})
	}
}

// TestRegistryObservationSensitivity shows observe sees each kind of change
// the atomic-failure tests assert is absent: a successful registration of a
// fresh pack changes the predicates, the entity types, and the serializer
// prefixes it observes.
func TestRegistryObservationSensitivity(t *testing.T) {
	key := uniqueKey("observed")
	ns := "http://example.org/semvocab-" + key + "/"
	pack := localPack(key, ns+key)
	pack.Prefixes = []vocabulary.PrefixSpec{{Prefix: "semvocab" + key, Namespace: ns}}
	before := observe(t, ns)
	if err := registerWithPrefixes(pack); err != nil {
		t.Fatal(err)
	}
	after := observe(t, ns)
	if reflect.DeepEqual(before.predicates, after.predicates) {
		t.Error("a new predicate went unobserved")
	}
	if reflect.DeepEqual(before.entityTypes, after.entityTypes) {
		t.Error("a new entity type went unobserved")
	}
	if before.prefixes == after.prefixes || !strings.Contains(after.prefixes, "@prefix semvocab"+key+":") {
		t.Errorf("a new serializer prefix went unobserved:\n%s", after.prefixes)
	}
}

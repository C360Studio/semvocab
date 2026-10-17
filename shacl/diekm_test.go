package shacl_test

// These tests use the package as a consumer does, and they touch SemEngine's
// process-global registries. Task test:repeat runs them five times in one
// process, so each asserts the state it causes and never restores state; the
// unrelated pack uses names unique to each call.

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/c360studio/semengine/message"
	"github.com/c360studio/semengine/pkg/projection/contract"
	semtypes "github.com/c360studio/semengine/pkg/types"
	"github.com/c360studio/semengine/vocabulary"
	"github.com/c360studio/semengine/vocabulary/export"
	"github.com/c360studio/semvocab/packs/diekm"
	"github.com/c360studio/semvocab/shacl"
)

const diekmNS = "https://w3id.org/c360/diekm#"

func diekmInputs(t *testing.T) (vocabulary.PackSpec, shacl.ShapeSet, shacl.Mapping) {
	t.Helper()
	spec, err := diekm.Spec()
	if err != nil {
		t.Fatal(err)
	}
	shapes, err := shacl.ParseTurtle(bytes.NewReader(diekm.Shapes()))
	if err != nil {
		t.Fatal(err)
	}
	m, err := shacl.DecodeMappingYAML(bytes.NewReader(diekm.Mapping()))
	if err != nil {
		t.Fatal(err)
	}
	return spec, shapes, m
}

func compileDIEKM(t *testing.T) *shacl.Bundle {
	t.Helper()
	spec, shapes, m := diekmInputs(t)
	b, err := shacl.Compile(spec, shapes, m)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestDIEKMBundle is task 5.8: the committed mapping binds the DIEKM shapes,
// the bundle compiles, its intel.platform contract is ruling Q23's, its
// pattern matches acme.edge.sensor.intel.platform.p1 and rejects
// acme.edge.intel.other.platform.p1 by SemEngine's matcher, and
// intel.observation is unwritable (Q18).
func TestDIEKMBundle(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "packs", "diekm", "diekm.mapping.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, diekm.Mapping()) {
		t.Fatal("diekm.Mapping() is not the committed diekm.mapping.yaml")
	}

	b := compileDIEKM(t)
	want := []contract.Contract{{
		Name:          "intel.platform",
		EntityPattern: "*.*.*.intel.platform.*",
		Groups: []contract.PredicateGroup{
			{Name: "measurement", Mode: contract.ModeReconcile, Predicates: []string{"intel.measurement.altitude-m"}},
			{Name: "role", Mode: contract.ModeReconcile, Predicates: []string{"intel.role.assigned"}},
		},
	}}
	if got := b.Contracts(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Contracts() =\n%+v\nwant\n%+v", got, want)
	}
	for id, want := range map[string]bool{
		"acme.edge.sensor.intel.platform.p1": true,
		"acme.edge.intel.other.platform.p1":  false,
	} {
		got, err := semtypes.MatchEntityIDPattern(b.Contracts()[0].EntityPattern, id)
		if err != nil || got != want {
			t.Errorf("MatchEntityIDPattern(%s) = %v, %v; want %v", id, got, err, want)
		}
	}
	if got := b.Unwritable(); !reflect.DeepEqual(got, []string{"intel.observation"}) {
		t.Errorf("Unwritable() = %v, want [intel.observation]", got)
	}

	// The DIEKM shapes in Turtle, JSON, and YAML compile to equal bundles
	// (design D15).
	spec, _, m := diekmInputs(t)
	for _, name := range []string{"diekm.shapes.json", "diekm.shapes.yaml"} {
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
			t.Fatal(err)
		}
		other, err := shacl.Compile(spec, shapes, m)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(other, b) {
			t.Errorf("%s compiles to a different bundle from the Turtle", name)
		}
	}
}

// registryState is what a caller can observe of SemEngine's global state:
// every registered predicate and entity type with its metadata, and the
// serializer's Turtle output for a probe in the DIEKM namespace, which shows
// whether the diekm prefix compacts.
type registryState struct {
	predicates  map[string]vocabulary.PredicateMetadata
	entityTypes map[string]vocabulary.EntityTypeMetadata
	turtle      string
}

func observe(t *testing.T) registryState {
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
	out, err := export.SerializeToString([]message.Triple{
		{Subject: "acme.edge.sensor.intel.platform.p1", Predicate: diekmNS + "probe", Object: "x"},
	}, export.Turtle)
	if err != nil {
		t.Fatal(err)
	}
	s.turtle = out
	return s
}

// TestSHACLCompilerOutputsValidPackAndContracts is task 5.9 (design D14,
// D15): the compiled DIEKM pack passes SemEngine's (*PackSpec).Validate,
// and fails it when one predicate name is changed to the issue fixture's
// altitude_m; each contract passes ValidateShape; and SemEngine's registry
// and prefix state are equal before and after a successful and a failed
// compile.
func TestSHACLCompilerOutputsValidPackAndContracts(t *testing.T) {
	before := observe(t)

	b := compileDIEKM(t)
	pack := b.Pack()
	if err := pack.Validate(); err != nil {
		t.Fatalf("compiled pack fails Validate: %v", err)
	}
	bad := b.Pack()
	for i := range bad.Predicates {
		if bad.Predicates[i].Name == "intel.measurement.altitude-m" {
			bad.Predicates[i].Name = "intel.measurement.altitude_m"
		}
	}
	if err := bad.Validate(); err == nil {
		t.Error("Validate accepts the predicate name altitude_m")
	}
	for _, c := range b.Contracts() {
		if err := c.ValidateShape(); err != nil {
			t.Errorf("contract %s fails ValidateShape: %v", c.Name, err)
		}
	}

	spec, shapes, m := diekmInputs(t)
	m.Predicates = m.Predicates[1:]
	if _, err := shacl.Compile(spec, shapes, m); err == nil {
		t.Fatal("a compile with an unmapped path succeeded")
	}

	after := observe(t)
	if !reflect.DeepEqual(before.predicates, after.predicates) {
		t.Error("the predicate registry changed")
	}
	if !reflect.DeepEqual(before.entityTypes, after.entityTypes) {
		t.Error("the entity-type registry changed")
	}
	if before.turtle != after.turtle {
		t.Errorf("serializer prefix output changed:\nbefore:\n%s\nafter:\n%s", before.turtle, after.turtle)
	}
}

var invocation atomic.Int64

// TestCompileIgnoresRegistrations is design D14's last check: a pack
// registered between two compiles leaves the bundles equal. The pack is
// unrelated to DIEKM by name but reuses its class and path IRIs, so a
// compiler that looked up either in SemEngine's registry would see it.
func TestCompileIgnoresRegistrations(t *testing.T) {
	first := compileDIEKM(t)

	n := invocation.Add(1)
	pred := fmt.Sprintf("probe.meta.alt%d", n)
	typ := fmt.Sprintf("probe.thing%d", n)
	prefix := fmt.Sprintf("probe%d", n)
	ns := fmt.Sprintf("http://example.org/probe%d#", n)
	err := vocabulary.RegisterPack(vocabulary.PackSpec{
		Name:        fmt.Sprintf("probe%d", n),
		Prefixes:    []vocabulary.PrefixSpec{{Prefix: prefix, Namespace: ns}},
		Predicates:  []vocabulary.PredicateSpec{{Name: pred, IRI: diekmNS + "altitudeMeters", DataType: vocabulary.DataTypeInt}},
		EntityTypes: []vocabulary.EntityTypeSpec{{DomainType: typ, ClassIRI: diekmNS + "IntelligencePlatform"}},
	}, vocabulary.WithPrefixRegistrar(export.RegisterPrefixes))
	if err != nil {
		t.Fatal(err)
	}
	if md := vocabulary.GetPredicateMetadata(pred); md == nil || md.StandardIRI != diekmNS+"altitudeMeters" {
		t.Fatalf("registration did not take effect: %+v", md)
	}
	if md := vocabulary.GetEntityTypeMetadata(typ); md == nil || md.ClassIRI != diekmNS+"IntelligencePlatform" {
		t.Fatalf("entity-type registration did not take effect: %+v", md)
	}
	out, err := export.SerializeToString([]message.Triple{
		{Subject: "acme.edge.sensor.intel.platform.p1", Predicate: ns + "x", Object: "y"},
	}, export.Turtle)
	if err != nil || !strings.Contains(out, "@prefix "+prefix+":") {
		t.Fatalf("prefix registration did not take effect: %v\n%s", err, out)
	}

	if second := compileDIEKM(t); !reflect.DeepEqual(first, second) {
		t.Error("a registration between two compiles changed the bundle")
	}
}

// TestValidateReadsNoRegistry is design D13's purity, from the outside:
// validation leaves SemEngine's registry and prefix state unchanged, and a
// registration between two validations that gives the bundle's predicate
// another data_type leaves the report unchanged. A validator that typed
// values from the registry, as export does (vocabulary/export/object.go:99
// at v0.1.0-alpha.2), would write float64 5 as "5.0"^^xsd:double there, which
// fails sh:datatype xsd:integer.
func TestValidateReadsNoRegistry(t *testing.T) {
	n := invocation.Add(1)
	pred := fmt.Sprintf("vprobe.meta.count%d", n)
	path := fmt.Sprintf("http://example.org/vprobe%d#count", n)
	class := fmt.Sprintf("http://example.org/vprobe%d#Thing", n)
	shapes, err := shacl.ParseTurtle(strings.NewReader(fmt.Sprintf(
		"<%s#S> <http://www.w3.org/ns/shacl#targetClass> <%s> ; <http://www.w3.org/ns/shacl#property> [ "+
			"<http://www.w3.org/ns/shacl#path> <%s> ; "+
			"<http://www.w3.org/ns/shacl#datatype> <http://www.w3.org/2001/XMLSchema#integer> ] .", class, class, path)))
	if err != nil {
		t.Fatal(err)
	}
	b, err := shacl.Compile(vocabulary.PackSpec{
		Name:        fmt.Sprintf("vprobe%d", n),
		Predicates:  []vocabulary.PredicateSpec{{Name: pred, IRI: path}},
		EntityTypes: []vocabulary.EntityTypeSpec{{DomainType: "vprobe.thing", ClassIRI: class}},
	}, shapes, shacl.Mapping{
		EntityTypes: []shacl.EntityTypeBinding{{ClassIRI: class, DomainTypes: []string{"vprobe.thing"}}},
		Predicates:  []shacl.PredicateBinding{{PathIRI: path, Name: pred}},
		Contracts: []shacl.ContractBinding{{DomainType: "vprobe.thing",
			Groups: []shacl.GroupBinding{{Mode: contract.ModeReconcile, Predicates: []string{pred}}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	const id = "acme.edge.sensor.vprobe.thing.t1"
	candidate := []message.Triple{{Subject: id, Predicate: pred, Object: float64(5)}}

	before := observe(t)
	first, err := b.ValidateCandidateState(id, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Targeted || !first.Conforms {
		t.Fatalf("first report = %+v, want targeted and conforming", first)
	}
	if _, err := b.ValidateCandidateState(id, []message.Triple{{Subject: id, Predicate: pred, Object: nil}}); err == nil {
		t.Fatal("a nil object validated")
	}
	after := observe(t)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("validation changed SemEngine's registry or prefix state")
	}

	err = vocabulary.RegisterPack(vocabulary.PackSpec{
		Name:       fmt.Sprintf("vprobe-other%d", n),
		Predicates: []vocabulary.PredicateSpec{{Name: pred, IRI: path, DataType: vocabulary.DataTypeFloat}},
	}, vocabulary.WithPrefixRegistrar(export.RegisterPrefixes))
	if err != nil {
		t.Fatal(err)
	}
	if md := vocabulary.GetPredicateMetadata(pred); md == nil || md.DataType != vocabulary.DataTypeFloat {
		t.Fatalf("registration did not take effect: %+v", md)
	}
	registered := observe(t)
	second, err := b.ValidateCandidateState(id, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Errorf("a registration between two validations changed the report:\nfirst  %+v\nsecond %+v", first, second)
	}
	if !reflect.DeepEqual(registered, observe(t)) {
		t.Error("validation changed SemEngine's registry or prefix state")
	}
}

// Package agreement_test is design D15's agreement test for design D9: the
// value terms the shacl validator uses must be the terms SemEngine's export
// writes at v0.1.0-alpha.2. It is a package of its own so its registrations
// run in their own test process, apart from the shacl tests that require
// SemEngine's registries to stay unchanged.
//
// SemEngine's registries are process-global, and task test:repeat runs this
// test five times in one process. The two packs registered here are the same
// every time and re-register idempotently; the test asserts the registry
// state they cause and never restores it.
package agreement_test

import (
	"bytes"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/c360studio/semengine/message"
	"github.com/c360studio/semengine/vocabulary"
	"github.com/c360studio/semengine/vocabulary/export"
	"github.com/c360studio/semvocab/internal/turtle"
	"github.com/c360studio/semvocab/packs/diekm"
	"github.com/c360studio/semvocab/shacl"
)

const (
	xsd      = "http://www.w3.org/2001/XMLSchema#"
	diekmNS  = "https://w3id.org/c360/diekm#"
	agreeNS  = "urn:semvocab:agreement#"
	platform = "acme.edge.sensor.intel.platform.p1"
	thing    = "acme.edge.sensor.agree.thing.t1"
	ref      = "acme.edge.sensor.intel.platform.p2"
	refIRI   = "<https://semengine.semanticstream.ing/entities/acme/edge/sensor/intel/platform/p2>"
)

// closedShape is a closed node shape that allows no native predicate, so
// every candidate triple is one sh:ClosedConstraintComponent result whose
// Value is the validator's term for that triple's object.
func closedShape(class string) string {
	return "<" + class + "Shape> <http://www.w3.org/ns/shacl#targetClass> <" + class + "> ; " +
		"<http://www.w3.org/ns/shacl#closed> true ; " +
		"<http://www.w3.org/ns/shacl#ignoredProperties> ( <http://www.w3.org/1999/02/22-rdf-syntax-ns#type> ) ."
}

// agreePack declares one predicate for each data_type DIEKM lacks, and one
// with no data_type.
var agreePack = vocabulary.PackSpec{
	Name:        "semvocab-agreement",
	EntityTypes: []vocabulary.EntityTypeSpec{{DomainType: "agree.thing", ClassIRI: agreeNS + "Thing"}},
	Predicates: []vocabulary.PredicateSpec{
		{Name: "agree.value.int", IRI: agreeNS + "int", DataType: vocabulary.DataTypeInt},
		{Name: "agree.value.bool", IRI: agreeNS + "bool", DataType: vocabulary.DataTypeBool},
		{Name: "agree.value.datetime", IRI: agreeNS + "datetime", DataType: vocabulary.DataTypeDateTime},
		{Name: "agree.value.json", IRI: agreeNS + "json", DataType: vocabulary.DataTypeJSON},
		{Name: "agree.value.entity-id", IRI: agreeNS + "entityId", DataType: vocabulary.DataTypeEntityID},
		{Name: "agree.value.float", IRI: agreeNS + "float", DataType: vocabulary.DataTypeFloat},
		{Name: "agree.value.untyped", IRI: agreeNS + "untyped"},
	},
}

// bundles compiles the DIEKM pack and the agreement pack each with a closed
// probe shape, registers both compiled packs with SemEngine, and asserts the
// registration: every predicate the rows use carries the data_type it was
// declared with.
func bundles(t *testing.T) map[string]*shacl.Bundle {
	t.Helper()
	diekmPack, err := diekm.Spec()
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]*shacl.Bundle{}
	for _, in := range []struct {
		subject string
		pack    vocabulary.PackSpec
		class   string
		types   []string
	}{
		{platform, diekmPack, diekmNS + "IntelligencePlatform", []string{"intel.platform"}},
		{thing, agreePack, agreeNS + "Thing", []string{"agree.thing"}},
	} {
		shapes, err := shacl.ParseTurtle(strings.NewReader(closedShape(in.class)))
		if err != nil {
			t.Fatal(err)
		}
		b, err := shacl.Compile(in.pack, shapes, shacl.Mapping{
			EntityTypes: []shacl.EntityTypeBinding{{ClassIRI: in.class, DomainTypes: in.types}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := vocabulary.RegisterPack(b.Pack(), vocabulary.WithPrefixRegistrar(export.RegisterPrefixes)); err != nil {
			t.Fatal(err)
		}
		out[in.subject] = b
	}
	for pred, dt := range map[string]string{
		"intel.role.assigned":          vocabulary.DataTypeString,
		"intel.measurement.altitude-m": vocabulary.DataTypeFloat,
		"agree.value.int":              vocabulary.DataTypeInt,
		"agree.value.bool":             vocabulary.DataTypeBool,
		"agree.value.datetime":         vocabulary.DataTypeDateTime,
		"agree.value.json":             vocabulary.DataTypeJSON,
		"agree.value.entity-id":        vocabulary.DataTypeEntityID,
		"agree.value.float":            vocabulary.DataTypeFloat,
		"agree.value.untyped":          "",
	} {
		if md := vocabulary.GetPredicateMetadata(pred); md == nil || md.DataType != dt {
			t.Fatalf("registration: %s has metadata %+v, want data_type %q", pred, md, dt)
		}
	}
	if md := vocabulary.GetPredicateMetadata("undeclared.value.x"); md != nil {
		t.Fatalf("undeclared.value.x is registered: %+v", md)
	}
	return out
}

// exported serializes one triple with SemEngine's export as N-Triples and
// reads it back with the repository's reader. It returns the object term
// in N-Triples syntax, or "" when export wrote nothing.
func exported(t *testing.T, tr message.Triple) string {
	t.Helper()
	nt, err := export.SerializeToString([]message.Triple{tr}, export.NTriples)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	triples, err := turtle.Parse(strings.NewReader(nt))
	if err != nil {
		t.Fatalf("read export's N-Triples %q: %v", nt, err)
	}
	switch len(triples) {
	case 0:
		return ""
	case 1:
		return render(triples[0].O)
	}
	t.Fatalf("export wrote %d triples for one: %q", len(triples), nt)
	return ""
}

// render writes a reader term in N-Triples syntax: an xsd:string literal
// without its datatype (RDF 1.1 N-Triples §2.5.1), and the reader's "_:L"
// blank-node namespace undone.
func render(term turtle.Term) string {
	switch term.Kind {
	case turtle.IRI:
		return "<" + term.Value + ">"
	case turtle.Blank:
		return "_:" + strings.TrimPrefix(term.Value, "_:L")
	}
	var b bytes.Buffer
	b.WriteByte('"')
	for _, r := range term.Value {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	if term.Lang != "" {
		return b.String() + "@" + term.Lang
	}
	if term.Datatype != xsd+"string" {
		return b.String() + "^^<" + term.Datatype + ">"
	}
	return b.String()
}

type row struct {
	pred string
	obj  any
	hint string
}

func (r row) triple() message.Triple {
	subject := thing
	if strings.HasPrefix(r.pred, "intel.") {
		subject = platform
	}
	return message.Triple{Subject: subject, Predicate: r.pred, Object: r.obj, Datatype: r.hint}
}

// validated returns the validator's term for one row: the Value of the one
// closed-shape result the row's triple produces.
func validated(t *testing.T, bs map[string]*shacl.Bundle, r row) (string, error) {
	t.Helper()
	tr := r.triple()
	rep, err := bs[tr.Subject].ValidateCandidateState(tr.Subject, []message.Triple{tr})
	if err != nil {
		return "", err
	}
	if len(rep.Results) != 1 || rep.Results[0].Component != "http://www.w3.org/ns/shacl#ClosedConstraintComponent" {
		t.Fatalf("%+v: want one closed-shape result, got %+v", r, rep)
	}
	return rep.Results[0].Value, nil
}

// TestValueTermsAgreeWithExport is task 6.2: for every row both accept, the
// validator's term equals the term SemEngine's export writes. The rows cover
// each data_type's accepted Go types, the fall back to the observed type, the
// 2^53 whole-float limit, entity-ID, absolute-IRI, and blank-node strings,
// and hints that name the data_type's own datatype. They include every row of
// probe P5 (notes/1-probes.md).
func TestValueTermsAgreeWithExport(t *testing.T) {
	bs := bundles(t)
	instant := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	plus2 := time.FixedZone("", 2*3600)
	for _, r := range []row{
		// int (object.go:169-172, 204-227)
		{"agree.value.int", int(42), ""},
		{"agree.value.int", int64(-7), ""},
		{"agree.value.int", uint8(3), ""},
		{"agree.value.int", uint64(math.MaxUint64), ""},
		{"agree.value.int", int8(math.MinInt8), ""},
		{"agree.value.int", float64(5), ""},
		{"agree.value.int", float32(5), ""},
		{"agree.value.int", math.Copysign(0, -1), ""},
		{"agree.value.int", float64(1 << 53), ""},
		{"agree.value.int", -float64(1 << 53), ""},
		{"agree.value.int", float64(1 << 54), ""},
		{"agree.value.int", 1.5, ""},
		{"agree.value.int", float32(2.5), ""},
		{"agree.value.int", "42", ""},
		{"agree.value.int", true, ""},
		{"agree.value.int", instant, ""},
		{"agree.value.int", 42, "xsd:integer"},
		{"agree.value.int", 42, xsd + "integer"},
		{"agree.value.int", float64(7), "xsd:integer"},
		// float (object.go:173-178, 321-330, 406-413)
		{"intel.measurement.altitude-m", 1.5, ""},
		{"intel.measurement.altitude-m", int(2), ""},
		{"intel.measurement.altitude-m", float32(1.1), ""},
		{"intel.measurement.altitude-m", 1e21, ""},
		{"intel.measurement.altitude-m", 1e-7, ""},
		{"intel.measurement.altitude-m", uint64(math.MaxUint64), ""},
		{"intel.measurement.altitude-m", "1.5", ""},
		{"intel.measurement.altitude-m", 1.5, "xsd:double"},
		{"agree.value.float", -0.0, ""},
		{"agree.value.float", 100.0, ""},
		// bool (object.go:179-184)
		{"agree.value.bool", true, ""},
		{"agree.value.bool", false, ""},
		{"agree.value.bool", "true", ""},
		{"agree.value.bool", 1, ""},
		{"agree.value.bool", false, "xsd:boolean"},
		// datetime (object.go:185-188, 265-278)
		{"agree.value.datetime", instant, ""},
		{"agree.value.datetime", instant.In(plus2), ""},
		{"agree.value.datetime", instant.Add(123456789), ""},
		{"agree.value.datetime", "2026-10-16T12:00:00Z", ""},
		{"agree.value.datetime", "2026-10-16T14:00:00+02:00", ""},
		{"agree.value.datetime", "2026-10-16T12:00:00.500+00:00", ""},
		{"agree.value.datetime", "yesterday", ""},
		{"agree.value.datetime", "2026-10-16", ""},
		{"agree.value.datetime", 3, ""},
		{"agree.value.datetime", "2026-10-16T12:00:00Z", "xsd:dateTime"},
		// json (object.go:189-199)
		{"agree.value.json", `{"a":1}`, ""},
		{"agree.value.json", `[1, 2]`, ""},
		{"agree.value.json", `"quoted"`, ""},
		{"agree.value.json", "hovering", ""},
		{"agree.value.json", 3, ""},
		{"agree.value.json", `{"a":1}`, "rdf:JSON"},
		// string, and observation (object.go:151-157, 281-310, 341-402)
		{"intel.role.assigned", "hello", ""},
		{"intel.role.assigned", "line\nbreak \"quoted\" tab\t back\\slash", ""},
		{"intel.role.assigned", ref, ""},
		{"intel.role.assigned", "http://example.org/x", ""},
		{"intel.role.assigned", "https://example.org/x?y=1#z", ""},
		{"intel.role.assigned", "urn:x:y", ""},
		{"intel.role.assigned", "http://", ""},
		{"intel.role.assigned", "http://a b", ""},
		{"intel.role.assigned", "ftp://example.org/x", ""},
		{"intel.role.assigned", "_:b1", ""},
		{"intel.role.assigned", "_:b.", ""},
		{"intel.role.assigned", "_:", ""},
		{"intel.role.assigned", "acme.edge.sensor.intel.platform", ""},
		{"intel.role.assigned", 42, ""},
		{"intel.role.assigned", 1.5, ""},
		{"intel.role.assigned", false, ""},
		{"intel.role.assigned", instant, ""},
		{"intel.role.assigned", "hello", "xsd:string"},
		// entity_id (object.go:126-136, 160-168)
		{"agree.value.entity-id", ref, ""},
		{"agree.value.entity-id", ref, message.EntityReferenceDatatype},
		{"agree.value.entity-id", "_:b1", ""},
		{"agree.value.entity-id", "not-an-id", ""},
		{"agree.value.entity-id", "not-an-id", message.EntityReferenceDatatype},
		{"agree.value.entity-id", "http://example.org/x", ""},
		{"agree.value.entity-id", 5, ""},
		// a predicate no bundle declares, with a hint: export's hint branch
		// (object.go:120-142), ruling Q29
		{"undeclared.value.x", ref, message.EntityReferenceDatatype},
		{"undeclared.value.x", "_:b1", message.EntityReferenceDatatype},
		{"undeclared.value.x", "not-an-id", message.EntityReferenceDatatype},
		{"undeclared.value.x", 7, message.EntityReferenceDatatype},
		{"undeclared.value.x", 42, "xsd:integer"},
		{"undeclared.value.x", 42, xsd + "integer"},
		{"undeclared.value.x", "x", "xsd:integer"},
		{"undeclared.value.x", 1.5, "xsd:double"},
		{"undeclared.value.x", ref, "xsd:string"},
		{"undeclared.value.x", `{"a":1}`, "rdf:JSON"},
		{"undeclared.value.x", instant, ""},
		// no data_type: observation only (object.go:98-106)
		{"agree.value.untyped", 42, ""},
		{"agree.value.untyped", float64(5), ""},
		{"agree.value.untyped", "x", ""},
		{"agree.value.untyped", ref, ""},
	} {
		want := exported(t, r.triple())
		got, err := validated(t, bs, r)
		if err != nil {
			t.Errorf("%s %T(%v) hint %q: validator refused a row export writes as %s: %v", r.pred, r.obj, r.obj, r.hint,
				want, err)
			continue
		}
		if want == "" || got != want {
			t.Errorf("%s %T(%v) hint %q: validator term %s, export writes %q", r.pred, r.obj, r.obj, r.hint, got, want)
		}
	}
}

// TestValueTermDifferencesFromExport is design D9's declared difference: on
// these rows the validator returns an input error, while export at the pin
// drops the triple, writes an unsupported Go value as a plain string from
// its %v form, or writes the hint verbatim (probe P5). Export's output is
// asserted as well, so a pin that changes it fails here.
func TestValueTermDifferencesFromExport(t *testing.T) {
	bs := bundles(t)
	instant := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		row
		export string // what export writes; "" when it drops the triple
	}{
		{row{"agree.value.int", nil, ""}, ""},
		{row{"intel.role.assigned", nil, ""}, ""},
		{row{"intel.measurement.altitude-m", math.NaN(), ""}, ""},
		{row{"intel.measurement.altitude-m", math.Inf(1), ""}, ""},
		{row{"agree.value.int", math.Inf(-1), ""}, ""},
		{row{"agree.value.untyped", float32(math.NaN()), ""}, ""},
		{row{"intel.role.assigned", struct{}{}, ""}, `"{}"`},
		{row{"intel.role.assigned", []any{1, "a"}, ""}, `"[1 a]"`},
		{row{"agree.value.json", map[string]any{"a": 1}, ""}, `"map[a:1]"`},
		{row{"agree.value.int", time.Duration(5), ""}, `"5ns"`},
		{row{"intel.role.assigned", &instant, ""}, `"2026-10-16 12:00:00 +0000 UTC"`},
		{row{"intel.role.assigned", "x", "xsd:integer"}, `"x"^^<` + xsd + `integer>`},
		{row{"agree.value.int", 42, "xsd:string"}, `"42"`},
		{row{"agree.value.int", "42", "xsd:integer"}, `"42"^^<` + xsd + `integer>`},
		{row{"agree.value.int", float64(1 << 53), "xsd:integer"}, `"9.007199254740992e+15"^^<` + xsd + `integer>`},
		{row{"intel.measurement.altitude-m", 2, "xsd:double"}, `"2"^^<` + xsd + `double>`},
		{row{"agree.value.datetime", instant, "xsd:dateTime"}, `"2026-10-16 12:00:00 +0000 UTC"^^<` + xsd + `dateTime>`},
		{row{"intel.role.assigned", ref, message.EntityReferenceDatatype}, refIRI},
		{row{"agree.value.untyped", 42, "xsd:integer"}, `"42"^^<` + xsd + `integer>`},
		{row{"agree.value.entity-id", ref, "xsd:string"}, `"` + ref + `"`},
		{row{"intel.role.assigned", ref, "xsd:string"}, `"` + ref + `"`},
	} {
		if got := exported(t, tt.triple()); got != tt.export {
			t.Errorf("%s %T(%v) hint %q: export writes %q, want %q", tt.pred, tt.obj, tt.obj, tt.hint, got, tt.export)
		}
		if got, err := validated(t, bs, tt.row); err == nil {
			t.Errorf("%s %T(%v) hint %q: validator accepted it as %s", tt.pred, tt.obj, tt.obj, tt.hint, got)
		}
	}
}

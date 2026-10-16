package shacl

import (
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/c360studio/semengine/message"
)

// The expectations below are SHACL Core's semantics (W3C Recommendation
// 2017-07-20) for each component, written out by hand: which value nodes
// produce a result, and the result's focus node, source shape, path,
// component, value, and severity (§3.6). Terms are written in N-Triples
// syntax (RDF 1.1 N-Triples §2.4–2.5) as SemEngine's export writes them
// (vocabulary/export/ntriples.go:46-60 at v0.1.0-alpha.2).

const (
	focus      = "acme.edge.sensor.test.thing.t1"
	violation  = sh + "Violation"
	entityBase = "https://semengine.semanticstream.ing/entities/"
)

func tr(pred string, obj any) message.Triple {
	return message.Triple{Subject: focus, Predicate: pred, Object: obj}
}

func validate(t *testing.T, b *Bundle, triples ...message.Triple) Report {
	t.Helper()
	r, err := b.ValidateCandidateState(focus, triples)
	if err != nil {
		t.Fatalf("ValidateCandidateState: %v", err)
	}
	return r
}

// requireResults compares a report's results with the expected ones, in
// order, and requires Conforms to be false exactly when there are results.
func requireResults(t *testing.T, r Report, want ...Result) {
	t.Helper()
	if !r.Targeted {
		t.Fatalf("report not targeted: %+v", r)
	}
	if r.Conforms != (len(want) == 0) {
		t.Errorf("Conforms = %v with %d expected result(s)", r.Conforms, len(want))
	}
	if len(want) == 0 {
		want = nil
	}
	if !reflect.DeepEqual(r.Results, want) {
		t.Errorf("results:\n%s\nwant:\n%s", formatResults(r.Results), formatResults(want))
	}
}

func formatResults(rs []Result) string {
	var b strings.Builder
	for _, r := range rs {
		b.WriteString("  " + strings.Join([]string{r.FocusNode, r.SourceShape, r.Path, r.Predicate, r.Component,
			r.Value, r.Severity, strings.Join(r.Messages, "|")}, " ; ") + "\n")
	}
	return b.String()
}

func res(shape, path, pred, component, value string) Result {
	return Result{FocusNode: focus, SourceShape: shape, Path: path, Predicate: pred, Component: sh + component,
		Value: value, Severity: violation}
}

// TestValidateInputErrors is design D13's input errors: a malformed entity
// ID, a triple about another subject, a malformed predicate, and the D9
// representation errors are returned as an error, all of them together,
// never as results.
func TestValidateInputErrors(t *testing.T) {
	b := mustCompile(t, `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:datatype xsd:integer ] .`)

	if _, err := b.ValidateCandidateState("acme.edge.sensor.test.thing", nil); err == nil ||
		!strings.Contains(err.Error(), "entity ID") {
		t.Errorf("five-part entity ID: err = %v", err)
	}

	for _, tt := range []struct {
		name   string
		triple message.Triple
		want   string
	}{
		{"another subject", message.Triple{Subject: "acme.edge.sensor.test.thing.t2", Predicate: "test.prop.p", Object: 1},
			"subject"},
		{"two-part predicate", tr("test.prop", 1), "test.prop"},
		{"uppercase predicate", tr("test.Prop.p", 1), "test.Prop.p"},
		{"nil object", tr("test.prop.p", nil), "nil"},
		{"struct object", tr("test.prop.p", struct{}{}), "struct {}"},
		{"named integer type", tr("test.prop.p", time.Duration(5)), "time.Duration"},
		{"map object", tr("test.prop.q", map[string]any{"a": 1}), "map[string]interface {}"},
		{"NaN", tr("test.prop.p", math.NaN()), "NaN"},
		{"float32 infinity", tr("test.prop.q", float32(math.Inf(1))), "+Inf"},
		{"hint on a predicate with no data_type", message.Triple{Subject: focus, Predicate: "test.prop.q", Object: 1,
			Datatype: "xsd:integer"}, "xsd:integer"},
		{"hint naming another datatype", message.Triple{Subject: focus, Predicate: "test.prop.p", Object: 1,
			Datatype: "xsd:string"}, "xsd:string"},
		{"@id on an int predicate", message.Triple{Subject: focus, Predicate: "test.prop.p", Object: 1,
			Datatype: message.EntityReferenceDatatype}, "@id"},
		{"hint that changes the term", message.Triple{Subject: focus, Predicate: "test.prop.p", Object: "42",
			Datatype: "xsd:integer"}, `"42"`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r, err := b.ValidateCandidateState(focus, []message.Triple{tr("test.prop.p", 1), tt.triple})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v, want one naming %q (report %+v)", err, tt.want, r)
			}
		})
	}

	// Every input error is reported, not only the first.
	_, err := b.ValidateCandidateState(focus, []message.Triple{tr("test.prop.p", nil), tr("bad", 1)})
	if err == nil || !strings.Contains(err.Error(), "nil") || !strings.Contains(err.Error(), `"bad"`) {
		t.Errorf("err = %v, want both input errors", err)
	}

	// An untargeted entity is still checked for input errors.
	_, err = b.ValidateCandidateState("acme.edge.sensor.test.other.o1",
		[]message.Triple{{Subject: "acme.edge.sensor.test.other.o1", Predicate: "test.prop.p", Object: nil}})
	if err == nil {
		t.Error("an untargeted entity with a nil object validated without error")
	}
}

// TestValidateTargeting is design D13's targeting: the entity's
// domain.type (ID positions 4 and 5) selects the shapes, and an entity no
// shape applies to reports Targeted false.
func TestValidateTargeting(t *testing.T) {
	b := mustCompile(t, `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:minCount 1 ] .`)

	other := "acme.edge.sensor.test.other.o1"
	r, err := b.ValidateCandidateState(other, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.Targeted || !r.Conforms || r.Results != nil {
		t.Errorf("untargeted report = %+v, want Targeted false, Conforms true, no results", r)
	}
	// "thing" in another position does not target it.
	r, err = b.ValidateCandidateState("test.thing.sensor.acme.edge.t1", nil)
	if err != nil || r.Targeted {
		t.Errorf("entity with test.thing in positions 1-2: report %+v, err %v", r, err)
	}

	requireResults(t, validate(t, b, tr("test.prop.p", "x")))
	requireResults(t, validate(t, b), res(ex+"S", ex+"p", "test.prop.p", "MinCountConstraintComponent", ""))
}

// TestValidateCountsDistinctTerms is design D13's counting: values are
// distinct RDF terms, so triples differing only in Source or Timestamp, or
// Go values export writes as one term, count once.
func TestValidateCountsDistinctTerms(t *testing.T) {
	b := mustCompile(t, `ex:S sh:targetClass ex:C ;
		sh:property [ sh:path ex:p ; sh:datatype xsd:integer ; sh:maxCount 1 ] ;
		sh:property [ sh:path ex:q ; sh:datatype xsd:dateTime ; sh:maxCount 1 ] .`)

	a, c := tr("test.prop.p", 5), tr("test.prop.p", 5)
	a.Source, c.Source = "radar", "operator"
	c.Timestamp = time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC)
	requireResults(t, validate(t, b, a, c))
	// int 5 and float64 5 are both "5"^^xsd:integer on an int predicate.
	requireResults(t, validate(t, b, tr("test.prop.p", 5), tr("test.prop.p", float64(5)), tr("test.prop.p", uint8(5))))
	// One instant as a time.Time and as an RFC 3339 string with an offset.
	requireResults(t, validate(t, b, tr("test.prop.q", time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)),
		tr("test.prop.q", "2026-10-16T12:00:00+00:00")))

	requireResults(t, validate(t, b, tr("test.prop.p", 5), tr("test.prop.p", 6)),
		res(ex+"S", ex+"p", "test.prop.p", "MaxCountConstraintComponent", ""))
	// The string "5" is a plain literal, another term, and not an integer.
	requireResults(t, validate(t, b, tr("test.prop.p", 5), tr("test.prop.p", "5")),
		res(ex+"S", ex+"p", "test.prop.p", "DatatypeConstraintComponent", `"5"`),
		res(ex+"S", ex+"p", "test.prop.p", "MaxCountConstraintComponent", ""))
}

// TestValidateEveryConstraint is design D13's completeness: every
// constraint of every applicable shape runs, and results are sorted by
// source shape, path, component, and value.
func TestValidateEveryConstraint(t *testing.T) {
	b := mustCompile(t, `
		ex:T sh:targetClass ex:C ; sh:property [ sh:path ex:q ; sh:minCount 1 ] .
		ex:S sh:targetClass ex:C ;
			sh:property ex:S-p .
		ex:S-p sh:path ex:p ; sh:datatype xsd:string ; sh:pattern "^a" ; sh:in ( "a" "ab" ) ; sh:maxCount 1 ;
			sh:severity sh:Warning ; sh:message "p is wrong" .`)

	warn := func(component, value string) Result {
		r := res(ex+"S-p", ex+"p", "test.prop.p", component, value)
		r.Severity, r.Messages = sh+"Warning", []string{"p is wrong"}
		return r
	}
	requireResults(t, validate(t, b, tr("test.prop.p", "c"), tr("test.prop.p", "b")),
		warn("InConstraintComponent", `"b"`),
		warn("InConstraintComponent", `"c"`),
		warn("MaxCountConstraintComponent", ""),
		warn("PatternConstraintComponent", `"b"`),
		warn("PatternConstraintComponent", `"c"`),
		res(ex+"T", ex+"q", "test.prop.q", "MinCountConstraintComponent", ""))

	// A result of any severity makes the report non-conforming (§3.6.1).
	r := validate(t, b, tr("test.prop.p", "b"), tr("test.prop.q", 1))
	if r.Conforms {
		t.Errorf("a report with only sh:Warning results conforms: %+v", r)
	}
}

// TestValidateSpecScenarios covers the spec scenarios this task owns:
// maxCount 0 with one value, and the large integer bound a float64
// comparison would miss.
func TestValidateSpecScenarios(t *testing.T) {
	b := mustCompile(t, `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:maxCount 0 ] .`)
	requireResults(t, validate(t, b))
	requireResults(t, validate(t, b, tr("test.prop.p", "x")),
		res(ex+"S", ex+"p", "test.prop.p", "MaxCountConstraintComponent", ""))

	b = mustCompile(t, `ex:S sh:targetClass ex:C ;
		sh:property [ sh:path ex:p ; sh:datatype xsd:integer ; sh:maxInclusive 9007199254740992 ] .`)
	if float64(int64(9007199254740993)) != float64(int64(9007199254740992)) {
		t.Fatal("premise: float64 tells 2^53 from 2^53+1")
	}
	requireResults(t, validate(t, b, tr("test.prop.p", int64(9007199254740992))))
	requireResults(t, validate(t, b, tr("test.prop.p", int64(9007199254740993))),
		res(ex+"S", ex+"p", "test.prop.p", "MaxInclusiveConstraintComponent", `"9007199254740993"^^<`+xsd+`integer>`))
}

// TestValidateBounds is §4.3: a value node violates a bound unless the
// comparison holds; a value that is not a number, or is not comparable,
// violates.
func TestValidateBounds(t *testing.T) {
	integer := func(v string) string { return `"` + v + `"^^<` + xsd + `integer>` }
	double := func(v string) string { return `"` + v + `"^^<` + xsd + `double>` }

	b := mustCompile(t, `ex:S sh:targetClass ex:C ;
		sh:property [ sh:path ex:p ; sh:datatype xsd:integer ; sh:minExclusive 40 ; sh:maxInclusive 100 ] .`)
	requireResults(t, validate(t, b, tr("test.prop.p", 41), tr("test.prop.p", 100)))
	requireResults(t, validate(t, b, tr("test.prop.p", 40), tr("test.prop.p", 101), tr("test.prop.p", "a"),
		tr("test.prop.p", "http://example.org/x")),
		res(ex+"S", ex+"p", "test.prop.p", "DatatypeConstraintComponent", `"a"`),
		res(ex+"S", ex+"p", "test.prop.p", "DatatypeConstraintComponent", `<http://example.org/x>`),
		res(ex+"S", ex+"p", "test.prop.p", "MaxInclusiveConstraintComponent", integer("101")),
		res(ex+"S", ex+"p", "test.prop.p", "MaxInclusiveConstraintComponent", `"a"`),
		res(ex+"S", ex+"p", "test.prop.p", "MaxInclusiveConstraintComponent", `<http://example.org/x>`),
		res(ex+"S", ex+"p", "test.prop.p", "MinExclusiveConstraintComponent", integer("40")),
		res(ex+"S", ex+"p", "test.prop.p", "MinExclusiveConstraintComponent", `"a"`),
		res(ex+"S", ex+"p", "test.prop.p", "MinExclusiveConstraintComponent", `<http://example.org/x>`))
	// A float64 1.5 on an int predicate is an xsd:double (design D9): it
	// fails the datatype, and compares exactly with the integer bounds.
	requireResults(t, validate(t, b, tr("test.prop.p", 40.5)),
		res(ex+"S", ex+"p", "test.prop.p", "DatatypeConstraintComponent", double("40.5")))
	requireResults(t, validate(t, b, tr("test.prop.p", 39.5)),
		res(ex+"S", ex+"p", "test.prop.p", "DatatypeConstraintComponent", double("39.5")),
		res(ex+"S", ex+"p", "test.prop.p", "MinExclusiveConstraintComponent", double("39.5")))

	b = mustCompile(t, `ex:S sh:targetClass ex:C ;
		sh:property [ sh:path ex:p ; sh:datatype xsd:double ; sh:minInclusive "0.5"^^xsd:double ; sh:maxExclusive "INF"^^xsd:double ] .`)
	requireResults(t, validate(t, b, tr("test.prop.p", 0.5), tr("test.prop.p", 1), tr("test.prop.p", math.MaxFloat64)))
	requireResults(t, validate(t, b, tr("test.prop.p", 0.25)),
		res(ex+"S", ex+"p", "test.prop.p", "MinInclusiveConstraintComponent", double("0.25")))

	// A NaN bound compares with nothing, so every value violates it.
	b = mustCompile(t, `ex:S sh:targetClass ex:C ;
		sh:property [ sh:path ex:p ; sh:datatype xsd:double ; sh:minInclusive "NaN"^^xsd:double ] .`)
	requireResults(t, validate(t, b, tr("test.prop.p", 1)),
		res(ex+"S", ex+"p", "test.prop.p", "MinInclusiveConstraintComponent", double("1.0")))
}

// TestValidateValueComponents is §4.1.2, §4.4.3, and §4.8.3 over the terms
// design D9 gives: an entity ID or an absolute IRI on a string predicate is
// an IRI, so it fails sh:datatype xsd:string; sh:pattern matches an IRI's
// string and fails a blank node; sh:in () rejects every value.
func TestValidateValueComponents(t *testing.T) {
	b := mustCompile(t, `ex:S sh:targetClass ex:C ;
		sh:property [ sh:path ex:p ; sh:datatype xsd:string ] ;
		sh:property [ sh:path ex:q ; sh:pattern "^http" ] ;
		sh:property [ sh:path ex:r ; sh:datatype xsd:string ; sh:in () ] .`)
	ref := "acme.edge.sensor.test.thing.t2"
	requireResults(t, validate(t, b, tr("test.prop.p", "plain"), tr("test.prop.p", ref), tr("test.prop.p", "_:b1"),
		tr("test.prop.q", "http://example.org/x"), tr("test.prop.q", "_:b2"), tr("test.prop.q", "https"),
		tr("test.prop.r", "a")),
		res(ex+"S", ex+"p", "test.prop.p", "DatatypeConstraintComponent", "<"+entityBase+"acme/edge/sensor/test/thing/t2>"),
		res(ex+"S", ex+"p", "test.prop.p", "DatatypeConstraintComponent", "_:b1"),
		res(ex+"S", ex+"q", "test.prop.q", "PatternConstraintComponent", "_:b2"),
		res(ex+"S", ex+"r", "test.prop.r", "InConstraintComponent", `"a"`))

	b = mustCompile(t, `ex:S sh:targetClass ex:C ;
		sh:property [ sh:path ex:p ; sh:datatype rdf:JSON ] ; sh:property [ sh:path ex:q ; sh:datatype xsd:boolean ] .`)
	requireResults(t, validate(t, b, tr("test.prop.p", `{"a":1}`), tr("test.prop.q", true)))
	requireResults(t, validate(t, b, tr("test.prop.p", "hovering"), tr("test.prop.q", "true")),
		res(ex+"S", ex+"p", "test.prop.p", "DatatypeConstraintComponent", `"hovering"`),
		res(ex+"S", ex+"q", "test.prop.q", "DatatypeConstraintComponent", `"true"`))
}

// TestValidateClosed is design D12 in native state: a triple whose
// predicate the closed shape does not allow is a result per value, under
// the node shape, with the predicate's IRI as path when the bundle has one.
func TestValidateClosed(t *testing.T) {
	b := mustCompile(t, `ex:S sh:targetClass ex:C ; sh:closed true ; sh:ignoredProperties ( rdf:type ex:q ) ;
		sh:severity sh:Info ; sh:message "closed" ;
		sh:property [ sh:path ex:p ] .`)
	closed := func(path, pred, value string) Result {
		r := res(ex+"S", path, pred, "ClosedConstraintComponent", value)
		r.Severity, r.Messages = sh+"Info", []string{"closed"}
		return r
	}
	requireResults(t, validate(t, b, tr("test.prop.p", 1), tr("test.prop.q", 2)))
	requireResults(t, validate(t, b, tr("test.prop.p", 1), tr("test.prop.r", "x"), tr("test.prop.r", "y"),
		tr("other.prop.x", true)),
		closed("", "other.prop.x", `"true"^^<`+xsd+`boolean>`),
		closed(ex+"r", "test.prop.r", `"x"`),
		closed(ex+"r", "test.prop.r", `"y"`))
}

// TestValidateDeterministic is design D13's ordering: permuting or
// repeating candidate triples gives an equal report.
func TestValidateDeterministic(t *testing.T) {
	b := mustCompile(t, `ex:S sh:targetClass ex:C ; sh:closed true ; sh:ignoredProperties ( rdf:type ) ;
		sh:property [ sh:path ex:p ; sh:datatype xsd:integer ; sh:maxCount 1 ; sh:maxInclusive 3 ] .`)
	triples := []message.Triple{tr("test.prop.p", 5), tr("test.prop.p", 4), tr("test.prop.q", "a"),
		tr("test.prop.r", "b"), tr("other.prop.x", 1)}
	want := validate(t, b, triples...)
	if len(want.Results) != 6 {
		t.Fatalf("want 6 results, got:\n%s", formatResults(want.Results))
	}
	for i := range triples {
		perm := append(slices.Clone(triples[i:]), triples[:i]...)
		slices.Reverse(perm)
		perm = append(perm, triples[i])
		if got := validate(t, b, perm...); !reflect.DeepEqual(got, want) {
			t.Errorf("permutation %d:\n%s\nwant:\n%s", i, formatResults(got.Results), formatResults(want.Results))
		}
	}
}

// TestValidateUndeclaredHints is ruling Q29: a hinted triple on a predicate
// the bundle does not declare is accepted, with the term export writes for
// that hint (vocabulary/export/object.go:120-142 at v0.1.0-alpha.2). A
// closed shape reports each as a disallowed value with that term; a shape
// that is not closed ignores them.
func TestValidateUndeclaredHints(t *testing.T) {
	hinted := []message.Triple{
		{Subject: focus, Predicate: "other.prop.a", Object: "acme.edge.sensor.intel.platform.p2",
			Datatype: message.EntityReferenceDatatype},
		{Subject: focus, Predicate: "other.prop.b", Object: "not-an-id", Datatype: message.EntityReferenceDatatype},
		{Subject: focus, Predicate: "other.prop.c", Object: 42, Datatype: "xsd:integer"},
		{Subject: focus, Predicate: "other.prop.d", Object: "x", Datatype: "xsd:integer"},
	}
	closed := mustCompile(t, `ex:S sh:targetClass ex:C ; sh:closed true ; sh:ignoredProperties ( rdf:type ) ;
		sh:property [ sh:path ex:p ] .`)
	requireResults(t, validate(t, closed, hinted...),
		res(ex+"S", "", "other.prop.c", "ClosedConstraintComponent", `"42"^^<`+xsd+`integer>`),
		res(ex+"S", "", "other.prop.b", "ClosedConstraintComponent", `"not-an-id"`),
		res(ex+"S", "", "other.prop.d", "ClosedConstraintComponent", `"x"^^<`+xsd+`integer>`),
		res(ex+"S", "", "other.prop.a", "ClosedConstraintComponent", "<"+entityBase+"acme/edge/sensor/intel/platform/p2>"))

	open := mustCompile(t, `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:maxCount 1 ] .`)
	requireResults(t, validate(t, open, hinted...))
}

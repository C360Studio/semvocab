package shacl

import (
	"math"
	"math/big"
	"reflect"
	"strings"
	"testing"
)

// TestCardinalityCompiles is task 5.3's cardinality half (design D4, §4.2)
// with #1's cases: an omitted maximum is unbounded, maxCount 0 prohibits
// values, maxCount 1 is one; negative, oversized, and contradictory counts
// fail.
func TestCardinalityCompiles(t *testing.T) {
	for _, tt := range []struct {
		name, constraints string
		wantMin           int
		wantMax           *int
	}{
		{"omitted maximum", `sh:minCount 1`, 1, nil},
		{"maxCount 0", `sh:maxCount 0`, 0, ptr(0)},
		{"maxCount 1", `sh:maxCount 1`, 0, ptr(1)},
		{"equal bounds", `sh:minCount 2 ; sh:maxCount 2`, 2, ptr(2)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := onlyProperty(t, mustCompile(t, `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; `+tt.constraints+` ] .`))
			if p.MinCount != tt.wantMin || !reflect.DeepEqual(p.MaxCount, tt.wantMax) {
				t.Errorf("min %d max %v, want min %d max %v", p.MinCount, p.MaxCount, tt.wantMin, tt.wantMax)
			}
		})
	}
	for _, tt := range []struct{ name, constraints, construct, msg string }{
		{"negative minCount", `sh:minCount -1`, sh + "minCount", "negative"},
		{"negative maxCount", `sh:maxCount -1`, sh + "maxCount", "negative"},
		{"minCount above maxCount", `sh:minCount 2 ; sh:maxCount 1`, sh + "maxCount", "sh:minCount 2 is above sh:maxCount 1"},
		{"maxCount above MaxInt32", `sh:maxCount 2147483648`, sh + "maxCount", "exceeds 2147483647"},
		{"minCount above MaxInt32", `sh:minCount 2147483648`, sh + "minCount", "exceeds 2147483647"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := compileTurtle(t, `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; `+tt.constraints+` ] .`)
			if is := requireIssue(t, err, tt.construct, tt.msg); is.Shape != ex+"S" || is.Path != ex+"p" {
				t.Errorf("issue = %+v, want shape ex:S and path ex:p", is)
			}
		})
	}
	// MaxInt32 itself is accepted.
	mustCompile(t, `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:maxCount 2147483647 ] .`)
}

// TestDatatypeTable is design D8: each accepted sh:datatype fills an empty
// pack data_type with the value the table states, written out here, not read
// from the compiler; every other datatype fails.
func TestDatatypeTable(t *testing.T) {
	for dt, want := range map[string]string{
		"xsd:string":   "string",
		"xsd:integer":  "int",
		"xsd:double":   "float",
		"xsd:boolean":  "bool",
		"xsd:dateTime": "datetime",
		"rdf:JSON":     "json",
	} {
		b := mustCompile(t, `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:datatype `+dt+` ] .`)
		if got := b.Pack().Predicates[0]; got.Name != "test.prop.p" || got.DataType != want {
			t.Errorf("%s: predicate %s data_type %q, want %q", dt, got.Name, got.DataType, want)
		}
	}
	for _, dt := range []string{"xsd:float", "xsd:decimal", "xsd:int", "xsd:long", "rdf:langString", "xsd:anyURI", "ex:custom"} {
		_, err := compileTurtle(t, `@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:datatype `+dt+` ] .`)
		requireIssue(t, err, sh+"datatype", "not in the datatype table")
	}
}

// TestNumericBoundsExact is design D8's bounds: integer bounds are exact, so
// 2^53+1 above 2^53 is caught where float64 would see two equal numbers.
func TestNumericBoundsExact(t *testing.T) {
	const two53, two53plus1 = "9007199254740992", "9007199254740993"
	f1, _ := new(big.Float).SetString(two53)
	f2, _ := new(big.Float).SetString(two53plus1)
	a, _ := f1.Float64()
	c, _ := f2.Float64()
	if a != c {
		t.Fatal("premise: float64 cannot tell 2^53 from 2^53+1")
	}
	_, err := compileTurtle(t, `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:datatype xsd:integer ;
  sh:minInclusive `+two53plus1+` ; sh:maxInclusive `+two53+` ] .`)
	requireIssue(t, err, sh+"minInclusive", "sh:minInclusive is above sh:maxInclusive")

	p := onlyProperty(t, mustCompile(t, `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:datatype xsd:integer ;
  sh:minInclusive `+two53+` ; sh:maxInclusive `+two53plus1+` ] .`))
	if p.MaxIncl == nil || p.MaxIncl.Int == nil || p.MaxIncl.Int.String() != two53plus1 {
		t.Errorf("maxInclusive = %+v, want exactly %s", p.MaxIncl, two53plus1)
	}

	for _, tt := range []struct{ name, constraints, construct, msg string }{
		{"no datatype", `sh:minInclusive 1`, "minInclusive", "needs an sh:datatype"},
		{"string datatype", `sh:datatype xsd:string ; sh:maxInclusive "z"`, "maxInclusive", "xsd:integer or xsd:double"},
		{"decimal bound on integer", `sh:datatype xsd:integer ; sh:minExclusive 1.5`, "minExclusive", "must be <" + xsd + "integer>"},
		{"integer bound on double", `sh:datatype xsd:double ; sh:maxExclusive 1`, "maxExclusive", "must be <" + xsd + "double>"},
		{"ill-typed integer", `sh:datatype xsd:integer ; sh:minInclusive "1.0"^^xsd:integer`, "minInclusive", "not an xsd:integer lexical form"},
		{"ill-typed double", `sh:datatype xsd:double ; sh:minInclusive "+INF"^^xsd:double`, "minInclusive", "not an xsd:double lexical form"},
		{"exclusive above inclusive", `sh:datatype xsd:double ; sh:minExclusive 2.0e0 ; sh:maxInclusive 1.0e0`, "minExclusive", "above sh:maxInclusive"},
		{"inclusive above exclusive", `sh:datatype xsd:integer ; sh:minInclusive 3 ; sh:maxExclusive 2`, "minInclusive", "above sh:maxExclusive"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := compileTurtle(t, `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; `+tt.constraints+` ] .`)
			requireIssue(t, err, sh+tt.construct, tt.msg)
		})
	}

	// The YAML form gives a bound its property's datatype (notes/4-model.md),
	// so the same rules apply to it.
	set, err := DecodeShapesYAML(strings.NewReader(`node_shapes:
  - id: "http://example.org/S"
    target_classes: ["http://example.org/C"]
    properties:
      - path: "http://example.org/p"
        datatype: "http://www.w3.org/2001/XMLSchema#integer"
        min_inclusive: "` + two53plus1 + `"
        max_inclusive: "` + two53 + `"
`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = Compile(testPack(), set, testMapping())
	requireIssue(t, err, sh+"minInclusive", "above")
}

// TestXSDLexicalForms checks the bound grammars against XML Schema Part 2
// §3.3.13.1 (integer) and §3.2.5.1 (double, XSD 1.0: INF, -INF, NaN, no
// +INF).
func TestXSDLexicalForms(t *testing.T) {
	for s, want := range map[string]string{"0": "0", "+5": "5", "-0": "0", "007": "7", "-12345678901234567890": "-12345678901234567890"} {
		n, ok := parseXSDInteger(s)
		if !ok || n.String() != want {
			t.Errorf("integer %q = %v, %v; want %s", s, n, ok, want)
		}
	}
	for _, s := range []string{"", "+", "1.0", "1e3", "1_000", "++1", "+-1", " 1", "0x10", "١"} {
		if _, ok := parseXSDInteger(s); ok {
			t.Errorf("integer %q accepted", s)
		}
	}
	for s, want := range map[string]float64{
		"1": 1, "1.5": 1.5, ".5": 0.5, "5.": 5, "-1E3": -1000, "1e-2": 0.01, "+0.0": 0,
		"INF": math.Inf(1), "-INF": math.Inf(-1), "1e400": math.Inf(1),
	} {
		f, nan, ok := parseXSDDouble(s)
		if !ok || nan || f != want {
			t.Errorf("double %q = %v (NaN %v), %v; want %v", s, f, nan, ok, want)
		}
	}
	if _, nan, ok := parseXSDDouble("NaN"); !ok || !nan {
		t.Error("double NaN not accepted as NaN")
	}
	for _, s := range []string{"", "+INF", "Inf", "inf", "Infinity", "nan", "1e", "e3", ".", "0x1p-2", "1_0", " 1", "1.5f"} {
		if _, _, ok := parseXSDDouble(s); ok {
			t.Errorf("double %q accepted", s)
		}
	}
}

// TestEnumeration is design D11 with #1's cases: sh:in () compiles to an
// empty set that no value satisfies, an absent sh:in to none, a typed member
// fails naming sh:in and the member, and member order does not matter.
func TestEnumeration(t *testing.T) {
	empty := onlyProperty(t, mustCompile(t, `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:datatype xsd:string ; sh:in () ] .`))
	if empty.In == nil || len(*empty.In) != 0 {
		t.Errorf("sh:in () = %v, want an empty, non-nil set", empty.In)
	}
	none := onlyProperty(t, mustCompile(t, `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:datatype xsd:string ] .`))
	if none.In != nil {
		t.Errorf("absent sh:in = %v, want nil", *none.In)
	}

	_, err := compileTurtle(t, `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:datatype xsd:string ; sh:in ( "a" 1 ) ] .`)
	if is := requireIssue(t, err, sh+"in", `member "1"^^<`+xsd+`integer>`); is.Path != ex+"p" {
		t.Errorf("issue = %+v, want path ex:p", is)
	}
	_, err = compileTurtle(t, `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:datatype xsd:string ; sh:in ( ex:a ) ] .`)
	requireIssue(t, err, sh+"in", "member <"+ex+"a>")
	_, err = compileTurtle(t, `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:datatype xsd:string ; sh:in ( "a"@en ) ] .`)
	requireIssue(t, err, sh+"in", `"a"@en`)
	_, err = compileTurtle(t, `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:datatype xsd:integer ; sh:in ( "1" ) ] .`)
	requireIssue(t, err, sh+"in", "needs sh:datatype xsd:string")
	_, err = compileTurtle(t, `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:in ( "1" ) ] .`)
	requireIssue(t, err, sh+"in", "needs sh:datatype xsd:string")

	ab := mustCompile(t, `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:datatype xsd:string ; sh:in ( "a" "b" ) ] .`)
	ba := mustCompile(t, `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:datatype xsd:string ; sh:in ( "b" "a" "b" ) ] .`)
	if !reflect.DeepEqual(ab, ba) {
		t.Error("sh:in member order or repeats changed the bundle")
	}
}

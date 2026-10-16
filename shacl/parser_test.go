package shacl

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/c360studio/semvocab/internal/oracle"
	"github.com/c360studio/semvocab/internal/turtle"
	"github.com/c360studio/semvocab/packs/diekm"
)

const (
	sh     = "http://www.w3.org/ns/shacl#"
	xsd    = "http://www.w3.org/2001/XMLSchema#"
	dk     = "https://w3id.org/c360/diekm#"
	prefix = "@prefix sh: <http://www.w3.org/ns/shacl#> .\n@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .\n" +
		"@prefix ex: <http://example.org/> .\n@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .\n"
)

func ptr[T any](v T) *T { return &v }

// diekmShapes is the shape set packs/diekm/diekm.shacl.ttl states, written
// out by hand from the Turtle.
var diekmShapes = ShapeSet{NodeShapes: []NodeShape{
	{ID: dk + "IntelligenceObservationShape", TargetClasses: []string{dk + "IntelligenceObservation"}},
	{
		ID: dk + "IntelligencePlatformShape", TargetClasses: []string{dk + "IntelligencePlatform"},
		Properties: []PropertyShape{
			{Path: dk + "altitudeMeters", Datatype: xsd + "double", MaxCount: ptr(1)},
			{Path: dk + "assignedRole", Datatype: xsd + "string"},
		},
	},
}}

func readTestdata(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// TestDIEKMShapesInThreeFormats is task 4.3: the DIEKM shapes in Turtle (the
// pack's own file), JSON, and YAML decode to equal shape sets, and to the set
// written out by hand.
func TestDIEKMShapesInThreeFormats(t *testing.T) {
	fromTurtle, err := ParseTurtle(bytes.NewReader(diekm.Shapes()))
	if err != nil {
		t.Fatal(err)
	}
	fromJSON, err := DecodeShapesJSON(bytes.NewReader(readTestdata(t, "diekm.shapes.json")))
	if err != nil {
		t.Fatal(err)
	}
	fromYAML, err := DecodeShapesYAML(bytes.NewReader(readTestdata(t, "diekm.shapes.yaml")))
	if err != nil {
		t.Fatal(err)
	}
	for name, got := range map[string]ShapeSet{"Turtle": fromTurtle, "JSON": fromJSON, "YAML": fromYAML} {
		if !reflect.DeepEqual(got, diekmShapes) {
			t.Errorf("%s:\n got %+v\nwant %+v", name, got, diekmShapes)
		}
	}
}

// richShape uses every model field once, in all three formats.
const richTurtle = prefix + "@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .\n" + `
ex:S a sh:NodeShape ;
  sh:targetClass ex:B , ex:A ;
  sh:closed true ;
  sh:ignoredProperties ( rdf:type ex:q ) ;
  sh:severity sh:Warning ;
  sh:message "second" , "first" ;
  sh:property ex:P , [
    sh:path ex:name ; sh:datatype xsd:string ; sh:pattern "^[a-z]+$" ; sh:in ( "b" "a" ) ;
    sh:minCount 1 ; sh:name "ignored annotation"
  ] ;
  rdfs:label "ignored" .
ex:P a sh:PropertyShape ;
  sh:path ex:age ; sh:datatype xsd:integer ; sh:maxCount 0 ;
  sh:minInclusive 0 ; sh:maxInclusive 150 ; sh:minExclusive -1 ; sh:maxExclusive 151 ;
  sh:in () ; sh:message "m" ; sh:severity sh:Info .
`

const richJSON = `{"node_shapes": [{
  "id": "http://example.org/S",
  "target_classes": ["http://example.org/A", "http://example.org/B"],
  "closed": true,
  "ignored_properties": ["http://example.org/q", "http://www.w3.org/1999/02/22-rdf-syntax-ns#type"],
  "severity": "http://www.w3.org/ns/shacl#Warning",
  "messages": ["first", "second"],
  "properties": [
    {"id": "http://example.org/P", "path": "http://example.org/age",
     "datatype": "http://www.w3.org/2001/XMLSchema#integer", "max_count": 0,
     "min_inclusive": "0", "max_inclusive": "150", "min_exclusive": "-1", "max_exclusive": "151",
     "in": [], "messages": ["m"], "severity": "http://www.w3.org/ns/shacl#Info"},
    {"path": "http://example.org/name", "datatype": "http://www.w3.org/2001/XMLSchema#string",
     "pattern": "^[a-z]+$", "in": ["a", "b"], "min_count": 1}
  ]
}]}`

const richYAML = `node_shapes:
  - id: "http://example.org/S"
    target_classes: ["http://example.org/B", "http://example.org/A"]
    closed: true
    ignored_properties: ["http://www.w3.org/1999/02/22-rdf-syntax-ns#type", "http://example.org/q"]
    severity: "http://www.w3.org/ns/shacl#Warning"
    messages: [second, first]
    properties:
      - path: "http://example.org/name"
        datatype: "http://www.w3.org/2001/XMLSchema#string"
        pattern: '^[a-z]+$'
        in: ["b", "a"]
        min_count: 1
      - id: "http://example.org/P"
        path: "http://example.org/age"
        datatype: "http://www.w3.org/2001/XMLSchema#integer"
        max_count: 0
        min_inclusive: "0"
        max_inclusive: "150"
        min_exclusive: "-1"
        max_exclusive: "151"
        in: []
        messages: [m]
        severity: "http://www.w3.org/ns/shacl#Info"
`

var richShapes = ShapeSet{NodeShapes: []NodeShape{{
	ID: "http://example.org/S", TargetClasses: []string{"http://example.org/A", "http://example.org/B"},
	Closed: true, IgnoredProperties: []string{"http://example.org/q", "http://www.w3.org/1999/02/22-rdf-syntax-ns#type"},
	Severity: sh + "Warning", Messages: []string{"first", "second"},
	Properties: []PropertyShape{
		{
			ID: "http://example.org/P", Path: "http://example.org/age", Datatype: xsd + "integer", MaxCount: ptr(0),
			MinInclusive: &Literal{"0", xsd + "integer"}, MaxInclusive: &Literal{"150", xsd + "integer"},
			MinExclusive: &Literal{"-1", xsd + "integer"}, MaxExclusive: &Literal{"151", xsd + "integer"},
			In: &[]string{}, Messages: []string{"m"}, Severity: sh + "Info",
		},
		{
			Path: "http://example.org/name", Datatype: xsd + "string", Pattern: ptr("^[a-z]+$"),
			In: &[]string{"a", "b"}, MinCount: 1,
		},
	},
}}}

func TestEveryFieldInThreeFormats(t *testing.T) {
	fromTurtle, err := ParseTurtle(strings.NewReader(richTurtle))
	if err != nil {
		t.Fatal(err)
	}
	fromJSON, err := DecodeShapesJSON(strings.NewReader(richJSON))
	if err != nil {
		t.Fatal(err)
	}
	fromYAML, err := DecodeShapesYAML(strings.NewReader(richYAML))
	if err != nil {
		t.Fatal(err)
	}
	for name, got := range map[string]ShapeSet{"Turtle": fromTurtle, "JSON": fromJSON, "YAML": fromYAML} {
		if !reflect.DeepEqual(got, richShapes) {
			t.Errorf("%s:\n got %+v\nwant %+v", name, got, richShapes)
		}
	}
}

// TestParseTurtleSizeLimit shows ParseTurtle rejects one byte over 1 MiB,
// naming the byte offset, and accepts exactly 1 MiB (ruling Q25).
func TestParseTurtleSizeLimit(t *testing.T) {
	const limit = 1 << 20
	doc := func(size int) string {
		body := prefix + "ex:S a sh:NodeShape ; sh:targetClass ex:C .\n#"
		return body + strings.Repeat("a", size-len(body))
	}
	got, err := ParseTurtle(strings.NewReader(doc(limit)))
	if err != nil || len(got.NodeShapes) != 1 {
		t.Fatalf("exactly %d bytes: %+v, %v; want one node shape", limit, got, err)
	}
	want := fmt.Sprintf("input exceeds the %d-byte limit at byte offset %d", limit, limit)
	if _, err := ParseTurtle(strings.NewReader(doc(limit + 1))); err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("%d bytes: err = %v, want containing %q", limit+1, err, want)
	}
}

// TestParseTurtleRefuses covers the inputs ParseTurtle fails on, rather than
// recording for Compile.
func TestParseTurtleRefuses(t *testing.T) {
	tests := []struct{ name, src, want string }{
		{"blank typed node shape", prefix + "[] a sh:NodeShape ; sh:property [ sh:path ex:p ] .", "blank-node node shape"},
		{"blank targeting node shape", prefix + "_:s sh:targetClass ex:C .", "blank-node node shape"},
		{"blank node shape with other target", prefix + "[ sh:targetNode ex:n ] .", "blank-node node shape"},
		{"relative IRI", prefix + "<S> a sh:NodeShape .", "line 5 col 1: unsupported relative IRI <S>"},
		{"syntax", prefix + "ex:S a sh:NodeShape", "expected '.'"},
		{"nesting", prefix + "ex:S ex:p " + strings.Repeat("[ ex:p ", 65) + strings.Repeat("]", 65) + " .", "nesting exceeds 64 levels"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseTurtle(strings.NewReader(tt.src)); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want containing %q", err, tt.want)
			}
		})
	}
}

// TestParseTurtleRemovesRepeats is design D3: an exact repeat is one
// constraint, while two different values are recorded for Compile.
func TestParseTurtleRemovesRepeats(t *testing.T) {
	repeated := prefix + "ex:S sh:targetClass ex:C ; sh:property ex:P .\nex:P sh:path ex:p ; sh:minCount 1 .\nex:P sh:minCount 1 .\n"
	got, err := ParseTurtle(strings.NewReader(repeated))
	if err != nil {
		t.Fatal(err)
	}
	if p := got.NodeShapes[0].Properties[0]; p.MinCount != 1 || len(got.rejections) != 0 {
		t.Fatalf("repeated minCount: %+v, rejections %+v", p, got.rejections)
	}
	got, err = ParseTurtle(strings.NewReader(strings.Replace(repeated, "ex:P sh:minCount 1 .", "ex:P sh:minCount 2 .", 1)))
	if err != nil {
		t.Fatal(err)
	}
	requireRejection(t, got, rejection{Construct: sh + "minCount", Shape: "http://example.org/P", Path: "http://example.org/p"})
}

// requireRejection fails unless set records want. An empty Detail in want
// matches any detail; otherwise want.Detail must be contained in it.
func requireRejection(t *testing.T, set ShapeSet, want rejection) {
	t.Helper()
	for _, r := range set.rejections {
		if r.Construct == want.Construct && r.Shape == want.Shape && r.Path == want.Path &&
			strings.Contains(r.Detail, want.Detail) {
			return
		}
	}
	t.Fatalf("no rejection %+v in:\n%s", want, formatRejections(set.rejections))
}

func formatRejections(rs []rejection) string {
	var b strings.Builder
	for _, r := range rs {
		fmt.Fprintf(&b, "  %+v\n", r)
	}
	return b.String()
}

// TestParseTurtleRecordsRejections shows out-of-profile constructs are
// recorded with construct, shape, and path (ruling Q26), not dropped.
func TestParseTurtleRecordsRejections(t *testing.T) {
	src := prefix + `@prefix owl: <http://www.w3.org/2002/07/owl#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
<http://example.org/doc> owl:imports <http://example.org/other> .
ex:S a sh:NodeShape , rdfs:Class ;
  sh:rule [ a sh:TripleRule ; sh:subject sh:this ] ;
  sh:property [ sh:path ex:typed ; sh:datatype xsd:string ; sh:in ( "a" 1 ) ] ;
  sh:property [ sh:path ( ex:a ex:b ) ] ;
  sh:property [ sh:path [ sh:inversePath ex:parent ] ; sh:maxCount 1 ] ;
  sh:property [ sh:path ex:lang ; sh:pattern "x"@en ; sh:minCount 1.5 ] ;
  sh:property [ sh:datatype xsd:string ] .
`
	got, err := ParseTurtle(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	const s = "http://example.org/S"
	for _, want := range []rejection{
		{Construct: "http://www.w3.org/2002/07/owl#imports", Shape: "http://example.org/doc"},
		{Construct: "http://www.w3.org/2000/01/rdf-schema#Class", Shape: s, Detail: "implicit class target"},
		{Construct: sh + "rule", Shape: s},
		{Construct: sh + "in", Shape: s, Path: "http://example.org/typed", Detail: `"1"^^<` + xsd + `integer>`},
		{Construct: sh + "path", Shape: s, Detail: "sequence path"},
		{Construct: sh + "inversePath", Shape: s},
		{Construct: sh + "pattern", Shape: s, Path: "http://example.org/lang", Detail: `"x"@en`},
		{Construct: sh + "minCount", Shape: s, Path: "http://example.org/lang", Detail: "1.5"},
		{Construct: sh + "path", Shape: s, Detail: "no sh:path"},
	} {
		requireRejection(t, got, want)
	}
	for _, r := range got.rejections {
		if strings.Contains(fmt.Sprint(r), "_:") {
			t.Errorf("rejection names a blank-node label: %+v", r)
		}
		// Triples inside sh:rule's blank node belong to the sh:rule record.
		if r.Construct == sh+"subject" || r.Construct == "http://www.w3.org/1999/02/22-rdf-syntax-ns#first" {
			t.Errorf("triple under a rejected construct recorded on its own: %+v", r)
		}
	}
}

// loadW3C reads one vendored SHACL Core case (sha256 checked) the way the
// test-suite loader does: with a base, through the same builder ParseTurtle
// uses.
func loadW3C(t *testing.T, rel string) ShapeSet {
	t.Helper()
	data, err := oracle.ReadFixture(oracle.W3CDataShapes, "data-shapes-test-suite/tests/core/"+rel)
	if err != nil {
		t.Fatal(err)
	}
	triples, err := turtle.Parse(bytes.NewReader(data), turtle.WithBase("http://semvocab.test/core/"+rel))
	if err != nil {
		t.Fatal(err)
	}
	set, err := fromTriples(triples)
	if err != nil {
		t.Fatalf("%s: %v", rel, err)
	}
	return set
}

// TestW3CRejectedFamiliesRecorded parses every vendored rejected-family case
// and requires its family's construct to be recorded. Blank-node shapes
// nested in sh:and or sh:xone do not fail parsing (ruling Q27).
func TestW3CRejectedFamiliesRecorded(t *testing.T) {
	cases := map[string]string{
		"targets/targetSubjectsOf-001.ttl":     sh + "targetSubjectsOf",
		"targets/targetObjectsOf-001.ttl":      sh + "targetObjectsOf",
		"targets/targetClassImplicit-001.ttl":  "http://www.w3.org/2000/01/rdf-schema#Class",
		"path/path-sequence-001.ttl":           sh + "path",
		"path/path-alternative-001.ttl":        sh + "alternativePath",
		"path/path-inverse-001.ttl":            sh + "inversePath",
		"path/path-zeroOrMore-001.ttl":         sh + "zeroOrMorePath",
		"path/path-oneOrMore-001.ttl":          sh + "oneOrMorePath",
		"path/path-zeroOrOne-001.ttl":          sh + "zeroOrOnePath",
		"property/class-001.ttl":               sh + "class",
		"property/node-002.ttl":                sh + "node",
		"node/nodeKind-001.ttl":                sh + "nodeKind",
		"property/minLength-001.ttl":           sh + "minLength",
		"property/maxLength-001.ttl":           sh + "maxLength",
		"property/languageIn-001.ttl":          sh + "languageIn",
		"property/uniqueLang-001.ttl":          sh + "uniqueLang",
		"property/pattern-002.ttl":             sh + "flags",
		"property/hasValue-001.ttl":            sh + "hasValue",
		"property/equals-001.ttl":              sh + "equals",
		"property/disjoint-001.ttl":            sh + "disjoint",
		"property/lessThan-001.ttl":            sh + "lessThan",
		"property/lessThanOrEquals-001.ttl":    sh + "lessThanOrEquals",
		"property/not-001.ttl":                 sh + "not",
		"property/and-001.ttl":                 sh + "and",
		"property/or-001.ttl":                  sh + "or",
		"node/xone-001.ttl":                    sh + "xone",
		"property/qualifiedValueShape-001.ttl": sh + "qualifiedValueShape",
		"misc/deactivated-001.ttl":             sh + "deactivated",
	}
	for rel, construct := range cases {
		set := loadW3C(t, rel)
		found := false
		for _, r := range set.rejections {
			found = found || r.Construct == construct
		}
		if !found {
			t.Errorf("%s: %s not recorded in:\n%s", rel, construct, formatRejections(set.rejections))
		}
	}
	// The spec's "several rejected constructs" case, with shape and path.
	and := loadW3C(t, "property/and-001.ttl")
	const ex = "http://datashapes.org/sh/tests/core/property/and-001.test#"
	requireRejection(t, and, rejection{Construct: sh + "and", Shape: ex + "AddressShape-address", Path: ex + "address"})
	requireRejection(t, and, rejection{Construct: sh + "targetNode", Shape: ex + "AddressShape"})
	for _, r := range and.rejections {
		if r.Construct == sh+"minCount" || r.Construct == sh+"property" {
			t.Errorf("and-001: shape inside sh:and recorded on its own: %+v", r)
		}
	}
}

func TestDecodeShapesRefuses(t *testing.T) {
	const node = `"id": "http://example.org/S"`
	const prop = `"path": "http://example.org/p"`
	jsonTests := []struct{ name, src, want string }{
		{"empty", ``, "EOF"},
		{"unknown top field", `{"node_shapes": [], "bogus": 1}`, `unknown field "bogus"`},
		{"unknown node field", `{"node_shapes": [{` + node + `, "bogus": 1}]}`, `unknown field "bogus"`},
		{"unknown property field", `{"node_shapes": [{` + node + `, "properties": [{` + prop + `, "bogus": 1}]}]}`, `unknown field "bogus"`},
		{"trailing value", `{"node_shapes": []} {}`, "trailing data"},
		{"number bound", `{"node_shapes": [{` + node + `, "properties": [{` + prop + `, "min_inclusive": 5}]}]}`, "min_inclusive"},
		{"number in member", `{"node_shapes": [{` + node + `, "properties": [{` + prop + `, "in": [1]}]}]}`, "in"},
		{"blank node shape", `{"node_shapes": [{"id": "_:s"}]}`, "blank-node node shape"},
		{"relative id", `{"node_shapes": [{"id": "S"}]}`, `"S" is not an absolute IRI`},
		{"relative target", `{"node_shapes": [{` + node + `, "target_classes": ["C"]}]}`, `"C" is not an absolute IRI`},
		{"relative path", `{"node_shapes": [{` + node + `, "properties": [{"path": "p"}]}]}`, `"p" is not an absolute IRI`},
		{"relative datatype", `{"node_shapes": [{` + node + `, "properties": [{` + prop + `, "datatype": "string"}]}]}`, `"string" is not an absolute IRI`},
		{"missing path", `{"node_shapes": [{` + node + `, "properties": [{}]}]}`, "no path"},
		{"duplicate node shape", `{"node_shapes": [{` + node + `}, {` + node + `}]}`, "listed twice"},
	}
	for _, tt := range jsonTests {
		t.Run("json "+tt.name, func(t *testing.T) {
			if _, err := DecodeShapesJSON(strings.NewReader(tt.src)); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want containing %q", err, tt.want)
			}
		})
	}
	const ynode = "node_shapes:\n  - id: \"http://example.org/S\"\n"
	const yprop = ynode + "    properties:\n      - path: \"http://example.org/p\"\n"
	yamlTests := []struct{ name, src, want string }{
		{"empty", ``, "no document"},
		{"unknown top field", "node_shapes: []\nbogus: 1\n", "field bogus not found"},
		{"unknown node field", ynode + "    bogus: x\n", "field bogus not found"},
		{"unknown property field", yprop + "        bogus: x\n", "field bogus not found"},
		{"trailing document", "node_shapes: []\n---\nnode_shapes: []\n", "trailing document"},
		{"unquoted bound", yprop + "        min_inclusive: 5\n", "line 5: min_inclusive must be a string"},
		{"tagged int bound", yprop + "        min_inclusive: !!int \"5\"\n", "min_inclusive must be a string"},
		{"unquoted in member", yprop + "        in: [1]\n", "in must be a string"},
		{"unquoted pattern", yprop + "        pattern: 5\n", "pattern must be a string"},
		{"null severity", ynode + "    severity: ~\n", "severity must be a string"},
		{"string count", yprop + "        min_count: \"1\"\n", "line 5: cannot unmarshal !!str `1` into int"},
		{"duplicate key", ynode + "    id: \"http://example.org/T\"\n", "already defined"},
		{"blank node shape", "node_shapes:\n  - id: \"_:s\"\n", "blank-node node shape"},
		{"relative id", "node_shapes:\n  - id: S\n", `"S" is not an absolute IRI`},
	}
	for _, tt := range yamlTests {
		t.Run("yaml "+tt.name, func(t *testing.T) {
			if _, err := DecodeShapesYAML(strings.NewReader(tt.src)); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want containing %q", err, tt.want)
			}
		})
	}
}

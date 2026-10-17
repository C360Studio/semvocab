package shacl

import (
	"errors"
	"testing"

	"github.com/c360studio/semengine/vocabulary"
)

// w3cRejectedFamilies maps each vendored rejected-family case
// (notes/2-fixtures.md) to the construct its rejection names.
var w3cRejectedFamilies = map[string]string{
	"targets/targetSubjectsOf-001.ttl":     sh + "targetSubjectsOf",
	"targets/targetObjectsOf-001.ttl":      sh + "targetObjectsOf",
	"targets/targetClassImplicit-001.ttl":  rdfsClass,
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

// TestW3CRejectedFamiliesFailCompile is task 5.2's W3C half (design D15):
// every vendored rejected-family case fails Compile with no adapter, no
// mapping, and an empty pack, naming its construct and the shape it is on,
// and returns no bundle.
func TestW3CRejectedFamiliesFailCompile(t *testing.T) {
	for rel, construct := range w3cRejectedFamilies {
		b, err := Compile(vocabulary.PackSpec{Name: "w3c"}, loadW3C(t, rel), Mapping{})
		if b != nil {
			t.Errorf("%s: a partial bundle was returned", rel)
		}
		var ce *CompileError
		if !errors.As(err, &ce) {
			t.Errorf("%s: err = %v, want a *CompileError", rel, err)
			continue
		}
		found := false
		for _, is := range ce.Issues {
			if is.Construct == construct && is.Shape != "" {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: no issue names %s with its shape:\n%v", rel, construct, err)
		}
	}

	// The spec's "several rejected constructs" scenario: sh:targetNode and
	// sh:and are both listed, each with its shape, and sh:and with its path.
	const and = "http://datashapes.org/sh/tests/core/property/and-001.test#"
	_, err := Compile(vocabulary.PackSpec{Name: "w3c"}, loadW3C(t, "property/and-001.ttl"), Mapping{})
	is := requireIssue(t, err, sh+"and", reasonLogical)
	if is.Shape != and+"AddressShape-address" || is.Path != and+"address" {
		t.Errorf("sh:and issue = %+v, want shape AddressShape-address and path address", is)
	}
	if is := requireIssue(t, err, sh+"targetNode", reasonTarget); is.Shape != and+"AddressShape" {
		t.Errorf("sh:targetNode issue = %+v, want shape AddressShape", is)
	}
}

// TestRejectedConstructsReportedTogether is task 5.2's hand-written half: a
// rule, an unsupported target, a non-predicate path, a logical constraint, a
// SPARQL constraint, and an unknown sh: term in one document are all
// reported, sorted by shape, path, and construct, with no bundle; sh:rule's
// message names the Phase 4 deferral.
func TestRejectedConstructsReportedTogether(t *testing.T) {
	b, err := compileTurtle(t, `
ex:S sh:targetClass ex:C ;
  sh:rule [ a sh:TripleRule ; sh:subject sh:this ; sh:predicate ex:p ; sh:object ex:o ] ;
  sh:targetNode ex:n ;
  sh:sparql [ sh:select "SELECT $this WHERE {}" ] ;
  sh:property [ sh:path [ sh:inversePath ex:p ] ] ;
  sh:property [ sh:path ex:q ; sh:or ( [ sh:datatype xsd:string ] [ sh:datatype xsd:integer ] ) ] ;
  sh:property [ sh:path ex:r ; sh:minSize 1 ] .
`)
	if b != nil {
		t.Error("a partial bundle was returned")
	}
	rule := requireIssue(t, err, sh+"rule", "rule translation is deferred to issue #1 Phase 4")
	if rule.Shape != ex+"S" {
		t.Errorf("sh:rule issue = %+v, want shape ex:S", rule)
	}
	requireIssue(t, err, sh+"targetNode", reasonTarget)
	requireIssue(t, err, sh+"sparql", reasonSPARQL)
	requireIssue(t, err, sh+"inversePath", reasonPath)
	if or := requireIssue(t, err, sh+"or", reasonLogical); or.Path != ex+"q" {
		t.Errorf("sh:or issue = %+v, want path ex:q", or)
	}
	if u := requireIssue(t, err, sh+"minSize", "not a term"); u.Path != ex+"r" {
		t.Errorf("unknown term issue = %+v, want path ex:r", u)
	}

	var ce *CompileError
	errors.As(err, &ce)
	for i := 1; i < len(ce.Issues); i++ {
		a, b := ce.Issues[i-1], ce.Issues[i]
		if a.Shape > b.Shape || a.Shape == b.Shape && (a.Path > b.Path || a.Path == b.Path && a.Construct > b.Construct) {
			t.Errorf("issues out of order: %+v before %+v", a, b)
		}
	}
	// Nothing inside the rule's or the SPARQL constraint's blank node is
	// reported on its own (ruling Q27).
	for _, is := range ce.Issues {
		switch is.Construct {
		case sh + "subject", sh + "predicate", sh + "object", sh + "select", sh + "datatype":
			t.Errorf("a triple inside a rejected construct was reported on its own: %+v", is)
		}
	}
}

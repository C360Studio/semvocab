package shacl

import (
	"bytes"
	"errors"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"
	"unicode"

	"github.com/c360studio/semvocab/internal/oracle"
	"github.com/c360studio/semvocab/internal/turtle"
)

// vocabularyTerms reads the vendored W3C SHACL vocabulary (sha256 checked)
// and returns every term it defines: each subject rdfs:isDefinedBy sh:,
// except the 38 parameter declarations (sh:AndConstraintComponent-and and
// the like). Those are sh:Parameter nodes that declare a constraint
// component's parameter; the term each declares is its sh:path, which the
// vocabulary defines on its own.
func vocabularyTerms(t *testing.T) []string {
	t.Helper()
	data, err := oracle.ReadFixture(oracle.W3CDataShapes, "shacl/shacl.ttl")
	if err != nil {
		t.Fatal(err)
	}
	triples, err := turtle.Parse(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	parameters := map[string]bool{}
	for _, tr := range triples {
		if tr.P.Value == rdfType && tr.O.Value == shNS+"Parameter" {
			parameters[tr.S.Value] = true
		}
	}
	var terms []string
	for _, tr := range triples {
		if tr.P.Value == rdfsNS+"isDefinedBy" && tr.O.Value == shNS && strings.HasPrefix(tr.S.Value, shNS) &&
			!parameters[tr.S.Value] {
			terms = append(terms, tr.S.Value)
		}
	}
	slices.Sort(terms)
	return slices.Compact(terms)
}

// profileGaps names every vocabulary term the table does not classify, and
// every table entry the vocabulary does not define.
func profileGaps(terms []string, table map[string]profileEntry) []string {
	var gaps []string
	for _, term := range terms {
		if _, ok := table[term]; !ok {
			gaps = append(gaps, "unclassified: "+term)
		}
	}
	for _, term := range slices.Sorted(maps.Keys(table)) {
		if !slices.Contains(terms, term) {
			gaps = append(gaps, "not in the vocabulary: "+term)
		}
	}
	return gaps
}

// supportedSnippets use each supported term where design D4 accepts it, and
// each must compile.
var supportedSnippets = map[string]string{
	"NodeShape": `ex:S a sh:NodeShape ; sh:targetClass ex:C .`,
	"PropertyShape": `ex:S sh:targetClass ex:C ; sh:property ex:P .
ex:P a sh:PropertyShape ; sh:path ex:p .`,
	"targetClass":       `ex:S sh:targetClass ex:C .`,
	"property":          `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ] .`,
	"path":              `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ] .`,
	"minCount":          `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:minCount 1 ] .`,
	"maxCount":          `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:maxCount 2 ] .`,
	"datatype":          `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:datatype xsd:integer ] .`,
	"minExclusive":      `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:datatype xsd:integer ; sh:minExclusive 0 ] .`,
	"minInclusive":      `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:datatype xsd:integer ; sh:minInclusive 0 ] .`,
	"maxExclusive":      `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:datatype xsd:double ; sh:maxExclusive 1.5e0 ] .`,
	"maxInclusive":      `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:datatype xsd:double ; sh:maxInclusive 1.5e0 ] .`,
	"pattern":           `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:pattern "^a" ] .`,
	"in":                `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:datatype xsd:string ; sh:in ( "a" "b" ) ] .`,
	"closed":            `ex:S sh:targetClass ex:C ; sh:closed true ; sh:ignoredProperties ( rdf:type ) .`,
	"ignoredProperties": `ex:S sh:targetClass ex:C ; sh:ignoredProperties ( rdf:type ex:p ) .`,
	"severity":          `ex:S sh:targetClass ex:C ; sh:severity sh:Warning ; sh:property [ sh:path ex:p ; sh:severity sh:Info ] .`,
	"Violation":         `ex:S sh:targetClass ex:C ; sh:severity sh:Violation .`,
	"Warning":           `ex:S sh:targetClass ex:C ; sh:severity sh:Warning .`,
	"Info":              `ex:S sh:targetClass ex:C ; sh:severity sh:Info .`,
	"message":           `ex:S sh:targetClass ex:C ; sh:message "m" ; sh:property [ sh:path ex:p ; sh:message "n" ] .`,
}

// requireIssue fails unless err is a *CompileError with an issue whose
// construct is construct and whose message contains each of contains. It
// returns that issue.
func requireIssue(t *testing.T, err error, construct string, contains ...string) Issue {
	t.Helper()
	var ce *CompileError
	if !errors.As(err, &ce) {
		t.Fatalf("err = %v, want a *CompileError naming %s", err, construct)
	}
	for _, is := range ce.Issues {
		if is.Construct != construct {
			continue
		}
		ok := true
		for _, c := range contains {
			ok = ok && strings.Contains(is.Message, c)
		}
		if ok {
			return is
		}
	}
	t.Fatalf("no issue %s with %q in:\n%v", construct, contains, err)
	return Issue{}
}

// TestProfileClassifiesEveryTerm is task 5.1 (design D4, D15): every term of
// the vendored W3C vocabulary has exactly one classification, and is handled
// as classified: supported terms compile where D4 accepts them, ignored
// annotations compile to the same bundle as without them, and rejected terms
// fail compilation with their IRI named.
func TestProfileClassifiesEveryTerm(t *testing.T) {
	terms := vocabularyTerms(t)
	if len(terms) != 183 {
		t.Fatalf("the vendored vocabulary defines %d terms, want 183 (design D4)", len(terms))
	}
	if gaps := profileGaps(terms, profile); len(gaps) > 0 {
		t.Fatalf("profile table and vocabulary disagree:\n%s", strings.Join(gaps, "\n"))
	}

	plain, err := compileTurtle(t, `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ] .`)
	if err != nil {
		t.Fatal(err)
	}
	for _, term := range terms {
		e := profile[term]
		local := strings.TrimPrefix(term, shNS)
		if e.section == "" {
			t.Errorf("%s has no section", local)
		}
		switch e.class {
		case termSupported:
			src, ok := supportedSnippets[local]
			if !ok {
				t.Errorf("supported term %s has no snippet showing it compiles", local)
				continue
			}
			if _, err := compileTurtle(t, src); err != nil {
				t.Errorf("supported term %s does not compile: %v", local, err)
			}
		case termIgnored:
			b, err := compileTurtle(t, `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:`+local+` "x" ] .`)
			if err != nil {
				t.Errorf("ignored annotation %s does not compile: %v", local, err)
			} else if !reflect.DeepEqual(b, plain) {
				t.Errorf("ignored annotation %s changed the bundle", local)
			}
		case termRejected:
			if e.reason == "" {
				t.Errorf("rejected term %s has no reason", local)
			}
			if unicode.IsUpper(rune(local[0])) {
				// A class or individual: rejected as a shape's type.
				_, err := compileTurtle(t, `ex:S a sh:NodeShape , sh:`+local+` ; sh:targetClass ex:C .`)
				if err == nil || !strings.Contains(err.Error(), "<"+term+">") {
					t.Errorf("rejected term %s as a type: err = %v, want it named", local, err)
				}
				continue
			}
			_, err := compileTurtle(t, `ex:S sh:targetClass ex:C ; sh:`+local+` ex:v .`)
			requireIssue(t, err, term, e.reason)
			inProperty := `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:` + local + ` ex:v ] .`
			if slices.Contains(targetPredicates, term) {
				// A target makes the blank property shape a blank-node node
				// shape, which parsing refuses, naming the target (ruling Q27).
				_, err := ParseTurtle(strings.NewReader(prefix + inProperty))
				if err == nil || !strings.Contains(err.Error(), term) {
					t.Errorf("target %s on a blank property shape: err = %v, want it named", local, err)
				}
				continue
			}
			_, err = compileTurtle(t, inProperty)
			requireIssue(t, err, term, e.reason)
		default:
			t.Errorf("%s has no valid classification", local)
		}
	}
	for local := range supportedSnippets {
		if profile[shNS+local].class != termSupported {
			t.Errorf("snippet for %s, which is not supported", local)
		}
	}
}

// TestProfileClassifiesEveryTermSensitivity shows the completeness check can
// fail: a removed classification and an entry for a term the vocabulary does
// not define are each named.
func TestProfileClassifiesEveryTermSensitivity(t *testing.T) {
	terms := vocabularyTerms(t)
	dropped := maps.Clone(profile)
	delete(dropped, shNS+"rule")
	if got := profileGaps(terms, dropped); !slices.Equal(got, []string{"unclassified: " + shNS + "rule"}) {
		t.Errorf("one classification removed: gaps = %v", got)
	}
	extra := maps.Clone(profile)
	extra[shNS+"notATerm"] = profileEntry{class: termRejected, section: "none", reason: "test"}
	if got := profileGaps(terms, extra); !slices.Equal(got, []string{"not in the vocabulary: " + shNS + "notATerm"}) {
		t.Errorf("one extra entry: gaps = %v", got)
	}
}

// TestUnknownSHACLTermRejected: an sh: term the vocabulary does not define
// is rejected as unknown (design D4).
func TestUnknownSHACLTermRejected(t *testing.T) {
	_, err := compileTurtle(t, `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:maxSize 3 ] .`)
	is := requireIssue(t, err, shNS+"maxSize", "not a term of the W3C SHACL vocabulary")
	if is.Shape != ex+"S" || is.Path != ex+"p" {
		t.Errorf("issue = %+v, want shape ex:S and path ex:p", is)
	}
}

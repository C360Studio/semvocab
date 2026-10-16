package turtle_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/c360studio/semvocab/internal/oracle"
	"github.com/c360studio/semvocab/internal/turtle"
)

// rdflibCounts are the distinct-triple counts rdflib 7.1.4 gives for the
// vendored files, each parsed with base http://semvocab.test/core/<path below>
// (phase-2-shacl-compilation notes/2-fixtures.md, from probe P2's script).
// rdflib is the independent parser; it is not a dependency.
var rdflibCounts = map[string]int{
	"misc/deactivated-001.ttl": 22, "node/closed-001.ttl": 39, "node/closed-002.ttl": 34,
	"node/minInclusive-001.ttl": 25, "node/nodeKind-001.ttl": 24, "node/xone-001.ttl": 45,
	"path/path-alternative-001.ttl": 58, "path/path-inverse-001.ttl": 51, "path/path-oneOrMore-001.ttl": 40,
	"path/path-sequence-001.ttl": 53, "path/path-zeroOrMore-001.ttl": 32, "path/path-zeroOrOne-001.ttl": 32,
	"property/and-001.ttl": 68, "property/class-001.ttl": 53, "property/datatype-001.ttl": 48,
	"property/datatype-002.ttl": 41, "property/disjoint-001.ttl": 50, "property/equals-001.ttl": 78,
	"property/hasValue-001.ttl": 39, "property/in-001.ttl": 44, "property/languageIn-001.ttl": 53,
	"property/lessThan-001.ttl": 59, "property/lessThanOrEquals-001.ttl": 51, "property/maxCount-001.ttl": 37,
	"property/maxCount-002.ttl": 32, "property/maxExclusive-001.ttl": 51, "property/maxInclusive-001.ttl": 41,
	"property/maxLength-001.ttl": 34, "property/minCount-001.ttl": 35, "property/minCount-002.ttl": 20,
	"property/minExclusive-001.ttl": 41, "property/minLength-001.ttl": 34, "property/node-002.ttl": 40,
	"property/not-001.ttl": 40, "property/or-001.ttl": 45, "property/pattern-001.ttl": 46,
	"property/pattern-002.ttl": 39, "property/qualifiedValueShape-001.ttl": 47,
	"property/uniqueLang-001.ttl": 57, "targets/targetClass-001.ttl": 33,
	"targets/targetClassImplicit-001.ttl": 33, "targets/targetObjectsOf-001.ttl": 37,
	"targets/targetSubjectsOf-001.ttl": 30,
}

const suite = "data-shapes-test-suite/tests/core/"

func distinct(ts []turtle.Triple) int {
	seen := map[turtle.Triple]bool{}
	for _, t := range ts {
		seen[t] = true
	}
	return len(seen)
}

// TestW3CTestSuiteWithBase reads every vendored SHACL Core case (sha256
// checked first) with a caller-supplied base and compares its distinct-triple
// count with rdflib's. Without the base, each case fails at its first relative
// IRI, the manifest's <>.
func TestW3CTestSuiteWithBase(t *testing.T) {
	read := 0
	for rel, want := range rdflibCounts {
		data, err := oracle.ReadFixture(oracle.W3CDataShapes, suite+rel)
		if err != nil {
			t.Fatal(err)
		}
		got, err := turtle.Parse(bytes.NewReader(data), turtle.WithBase("http://semvocab.test/core/"+rel))
		if err != nil {
			t.Errorf("%s: %v", rel, err)
			continue
		}
		if n := distinct(got); n != want {
			t.Errorf("%s: %d distinct triples, rdflib %d", rel, n, want)
		}
		if _, err := turtle.Parse(bytes.NewReader(data)); err == nil ||
			!strings.Contains(err.Error(), "unsupported relative IRI <>") {
			t.Errorf("%s without a base: err = %v, want the relative IRI <> rejected", rel, err)
		}
		read++
	}
	if read != 43 {
		t.Errorf("read %d cases, want the 43 vendored", read)
	}
}

// TestW3CVocabulary reads the vendored SHACL vocabulary, which needs no base.
func TestW3CVocabulary(t *testing.T) {
	data, err := oracle.ReadFixture(oracle.W3CDataShapes, "shacl/shacl.ttl")
	if err != nil {
		t.Fatal(err)
	}
	got, err := turtle.Parse(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1127 || distinct(got) != 1127 {
		t.Fatalf("%d triples (%d distinct), rdflib 1127", len(got), distinct(got))
	}
}

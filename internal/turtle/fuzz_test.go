package turtle_test

// Task 7.2: FuzzTurtleReader (design D16). Plain go test runs the seeds in
// testdata/fuzz/FuzzTurtleReader and those added below; a bounded -fuzz run
// is recorded in phase-2-shacl-compilation notes/7-generated.md.

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/c360studio/semvocab/internal/oracle"
	"github.com/c360studio/semvocab/internal/turtle"
)

// readerError is every error form the reader returns for its input: each
// names a line and column, or the byte offset of invalid UTF-8.
var readerError = regexp.MustCompile(`(?s)^turtle: (line \d+ col \d+: .+|input is not valid UTF-8 at byte offset \d+)$`)

const fuzzBase = "http://semvocab.test/fuzz/doc"

// FuzzTurtleReader: the reader never panics, and returns either an error
// naming a location or triples whose terms are well formed: subjects are IRIs
// or blank nodes, predicates are absolute IRIs, every literal has a datatype,
// and every IRI is absolute whether or not a base resolved it.
func FuzzTurtleReader(f *testing.F) {
	// The 53 KB SHACL vocabulary is left out: TestW3CVocabulary reads it, and
	// as a seed it slows every mutation and minimization.
	for rel := range rdflibCounts {
		data, err := oracle.ReadFixture(oracle.W3CDataShapes, suite+rel)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(data, true)
	}
	diekm, err := os.ReadFile(filepath.Join("..", "..", "packs", "diekm", "diekm.shacl.ttl"))
	if err != nil {
		f.Fatal(err)
	}
	f.Add(diekm, false)
	// Nesting at the cap and one over (design D2).
	for _, depth := range []int{64, 65} {
		f.Add([]byte("<http://ex/s> <http://ex/p> "+strings.Repeat("[ <http://ex/p> ", depth)+
			"1"+strings.Repeat(" ]", depth)+" ."), false)
	}

	f.Fuzz(func(t *testing.T, data []byte, withBase bool) {
		var opts []turtle.Option
		if withBase {
			opts = append(opts, turtle.WithBase(fuzzBase))
		}
		triples, err := turtle.Parse(bytes.NewReader(data), opts...)
		if err != nil {
			if triples != nil {
				t.Fatalf("triples returned with error %v", err)
			}
			if !readerError.MatchString(err.Error()) {
				t.Fatalf("error names no location: %q", err)
			}
			return
		}
		for _, tr := range triples {
			if tr.S.Kind != turtle.IRI && tr.S.Kind != turtle.Blank {
				t.Fatalf("subject %+v is not an IRI or blank node", tr.S)
			}
			if tr.P.Kind != turtle.IRI {
				t.Fatalf("predicate %+v is not an IRI", tr.P)
			}
			for _, x := range []turtle.Term{tr.S, tr.P, tr.O} {
				switch x.Kind {
				case turtle.IRI:
					if !absoluteIRI.MatchString(x.Value) {
						t.Fatalf("IRI %q is not absolute", x.Value)
					}
				case turtle.Literal:
					if x.Datatype == "" {
						t.Fatalf("literal %+v has no datatype", x)
					}
				case turtle.Blank:
					if x.Value == "" {
						t.Fatalf("blank node with no label in %+v", tr)
					}
				default:
					t.Fatalf("term %+v has no kind", x)
				}
			}
		}
	})
}

// absoluteIRI is RFC 3986 §3.1's scheme followed by ':'.
var absoluteIRI = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*:`)

// TestParseRejectsInvalidScheme: an IRI whose text before the first ':' is
// not an RFC 3986 §3.1 scheme (ALPHA *( ALPHA / DIGIT / "+" / "-" / "." ))
// is neither absolute nor a usable relative reference (§4.2 forbids a colon
// in a relative path's first segment), so it fails with or without a base.
// FuzzTurtleReader found <0:> passed through as absolute (corpus entry
// testdata/fuzz/FuzzTurtleReader/7aa994686e8f9be9).
func TestParseRejectsInvalidScheme(t *testing.T) {
	for _, iri := range []string{"0:", "1a:b", "+x:y", "a_b:c", "é:x"} {
		src := "<http://ex/s> <http://ex/p> <" + iri + "> ."
		for _, opts := range [][]turtle.Option{nil, {turtle.WithBase("http://ex/base/")}} {
			_, err := turtle.Parse(strings.NewReader(src), opts...)
			if err == nil || !strings.Contains(err.Error(), "scheme") {
				t.Errorf("<%s> with %d option(s): err = %v, want an invalid-scheme error", iri, len(opts), err)
			}
		}
	}
	for _, iri := range []string{"a:", "urn:x", "h+t-t.p2:x"} {
		if _, err := turtle.Parse(strings.NewReader("<http://ex/s> <http://ex/p> <" + iri + "> .")); err != nil {
			t.Errorf("<%s>: %v", iri, err)
		}
	}
}

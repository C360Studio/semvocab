package shacl

// Task 7.2: the fuzz targets of design D16 in this package. Plain go test
// runs the seeds in testdata/fuzz/<target> and those added below; bounded
// -fuzz runs are recorded in notes/7-generated.md. The property for each:
// no panic; an error or a result; every error names a construct or a
// location. Each target lists the error forms it accepts, so a new form fails
// until someone reads it and adds it.

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/c360studio/semengine/vocabulary"
	"github.com/c360studio/semvocab/internal/oracle"
	"github.com/c360studio/semvocab/internal/turtle"
	"github.com/c360studio/semvocab/packs/diekm"
)

// requireNamed fails unless msg matches one of forms.
func requireNamed(t *testing.T, err error, forms ...*regexp.Regexp) {
	t.Helper()
	for _, f := range forms {
		if f.MatchString(err.Error()) {
			return
		}
	}
	t.Fatalf("error names no construct or location: %q", err)
}

// requireIssuesNamed compiles a decoded set against testPack and
// testMapping and DIEKM's pack and mapping. A failure must be a
// *CompileError whose every issue names its construct.
func requireIssuesNamed(t *testing.T, set ShapeSet) {
	t.Helper()
	spec, dm, err := fuzzDIEKM()
	if err != nil {
		t.Fatal(err)
	}
	for _, in := range []struct {
		name string
		run  func() error
	}{
		{"test", func() error { _, err := Compile(testPack(), set, testMapping()); return err }},
		{"diekm", func() error { _, err := Compile(spec, set, dm); return err }},
	} {
		err := in.run()
		if err == nil {
			continue
		}
		var ce *CompileError
		if !errors.As(err, &ce) || len(ce.Issues) == 0 {
			t.Fatalf("%s: compile error is not a *CompileError with issues: %v", in.name, err)
		}
		for _, is := range ce.Issues {
			if is.Construct == "" || is.Message == "" {
				t.Fatalf("%s: issue names no construct: %+v", in.name, is)
			}
		}
	}
}

// fuzzDIEKM loads DIEKM's pack and mapping once per process, outside the
// inputs the fuzzer times.
var fuzzDIEKMOnce = sync.OnceValues(func() (Mapping, error) {
	return DecodeMappingYAML(bytes.NewReader(diekm.Mapping()))
})

var fuzzDIEKMSpec = sync.OnceValues(diekm.Spec)

func fuzzDIEKM() (vocabulary.PackSpec, Mapping, error) {
	spec, err := fuzzDIEKMSpec()
	if err != nil {
		return vocabulary.PackSpec{}, Mapping{}, err
	}
	m, err := fuzzDIEKMOnce()
	return spec, m, err
}

var (
	turtleLocated  = regexp.MustCompile(`(?s)^shacl: turtle: (line \d+ col \d+: .+|input is not valid UTF-8 at byte offset \d+)$`)
	sizeLimit      = regexp.MustCompile(`^shacl: input exceeds the 1048576-byte limit at byte offset 1048576$`)
	blankNodeShape = regexp.MustCompile(`^shacl: blank-node node shape with \S.*: node shapes must be IRIs \(design D3\)$`)
)

// FuzzParseTurtleShapes fuzzes ParseTurtle, then Compile on what it builds.
func FuzzParseTurtleShapes(f *testing.F) {
	f.Add(diekm.Shapes())
	// The W3C cases, read with their base and written back as N-Triples, so
	// every IRI is absolute: they carry each rejected family.
	for rel := range w3cRejectedFamilies {
		data, err := oracle.ReadFixture(oracle.W3CDataShapes, "data-shapes-test-suite/tests/core/"+rel)
		if err != nil {
			f.Fatal(err)
		}
		triples, err := turtle.Parse(bytes.NewReader(data), turtle.WithBase("http://semvocab.test/core/"+rel))
		if err != nil {
			f.Fatal(err)
		}
		f.Add([]byte(writeNTriples(triples, identityLabels(triples))))
	}
	// The 1 MiB cap (design D2, ruling Q25): exactly at it, and one byte over.
	// These two seeds run under plain go test only. In a -fuzz session the
	// engine mutates and minimizes every seed, and two 1 MiB inputs held the
	// worker for most of a 60-second run (notes/7-generated.md).
	if fl := flag.Lookup("test.fuzz"); fl == nil || fl.Value.String() == "" {
		pad := func(n int) []byte {
			doc := []byte("<http://example.org/S> <http://www.w3.org/ns/shacl#targetClass> <http://example.org/C> .\n#")
			return append(doc, bytes.Repeat([]byte("x"), n-len(doc))...)
		}
		f.Add(pad(maxTurtleBytes))
		f.Add(pad(maxTurtleBytes + 1))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		set, err := ParseTurtle(bytes.NewReader(data))
		if err != nil {
			requireNamed(t, err, turtleLocated, sizeLimit, blankNodeShape)
			return
		}
		requireIssuesNamed(t, set)
	})
}

var (
	// jsonLocated is an error at a byte offset, from encoding/json's syntax
	// and type errors, or after the document.
	jsonLocated = regexp.MustCompile(`(?s)^shacl: shapes: byte offset \d+: .+$`)
	// jsonField is encoding/json naming the field it refused.
	jsonField = regexp.MustCompile(`(?s)^shacl: shapes: json: (unknown field "[^"]*"|cannot unmarshal \S+ into Go struct field \S+ of type .+)$`)
	// jsonPath is a null or a repeated key, named by its JSON path.
	jsonPath = regexp.MustCompile(`(?s)^shacl: shapes: \$.* is null; omit the field or give a value$|^shacl: shapes: \$.*: key ".*" repeated$`)
	// shapeRule is a model rule, naming the shape and the field.
	shapeRule = regexp.MustCompile(`(?s)^shacl: shapes: (blank-node node shape ".*": node shapes must be IRIs \(design D3\)|node shape .+ listed twice|node shape .+ has a property shape with no path|.+: ".*" is not an absolute IRI)$`)
	// emptyDocument names the whole input.
	emptyDocument = regexp.MustCompile(`^shacl: shapes: no document$`)
)

// FuzzDecodeShapesJSON fuzzes DecodeShapesJSON, then Compile.
func FuzzDecodeShapesJSON(f *testing.F) {
	f.Add(readFuzzSeed(f, "diekm.shapes.json"))
	f.Fuzz(func(t *testing.T, data []byte) {
		set, err := DecodeShapesJSON(bytes.NewReader(data))
		if err != nil {
			requireNamed(t, err, jsonLocated, jsonField, jsonPath, shapeRule, emptyDocument)
			return
		}
		requireIssuesNamed(t, set)
	})
}

var (
	// yamlLine is an error yaml.v3 or the decoder places on a line.
	yamlLine = regexp.MustCompile(`(?s)^shacl: shapes: (yaml: )?(unmarshal errors:\n)?\s*line \d+: .+$`)
	// yamlLocated is an error at a byte offset, found before yaml.v3 reads.
	yamlLocated = regexp.MustCompile(`(?s)^shacl: shapes: byte offset \d+: .+$`)
	// yamlConstruct is a yaml.v3 decoder error naming its construct.
	yamlConstruct = regexp.MustCompile(`^shacl: shapes: yaml: (unknown anchor '.*' referenced|anchor '.*' value contains itself|` +
		`document contains excessive aliasing|!!binary value contains invalid base64 data|invalid map key: .+|map merge requires .+)$`)
)

// FuzzDecodeShapesYAML fuzzes DecodeShapesYAML, then Compile.
func FuzzDecodeShapesYAML(f *testing.F) {
	f.Add(readFuzzSeed(f, "diekm.shapes.yaml"))
	f.Fuzz(func(t *testing.T, data []byte) {
		set, err := DecodeShapesYAML(bytes.NewReader(data))
		if err != nil {
			requireNamed(t, err, yamlLine, yamlLocated, yamlConstruct, shapeRule, emptyDocument)
			return
		}
		requireIssuesNamed(t, set)
	})
}

func readFuzzSeed(f *testing.F, name string) []byte {
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		f.Fatal(err)
	}
	return data
}

// FuzzPatternSubset fuzzes the regex subset translator (design D10). A
// refusal is a *patternError naming the construct and an offset inside the
// pattern. An acceptance compiles, and its text uses no escape outside XSD's
// single-character escapes and \p/\P: the multi-character escapes RE2 reads
// differently (\d \w \s and their negations, \i \c) and every escape XSD does
// not define must have been refused. The escape scan is independent of the
// translator's parser.
func FuzzPatternSubset(f *testing.F) {
	for _, p := range genPatterns {
		f.Add(p)
	}
	for _, p := range []string{`\d`, `[\w]`, `\p{IsBasicLatin}`, `\p{Cs}`, `(?i)a`, `[a-z-[aeiou]]`, `\1`, `\x41`,
		`a{1001}`, `\p{Lu`, `[`, `(`, `)`, `a**`, "\xff", strings.Repeat("(", 1001) + strings.Repeat(")", 1001)} {
		f.Add(p)
	}
	f.Fuzz(func(t *testing.T, pattern string) {
		re, err := compilePattern(pattern)
		if err != nil {
			var pe *patternError
			if !errors.As(err, &pe) || pe.construct == "" || pe.offset < 0 || pe.offset > len(pattern) {
				t.Fatalf("refusal names no construct or location: %#v", err)
			}
			return
		}
		if re == nil {
			t.Fatal("accepted with no regexp")
		}
		for i := 0; i < len(pattern); i++ {
			if pattern[i] != '\\' {
				continue
			}
			if i+1 == len(pattern) {
				t.Fatalf("accepted a trailing backslash in %q", pattern)
			}
			c := rune(pattern[i+1])
			if _, ok := singleCharEscapes[c]; !ok && c != 'p' && c != 'P' {
				t.Fatalf("accepted the escape \\%c in %q", c, pattern)
			}
			i++
		}
		re.MatchString(pattern)
	})
}

// fuzzErrorForms lists the patterns above, so a test can show each is
// reachable (TestFuzzErrorFormsReachable).
var fuzzErrorForms = map[string]*regexp.Regexp{
	"turtleLocated": turtleLocated, "sizeLimit": sizeLimit, "blankNodeShape": blankNodeShape,
	"jsonLocated": jsonLocated, "jsonField": jsonField, "jsonPath": jsonPath, "shapeRule": shapeRule,
	"emptyDocument": emptyDocument, "yamlLine": yamlLine, "yamlLocated": yamlLocated, "yamlConstruct": yamlConstruct,
}

// TestFuzzErrorFormsReachable shows each accepted error form matches an
// error the decoders really return, and that requireNamed refuses an error
// naming nothing, so the forms are neither dead nor so wide they accept
// anything.
func TestFuzzErrorFormsReachable(t *testing.T) {
	samples := map[string]func() error{
		"turtleLocated": func() error { _, err := ParseTurtle(strings.NewReader("@@")); return err },
		"sizeLimit": func() error {
			_, err := ParseTurtle(bytes.NewReader(make([]byte, maxTurtleBytes+1)))
			return err
		},
		"blankNodeShape": func() error {
			_, err := ParseTurtle(strings.NewReader(`[] <http://www.w3.org/ns/shacl#targetClass> <http://ex/C> .`))
			return err
		},
		"jsonLocated": func() error { _, err := DecodeShapesJSON(strings.NewReader(`{"node_shapes": [`)); return err },
		"jsonField":   func() error { _, err := DecodeShapesJSON(strings.NewReader(`{"nodes": []}`)); return err },
		"jsonPath":    func() error { _, err := DecodeShapesJSON(strings.NewReader(`{"node_shapes": null}`)); return err },
		"shapeRule": func() error {
			_, err := DecodeShapesJSON(strings.NewReader(`{"node_shapes": [{"id": "rel"}]}`))
			return err
		},
		"emptyDocument": func() error { _, err := DecodeShapesYAML(strings.NewReader("")); return err },
		"yamlLine":      func() error { _, err := DecodeShapesYAML(strings.NewReader("node_shapes: [\n")); return err },
		"yamlLocated":   func() error { _, err := DecodeShapesYAML(strings.NewReader("a: \"\x01\"\n")); return err },
		"yamlConstruct": func() error { _, err := DecodeShapesYAML(strings.NewReader("node_shapes: *x\n")); return err },
	}
	names := make([]string, 0, len(fuzzErrorForms))
	for name := range fuzzErrorForms {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		sample, ok := samples[name]
		if !ok {
			t.Errorf("%s: no sample error", name)
			continue
		}
		err := sample()
		if err == nil || !fuzzErrorForms[name].MatchString(err.Error()) {
			t.Errorf("%s does not match its sample error %v", name, err)
		}
	}
	for _, msg := range []string{"shacl: shapes: unexpected EOF", "shacl: shapes: yaml: control characters are not allowed",
		"shacl: shapes: yaml: found unexpected end of stream",
		"turtle: input is not valid UTF-8", "shacl: shapes: EOF"} {
		for name, f := range fuzzErrorForms {
			if f.MatchString(msg) {
				t.Errorf("%s accepts %q, which names no construct or location", name, msg)
			}
		}
	}
}

package turtle

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// TestNestingLimit shows the nesting limit rejects the 65th level of blank-node
// property lists, collections, or a mix of both, naming the position of the
// opening bracket, and accepts exactly 64 levels.
func TestNestingLimit(t *testing.T) {
	const limit = 64
	const head = "<http://ex/s> <http://ex/p> "
	blankOpen, blankClose := "[ <http://ex/p> ", " ]"
	listOpen, listClose := "( ", " )"
	// nested returns a one-line document with levels of nesting, and the
	// column of the opening bracket of level limit+1 (0 when absent).
	nested := func(levels int, open func(i int) (string, string)) (string, int) {
		var b strings.Builder
		b.WriteString(head)
		col := 0
		for i := range levels {
			o, _ := open(i)
			if i == limit {
				col = b.Len() + 1
			}
			b.WriteString(o)
		}
		b.WriteString(`"o"`)
		for i := levels - 1; i >= 0; i-- {
			_, c := open(i)
			b.WriteString(c)
		}
		b.WriteString(" .\n")
		return b.String(), col
	}
	kinds := map[string]func(i int) (string, string){
		"blank nodes": func(int) (string, string) { return blankOpen, blankClose },
		"collections": func(int) (string, string) { return listOpen, listClose },
		"mixed": func(i int) (string, string) {
			if i%2 == 0 {
				return blankOpen, blankClose
			}
			return listOpen, listClose
		},
	}
	for name, open := range kinds {
		t.Run(name, func(t *testing.T) {
			src, _ := nested(limit, open)
			if _, err := Parse(strings.NewReader(src)); err != nil {
				t.Fatalf("%d levels: %v", limit, err)
			}
			src, col := nested(limit+1, open)
			_, err := Parse(strings.NewReader(src))
			want := fmt.Sprintf("line 1 col %d: nesting exceeds %d levels", col, limit)
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("%d levels: err = %v, want containing %q", limit+1, err, want)
			}
		})
	}
	t.Run("subject position", func(t *testing.T) {
		src := strings.Repeat("[ <http://ex/p> ", limit+1) + `"o"` + strings.Repeat(" ]", limit+1) + " .\n"
		want := fmt.Sprintf("line 1 col %d: nesting exceeds %d levels", limit*len("[ <http://ex/p> ")+1, limit)
		if _, err := Parse(strings.NewReader(src)); err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("err = %v, want containing %q", err, want)
		}
	})
}

// TestNestingLimitBoundsDeepInput is probe P1's densest input: nesting as deep
// as fits in 1 MiB (174,755 levels unbounded) now stops at the 65th level
// with an error instead of growing the stack.
func TestNestingLimitBoundsDeepInput(t *testing.T) {
	const unit = "[<http://e/p>"
	src := "<http://e/s> <http://e/p> " + strings.Repeat(unit, (1<<20-64)/len(unit))
	_, err := Parse(strings.NewReader(src))
	if err == nil || !strings.Contains(err.Error(), "nesting exceeds 64 levels") {
		t.Fatalf("err = %v, want the nesting limit", err)
	}
}

// TestBaseResolvesRFC3986Examples resolves every example of RFC 3986 §5.4.1
// (normal) and §5.4.2 (abnormal, strict parser) against the RFC's base
// http://a/b/c/d;p?q, and compares with the results the RFC states.
func TestBaseResolvesRFC3986Examples(t *testing.T) {
	const base = "http://a/b/c/d;p?q"
	examples := map[string]string{
		// §5.4.1
		"g:h": "g:h", "g": "http://a/b/c/g", "./g": "http://a/b/c/g", "g/": "http://a/b/c/g/",
		"/g": "http://a/g", "//g": "http://g", "?y": "http://a/b/c/d;p?y", "g?y": "http://a/b/c/g?y",
		"#s": "http://a/b/c/d;p?q#s", "g#s": "http://a/b/c/g#s", "g?y#s": "http://a/b/c/g?y#s",
		";x": "http://a/b/c/;x", "g;x": "http://a/b/c/g;x", "g;x?y#s": "http://a/b/c/g;x?y#s",
		"": "http://a/b/c/d;p?q", ".": "http://a/b/c/", "./": "http://a/b/c/", "..": "http://a/b/",
		"../": "http://a/b/", "../g": "http://a/b/g", "../..": "http://a/", "../../": "http://a/",
		"../../g": "http://a/g",
		// §5.4.2
		"../../../g": "http://a/g", "../../../../g": "http://a/g", "/./g": "http://a/g", "/../g": "http://a/g",
		"g.": "http://a/b/c/g.", ".g": "http://a/b/c/.g", "g..": "http://a/b/c/g..", "..g": "http://a/b/c/..g",
		"./../g": "http://a/b/g", "./g/.": "http://a/b/c/g/", "g/./h": "http://a/b/c/g/h",
		"g/../h": "http://a/b/c/h", "g;x=1/./y": "http://a/b/c/g;x=1/y", "g;x=1/../y": "http://a/b/c/y",
		"g?y/./x": "http://a/b/c/g?y/./x", "g?y/../x": "http://a/b/c/g?y/../x",
		"g#s/./x": "http://a/b/c/g#s/./x", "g#s/../x": "http://a/b/c/g#s/../x", "http:g": "http:g",
	}
	for ref, want := range examples {
		got, err := Parse(strings.NewReader("<"+ref+"> <http://ex/p> <http://ex/o> ."), WithBase(base))
		if err != nil {
			t.Errorf("<%s>: %v", ref, err)
			continue
		}
		if got[0].S.Value != want {
			t.Errorf("<%s> resolved to %q, want %q (RFC 3986 §5.4)", ref, got[0].S.Value, want)
		}
	}
}

// TestBaseApplies covers where the base applies: prefix IRIs, datatypes, and
// @base directives, which resolve against the current base only when the
// caller supplied one.
func TestBaseApplies(t *testing.T) {
	src := `@prefix ex: <vocab#> .
@base <sub/> .
<s> ex:p "1"^^<dt> .
`
	got, err := Parse(strings.NewReader(src), WithBase("http://a/b/doc.ttl"))
	if err != nil {
		t.Fatal(err)
	}
	want := Triple{
		S: iri("http://a/b/sub/s"), P: iri("http://a/b/vocab#p"), O: typed("1", "http://a/b/sub/dt"),
	}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	// Without a caller base, @base does not enable relative IRIs.
	_, err = Parse(strings.NewReader("@base <http://a/> .\n<s> <http://ex/p> <http://ex/o> ."))
	if err == nil || !strings.Contains(err.Error(), "line 2 col 1: unsupported relative IRI <s>") {
		t.Fatalf("no caller base: err = %v, want the relative IRI rejected", err)
	}
	for _, bad := range []string{"", "relative/doc.ttl", "#frag"} {
		_, err := Parse(strings.NewReader("<http://ex/s> <http://ex/p> <http://ex/o> ."), WithBase(bad))
		if err == nil || !strings.Contains(err.Error(), "base") {
			t.Errorf("WithBase(%q): err = %v, want the base refused", bad, err)
		}
	}
}

var errStop = errors.New("stop")

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errStop }

// TestReadErrorIsReturned keeps the reader's I/O error visible.
func TestReadErrorIsReturned(t *testing.T) {
	if _, err := Parse(failingReader{}); !errors.Is(err, errStop) {
		t.Fatalf("err = %v, want the read error", err)
	}
}

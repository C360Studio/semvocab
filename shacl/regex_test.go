package shacl

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/c360studio/semvocab/internal/oracle"
	"github.com/c360studio/semvocab/internal/turtle"
)

// TestPatternSubsetAccepts is design D10's accepted half. Each case states
// what XSD Appendix F and F&O §5.6 say the pattern matches; the expectations
// are written from those texts and probe P4, not from RE2's behaviour.
func TestPatternSubsetAccepts(t *testing.T) {
	for _, tt := range []struct {
		pattern     string
		match, miss []string
	}{
		// XSD's . is [^\n\r]; RE2's . matches \r (probe P4).
		{`^.$`, []string{"a", "é"}, []string{"\n", "\r", ""}},
		{`a.c`, []string{"abc", "xxabcxx"}, []string{"a\rc", "a\nc"}},
		// Unanchored, like fn:matches; ^ and $ are the whole string.
		{`bra`, []string{"abracadabra"}, []string{"BRA"}},
		{`^a.*a$`, []string{"abracadabra"}, []string{"abracadabrax", "a\na"}},
		{`^b`, []string{"bra"}, []string{"abra", "a\nbra"}},
		{`a$`, []string{"bra"}, []string{"bra\n"}},
		{`a|b|`, []string{"a", "b", ""}, nil},
		{`(ab)+c`, []string{"ababc"}, []string{"ac"}},
		{`a{2}`, []string{"aa"}, []string{"a"}},
		{`^a{2,}$`, []string{"aa", "aaaa"}, []string{"a"}},
		{`^a{1,2}$`, []string{"a", "aa"}, []string{"aaa", ""}},
		{`^a{0,1000}$`, []string{"", "a"}, nil},
		{`^a*?b+?c??$`, []string{"bc", "aab", "abbbc"}, []string{"ac"}},
		{`^a{2}?$`, []string{"aa"}, []string{"a"}},
		// Classes, negation, ranges, a dash first or last.
		{`^[a-cx]$`, []string{"b", "x"}, []string{"d", "-"}},
		{`^[^a-c]$`, []string{"d", "\n", "\r"}, []string{"a"}},
		{`^[-a]$`, []string{"-", "a"}, []string{"b"}},
		{`^[a-]$`, []string{"-", "a"}, []string{"b"}},
		{`^[^-]$`, []string{"a"}, []string{"-"}},
		{`^[a^]$`, []string{"^", "a"}, []string{"b"}},
		{`^[\n\t\-\[\]\\]+$`, []string{"\n\t-[]\\"}, []string{"a"}},
		// Single-character escapes, including F&O's \$.
		{`^\.\?\*\+\(\)\{\}\|\^\$\\$`, []string{`.?*+(){}|^$\`}, []string{"x"}},
		{`^\n\r\t$`, []string{"\n\r\t"}, nil},
		// Category escapes from XSD's list of 36 (Unicode 15.0 tables at
		// go1.26.9, probe P4).
		{`^\p{Nd}$`, []string{"5", "٥"}, []string{"a"}},
		{`^\p{L}+$`, []string{"Joh", "é"}, []string{"5"}},
		{`^\P{Lu}$`, []string{"a"}, []string{"A"}},
		{`^[\p{Lu}\p{Nd}]+$`, []string{"A5"}, []string{"a"}},
		{`^[^\p{Ll}]$`, []string{"A"}, []string{"a"}},
		{`^\p{Cn}$`, []string{"\u0378"}, []string{"a"}},
		{``, []string{"", "anything"}, nil},
	} {
		re, err := compilePattern(tt.pattern)
		if err != nil {
			t.Errorf("%q rejected: %v", tt.pattern, err)
			continue
		}
		for _, s := range tt.match {
			if !re.MatchString(s) {
				t.Errorf("%q does not match %q", tt.pattern, s)
			}
		}
		for _, s := range tt.miss {
			if re.MatchString(s) {
				t.Errorf("%q matches %q", tt.pattern, s)
			}
		}
	}
}

// TestPatternSubsetRejects is design D10's rejected half, as probe P4
// corrected it: each construct is named in the error.
func TestPatternSubsetRejects(t *testing.T) {
	for pattern, construct := range map[string]string{
		`\d`: `\d`, `a\D`: `\D`, `\w+`: `\w`, `\W`: `\W`, `x\s`: `\s`, `\S`: `\S`,
		`\i`: `\i`, `\I`: `\I`, `\c`: `\c`, `\C`: `\C`,
		`[a\d]`: `\d`, `[\s]`: `\s`,
		`\p{IsBasicLatin}`: `\p{IsBasicLatin}`, `\p{Cs}`: `\p{Cs}`, `\p{Greek}`: `\p{Greek}`,
		`\P{Latin}`: `\P{Latin}`, `\pL`: `\p`, `\p{L`: `\p{L`,
		`(a)\1`: `\1`, `(?:a)`: `(?`, `(?i)a`: `(?`, `[a-z-[aeiou]]`: `-[`,
		`\x41`: `\x`, `\A`: `\A`, `\z`: `\z`, `\b`: `\b`, `\u0041`: `\u`, `[\1]`: `\1`, `\`: `\`,
		`a{1001}`: `{1001`, `a{3,2}`: `{3,2`, `a{,3}`: `{,`, `a{`: `{`, `a{2`: `{2`,
		`a**`: `*`, `*a`: `*`, `+`: `+`, `a|?`: `?`, `^*`: `*`, `$+`: `+`, `a{2}{3}`: `{`,
		`a}`: `}`, `a]`: `]`, `)`: `)`, `(a`: `(`, `[a`: `[`, `[]`: `[]`, `[a[b]`: `[`,
		`[a-b-c]`: `-`, `[z-a]`: `z-a`, `[a-\p{L}]`: `-\p`,
	} {
		_, err := compilePattern(pattern)
		var pe *patternError
		if !errors.As(err, &pe) {
			t.Errorf("%q: err = %v, want a pattern error", pattern, err)
			continue
		}
		if !strings.Contains(pe.construct, construct) {
			t.Errorf("%q: error %q does not name %q", pattern, err, construct)
		}
		// The escapes XSD defines differently say how, not merely that
		// they are unknown.
		if len(construct) == 2 && strings.ContainsRune("dDwWsSiIcC", rune(construct[1])) &&
			!strings.HasPrefix(pe.reason, "XSD's") {
			t.Errorf("%q: reason %q does not say how XSD differs", pattern, pe.reason)
		}
	}
}

// TestPatternCompileIssues: through Compile, a pattern outside the subset
// fails naming sh:pattern and the construct (the spec's \d scenario), and
// sh:flags fails by name (ruling Q15).
func TestPatternCompileIssues(t *testing.T) {
	_, err := compileTurtle(t, `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:pattern "^\\d+$" ] .`)
	if is := requireIssue(t, err, sh+"pattern", `\d`, "ASCII-only"); is.Path != ex+"p" {
		t.Errorf("issue = %+v, want path ex:p", is)
	}
	_, err = compileTurtle(t, `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:pattern "^a" ; sh:flags "i" ] .`)
	requireIssue(t, err, sh+"flags", reasonFlags)
	p := onlyProperty(t, mustCompile(t, `ex:S sh:targetClass ex:C ; sh:property [ sh:path ex:p ; sh:pattern "^J.h$" ] .`))
	if p.Pattern == nil || !p.Pattern.MatchString("Joh") || p.Pattern.MatchString("J\rh") {
		t.Errorf("compiled pattern = %v", p.Pattern)
	}
}

// TestPatternAgreesWithW3C runs the pattern of the vendored W3C case
// property/pattern-001 over the case's own data: a value matches exactly when
// the case's expected report has no sh:PatternConstraintComponent result for
// it. The flag-less fn:matches examples of F&O 3.1 §5.6.3 agree too.
func TestPatternAgreesWithW3C(t *testing.T) {
	const rel = "property/pattern-001.ttl"
	const base = "http://datashapes.org/sh/tests/core/property/pattern-001.test#"
	data, err := oracle.ReadFixture(oracle.W3CDataShapes, "data-shapes-test-suite/tests/core/"+rel)
	if err != nil {
		t.Fatal(err)
	}
	triples, err := turtle.Parse(bytes.NewReader(data), turtle.WithBase("http://semvocab.test/core/"+rel))
	if err != nil {
		t.Fatal(err)
	}
	var pattern string
	var values, violating []string
	component := map[turtle.Term]bool{}
	for _, tr := range triples {
		switch {
		case tr.P.Value == sh+"pattern":
			pattern = tr.O.Value
		case tr.P.Value == base+"property":
			values = append(values, tr.O.Value)
		case tr.P.Value == sh+"sourceConstraintComponent" && tr.O.Value == sh+"PatternConstraintComponent":
			component[tr.S] = true
		}
	}
	for _, tr := range triples {
		if tr.P.Value == sh+"value" && component[tr.S] {
			violating = append(violating, tr.O.Value)
		}
	}
	if pattern == "" || len(values) != 4 || len(violating) != 2 {
		t.Fatalf("read pattern %q, values %v, violating %v from %s", pattern, values, violating, rel)
	}
	re, err := compilePattern(pattern)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range values {
		if re.MatchString(v) == slices.Contains(violating, v) {
			t.Errorf("%s: %q on %q: match %v, but the expected report lists it %v", rel, pattern, v,
				re.MatchString(v), slices.Contains(violating, v))
		}
	}

	poem := "\nKaum hat dies der Hahn gesehen,\nFängt er auch schon an zu krähen:\nKikeriki! Kikikerikih!!\nTak, tak, tak! - da kommen sie.\n"
	for _, tt := range []struct {
		input, pattern string
		want           bool
	}{
		{"abracadabra", "bra", true},
		{"abracadabra", "^a.*a$", true},
		{"abracadabra", "^bra", false},
		{poem, "Kaum.*krähen", false},
		{poem, "^Kaum.*gesehen,$", false},
	} {
		re, err := compilePattern(tt.pattern)
		if err != nil {
			t.Errorf("%q: %v", tt.pattern, err)
			continue
		}
		if got := re.MatchString(tt.input); got != tt.want {
			t.Errorf("fn:matches(%q, %q) = %v, want %v (F&O 3.1 §5.6.3)", tt.input, tt.pattern, got, tt.want)
		}
	}
}

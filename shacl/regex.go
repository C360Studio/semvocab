package shacl

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// sh:pattern is SPARQL REGEX (SPARQL 1.1 §17.4.3.14), that is fn:matches
// (XPath and XQuery Functions and Operators 3.1 §5.6), whose syntax is XML
// Schema Part 2 Appendix F with the §5.6.1 additions. Go's regexp is RE2, not
// that, so the compiler parses each pattern against its own allowlist (design
// D10, corrected by probe P4) and emits an equivalent RE2 expression. Anything
// whose meaning differs between the two, or that the allowlist does not
// name, is rejected by name.

// xsdCategories are the 36 general-category names XSD Appendix F lists. RE2
// also accepts Cs and script names, which XSD does not list (probe P4).
var xsdCategories = []string{
	"L", "Lu", "Ll", "Lt", "Lm", "Lo", "M", "Mn", "Mc", "Me", "N", "Nd", "Nl", "No",
	"P", "Pc", "Pd", "Ps", "Pe", "Pi", "Pf", "Po", "Z", "Zs", "Zl", "Zp",
	"S", "Sm", "Sc", "Sk", "So", "C", "Cc", "Cf", "Co", "Cn",
}

// singleCharEscapes are the characters XSD's SingleCharEsc may escape, with
// the $ F&O §5.6.1.1 adds, mapped to the character they stand for.
var singleCharEscapes = map[rune]rune{
	'n': '\n', 'r': '\r', 't': '\t', '\\': '\\', '|': '|', '.': '.', '?': '?', '*': '*', '+': '+',
	'(': '(', ')': ')', '{': '{', '}': '}', '-': '-', '[': '[', ']': ']', '^': '^', '$': '$',
}

// rejectedEscapes name the escapes the allowlist refuses, with the reason.
var rejectedEscapes = map[rune]string{
	'd': "XSD's \\d is \\p{Nd}, Unicode-wide; RE2's is ASCII-only",
	'D': "XSD's \\D is the complement of \\p{Nd}; RE2's is the complement of ASCII digits",
	'w': "XSD's \\w excludes only P, Z, and C characters; RE2's is ASCII-only",
	'W': "XSD's \\W differs from RE2's, as \\w does",
	's': "XSD's \\s is [ \\t\\n\\r]; RE2's adds \\f",
	'S': "XSD's \\S differs from RE2's, as \\s does",
	'i': "XSD's \\i follows the XML name rules; RE2 has no equivalent",
	'I': "XSD's \\I follows the XML name rules; RE2 has no equivalent",
	'c': "XSD's \\c follows the XML name rules; RE2 has no equivalent",
	'C': "XSD's \\C follows the XML name rules; RE2 has no equivalent",
}

// maxRepeat is RE2's largest repetition count; XSD sets none.
const maxRepeat = 1000

// maxRegexNesting bounds group nesting, as RE2 bounds its own.
const maxRegexNesting = 1000

// patternError names the construct that put a pattern outside the subset.
type patternError struct {
	construct string // the construct as written, for example `\d`
	offset    int    // byte offset in the pattern
	reason    string
}

func (e *patternError) Error() string {
	return fmt.Sprintf("%s at byte %d: %s", e.construct, e.offset, e.reason)
}

// compilePattern translates an XPath regular expression in the subset to RE2
// and compiles it. The result matches when some substring matches, as
// fn:matches does; ^ and $ anchor to the whole string (F&O §5.6.1.1), which is
// RE2's meaning without the m flag.
func compilePattern(pattern string) (*regexp.Regexp, error) {
	if !utf8.ValidString(pattern) {
		return nil, &patternError{construct: "invalid UTF-8", reason: "the pattern is not valid UTF-8"}
	}
	p := &patternParser{src: pattern}
	if err := p.regExp(0); err != nil {
		return nil, err
	}
	if p.pos < len(p.src) {
		return nil, p.fail(string(p.peek()), "unbalanced ')'")
	}
	re, err := regexp.Compile(p.out.String())
	if err != nil {
		// The emitted expression is RE2-valid by construction except for
		// RE2's size limits, which XSD does not have.
		return nil, &patternError{construct: "pattern", reason: "RE2 refuses the translation: " + err.Error()}
	}
	return re, nil
}

type patternParser struct {
	src string
	pos int
	out strings.Builder
}

func (p *patternParser) peek() rune {
	r, _ := utf8.DecodeRuneInString(p.src[p.pos:])
	return r
}

func (p *patternParser) next() rune {
	r, n := utf8.DecodeRuneInString(p.src[p.pos:])
	p.pos += n
	return r
}

func (p *patternParser) eof() bool { return p.pos >= len(p.src) }

func (p *patternParser) fail(construct, reason string) error {
	return &patternError{construct: construct, offset: p.pos, reason: reason}
}

// regExp ::= branch ( '|' branch )*
func (p *patternParser) regExp(depth int) error {
	for {
		if err := p.branch(depth); err != nil {
			return err
		}
		if p.eof() || p.peek() != '|' {
			return nil
		}
		p.next()
		p.out.WriteByte('|')
	}
}

// branch ::= piece* ; piece ::= atom quantifier?
func (p *patternParser) branch(depth int) error {
	for !p.eof() && p.peek() != '|' && p.peek() != ')' {
		quantifiable, err := p.atom(depth)
		if err != nil {
			return err
		}
		if err := p.quantifier(quantifiable); err != nil {
			return err
		}
	}
	return nil
}

// atom ::= Char | charClass | '(' regExp ')' | '^' | '$'
// It reports whether a quantifier may follow.
func (p *patternParser) atom(depth int) (bool, error) {
	start := p.pos
	r := p.next()
	switch r {
	case '(':
		if !p.eof() && p.peek() == '?' {
			p.pos = start
			return false, p.fail("(?", "groups with (? are outside the subset")
		}
		if depth+1 > maxRegexNesting {
			p.pos = start
			return false, p.fail("(", fmt.Sprintf("groups nest deeper than %d", maxRegexNesting))
		}
		p.out.WriteByte('(')
		if err := p.regExp(depth + 1); err != nil {
			return false, err
		}
		if p.eof() {
			p.pos = start
			return false, p.fail("(", "unclosed group")
		}
		p.next()
		p.out.WriteByte(')')
		return true, nil
	case '^', '$':
		p.out.WriteRune(r)
		return false, nil
	case '.':
		// XSD's . is [^\n\r]; RE2's matches \r (probe P4).
		p.out.WriteString(`[^\n\r]`)
		return true, nil
	case '[':
		p.pos = start
		return true, p.charClassExpr()
	case '\\':
		p.pos = start
		return true, p.escape(false)
	case '?', '*', '+', '{':
		p.pos = start
		return false, p.fail(string(r), "a quantifier needs something to repeat")
	case '}', ']', ')':
		p.pos = start
		return false, p.fail(string(r), "must be escaped")
	}
	p.out.WriteString(regexp.QuoteMeta(string(r)))
	return true, nil
}

// quantifier ::= ( [?*+] | '{' quantity '}' ) '?'?
func (p *patternParser) quantifier(quantifiable bool) error {
	if p.eof() {
		return nil
	}
	start := p.pos
	switch r := p.peek(); r {
	case '?', '*', '+':
		if !quantifiable {
			return p.fail(string(r), "a quantifier on an anchor is outside the subset")
		}
		p.next()
		p.out.WriteRune(r)
	case '{':
		if !quantifiable {
			return p.fail("{", "a quantifier on an anchor is outside the subset")
		}
		p.next()
		q, err := p.quantity(start)
		if err != nil {
			return err
		}
		p.out.WriteString(q)
	default:
		return nil
	}
	if !p.eof() && p.peek() == '?' {
		p.next()
		p.out.WriteByte('?')
	}
	if !p.eof() && strings.ContainsRune("?*+{", p.peek()) {
		return p.fail(string(p.peek()), "a quantifier cannot follow a quantifier")
	}
	return nil
}

// quantity ::= n | n ',' | n ',' m, followed by '}'.
func (p *patternParser) quantity(start int) (string, error) {
	digits := func() (int, bool, error) {
		from := p.pos
		for !p.eof() && p.peek() >= '0' && p.peek() <= '9' {
			p.next()
		}
		if p.pos == from {
			return 0, false, nil
		}
		n, err := strconv.Atoi(p.src[from:p.pos])
		if err != nil || n > maxRepeat {
			return 0, true, &patternError{construct: p.src[start:p.pos], offset: start,
				reason: fmt.Sprintf("a repetition count above %d exceeds RE2's limit", maxRepeat)}
		}
		return n, true, nil
	}
	bad := func() error {
		return &patternError{construct: p.src[start:min(p.pos+1, len(p.src))], offset: start,
			reason: "a quantity is {n}, {n,}, or {n,m}"}
	}
	n, ok, err := digits()
	if err != nil {
		return "", err
	}
	if !ok {
		return "", bad()
	}
	out := "{" + strconv.Itoa(n)
	if !p.eof() && p.peek() == ',' {
		p.next()
		out += ","
		m, ok, err := digits()
		if err != nil {
			return "", err
		}
		if ok {
			if m < n {
				return "", &patternError{construct: p.src[start:p.pos], offset: start,
					reason: "the upper count is below the lower"}
			}
			out += strconv.Itoa(m)
		}
	}
	if p.eof() || p.peek() != '}' {
		return "", bad()
	}
	p.next()
	return out + "}", nil
}

// escape reads one escape at p.pos. Inside a class only single-character
// escapes and category escapes are allowed; a back-reference is never.
func (p *patternParser) escape(inClass bool) error {
	start := p.pos
	p.next() // the backslash
	if p.eof() {
		p.pos = start
		return p.fail(`\`, "a trailing backslash")
	}
	r := p.next()
	written := p.src[start:p.pos]
	if c, ok := singleCharEscapes[r]; ok {
		if inClass {
			p.out.WriteString(classChar(c))
		} else {
			p.out.WriteString(regexp.QuoteMeta(string(c)))
		}
		return nil
	}
	if reason, ok := rejectedEscapes[r]; ok {
		p.pos = start
		return p.fail(written, reason)
	}
	switch {
	case r == 'p' || r == 'P':
		return p.category(start, r)
	case r >= '1' && r <= '9':
		p.pos = start
		return p.fail(written, "back-references are outside the subset")
	}
	p.pos = start
	return p.fail(written, "an escape XSD does not define")
}

// category reads {Name} after \p or \P; Name must be one of the 36 XSD
// general categories. Block escapes (\p{IsBasicLatin}) are refused by name.
func (p *patternParser) category(start int, r rune) error {
	if p.eof() || p.peek() != '{' {
		p.pos = start
		return p.fail(p.src[start:min(start+2, len(p.src))], "a category escape is \\p{Name}")
	}
	end := strings.IndexByte(p.src[p.pos:], '}')
	if end < 0 {
		p.pos = start
		return p.fail(p.src[start:], "unclosed category escape")
	}
	name := p.src[p.pos+1 : p.pos+end]
	p.pos += end + 1
	written := p.src[start:p.pos]
	switch {
	case strings.HasPrefix(name, "Is"):
		p.pos = start
		return p.fail(written, "Unicode block escapes are outside the subset")
	case !slices.Contains(xsdCategories, name):
		p.pos = start
		return p.fail(written, "not one of the 36 general categories XSD lists")
	}
	p.out.WriteString(`\` + string(r) + "{" + name + "}")
	return nil
}

// charClassExpr ::= '[' '^'? posCharGroup ']'. Class subtraction is refused
// by name: RE2 compiles [a-z-[aeiou]] silently as a different class (probe
// P4).
func (p *patternParser) charClassExpr() error {
	open := p.pos
	p.next() // [
	p.out.WriteByte('[')
	if !p.eof() && p.peek() == '^' {
		p.next()
		p.out.WriteByte('^')
	}
	first := true
	for {
		if p.eof() {
			p.pos = open
			return p.fail("[", "unclosed character class")
		}
		r := p.peek()
		switch {
		case r == ']':
			if first {
				return p.fail("[]", "an empty character class")
			}
			p.next()
			p.out.WriteByte(']')
			return nil
		case r == '-':
			p.next()
			switch {
			case !p.eof() && p.peek() == '[':
				p.pos--
				return p.fail("-[", "class subtraction is outside the subset")
			case first || !p.eof() && p.peek() == ']':
				p.out.WriteString(classChar('-'))
			default:
				p.pos--
				return p.fail("-", "a '-' in a class must be first, last, or between a range's ends")
			}
		case r == '[':
			return p.fail("[", "must be escaped inside a class")
		case r == '\\' && p.pos+1 < len(p.src) && (p.src[p.pos+1] == 'p' || p.src[p.pos+1] == 'P'):
			if err := p.escape(true); err != nil {
				return err
			}
		default:
			lo, err := p.charOrEsc()
			if err != nil {
				return err
			}
			if p.pos+1 < len(p.src) && p.peek() == '-' && p.src[p.pos+1] != ']' && p.src[p.pos+1] != '[' {
				rangeStart := p.pos
				p.next() // -
				if p.peek() == '\\' && p.pos+1 < len(p.src) && (p.src[p.pos+1] == 'p' || p.src[p.pos+1] == 'P') {
					return p.fail(p.src[rangeStart:p.pos+2], "a range end must be one character")
				}
				hi, err := p.charOrEsc()
				if err != nil {
					return err
				}
				if hi < lo {
					return &patternError{construct: p.src[rangeStart-utf8.RuneLen(lo) : p.pos], offset: rangeStart,
						reason: "a range whose end is below its start"}
				}
				p.out.WriteString(classChar(lo) + "-" + classChar(hi))
			} else {
				p.out.WriteString(classChar(lo))
			}
		}
		first = false
	}
}

// charOrEsc ::= XmlChar | SingleCharEsc, where XmlChar excludes \ - [ ].
func (p *patternParser) charOrEsc() (rune, error) {
	if p.peek() != '\\' {
		return p.next(), nil
	}
	start := p.pos
	p.next()
	if p.eof() {
		p.pos = start
		return 0, p.fail(`\`, "a trailing backslash")
	}
	r := p.next()
	if c, ok := singleCharEscapes[r]; ok {
		return c, nil
	}
	written := p.src[start:p.pos]
	p.pos = start
	if reason, ok := rejectedEscapes[r]; ok {
		return 0, p.fail(written, reason)
	}
	if r >= '0' && r <= '9' {
		return 0, p.fail(written, "a backslash before a digit is invalid inside a class")
	}
	return 0, p.fail(written, "an escape XSD does not define")
}

// classChar writes one character for use inside an RE2 class.
func classChar(r rune) string {
	return fmt.Sprintf(`\x{%x}`, r)
}

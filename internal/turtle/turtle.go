// Package turtle reads the subset of W3C Turtle
// (https://www.w3.org/TR/turtle/) that SemVocab's vendored ontology and SHACL
// fixtures, SemEngine's serializer output, and author-supplied SHACL shapes
// documents use. It was written for tests (design P4 of phase-1-catalogs) and
// is promoted to production use by the SHACL compiler (design D2 of
// phase-2-shacl-compilation, owner ruling Q12), so SemVocab needs no
// third-party RDF module.
//
// Supported: @prefix and @base directives; absolute IRIREFs; prefixed names
// whose local part uses letters, digits, '_', '-', and inner '.'; the "a"
// keyword; labelled blank nodes (_:label); blank-node property lists [ ... ];
// collections ( ... ); predicate-object lists with ';' and object lists with
// ','; string literals in "..." and """...""" with ECHAR and UCHAR escapes,
// optional language tag or ^^datatype; INTEGER, DECIMAL, DOUBLE, and boolean
// literals; '#' comments.
//
// Limit: nesting of blank-node property lists and collections deeper than 64
// levels fails with the line and column of the opening bracket. The limit
// bounds the reader's recursion: without it, deeply nested input grows the
// goroutine stack until the process dies (probe P1 of
// phase-2-shacl-compilation). The reader has no size limit, because it also
// loads the 1.99 MB CCO v2.2 oracle; shacl.ParseTurtle caps author-supplied
// shapes documents at 1 MiB (ruling Q25).
//
// Relative IRIs are rejected unless the caller passes WithBase, which resolves
// them by RFC 3986 §5.2. Only the loader for the W3C SHACL test suite does;
// shapes documents must use absolute IRIs. Without WithBase an @base directive
// is checked but changes nothing.
//
// Everything else fails loudly with its line and column: SPARQL-style PREFIX
// and BASE, single-quoted strings, prefixed-name escapes ('\' and '%' in a
// local name), and any other unexpected character. A failure returns an
// error; the reader never skips input.
package turtle

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Kind distinguishes the RDF term kinds the reader produces.
type Kind int

// Term kinds.
const (
	IRI Kind = iota + 1
	Blank
	Literal
)

// Well-known IRIs the reader emits itself.
const (
	rdfNS     = "http://www.w3.org/1999/02/22-rdf-syntax-ns#"
	xsdNS     = "http://www.w3.org/2001/XMLSchema#"
	rdfType   = rdfNS + "type"
	rdfFirst  = rdfNS + "first"
	rdfRest   = rdfNS + "rest"
	rdfNil    = rdfNS + "nil"
	xsdString = xsdNS + "string"
)

// maxNesting bounds blank-node property lists and collections (design D2 of
// phase-2-shacl-compilation).
const maxNesting = 64

// Term is one RDF term. Value is the IRI, the blank-node label, or the literal
// lexical form. Datatype is set on every literal (xsd:string when untyped,
// rdf:langString when Lang is set).
type Term struct {
	Kind     Kind
	Value    string
	Lang     string
	Datatype string
}

// Triple is one parsed statement.
type Triple struct {
	S, P, O Term
}

// Option configures Parse.
type Option func(*config)

type config struct {
	base    string
	baseErr error
}

// WithBase resolves relative IRIs against base, which must be an absolute IRI
// with no fragment. An @base directive in the document then replaces it,
// itself resolved against the current base.
func WithBase(base string) Option {
	return func(c *config) {
		if !isAbsolute(base) || strings.ContainsRune(base, '#') {
			c.baseErr = fmt.Errorf("turtle: base %q is not an absolute IRI without a fragment", base)
			return
		}
		c.base = base
	}
}

// Parse reads a Turtle document and returns its triples in document order,
// repeats included.
func Parse(r io.Reader, opts ...Option) ([]Triple, error) {
	var cfg config
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.baseErr != nil {
		return nil, cfg.baseErr
	}
	src, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("turtle: read: %w", err)
	}
	if !utf8.Valid(src) {
		return nil, errors.New("turtle: input is not valid UTF-8")
	}
	p := &parser{src: string(src), prefixes: map[string]string{}, base: cfg.base}
	for {
		p.skipWS()
		if p.eof() {
			return p.out, nil
		}
		if err := p.statement(); err != nil {
			return nil, err
		}
	}
}

type parser struct {
	src      string
	pos      int
	prefixes map[string]string
	bnodes   int
	depth    int
	base     string // empty when relative IRIs are rejected
	out      []Triple
}

// enter counts one level of nesting at the opening bracket under p.pos and
// fails past maxNesting. The caller defers p.leave.
func (p *parser) enter() error {
	if p.depth == maxNesting {
		return p.errorf("nesting exceeds %d levels", maxNesting)
	}
	p.depth++
	return nil
}

func (p *parser) leave() { p.depth-- }

func (p *parser) eof() bool { return p.pos >= len(p.src) }

func (p *parser) peek() byte {
	if p.eof() {
		return 0
	}
	return p.src[p.pos]
}

func (p *parser) errorf(format string, args ...any) error {
	line := 1 + strings.Count(p.src[:min(p.pos, len(p.src))], "\n")
	col := p.pos - strings.LastIndexByte(p.src[:min(p.pos, len(p.src))], '\n')
	return fmt.Errorf("turtle: line %d col %d: %s", line, col, fmt.Sprintf(format, args...))
}

func (p *parser) skipWS() {
	for !p.eof() {
		switch c := p.peek(); {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			p.pos++
		case c == '#':
			for !p.eof() && p.peek() != '\n' {
				p.pos++
			}
		default:
			return
		}
	}
}

func (p *parser) expect(c byte) error {
	p.skipWS()
	if p.peek() != c {
		return p.errorf("expected %q, found %s", c, p.found())
	}
	p.pos++
	return nil
}

func (p *parser) found() string {
	if p.eof() {
		return "end of input"
	}
	r, _ := utf8.DecodeRuneInString(p.src[p.pos:])
	return strconv.QuoteRune(r)
}

func (p *parser) statement() error {
	if p.peek() == '@' {
		return p.directive()
	}
	if p.hasKeyword("PREFIX") || p.hasKeyword("BASE") {
		return p.errorf("unsupported SPARQL-style directive")
	}
	if err := p.triples(); err != nil {
		return err
	}
	return p.expect('.')
}

func (p *parser) hasKeyword(kw string) bool {
	if !strings.HasPrefix(strings.ToUpper(p.src[p.pos:min(p.pos+len(kw), len(p.src))]), kw) {
		return false
	}
	next := p.pos + len(kw)
	return next < len(p.src) && (p.src[next] == ' ' || p.src[next] == '\t')
}

func (p *parser) directive() error {
	switch {
	case strings.HasPrefix(p.src[p.pos:], "@prefix"):
		p.pos += len("@prefix")
		p.skipWS()
		start := p.pos
		for !p.eof() && p.peek() != ':' && isNameChar(p.peek()) {
			p.pos++
		}
		if p.peek() != ':' {
			return p.errorf("expected prefix name ending in ':', found %s", p.found())
		}
		name := p.src[start:p.pos]
		p.pos++
		p.skipWS()
		iri, err := p.iriRef()
		if err != nil {
			return err
		}
		p.prefixes[name] = iri
	case strings.HasPrefix(p.src[p.pos:], "@base"):
		p.pos += len("@base")
		p.skipWS()
		// Without a caller base the directive is read for validity only:
		// relative IRIs stay rejected.
		iri, err := p.iriRef()
		if err != nil {
			return err
		}
		if p.base != "" {
			p.base = iri
		}
	default:
		return p.errorf("unsupported directive")
	}
	return p.expect('.')
}

func (p *parser) triples() error {
	p.skipWS()
	if p.peek() == '[' {
		subj, err := p.blankNodePropertyList()
		if err != nil {
			return err
		}
		p.skipWS()
		if p.peek() == '.' {
			return nil
		}
		return p.predicateObjectList(subj)
	}
	subj, err := p.subject()
	if err != nil {
		return err
	}
	return p.predicateObjectList(subj)
}

func (p *parser) subject() (Term, error) {
	p.skipWS()
	switch c := p.peek(); {
	case c == '<':
		v, err := p.iriRef()
		return Term{Kind: IRI, Value: v}, err
	case c == '_':
		return p.blankLabel()
	case c == '(':
		return p.collection()
	default:
		v, err := p.prefixedName()
		return Term{Kind: IRI, Value: v}, err
	}
}

func (p *parser) predicateObjectList(subj Term) error {
	for {
		p.skipWS()
		pred, err := p.verb()
		if err != nil {
			return err
		}
		if err := p.objectList(subj, pred); err != nil {
			return err
		}
		p.skipWS()
		if p.peek() != ';' {
			return nil
		}
		for p.peek() == ';' {
			p.pos++
			p.skipWS()
		}
		// A trailing ';' before '.' or ']' is legal Turtle.
		if c := p.peek(); c == '.' || c == ']' {
			return nil
		}
	}
}

func (p *parser) verb() (Term, error) {
	if p.peek() == 'a' && p.pos+1 < len(p.src) && isSpace(p.src[p.pos+1]) {
		p.pos++
		return Term{Kind: IRI, Value: rdfType}, nil
	}
	if p.peek() == '<' {
		v, err := p.iriRef()
		return Term{Kind: IRI, Value: v}, err
	}
	v, err := p.prefixedName()
	return Term{Kind: IRI, Value: v}, err
}

func (p *parser) objectList(subj, pred Term) error {
	for {
		obj, err := p.object()
		if err != nil {
			return err
		}
		p.out = append(p.out, Triple{S: subj, P: pred, O: obj})
		p.skipWS()
		if p.peek() != ',' {
			return nil
		}
		p.pos++
	}
}

func (p *parser) object() (Term, error) {
	p.skipWS()
	switch c := p.peek(); {
	case c == '<':
		v, err := p.iriRef()
		return Term{Kind: IRI, Value: v}, err
	case c == '_':
		return p.blankLabel()
	case c == '[':
		return p.blankNodePropertyList()
	case c == '(':
		return p.collection()
	case c == '"':
		return p.literal()
	case c == '\'':
		return Term{}, p.errorf("unsupported single-quoted string literal")
	case c == '+' || c == '-' || c == '.' || (c >= '0' && c <= '9'):
		return p.numeric()
	case strings.HasPrefix(p.src[p.pos:], "true") && p.boolEnds(4):
		p.pos += 4
		return Term{Kind: Literal, Value: "true", Datatype: xsdNS + "boolean"}, nil
	case strings.HasPrefix(p.src[p.pos:], "false") && p.boolEnds(5):
		p.pos += 5
		return Term{Kind: Literal, Value: "false", Datatype: xsdNS + "boolean"}, nil
	default:
		v, err := p.prefixedName()
		return Term{Kind: IRI, Value: v}, err
	}
}

func (p *parser) boolEnds(n int) bool {
	i := p.pos + n
	return i >= len(p.src) || !isNameChar(p.src[i]) && p.src[i] != ':'
}

func (p *parser) newBlank() Term {
	p.bnodes++
	return Term{Kind: Blank, Value: fmt.Sprintf("_:b%d", p.bnodes)}
}

func (p *parser) blankLabel() (Term, error) {
	if !strings.HasPrefix(p.src[p.pos:], "_:") {
		return Term{}, p.errorf("expected blank node label, found %s", p.found())
	}
	p.pos += 2
	start := p.pos
	for !p.eof() && isNameChar(p.peek()) {
		p.pos++
	}
	if p.pos == start {
		return Term{}, p.errorf("empty blank node label")
	}
	// Labels share the namespace "_:L" so they cannot collide with the "_:bN"
	// labels the reader allocates for [ ] and ( ).
	return Term{Kind: Blank, Value: "_:L" + p.src[start:p.pos]}, nil
}

func (p *parser) blankNodePropertyList() (Term, error) {
	if err := p.enter(); err != nil {
		return Term{}, err
	}
	defer p.leave()
	p.pos++ // '['
	node := p.newBlank()
	p.skipWS()
	if p.peek() == ']' {
		p.pos++
		return node, nil
	}
	if err := p.predicateObjectList(node); err != nil {
		return Term{}, err
	}
	return node, p.expect(']')
}

func (p *parser) collection() (Term, error) {
	if err := p.enter(); err != nil {
		return Term{}, err
	}
	defer p.leave()
	p.pos++ // '('
	var items []Term
	for {
		p.skipWS()
		if p.eof() {
			return Term{}, p.errorf("unterminated collection")
		}
		if p.peek() == ')' {
			p.pos++
			break
		}
		obj, err := p.object()
		if err != nil {
			return Term{}, err
		}
		items = append(items, obj)
	}
	head := Term{Kind: IRI, Value: rdfNil}
	for i := len(items) - 1; i >= 0; i-- {
		node := p.newBlank()
		p.out = append(p.out,
			Triple{S: node, P: Term{Kind: IRI, Value: rdfFirst}, O: items[i]},
			Triple{S: node, P: Term{Kind: IRI, Value: rdfRest}, O: head})
		head = node
	}
	return head, nil
}

func (p *parser) iriRef() (string, error) {
	if p.peek() != '<' {
		return "", p.errorf("expected IRI, found %s", p.found())
	}
	end := strings.IndexByte(p.src[p.pos:], '>')
	if end < 0 {
		return "", p.errorf("unterminated IRI")
	}
	iri := p.src[p.pos+1 : p.pos+end]
	for _, r := range iri {
		if r <= 0x20 || strings.ContainsRune("<\"{}|^`\\", r) {
			return "", p.errorf("unsupported character %q in IRI", r)
		}
	}
	if !isAbsolute(iri) {
		if p.base == "" {
			return "", p.errorf("unsupported relative IRI <%s>", iri)
		}
		iri = resolve(p.base, iri)
	}
	p.pos += end + 1
	return iri, nil
}

func (p *parser) prefixedName() (string, error) {
	start := p.pos
	for !p.eof() && p.peek() != ':' && isNameChar(p.peek()) {
		p.pos++
	}
	if p.peek() != ':' {
		p.pos = start
		return "", p.errorf("unsupported term starting with %s", p.found())
	}
	prefix := p.src[start:p.pos]
	ns, ok := p.prefixes[prefix]
	if !ok {
		p.pos = start
		return "", p.errorf("undeclared prefix %q", prefix)
	}
	p.pos++
	localStart := p.pos
	for !p.eof() {
		c := p.peek()
		if c == '\\' || c == '%' {
			return "", p.errorf("unsupported escape in prefixed name local part")
		}
		if isNameChar(c) || (c == '.' && p.pos+1 < len(p.src) && isNameChar(p.src[p.pos+1])) {
			p.pos++
			continue
		}
		break
	}
	return ns + p.src[localStart:p.pos], nil
}

func (p *parser) literal() (Term, error) {
	var lex string
	var err error
	if strings.HasPrefix(p.src[p.pos:], `"""`) {
		lex, err = p.longString()
	} else {
		lex, err = p.shortString()
	}
	if err != nil {
		return Term{}, err
	}
	t := Term{Kind: Literal, Value: lex, Datatype: xsdString}
	switch {
	case p.peek() == '@':
		p.pos++
		start := p.pos
		for !p.eof() && (isAlpha(p.peek()) || isDigit(p.peek()) || p.peek() == '-') {
			p.pos++
		}
		if p.pos == start {
			return Term{}, p.errorf("empty language tag")
		}
		t.Lang = strings.ToLower(p.src[start:p.pos])
		t.Datatype = rdfNS + "langString"
	case strings.HasPrefix(p.src[p.pos:], "^^"):
		p.pos += 2
		var dt string
		if p.peek() == '<' {
			dt, err = p.iriRef()
		} else {
			dt, err = p.prefixedName()
		}
		if err != nil {
			return Term{}, err
		}
		t.Datatype = dt
	}
	return t, nil
}

func (p *parser) shortString() (string, error) {
	p.pos++ // opening quote
	var b strings.Builder
	for {
		if p.eof() {
			return "", p.errorf("unterminated string")
		}
		c := p.peek()
		switch c {
		case '"':
			p.pos++
			return b.String(), nil
		case '\n', '\r':
			return "", p.errorf("newline in short string")
		case '\\':
			if err := p.escape(&b); err != nil {
				return "", err
			}
		default:
			b.WriteByte(c)
			p.pos++
		}
	}
}

func (p *parser) longString() (string, error) {
	p.pos += 3
	var b strings.Builder
	for {
		if p.eof() {
			return "", p.errorf("unterminated long string")
		}
		if strings.HasPrefix(p.src[p.pos:], `"""`) {
			p.pos += 3
			// Up to two further quotes belong to the string content.
			for p.peek() == '"' {
				b.WriteByte('"')
				p.pos++
			}
			return b.String(), nil
		}
		if p.peek() == '\\' {
			if err := p.escape(&b); err != nil {
				return "", err
			}
			continue
		}
		b.WriteByte(p.peek())
		p.pos++
	}
}

func (p *parser) escape(b *strings.Builder) error {
	if p.pos+1 >= len(p.src) {
		return p.errorf("unterminated escape")
	}
	c := p.src[p.pos+1]
	p.pos += 2
	switch c {
	case 't':
		b.WriteByte('\t')
	case 'b':
		b.WriteByte('\b')
	case 'n':
		b.WriteByte('\n')
	case 'r':
		b.WriteByte('\r')
	case 'f':
		b.WriteByte('\f')
	case '"', '\'', '\\':
		b.WriteByte(c)
	case 'u', 'U':
		n := 4
		if c == 'U' {
			n = 8
		}
		if p.pos+n > len(p.src) {
			return p.errorf("short \\%c escape", c)
		}
		v, err := strconv.ParseUint(p.src[p.pos:p.pos+n], 16, 32)
		if err != nil || !utf8.ValidRune(rune(v)) {
			return p.errorf("invalid \\%c escape", c)
		}
		b.WriteRune(rune(v))
		p.pos += n
	default:
		return p.errorf("unsupported string escape \\%c", c)
	}
	return nil
}

// numeric reads INTEGER, DECIMAL, or DOUBLE (Turtle §6.5 productions [19]-[21]).
func (p *parser) numeric() (Term, error) {
	start := p.pos
	if c := p.peek(); c == '+' || c == '-' {
		p.pos++
	}
	digits := func() int {
		s := p.pos
		for !p.eof() && isDigit(p.peek()) {
			p.pos++
		}
		return p.pos - s
	}
	intDigits := digits()
	dt := xsdNS + "integer"
	fracDigits := 0
	if p.peek() == '.' && p.pos+1 < len(p.src) && isDigit(p.src[p.pos+1]) {
		p.pos++
		fracDigits = digits()
		dt = xsdNS + "decimal"
	}
	if intDigits == 0 && fracDigits == 0 {
		return Term{}, p.errorf("malformed numeric literal")
	}
	if c := p.peek(); c == 'e' || c == 'E' {
		p.pos++
		if c := p.peek(); c == '+' || c == '-' {
			p.pos++
		}
		if digits() == 0 {
			return Term{}, p.errorf("malformed exponent")
		}
		dt = xsdNS + "double"
	}
	return Term{Kind: Literal, Value: p.src[start:p.pos], Datatype: dt}, nil
}

func isSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }
func isAlpha(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }
func isDigit(c byte) bool { return c >= '0' && c <= '9' }

// isNameChar covers the ASCII subset of PN_CHARS used by the oracle files.
// Non-ASCII name characters are deliberately unsupported.
func isNameChar(c byte) bool { return isAlpha(c) || isDigit(c) || c == '_' || c == '-' }

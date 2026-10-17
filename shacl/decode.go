package shacl

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

// The JSON and YAML shape form. Field names are SemVocab's own (design D3).
type wireShapeSet struct {
	NodeShapes []wireNodeShape `json:"node_shapes" yaml:"node_shapes"`
}

type wireNodeShape struct {
	ID                string              `json:"id" yaml:"id"`
	TargetClasses     []string            `json:"target_classes" yaml:"target_classes"`
	Closed            bool                `json:"closed" yaml:"closed"`
	IgnoredProperties []string            `json:"ignored_properties" yaml:"ignored_properties"`
	Severity          string              `json:"severity" yaml:"severity"`
	Messages          []string            `json:"messages" yaml:"messages"`
	Properties        []wirePropertyShape `json:"properties" yaml:"properties"`
}

type wirePropertyShape struct {
	ID           string    `json:"id" yaml:"id"`
	Path         string    `json:"path" yaml:"path"`
	MinCount     int       `json:"min_count" yaml:"min_count"`
	MaxCount     *int      `json:"max_count" yaml:"max_count"`
	Datatype     string    `json:"datatype" yaml:"datatype"`
	MinInclusive *string   `json:"min_inclusive" yaml:"min_inclusive"`
	MaxInclusive *string   `json:"max_inclusive" yaml:"max_inclusive"`
	MinExclusive *string   `json:"min_exclusive" yaml:"min_exclusive"`
	MaxExclusive *string   `json:"max_exclusive" yaml:"max_exclusive"`
	Pattern      *string   `json:"pattern" yaml:"pattern"`
	In           *[]string `json:"in" yaml:"in"`
	Severity     string    `json:"severity" yaml:"severity"`
	Messages     []string  `json:"messages" yaml:"messages"`
}

// errEmpty is returned for a YAML document with no value.
var errEmpty = errors.New("no document")

// nonStringKeys are the YAML keys whose values are not strings. Every other
// scalar in a shapes or mapping document must carry the !!str tag (ruling
// Q28), so yaml.v3 never coerces a number, boolean, or null into a string.
// The struct decode types these keys' values; a null there decodes to the
// field's zero value, the same as leaving the field out.
var nonStringKeys = map[string]bool{"min_count": true, "max_count": true, "closed": true}

// DecodeShapesJSON decodes SemVocab's JSON shape form. It refuses unknown
// fields, a trailing value, any null, a repeated object key, a numeric bound
// or other non-string where a string belongs, relative IRIs, a blank-node
// node shape, a property shape with no path, and a node shape listed twice.
func DecodeShapesJSON(r io.Reader) (ShapeSet, error) {
	var w wireShapeSet
	if err := decodeJSON(r, &w); err != nil {
		return ShapeSet{}, fmt.Errorf("shacl: shapes: %w", err)
	}
	return w.build()
}

// DecodeShapesYAML decodes SemVocab's YAML shape form. It applies the same
// rules as DecodeShapesJSON except for null: a null is refused wherever a
// string belongs, as is any scalar not tagged !!str, such as an unquoted bound
// (ruling Q28), but a null on min_count, max_count, or closed decodes to the
// field's zero value, the same as leaving the field out. It also refuses a
// trailing document and a duplicate key.
func DecodeShapesYAML(r io.Reader) (ShapeSet, error) {
	var w wireShapeSet
	if err := decodeYAML(r, &w); err != nil {
		return ShapeSet{}, fmt.Errorf("shacl: shapes: %w", err)
	}
	return w.build()
}

// DecodeMappingJSON decodes a mapping document, refusing unknown fields, a
// trailing value, any null, a repeated object key, and a non-string where a
// string belongs. Compile checks the mapping's rules (design D6).
func DecodeMappingJSON(r io.Reader) (Mapping, error) {
	var m Mapping
	if err := decodeJSON(r, &m); err != nil {
		return Mapping{}, fmt.Errorf("shacl: mapping: %w", err)
	}
	return m, nil
}

// DecodeMappingYAML decodes a mapping document with the rules of
// DecodeMappingJSON, and also refuses a trailing document, a duplicate key,
// and any scalar not tagged !!str (ruling Q28).
func DecodeMappingYAML(r io.Reader) (Mapping, error) {
	var m Mapping
	if err := decodeYAML(r, &m); err != nil {
		return Mapping{}, fmt.Errorf("shacl: mapping: %w", err)
	}
	return m, nil
}

func decodeJSON(r io.Reader, out any) error {
	data, err := readDocument(r)
	if err != nil {
		return err
	}
	if err := checkJSONText(data); err != nil {
		return err
	}
	// encoding/json decodes null into a string as "" (in a list) or leaves
	// the field unset, and merges a repeated object key into the members it
	// already decoded (encoding/json/decode.go:50-52 at go1.26.9). So the
	// first value's token stream is checked before the struct decode: every
	// null and every repeated key is refused. No field of either JSON form
	// takes null. A syntax error is left to the struct decode to report.
	if err := scanJSON(json.NewDecoder(bytes.NewReader(data)), "$"); err != nil && !errors.Is(err, errJSONSyntax) {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return locateJSONError(err, data)
	}
	end := dec.InputOffset()
	var extra json.RawMessage
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		rest := data[end:]
		return fmt.Errorf("byte offset %d: trailing data after the document",
			end+int64(len(rest)-len(bytes.TrimLeft(rest, " \t\r\n"))))
	}
	return nil
}

// locateJSONError adds the byte offset to the encoding/json errors that do
// not name a field: a syntax error, a value of the wrong type (whose field
// is named when there is one), and input that ends early. Empty input is no
// document, as in YAML.
func locateJSONError(err error, data []byte) error {
	var syntax *json.SyntaxError
	var typ *json.UnmarshalTypeError
	switch {
	case errors.Is(err, io.EOF):
		return errEmpty
	case errors.Is(err, io.ErrUnexpectedEOF):
		return fmt.Errorf("byte offset %d: the input ends inside the document", len(data))
	case errors.As(err, &syntax):
		return fmt.Errorf("byte offset %d: %w", syntax.Offset, err)
	case errors.As(err, &typ):
		return fmt.Errorf("byte offset %d: %w", typ.Offset, err)
	}
	return err
}

// checkJSONText refuses the text encoding/json would replace with U+FFFD
// without error: invalid UTF-8, and a \u escape naming a lone UTF-16
// surrogate (encoding/json/decode.go:98-100 at go1.26.9). The YAML and
// Turtle readers refuse the same input (issue #18). A backslash outside a
// string is a syntax error either way, so escapes are found without tracking
// strings.
func checkJSONText(data []byte) error {
	for i := 0; i < len(data); {
		r, n := utf8.DecodeRune(data[i:])
		if r == utf8.RuneError && n == 1 {
			return fmt.Errorf("byte offset %d: the input is not valid UTF-8", i)
		}
		if r != '\\' {
			i += n
			continue
		}
		hi, ok := jsonEscapedUnit(data[i:])
		switch {
		case !ok && i+1 < len(data) && data[i+1] == '\\':
			i += 2 // an escaped backslash, which a following u cannot extend
		case !ok:
			i++ // another escape, or a syntax error the struct decode reports
		case hi >= 0xDC00 && hi <= 0xDFFF:
			return fmt.Errorf("byte offset %d: \\u%04x is a lone UTF-16 surrogate", i, hi)
		case hi >= 0xD800 && hi <= 0xDBFF:
			if lo, ok := jsonEscapedUnit(data[i+6:]); !ok || lo < 0xDC00 || lo > 0xDFFF {
				return fmt.Errorf("byte offset %d: \\u%04x is a lone UTF-16 surrogate", i, hi)
			}
			i += 12
		default:
			i += 6
		}
	}
	return nil
}

// jsonEscapedUnit reads the UTF-16 code unit of a \uXXXX escape at the start
// of b.
func jsonEscapedUnit(b []byte) (rune, bool) {
	if len(b) < 6 || b[0] != '\\' || b[1] != 'u' {
		return 0, false
	}
	var u rune
	for _, c := range b[2:6] {
		switch {
		case '0' <= c && c <= '9':
			u = u<<4 | rune(c-'0')
		case 'a' <= c && c <= 'f':
			u = u<<4 | rune(c-'a'+10)
		case 'A' <= c && c <= 'F':
			u = u<<4 | rune(c-'A'+10)
		default:
			return 0, false
		}
	}
	return u, true
}

// errJSONSyntax marks a token-stream error in scanJSON; the struct decode
// reports it with its own message.
var errJSONSyntax = errors.New("json syntax")

// scanJSON reads one JSON value from dec's token stream and fails on a null
// or a repeated object key, naming its JSON path.
func scanJSON(dec *json.Decoder, path string) error {
	tok, err := dec.Token()
	if err != nil {
		return errJSONSyntax
	}
	switch tok {
	case nil:
		return fmt.Errorf("%s is null; omit the field or give a value", path)
	case json.Delim('['):
		for i := 0; dec.More(); i++ {
			if err := scanJSON(dec, fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	case json.Delim('{'):
		seen := map[string]bool{}
		for dec.More() {
			kt, err := dec.Token()
			if err != nil {
				return errJSONSyntax
			}
			key, _ := kt.(string)
			if seen[key] {
				return fmt.Errorf("%s: key %q repeated", path, key)
			}
			seen[key] = true
			if err := scanJSON(dec, path+"."+key); err != nil {
				return err
			}
		}
	default:
		return nil
	}
	if _, err := dec.Token(); err != nil { // the closing delimiter
		return errJSONSyntax
	}
	return nil
}

func decodeYAML(r io.Reader, out any) error {
	data, err := readDocument(r)
	if err != nil {
		return err
	}
	// Strict struct decode first, so an unknown field is named as such.
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(out); err != nil {
		if errors.Is(err, io.EOF) {
			return errEmpty
		}
		return locateYAMLError(err, data)
	}
	var extra yaml.Node
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		if err != nil {
			return locateYAMLError(err, data)
		}
		return fmt.Errorf("line %d: trailing document", extra.Line)
	}
	// Then the tags: yaml.v3 decodes an unquoted 5 into a string field
	// without error (probe P3).
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return locateYAMLError(err, data)
	}
	return checkStringTags(&doc, "")
}

// locateYAMLError places the yaml.v3 errors that carry no line
// (gopkg.in/yaml.v3 v3.0.1). Its reader refuses invalid UTF-8 and characters
// YAML 1.2 does not allow (§5.1, c-printable) with no position
// (readerc.go:214, :379); they get the byte offset of the first such byte.
// Its parser and scanner drop the line when the error's marks are on the
// first line, which it counts from 0 (decode.go:112-137); those get line 1.
// The decoder's own errors name their construct (an anchor, an alias, a
// merge key, a tag) and pass through.
func locateYAMLError(err error, data []byte) error {
	msg := err.Error()
	if yamlLineNumber.MatchString(msg) || yamlDecoderError.MatchString(msg) {
		return err
	}
	for i := 0; i < len(data); {
		r, n := utf8.DecodeRune(data[i:])
		if r == utf8.RuneError && n == 1 || !yamlPrintable(r) {
			return fmt.Errorf("byte offset %d: %w", i, err)
		}
		i += n
	}
	return fmt.Errorf("line 1: %w", err)
}

// yamlLineNumber matches the line yaml.v3 puts in an error ("line 3: ").
var yamlLineNumber = regexp.MustCompile(`\bline \d+: `)

// yamlDecoderError matches the errors yaml.v3's decoder raises without a
// line (decode.go:87, :218, :490, :519, :536, :576, :738, :845, :956).
var yamlDecoderError = regexp.MustCompile(`^yaml: (attempted to go past the end of stream|unknown anchor '.*' referenced|` +
	`document contains excessive aliasing|cannot decode node with unknown kind|anchor '.*' value contains itself|` +
	`!!binary value contains invalid base64 data|invalid array:|invalid map key:|map merge requires)`)

// yamlPrintable is YAML 1.2's c-printable, the set yaml.v3's reader checks.
func yamlPrintable(r rune) bool {
	return r == 0x09 || r == 0x0A || r == 0x0D || r >= 0x20 && r <= 0x7E || r == 0x85 ||
		r >= 0xA0 && r <= 0xD7FF || r >= 0xE000 && r <= 0xFFFD || r >= 0x10000 && r <= 0x10FFFF
}

// checkStringTags walks a YAML document and fails on a scalar that is not
// tagged !!str, except the values of nonStringKeys, which the struct decode
// types. key is the mapping key the node is the value of (inherited by
// sequence items).
func checkStringTags(n *yaml.Node, key string) error {
	switch n.Kind {
	case yaml.DocumentNode, yaml.SequenceNode:
		for _, c := range n.Content {
			if err := checkStringTags(c, key); err != nil {
				return err
			}
		}
	case yaml.MappingNode:
		for i := 0; i+1 < len(n.Content); i += 2 {
			k := n.Content[i]
			if k.Kind != yaml.ScalarNode || k.ShortTag() != "!!str" {
				return fmt.Errorf("line %d: mapping keys must be strings", k.Line)
			}
			if err := checkStringTags(n.Content[i+1], k.Value); err != nil {
				return err
			}
		}
	case yaml.AliasNode:
		return checkStringTags(n.Alias, key)
	case yaml.ScalarNode:
		if !nonStringKeys[key] && n.ShortTag() != "!!str" {
			return fmt.Errorf("line %d: %s must be a string (YAML tag %s); quote it", n.Line, key, n.ShortTag())
		}
	}
	return nil
}

func (w wireShapeSet) build() (ShapeSet, error) {
	var set ShapeSet
	seen := map[string]bool{}
	for _, wn := range w.NodeShapes {
		if strings.HasPrefix(wn.ID, "_:") {
			return ShapeSet{}, fmt.Errorf("shacl: shapes: blank-node node shape %q: node shapes must be IRIs (design D3)", wn.ID)
		}
		if err := checkIRIs("node shape id", wn.ID); err != nil {
			return ShapeSet{}, err
		}
		if seen[wn.ID] {
			return ShapeSet{}, fmt.Errorf("shacl: shapes: node shape %s listed twice", wn.ID)
		}
		seen[wn.ID] = true
		if err := checkIRIs("node shape "+wn.ID, append(slices.Clone(wn.TargetClasses), wn.IgnoredProperties...)...); err != nil {
			return ShapeSet{}, err
		}
		if err := checkOptionalIRI("node shape "+wn.ID+" severity", wn.Severity); err != nil {
			return ShapeSet{}, err
		}
		ns := NodeShape{
			ID: wn.ID, TargetClasses: wn.TargetClasses, Closed: wn.Closed, IgnoredProperties: wn.IgnoredProperties,
			Severity: wn.Severity, Messages: wn.Messages,
		}
		for _, wp := range wn.Properties {
			where := "node shape " + wn.ID + " property " + wp.Path
			if wp.Path == "" {
				return ShapeSet{}, fmt.Errorf("shacl: shapes: node shape %s has a property shape with no path", wn.ID)
			}
			if err := checkIRIs(where, wp.Path); err != nil {
				return ShapeSet{}, err
			}
			for _, opt := range []string{wp.ID, wp.Datatype, wp.Severity} {
				if err := checkOptionalIRI(where, opt); err != nil {
					return ShapeSet{}, err
				}
			}
			// A bound's datatype is its property's (design D8); Compile
			// rejects a bound on a property with no sh:datatype.
			lit := func(s *string) *Literal {
				if s == nil {
					return nil
				}
				return &Literal{Lexical: *s, Datatype: wp.Datatype}
			}
			ns.Properties = append(ns.Properties, PropertyShape{
				ID: wp.ID, Path: wp.Path, MinCount: wp.MinCount, MaxCount: wp.MaxCount, Datatype: wp.Datatype,
				MinInclusive: lit(wp.MinInclusive), MaxInclusive: lit(wp.MaxInclusive),
				MinExclusive: lit(wp.MinExclusive), MaxExclusive: lit(wp.MaxExclusive),
				Pattern: wp.Pattern, In: wp.In, Severity: wp.Severity, Messages: wp.Messages,
			})
		}
		set.NodeShapes = append(set.NodeShapes, ns)
	}
	return normalize(set), nil
}

func checkOptionalIRI(where, iri string) error {
	if iri == "" {
		return nil
	}
	return checkIRIs(where, iri)
}

func checkIRIs(where string, iris ...string) error {
	for _, iri := range iris {
		if !isAbsoluteIRI(iri) {
			return fmt.Errorf("shacl: shapes: %s: %q is not an absolute IRI", where, iri)
		}
	}
	return nil
}

// isAbsoluteIRI checks RFC 3986 §3.1 scheme syntax, ALPHA *( ALPHA / DIGIT /
// "+" / "-" / "." ) followed by ':', and refuses the characters Turtle's
// IRIREF excludes (Turtle §6.4 production [18]).
func isAbsoluteIRI(iri string) bool {
	colon := strings.IndexByte(iri, ':')
	if colon <= 0 {
		return false
	}
	for i, c := range iri[:colon] {
		alpha := c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
		if !alpha && (i == 0 || !(c >= '0' && c <= '9' || c == '+' || c == '-' || c == '.')) {
			return false
		}
	}
	for _, c := range iri {
		if c <= 0x20 || strings.ContainsRune("<>\"{}|^`\\", c) {
			return false
		}
	}
	return true
}

// normalize puts a shape set in its canonical form, so equal shapes decode to
// equal sets whatever the input order: node shapes by ID; properties by path,
// ID, then content; IRI and string lists sorted with repeats removed (they
// are sets in RDF); empty lists nil, except an empty In, which means sh:in ().
func normalize(set ShapeSet) ShapeSet {
	for i := range set.NodeShapes {
		ns := &set.NodeShapes[i]
		ns.TargetClasses = sortedSet(ns.TargetClasses)
		ns.IgnoredProperties = sortedSet(ns.IgnoredProperties)
		ns.Messages = sortedSet(ns.Messages)
		for j := range ns.Properties {
			p := &ns.Properties[j]
			p.Messages = sortedSet(p.Messages)
			if p.In != nil {
				in := sortedSet(*p.In)
				if in == nil {
					in = []string{}
				}
				p.In = &in
			}
		}
		slices.SortFunc(ns.Properties, func(a, b PropertyShape) int {
			return cmp.Or(cmp.Compare(a.Path, b.Path), cmp.Compare(a.ID, b.ID), cmp.Compare(contentKey(a), contentKey(b)))
		})
		ns.Properties = slices.CompactFunc(ns.Properties, func(a, b PropertyShape) bool {
			return contentKey(a) == contentKey(b)
		})
		if len(ns.Properties) == 0 {
			ns.Properties = nil
		}
	}
	slices.SortFunc(set.NodeShapes, func(a, b NodeShape) int { return cmp.Compare(a.ID, b.ID) })
	if len(set.NodeShapes) == 0 {
		set.NodeShapes = nil
	}
	slices.SortFunc(set.rejections, func(a, b rejection) int {
		return cmp.Or(cmp.Compare(a.Shape, b.Shape), cmp.Compare(a.Path, b.Path),
			cmp.Compare(a.Construct, b.Construct), cmp.Compare(a.Detail, b.Detail))
	})
	set.rejections = slices.Compact(set.rejections)
	if len(set.rejections) == 0 {
		set.rejections = nil
	}
	return set
}

// contentKey is a property shape's full content, used only to order shapes
// whose path and ID are equal.
func contentKey(p PropertyShape) string {
	// A PropertyShape holds only strings, ints, and pointers to them, which
	// json.Marshal always encodes.
	b, _ := json.Marshal(p)
	return string(b)
}

func sortedSet(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := slices.Clone(in)
	slices.Sort(out)
	return slices.Compact(out)
}

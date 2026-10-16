package shacl

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

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
var nonStringKeys = map[string]bool{"min_count": true, "max_count": true, "closed": true}

// DecodeShapesJSON decodes SemVocab's JSON shape form. It refuses unknown
// fields, a trailing value, a numeric bound or other non-string where a
// string belongs, relative IRIs, a blank-node node shape, a property shape
// with no path, and a node shape listed twice.
func DecodeShapesJSON(r io.Reader) (ShapeSet, error) {
	var w wireShapeSet
	if err := decodeJSON(r, &w); err != nil {
		return ShapeSet{}, fmt.Errorf("shacl: shapes: %w", err)
	}
	return w.build()
}

// DecodeShapesYAML decodes SemVocab's YAML shape form with the same rules as
// DecodeShapesJSON. It also refuses a trailing document, a duplicate key, and
// any scalar not tagged !!str where a string belongs, such as an unquoted
// bound (ruling Q28).
func DecodeShapesYAML(r io.Reader) (ShapeSet, error) {
	var w wireShapeSet
	if err := decodeYAML(r, &w); err != nil {
		return ShapeSet{}, fmt.Errorf("shacl: shapes: %w", err)
	}
	return w.build()
}

// DecodeMappingJSON decodes a mapping document, refusing unknown fields, a
// trailing value, and a non-string where a string belongs. Compile checks the
// mapping's rules (design D6).
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
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	// encoding/json decodes null into a string as "" (in a list) or leaves
	// the field unset, so a null is refused before the struct decode. Every
	// field of both forms is a string, a number, a boolean, or a list or
	// object of them; none takes null. YAML refuses null through the !!str
	// check.
	var tree any
	if json.Unmarshal(data, &tree) == nil {
		if where, ok := findNull(tree, "$"); ok {
			return fmt.Errorf("%s is null; omit the field or give a value", where)
		}
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	var extra json.RawMessage
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("trailing data after the document")
	}
	return nil
}

// findNull returns the JSON path of the first null in v, visiting object
// keys in sorted order so the reported path does not depend on map order.
func findNull(v any, path string) (string, bool) {
	switch v := v.(type) {
	case nil:
		return path, true
	case []any:
		for i, e := range v {
			if p, ok := findNull(e, fmt.Sprintf("%s[%d]", path, i)); ok {
				return p, true
			}
		}
	case map[string]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, k := range keys {
			if p, ok := findNull(v[k], path+"."+k); ok {
				return p, true
			}
		}
	}
	return "", false
}

func decodeYAML(r io.Reader, out any) error {
	data, err := io.ReadAll(r)
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
		return err
	}
	var extra yaml.Node
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		if err != nil {
			return err
		}
		return fmt.Errorf("line %d: trailing document", extra.Line)
	}
	// Then the tags: yaml.v3 decodes an unquoted 5 into a string field
	// without error (probe P3).
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return err
	}
	return checkStringTags(&doc, "")
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

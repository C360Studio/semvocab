package shacl

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/c360studio/semengine/message"
	"github.com/c360studio/semengine/vocabulary"
	"github.com/c360studio/semengine/vocabulary/export"
)

// This file is design D9: the RDF term SemEngine's export would write for a
// candidate value, restated because export's classifier is unexported and
// reads the global registry (vocabulary/export/object.go:88-107 at
// v0.1.0-alpha.2). Every citation below is to that file at that version
// unless it names another. The agreement test in shacl/internal/agreement
// checks the mapping against export.Serialize, and fails when one row here
// changes.

type termKind int

const (
	termIRI termKind = iota + 1
	termBlank
	termLiteral
)

// term is an RDF term. A literal's datatype is never empty: a plain string
// is an xsd:string literal (RDF 1.1 Concepts §3.3).
type term struct {
	kind     termKind
	value    string // the IRI, the blank-node label with its "_:", or the lexical form
	datatype string // literals only
}

// String writes the term in N-Triples syntax (RDF 1.1 N-Triples §2.4–2.5),
// as export writes it (vocabulary/export/ntriples.go:46-60): an xsd:string
// literal without its datatype, and the same four string escapes
// (object.go:415-435).
func (t term) String() string {
	switch t.kind {
	case termIRI:
		return "<" + t.value + ">"
	case termBlank:
		return t.value
	}
	lit := `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\r", `\r`, "\t", `\t`).Replace(t.value) + `"`
	if t.datatype == xsdString {
		return lit
	}
	return lit + "^^<" + t.datatype + ">"
}

// wholeFloatBound is the largest magnitude at which export renders a float as
// an integer (object.go:34).
const wholeFloatBound = 1 << 53

// declaredDatatype is the datatype each pack data_type exports as
// (object.go:41-46). entity_id exports as an IRI, not a literal, so it has
// no entry; its hint is message.EntityReferenceDatatype.
var declaredDatatype = map[string]string{
	vocabulary.DataTypeString:   xsdString,
	vocabulary.DataTypeInt:      xsdInt,
	vocabulary.DataTypeFloat:    xsdDouble,
	vocabulary.DataTypeBool:     xsdBool,
	vocabulary.DataTypeDateTime: xsdDateTime,
	vocabulary.DataTypeJSON:     rdfJSON,
}

// valueTerm maps one candidate triple's object to its term. dataType is the
// predicate's data_type in the bundle's pack, and isDeclared says whether
// the pack declares the predicate at all.
//
// It returns an error, where export would write nothing or something else
// (probe P5 in notes/1-probes.md), for: a nil object, a NaN or infinite
// float (export drops all three, ntriples.go:19-22, object.go:89-91 and
// :321-324); a Go type export has no case for (it writes the %v form as a
// plain string, object.go:304-308); and a Datatype hint other than the one
// the data_type exports as, or @id on an entity_id predicate (export writes
// any hint verbatim, object.go:138-142). An accepted hint must also leave the
// term unchanged: export writes a hinted value's %v form, which can differ
// from the term the data_type gives (notes/6-validator.md).
//
// On a predicate the pack does not declare, any hint is accepted and the
// term is the one export writes for it (ruling Q29; hintedTerm).
func valueTerm(dataType string, isDeclared bool, t message.Triple) (term, error) {
	if err := checkObject(t.Object); err != nil {
		return term{}, err
	}
	if !utf8.ValidString(t.Datatype) {
		return term{}, fmt.Errorf("the Datatype hint %q is not valid UTF-8 (ruling Q31)", t.Datatype)
	}
	if !isDeclared && t.Datatype != "" {
		return hintedTerm(t), nil
	}
	declared := classify(dataType, t.Object)
	if t.Datatype == "" {
		return declared, nil
	}
	if t.Datatype == message.EntityReferenceDatatype {
		if dataType != vocabulary.DataTypeEntityID {
			return term{}, fmt.Errorf("Datatype hint %q is accepted only on an entity_id predicate, and the predicate's data_type is %q",
				t.Datatype, dataType)
		}
		// Export's @id branch (object.go:126-136) and its entity_id branch
		// (:160-168) give the same term for every value.
		return declared, nil
	}
	want, ok := declaredDatatype[dataType]
	if !ok {
		return term{}, fmt.Errorf("Datatype hint %q is not accepted: the predicate's data_type is %q, which names no XSD datatype",
			t.Datatype, dataType)
	}
	hinted := term{kind: termLiteral, value: fmt.Sprintf("%v", t.Object), datatype: expandDatatype(t.Datatype)}
	if hinted.datatype != want {
		return term{}, fmt.Errorf("Datatype hint %q is not <%s>, which data_type %q exports as", t.Datatype, want, dataType)
	}
	if hinted != declared {
		return term{}, fmt.Errorf("Datatype hint %q would make export write %s where data_type %q gives %s",
			t.Datatype, hinted, dataType, declared)
	}
	return declared, nil
}

// hintedTerm is export's hint branch, object.go:125-143: @id on a
// blank-node label or a valid entity ID gives that blank node or IRI, and
// otherwise falls back to observation; any other hint gives the %v lexical
// form with the hint, expanded, as datatype, ill-typed or not.
func hintedTerm(t message.Triple) term {
	if t.Datatype == message.EntityReferenceDatatype {
		if s, ok := t.Object.(string); ok {
			if isBlankNodeLabel(s) {
				return term{kind: termBlank, value: s}
			}
			if message.IsValidEntityID(s) {
				return term{kind: termIRI, value: export.SubjectIRI(s, "")}
			}
		}
		return classifyObserved(t.Object)
	}
	return term{kind: termLiteral, value: fmt.Sprintf("%v", t.Object), datatype: expandDatatype(t.Datatype)}
}

// checkObject refuses the values D9 makes input errors whatever the
// predicate: nil, a NaN or infinite float, and a Go type export's observation
// has no case for (object.go:281-310).
func checkObject(obj any) error {
	switch v := obj.(type) {
	case nil:
		return fmt.Errorf("the object is nil, which export drops")
	case float32:
		return checkFloat(float64(v))
	case float64:
		return checkFloat(v)
	case string:
		if !utf8.ValidString(v) {
			return fmt.Errorf("the object %q is not valid UTF-8, which export would write with U+FFFD (ruling Q31)", v)
		}
		return nil
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, time.Time:
		return nil
	default:
		return fmt.Errorf("the object's Go type %T is not one export classifies (string, bool, an integer or float type, time.Time)", obj)
	}
}

func checkFloat(f float64) error {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return fmt.Errorf("the object is %v, which export drops", f)
	}
	return nil
}

// classify is export's order without the hint (object.go:98-106): the
// declared data_type when the value can carry it, else observation.
func classify(dataType string, obj any) term {
	if t, ok := classifyDeclared(dataType, obj); ok {
		return t
	}
	return classifyObserved(obj)
}

// classifyDeclared is object.go:158-202. data_type string is absent there on
// purpose (object.go:151-157): it confirms observation.
func classifyDeclared(dataType string, obj any) (term, bool) {
	switch dataType {
	case vocabulary.DataTypeEntityID:
		if s, ok := obj.(string); ok {
			if isBlankNodeLabel(s) {
				return term{kind: termBlank, value: s}, true
			}
			if message.IsValidEntityID(s) {
				return term{kind: termIRI, value: export.SubjectIRI(s, "")}, true
			}
		}
	case vocabulary.DataTypeInt:
		if lex, ok := integerLexical(obj); ok {
			return term{kind: termLiteral, value: lex, datatype: xsdInt}, true
		}
	case vocabulary.DataTypeFloat:
		if f, ok := floatValue(obj); ok {
			return term{kind: termLiteral, value: formatFloat(f), datatype: xsdDouble}, true
		}
	case vocabulary.DataTypeBool:
		if b, ok := obj.(bool); ok {
			return term{kind: termLiteral, value: strconv.FormatBool(b), datatype: xsdBool}, true
		}
	case vocabulary.DataTypeDateTime:
		if lex, ok := dateTimeLexical(obj); ok {
			return term{kind: termLiteral, value: lex, datatype: xsdDateTime}, true
		}
	case vocabulary.DataTypeJSON:
		if s, ok := obj.(string); ok && json.Valid([]byte(s)) {
			return term{kind: termLiteral, value: s, datatype: rdfJSON}, true
		}
	}
	return term{}, false
}

// classifyObserved is object.go:281-310 for the types checkObject admits.
func classifyObserved(obj any) term {
	switch v := obj.(type) {
	case string:
		return classifyString(v)
	case bool:
		return term{kind: termLiteral, value: strconv.FormatBool(v), datatype: xsdBool}
	case float32:
		return term{kind: termLiteral, value: formatFloat(float64(v)), datatype: xsdDouble}
	case float64:
		return term{kind: termLiteral, value: formatFloat(v), datatype: xsdDouble}
	case time.Time:
		return term{kind: termLiteral, value: v.Format(time.RFC3339Nano), datatype: xsdDateTime}
	default: // the integer types
		return term{kind: termLiteral, value: fmt.Sprintf("%d", v), datatype: xsdInt}
	}
}

// classifyString is object.go:341-365: a blank-node label, then an entity ID
// (an IRI under SemEngine's default base, export.go:128-144), then an
// absolute IRI, else a plain literal.
func classifyString(s string) term {
	switch {
	case isBlankNodeLabel(s):
		return term{kind: termBlank, value: s}
	case message.IsValidEntityID(s):
		return term{kind: termIRI, value: export.SubjectIRI(s, "")}
	case isExportIRI(s):
		return term{kind: termIRI, value: s}
	}
	return term{kind: termLiteral, value: s, datatype: xsdString}
}

// integerLexical is object.go:209-227.
func integerLexical(obj any) (string, bool) {
	switch v := obj.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v), true
	case float32:
		return wholeFloatLexical(float64(v))
	case float64:
		return wholeFloatLexical(v)
	}
	return "", false
}

func wholeFloatLexical(f float64) (string, bool) {
	if math.IsNaN(f) || math.IsInf(f, 0) || f != math.Trunc(f) || math.Abs(f) > wholeFloatBound {
		return "", false
	}
	return strconv.FormatInt(int64(f), 10), true
}

// floatValue is object.go:230-259.
func floatValue(obj any) (float64, bool) {
	switch v := obj.(type) {
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	}
	return 0, false
}

// dateTimeLexical is object.go:265-278.
func dateTimeLexical(obj any) (string, bool) {
	switch v := obj.(type) {
	case time.Time:
		return v.Format(time.RFC3339Nano), true
	case string:
		parsed, err := time.Parse(time.RFC3339Nano, v)
		if err != nil {
			return "", false
		}
		return parsed.Format(time.RFC3339Nano), true
	}
	return "", false
}

// formatFloat is object.go:406-413.
func formatFloat(f float64) string {
	s := fmt.Sprintf("%g", f)
	if !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return s
}

// isExportIRI is object.go:381-402: http, https, or urn, with something
// after the scheme and no character a Turtle IRIREF excludes.
func isExportIRI(s string) bool {
	var rest string
	switch {
	case strings.HasPrefix(s, "http://"):
		rest = s[len("http://"):]
	case strings.HasPrefix(s, "https://"):
		rest = s[len("https://"):]
	case strings.HasPrefix(s, "urn:"):
		rest = s[len("urn:"):]
	default:
		return false
	}
	if rest == "" {
		return false
	}
	for _, r := range s {
		if r <= 0x20 || strings.ContainsRune("<>\"{}|^`\\", r) {
			return false
		}
	}
	return true
}

// isBlankNodeLabel is object.go:369-371 with prefix.go:148-197: "_:" then a
// Turtle PN_LOCAL with no colon, escape, or trailing dot.
func isBlankNodeLabel(s string) bool {
	local, ok := strings.CutPrefix(s, "_:")
	if !ok || local == "" || strings.HasSuffix(local, ".") {
		return false
	}
	for i, r := range local {
		if i == 0 && !isPNLocalStart(r) || i > 0 && r != '.' && !isPNLocalChar(r) {
			return false
		}
	}
	return true
}

func isPNLocalStart(r rune) bool {
	return isPNCharsBase(r) || r == '_' || r >= '0' && r <= '9'
}

func isPNLocalChar(r rune) bool {
	return isPNLocalStart(r) || r == '-' || r == 0x00B7 || r >= 0x0300 && r <= 0x036F || r >= 0x203F && r <= 0x2040
}

func isPNCharsBase(r rune) bool {
	return r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' ||
		r >= 0x00C0 && r <= 0x00D6 || r >= 0x00D8 && r <= 0x00F6 || r >= 0x00F8 && r <= 0x02FF ||
		r >= 0x0370 && r <= 0x037D || r >= 0x037F && r <= 0x1FFF || r >= 0x200C && r <= 0x200D ||
		r >= 0x2070 && r <= 0x218F || r >= 0x2C00 && r <= 0x2FEF || r >= 0x3001 && r <= 0xD7FF ||
		r >= 0xF900 && r <= 0xFDCF || r >= 0xFDF0 && r <= 0xFFFD || r >= 0x10000 && r <= 0xEFFFF
}

// expandDatatype is object.go:443-456: the xsd: and rdf: prefixes expand,
// and anything else is kept as written.
func expandDatatype(dt string) string {
	if rest, ok := strings.CutPrefix(dt, "xsd:"); ok {
		return xsdNS + rest
	}
	if rest, ok := strings.CutPrefix(dt, "rdf:"); ok {
		return rdfNS + rest
	}
	return dt
}

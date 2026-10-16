package shacl

import (
	"cmp"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"strings"

	"github.com/c360studio/semengine/message"
	semtypes "github.com/c360studio/semengine/pkg/types"
	"github.com/c360studio/semengine/vocabulary"
)

// Report is the outcome of validating one candidate entity state (design
// D13).
type Report struct {
	// Targeted is false when no shape applies to the entity's type, so "no
	// shapes" is never mistaken for "conforms".
	Targeted bool
	// Conforms is true when there are no results, whatever their severity
	// (SHACL §3.6.1).
	Conforms bool
	// Results are sorted by source shape, path, component, value, and then
	// predicate, with exact repeats removed.
	Results []Result
}

// Result is one validation result (SHACL §3.6.2), in native terms.
type Result struct {
	// FocusNode is the entity ID.
	FocusNode string
	// SourceShape is the property shape's IRI, or its node shape's when the
	// property shape is a blank node; for sh:closed, the node shape's.
	SourceShape string
	// Path is the sh:path IRI. For sh:closed it is the IRI the bundle's pack
	// gives the predicate, and empty when the pack does not declare it.
	Path string
	// Predicate is the dotted predicate the path maps to.
	Predicate string
	// Component is the constraint component's IRI, such as
	// http://www.w3.org/ns/shacl#MaxCountConstraintComponent.
	Component string
	// Value is the value node in N-Triples syntax, as SemEngine's export
	// would write it (design D9); empty for sh:minCount and sh:maxCount.
	Value string
	// Severity is the shape's sh:severity IRI, sh:Violation when it has none.
	Severity string
	// Messages are the shape's sh:message strings.
	Messages []string
}

// ValidateCandidateState checks the complete state of one entity after a
// mutation, not a patch, against every shape that applies to the entity's
// type (design D13). Building the candidate (stored values with a reconcile
// group replaced, or an append added) is the caller's job.
//
// It returns an error, and no report, when the input is malformed: an
// entityID SemEngine's ValidateEntityID refuses
// (pkg/types/entity_id.go:149-152 at v0.1.0-alpha.2), a triple about another
// subject, a predicate ParsePredicate refuses
// (vocabulary/predicate_contract.go:99), or a value design D9 cannot map to
// the term export would write. Every input error is reported together.
//
// Values are counted as distinct RDF terms. A predicate no applicable shape
// constrains is ignored, except by a closed shape. A validated candidate is a
// snapshot: an append is not revision-fenced at the pinned SemEngine
// (pkg/projection/mutation_types.go:61-67), so two appends each validated
// against one stored state can together exceed sh:maxCount. Revalidate after
// re-reading on a revision conflict, and use a fenced reconcile when a
// maximum must hold.
//
// It reads only the bundle, never SemEngine's registries.
func (b *Bundle) ValidateCandidateState(entityID string, triples []message.Triple) (Report, error) {
	id, err := semtypes.ParseEntityID(entityID)
	if err == nil {
		err = semtypes.ValidateEntityID(entityID)
	}
	if err != nil {
		return Report{}, fmt.Errorf("shacl: entity ID %q: %w", entityID, err)
	}

	values := map[string][]term{} // predicate → distinct terms, sorted
	var errs []string
	for _, t := range triples {
		if t.Subject != entityID {
			errs = append(errs, fmt.Sprintf("triple %s: subject %q is not the entity %q", t.Predicate, t.Subject, entityID))
			continue
		}
		if _, err := vocabulary.ParsePredicate(t.Predicate); err != nil {
			errs = append(errs, fmt.Sprintf("triple %q: %v", t.Predicate, err))
			continue
		}
		v, err := valueTerm(b.dataTypes[t.Predicate], t)
		if err != nil {
			errs = append(errs, fmt.Sprintf("triple %s: %v", t.Predicate, err))
			continue
		}
		if !slices.Contains(values[t.Predicate], v) {
			values[t.Predicate] = append(values[t.Predicate], v)
		}
	}
	if len(errs) > 0 {
		slices.Sort(errs)
		return Report{}, errors.New("shacl: invalid candidate state:\n  " + strings.Join(slices.Compact(errs), "\n  "))
	}
	for p := range values {
		slices.SortFunc(values[p], func(a, b term) int { return strings.Compare(a.String(), b.String()) })
	}

	shapes := b.shapes[id.Domain+"."+id.Type]
	v := validation{focus: entityID, values: values, iris: b.iris}
	for _, ns := range shapes {
		for _, p := range ns.Props {
			v.property(p)
		}
		if ns.Closed {
			v.closed(ns)
		}
	}
	slices.SortFunc(v.results, compareResults)
	results := slices.CompactFunc(v.results, func(a, b Result) bool { return compareResults(a, b) == 0 })
	return Report{Targeted: len(shapes) > 0, Conforms: len(results) == 0, Results: results}, nil
}

func compareResults(a, b Result) int {
	return cmp.Or(cmp.Compare(a.SourceShape, b.SourceShape), cmp.Compare(a.Path, b.Path),
		cmp.Compare(a.Component, b.Component), cmp.Compare(a.Value, b.Value), cmp.Compare(a.Predicate, b.Predicate),
		cmp.Compare(a.Severity, b.Severity), slices.Compare(a.Messages, b.Messages))
}

// validation collects the results for one candidate.
type validation struct {
	focus   string
	values  map[string][]term
	iris    map[string]string
	results []Result
}

func (v *validation) add(shape, path, pred, component, severity string, messages []string, value string) {
	if severity == "" {
		severity = shNS + "Violation"
	}
	v.results = append(v.results, Result{
		FocusNode: v.focus, SourceShape: shape, Path: path, Predicate: pred, Component: shNS + component,
		Value: value, Severity: severity, Messages: slices.Clone(messages),
	})
}

// property runs every constraint of one property shape.
func (v *validation) property(p compiledProperty) {
	vals := v.values[p.Predicate]
	fail := func(component string, value string) {
		v.add(p.Shape, p.Path, p.Predicate, component, p.Severity, p.Messages, value)
	}
	if len(vals) < p.MinCount {
		fail("MinCountConstraintComponent", "")
	}
	if p.MaxCount != nil && len(vals) > *p.MaxCount {
		fail("MaxCountConstraintComponent", "")
	}
	for _, t := range vals {
		if p.Datatype != "" && !hasDatatype(t, p.Datatype) {
			fail("DatatypeConstraintComponent", t.String())
		}
		for _, c := range []struct {
			component string
			bound     *bound
			holds     func(int) bool // the comparison of value with bound that must hold
		}{
			{"MinInclusiveConstraintComponent", p.MinIncl, func(n int) bool { return n >= 0 }},
			{"MaxInclusiveConstraintComponent", p.MaxIncl, func(n int) bool { return n <= 0 }},
			{"MinExclusiveConstraintComponent", p.MinExcl, func(n int) bool { return n > 0 }},
			{"MaxExclusiveConstraintComponent", p.MaxExcl, func(n int) bool { return n < 0 }},
		} {
			if c.bound == nil {
				continue
			}
			if n, ok := compareNumeric(t, c.bound); !ok || !c.holds(n) {
				fail(c.component, t.String())
			}
		}
		// §4.4.3: a blank node violates; otherwise the pattern is matched
		// against the IRI or the lexical form.
		if p.Pattern != nil && (t.kind == termBlank || !p.Pattern.MatchString(t.value)) {
			fail("PatternConstraintComponent", t.String())
		}
		// §4.8.3 over D11's string-only members: the value must be an
		// xsd:string literal equal to a member.
		if p.In != nil && (t.kind != termLiteral || t.datatype != xsdString || !slices.Contains(*p.In, t.value)) {
			fail("InConstraintComponent", t.String())
		}
	}
}

// closed is design D12: each value of a predicate the closed shape does not
// allow is a result.
func (v *validation) closed(ns compiledNodeShape) {
	for pred, vals := range v.values {
		if slices.Contains(ns.Allowed, pred) {
			continue
		}
		for _, t := range vals {
			v.add(ns.ID, v.iris[pred], pred, "ClosedConstraintComponent", ns.Severity, ns.Messages, t.String())
		}
	}
}

// hasDatatype is §4.1.2 for the D8 table. Every literal valueTerm returns is
// well formed for its datatype, because it is built from the Go value by
// export's own rules, so only the datatype IRI is compared.
func hasDatatype(t term, datatype string) bool {
	return t.kind == termLiteral && t.datatype == datatype
}

// compareNumeric compares a value with a bound exactly: the value must be an
// xsd:integer or xsd:double literal; an integer is never rounded through
// float64. It reports false when the two are not comparable: the value is
// not numeric, or either is NaN.
func compareNumeric(t term, b *bound) (int, bool) {
	if t.kind != termLiteral {
		return 0, false
	}
	var value *big.Float
	switch t.datatype {
	case xsdInt:
		n, ok := parseXSDInteger(t.value)
		if !ok {
			return 0, false
		}
		if b.Int != nil {
			return n.Cmp(b.Int), true
		}
		value = new(big.Float).SetInt(n)
	case xsdDouble:
		f, nan, ok := parseXSDDouble(t.value)
		if !ok || nan {
			return 0, false
		}
		value = new(big.Float).SetFloat64(f)
	default:
		return 0, false
	}
	var limit *big.Float
	switch {
	case b.Int != nil:
		limit = new(big.Float).SetInt(b.Int)
	case b.NaN:
		return 0, false
	default:
		limit = new(big.Float).SetFloat64(b.Double)
	}
	return value.Cmp(limit), true
}

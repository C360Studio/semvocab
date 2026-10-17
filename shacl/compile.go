package shacl

import (
	"cmp"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/c360studio/semengine/pkg/projection/contract"
	"github.com/c360studio/semengine/vocabulary"
)

// More IRIs the compiler recognises. SemEngine exports none of them: its XSD
// and rdf: constants are unexported (vocabulary/export/object.go:17-21 at
// v0.1.0-alpha.2).
const (
	xsdDouble   = xsdNS + "double"
	xsdDateTime = xsdNS + "dateTime"
	rdfJSON     = rdfNS + "JSON"
)

// datatypeTable is design D8: each accepted sh:datatype and the pack
// data_type it implies. SemEngine's export writes each data_type as that
// datatype (vocabulary/export/object.go:41-46 at v0.1.0-alpha.2). Every other
// datatype is rejected.
var datatypeTable = map[string]string{
	xsdString:   vocabulary.DataTypeString,
	xsdInt:      vocabulary.DataTypeInt,
	xsdDouble:   vocabulary.DataTypeFloat,
	xsdBool:     vocabulary.DataTypeBool,
	xsdDateTime: vocabulary.DataTypeDateTime,
	rdfJSON:     vocabulary.DataTypeJSON,
}

// severities are the sh:severity values the profile accepts (design D4).
var severities = []string{shNS + "Violation", shNS + "Warning", shNS + "Info"}

// Issue is one reason a compile failed. Shape is the shape's IRI (for a blank
// property shape, its node shape's), Path the property path, and Construct
// the IRI of the offending SHACL construct or, for a mapping or pack problem,
// the mapping field ("entity_types", "predicates", "contracts",
// "closed_ignored_predicates") or "pack".
type Issue struct {
	Shape     string
	Path      string
	Construct string
	Message   string
}

func (i Issue) String() string {
	var b strings.Builder
	b.WriteString(i.Construct)
	if i.Shape != "" {
		b.WriteString(" on " + i.Shape)
	}
	if i.Path != "" {
		b.WriteString(" path " + i.Path)
	}
	b.WriteString(": " + i.Message)
	return b.String()
}

// CompileError lists every issue a compile found, sorted by shape, path,
// construct, and message (design D4).
type CompileError struct {
	Issues []Issue
}

func (e *CompileError) Error() string {
	lines := make([]string, len(e.Issues))
	for i, is := range e.Issues {
		lines[i] = "  " + is.String()
	}
	return fmt.Sprintf("shacl: compile failed with %d issue(s):\n%s", len(e.Issues), strings.Join(lines, "\n"))
}

// Bundle is a compiled shapes document: the pack to register, the projection
// contracts, and the compiled constraints the native validator checks.
type Bundle struct {
	pack       vocabulary.PackSpec
	contracts  []contract.Contract
	unwritable []string
	// shapes holds, per entity type, every node shape that applies to it
	// through the mapping, sorted by node shape IRI.
	shapes map[string][]compiledNodeShape
	// dataTypes and iris give each pack predicate's data_type and iri, for
	// the validator's value terms (design D9) and closed-shape paths.
	dataTypes map[string]string
	iris      map[string]string
}

// Pack returns a copy of the compiled pack: the base pack in its original
// entry order, with an empty data_type filled from the shapes (design D5).
// A composition root registers it with SemEngine's vocabulary.RegisterPack.
func (b *Bundle) Pack() vocabulary.PackSpec { return clonePack(b.pack) }

// Contracts returns a copy of the projection contracts, sorted by name, with
// groups and predicate lists sorted (design D7). MessageType and
// IndexingProfile are zero: the composition root binds them when it
// registers a contract with its payload type.
func (b *Bundle) Contracts() []contract.Contract {
	out := make([]contract.Contract, len(b.contracts))
	for i, c := range b.contracts {
		out[i] = c
		out[i].BirthPredicates = slices.Clone(c.BirthPredicates)
		out[i].Groups = make([]contract.PredicateGroup, len(c.Groups))
		for j, g := range c.Groups {
			out[i].Groups[j] = g
			out[i].Groups[j].Predicates = slices.Clone(g.Predicates)
		}
	}
	return out
}

// Unwritable returns the mapped entity types that get no contract because
// the mapping gives them no birth or group predicate (ruling Q18), sorted.
// Shapes still apply to them in validation; no contract declares how a
// writer changes them.
func (b *Bundle) Unwritable() []string { return slices.Clone(b.unwritable) }

type compiledNodeShape struct {
	ID       string
	Closed   bool
	Allowed  []string // dotted predicates a closed shape allows, sorted
	Severity string
	Messages []string
	Props    []compiledProperty
}

type compiledProperty struct {
	Shape     string // property shape IRI, or its node shape's when blank
	Path      string
	Predicate string
	MinCount  int
	MaxCount  *int
	Datatype  string
	MinIncl   *bound
	MaxIncl   *bound
	MinExcl   *bound
	MaxExcl   *bound
	Pattern   *regexp.Regexp
	In        *[]string
	Severity  string
	Messages  []string
}

// bound is a numeric bound held exactly: an xsd:integer as a big.Int, an
// xsd:double as a float64 (NaN flagged, so bundles compare equal).
type bound struct {
	Int    *big.Int
	Double float64
	NaN    bool
}

// cmpBound compares two bounds of one datatype. NaN compares as unordered.
func cmpBound(a, b *bound) (int, bool) {
	if a.Int != nil {
		return a.Int.Cmp(b.Int), true
	}
	if a.NaN || b.NaN {
		return 0, false
	}
	return cmp.Compare(a.Double, b.Double), true
}

// compiler accumulates issues across every check, so a compile reports them
// all together (design D4).
type compiler struct {
	issues []Issue
}

func (c *compiler) add(shape, path, construct, format string, args ...any) {
	c.issues = append(c.issues, Issue{Shape: shape, Path: path, Construct: construct, Message: fmt.Sprintf(format, args...)})
}

// Compile compiles shapes against a reviewed base pack and a mapping (design
// D5–D12). It reports every out-of-profile construct the shapes recorded
// (ruling Q26) and every mapping, datatype, and contract problem, together,
// in a *CompileError, and then returns no bundle. It neither reads nor writes
// SemEngine's registries: the checks it borrows are SemEngine's pure ones,
// (*PackSpec).Validate (vocabulary/pack.go:180-186), contract.ValidateShape
// (pkg/projection/contract/contract.go:63-69), which checks the entity
// pattern with ValidateEntityIDPattern (:75) and predicates with
// IsValidPredicate (:134), and IsValidPredicate for
// closed_ignored_predicates, all at v0.1.0-alpha.2. Validate and
// IsValidPredicate fill SemEngine's predicate-parse memo
// (vocabulary/predicate_contract.go:85-142), which is not a registry.
func Compile(pack vocabulary.PackSpec, shapes ShapeSet, m Mapping) (*Bundle, error) {
	c := &compiler{}
	m = canonicalMapping(m)
	for _, r := range shapes.rejections {
		c.add(r.Shape, r.Path, r.Construct, "%s", rejectionMessage(r))
	}
	idx := c.indexMapping(pack, m)
	compiled := c.compileShapes(shapes, idx)
	out := c.fillDatatypes(pack, shapes, idx)
	contracts, unwritable := c.compileContracts(out, shapes, m, idx)
	if err := out.Validate(); err != nil {
		c.add("", "", "pack", "the compiled pack fails SemEngine's (*PackSpec).Validate: %v", err)
	}
	if len(c.issues) > 0 {
		slices.SortFunc(c.issues, func(a, b Issue) int {
			return cmp.Or(cmp.Compare(a.Shape, b.Shape), cmp.Compare(a.Path, b.Path),
				cmp.Compare(a.Construct, b.Construct), cmp.Compare(a.Message, b.Message))
		})
		return nil, &CompileError{Issues: slices.Compact(c.issues)}
	}
	b := &Bundle{pack: out, contracts: contracts, unwritable: unwritable, shapes: compiled,
		dataTypes: map[string]string{}, iris: map[string]string{}}
	for _, p := range out.Predicates {
		b.dataTypes[p.Name] = p.DataType
		if p.IRI != "" {
			b.iris[p.Name] = p.IRI
		}
	}
	return b, nil
}

// canonicalMapping returns a copy of m with every row list sorted. Rows are
// sets: where two rows conflict, the checks below keep the first and report
// the rest, so without a fixed order the issues would depend on how the
// author ordered the document (design D16). Repeats are kept, so each
// duplicate check still sees them.
func canonicalMapping(m Mapping) Mapping {
	out := Mapping{
		EntityTypes:             slices.Clone(m.EntityTypes),
		Predicates:              slices.Clone(m.Predicates),
		Contracts:               slices.Clone(m.Contracts),
		ClosedIgnoredPredicates: slices.Sorted(slices.Values(m.ClosedIgnoredPredicates)),
	}
	for i := range out.EntityTypes {
		out.EntityTypes[i].DomainTypes = slices.Sorted(slices.Values(m.EntityTypes[i].DomainTypes))
	}
	slices.SortFunc(out.EntityTypes, func(a, b EntityTypeBinding) int {
		return cmp.Or(cmp.Compare(a.ClassIRI, b.ClassIRI), slices.Compare(a.DomainTypes, b.DomainTypes))
	})
	slices.SortFunc(out.Predicates, func(a, b PredicateBinding) int {
		return cmp.Or(cmp.Compare(a.PathIRI, b.PathIRI), cmp.Compare(a.Name, b.Name))
	})
	for i := range out.Contracts {
		ct := &out.Contracts[i]
		ct.Birth = slices.Sorted(slices.Values(m.Contracts[i].Birth))
		ct.Groups = slices.Clone(m.Contracts[i].Groups)
		for j := range ct.Groups {
			ct.Groups[j].Predicates = slices.Sorted(slices.Values(ct.Groups[j].Predicates))
		}
		slices.SortFunc(ct.Groups, func(a, b GroupBinding) int {
			return cmp.Or(cmp.Compare(a.Name, b.Name), cmp.Compare(a.Mode, b.Mode), slices.Compare(a.Predicates, b.Predicates))
		})
	}
	slices.SortFunc(out.Contracts, func(a, b ContractBinding) int {
		return cmp.Or(cmp.Compare(a.DomainType, b.DomainType), cmp.Compare(a.Name, b.Name),
			slices.Compare(a.Birth, b.Birth), slices.CompareFunc(a.Groups, b.Groups, func(x, y GroupBinding) int {
				return cmp.Or(cmp.Compare(x.Name, y.Name), cmp.Compare(x.Mode, y.Mode), slices.Compare(x.Predicates, y.Predicates))
			}))
	})
	return out
}

// rejectionMessage words a parser record by the profile's classification of
// its construct.
func rejectionMessage(r rejection) string {
	var msg string
	switch e, ok := profile[r.Construct]; {
	case ok && e.class == termRejected:
		msg = e.reason + " (SHACL " + e.section + ")"
	case ok:
		msg = "not accepted in this position or form"
	case strings.HasPrefix(r.Construct, shNS):
		msg = "not a term of the W3C SHACL vocabulary (2017-07-20)"
	default:
		msg = "outside the profile"
	}
	if r.Detail != "" {
		msg += ": " + r.Detail
	}
	return msg
}

// mappingIndex is the mapping, checked and keyed.
type mappingIndex struct {
	classTypes map[string][]string // class IRI → domain types
	predicate  map[string]string   // path IRI → dotted predicate
	mapped     map[string]bool     // every domain type a row lists
	entityID   map[string]bool     // path IRI → mapped to an entity_id pack predicate
	ignored    []string            // closed_ignored_predicates
}

func (c *compiler) indexMapping(pack vocabulary.PackSpec, m Mapping) mappingIndex {
	idx := mappingIndex{
		classTypes: map[string][]string{}, predicate: map[string]string{}, mapped: map[string]bool{},
		entityID: map[string]bool{},
	}
	packTypes := map[string]vocabulary.EntityTypeSpec{}
	for _, et := range pack.EntityTypes {
		packTypes[et.DomainType] = et
	}
	packPreds := map[string]vocabulary.PredicateSpec{}
	for _, p := range pack.Predicates {
		packPreds[p.Name] = p
	}

	for _, row := range m.EntityTypes {
		if _, dup := idx.classTypes[row.ClassIRI]; dup {
			c.add("", "", "entity_types", "class %s has more than one row", row.ClassIRI)
			continue
		}
		listed := sortedSet(row.DomainTypes)
		if len(listed) != len(row.DomainTypes) {
			c.add("", "", "entity_types", "class %s lists a domain type twice", row.ClassIRI)
		}
		idx.classTypes[row.ClassIRI] = listed
		// SHACL targets instances of subclasses (§2.1.3.2): the row lists
		// exactly the pack types whose class or superclass closure holds the
		// class (ruling Q24).
		var closure []string
		for _, et := range pack.EntityTypes {
			if et.ClassIRI == row.ClassIRI || slices.Contains(et.Superclasses, row.ClassIRI) {
				closure = append(closure, et.DomainType)
			}
		}
		for _, dt := range listed {
			idx.mapped[dt] = true
			switch _, ok := packTypes[dt]; {
			case !ok:
				c.add("", "", "entity_types", "class %s lists %s, which the pack does not declare", row.ClassIRI, dt)
			case !slices.Contains(closure, dt):
				c.add("", "", "entity_types",
					"class %s lists %s, whose class_iri and superclasses do not contain it", row.ClassIRI, dt)
			}
		}
		for _, dt := range closure {
			if !slices.Contains(listed, dt) {
				c.add("", "", "entity_types",
					"class %s does not list %s, whose class_iri or superclasses contain it; SHACL targets subclass instances (§2.1.3.2)",
					row.ClassIRI, dt)
			}
		}
	}

	for _, row := range m.Predicates {
		if prev, dup := idx.predicate[row.PathIRI]; dup {
			c.add("", row.PathIRI, "predicates", "path %s is mapped twice (%s and %s)", row.PathIRI, prev, row.Name)
			continue
		}
		idx.predicate[row.PathIRI] = row.Name
		idx.entityID[row.PathIRI] = packPreds[row.Name].DataType == vocabulary.DataTypeEntityID
		switch p, ok := packPreds[row.Name]; {
		case !ok:
			c.add("", row.PathIRI, "predicates", "path %s maps to %s, which the pack does not declare", row.PathIRI, row.Name)
		case p.IRI != row.PathIRI:
			c.add("", row.PathIRI, "predicates",
				"path %s maps to %s, whose pack iri is %q; export writes the pack iri, so they must be equal",
				row.PathIRI, row.Name, p.IRI)
		}
	}

	for _, p := range m.ClosedIgnoredPredicates {
		if !vocabulary.IsValidPredicate(p) {
			c.add("", "", "closed_ignored_predicates", "%q is not a three-part dotted predicate", p)
		}
	}
	idx.ignored = sortedSet(m.ClosedIgnoredPredicates)
	return idx
}

// propShape is the shape a property shape's issues name.
func propShape(ns NodeShape, p PropertyShape) string {
	if p.ID != "" {
		return p.ID
	}
	return ns.ID
}

func (c *compiler) compileShapes(shapes ShapeSet, idx mappingIndex) map[string][]compiledNodeShape {
	out := map[string][]compiledNodeShape{}
	for _, ns := range shapes.NodeShapes {
		cn := compiledNodeShape{ID: ns.ID, Closed: ns.Closed, Messages: ns.Messages}
		cn.Severity = c.severity(ns.ID, "", ns.Severity)
		if len(ns.TargetClasses) == 0 {
			c.add(ns.ID, "", shNS+"targetClass", "the node shape has no sh:targetClass, so it would never apply")
		}
		var types []string
		for _, class := range ns.TargetClasses {
			dts, ok := idx.classTypes[class]
			if !ok {
				c.add(ns.ID, "", shNS+"targetClass", "class %s has no entity_types mapping row", class)
				continue
			}
			types = append(types, dts...)
		}
		for _, p := range ns.Properties {
			cn.Props = append(cn.Props, c.compileProperty(ns, p, idx))
		}
		if ns.Closed {
			cn.Allowed = c.closedAllowed(ns, idx)
		}
		for _, dt := range sortedSet(types) {
			out[dt] = append(out[dt], cn)
		}
	}
	return out
}

// closedAllowed is design D12: a closed shape allows the predicates its
// paths map to, the mapped sh:ignoredProperties, and the mapping's
// closed_ignored_predicates. rdf:type must be ignored: native state has no
// rdf:type triple, but the exported graph does, and SHACL reports it
// (W3C test suite core/node/closed-001).
func (c *compiler) closedAllowed(ns NodeShape, idx mappingIndex) []string {
	if !slices.Contains(ns.IgnoredProperties, rdfType) {
		c.add(ns.ID, "", shNS+"closed",
			"a closed shape must list rdf:type in sh:ignoredProperties: the exported graph carries rdf:type, which SHACL reports as a closed-shape violation (W3C core/node/closed-001)")
	}
	allowed := slices.Clone(idx.ignored)
	for _, p := range ns.Properties {
		if name, ok := idx.predicate[p.Path]; ok {
			allowed = append(allowed, name)
		}
	}
	for _, iri := range ns.IgnoredProperties {
		if name, ok := idx.predicate[iri]; ok {
			allowed = append(allowed, name)
		}
	}
	return sortedSet(allowed)
}

func (c *compiler) severity(shape, path, sev string) string {
	if sev != "" && !slices.Contains(severities, sev) {
		c.add(shape, path, shNS+"severity", "<%s> is not sh:Violation, sh:Warning, or sh:Info", sev)
	}
	return sev
}

func (c *compiler) compileProperty(ns NodeShape, p PropertyShape, idx mappingIndex) compiledProperty {
	shape := propShape(ns, p)
	cp := compiledProperty{
		Shape: shape, Path: p.Path, MinCount: p.MinCount, MaxCount: p.MaxCount, Datatype: p.Datatype,
		Messages: p.Messages,
	}
	cp.Severity = c.severity(shape, p.Path, p.Severity)
	if p.Path != "" {
		name, ok := idx.predicate[p.Path]
		if !ok {
			c.add(shape, p.Path, shNS+"path", "path %s has no predicates mapping row", p.Path)
		}
		cp.Predicate = name
	}

	// Cardinality (design D4, §4.2).
	if p.MinCount < 0 {
		c.add(shape, p.Path, shNS+"minCount", "%d is negative", p.MinCount)
	}
	if p.MinCount > math.MaxInt32 {
		c.add(shape, p.Path, shNS+"minCount", "%d exceeds %d", p.MinCount, math.MaxInt32)
	}
	if p.MaxCount != nil {
		if *p.MaxCount < 0 {
			c.add(shape, p.Path, shNS+"maxCount", "%d is negative", *p.MaxCount)
		}
		if *p.MaxCount > math.MaxInt32 {
			c.add(shape, p.Path, shNS+"maxCount", "%d exceeds %d", *p.MaxCount, math.MaxInt32)
		}
		if p.MinCount > *p.MaxCount && *p.MaxCount >= 0 {
			c.add(shape, p.Path, shNS+"maxCount", "sh:minCount %d is above sh:maxCount %d", p.MinCount, *p.MaxCount)
		}
	}

	// Entity references take cardinality only (ruling Q17, design D8): the
	// check follows the mapped pack predicate, not sh:datatype, and replaces
	// every other check on those constructs, so each gets one issue.
	if idx.entityID[p.Path] {
		c.entityIDConstraints(shape, p)
		return cp
	}

	// Datatype (design D8).
	if p.Datatype != "" {
		if _, ok := datatypeTable[p.Datatype]; !ok {
			c.add(shape, p.Path, shNS+"datatype",
				"<%s> is not in the datatype table (xsd:string, xsd:integer, xsd:double, xsd:boolean, xsd:dateTime, rdf:JSON)",
				p.Datatype)
		}
	}

	// Numeric bounds (design D8).
	cp.MinIncl = c.bound(shape, p, "minInclusive", p.MinInclusive)
	cp.MaxIncl = c.bound(shape, p, "maxInclusive", p.MaxInclusive)
	cp.MinExcl = c.bound(shape, p, "minExclusive", p.MinExclusive)
	cp.MaxExcl = c.bound(shape, p, "maxExclusive", p.MaxExclusive)
	for _, lo := range []struct {
		name string
		b    *bound
	}{{"minInclusive", cp.MinIncl}, {"minExclusive", cp.MinExcl}} {
		for _, hi := range []struct {
			name string
			b    *bound
		}{{"maxInclusive", cp.MaxIncl}, {"maxExclusive", cp.MaxExcl}} {
			if lo.b == nil || hi.b == nil {
				continue
			}
			if n, ok := cmpBound(lo.b, hi.b); ok && n > 0 {
				c.add(shape, p.Path, shNS+lo.name, "sh:%s is above sh:%s", lo.name, hi.name)
			}
		}
	}

	// Pattern (design D10). sh:flags never reaches here: the parser records
	// it, and the profile rejects it.
	if p.Pattern != nil {
		re, err := compilePattern(*p.Pattern)
		if err != nil {
			c.add(shape, p.Path, shNS+"pattern", "%q is outside the regex subset: %v", *p.Pattern, err)
		}
		cp.Pattern = re
	}

	// Enumeration (design D11).
	if p.In != nil {
		if p.Datatype != xsdString {
			c.add(shape, p.Path, shNS+"in", "sh:in needs sh:datatype xsd:string on its property shape")
		}
		in := slices.Clone(*p.In)
		cp.In = &in
	}
	return cp
}

// entityIDConstraints reports each non-cardinality constraint on a property
// shape whose path maps to an entity_id predicate.
func (c *compiler) entityIDConstraints(shape string, p PropertyShape) {
	for _, k := range []struct {
		name    string
		present bool
	}{
		{"datatype", p.Datatype != ""}, {"minInclusive", p.MinInclusive != nil}, {"maxInclusive", p.MaxInclusive != nil},
		{"minExclusive", p.MinExclusive != nil}, {"maxExclusive", p.MaxExclusive != nil},
		{"pattern", p.Pattern != nil}, {"in", p.In != nil},
	} {
		if k.present {
			c.add(shape, p.Path, shNS+k.name,
				"the path maps to an entity_id predicate, which takes cardinality constraints only (ruling Q17)")
		}
	}
}

// bound checks one numeric bound: its property needs an sh:datatype of
// xsd:integer or xsd:double, and the bound must be a literal of that same
// datatype, so no comparison goes through SPARQL numeric promotion.
func (c *compiler) bound(shape string, p PropertyShape, name string, lit *Literal) *bound {
	if lit == nil {
		return nil
	}
	construct := shNS + name
	switch {
	case p.Datatype == "":
		c.add(shape, p.Path, construct, "a bound needs an sh:datatype (xsd:integer or xsd:double) on its property shape")
		return nil
	case p.Datatype != xsdInt && p.Datatype != xsdDouble:
		c.add(shape, p.Path, construct, "a bound needs sh:datatype xsd:integer or xsd:double, not <%s>", p.Datatype)
		return nil
	case lit.Datatype != p.Datatype:
		c.add(shape, p.Path, construct, "%q has datatype <%s>; it must be <%s>, its property's sh:datatype",
			lit.Lexical, lit.Datatype, p.Datatype)
		return nil
	}
	if p.Datatype == xsdInt {
		n, ok := parseXSDInteger(lit.Lexical)
		if !ok {
			c.add(shape, p.Path, construct, "%q is not an xsd:integer lexical form", lit.Lexical)
			return nil
		}
		return &bound{Int: n}
	}
	f, nan, ok := parseXSDDouble(lit.Lexical)
	if !ok {
		c.add(shape, p.Path, construct, "%q is not an xsd:double lexical form", lit.Lexical)
		return nil
	}
	return &bound{Double: f, NaN: nan}
}

// parseXSDInteger reads XSD's integer lexical space, [+-]?[0-9]+ (XML Schema
// Part 2 §3.3.13.1), into an exact big.Int.
func parseXSDInteger(s string) (*big.Int, bool) {
	digits := strings.TrimLeft(s, "+-")
	if len(s)-len(digits) > 1 || digits == "" || strings.Trim(digits, "0123456789") != "" {
		return nil, false
	}
	n, ok := new(big.Int).SetString(s, 10)
	return n, ok
}

// parseXSDDouble reads XSD 1.0's double lexical space (XML Schema Part 2
// §3.2.5.1): a decimal mantissa with an optional exponent, INF, -INF, or
// NaN. strconv.ParseFloat alone accepts spellings XSD does not, such as
// "Inf", "0x1p-2", and "1_0".
func parseXSDDouble(s string) (float64, bool, bool) {
	switch s {
	case "INF":
		return math.Inf(1), false, true
	case "-INF":
		return math.Inf(-1), false, true
	case "NaN":
		return 0, true, true
	}
	if !xsdDoubleLexical.MatchString(s) {
		return 0, false, false
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		// Out of range: XSD rounds to the nearest double, INF included;
		// ParseFloat returns ±Inf with ErrRange, which is that rounding.
		if ne, ok := err.(*strconv.NumError); ok && ne.Err == strconv.ErrRange {
			return f, false, true
		}
		return 0, false, false
	}
	return f, false, true
}

var xsdDoubleLexical = regexp.MustCompile(`^[+-]?([0-9]+(\.[0-9]*)?|\.[0-9]+)([Ee][+-]?[0-9]+)?$`)

// fillDatatypes is design D5: the compiled pack is the base pack, copied,
// with an empty data_type filled from the shapes. A base data_type that
// disagrees with a shape, or two shapes that imply different data types for
// one predicate, fail.
func (c *compiler) fillDatatypes(pack vocabulary.PackSpec, shapes ShapeSet, idx mappingIndex) vocabulary.PackSpec {
	out := clonePack(pack)
	type use struct{ shape, path, datatype string }
	implied := map[string][]use{}
	for _, ns := range shapes.NodeShapes {
		for _, p := range ns.Properties {
			name, ok := idx.predicate[p.Path]
			// An entity_id path's sh:datatype is already its Q17 issue.
			if _, known := datatypeTable[p.Datatype]; !ok || !known || idx.entityID[p.Path] {
				continue
			}
			implied[name] = append(implied[name], use{propShape(ns, p), p.Path, p.Datatype})
		}
	}
	for i := range out.Predicates {
		pred := &out.Predicates[i]
		uses := implied[pred.Name]
		if len(uses) == 0 {
			continue
		}
		for _, u := range uses[1:] {
			if u.datatype != uses[0].datatype {
				c.add(u.shape, u.path, shNS+"datatype",
					"predicate %s gets <%s> here and <%s> from shape %s", pred.Name, u.datatype, uses[0].datatype, uses[0].shape)
			}
		}
		want := datatypeTable[uses[0].datatype]
		switch pred.DataType {
		case "":
			pred.DataType = want
		case want:
		default:
			for _, u := range uses {
				if datatypeTable[u.datatype] != pred.DataType {
					c.add(u.shape, u.path, shNS+"datatype",
						"predicate %s has pack data_type %q, which <%s> contradicts", pred.Name, pred.DataType, u.datatype)
				}
			}
		}
	}
	return out
}

// compileContracts is design D7.
func (c *compiler) compileContracts(pack vocabulary.PackSpec, shapes ShapeSet, m Mapping, idx mappingIndex,
) ([]contract.Contract, []string) {
	declared := map[string]bool{}
	for _, p := range pack.Predicates {
		declared[p.Name] = true
	}
	written := map[string]map[string]bool{} // domain type → predicates its contract covers
	var contracts []contract.Contract
	names := map[string]string{}
	for _, b := range m.Contracts {
		if !idx.mapped[b.DomainType] {
			c.add("", "", "contracts", "contract for %s, which no entity_types row lists", b.DomainType)
			continue
		}
		if _, dup := written[b.DomainType]; dup {
			c.add("", "", "contracts", "%s has more than one contract", b.DomainType)
			continue
		}
		cover := map[string]bool{}
		written[b.DomainType] = cover
		ct, ok := c.contract(b, declared, cover)
		if !ok {
			continue
		}
		if prev, dup := names[ct.Name]; dup {
			c.add("", "", "contracts", "contracts for %s and %s are both named %q", prev, b.DomainType, ct.Name)
		}
		names[ct.Name] = b.DomainType
		if err := ct.ValidateShape(); err != nil {
			c.add("", "", "contracts", "contract %q fails SemEngine's ValidateShape: %v", ct.Name, err)
		}
		contracts = append(contracts, ct)
	}

	// Ruling Q19: every predicate a shape constrains for an entity type is in
	// that type's contract.
	for _, ns := range shapes.NodeShapes {
		var types []string
		for _, class := range ns.TargetClasses {
			types = append(types, idx.classTypes[class]...)
		}
		for _, p := range ns.Properties {
			name, ok := idx.predicate[p.Path]
			if !ok {
				continue
			}
			for _, dt := range sortedSet(types) {
				if !written[dt][name] {
					c.add(propShape(ns, p), p.Path, "contracts",
						"%s constrains %s for %s, but its contract has it in no birth or group list", ns.ID, name, dt)
				}
			}
		}
	}

	var unwritable []string
	for dt := range idx.mapped {
		if len(written[dt]) == 0 {
			unwritable = append(unwritable, dt)
		}
	}
	slices.Sort(unwritable)
	slices.SortFunc(contracts, func(a, b contract.Contract) int { return cmp.Compare(a.Name, b.Name) })
	return contracts, unwritable
}

// contract builds one contract from its binding. It returns false when the
// binding declares no predicate: the entity type is then unwritable.
func (c *compiler) contract(b ContractBinding, declared, cover map[string]bool) (contract.Contract, bool) {
	ct := contract.Contract{Name: cmp.Or(b.Name, b.DomainType)}
	domain, typ, _ := strings.Cut(b.DomainType, ".")
	ct.EntityPattern = "*.*.*." + domain + "." + typ + ".*"
	where := "contract " + ct.Name

	inGroup := map[string]string{}
	groups := map[string]*contract.PredicateGroup{}
	for _, g := range b.Groups {
		if len(g.Predicates) == 0 {
			c.add("", "", "contracts", "%s: group %q has no predicates", where, g.Name)
			continue
		}
		name := g.Name
		if name == "" {
			name = defaultGroupName(g.Predicates)
			if name == "" {
				c.add("", "", "contracts",
					"%s: a group with no name needs predicates that share one category, the default name", where)
				continue
			}
		}
		if g.Mode != contract.ModeReconcile && g.Mode != contract.ModeAppend {
			c.add("", "", "contracts", "%s group %s: mode %q must be reconcile or append", where, name, g.Mode)
			continue
		}
		pg, seen := groups[name]
		if !seen {
			pg = &contract.PredicateGroup{Name: name, Mode: g.Mode}
			groups[name] = pg
		} else if pg.Mode != g.Mode {
			c.add("", "", "contracts", "%s group %s is declared with modes %s and %s", where, name, pg.Mode, g.Mode)
			continue
		}
		for _, p := range g.Predicates {
			if prev, dup := inGroup[p]; dup && prev != name {
				c.add("", "", "contracts", "%s: %s is in groups %s and %s", where, p, prev, name)
				continue
			}
			inGroup[p] = name
			pg.Predicates = append(pg.Predicates, p)
		}
	}
	for _, p := range b.Birth {
		if g, dup := inGroup[p]; dup {
			c.add("", "", "contracts", "%s: %s is a birth predicate and in group %s", where, p, g)
		}
	}

	ct.BirthPredicates = sortedSet(b.Birth)
	for _, p := range ct.BirthPredicates {
		cover[p] = true
	}
	for _, name := range sortedKeys(groups) {
		g := groups[name]
		g.Predicates = sortedSet(g.Predicates)
		for _, p := range g.Predicates {
			cover[p] = true
		}
		ct.Groups = append(ct.Groups, *g)
	}
	for _, p := range sortedKeys(cover) {
		if !declared[p] {
			c.add("", "", "contracts", "%s names %s, which the compiled pack does not declare", where, p)
		}
	}
	return ct, len(cover) > 0
}

// defaultGroupName is the predicate category, the middle dotted segment,
// when every predicate shares it (design D7).
func defaultGroupName(preds []string) string {
	var name string
	for _, p := range preds {
		parts := strings.Split(p, ".")
		if len(parts) != 3 || name != "" && parts[1] != name {
			return ""
		}
		name = parts[1]
	}
	return name
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

// clonePack deep-copies a pack, keeping entry order.
func clonePack(p vocabulary.PackSpec) vocabulary.PackSpec {
	out := p
	out.Prefixes = slices.Clone(p.Prefixes)
	out.Predicates = slices.Clone(p.Predicates)
	for i := range out.Predicates {
		if proj := out.Predicates[i].Projection; proj != nil {
			cp := *proj
			cp.Steps = slices.Clone(proj.Steps)
			out.Predicates[i].Projection = &cp
		}
	}
	out.EntityTypes = slices.Clone(p.EntityTypes)
	for i := range out.EntityTypes {
		out.EntityTypes[i].Superclasses = slices.Clone(p.EntityTypes[i].Superclasses)
	}
	return out
}

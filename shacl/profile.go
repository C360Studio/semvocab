package shacl

// The compiler's profile (design D4): every term of the W3C SHACL vocabulary
// (shacl.ttl, 2017-07-20, vendored in internal/oracle; 183 terms) is
// supported, an ignored annotation, or rejected. A supported term is accepted
// only where D4 accepts it (sh:minCount on a property shape, not on a node
// shape); anywhere else the parser records it and Compile rejects it by name.
// An sh: term absent from the vocabulary is rejected as unknown.
//
// Sections are those of the SHACL Recommendation (W3C, 2017-07-20) unless
// marked AF (SHACL Advanced Features, W3C Working Group Note, 2017-06-08) or
// JS (SHACL JavaScript Extensions, W3C Working Group Note, 2017-06-08).
// "vocabulary only" marks a term the vocabulary declares and none of the three
// documents describes.

type termClass int

const (
	termSupported termClass = iota + 1
	termIgnored
	termRejected
)

func (c termClass) String() string {
	switch c {
	case termSupported:
		return "supported"
	case termIgnored:
		return "ignored annotation"
	case termRejected:
		return "rejected"
	}
	return "unclassified"
}

type profileEntry struct {
	class   termClass
	section string
	reason  string // why a term is rejected or ignored; empty for supported terms
}

// Reasons, one per family of rejected terms.
const (
	reasonAnnotation   = "non-validating annotation; accepted and dropped"
	reasonTarget       = "only sh:targetClass targets are compiled"
	reasonPath         = "only a predicate path (one IRI) is compiled"
	reasonGraphContext = "needs the type or shape of the referenced node, which complete entity state does not hold"
	reasonNodeKind     = "node kinds are not compiled (ruling Q17); entity_id predicates take cardinality only"
	reasonString       = "string-based component outside the profile"
	reasonFlags        = "regular-expression flags are not compiled (ruling Q15)"
	reasonHasValue     = "sh:hasValue is outside the profile"
	reasonPair         = "property-pair components compare two predicates and are outside the profile"
	reasonLogical      = "logical components are outside the profile"
	reasonQualified    = "qualified value shapes are outside the profile"
	reasonDeactivated  = "deactivation is outside the profile; remove the shape instead"
	reasonGroup        = "property groups are outside the profile"
	reasonShapeType    = "only sh:NodeShape and sh:PropertyShape are accepted as a shape's type"
	reasonSeverityType = "custom severities are outside the profile; use sh:Violation, sh:Warning, or sh:Info"
	reasonComponent    = "a constraint component IRI names a component in validation results; it is not a shapes construct"
	reasonReport       = "validation-report vocabulary; it is not a shapes construct"
	reasonDataGraph    = "a data-graph declaration; it is not a shapes construct"
	reasonSPARQL       = "SPARQL-based constraints and components are outside the profile"
	reasonJS           = "JavaScript extensions are outside the profile"
	reasonAF           = "SHACL Advanced Features are outside the profile"
	reasonRule         = "SHACL rules are not compiled: rule translation is deferred to issue #1 Phase 4"
)

type termSection struct{ term, section string }

// profile maps each vocabulary term's IRI to its classification.
var profile = buildProfile()

func buildProfile() map[string]profileEntry {
	p := map[string]profileEntry{}
	add := func(class termClass, reason string, terms ...termSection) {
		for _, t := range terms {
			if _, dup := p[shNS+t.term]; dup {
				panic("shacl: profile classifies " + t.term + " twice")
			}
			p[shNS+t.term] = profileEntry{class: class, section: t.section, reason: reason}
		}
	}
	add(termSupported, "",
		termSection{"NodeShape", "2.2"}, termSection{"PropertyShape", "2.3"},
		termSection{"targetClass", "2.1.3.2"}, termSection{"property", "4.7.2"}, termSection{"path", "2.3.1.1"},
		termSection{"minCount", "4.2.1"}, termSection{"maxCount", "4.2.2"}, termSection{"datatype", "4.1.2"},
		termSection{"minExclusive", "4.3.1"}, termSection{"minInclusive", "4.3.2"},
		termSection{"maxExclusive", "4.3.3"}, termSection{"maxInclusive", "4.3.4"},
		termSection{"pattern", "4.4.3"}, termSection{"in", "4.8.3"},
		termSection{"closed", "4.8.1"}, termSection{"ignoredProperties", "4.8.1"},
		termSection{"severity", "2.1.4"}, termSection{"Violation", "2.1.4"}, termSection{"Warning", "2.1.4"},
		termSection{"Info", "2.1.4"}, termSection{"message", "2.1.5"},
	)
	add(termIgnored, reasonAnnotation,
		termSection{"name", "2.3.2.1"}, termSection{"description", "2.3.2.1"}, termSection{"order", "2.3.2.2"},
		termSection{"defaultValue", "2.3.2.4"},
	)
	add(termRejected, reasonTarget,
		termSection{"targetNode", "2.1.3.1"}, termSection{"targetSubjectsOf", "2.1.3.4"},
		termSection{"targetObjectsOf", "2.1.3.5"},
	)
	add(termRejected, reasonPath,
		termSection{"alternativePath", "2.3.1.3"}, termSection{"inversePath", "2.3.1.4"},
		termSection{"zeroOrMorePath", "2.3.1.5"}, termSection{"oneOrMorePath", "2.3.1.6"},
		termSection{"zeroOrOnePath", "2.3.1.7"},
	)
	add(termRejected, reasonGraphContext, termSection{"class", "4.1.1"}, termSection{"node", "4.7.1"})
	add(termRejected, reasonNodeKind,
		termSection{"nodeKind", "4.1.3"}, termSection{"NodeKind", "4.1.3"}, termSection{"IRI", "4.1.3"},
		termSection{"BlankNode", "4.1.3"}, termSection{"Literal", "4.1.3"}, termSection{"BlankNodeOrIRI", "4.1.3"},
		termSection{"BlankNodeOrLiteral", "4.1.3"}, termSection{"IRIOrLiteral", "4.1.3"},
	)
	add(termRejected, reasonString,
		termSection{"minLength", "4.4.1"}, termSection{"maxLength", "4.4.2"}, termSection{"languageIn", "4.4.4"},
		termSection{"uniqueLang", "4.4.5"},
	)
	add(termRejected, reasonFlags, termSection{"flags", "4.4.3"})
	add(termRejected, reasonHasValue, termSection{"hasValue", "4.8.2"})
	add(termRejected, reasonPair,
		termSection{"equals", "4.5.1"}, termSection{"disjoint", "4.5.2"}, termSection{"lessThan", "4.5.3"},
		termSection{"lessThanOrEquals", "4.5.4"},
	)
	add(termRejected, reasonLogical,
		termSection{"not", "4.6.1"}, termSection{"and", "4.6.2"}, termSection{"or", "4.6.3"},
		termSection{"xone", "4.6.4"},
	)
	add(termRejected, reasonQualified,
		termSection{"qualifiedValueShape", "4.7.3"}, termSection{"qualifiedMinCount", "4.7.3"},
		termSection{"qualifiedMaxCount", "4.7.3"}, termSection{"qualifiedValueShapesDisjoint", "4.7.3"},
	)
	add(termRejected, reasonDeactivated, termSection{"deactivated", "2.1.6"})
	add(termRejected, reasonGroup, termSection{"group", "2.3.2.3"}, termSection{"PropertyGroup", "2.3.2.3"})
	add(termRejected, reasonShapeType, termSection{"Shape", "2.1"})
	add(termRejected, reasonSeverityType, termSection{"Severity", "2.1.4"})
	add(termRejected, reasonComponent,
		termSection{"AndConstraintComponent", "4.6.2"}, termSection{"ClassConstraintComponent", "4.1.1"},
		termSection{"ClosedConstraintComponent", "4.8.1"}, termSection{"DatatypeConstraintComponent", "4.1.2"},
		termSection{"DisjointConstraintComponent", "4.5.2"}, termSection{"EqualsConstraintComponent", "4.5.1"},
		termSection{"HasValueConstraintComponent", "4.8.2"}, termSection{"InConstraintComponent", "4.8.3"},
		termSection{"LanguageInConstraintComponent", "4.4.4"},
		termSection{"LessThanConstraintComponent", "4.5.3"},
		termSection{"LessThanOrEqualsConstraintComponent", "4.5.4"},
		termSection{"MaxCountConstraintComponent", "4.2.2"},
		termSection{"MaxExclusiveConstraintComponent", "4.3.3"},
		termSection{"MaxInclusiveConstraintComponent", "4.3.4"},
		termSection{"MaxLengthConstraintComponent", "4.4.2"},
		termSection{"MinCountConstraintComponent", "4.2.1"},
		termSection{"MinExclusiveConstraintComponent", "4.3.1"},
		termSection{"MinInclusiveConstraintComponent", "4.3.2"},
		termSection{"MinLengthConstraintComponent", "4.4.1"}, termSection{"NodeConstraintComponent", "4.7.1"},
		termSection{"NodeKindConstraintComponent", "4.1.3"}, termSection{"NotConstraintComponent", "4.6.1"},
		termSection{"OrConstraintComponent", "4.6.3"}, termSection{"PatternConstraintComponent", "4.4.3"},
		termSection{"PropertyConstraintComponent", "4.7.2"},
		termSection{"QualifiedMaxCountConstraintComponent", "4.7.3"},
		termSection{"QualifiedMinCountConstraintComponent", "4.7.3"},
		termSection{"UniqueLangConstraintComponent", "4.4.5"}, termSection{"XoneConstraintComponent", "4.6.4"},
	)
	add(termRejected, reasonReport,
		termSection{"ValidationReport", "3.6.1"}, termSection{"conforms", "3.6.1.1"},
		termSection{"result", "3.6.1.2"}, termSection{"shapesGraphWellFormed", "3.6.1.3"},
		termSection{"ValidationResult", "3.6.2"}, termSection{"AbstractResult", "3.6.2"},
		termSection{"focusNode", "3.6.2.1"}, termSection{"resultPath", "3.6.2.2"}, termSection{"value", "3.6.2.3"},
		termSection{"sourceShape", "3.6.2.4"}, termSection{"sourceConstraintComponent", "3.6.2.5"},
		termSection{"detail", "3.6.2.6"}, termSection{"resultMessage", "3.6.2.7"},
		termSection{"resultSeverity", "3.6.2.8"}, termSection{"sourceConstraint", "5.3.2"},
	)
	add(termRejected, reasonDataGraph,
		termSection{"shapesGraph", "3.3"}, termSection{"suggestedShapesGraph", "vocabulary only"},
		termSection{"entailment", "1.5"},
	)
	add(termRejected, reasonSPARQL,
		termSection{"sparql", "5.2"}, termSection{"SPARQLConstraint", "5.2"}, termSection{"select", "5.2"},
		termSection{"SPARQLConstraintComponent", "5"}, termSection{"prefixes", "5.2.1"},
		termSection{"declare", "5.2.1"}, termSection{"PrefixDeclaration", "5.2.1"}, termSection{"prefix", "5.2.1"},
		termSection{"namespace", "5.2.1"}, termSection{"this", "5.3.1"},
		termSection{"ConstraintComponent", "6.2"}, termSection{"Parameterizable", "vocabulary only"},
		termSection{"Parameter", "6.2.1"}, termSection{"parameter", "6.2.1"}, termSection{"optional", "6.2.1"},
		termSection{"labelTemplate", "6.2.2"}, termSection{"Validator", "6.2.3"}, termSection{"validator", "6.2.3"},
		termSection{"nodeValidator", "6.2.3"}, termSection{"propertyValidator", "6.2.3"},
		termSection{"SPARQLSelectValidator", "6.2.3.1"}, termSection{"SPARQLAskValidator", "6.2.3.2"},
		termSection{"ask", "6.2.3.2"}, termSection{"SPARQLExecutable", "vocabulary only"},
		termSection{"SPARQLAskExecutable", "vocabulary only"},
		termSection{"SPARQLSelectExecutable", "vocabulary only"},
		termSection{"SPARQLConstructExecutable", "vocabulary only"},
		termSection{"SPARQLUpdateExecutable", "vocabulary only"}, termSection{"update", "vocabulary only"},
	)
	add(termRejected, reasonJS,
		termSection{"JSExecutable", "JS 4"}, termSection{"jsFunctionName", "JS 4.1"},
		termSection{"jsLibrary", "JS 4.2"}, termSection{"JSLibrary", "JS 4.3"}, termSection{"jsLibraryURL", "JS 4.3"},
		termSection{"JSConstraint", "JS 6"}, termSection{"js", "JS 6"}, termSection{"JSConstraintComponent", "JS 7"},
		termSection{"JSValidator", "JS 7.2"}, termSection{"JSFunction", "JS 8"}, termSection{"JSRule", "JS 9"},
		termSection{"JSTarget", "JS 10.1"}, termSection{"JSTargetType", "JS 10.2"},
	)
	add(termRejected, reasonAF,
		termSection{"target", "AF 3"}, termSection{"Target", "AF 3"}, termSection{"SPARQLTarget", "AF 3.1"},
		termSection{"TargetType", "AF 3.2"}, termSection{"SPARQLTargetType", "AF 3.2"},
		termSection{"resultAnnotation", "AF 4"}, termSection{"ResultAnnotation", "AF 4"},
		termSection{"annotationProperty", "AF 4"}, termSection{"annotationValue", "AF 4"},
		termSection{"annotationVarName", "AF 4"}, termSection{"Function", "AF 5"},
		termSection{"returnType", "AF 5.3"}, termSection{"SPARQLFunction", "AF 5.4"},
		termSection{"filterShape", "AF 6.3"}, termSection{"nodes", "AF 6.3"},
		termSection{"intersection", "AF 6.6"}, termSection{"union", "AF 6.7"},
		termSection{"expression", "AF 7"}, termSection{"ExpressionConstraintComponent", "AF 7"},
	)
	add(termRejected, reasonRule,
		termSection{"rule", "AF 8"}, termSection{"Rule", "AF 8"}, termSection{"condition", "AF 8.2.1"},
		termSection{"TripleRule", "AF 8.5"}, termSection{"subject", "AF 8.5"}, termSection{"predicate", "AF 8.5"},
		termSection{"object", "AF 8.5"}, termSection{"SPARQLRule", "AF 8.6"}, termSection{"construct", "AF 8.6"},
	)
	return p
}

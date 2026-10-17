// Package shacl compiles SHACL shapes in a bounded profile, ahead of time,
// into a SemEngine pack, projection contracts, and a native validator for
// complete entity state (issue #1 Phase 2; design in
// openspec/changes/phase-2-shacl-compilation). No RDF or SHACL engine runs in
// SemEngine's ingest or query paths.
//
// This file holds the shape model and the mapping document. Shapes come from
// Turtle (ParseTurtle) or from SemVocab's own JSON and YAML form
// (DecodeShapesJSON, DecodeShapesYAML); all three build the same model. The
// model is SemVocab's serialization, not W3C SHACL or JSON-LD.
//
// Importing the package registers nothing, and it reads and writes no
// SemEngine registry.
package shacl

import "github.com/c360studio/semengine/pkg/projection/contract"

// ShapeSet is a decoded shapes document: its node shapes, sorted by IRI.
//
// Parsing never fails on a construct outside the compiler's profile (ruling
// Q26). It records the construct, with its shape and path, in the set, and
// Compile reports every such record together. Only ParseTurtle can record
// them; the JSON and YAML forms cannot express one.
type ShapeSet struct {
	NodeShapes []NodeShape
	rejections []rejection
}

// NodeShape is one node shape. ID is its IRI; a node shape is never a blank
// node, because a blank-node label depends on document order (design D3).
type NodeShape struct {
	ID                string
	TargetClasses     []string // sh:targetClass IRIs, sorted
	Properties        []PropertyShape
	Closed            bool     // sh:closed
	IgnoredProperties []string // sh:ignoredProperties IRIs, sorted
	Severity          string   // sh:severity IRI, empty when absent
	Messages          []string // sh:message strings, sorted
}

// PropertyShape is one property shape of a node shape. ID is its IRI, or
// empty for a blank node; a property shape's identity is its node shape plus
// its path (design D3). Properties are sorted by path, then ID.
type PropertyShape struct {
	ID       string
	Path     string // sh:path, one predicate IRI
	MinCount int    // sh:minCount; absent means 0
	MaxCount *int   // sh:maxCount; nil means unbounded, 0 prohibits values
	Datatype string // sh:datatype IRI, empty when absent

	// Numeric bounds keep their lexical form and datatype, so no bound passes
	// through float64 before compilation (design D8).
	MinInclusive *Literal
	MaxInclusive *Literal
	MinExclusive *Literal
	MaxExclusive *Literal

	Pattern  *string   // sh:pattern; nil when absent
	In       *[]string // sh:in members, sorted; nil when absent, empty for sh:in ()
	Severity string    // sh:severity IRI, empty when absent
	Messages []string  // sh:message strings, sorted
}

// Literal is an RDF literal by lexical form and datatype IRI.
type Literal struct {
	Lexical  string
	Datatype string
}

// rejection is one out-of-profile construct found while parsing: the
// construct's IRI (or the sh: term that introduces it), the shape it appears
// on (empty for a blank subject outside every shape), the property path
// (empty on a node shape), and a detail naming the offending value. Blank-node
// labels never appear, so records do not depend on document order.
type rejection struct {
	Construct, Shape, Path, Detail string
}

// Mapping binds shapes to a SemEngine pack (design D6). It is decoded
// strictly by DecodeMappingJSON and DecodeMappingYAML; Compile checks its
// rules.
type Mapping struct {
	EntityTypes             []EntityTypeBinding `json:"entity_types" yaml:"entity_types"`
	Predicates              []PredicateBinding  `json:"predicates" yaml:"predicates"`
	Contracts               []ContractBinding   `json:"contracts" yaml:"contracts"`
	ClosedIgnoredPredicates []string            `json:"closed_ignored_predicates" yaml:"closed_ignored_predicates"`
}

// EntityTypeBinding lists the pack entity types an sh:targetClass value
// targets.
type EntityTypeBinding struct {
	ClassIRI    string   `json:"class_iri" yaml:"class_iri"`
	DomainTypes []string `json:"domain_types" yaml:"domain_types"`
}

// PredicateBinding names the dotted predicate an sh:path value maps to.
type PredicateBinding struct {
	PathIRI string `json:"path_iri" yaml:"path_iri"`
	Name    string `json:"name" yaml:"name"`
}

// ContractBinding declares the projection contract of one entity type.
type ContractBinding struct {
	DomainType string         `json:"domain_type" yaml:"domain_type"`
	Name       string         `json:"name" yaml:"name"`
	Birth      []string       `json:"birth" yaml:"birth"`
	Groups     []GroupBinding `json:"groups" yaml:"groups"`
}

// GroupBinding declares one predicate group and its write mode.
type GroupBinding struct {
	Name       string             `json:"name" yaml:"name"`
	Mode       contract.WriteMode `json:"mode" yaml:"mode"`
	Predicates []string           `json:"predicates" yaml:"predicates"`
}

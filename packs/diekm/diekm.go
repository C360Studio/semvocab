// Package diekm is the DIEKM vocabulary pack: the intel.platform and
// intel.observation entity types and the intel.role.assigned and
// intel.measurement.altitude_m predicates with their ontology projections,
// over BFO 2020 and CCO v2.2. diekm.yaml is the pack and diekm.shacl.ttl its
// initial SHACL shapes; both are embedded.
//
// https://w3id.org/c360/diekm# is a C360-local vocabulary. It is not an
// externally registered or certified ontology, and its IRIs do not
// dereference. Provenance and the owner rulings the pack applies are recorded
// in docs/admission-ledger.yaml (pack "diekm").
//
// The pack does not yet pass SemEngine's validation: the predicate name
// intel.measurement.altitude_m, carried from the semvocab#1 fixture, breaks
// SemEngine's lower-kebab segment grammar ([a-z][a-z0-9-]*), so
// vocabulary.RegisterPack rejects the pack. This package therefore exports no
// registration function until the name is ruled on (semvocab#1). Importing it
// registers nothing.
package diekm

import (
	"bytes"
	_ "embed" // embeds diekm.yaml and diekm.shacl.ttl
	"fmt"

	"github.com/c360studio/semengine/vocabulary"
)

var (
	//go:embed diekm.yaml
	packYAML []byte
	//go:embed diekm.shacl.ttl
	shapesTurtle []byte
)

// Spec decodes the embedded pack with SemEngine's strict YAML loader. It
// registers nothing and does not validate: (*vocabulary.PackSpec).Validate
// currently fails on the pack (see the package comment).
func Spec() (vocabulary.PackSpec, error) {
	spec, err := vocabulary.LoadPackYAML(bytes.NewReader(packYAML))
	if err != nil {
		return vocabulary.PackSpec{}, fmt.Errorf("diekm: %w", err)
	}
	return spec, nil
}

// Shapes returns a copy of the embedded SHACL shapes (Turtle). Phase 2 of
// semvocab#1 compiles them; nothing in SemVocab evaluates them yet.
func Shapes() []byte {
	return bytes.Clone(shapesTurtle)
}

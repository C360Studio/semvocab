// Package diekm is the DIEKM vocabulary pack: the intel.platform and
// intel.observation entity types and the intel.role.assigned and
// intel.measurement.altitude-m predicates with their ontology projections,
// over BFO 2020 and CCO v2.2. diekm.yaml is the pack and diekm.shacl.ttl its
// initial SHACL shapes; both are embedded.
//
// https://w3id.org/c360/diekm# is a C360-local vocabulary. It is not an
// externally registered or certified ontology, and its IRIs do not
// dereference. Provenance and the owner rulings the pack applies are recorded
// in docs/admission-ledger.yaml (pack "diekm").
//
// The altitude predicate is intel.measurement.altitude-m (owner ruling Q11 on
// semvocab#1); the issue fixture's altitude_m breaks SemEngine's predicate
// segment grammar [a-z][a-z0-9-]*.
//
// Importing this package registers nothing; a composition root calls
// Register.
package diekm

import (
	"bytes"
	_ "embed" // embeds diekm.yaml and diekm.shacl.ttl
	"fmt"

	"github.com/c360studio/semengine/vocabulary"
	"github.com/c360studio/semengine/vocabulary/export"
)

var (
	//go:embed diekm.yaml
	packYAML []byte
	//go:embed diekm.shacl.ttl
	shapesTurtle []byte
)

// Spec decodes the embedded pack with SemEngine's strict YAML loader. It
// registers nothing.
func Spec() (vocabulary.PackSpec, error) {
	spec, err := vocabulary.LoadPackYAML(bytes.NewReader(packYAML))
	if err != nil {
		return vocabulary.PackSpec{}, fmt.Errorf("diekm: %w", err)
	}
	return spec, nil
}

// Register registers the DIEKM pack through vocabulary.RegisterPack with
// vocabulary.WithPrefixRegistrar(export.RegisterPrefixes), followed by opts; a
// later WithPrefixRegistrar in opts replaces that default. On error,
// RegisterPack leaves SemEngine's registries and serializer prefixes
// unchanged. Registering it again leaves state equal; a conflicting later
// registration follows RegisterPack (scalar fields last-writer-wins,
// superclasses unioned), and rebinding one of its prefixes fails.
func Register(opts ...vocabulary.PackOption) error {
	spec, err := Spec()
	if err != nil {
		return err
	}
	all := append([]vocabulary.PackOption{vocabulary.WithPrefixRegistrar(export.RegisterPrefixes)}, opts...)
	if err := vocabulary.RegisterPack(spec, all...); err != nil {
		return fmt.Errorf("diekm: register: %w", err)
	}
	return nil
}

// Shapes returns a copy of the embedded SHACL shapes (Turtle). Phase 2 of
// semvocab#1 compiles them; nothing in SemVocab evaluates them yet.
func Shapes() []byte {
	return bytes.Clone(shapesTurtle)
}

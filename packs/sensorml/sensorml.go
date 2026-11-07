// Package sensorml is SemVocab's SensorML 3.0 and SOSA/SSN vocabulary pack for
// OGC API — Connected Systems Part 1 (OGC 23-001) resources: 5 csapi.* entity
// types (csapi.system, csapi.procedure, csapi.deployment,
// csapi.samplingfeature, csapi.property), 20 sensorml.* predicates, 2 1-step
// ProjectionSpec chains (sensorml.capability.value via sml:CapabilityList and
// sensorml.characteristic.value via sml:CharacteristicList), 5 open SHACL node
// shapes, and 5 reconcile projection contracts over W3C SOSA/SSN 2017, OGC SWE
// Common 3.0, and OGC SensorML 3.0 (owner rulings R4 and R5 on semvocab#26).
// sensorml.yaml is the pack, sensorml.shacl.ttl its SHACL shapes, and
// sensorml.mapping.yaml the mapping that binds the shapes to the pack for the
// shacl compiler; all three are embedded, and Working-Draft OGC CS API Part 2
// csapi: IRIs are excluded.
//
// Namespace honesty (owner ruling R3 on semvocab#26): the swe
// (http://www.opengis.net/spec/SWE/3.0/) and sml
// (http://www.opengis.net/spec/sensorML/3.0/) prefixes bind unregistered C360
// RDF conventions over OGC 24-014 and OGC 23-000 Clause 5 normative
// specification base URIs; they are not externally certified or dereferenceable
// OGC OWL ontologies. Provenance and owner rulings are recorded in
// docs/admission-ledger.yaml (entry "pack:sensorml").
//
// Importing this package registers nothing; a composition root calls Register.
package sensorml

import (
	"bytes"
	_ "embed" // embeds sensorml.yaml, sensorml.shacl.ttl, and sensorml.mapping.yaml
	"fmt"

	"github.com/c360studio/semengine/vocabulary"
	"github.com/c360studio/semengine/vocabulary/export"
)

var (
	//go:embed sensorml.yaml
	packYAML []byte
	//go:embed sensorml.shacl.ttl
	shapesTurtle []byte
	//go:embed sensorml.mapping.yaml
	mappingYAML []byte
)

// Spec decodes the embedded pack with SemEngine's strict YAML loader. It
// registers nothing.
func Spec() (vocabulary.PackSpec, error) {
	spec, err := vocabulary.LoadPackYAML(bytes.NewReader(packYAML))
	if err != nil {
		return vocabulary.PackSpec{}, fmt.Errorf("sensorml: %w", err)
	}
	return spec, nil
}

// Register registers the SensorML pack through vocabulary.RegisterPack with
// vocabulary.WithPrefixRegistrar(export.RegisterPrefixes), followed by opts; a
// later WithPrefixRegistrar in opts replaces that default. On error,
// RegisterPack leaves SemEngine's registries and serializer prefixes unchanged.
// Registering it again leaves state equal; a conflicting later registration
// follows RegisterPack (scalar fields last-writer-wins, superclasses unioned),
// and rebinding one of its prefixes fails.
func Register(opts ...vocabulary.PackOption) error {
	spec, err := Spec()
	if err != nil {
		return err
	}
	all := append([]vocabulary.PackOption{vocabulary.WithPrefixRegistrar(export.RegisterPrefixes)}, opts...)
	if err := vocabulary.RegisterPack(spec, all...); err != nil {
		return fmt.Errorf("sensorml: register: %w", err)
	}
	return nil
}

// Shapes returns a copy of the embedded SHACL shapes (Turtle), which
// shacl.ParseTurtle reads and shacl.Compile compiles with Mapping.
func Shapes() []byte {
	return bytes.Clone(shapesTurtle)
}

// Mapping returns a copy of the embedded mapping document (YAML) that binds
// Shapes to Spec: decode it with shacl.DecodeMappingYAML and pass it to
// shacl.Compile with Spec and the parsed Shapes.
func Mapping() []byte {
	return bytes.Clone(mappingYAML)
}

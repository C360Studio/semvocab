// Package packs holds SemVocab's built-in vocabulary packs that are not tied
// to one domain:
//
//   - bfo-cco-core (bfo-cco-core.yaml, embedded): the "obo" prefix for
//     http://purl.obolibrary.org/obo/ (BFO 2020) and the "cco" prefix for
//     https://www.commoncoreontologies.org/ (CCO v2.2), and nothing else.
//   - swe-core (swe-core.yaml, embedded): the "sosa", "ssn", and "ssn-system"
//     prefixes for the W3C 2017 SOSA/SSN ontologies and the "swe" and "sml"
//     prefixes for the C360 RDF convention stems over OGC SWE Common 3.0 and
//     SensorML 3.0 (owner ruling R3 on semvocab#26), and nothing else.
//
// Domain packs live in subpackages, such as packs/diekm.
//
// Importing this package registers nothing; a composition root calls
// RegisterBFOCCOCore or RegisterSWECore.
package packs

import (
	"bytes"
	_ "embed" // embeds bfo-cco-core.yaml and swe-core.yaml
	"fmt"

	"github.com/c360studio/semengine/vocabulary"
	"github.com/c360studio/semengine/vocabulary/export"
)

//go:embed bfo-cco-core.yaml
var bfoCCOCoreYAML []byte

//go:embed swe-core.yaml
var sweCoreYAML []byte

// BFOCCOCoreSpec decodes the embedded bfo-cco-core pack with SemEngine's
// strict YAML loader. It registers nothing.
func BFOCCOCoreSpec() (vocabulary.PackSpec, error) {
	spec, err := vocabulary.LoadPackYAML(bytes.NewReader(bfoCCOCoreYAML))
	if err != nil {
		return vocabulary.PackSpec{}, fmt.Errorf("packs: bfo-cco-core: %w", err)
	}
	return spec, nil
}

// RegisterBFOCCOCore registers the bfo-cco-core pack through
// vocabulary.RegisterPack with vocabulary.WithPrefixRegistrar(export.RegisterPrefixes),
// followed by opts; a later WithPrefixRegistrar in opts replaces that default.
// On error, RegisterPack leaves SemEngine's registries and serializer prefixes
// unchanged. Registering it again is a no-op; registering it after another
// pack bound "obo" or "cco" to a different namespace fails.
func RegisterBFOCCOCore(opts ...vocabulary.PackOption) error {
	spec, err := BFOCCOCoreSpec()
	if err != nil {
		return err
	}
	all := append([]vocabulary.PackOption{vocabulary.WithPrefixRegistrar(export.RegisterPrefixes)}, opts...)
	if err := vocabulary.RegisterPack(spec, all...); err != nil {
		return fmt.Errorf("packs: register bfo-cco-core: %w", err)
	}
	return nil
}

// SWECoreSpec decodes the embedded swe-core pack with SemEngine's strict YAML
// loader. It registers nothing.
func SWECoreSpec() (vocabulary.PackSpec, error) {
	spec, err := vocabulary.LoadPackYAML(bytes.NewReader(sweCoreYAML))
	if err != nil {
		return vocabulary.PackSpec{}, fmt.Errorf("packs: swe-core: %w", err)
	}
	return spec, nil
}

// RegisterSWECore registers the swe-core pack through vocabulary.RegisterPack
// with vocabulary.WithPrefixRegistrar(export.RegisterPrefixes), followed by
// opts; a later WithPrefixRegistrar in opts replaces that default. On error,
// RegisterPack leaves SemEngine's registries and serializer prefixes
// unchanged. Registering it again is a no-op; registering it after another
// pack bound "sosa", "ssn", "ssn-system", "swe", or "sml" to a different
// namespace fails.
func RegisterSWECore(opts ...vocabulary.PackOption) error {
	spec, err := SWECoreSpec()
	if err != nil {
		return err
	}
	all := append([]vocabulary.PackOption{vocabulary.WithPrefixRegistrar(export.RegisterPrefixes)}, opts...)
	if err := vocabulary.RegisterPack(spec, all...); err != nil {
		return fmt.Errorf("packs: register swe-core: %w", err)
	}
	return nil
}

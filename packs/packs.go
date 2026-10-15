// Package packs holds SemVocab's built-in vocabulary packs that are not tied
// to one domain. Today that is bfo-cco-core (bfo-cco-core.yaml, embedded): the
// "obo" prefix for http://purl.obolibrary.org/obo/ (BFO 2020) and the "cco"
// prefix for https://www.commoncoreontologies.org/ (CCO v2.2), and nothing
// else. Domain packs live in subpackages, such as packs/diekm.
//
// Importing this package registers nothing; a composition root calls
// RegisterBFOCCOCore.
package packs

import (
	"bytes"
	_ "embed" // embeds bfo-cco-core.yaml
	"fmt"

	"github.com/c360studio/semengine/vocabulary"
	"github.com/c360studio/semengine/vocabulary/export"
)

//go:embed bfo-cco-core.yaml
var bfoCCOCoreYAML []byte

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

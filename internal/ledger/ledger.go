// Package ledger decodes docs/admission-ledger.yaml for tests. The ledger is
// the single table of admitted terms (design P3 of phase-1-catalogs): the
// catalog and pack tests read their expected term lists from it instead of
// restating them. Decoding is strict, so a field the types below do not name
// fails with the field's name. Production code must not import this package.
package ledger

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

// Ledger is the decoded admission ledger.
type Ledger struct {
	SchemaVersion int                  `yaml:"schema_version"`
	Fields        map[string]string    `yaml:"fields"`
	Dispositions  map[string]string    `yaml:"dispositions"`
	Pins          Pins                 `yaml:"pins"`
	PhaseGates    map[string]PhaseGate `yaml:"phase_gates"`
	Entries       []Entry              `yaml:"entries"`
}

// Pins records the exact revisions every admitted term traces to.
type Pins struct {
	SemStreams struct {
		Repository string `yaml:"repository"`
		Revision   string `yaml:"revision"`
		License    string `yaml:"license"`
	} `yaml:"semstreams"`
	SemEngine struct {
		Module   string   `yaml:"module"`
		Version  string   `yaml:"version"`
		Revision string   `yaml:"revision"`
		Includes []string `yaml:"includes"`
		Excludes []string `yaml:"excludes"`
		License  string   `yaml:"license"`
		GoMod    string   `yaml:"go_mod"`
	} `yaml:"semengine"`
	Ontologies map[string]Ontology     `yaml:"ontologies"`
	Schemas    map[string]SchemaBundle `yaml:"schemas"`
	Fixtures   map[string]FixtureSet   `yaml:"fixtures"`
}

// SchemaBundle is one pinned external schema bundle (such as OGC Draft 2020-12
// JSON Schemas), with per-file sha256 digests.
type SchemaBundle struct {
	Repository    string        `yaml:"repository"`
	Tag           string        `yaml:"tag"`
	Commit        string        `yaml:"commit"`
	Standard      string        `yaml:"standard"`
	Edition       string        `yaml:"edition"`
	SpecBaseURI   string        `yaml:"spec_base_uri"`
	License       string        `yaml:"license"`
	VendoredAt    string        `yaml:"vendored_at"`
	ChosenBecause string        `yaml:"chosen_because"`
	Note          string        `yaml:"note"`
	Files         []FixtureFile `yaml:"files"`
}

// File returns the pinned schema file at path, by its path in the source
// repository.
func (b SchemaBundle) File(path string) (FixtureFile, bool) {
	for _, f := range b.Files {
		if f.Path == path {
			return f, true
		}
	}
	return FixtureFile{}, false
}

// FixtureSet is one pinned set of vendored external test fixtures: files
// copied unmodified from one repository at one commit.
type FixtureSet struct {
	Repository string        `yaml:"repository"`
	Commit     string        `yaml:"commit"`
	License    string        `yaml:"license"`
	VendoredAt string        `yaml:"vendored_at"`
	Selection  string        `yaml:"selection"`
	Files      []FixtureFile `yaml:"files"`
}

// FixtureFile is one vendored file, by its path in the source repository.
type FixtureFile struct {
	Path   string `yaml:"path"`
	SHA256 string `yaml:"sha256"`
}

// File returns the pinned file at path, by its path in the source repository.
func (s FixtureSet) File(path string) (FixtureFile, bool) {
	for _, f := range s.Files {
		if f.Path == path {
			return f, true
		}
	}
	return FixtureFile{}, false
}

// Ontology is one pinned external ontology release.
type Ontology struct {
	Repository    string `yaml:"repository"`
	Release       string `yaml:"release"`
	Tag           string `yaml:"tag"`
	Commit        string `yaml:"commit"`
	File          string `yaml:"file"`
	SHA256        string `yaml:"sha256"`
	VersionIRI    string `yaml:"version_iri"`
	Created       string `yaml:"created"`
	Namespace     string `yaml:"namespace"`
	DependsOn     string `yaml:"depends_on"`
	License       string `yaml:"license"`
	ChosenBecause string `yaml:"chosen_because"`
	Divergence    string `yaml:"divergence"`
	Note          string `yaml:"note"`
}

// PhaseGate is the dependency gate of one issue #1 phase.
type PhaseGate struct {
	Requires string `yaml:"requires"`
	Status   string `yaml:"status"`
}

// Entry is one candidate catalog or pack.
type Entry struct {
	Catalog                       string            `yaml:"catalog"`
	Pack                          string            `yaml:"pack"`
	Disposition                   string            `yaml:"disposition"`
	Phase                         *int              `yaml:"phase"`
	SemEngineOverlap              string            `yaml:"semengine_overlap"`
	Source                        yaml.Node         `yaml:"source"`
	SourceAvailable               string            `yaml:"source_available"`
	AdmittedTerms                 string            `yaml:"admitted_terms"`
	ExcludedSourceSymbols         []Exclusion       `yaml:"excluded_source_symbols"`
	HierarchyDivergenceFromSource string            `yaml:"hierarchy_divergence_from_source"`
	DeferredTerms                 map[string]string `yaml:"deferred_terms"`
	DeferralReason                string            `yaml:"deferral_reason"`
	License                       string            `yaml:"license"`
	TestOracle                    string            `yaml:"test_oracle"`
	Files                         []string          `yaml:"files"`
	Namespace                     PackNamespace     `yaml:"namespace"`
	References                    []string          `yaml:"references"`
	ReferencedTermsVerified       []string          `yaml:"referenced_terms_verified"`
	LocalAssertions               []string          `yaml:"local_assertions"`
	Rulings                       map[string]string `yaml:"rulings"`
	Contents                      string            `yaml:"contents"`
}

// Name is the entry's catalog or pack name.
func (e Entry) Name() string {
	if e.Catalog != "" {
		return e.Catalog
	}
	return e.Pack
}

// Exclusion is a group of source symbols deliberately not admitted.
type Exclusion struct {
	Symbols []string `yaml:"symbols"`
	Reason  string   `yaml:"reason"`
}

// PackNamespace records a pack's own namespace or a catalog's C360 RDF IRI
// convention namespace declaration.
type PackNamespace struct {
	IRI        string `yaml:"iri"`
	Authority  string `yaml:"authority"`
	Registered bool   `yaml:"registered"`
	Statement  string `yaml:"statement"`
}

// BFOSource is the source block of the bfo entry.
type BFOSource struct {
	SemStreamsFiles          []string `yaml:"semstreams_files"`
	SymbolsAdmittedClasses   []string `yaml:"symbols_admitted_classes"`
	SymbolsAdmittedRelations []string `yaml:"symbols_admitted_relations"`
	Hierarchy                string   `yaml:"hierarchy"`
	ExternalAdditions        struct {
		Ontology string   `yaml:"ontology"`
		Terms    []string `yaml:"terms"`
	} `yaml:"external_additions"`
}

// CCOSource is the source block of the cco entry. Admitted maps take a Go
// name to [v2.2 IRI local name, v2.2 rdfs:label].
type CCOSource struct {
	Ontology              string               `yaml:"ontology"`
	GoNameMapping         string               `yaml:"go_name_mapping"`
	AdmittedClasses       map[string][2]string `yaml:"admitted_classes"`
	AdmittedProperties    map[string][2]string `yaml:"admitted_properties"`
	RenamedFromSemStreams map[string]string    `yaml:"renamed_from_semstreams"`
	Hierarchy             string               `yaml:"hierarchy"`
}

// SOSASource is the source block of the sosa entry. Admitted maps take a Go
// name to [IRI local name, rdfs:label].
type SOSASource struct {
	Ontology                   string               `yaml:"ontology"`
	GoNameMapping              string               `yaml:"go_name_mapping"`
	AdmittedClasses            map[string][2]string `yaml:"admitted_classes"`
	AdmittedObjectProperties   map[string][2]string `yaml:"admitted_object_properties"`
	AdmittedDatatypeProperties map[string][2]string `yaml:"admitted_datatype_properties"`
	Hierarchy                  string               `yaml:"hierarchy"`
}

// SSNSource is the source block of the ssn entry (covering both ssn-2017 and
// ssn-system-2017). Admitted maps take a Go name to [IRI local name, rdfs:label].
type SSNSource struct {
	Ontologies                     []string             `yaml:"ontologies"`
	GoNameMapping                  string               `yaml:"go_name_mapping"`
	AdmittedClasses                map[string][2]string `yaml:"admitted_classes"`
	AdmittedObjectProperties       map[string][2]string `yaml:"admitted_object_properties"`
	AdmittedSystemClasses          map[string][2]string `yaml:"admitted_system_classes"`
	AdmittedSystemObjectProperties map[string][2]string `yaml:"admitted_system_object_properties"`
	Hierarchy                      string               `yaml:"hierarchy"`
}

// SWESource is the source block of the swe entry. AdmittedClasses maps a Go
// name to [JSON Schema local name, schema location, direct parent];
// AdmittedProperties maps a Prop<PascalCase> Go name to its JSON property key.
type SWESource struct {
	SchemaBundle       string               `yaml:"schema_bundle"`
	GoNameMapping      string               `yaml:"go_name_mapping"`
	AdmittedClasses    map[string][3]string `yaml:"admitted_classes"`
	AdmittedProperties map[string]string    `yaml:"admitted_properties"`
	Hierarchy          string               `yaml:"hierarchy"`
}

// SensorMLSource is the source block of the sensorml entry. AdmittedClasses
// maps a Go name to [JSON Schema local name, schema location, direct parent];
// AdmittedProperties maps a Prop<PascalCase> Go name to its JSON property key.
type SensorMLSource struct {
	SchemaBundle       string               `yaml:"schema_bundle"`
	GoNameMapping      string               `yaml:"go_name_mapping"`
	AdmittedClasses    map[string][3]string `yaml:"admitted_classes"`
	AdmittedProperties map[string]string    `yaml:"admitted_properties"`
	Hierarchy          string               `yaml:"hierarchy"`
}

// Path is the absolute path of docs/admission-ledger.yaml in this checkout.
func Path() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "docs", "admission-ledger.yaml")
}

// Load reads and strictly decodes the ledger at Path.
func Load() (Ledger, error) {
	data, err := os.ReadFile(Path())
	if err != nil {
		return Ledger{}, fmt.Errorf("read ledger: %w", err)
	}
	return Decode(data)
}

// Decode strictly decodes ledger YAML, rejecting unknown fields.
func Decode(data []byte) (Ledger, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var l Ledger
	if err := dec.Decode(&l); err != nil {
		return Ledger{}, fmt.Errorf("decode ledger: %w", err)
	}
	return l, nil
}

// Entry returns the entry named name (catalog or pack).
func (l Ledger) Entry(name string) (Entry, error) {
	for _, e := range l.Entries {
		if e.Name() == name {
			return e, nil
		}
	}
	return Entry{}, fmt.Errorf("ledger has no entry %q", name)
}

// BFO decodes the bfo entry's source block.
func (l Ledger) BFO() (BFOSource, error) {
	var s BFOSource
	return s, l.decodeSource("bfo", &s)
}

// CCO decodes the cco entry's source block.
func (l Ledger) CCO() (CCOSource, error) {
	var s CCOSource
	return s, l.decodeSource("cco", &s)
}

// SOSA decodes the sosa entry's source block.
func (l Ledger) SOSA() (SOSASource, error) {
	var s SOSASource
	return s, l.decodeSource("sosa", &s)
}

// SSN decodes the ssn entry's source block.
func (l Ledger) SSN() (SSNSource, error) {
	var s SSNSource
	return s, l.decodeSource("ssn", &s)
}

// SWE decodes the swe entry's source block.
func (l Ledger) SWE() (SWESource, error) {
	var s SWESource
	return s, l.decodeSource("swe", &s)
}

// SensorML decodes the sensorml entry's source block.
func (l Ledger) SensorML() (SensorMLSource, error) {
	var s SensorMLSource
	return s, l.decodeSource("sensorml", &s)
}

func (l Ledger) decodeSource(name string, out any) error {
	e, err := l.Entry(name)
	if err != nil {
		return err
	}
	if e.Source.IsZero() {
		return fmt.Errorf("ledger %s source: missing source block", name)
	}
	// Re-encode the node so the strict decoder applies to the source block too.
	raw, err := yaml.Marshal(&e.Source)
	if err != nil {
		return fmt.Errorf("ledger %s source: %w", name, err)
	}
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("ledger %s source: %w", name, err)
	}
	return nil
}

// Excluded returns every excluded source symbol of the named entry, with any
// IRI suffix removed ("PartOf BFO_0000050" yields "PartOf").
func (e Entry) Excluded() []string {
	var out []string
	for _, x := range e.ExcludedSourceSymbols {
		for _, s := range x.Symbols {
			if f := strings.Fields(s); len(f) > 0 {
				out = append(out, f[0])
			}
		}
	}
	return out
}

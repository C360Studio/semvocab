package ledger

import (
	"fmt"
	"maps"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// candidates is the closed list the catalog-provenance spec ("One disposition
// per candidate") requires, written here independently of the ledger.
var candidates = []string{
	"actr", "agnt", "bfo", "cco", "cyber", "dublincore", "foaf", "geosparql", "iana", "owl", "prov", "rdf",
	"rdfs", "schema", "skos", "sosa", "ssn", "swe", "sensorml", "diekm", "bfo-cco-core", "swe-core",
}

// typedSourceCatalogs name the catalogs whose admitted terms live in typed maps
// under Entry.Source rather than a prose Entry.AdmittedTerms string.
var typedSourceCatalogs = []string{"cco", "sosa", "ssn", "swe", "sensorml"}

var (
	hex40 = regexp.MustCompile(`^[0-9a-f]{40}$`)
	hex64 = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// checkShape returns every way l departs from the shape the catalog-provenance
// spec requires. An empty result means the shape holds.
func checkShape(l Ledger) []string {
	var bad []string
	add := func(format string, args ...any) { bad = append(bad, fmt.Sprintf(format, args...)) }

	checkEntries(l, add)
	checkPins(l, add)
	checkSources(l, add)
	for _, phase := range []string{"phase-1", "phase-2", "phase-3", "phase-4"} {
		g, ok := l.PhaseGates[phase]
		if !ok || g.Requires == "" || g.Status == "" {
			add("phase_gates.%s needs requires and status", phase)
		}
	}
	return bad
}

func checkEntries(l Ledger, add func(string, ...any)) {
	seen := map[string]int{}
	for _, e := range l.Entries {
		seen[e.Name()]++
		if (e.Catalog == "") == (e.Pack == "") {
			add("entry %q: exactly one of catalog or pack must be set", e.Name())
		}
		if _, ok := l.Dispositions[e.Disposition]; !ok {
			add("entry %q: disposition %q is not one of dispositions", e.Name(), e.Disposition)
		}
		switch e.Disposition {
		case "deferred":
			if strings.TrimSpace(e.DeferralReason) == "" {
				add("entry %q: deferred without deferral_reason", e.Name())
			}
		case "port", "extraction", "new-content":
			hasSource := !e.Source.IsZero() || len(e.References) > 0
			if !hasSource {
				add("entry %q: %s without source", e.Name(), e.Disposition)
			}
			hasTerms := e.AdmittedTerms != "" || e.Contents != "" || slices.Contains(typedSourceCatalogs, e.Name())
			if !hasTerms {
				add("entry %q: %s without admitted terms", e.Name(), e.Disposition)
			}
			if strings.TrimSpace(e.License) == "" {
				add("entry %q: %s without license", e.Name(), e.Disposition)
			}
			if strings.TrimSpace(e.TestOracle) == "" {
				add("entry %q: %s without test_oracle", e.Name(), e.Disposition)
			}
		case "use-semengine-symbol", "thin-reexport":
			if strings.TrimSpace(e.SemEngineOverlap) == "" {
				add("entry %q: %s without semengine_overlap", e.Name(), e.Disposition)
			}
		}
		for _, ref := range e.References {
			_, inOnt := l.Pins.Ontologies[ref]
			_, inSch := l.Pins.Schemas[ref]
			if !inOnt && !inSch {
				add("entry %q: references unpinned key %q", e.Name(), ref)
			}
		}
	}
	for _, c := range candidates {
		if seen[c] != 1 {
			add("candidate %q has %d entries, want 1", c, seen[c])
		}
	}
	for _, name := range slices.Sorted(maps.Keys(seen)) {
		if !slices.Contains(candidates, name) {
			add("entry %q is not a candidate", name)
		}
	}
	for _, name := range []string{"swe", "sensorml"} {
		e, err := l.Entry(name)
		if err != nil {
			continue
		}
		if strings.TrimSpace(e.Namespace.IRI) == "" || strings.TrimSpace(e.Namespace.Authority) == "" ||
			strings.TrimSpace(e.Namespace.Statement) == "" {
			add("entry %q: namespace needs iri, authority, and statement", name)
		}
		if e.Namespace.Registered {
			add("entry %q: namespace.registered must be false", name)
		}
	}
}

func checkPins(l Ledger, add func(string, ...any)) {
	if !hex40.MatchString(l.Pins.SemStreams.Revision) {
		add("pins.semstreams.revision %q is not a full 40-hex revision", l.Pins.SemStreams.Revision)
	}
	if l.Pins.SemEngine.Version == "" || !hex40.MatchString(l.Pins.SemEngine.Revision) {
		add("pins.semengine needs a version and a full 40-hex revision")
	}
	for _, name := range []string{"bfo-2020", "cco-2.2", "sosa-2017", "ssn-2017", "ssn-system-2017"} {
		if _, ok := l.Pins.Ontologies[name]; !ok {
			add("pins.ontologies.%s missing", name)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(l.Pins.Ontologies)) {
		o := l.Pins.Ontologies[name]
		if o.Repository == "" || o.File == "" || o.Namespace == "" || o.License == "" ||
			(o.Tag == "" && o.Release == "") || (o.VersionIRI == "" && o.Created == "") {
			add("pins.ontologies.%s lacks repository, tag/release, file, version_iri/created, namespace, or license", name)
		}
		if !hex40.MatchString(o.Commit) {
			add("pins.ontologies.%s.commit %q is not a full 40-hex commit", name, o.Commit)
		}
		if !hex64.MatchString(o.SHA256) {
			add("pins.ontologies.%s.sha256 %q is not a 64-hex digest", name, o.SHA256)
		}
	}
	for _, name := range []string{"swe-3.0", "sensorml-3.0"} {
		if _, ok := l.Pins.Schemas[name]; !ok {
			add("pins.schemas.%s missing", name)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(l.Pins.Schemas)) {
		b := l.Pins.Schemas[name]
		if b.Repository == "" || b.Tag == "" || b.Standard == "" || b.Edition == "" ||
			b.SpecBaseURI == "" || b.License == "" || b.VendoredAt == "" {
			add("pins.schemas.%s lacks repository, tag, standard, edition, spec_base_uri, license, or vendored_at", name)
		}
		if !hex40.MatchString(b.Commit) {
			add("pins.schemas.%s.commit %q is not a full 40-hex commit", name, b.Commit)
		}
		checkPinnedFiles("pins.schemas."+name, b.Files, add)
	}
	if _, ok := l.Pins.Fixtures["w3c-data-shapes"]; !ok {
		add("pins.fixtures.w3c-data-shapes missing")
	}
	for _, name := range slices.Sorted(maps.Keys(l.Pins.Fixtures)) {
		set := l.Pins.Fixtures[name]
		if set.Repository == "" || set.License == "" || set.VendoredAt == "" || set.Selection == "" {
			add("pins.fixtures.%s lacks repository, license, vendored_at, or selection", name)
		}
		if !hex40.MatchString(set.Commit) {
			add("pins.fixtures.%s.commit %q is not a full 40-hex commit", name, set.Commit)
		}
		checkPinnedFiles("pins.fixtures."+name, set.Files, add)
	}
}

func checkPinnedFiles(prefix string, files []FixtureFile, add func(string, ...any)) {
	if len(files) == 0 {
		add("%s lists no files", prefix)
	}
	paths := map[string]bool{}
	for i, f := range files {
		if f.Path == "" {
			add("%s.files[%d] has no path", prefix, i)
		}
		if paths[f.Path] {
			add("%s lists %s twice", prefix, f.Path)
		}
		paths[f.Path] = true
		if !hex64.MatchString(f.SHA256) {
			add("%s file %s sha256 %q is not a 64-hex digest", prefix, f.Path, f.SHA256)
		}
	}
}

func checkSources(l Ledger, add func(string, ...any)) {
	checkOntPin := func(entry, key string) {
		if _, ok := l.Pins.Ontologies[key]; !ok {
			add("ledger %s source: unpinned ontology %q", entry, key)
		}
	}
	checkSchemaPin := func(entry, key string) {
		if _, ok := l.Pins.Schemas[key]; !ok {
			add("ledger %s source: unpinned schema_bundle %q", entry, key)
		}
	}

	if bfo, err := l.BFO(); err != nil {
		add("%v", err)
	} else {
		if len(bfo.SymbolsAdmittedClasses) == 0 || len(bfo.SymbolsAdmittedRelations) == 0 {
			add("ledger bfo source: empty admitted term list")
		}
		if bfo.ExternalAdditions.Ontology != "" {
			checkOntPin("bfo", bfo.ExternalAdditions.Ontology)
		}
	}
	if cco, err := l.CCO(); err != nil {
		add("%v", err)
	} else {
		if len(cco.AdmittedClasses) == 0 || len(cco.AdmittedProperties) == 0 {
			add("ledger cco source: empty admitted term map")
		}
		checkOntPin("cco", cco.Ontology)
	}
	if sosa, err := l.SOSA(); err != nil {
		add("%v", err)
	} else {
		if len(sosa.AdmittedClasses) == 0 || len(sosa.AdmittedObjectProperties) == 0 || len(sosa.AdmittedDatatypeProperties) == 0 {
			add("ledger sosa source: empty admitted term map")
		}
		checkOntPin("sosa", sosa.Ontology)
	}
	if ssn, err := l.SSN(); err != nil {
		add("%v", err)
	} else {
		if len(ssn.Ontologies) == 0 || len(ssn.AdmittedClasses) == 0 || len(ssn.AdmittedObjectProperties) == 0 ||
			len(ssn.AdmittedSystemClasses) == 0 || len(ssn.AdmittedSystemObjectProperties) == 0 {
			add("ledger ssn source: empty admitted term map")
		}
		for _, ont := range ssn.Ontologies {
			checkOntPin("ssn", ont)
		}
	}
	if swe, err := l.SWE(); err != nil {
		add("%v", err)
	} else {
		if len(swe.AdmittedClasses) == 0 || len(swe.AdmittedProperties) == 0 {
			add("ledger swe source: empty admitted term map")
		}
		checkSchemaPin("swe", swe.SchemaBundle)
	}
	if sml, err := l.SensorML(); err != nil {
		add("%v", err)
	} else {
		if len(sml.AdmittedClasses) == 0 || len(sml.AdmittedProperties) == 0 {
			add("ledger sensorml source: empty admitted term map")
		}
		checkSchemaPin("sensorml", sml.SchemaBundle)
	}
}

func TestAdmissionLedgerShape(t *testing.T) {
	l, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, problem := range checkShape(l) {
		t.Error(problem)
	}

	// Spec scenarios (catalog-provenance), checked against the committed ledger.
	se := l.Pins.SemEngine
	if se.Version != "v0.1.0-alpha.2" || se.Revision != "11da4721c2233a8e2b5b1d6236a97a44e06abbb6" {
		t.Errorf("pins.semengine = %s %s, want v0.1.0-alpha.2 11da4721c2233a8e2b5b1d6236a97a44e06abbb6", se.Version, se.Revision)
	}
	if !slices.ContainsFunc(se.Includes, func(s string) bool { return strings.Contains(s, "semengine#256") }) {
		t.Errorf("pins.semengine.includes does not list semengine#256: %q", se.Includes)
	}
	if g := l.PhaseGates["phase-2"]; g.Status != "available" || !strings.Contains(g.Requires, "semengine#256") {
		t.Errorf("phase_gates.phase-2 = %+v, want semengine#256 with status available", g)
	}
	shapes := l.Pins.Fixtures["w3c-data-shapes"]
	if shapes.Repository != "https://github.com/w3c/data-shapes" ||
		shapes.Commit != "4cd469867a1cc6d2fff80e8a40723b26c695d67a" ||
		!strings.Contains(shapes.License, "W3C Software and Document License") {
		t.Errorf("pins.fixtures.w3c-data-shapes = %s %s %q, want w3c/data-shapes at 4cd46986 under the W3C Software and Document License",
			shapes.Repository, shapes.Commit, shapes.License)
	}
	// The vocabulary digest is owner ruling Q21 on #1, restated here independently of the ledger.
	if f, ok := shapes.File("shacl/shacl.ttl"); !ok ||
		f.SHA256 != "1a534caa9d3df5bbc5e5351002bda56ead9226fda0a617c165b218d8bd06b9af" {
		t.Errorf("pins.fixtures.w3c-data-shapes shacl/shacl.ttl = %+v (listed %v), want the Q21 sha256", f, ok)
	}
	if actr, err := l.Entry("actr"); err != nil || actr.Disposition != "deferred" {
		t.Errorf("actr entry = %+v, %v; want deferred", actr, err)
	}
	if bfo, err := l.Entry("bfo"); err != nil || !slices.Contains(bfo.Excluded(), "PrecedesTemporally") {
		t.Errorf("bfo entry does not exclude PrecedesTemporally (err %v)", err)
	}
	if diekm, err := l.Entry("diekm"); err != nil || diekm.Namespace.Registered ||
		!strings.Contains(diekm.Namespace.Statement, "C360-local") {
		t.Errorf("diekm namespace = %+v, %v; want registered false and a C360-local statement", diekm.Namespace, err)
	}

	assertSWEAndSOSAScenarios(t, l)
}

func assertSWEAndSOSAScenarios(t *testing.T, l Ledger) {
	t.Helper()

	sosaPin := l.Pins.Ontologies["sosa-2017"]
	if sosaPin.Repository != "https://github.com/w3c/sdw" ||
		sosaPin.Release != "REC-vocab-ssn-20171019" ||
		sosaPin.Commit != "cb31ee9de636a7101337815ba67ba9af4a7c1348" ||
		sosaPin.File != "ssn/integrated/sosa.ttl" ||
		sosaPin.SHA256 != "c70b0d1c843c0047e4c98e0a89392e46a05752ff6365bda7f7f55303905a0392" ||
		sosaPin.Created != "2017-04-17" ||
		sosaPin.Namespace != "http://www.w3.org/ns/sosa/" ||
		!strings.Contains(sosaPin.License, "W3C") || !strings.Contains(sosaPin.License, "OGC") {
		t.Errorf("pins.ontologies.sosa-2017 = %+v, want w3c/sdw REC-vocab-ssn-20171019 at cb31ee9d (2017-04-17)", sosaPin)
	}

	swePin := l.Pins.Schemas["swe-3.0"]
	if swePin.Repository != "https://github.com/opengeospatial/ogcapi-connected-systems" ||
		swePin.Tag != "v1.0.0" ||
		swePin.Commit != "8e03b236a049849f2ccc24b4fd9fdce5ff69bed2" ||
		swePin.Standard != "OGC 24-014" ||
		swePin.Edition != "3.0.0" ||
		swePin.SpecBaseURI != "http://www.opengis.net/spec/SWE/3.0" ||
		!strings.Contains(swePin.License, "OGC Software License") {
		t.Errorf("pins.schemas.swe-3.0 = %+v, want opengeospatial/ogcapi-connected-systems v1.0.0 OGC 24-014 3.0.0", swePin)
	}
	if f, ok := swePin.File("swecommon/schemas/json/Quantity.json"); !ok ||
		f.SHA256 != "dc23d3496ae02a6d1de756d441aa12f248e74a1ff3feafda826640af04115b52" {
		t.Errorf("pins.schemas.swe-3.0 Quantity.json = %+v (listed %v), want pinned sha256", f, ok)
	}

	for _, name := range []string{"sosa", "ssn", "swe", "sensorml", "swe-core"} {
		e, err := l.Entry(name)
		if err != nil || e.Disposition != "new-content" || e.Phase == nil || *e.Phase != 1 {
			t.Errorf("%s entry = %+v (err %v), want new-content in phase 1", name, e, err)
		}
	}

	for _, tc := range []struct {
		name, iri, baseURI string
	}{
		{"swe", "http://www.opengis.net/spec/SWE/3.0/", "http://www.opengis.net/spec/SWE/3.0"},
		{"sensorml", "http://www.opengis.net/spec/sensorML/3.0/", "http://www.opengis.net/spec/sensorML/3.0"},
	} {
		e, err := l.Entry(tc.name)
		if err != nil || e.Namespace.IRI != tc.iri || e.Namespace.Registered ||
			!strings.Contains(e.Namespace.Statement, "C360 RDF convention") ||
			!strings.Contains(e.Namespace.Statement, tc.baseURI) {
			t.Errorf("%s namespace = %+v (err %v), want %s registered=false with C360 RDF convention over %s",
				tc.name, e.Namespace, err, tc.iri, tc.baseURI)
		}
	}

	if sosa, err := l.Entry("sosa"); err != nil ||
		!slices.Contains(sosa.Excluded(), "SamplingFeature") ||
		!slices.Contains(sosa.Excluded(), "HasLocation") {
		t.Errorf("sosa exclusions = %v (err %v), want SamplingFeature and HasLocation excluded", sosa.Excluded(), err)
	}
	if swe, err := l.Entry("swe"); err != nil || !slices.Contains(swe.Excluded(), "NilValue") {
		t.Errorf("swe exclusions = %v (err %v), want NilValue excluded", swe.Excluded(), err)
	}
}

// TestAdmissionLedgerShapeSensitivity seeds each violation into a copy of the
// committed ledger and requires checkShape (or the strict decoder) to name it.
func TestAdmissionLedgerShapeSensitivity(t *testing.T) {
	raw, err := os.ReadFile(Path())
	if err != nil {
		t.Fatal(err)
	}
	base := string(raw)
	tests := []struct {
		name     string
		mutate   func(string) string
		wantText string
	}{
		{"remove an entry", func(s string) string {
			return strings.Replace(s, "  - catalog: iana\n", "  - catalog: iana-removed\n", 1)
		}, `candidate "iana" has 0 entries`},
		{"duplicate an entry", func(s string) string {
			return strings.Replace(s, "  - catalog: iana\n", "  - catalog: actr\n", 1)
		}, `candidate "actr" has 2 entries`},
		{"drop a deferral reason", func(s string) string {
			return strings.Replace(s, `deferral_reason: "no source; issue #1 does not name which IANA registry is meant"`, `deferral_reason: ""`, 1)
		}, `entry "iana": deferred without deferral_reason`},
		{"drop a test oracle", func(s string) string {
			return strings.Replace(s, `test_oracle: "loads and registers through SemEngine`, `test_oracle_x: "loads and registers through SemEngine`, 1)
		}, `field test_oracle_x not found`},
		{"blank a test oracle", func(s string) string {
			return strings.Replace(s, `test_oracle: "loads and registers through SemEngine v0.1.0-alpha.2; namespaces equal the ontology pins"`, `test_oracle: ""`, 1)
		}, `entry "bfo-cco-core": new-content without test_oracle`},
		{"unknown disposition", func(s string) string {
			return strings.Replace(s, "disposition: port\n", "disposition: borrowed\n", 1)
		}, `entry "bfo": disposition "borrowed"`},
		{"short revision", func(s string) string {
			return strings.Replace(s, "revision: 8b99efe9c66a4faa4fa509f9f62cc6bad8392128", "revision: 8b99efe", 1)
		}, "pins.semstreams.revision"},
		{"short sha256", func(s string) string {
			return strings.Replace(s, "sha256: 128f5fe1dab7ee804a3037a836813e5953c846eeeefee883a47d5db0b37a106b", "sha256: 128f5fe1", 1)
		}, "pins.ontologies.bfo-2020.sha256"},
		{"missing sosa-2017 created and version_iri", func(s string) string {
			return strings.Replace(s, "      created: \"2017-04-17\"\n      namespace: http://www.w3.org/ns/sosa/\n",
				"      namespace: http://www.w3.org/ns/sosa/\n", 1)
		}, "pins.ontologies.sosa-2017 lacks repository, tag/release, file, version_iri/created"},
		{"missing schema bundle swe-3.0", func(s string) string {
			return strings.Replace(s, "    swe-3.0:\n", "    swe-3.0-x:\n", 1)
		}, "pins.schemas.swe-3.0 missing"},
		{"missing required field in swe-3.0 schema bundle", func(s string) string {
			return strings.Replace(s, "      spec_base_uri: http://www.opengis.net/spec/SWE/3.0\n",
				"      spec_base_uri: \"\"\n", 1)
		}, "pins.schemas.swe-3.0 lacks repository, tag, standard, edition, spec_base_uri, license, or vendored_at"},
		{"short swe-3.0 commit", func(s string) string {
			return strings.Replace(s, "      commit: 8e03b236a049849f2ccc24b4fd9fdce5ff69bed2\n      standard: OGC 24-014\n",
				"      commit: 8e03b236\n      standard: OGC 24-014\n", 1)
		}, "pins.schemas.swe-3.0.commit"},
		{"short swe-3.0 file sha256", func(s string) string {
			return strings.Replace(s, "sha256: dc23d3496ae02a6d1de756d441aa12f248e74a1ff3feafda826640af04115b52",
				"sha256: dc23d349", 1)
		}, "pins.schemas.swe-3.0 file swecommon/schemas/json/Quantity.json sha256"},
		{"duplicate swe-3.0 file path", func(s string) string {
			return strings.Replace(s, "path: swecommon/schemas/json/QuantityRange.json",
				"path: swecommon/schemas/json/Quantity.json", 1)
		}, "pins.schemas.swe-3.0 lists swecommon/schemas/json/Quantity.json twice"},
		{"remove swe-3.0 files", func(s string) string {
			return regexp.MustCompile(`(?s)(    swe-3\.0:\n.*?)      files:\n.*?\n    sensorml-3\.0:`).
				ReplaceAllString(s, "${1}    sensorml-3.0:")
		}, "pins.schemas.swe-3.0 lists no files"},
		{"swe namespace registered true", func(s string) string {
			return strings.Replace(s,
				"      iri: http://www.opengis.net/spec/SWE/3.0/\n      authority: C360 (RDF convention over OGC 24-014 Clause 5 base URI)\n      registered: false",
				"      iri: http://www.opengis.net/spec/SWE/3.0/\n      authority: C360 (RDF convention over OGC 24-014 Clause 5 base URI)\n      registered: true", 1)
		}, `entry "swe": namespace.registered must be false`},
		{"swe blank namespace statement", func(s string) string {
			return strings.Replace(s,
				`statement: "A C360 RDF convention over OGC SWE Common 3.0's Clause 5 normative specification base URI (http://www.opengis.net/spec/SWE/3.0). OGC 24-014 publishes Draft 2020-12 JSON Schemas rather than an OWL ontology; these IRIs are not an externally certified or dereferenceable OGC OWL ontology (owner ruling R3 on #26)."`,
				`statement: ""`, 1)
		}, `entry "swe": namespace needs iri, authority, and statement`},
		{"unknown field in sosa source", func(s string) string {
			return strings.Replace(s, "      ontology: sosa-2017\n", "      ontology: sosa-2017\n      unknown_sosa_key: 1\n", 1)
		}, "field unknown_sosa_key not found"},
		{"unknown field in ssn source", func(s string) string {
			return strings.Replace(s, "      ontologies: [ssn-2017, ssn-system-2017]\n",
				"      ontologies: [ssn-2017, ssn-system-2017]\n      unknown_ssn_key: 1\n", 1)
		}, "field unknown_ssn_key not found"},
		{"unknown field in swe source", func(s string) string {
			return strings.Replace(s, "      schema_bundle: swe-3.0\n", "      schema_bundle: swe-3.0\n      unknown_swe_key: 1\n", 1)
		}, "field unknown_swe_key not found"},
		{"unknown field in sensorml source", func(s string) string {
			return strings.Replace(s, "      schema_bundle: sensorml-3.0\n",
				"      schema_bundle: sensorml-3.0\n      unknown_sensorml_key: 1\n", 1)
		}, "field unknown_sensorml_key not found"},
		{"empty admitted term map in sosa source", func(s string) string {
			return strings.Replace(s,
				"      admitted_datatype_properties: # Go name: [local name, rdfs:label]\n        HasSimpleResult: [hasSimpleResult, \"has simple result\"]\n        ResultTime: [resultTime, \"result time\"]\n",
				"      admitted_datatype_properties: {}\n", 1)
		}, "ledger sosa source: empty admitted term map"},
		{"unresolved ontology pin key in sosa source", func(s string) string {
			return strings.Replace(s, "      ontology: sosa-2017\n", "      ontology: sosa-9999\n", 1)
		}, `ledger sosa source: unpinned ontology "sosa-9999"`},
		{"unresolved ontology pin key in ssn source", func(s string) string {
			return strings.Replace(s, "      ontologies: [ssn-2017, ssn-system-2017]\n",
				"      ontologies: [ssn-2017, ssn-system-9999]\n", 1)
		}, `ledger ssn source: unpinned ontology "ssn-system-9999"`},
		{"unresolved schema_bundle pin key", func(s string) string {
			return strings.Replace(s, "      schema_bundle: swe-3.0\n", "      schema_bundle: swe-9.9\n", 1)
		}, `ledger swe source: unpinned schema_bundle "swe-9.9"`},
		{"unresolved references pin key", func(s string) string {
			return strings.Replace(s, "    references: [sosa-2017, ssn-2017, ssn-system-2017, swe-3.0, sensorml-3.0]\n",
				"    references: [sosa-2017, ssn-2017, ssn-system-2017, swe-9.9, sensorml-3.0]\n", 1)
		}, `entry "swe-core": references unpinned key "swe-9.9"`},
		{"drop a fixture license", func(s string) string {
			return strings.Replace(s, "      license: W3C Software and Document License", "      licence_removed: x\n      x: W3C Software and Document License", 1)
		}, "field licence_removed not found"},
		{"remove a fixture license", func(s string) string {
			return regexp.MustCompile(`(?m)^      license: W3C Software and Document License.*\n`).ReplaceAllString(s, "")
		}, "pins.fixtures.w3c-data-shapes lacks repository, license"},
		{"remove a fixture commit", func(s string) string {
			return strings.Replace(s, "      commit: 4cd469867a1cc6d2fff80e8a40723b26c695d67a\n", "", 1)
		}, `pins.fixtures.w3c-data-shapes.commit ""`},
		{"remove a fixture file digest", func(s string) string {
			return strings.Replace(s, "\n          sha256: 1a534caa9d3df5bbc5e5351002bda56ead9226fda0a617c165b218d8bd06b9af", "", 1)
		}, "file shacl/shacl.ttl sha256"},
		{"remove a fixture file path", func(s string) string {
			return strings.Replace(s, "path: shacl/shacl.ttl\n          sha256:", "sha256:", 1)
		}, "has no path"},
		{"remove the fixture files", func(s string) string {
			return regexp.MustCompile(`(?s)(    w3c-data-shapes:\n.*?)      files:\n.*?\n\nphase_gates:`).
				ReplaceAllString(s, "${1}\nphase_gates:")
		}, "pins.fixtures.w3c-data-shapes lists no files"},
		{"remove the fixture set", func(s string) string {
			return strings.Replace(s, "    w3c-data-shapes:\n", "    w3c-data-shapes-x:\n", 1)
		}, "pins.fixtures.w3c-data-shapes missing"},
		{"drop a phase gate", func(s string) string {
			return strings.Replace(s, "  phase-4: {", "  phase-9: {", 1)
		}, "phase_gates.phase-4"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutated := tt.mutate(base)
			if mutated == base {
				t.Fatal("mutation did not apply; the seed text is stale")
			}
			l, err := Decode([]byte(mutated))
			var got string
			if err != nil {
				got = err.Error()
			} else {
				got = strings.Join(checkShape(l), "\n")
			}
			if !strings.Contains(got, tt.wantText) {
				t.Fatalf("seeded violation not named; want %q in:\n%s", tt.wantText, got)
			}
		})
	}
}

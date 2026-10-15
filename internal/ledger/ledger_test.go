package ledger

import (
	"fmt"
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
	"rdfs", "schema", "skos", "sosa", "ssn", "diekm", "bfo-cco-core",
}

var (
	hex40 = regexp.MustCompile(`^[0-9a-f]{40}$`)
	hex64 = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// checkShape returns every way l departs from the shape the catalog-provenance
// spec requires. An empty result means the shape holds.
func checkShape(l Ledger) []string {
	var bad []string
	add := func(format string, args ...any) { bad = append(bad, fmt.Sprintf(format, args...)) }

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
			hasTerms := e.AdmittedTerms != "" || e.Contents != "" || e.Name() == "cco"
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
	}
	for _, c := range candidates {
		if seen[c] != 1 {
			add("candidate %q has %d entries, want 1", c, seen[c])
		}
	}
	for name := range seen {
		if !slices.Contains(candidates, name) {
			add("entry %q is not a candidate", name)
		}
	}

	if !hex40.MatchString(l.Pins.SemStreams.Revision) {
		add("pins.semstreams.revision %q is not a full 40-hex revision", l.Pins.SemStreams.Revision)
	}
	if l.Pins.SemEngine.Version == "" || !hex40.MatchString(l.Pins.SemEngine.Revision) {
		add("pins.semengine needs a version and a full 40-hex revision")
	}
	for _, name := range []string{"bfo-2020", "cco-2.2"} {
		o, ok := l.Pins.Ontologies[name]
		if !ok {
			add("pins.ontologies.%s missing", name)
			continue
		}
		if o.Repository == "" || o.File == "" || o.VersionIRI == "" || o.Namespace == "" || o.License == "" ||
			(o.Tag == "" && o.Release == "") {
			add("pins.ontologies.%s lacks repository, tag/release, file, version_iri, namespace, or license", name)
		}
		if !hex40.MatchString(o.Commit) {
			add("pins.ontologies.%s.commit %q is not a full 40-hex commit", name, o.Commit)
		}
		if !hex64.MatchString(o.SHA256) {
			add("pins.ontologies.%s.sha256 %q is not a 64-hex digest", name, o.SHA256)
		}
	}
	for _, phase := range []string{"phase-1", "phase-2", "phase-3", "phase-4"} {
		g, ok := l.PhaseGates[phase]
		if !ok || g.Requires == "" || g.Status == "" {
			add("phase_gates.%s needs requires and status", phase)
		}
	}
	return bad
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

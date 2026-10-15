package packs_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/c360studio/semengine/vocabulary"
	"github.com/c360studio/semvocab/catalog/bfo"
	"github.com/c360studio/semvocab/catalog/cco"
	"github.com/c360studio/semvocab/internal/ledger"
	"github.com/c360studio/semvocab/internal/oracle"
	"github.com/c360studio/semvocab/packs"
	"github.com/c360studio/semvocab/packs/diekm"
)

const rdfsLabel = "http://www.w3.org/2000/01/rdf-schema#label"

// declaredParents are each entity type's declared parents: the issue #1
// fixture's superclasses with the Phase 0 design's corrections (CCO v2.2
// IRIs), excluding the DIEKM-local assertion, which comes from the ledger.
var declaredParents = map[string]map[string][]string{
	"diekm": {
		"intel.platform":    {cco.Vehicle},
		"intel.observation": {cco.ActOfObservation, bfo.Process},
	},
	"bfo-cco-core": {},
}

type embeddedPack struct {
	name string
	spec func() (vocabulary.PackSpec, error)
}

var embedded = []embeddedPack{
	{"bfo-cco-core", packs.BFOCCOCoreSpec},
	{"diekm", diekm.Spec},
}

// expand turns a ledger CURIE (cco:…, obo:…, diekm:…, rdfs:…) into an IRI
// using the pack's own prefixes.
func expand(spec vocabulary.PackSpec, curie string) (string, error) {
	prefix, local, ok := strings.Cut(curie, ":")
	if !ok {
		return "", fmt.Errorf("not a CURIE: %q", curie)
	}
	for _, p := range spec.Prefixes {
		if p.Prefix == prefix {
			return p.Namespace + local, nil
		}
	}
	return "", fmt.Errorf("CURIE %q uses a prefix the pack does not declare", curie)
}

// localAssertions parses the ledger's "<class> rdfs:subClassOf <parent> (…)"
// lines into class IRI -> parent IRIs.
func localAssertions(spec vocabulary.PackSpec, e ledger.Entry) (map[string][]string, error) {
	out := map[string][]string{}
	for _, a := range e.LocalAssertions {
		f := strings.Fields(a)
		if len(f) < 3 || f[1] != "rdfs:subClassOf" {
			return nil, fmt.Errorf("unparsed local assertion %q", a)
		}
		class, err := expand(spec, f[0])
		if err != nil {
			return nil, err
		}
		parent, err := expand(spec, f[2])
		if err != nil {
			return nil, err
		}
		out[class] = append(out[class], parent)
	}
	return out, nil
}

// packIRIs lists every IRI the pack carries, with where it occurs.
func packIRIs(spec vocabulary.PackSpec) map[string]string {
	out := map[string]string{}
	for i, et := range spec.EntityTypes {
		out[fmt.Sprintf("entity_types[%d].class_iri", i)] = et.ClassIRI
		for j, sc := range et.Superclasses {
			out[fmt.Sprintf("entity_types[%d].superclasses[%d]", i, j)] = sc
		}
	}
	for i, p := range spec.Predicates {
		if p.IRI != "" {
			out[fmt.Sprintf("predicates[%d].iri", i)] = p.IRI
		}
		if p.Projection == nil {
			continue
		}
		for j, s := range p.Projection.Steps {
			out[fmt.Sprintf("predicates[%d].projection.steps[%d].predicate_iri", i, j)] = s.PredicateIRI
			if s.NodeClassIRI != "" {
				out[fmt.Sprintf("predicates[%d].projection.steps[%d].node_class_iri", i, j)] = s.NodeClassIRI
			}
			if s.ValuePredicateIRI != "" {
				out[fmt.Sprintf("predicates[%d].projection.steps[%d].value_predicate_iri", i, j)] = s.ValuePredicateIRI
			}
		}
	}
	return out
}

// closureProblems checks one pack: full closures (D5) and expanded in-namespace
// IRIs (D6), each IRI evidenced by the release file or the ledger.
func closureProblems(name string, spec vocabulary.PackSpec, l ledger.Ledger, release *oracle.Graph) []string {
	var bad []string
	add := func(format string, args ...any) { bad = append(bad, fmt.Sprintf(name+": "+format, args...)) }

	// Prefix namespaces equal the ontology pins and the catalogs.
	pins := map[string]string{
		"obo": l.Pins.Ontologies["bfo-2020"].Namespace,
		"cco": l.Pins.Ontologies["cco-2.2"].Namespace,
	}
	if pins["obo"] != bfo.Namespace || pins["cco"] != cco.Namespace {
		add("ledger pins %v disagree with catalog namespaces %s, %s", pins, bfo.Namespace, cco.Namespace)
	}
	if e, err := l.Entry("diekm"); err == nil {
		pins["diekm"] = e.Namespace.IRI
	}
	for _, p := range spec.Prefixes {
		if want, ok := pins[p.Prefix]; ok && p.Namespace != want {
			add("prefix %s = %s, ledger pin %s", p.Prefix, p.Namespace, want)
		}
		if p.Prefix == "bfo" {
			add("prefix bfo declared; design P5 names the OBO namespace obo")
		}
	}

	entry, err := l.Entry(name)
	if err != nil {
		add("%v", err)
		return bad
	}
	local, err := localAssertions(spec, entry)
	if err != nil {
		add("%v", err)
	}

	// D5: superclasses equal the closure of declared parents plus local assertions.
	for _, et := range spec.EntityTypes {
		declared, ok := declaredParents[name][et.DomainType]
		if !ok {
			add("entity type %s has no declared parents on record in this test", et.DomainType)
			continue
		}
		roots := append(slices.Clone(declared), local[et.ClassIRI]...)
		var want []string
		for _, r := range roots {
			for _, iri := range append([]string{r}, release.Closure(r)...) {
				if !slices.Contains(want, iri) {
					want = append(want, iri)
				}
			}
		}
		got := slices.Clone(et.Superclasses)
		slices.Sort(got)
		if len(slices.Compact(got)) != len(et.Superclasses) {
			add("%s superclasses repeat an IRI: %v", et.DomainType, et.Superclasses)
		}
		slices.Sort(want)
		if !slices.Equal(got, want) {
			add("%s superclasses %v, want closure %v", et.DomainType, et.Superclasses, want)
		}
	}

	// D6: every IRI is expanded http(s) in a declared namespace, and evidenced.
	var verified []string
	for _, c := range entry.ReferencedTermsVerified {
		iri, err := expand(spec, c)
		if err != nil {
			add("ledger referenced term: %v", err)
			continue
		}
		verified = append(verified, iri)
	}
	diekmNS := pins["diekm"]
	for where, iri := range packIRIs(spec) {
		if !strings.HasPrefix(iri, "http://") && !strings.HasPrefix(iri, "https://") {
			add("%s = %q is not an expanded http(s) IRI", where, iri)
			continue
		}
		if !slices.ContainsFunc(spec.Prefixes, func(p vocabulary.PrefixSpec) bool {
			return strings.HasPrefix(iri, p.Namespace) && len(iri) > len(p.Namespace)
		}) {
			add("%s = %q lies in no declared prefix namespace", where, iri)
		}
		switch {
		case strings.HasPrefix(iri, diekmNS):
			if !strings.Contains(entry.AdmittedTerms, "diekm:"+strings.TrimPrefix(iri, diekmNS)) {
				add("%s = %q is not a ledger-admitted DIEKM term", where, iri)
			}
		case !slices.Contains(verified, iri):
			add("%s = %q is not in the ledger's referenced_terms_verified", where, iri)
		case iri != rdfsLabel && (len(release.Types(iri)) == 0 || release.Deprecated(iri)):
			add("%s = %q is not declared, or is deprecated, in the CCO v2.2 file", where, iri)
		}
	}
	return bad
}

func loadOracles(t *testing.T) (ledger.Ledger, *oracle.Graph) {
	t.Helper()
	l, err := ledger.Load()
	if err != nil {
		t.Fatal(err)
	}
	release, err := oracle.Load(oracle.CCO22)
	if err != nil {
		t.Fatal(err)
	}
	return l, release
}

func TestPackSuperclassClosure(t *testing.T) {
	l, release := loadOracles(t)
	for _, p := range embedded {
		spec, err := p.spec()
		if err != nil {
			t.Fatal(err)
		}
		if spec.Name != p.name {
			t.Errorf("embedded pack name %q, want %q", spec.Name, p.name)
		}
		for _, problem := range closureProblems(p.name, spec, l, release) {
			t.Error(problem)
		}
	}

	// Spec scenario: intel.platform lists the Vehicle closure plus BFO_0000030.
	spec, err := diekm.Spec()
	if err != nil {
		t.Fatal(err)
	}
	platform := spec.EntityTypes[slices.IndexFunc(spec.EntityTypes, func(e vocabulary.EntityTypeSpec) bool {
		return e.DomainType == "intel.platform"
	})]
	if len(platform.Superclasses) != 7 || !slices.Contains(platform.Superclasses, bfo.Object) {
		t.Errorf("intel.platform superclasses %v, want 7 IRIs including %s", platform.Superclasses, bfo.Object)
	}
}

// TestPackSuperclassClosureSensitivity seeds a missing ancestor, a CURIE in
// place of an IRI, and an IRI outside every declared namespace into the DIEKM
// pack and requires closureProblems to name each.
func TestPackSuperclassClosureSensitivity(t *testing.T) {
	l, release := loadOracles(t)
	tests := []struct {
		name   string
		mutate func(*vocabulary.PackSpec)
		want   string
	}{
		{"missing ancestor", func(s *vocabulary.PackSpec) {
			s.EntityTypes[0].Superclasses = slices.DeleteFunc(s.EntityTypes[0].Superclasses,
				func(iri string) bool { return iri == bfo.IndependentContinuant })
		}, "intel.platform superclasses"},
		{"CURIE superclass", func(s *vocabulary.PackSpec) {
			s.EntityTypes[0].Superclasses[0] = "cco:ont00000713"
		}, `is not an expanded http(s) IRI`},
		{"undeclared namespace", func(s *vocabulary.PackSpec) {
			s.Predicates[0].Projection.Steps[0].NodeClassIRI = "http://example.org/Role"
		}, "lies in no declared prefix namespace"},
		{"deprecated term", func(s *vocabulary.PackSpec) {
			s.Predicates[1].Projection.Steps[2].ValuePredicateIRI = cco.Namespace + "ont00001769"
		}, "not in the ledger's referenced_terms_verified"},
		{"bfo prefix", func(s *vocabulary.PackSpec) { s.Prefixes[0].Prefix = "bfo" }, "prefix bfo declared"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, err := diekm.Spec()
			if err != nil {
				t.Fatal(err)
			}
			if spec.EntityTypes[0].DomainType != "intel.platform" {
				t.Fatalf("seed expects intel.platform first, got %s", spec.EntityTypes[0].DomainType)
			}
			tt.mutate(&spec)
			got := strings.Join(closureProblems("diekm", spec, l, release), "\n")
			if !strings.Contains(got, tt.want) {
				t.Fatalf("seeded fault not named; want %q in:\n%s", tt.want, got)
			}
		})
	}
}

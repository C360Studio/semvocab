package packs_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/c360studio/semengine/vocabulary"
	"github.com/c360studio/semvocab/catalog/bfo"
	"github.com/c360studio/semvocab/catalog/cco"
	"github.com/c360studio/semvocab/catalog/sensorml"
	"github.com/c360studio/semvocab/catalog/sosa"
	"github.com/c360studio/semvocab/catalog/ssn"
	"github.com/c360studio/semvocab/catalog/swe"
	"github.com/c360studio/semvocab/internal/ledger"
	"github.com/c360studio/semvocab/internal/oracle"
	"github.com/c360studio/semvocab/packs"
	"github.com/c360studio/semvocab/packs/diekm"
	smlpack "github.com/c360studio/semvocab/packs/sensorml"
)

const (
	rdfsIsDefinedBy = "http://www.w3.org/2000/01/rdf-schema#isDefinedBy"
	dcDescription   = "http://purl.org/dc/terms/description"
)

var standardIRIs = []string{
	oracle.RDFType,
	oracle.RDFSLabel,
	rdfsIsDefinedBy,
	vocabulary.DcIdentifier,
	vocabulary.DcTitle,
	dcDescription,
	vocabulary.SkosNotation,
	vocabulary.SkosBroader,
}

// declaredParents are each entity type's declared direct parents in its home
// ontology or schema bundle, excluding pack local_assertions from the ledger.
var declaredParents = map[string]map[string][]string{
	"diekm": {
		"intel.platform":    {cco.Vehicle},
		"intel.observation": {cco.ActOfObservation, bfo.Process},
	},
	"bfo-cco-core": {},
	"swe-core":     {},
	"sensorml": {
		"csapi.system":          {sensorml.AbstractPhysicalProcess},
		"csapi.procedure":       {sensorml.AbstractProcess},
		"csapi.deployment":      {sensorml.DescribedObject},
		"csapi.samplingfeature": {sosa.FeatureOfInterest, sosa.Result},
		"csapi.property":        {swe.AbstractSweIdentifiable},
	},
}

type embeddedPack struct {
	name string
	spec func() (vocabulary.PackSpec, error)
}

var embedded = []embeddedPack{
	{"bfo-cco-core", packs.BFOCCOCoreSpec},
	{"swe-core", packs.SWECoreSpec},
	{"diekm", diekm.Spec},
	{"sensorml", smlpack.Spec},
}

type packOracles struct {
	cco        *oracle.Graph
	ssn        *oracle.Graph
	swe        *oracle.SchemaCatalog
	sml        *oracle.SchemaCatalog
	sweClasses map[string]string
	smlClasses map[string]string
	sweProps   map[string]bool
	smlProps   map[string]bool
}

// expand turns a ledger CURIE into an IRI using the pack's own prefixes.
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

func checkPrefixes(spec vocabulary.PackSpec, l ledger.Ledger, add func(string, ...any)) map[string]string {
	pins := map[string]string{
		"obo":        l.Pins.Ontologies["bfo-2020"].Namespace,
		"cco":        l.Pins.Ontologies["cco-2.2"].Namespace,
		"sosa":       l.Pins.Ontologies["sosa-2017"].Namespace,
		"ssn":        l.Pins.Ontologies["ssn-2017"].Namespace,
		"ssn-system": l.Pins.Ontologies["ssn-system-2017"].Namespace,
		"rdf":        strings.TrimSuffix(oracle.RDFType, "type"),
		"rdfs":       strings.TrimSuffix(oracle.RDFSLabel, "label"),
		"dc":         strings.TrimSuffix(vocabulary.DcIdentifier, "identifier"),
		"skos":       strings.TrimSuffix(vocabulary.SkosBroader, "broader"),
	}
	if e, err := l.Catalog("swe"); err == nil {
		pins["swe"] = e.Namespace.IRI
	}
	if e, err := l.Catalog("sensorml"); err == nil {
		pins["sml"] = e.Namespace.IRI
	}
	if pins["obo"] != bfo.Namespace || pins["cco"] != cco.Namespace ||
		pins["sosa"] != sosa.Namespace || pins["ssn"] != ssn.Namespace ||
		pins["ssn-system"] != ssn.SystemsNamespace ||
		pins["swe"] != swe.Namespace || pins["sml"] != sensorml.Namespace {
		add("ledger pins %v disagree with catalog namespaces", pins)
	}
	if e, err := l.Pack("diekm"); err == nil {
		pins["diekm"] = e.Namespace.IRI
	}
	for _, p := range spec.Prefixes {
		if want, ok := pins[p.Prefix]; ok && p.Namespace != want {
			add("prefix %s = %s, ledger pin %s", p.Prefix, p.Namespace, want)
		}
		if p.Prefix == "bfo" {
			add("prefix bfo declared; design P5 names the OBO namespace obo")
		}
		if p.Prefix == "dcterms" {
			add("prefix dcterms declared; design D4 names %s dc", pins["dc"])
		}
		if strings.HasPrefix(p.Prefix, "csapi") {
			add("prefix %s declared; ruling R4 excludes Working-Draft CS API Part 2", p.Prefix)
		}
	}
	return pins
}

func bfsClosure(roots []string, o packOracles) []string {
	var out, queue []string
	seen := map[string]bool{}
	for _, r := range roots {
		if !seen[r] {
			seen[r] = true
			out = append(out, r)
			queue = append(queue, r)
		}
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		parents := append(slices.Clone(o.cco.DirectSuperclasses(cur)), o.ssn.DirectSuperclasses(cur)...)
		if smlAnc := o.sml.Closure(cur); len(smlAnc) > 0 {
			parents = append(parents, smlAnc[0])
		}
		for _, p := range parents {
			if !seen[p] {
				seen[p] = true
				out = append(out, p)
				queue = append(queue, p)
			}
		}
	}
	return out
}

func verifyIRIOracle(where, iri string, o packOracles, add func(string, ...any)) {
	switch {
	case strings.HasPrefix(iri, bfo.Namespace) || strings.HasPrefix(iri, cco.Namespace):
		if len(o.cco.Types(iri)) == 0 || o.cco.Deprecated(iri) {
			add("%s = %q is not declared, or is deprecated, in the CCO v2.2 file", where, iri)
		}
	case strings.HasPrefix(iri, sosa.Namespace) || strings.HasPrefix(iri, ssn.Namespace):
		if len(o.ssn.Types(iri)) == 0 || o.ssn.Deprecated(iri) {
			add("%s = %q is not declared, or is deprecated, in SOSA/SSN/ssn-system 2017", where, iri)
		}
	case strings.HasPrefix(iri, swe.Namespace):
		local := strings.TrimPrefix(iri, swe.Namespace)
		loc, isClass := o.sweClasses[local]
		if (!isClass || !o.swe.HasObjectClass(local, loc)) && !o.sweProps[local] {
			add("%s = %q is not an object class or declared property in the SWE Common 3.0 schema bundle", where, iri)
		}
	case strings.HasPrefix(iri, sensorml.Namespace):
		local := strings.TrimPrefix(iri, sensorml.Namespace)
		loc, isClass := o.smlClasses[local]
		if (!isClass || !o.sml.HasObjectClass(local, loc)) && !o.smlProps[local] {
			add("%s = %q is not an object class or declared property in the SensorML 3.0 schema bundle", where, iri)
		}
	case slices.Contains(standardIRIs, iri):
	default:
		add("%s = %q is not evidenced by any pinned oracle or standard IRI", where, iri)
	}
}

// closureProblems checks one pack: full closures (D5) and expanded in-namespace
// IRIs (D6), each IRI evidenced by its vendored release/schema bundle or the
// ledger.
func closureProblems(name string, spec vocabulary.PackSpec, l ledger.Ledger, o packOracles) []string {
	var bad []string
	add := func(format string, args ...any) { bad = append(bad, fmt.Sprintf(name+": "+format, args...)) }

	pins := checkPrefixes(spec, l, add)
	entry, err := l.Pack(name)
	if err != nil {
		add("%v", err)
		return bad
	}
	local, err := localAssertions(spec, entry)
	if err != nil {
		add("%v", err)
	}

	for _, et := range spec.EntityTypes {
		declared, ok := declaredParents[name][et.DomainType]
		if !ok {
			add("entity type %s has no declared parents on record in this test", et.DomainType)
			continue
		}
		wantBFS := bfsClosure(append(slices.Clone(declared), local[et.ClassIRI]...), o)
		got := slices.Clone(et.Superclasses)
		slices.Sort(got)
		if len(slices.Compact(got)) != len(et.Superclasses) {
			add("%s superclasses repeat an IRI: %v", et.DomainType, et.Superclasses)
		}
		wantSet := slices.Clone(wantBFS)
		slices.Sort(wantSet)
		if !slices.Equal(got, wantSet) {
			add("%s superclasses %v, want closure %v", et.DomainType, et.Superclasses, wantBFS)
		} else if name == "sensorml" && !slices.Equal(et.Superclasses, wantBFS) {
			add("%s superclasses %v are out of breadth-first order %v", et.DomainType, et.Superclasses, wantBFS)
		}
	}

	verified := map[string]string{}
	for _, c := range entry.ReferencedTermsVerified {
		if strings.HasPrefix(c, "csapi:") {
			add("ledger referenced_terms_verified admits Working-Draft %s", c)
		}
		iri, err := expand(spec, c)
		if err != nil {
			add("ledger referenced term: %v", err)
			continue
		}
		verified[iri] = c
	}

	diekmNS := pins["diekm"]
	usedVerified := map[string]bool{}
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
		case diekmNS != "" && strings.HasPrefix(iri, diekmNS):
			if !strings.Contains(entry.AdmittedTerms, "diekm:"+strings.TrimPrefix(iri, diekmNS)) {
				add("%s = %q is not a ledger-admitted DIEKM term", where, iri)
			}
		case verified[iri] == "":
			add("%s = %q is not in the ledger's referenced_terms_verified", where, iri)
		default:
			usedVerified[iri] = true
			verifyIRIOracle(where, iri, o, add)
		}
	}
	for iri, curie := range verified {
		if !usedVerified[iri] {
			add("unused referenced_terms_verified %s (%s)", curie, iri)
		}
	}
	return bad
}

func loadOracles(t *testing.T) (ledger.Ledger, packOracles) {
	t.Helper()
	l, err := ledger.Load()
	if err != nil {
		t.Fatal(err)
	}
	ccoGraph, err := oracle.Load(oracle.CCO22)
	if err != nil {
		t.Fatal(err)
	}
	ssnGraph, err := oracle.LoadMerged(oracle.SOSA2017, oracle.SSN2017, oracle.SSNSystem2017)
	if err != nil {
		t.Fatal(err)
	}
	sweCat, err := oracle.LoadSchemaCatalog(oracle.SWE30)
	if err != nil {
		t.Fatal(err)
	}
	smlCat, err := oracle.LoadSchemaCatalog(oracle.SensorML30)
	if err != nil {
		t.Fatal(err)
	}
	sweSrc, err := l.SWE()
	if err != nil {
		t.Fatal(err)
	}
	smlSrc, err := l.SensorML()
	if err != nil {
		t.Fatal(err)
	}
	sweClasses := map[string]string{}
	for _, v := range sweSrc.AdmittedClasses {
		sweClasses[v[0]] = v[1]
	}
	smlClasses := map[string]string{}
	for _, v := range smlSrc.AdmittedClasses {
		smlClasses[v[0]] = v[1]
	}
	sweProps := map[string]bool{}
	for _, p := range sweCat.DeclaredProperties() {
		sweProps[p] = true
	}
	smlProps := map[string]bool{}
	for _, p := range smlCat.DeclaredProperties() {
		smlProps[p] = true
	}
	return l, packOracles{
		cco:        ccoGraph,
		ssn:        ssnGraph,
		swe:        sweCat,
		sml:        smlCat,
		sweClasses: sweClasses,
		smlClasses: smlClasses,
		sweProps:   sweProps,
		smlProps:   smlProps,
	}
}

func TestPackSuperclassClosure(t *testing.T) {
	l, o := loadOracles(t)
	for _, p := range embedded {
		spec, err := p.spec()
		if err != nil {
			t.Fatalf("%s: %v", p.name, err)
		}
		if spec.Name != p.name {
			t.Errorf("embedded pack name %q, want %q", spec.Name, p.name)
		}
		for _, problem := range closureProblems(p.name, spec, l, o) {
			t.Error(problem)
		}
	}

	// Spec scenario: intel.platform lists the Vehicle closure plus BFO_0000030.
	diekmSpec, err := diekm.Spec()
	if err != nil {
		t.Fatal(err)
	}
	platform := diekmSpec.EntityTypes[slices.IndexFunc(diekmSpec.EntityTypes, func(e vocabulary.EntityTypeSpec) bool {
		return e.DomainType == "intel.platform"
	})]
	if len(platform.Superclasses) != 7 || !slices.Contains(platform.Superclasses, bfo.Object) {
		t.Errorf("intel.platform superclasses %v, want 7 IRIs including %s", platform.Superclasses, bfo.Object)
	}

	// Spec scenario: Dual SensorML 3.0 and SOSA/SSN closures on csapi.system.
	smlSpec, err := smlpack.Spec()
	if err != nil {
		t.Fatal(err)
	}
	sysIdx := slices.IndexFunc(smlSpec.EntityTypes, func(e vocabulary.EntityTypeSpec) bool {
		return e.DomainType == "csapi.system"
	})
	if sysIdx < 0 {
		t.Fatal("sensorml lacks csapi.system")
	}
	wantSys := []string{
		sensorml.AbstractPhysicalProcess,
		ssn.System,
		sensorml.AbstractProcess,
		sensorml.DescribedObject,
	}
	if !slices.Equal(smlSpec.EntityTypes[sysIdx].Superclasses, wantSys) {
		t.Errorf("csapi.system superclasses = %v, want %v", smlSpec.EntityTypes[sysIdx].Superclasses, wantSys)
	}
}

// TestPackSuperclassClosureSensitivity seeds faults into diekm, swe-core, and
// sensorml and requires closureProblems to name each.
func TestPackSuperclassClosureSensitivity(t *testing.T) {
	l, o := loadOracles(t)
	diekmTests := []struct {
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
	for _, tt := range diekmTests {
		t.Run("diekm/"+tt.name, func(t *testing.T) {
			spec, err := diekm.Spec()
			if err != nil {
				t.Fatal(err)
			}
			tt.mutate(&spec)
			got := strings.Join(closureProblems("diekm", spec, l, o), "\n")
			if !strings.Contains(got, tt.want) {
				t.Fatalf("seeded fault not named; want %q in:\n%s", tt.want, got)
			}
		})
	}

	t.Run("swe-core/wrong prefix namespace", func(t *testing.T) {
		spec, err := packs.SWECoreSpec()
		if err != nil {
			t.Fatal(err)
		}
		spec.Prefixes[3].Namespace = "http://www.opengis.net/swe/2.0/"
		got := strings.Join(closureProblems("swe-core", spec, l, o), "\n")
		if !strings.Contains(got, "swe-core: prefix swe = http://www.opengis.net/swe/2.0/") {
			t.Fatalf("seeded fault not named; got:\n%s", got)
		}
	})

	runSensorMLSensitivity(t, l, o)
}

func cloneLedgerWithSensorMLTerms(l ledger.Ledger, fn func([]string) []string) ledger.Ledger {
	cp := l
	cp.Entries = slices.Clone(l.Entries)
	for i := range cp.Entries {
		if cp.Entries[i].Pack == "sensorml" {
			cp.Entries[i].ReferencedTermsVerified = fn(slices.Clone(cp.Entries[i].ReferencedTermsVerified))
		}
	}
	return cp
}

func runSensorMLSensitivity(t *testing.T, l ledger.Ledger, o packOracles) {
	t.Helper()
	tests := []struct {
		name   string
		mutate func(*vocabulary.PackSpec, *ledger.Ledger)
		want   string
	}{
		{"missing ssn.Property from csapi.property", func(s *vocabulary.PackSpec, _ *ledger.Ledger) {
			idx := slices.IndexFunc(s.EntityTypes, func(e vocabulary.EntityTypeSpec) bool {
				return e.DomainType == "csapi.property"
			})
			s.EntityTypes[idx].Superclasses = slices.DeleteFunc(s.EntityTypes[idx].Superclasses,
				func(iri string) bool { return iri == ssn.Property })
		}, "csapi.property superclasses"},
		{"unlisted sml IRI", func(s *vocabulary.PackSpec, _ *ledger.Ledger) {
			s.Predicates[0].IRI = sensorml.Namespace + "bogusProperty"
		}, "not in the ledger's referenced_terms_verified"},
		{"undeclared sml IRI in schema oracle", func(s *vocabulary.PackSpec, lg *ledger.Ledger) {
			s.Predicates[9].IRI = sensorml.Namespace + "bogusProperty"
			*lg = cloneLedgerWithSensorMLTerms(*lg, func(terms []string) []string {
				return append(slices.DeleteFunc(terms, func(c string) bool { return c == "sml:attachedTo" }), "sml:bogusProperty")
			})
		}, "is not an object class or declared property in the SensorML 3.0 schema bundle"},
		{"unused referenced_terms_verified", func(_ *vocabulary.PackSpec, lg *ledger.Ledger) {
			*lg = cloneLedgerWithSensorMLTerms(*lg, func(terms []string) []string {
				return append(terms, "sosa:Sensor")
			})
		}, "unused referenced_terms_verified sosa:Sensor"},
		{"dcterms prefix", func(s *vocabulary.PackSpec, _ *ledger.Ledger) {
			idx := slices.IndexFunc(s.Prefixes, func(p vocabulary.PrefixSpec) bool { return p.Prefix == "dc" })
			s.Prefixes[idx].Prefix = "dcterms"
		}, "prefix dcterms declared"},
	}
	for _, tt := range tests {
		t.Run("sensorml/"+tt.name, func(t *testing.T) {
			spec, err := smlpack.Spec()
			if err != nil {
				t.Fatal(err)
			}
			lg := l
			tt.mutate(&spec, &lg)
			got := strings.Join(closureProblems("sensorml", spec, lg, o), "\n")
			if !strings.Contains(got, tt.want) {
				t.Fatalf("seeded fault not named; want %q in:\n%s", tt.want, got)
			}
		})
	}
}

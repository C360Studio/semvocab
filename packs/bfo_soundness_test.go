package packs_test

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/c360studio/semengine/vocabulary"
	"github.com/c360studio/semvocab/catalog/bfo"
	"github.com/c360studio/semvocab/catalog/cco"
	"github.com/c360studio/semvocab/internal/ledger"
	"github.com/c360studio/semvocab/internal/oracle"
	"github.com/c360studio/semvocab/internal/turtle"
	"github.com/c360studio/semvocab/packs/diekm"
	"github.com/c360studio/semvocab/shacl"
)

// The BFO soundness checks of issue #32 (ruling R1: test harness only, no
// exported API). They run over every embedded pack whose ledger entry
// references bfo-2020 or cco-2.2, classify classes through cco.IsA (which
// falls back to bfo.IsA), and read disjointness, domains, and ranges from the
// pinned bfo-core.ttl. Design: openspec/changes/bfo-soundness-checks/design.md.

const (
	rdfsDomain            = "http://www.w3.org/2000/01/rdf-schema#domain"
	rdfsRange             = "http://www.w3.org/2000/01/rdf-schema#range"
	rdfFirst              = "http://www.w3.org/1999/02/22-rdf-syntax-ns#first"
	rdfRest               = "http://www.w3.org/1999/02/22-rdf-syntax-ns#rest"
	rdfNil                = "http://www.w3.org/1999/02/22-rdf-syntax-ns#nil"
	owlUnionOf            = "http://www.w3.org/2002/07/owl#unionOf"
	owlIntersectionOf     = "http://www.w3.org/2002/07/owl#intersectionOf"
	owlComplementOf       = "http://www.w3.org/2002/07/owl#complementOf"
	owlDisjointWith       = "http://www.w3.org/2002/07/owl#disjointWith"
	owlAllDisjointClasses = "http://www.w3.org/2002/07/owl#AllDisjointClasses"
	owlMembers            = "http://www.w3.org/2002/07/owl#members"
)

// bfoBucketRoots are the classes of issue #32's seven buckets (a grouping
// after jupyter-naas/abi, not a BFO 2020 term): what (Process,
// ProcessBoundary), when (TemporalRegion, SpatiotemporalRegion), who
// (MaterialEntity), where (Site, SpatialRegion), how we know
// (GenericallyDependentContinuant), how it is (Quality), and why
// (RealizableEntity). The continuant fiat boundaries (BFO_0000140, 142, 146,
// 147) reach none of them and are rejected (owner answer 3, 2026-11-07).
var bfoBucketRoots = []string{
	bfo.Process, bfo.ProcessBoundary,
	bfo.TemporalRegion, bfo.SpatiotemporalRegion,
	bfo.MaterialEntity,
	bfo.Site, bfo.SpatialRegion,
	bfo.GenericallyDependentContinuant,
	bfo.Quality,
	bfo.RealizableEntity,
}

// bfoHopRelations are the 17 relations whose bfo-2020 rdfs:domain and
// rdfs:range the hop check enforces (design D4). Other OBO predicates stay
// unchecked for domain and range (owner answer 4, 2026-11-07).
var bfoHopRelations = []string{
	bfo.BearerOf, bfo.InheresIn,
	bfo.GenericallyDependsOn, bfo.IsCarrierOf,
	bfo.Realizes, bfo.RealizedIn,
	bfo.Concretizes, bfo.IsConcretizedBy,
	bfo.HasMemberPart, bfo.MemberPartOf,
	bfo.ContinuantPartOf, bfo.HasContinuantPart,
	bfo.HasOccurrentPart, bfo.OccurrentPartOf,
	bfo.OccupiesTemporalRegion,
	bfo.Precedes, bfo.PrecededBy,
}

// bfoPackMappings holds the SHACL mapping of each BFO pack with projections:
// step 0's subject is each entity type the mapping binds the predicate to
// (design D3). A BFO pack with projections and no entry here fails closed
// (owner answer 2, 2026-11-07).
var bfoPackMappings = map[string]func() []byte{
	"diekm": diekm.Mapping,
}

// processConnectedPacks opts packs in to the process connectivity diagnostic
// by name (design D5). No embedded pack is opted in.
var processConnectedPacks = map[string]bool{}

// selectBFOPacks returns the embedded packs whose ledger entry references
// bfo-2020 or cco-2.2 (design D2). A pack with no ledger entry is an error.
func selectBFOPacks(l ledger.Ledger) ([]embeddedPack, error) {
	var out []embeddedPack
	for _, p := range embedded {
		e, err := l.Pack(p.name)
		if err != nil {
			return nil, err
		}
		if slices.Contains(e.References, string(oracle.BFO2020)) || slices.Contains(e.References, string(oracle.CCO22)) {
			out = append(out, p)
		}
	}
	return out, nil
}

// knownClass reports whether iri is a catalog/bfo or catalog/cco class.
func knownClass(iri string) bool {
	if iri == bfo.Entity {
		return true
	}
	_, ok := cco.SubClassOf(iri)
	return ok
}

// classSetIsA reports whether some member of set is, or is a catalog subclass
// of, class.
func classSetIsA(set []string, class string) bool {
	return slices.ContainsFunc(set, func(m string) bool { return cco.IsA(m, class) })
}

// entityClassSet is an entity type's class_iri followed by its superclasses.
func entityClassSet(et vocabulary.EntityTypeSpec) []string {
	return append([]string{et.ClassIRI}, et.Superclasses...)
}

// bucketRootsOf returns the bucket roots that set reaches, in
// bfoBucketRoots order.
func bucketRootsOf(set []string) []string {
	var out []string
	for _, r := range bfoBucketRoots {
		if classSetIsA(set, r) {
			out = append(out, r)
		}
	}
	return out
}

// disjointPairs reads every pairwise disjointness the bfo-2020 release
// asserts: owl:disjointWith between named BFO classes, and each pair of an
// owl:AllDisjointClasses member list.
func disjointPairs(g *oracle.Graph) ([][2]string, error) {
	var out [][2]string
	for _, s := range g.SubjectsWithPrefix(bfo.Namespace) {
		for _, o := range g.IRIObjects(s, owlDisjointWith) {
			out = append(out, [2]string{s, o})
		}
	}
	for _, s := range g.SubjectsWithPrefix("_:") {
		if !g.HasType(s, owlAllDisjointClasses) {
			continue
		}
		heads := g.Objects(s, owlMembers)
		if len(heads) != 1 {
			return nil, fmt.Errorf("owl:AllDisjointClasses %s has %d owl:members lists, want 1", s, len(heads))
		}
		items, err := readList(g, heads[0])
		if err != nil {
			return nil, err
		}
		for i, a := range items {
			if a.Kind != turtle.IRI {
				return nil, fmt.Errorf("owl:AllDisjointClasses %s member %d is not a named class", s, i)
			}
			for _, b := range items[i+1:] {
				if b.Kind != turtle.IRI {
					return nil, fmt.Errorf("owl:AllDisjointClasses %s has a member that is not a named class", s)
				}
				out = append(out, [2]string{a.Value, b.Value})
			}
		}
	}
	return out, nil
}

// bucketProblems names each entity type of a selected pack that reaches no
// bucket root, or whose class set meets both classes of a disjoint pair.
func bucketProblems(name string, spec vocabulary.PackSpec, pairs [][2]string) []string {
	var out []string
	for _, et := range spec.EntityTypes {
		set := entityClassSet(et)
		if len(bucketRootsOf(set)) == 0 {
			out = append(out, fmt.Sprintf("%s: entity type %s has no BFO bucket root in its class or superclasses",
				name, et.DomainType))
		}
		for _, p := range pairs {
			if classSetIsA(set, p[0]) && classSetIsA(set, p[1]) {
				out = append(out, fmt.Sprintf("%s: entity type %s meets disjoint classes %s and %s (bfo-2020 disjointness)",
					name, et.DomainType, p[0], p[1]))
			}
		}
	}
	return out
}

// objectReader is the read surface the class-expression interpreter needs;
// *oracle.Graph satisfies it.
type objectReader interface {
	Objects(s, p string) []turtle.Term
}

// readList walks an rdf:first/rdf:rest collection from head to rdf:nil,
// failing closed on a malformed or cyclic list.
func readList(g objectReader, head turtle.Term) ([]turtle.Term, error) {
	var out []turtle.Term
	seen := map[string]bool{}
	for node := head; !(node.Kind == turtle.IRI && node.Value == rdfNil); {
		if node.Kind != turtle.Blank || seen[node.Value] {
			return nil, fmt.Errorf("malformed RDF list at %q", node.Value)
		}
		seen[node.Value] = true
		first, rest := g.Objects(node.Value, rdfFirst), g.Objects(node.Value, rdfRest)
		if len(first) != 1 || len(rest) != 1 {
			return nil, fmt.Errorf("malformed RDF list node %s", node.Value)
		}
		out = append(out, first[0])
		node = rest[0]
	}
	return out, nil
}

// classExpr is an interpreted rdfs:domain or rdfs:range: a class set
// satisfies it when some member is under one of AnyOf and no member is under
// any of NoneOf (closed world over the declared closure, design D4).
type classExpr struct {
	AnyOf  []string
	NoneOf []string
}

func (e classExpr) holds(set []string) bool {
	if !slices.ContainsFunc(e.AnyOf, func(c string) bool { return classSetIsA(set, c) }) {
		return false
	}
	return !slices.ContainsFunc(e.NoneOf, func(c string) bool { return classSetIsA(set, c) })
}

// readClassExpr interprets the three class-expression shapes bfo-2020 uses on
// the hop relations: a named class, owl:unionOf named classes, and
// owl:intersectionOf one named class with owl:complementOf named classes. Any
// other shape fails closed.
func readClassExpr(g objectReader, t turtle.Term) (classExpr, error) {
	if t.Kind == turtle.IRI {
		return classExpr{AnyOf: []string{t.Value}}, nil
	}
	if t.Kind != turtle.Blank {
		return classExpr{}, fmt.Errorf("class expression %q is a literal", t.Value)
	}
	union, inter, compl := g.Objects(t.Value, owlUnionOf), g.Objects(t.Value, owlIntersectionOf), g.Objects(t.Value, owlComplementOf)
	switch {
	case len(union) == 1 && len(inter) == 0 && len(compl) == 0:
		items, err := readList(g, union[0])
		if err != nil {
			return classExpr{}, err
		}
		var e classExpr
		for _, it := range items {
			if it.Kind != turtle.IRI {
				return classExpr{}, fmt.Errorf("owl:unionOf member %q is not a named class", it.Value)
			}
			e.AnyOf = append(e.AnyOf, it.Value)
		}
		if len(e.AnyOf) == 0 {
			return classExpr{}, fmt.Errorf("empty owl:unionOf")
		}
		return e, nil
	case len(inter) == 1 && len(union) == 0 && len(compl) == 0:
		items, err := readList(g, inter[0])
		if err != nil {
			return classExpr{}, err
		}
		var e classExpr
		for _, it := range items {
			if it.Kind == turtle.IRI {
				e.AnyOf = append(e.AnyOf, it.Value)
				continue
			}
			c := g.Objects(it.Value, owlComplementOf)
			if it.Kind != turtle.Blank || len(c) != 1 || c[0].Kind != turtle.IRI ||
				len(g.Objects(it.Value, owlUnionOf)) != 0 || len(g.Objects(it.Value, owlIntersectionOf)) != 0 {
				return classExpr{}, fmt.Errorf("owl:intersectionOf member %q is neither a named class nor owl:complementOf one", it.Value)
			}
			e.NoneOf = append(e.NoneOf, c[0].Value)
		}
		if len(e.AnyOf) != 1 {
			return classExpr{}, fmt.Errorf("owl:intersectionOf has %d named classes, want 1", len(e.AnyOf))
		}
		return e, nil
	default:
		return classExpr{}, fmt.Errorf("unsupported class expression %s (unionOf %d, intersectionOf %d, complementOf %d)",
			t.Value, len(union), len(inter), len(compl))
	}
}

// relationBounds is one hop relation's interpreted domains and ranges; every
// listed expression must hold (RDFS semantics).
type relationBounds struct {
	Domain, Range []classExpr
}

// readRelationBounds interprets rdfs:domain and rdfs:range of each relation,
// failing closed when one declares neither or uses an unsupported shape.
func readRelationBounds(g objectReader, relations []string) (map[string]relationBounds, error) {
	out := map[string]relationBounds{}
	for _, r := range relations {
		var b relationBounds
		for _, side := range []struct {
			p   string
			dst *[]classExpr
		}{{rdfsDomain, &b.Domain}, {rdfsRange, &b.Range}} {
			terms := g.Objects(r, side.p)
			if len(terms) == 0 {
				return nil, fmt.Errorf("%s declares no %s", r, side.p)
			}
			for _, t := range terms {
				e, err := readClassExpr(g, t)
				if err != nil {
					return nil, fmt.Errorf("%s %s: %w", r, side.p, err)
				}
				*side.dst = append(*side.dst, e)
			}
		}
		out[r] = b
	}
	return out, nil
}

// boundSubjects maps each predicate name to the entity types whose mapping
// contract lists it, as a birth predicate or in a group (design D3).
func boundSubjects(m shacl.Mapping) map[string][]string {
	out := map[string][]string{}
	for _, c := range m.Contracts {
		preds := slices.Clone(c.Birth)
		for _, g := range c.Groups {
			preds = append(preds, g.Predicates...)
		}
		for _, p := range preds {
			if !slices.Contains(out[p], c.DomainType) {
				out[p] = append(out[p], c.DomainType)
			}
		}
	}
	return out
}

// packMapping decodes a selected pack's SHACL mapping. A pack with a projected
// predicate and no mapping fails closed; a pack with no projection needs none.
func packMapping(name string, spec vocabulary.PackSpec) (shacl.Mapping, error) {
	src, ok := bfoPackMappings[name]
	if !ok {
		if slices.ContainsFunc(spec.Predicates, func(p vocabulary.PredicateSpec) bool { return p.Projection != nil }) {
			return shacl.Mapping{}, fmt.Errorf("%s: pack has projections but no SHACL mapping binds their subjects", name)
		}
		return shacl.Mapping{}, nil
	}
	return shacl.DecodeMappingYAML(bytes.NewReader(src()))
}

// hopProblems checks every projection hop of a selected pack (design D3-D5)
// and returns the problems and the hops it checked against a relation's
// domain and range ("<predicate> step <i> <relation>").
//
// Every hop: a node class unknown to both catalogs fails closed; a hop from an
// independent to a specifically dependent continuant must use bfo.BearerOf,
// and the reverse bfo.InheresIn (owner answer 1, 2026-11-07). A hop through
// one of bfoHopRelations must also have its subject inside the relation's
// domain and its node inside the range. Step 0's subject is each entity type
// the mapping binds the predicate to; a projected predicate bound to none is
// a problem.
func hopProblems(name string, spec vocabulary.PackSpec, m shacl.Mapping, rel map[string]relationBounds) (problems, checked []string) {
	subjects := boundSubjects(m)
	for _, p := range spec.Predicates {
		if p.Projection == nil || len(p.Projection.Steps) == 0 {
			continue
		}
		bound := subjects[p.Name]
		if len(bound) == 0 {
			problems = append(problems, fmt.Sprintf("%s: projected predicate %s is bound to no entity type by the mapping",
				name, p.Name))
			continue
		}
		for _, dt := range bound {
			i := slices.IndexFunc(spec.EntityTypes, func(e vocabulary.EntityTypeSpec) bool { return e.DomainType == dt })
			if i < 0 {
				problems = append(problems, fmt.Sprintf("%s: %s is bound to %s, which the pack does not declare", name, p.Name, dt))
				continue
			}
			prev := entityClassSet(spec.EntityTypes[i])
			if !slices.ContainsFunc(prev, knownClass) {
				problems = append(problems, fmt.Sprintf("%s: %s subject %s has no class known to catalog/bfo or catalog/cco",
					name, p.Name, dt))
				continue
			}
			for si, step := range p.Projection.Steps {
				where := fmt.Sprintf("%s: %s step %d (chain from %s)", name, p.Name, si, dt)
				if !knownClass(step.NodeClassIRI) {
					problems = append(problems, fmt.Sprintf("%s: node class %s is unknown to catalog/bfo and catalog/cco",
						where, step.NodeClassIRI))
					break
				}
				node := []string{step.NodeClassIRI}
				if classSetIsA(prev, bfo.IndependentContinuant) && classSetIsA(node, bfo.SpecificallyDependentContinuant) &&
					step.PredicateIRI != bfo.BearerOf {
					problems = append(problems, fmt.Sprintf("%s: reaches an SDC %s from an independent continuant without bfo.BearerOf (predicate %s)",
						where, step.NodeClassIRI, step.PredicateIRI))
				}
				if classSetIsA(prev, bfo.SpecificallyDependentContinuant) && classSetIsA(node, bfo.IndependentContinuant) &&
					step.PredicateIRI != bfo.InheresIn {
					problems = append(problems, fmt.Sprintf("%s: reaches an independent continuant %s from an SDC without bfo.InheresIn (predicate %s)",
						where, step.NodeClassIRI, step.PredicateIRI))
				}
				if b, ok := rel[step.PredicateIRI]; ok {
					checked = append(checked, fmt.Sprintf("%s step %d %s", p.Name, si, step.PredicateIRI))
					for _, d := range b.Domain {
						if !d.holds(prev) {
							problems = append(problems, fmt.Sprintf("%s: subject is outside the rdfs:domain of %s", where, step.PredicateIRI))
						}
					}
					for _, r := range b.Range {
						if !r.holds(node) {
							problems = append(problems, fmt.Sprintf("%s: node class %s is outside the rdfs:range of %s",
								where, step.NodeClassIRI, step.PredicateIRI))
						}
					}
				}
				prev = node
			}
		}
	}
	return problems, checked
}

// connectivityProblems is the opt-in diagnostic (design D5): it names each
// continuant entity type that is the subject of no predicate whose projection
// reaches a node class under bfo.Process.
func connectivityProblems(name string, spec vocabulary.PackSpec, m shacl.Mapping) []string {
	subjects := boundSubjects(m)
	var out []string
	for _, et := range spec.EntityTypes {
		if !classSetIsA(entityClassSet(et), bfo.Continuant) {
			continue
		}
		reached := slices.ContainsFunc(spec.Predicates, func(p vocabulary.PredicateSpec) bool {
			return p.Projection != nil && slices.Contains(subjects[p.Name], et.DomainType) &&
				slices.ContainsFunc(p.Projection.Steps, func(s vocabulary.ProjectionStep) bool {
					return classSetIsA([]string{s.NodeClassIRI}, bfo.Process)
				})
		})
		if !reached {
			out = append(out, fmt.Sprintf("%s: continuant entity type %s is the subject of no projection reaching a bfo.Process",
				name, et.DomainType))
		}
	}
	return out
}

func loadBFOGraph(t *testing.T) *oracle.Graph {
	t.Helper()
	g, err := oracle.Load(oracle.BFO2020)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func loadDisjointPairs(t *testing.T) [][2]string {
	t.Helper()
	pairs, err := disjointPairs(loadBFOGraph(t))
	if err != nil {
		t.Fatal(err)
	}
	return pairs
}

func loadRelationBounds(t *testing.T) map[string]relationBounds {
	t.Helper()
	rel, err := readRelationBounds(loadBFOGraph(t), bfoHopRelations)
	if err != nil {
		t.Fatal(err)
	}
	return rel
}

func diekmSpecAndMapping(t *testing.T) (vocabulary.PackSpec, shacl.Mapping) {
	t.Helper()
	spec, err := diekm.Spec()
	if err != nil {
		t.Fatal(err)
	}
	m, err := packMapping("diekm", spec)
	if err != nil {
		t.Fatal(err)
	}
	return spec, m
}

func entityTypeIndex(t *testing.T, spec vocabulary.PackSpec, dt string) int {
	t.Helper()
	i := slices.IndexFunc(spec.EntityTypes, func(e vocabulary.EntityTypeSpec) bool { return e.DomainType == dt })
	if i < 0 {
		t.Fatalf("%s declares no entity type %s", spec.Name, dt)
	}
	return i
}

func predicateIndex(t *testing.T, spec vocabulary.PackSpec, name string) int {
	t.Helper()
	i := slices.IndexFunc(spec.Predicates, func(p vocabulary.PredicateSpec) bool { return p.Name == name })
	if i < 0 || spec.Predicates[i].Projection == nil {
		t.Fatalf("%s declares no projected predicate %s", spec.Name, name)
	}
	return i
}

func wantProblem(t *testing.T, problems []string, parts ...string) {
	t.Helper()
	for _, p := range problems {
		if !slices.ContainsFunc(parts, func(s string) bool { return !strings.Contains(p, s) }) {
			return
		}
	}
	t.Errorf("no problem contains all of %q; got:\n%s", parts, strings.Join(problems, "\n"))
}

// TestBFOPackSelection pins the selected set, so a selector that selects
// nothing or everything fails.
//
// spec: openspec/changes/bfo-soundness-checks/specs/catalogs-and-packs/spec.md#Requirement: BFO-aligned packs anchor every entity type
func TestBFOPackSelection(t *testing.T) {
	l, err := ledger.Load()
	if err != nil {
		t.Fatal(err)
	}
	selected, err := selectBFOPacks(l)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, p := range selected {
		names = append(names, p.name)
	}
	if want := []string{"bfo-cco-core", "diekm"}; !slices.Equal(names, want) {
		t.Errorf("selected packs = %v, want %v", names, want)
	}
}

// TestBFODisjointPairsFromRelease pins the pairs read from bfo-core.ttl:
// five owl:disjointWith axioms (lines 692, 816, 827, 1032, 1055) and five
// owl:AllDisjointClasses lists of 3, 3, 4, 4, and 3 members (lines
// 1185-1223), 26 pairs in all, counted by hand from the file.
//
// spec: openspec/changes/bfo-soundness-checks/specs/catalogs-and-packs/spec.md#Requirement: BFO-aligned packs anchor every entity type
func TestBFODisjointPairsFromRelease(t *testing.T) {
	pairs := loadDisjointPairs(t)
	if len(pairs) != 26 {
		t.Errorf("read %d disjoint pairs, want 26: %v", len(pairs), pairs)
	}
	for _, want := range [][2]string{
		{bfo.Continuant, bfo.Occurrent},                                           // :692
		{bfo.RealizableEntity, bfo.Quality},                                       // :827
		{bfo.IndependentContinuant, bfo.SpecificallyDependentContinuant},          // list at :1185
		{bfo.SpecificallyDependentContinuant, bfo.GenericallyDependentContinuant}, // list at :1185
		{bfo.SpatiotemporalRegion, bfo.ProcessBoundary},                           // list at :1201
	} {
		if !slices.Contains(pairs, want) {
			t.Errorf("disjoint pairs lack %v", want)
		}
	}
}

// TestBFOPacksAnchorEveryEntityType runs the bucket and disjointness checks
// over every selected pack, and pins diekm's anchors.
//
// spec: openspec/changes/bfo-soundness-checks/specs/catalogs-and-packs/spec.md#Requirement: BFO-aligned packs anchor every entity type
func TestBFOPacksAnchorEveryEntityType(t *testing.T) {
	l, err := ledger.Load()
	if err != nil {
		t.Fatal(err)
	}
	selected, err := selectBFOPacks(l)
	if err != nil {
		t.Fatal(err)
	}
	pairs := loadDisjointPairs(t)
	for _, p := range selected {
		spec, err := p.spec()
		if err != nil {
			t.Fatalf("%s: %v", p.name, err)
		}
		for _, problem := range bucketProblems(p.name, spec, pairs) {
			t.Error(problem)
		}
	}

	// Spec scenario: DIEKM is anchored.
	spec, _ := diekmSpecAndMapping(t)
	for dt, want := range map[string][]string{
		"intel.platform":    {bfo.MaterialEntity},
		"intel.observation": {bfo.Process},
	} {
		if got := bucketRootsOf(entityClassSet(spec.EntityTypes[entityTypeIndex(t, spec, dt)])); !slices.Equal(got, want) {
			t.Errorf("%s anchors at %v, want %v", dt, got, want)
		}
	}
}

// TestBFOPackAnchoringSensitivity seeds faults into diekm's intel.platform
// and requires the checks to name the entity type and the rule.
//
// spec: openspec/changes/bfo-soundness-checks/specs/catalogs-and-packs/spec.md#Requirement: BFO-aligned packs anchor every entity type
func TestBFOPackAnchoringSensitivity(t *testing.T) {
	pairs := loadDisjointPairs(t)
	tests := []struct {
		name   string
		mutate func(*vocabulary.EntityTypeSpec)
		want   []string
	}{
		{"missing bucket root", func(e *vocabulary.EntityTypeSpec) {
			e.Superclasses = []string{bfo.IndependentContinuant, bfo.Continuant, bfo.Entity}
		}, []string{"intel.platform", "has no BFO bucket root"}},
		{"continuant and occurrent collide", func(e *vocabulary.EntityTypeSpec) {
			e.Superclasses = append(e.Superclasses, bfo.Occurrent)
		}, []string{"intel.platform", "disjoint classes " + bfo.Continuant + " and " + bfo.Occurrent}},
		{"independent continuant and quality collide (AllDisjointClasses)", func(e *vocabulary.EntityTypeSpec) {
			e.Superclasses = append(e.Superclasses, bfo.Quality)
		}, []string{"intel.platform", "disjoint classes " + bfo.IndependentContinuant + " and " + bfo.SpecificallyDependentContinuant}},
		{"fiat boundary has no bucket", func(e *vocabulary.EntityTypeSpec) {
			e.Superclasses = []string{bfo.Namespace + "BFO_0000142", bfo.Namespace + "BFO_0000140",
				bfo.ImmaterialEntity, bfo.IndependentContinuant, bfo.Continuant, bfo.Entity}
		}, []string{"intel.platform", "has no BFO bucket root"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, _ := diekmSpecAndMapping(t)
			tt.mutate(&spec.EntityTypes[entityTypeIndex(t, spec, "intel.platform")])
			wantProblem(t, bucketProblems("diekm", spec, pairs), tt.want...)
		})
	}
}

// TestBFORelationBounds interprets the 17 hop relations' domains and ranges
// from bfo-core.ttl and pins three rows written out from the file by hand:
// BFO_0000196 domain IC and not SpatialRegion (:532-538), BFO_0000059 domain
// Process or SDC (:182-186), BFO_0000199 range TemporalRegion (:571).
//
// spec: openspec/changes/bfo-soundness-checks/specs/catalogs-and-packs/spec.md#Requirement: BFO projection hops follow the release
func TestBFORelationBounds(t *testing.T) {
	rel := loadRelationBounds(t)
	if len(rel) != 17 {
		t.Errorf("interpreted %d relations, want 17", len(rel))
	}
	for _, tt := range []struct {
		name string
		got  []classExpr
		want classExpr
	}{
		{"BFO_0000196 domain", rel[bfo.BearerOf].Domain,
			classExpr{AnyOf: []string{bfo.IndependentContinuant}, NoneOf: []string{bfo.SpatialRegion}}},
		{"BFO_0000059 domain", rel[bfo.Concretizes].Domain,
			classExpr{AnyOf: []string{bfo.Process, bfo.SpecificallyDependentContinuant}}},
		{"BFO_0000199 range", rel[bfo.OccupiesTemporalRegion].Range,
			classExpr{AnyOf: []string{bfo.TemporalRegion}}},
	} {
		if len(tt.got) != 1 || !slices.Equal(tt.got[0].AnyOf, tt.want.AnyOf) || !slices.Equal(tt.got[0].NoneOf, tt.want.NoneOf) {
			t.Errorf("%s = %+v, want [%+v]", tt.name, tt.got, tt.want)
		}
	}
}

// turtleReader is an objectReader over a parsed Turtle snippet.
type turtleReader map[string][]turtle.Triple

func (r turtleReader) Objects(s, p string) []turtle.Term {
	var out []turtle.Term
	for _, tr := range r[s] {
		if tr.P.Value == p {
			out = append(out, tr.O)
		}
	}
	return out
}

// TestBFORelationBoundsFailClosed feeds the interpreter class-expression
// shapes outside its three, and a relation with no domain, and requires each
// to be refused.
//
// spec: openspec/changes/bfo-soundness-checks/specs/catalogs-and-packs/spec.md#Requirement: BFO projection hops follow the release
func TestBFORelationBoundsFailClosed(t *testing.T) {
	const head = `@prefix owl: <http://www.w3.org/2002/07/owl#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix ex: <http://example.org/> .
`
	tests := []struct {
		name, ttl, want string
	}{
		{"restriction", `ex:r rdfs:domain [ a owl:Restriction ; owl:onProperty ex:p ; owl:someValuesFrom ex:C ] ; rdfs:range ex:C .`,
			"unsupported class expression"},
		{"nested union", `ex:r rdfs:domain [ owl:unionOf ( ex:A [ owl:unionOf ( ex:B ex:C ) ] ) ] ; rdfs:range ex:C .`,
			"is not a named class"},
		{"two named in intersection", `ex:r rdfs:domain [ owl:intersectionOf ( ex:A ex:B ) ] ; rdfs:range ex:C .`,
			"want 1"},
		{"bare complement", `ex:r rdfs:domain [ owl:complementOf ex:A ] ; rdfs:range ex:C .`,
			"unsupported class expression"},
		{"literal", `ex:r rdfs:domain "A" ; rdfs:range ex:C .`, "is a literal"},
		{"no domain", `ex:r rdfs:range ex:C .`, "declares no " + rdfsDomain},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			triples, err := turtle.Parse(strings.NewReader(head + tt.ttl))
			if err != nil {
				t.Fatal(err)
			}
			r := turtleReader{}
			for _, tr := range triples {
				r[tr.S.Value] = append(r[tr.S.Value], tr)
			}
			_, err = readRelationBounds(r, []string{"http://example.org/r"})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %v, want one containing %q", err, tt.want)
			}
		})
	}
}

// TestBFOProjectionHops runs the hop check over every selected pack and pins
// the hops it checks in diekm.
//
// spec: openspec/changes/bfo-soundness-checks/specs/catalogs-and-packs/spec.md#Requirement: BFO projection hops follow the release
func TestBFOProjectionHops(t *testing.T) {
	l, err := ledger.Load()
	if err != nil {
		t.Fatal(err)
	}
	selected, err := selectBFOPacks(l)
	if err != nil {
		t.Fatal(err)
	}
	rel := loadRelationBounds(t)
	for _, p := range selected {
		spec, err := p.spec()
		if err != nil {
			t.Fatalf("%s: %v", p.name, err)
		}
		m, err := packMapping(p.name, spec)
		if err != nil {
			t.Error(err)
			continue
		}
		problems, _ := hopProblems(p.name, spec, m, rel)
		for _, problem := range problems {
			t.Error(problem)
		}
		if processConnectedPacks[p.name] {
			for _, problem := range connectivityProblems(p.name, spec, m) {
				t.Error(problem)
			}
		}
	}

	// Spec scenario: DIEKM chains are sound.
	spec, m := diekmSpecAndMapping(t)
	problems, checked := hopProblems("diekm", spec, m, rel)
	if len(problems) != 0 {
		t.Errorf("diekm hop problems: %v", problems)
	}
	want := []string{
		"intel.role.assigned step 0 " + bfo.BearerOf,
		"intel.measurement.altitude-m step 0 " + bfo.BearerOf,
		"intel.measurement.altitude-m step 2 " + bfo.GenericallyDependsOn,
	}
	if !slices.Equal(checked, want) {
		t.Errorf("checked hops = %q, want %q", checked, want)
	}
}

// TestBFOProjectionHopsSensitivity seeds faults into diekm's projections and
// mapping and requires the hop check to name the predicate and step.
//
// spec: openspec/changes/bfo-soundness-checks/specs/catalogs-and-packs/spec.md#Requirement: BFO projection hops follow the release
func TestBFOProjectionHopsSensitivity(t *testing.T) {
	rel := loadRelationBounds(t)
	const (
		altitude = "intel.measurement.altitude-m"
		role     = "intel.role.assigned"
	)
	tests := []struct {
		name   string
		mutate func(*testing.T, *vocabulary.PackSpec, *shacl.Mapping)
		want   []string
	}{
		{"unanchored quality", func(t *testing.T, s *vocabulary.PackSpec, _ *shacl.Mapping) {
			s.Predicates[predicateIndex(t, *s, altitude)].Projection.Steps[0].PredicateIRI = cco.IsMeasuredBy
		}, []string{altitude + " step 0", "without bfo.BearerOf"}},
		{"unanchored role", func(t *testing.T, s *vocabulary.PackSpec, _ *shacl.Mapping) {
			s.Predicates[predicateIndex(t, *s, role)].Projection.Steps[0].PredicateIRI = bfo.ParticipatesIn
		}, []string{role + " step 0", "without bfo.BearerOf"}},
		{"carrier hop between information entities", func(t *testing.T, s *vocabulary.PackSpec, _ *shacl.Mapping) {
			st := &s.Predicates[predicateIndex(t, *s, altitude)].Projection.Steps[2]
			st.PredicateIRI, st.NodeClassIRI = bfo.IsCarrierOf, cco.InformationContentEntity
		}, []string{altitude + " step 2", "subject is outside the rdfs:domain of " + bfo.IsCarrierOf}},
		{"process bearer", func(_ *testing.T, _ *vocabulary.PackSpec, m *shacl.Mapping) {
			m.Contracts = append(m.Contracts, shacl.ContractBinding{DomainType: "intel.observation",
				Groups: []shacl.GroupBinding{{Name: "role", Predicates: []string{role}}}})
		}, []string{role + " step 0 (chain from intel.observation)", "outside the rdfs:domain of " + bfo.BearerOf}},
		{"spatial region bearer", func(t *testing.T, s *vocabulary.PackSpec, _ *shacl.Mapping) {
			s.EntityTypes[entityTypeIndex(t, *s, "intel.platform")].Superclasses = []string{bfo.SpatialRegion,
				bfo.ImmaterialEntity, bfo.IndependentContinuant, bfo.Continuant, bfo.Entity}
		}, []string{role + " step 0 (chain from intel.platform)", "outside the rdfs:domain of " + bfo.BearerOf}},
		{"projected step with no bound subject", func(_ *testing.T, _ *vocabulary.PackSpec, m *shacl.Mapping) {
			for i := range m.Contracts {
				for j := range m.Contracts[i].Groups {
					m.Contracts[i].Groups[j].Predicates = slices.DeleteFunc(m.Contracts[i].Groups[j].Predicates,
						func(p string) bool { return p == role })
				}
			}
		}, []string{"projected predicate " + role, "bound to no entity type"}},
		{"node class unknown to the catalogs", func(t *testing.T, s *vocabulary.PackSpec, _ *shacl.Mapping) {
			s.Predicates[predicateIndex(t, *s, altitude)].Projection.Steps[1].NodeClassIRI = bfo.Namespace + "BFO_0000140"
		}, []string{altitude + " step 1", "unknown to catalog/bfo and catalog/cco"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, m := diekmSpecAndMapping(t)
			tt.mutate(t, &spec, &m)
			problems, _ := hopProblems("diekm", spec, m, rel)
			wantProblem(t, problems, tt.want...)
		})
	}

	// A carrier hop names the domain only: the node is inside the range.
	spec, m := diekmSpecAndMapping(t)
	st := &spec.Predicates[predicateIndex(t, spec, altitude)].Projection.Steps[2]
	st.PredicateIRI, st.NodeClassIRI = bfo.IsCarrierOf, cco.InformationContentEntity
	problems, _ := hopProblems("diekm", spec, m, rel)
	if slices.ContainsFunc(problems, func(p string) bool { return strings.Contains(p, "rdfs:range of "+bfo.IsCarrierOf) }) {
		t.Errorf("carrier hop to an ICE named as outside the range: %v", problems)
	}

	// A BFO pack with projections and no SHACL mapping fails closed.
	if _, err := packMapping("bfo-cco-core", spec); err == nil || !strings.Contains(err.Error(), "no SHACL mapping") {
		t.Errorf("packMapping with projections and no mapping: err = %v, want a no-SHACL-mapping error", err)
	}
}

// TestBFOProcessConnectivity runs the opt-in diagnostic on diekm unchanged
// and with a realization step.
//
// spec: openspec/changes/bfo-soundness-checks/specs/catalogs-and-packs/spec.md#Requirement: Process connectivity is opt-in
func TestBFOProcessConnectivity(t *testing.T) {
	// Spec scenario: DIEKM opted in.
	spec, m := diekmSpecAndMapping(t)
	got := connectivityProblems("diekm", spec, m)
	if len(got) != 1 || !strings.Contains(got[0], "intel.platform") {
		t.Errorf("diekm connectivity = %q, want one problem naming intel.platform", got)
	}

	// Spec scenario: DIEKM with a realization step.
	i := predicateIndex(t, spec, "intel.role.assigned")
	spec.Predicates[i].Projection.Steps = append(spec.Predicates[i].Projection.Steps,
		vocabulary.ProjectionStep{PredicateIRI: bfo.RealizedIn, NodeClassIRI: cco.Act})
	if got := connectivityProblems("diekm", spec, m); len(got) != 0 {
		t.Errorf("diekm with a realization step: connectivity = %q, want none", got)
	}
	// The seeded step is itself a sound hop (Role -> has realization -> Act).
	if problems, _ := hopProblems("diekm", spec, m, loadRelationBounds(t)); len(problems) != 0 {
		t.Errorf("seeded realization step: hop problems %v", problems)
	}
}

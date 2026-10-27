// Package oracle loads the vendored ontology release files and schema bundles
// under testdata/ (BFO 2020 core, CCO v2.2 merged, W3C 2017 SOSA/SSN/SSN-System,
// OGC 3.0 SWE Common and SensorML JSON Schemas, and W3C SHACL fixtures) for
// tests. Each file's sha256 is checked against docs/admission-ledger.yaml
// before it is parsed or returned, so a test never trusts a file the ledger
// does not pin. License notices sit beside each vendored tree
// (testdata/*/NOTICE, testdata/*/LICENSE). Production code must not import this
// package.
package oracle

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"

	"github.com/c360studio/semvocab/internal/ledger"
	"github.com/c360studio/semvocab/internal/turtle"
)

// Well-known IRIs.
const (
	RDFType       = "http://www.w3.org/1999/02/22-rdf-syntax-ns#type"
	RDFSLabel     = "http://www.w3.org/2000/01/rdf-schema#label"
	RDFSSubClass  = "http://www.w3.org/2000/01/rdf-schema#subClassOf"
	OWLClass      = "http://www.w3.org/2002/07/owl#Class"
	OWLObjectProp = "http://www.w3.org/2002/07/owl#ObjectProperty"
	OWLDataProp   = "http://www.w3.org/2002/07/owl#DatatypeProperty"
	OWLDeprecated = "http://www.w3.org/2002/07/owl#deprecated"
	OWLThing      = "http://www.w3.org/2002/07/owl#Thing"
)

// Release names a vendored file by its ledger pin key.
type Release string

// The vendored releases.
const (
	BFO2020       Release = "bfo-2020"
	CCO22         Release = "cco-2.2"
	SOSA2017      Release = "sosa-2017"
	SSN2017       Release = "ssn-2017"
	SSNSystem2017 Release = "ssn-system-2017"
)

var files = map[Release]string{
	BFO2020:       filepath.Join("bfo-2020", "bfo-core.ttl"),
	CCO22:         filepath.Join("cco-2.2", "CommonCoreOntologiesMerged.ttl"),
	SOSA2017:      filepath.Join("w3c-sdw", "ssn", "integrated", "sosa.ttl"),
	SSN2017:       filepath.Join("w3c-sdw", "ssn", "integrated", "ssn.ttl"),
	SSNSystem2017: filepath.Join("w3c-sdw", "ssn", "integrated", "ssn-system.ttl"),
}

// Graph is a parsed release file indexed by subject.
type Graph struct {
	bySubject map[string][]turtle.Triple
}

var cache sync.Map // Release -> *loadResult

type loadResult struct {
	once  sync.Once
	graph *Graph
	err   error
}

// Load returns the parsed release after checking its sha256 against the
// ledger. Results are cached per process; the files are read-only.
func Load(r Release) (*Graph, error) {
	v, _ := cache.LoadOrStore(r, &loadResult{})
	res := v.(*loadResult)
	res.once.Do(func() { res.graph, res.err = load(r) })
	return res.graph, res.err
}

// LoadMerged loads every release in releases (verifying each sha256 against
// pins.ontologies) and merges their triples in argument order so
// DirectSuperclasses and Closure evaluate over an integrated owl:imports
// closure.
func LoadMerged(releases ...Release) (*Graph, error) {
	merged := &Graph{bySubject: map[string][]turtle.Triple{}}
	for _, r := range releases {
		g, err := Load(r)
		if err != nil {
			return nil, err
		}
		for s, ts := range g.bySubject {
			merged.bySubject[s] = append(merged.bySubject[s], ts...)
		}
	}
	return merged, nil
}

// Path is the absolute path of a vendored release file.
func Path(r Release) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata", files[r])
}

func load(r Release) (*Graph, error) {
	if _, ok := files[r]; !ok {
		return nil, fmt.Errorf("oracle: unknown release %q", r)
	}
	l, err := ledger.Load()
	if err != nil {
		return nil, err
	}
	pin, ok := l.Pins.Ontologies[string(r)]
	if !ok {
		return nil, fmt.Errorf("oracle: ledger pins no ontology %q", r)
	}
	data, err := os.ReadFile(Path(r))
	if err != nil {
		return nil, fmt.Errorf("oracle: %w", err)
	}
	if err := VerifySHA256(data, pin.SHA256); err != nil {
		return nil, fmt.Errorf("oracle: %s: %w", r, err)
	}
	triples, err := turtle.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("oracle: %s: %w", r, err)
	}
	g := &Graph{bySubject: map[string][]turtle.Triple{}}
	for _, t := range triples {
		g.bySubject[t.S.Value] = append(g.bySubject[t.S.Value], t)
	}
	return g, nil
}

// VerifySHA256 fails unless data hashes to want (lower-case hex).
func VerifySHA256(data []byte, want string) error {
	sum := sha256.Sum256(data)
	if got := hex.EncodeToString(sum[:]); got != want {
		return fmt.Errorf("sha256 %s does not match the ledger's %s", got, want)
	}
	return nil
}

// objects returns the objects of (s, p, ?o) in document order.
func (g *Graph) objects(s, p string) []turtle.Term {
	var out []turtle.Term
	for _, t := range g.bySubject[s] {
		if t.P.Value == p {
			out = append(out, t.O)
		}
	}
	return out
}

// Types returns the rdf:type IRIs asserted for iri.
func (g *Graph) Types(iri string) []string {
	var out []string
	for _, o := range g.objects(iri, RDFType) {
		if o.Kind == turtle.IRI {
			out = append(out, o.Value)
		}
	}
	return out
}

// HasType reports whether iri is declared with rdf:type typ.
func (g *Graph) HasType(iri, typ string) bool { return slices.Contains(g.Types(iri), typ) }

// LiteralObjects returns the lexical forms of every literal object of (s, p, ?o)
// in document order.
func (g *Graph) LiteralObjects(s, p string) []string {
	var out []string
	for _, o := range g.objects(s, p) {
		if o.Kind == turtle.Literal {
			out = append(out, o.Value)
		}
	}
	return out
}

// Labels returns the lexical forms of every rdfs:label of iri.
func (g *Graph) Labels(iri string) []string {
	return g.LiteralObjects(iri, RDFSLabel)
}

// Deprecated reports whether iri carries owl:deprecated true.
func (g *Graph) Deprecated(iri string) bool {
	for _, o := range g.objects(iri, OWLDeprecated) {
		if o.Kind == turtle.Literal && o.Value == "true" {
			return true
		}
	}
	return false
}

// DirectSuperclasses returns the named (IRI, not blank-node restriction)
// rdfs:subClassOf objects of iri, without owl:Thing, in document order.
func (g *Graph) DirectSuperclasses(iri string) []string {
	var out []string
	for _, o := range g.objects(iri, RDFSSubClass) {
		if o.Kind == turtle.IRI && o.Value != OWLThing && !slices.Contains(out, o.Value) {
			out = append(out, o.Value)
		}
	}
	return out
}

// Closure returns the transitive closure of asserted named rdfs:subClassOf
// from iri, without iri itself and without owl:Thing: direct parents first,
// then breadth-first (design D5's order).
func (g *Graph) Closure(iri string) []string {
	var out []string
	seen := map[string]bool{iri: true}
	queue := []string{iri}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, p := range g.DirectSuperclasses(cur) {
			if !seen[p] {
				seen[p] = true
				out = append(out, p)
				queue = append(queue, p)
			}
		}
	}
	return out
}

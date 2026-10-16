package shacl

import (
	"bytes"
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/c360studio/semengine/message"
	"github.com/c360studio/semengine/pkg/projection/contract"
	"github.com/c360studio/semengine/vocabulary"
	"github.com/c360studio/semvocab/internal/oracle"
	"github.com/c360studio/semvocab/internal/turtle"
)

// This file is design D15's reviewed test adapter: it runs the vendored W3C
// SHACL Core test-suite cases for the supported components through Compile
// and ValidateCandidateState, and compares the results with the expected
// report in each case file. Its rules, and every case or focus node it
// cannot map, are recorded in notes/6-validator.md.

const (
	mfNS         = "http://www.w3.org/2001/sw/DataAccess/tests/test-manifest#"
	adapterClass = "urn:semvocab:w3c-adapter#Focus"
	adapterType  = "adapter.focus"
	focusPrefix  = "w3c.suite.core.adapter.focus.f"
	otherPrefix  = "w3c.suite.core.adapter.other.o"
)

// w3cResult is one validation result in W3C terms. Terms are in N-Triples
// syntax; Path is empty when the result has none.
type w3cResult struct {
	Focus, Path, Component, Shape, Value, Severity string
}

func (r w3cResult) String() string {
	return fmt.Sprintf("focus %s path %s %s shape %s value %s %s", r.Focus, r.Path, r.Component, r.Shape, r.Value,
		r.Severity)
}

// adapted is one case after the adapter has run it.
type adapted struct {
	excluded string            // why the case cannot be mapped; empty when it ran
	dropped  map[string]string // focus node → why it cannot be mapped
	expected []w3cResult       // the case file's results, minus dropped focus nodes
	conforms bool              // the case file's sh:conforms
	results  int               // the number of results in the case file
	got      []w3cResult       // the validator's results, mapped back
}

type w3cGraph struct {
	triples   []turtle.Triple
	bySubject map[turtle.Term][]turtle.Triple
}

func (g *w3cGraph) objects(s turtle.Term, p string) []turtle.Term {
	var out []turtle.Term
	for _, t := range g.bySubject[s] {
		if t.P.Value == p {
			out = append(out, t.O)
		}
	}
	return out
}

func (g *w3cGraph) object(s turtle.Term, p string) (turtle.Term, bool) {
	o := g.objects(s, p)
	if len(o) != 1 {
		return turtle.Term{}, false
	}
	return o[0], true
}

// closure is every blank node reachable from the start terms through
// objects, with the start terms; IRIs other than the start terms are not
// followed.
func (g *w3cGraph) closure(start []turtle.Term) map[turtle.Term]bool {
	seen := map[turtle.Term]bool{}
	queue := slices.Clone(start)
	for len(queue) > 0 {
		s := queue[0]
		queue = queue[1:]
		if seen[s] {
			continue
		}
		seen[s] = true
		for _, t := range g.bySubject[s] {
			if t.O.Kind == turtle.Blank {
				queue = append(queue, t.O)
			}
		}
	}
	return seen
}

// list reads an RDF list.
func (g *w3cGraph) list(head turtle.Term) []turtle.Term {
	var out []turtle.Term
	for head.Value != rdfNil {
		first, ok1 := g.object(head, rdfFirst)
		rest, ok2 := g.object(head, rdfRest)
		if !ok1 || !ok2 {
			return out
		}
		out = append(out, first)
		head = rest
	}
	return out
}

// ntriples writes a reader term in N-Triples syntax, as Result.Value does.
func ntriples(t turtle.Term) string {
	switch t.Kind {
	case turtle.IRI:
		return term{kind: termIRI, value: t.Value}.String()
	case turtle.Blank:
		return "a blank node"
	}
	if t.Lang != "" {
		return term{kind: termLiteral, value: t.Value, datatype: xsdString}.String() + "@" + t.Lang
	}
	return term{kind: termLiteral, value: t.Value, datatype: t.Datatype}.String()
}

// nativeValue maps a data-graph object to the Go value whose D9 term is the
// same RDF term, or says why there is none. Only terms that round-trip
// exactly are mapped, so an expected sh:value compares with Result.Value as
// written in the case file.
func nativeValue(o turtle.Term) (any, string) {
	switch {
	case o.Kind == turtle.IRI:
		if !strings.HasPrefix(o.Value, "http://") && !strings.HasPrefix(o.Value, "https://") &&
			!strings.HasPrefix(o.Value, "urn:") {
			return nil, "IRI " + o.Value + " has a scheme export writes as a literal"
		}
		return o.Value, ""
	case o.Kind == turtle.Blank:
		return nil, "a blank-node object has no stable native label"
	case o.Lang != "":
		return nil, fmt.Sprintf("value %s is language-tagged; no Go value exports as rdf:langString", ntriples(o))
	case o.Datatype == xsdString:
		if message.IsValidEntityID(o.Value) || isExportIRI(o.Value) || isBlankNodeLabel(o.Value) {
			return nil, fmt.Sprintf("string %s would export as an IRI or blank node", ntriples(o))
		}
		return o.Value, ""
	case o.Datatype == xsdInt:
		n, err := strconv.ParseInt(o.Value, 10, 64)
		if err != nil || strconv.FormatInt(n, 10) != o.Value {
			return nil, fmt.Sprintf("integer %s is not a canonical int64", ntriples(o))
		}
		return n, ""
	case o.Datatype == xsdBool && (o.Value == "true" || o.Value == "false"):
		return o.Value == "true", ""
	}
	return nil, fmt.Sprintf("value %s has a datatype no mapped Go value exports as", ntriples(o))
}

func iriTerm(v string) turtle.Term { return turtle.Term{Kind: turtle.IRI, Value: v} }

func byTerm(a, b turtle.Term) int {
	return cmp.Or(cmp.Compare(a.Kind, b.Kind), strings.Compare(a.Value, b.Value))
}

// loadW3CGraph reads one vendored case (sha256 checked) with a base, and
// removes exact repeats, since an RDF graph is a set.
func loadW3CGraph(t *testing.T, rel string) *w3cGraph {
	t.Helper()
	data, err := oracle.ReadFixture(oracle.W3CDataShapes, "data-shapes-test-suite/tests/core/"+rel)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := turtle.Parse(bytes.NewReader(data), turtle.WithBase("http://semvocab.test/core/"+rel))
	if err != nil {
		t.Fatal(err)
	}
	g := &w3cGraph{bySubject: map[turtle.Term][]turtle.Triple{}}
	seen := map[turtle.Triple]bool{}
	for _, tr := range parsed {
		if !seen[tr] {
			seen[tr] = true
			g.triples = append(g.triples, tr)
			g.bySubject[tr.S] = append(g.bySubject[tr.S], tr)
		}
	}
	return g
}

// readExpected reads the manifest's one entry and its expected report. It
// returns the manifest's subjects (root, entry, and the blank nodes under
// them), so the adapter can leave them out of the shapes and data graphs.
func (g *w3cGraph) readExpected(t *testing.T, out *adapted) map[turtle.Term]bool {
	t.Helper()
	var root turtle.Term
	for _, tr := range g.triples {
		if tr.P.Value == rdfType && tr.O.Value == mfNS+"Manifest" {
			root = tr.S
		}
	}
	entries := g.list(mustObject(t, g, root, mfNS+"entries"))
	if len(entries) != 1 {
		t.Fatalf("%d manifest entries, want 1", len(entries))
	}
	report := mustObject(t, g, entries[0], mfNS+"result")
	out.conforms = mustObject(t, g, report, shNS+"conforms").Value == "true"
	for _, r := range g.objects(report, shNS+"result") {
		res := w3cResult{
			Focus:     ntriples(mustObject(t, g, r, shNS+"focusNode")),
			Component: mustObject(t, g, r, shNS+"sourceConstraintComponent").Value,
			Shape:     ntriples(mustObject(t, g, r, shNS+"sourceShape")),
			Severity:  mustObject(t, g, r, shNS+"resultSeverity").Value,
		}
		if p, ok := g.object(r, shNS+"resultPath"); ok {
			res.Path = p.Value
		}
		if v, ok := g.object(r, shNS+"value"); ok {
			res.Value = ntriples(v)
		}
		out.expected = append(out.expected, res)
	}
	out.results = len(out.expected)
	return g.closure(append([]turtle.Term{root}, entries...))
}

// nodeShapes is every subject outside the manifest typed sh:NodeShape or
// carrying sh:targetNode or sh:targetClass, sorted.
func (g *w3cGraph) nodeShapes(manifest map[turtle.Term]bool) []turtle.Term {
	var out []turtle.Term
	for s, ts := range g.bySubject {
		if !manifest[s] && slices.ContainsFunc(ts, func(tr turtle.Triple) bool {
			return tr.P.Value == rdfType && tr.O.Value == shNS+"NodeShape" ||
				tr.P.Value == shNS+"targetNode" || tr.P.Value == shNS+"targetClass"
		}) {
			out = append(out, s)
		}
	}
	slices.SortFunc(out, byTerm)
	return out
}

// focusNodes follows SHACL's targets (§2.1.3): the sh:targetNode objects, and
// the instances of each sh:targetClass and of an implicit class target,
// through rdfs:subClassOf. They are sorted.
func (g *w3cGraph) focusNodes(shape turtle.Term) []turtle.Term {
	set := map[turtle.Term]bool{}
	for _, n := range g.objects(shape, shNS+"targetNode") {
		set[n] = true
	}
	classes := g.objects(shape, shNS+"targetClass")
	if slices.Contains(g.objects(shape, rdfType), iriTerm(rdfsClass)) {
		classes = append(classes, shape)
	}
	for _, tr := range g.triples {
		if tr.P.Value == rdfType && slices.ContainsFunc(classes, func(c turtle.Term) bool { return g.subClassOf(tr.O, c) }) {
			set[tr.S] = true
		}
	}
	out := slices.Collect(maps.Keys(set))
	slices.SortFunc(out, byTerm)
	return out
}

// adaptedShapes is the shapes graph with the node shape's targets replaced by
// one sh:targetClass of the adapter's class. addDatatype, when set, is added
// as sh:datatype to each property shape that has a bound and no datatype.
func (g *w3cGraph) adaptedShapes(shape turtle.Term, closure map[turtle.Term]bool, addDatatype string) []turtle.Triple {
	var out []turtle.Triple
	for _, tr := range g.triples {
		if !closure[tr.S] || tr.S == shape && (tr.P.Value == shNS+"targetNode" || tr.P.Value == shNS+"targetClass" ||
			tr.P.Value == rdfType && tr.O.Value == rdfsClass || tr.P.Value == rdfsNS+"subClassOf") {
			continue
		}
		out = append(out, tr)
	}
	out = append(out, turtle.Triple{S: shape, P: iriTerm(shNS + "targetClass"), O: iriTerm(adapterClass)})
	if addDatatype == "" {
		return out
	}
	for _, ps := range g.objects(shape, shNS+"property") {
		hasBound := slices.ContainsFunc([]string{"minInclusive", "maxInclusive", "minExclusive", "maxExclusive"},
			func(b string) bool { return len(g.objects(ps, shNS+b)) > 0 })
		if hasBound && len(g.objects(ps, shNS+"datatype")) == 0 {
			out = append(out, turtle.Triple{S: ps, P: iriTerm(shNS + "datatype"), O: iriTerm(addDatatype)})
		}
	}
	return out
}

// adapterBundle compiles the shapes against a synthetic pack: each path
// becomes a predicate adapter.path.p<n> with that iri and no data_type, the
// adapter's class maps to adapter.focus, and one reconcile group holds every
// predicate. It returns the path → predicate map with the bundle.
func adapterBundle(set ShapeSet, paths map[string]bool) (*Bundle, map[string]string, error) {
	for _, ns := range set.NodeShapes {
		for _, p := range ns.Properties {
			paths[p.Path] = true
		}
	}
	pack := vocabulary.PackSpec{Name: "w3c-adapter",
		EntityTypes: []vocabulary.EntityTypeSpec{{DomainType: adapterType, ClassIRI: adapterClass}}}
	m := Mapping{EntityTypes: []EntityTypeBinding{{ClassIRI: adapterClass, DomainTypes: []string{adapterType}}}}
	predicate := map[string]string{}
	var group []string
	for i, p := range sortedKeys(paths) {
		name := fmt.Sprintf("adapter.path.p%d", i+1)
		predicate[p] = name
		pack.Predicates = append(pack.Predicates, vocabulary.PredicateSpec{Name: name, IRI: p})
		m.Predicates = append(m.Predicates, PredicateBinding{PathIRI: p, Name: name})
		group = append(group, name)
	}
	m.Contracts = []ContractBinding{{DomainType: adapterType,
		Groups: []GroupBinding{{Mode: contract.ModeReconcile, Predicates: group}}}}
	b, err := Compile(pack, set, m)
	return b, predicate, err
}

// runW3C is the adapter: it runs one case and returns its expected and
// actual results in W3C terms, or the reason it cannot run the case.
// addDatatype selects the declared variant (adaptedShapes); the
// sh:DatatypeConstraintComponent results the addition causes are then not
// compared.
func runW3C(t *testing.T, rel, addDatatype string) adapted {
	t.Helper()
	g := loadW3CGraph(t, rel)
	out := adapted{dropped: map[string]string{}}
	manifest := g.readExpected(t, &out)

	shapes := g.nodeShapes(manifest)
	if len(shapes) != 1 {
		out.excluded = fmt.Sprintf("%d node shapes; the adapter maps one", len(shapes))
		return out
	}
	shape := shapes[0]
	closure := g.closure(append([]turtle.Term{shape}, g.objects(shape, shNS+"property")...))
	focus := g.focusNodes(shape)
	for _, n := range focus {
		if n.Kind != turtle.IRI {
			out.excluded = fmt.Sprintf("focus node %s is not an IRI; native focus nodes are entities", ntriples(n))
			return out
		}
	}
	set, err := fromTriples(g.adaptedShapes(shape, closure, addDatatype))
	if err != nil {
		out.excluded = err.Error()
		return out
	}

	// Every other IRI subject outside the manifest and the shapes graph that
	// is not a class becomes an entity of an untargeted type.
	var other []turtle.Term
	paths := map[string]bool{}
	for s, ts := range g.bySubject {
		if !manifest[s] && !closure[s] && !slices.Contains(focus, s) && s.Kind == turtle.IRI &&
			!slices.Contains(g.objects(s, rdfType), iriTerm(rdfsClass)) {
			other = append(other, s)
		}
		if !manifest[s] && !closure[s] {
			for _, tr := range ts {
				if tr.P.Value != rdfType {
					paths[tr.P.Value] = true
				}
			}
		}
	}
	slices.SortFunc(other, byTerm)
	b, predicate, err := adapterBundle(set, paths)
	if err != nil {
		out.excluded = err.Error()
		return out
	}
	run := w3cRun{t: t, g: g, b: b, predicate: predicate, addDatatype: addDatatype, out: &out}
	run.validate(focus, focusPrefix, true)
	run.validate(other, otherPrefix, false)

	// A blank source shape is a blank property shape, which the validator
	// names by its node shape (notes/5-compiler.md).
	var expected []w3cResult
	for _, r := range out.expected {
		if r.Shape == "a blank node" {
			r.Shape = ntriples(shape)
		}
		if _, gone := out.dropped[r.Focus]; !gone {
			expected = append(expected, r)
		}
	}
	out.expected = expected
	return out
}

// w3cRun validates the case's subjects as native entities.
type w3cRun struct {
	t           *testing.T
	g           *w3cGraph
	b           *Bundle
	predicate   map[string]string
	addDatatype string
	out         *adapted
}

// validate turns each subject into an entity <prefix><n>, its data triples
// into native triples, and maps the results back to W3C terms. A subject
// with a value nativeValue cannot map is dropped, with the reason.
func (r *w3cRun) validate(subjects []turtle.Term, prefix string, targeted bool) {
	r.t.Helper()
	for i, s := range subjects {
		id := fmt.Sprintf("%s%d", prefix, i+1)
		var triples []message.Triple
		why := ""
		for _, tr := range r.g.bySubject[s] {
			if tr.P.Value == rdfType {
				continue // the entity's type stands for it (design D12)
			}
			var v any
			if v, why = nativeValue(tr.O); why != "" {
				break
			}
			triples = append(triples, message.Triple{Subject: id, Predicate: r.predicate[tr.P.Value], Object: v})
		}
		if why != "" {
			r.out.dropped[ntriples(s)] = why
			continue
		}
		rep, err := r.b.ValidateCandidateState(id, triples)
		if err != nil {
			r.t.Fatalf("%s as %s: %v", ntriples(s), id, err)
		}
		if rep.Targeted != targeted {
			r.t.Errorf("%s as %s: Targeted %v, want %v", ntriples(s), id, rep.Targeted, targeted)
		}
		for _, res := range rep.Results {
			if r.addDatatype != "" && res.Component == shNS+"DatatypeConstraintComponent" {
				continue
			}
			r.out.got = append(r.out.got, w3cResult{Focus: ntriples(s), Path: res.Path, Component: res.Component,
				Shape: "<" + res.SourceShape + ">", Value: res.Value, Severity: res.Severity})
		}
	}
}

// subClassOf reports whether class is c or reaches c through rdfs:subClassOf.
func (g *w3cGraph) subClassOf(class, c turtle.Term) bool {
	seen := map[turtle.Term]bool{}
	queue := []turtle.Term{class}
	for len(queue) > 0 {
		x := queue[0]
		queue = queue[1:]
		if x == c {
			return true
		}
		if seen[x] {
			continue
		}
		seen[x] = true
		queue = append(queue, g.objects(x, rdfsNS+"subClassOf")...)
	}
	return false
}

func mustObject(t *testing.T, g *w3cGraph, s turtle.Term, p string) turtle.Term {
	t.Helper()
	o, ok := g.object(s, p)
	if !ok {
		t.Fatalf("%v has %d values for %s, want 1", s, len(g.objects(s, p)), p)
	}
	return o
}

func sortW3C(rs []w3cResult) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.String()
	}
	slices.Sort(out)
	return out
}

// TestW3CSupportedCases is task 6.3: every vendored case for a supported
// component runs through the adapter, and its results equal the expected
// report in the case file, or the case is excluded for the reason listed
// here. The adapter must find each listed reason itself.
func TestW3CSupportedCases(t *testing.T) {
	const (
		boundNeedsDatatype = "a bound needs an sh:datatype (xsd:integer or xsd:double) on its property shape"
	)
	for _, tt := range []struct {
		rel         string
		addDatatype string            // the declared variant: sh:datatype added to bound property shapes
		excluded    string            // the reason the case cannot run; empty when it runs
		dropped     map[string]string // focus node → reason
		results     int               // results in the case file, counted by hand
	}{
		{rel: "property/minCount-001.ttl", results: 1},
		{rel: "property/minCount-002.ttl", results: 0},
		{rel: "property/maxCount-001.ttl", results: 1},
		{rel: "property/maxCount-002.ttl", results: 1},
		{rel: "property/datatype-001.ttl",
			excluded: "<http://www.w3.org/2001/XMLSchema#date> is not in the datatype table"},
		{rel: "property/datatype-002.ttl", results: 2, dropped: map[string]string{
			"<http://datashapes.org/sh/tests/core/property/datatype-002.test#InvalidInstance1>": `value "A"@en is language-tagged`,
		}},
		{rel: "property/minExclusive-001.ttl", excluded: boundNeedsDatatype},
		{rel: "property/minExclusive-001.ttl", addDatatype: xsdInt, results: 2},
		{rel: "property/maxExclusive-001.ttl", excluded: boundNeedsDatatype},
		{rel: "property/maxExclusive-001.ttl", addDatatype: xsdInt, results: 3},
		{rel: "property/maxInclusive-001.ttl", excluded: boundNeedsDatatype},
		{rel: "property/maxInclusive-001.ttl", addDatatype: xsdInt, results: 2},
		{rel: "node/minInclusive-001.ttl", excluded: `focus node "7"^^<` + xsdInt + `> is not an IRI`},
		{rel: "property/pattern-001.ttl", results: 2},
		{rel: "property/in-001.ttl", results: 1},
		{rel: "node/closed-001.ttl", excluded: "a closed shape must list rdf:type in sh:ignoredProperties"},
		{rel: "node/closed-002.ttl", results: 1},
		{rel: "targets/targetClass-001.ttl", results: 1},
	} {
		name := tt.rel
		if tt.addDatatype != "" {
			name += " with sh:datatype added"
		}
		t.Run(name, func(t *testing.T) {
			got := runW3C(t, tt.rel, tt.addDatatype)
			if tt.excluded != "" || got.excluded != "" {
				if tt.excluded == "" || !strings.Contains(got.excluded, tt.excluded) {
					t.Fatalf("excluded for %q, want %q", got.excluded, tt.excluded)
				}
				return
			}
			if len(got.dropped) != len(tt.dropped) {
				t.Errorf("dropped focus nodes %v, want %v", got.dropped, tt.dropped)
			}
			for focus, why := range tt.dropped {
				if !strings.Contains(got.dropped[focus], why) {
					t.Errorf("focus %s dropped for %q, want %q", focus, got.dropped[focus], why)
				}
			}
			if got.results != tt.results {
				t.Errorf("read %d results from the case file, want %d", got.results, tt.results)
			}
			if got.conforms != (got.results == 0) {
				t.Errorf("the case file says conforms %v with %d results", got.conforms, got.results)
			}
			if want, have := sortW3C(got.expected), sortW3C(got.got); !slices.Equal(want, have) {
				t.Errorf("results:\n  %s\nwant (from the case file):\n  %s", strings.Join(have, "\n  "),
					strings.Join(want, "\n  "))
			}
		})
	}
}

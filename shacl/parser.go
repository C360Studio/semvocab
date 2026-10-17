package shacl

import (
	"bytes"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/c360studio/semvocab/internal/turtle"
)

// maxTurtleBytes caps a shapes document (design D2, ruling Q25). The reader
// itself has no size limit, because it also loads large vendored oracles.
const maxTurtleBytes = 1 << 20

// IRIs the parser recognises. SemEngine exports none of them (design,
// ownership inventory).
const (
	shNS      = "http://www.w3.org/ns/shacl#"
	rdfNS     = "http://www.w3.org/1999/02/22-rdf-syntax-ns#"
	rdfsNS    = "http://www.w3.org/2000/01/rdf-schema#"
	xsdNS     = "http://www.w3.org/2001/XMLSchema#"
	rdfType   = rdfNS + "type"
	rdfFirst  = rdfNS + "first"
	rdfRest   = rdfNS + "rest"
	rdfNil    = rdfNS + "nil"
	rdfsClass = rdfsNS + "Class"
	xsdString = xsdNS + "string"
	xsdInt    = xsdNS + "integer"
	xsdBool   = xsdNS + "boolean"
)

// targetPredicates make a subject a node shape (ruling Q27), whether or not
// the compiler supports the target.
var targetPredicates = []string{
	shNS + "targetClass", shNS + "targetNode", shNS + "targetSubjectsOf", shNS + "targetObjectsOf", shNS + "target",
}

// annotations are accepted on any shape and carry no constraint (design D4).
var annotations = []string{
	shNS + "name", shNS + "description", shNS + "order", shNS + "defaultValue", rdfsNS + "label", rdfsNS + "comment",
}

// ParseTurtle reads a shapes document in the Turtle subset internal/turtle
// accepts, with absolute IRIs only. It fails on input over 1 MiB (naming the
// byte offset where the limit is crossed), on Turtle the reader rejects (with
// line and column), and on a node shape that is a blank node. Exact repeated
// triples count once, because an RDF graph is a set (design D3). Constructs
// outside the compiler's profile do not fail parsing: they are recorded in
// the ShapeSet for Compile to report (ruling Q26). No triple is skipped
// silently.
func ParseTurtle(r io.Reader) (ShapeSet, error) {
	src, err := io.ReadAll(io.LimitReader(r, maxTurtleBytes+1))
	if err != nil {
		return ShapeSet{}, fmt.Errorf("shacl: read: %w", err)
	}
	if len(src) > maxTurtleBytes {
		return ShapeSet{}, fmt.Errorf("shacl: input exceeds the %d-byte limit at byte offset %d",
			maxTurtleBytes, maxTurtleBytes)
	}
	triples, err := turtle.Parse(bytes.NewReader(src))
	if err != nil {
		return ShapeSet{}, fmt.Errorf("shacl: %w", err)
	}
	return fromTriples(triples)
}

// graph indexes a deduplicated triple set and tracks which subjects the
// builder has accounted for.
type graph struct {
	bySubject  map[turtle.Term][]turtle.Triple
	subjects   []turtle.Term // in first-seen order
	isObject   map[turtle.Term]bool
	consumed   map[turtle.Term]bool // shapes and list cells built into the model
	owned      map[turtle.Term]bool // blank nodes under a recorded rejection
	ownRoots   []turtle.Term
	rejections []rejection
}

// fromTriples builds the model from parsed triples. ParseTurtle and the W3C
// test-suite loader (which parses with a base) share it.
func fromTriples(triples []turtle.Triple) (ShapeSet, error) {
	g := &graph{
		bySubject: map[turtle.Term][]turtle.Triple{}, isObject: map[turtle.Term]bool{},
		consumed: map[turtle.Term]bool{}, owned: map[turtle.Term]bool{},
	}
	seen := map[turtle.Triple]bool{}
	for _, t := range triples {
		if seen[t] {
			continue
		}
		seen[t] = true
		if _, ok := g.bySubject[t.S]; !ok {
			g.subjects = append(g.subjects, t.S)
		}
		g.bySubject[t.S] = append(g.bySubject[t.S], t)
		g.isObject[t.O] = true
	}

	var set ShapeSet
	for _, s := range g.subjects {
		if !g.isNodeShape(s) {
			continue
		}
		if s.Kind != turtle.IRI {
			return ShapeSet{}, fmt.Errorf("shacl: blank-node node shape with %s: node shapes must be IRIs (design D3)",
				strings.Join(g.predicates(s), ", "))
		}
		g.consumed[s] = true
	}
	for _, s := range g.subjects {
		if g.consumed[s] && s.Kind == turtle.IRI && g.isNodeShape(s) {
			set.NodeShapes = append(set.NodeShapes, g.nodeShape(s))
		}
	}
	g.recordStray()
	set.rejections = g.rejections
	return normalize(set), nil
}

func (g *graph) predicates(s turtle.Term) []string {
	var out []string
	for _, t := range g.bySubject[s] {
		if !slices.Contains(out, t.P.Value) {
			out = append(out, t.P.Value)
		}
	}
	slices.Sort(out)
	return out
}

func (g *graph) isNodeShape(s turtle.Term) bool {
	for _, t := range g.bySubject[s] {
		if t.P.Value == rdfType && t.O.Kind == turtle.IRI && t.O.Value == shNS+"NodeShape" ||
			slices.Contains(targetPredicates, t.P.Value) {
			return true
		}
	}
	return false
}

// reject records an out-of-profile use and hands the object's blank-node
// subgraph to it, so nothing under the construct is reported again.
func (g *graph) reject(construct, shape, path, detail string, obj turtle.Term) {
	g.rejections = append(g.rejections, rejection{Construct: construct, Shape: shape, Path: path, Detail: detail})
	g.own(obj)
}

func (g *graph) own(t turtle.Term) {
	if t.Kind == turtle.Blank {
		g.ownRoots = append(g.ownRoots, t)
	}
}

func (g *graph) nodeShape(s turtle.Term) NodeShape {
	ns := NodeShape{ID: s.Value}
	var closed, ignored, severity []turtle.Term
	for _, t := range g.bySubject[s] {
		p, o := t.P.Value, t.O
		switch {
		case p == rdfType && o.Kind == turtle.IRI && o.Value == shNS+"NodeShape":
		case p == rdfType && o.Kind == turtle.IRI && o.Value == rdfsClass:
			g.reject(rdfsClass, ns.ID, "", "implicit class target: the shape is also an rdfs:Class", o)
		case p == shNS+"targetClass" && o.Kind == turtle.IRI:
			ns.TargetClasses = append(ns.TargetClasses, o.Value)
		case p == shNS+"property" && o.Kind != turtle.Literal:
			if g.consumed[o] && o.Kind == turtle.Blank {
				g.reject(p, ns.ID, "", "a blank-node property shape shared by several shapes", turtle.Term{})
				continue
			}
			g.consumed[o] = true
			ns.Properties = append(ns.Properties, g.propertyShape(ns.ID, o))
		case p == shNS+"closed":
			closed = append(closed, o)
		case p == shNS+"ignoredProperties":
			ignored = append(ignored, o)
		case p == shNS+"severity":
			severity = append(severity, o)
		case p == shNS+"message" && isString(o):
			ns.Messages = append(ns.Messages, o.Value)
		case slices.Contains(annotations, p):
			g.own(o)
		case p == shNS+"path" && o.Kind == turtle.Blank:
			// A property shape with its own target: name the path construct too.
			g.reject(p, ns.ID, "", "sh:path on a shape with a target", turtle.Term{})
			g.complexPath(ns.ID, o)
		default:
			g.reject(p, ns.ID, "", describe(o), o)
		}
	}
	if v, ok := g.single(shNS+"closed", ns.ID, "", closed); ok {
		switch {
		case v.Kind == turtle.Literal && v.Datatype == xsdBool && (v.Value == "true" || v.Value == "1"):
			ns.Closed = true
		case v.Kind == turtle.Literal && v.Datatype == xsdBool && (v.Value == "false" || v.Value == "0"):
		default:
			g.reject(shNS+"closed", ns.ID, "", describe(v)+" is not an xsd:boolean literal", v)
		}
	}
	if v, ok := g.single(shNS+"ignoredProperties", ns.ID, "", ignored); ok {
		items, ok := g.list(v)
		for _, it := range items {
			if it.Kind != turtle.IRI {
				ok = false
			}
		}
		if ok {
			for _, it := range items {
				ns.IgnoredProperties = append(ns.IgnoredProperties, it.Value)
			}
		} else {
			g.reject(shNS+"ignoredProperties", ns.ID, "", "not a well-formed list of IRIs", v)
		}
	}
	ns.Severity = g.severity(ns.ID, "", severity)
	return ns
}

func (g *graph) propertyShape(nodeID string, s turtle.Term) PropertyShape {
	var ps PropertyShape
	if s.Kind == turtle.IRI {
		ps.ID = s.Value
	}
	shape := ps.ID
	if shape == "" {
		shape = nodeID
	}
	byPred := map[string][]turtle.Term{}
	var rest []turtle.Triple
	single := []string{"path", "minCount", "maxCount", "datatype", "minInclusive", "maxInclusive", "minExclusive",
		"maxExclusive", "pattern", "in", "severity"}
	for _, t := range g.bySubject[s] {
		if slices.Contains(single, strings.TrimPrefix(t.P.Value, shNS)) && strings.HasPrefix(t.P.Value, shNS) {
			byPred[t.P.Value] = append(byPred[t.P.Value], t.O)
			continue
		}
		rest = append(rest, t)
	}

	// The path comes first: every later record names it.
	paths := byPred[shNS+"path"]
	switch {
	case len(paths) == 0:
		g.reject(shNS+"path", shape, "", "property shape has no sh:path", turtle.Term{})
	case len(paths) > 1:
		g.reject(shNS+"path", shape, "", "property shape has several sh:path values", turtle.Term{})
		for _, p := range paths {
			g.own(p)
		}
	case paths[0].Kind == turtle.IRI:
		ps.Path = paths[0].Value
	default:
		g.complexPath(shape, paths[0])
	}
	path := ps.Path

	for _, t := range rest {
		p, o := t.P.Value, t.O
		switch {
		case p == rdfType && o.Kind == turtle.IRI && o.Value == shNS+"PropertyShape":
		case p == shNS+"message" && isString(o):
			ps.Messages = append(ps.Messages, o.Value)
		case slices.Contains(annotations, p):
			g.own(o)
		default:
			g.reject(p, shape, path, describe(o), o)
		}
	}
	if v, ok := g.single(shNS+"minCount", shape, path, byPred[shNS+"minCount"]); ok {
		if n, ok := g.count(shNS+"minCount", shape, path, v); ok {
			ps.MinCount = n
		}
	}
	if v, ok := g.single(shNS+"maxCount", shape, path, byPred[shNS+"maxCount"]); ok {
		if n, ok := g.count(shNS+"maxCount", shape, path, v); ok {
			ps.MaxCount = &n
		}
	}
	if v, ok := g.single(shNS+"datatype", shape, path, byPred[shNS+"datatype"]); ok {
		if v.Kind == turtle.IRI {
			ps.Datatype = v.Value
		} else {
			g.reject(shNS+"datatype", shape, path, describe(v)+" is not an IRI", v)
		}
	}
	for _, b := range []struct {
		name string
		dst  **Literal
	}{
		{"minInclusive", &ps.MinInclusive}, {"maxInclusive", &ps.MaxInclusive},
		{"minExclusive", &ps.MinExclusive}, {"maxExclusive", &ps.MaxExclusive},
	} {
		if v, ok := g.single(shNS+b.name, shape, path, byPred[shNS+b.name]); ok {
			if v.Kind == turtle.Literal {
				*b.dst = &Literal{Lexical: v.Value, Datatype: v.Datatype}
			} else {
				g.reject(shNS+b.name, shape, path, describe(v)+" is not a literal", v)
			}
		}
	}
	if v, ok := g.single(shNS+"pattern", shape, path, byPred[shNS+"pattern"]); ok {
		if isString(v) {
			ps.Pattern = &v.Value
		} else {
			g.reject(shNS+"pattern", shape, path, describe(v)+" is not an xsd:string literal", v)
		}
	}
	if v, ok := g.single(shNS+"in", shape, path, byPred[shNS+"in"]); ok {
		ps.In = g.in(shape, path, v)
	}
	ps.Severity = g.severity(shape, path, byPred[shNS+"severity"])
	return ps
}

// complexPath records a path that is not one predicate IRI, by the path
// construct that introduces it.
func (g *graph) complexPath(shape string, p turtle.Term) {
	if p.Kind == turtle.Blank {
		preds := g.predicates(p)
		if slices.Contains(preds, rdfFirst) {
			g.reject(shNS+"path", shape, "", "sequence path (an RDF list)", p)
			return
		}
		if len(preds) == 1 && strings.HasPrefix(preds[0], shNS) {
			g.reject(preds[0], shape, "", "path", p)
			return
		}
	}
	g.reject(shNS+"path", shape, "", describe(p)+" is not a predicate IRI", p)
}

// single returns the one value of a functional parameter. Several distinct
// values are recorded as a conflict.
func (g *graph) single(construct, shape, path string, vals []turtle.Term) (turtle.Term, bool) {
	switch len(vals) {
	case 0:
		return turtle.Term{}, false
	case 1:
		return vals[0], true
	}
	var ds []string
	for _, v := range vals {
		ds = append(ds, describe(v))
		g.own(v)
	}
	slices.Sort(ds)
	g.reject(construct, shape, path, "several values: "+strings.Join(ds, ", "), turtle.Term{})
	return turtle.Term{}, false
}

func (g *graph) count(construct, shape, path string, v turtle.Term) (int, bool) {
	if v.Kind == turtle.Literal && v.Datatype == xsdInt {
		if n, err := strconv.ParseInt(v.Value, 10, 64); err == nil {
			return int(n), true
		}
		g.reject(construct, shape, path, describe(v)+" is out of range", v)
		return 0, false
	}
	g.reject(construct, shape, path, describe(v)+" is not an xsd:integer literal", v)
	return 0, false
}

func (g *graph) severity(shape, path string, vals []turtle.Term) string {
	v, ok := g.single(shNS+"severity", shape, path, vals)
	if !ok {
		return ""
	}
	if v.Kind != turtle.IRI {
		g.reject(shNS+"severity", shape, path, describe(v)+" is not an IRI", v)
		return ""
	}
	return v.Value
}

func (g *graph) in(shape, path string, head turtle.Term) *[]string {
	items, ok := g.list(head)
	if !ok {
		g.reject(shNS+"in", shape, path, "not a well-formed RDF list", head)
		return nil
	}
	members := []string{}
	bad := false
	for _, it := range items {
		if !isString(it) {
			g.reject(shNS+"in", shape, path, "member "+describe(it)+" is not an xsd:string literal", it)
			bad = true
			continue
		}
		members = append(members, it.Value)
	}
	if bad {
		return nil
	}
	return &members
}

// list reads a well-formed RDF list: blank cells, each with exactly one
// rdf:first and one rdf:rest and nothing else, ending at rdf:nil, with no
// cycle. Cycle detection is local to this walk; the cells are then marked
// consumed for triple accounting only, so a list read again (a named property
// shape used by several node shapes) reads the same way. A malformed list
// consumes nothing, and the caller records it.
func (g *graph) list(head turtle.Term) ([]turtle.Term, bool) {
	var items []turtle.Term
	var cells []turtle.Term
	visited := map[turtle.Term]bool{}
	for cur := head; !(cur.Kind == turtle.IRI && cur.Value == rdfNil); {
		ts := g.bySubject[cur]
		if cur.Kind != turtle.Blank || visited[cur] || len(ts) != 2 {
			return nil, false
		}
		visited[cur] = true
		var first, rest *turtle.Term
		for i := range ts {
			switch ts[i].P.Value {
			case rdfFirst:
				first = &ts[i].O
			case rdfRest:
				rest = &ts[i].O
			}
		}
		if first == nil || rest == nil {
			return nil, false
		}
		items = append(items, *first)
		cells = append(cells, cur)
		cur = *rest
	}
	for _, c := range cells {
		g.consumed[c] = true
	}
	return items, true
}

// recordStray records every triple whose subject is not a shape or a list
// cell the model used (design D4), except triples under a blank node already
// handed to another record. Records name the subject when it is an IRI.
// Blank subjects reached from nothing are reported first, each owning the
// blank nodes under it. The blank subjects left after that lie on or under a
// cycle that nothing reports reaches; which cycle node is met first depends
// on triple order, so each of them reports its own triples and none owns
// another. Either way the records do not depend on triple order.
func (g *graph) recordStray() {
	g.closeOwned()
	for _, s := range g.subjects {
		if s.Kind == turtle.IRI && !g.consumed[s] {
			g.rejectSubject(s.Value, s)
		}
	}
	g.closeOwned()
	for _, s := range g.subjects {
		if g.unclaimedBlank(s) && !g.isObject[s] {
			g.rejectSubject("", s)
			g.closeOwned()
		}
	}
	var cyclic []turtle.Term
	for _, s := range g.subjects {
		if g.unclaimedBlank(s) {
			cyclic = append(cyclic, s)
		}
	}
	for _, s := range cyclic {
		g.rejectSubject("", s)
	}
	g.closeOwned()
}

// unclaimedBlank reports a blank subject no shape, list, or record holds.
func (g *graph) unclaimedBlank(s turtle.Term) bool {
	return s.Kind == turtle.Blank && !g.consumed[s] && !g.owned[s]
}

func (g *graph) rejectSubject(shape string, s turtle.Term) {
	for _, t := range g.bySubject[s] {
		g.reject(t.P.Value, shape, "", "the subject is not a node shape, property shape, or list cell they use", t.O)
	}
	g.owned[s] = true
}

// closeOwned extends owned over every blank node reachable from ownRoots.
func (g *graph) closeOwned() {
	for len(g.ownRoots) > 0 {
		n := g.ownRoots[len(g.ownRoots)-1]
		g.ownRoots = g.ownRoots[:len(g.ownRoots)-1]
		if g.owned[n] || g.consumed[n] {
			continue
		}
		g.owned[n] = true
		for _, t := range g.bySubject[n] {
			g.own(t.O)
		}
	}
}

func isString(t turtle.Term) bool {
	return t.Kind == turtle.Literal && t.Datatype == xsdString && t.Lang == ""
}

// describe renders a term for a record. Blank-node labels are never shown.
func describe(t turtle.Term) string {
	switch t.Kind {
	case turtle.IRI:
		return "<" + t.Value + ">"
	case turtle.Blank:
		return "a blank node"
	case turtle.Literal:
		if t.Lang != "" {
			return strconv.Quote(t.Value) + "@" + t.Lang
		}
		return strconv.Quote(t.Value) + "^^<" + t.Datatype + ">"
	}
	return "nothing"
}

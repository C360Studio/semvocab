package shacl

// Task 7.1: the Rapid properties of design D16. Each property states its
// oracle: a generated case is compiled from one form and compared with the
// same case compiled from another order or another form, so the comparison
// never takes the code under test's own word for what the answer is. Run
// with -rapid.seed and -rapid.checks to replay; a failure writes its case
// under testdata/rapid/ (notes/7-generated.md).

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/c360studio/semengine/message"
	"github.com/c360studio/semengine/pkg/projection/contract"
	"github.com/c360studio/semengine/vocabulary"
	"github.com/c360studio/semvocab/internal/oracle"
	"github.com/c360studio/semvocab/internal/turtle"
	"github.com/c360studio/semvocab/packs/diekm"
	"gopkg.in/yaml.v3"
	"pgregory.net/rapid"
)

const genNS = "http://example.org/gen/"

// genDatatypes are the D8 table's datatypes, in a fixed order so draws
// replay from a seed.
var genDatatypes = []string{xsdString, xsdInt, xsdDouble, xsdBool, xsdDateTime, rdfJSON}

// genStrings holds string values that exercise Turtle and YAML escaping.
var genStrings = []string{"a", "b", "", "z z", `q"uote`, `back\slash`, "line\nbreak", "tab\there", "é", "日本", "1", "true", "null"}

// genPatterns are patterns inside the D10 subset.
var genPatterns = []string{`^[a-z]+$`, `a.c`, `\p{Lu}`, `(ab|cd){2,3}`, `x*?`, `[^0-9]`, `\.`, `^$`}

// genCase is a generated in-profile compile input. The generator knows which
// predicate each path names and the data type its shapes imply, which the
// properties use as their oracle.
type genCase struct {
	pack    vocabulary.PackSpec
	shapes  ShapeSet
	mapping Mapping
	implied map[string]string // predicate → pack data_type the shapes imply
	types   []string          // pack domain types
}

func genInput(t *rapid.T) genCase {
	gc := genCase{pack: vocabulary.PackSpec{Name: "gen"}, implied: map[string]string{}}
	nPreds := rapid.IntRange(1, 5).Draw(t, "predicates")
	dts := make([]string, nPreds)
	for i := range nPreds {
		dts[i] = rapid.SampledFrom(genDatatypes).Draw(t, fmt.Sprintf("p%d.datatype", i))
		p := vocabulary.PredicateSpec{Name: fmt.Sprintf("gen.prop.p%d", i), IRI: fmt.Sprintf("%sp%d", genNS, i)}
		if rapid.Bool().Draw(t, fmt.Sprintf("p%d.declared", i)) {
			p.DataType = datatypeTable[dts[i]]
		}
		if rapid.Bool().Draw(t, fmt.Sprintf("p%d.projected", i)) {
			p.Projection = &vocabulary.ProjectionSpec{Steps: genSteps(t, i)}
		}
		gc.pack.Predicates = append(gc.pack.Predicates, p)
	}

	nTypes := rapid.IntRange(1, 3).Draw(t, "types")
	var classes, topTypes []string
	for j := range nTypes {
		et := vocabulary.EntityTypeSpec{DomainType: fmt.Sprintf("gen.t%d", j), ClassIRI: fmt.Sprintf("%sC%d", genNS, j)}
		if rapid.Bool().Draw(t, fmt.Sprintf("t%d.top", j)) {
			et.Superclasses = []string{genNS + "Top"}
			topTypes = append(topTypes, et.DomainType)
		}
		gc.pack.EntityTypes = append(gc.pack.EntityTypes, et)
		gc.types = append(gc.types, et.DomainType)
		classes = append(classes, et.ClassIRI)
		gc.mapping.EntityTypes = append(gc.mapping.EntityTypes,
			EntityTypeBinding{ClassIRI: et.ClassIRI, DomainTypes: []string{et.DomainType}})
	}
	classTypes := map[string][]string{}
	for j, c := range classes {
		classTypes[c] = []string{gc.types[j]}
	}
	if len(topTypes) > 0 {
		classes = append(classes, genNS+"Top")
		classTypes[genNS+"Top"] = topTypes
		gc.mapping.EntityTypes = append(gc.mapping.EntityTypes,
			EntityTypeBinding{ClassIRI: genNS + "Top", DomainTypes: topTypes})
	}
	for i := range nPreds {
		gc.mapping.Predicates = append(gc.mapping.Predicates,
			PredicateBinding{PathIRI: gc.pack.Predicates[i].IRI, Name: gc.pack.Predicates[i].Name})
	}

	constrained := map[string]bool{} // domain type → some shape constrains a predicate on it
	nShapes := rapid.IntRange(1, 3).Draw(t, "shapes")
	for k := range nShapes {
		ns := NodeShape{ID: fmt.Sprintf("%sS%d", genNS, k)}
		ns.TargetClasses = rapid.SliceOfNDistinct(rapid.SampledFrom(classes), 1, 2, rapid.ID[string]).
			Draw(t, fmt.Sprintf("S%d.targets", k))
		preds := rapid.SliceOfNDistinct(rapid.IntRange(0, nPreds-1), 0, nPreds, rapid.ID[int]).
			Draw(t, fmt.Sprintf("S%d.properties", k))
		for _, i := range preds {
			ps := genProperty(t, k, i, gc.pack.Predicates[i].IRI, dts[i])
			if ps.Datatype != "" {
				gc.implied[gc.pack.Predicates[i].Name] = datatypeTable[ps.Datatype]
			}
			ns.Properties = append(ns.Properties, ps)
		}
		if len(preds) > 0 {
			for _, c := range ns.TargetClasses {
				for _, dt := range classTypes[c] {
					constrained[dt] = true
				}
			}
		}
		if rapid.Bool().Draw(t, fmt.Sprintf("S%d.closed", k)) {
			ns.Closed = true
			ns.IgnoredProperties = append([]string{rdfType},
				rapid.SliceOfDistinct(rapid.SampledFrom([]string{genNS + "p0", genNS + "p1", genNS + "unmapped"}), rapid.ID[string]).
					Draw(t, fmt.Sprintf("S%d.ignored", k))...)
		}
		ns.Severity = rapid.SampledFrom([]string{"", shNS + "Violation", shNS + "Warning", shNS + "Info"}).
			Draw(t, fmt.Sprintf("S%d.severity", k))
		ns.Messages = rapid.SliceOfN(rapid.SampledFrom(genStrings), 0, 2).Draw(t, fmt.Sprintf("S%d.messages", k))
		gc.shapes.NodeShapes = append(gc.shapes.NodeShapes, ns)
	}

	// Contracts: every predicate goes to birth or one of two groups, so every
	// constrained predicate is written (ruling Q19). A type nothing
	// constrains may have no binding, and is then unwritable (Q18).
	for _, dt := range gc.types {
		if !constrained[dt] && rapid.Bool().Draw(t, dt+".unbound") {
			continue
		}
		b := ContractBinding{DomainType: dt}
		if rapid.Bool().Draw(t, dt+".named") {
			b.Name = "contract-" + dt
		}
		groups := make([]GroupBinding, 2)
		for g := range groups {
			groups[g] = GroupBinding{Name: fmt.Sprintf("g%d", g),
				Mode: rapid.SampledFrom([]contract.WriteMode{contract.ModeReconcile, contract.ModeAppend}).
					Draw(t, fmt.Sprintf("%s.g%d.mode", dt, g))}
		}
		for _, p := range gc.pack.Predicates {
			switch slot := rapid.IntRange(0, 2).Draw(t, dt+"."+p.Name+".slot"); slot {
			case 0:
				b.Birth = append(b.Birth, p.Name)
			default:
				groups[slot-1].Predicates = append(groups[slot-1].Predicates, p.Name)
			}
		}
		for _, g := range groups {
			if len(g.Predicates) > 0 {
				b.Groups = append(b.Groups, g)
			}
		}
		gc.mapping.Contracts = append(gc.mapping.Contracts, b)
	}
	gc.mapping.ClosedIgnoredPredicates = rapid.SliceOfDistinct(
		rapid.SampledFrom([]string{"other.ns.a", "other.ns.b"}), rapid.ID[string]).Draw(t, "closed_ignored")
	return gc
}

// genSteps draws a projection chain whose step IRIs are not in sorted
// order, so a compiler that reordered steps would be seen.
func genSteps(t *rapid.T, i int) []vocabulary.ProjectionStep {
	n := rapid.IntRange(1, 5).Draw(t, fmt.Sprintf("p%d.steps", i))
	// Rapid draws small values first, so distinct draws tend to come out
	// ascending; the permutation keeps sorted chains from dominating.
	ids := rapid.SliceOfNDistinct(rapid.IntRange(0, 999), n, n, rapid.ID[int]).Draw(t, fmt.Sprintf("p%d.stepIRIs", i))
	ids = rapid.Permutation(ids).Draw(t, fmt.Sprintf("p%d.stepOrder", i))
	steps := make([]vocabulary.ProjectionStep, n)
	for j := range steps {
		steps[j].PredicateIRI = fmt.Sprintf("%sstep/%d", genNS, ids[j])
		if j < n-1 || rapid.Bool().Draw(t, fmt.Sprintf("p%d.terminalClass", i)) {
			steps[j].NodeClassIRI = fmt.Sprintf("%snode/%d", genNS, ids[j])
		}
	}
	steps[n-1].ValuePredicateIRI = fmt.Sprintf("%svalue/%d", genNS, i)
	return steps
}

func genProperty(t *rapid.T, k, i int, path, dt string) PropertyShape {
	label := fmt.Sprintf("S%d.p%d.", k, i)
	ps := PropertyShape{Path: path}
	if rapid.Bool().Draw(t, label+"iri") {
		ps.ID = fmt.Sprintf("%sS%d/p%d", genNS, k, i)
	}
	ps.MinCount = rapid.IntRange(0, 3).Draw(t, label+"min")
	if rapid.Bool().Draw(t, label+"bounded") {
		m := ps.MinCount + rapid.IntRange(0, 3).Draw(t, label+"max")
		ps.MaxCount = &m
	}
	if rapid.Bool().Draw(t, label+"typed") {
		ps.Datatype = dt
	}
	// Every lower bound drawn is at or below every upper bound drawn.
	var lows, highs []string
	switch ps.Datatype {
	case xsdInt:
		lows = []string{"-50", "-7", "0", "+0", "-007"}
		highs = []string{"0", "3", "+12", "0050", "123456789012345678901234567890"}
	case xsdDouble:
		lows = []string{"-INF", "-1.5E2", "-0", "0.0", "-.5"}
		highs = []string{"0", "2.5", "1E3", "INF", "+7."}
	}
	if lows != nil {
		for _, b := range []struct {
			name string
			set  []string
			dst  **Literal
		}{
			{"minInclusive", lows, &ps.MinInclusive}, {"minExclusive", lows, &ps.MinExclusive},
			{"maxInclusive", highs, &ps.MaxInclusive}, {"maxExclusive", highs, &ps.MaxExclusive},
		} {
			if rapid.Bool().Draw(t, label+b.name) {
				*b.dst = &Literal{Lexical: rapid.SampledFrom(b.set).Draw(t, label+b.name+".value"), Datatype: ps.Datatype}
			}
		}
	}
	if dt == xsdString && rapid.Bool().Draw(t, label+"patterned") {
		p := rapid.SampledFrom(genPatterns).Draw(t, label+"pattern")
		ps.Pattern = &p
	}
	if ps.Datatype == xsdString && rapid.Bool().Draw(t, label+"enumerated") {
		in := rapid.SliceOfN(rapid.SampledFrom(genStrings), 0, 4).Draw(t, label+"in")
		ps.In = &in
	}
	ps.Severity = rapid.SampledFrom([]string{"", shNS + "Violation", shNS + "Warning", shNS + "Info"}).Draw(t, label+"severity")
	ps.Messages = rapid.SliceOfN(rapid.SampledFrom(genStrings), 0, 2).Draw(t, label+"messages")
	return ps
}

// turtleString writes s as a Turtle string literal, escaping what the
// grammar requires (Turtle §6.4 ECHAR).
func turtleString(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\r", `\r`, "\t", `\t`)
	return `"` + r.Replace(s) + `"`
}

// writeTurtle renders a shape set as a Turtle shapes document. order
// rearranges each sh:in list before it is written.
func writeTurtle(set ShapeSet, order func([]string) []string) string {
	var b strings.Builder
	b.WriteString("@prefix sh: <http://www.w3.org/ns/shacl#> .\n")
	var props []string
	for _, ns := range set.NodeShapes {
		fmt.Fprintf(&b, "<%s> a sh:NodeShape", ns.ID)
		for _, c := range ns.TargetClasses {
			fmt.Fprintf(&b, " ;\n  sh:targetClass <%s>", c)
		}
		if ns.Closed {
			b.WriteString(" ;\n  sh:closed true ;\n  sh:ignoredProperties (")
			for _, p := range ns.IgnoredProperties {
				fmt.Fprintf(&b, " <%s>", p)
			}
			b.WriteString(" )")
		}
		writeCommon(&b, ns.Severity, ns.Messages)
		for _, p := range ns.Properties {
			if p.ID != "" {
				fmt.Fprintf(&b, " ;\n  sh:property <%s>", p.ID)
				props = append(props, fmt.Sprintf("<%s>%s .\n", p.ID, propertyBody(p, order)))
				continue
			}
			fmt.Fprintf(&b, " ;\n  sh:property [%s ]", propertyBody(p, order))
		}
		b.WriteString(" .\n")
	}
	for _, p := range props {
		b.WriteString(p)
	}
	return b.String()
}

func writeCommon(b *strings.Builder, severity string, messages []string) {
	if severity != "" {
		fmt.Fprintf(b, " ;\n  sh:severity <%s>", severity)
	}
	for _, m := range messages {
		fmt.Fprintf(b, " ;\n  sh:message %s", turtleString(m))
	}
}

func propertyBody(p PropertyShape, order func([]string) []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, " sh:path <%s>", p.Path)
	if p.MinCount != 0 {
		fmt.Fprintf(&b, " ; sh:minCount %d", p.MinCount)
	}
	if p.MaxCount != nil {
		fmt.Fprintf(&b, ` ; sh:maxCount "%d"^^<%s>`, *p.MaxCount, xsdInt)
	}
	if p.Datatype != "" {
		fmt.Fprintf(&b, " ; sh:datatype <%s>", p.Datatype)
	}
	for _, bd := range []struct {
		name string
		lit  *Literal
	}{{"minInclusive", p.MinInclusive}, {"maxInclusive", p.MaxInclusive}, {"minExclusive", p.MinExclusive}, {"maxExclusive", p.MaxExclusive}} {
		if bd.lit != nil {
			fmt.Fprintf(&b, " ; sh:%s %s^^<%s>", bd.name, turtleString(bd.lit.Lexical), bd.lit.Datatype)
		}
	}
	if p.Pattern != nil {
		fmt.Fprintf(&b, " ; sh:pattern %s", turtleString(*p.Pattern))
	}
	if p.In != nil {
		b.WriteString(" ; sh:in (")
		for _, m := range order(*p.In) {
			b.WriteString(" " + turtleString(m))
		}
		b.WriteString(" )")
	}
	var c strings.Builder
	writeCommon(&c, p.Severity, p.Messages)
	b.WriteString(strings.ReplaceAll(c.String(), "\n  ", " "))
	return b.String()
}

// wireForm renders a shape set in SemVocab's JSON/YAML form with the field
// names of design D3, written out here rather than taken from the decoder's
// struct tags.
func wireForm(set ShapeSet, order func([]string) []string) map[string]any {
	var nodes []any
	for _, ns := range set.NodeShapes {
		n := map[string]any{"id": ns.ID, "target_classes": ns.TargetClasses}
		if ns.Closed {
			n["closed"] = true
			n["ignored_properties"] = ns.IgnoredProperties
		}
		if ns.Severity != "" {
			n["severity"] = ns.Severity
		}
		if len(ns.Messages) > 0 {
			n["messages"] = ns.Messages
		}
		var props []any
		for _, p := range ns.Properties {
			m := map[string]any{"path": p.Path}
			if p.ID != "" {
				m["id"] = p.ID
			}
			if p.MinCount != 0 {
				m["min_count"] = p.MinCount
			}
			if p.MaxCount != nil {
				m["max_count"] = *p.MaxCount
			}
			if p.Datatype != "" {
				m["datatype"] = p.Datatype
			}
			for k, lit := range map[string]*Literal{"min_inclusive": p.MinInclusive, "max_inclusive": p.MaxInclusive,
				"min_exclusive": p.MinExclusive, "max_exclusive": p.MaxExclusive} {
				if lit != nil {
					m[k] = lit.Lexical
				}
			}
			if p.Pattern != nil {
				m["pattern"] = *p.Pattern
			}
			if p.In != nil {
				m["in"] = append([]string{}, order(*p.In)...)
			}
			if p.Severity != "" {
				m["severity"] = p.Severity
			}
			if len(p.Messages) > 0 {
				m["messages"] = p.Messages
			}
			props = append(props, m)
		}
		if props != nil {
			n["properties"] = props
		}
		nodes = append(nodes, n)
	}
	return map[string]any{"node_shapes": nodes}
}

func keepOrder(s []string) []string { return s }

// outcome is a compile result in comparable form: the bundle, or the issue
// list. Two outcomes are equal when both bundles are reflect.DeepEqual or
// both issue lists are.
type outcome struct {
	bundle *Bundle
	issues []Issue
	err    string
}

func compileOutcome(pack vocabulary.PackSpec, set ShapeSet, m Mapping) outcome {
	b, err := Compile(pack, set, m)
	if err == nil {
		return outcome{bundle: b}
	}
	var ce *CompileError
	if errors.As(err, &ce) {
		return outcome{issues: ce.Issues, err: err.Error()}
	}
	return outcome{err: err.Error()}
}

func (o outcome) equal(p outcome) bool {
	return reflect.DeepEqual(o.bundle, p.bundle) && reflect.DeepEqual(o.issues, p.issues) && o.err == p.err
}

func (o outcome) String() string {
	if o.bundle != nil {
		return fmt.Sprintf("bundle %+v", *o.bundle)
	}
	return o.err
}

func mustParseTurtle(t *rapid.T, src string) ShapeSet {
	set, err := ParseTurtle(strings.NewReader(src))
	if err != nil {
		t.Fatalf("ParseTurtle: %v\n%s", err, src)
	}
	return set
}

// TestRapidRoundTripFormats: a generated in-profile shape set written as
// Turtle, JSON, and YAML decodes to one shape set and compiles to one bundle
// (design D16). The generator builds only in-profile cases, so the compile
// must also succeed.
func TestRapidRoundTripFormats(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		gc := genInput(t)
		ttl := writeTurtle(gc.shapes, keepOrder)
		fromTTL := mustParseTurtle(t, ttl)

		wire := wireForm(gc.shapes, keepOrder)
		js, err := json.Marshal(wire)
		if err != nil {
			t.Fatal(err)
		}
		fromJSON, err := DecodeShapesJSON(bytes.NewReader(js))
		if err != nil {
			t.Fatalf("DecodeShapesJSON: %v\n%s", err, js)
		}
		ym, err := yaml.Marshal(wire)
		if err != nil {
			t.Fatal(err)
		}
		fromYAML, err := DecodeShapesYAML(bytes.NewReader(ym))
		if err != nil {
			t.Fatalf("DecodeShapesYAML: %v\n%s", err, ym)
		}
		if !reflect.DeepEqual(fromTTL, fromJSON) || !reflect.DeepEqual(fromTTL, fromYAML) {
			t.Fatalf("shape sets differ:\nturtle %+v\njson   %+v\nyaml   %+v", fromTTL, fromJSON, fromYAML)
		}
		if got, want := countProperties(fromTTL), countProperties(gc.shapes); got != want {
			t.Fatalf("decoded %d property shapes, generated %d", got, want)
		}

		ot := compileOutcome(gc.pack, fromTTL, gc.mapping)
		if ot.bundle == nil {
			t.Fatalf("in-profile case failed to compile: %s\n%s", ot, ttl)
		}
		for name, o := range map[string]outcome{
			"json": compileOutcome(gc.pack, fromJSON, gc.mapping),
			"yaml": compileOutcome(gc.pack, fromYAML, gc.mapping),
		} {
			if !o.equal(ot) {
				t.Fatalf("%s bundle differs from turtle:\n%s\n%s", name, o, ot)
			}
		}
	})
}

func countProperties(set ShapeSet) int {
	n := 0
	for _, ns := range set.NodeShapes {
		n += len(ns.Properties)
	}
	return n
}

// writeNTriples writes triples, one per line, with blank nodes relabelled
// by labels.
func writeNTriples(triples []turtle.Triple, labels map[string]string) string {
	term := func(x turtle.Term) string {
		switch x.Kind {
		case turtle.IRI:
			return "<" + x.Value + ">"
		case turtle.Blank:
			return "_:" + labels[x.Value]
		}
		if x.Lang != "" {
			return turtleString(x.Value) + "@" + x.Lang
		}
		return turtleString(x.Value) + "^^<" + x.Datatype + ">"
	}
	var b strings.Builder
	for _, tr := range triples {
		b.WriteString(term(tr.S) + " " + term(tr.P) + " " + term(tr.O) + " .\n")
	}
	return b.String()
}

// permuteStatements draws a permutation of triples, optionally repeats some
// (an RDF graph is a set), and draws fresh blank-node labels.
func permuteStatements(t *rapid.T, triples []turtle.Triple) string {
	perm := rapid.Permutation(triples).Draw(t, "statements")
	repeats := rapid.SliceOfN(rapid.IntRange(0, len(perm)-1), 0, 3).Draw(t, "repeats")
	for _, i := range repeats {
		perm = append(perm, perm[i])
	}
	var blanks []string
	seen := map[string]bool{}
	for _, tr := range triples {
		for _, x := range []turtle.Term{tr.S, tr.O} {
			if x.Kind == turtle.Blank && !seen[x.Value] {
				seen[x.Value] = true
				blanks = append(blanks, x.Value)
			}
		}
	}
	fresh := make([]string, len(blanks))
	for i := range fresh {
		fresh[i] = "g" + strconv.Itoa(i)
	}
	fresh = rapid.Permutation(fresh).Draw(t, "labels")
	labels := map[string]string{}
	for i, l := range blanks {
		labels[l] = fresh[i]
	}
	return writeNTriples(perm, labels)
}

// outOfProfile are Turtle statements with constructs the profile rejects,
// several with blank-node structure, for the permutation property's
// compile-issue lists. The blank cycle is asymmetric, so which of its nodes
// is reported first could depend on order.
var outOfProfile = []string{
	"<" + genNS + "S0> <" + shNS + "and> ( [ <" + shNS + "path> <" + genNS + "p0> ; <" + shNS + "minCount> 1 ] ) .",
	"<" + genNS + "S0> <" + shNS + "rule> [ a <" + shNS + "TripleRule> ; <" + shNS + "subject> <" + shNS + "this> ] .",
	"<" + genNS + "S0> <" + shNS + "bogus> \"x\" .",
	"_:stray <" + genNS + "r> \"v\" .",
	"_:c1 <" + genNS + "r> _:c2 . _:c2 <" + genNS + "q> \"w\" . _:c2 <" + genNS + "r> _:c1 .",
	"<" + genNS + "data> <" + genNS + "r> [ <" + genNS + "q> ( 1 2 ) ] .",
}

// TestRapidShapesStatementPermutation: permuting a shapes graph's
// statements, repeating some, and relabelling its blank nodes yields an
// equal bundle or an equal issue list (design D16). The graph is a generated
// in-profile case, sometimes with out-of-profile statements added.
func TestRapidShapesStatementPermutation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		gc := genInput(t)
		src := writeTurtle(gc.shapes, keepOrder)
		for _, extra := range rapid.SliceOfDistinct(rapid.SampledFrom(outOfProfile), rapid.ID[string]).Draw(t, "extra") {
			src += extra + "\n"
		}
		requirePermutationInvariant(t, src, gc.pack, gc.mapping)
	})
}

// TestRapidDIEKMStatementPermutation is the spec scenario "Permuted shapes":
// the DIEKM shapes compiled from a random permutation of their statements
// equal the bundle compiled from the file.
func TestRapidDIEKMStatementPermutation(t *testing.T) {
	spec, err := diekm.Spec()
	if err != nil {
		t.Fatal(err)
	}
	m, err := DecodeMappingYAML(bytes.NewReader(diekm.Mapping()))
	if err != nil {
		t.Fatal(err)
	}
	rapid.Check(t, func(t *rapid.T) {
		requirePermutationInvariant(t, string(diekm.Shapes()), spec, m)
	})
}

// TestRapidW3CStatementPermutation permutes the vendored W3C cases (read
// with their base, so every IRI is absolute) and compares compile issue
// lists. They carry manifests, data graphs, and rejected constructs over
// blank nodes, none of which the profile accepts.
func TestRapidW3CStatementPermutation(t *testing.T) {
	var rels []string
	for rel := range w3cRejectedFamilies {
		rels = append(rels, rel)
	}
	rels = append(rels, "property/minCount-001.ttl", "property/in-001.ttl", "node/closed-001.ttl",
		"property/pattern-001.ttl")
	slices.Sort(rels)
	sources := map[string]string{}
	for _, rel := range rels {
		data, err := oracle.ReadFixture(oracle.W3CDataShapes, "data-shapes-test-suite/tests/core/"+rel)
		if err != nil {
			t.Fatal(err)
		}
		triples, err := turtle.Parse(bytes.NewReader(data), turtle.WithBase("http://semvocab.test/core/"+rel))
		if err != nil {
			t.Fatal(err)
		}
		sources[rel] = writeNTriples(triples, identityLabels(triples))
	}
	rapid.Check(t, func(t *rapid.T) {
		rel := rapid.SampledFrom(rels).Draw(t, "case")
		requirePermutationInvariant(t, sources[rel], vocabulary.PackSpec{Name: "w3c"}, Mapping{})
	})
}

func identityLabels(triples []turtle.Triple) map[string]string {
	labels := map[string]string{}
	for _, tr := range triples {
		for _, x := range []turtle.Term{tr.S, tr.O} {
			if x.Kind == turtle.Blank {
				labels[x.Value] = "n" + strings.TrimLeft(x.Value, "_:")
			}
		}
	}
	return labels
}

func requirePermutationInvariant(t *rapid.T, src string, pack vocabulary.PackSpec, m Mapping) {
	want := compileOutcome(pack, mustParseTurtle(t, src), m)
	triples, err := turtle.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	permuted := permuteStatements(t, triples)
	got := compileOutcome(pack, mustParseTurtle(t, permuted), m)
	if !got.equal(want) {
		t.Fatalf("permuted statements compile differently:\n%s\nwant\n%s\npermuted document:\n%s", got, want, permuted)
	}
}

// TestRapidInMemberOrder: reordering and repeating sh:in members, in each
// of the three forms, yields an equal bundle (design D11, D16).
func TestRapidInMemberOrder(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		gc := genInput(t)
		want := compileOutcome(gc.pack, mustParseTurtle(t, writeTurtle(gc.shapes, keepOrder)), gc.mapping)
		n := 0
		shuffle := func(in []string) []string {
			n++
			out := rapid.Permutation(in).Draw(t, fmt.Sprintf("in%d", n))
			for _, i := range rapid.SliceOfN(rapid.IntRange(0, max(len(in)-1, 0)), 0, min(len(in), 2)).Draw(t, fmt.Sprintf("in%d.repeat", n)) {
				out = append(out, in[i])
			}
			return out
		}
		set := mustParseTurtle(t, writeTurtle(gc.shapes, shuffle))
		if got := compileOutcome(gc.pack, set, gc.mapping); !got.equal(want) {
			t.Fatalf("turtle with reordered sh:in compiles differently:\n%s\nwant\n%s", got, want)
		}
		js, err := json.Marshal(wireForm(gc.shapes, shuffle))
		if err != nil {
			t.Fatal(err)
		}
		set, err = DecodeShapesJSON(bytes.NewReader(js))
		if err != nil {
			t.Fatal(err)
		}
		if got := compileOutcome(gc.pack, set, gc.mapping); !got.equal(want) {
			t.Fatalf("json with reordered in compiles differently:\n%s\nwant\n%s", got, want)
		}
		ym, err := yaml.Marshal(wireForm(gc.shapes, shuffle))
		if err != nil {
			t.Fatal(err)
		}
		set, err = DecodeShapesYAML(bytes.NewReader(ym))
		if err != nil {
			t.Fatal(err)
		}
		if got := compileOutcome(gc.pack, set, gc.mapping); !got.equal(want) {
			t.Fatalf("yaml with reordered in compiles differently:\n%s\nwant\n%s", got, want)
		}
	})
}

// mappingFaults are mapping changes that make compilation fail, so the
// mapping-order property also compares issue lists. Several name two rows,
// where the first row seen could otherwise decide the message.
var mappingFaults = []func(m *Mapping){
	func(m *Mapping) {
		m.Predicates = slices.DeleteFunc(m.Predicates, func(p PredicateBinding) bool { return p.Name == "gen.prop.p0" })
	},
	func(m *Mapping) {
		m.Predicates = append(m.Predicates, PredicateBinding{PathIRI: genNS + "p0", Name: "gen.prop.zz"})
	},
	func(m *Mapping) {
		m.EntityTypes = append(m.EntityTypes, EntityTypeBinding{ClassIRI: genNS + "C0", DomainTypes: []string{"gen.t9"}})
	},
	func(m *Mapping) { m.EntityTypes[0].DomainTypes = append(m.EntityTypes[0].DomainTypes, "gen.t9") },
	func(m *Mapping) {
		m.Contracts = append(m.Contracts, ContractBinding{DomainType: "gen.t0",
			Groups: []GroupBinding{{Name: "late", Mode: contract.ModeAppend, Predicates: []string{"gen.prop.p0"}}}})
	},
	func(m *Mapping) {
		m.Contracts = append(m.Contracts, ContractBinding{DomainType: "gen.t9", Birth: []string{"gen.prop.p0"}})
	},
	func(m *Mapping) { m.ClosedIgnoredPredicates = append(m.ClosedIgnoredPredicates, "bad", "also.bad") },
	func(m *Mapping) {
		if len(m.Contracts) > 0 {
			m.Contracts[0].Groups = append(m.Contracts[0].Groups,
				GroupBinding{Name: "g0", Mode: "append", Predicates: []string{"gen.prop.p0"}},
				GroupBinding{Name: "g0", Mode: "reconcile", Predicates: []string{"gen.prop.p0"}})
		}
	},
}

func cloneMapping(m Mapping) Mapping {
	out := Mapping{
		EntityTypes:             slices.Clone(m.EntityTypes),
		Predicates:              slices.Clone(m.Predicates),
		Contracts:               slices.Clone(m.Contracts),
		ClosedIgnoredPredicates: slices.Clone(m.ClosedIgnoredPredicates),
	}
	for i := range out.EntityTypes {
		out.EntityTypes[i].DomainTypes = slices.Clone(m.EntityTypes[i].DomainTypes)
	}
	for i := range out.Contracts {
		out.Contracts[i].Birth = slices.Clone(m.Contracts[i].Birth)
		out.Contracts[i].Groups = slices.Clone(m.Contracts[i].Groups)
		for j := range out.Contracts[i].Groups {
			out.Contracts[i].Groups[j].Predicates = slices.Clone(m.Contracts[i].Groups[j].Predicates)
		}
	}
	return out
}

// shuffleMapping draws a reordering of every row list in m.
func shuffleMapping(t *rapid.T, m Mapping) Mapping {
	out := cloneMapping(m)
	out.EntityTypes = rapid.Permutation(out.EntityTypes).Draw(t, "entity_types")
	for i := range out.EntityTypes {
		out.EntityTypes[i].DomainTypes = rapid.Permutation(out.EntityTypes[i].DomainTypes).Draw(t, fmt.Sprintf("entity_types%d", i))
	}
	out.Predicates = rapid.Permutation(out.Predicates).Draw(t, "predicates")
	out.Contracts = rapid.Permutation(out.Contracts).Draw(t, "contracts")
	for i := range out.Contracts {
		c := &out.Contracts[i]
		c.Birth = rapid.Permutation(c.Birth).Draw(t, fmt.Sprintf("contracts%d.birth", i))
		c.Groups = rapid.Permutation(c.Groups).Draw(t, fmt.Sprintf("contracts%d.groups", i))
		for j := range c.Groups {
			c.Groups[j].Predicates = rapid.Permutation(c.Groups[j].Predicates).Draw(t, fmt.Sprintf("contracts%d.groups%d", i, j))
		}
	}
	out.ClosedIgnoredPredicates = rapid.Permutation(out.ClosedIgnoredPredicates).Draw(t, "closed_ignored")
	return out
}

// TestRapidMappingRowOrder: reordering every row list of the mapping yields
// an equal bundle or an equal issue list (design D16), with and without
// mapping faults.
func TestRapidMappingRowOrder(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		gc := genInput(t)
		m := cloneMapping(gc.mapping)
		for _, i := range rapid.SliceOfNDistinct(rapid.IntRange(0, len(mappingFaults)-1), 0, 3, rapid.ID[int]).Draw(t, "faults") {
			mappingFaults[i](&m)
		}
		set := mustParseTurtle(t, writeTurtle(gc.shapes, keepOrder))
		want := compileOutcome(gc.pack, set, m)
		got := compileOutcome(gc.pack, set, shuffleMapping(t, m))
		if !got.equal(want) {
			t.Fatalf("reordered mapping compiles differently:\n%s\nwant\n%s", got, want)
		}
	})
}

// genValue draws a candidate value: mostly ones the predicate's data type
// carries, sometimes another type's, rarely an input error (D9).
func genValue(t *rapid.T, label, dataType string) any {
	pools := map[string][]any{
		vocabulary.DataTypeString:   {"a", "b", "acme.edge.sensor.gen.t0.e2", "http://example.org/x", "", "é"},
		vocabulary.DataTypeInt:      {1, int64(2), float64(3), int32(-4), 1.5},
		vocabulary.DataTypeFloat:    {1.5, 2, float32(0.5), math.Copysign(0, -1)},
		vocabulary.DataTypeBool:     {true, false},
		vocabulary.DataTypeDateTime: {time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC), "2024-01-02T03:04:05Z"},
		vocabulary.DataTypeJSON:     {`{"a":1}`, `[1,2]`, "not json"},
	}
	switch rapid.IntRange(0, 19).Draw(t, label+".kind") {
	case 0:
		return rapid.SampledFrom([]any{nil, math.NaN(), math.Inf(1), struct{}{}, "\xff"}).Draw(t, label+".invalid")
	case 1, 2:
		return rapid.SampledFrom([]any{"a", 7, 2.5, true}).Draw(t, label+".other")
	}
	pool, ok := pools[dataType]
	if !ok {
		pool = pools[vocabulary.DataTypeString]
	}
	return rapid.SampledFrom(pool).Draw(t, label+".value")
}

// TestRapidCandidateTriplePermutation: permuting candidate triples, and
// repeating some with another source and timestamp, yields an equal report,
// or an equal input error (design D13, D16).
func TestRapidCandidateTriplePermutation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		gc := genInput(t)
		b, err := Compile(gc.pack, mustParseTurtle(t, writeTurtle(gc.shapes, keepOrder)), gc.mapping)
		if err != nil {
			t.Fatalf("in-profile case failed to compile: %v", err)
		}
		dataTypes := map[string]string{}
		preds := []string{"other.ns.a", "other.ns.c"}
		for _, p := range b.Pack().Predicates {
			dataTypes[p.Name] = p.DataType
			preds = append(preds, p.Name)
		}
		dt := rapid.SampledFrom(append(slices.Clone(gc.types), "gen.none")).Draw(t, "type")
		entity := "acme.edge.sensor." + dt + ".e1"
		n := rapid.IntRange(0, 8).Draw(t, "triples")
		triples := make([]message.Triple, n)
		for i := range triples {
			p := rapid.SampledFrom(preds).Draw(t, fmt.Sprintf("t%d.predicate", i))
			triples[i] = message.Triple{Subject: entity, Predicate: p, Object: genValue(t, fmt.Sprintf("t%d", i), dataTypes[p])}
			if rapid.IntRange(0, 29).Draw(t, fmt.Sprintf("t%d.foreign", i)) == 0 {
				triples[i].Subject = "acme.edge.sensor.gen.t0.other"
			}
		}
		want, wantErr := b.ValidateCandidateState(entity, triples)

		perm := rapid.Permutation(triples).Draw(t, "permutation")
		for j, i := range rapid.SliceOfN(rapid.IntRange(0, max(n-1, 0)), 0, min(n, 3)).Draw(t, "repeats") {
			dup := triples[i]
			dup.Source = fmt.Sprintf("repeat-%d", j)
			dup.Timestamp = time.Date(2025, 1, 1, 0, 0, j, 0, time.UTC)
			at := rapid.IntRange(0, len(perm)).Draw(t, fmt.Sprintf("repeat%d.at", j))
			perm = slices.Insert(perm, at, dup)
		}
		got, gotErr := b.ValidateCandidateState(entity, perm)
		if fmt.Sprint(gotErr) != fmt.Sprint(wantErr) || !reflect.DeepEqual(got, want) {
			t.Fatalf("permuted candidate validates differently:\n%+v %v\nwant\n%+v %v", got, gotErr, want, wantErr)
		}
	})
}

// copyPack deep-copies a pack for the oracle. It is written here, not taken
// from the package's clonePack, so a defect there cannot hide in both sides.
func copyPack(p vocabulary.PackSpec) vocabulary.PackSpec {
	data, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	var out vocabulary.PackSpec
	if err := json.Unmarshal(data, &out); err != nil {
		panic(err)
	}
	return out
}

// packOrderMismatch compares a compiled pack with the base pack the
// generator built, filled with the data types the generator knows its
// shapes imply (design D5). It returns "" when they agree, and otherwise
// names the first difference: entry order and projection step order
// included.
func packOrderMismatch(base vocabulary.PackSpec, implied map[string]string, got vocabulary.PackSpec) string {
	want := copyPack(base)
	for i := range want.Predicates {
		if want.Predicates[i].DataType == "" {
			want.Predicates[i].DataType = implied[want.Predicates[i].Name]
		}
	}
	if len(got.Predicates) != len(want.Predicates) {
		return fmt.Sprintf("%d predicates, want %d", len(got.Predicates), len(want.Predicates))
	}
	for i := range want.Predicates {
		g, w := got.Predicates[i], want.Predicates[i]
		if !reflect.DeepEqual(g.Projection, w.Projection) {
			return fmt.Sprintf("predicate %d (%s) has projection %+v, want %+v", i, w.Name, g.Projection, w.Projection)
		}
		if !reflect.DeepEqual(g, w) {
			return fmt.Sprintf("predicate %d is %+v, want %+v", i, g, w)
		}
	}
	if !reflect.DeepEqual(got, want) {
		return fmt.Sprintf("pack %+v, want %+v", got, want)
	}
	return ""
}

// TestRapidProjectionStepsKeepBasePackOrder: the compiled pack is the base
// pack in its entry order, projection steps in their order, with only empty
// data types filled (design D5, D16). The oracle is the generated base pack
// and the generator's own record of the data types its shapes imply; the
// mapping rows are also reordered, which must not move a step.
func TestRapidProjectionStepsKeepBasePackOrder(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		gc := genInput(t)
		b, err := Compile(gc.pack, mustParseTurtle(t, writeTurtle(gc.shapes, keepOrder)), shuffleMapping(t, gc.mapping))
		if err != nil {
			t.Fatalf("in-profile case failed to compile: %v", err)
		}
		if diff := packOrderMismatch(gc.pack, gc.implied, b.Pack()); diff != "" {
			t.Fatal(diff)
		}
	})
}

// TestProjectionStepSwapChangesBundle is the sensitivity pair for the
// property above. Swapping two steps of DIEKM's altitude chain in the base
// pack changes the bundle, and the compiled pack carries the swapped order;
// packOrderMismatch reports a compiled pack whose steps were swapped.
func TestProjectionStepSwapChangesBundle(t *testing.T) {
	spec, err := diekm.Spec()
	if err != nil {
		t.Fatal(err)
	}
	m, err := DecodeMappingYAML(bytes.NewReader(diekm.Mapping()))
	if err != nil {
		t.Fatal(err)
	}
	shapes, err := ParseTurtle(bytes.NewReader(diekm.Shapes()))
	if err != nil {
		t.Fatal(err)
	}
	const altitude = "intel.measurement.altitude-m"
	idx := slices.IndexFunc(spec.Predicates, func(p vocabulary.PredicateSpec) bool { return p.Name == altitude })
	if idx < 0 || len(spec.Predicates[idx].Projection.Steps) < 3 {
		t.Fatalf("premise: DIEKM's %s has a projection of at least three steps", altitude)
	}

	plain, err := Compile(spec, shapes, m)
	if err != nil {
		t.Fatal(err)
	}
	swapped := copyPack(spec)
	steps := swapped.Predicates[idx].Projection.Steps
	steps[0], steps[1] = steps[1], steps[0]
	if err := swapped.Validate(); err != nil {
		t.Fatalf("premise: the swapped pack is still valid: %v", err)
	}
	b, err := Compile(swapped, shapes, m)
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(b, plain) {
		t.Fatal("swapping two projection steps left the bundle unchanged")
	}
	if got := b.Pack().Predicates[idx].Projection.Steps; !reflect.DeepEqual(got, steps) {
		t.Fatalf("compiled steps %+v, want the swapped base order %+v", got, steps)
	}

	implied := map[string]string{} // DIEKM declares every data type
	if diff := packOrderMismatch(spec, implied, plain.Pack()); diff != "" {
		t.Fatalf("premise: the plain bundle keeps DIEKM's order: %s", diff)
	}
	if diff := packOrderMismatch(spec, implied, b.Pack()); diff == "" {
		t.Fatal("packOrderMismatch accepted a compiled pack with two steps swapped")
	}
}

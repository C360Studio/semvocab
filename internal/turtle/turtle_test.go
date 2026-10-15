package turtle

import (
	"reflect"
	"strings"
	"testing"
)

func iri(v string) Term   { return Term{Kind: IRI, Value: v} }
func blank(v string) Term { return Term{Kind: Blank, Value: v} }
func str(v string) Term   { return Term{Kind: Literal, Value: v, Datatype: xsdString} }
func typed(v, dt string) Term {
	return Term{Kind: Literal, Value: v, Datatype: dt}
}

// TestParseW3CExamples parses examples from the W3C Turtle Recommendation
// (https://www.w3.org/TR/turtle/, 25 February 2014) and compares with the
// triples the specification states they denote.
func TestParseW3CExamples(t *testing.T) {
	const ex = "http://example.org/#"
	tests := []struct {
		name string
		src  string
		want []Triple
	}{
		{
			// §2.3 Example 4 (predicate lists) with prefixes per §2.4.
			name: "predicate list",
			src: `@prefix ex: <http://example.org/#> .
<http://example.org/#spiderman> <http://www.perceive.net/schemas/relationship/enemyOf> <http://example.org/#green-goblin> ;
				<http://xmlns.com/foaf/0.1/name> "Spiderman" .`,
			want: []Triple{
				{iri(ex + "spiderman"), iri("http://www.perceive.net/schemas/relationship/enemyOf"), iri(ex + "green-goblin")},
				{iri(ex + "spiderman"), iri("http://xmlns.com/foaf/0.1/name"), str("Spiderman")},
			},
		},
		{
			// §2.3 Example 6 (object lists) and §2.5.2 language tag.
			name: "object list and language tag",
			src:  `<http://example.org/#spiderman> <http://xmlns.com/foaf/0.1/name> "Spiderman", "Человек-паук"@ru .`,
			want: []Triple{
				{iri(ex + "spiderman"), iri("http://xmlns.com/foaf/0.1/name"), str("Spiderman")},
				{iri(ex + "spiderman"), iri("http://xmlns.com/foaf/0.1/name"),
					Term{Kind: Literal, Value: "Человек-паук", Lang: "ru", Datatype: rdfNS + "langString"}},
			},
		},
		{
			// §2.5.1 Example 10 (long string) and §6.4 escapes.
			name: "long string and escapes",
			src: `@prefix show: <http://example.org/vocab/show/> .
show:218 show:localName """That Seventies Show""" .
show:218 show:blurb """This is a multi-line
literal with "quotes".""" .
show:218 show:quoted "say \"hi\"é" .`,
			want: []Triple{
				{iri("http://example.org/vocab/show/218"), iri("http://example.org/vocab/show/localName"), str("That Seventies Show")},
				{iri("http://example.org/vocab/show/218"), iri("http://example.org/vocab/show/blurb"),
					str("This is a multi-line\nliteral with \"quotes\".")},
				{iri("http://example.org/vocab/show/218"), iri("http://example.org/vocab/show/quoted"), str("say \"hi\"é")},
			},
		},
		{
			// §2.5.2 Example 13 (numbers and booleans) and "a" (§2.4).
			name: "numeric, boolean, and a",
			src: `@prefix : <http://example.org/elements> .
<http://en.wikipedia.org/wiki/Helium> :atomicNumber 2 ; :atomicMass 4.002602 ; :specificGravity 1.663E-4 ; :noble true ; a :Element .`,
			want: []Triple{
				{iri("http://en.wikipedia.org/wiki/Helium"), iri("http://example.org/elementsatomicNumber"), typed("2", xsdNS+"integer")},
				{iri("http://en.wikipedia.org/wiki/Helium"), iri("http://example.org/elementsatomicMass"), typed("4.002602", xsdNS+"decimal")},
				{iri("http://en.wikipedia.org/wiki/Helium"), iri("http://example.org/elementsspecificGravity"), typed("1.663E-4", xsdNS+"double")},
				{iri("http://en.wikipedia.org/wiki/Helium"), iri("http://example.org/elementsnoble"), typed("true", xsdNS+"boolean")},
				{iri("http://en.wikipedia.org/wiki/Helium"), iri(rdfType), iri("http://example.org/elementsElement")},
			},
		},
		{
			// §2.6 Example 15 (blank node property list) and datatype IRI.
			name: "blank node property list and datatype",
			src: `@prefix foaf: <http://xmlns.com/foaf/0.1/> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .
[ foaf:name "Alice" ] foaf:knows [ foaf:name "Bob" ; foaf:age "7"^^xsd:integer ] .`,
			want: []Triple{
				{blank("_:b1"), iri("http://xmlns.com/foaf/0.1/name"), str("Alice")},
				{blank("_:b2"), iri("http://xmlns.com/foaf/0.1/name"), str("Bob")},
				{blank("_:b2"), iri("http://xmlns.com/foaf/0.1/age"), typed("7", xsdNS+"integer")},
				{blank("_:b1"), iri("http://xmlns.com/foaf/0.1/knows"), blank("_:b2")},
			},
		},
		{
			// §2.8 Example 21 (collections), with an empty collection.
			name: "collection",
			src: `@prefix : <http://example.org/foo> .
:subject :predicate ( :a :b ) ; :empty () .`,
			want: []Triple{
				{blank("_:b1"), iri(rdfFirst), iri("http://example.org/foob")},
				{blank("_:b1"), iri(rdfRest), iri(rdfNil)},
				{blank("_:b2"), iri(rdfFirst), iri("http://example.org/fooa")},
				{blank("_:b2"), iri(rdfRest), blank("_:b1")},
				{iri("http://example.org/foosubject"), iri("http://example.org/foopredicate"), blank("_:b2")},
				{iri("http://example.org/foosubject"), iri("http://example.org/fooempty"), iri(rdfNil)},
			},
		},
		{
			// N-Triples lines are Turtle (§1); labelled blank nodes and comments.
			name: "n-triples line with blank label and comment",
			src:  "# comment\n_:x <http://ex/p> <http://ex/o> . # trailing\n",
			want: []Triple{{blank("_:Lx"), iri("http://ex/p"), iri("http://ex/o")}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(strings.NewReader(tt.src))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse mismatch\n got: %+v\nwant: %+v", got, tt.want)
			}
		})
	}
}

// TestParseRejectsUnsupported shows the reader fails loudly, naming the
// location, on syntax outside its documented subset.
func TestParseRejectsUnsupported(t *testing.T) {
	tests := []struct {
		name, src, wantErr string
	}{
		{"sparql prefix", "PREFIX ex: <http://ex/>\n", "SPARQL-style directive"},
		{"relative IRI", "<foo> <http://ex/p> <http://ex/o> .", "relative IRI"},
		{"single quotes", "<http://ex/s> <http://ex/p> 'x' .", "single-quoted"},
		{"undeclared prefix", "ex:s <http://ex/p> <http://ex/o> .", "undeclared prefix"},
		{"local escape", "@prefix ex: <http://ex/> .\nex:a\\.b <http://ex/p> <http://ex/o> .", "escape in prefixed name"},
		{"bad escape", `<http://ex/s> <http://ex/p> "\q" .`, `escape \q`},
		{"missing dot", "<http://ex/s> <http://ex/p> <http://ex/o>", "expected '.'"},
		{"unterminated string", `<http://ex/s> <http://ex/p> "abc`, "unterminated string"},
		{"unknown directive", "@keywords a .", "unsupported directive"},
		{"line and column", "<http://ex/s> <http://ex/p> <http://ex/o> .\n<http://ex/s> <http://ex/p> 'x' .", "line 2 col 29"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(tt.src))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Parse error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

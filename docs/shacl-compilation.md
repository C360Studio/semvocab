# SHACL compilation

The `shacl` package turns a SHACL shapes document into three things, ahead of time:

- a SemEngine vocabulary pack (`vocabulary.PackSpec`) that a program registers with SemEngine's `RegisterPack`;
- projection contracts (`contract.Contract`), which say which predicates a writer owns and whether it replaces or
  appends them;
- a native validator, which checks the complete state of one entity against the shapes without an RDF or SHACL engine.

SHACL is the W3C Shapes Constraint Language: a document of shapes, each stating what values an entity's properties
may hold ([SHACL Recommendation](https://www.w3.org/TR/shacl/)). The compiler reads a small, fixed part of it, the
profile, and rejects everything else by name. Nothing here runs in SemEngine's ingest or query paths, and compiling
or validating never reads or changes SemEngine's global registries.

This page describes the code as it is in the tree. It does not cover:

- SHACL rules (`sh:rule`). They are rejected. Translating them is planned work, not present (see
  [Rules are not compiled](#rules-are-not-compiled)).
- Shapes over reified paths (a triple about a triple), exporting shapes to an external validator, and checking this
  validator's results against an external one. None of these is present.

A note on citations. The archived change `openspec/changes/archive/2026-10-17-phase-2-shacl-compilation/` holds the
design (`design.md`, decisions D1–D17) and the probes (`notes/1-probes.md`, probes P1–P5) behind statements about
libraries and standards below. Citations such as "probe P4" mean that note. The current requirements are in
`openspec/specs/shacl-compilation/spec.md`. SemEngine citations are `file:line` in SemEngine `v0.1.0-alpha.2`, the
version `go.mod` pins.

## Quick start

DIEKM is the vocabulary pack in `packs/diekm`. It embeds its pack, its shapes, and a mapping that binds the two.
This program compiles it and validates one entity. Run it from a module that requires `github.com/c360studio/semvocab`
and `github.com/c360studio/semengine`:

```go
package main

import (
    "bytes"
    "fmt"
    "log"
    "time"

    "github.com/c360studio/semengine/message"
    "github.com/c360studio/semvocab/packs/diekm"
    "github.com/c360studio/semvocab/shacl"
)

func main() {
    spec, err := diekm.Spec()
    check(err)
    shapes, err := shacl.ParseTurtle(bytes.NewReader(diekm.Shapes()))
    check(err)
    mapping, err := shacl.DecodeMappingYAML(bytes.NewReader(diekm.Mapping()))
    check(err)

    bundle, err := shacl.Compile(spec, shapes, mapping)
    check(err)
    fmt.Println(bundle.Contracts()[0].EntityPattern) // *.*.*.intel.platform.*
    fmt.Println(bundle.Unwritable())                 // [intel.observation]

    id := "acme.edge.sensor.intel.platform.p1"
    now := time.Now()
    report, err := bundle.ValidateCandidateState(id, []message.Triple{
        {Subject: id, Predicate: "intel.measurement.altitude-m", Object: 120.5, Source: "demo", Timestamp: now},
        {Subject: id, Predicate: "intel.measurement.altitude-m", Object: 130.5, Source: "demo", Timestamp: now},
    })
    check(err)
    fmt.Println(report.Targeted, report.Conforms, report.Results[0].Component)
    // true false http://www.w3.org/ns/shacl#MaxCountConstraintComponent
}

func check(err error) {
    if err != nil {
        log.Fatal(err)
    }
}
```

The second altitude value breaks the shape's `sh:maxCount 1`, so the report does not conform. A program that wants
the pack and contracts in SemEngine registers `bundle.Pack()` with `vocabulary.RegisterPack`; the compiler never does.

The exported surface is:

| Name | What it does |
| --- | --- |
| `ParseTurtle`, `DecodeShapesJSON`, `DecodeShapesYAML` | Read a shapes document into a `ShapeSet` |
| `DecodeMappingJSON`, `DecodeMappingYAML` | Read a mapping document into a `Mapping` |
| `Compile(pack, shapes, mapping)` | Check everything and return a `*Bundle`, or a `*CompileError` listing every issue |
| `(*Bundle).Pack()`, `Contracts()`, `Unwritable()` | Copies of the compiled pack, the contracts, and the types with no contract |
| `(*Bundle).ValidateCandidateState(id, triples)` | Check one entity's complete state; returns a `Report` |

Importing `shacl` registers nothing. A failed `Compile` returns no bundle.

## Input formats

Three formats build the same in-memory `ShapeSet`. The DIEKM shapes in Turtle, JSON, and YAML decode to equal sets
and compile to equal bundles (`TestDIEKMShapesInThreeFormats`, `TestDIEKMBundle`).

### Turtle

`shacl.ParseTurtle` reads Turtle (the W3C RDF text syntax, [Turtle](https://www.w3.org/TR/turtle/)) through
`internal/turtle`, which implements a subset:

- Supported: `@prefix` and `@base`; absolute IRIs in `<...>`; prefixed names; the `a` keyword; labelled blank nodes
  (`_:label`); `[ ... ]` property lists; `( ... )` collections; `;` and `,` lists; `"..."` and `"""..."""` strings
  with the standard escapes, language tags and `^^datatype`; integer, decimal, double and boolean literals; `#`
  comments.
- Rejected by name, with the line and column: SPARQL-style `PREFIX` and `BASE`, single-quoted strings, and escapes in
  prefixed names. A shapes file written by another tool that uses them must be rewritten.
- Relative IRIs are rejected. Shapes documents must use absolute IRIs, even if they declare `@base`: with no base
  supplied, `@base` is checked but changes nothing. A scheme that RFC 3986 §3.1 does not allow (`<0:>`) is rejected
  too. Only the loader for the W3C test suite supplies a base.
- Input over 1 MiB (1,048,576 bytes) fails, naming the byte offset where the limit was crossed. Exactly 1 MiB is
  accepted (`TestParseTurtleSizeLimit`).
- Nesting of `[` and `(` deeper than 64 levels fails with the position of the 65th. Without a bound, 1 MiB of `[`
  reaches 174,755 levels and 256 MiB of stack, and one million levels kill the process (probe P1).
- Input that is not valid UTF-8 fails with the byte offset of the first bad byte.
- An RDF graph is a set of triples, but the reader returns every triple it reads. A W3C test file states two triples
  twice (probe P2), so `ParseTurtle` removes exact repeats. A repeated `sh:minCount 1` is one constraint.

Node shapes must be IRIs. A subject typed `sh:NodeShape`, or carrying any target predicate, is a node shape; if it is
a blank node, `ParseTurtle` fails, because a blank-node label depends on document order and would make names and
results order-dependent. Property shapes (the objects of `sh:property`) may be blank nodes. A property shape's
identity is its node shape plus its path.

`ParseTurtle` does not fail on a construct outside the profile. It records the construct, with the shape and path it
appears on, in the `ShapeSet`, and `Compile` reports all of them together. So `sh:in (1)` is reported with its
member, and a document with ten out-of-profile constructs gives one error naming ten issues.

### JSON and YAML

`DecodeShapesJSON` and `DecodeShapesYAML` read SemVocab's own form, not W3C SHACL or JSON-LD. The YAML below is
`shacl/testdata/diekm.shapes.yaml`:

```yaml
node_shapes:
  - id: "https://w3id.org/c360/diekm#IntelligencePlatformShape"
    target_classes: ["https://w3id.org/c360/diekm#IntelligencePlatform"]
    properties:
      - path: "https://w3id.org/c360/diekm#altitudeMeters"
        datatype: "http://www.w3.org/2001/XMLSchema#double"
        max_count: 1
      - path: "https://w3id.org/c360/diekm#assignedRole"
        datatype: "http://www.w3.org/2001/XMLSchema#string"
```

A node shape takes `id`, `target_classes`, `closed`, `ignored_properties`, `severity`, `messages`, and `properties`. A
property shape takes `id` (optional), `path`, `min_count`, `max_count`, `datatype`, `min_inclusive`, `max_inclusive`,
`min_exclusive`, `max_exclusive`, `pattern`, `in`, `severity`, and `messages`. Every IRI is absolute, and every
property shape has a path.

The decoders refuse:

- unknown fields at any depth, and a trailing value (JSON) or second document (YAML);
- a repeated object key;
- `null`: in JSON anywhere; in YAML wherever a string belongs. YAML `null` on `min_count`, `max_count`, or `closed`
  is the same as leaving the field out;
- a numeric bound. A bound is a string holding the lexical form (`min_inclusive: "0"`), and it takes its property's
  `datatype`, so it never passes through a float. In YAML every string must be tagged `!!str`, which quoting gives:
  `yaml.v3` otherwise decodes an unquoted `5` into a string field without error, so a decode into a struct alone
  would not enforce it (probe P3). `in: [1]` is refused rather than turned into `"1"`;
- a blank-node node shape (`id` starting `_:`), and a node shape listed twice.

The JSON and YAML forms cannot express an out-of-profile construct, so they never produce the issues Turtle can.
Every error names what it refused, but not every error carries a position. Syntax errors give a line (YAML) or a
byte offset (JSON), and invalid UTF-8 in YAML gives a byte offset. A JSON null or repeated key names its JSON path,
such as `$.node_shapes is null`. An unknown field names the field, and a model error, such as a relative IRI, names
the field and value. `yaml.v3` rejects unknown keys at every nesting level with the line (probe P3).

### The mapping document

SHACL says what values are valid. The mapping says which SemEngine names the shapes talk about. It is a separate
document so the shapes file stays plain W3C SHACL. `DecodeMappingYAML` and `DecodeMappingJSON` read it with the same
strictness as the shapes decoders. This is `packs/diekm/diekm.mapping.yaml`:

```yaml
entity_types:
  - class_iri: "https://w3id.org/c360/diekm#IntelligencePlatform"
    domain_types: [intel.platform]
  - class_iri: "https://w3id.org/c360/diekm#IntelligenceObservation"
    domain_types: [intel.observation]
predicates:
  - path_iri: "https://w3id.org/c360/diekm#altitudeMeters"
    name: intel.measurement.altitude-m
  - path_iri: "https://w3id.org/c360/diekm#assignedRole"
    name: intel.role.assigned
contracts:
  - domain_type: intel.platform
    groups:
      - name: measurement
        mode: reconcile
        predicates: [intel.measurement.altitude-m]
      - name: role
        mode: reconcile
        predicates: [intel.role.assigned]
```

The fields are `entity_types`, `predicates`, `contracts` (each with `domain_type`, optional `name`, `birth`, and
`groups` of `name`, `mode` and `predicates`), and `closed_ignored_predicates`. The rules are under
[Mapping rules](#mapping-rules) and [Projection contracts](#projection-contracts).

## Support matrix

The profile classifies every term of the W3C SHACL vocabulary: 183 terms in `shacl.ttl` (2017-07-20), vendored at
`internal/oracle/testdata/w3c-data-shapes/shacl/shacl.ttl`. Of the 221 subjects the file defines in the `sh:`
namespace, 38 are parameter declarations (such as `sh:AndConstraintComponent-and`), each declaring a term that is
defined on its own; leaving them out gives 183. Each term is:

- **supported**: the compiler accepts it where the table says;
- **ignored annotation**: accepted and dropped, because it validates nothing;
- **rejected**: compilation fails, naming the term, the shape, and the path.

The result is 21 supported, 4 ignored, and 158 rejected. A term the vocabulary does not contain, such as
`sh:minLenght`, is rejected as "not a term of the W3C SHACL vocabulary". A supported term in a position the table does
not allow (`sh:minCount` on a node shape) is rejected with "not accepted in this position or form".
`TestProfileClassifiesEveryTerm` reads the vendored vocabulary and fails if any term lacks exactly one classification
or is not handled as classified, and `TestProfileClassifiesEveryTermSensitivity` shows it can fail.

The tables below were produced from `shacl/profile.go`. No test compares this page with that file. If they disagree,
the Go table is right and this page needs fixing. Section numbers without a prefix are in the SHACL Recommendation. `AF`
marks [SHACL Advanced Features](https://www.w3.org/TR/shacl-af/) and `JS` marks
[SHACL JavaScript Extensions](https://www.w3.org/TR/shacl-js/), both W3C Working Group Notes (2017-06-08).
"Vocabulary only" marks eight terms the vocabulary declares and none of the three documents describes.

### Supported (21)

| Term | Section | Where it is accepted |
| --- | --- | --- |
| `sh:NodeShape` | 2.2 | As the optional `rdf:type` of a node shape |
| `sh:PropertyShape` | 2.3 | As the optional `rdf:type` of a property shape |
| `sh:targetClass` | 2.1.3.2 | On a node shape, with an IRI; each class needs a mapping row |
| `sh:property` | 4.7.2 | From a node shape to a property shape |
| `sh:path` | 2.3.1.1 | On a property shape, one predicate IRI |
| `sh:minCount` | 4.2.1 | On a property shape; an `xsd:integer` from 0 to 2,147,483,647; absent means 0 |
| `sh:maxCount` | 4.2.2 | On a property shape; same range; absent means unbounded; 0 means no value is allowed |
| `sh:datatype` | 4.1.2 | On a property shape; one of the six in [Datatypes](#datatypes-and-bounds) |
| `sh:minExclusive` | 4.3.1 | On a property shape that has `sh:datatype xsd:integer` or `xsd:double` |
| `sh:minInclusive` | 4.3.2 | As `sh:minExclusive` |
| `sh:maxExclusive` | 4.3.3 | As `sh:minExclusive` |
| `sh:maxInclusive` | 4.3.4 | As `sh:minExclusive` |
| `sh:pattern` | 4.4.3 | On a property shape; an `xsd:string` in the [regex subset](#patterns) |
| `sh:in` | 4.8.3 | On a property shape that has `sh:datatype xsd:string`; a list of `xsd:string` literals |
| `sh:closed` | 4.8.1 | On a node shape; an `xsd:boolean`. `false` is accepted and has no effect |
| `sh:ignoredProperties` | 4.8.1 | On a node shape; a list of IRIs. Without `sh:closed true` it has no effect |
| `sh:severity` | 2.1.4 | On a node or property shape; an IRI that is `sh:Violation`, `sh:Warning`, or `sh:Info` |
| `sh:Violation` | 2.1.4 | As a value of `sh:severity` |
| `sh:Warning` | 2.1.4 | As a value of `sh:severity` |
| `sh:Info` | 2.1.4 | As a value of `sh:severity` |
| `sh:message` | 2.1.5 | On a node or property shape; `xsd:string` literals, copied into results |

A property shape with a node-shape-only term (`sh:closed`), a second value for a single-valued term (two
`sh:maxCount` values), a value of the wrong kind (`sh:minCount 1.5`), or a missing `sh:path` is recorded and rejected.

### Ignored annotations (4)

These are accepted on any shape and carry no constraint. `rdfs:label` and `rdfs:comment` are ignored the same way;
they are not `sh:` terms, so they are not in the 183.

| Term | Section |
| --- | --- |
| `sh:name` | 2.3.2.1 |
| `sh:description` | 2.3.2.1 |
| `sh:order` | 2.3.2.2 |
| `sh:defaultValue` | 2.3.2.4 |

### Rejected (158)

Every other term is rejected. Each family below shares one reason, which is the text the error carries, followed by
the section in parentheses. The error also carries the shape and path. A node shape that is also an `rdfs:Class`
(an implicit class target) is rejected under `rdfs:Class`. A path that is an RDF list (a sequence path,
section 2.3.1.2) is rejected under `sh:path`.

#### Other targets (3)

Reason given in the error: only sh:targetClass targets are compiled.

| Term | Section |
| --- | --- |
| `sh:targetNode` | 2.1.3.1 |
| `sh:targetSubjectsOf` | 2.1.3.4 |
| `sh:targetObjectsOf` | 2.1.3.5 |

#### Other paths (5)

Reason given in the error: only a predicate path (one IRI) is compiled.

| Term | Section |
| --- | --- |
| `sh:alternativePath` | 2.3.1.3 |
| `sh:inversePath` | 2.3.1.4 |
| `sh:zeroOrMorePath` | 2.3.1.5 |
| `sh:oneOrMorePath` | 2.3.1.6 |
| `sh:zeroOrOnePath` | 2.3.1.7 |

#### Constraints on the referenced node (2)

Reason given in the error: needs the type or shape of the referenced node, which complete entity state does not hold.

| Term | Section |
| --- | --- |
| `sh:class` | 4.1.1 |
| `sh:node` | 4.7.1 |

#### Node kinds (8)

Reason given in the error: node kinds are not compiled (ruling Q17); entity_id predicates take cardinality only.

| Term | Section |
| --- | --- |
| `sh:nodeKind` | 4.1.3 |
| `sh:NodeKind` | 4.1.3 |
| `sh:IRI` | 4.1.3 |
| `sh:BlankNode` | 4.1.3 |
| `sh:Literal` | 4.1.3 |
| `sh:BlankNodeOrIRI` | 4.1.3 |
| `sh:BlankNodeOrLiteral` | 4.1.3 |
| `sh:IRIOrLiteral` | 4.1.3 |

#### String length and language constraints (4)

Reason given in the error: string-based component outside the profile.

| Term | Section |
| --- | --- |
| `sh:minLength` | 4.4.1 |
| `sh:maxLength` | 4.4.2 |
| `sh:languageIn` | 4.4.4 |
| `sh:uniqueLang` | 4.4.5 |

#### Regular-expression flags (1)

Reason given in the error: regular-expression flags are not compiled (ruling Q15).

| Term | Section |
| --- | --- |
| `sh:flags` | 4.4.3 |

#### Required values (1)

Reason given in the error: sh:hasValue is outside the profile.

| Term | Section |
| --- | --- |
| `sh:hasValue` | 4.8.2 |

#### Property-pair comparisons (4)

Reason given in the error: property-pair components compare two predicates and are outside the profile.

| Term | Section |
| --- | --- |
| `sh:equals` | 4.5.1 |
| `sh:disjoint` | 4.5.2 |
| `sh:lessThan` | 4.5.3 |
| `sh:lessThanOrEquals` | 4.5.4 |

#### Logical components (4)

Reason given in the error: logical components are outside the profile.

| Term | Section |
| --- | --- |
| `sh:not` | 4.6.1 |
| `sh:and` | 4.6.2 |
| `sh:or` | 4.6.3 |
| `sh:xone` | 4.6.4 |

#### Qualified value shapes (4)

Reason given in the error: qualified value shapes are outside the profile.

| Term | Section |
| --- | --- |
| `sh:qualifiedValueShape` | 4.7.3 |
| `sh:qualifiedMinCount` | 4.7.3 |
| `sh:qualifiedMaxCount` | 4.7.3 |
| `sh:qualifiedValueShapesDisjoint` | 4.7.3 |

#### Deactivation (1)

Reason given in the error: deactivation is outside the profile; remove the shape instead.

| Term | Section |
| --- | --- |
| `sh:deactivated` | 2.1.6 |

#### Property groups (2)

Reason given in the error: property groups are outside the profile.

| Term | Section |
| --- | --- |
| `sh:group` | 2.3.2.3 |
| `sh:PropertyGroup` | 2.3.2.3 |

#### The sh:Shape class (1)

Reason given in the error: only sh:NodeShape and sh:PropertyShape are accepted as a shape's type.

| Term | Section |
| --- | --- |
| `sh:Shape` | 2.1 |

#### The sh:Severity class (1)

Reason given in the error: custom severities are outside the profile; use sh:Violation, sh:Warning, or sh:Info.

| Term | Section |
| --- | --- |
| `sh:Severity` | 2.1.4 |

#### Constraint-component IRIs (29)

Reason given in the error: a constraint component IRI names a component in validation results; it is not a shapes construct.

| Term | Section |
| --- | --- |
| `sh:AndConstraintComponent` | 4.6.2 |
| `sh:ClassConstraintComponent` | 4.1.1 |
| `sh:ClosedConstraintComponent` | 4.8.1 |
| `sh:DatatypeConstraintComponent` | 4.1.2 |
| `sh:DisjointConstraintComponent` | 4.5.2 |
| `sh:EqualsConstraintComponent` | 4.5.1 |
| `sh:HasValueConstraintComponent` | 4.8.2 |
| `sh:InConstraintComponent` | 4.8.3 |
| `sh:LanguageInConstraintComponent` | 4.4.4 |
| `sh:LessThanConstraintComponent` | 4.5.3 |
| `sh:LessThanOrEqualsConstraintComponent` | 4.5.4 |
| `sh:MaxCountConstraintComponent` | 4.2.2 |
| `sh:MaxExclusiveConstraintComponent` | 4.3.3 |
| `sh:MaxInclusiveConstraintComponent` | 4.3.4 |
| `sh:MaxLengthConstraintComponent` | 4.4.2 |
| `sh:MinCountConstraintComponent` | 4.2.1 |
| `sh:MinExclusiveConstraintComponent` | 4.3.1 |
| `sh:MinInclusiveConstraintComponent` | 4.3.2 |
| `sh:MinLengthConstraintComponent` | 4.4.1 |
| `sh:NodeConstraintComponent` | 4.7.1 |
| `sh:NodeKindConstraintComponent` | 4.1.3 |
| `sh:NotConstraintComponent` | 4.6.1 |
| `sh:OrConstraintComponent` | 4.6.3 |
| `sh:PatternConstraintComponent` | 4.4.3 |
| `sh:PropertyConstraintComponent` | 4.7.2 |
| `sh:QualifiedMaxCountConstraintComponent` | 4.7.3 |
| `sh:QualifiedMinCountConstraintComponent` | 4.7.3 |
| `sh:UniqueLangConstraintComponent` | 4.4.5 |
| `sh:XoneConstraintComponent` | 4.6.4 |

#### Validation-report vocabulary (15)

Reason given in the error: validation-report vocabulary; it is not a shapes construct.

| Term | Section |
| --- | --- |
| `sh:ValidationReport` | 3.6.1 |
| `sh:conforms` | 3.6.1.1 |
| `sh:result` | 3.6.1.2 |
| `sh:shapesGraphWellFormed` | 3.6.1.3 |
| `sh:ValidationResult` | 3.6.2 |
| `sh:AbstractResult` | 3.6.2 |
| `sh:focusNode` | 3.6.2.1 |
| `sh:resultPath` | 3.6.2.2 |
| `sh:value` | 3.6.2.3 |
| `sh:sourceShape` | 3.6.2.4 |
| `sh:sourceConstraintComponent` | 3.6.2.5 |
| `sh:detail` | 3.6.2.6 |
| `sh:resultMessage` | 3.6.2.7 |
| `sh:resultSeverity` | 3.6.2.8 |
| `sh:sourceConstraint` | 5.3.2 |

#### Data-graph declarations (3)

Reason given in the error: a data-graph declaration; it is not a shapes construct.

| Term | Section |
| --- | --- |
| `sh:shapesGraph` | 3.3 |
| `sh:suggestedShapesGraph` | vocabulary only |
| `sh:entailment` | 1.5 |

#### SPARQL-based constraints and components (29)

Reason given in the error: SPARQL-based constraints and components are outside the profile.

| Term | Section |
| --- | --- |
| `sh:sparql` | 5.2 |
| `sh:SPARQLConstraint` | 5.2 |
| `sh:select` | 5.2 |
| `sh:SPARQLConstraintComponent` | 5 |
| `sh:prefixes` | 5.2.1 |
| `sh:declare` | 5.2.1 |
| `sh:PrefixDeclaration` | 5.2.1 |
| `sh:prefix` | 5.2.1 |
| `sh:namespace` | 5.2.1 |
| `sh:this` | 5.3.1 |
| `sh:ConstraintComponent` | 6.2 |
| `sh:Parameterizable` | vocabulary only |
| `sh:Parameter` | 6.2.1 |
| `sh:parameter` | 6.2.1 |
| `sh:optional` | 6.2.1 |
| `sh:labelTemplate` | 6.2.2 |
| `sh:Validator` | 6.2.3 |
| `sh:validator` | 6.2.3 |
| `sh:nodeValidator` | 6.2.3 |
| `sh:propertyValidator` | 6.2.3 |
| `sh:SPARQLSelectValidator` | 6.2.3.1 |
| `sh:SPARQLAskValidator` | 6.2.3.2 |
| `sh:ask` | 6.2.3.2 |
| `sh:SPARQLExecutable` | vocabulary only |
| `sh:SPARQLAskExecutable` | vocabulary only |
| `sh:SPARQLSelectExecutable` | vocabulary only |
| `sh:SPARQLConstructExecutable` | vocabulary only |
| `sh:SPARQLUpdateExecutable` | vocabulary only |
| `sh:update` | vocabulary only |

#### JavaScript extensions (13)

Reason given in the error: JavaScript extensions are outside the profile.

| Term | Section |
| --- | --- |
| `sh:JSExecutable` | JS 4 |
| `sh:jsFunctionName` | JS 4.1 |
| `sh:jsLibrary` | JS 4.2 |
| `sh:JSLibrary` | JS 4.3 |
| `sh:jsLibraryURL` | JS 4.3 |
| `sh:JSConstraint` | JS 6 |
| `sh:js` | JS 6 |
| `sh:JSConstraintComponent` | JS 7 |
| `sh:JSValidator` | JS 7.2 |
| `sh:JSFunction` | JS 8 |
| `sh:JSRule` | JS 9 |
| `sh:JSTarget` | JS 10.1 |
| `sh:JSTargetType` | JS 10.2 |

#### SHACL Advanced Features (other than rules) (19)

Reason given in the error: SHACL Advanced Features are outside the profile.

| Term | Section |
| --- | --- |
| `sh:target` | AF 3 |
| `sh:Target` | AF 3 |
| `sh:SPARQLTarget` | AF 3.1 |
| `sh:TargetType` | AF 3.2 |
| `sh:SPARQLTargetType` | AF 3.2 |
| `sh:resultAnnotation` | AF 4 |
| `sh:ResultAnnotation` | AF 4 |
| `sh:annotationProperty` | AF 4 |
| `sh:annotationValue` | AF 4 |
| `sh:annotationVarName` | AF 4 |
| `sh:Function` | AF 5 |
| `sh:returnType` | AF 5.3 |
| `sh:SPARQLFunction` | AF 5.4 |
| `sh:filterShape` | AF 6.3 |
| `sh:nodes` | AF 6.3 |
| `sh:intersection` | AF 6.6 |
| `sh:union` | AF 6.7 |
| `sh:expression` | AF 7 |
| `sh:ExpressionConstraintComponent` | AF 7 |

#### Rules (9)

Reason given in the error: SHACL rules are not compiled: rule translation is deferred to issue #1 Phase 4.

| Term | Section |
| --- | --- |
| `sh:rule` | AF 8 |
| `sh:Rule` | AF 8 |
| `sh:condition` | AF 8.2.1 |
| `sh:TripleRule` | AF 8.5 |
| `sh:subject` | AF 8.5 |
| `sh:predicate` | AF 8.5 |
| `sh:object` | AF 8.5 |
| `sh:SPARQLRule` | AF 8.6 |
| `sh:construct` | AF 8.6 |

### Other constructs in a shapes document

A shapes document may contain only node shapes, their property shapes, and the list cells those use. Anything else is
rejected:

- a triple whose subject is none of those, such as `owl:imports` or a data triple, is reported under its predicate
  with "outside the profile";
- a predicate on a shape that is not an `sh:` term and not `rdfs:label` or `rdfs:comment` is "outside the profile";
- an `rdf:type` on a shape other than `sh:NodeShape` or `sh:PropertyShape`, such as a rejected class, is reported
  under `rdf:type` with the class named;
- a node shape with no `sh:targetClass` fails, because it would never apply and its constraints would silently drop.

A compile collects every issue before it fails. `CompileError.Issues` is sorted by shape, path, construct, and message,
and no partial bundle is returned. The W3C SHACL test-suite files for each rejected family fail compilation with no
adapter (`TestW3CRejectedFamiliesFailCompile`).

## Datatypes and bounds

| `sh:datatype` | Pack `data_type` | SemEngine export writes (`vocabulary/export/object.go:36-47`) |
| --- | --- | --- |
| `xsd:string` | `string` | plain literal |
| `xsd:integer` | `int` | `xsd:integer` |
| `xsd:double` | `float` | `xsd:double` |
| `xsd:boolean` | `bool` | `xsd:boolean` |
| `xsd:dateTime` | `datetime` | `xsd:dateTime` |
| `rdf:JSON` | `json` | `rdf:JSON` |

Every other datatype is rejected, including `xsd:float`, `xsd:decimal`, the derived integer types, and
`rdf:langString`. Export never writes `xsd:float` or `xsd:decimal` for a native value, so a shape requiring either
could never pass once shapes are checked against exported RDF (`TestDatatypeTable`).

The compiled pack is the base pack, copied, in its original entry order. The one change: a predicate whose `data_type`
is empty takes the type its shapes imply. A `data_type` that a shape contradicts, or two shapes that imply different
types for one predicate, fail compilation.

**Entity references.** A path that maps to a pack predicate of `data_type` `entity_id` takes cardinality, severity,
and message only. A `sh:datatype`, bound, `sh:pattern`, or `sh:in` on it is rejected, one issue each
(`TestEntityIDCardinalityOnly`).

**Bounds.** A bound needs `sh:datatype xsd:integer` or `xsd:double` on the same property shape, and must be a literal
of that same datatype. Anything else is rejected, because SPARQL numeric promotion would compare through
`xsd:double` and could round.

- Integer bounds are held exactly as big integers. Their lexical space is `[+-]?[0-9]+` (XML Schema Part 2 §3.3.13.1).
- Double bounds use XSD 1.0's double lexical space (XML Schema Part 2 §3.2.5.1): `INF`, `-INF`, `NaN`, and a decimal
  mantissa with an optional exponent. `+INF` is not accepted. The compiler checks that grammar first because Go's
  `strconv.ParseFloat` accepts spellings XSD does not, such as `Inf`, hexadecimal floats, and underscores
  (`TestXSDLexicalForms`).
- A lower bound above an upper bound is rejected. Equal exclusive bounds are accepted although no value can pass.
- At validation, a value must be an `xsd:integer` or `xsd:double` literal, or it violates the bound. A `NaN` bound
  compares with nothing, so every value violates it. An integer is never rounded through `float64`: the test
  `TestNumericBoundsExact` shows that 2^53 and 2^53+1 compare correctly.

**`sh:datatype` at validation.** A value passes only if it is a literal of that datatype whose lexical form is valid
for it (SHACL §4.1.2 makes an ill-typed literal a violation). SemEngine's export can write ill-typed forms under a
supported datatype: a `dateTime` with a timezone offset beyond ±14:00 (XML Schema Part 2 1.1 §3.3.7.2 allows at
most that), and an `xsd:string` holding a character outside XML 1.0's `Char` (§2.2). The validator keeps export's
term and reports the violation (`TestValidateDatatypeLexicalSpace`).

## Patterns

`sh:pattern` means SPARQL `REGEX` ([SPARQL 1.1 §17.4.3.14](https://www.w3.org/TR/sparql11-query/)), which means
`fn:matches` in XPath and XQuery Functions and Operators 3.1 §5.6, whose syntax extends XML Schema Part 2 Appendix F.
Go's `regexp` package is RE2, a different dialect, so the compiler does not pass patterns through. It parses each one
against an allowlist and emits an equivalent RE2 expression. A pattern matches when some substring matches, as
`fn:matches` does; `^` and `$` anchor to the whole string. A value is matched as its IRI string or its lexical form;
a blank node violates (SHACL §4.4.3).

**Accepted:**

- literals; `|`; capturing groups `( )`; `^` and `$`;
- `*`, `+`, `?`, `{n}`, `{n,}`, `{n,m}` and their reluctant forms (a trailing `?`), with counts up to 1,000, the most
  RE2 accepts;
- `.`, which is written `[^\n\r]`, because XSD's `.` excludes both characters and RE2's excludes only `\n`;
- classes `[...]` and `[^...]` with ranges, where `-` is first, last, or between a range's ends;
- the single-character escapes `\n \r \t \\ \| \. \? \* \+ \( \) \{ \} \- \[ \] \^ \$`;
- `\p{X}` and `\P{X}` where X is one of the 36 general categories XSD lists: `L Lu Ll Lt Lm Lo`, `M Mn Mc Me`,
  `N Nd Nl No`, `P Pc Pd Ps Pe Pi Pf Po`, `Z Zs Zl Zp`, `S Sm Sc Sk So`, `C Cc Cf Co Cn`. Membership follows Unicode
  15.0, the version of Go 1.26.9's tables.

**Rejected, each by name:**

- `\d \D \w \W \s \S`: XSD's `\d` is `\p{Nd}` and its `\w` excludes only punctuation, separators and other control
  characters, so both are Unicode-wide, while RE2's are ASCII-only. XSD's `\s` is `[ \t\n\r]` and RE2's also matches
  `\f`;
- `\i \I \c \C`, which follow XML name rules RE2 has no equivalent for;
- block escapes such as `\p{IsBasicLatin}`; `\p{Cs}` and script names such as `\p{Greek}`, which RE2 compiles and XSD
  does not list;
- back-references; groups starting `(?`; class subtraction `-[` (RE2 compiles `[a-z-[aeiou]]` silently as a different
  class); any other escape, including `\x41`, `\A`, `\z`, and `\b`, which RE2 accepts and XSD does not define;
- an unescaped `{`, `}`, `]` or `)` outside its syntax, a quantifier on `^` or `$`, a quantifier after a quantifier,
  a range whose end is below its start, a count above 1,000, and group nesting deeper than 1,000;
- any `sh:flags`.

The RE2 and XSD facts above were checked at go1.26.9 and against XML Schema Part 2 Appendix F in probe P4. The
accepted-and-rejected sets are pinned by `TestPatternSubsetAccepts`, `TestPatternSubsetRejects` and
`TestPatternCompileIssues`, and agreement with the W3C `property/pattern-001` case and the `fn:matches` examples of
Functions and Operators 3.1 §5.6.3 by `TestPatternAgreesWithW3C`.

## Enumerations and closed shapes

**`sh:in`** takes a list of `xsd:string` literals on a property shape with `sh:datatype xsd:string`. An IRI, typed, or
language-tagged member is rejected, as is a missing or different datatype. In the model, `In` is a `*[]string`: nil
means no enumeration, and an empty slice means `sh:in ()`, which no value satisfies (SHACL §4.8.3). The compiled set
does not depend on list order. At validation a value passes only if it is an `xsd:string` literal equal to a member.

**`sh:closed true`** allows only the node shape's own property paths plus `sh:ignoredProperties`. In native state this
is checked over dotted predicates: a candidate triple violates the shape unless its predicate is mapped to an allowed
path or is listed in the mapping's `closed_ignored_predicates`. An ignored IRI that has a `predicates` mapping row
allows that predicate. An ignored IRI with no row (such as `rdf:type`) allows nothing in native state, since no native
predicate stands for it. A predicate the pack does not declare is a violation unless ignored.

A closed shape must list `rdf:type` in `sh:ignoredProperties`, or compilation fails. Native state carries no
`rdf:type` triple, but the exported graph will, and the W3C suite reports `rdf:type` as a closed-shape violation
(`core/node/closed-001`). The failing message cites that case, and `TestClosedShapes` compiles the vendored case
and requires the rejection.

## Mapping rules

Each rule below fails compilation when broken (`TestMappingRules`). `Compile` also checks the compiled pack with
SemEngine's `(*PackSpec).Validate`.

- **Explicit binding.** Every `sh:targetClass` has an `entity_types` row and every `sh:path` has a `predicates` row.
  The compiler never derives a dotted name from an IRI.
- **Agreement with the pack.** A predicate a path maps to is in the base pack, and its `iri` equals the path. Export
  writes that `iri` (`vocabulary/export/export.go:231-251`), so a mismatch would make native and exported graphs
  disagree.
- **Agreement with SHACL targeting.** SHACL targets instances of subclasses (§2.1.3.2). A class's row must list every
  base-pack entity type whose `class_iri` is the class or whose `superclasses` contain it, and nothing else. Leaving
  one out would quietly weaken validation.
- **One mapping per key.** A `class_iri` or `path_iri` with two rows fails, whether or not they agree. The same path in
  several shapes maps to the same predicate. Several shapes may target one class, and all their constraints apply.
- **Order does not matter.** `Compile` sorts a copy of every row list first, so the issues do not depend on how the
  author ordered the document.
- **Unused rows** are accepted.
- **`closed_ignored_predicates`** entries must be three-part dotted predicates; they need not be in the pack.

Predicates are three dotted parts (`intel.measurement.altitude-m`), and entity IDs are six parts
(`org.platform.system.domain.type.instance`), as SemEngine defines them.

## Projection contracts

A projection contract declares which predicates a writer owns and how it writes them: `reconcile` replaces a group's
whole set, `append` adds to it (`pkg/projection/contract/contract.go:22-31`). SemVocab cannot guess this from the
shapes, so the mapping states it. For each entity type with at least one predicate in `birth` or `groups`, `Compile`
emits one `contract.Contract`:

- `Name` is the mapping's `name`, else the `domain_type`. Two contracts with one name fail.
- `EntityPattern` is `*.*.*.<domain>.<type>.*`, so `*.*.*.intel.platform.*`. It matches
  `acme.edge.sensor.intel.platform.p1` and rejects `acme.edge.intel.other.platform.p1`.
- `Groups` come from the mapping. `mode` is required and must be `reconcile` or `append`. A group's name defaults to the
  predicate category, the middle dotted segment (`measurement` for `intel.measurement.altitude-m`), when every
  predicate shares it. Two declarations of one group name with different modes fail, as does a predicate in two
  groups. Write mode is never derived from `sh:maxCount`.
- `BirthPredicates` is exactly the mapping's `birth` list. Birth is never inferred from `sh:minCount`. A predicate in
  both birth and a group fails.
- `MessageType` is zero and `IndexingProfile` is empty. The program that registers the contract with SemEngine binds
  both with its payload type (`contract.go:43-47`).

Every contract passes SemEngine's `ValidateShape` (`contract.go:67-69`). `Compile` does not call `Validate`, which
reads the global registry; it checks instead that each contract predicate is declared in the compiled pack.

**Every predicate a shape constrains for an entity type must be in that type's contract**, in `birth` or a group. If it
is not, compilation fails: otherwise validation would cover a predicate no writer owns.

**The unwritable list.** A mapped entity type with no birth or group predicate gets no contract. `Bundle.Unwritable()`
lists it, sorted. Shapes still apply to it in validation; no contract declares how a writer changes it. DIEKM's
`intel.observation` has a shape with no property shapes, so it is unwritable.

## Validating candidate state

`(*Bundle).ValidateCandidateState(entityID, triples)` takes the complete state of one entity after a mutation, not a
patch. A consumer builds that candidate: the stored values, with a `reconcile` group replaced or an `append` added.
SemVocab exports no helper for it, because that would restate SemEngine's write semantics.

- **Targeting.** The entity ID's domain and type (positions 4 and 5) select the shapes mapped to that entity type.
  `Report.Targeted` is false when none apply, so "no shapes" is never mistaken for "conforms".
- **Counting.** Values are counted as distinct RDF terms, so two triples with the same term and different `Source` or
  `Timestamp` count once. `int(5)`, `float64(5)` and `uint8(5)` on an `int` predicate are one term.
- **Scope.** A predicate no applicable shape constrains is ignored, except by a closed shape. Every constraint of
  every applicable shape runs.
- **Purity.** The validator reads only the bundle. It never reads SemEngine's registries
  (`TestValidateReadsNoRegistry` registers a conflicting `data_type` and requires the same report).

### Reports

A `Report` has `Targeted`, `Conforms`, and `Results`. `Conforms` is true when there are no results, whatever their
severity (SHACL §3.6.1). A `Result` has:

| Field | Meaning |
| --- | --- |
| `FocusNode` | The entity ID |
| `SourceShape` | The property shape's IRI, or its node shape's when the property shape is a blank node; for `sh:closed`, the node shape's |
| `Path` | The `sh:path` IRI. For `sh:closed`, the IRI the pack gives the predicate, empty if the pack does not declare it |
| `Predicate` | The dotted predicate. SHACL's result has no such field; it is here because a closed-shape result can name a predicate with no IRI |
| `Component` | The constraint component IRI, such as `http://www.w3.org/ns/shacl#MaxCountConstraintComponent` |
| `Value` | The value node in N-Triples syntax, as export would write it; empty for `sh:minCount` and `sh:maxCount` |
| `Severity` | The shape's own `sh:severity`, or `sh:Violation` when it has none. A property shape does not inherit its node shape's |
| `Messages` | The shape's `sh:message` strings |

Results are sorted by source shape, path, component, value, predicate, severity, and messages, with exact repeats
removed. An entity IRI in `Value` is under SemEngine's default base `https://semengine.semanticstream.ing/entities/`
(`vocabulary/export/export.go:128-144`).

### Value terms

Native state holds Go values, and SHACL talks about RDF terms. The validator maps each value to the term SemEngine's
export would write at the pinned version, so native results and results on exported RDF can agree. The mapping is
SemVocab's restatement of an unexported SemEngine classifier (`vocabulary/export/object.go:88-107`).
`TestValueTermsAgreeWithExport` serializes a table of sample values through `export.SerializeToString`, reads the N-Triples
back, and compares terms. It fails when one mapping row changes. Probe P5 ran the first version of the table.

| Declared `data_type` | Accepted Go values | Term |
| --- | --- | --- |
| `int` | Go integer types; `float32` or `float64` that are whole with magnitude at most 2^53 | `xsd:integer` |
| `float` | Go numeric types, but not NaN or infinity | `xsd:double`, with `.0` added to a whole number |
| `bool` | `bool` | `xsd:boolean` |
| `datetime` | `time.Time`; an RFC 3339 string | `xsd:dateTime` |
| `json` | a string that `json.Valid` accepts | `rdf:JSON` |
| `string` | a string | a literal, unless the string is a valid entity ID, an absolute `http`, `https` or `urn` IRI, or a blank-node label, which export writes as an IRI or blank node |
| `entity_id` | a string that is a valid entity ID or blank-node label | an IRI or blank node |

A value the declared type cannot carry falls back to its observed type, as export does (`object.go:76-108`). So a
`float64` 1.5 on an `int` predicate is an `xsd:double` and fails `sh:datatype xsd:integer`. A string that export
turns into an IRI fails `sh:datatype xsd:string`, which is how an entity reference stored on a `string` predicate shows
up.

A `Datatype` hint on a triple is accepted on a declared predicate only when it names the datatype the predicate's
`data_type` exports as (or `@id` on an `entity_id` predicate) and export would write the same term with and without
it. Export writes a hinted value's `%v` form, which can differ (`float64(2^53)` with `xsd:integer` is written
`"9.007199254740992e+15"^^xsd:integer`), so those hints are input errors. On a predicate the bundle's pack does not
declare, any hint is accepted and the term is the one export writes for it (`object.go:120-143`), so a caller need not
filter other packs' triples; such a predicate carries no constraint, so the term matters only to closed shapes.

### Input errors

These return an `error` and no report, all listed together, sorted:

- an entity ID SemEngine's `ValidateEntityID` refuses (this returns at once);
- a triple whose subject is not the entity;
- a predicate `ParsePredicate` refuses;
- a nil object, NaN or infinity, a Go type outside string, `bool`, the integer and float types and `time.Time`, a
  refused hint, and a string object or hint that is not valid UTF-8.

Each is an error, not a violation, because the candidate cannot be mapped to a term. This deliberately differs from
export, which at the pinned version silently drops a triple whose object is nil, NaN or infinite, writes an
unsupported Go value as a plain string from its `%v` form, writes any hint verbatim even when ill-typed, and writes
each malformed UTF-8 byte as U+FFFD (`object.go:415-435`), which would make two distinct native values one exported
term. `TestValueTermDifferencesFromExport` asserts each of these differences.

### Revision boundary

A validated candidate is a snapshot, not a persisted guarantee. At the pinned SemEngine, a reconcile can be fenced on
`ExpectedRevision` (`pkg/projection/mutation_types.go:46-58`, the field at `:55`): graph-ingest applies it only while
the entity is still at that revision. An append carries no revision (`mutation_types.go:61-67`). A consumer that
appends against a stale read can therefore exceed `sh:maxCount` even though validation passed.

`TestStaleAppendRevisionBoundary` shows it: two writers read `{A}` and append `B` and `C`; each candidate conforms
alone, both appends can land, and the persisted `{A,B,C}` fails a maximum of 2. The rule:

- after a revision conflict, re-read the entity, rebuild the candidate, and validate again;
- when a maximum must hold, write through a reconcile fenced on `ExpectedRevision`.

Neither the compiler nor the validator enforces graph-wide invariants.

## Rules are not compiled

`sh:rule` is rejected, and so are the rest of SHACL Advanced Features §8: `sh:Rule`, `sh:TripleRule`, `sh:SPARQLRule`,
`sh:condition`, `sh:subject`, `sh:predicate`, `sh:object`, and `sh:construct`. The error says: "SHACL rules are not
compiled: rule translation is deferred to issue #1 Phase 4". Phase 4 is not present.

## Reproducing results

Run these from the repository root. `task verify` runs the unit tests, the committed fuzz seeds, and the Rapid
properties; it does not run benchmarks or a timed fuzz session.

| To check | Command |
| --- | --- |
| Everything the gates check | `task verify` |
| The profile classifies every term | `go test ./shacl -run 'TestProfile' -v` |
| The W3C SHACL test-suite cases | `go test ./shacl -run 'TestW3C' -v` |
| Value terms against SemEngine's serializer | `go test ./shacl/internal/agreement -v` |

The Rapid properties, with a fixed seed:

```bash
go test ./shacl -run '^(TestRapid.*|TestProjectionStepSwapChangesBundle)$' -count=1 -cpu 1 -race -v \
  -rapid.seed=20261016 -rapid.checks=100
```

**Rapid** is the property-testing library (`pgregory.net/rapid`); a property is a statement checked over generated
inputs. The properties cover reordering and relabelling a shapes document, reordering mapping rows, reordering and
repeating candidate triples, round-tripping the three formats, and keeping projection steps in the base pack's order.
A failing run prints the seed. Replay it with `-rapid.seed=<seed>`. Rapid also writes the failing case under
`shacl/testdata/rapid/<test>/` and replays every file there first; commit a found failure as a named regression test
(design D16).

**Fuzzing** uses Go's native fuzzing. The five targets are `FuzzTurtleReader` in `internal/turtle` and
`FuzzParseTurtleShapes`, `FuzzDecodeShapesJSON`, `FuzzDecodeShapesYAML`, and `FuzzPatternSubset` in `shacl`. The
property is: no panic, and an error or a result, where each error names a construct or a location. To run one:

```bash
go test ./shacl -run '^$' -fuzz '^FuzzPatternSubset$' -fuzztime 60s -fuzzminimizetime 10s -parallel 1
```

Go writes a crashing input under `testdata/fuzz/<Target>/` beside the test. Commit it there; plain `go test` then
runs it as a seed, so `task verify` guards against its return. The committed seeds include inputs at the size and
nesting limits, one sample of each rejected construct, and the DIEKM shapes.

## Benchmarks

These are measurements from one run on one machine. They are not guarantees and are not checked by any gate;
`task verify` does not run benchmarks. Expect other numbers on other hardware, Go versions, or load.

- **Command:** `go test ./shacl -run '^$' -bench . -benchmem -count 10`, from the repository root.
- **Code:** branch `claude/phase-2-shacl`, the commit that added this section.
- **Go:** go1.26.9, GOOS `darwin`, GOARCH `arm64`, GOMAXPROCS 12 (the default).
- **Machine:** Apple M3 Pro, 12 cores, a shared development host. The one-minute load average was 1.8 when the
  run started and 5.8 when it ended, so the slowest samples include contention.
- **Date:** 2026-10-16.

What each benchmark measures (`shacl/bench_test.go`):

- `BenchmarkCompile` times `Compile` only. The shapes, mapping, and pack are decoded before the timer starts.
  - `diekm`: the DIEKM pack, its two node shapes and two property shapes, and its mapping.
  - `shapes=N`: a synthetic pack of N predicates, with one property shape per predicate and ten property shapes
    per node shape. Every node shape targets one class, and one reconcile group holds every predicate. The
    property shapes rotate through four forms: `xsd:string` with a pattern and a three-member `sh:in`;
    `xsd:integer` with two bounds; `xsd:double` with one bound; and `xsd:boolean`. Each has `sh:maxCount 1`.
- `BenchmarkValidateCandidateState` times one `ValidateCandidateState` call. Each candidate conforms, so every
  constraint runs and no result is built.
  - `diekm`: an `intel.platform` entity with one altitude and one role.
  - `triples=N`: N triples, one per predicate, checked against the bundle compiled from `shapes=1000`. All 1,000
    property shapes run on every call, including those whose predicates have no triple. That is why ten triples
    cost far more than DIEKM's two.

Results over the 10 samples. Each column is the median; ns/op also gives the fastest and slowest sample. Across
samples, B/op varied by at most 177 bytes and allocs/op by at most 1.

| Benchmark | ns/op (median) | ns/op (min–max) | B/op | allocs/op |
| --- | ---: | ---: | ---: | ---: |
| `BenchmarkCompile/diekm` | 20,160 | 19,899–20,563 | 9,394 | 149 |
| `BenchmarkCompile/shapes=10` | 21,554 | 21,374–21,741 | 32,371 | 299 |
| `BenchmarkCompile/shapes=100` | 206,276 | 199,871–214,050 | 304,006 | 2,018 |
| `BenchmarkCompile/shapes=1000` | 2,205,978 | 2,172,153–2,362,969 | 3,589,413 | 19,587 |
| `BenchmarkValidateCandidateState/diekm` | 1,001 | 980–1,179 | 1,250 | 13 |
| `BenchmarkValidateCandidateState/triples=10` | 19,300 | 18,436–32,061 | 3,989 | 65 |
| `BenchmarkValidateCandidateState/triples=100` | 73,949 | 65,478–153,391 | 33,687 | 567 |
| `BenchmarkValidateCandidateState/triples=1000` | 551,464 | 519,709–684,225 | 418,428 | 5,604 |

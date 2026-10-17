# shacl-compilation Specification

## Purpose
SemVocab's `shacl` package compiles a bounded profile of W3C SHACL shapes, plus a mapping document, ahead of time
into a SemEngine `PackSpec`, projection contracts, and a native validator for complete candidate entity state, without
touching SemEngine's global registries; every construct outside the profile is rejected by name.

## Requirements

### Requirement: Published profile

`docs/shacl-compilation.md` SHALL classify every `sh:` term of the vendored W3C SHACL vocabulary as supported,
ignored annotation, or rejected, with its SHACL section and, for a rejected term, the reason. It SHALL publish the
accepted Turtle grammar, the datatype table, and the regex subset.

#### Scenario: A vocabulary term without a classification

- **WHEN** one term is removed from the profile table
- **THEN** `TestProfileClassifiesEveryTerm` fails and names that term

### Requirement: Constructs outside the profile are rejected by name

`Compile` SHALL fail on every construct outside the profile, including every `sh:rule` input, unsupported targets,
non-predicate paths, logical and SPARQL constraints, and unknown `sh:` terms. Each issue SHALL name the construct's
IRI, its shape, and its path. A node shape with no `sh:targetClass` SHALL also fail. `Compile` SHALL report every
issue and SHALL return no partial bundle.

#### Scenario: A shape with a rule

- **WHEN** a shapes graph contains `sh:rule`
- **THEN** compilation fails, the issue names `sh:rule` and the shape, and the message says rule translation is
  deferred

#### Scenario: Several rejected constructs

- **WHEN** a W3C test-suite case uses `sh:targetNode` and `sh:and`
- **THEN** the error lists both constructs

### Requirement: Bounded Turtle input

`ParseTurtle` SHALL accept only the documented Turtle subset with absolute IRIs. It SHALL fail with a line and column
on anything else or on nesting deeper than 64 levels, and with a byte offset on input over 1 MiB or input that is not
valid UTF-8. It SHALL never skip input.

#### Scenario: Deep nesting

- **WHEN** a shapes document nests 65 blank-node property lists
- **THEN** parsing fails with the nesting limit and its position

### Requirement: Equivalent input formats

A shape set expressed in Turtle, in SemVocab's JSON shape model, and in its YAML shape model SHALL compile to equal
bundles. The JSON and YAML decoders SHALL reject unknown fields, trailing documents, and numeric bounds that are not
strings.

#### Scenario: DIEKM in three formats

- **WHEN** the DIEKM shapes are compiled from Turtle, JSON, and YAML with the same pack and mapping
- **THEN** the three bundles are equal

### Requirement: Explicit mapping

Every `sh:targetClass` and `sh:path` SHALL be bound by the mapping document. A mapped predicate's pack `iri` SHALL
equal its path. A target class SHALL list exactly the pack entity types whose `class_iri` or `superclasses` contain
it. The compiler SHALL NOT derive a dotted name from an IRI.

#### Scenario: An unmapped path

- **WHEN** a property shape's path has no mapping row
- **THEN** compilation fails and names the path and its shape

### Requirement: Pure pack validation

The compiled `PackSpec` SHALL pass SemEngine's `(*PackSpec).Validate`. A shape datatype that conflicts with the base
pack's `data_type`, or with another shape for the same predicate, SHALL fail compilation. Projection steps SHALL keep
their base-pack order.

#### Scenario: Conflicting datatypes

- **WHEN** two shapes give one predicate `xsd:double` and `xsd:integer`
- **THEN** compilation fails and names the predicate and both shapes

### Requirement: No global registry access

`Compile` and `ValidateCandidateState` SHALL NOT read or mutate SemEngine's `vocabulary` or `vocabulary/export`
registries, and SHALL NOT snapshot, clear, or restore them. Their results SHALL NOT change when unrelated packs are
registered.

#### Scenario: Compiling leaves state unchanged

- **WHEN** DIEKM compiles once successfully and once with an invalid input
- **THEN** the registered predicates, entity types, their metadata, and serializer prefix output equal what they were
  before

### Requirement: Projection contracts

Each mapped entity type with birth or group predicates SHALL yield one contract with `EntityPattern`
`*.*.*.<domain>.<type>.*`, mapping-supplied group modes, and explicit birth predicates, passing `ValidateShape`. Mode
SHALL NOT be derived from `sh:maxCount`. Conflicting group modes and duplicate contract names SHALL fail. Other entity
types SHALL be listed as unwritable.

#### Scenario: The DIEKM platform pattern

- **WHEN** the DIEKM bundle's `intel.platform` contract is matched
- **THEN** `acme.edge.sensor.intel.platform.p1` matches and `acme.edge.intel.other.platform.p1` does not

#### Scenario: An entity type with no writable predicates

- **WHEN** the DIEKM shapes are compiled
- **THEN** `intel.observation` has no contract and is listed by `Unwritable()`

### Requirement: Cardinality semantics

An absent `sh:maxCount` SHALL mean unbounded and `sh:maxCount 0` SHALL prohibit values. Negative counts, counts above
2,147,483,647, and a `sh:minCount` above `sh:maxCount` SHALL fail compilation. Counts SHALL be taken over distinct RDF
terms.

#### Scenario: Maximum zero

- **WHEN** a candidate has one value for a property with `sh:maxCount 0`
- **THEN** the report has one `sh:MaxCountConstraintComponent` result

### Requirement: Datatypes and numeric bounds

`sh:datatype` SHALL accept only the published table. A numeric bound SHALL be a literal of its property's datatype,
`xsd:integer` or `xsd:double`. Integer bounds SHALL be compared exactly, never through `float64`. On a path mapped to an
`entity_id` predicate, only cardinality SHALL be accepted. A candidate value SHALL be typed as SemEngine's serializer at
the pin would type it, and `sh:datatype` SHALL also require a lexical form valid for the datatype.

#### Scenario: A large integer bound

- **WHEN** `sh:maxInclusive 9007199254740992` applies to the `int64` value 9007199254740993 on an `int` predicate
- **THEN** the report has one `sh:MaxInclusiveConstraintComponent` result, which a `float64` comparison would miss

### Requirement: Enumerations

`sh:in` SHALL accept only `xsd:string` members on a property with `sh:datatype xsd:string`. An absent `sh:in` SHALL
impose no constraint, and `sh:in ()` SHALL reject every value.

#### Scenario: A typed member

- **WHEN** `sh:in` lists `1`
- **THEN** compilation fails and names `sh:in` and the member

### Requirement: Regex subset

`sh:pattern` SHALL accept only the published subset of XPath regular expressions, and SHALL reject `sh:flags` and
every construct whose meaning differs between XPath and Go's RE2, naming it.

#### Scenario: A Unicode-sensitive escape

- **WHEN** a pattern contains `\d`
- **THEN** compilation fails and names `\d`

### Requirement: Complete-candidate-state validation

`ValidateCandidateState` SHALL take the complete post-mutation entity, not a patch. It SHALL return an error for a
malformed entity ID, a mismatched subject, or a value it cannot map to an RDF term (nil, NaN, infinity, an unsupported
Go type, invalid UTF-8, or a hint that would change the term). It SHALL run every constraint of every applicable shape,
report `Targeted` false when none applies, and order results deterministically.

#### Scenario: Reconcile versus append

- **WHEN** stored values `{A,B}` take reconcile `{B,C}` or append `{C}`, and each candidate is validated
- **THEN** the candidates are `{B,C}` and `{A,B,C}`, and each is checked against the shape's maximum

#### Scenario: Birth without a required value

- **WHEN** a birth candidate lacks a value for a property with `sh:minCount 1`
- **THEN** the report has a `sh:MinCountConstraintComponent` result

### Requirement: Revalidation boundary

`docs/shacl-compilation.md` SHALL state that a validated candidate is not a persisted-state guarantee. It SHALL state
that an append is not revision-fenced at the pinned SemEngine, and that a consumer SHALL revalidate after
re-reading on a revision conflict.

#### Scenario: A stale append

- **WHEN** two appends each validated against the same stored state would together exceed `sh:maxCount`
- **THEN** the documented boundary names this case, and the test shows that each candidate conformed alone

### Requirement: Deterministic artifacts

Bundles, compile issues, and reports SHALL be equal across permutations of shapes-graph statements, blank-node
labels, mapping rows, `sh:in` members, and candidate triples. Rapid and fuzz failures SHALL be reproducible from a
recorded seed or corpus file.

#### Scenario: Permuted shapes

- **WHEN** the DIEKM shapes are compiled from a random permutation of their statements
- **THEN** the bundle equals the bundle compiled from the file

### Requirement: A shape in both roles is rejected

An IRI that is a node shape and also the object of a node shape's `sh:property` SHALL be built in neither role.
`Compile` SHALL fail with exactly one issue for it. The issue SHALL name the IRI, what makes it a node shape, and
every node shape that refers to it, whatever the statement order.

#### Scenario: A property shape with its own target

- **WHEN** `ex:P` has `sh:targetClass ex:C`, `sh:path ex:p`, and `sh:minCount 1`, and `ex:A` has
  `sh:property ex:P`
- **THEN** compilation fails with one issue on `ex:P` that names `sh:targetClass` and `ex:A`, and no other issue is
  recorded against `ex:P`

### Requirement: JSON text is never replaced

`DecodeShapesJSON` and `DecodeMappingJSON` SHALL refuse input that is not valid UTF-8, and any `\u` escape naming a
lone UTF-16 surrogate, with the byte offset of the first such byte or escape. They SHALL NOT decode either into
U+FFFD.

#### Scenario: A malformed byte in an enumeration member

- **WHEN** a JSON shapes document has the byte `0xff` inside an `in` member
- **THEN** decoding fails with that byte's offset

#### Scenario: A lone surrogate escape

- **WHEN** a JSON mapping document has `"\ud800"` as a predicate name
- **THEN** decoding fails with the escape's offset, and `"😀"` in the same place still decodes

# Notes: task group 6 (validator)

Written 2026-10-16 with go1.26.9 darwin/arm64 and SemEngine `v0.1.0-alpha.2`
(`11da4721c2233a8e2b5b1d6236a97a44e06abbb6`). SemEngine citations are `file:line` in that module. The choices below
are the ones the design leaves open.

## Files and API

- `shacl/validate.go`: `Report`, `Result`, and `(*Bundle).ValidateCandidateState` (design D13).
  `shacl/terms.go`: the D9 value-term mapping.
- `Bundle` gains two unexported maps built from the compiled pack: each predicate's `data_type` and its `iri`. They
  are derived from the pack, so equal packs still give `reflect.DeepEqual` bundles.
- **`Result`** has `FocusNode` (the entity ID), `SourceShape`, `Path`, `Predicate`, `Component`, `Value`, `Severity`,
  and `Messages`. SHACL's result (§3.6.2) has no predicate; `Predicate` is there because a closed-shape result can
  name a predicate the bundle has no IRI for (D12), and its `Path` is then empty.
- **`Value`** is the value node in N-Triples syntax, as export writes it (`vocabulary/export/ntriples.go:46-60`):
  `<iri>`, `_:label`, `"lex"` for an `xsd:string` literal, `"lex"^^<datatype>` otherwise, with export's four string
  escapes (`object.go:415-435`). It is empty for `sh:minCount` and `sh:maxCount`, whose results have no value node.
  One string keeps the exported surface small and sorts and compares directly. An entity reference is an IRI under
  SemEngine's default base (`export.SubjectIRI(id, "")`, `export.go:128-144`); a consumer that exports with another
  base sees another IRI, but the same distinct-term counts.
- **`SourceShape`** for a blank property shape is its node shape's IRI, the rule compile issues already use
  (`notes/5-compiler.md`). For `sh:closed` it is the node shape.
- **`Severity`** is the shape's own `sh:severity`, or `sh:Violation` when it has none (§2.1.5). A property shape does
  not inherit its node shape's severity. `Messages` are the shape's `sh:message` strings.
- **Input errors** are one `error` listing every problem, sorted, prefixed `shacl: invalid candidate state`. There is
  no error type; D1 names none. A malformed entity ID returns at once, because no triple can be checked against it.

## 6.1 Validation (design D13)

- **Entity ID.** `semtypes.ParseEntityID` gives the domain and type (positions 4 and 5) and
  `semtypes.ValidateEntityID` (`pkg/types/entity_id.go:149-152`) checks the canonical form. Targeting is
  `bundle.shapes["<domain>.<type>"]`, which Compile keyed by entity type.
- **Which triples D9 maps.** Every triple is checked for subject and predicate, and every object goes through D9,
  whether or not a shape constrains its predicate. The `data_type` used is the bundle pack's; a predicate the pack
  does not declare, or declares with no `data_type`, is typed by observation, as export types an undeclared
  predicate (`object.go:98-106`). So the type-independent errors (nil, NaN, infinity, an unsupported Go type) are
  raised for every triple, targeted or not, which fails closed.
- **Hints on a predicate with no `data_type`.** D9 accepts a hint only when it names the datatype the predicate's
  `data_type` exports as, or `@id` on an `entity_id` predicate. With no `data_type` in the bundle there is no such
  datatype, so every hint there is an input error, including `@id` on another pack's entity reference. That reading
  is the design's text and fails closed. It means a consumer must pass only the bundle's predicates when its entity
  state carries hinted triples from other packs, which loses those predicates for closed shapes. This is raised on
  the pull request as an open question.
- **Distinct terms.** Values are grouped per predicate and deduplicated by term (kind, value, datatype). `int(5)`,
  `float64(5)`, and `uint8(5)` on an `int` predicate are one term, `"5"^^xsd:integer`, as export writes them.
- **Constraints.** Every property shape of every applicable node shape runs, then each closed shape:
  - `sh:minCount` and `sh:maxCount` count distinct terms.
  - `sh:datatype` compares the literal's datatype IRI. Every literal D9 returns is well formed for its datatype,
    because it is built from the Go value by export's own rules and hints may not change the term, so no lexical
    check is needed.
  - Bounds (§4.3) compare exactly: an integer value with an integer bound through `big.Int`, anything else through
    `big.Float` set exactly from the operands, never through a rounded `float64`. A value that is not an `xsd:integer`
    or `xsd:double` literal, or a `NaN` bound, is not comparable and violates. A mixed comparison (an `xsd:double`
    value on an integer bound) is exact where SPARQL would promote the integer to `xsd:double` (SPARQL 1.1 §17.3,
    operator mapping); the two differ only beyond 2^53, and such a value already fails `sh:datatype`.
  - `sh:pattern` (§4.4.3) matches an IRI's string or a literal's lexical form; a blank node violates.
  - `sh:in` (§4.8.3) over D11's string members: the value must be an `xsd:string` literal equal to a member.
  - `sh:closed` (D12): each value of a predicate outside the shape's allowed set is a result, with the pack's IRI for
    the predicate as path when it has one.
- **Order.** Results sort by source shape, path, component, value, then predicate, severity, and messages, so the
  order is total. Exact repeats are removed: a named property shape shared by two node shapes yields one result, as
  in a results graph.
- **Purity.** The validator reads `b.shapes`, `b.dataTypes`, and `b.iris` only. Its SemEngine calls are pure:
  `ParseEntityID` and `ValidateEntityID` (`pkg/types/entity_id.go:132-152`), `message.IsValidEntityID`
  (`message/triple.go:125-127`), `export.SubjectIRI` (`vocabulary/export/export.go:128-144`), and `ParsePredicate`
  (`vocabulary/predicate_contract.go:99`), which fills SemEngine's predicate-parse memo, not a registry (design,
  ownership inventory).

### Evidence (6.1)

- Tests: `TestValidateInputErrors`, `TestValidateTargeting`, `TestValidateCountsDistinctTerms`,
  `TestValidateEveryConstraint`, `TestValidateSpecScenarios` (the spec's `sh:maxCount 0` and 2^53 + 1 scenarios),
  `TestValidateBounds`, `TestValidateValueComponents`, `TestValidateClosed`, and `TestValidateDeterministic` in
  `shacl/validate_test.go`; `TestValidateReadsNoRegistry` in `shacl/diekm_test.go`. The expectations are SHACL
  Core's semantics written out by hand per component.
- **Fail-before.** Against a stub `ValidateCandidateState` that returned an empty report and no error, every test in
  `validate_test.go` failed (14 input-error subtests, and each of the other eight).
- **Registry.** `TestValidateReadsNoRegistry` records registered predicates, entity types with their metadata, and
  the serializer's prefix output before and after a validation and a failed validation, and requires them equal. It
  then registers the bundle's own predicate name with `data_type` `float`, asserts that the registration took effect,
  and requires the report to be unchanged: a validator that typed values from the registry would write float64 5 as
  `"5.0"^^xsd:double` and fail `sh:datatype xsd:integer`. Names are unique to each call, because `test:repeat` runs
  the test five times in one process. The test never restores state.
- **Shown able to fail.** Each seeded change was reverted:
  - data_type read from `vocabulary.GetPredicateMetadata` when registered: `TestValidateReadsNoRegistry` fails;
  - every triple counted, not every distinct term: `TestValidateCountsDistinctTerms` fails;
  - `Targeted` always true: `TestValidateTargeting` fails;
  - results left unsorted: five tests fail, `TestValidateDeterministic` among them;
  - only the first applicable shape run: `TestValidateEveryConstraint` fails;
  - only the first input error reported: `TestValidateInputErrors` fails;
  - integer comparison through `float64`: `TestValidateSpecScenarios` fails;
  - the closed shape's allowed set ignored: `TestValidateClosed` and `TestValidateDeterministic` fail.

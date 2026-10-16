# Design: phase-2-shacl-compilation

## Context

Issue #1 Phase 2 asks for a bounded, ahead-of-time SHACL compiler. Ahead of time means it runs when a pack is built
or loaded, never inside SemEngine's ingest or query paths. Its output, the *bundle*, holds three things:

- a SemEngine `vocabulary.PackSpec` (the pack metadata SemEngine registers);
- *projection contracts*: `contract.Contract` values from SemEngine's `pkg/projection/contract`, which declare which
  predicates a writer changes together and how (reconcile or append). They are unrelated to `ProjectionSpec`, the
  per-predicate reification chain;
- a *native validator*: Go code that checks a complete candidate entity state against the compiled constraints.

This design is drafted for the owner's review; nothing in it is implemented. All SemEngine citations are at
`github.com/c360studio/semengine v0.1.0-alpha.2` (`11da4721c2233a8e2b5b1d6236a97a44e06abbb6`), read from the module
cache. Owner rulings Q1–Q11 stand; Q12–Q24 below are open, each with a proposed default.

Review: round 1 pending (architect contract: three rounds, then the owner).

## Ownership inventory

| Surface | SemEngine at the pin | Disposition |
| --- | --- | --- |
| Pack validation | `(*PackSpec).Validate` `vocabulary/pack.go:186-230`, pure by its doc comment (`pack.go:180-185`); it calls `PrefixSpec.Validate` (`pack.go:24`), `ParsePredicate` (`predicate_contract.go:99`), `ParseEntityTypeKey` (`entity_type.go:80`), and `validatePredicateMetadataIn` (`registry.go:379-414`), which runs `ProjectionSpec.Validate` (`projection.go:65-95`) | Use directly; no SemVocab copy |
| Datatype tokens | `DataTypeString` … `DataTypeJSON`, `vocabulary/predicates.go:157-173`; membership check unexported by owner ruling (`predicates.go:179-183`) | Use the exported constants directly |
| Contract shape | `Contract`, `PredicateGroup`, `ModeReconcile`/`ModeAppend`, `ValidateShape`, `pkg/projection/contract/contract.go:24-69` | Use directly. `Contract.Validate` and `ValidateContracts` read the global registry through `RequireDeclaredPredicate` (`contract.go:130-133`, `141-155`), so the compiler never calls them |
| Entity IDs and patterns | `ValidateEntityID`, `ValidateEntityIDPattern`, `MatchEntityIDPattern`, `pkg/types/entity_id.go:149-185` | Use directly |
| Candidate triples | `message.Triple` (`message/triple.go:37-93`, including `Datatype` at `:93`) and `EntityReferenceDatatype = "@id"` (`:15`) | Use directly; `message` pulls in no NATS package (`go list -deps ./message` at the pin) |
| SHACL IRIs (`sh:`) | None: no constant, no `sh` serializer prefix (`vocabulary/export/prefix.go:25-37`) | New content, unexported, in `shacl`; source: the W3C SHACL vocabulary, 2017-07-20 (Q21) |
| XSD and `rdf:` IRIs | Unexported constants `vocabulary/export/object.go:17-21`; `xsd` and `rdf` serializer prefixes only (`prefix.go:25`, `:35`) | New content, unexported, in `shacl`; not a catalog. The exported-constant rule does not apply because SemEngine exports none |
| Native value to RDF term | `classifyObject` and helpers, `vocabulary/export/object.go:88-201`, `321-365`; unexported, and it reads the global registry (`object.go:99`) | Not callable purely. The validator defines its own documented mapping and a test checks it against SemEngine's serializer (D9, Q22) |
| Turtle reader | None | SemVocab's `internal/turtle` (Phase 1 design P4), promoted to production use (D2, Q12) |
| Property-based tests | `pgregory.net/rapid v1.3.0` already in SemEngine's `go.mod:25` and this repo's `go.sum` | Direct test-only dependency; MPL-2.0 |

No upstream gap blocks Phase 2. One optional upstream request is named in Q22.

`(*PackSpec).Validate` writes one process-wide cache: `ParsePredicate` memoizes predicates it accepts
(`predicate_contract.go:85`, `:128`, `:135-142`). The cache is not a vocabulary registry and changes no getter's
result. It is recorded here so a review does not mistake it for a registry mutation.

## Adopter seam inventory

- **Inbound.** No caller of `shacl` exists. A GitHub code search of the C360Studio organization for `semvocab`
  (2026-10-16) found one hit, `C360Studio/semsource` `graph/contract.go` at `e4febc0d2e23bf33811edc20b9a4159cf5275dc4`.
  That file imports `github.com/c360studio/semstreams/vocabulary` under the alias `semvocab`, which is not this
  module. The same file builds a reconcile contract by hand (pattern `*.*.*.*.*.*`, group `source`): the type that
  compiled contracts will replace for consumers that adopt shapes. Issue #1 names semsource, semconnect, and semteams
  as the consumers of adapter-side validation.
- **SemSPARQL** reads packs, not bundles. semsparql#1 states that it "does not run SHACL itself"; its export feeds an
  external validator in Phase 3. Phase 2 adds nothing SemSPARQL calls.
- **Outbound.** The SemEngine symbols in the ownership table, all at the pin. The behaviours this design relies on
  are each covered by a test named below, so a pin move that changes one fails loudly.
- **Merge order.** The only open pull request is this change's draft, #9. Nothing overlaps.

## Goals and non-goals

Goals: the #1 Phase 2 acceptance list, unchanged. Non-goals: as the proposal lists. Phase 3 export shapes, reified
paths, and external-validator agreement are not designed here.

## Decisions

### D1. Package layout and API

A new public package `shacl` (the files #1 names) exports:

- the shape model: `ShapeSet`, `NodeShape`, `PropertyShape`, `Literal`;
- `ParseTurtle`, `DecodeShapesJSON`, `DecodeShapesYAML`, each returning a `ShapeSet`;
- the mapping document `Mapping`, with `DecodeMappingJSON` and `DecodeMappingYAML`;
- `Compile(pack vocabulary.PackSpec, shapes ShapeSet, m Mapping) (*Bundle, error)`;
- `Bundle`, with `Pack()`, `Contracts()`, and `Unwritable()` returning copies, and
  `ValidateCandidateState(entityID string, triples []message.Triple) (Report, error)`;
- `Report`, `Result`, and `CompileError`, whose `Issues` list each rejection.

Exported signatures name no `internal/` type. Importing the package registers nothing; there is no `Register…`
function, because registration is SemEngine's `RegisterPack`, called by a composition root with `bundle.Pack()`.
Final identifiers may change during implementation; the observable behaviour below may not.

### D2. Turtle input: promote `internal/turtle` (Q12)

The probe (2026-10-16, Go proxy) found no tagged release of the two Go Turtle parsers searched for:
`github.com/knakk/rdf` and `github.com/deiu/rdf2go` list no versions. A third-party RDF module would also become a
transitive dependency of every consumer. The proposed default promotes the existing reader. A probe on 2026-10-16
copied `internal/turtle/turtle.go` at `5fdff15` into a scratch module. It parsed `packs/diekm/diekm.shacl.ttl` (11
triples) and the W3C SHACL vocabulary (1,127 triples) without error. It rejected the SHACL test-suite file
`core/node/closed-001.ttl` at its first relative IRI (`<>`, line 28).

Changes to the reader:

- The package comment stops saying "test-only"; production code may import it.
- New caps: input of at most 1 MiB and nesting (`[`, `(`) of at most 64 levels. Either limit fails with the line and
  column. Today the reader recurses without a depth bound.
- An option resolves relative IRIs against a base the caller supplies (RFC 3986 §5.2). Only the test-suite loader
  passes one. `shacl.ParseTurtle` passes none, so shapes documents must use absolute IRIs.
- The grammar subset is otherwise unchanged (`internal/turtle/turtle.go:1-20`). SPARQL-style `PREFIX`/`BASE`,
  single-quoted strings, and escapes in prefixed names stay rejected, by name. A shapes file from another tool that
  uses them must be rewritten; the support matrix says so.

### D3. Shape model and the JSON/YAML form

The model is SemVocab's own serialization, not W3C SHACL or JSON-LD. JSON decoding refuses unknown fields and
trailing values; YAML decoding uses `KnownFields(true)` and refuses trailing documents, as SemEngine's loaders do
(`vocabulary/pack.go:136-178`). Numeric bounds are strings holding the lexical form (`min_inclusive: "0"`). A JSON
number or a YAML node whose tag is not `!!str` is rejected, so no bound passes through `float64` and the design does
not depend on how `yaml.v3` coerces scalars. Node shapes must be IRIs: a blank-node node shape gets a label that
depends on document order, which would make names and results order-dependent. Property shapes may be blank nodes;
their identity is their node shape plus their path.

### D4. Supported profile

The profile is the set of shapes the compiler accepts. Every `sh:` term in the vendored W3C vocabulary (183 terms) is
classified in `docs/shacl-compilation.md` as *supported*, *ignored annotation*, or *rejected*. An `sh:` term absent
from the vocabulary is rejected as unknown. Supported (SHACL Core section in brackets):

- **Shapes:** `rdf:type sh:NodeShape` / `sh:PropertyShape` (optional); `sh:property` from a node shape to a property
  shape.
- **Target:** `sh:targetClass` with an IRI (§2.1.3.2), bound by the mapping (D6).
- **Path:** a predicate path, one IRI (§2.3.1.1).
- **Cardinality** (§4.2): `sh:minCount` and `sh:maxCount`, each an `xsd:integer` literal (§4.2.1, §4.2.2). In the
  model `MinCount` is an `int` (absent means 0) and `MaxCount` is a `*int` (nil means unbounded; 0 prohibits values).
  Negative values, values above `math.MaxInt32`, and `minCount > maxCount` are rejected.
- **Datatype** (§4.1.2): `sh:datatype` from the D8 table. Any other datatype is rejected.
- **Numeric bounds** (§4.3): `sh:minInclusive`, `sh:maxInclusive`, `sh:minExclusive`, `sh:maxExclusive` (D8).
- **Pattern** (§4.4.3): `sh:pattern` in the regex subset (D10), with no `sh:flags` (Q15).
- **Enumeration** (§4.8.3): `sh:in`, string-only (D11, Q16).
- **Closed** (§4.8.1): `sh:closed true` with `sh:ignoredProperties` (D12, Q20). `sh:closed false` is accepted and
  has no effect.
- **Severity and messages:** `sh:severity` with `sh:Violation`, `sh:Warning`, or `sh:Info`, and `sh:message` string
  literals; both are copied into results. Any result makes `Conforms` false, whatever its severity (§3.6.1).
- **Ignored annotations:** `sh:name`, `sh:description`, `sh:order`, `sh:defaultValue` (non-validating, §2.3.2),
  `rdfs:label`, `rdfs:comment`.

Rejected, each by its IRI with the shape and path where it appears:

- **Other targets:** `sh:targetNode`, `sh:targetSubjectsOf`, `sh:targetObjectsOf`, implicit class targets (a shape
  that is also an `rdfs:Class`), and `sh:target`.
- **Other paths** (§2.3.1.2–§2.3.1.7): sequence paths (an RDF list), `sh:alternativePath`, `sh:inversePath`,
  `sh:zeroOrMorePath`, `sh:oneOrMorePath`, `sh:zeroOrOnePath`.
- **Constraints needing graph context or not in the profile:** `sh:class` and `sh:node` need the referenced node's
  type or shape, which complete entity state does not hold. Also `sh:nodeKind` (Q17), `sh:minLength`, `sh:maxLength`,
  `sh:languageIn`, `sh:uniqueLang`, `sh:flags`, `sh:hasValue`, the property-pair components (`sh:equals`,
  `sh:disjoint`, `sh:lessThan`, `sh:lessThanOrEquals`), the logical components (`sh:not`, `sh:and`, `sh:or`,
  `sh:xone`), the qualified components, `sh:deactivated`, and `sh:group`.
- **SPARQL and JavaScript:** `sh:sparql`, `sh:select`, `sh:ask`, `sh:prefixes`, `sh:declare`, custom constraint
  components (`sh:ConstraintComponent`, `sh:parameter`, the validators), and every `sh:JS…` term.
- **SHACL Advanced Features:** `sh:rule`, `sh:TripleRule`, `sh:SPARQLRule`, `sh:condition`, `sh:expression`,
  `sh:SPARQLTarget`, `sh:SPARQLFunction`, and the rest of the vocabulary's AF terms. Rule translation is deferred to
  #1 Phase 4. The message for `sh:rule` says so.
- **Anything else in the shapes document:** a triple whose subject is not a node shape, a property shape, or an RDF
  list cell those shapes use is rejected. This covers `owl:imports`. A non-`sh:` predicate on a shape, other than
  the annotations above, is also rejected. Nothing is skipped.

A compile collects every rejection before failing. `CompileError.Issues` lists them sorted by shape, path, and term,
and no partial bundle is returned.

### D5. Compilation inputs (Q14)

`Compile` takes a base `PackSpec` as well as shapes and a mapping. SHACL carries no superclass closure, description,
units, or reification chain, and the pack that holds them is already the reviewed artifact (Phase 1,
`packs/diekm/diekm.yaml`). The compiled pack is the base pack in its original entry order, with one change: a mapped
predicate whose base `data_type` is empty takes the datatype its shapes imply (D8). A base `data_type` that disagrees
with a shape, or two shapes that imply different datatypes for one predicate, fails compilation. That is the "conflicting
global predicate metadata" case in #1. The compiled pack must pass `(*PackSpec).Validate`.

### D6. Mapping document (Q13, Q24)

The mapping is a SemVocab YAML or JSON document. The proposed default keeps it separate from the Turtle, so the
shapes file stays plain W3C SHACL that a Phase 3 external validator reads unchanged, and SemVocab mints no annotation
namespace that would need its own provenance. Fields:

- `entity_types`: each `class_iri` (an `sh:targetClass` value) with the list of `domain_type` keys it targets.
- `predicates`: each `path_iri` (an `sh:path` value) with exactly one dotted predicate `name`.
- `contracts`: per `domain_type`, an optional `name`, `birth` predicates, and `groups` (`name` optional, `mode`
  required, `predicates`).
- `closed_ignored_predicates`: dotted predicates a closed shape ignores in native state (D12).

Rules, each failing compilation when broken:

- **Explicit binding.** Every `sh:targetClass` and every `sh:path` in the shapes has a mapping row. The compiler never
  derives a dotted name from an IRI.
- **Agreement with the pack.** A predicate a path maps to is in the base pack, and its `iri` equals the path. Export
  writes that `iri` (`vocabulary/export/export.go:231-251`), so a mismatch would make native and exported graphs disagree.
- **Agreement with SHACL targeting.** Every `domain_type` listed for class C is a base-pack entity type whose
  `class_iri` is C or whose `superclasses` contain C. Every base-pack entity type that meets that condition is listed.
  SHACL targets instances of subclasses (§2.1.3.2), and Phase 1 packs carry full closures (Phase 0 design D5), so
  leaving one out would quietly weaken validation (Q24).
- **One mapping per path.** The same path in several shapes maps to the same predicate. Several shapes may target one
  class; all their constraints apply.

### D7. Projection contracts

For each mapped entity type with at least one predicate in `birth` or `groups`, the compiler emits one
`contract.Contract`:

- `Name`: the mapping's `name`, else the `domain_type` (`intel.platform`).
- `EntityPattern`: `*.*.*.<domain>.<type>.*`, so `*.*.*.intel.platform.*`, checked by `ValidateEntityIDPattern`.
  The pattern matches `acme.edge.sensor.intel.platform.p1` and rejects `acme.edge.intel.other.platform.p1`
  (`MatchEntityIDPattern`, `entity_id.go:169-185`).
- `Groups`: from the mapping. A group's name defaults to the predicate category, the middle dotted segment
  (`measurement` for `intel.measurement.altitude-m`). `mode` is required. Two declarations of one group name in a
  contract with different modes fail. Write mode is never derived from `sh:maxCount`.
- `BirthPredicates`: exactly the mapping's `birth` list. Birth membership is never inferred from `sh:minCount` or a
  group named `birth`. A predicate in both birth and a group fails, as `ValidateShape` requires
  (`contract.go:116-124`).
- `MessageType` zero and `IndexingProfile` empty: the composition root binds both when it registers the contract
  with a payload type (`contract.go:43-47`).

Every contract passes `ValidateShape` (`contract.go:67-69`). Instead of `Validate`, the compiler checks that each
contract predicate is declared in the compiled pack, which needs no registry. A predicate constrained by a shape for
that entity type and absent from both birth and groups fails compilation (Q19). An entity type with no birth or
group predicates gets no contract. It is listed in `Bundle.Unwritable()`, and `docs/shacl-compilation.md` explains
why (Q18). DIEKM's `intel.observation` shape has no property shapes, so it is the first such case. Contracts, groups,
and predicate lists are sorted by name.

### D8. Datatype table and numeric bounds

| `sh:datatype` | Pack `data_type` | Export writes (`object.go:36-47`) |
| --- | --- | --- |
| `xsd:string` | `string` | plain literal |
| `xsd:integer` | `int` | `xsd:integer` |
| `xsd:double` | `float` | `xsd:double` |
| `xsd:boolean` | `bool` | `xsd:boolean` |
| `xsd:dateTime` | `datetime` | `xsd:dateTime` |
| `rdf:JSON` | `json` | `rdf:JSON` |

Every other datatype is rejected. That includes `xsd:float` and `xsd:decimal`: SemEngine's export never writes them
for a native value, so a shape requiring either could never pass in Phase 3. It also includes the derived integer
types and `rdf:langString`. `entity_id` has no `sh:datatype`, so such predicates take cardinality constraints only
(Q17).

A bound needs an `sh:datatype` on its property shape. It must be a literal of that same datatype, which must be
`xsd:integer` or `xsd:double`; any other combination is rejected, because SPARQL numeric promotion would compare
through `xsd:double` and could round. Integer bounds are held as `math/big.Int` and compared exactly. Double bounds use
the XSD lexical space, including `INF`, `-INF`, and `NaN`, checked by an explicit grammar before `strconv.ParseFloat`,
which accepts spellings XSD does not. A lower bound above an upper bound is rejected.

### D9. Native value terms (Q22)

The validator maps each candidate value to the RDF term SemEngine's export would write at the pin, so native and
Phase 3 results can agree:

- `int` predicates take Go integer types, and `float32`/`float64` values that are whole with magnitude at most 2^53
  (`object.go:204-227`).
- `float` predicates take Go numeric types, but not NaN or infinities, which export refuses (`object.go:321-330`).
- `bool` takes `bool`; `datetime` takes `time.Time` or an RFC 3339 string; `json` takes a string that `json.Valid`
  accepts.
- A value the declared type cannot carry falls back to its observed type, as export does (`object.go:76-108`). So a
  `float64` 1.5 on an `int` predicate becomes an `xsd:double` value that fails `sh:datatype xsd:integer`.
- On a `string` predicate, a string that is a valid entity ID, an absolute IRI, or a blank-node label is an IRI or
  blank node in export (`object.go:151-157`, `341-365`). It therefore fails `sh:datatype xsd:string`.
- A triple with a `Datatype` hint is accepted only when the hint names the XSD datatype the predicate's `data_type`
  exports as, or `@id` on an `entity_id` predicate.

Each of these is checked against SemEngine's serializer by D15's agreement test. A nil object, an unsupported Go
type, NaN or infinity, or any other hint is an input error, not a violation.

### D10. Regex subset (Q15)

`sh:pattern` means SPARQL `REGEX` (SPARQL 1.1 §17.4.3.14). That in turn means `fn:matches` from XPath Functions and
Operators, whose syntax extends XML Schema Part 2 Appendix F. Handing the text to Go's `regexp.Compile` is not that.
The compiler parses each pattern with its own allowlist and emits RE2:

- **Accepted:** literals; `.`, translated to `[^\n\r]` because XSD's `.` excludes both characters; `^` and `$`;
  `|`; capturing `( )`; `* + ? {n} {n,} {n,m}` and their reluctant forms; classes and negated classes with ranges;
  single-character escapes; `\p{…}` and `\P{…}` for Unicode general categories.
- **Rejected by name:** `\d \D \w \W \s \S`, which are Unicode-wide in XSD and ASCII-only in RE2; `\i \I \c \C`;
  block escapes `\p{Is…}`; back-references; `(?`; class subtraction `-[`; any escape XSD does not define; any
  `sh:flags`.

Each claim on XSD or RE2 behaviour above is confirmed by probe P4 at go1.26.9 before implementation. Agreement is
then checked against the W3C test-suite pattern cases and the XPath examples.

### D11. Enumerations (Q16)

`sh:in` takes a list of `xsd:string` literals on a property shape with `sh:datatype xsd:string`. An IRI, typed, or
language-tagged member is rejected, which keeps RDF term distinctions honest. In the model, `In` is a `*[]string`:
nil means no enumeration, and an empty slice means `sh:in ()`, which no value satisfies (§4.8.3). The compiled set
does not depend on list order.

### D12. Closed shapes and ignored predicates (Q20)

A closed shape's allowed predicates are its own property paths plus `sh:ignoredProperties`. In native state the
check runs over dotted predicates: a candidate triple violates the shape unless its predicate is mapped to an
allowed path or appears in `closed_ignored_predicates`. A predicate the bundle does not map has no IRI here; export
would write a fallback IRI for it, so it is a violation unless ignored. Native state carries no `rdf:type` triple,
but the exported graph will, and the W3C suite reports `rdf:type` as a closed-shape violation (`core/node/closed-001`).
So a closed shape whose `sh:ignoredProperties` lacks `rdf:type` fails compilation, with that reason.

### D13. Complete-candidate-state validation

`ValidateCandidateState(entityID, triples)` takes the complete entity state after the mutation, within the bundle's
scope. It does not take a patch.

- **Input errors** (returned as `error`): an `entityID` that fails `ValidateEntityID`, a triple whose subject differs,
  a predicate that fails `ParsePredicate`, and the D9 representation errors.
- **Targeting.** The entity's `domain.type` (ID positions 4 and 5) selects the shapes mapped to that entity type.
  `Report.Targeted` is false when none apply, so a caller cannot mistake "no shapes" for "conforms".
- **Counting.** Values are counted as distinct RDF terms (D9). Two triples with the same term and different `Source`
  or `Timestamp` count once.
- **Scope.** Predicates no applicable shape constrains are ignored, except by closed shapes (D12).
- **Completeness.** Every constraint of every applicable shape runs. Results are sorted by source shape, path,
  constraint component, and value.
- **Purity.** The validator reads only the bundle and never SemEngine's registries.

Candidate construction is the consumer's job. SemVocab exports no helper, because that would restate SemEngine's
reconcile and append semantics. Tests build candidates with a test-only model of `contract.go:22-31`: reconcile
replaces the group's set, and append adds to it.

**Revision boundary.** A validated candidate is a snapshot, not a persisted guarantee. At the pin, a reconcile can be
fenced on `ExpectedRevision` (`pkg/projection/mutation_types.go:46-58`, field at `:55`). An append carries no revision
(`mutation_types.go:61-67`). A consumer appending against a stale read can therefore exceed `sh:maxCount` even after
validation passes. `docs/shacl-compilation.md` states the rule: revalidate after re-reading on a revision conflict,
and use a fenced reconcile when a maximum must hold. Neither the compiler nor the validator enforces graph-wide
invariants.

### D14. Side-effect freedom

Neither `Compile` nor `ValidateCandidateState` calls `RegisterPack`, `Register`, `RegisterEntityType`,
`export.Register`, `export.RegisterPrefixes`, `Contract.Validate`, `ValidateContracts`, `GetPredicateMetadata`, or
`ClearRegistry`/`SnapshotRegistry`. `TestSHACLCompilerOutputsValidPackAndContracts` checks this from the outside:

- It records `ListRegisteredPredicates`, `ListRegisteredEntityTypes`, each getter's metadata, and whether the `diekm`
  prefix compacts in Turtle output.
- It compiles DIEKM once successfully and once with an invalid input.
- It asserts that the recorded state is equal afterwards; the test never restores state.
- In a separate test, it registers an unrelated pack between two compiles and asserts that the two bundles are equal,
  after asserting that the registration took effect.

### D15. Test oracles

Each oracle is independent of the code under test, and each structural guard has a paired test showing it can fail.

- **SHACL vocabulary completeness:** `TestProfileClassifiesEveryTerm` reads the vendored `shacl.ttl` with the reader
  and checks that every `sh:` term has exactly one classification and is handled as classified: supported terms
  compile, and rejected terms fail with their IRI named. Paired: dropping one classification fails the test.
- **SHACL Core test suite:** the vendored cases for the supported components (minCount, maxCount, datatype, the four
  bounds, pattern, in, closed) and one case for each rejected family.
  - Supported cases go through a reviewed test adapter. It turns `sh:targetNode` focus nodes into entities of a
    synthetic class, and data properties into synthetic dotted predicates. The W3C expected results (focus node,
    path, component, value) are compared after mapping back.
  - Rejected-family cases (`and-001`, `or-001`, `class-001`, `qualifiedValueShape-001`, and others) must fail
    compilation, naming the construct, with no adapter at all.
- **SemEngine as oracle:**
  - The compiled pack passes `(*PackSpec).Validate`; mutating one predicate name to `altitude_m` fails it.
  - Each contract passes `ValidateShape`.
  - Pattern matching uses `MatchEntityIDPattern`.
  - The D9 agreement test registers the compiled DIEKM pack in its own test package, asserts registration state,
    serializes each sample through `export.Serialize`, parses the N-Triples with the reader, and compares terms with
    the validator's.
- **Acceptance cases from #1:**
  - omitted maximum, `maxCount 0`, `maxCount 1`, negative and contradictory bounds, `sh:in ()`, a typed `sh:in`
    member, and a rejected regex;
  - several shapes on one class, and conflicting mappings;
  - reconcile `{A,B}` to `{B,C}`, append to `{A,B,C}`, and conflicting group modes;
  - birth missing a required value; a partial update rebuilt with stored required values; append past the stored
    maximum; a stale-read case documenting the revision boundary.
- **Equivalence:** DIEKM's shapes in Turtle, JSON, and YAML compile to equal bundles.

### D16. Determinism, fuzzing, and benchmarks

Rapid properties (`pgregory.net/rapid v1.3.0`):

- Permuting shapes-graph statements, relabelling blank nodes, reordering mapping rows, and reordering `sh:in` members
  yield an equal bundle and an equal issue list.
- Permuting or duplicating candidate triples yields an equal report.
- Generated in-profile shape sets round-trip through Turtle, JSON, and YAML to equal bundles.
- Projection steps keep their base-pack order. A paired case shows that swapping two steps changes the bundle.

Each property runs at Rapid's default check count under `task verify`. The pull request records the command,
`-rapid.seed`, `-rapid.checks`, and the result. Rapid writes a failing case to `testdata/rapid/` for replay, and a
found failure is committed as a regression test.

Fuzz targets: `FuzzTurtleReader`, `FuzzParseTurtleShapes`, `FuzzDecodeShapesJSON`, `FuzzDecodeShapesYAML`,
`FuzzPatternSubset`.

- Seeds live in `testdata/fuzz/`: DIEKM, the W3C cases, a sample of each rejected construct, and inputs at the size
  and depth caps.
- The property: no panic; an error or a result; every error names a construct or a location.
- `go test` runs the seeds under `task verify`. A bounded run (`-fuzztime 60s` per target) is recorded on the pull
  request with the Go version. Any crasher is committed to the corpus.

Benchmarks: `BenchmarkCompile` (DIEKM, and synthetic sets of 10, 100, and 1,000 property shapes) and
`BenchmarkValidateCandidateState` (DIEKM, and 10, 100, and 1,000 triples), run with `-benchmem -count 10`.
`docs/shacl-compilation.md` records the workload, `go version`, GOOS/GOARCH, CPU, ns/op, B/op, and allocs/op, and
says that these are measurements, not guarantees. Benchmarks are not part of `task verify`.

### D17. Documentation

`docs/shacl-compilation.md` covers:

- what the compiler is for, and what it is not (Phase 3, Phase 4);
- the three input formats;
- the support matrix, one row per vocabulary term, with section reference and reason;
- the datatype table, the regex subset, and the mapping rules;
- the contract rules and the unwritable list;
- the validation API, its errors, and the revision boundary;
- the statement that the initial compiler rejects `sh:rule` and that rule translation is deferred to Phase 4;
- reproduction commands for Rapid and fuzz failures, and the benchmark table.

`AGENTS.md` ("What this is for") updates the description of the tree.

## Probe plan

Each probe runs before the code that depends on it. Its result goes in a `notes/` file of this change.

- **P1, reader caps:** generate inputs at and beyond 1 MiB and 64 levels, and confirm a bounded error with no stack
  growth.
- **P2, relative IRIs:** resolve the SHACL test-suite files against a supplied base, and compare triple counts with
  an independent parser run by hand. The command and version are recorded; the parser is not a dependency.
- **P3, `yaml.v3` v3.0.1:** confirm that `KnownFields(true)` rejects unknown keys in nested shape structs, and that
  the decoder exposes the node tag the `!!str` check needs.
- **P4, regex at go1.26.9:** confirm the RE2 behaviour D10 relies on: `\d` is ASCII-only, `.` excludes `\n` only,
  and which `\p{…}` category names RE2 accepts. Confirm the XSD Appendix F definitions of `.` and the multi-character
  escapes.
- **P5, export agreement:** run D9's table through `export.Serialize` at the pin before writing the validator.

## Declared costs

- `internal/turtle` becomes production code that parses author-supplied input. It keeps a Turtle subset, so some
  valid W3C Turtle shapes files must be rewritten.
- The profile is far narrower than SHACL Core, so many real shapes graphs are rejected. That is the intent.
- D9 restates SemEngine's export classification. Until SemEngine exports a pure classifier, a pin move that changes
  export is caught only by the agreement test (Q22).
- Vendored W3C files: `shacl.ttl` (about 53 KB) and roughly 20 SHACL test-suite cases of 1–3 KB each, under the W3C
  Software and Document License. The ledger gains a fixtures section and `TestAdmissionLedgerShape` checks it (Q21).
- Rapid becomes a direct test dependency (MPL-2.0). It is imported only by test files, so it is not built into
  consumer binaries.
- Rapid properties add run time to `task test:repeat`, which runs every test five times.
- Native validation covers the direct (unreified) form only. Shapes over reified paths are Phase 3.
- An append is not revision-fenced at the pin, so `sh:maxCount` under concurrent appends is not guaranteed (D13).
- `(*PackSpec).Validate` fills SemEngine's predicate-parse cache (ownership inventory).

## Open questions for the owner

Each has a proposed default. A task that depends on one carries a `Hold:` until it is ruled.

- **Q12 Turtle reader:** promote `internal/turtle`, with caps and a test-only base option (proposed). The other
  options are a third-party module (none has a tagged release) or a second, separate reader.
- **Q13 Mapping form:** a separate YAML/JSON mapping document (proposed), or annotations in a new SemVocab namespace
  inside the Turtle.
- **Q14 Base pack:** `Compile` takes the reviewed `PackSpec`, and the compiled pack fills an empty `data_type` from
  the shapes (proposed). Alternatives: require `data_type` to be declared already, or generate the whole pack from
  shapes and mapping.
- **Q15 Regex flags:** no `sh:flags` in the first profile (proposed), or support `i`, whose Unicode folding differs
  between XPath and RE2 for a few characters.
- **Q16 `sh:in`:** string-only (proposed), or typed members compared by datatype and value.
- **Q17 Entity references:** reject `sh:nodeKind`, so `entity_id` predicates take cardinality only (proposed), or
  accept `sh:nodeKind sh:IRI` on `entity_id` predicates.
- **Q18 Shapes with no writable contract:** no contract, with the entity type listed in `Unwritable()` (proposed), or
  fail compilation.
- **Q19 Constrained predicates outside the contract:** fail compilation (proposed), or allow predicates another
  writer owns.
- **Q20 Closed shapes:** require `rdf:type` in `sh:ignoredProperties` (proposed), or accept, with a documented
  Phase 3 disagreement.
- **Q21 W3C fixtures:** vendor `shacl.ttl` and the selected SHACL test-suite cases from `w3c/data-shapes` at
  `4cd469867a1cc6d2fff80e8a40723b26c695d67a`, recorded in a new ledger section `pins.fixtures` (proposed). The
  vocabulary file is `shacl/shacl.ttl`, sha256 `1a534caa9d3df5bbc5e5351002bda56ead9226fda0a617c165b218d8bd06b9af`.
  Its `sh:` term set equals the one served at `https://www.w3.org/ns/shacl.ttl` (probe, 2026-10-16). The alternative
  is hand-written expectations only.
- **Q22 Value-term mapping:** SemVocab defines it and checks it against SemEngine's serializer (proposed). The
  alternative is to file a SemEngine issue for a pure exported classifier first, and hold D9 on it.
- **Q23 DIEKM contract:** the mapping needs modes and birth for `intel.platform`. Proposed: group `measurement`
  (`intel.measurement.altitude-m`) as `reconcile`; group `role` (`intel.role.assigned`) as `reconcile`; no birth
  predicates.
- **Q24 Subclass targeting:** list every pack entity type in the target class's closure, checked against
  `superclasses` (proposed), or exact `class_iri` matches only, documented as narrower than SHACL.

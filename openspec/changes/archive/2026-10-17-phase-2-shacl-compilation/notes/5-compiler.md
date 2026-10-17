# Notes: task group 5 (profile and compiler)

Written 2026-10-16 with go1.26.9 darwin/arm64 and SemEngine `v0.1.0-alpha.2`
(`11da4721c2233a8e2b5b1d6236a97a44e06abbb6`). SemEngine citations are `file:line` in that module. The choices
below are the ones the design leaves open.

## Files and API

- `shacl/profile.go`: the profile table. `shacl/regex.go`: the XPath regex subset. `shacl/compile.go`: `Compile`,
  `Bundle`, `CompileError`, and `Issue`.
- `Issue` has `Shape`, `Path`, `Construct`, and `Message`. `Construct` is the IRI of the SHACL construct. For a
  mapping or pack problem it is the mapping field (`entity_types`, `predicates`, `contracts`,
  `closed_ignored_predicates`) or `pack`. `CompileError.Issues` is sorted by shape, path, construct, then message,
  with exact repeats removed.
- `Compile` runs every check before it fails, so independent problems are all reported. No bundle comes back with an
  error.
- The compiled constraints (`compiledNodeShape`, `compiledProperty`) are unexported. Group 6's validator reads them.
  They are keyed by entity type: a node shape that targets a class mapped to two entity types appears under both.

## 5.1 The profile table

- **Where the table lives.** D4 says the classification is published in `docs/shacl-compilation.md`. That document
  is task 8.1. The table the compiler uses is Go (`profile` in `shacl/profile.go`), and 8.1 renders the support
  matrix from it. Each entry carries the class, the section, and, for rejected and ignored terms, the reason.
- **Which IRIs count as terms.** The vendored `shacl.ttl` gives 221 subjects `rdfs:isDefinedBy sh:`. Of those, 38
  are parameter declarations such as `sh:AndConstraintComponent-and`, typed `sh:Parameter`. Each declares a
  component's parameter, and the term it declares (its `sh:path`) is defined on its own. Leaving them out gives the
  183 terms D4 counts. `vocabularyTerms` in `profile_test.go` applies that rule.
- **Sections.** Taken from the headings of the three documents, fetched 2026-10-16: the SHACL Recommendation
  (`https://www.w3.org/TR/shacl/`, 2017-07-20), SHACL Advanced Features (`/TR/shacl-af/`, Note 2017-06-08, marked
  `AF`), and SHACL JavaScript Extensions (`/TR/shacl-js/`, Note 2017-06-08, marked `JS`). Eight terms appear in the
  vocabulary and in none of the three documents' text: `sh:SPARQLExecutable`, its four `…Executable` subclasses,
  `sh:update`, `sh:Parameterizable`, and `sh:suggestedShapesGraph`. Their section is "vocabulary only".
- **Classes.** 21 supported, 4 ignored annotations (`sh:name`, `sh:description`, `sh:order`, `sh:defaultValue`),
  158 rejected. `rdfs:label` and `rdfs:comment` are ignored too (D4), but they are not `sh:` terms.
- **Constraint-component IRIs.** All 29 Core `…ConstraintComponent` IRIs are rejected as shapes input, the supported
  components' included: they name components in validation results and are never a shapes construct. The SPARQL,
  JavaScript, and expression components are rejected with their families.
- **Rule family.** `sh:rule` and the rest of SHACL Advanced Features §8 (`sh:Rule`, `sh:TripleRule`,
  `sh:SPARQLRule`, `sh:condition`, `sh:subject`, `sh:predicate`, `sh:object`, `sh:construct`) all carry the Phase 4
  deferral message, not only `sh:rule`.
- **Position.** A supported term is supported only where D4 accepts it. On another kind of shape (for example
  `sh:minCount` on a node shape) the parser records it and the message says "not accepted in this position or
  form".
- **How "handled as classified" is tested.** Supported terms each have a snippet that must compile. Ignored
  annotations must compile to a bundle equal to the one without them. Rejected property terms are placed on a node
  shape and on a property shape, and the issue must name the term and its family reason. Rejected classes and
  individuals (capitalised local names) are used as a shape's `rdf:type`, and the error must name the IRI. A target
  predicate on a blank property shape makes it a blank node shape, which parsing refuses, naming the target
  (ruling Q27).

## 5.2 Rejected families

- A parser record whose construct is a rejected `sh:` term gets the family reason and the section. An `sh:` term
  outside the vocabulary gets "not a term of the W3C SHACL vocabulary (2017-07-20)". Any other construct (for
  example `owl:imports`, `rdfs:Class`, or a data triple) gets "outside the profile". The record's detail follows.
- A rejected class used as a shape's type is recorded under `rdf:type`, so the issue's construct is `rdf:type` and
  the class IRI is in the message. That is the parser's existing record (group 4); it was not changed.
- **New check: a node shape with no `sh:targetClass`.** Such a shape never applies, which would drop its
  constraints without a word. It fails, construct `sh:targetClass`. The design does not state this. Fail-closed
  (`AGENTS.md`, domain 1) decides it.
- The W3C rejected-family cases compile against `PackSpec{Name: "w3c"}` and an empty mapping, with no adapter. Each
  must fail with an issue naming its construct and a shape, and with no bundle.

## 5.3 Cardinality, datatypes, bounds, and `sh:in`

- The parser reads counts as `int64`, so a count above `math.MaxInt32` reaches Compile and is rejected there.
  `math.MaxInt32` itself is accepted.
- **Integer bounds** follow XSD's integer lexical space, `[+-]?[0-9]+` (XML Schema Part 2 §3.3.13.1), and are held
  as `*big.Int`.
- **Double bounds** follow XSD 1.0's double lexical space (§3.2.5.1): `INF`, `-INF`, and `NaN` are accepted, `+INF`
  is not (only XSD 1.1 adds it). The mantissa accepts `.5` and `5.`, as XSD 1.1's decimal grammar states
  explicitly. A mantissa whose value overflows (`1e400`) is accepted as infinity, which is `strconv.ParseFloat`'s
  rounding with `ErrRange`.
- **"A lower bound above an upper bound."** Each of `sh:minInclusive` and `sh:minExclusive` is compared with each of
  `sh:maxInclusive` and `sh:maxExclusive`. Only a lower bound strictly above an upper bound fails. Equal exclusive
  bounds (`minExclusive 5`, `maxExclusive 5`) are accepted though no value can pass, and so is a `NaN` bound, which
  compares with nothing. Validation reports the violation.
- **`sh:in`** needs `sh:datatype xsd:string` on the same property shape: a missing datatype fails as well as another
  one.
- **Entity references (ruling Q17).** On a path mapped to a pack predicate whose `data_type` is `entity_id`, every
  constraint other than `sh:minCount` and `sh:maxCount` fails: `sh:datatype`, the four bounds, `sh:pattern`, and
  `sh:in`, each named with its shape and path. The check follows the mapped pack predicate, not `sh:datatype`, and
  it replaces the other checks on those constructs, so each gets exactly one issue (the datatype-conflict check
  skips such paths). `sh:severity` and `sh:message` stay accepted: they constrain no value. The first version
  accepted `sh:pattern` there whenever `sh:datatype` was absent (issue #13, from Codex's review of `feae567`).
- **`sh:pattern` on other predicates without `sh:datatype`** is accepted: outside Q17, D4 and D10 place no datatype
  requirement on it. What string a non-string value is matched against is group 6's question.
- **The same class elsewhere.** Swept for other datatype-specific rules checked only when `sh:datatype` is present.
  Bounds (D8) require `sh:datatype` `xsd:integer` or `xsd:double` and fail without it; `sh:in` (D11) requires
  `sh:datatype xsd:string` and fails without it. A shape datatype that disagrees with the pack `data_type` fails in
  the fill (D5). Q17 was the only gap.

## 5.4 The regex subset

The grammar is XSD 1.0 Second Edition Appendix F with F&O 3.1 §5.6.1's changes, fetched 2026-10-16
(`https://www.w3.org/TR/xpath-functions-31/`). Beyond D10:

- **Unescaped `{`, `}`, and `]`** outside their syntax are rejected. F&O §5.6.1.1 corrects XSD's `Char` production to
  `[^.\?*+{}()|^$#x5B#x5D]`, so they are not normal characters.
- **Repetition counts above 1,000** are rejected, naming the count. RE2 refuses them; XSD sets no limit. Group
  nesting deeper than 1,000 is rejected too, for the same reason.
- **A quantifier on `^` or `$`** is rejected. F&O makes them atoms, so `^*` is valid XPath, but RE2 refuses it.
- **A `-` inside a class** must be first, last, or between a range's ends. XSD 1.0 Appendix F says "the - character
  is a valid character range only at the beginning or end of a positive character group". A `-` before `[` is
  class subtraction, which D10 rejects by name.
- **A range end** must be one character, and a range whose end is below its start is rejected.
- **Translation.** `.` becomes `[^\n\r]`. A literal outside a class is quoted with `regexp.QuoteMeta`. Inside a class
  every character is written as `\x{…}`, so no character is read as RE2 class syntax. Category escapes are copied as
  `\p{Name}` once the name is checked against XSD's 36.
- **Anchors.** Go's `$` without the `m` flag matches only at the end of the text, which is F&O's string mode
  ("`$` matches the end of the entire string", §5.6.1.1). The test checks that `a$` does not match `"bra\n"`.

## 5.5 Closed shapes

- The allowed set of a closed shape is the dotted predicates its own paths map to, the `sh:ignoredProperties` IRIs
  that have a `predicates` row, and every `closed_ignored_predicates` entry. An ignored IRI with no row (`rdf:type`,
  or any IRI the pack does not map) allows nothing in native state, because no native predicate stands for it.
- `closed_ignored_predicates` entries must pass SemEngine's `IsValidPredicate`. They need not be in the pack: they
  name native predicates a closed shape tolerates, which another pack may declare.
- `sh:ignoredProperties` on a shape that is not closed is accepted and has no effect, as in SHACL §4.8.1.
- The `rdf:type` rule's message cites the W3C case it follows. The test also compiles the vendored
  `core/node/closed-001` with no adapter and requires the rule to reject it. Task 6.3's adapter cannot run that case
  as a passing case for this reason; that is design D12's declared consequence.

## 5.6 Mapping rules

- **Duplicates.** A `class_iri` or `path_iri` with two rows fails, whether or not the rows agree ("conflicting
  mappings"). A domain type listed twice for one class fails.
- **Unused rows.** A `predicates` or `entity_types` row that no shape uses is accepted. It drops no constraint, and
  an unused `predicates` row can still serve `sh:ignoredProperties`. Rows are still checked against the pack:
  every `predicates` row must name a pack predicate whose `iri` equals the path, and every `entity_types` row must
  list exactly its class's closure.
- **Mapped entity types** are every domain type any `entity_types` row lists, targeted or not. They are the types
  `Unwritable()` draws from.
- **Datatype fill.** Only shape datatypes in the D8 table take part; another datatype is already an issue. A
  conflict between two shapes is reported on the later shape in document order (sorted by node shape IRI, then
  path), naming the first. A conflict with the base pack is reported on every disagreeing shape.
- **The compiled pack** is a deep copy of the base pack (prefixes, predicates with their projection steps, entity
  types with their superclasses), so neither the caller's pack nor the bundle can be changed through the other.
  `Pack()` returns a further copy.

## 5.7 Contracts

- **Groups.** An unnamed group takes the category of its predicates when they all share one, and fails otherwise.
  Two declarations of one group name with one mode are merged; with different modes they fail. A predicate listed
  twice in one group counts once. A predicate in two different groups fails, as does one in birth and in a group.
  These are the conditions `ValidateShape` would reject (`contract.go:96-127`), reported with their names first.
- **Bindings.** A contract binding for a domain type no `entity_types` row lists fails, because its contract would
  otherwise be dropped. Two bindings for one domain type fail. Two contracts with the same name fail: SemEngine's
  `ValidateContracts` requires unique names (`contract.go:144-149`), though the compiler does not call it,
  because it calls `Validate`, which reads the registry (`:151`, `:130-133`).
- **Unwritable.** A mapped type with no binding and no constrained predicate, and a type whose binding declares no
  predicate, are both unwritable. A type with constrained predicates and no binding fails by ruling Q19.
- **Declared predicates.** Each contract predicate must be declared in the compiled pack, which replaces
  `Contract.Validate`'s registry lookup (`contract.go:130-133`).

## 5.8 DIEKM

- `packs/diekm/diekm.mapping.yaml` is embedded and returned by `diekm.Mapping()`, a copy of the bytes, like
  `Shapes()`. It returns bytes rather than a `shacl.Mapping` because `shacl`'s own tests import `packs/diekm`, so
  `packs/diekm` importing `shacl` would make an import cycle in those tests. The caller is
  `TestDIEKMBundle` and `TestSHACLCompilerOutputsValidPackAndContracts`; a consumer compiles DIEKM with `Spec()`,
  `Shapes()`, and `Mapping()`.
- The contract has no explicit name, so it is named `intel.platform` (D7). The groups are named explicitly though
  their defaults would be the same, so the file reads as ruling Q23 does.
- The ledger's `diekm` entry lists the new file and ruling Q23.

## 5.9 Registry state

- The tests are in the external test package `shacl_test` (`shacl/diekm_test.go`), so they use only the exported
  API. They share the shacl test binary, whose other tests never touch the registries.
- **Observed state.** Every registered predicate and entity type with its metadata, and the Turtle output for a probe
  triple in the DIEKM namespace, which shows whether the `diekm` prefix compacts. This is the observation
  `packs/registration_test.go` already uses.
- **The failed compile** drops the first `predicates` row, so the altitude path is unmapped.
- **The unrelated pack** in `TestCompileIgnoresRegistrations` has names unique to each call, because `test:repeat`
  runs the test five times in one process. It reuses DIEKM's class IRI and altitude path IRI, so a compiler that
  consulted `LookupEntityTypesByClassIRI` or `LookupByIRI` would see it. The test asserts that the predicate, the
  entity type, and the prefix took effect before the second compile.

## Evidence

- **5.1:** `TestProfileClassifiesEveryTerm` and `TestProfileClassifiesEveryTermSensitivity`. Fail-before: with
  `profile` empty, the test fails listing every term as unclassified. In-tree sensitivity: removing `sh:rule` from a
  copy of the table names it, and an extra entry names that.
- **5.2:** `TestW3CRejectedFamiliesFailCompile`, `TestRejectedConstructsReportedTogether`, and
  `TestUnknownSHACLTermRejected`. Seeded: with Compile not reporting the parser's records, all three fail and the
  profile test fails on every rejected class.
- **5.3:** `TestCardinalityCompiles`, `TestDatatypeTable`, `TestNumericBoundsExact`, `TestXSDLexicalForms`, and
  `TestEnumeration`. `TestNumericBoundsExact` first checks its premise, that float64 cannot tell 2^53 from 2^53+1.
  Seeded, each reverted:
  - integer bounds compared through float64: `TestNumericBoundsExact` fails;
  - `xsd:double` mapped to `int`: `TestDatatypeTable` fails;
  - the `minCount > maxCount` check removed: `TestCardinalityCompiles` fails;
  - the `sh:in` datatype check removed: `TestEnumeration` fails;
  - the bound-datatype check removed: `TestNumericBoundsExact` fails.
- **5.4:** `TestPatternSubsetAccepts`, `TestPatternSubsetRejects`, `TestPatternCompileIssues`, and
  `TestPatternAgreesWithW3C`. The last reads `property/pattern-001`'s pattern, data values, and expected
  `sh:PatternConstraintComponent` results from the vendored file, and checks the five flag-less `fn:matches`
  examples of F&O 3.1 §5.6.3. Seeded, each reverted:
  - `.` emitted as RE2's `.`: the accept and compile tests fail on `\r`;
  - the `\s` entry removed from the named escapes: the reject test fails, because the reason no longer says how
    XSD differs;
  - the category-name check off: `\p{Cs}`, `\p{Greek}`, and `\P{Latin}` are accepted and the test fails;
  - the class-subtraction check off: the error names `-`, not `-[`, and the test fails;
  - the repetition limit off: RE2's own error does not name the count, and the test fails.
- **5.5:** `TestClosedShapes`. Seeded, each reverted: the `rdf:type` check off, and the allowed set without
  `closed_ignored_predicates`; the test fails on each.
- **5.6:** `TestMappingRules` (explicit bindings, `iri` agreement, conflicting mappings, closure-checked targeting,
  datatype fill and conflict, several shapes on one class, the pack copy). Seeded, each reverted: the "does not list"
  closure check off, the `iri` check off, and the two-shape datatype conflict off; the test fails on each.
- **5.7:** `TestContracts`. Seeded, each reverted: the Q19 check off, the mode-conflict check off, contracts left
  unsorted, and `Unwritable()` always empty; the test fails on each.
- **5.8:** `TestDIEKMBundle`, which also checks that the DIEKM shapes in Turtle, JSON, and YAML compile to equal
  bundles. Seeded, each reverted: the pattern built as `*.*.<domain>.<type>.*.*`, and the role group's mode changed to
  `append` in the mapping file; the test fails on each.
- **5.9:** `TestSHACLCompilerOutputsValidPackAndContracts` and `TestCompileIgnoresRegistrations`. Seeded, each
  reverted: Compile calling `vocabulary.RegisterEntityType("seed.leak")` fails the first ("the entity-type registry
  changed"); Compile adding `LookupEntityTypesByClassIRI` results to a class's closure fails the second.
- **Issue #13:** `TestEntityIDCardinalityOnly` compiles, through `ParseTurtle` and `Compile`, a property on an
  `entity_id` predicate with each of `sh:pattern`, `sh:in`, the four bounds, and `sh:datatype`. Each must fail with
  exactly one issue naming the construct, shape, and path; cardinality with severity and message must compile.
  Fail-before at `feae567`: the pattern case returned a bundle, and the other six had no Q17 issue (bounds and
  `sh:in` failed for a missing datatype, `sh:datatype` as a pack conflict).

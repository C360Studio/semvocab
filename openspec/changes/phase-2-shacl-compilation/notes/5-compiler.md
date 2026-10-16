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
- **`sh:pattern` without `sh:datatype`** is accepted: D4 places no datatype requirement on it. What string a
  non-string value is matched against is group 6's question.

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

# Notes: task group 4 (shape model and inputs)

Written 2026-10-16 with go1.26.9 darwin/arm64, SemEngine `v0.1.0-alpha.2`, and `gopkg.in/yaml.v3 v3.0.1`. Rulings
Q25–Q28 (`design.md`) settle the questions raised during task 3.1. The choices below are the ones the design leaves
open.

## Files and API

- `shacl/model.go` holds the package comment, `ShapeSet`, `NodeShape`, `PropertyShape`, `Literal`, and `Mapping` with
  its row types. `shacl/parser.go` has `ParseTurtle` and the triple-to-model builder. `shacl/decode.go` has the JSON
  and YAML decoders and the canonical form.
- Every decoder takes an `io.Reader`, as SemEngine's `LoadPackYAML` does.
- `GroupBinding.Mode` is SemEngine's `contract.WriteMode`, used directly. Decoding does not check it; Compile does
  (D7).

## The model

- **Bounds.** A bound is a `*Literal` (lexical form plus datatype). Turtle keeps the literal's own datatype. The JSON
  and YAML forms hold a bound as a string, and decoding gives it the property's `datatype`. So a Turtle bound whose
  datatype differs from `sh:datatype` stays visible to Compile (D8), and a JSON or YAML bound with no `datatype` has
  an empty datatype, which Compile rejects.
- **Optional fields.** `Pattern` is a `*string`, so an empty pattern is not mistaken for none. `In` is a `*[]string`,
  as D11 states.
- **Canonical form.** Node shapes are sorted by IRI. Properties are sorted by path, then ID, then full content, and
  exact duplicates are merged. IRI and string lists (targets, ignored properties, messages, `in` members) are sorted
  with repeats removed, because they are sets in RDF. An empty list is nil, except an empty `In`, which means
  `sh:in ()`. All three inputs pass through the same `normalize`, so equal shapes decode to `reflect.DeepEqual` sets.

## Turtle (ruling Q26 and Q27)

- **Size.** `ParseTurtle` reads at most 1 MiB + 1 byte. Over the limit, it fails with
  `input exceeds the 1048576-byte limit at byte offset 1048576` (Q25).
- **Order of work.** The reader parses with no base, exact repeated triples are removed (D3), then the model is built.
- **Node shapes.** A node shape is a subject typed `sh:NodeShape` or carrying any target predicate (`sh:targetClass`,
  `sh:targetNode`, `sh:targetSubjectsOf`, `sh:targetObjectsOf`, `sh:target`). A blank one fails parsing, naming its
  predicates (Q27).
- **Property shapes.** A property shape is the object of `sh:property` from a node shape. Both are built field by
  field. A blank property shape referenced by two node shapes is recorded.
- **Records.** Everything the model cannot hold becomes a record for Compile (`rejection`: construct, shape, path,
  detail). That covers an out-of-profile predicate, a wrongly typed value, several values for a single-valued
  parameter, a malformed list, an `rdf:type` other than the shape's own, a non-string `sh:in` member, and a path
  other than one IRI.
  - The shape is the property shape's IRI, or its node shape's IRI when the property shape is blank.
  - A path that is not one IRI is recorded by its construct: `sh:inversePath`, `sh:alternativePath`, and so on, or
    `sh:path` with "sequence path" for a list.
  - A shape with a target that also carries `sh:path` (the W3C path cases use such shapes) records that `sh:path`
    and the path construct.
- **Ownership.** When a record's object is a blank node, the blank nodes reachable from it belong to that record, and
  none of their triples is recorded again. So `sh:and ( [ sh:property … ] )` is one `sh:and` record (Q27).
- **Stray triples.** Every other triple is recorded under its subject's IRI: one whose subject is not a shape, a
  list cell the model used, or a blank node a record owns. Data and manifest triples in the W3C files fall here. A
  blank subject is recorded with an empty shape. Unreferenced blank subjects go first, then blank cycles, so records
  do not depend on triple order.
- **No labels.** Records never contain a blank-node label. Values are shown as `<iri>`, `"lex"^^<dt>`, `"lex"@lang`,
  or "a blank node". Records are sorted and deduplicated.
- **Annotations.** `sh:name`, `sh:description`, `sh:order`, `sh:defaultValue`, `rdfs:label`, and `rdfs:comment` are
  accepted and dropped (D4). Their blank objects are owned, not recorded.
- **Classification is Compile's.** Each record names the construct. Task 5 classifies it and writes the message, for
  example `sh:rule`'s Phase 4 deferral.

## JSON and YAML

- **Wire types.** Unexported wire structs (`wireShapeSet` and the rest) are built into the model, so the exported
  model carries no struct tags for a form it does not use.
- **JSON.** `DisallowUnknownFields`. A second `Decode` must return `io.EOF`, otherwise "trailing data". A number where
  a string belongs (a bound, an `in` member, a name) fails in `encoding/json`, naming the field.
- **YAML.** First a strict decode with `KnownFields(true)`, so an unknown field is named as such. A second `Decode`
  must return `io.EOF`, otherwise "trailing document". Then the bytes are parsed into a `yaml.Node`, and every scalar
  must be tagged `!!str` (Q28). The exceptions are the values of `min_count`, `max_count`, and `closed`, whose types
  the struct decode enforces. Mapping keys must also be strings. Aliases are followed. The error names the line and
  key, for example `line 5: min_inclusive must be a string (YAML tag !!int); quote it`. A duplicate key fails in
  yaml.v3.
- **Shape checks.** Node shape IDs, targets, ignored properties, paths, and any non-empty property ID, datatype, or
  severity must be absolute IRIs: an RFC 3986 scheme, and none of the characters Turtle's IRIREF excludes. An ID
  starting `_:` is refused as a blank-node node shape. A property shape needs a path. A node shape listed twice is
  refused.
- **Mapping.** Decoded strictly, with no rule checks: D6 and D7 are task 5.6 and task 5.7.

## Evidence

- **4.1:**
  - `TestEveryFieldInThreeFormats` decodes one shape that uses every model field, in Turtle, JSON, and YAML, to the
    same set written out by hand.
  - `TestParseTurtleRefuses` covers blank-node node shapes (typed, `sh:targetClass`, `sh:targetNode`), a relative
    IRI (with line and column), a syntax error, and 65 levels of nesting.
  - `TestParseTurtleRemovesRepeats`: a repeated `sh:minCount 1` is one constraint, while `1` and `2` are recorded.
  - `TestParseTurtleRecordsRejections` covers `owl:imports`, an implicit class target, `sh:rule`, `sh:in ( "a" 1 )`
    (naming the member), a sequence path, an inverse path, a language-tagged pattern, `sh:minCount 1.5`, and a
    missing path. It also checks that no record names a blank label or a triple under `sh:rule`.
  - `TestW3CRejectedFamiliesRecorded` loads all 28 vendored rejected-family cases through the test-suite loader
    (`turtle.WithBase`, then the same builder `ParseTurtle` uses). Each one records its family's construct. For
    `property/and-001` it checks `sh:and` with its shape and path, and `sh:targetNode`, and that nothing inside
    `sh:and` is recorded on its own.
  - `TestDecodeShapesRefuses` covers unknown fields at all three levels, trailing values and documents, numeric
    bounds, unquoted and `!!int` bounds, `in: [1]`, `pattern: 5`, `severity: ~`, duplicate keys, blank-node IDs,
    relative IRIs, a missing path, and a duplicate node shape.
- **3.2 with 4.1:** `TestParseTurtleSizeLimit` accepts exactly 1 MiB and rejects 1 MiB + 1, naming the offset.
- **4.2:** `TestDecodeMapping` decodes JSON and YAML to the mapping written out by hand. `TestDecodeMappingRefuses`
  covers unknown fields, trailing values and documents, and non-string scalars.
- **4.3:** `TestDIEKMShapesInThreeFormats` decodes the pack's own `diekm.shacl.ttl` (through `diekm.Shapes()`) and
  the hand-written `shacl/testdata/diekm.shapes.json` and `.yaml` to equal sets, each equal to the set written out by
  hand.
- **Shown able to fail.** Each seeded change was reverted:
  - no repeat removal: `TestParseTurtleRemovesRepeats` fails;
  - no `!!str` check: the YAML scalar cases fail;
  - size limit off by one: `TestParseTurtleSizeLimit` fails;
  - no ownership: `TestParseTurtleRecordsRejections` and `TestW3CRejectedFamiliesRecorded` fail;
  - no blank-node refusal: the three refusal cases fail;
  - target classes unsorted: `TestEveryFieldInThreeFormats` fails.

## Fixes from the review of 3833987 (PR #9, Codex, CHANGES REQUESTED)

- **Finding 1, a named property shape reused by two node shapes.** `list` refused any cell already marked consumed,
  so the second node shape to read a shared property shape's `sh:in` lost it and gained a "not a well-formed RDF list"
  record. Which node shape lost it depended on statement order.
  - Fix: cycle detection is now local to each walk (`visited`). Marking cells consumed only accounts for their
    triples, so any number of readers get the same list.
  - Sweep: the one other place that reads `consumed` as a refusal is the shared blank property shape check, which is
    deliberate (D3: a blank property shape's identity is its node shape plus its path). Every other use only accounts
    for triples.
  - `TestNamedPropertyShapeSharedByTwoNodeShapes` requires both node shapes to keep `("a" "b")` in either statement
    order with no record. It also requires a cyclic list, a cell with no `rdf:rest`, and a cell with an extra arc to
    still be recorded, with `In` nil on both node shapes.
- **Finding 2, JSON null.** `encoding/json` decodes `null` in a `[]string` as `""`, and leaves a string or pointer
  field unset.
  - Fix: before the strict decode, `decodeJSON` walks the first value's token stream (`json.Decoder.Token`). It
    refuses the first null, naming its JSON path (for example `$.node_shapes[0].properties[0].in[0] is null`).
  - Choice: null is refused everywhere in both JSON forms, including optional fields (`"in": null`,
    `"min_inclusive": null`), where it could be read as "absent". No field of either JSON form takes null. To mean
    "absent", leave the field out.
  - The YAML form is not the same. Q28 requires `!!str` on every string position, so a null there is refused. The
    values of `min_count`, `max_count`, and `closed` are exempt from that check, and yaml.v3 decodes a null there to
    the field's zero value: `min_count: null` is 0, `max_count: null` is nil (unbounded), and `closed: null` is false.
    Each matches leaving the field out. This fix does not change YAML.
  - `TestDecodeJSONRefusesNull` covers null in `in` members, messages, targets, ignored properties, the node ID, the
    path, the datatype, a bound, `in` itself, and a node shape entry. In mappings it covers domain-type members, a
    predicate name, birth members, a group mode, group predicates, and closed-ignored predicates. It also checks that
    `""` decodes in `in`, `pattern`, and `messages`.
- **Re-review of ddae5a7 (issue #11), duplicate JSON keys.** The first version of the null check decoded into a
  generic map, which keeps only the last of a repeated key. `encoding/json` merges repeated members into the slice
  elements it has already decoded (`encoding/json/decode.go:50-52` and `:550-564` at go1.26.9). So
  `"properties":[{"path":"urn:p","in":[null]}],"properties":[{"path":"urn:p"}]` hid the null and still gave
  `In == []string{""}`.
  - Fix: the token-stream walk now refuses a repeated object key, naming the key and its JSON path, in the same pass
    as the null check. JSON now matches YAML, where yaml.v3 refuses duplicate keys (probe P3).
  - `TestDecodeJSONRefusesDuplicateKeys` covers that input, a repeated property key, and a repeated top-level key in
    shapes, and a repeated predicate key and a repeated top-level key in mappings.

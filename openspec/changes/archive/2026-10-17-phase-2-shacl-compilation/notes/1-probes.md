# Notes: task 1.2 (probes P1–P5)

Run on 2026-10-16 at claim head `0840704`, with go1.26.9 darwin/arm64, SemEngine `v0.1.0-alpha.2`, and
`gopkg.in/yaml.v3 v3.0.1`. Each probe ran from a scratch Go module outside the repository that required those two
modules. The scratch programs are local only and are not committed. Each section names its input exactly, so the
probe can be rebuilt from this note. Results that changed `design.md` are marked **Design change**.

## P1: reader limits

**Input.** `internal/turtle/turtle.go` at `0840704`, copied into the scratch module. Nested blank-node property lists
(`ex:s ex:p [ ex:p [ ex:p … ex:o ] ] ] .`) and nested collections (`( ( … ex:o ) )`) at several depths. Flat documents
of `ex:sN ex:p "value N" .` lines at 1 MiB and 16 MiB. The program measured `runtime.MemStats.StackInuse` and
`TotalAlloc` around each `turtle.Parse`.

| Input | Bytes | Triples | Time | Stack in use | Allocated |
| --- | --- | --- | --- | --- | --- |
| blank nodes, depth 64 | 629 | 65 | 0.25 ms | 512 KiB | 29 KiB |
| blank nodes, depth 1,000 | 9,053 | 1,001 | 1.5 ms | 2.4 MiB | 639 KiB |
| blank nodes, depth 100,000 | 900,053 | 100,001 | 86 ms | 128 MiB | 102 MiB |
| collections, depth 100,000 | 400,053 | 200,001 | 73 ms | 65 MiB | 197 MiB |
| densest nesting in 1 MiB (`[x:p`, depth 174,755) | 1,048,569 | 174,756 | 139 ms | 256 MiB | 159 MiB |
| blank nodes, depth 1,000,000 | about 9 MB | none | none | over 1 GB | fatal |
| flat, 1 MiB | 1,048,587 | 34,541 | 12 ms | 2.5 MiB | 36 MiB |
| flat, 16 MiB | 16,777,238 | 515,134 | 142 ms | 0.6 MiB | 563 MiB |

At depth 1,000,000 the process died with `runtime: goroutine stack exceeds 1000000000-byte limit` and
`fatal error: stack overflow`. That is a process kill, not an error a caller can recover from.

**Finding.** Stack use grows linearly with nesting, at about 1.3 KiB per blank-node level. The 1 MiB size limit alone
still allows 256 MiB of stack, so the 64-level depth limit D2 adds is needed as well. Memory allocated is about 35
times the input size on flat input, which the size limit bounds. Proving the limits once they exist (error with
position, nothing growing past the limit) is task 3.2's sensitivity tests; this probe measured the reader without
them. **Design change:** D2 records these measurements.

## P2: relative IRIs

**Input.** All 121 `.ttl` files under `data-shapes-test-suite/tests/core/` in `w3c/data-shapes` at
`4cd469867a1cc6d2fff80e8a40723b26c695d67a` (tarball from `codeload.github.com`). The independent parser was rdflib
7.1.4 (Python 3.14, installed in a throwaway venv, not a dependency). Each file was parsed with
`Graph().parse(f, format="turtle", publicID="http://semvocab.test/core/<relative path>")` and the result counted with
`len(graph)`. The current reader ran over the same files.

**Results.**

- rdflib parsed all 121 files with a base. It logged literal-conversion warnings for ill-typed literals the suite uses
  on purpose (`"aldi"^^xsd:integer`, a weird boolean `none`); these do not change triple counts.
- The current reader rejected 106 of the 121 files, every one at its first relative IRI, the `<>` document IRI
  (`unsupported relative IRI <>`). No other error occurred.
- On the 15 files the reader parsed, its counts equal rdflib's on 14. The exception is
  `complex/shacl-shacl-data-shapes.ttl`: 417 from the reader, 415 from rdflib. The file states two triples twice:
  `shsh:ShapeShape sh:targetSubjectsOf sh:qualifiedValueShape` and
  `… sh:qualifiedValueShapesDisjoint`. The reader returns every triple it reads, while an rdflib `Graph` is a set.
  After removing exact repeats the reader has 415 unique triples, which agrees with rdflib.

**Finding.** The reader needs a base option to read the suite at all (D2). The rdflib counts per file are the expected
values for task 3.1's base option, compared after removing exact repeats. **Design change:** D3 now says the Turtle
input path removes exact repeats before building the model, because RDF graphs are sets.

## P3: `yaml.v3` v3.0.1

**Input.** A decoder with `KnownFields(true)` over three levels of structs (`Doc{Shapes []Shape}`,
`Shape{ID, Properties []Prop}`, `Prop{Path, MinCount *int, MinIncl *string}`), followed by a second `Decode` call to
detect a trailing document. Then `yaml.Unmarshal` into a `yaml.Node` for scalar forms, reading `Tag`, `ShortTag()`,
and `Style`.

| Document | First `Decode` | Second `Decode` |
| --- | --- | --- |
| valid | nil | `EOF` |
| unknown key at the top level | `line 2: field bogus not found in type main.Doc` | `EOF` |
| unknown key in a shape | `line 3: field bogus not found in type main.Shape` | `EOF` |
| unknown key in a nested property | `line 5: field bogus not found in type main.Prop` | `EOF` |
| unquoted `min_inclusive: 5` into `*string` | **nil (accepted, value "5")** | `EOF` |
| `shapes: []` then `---` then a second document | nil | nil (a second document) |
| duplicate key | `mapping key "shapes" already defined at line 1` | `EOF` |

| Scalar | `Tag` | `Style` |
| --- | --- | --- |
| `5` | `!!int` | 0 |
| `'5'` | `!!str` | 4 (single-quoted) |
| `"5"` | `!!str` | 2 (double-quoted) |
| `!!str 5` | `!!str` | 1 (tagged) |
| `5.0`, `1e3` | `!!float` | 0 |
| `0x10` | `!!int` | 0 |
| `true` | `!!bool` | 0 |
| `~` | `!!null` | 0 |

**Finding.** `KnownFields(true)` rejects unknown keys at every nesting level, with the line. A trailing document is
detected only by a second `Decode` that must return `io.EOF`. Decoding into a `string` field silently accepts an
unquoted number, so D3's rule that rejects any bound whose node tag is not `!!str` is what actually enforces
string-only bounds: a struct decode alone would not. The node exposes the tag the check needs. D3 already says
this; no design change.

## P4: regex at go1.26.9, and XSD Appendix F

**RE2 input.** `regexp.Compile` and `MatchString` on single-character subjects. `unicode.Version` is `15.0.0`.

| Pattern | Subject | RE2 result | XSD meaning |
| --- | --- | --- | --- |
| `^\d$` | `٥` (U+0665, Nd) | false | true: `\d` is `\p{Nd}` |
| `^\w$` | `é` | false | true: `\w` excludes only P, Z, C |
| `^\s$` | `\f` | true | false: `\s` is `[#x20\t\n\r]` |
| `^\s$` | U+00A0, U+2028 | false | false |
| `^.$` | `\n` | false | false |
| `^.$` | `\r` | true | false: `.` is `[^\n\r]` |
| `^[^\n\r]$` | `\r` / `a` | false / true | the D10 translation of `.` |
| `^[a-z-[aeiou]]$` | `e` / `e]` | false / true | `e` is excluded; RE2 read it as a different class |

- RE2 compiles all 36 XSD general categories, and also `\p{Cs}` and script names such as `\p{Greek}`. `\p{Cn}`
  matches the unassigned U+0378.
- RE2 rejects `\p{IsBasicLatin}`, `\i`, `\c`, and `(a)\1`.
- RE2 accepts `(?i)`, `\x41`, `\A`, `\z`, and `\b`, none of which XSD defines.

**XSD input.** The Appendix F text of XML Schema Part 2 Second Edition (`https://www.w3.org/TR/xmlschema-2/`) and of
XSD 1.1 Part 2 (`https://www.w3.org/TR/xmlschema11-2/`), fetched 2026-10-16. XPath and XQuery Functions and Operators
3.1 §5.6.1 says its regex syntax is XSD 1.0 Second Edition's plus the listed additions, and that XSD 1.1 made no
substantive change. Both editions define `.` as `[^\n\r]`, `\s` as `[#x20\t\n\r]`, `\d` as `\p{Nd}`, `\w` as
`[#x0000-#x10FFFF]-[\p{P}\p{Z}\p{C}]`, and `\i`/`\c` from the XML name rules. The General Category table lists 36
values: L Lu Ll Lt Lm Lo, M Mn Mc Me, N Nd Nl No, P Pc Pd Ps Pe Pi Pf Po, Z Zs Zl Zp, S Sm Sc Sk So, C Cc Cf Co Cn.
It has no `Cs`.

**Finding.** The `.`, `\d`, and `\w` claims in D10 hold. The `\s` claim was wrong: XSD's `\s` is narrower than RE2's,
not Unicode-wide, so they differ on `\f`. Rejecting it still stands. An allowlist must decide category names,
because RE2 accepts names XSD does not list, and class subtraction must be rejected by name because RE2 compiles it
silently with a different meaning. **Design change:** D10 corrects the `\s` reason, checks category names against
XSD's 36, rejects `\p{Cs}`, script names, and the RE2-only escapes by name, and records Unicode 15.0.

## P5: export agreement

**Input.** One predicate per data type, each registered with
`vocabulary.Register("probe.p5.<type>", WithDataType(<type>), WithIRI("http://example.org/p5#<type>"))`. The
`entity_id` predicate is named `probe.p5.entity-id`, because `_` fails SemEngine's segment rule. Each row was one
triple with subject `acme.edge.sensor.intel.platform.p1`, serialized with
`export.SerializeToString(…, export.NTriples)`. Entity IRIs print under SemEngine's default base
`https://semengine.semanticstream.ing/entities/`.

| Declared | Go value | Hint | Export writes |
| --- | --- | --- | --- |
| int | `int(42)`, `int64(-7)`, `uint8(3)` | | `"42"^^xsd:integer` and so on |
| int | `float64(5)`, `float32(5)`, `float64(2^53)` | | `xsd:integer` |
| int | `float64(2^54)` | | `"1.8014398509481984e+16"^^xsd:double` |
| int | `float64(1.5)` | | `"1.5"^^xsd:double` |
| int | `"42"` (string) | | `"42"` (plain string) |
| float | `1.5` / `int(2)` | | `"1.5"^^xsd:double` / `"2.0"^^xsd:double` |
| float | NaN, +Inf | | nothing: triple dropped, no error |
| bool | `true` / `"true"` | | `"true"^^xsd:boolean` / `"true"` |
| datetime | `time.Time`, `"2026-10-16T12:00:00Z"` | | `"2026-10-16T12:00:00Z"^^xsd:dateTime` |
| datetime | `"yesterday"` | | `"yesterday"` |
| json | `{"a":1}` / `"hovering"` | | `^^rdf:JSON` / `"hovering"` |
| string | `"hello"` | | `"hello"` |
| string | entity ID, `http://example.org/x`, `_:b1` | | IRI, IRI, blank node |
| entity_id | entity ID (with or without `@id`) / `"not-an-id"` | | IRI / `"not-an-id"` |
| int | `42` | `xsd:integer` / `xsd:string` | `"42"^^xsd:integer` / `"42"` |
| string | `"x"` | `xsd:integer` | `"x"^^xsd:integer` (ill-typed, written as given) |
| int | `nil` | | nothing: triple dropped, no error |
| int | `struct{}{}` | | `"{}"` (plain string from `%v`) |

**Finding.** Every typed row agrees with D9, including the 2^53 limit, the fall back to the observed type, and IRIs
for entity-ID, absolute-IRI, and blank-node strings on a `string` predicate. Export differs from D9 on rows D9 makes
input errors: it drops nil, NaN, and infinite objects with no error, writes an unsupported Go value as a string, and
writes any hint verbatim. **Design change:** D9 records the difference as deliberate, and the agreement test covers
only the rows both accept, asserting the difference for the rest.

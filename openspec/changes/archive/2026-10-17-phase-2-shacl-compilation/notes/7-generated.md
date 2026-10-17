# Notes: task group 7 (determinism, fuzzing, and benchmarks)

Written 2026-10-16 with go1.26.9 darwin/arm64 (Apple M3 Pro, a shared host), SemEngine `v0.1.0-alpha.2`,
`pgregory.net/rapid v1.3.0`, and `gopkg.in/yaml.v3 v3.0.1`. Design D16 states what this group must show. The choices
below are the ones it leaves open, and the evidence for tasks 7.1–7.3.

## 7.1 Rapid properties

All are in `shacl/rapid_test.go`. Each runs at Rapid's default 100 checks under `task verify`, with a fresh random
seed each run, so `test:repeat` tries five more seeds per property.

### The generator

`genInput` builds an in-profile compile input: 1–5 predicates, one D8 datatype each; 1–3 entity types, some under a
shared superclass `Top` (so the Q24 closure rows are exercised); and 1–3 node shapes. Each node shape targets one or
two classes and has property shapes on a subset of the predicates. The property shapes carry the parts of the profile
that apply to their datatype: blank or IRI IDs, cardinality, integer and double bounds with every lower bound at or
below every upper bound, patterns from the D10 subset, and `sh:in` lists with repeats. Node shapes may be closed,
with `rdf:type` and other ignored properties. Severities and messages use strings that need Turtle and YAML escaping.
Some predicates carry a projection chain of 1–5 steps. Its step IRIs are drawn and then permuted, because Rapid draws
small values first and distinct draws otherwise tend to come out sorted. Every predicate goes to birth or one of two
groups, so Q19 holds. A type nothing constrains may have no contract, which makes it unwritable (Q18). The generator
records the data type its shapes imply for each predicate, and the step-order property uses that record as its oracle.

The writers (`writeTurtle`, `wireForm`, `writeNTriples`) are test code. The JSON/YAML writer names its fields from
design D3 and `testdata/diekm.shapes.yaml`, not from the decoder's struct tags. YAML is written by `yaml.Marshal`,
which quotes any string that would otherwise resolve to another type.

### Properties

Two outcomes are equal when both bundles are `reflect.DeepEqual`, or both issue lists are, and the error text matches.

| Property | Input change | Oracle |
| --- | --- | --- |
| `TestRapidRoundTripFormats` | The generated set written as Turtle, JSON, and YAML | The three decoded sets and bundles are equal; the compile must succeed; the decoded property-shape count equals the generated count |
| `TestRapidShapesStatementPermutation` | The generated Turtle, plus 0–6 out-of-profile statements (`sh:and` and `sh:rule` over blank nodes, an unknown `sh:` term, a stray blank subject, an asymmetric blank cycle, a stray IRI subject with a nested list), with statements permuted, up to 3 repeated, and blank nodes relabelled | Compiled from the unpermuted document |
| `TestRapidDIEKMStatementPermutation` | The spec scenario "Permuted shapes": `diekm.shacl.ttl` permuted and relabelled | Compiled from the file |
| `TestRapidW3CStatementPermutation` | One of the 28 rejected-family cases or 4 supported cases, read with its base and written back as N-Triples, then permuted and relabelled | The issue list compiled from the unpermuted case |
| `TestRapidInMemberOrder` | Each `sh:in` list reordered and partly repeated, in each of the three forms | Compiled from the Turtle in generated order |
| `TestRapidMappingRowOrder` | Every row list of the mapping reordered (rows, domain types, contracts, birth, groups, group predicates, ignored predicates), with 0–3 of 8 mapping faults applied first | Compiled from the unreordered mapping |
| `TestRapidCandidateTriplePermutation` | Candidate triples (mostly well typed, some of another type, 1 in 20 an input error such as nil, NaN, a struct, or invalid UTF-8, 1 in 30 about another subject) permuted, with up to 3 repeated under another `Source` and `Timestamp` | The report and error from the original order |
| `TestRapidProjectionStepsKeepBasePackOrder` | Generated packs with projection chains, and the mapping reordered | `packOrderMismatch`: the base pack copied by a JSON round trip, with only the generator's implied data types filled; entry and step order compared |

`TestProjectionStepSwapChangesBundle` is the step-order sensitivity pair. It swaps the first two steps of DIEKM's
three-step altitude chain in the base pack. The swapped pack must still pass `(*PackSpec).Validate`, the bundle must
change, and the compiled steps must be in the swapped order. `packOrderMismatch` must accept the plain bundle and
report the swapped one.

### Recorded runs

```text
go test ./shacl -run '^(TestRapid.*|TestProjectionStepSwapChangesBundle)$' -count=1 -cpu 1 -race -v \
  -rapid.seed=20261016 -rapid.checks=100
```

Every property printed `[rapid] OK, passed 100 tests`. `TestProjectionStepSwapChangesBundle` passed, and the package
took 3.9 s under `-race`. The same command without `-race` and with `-rapid.checks=10000` printed
`OK, passed 10000 tests` for all eight properties in 20.2 s.

Rapid writes a failing case under `shacl/testdata/rapid/<test>/` and replays every file there first. None is
committed: both real failures below are kept as named tests instead. The failfiles from seeded runs were deleted.

### Defects the properties found, now fixed

- **Blank-node cycles (parser).** `TestRapidShapesStatementPermutation` failed at
  `-rapid.seed=1209462990388893792`. A blank-node cycle outside every shape, `_:c1 r _:c2 . _:c2 q "w" . _:c2 r _:c1`,
  was recorded by whichever node came first in the document. That node took the other as owned, so `_:c2`'s `q`
  triple was reported in one order and not in the other. Now every blank subject left after the unreferenced ones
  (that is, every node on or under such a cycle) reports its own triples, and none owns another (`recordStray`,
  `shacl/parser.go`). Named test: `TestBlankCycleRecordsIndependentOfOrder` (`shacl/regression_test.go`). It failed
  before the fix, listing the `q` record in one order only.
- **Conflicting mapping rows (compiler).** `TestRapidMappingRowOrder` failed at `-rapid.seed=69982396324603329`. With
  two rows for one class, the compiler kept the first row read, so the issues about the other row's types depended on
  row order. A path mapped twice named its two predicates in row order, and a group declared twice with two modes
  kept the first mode. `Compile` now sorts a copy of every mapping row list before checking it (`canonicalMapping`,
  `shacl/compile.go`), so the first row is always the same one. Repeats are kept so the duplicate checks still see
  them. Named test: `TestConflictingMappingRowsIndependentOfOrder`. It failed before the fix, with five issues in one
  order and seven in the other. One existing expectation changed with it: `TestContracts/conflicting_modes` now reads
  "declared with modes append and reconcile", in sorted order.

### Shown able to fail

Each seeded change was applied, the property run with `-rapid.checks=100 -rapid.seed=20261016`, and the change
reverted (the script was a local scratch file, not committed). The replay seed is the one Rapid printed.

| Seeded change | Property | Result |
| --- | --- | --- |
| The JSON/YAML builder drops property `messages` | `TestRapidRoundTripFormats` | failed after 0 tests, seed 20261016 |
| Blank cycles record the first node only (the defect above, restored) | `TestRapidShapesStatementPermutation` | failed after 9 tests, seed 20261061 |
| Referenced blank subjects take part in the first pass | `TestRapidShapesStatementPermutation` | failed after 9 tests, seed 20261061 |
| `normalize` does not sort property shapes | `TestRapidDIEKMStatementPermutation` | failed after 0 tests, seed 20261016 |
| The Turtle builder keeps repeated triples | `TestRapidW3CStatementPermutation` | failed after 0 tests, seed 20261016 |
| `normalize` keeps `sh:in` in document order | `TestRapidInMemberOrder` | failed after 38 tests, seed 20261757 |
| `Compile` skips `canonicalMapping` (the defect above, restored) | `TestRapidMappingRowOrder` | failed after 5 tests, seed 20261031 |
| The validator counts repeated terms | `TestRapidCandidateTriplePermutation` | failed after 20 tests, seed 20261226 |
| `clonePack` sorts every step but the terminal one | `TestRapidProjectionStepsKeepBasePackOrder`, `TestProjectionStepSwapChangesBundle` | both failed; the property after 1 test, seed 20261017 |

**Survivor.** `normalize` not sorting the parser's rejection records survives `TestRapidW3CStatementPermutation` and
the shapes permutation. `Compile` sorts its issues, so at the `Compile` boundary this mutant is equivalent. No
property observes `ShapeSet` records directly.

**Corrected during the work.** The step-order property first built its expected pack with the package's own
`clonePack`. The sorted-steps mutant then changed both sides and survived 100 checks on five seeds. The oracle now
copies the pack through `encoding/json` (`copyPack`). After that, and after the step IRIs were permuted, the mutant
fails on the first test.

### Not covered

A `ShapeSet` built by hand rather than decoded can hold unsorted lists. `Compile` copies `In`, messages, and
properties in the order given, so two such sets differing only in order give bundles that are not `DeepEqual`,
though they validate alike. The model's doc comments state the sorted form, and the decoders always produce it.
D16 states the properties over documents, and that is what is tested.

## 7.2 Fuzz targets

| Target | File | Committed corpus | Added in code |
| --- | --- | --- | --- |
| `FuzzTurtleReader` | `internal/turtle/fuzz_test.go` | 13 files: rejected syntax (SPARQL `PREFIX`/`BASE`, single quotes, name escapes), relative IRIs with and without a base, literals, collections, invalid UTF-8, empty input, and the crasher below | the 43 W3C cases (with base), DIEKM, nesting at 64 and 65 levels |
| `FuzzParseTurtleShapes` | `shacl/fuzz_test.go` | 13 files: `sh:rule`, logical and SPARQL constructs, `sh:flags`, a blank-node node shape, closed without `rdf:type`, bad bounds and counts, a typed `sh:in` member, complex paths, an unknown term, `owl:imports`, a relative IRI, nesting at 65 | DIEKM; the 28 rejected-family cases as N-Triples; the 1 MiB cap and one byte over |
| `FuzzDecodeShapesJSON` | `shacl/fuzz_test.go` | 13 files: each refusal the decoder names, syntax errors, and one input using every field | `testdata/diekm.shapes.json` |
| `FuzzDecodeShapesYAML` | `shacl/fuzz_test.go` | 14 files: each refusal, aliases, control characters, invalid UTF-8, tabs, and the two crashers below | `testdata/diekm.shapes.yaml` |
| `FuzzPatternSubset` | `shacl/fuzz_test.go` | 6 files: refused escapes, class subtraction, categories, reluctant quantifiers, nesting, a trailing backslash | the 8 generator patterns and 16 refusals |

Plain `go test` runs every committed and added seed, so `task verify` runs them under `test:unit` and `test:repeat`.

**The property.** No panic, and either an error or a result.

- The reader's triples must be well formed: an IRI or blank subject, an IRI predicate, an absolute IRI wherever an
  IRI appears, a datatype on every literal.
- The shapes targets compile what they decode, against both `testPack`/`testMapping` and DIEKM's pack and mapping. A
  failure must be a `*CompileError` whose every issue names its construct.
- A refused pattern must be a `*patternError` with a construct and an offset inside the pattern. An accepted pattern
  must use no escape other than XSD's single-character escapes and `\p`/`\P`. That escape scan is independent of the
  translator's parser.

**"Names a construct or a location."** Each target lists the error forms it accepts as regular expressions: a line
and column, a byte offset, a JSON path, a field name, a shape or IRI, or a named YAML construct (an anchor, an alias,
a merge key, `!!binary`). An error matching none fails the target, so a new form is read before it is accepted.
`TestFuzzErrorFormsReachable` shows each form matches an error the code really returns. It also shows that none
accepts five errors that name nothing: `EOF`, `unexpected EOF`, the reader's old UTF-8 message, and two yaml.v3
messages.

### Fixes the targets forced

Each was first a failing seed or crasher, then fixed.

- **Reader, invalid UTF-8.** The error named no position. It now gives the byte offset of the first invalid byte
  (`turtle: input is not valid UTF-8 at byte offset N`). Seed `invalid-utf8`.
- **Reader, IRI schemes.** A 60-second run found `<0:>` accepted as an absolute IRI (crasher `7aa994686e8f9be9`,
  committed). `isAbsolute` only checked for a colon before any `/?#`. RFC 3986 §3.1 requires a scheme of
  `ALPHA *( ALPHA / DIGIT / "+" / "-" / "." )`, and §4.2 forbids a colon in the first segment of a relative path, so
  `0:` is neither an absolute IRI nor a relative reference. The reader now fails such an IRI with or without a base:
  `IRI <0:> has an invalid scheme (RFC 3986 §3.1)`. A base passed to `WithBase` must also have a valid scheme. The
  JSON and YAML forms already refused these (`isAbsoluteIRI` in `shacl/decode.go`), so the Turtle form agreed with
  them on less than it should have. Test: `TestParseRejectsInvalidScheme`.
- **JSON decoder.** encoding/json's syntax errors, type errors without a field, and truncated input named no
  position. Trailing data named none either. Each now carries `byte offset N:`. The offset comes from
  `json.SyntaxError.Offset`, `json.UnmarshalTypeError.Offset`, the input length for a truncated document, and the
  first byte after the document for trailing data. Empty input is now `no document`, as YAML's already was. The two
  `json empty` cases in `TestDecodeShapesRefuses` and `TestDecodeMappingRefuses` now expect that text.
- **YAML decoder.**
  - yaml.v3's reader refuses invalid UTF-8 and characters YAML does not allow with no position (`readerc.go:214`,
    `:379`). Those errors now carry the byte offset of the first such byte.
  - yaml.v3's parser and scanner drop the line when an error's marks are on the first line, which they count from 0
    (`decode.go:112-137`). Crashers `9cc52906ed53ef5f` (`"`) and `4fd144b8dddb06ef` (`node_shapes: !"`) showed this.
    Such errors now read `line 1: …`.
  - yaml.v3's own decoder errors already name their construct and pass through unchanged. The YAML checks apply to
    all three decode passes.

### Choices

- **Seeds left out of fuzzing.** The 53 KB SHACL vocabulary is not a reader seed; `TestW3CVocabulary` reads it. As a
  seed it slowed every mutation and minimization. The two 1 MiB size-cap seeds of `FuzzParseTurtleShapes` run under
  plain `go test` only (the target checks whether `-test.fuzz` is set). In a fuzz session the engine mutates and
  minimizes every seed.
- **Minimization.** Go minimizes every new-coverage input for up to `-fuzzminimizetime` (default 60 s), and the
  coordinator counts no executions meanwhile. An early run of the reader target showed 76 executions in 60 s. The
  recorded runs pass `-fuzzminimizetime 10s`.
- **DIEKM's pack and mapping** are loaded once per process (`sync.OnceValues`), outside the inputs the fuzzer times.

### Recorded runs

Go 1.26.9 darwin/arm64, Apple M3 Pro, run one after another on 2026-10-16 from 22:43 UTC. Each used a fresh cache
directory, so only the seeds above fed it:

```text
go test ./<pkg> -run '^$' -fuzz '^<Target>$' -fuzztime 60s -fuzzminimizetime 10s -parallel 1 \
  -args -test.fuzzcachedir=/tmp/semvocab-t7/fuzzcache/<Target>
```

| Target | Package | Duration | Executions | New interesting | Result |
| --- | --- | --- | --- | --- | --- |
| `FuzzTurtleReader` | `internal/turtle` | 60 s | 1,689,019 | 182 | PASS |
| `FuzzParseTurtleShapes` | `shacl` | 60 s | 34,761 | 9 | PASS |
| `FuzzDecodeShapesJSON` | `shacl` | 60 s | 169,471 | 63 | PASS |
| `FuzzDecodeShapesYAML` | `shacl` | 60 s | 239,876 | 108 | PASS |
| `FuzzPatternSubset` | `shacl` | 60 s | 229,792 | 122 | PASS |

`FuzzParseTurtleShapes` runs fewest executions. Each one parses and then compiles twice, and most of its minute went
to minimizing inputs derived from the 1–3 KB W3C seeds. Earlier 60-second runs, before the fixes above, found the
three crashers listed. They are committed under `testdata/fuzz/<Target>/` with Go's names.

## 7.3 Benchmarks

`shacl/bench_test.go` has `BenchmarkCompile` and `BenchmarkValidateCandidateState`. The workloads, the command, the
machine, and the results table are in `docs/shacl-compilation.md`, whose other sections task 8.1 writes. The run
took 96 s, and the host's load average rose from 1.8 to 5.8 during it, so the table gives the median with the
fastest and slowest sample.

## Cost under `task verify`

On this host, comparing `071286c` with this group's tree. The timings are noisy because the host is shared.

- `go test -race -count=1 -cpu 1`: `shacl` went from 2.5–2.8 s to 4.7–6.8 s, and `internal/turtle` from 2.0–2.1 s to
  2.7–3.1 s.
- `go test -count=5 -cpu 1 -shuffle=on`: `shacl` went from 0.6–2.1 s to 2.0–4.4 s, and `internal/turtle` from
  0.5 s to 0.7–3.1 s.

The whole `task verify` before this group spent 26 s in `test:unit` and 137 s in `test:repeat`, most of it in
`internal/harness/contract`.

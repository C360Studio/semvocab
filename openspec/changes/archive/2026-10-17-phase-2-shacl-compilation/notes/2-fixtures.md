# Notes: task group 2 (fixtures and dependencies)

Written 2026-10-16 with go1.26.9 darwin/arm64. The source tree is `w3c/data-shapes` at
`4cd469867a1cc6d2fff80e8a40723b26c695d67a` (tarball from `codeload.github.com`, the same download probe P2 used).
The files were copied byte for byte. Nothing from the tree was executed.

## 2.1 Where the fixtures live and how they are read

- **Location.** `internal/oracle/testdata/w3c-data-shapes/`, keeping each file's upstream path, beside the BFO and
  CCO oracles. The ledger section `pins.fixtures.w3c-data-shapes` lists every file by its upstream path with its
  sha256. `vendored_at` names the directory.
- **Reading.** `oracle.ReadFixture(oracle.W3CDataShapes, path)` refuses a path the ledger does not list, then checks
  the sha256 before it returns the bytes. `TestFixturesMatchLedger` requires the files on disk and the ledger list to
  be the same set, and reads each one. `TestReadFixtureRefusesAlteredOrUnlisted` flips one byte in a copy and
  requires a digest mismatch.
- **License.** `LICENSE.md` from the repository root is vendored and pinned like the other files. It names the W3C
  Software and Document License by its undated URL, which now serves the 2023 version. The header of
  `shacl/shacl.ttl` names the 2015 version. `NOTICE` (SemVocab's own file, outside the ledger list) carries the full
  text of both versions as published on 2026-10-16, which is what the license requires of a copy. The two versions
  differ only in how they name the copyright holder.
- **Markdown lint.** `.markdownlint-cli2.yaml` now ignores `internal/oracle/testdata/**`. The vendored `LICENSE.md`
  fails MD041 and MD009, and fixing that would mean changing a pinned file.
- **Ledger checks.** `checkShape` requires the `w3c-data-shapes` set and, for each set, a repository, license,
  `vendored_at`, `selection`, a 40-hex commit, at least one file, and a path plus 64-hex sha256 on each file with no
  repeated path. `TestAdmissionLedgerShape` also checks the spec scenario: `w3c/data-shapes`, the full commit, the
  W3C license, and `shacl/shacl.ttl` with the Q21 sha256, which the test states on its own rather than reading it
  from the ledger. `TestAdmissionLedgerShapeSensitivity` removes, in turn, the license, the commit, a file's
  sha256, a file's path, the whole file list, and the set itself, and requires each removal to be named. It also
  renames the license key, which the strict decoder rejects.

`shacl/shacl.ttl` hashed to `1a534caa9d3df5bbc5e5351002bda56ead9226fda0a617c165b218d8bd06b9af`, the Q21 value.

## 2.1 Which test-suite cases, and why

D15 asks for the cases for the supported components (minCount, maxCount, datatype, the four bounds, pattern, `in`,
closed) and one case for each rejected family. The two rules used:

- **Self-contained files only.** Every case chosen holds its data, shapes, and expected report in one file. The split
  cases (`*-data.ttl` with `*-shapes.ttl`) were left out, so the test-suite loader never has to follow `owl:imports`
  or a second file.
- **A rejected family is one rejected target kind, path kind, or constraint component.** D15 names `and-001` and
  `or-001` separately, though both are logical components, so a family here is one component, not one D4 bullet. That
  gives every rejected construct in D4 that has a self-contained Core case its own W3C witness. Families with no Core
  case cannot be vendored: `sh:target`, the SPARQL-based and JavaScript terms, `sh:rule` and the other Advanced
  Features terms, and `sh:group`. Their rejection is covered by `TestProfileClassifiesEveryTerm` (task 5.1) and by
  hand-written cases.

This gives 43 cases (88 KB; 142 KB with `shacl.ttl` and `LICENSE.md`), not the "roughly 20" the design's declared
costs estimated. The estimate assumed fewer families. Keeping one path case and one target case would bring it near
35, if a reviewer prefers that.

### Supported components (15)

| Case | Component | Note |
| --- | --- | --- |
| `property/minCount-001` | `sh:minCount` | |
| `property/minCount-002` | `sh:minCount` | with `sh:name`, an ignored annotation |
| `property/maxCount-001` | `sh:maxCount` | |
| `property/maxCount-002` | `sh:maxCount` | without `sh:datatype` |
| `property/datatype-001` | `sh:datatype` | |
| `property/datatype-002` | `sh:datatype` | |
| `property/minExclusive-001` | `sh:minExclusive` | |
| `property/maxExclusive-001` | `sh:maxExclusive` | |
| `property/maxInclusive-001` | `sh:maxInclusive` | |
| `node/minInclusive-001` | `sh:minInclusive` | the only Core `sh:minInclusive` cases are node-level |
| `property/pattern-001` | `sh:pattern` | |
| `property/in-001` | `sh:in` | |
| `node/closed-001` | `sh:closed` | reports `rdf:type` as a violation (D12) |
| `node/closed-002` | `sh:closed`, `sh:ignoredProperties` | |
| `targets/targetClass-001` | `sh:targetClass`, `sh:maxCount` | the one in-profile target, with no `sh:targetNode` |

Most supported cases also use `sh:targetNode`, so task 6.3's adapter must turn focus nodes into entities, as D15
says. `node/minInclusive-001` constrains literal focus nodes on a node shape. The adapter may not be able to run it,
and task 6.3 decides that. `misc/severity-001` and `misc/message-001` exercise `sh:severity` and `sh:message`, which
D4 supports. D15 does not list them, so they are not vendored.

### Rejected families (28)

| Case | Family (construct named in the rejection) |
| --- | --- |
| `targets/targetSubjectsOf-001` | `sh:targetSubjectsOf` |
| `targets/targetObjectsOf-001` | `sh:targetObjectsOf` |
| `targets/targetClassImplicit-001` | implicit class target (a shape that is an `rdfs:Class`) |
| `path/path-sequence-001` | sequence path (an RDF list) |
| `path/path-alternative-001` | `sh:alternativePath` |
| `path/path-inverse-001` | `sh:inversePath` |
| `path/path-zeroOrMore-001` | `sh:zeroOrMorePath` |
| `path/path-oneOrMore-001` | `sh:oneOrMorePath` |
| `path/path-zeroOrOne-001` | `sh:zeroOrOnePath` |
| `property/class-001` | `sh:class` |
| `property/node-002` | `sh:node` |
| `node/nodeKind-001` | `sh:nodeKind` |
| `property/minLength-001` | `sh:minLength` |
| `property/maxLength-001` | `sh:maxLength` |
| `property/languageIn-001` | `sh:languageIn` |
| `property/uniqueLang-001` | `sh:uniqueLang` |
| `property/pattern-002` | `sh:flags` |
| `property/hasValue-001` | `sh:hasValue` |
| `property/equals-001` | `sh:equals` |
| `property/disjoint-001` | `sh:disjoint` |
| `property/lessThan-001` | `sh:lessThan` |
| `property/lessThanOrEquals-001` | `sh:lessThanOrEquals` |
| `property/not-001` | `sh:not` |
| `property/and-001` | `sh:and`, with `sh:targetNode` (the spec's "several rejected constructs" scenario) |
| `property/or-001` | `sh:or` |
| `node/xone-001` | `sh:xone` |
| `property/qualifiedValueShape-001` | `sh:qualifiedValueShape` and the qualified counts |
| `misc/deactivated-001` | `sh:deactivated` |

`sh:targetNode` gets no case of its own: `property/and-001` and most supported cases use it.

### Expected triple counts (for task 3.1)

rdflib 7.1.4, run as `/tmp/semvocab-probes-venv/bin/python -I /tmp/semvocab-probes/p2/count.py <dir>` over the
vendored `data-shapes-test-suite/tests/core` and `shacl` directories, with base
`http://semvocab.test/core/<relative path>`. rdflib's `Graph` is a set, so these are counts of distinct triples.

| File | Triples |
| --- | --- |
| `shacl/shacl.ttl` | 1127 |
| `misc/deactivated-001` | 22 |
| `node/closed-001` | 39 |
| `node/closed-002` | 34 |
| `node/minInclusive-001` | 25 |
| `node/nodeKind-001` | 24 |
| `node/xone-001` | 45 |
| `path/path-alternative-001` | 58 |
| `path/path-inverse-001` | 51 |
| `path/path-oneOrMore-001` | 40 |
| `path/path-sequence-001` | 53 |
| `path/path-zeroOrMore-001` | 32 |
| `path/path-zeroOrOne-001` | 32 |
| `property/and-001` | 68 |
| `property/class-001` | 53 |
| `property/datatype-001` | 48 |
| `property/datatype-002` | 41 |
| `property/disjoint-001` | 50 |
| `property/equals-001` | 78 |
| `property/hasValue-001` | 39 |
| `property/in-001` | 44 |
| `property/languageIn-001` | 53 |
| `property/lessThan-001` | 59 |
| `property/lessThanOrEquals-001` | 51 |
| `property/maxCount-001` | 37 |
| `property/maxCount-002` | 32 |
| `property/maxExclusive-001` | 51 |
| `property/maxInclusive-001` | 41 |
| `property/maxLength-001` | 34 |
| `property/minCount-001` | 35 |
| `property/minCount-002` | 20 |
| `property/minExclusive-001` | 41 |
| `property/minLength-001` | 34 |
| `property/node-002` | 40 |
| `property/not-001` | 40 |
| `property/or-001` | 45 |
| `property/pattern-001` | 46 |
| `property/pattern-002` | 39 |
| `property/qualifiedValueShape-001` | 47 |
| `property/uniqueLang-001` | 57 |
| `targets/targetClass-001` | 33 |
| `targets/targetClassImplicit-001` | 33 |
| `targets/targetObjectsOf-001` | 37 |
| `targets/targetSubjectsOf-001` | 30 |

## 2.2 Rapid

- `go get pgregory.net/rapid@v1.3.0` then `go mod tidy` added the direct requirement. `go.sum` already held its
  hashes, so it did not change.
- `go mod tidy` drops a requirement nothing imports, so task 2.2 needs a first test that uses Rapid.
  `TestVerifySHA256RejectsEveryByteFlip` in `internal/oracle` is that test: XOR any one byte of the vendored
  `shacl.ttl` with any non-zero value, and the digest check must fail. It runs at Rapid's default 100 checks.
  Shown able to fail: with `VerifySHA256` changed to return nil for non-empty input, it failed on its first draw
  (`byte 0 xor 0x1 passed the digest check`, `-rapid.seed=4784574608672249962`). Rapid's failfile from that seeded
  run was deleted, because it records the seeded defect, not a real failure.
- "Imported only from test files" is enforced by `TestTestOnlyImports` in `internal/harness/contract`. It parses the
  imports of every non-test `.go` file git lists and fails on `pgregory.net/rapid`. It also fails unless `go.mod`
  requires `pgregory.net/rapid v1.3.0` directly. `TestTestOnlyImportsSensitivity` seeds a production import, an
  `// indirect` requirement, and another version, and requires each to be named. The `AGENTS.md` and
  `.agents/rules-index.md` rows for "Production code imports no test library" now name the test. The gate stays
  `review`, because no guard covers other test libraries or `internal/harness`.

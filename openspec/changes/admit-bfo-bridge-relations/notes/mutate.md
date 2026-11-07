# Notes: Task 3.5 (mutation evidence)

Each mutant is a whole-file copy outside the repository (`mktemp -d`), run with `ateam v0.3.8` through
`task mutate:check`, which overlays it with `go test -overlay` and writes nothing in the tree.

## Why `parityProblems` cannot see a bfo.go mutant

`internal/gosrc.Load` parses `catalog/bfo/*.go` from disk (`os.ReadDir` and `parser.ParseFile`), and
`go test -overlay` does not change what those calls read. So `parityProblems`, which reads constants through `gosrc`,
cannot see an overlay mutant of `bfo.go`. Only checks that read the compiled `bfo.X` values can:

- `TestCatalogPortParity`'s scenario loop (`BearerOf`, `InheresIn`, `OccupiesTemporalRegion`, `Precedes`,
  `PrecededBy`, each against its `bfo-core.ttl` label).
- `TestRelationConstantsMatchLedger`, which compares the compiled value of all 15 ruling R1 constants
  (`compiledRelations`) with the ledger row and the release declaration. `design.md` D5 does not name this test; it
  was added so that D5 mutants 1-3 are reachable.

## Mutants

`$D` and `$M` are `mktemp -d` directories. Each bfo.go mutant changes only the named constant lines. The runs below
use the committed ledger, with D3 decided as deletion of the three conflicting exclusion rows.

| # | D5 mutant | Command (`task mutate:check -- …`) | Verdict |
| --- | --- | --- | --- |
| 1 | `BearerOf` -> `BFO_0000053` | `-pkg ./catalog/bfo -test TestCatalogPortParity -file catalog/bfo/bfo.go -mutant $D/m1/bfo.go -expect-text "bfo."` | detection (`bfo.BearerOf = …BFO_0000053 labelled [], want BFO_0000196 labelled "bearer of"`) |
| 1 | same | `-test TestRelationConstantsMatchLedger`, same `-pkg`, `-file`, `-mutant`, `-expect-text` | detection (`ledger records BFO_0000196`) |
| 1 | same | `-pkg ./packs/diekm -test TestDIEKMProjectionStepsUseBFORelations -file catalog/bfo/bfo.go -mutant $D/bfo.go -expect-text "is not bfo.BearerOf"` | detection (`intel.role.assigned step 0: predicate …BFO_0000196 is not bfo.BearerOf …`) |
| 2 | `PrecededBy` / `Precedes` IRIs swapped | `-pkg ./catalog/bfo -test TestCatalogPortParity -file catalog/bfo/bfo.go -mutant $D/m2/bfo.go -expect-text "bfo."` | detection (both constants named with the wrong label) |
| 2 | same | `-test TestRelationConstantsMatchLedger`, same arguments | detection (both constants named) |
| 3 | `OccupiesTemporalRegion` -> `BFO_0000132` | `-pkg ./catalog/bfo -test TestCatalogPortParity -file catalog/bfo/bfo.go -mutant $D/m3/bfo.go -expect-text "bfo."` | detection (`labelled ["occurrent part of"], want BFO_0000199 labelled "occupies temporal region"`) |
| 3 | same | `-test TestRelationConstantsMatchLedger`, same arguments | detection (`ledger records BFO_0000199`) |
| 4 | `inverseKey` direction normalisation removed (one-direction match) | `-pkg ./catalog/bfo -test TestRelationAxiomsMatchReleaseSensitivity -file catalog/bfo/bfo_test.go …` | refused: `-file must be a non-test .go source file` (see below) |
| 5 | `IRIObjects` keeps literal objects | `-pkg ./internal/oracle -test TestIRIObjects -file internal/oracle/oracle.go -mutant $M/oracle.go -expect-text "IRIObjects("` | detection (`IRIObjects(…BFO_0000196, rdfs:label) = ["bearer of"], want []`) |

## Mutant 4 (test-file target)

`ateam mutate` refuses `*_test.go` targets, and the inverse comparison (`inverseKey`, `axiomProblems`) is test code in
`catalog/bfo/bfo_test.go`. The same mutant was run by hand through the same mechanism:
`go test -overlay $M/overlay.json ./catalog/bfo -run '^TestRelationAxiomsMatchRelease' -count=1`, where the overlay
replaces `bfo_test.go` with a copy whose `inverseKey` returns `a + " " + b` unsorted.

- `TestRelationAxiomsMatchRelease`: **survivor**. Every ledger pair is written subject-first, the same direction as
  the `bfo-core.ttl` assertion, so a one-direction match still passes on the committed ledger.
- `TestRelationAxiomsMatchReleaseSensitivity`: **detection**. The `pair recorded in reverse direction` case fails with
  `want no problem, got: ledger inverse pair InheresIn BearerOf is not asserted …`. The `invented pair` case also
  fails, because its key is no longer normalised.

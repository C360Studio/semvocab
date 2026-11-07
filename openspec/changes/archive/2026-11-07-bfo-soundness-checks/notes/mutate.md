# Notes: Task 3.7 (mutation evidence)

Each mutant is a whole-file copy in a `mktemp -d` directory (`$M`) outside the repository, with `ateam v0.3.8`. Runs
were made at commit `aade3ad`; nothing in the tree was written.

## Mutant 7 (non-test file): `task mutate:check`

Mutant: `oracle.Objects` keeps only non-blank objects (`if t.P.Value == p && t.O.Kind != turtle.Blank`).

| Command (`task mutate:check -- …`) | Verdict |
| --- | --- |
| `-pkg ./internal/oracle -test TestObjects -file internal/oracle/oracle.go -mutant $M/m7/oracle.go -expect-text "Objects("` | detection (`Objects(…BFO_0000196, …rdf-schema#domain) = [], want one term`) |
| `-pkg ./packs -test TestBFORelationBounds -file internal/oracle/oracle.go -mutant $M/m7/oracle.go -expect-text "declares no"` | detection (`…BFO_0000196 declares no …rdf-schema#domain`) |
| `-pkg ./packs -test TestBFODisjointPairsFromRelease -file internal/oracle/oracle.go -mutant $M/m7/oracle.go -expect-text "owl:members"` | detection (`owl:AllDisjointClasses _:b154 has 0 owl:members lists, want 1`) |

A first `TestBFORelationBounds` run with `-expect-text "class expression"` was **inconclusive** (the test failed, but
not with that text); the row above is the rerun with the text the failure actually carries.

## Mutants 1-6 (test file): by hand through `go test -overlay`

`ateam mutate` refuses `*_test.go` targets, and every check lives in `packs/bfo_soundness_test.go` (design D1):

```text
task mutate:check -- -pkg ./packs -test TestBFODisjointPairsFromRelease -file packs/bfo_soundness_test.go \
  -mutant $M/m1/bfo_soundness_test.go -expect-text "disjoint"
ateam mutate: refused, and no run was started: -file must be a non-test .go source file
```

So each was run by hand with the same mechanism ateam uses:
`go test -overlay $M/mN/overlay.json ./packs -run '^TestBFO' -count=1`, where `overlay.json` replaces
`packs/bfo_soundness_test.go` with the mutant. The after-run without the overlay passes.

| # | D6 mutant (edit in the copy) | Failing tests (verdict) |
| --- | --- | --- |
| 1 | disjointness reader drops `AllDisjointClasses` (`range []string(nil)` for the blank-subject loop) | detection: `TestBFODisjointPairsFromRelease` (`read 5 disjoint pairs, want 26`), `TestBFOPackAnchoringSensitivity/independent_continuant_and_quality_collide_(AllDisjointClasses)` |
| 2 | bucket check reads only `class_iri` (`set := []string{et.ClassIRI}`) | detection: `TestBFOPacksAnchorEveryEntityType` (`intel.platform has no BFO bucket root`, same for `intel.observation`), `TestBFOPackAnchoringSensitivity` (both collision twins) |
| 3 | hop check skips the previous-node domain on step > 0 (`if si == 0 && !d.holds(prev)`) | detection: `TestBFOProjectionHopsSensitivity/carrier_hop_between_information_entities` |
| 4 | interpreter ignores `complementOf` (`NoneOf` never appended) | detection: `TestBFORelationBounds` (`BFO_0000196 domain … NoneOf:[]`), `TestBFOProjectionHopsSensitivity/spatial_region_bearer` |
| 5 | anchoring accepts any OBO predicate (`!strings.HasPrefix(step.PredicateIRI, bfo.Namespace)` on both rules) | detection: `TestBFOProjectionHopsSensitivity/unanchored_role` |
| 6 | selector takes every pack (`out = append(out, p); continue` before the test) | detection: `TestBFOPackSelection` (`selected packs = [bfo-cco-core swe-core diekm sensorml]`), `TestBFOPacksAnchorEveryEntityType`, `TestBFOProjectionHops` |

No mutant survived every test. Per-test survivors worth naming: mutant 5 passes the `unanchored quality` twin (its
seeded predicate, `cco:ont00001904`, is outside the OBO namespace, so the mutant still rejects it); only the
`unanchored role` twin (`bfo.ParticipatesIn`) kills it. Mutant 3 is killed by one twin only.

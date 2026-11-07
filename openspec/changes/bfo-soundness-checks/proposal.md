# Proposal: bfo-soundness-checks

Track: design (adds three `catalogs-and-packs` Requirements; `design.md`).

## Why

Packs over BFO 2020 and CCO v2.2 (`packs/diekm/diekm.yaml:29-77`) hand-write `superclasses` closures and
`ProjectionSpec` chains. Their BFO soundness is checked only case by case (rulings Q2-Q4b on #1;
`packs/diekm/diekm_test.go:247-298`). Issue #32 asks for static checks, in pure Go and ahead of time, of the BFO
structural invariants. Ruling R1 on #32 (2026-11-06) keeps them in the test harness (`internal/`, `packs/*_test.go`)
with no new exported production API, and adds the #31 relations to the projection table.

## What Changes

- `internal/oracle`: export `Objects(s, p)` (blank-node and literal objects) so tests can read `owl:unionOf`,
  `owl:intersectionOf`, `owl:complementOf`, and `owl:members` lists from `bfo-core.ttl`.
- `packs/bfo_soundness_test.go` (new), over every embedded pack whose ledger entry references `bfo-2020` or
  `cco-2.2` (today `bfo-cco-core`, `diekm`):
  - every entity type reaches one of seven BFO bucket roots;
  - no entity type's closure meets two members of a `bfo-2020` disjointness axiom;
  - every projection hop through one of 17 BFO relations has its subject and node class inside the relation's
    `bfo-2020` `rdfs:domain` and `rdfs:range`; a hop from an independent to a specifically dependent continuant
    uses `bearer of`, and the reverse hop uses `inheres in`;
  - an opt-in diagnostic: each continuant entity type has a projection reaching a `bfo:Process`.
- Each check has seeded-fault twins naming the offending entity type or step. `task mutate:check` evidence.
- `docs/catalogs-and-packs.md`: the anchoring rules and the bearer-less metric convention.

## Non-goals

- No exported API, no runtime or SemEngine hook, no OWL reasoner, no `owl:Restriction` evaluator.
- No check on packs that do not reference `bfo-2020`/`cco-2.2` (`swe-core`, `sensorml`).
- No change to `diekm.yaml`, its mapping, `catalog/bfo` constants, the ledger, or any pin.
- No CCO relation domain/range table (`cco:ont00001904` steps stay unchecked).

## Capabilities

- `catalogs-and-packs`: three new requirements (BFO bucket anchoring; BFO projection hops; process connectivity).

## Impact

Test-only plus one exported method in `internal/oracle`. SemEngine stays at `v0.1.0-alpha.2`. Closes #32
(draft PR #39).

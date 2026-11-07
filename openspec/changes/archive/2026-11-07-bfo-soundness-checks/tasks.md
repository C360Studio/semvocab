# Tasks: bfo-soundness-checks

Each task ends when `task verify` passes on a clean tree with `$(go env GOPATH)/bin` first on `PATH`. Every new or
modified Go test carries a `// spec:` citation to a requirement in this change's `specs/` (`task test:spec`).

## 1. Approval

- [x] 1.1 Approve `proposal.md` and `design.md` for #32 (ruling R1, 2026-11-06), answering design Open Questions
  1-4 (`.agents/contracts/architect.md`, `.agents/protocol.md:47-53`). Acceptance: `proposal.md` carries
  `Approved-by: <name> (<YYYY-MM-DD>)` and this hold is removed in the same edit.

## 2. Oracle

- [x] 2.1 Export `(*Graph).Objects(s, p string) []turtle.Term` (premise: unexported `objects`,
  `internal/oracle/oracle.go:146-155`; `IRIObjects` skips blank nodes at `:171-181`). Acceptance: an
  `internal/oracle` test reads `BFO_0000196`'s `rdfs:domain` as one blank node whose `owl:intersectionOf` list
  holds `BFO_0000004` and a blank node with `owl:complementOf BFO_0000006` (`bfo-core.ttl:532-538`).

## 3. Checks, sensitivity, and mutation evidence

- [x] 3.1 In new `packs/bfo_soundness_test.go`: the selector (design D2; premise `internal/ledger/ledger.go:146`,
  `docs/admission-ledger.yaml:1075,1104`, `packs/packs_test.go:62-67`), the bucket check, and the disjointness check
  over sets read from `bfo-core.ttl:686-1055,1185-1223` (D6), classifying through `cco.IsA`
  (`catalog/cco/hierarchy.go:78-88`). Acceptance: the "Pack selection" and "DIEKM is anchored" scenarios pass.
- [x] 3.2 Seeded twins for 3.1: missing bucket root; `Continuant`+`Occurrent`; `IndependentContinuant`+`Quality`
  (via `AllDisjointClasses` at `:1185`). Acceptance: each names the entity type and the rule.
- [x] 3.3 The domain/range interpreter over the 17 relations (D4; premise `catalog/bfo/bfo.go:103-161`,
  `bfo-core.ttl:102-573`), failing closed on any other class-expression shape. Acceptance: a test pins `196` domain,
  `059` domain, and `199` range as written in design D4.
- [x] 3.4 The hop check with step-0 subjects from `shacl.DecodeMappingYAML(diekm.Mapping())` (D3; premise
  `vocabulary/projection.go:13-15`, `packs/diekm/diekm.mapping.yaml:21-29`, `shacl/model.go:104-117`) and the D5
  anchoring rule. Acceptance: the "DIEKM chains are sound" scenario passes.
- [x] 3.5 Seeded twins for 3.4: unanchored quality; unanchored role; `IsCarrierOf` from a GDC; a process bearer;
  a `SpatialRegion` bearer; a projected BFO step with no bound subject; a node class unknown to the catalogs.
  Acceptance: each names the predicate and step.
- [x] 3.6 The opt-in connectivity diagnostic (D5). Acceptance: both "Process connectivity is opt-in" scenarios pass.
- [x] 3.7 Run `task mutate:check` on design D6 mutants 1-7 and record each command and verdict in
  `openspec/changes/bfo-soundness-checks/notes/mutate.md`. Acceptance: each mutant is killed or recorded as a
  survivor.

## 4. Docs and archive

- [x] 4.1 Add "BFO alignment" to `docs/catalogs-and-packs.md` after the `diekm` section (premise `:241-268`;
  "Using your own pack" at `:440-445`): the seven buckets (issue #32's grouping, not a BFO term), the disjointness
  rule, the hop table, and the bearer-less metric convention (a score with no material bearer is a
  `cco.InformationContentEntity`, not an unanchored `bfo.Quality`). Correct the `HasMemberPart` and
  `OccupiesTemporalRegion` comments at `catalog/bfo/bfo.go:142,153` to the fixture's domains. Acceptance:
  `task docs:check` passes.
- [x] 4.2 Run `task verify` on a clean tree at the branch head. Acceptance: it exits 0 with no tracked-file change.
- [x] 4.3 Archive with `npx --no-install openspec archive bfo-soundness-checks --yes` before marking PR #39 ready.
  Acceptance: `openspec/specs/catalogs-and-packs` carries the three requirements and `task spec:check` passes.

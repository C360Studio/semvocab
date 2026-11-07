# Tasks: admit-bfo-bridge-relations

Each task ends when `task verify` passes on a clean tree with `$(go env GOPATH)/bin` first on `PATH`. Every new or
modified Go test carries a `// spec:` citation to a requirement in this change's `specs/` (`task test:spec`).

## 1. Approval

- [x] 1.1 Approve `proposal.md` and `design.md` for #31 (ruling R1, 2026-11-06), answering design Open Questions
  1-3 (`.agents/contracts/architect.md`, `.agents/protocol.md:47-53`). Acceptance: `proposal.md` carries
  `Approved-by: <name> (<YYYY-MM-DD>)` and this hold is removed in the same edit.

## 2. Oracle and ledger

- [x] 2.1 Add `OWLInverseOf`, `RDFSSubPropertyOf`, and `(*Graph).IRIObjects(s, p string) []string` (IRI objects in
  document order, literals and blank nodes skipped) to `internal/oracle/oracle.go` (premise: only unexported
  `objects` exists, `internal/oracle/oracle.go:144-153`; well-known IRIs at `:27-37`). Acceptance: a new
  `internal/oracle` test shows `IRIObjects(BFO_0000196, OWLInverseOf)` is `[BFO_0000197]` and
  `IRIObjects(BFO_0000196, RDFSLabel)` is empty.
- [x] 2.2 Extend `ledger.BFOSource` with `ExternalAdditions.Relations map[string][2]string` and `RelationAxioms`
  (`InversePairs [][2]string`, `SubPropertyOf map[string]string`); `ledger.Exclusion` is unchanged (`design.md` D3;
  premise: `internal/ledger/ledger.go:184-188,199-209`; strict decoding at `:410-429`). Acceptance:
  `TestAdmissionLedgerShape` decodes the updated ledger.
- [x] 2.3 Update the `bfo` entry of `docs/admission-ledger.yaml` (premise: `:360-395`) as `design.md` D2-D3 state:
  15 `external_additions.relations`, 11 `relation_axioms.inverse_pairs`, 4 `sub_property_of` edges, deletion of the
  exclusion rows for `BearerOf`/`InheresIn` and `OccupiesTemporalRegion` (the `PartOf`/`HasPart` and
  `PrecedesTemporally` rows stay), and `admitted_terms` / `test_oracle` text naming them. Acceptance: task 3.2 and 3.3
  tests pass against it.

## 3. Catalog, pack test, and mutation evidence

- [x] 3.1 Add the 15 constants in a second relation block after `catalog/bfo/bfo.go:123`, in R1 order, doc comments
  `// Name (BFO_x, "label") …`, and extend the package comment at `:8-11` to name the BFO 2020 relation additions.
  Acceptance: `task build` and `task lint` pass; the constants equal the ledger rows (task 3.2).
- [x] 3.2 Extend `parityProblems` in `catalog/bfo/bfo_test.go` (premise: `:119-183`, exclusion check `:144-148`): each
  external relation is exported, an `owl:ObjectProperty`, not deprecated, with exactly its recorded label; each R1
  relation's declared inverse is admitted; the exclusion check stays name-based (`design.md` D3). Add seeded cases to
  `TestCatalogPortParitySensitivity` (`:194-217`): `BearerOf` at `BFO_0000053`, the proof that the old wrong IRIs are
  rejected; `Precedes`/`PrecededBy` swapped; a wrong label; a deprecated term via `oracle.LoadMerged(BFO2020, CCO22)`;
  `InheresIn` dropped. Acceptance: each seeded case is named.
- [x] 3.3 Add `TestRelationAxiomsMatchRelease` and its sensitivity twin to `catalog/bfo/bfo_test.go`: ledger
  `relation_axioms` equal the fixture's `owl:inverseOf` (unordered) and `rdfs:subPropertyOf` triples between admitted
  relations. Seeded: a dropped pair, an invented pair, a reversed subproperty edge. Acceptance: each is named.
- [x] 3.4 Add a `packs/diekm/diekm_test.go` check that every OBO-namespace `PredicateIRI` in the pack's projection
  steps equals `bfo.BearerOf` or `bfo.GenericallyDependsOn` (premise: `packs/diekm/diekm.yaml:58,71,75`), with a
  seeded `BFO_0000053` step that the check names. Acceptance: both tests pass.
- [x] 3.5 Run `task mutate:check` on the `design.md` D5 mutants and record each command and outcome in
  `openspec/changes/admit-bfo-bridge-relations/notes/mutate.md`. Acceptance: every mutant is killed or recorded as a
  survivor.

## 4. Docs and archive

- [x] 4.1 List the admitted BFO relations, inverse pairs, and subproperty edges in `docs/catalogs-and-packs.md`
  (premise: `:21-33` names relations only by `bfo.LocatedIn`). Acceptance: `task docs:check` passes.
- [ ] 4.2 Run `task verify` on a clean tree at the branch head. Acceptance: it exits 0 with no tracked-file change.
- [ ] 4.3 Archive this change into `openspec/changes/archive/` with `npx --no-install openspec archive
  admit-bfo-bridge-relations --yes` before the PR is marked ready. Acceptance: `openspec/specs/catalogs-and-packs` and
  `openspec/specs/catalog-provenance` carry the requirements and `task spec:check` passes.

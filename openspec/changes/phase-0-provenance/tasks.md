# Tasks: phase-0-provenance

Each task names the outcome that proves it. Command evidence is recorded on PR #5.

## 1. Provenance artifacts

- [x] 1.1 `docs/admission-ledger.yaml` pins SemStreams `8b99efe9c66a4faa4fa509f9f62cc6bad8392128`, SemEngine
      `v0.1.0-alpha.1` = `e60229334b8d8da3c0350e17802e2ea9631f91c8`, BFO-2020 `release-2024-01-29` (commit
      `5ed46876`), and CCO `v2.2` (owner ruling Q1); it has one entry for each of the 17 candidate catalogs and both
      packs
- [x] 1.2 `docs/catalog-ownership.md` lists, per candidate, SemEngine exported constants, serializer prefixes, and
      doc mentions with `file:line` at `e602293`
- [x] 1.3 The ledger's DIEKM entry states the namespace's registration status and the owner's DIEKM rulings
      (Q1–Q5, Q4b, Q10 of 2026-10-15)
- [x] 1.4 The SemStreams-pin source defects (BFO `BFO_0000062`/`BFO_0000132`, unverified relation IRIs, 22 non-CCO
      IRIs in the CCO namespace) and the #1 fixture corrections are posted on #1 for the owner (comment 6198595397)

## 2. Verification

- [x] 2.1 `task spec:check` and `task docs:check` pass
- [x] 2.2 `task verify` passes on a clean tree

## 3. Archive

- [ ] 3.1 `openspec archive phase-0-provenance` moves the spec to `openspec/specs/` as the last content commit

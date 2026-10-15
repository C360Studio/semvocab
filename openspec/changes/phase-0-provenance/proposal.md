# phase-0-provenance

## Why

Issue #1 Phase 0 needs provenance before any catalog is ported: an admission ledger with the SemStreams and SemEngine
pins, a disposition per candidate catalog, a symbol-level ownership table against SemEngine, BFO/CCO release choices,
and the DIEKM namespace's status. It also needs an OpenSpec change that fixes the scope, dependency order, and
observable acceptance tests of Phases 1–3. The agent setup (`phase-0-agent-setup`, archived) carried no catalog.

The owner ruled (2026-10-15) that this lands as its own PR, then Phase 1 (catalogs and the DIEKM pack) as a second
PR, then SemVocab tags `v0.1.0` for SemSPARQL. The SemEngine pin is the pre-release `v0.1.0-alpha.1`
(`e60229334b8d8da3c0350e17802e2ea9631f91c8`). It contains #235 (`7e2724a`) and not Stage 1.1 (semengine#256, PR #261).

## What Changes

- `docs/admission-ledger.yaml`: pins (SemStreams, SemEngine, BFO 2020, CCO 1.7), one entry per candidate catalog
  (17) and pack (`diekm`, `bfo-cco-core`), and phase gates.
- `docs/catalog-ownership.md`: overlap with SemEngine `e602293` by symbol, separating exported constants, serializer
  prefixes, and doc mentions.
- Capability spec `catalog-provenance`.
- `design.md`: release and namespace decisions, the adopter seam inventory, and the Phase 1–3 plan with acceptance
  tests (Phase 4 deferred).

## Non-goals

- No Go package under `catalog/`, `packs/`, or `shacl/`; no SemEngine `require` in `go.mod` (Phase 1).
- No change to SemEngine, SemStreams, or SemSPARQL; the source defects found there are reported to their owners.
- No claim that `https://w3id.org/c360/diekm#` is registered or certified.

## Impact

Docs, a YAML ledger, and OpenSpec. The ledger is not Markdown, so this is a code pull request under the protocol and
needs a cross-agent review. Phase 1 depends on this ledger's admitted-term lists.

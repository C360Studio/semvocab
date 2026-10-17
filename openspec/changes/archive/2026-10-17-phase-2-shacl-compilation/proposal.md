# phase-2-shacl-compilation

## Why

Issue #1 Phase 2 delivers the bounded ahead-of-time SHACL compiler: shapes in a published profile become a
SemEngine `PackSpec`, projection contracts, and a native validator for complete entity state, with no RDF or SHACL
engine in SemEngine's hot paths. Its gate is met: semengine#256 (pure `PackSpec`/`ProjectionSpec` validation) is in
the pinned `v0.1.0-alpha.2` (`docs/admission-ledger.yaml`, `phase_gates.phase-2: available`). The target state is
issue #1 "Phase 2", unchanged, and the archived Phase 0 design's "Phase 2" section.

## What Changes

- New public package `shacl`: `model.go` (shape model and mapping document), `parser.go` (Turtle, JSON, and YAML
  input), `compiler.go` (`Compile`), and `validator.go` (`ValidateCandidateState`). Importing it registers nothing.
- `internal/turtle` becomes a production reader for author-supplied shapes, with input-size and nesting caps; its
  grammar subset is unchanged (Q12).
- `packs/diekm` gains `diekm.mapping.yaml`, which binds the existing `diekm.shacl.ttl` shapes to the DIEKM pack's
  entity types, predicates, and contract groups (Q13, Q23).
- Vendored W3C fixtures: the SHACL vocabulary (`shacl.ttl`) and a subset of the SHACL Core test suite, pinned in the
  ledger (Q21).
- `pgregory.net/rapid v1.3.0` becomes a direct test dependency.
- `docs/shacl-compilation.md` publishes the support matrix, datatype table, regex subset, mapping rules, and the
  validation state boundary.
- Capability spec `shacl-compilation`; `catalog-provenance` gains a requirement for pinned test fixtures.

## Non-goals

- No projection expansion, export of reified graphs, or external SHACL validator (Phase 3).
- No rule translation (Phase 4): every `sh:rule` input is rejected.
- No registration of compiled bundles; a composition root calls `vocabulary.RegisterPack` itself.
- No command-line generator, no `cyber` catalog, no SHACL 1.2 terms.
- No change to SemEngine, SemStreams, or SemSPARQL.

## Impact

A new public package and a code pull request that needs a cross-agent review. Consumers named in #1 (semsource,
semconnect, semteams) gain an adapter-side validator; none calls it yet. `AGENTS.md` stops describing `shacl/` as
planned once the change lands.

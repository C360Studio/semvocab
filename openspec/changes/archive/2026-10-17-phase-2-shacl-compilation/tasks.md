# Tasks: phase-2-shacl-compilation

Each task names the outcome that proves it. Command evidence is recorded on the pull request. Design references are
to `design.md` in this change.

## 1. Design rulings and probes

- [x] 1.1 The owner rules Q12–Q24 on #1, and `design.md` records each ruling in place of its proposed default.
- [x] 1.2 Probes P1–P5 run, each with its command and result in `notes/1-probes.md`. A probe that contradicts the
      design changes `design.md` before the dependent task starts

## 2. Fixtures and dependencies

- [x] 2.1 The W3C SHACL vocabulary and the selected SHACL Core test-suite cases are vendored with their license
      notice. The ledger records them under `pins.fixtures`. `TestAdmissionLedgerShape` checks the section and fails
      when a field is removed. The tests check each sha256 before reading.
- [x] 2.2 `go.mod` requires `pgregory.net/rapid v1.3.0`, imported only from test files

## 3. Turtle reader

- [x] 3.1 `internal/turtle` serves production callers: its package comment drops "test-only"; nesting deeper than 64
      levels fails with a position; a caller-supplied base resolves relative IRIs, used only by the test-suite loader;
      existing tests still pass (D2, Q25). The 1 MiB cap is `shacl.ParseTurtle`'s, delivered with task 4.1
- [x] 3.2 Sensitivity tests show the nesting cap rejects one level over and accepts exactly at the limit, and (with
      4.1) `shacl.ParseTurtle` rejects one byte over 1 MiB and accepts exactly 1 MiB

## 4. Shape model and inputs

- [x] 4.1 `shacl.ParseTurtle`, `DecodeShapesJSON`, and `DecodeShapesYAML` build the model, refusing blank-node node
      shapes, relative IRIs, unknown fields, trailing documents, and non-string bounds (D3)
- [x] 4.2 `DecodeMappingJSON` and `DecodeMappingYAML` decode the mapping document strictly (D6).
- [x] 4.3 The DIEKM shapes in Turtle, JSON, and YAML decode to equal shape sets

## 5. Profile and compiler

- [x] 5.1 The profile table classifies all 183 vocabulary terms. `TestProfileClassifiesEveryTerm` passes, and fails
      when one classification is removed (D4, D15)
- [x] 5.2 Every rejected family fails compilation, naming construct, shape, and path, and all issues are reported
      together; `sh:rule` names the Phase 4 deferral. The W3C rejected-family cases pass with no adapter
- [x] 5.3 Cardinality, datatype table, numeric bounds with exact integers, and `sh:in` compile as D4, D8, and D11
      state. The #1 cases pass: omitted maximum, `maxCount 0`, `maxCount 1`, invalid bounds, empty and typed
      enumeration.
- [x] 5.4 The regex subset compiler accepts and rejects as D10 states, and agrees with the W3C pattern cases.
- [x] 5.5 Closed shapes follow D12, including the `rdf:type` rule.
- [x] 5.6 Mapping rules follow D5 and D6: explicit bindings, `iri` agreement, closure-checked targeting, datatype
      fill and conflict. Several shapes on one class keep all constraints. Conflicting mappings fail.
- [x] 5.7 Contracts follow D7: pattern, explicit modes and birth, conflicting modes fail, every constrained predicate
      in the contract, sorted names, the `Unwritable()` list. Each contract passes `ValidateShape`.
- [x] 5.8 `packs/diekm/diekm.mapping.yaml` binds the DIEKM shapes. The DIEKM bundle compiles, and its pattern matches
      `acme.edge.sensor.intel.platform.p1` and rejects `acme.edge.intel.other.platform.p1`.
- [x] 5.9 `TestSHACLCompilerOutputsValidPackAndContracts` passes. The compiled pack passes `(*PackSpec).Validate`.
      Global registry and prefix state are asserted unchanged after a successful and a failed compile. An unrelated
      registration between two compiles leaves the bundles equal (D14)

## 6. Validator

- [x] 6.1 `ValidateCandidateState` follows D13: input errors, targeting with `Targeted`, distinct-term counting,
      every applicable shape, sorted results, no registry reads
- [x] 6.2 The D9 value-term mapping passes the agreement test against `export.Serialize`, and the test fails when one
      mapping row is changed.
- [x] 6.3 The W3C supported-component cases pass through the reviewed adapter, and their expected results come from
      the vendored files
- [x] 6.4 The #1 state cases pass: reconcile `{A,B}` to `{B,C}`; append to `{A,B,C}`; mixed cardinalities do not
      choose the mode; birth missing a required value fails; a rebuilt partial update passes; an append past the
      stored maximum fails; the stale-append case shows the revision boundary

## 7. Determinism, fuzzing, and benchmarks

- [x] 7.1 The D16 Rapid properties pass under `task verify`. The pull request records the command, seed, and check
      count. The step-order sensitivity pair fails when two steps are swapped
- [x] 7.2 The five fuzz targets have committed seed corpora that `task verify` runs. A bounded run of each is
      recorded on the pull request with the Go version, and any crasher is committed as a corpus entry
- [x] 7.3 The benchmarks run with `-benchmem -count 10`, and the results table is in `docs/shacl-compilation.md`

## 8. Documentation

- [x] 8.1 `docs/shacl-compilation.md` covers D17: the support matrix, datatype table, regex subset, mapping and
      contract rules, validation API, revision boundary, the `sh:rule` deferral, reproduction commands, and
      benchmarks. `task docs:check` passes
- [x] 8.2 `AGENTS.md` describes `shacl/` as present, and the `internal/turtle` and `packs/diekm` package comments
      match the code

## 9. Verification

- [x] 9.1 `task verify` passes on a clean tree

## 10. Archive

- [x] 10.1 `openspec archive phase-2-shacl-compilation` syncs the specs as the last content commit

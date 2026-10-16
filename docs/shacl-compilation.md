# SHACL compilation

<!-- Task 8.1 of openspec/changes/phase-2-shacl-compilation writes the rest of this document (design D17):
     purpose, input formats, support matrix, datatype table, regex subset, mapping and contract rules, the
     validation API and its revision boundary, the sh:rule deferral, and reproduction commands. Task 7.3
     added only the benchmarks section below. -->

This document is incomplete. Task 8.1 of the `phase-2-shacl-compilation` change adds the rest. Until then, the
design is `openspec/changes/phase-2-shacl-compilation/design.md`.

## Benchmarks

These are measurements from one run on one machine. They are not guarantees and are not checked by any gate;
`task verify` does not run benchmarks. Expect other numbers on other hardware, Go versions, or load.

- **Command:** `go test ./shacl -run '^$' -bench . -benchmem -count 10`, from the repository root.
- **Code:** branch `claude/phase-2-shacl`, the commit that added this section.
- **Go:** go1.26.9, GOOS `darwin`, GOARCH `arm64`, GOMAXPROCS 12 (the default).
- **Machine:** Apple M3 Pro, 12 cores, a shared development host. The one-minute load average was 1.8 when the
  run started and 5.8 when it ended, so the slowest samples include contention.
- **Date:** 2026-10-16.

What each benchmark measures (`shacl/bench_test.go`):

- `BenchmarkCompile` times `Compile` only. The shapes, mapping, and pack are decoded before the timer starts.
  - `diekm`: the DIEKM pack, its two node shapes and two property shapes, and its mapping.
  - `shapes=N`: a synthetic pack of N predicates, with one property shape per predicate and ten property shapes
    per node shape. Every node shape targets one class, and one reconcile group holds every predicate. The
    property shapes rotate through four forms: `xsd:string` with a pattern and a three-member `sh:in`;
    `xsd:integer` with two bounds; `xsd:double` with one bound; and `xsd:boolean`. Each has `sh:maxCount 1`.
- `BenchmarkValidateCandidateState` times one `ValidateCandidateState` call. Each candidate conforms, so every
  constraint runs and no result is built.
  - `diekm`: an `intel.platform` entity with one altitude and one role.
  - `triples=N`: N triples, one per predicate, checked against the bundle compiled from `shapes=1000`. All 1,000
    property shapes run on every call, including those whose predicates have no triple. That is why ten triples
    cost far more than DIEKM's two.

Results over the 10 samples. Each column is the median; ns/op also gives the fastest and slowest sample. Across
samples, B/op varied by at most 177 bytes and allocs/op by at most 1.

| Benchmark | ns/op (median) | ns/op (min–max) | B/op | allocs/op |
| --- | ---: | ---: | ---: | ---: |
| `BenchmarkCompile/diekm` | 20,160 | 19,899–20,563 | 9,394 | 149 |
| `BenchmarkCompile/shapes=10` | 21,554 | 21,374–21,741 | 32,371 | 299 |
| `BenchmarkCompile/shapes=100` | 206,276 | 199,871–214,050 | 304,006 | 2,018 |
| `BenchmarkCompile/shapes=1000` | 2,205,978 | 2,172,153–2,362,969 | 3,589,413 | 19,587 |
| `BenchmarkValidateCandidateState/diekm` | 1,001 | 980–1,179 | 1,250 | 13 |
| `BenchmarkValidateCandidateState/triples=10` | 19,300 | 18,436–32,061 | 3,989 | 65 |
| `BenchmarkValidateCandidateState/triples=100` | 73,949 | 65,478–153,391 | 33,687 | 567 |
| `BenchmarkValidateCandidateState/triples=1000` | 551,464 | 519,709–684,225 | 418,428 | 5,604 |

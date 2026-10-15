# Notes: tasks 3.1–3.5 (packs)

## The altitude predicate name (tasks 3.2, 3.4)

The issue #1 fixture names the altitude predicate `intel.measurement.altitude_m`. SemEngine rejects it at the pin:
predicate segments follow `[a-z][a-z0-9-]*` (`validatePredicateSegment`, `vocabulary/predicate_contract.go` at
alpha.1 `:144-171`; unchanged at alpha.2), so `(*PackSpec).Validate` and `RegisterPack` fail with
`predicate "intel.measurement.altitude_m" is invalid: segment_character at segment 3`. A probe that changed only that
name validated and registered the whole pack. Tasks 3.2 and 3.4 were held until owner ruling Q11 (design P6) named
it `intel.measurement.altitude-m`; the pack, its JSON form, and the tests use that name, and `diekm.Register` was
added then. The DIEKM atomic-failure test keeps the old name as its invalid entry.

## Choices

- Registration entry points match the design's adopter seam with the alpha.2 option type, `vocabulary.PackOption`
  (`vocabulary/pack.go:123`). `packs.RegisterBFOCCOCore(opts ...vocabulary.PackOption)` puts
  `vocabulary.WithPrefixRegistrar(export.RegisterPrefixes)` first and the caller's options after, so the spec's
  prefix registrar is the default and a caller's `WithPrefixRegistrar` replaces it (`packConfig` keeps the last).
- `bfo-cco-core.yaml` sits at `packs/`, so its Go package is `packs` (`BFOCCOCoreSpec`, `RegisterBFOCCOCore`).
- `diekm.yaml` declares the `rdfs` prefix because a projection step uses `rdfs:label` and D6 requires every IRI to
  lie in a declared namespace. SemEngine's serializer already binds `rdfs` to the same namespace, so registering it
  is a no-op there (`export.RegisterPrefixes` accepts an identical binding).
- Superclass order is declared parents first, then breadth-first (D5): Vehicle, the local `BFO_0000030`, then the
  rest. The closure test compares sets.
- The declared parents the closure test starts from (Vehicle; Act of Observation and process) are the issue #1
  fixture's, with v2.2 IRIs, and live in the test; the DIEKM-local assertion is parsed from the ledger's
  `local_assertions`.
- `diekm.shacl.ttl` is initial: two `sh:NodeShape`s targeting the pack classes, with `sh:path`, `sh:datatype`
  (float is exported as `xsd:double`, `vocabulary/export/object.go` table), and `sh:maxCount`. `TestDIEKMShapes`
  keeps it in that subset and on pack IRIs. `diekm.Shapes()` returns a copy of the embedded bytes; Phase 2 is its
  consumer.
- The 3.5 tests run on the DIEKM pack itself and on `bfo-cco-core`: repeated registration of both; a rebinding of
  the `diekm` prefix that fails and changes nothing; a later DIEKM description that wins and is restored by
  re-registering the pack (D8); atomic failure of DIEKM copies with an unknown field, an invalid predicate, or a
  projection without a terminal value predicate; direct serialization of DIEKM's dotted predicates. The superclass
  union and class-IRI conflict cases run on test-local packs, because a union on DIEKM's keys could not be undone
  without restoring state by hand. Test-local keys carry a per-call counter, so `task test:repeat` (five runs in
  one process) never sees an earlier run's entries. JSON/YAML equivalence uses a hand-written
  `packs/diekm/testdata/diekm.json`.
- `TestDIEKMPackLoadsAndRegisters` writes its expected entity types, predicates, and projections out from the design
  (D5, Q2–Q4b, Q11) instead of reading them from the YAML, so it is an oracle independent of the pack file.

## Shown able to fail

- `TestPackSuperclassClosureSensitivity`: a removed ancestor (`BFO_0000004`), a CURIE superclass, an IRI outside
  every declared namespace, the deprecated `ont00001769`, and a `bfo` prefix are each named.
- `TestRegistryObservationSensitivity`: a successful registration changes all three things the atomic-failure tests
  observe (predicates, entity types, serializer prefixes).
- `TestDIEKMPackJSONYAMLEquivalence` checks that one altered superclass breaks equality.
- Seeded in `diekm.yaml` and reverted: `is measured by` changed to `ont00001905` fails
  `TestDIEKMPackLoadsAndRegisters` (projection) and `TestDIEKMPackJSONYAMLEquivalence`.

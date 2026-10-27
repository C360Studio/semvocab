# Notes: task 3.1 (swe-core pack)

## Implementation choices and SemEngine v0.1.0-alpha.2 probes

- **Entry points (`packs/packs.go`).** `SWECoreSpec()` decodes embedded `packs/swe-core.yaml` strictly through
  `vocabulary.LoadPackYAML` (`vocabulary/pack.go:159-178`: `KnownFields(true)` at `:164` + trailing-document check at
  `:170-176`) like `BFOCCOCoreSpec()`. `RegisterSWECore(opts ...)` prepends
  `vocabulary.WithPrefixRegistrar(export.RegisterPrefixes)` ahead of caller options (`vocabulary/pack.go:239-243`
  applies options in order so a caller-supplied `WithPrefixRegistrar` overrides the default).
- **Turtle vs JSON-LD compaction at SemEngine `v0.1.0-alpha.2` (`11da4721c2233a8e2b5b1d6236a97a44e06abbb6`).**
  `(*prefixMap).compact` (`vocabulary/export/prefix.go:121-145`) selects the longest matching namespace
  (`len(ns) > len(bestNS)`, breaking ties lexicographically by prefix), so `ssn-system`
  (`http://www.w3.org/ns/ssn/systems/`, 33 bytes) wins over `ssn` (`http://www.w3.org/ns/ssn/`, 25 bytes). In
  `export.Turtle` (`vocabulary/export/turtle.go:86-105`), both predicate IRIs and IRI objects are compacted
  (`ssn-system:hasSystemCapability ssn-system:Accuracy`). In `export.JSONLD` (`vocabulary/export/jsonld.go:34-98`),
  predicate keys are compacted via `pm.compact` (`"ssn-system:hasSystemCapability"`) and `@context` emits all five
  registered prefixes (`sosa`, `ssn`, `ssn-system`, `swe`, `sml`), while `jsonldObjectValue` emits full `@id` strings
  for resource objects (`{"@id": "http://www.w3.org/ns/ssn/systems/Accuracy"}`). `TestSWECorePackLoadsAndRegisters`
  verifies both serialized forms and asserts RDF graph equality between Turtle and JSON-LD via `graphFromJSONLD`.
- **Closure, idempotence, conflict, and atomic-failure coverage.** `packs_test.go` verifies `swe-core` prefixes against
  `pins.ontologies` (`sosa-2017`, `ssn-2017`, `ssn-system-2017`) and the ledger entries' `Namespace.IRI` (`swe`,
  `sensorml`, which `swe_test.go:84` and `sensorml_test.go:88` match against `pins.schemas[*].spec_base_uri + "/"`).
  `registration_test.go` asserts zero predicates and entity types on `swe-core` (`TestSWECorePackLoadsAndRegisters` at
  `:180-182`, owner ruling R2), exercises repeated `RegisterSWECore()` calls (`TestPackRepeatedRegistration`), rejects
  conflicting `ssn-system` prefix rebinding without mutating registry or serializer prefix state
  (`TestPackConflictingRegistration` / `"swe-core prefix rebinding fails atomically"`), and rejects a malformed
  `swe-core.yaml` copy carrying an unknown YAML field (`TestPackAtomicFailure` / `"swe-core: unknown YAML field"`),
  alongside the suite's DIEKM and generic invalid-predicate, missing-terminal-value-predicate, and
  invalid-superclass-IRI atomic-failure cases.

## Shown able to fail (seeded, observed, and reverted)

- `packs/swe-core.yaml`: changing `ssn` namespace from `http://www.w3.org/ns/ssn/` to `http://www.w3.org/ns/ssn#` fails
  `TestPackSuperclassClosure` (`swe-core: prefix ssn = http://www.w3.org/ns/ssn#, ledger pin http://www.w3.org/ns/ssn/`)
  and `TestSWECorePackLoadsAndRegisters`.

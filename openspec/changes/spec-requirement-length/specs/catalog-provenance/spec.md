## MODIFIED Requirements

### Requirement: One disposition per candidate

The ledger SHALL have exactly one entry for each of `actr`, `agnt`, `bfo`, `cco`, `cyber`, `dublincore`, `foaf`,
`geosparql`, `iana`, `owl`, `prov`, `rdf`, `rdfs`, `schema`, `skos`, `sosa`, `ssn`, and the `diekm` and
`bfo-cco-core` packs. Each entry SHALL carry one disposition from `use-semengine-symbol`, `thin-reexport`,
`new-content`, `port`, `extraction`, `deferred`.

#### Scenario: A catalog with no source

- **WHEN** a reader looks up `actr`
- **THEN** its disposition is `deferred` and its reason says no source exists at the SemStreams pin

## ADDED Requirements

### Requirement: Disposition fields

A `deferred` entry SHALL state a reason. A `port`, `extraction`, or `new-content` entry SHALL state its source,
admitted terms, license, and test oracle.

#### Scenario: A ported entry without a test oracle

- **WHEN** a `port`, `extraction`, or `new-content` entry has no `test_oracle`
- **THEN** `TestAdmissionLedgerShape` fails naming the entry and the missing field

# catalog-provenance Specification

## Purpose
Catalog provenance records where every admitted ontology term comes from, at an exact revision. It also records
what SemEngine already owns, so SemVocab neither copies SemEngine's constants nor admits unevidenced terms.

## Requirements

### Requirement: Pinned sources

`docs/admission-ledger.yaml` SHALL record the full 40-hex SemStreams revision. It SHALL record the SemEngine module
version together with the commit it resolves to. For each external ontology it SHALL record repository, tag, commit,
file path, sha256, version IRI, namespace, and license.

#### Scenario: Reading the SemEngine pin

- **WHEN** a reader looks up `pins.semengine` in the ledger
- **THEN** it names `v0.1.0-alpha.2` and `11da4721c2233a8e2b5b1d6236a97a44e06abbb6`, and lists semengine#256 as
  included

### Requirement: One disposition per candidate

The ledger SHALL have exactly one entry for each of `actr`, `agnt`, `bfo`, `cco`, `cyber`, `dublincore`, `foaf`,
`geosparql`, `iana`, `owl`, `prov`, `rdf`, `rdfs`, `schema`, `skos`, `sosa`, `ssn`, and the `diekm` and
`bfo-cco-core` packs. Each entry SHALL carry one disposition from `use-semengine-symbol`, `thin-reexport`,
`new-content`, `port`, `extraction`, `deferred`.

#### Scenario: A catalog with no source

- **WHEN** a reader looks up `actr`
- **THEN** its disposition is `deferred` and its reason says no source exists at the SemStreams pin

### Requirement: Admitted terms are evidenced

Every admitted term SHALL be declared in the named source at its recorded revision. A source symbol that the named
ontology release does not declare, or declares with a different meaning, SHALL be listed under
`excluded_source_symbols` with a reason and SHALL NOT be admitted.

#### Scenario: A SemStreams symbol whose IRI means the inverse

- **WHEN** a reader looks up `PrecedesTemporally` in the `bfo` entry
- **THEN** it is excluded because `BFO_0000062` is "preceded by" in BFO 2020

### Requirement: Ownership against SemEngine

`docs/catalog-ownership.md` SHALL list, for every candidate, SemEngine's exported constants, serializer prefix
mappings, and documentation mentions at the pinned commit as separate columns with `file:line`. A catalog whose terms
SemEngine exports SHALL NOT receive a SemVocab copy of those IRI strings.

#### Scenario: A namespace present only as a serializer prefix

- **WHEN** a reader looks up `foaf`
- **THEN** the table shows the `foaf` serializer prefix at `vocabulary/export/prefix.go:31` and no exported constant

### Requirement: Namespace honesty

The ledger SHALL state whether each C360-minted namespace is registered with its resolver. It SHALL NOT describe a
C360-local namespace as externally registered or certified.

#### Scenario: The DIEKM namespace

- **WHEN** a reader looks up the `diekm` pack
- **THEN** `namespace.registered` is `false` and the statement calls it a C360-local vocabulary

### Requirement: Phase gates

The ledger SHALL record each issue #1 phase's dependency gate and its status at the time of the last ledger change.

#### Scenario: Phase 2 gate

- **WHEN** a reader looks up `phase_gates.phase-2`
- **THEN** it names semengine#256 merged and tagged, with status `available`

### Requirement: Disposition fields

A `deferred` entry SHALL state a reason. A `port`, `extraction`, or `new-content` entry SHALL state its source,
admitted terms, license, and test oracle.

#### Scenario: A ported entry without a test oracle

- **WHEN** a `port`, `extraction`, or `new-content` entry has no `test_oracle`
- **THEN** `TestAdmissionLedgerShape` fails naming the entry and the missing field

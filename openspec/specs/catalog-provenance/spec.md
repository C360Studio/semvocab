# catalog-provenance Specification

## Purpose
Catalog provenance records where every admitted ontology or schema term comes from, at an exact revision. It also
records what SemEngine already owns, so SemVocab neither copies SemEngine's constants nor admits unevidenced terms.

## Requirements

### Requirement: Pinned sources

`docs/admission-ledger.yaml` SHALL record the 40-hex SemStreams revision and the SemEngine version with its 40-hex
commit. Each `pins.ontologies` entry SHALL record repository, tag or release, commit, file, sha256, version IRI or
created date, namespace, and license. Each `pins.schemas` entry SHALL record repository, tag, commit, standard,
edition, specification base URI, license, and files with sha256.

#### Scenario: Reading the SemEngine pin

- **WHEN** a reader looks up `pins.semengine` in the ledger
- **THEN** it names `v0.1.0-alpha.2` and `11da4721c2233a8e2b5b1d6236a97a44e06abbb6`, and lists semengine#256 as
  included

#### Scenario: Reading an unversioned W3C ontology pin

- **WHEN** a reader looks up `pins.ontologies.sosa-2017` in the ledger
- **THEN** it names `w3c/sdw`, `REC-vocab-ssn-20171019`, `cb31ee9de636a7101337815ba67ba9af4a7c1348`,
  `ssn/integrated/sosa.ttl`, its sha256, created date `2017-04-17`, `http://www.w3.org/ns/sosa/`, and its W3C/OGC
  licenses

#### Scenario: Reading an OGC 3.0 schema bundle pin

- **WHEN** a reader looks up `pins.schemas.swe-3.0` in the ledger
- **THEN** it names `opengeospatial/ogcapi-connected-systems`, `v1.0.0`,
  `8e03b236a049849f2ccc24b4fd9fdce5ff69bed2`, `OGC 24-014`, edition `3.0.0`, `http://www.opengis.net/spec/SWE/3.0`,
  the OGC Software License, and per-file sha256 digests

### Requirement: One disposition per candidate

The ledger SHALL have one entry for each of `actr`, `agnt`, `bfo`, `cco`, `cyber`, `dublincore`, `foaf`, `geosparql`,
`iana`, `owl`, `prov`, `rdf`, `rdfs`, `schema`, `skos`, `sosa`, `ssn`, `swe`, `sensorml`, and the `diekm`,
`bfo-cco-core`, and `swe-core` packs. Each entry SHALL carry one disposition from `use-semengine-symbol`,
`thin-reexport`, `new-content`, `port`, `extraction`, `deferred`.

#### Scenario: A catalog with no source

- **WHEN** a reader looks up `actr`
- **THEN** its disposition is `deferred` and its reason says no source exists at the SemStreams pin

#### Scenario: OGC 3.0 and W3C SOSA/SSN candidates

- **WHEN** a reader looks up `sosa`, `ssn`, `swe`, `sensorml`, and `swe-core`
- **THEN** each entry carries disposition `new-content` for Phase 1

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

The ledger SHALL state whether each C360-minted namespace or C360 RDF IRI convention over a non-RDF specification base
URI is registered or externally certified. It SHALL NOT describe a C360-local namespace or C360 RDF convention as an
externally registered, dereferenceable, or certified ontology.

#### Scenario: The DIEKM namespace

- **WHEN** a reader looks up the `diekm` pack
- **THEN** `namespace.registered` is `false` and the statement calls it a C360-local vocabulary

#### Scenario: OGC 3.0 RDF IRI convention honesty

- **WHEN** a reader looks up `swe` or `sensorml`
- **THEN** `namespace.registered` is `false` and the statement declares a C360 RDF convention over OGC's Clause 5
  specification base URI rather than an externally certified OGC OWL ontology

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

### Requirement: Pinned test fixtures

For each vendored external test fixture, `docs/admission-ledger.yaml` SHALL record, under `pins.fixtures`, the
repository, commit, file paths, sha256, and license. Tests SHALL check the sha256 before reading the fixture.

#### Scenario: The SHACL vocabulary

- **WHEN** a reader looks up the SHACL vocabulary in `pins.fixtures`
- **THEN** it names `w3c/data-shapes`, a full commit, `shacl/shacl.ttl`, its sha256, and the W3C Software and
  Document License

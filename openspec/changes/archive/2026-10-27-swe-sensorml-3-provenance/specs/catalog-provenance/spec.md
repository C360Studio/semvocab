# catalog-provenance

## MODIFIED Requirements

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

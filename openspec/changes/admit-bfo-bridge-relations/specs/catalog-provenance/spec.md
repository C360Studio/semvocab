# catalog-provenance

## MODIFIED Requirements

### Requirement: Admitted terms are evidenced

Every admitted term SHALL be declared in the named source at its recorded revision. A source symbol that the named
ontology release does not declare, or declares with a different meaning, SHALL be listed under
`excluded_source_symbols` with a reason and SHALL NOT be admitted at its source IRI. A release term admitted in its
place SHALL be named on that row under `admitted_instead` as `<GoName> <IRI local name>`.

#### Scenario: A SemStreams symbol whose IRI means the inverse

- **WHEN** a reader looks up `PrecedesTemporally` in the `bfo` entry
- **THEN** it is excluded because `BFO_0000062` is "preceded by" in BFO 2020

#### Scenario: An excluded SemStreams relation and its BFO 2020 replacement

- **WHEN** a reader looks up `OccupiesTemporalRegion BFO_0000132` in the `bfo` entry
- **THEN** it is excluded because `BFO_0000132` is "occurrent part of" in BFO 2020, and its row names
  `OccupiesTemporalRegion BFO_0000199` under `admitted_instead`

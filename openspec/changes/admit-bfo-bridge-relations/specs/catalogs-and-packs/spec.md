# catalogs-and-packs

## MODIFIED Requirements

### Requirement: Catalogs admit only ledger terms

`catalog/bfo`, `catalog/cco`, `catalog/sosa`, `catalog/ssn`, `catalog/swe`, and `catalog/sensorml` SHALL export
exactly the terms the admission ledger admits for them, with IRIs equal to the pinned release or schema-bundle
declaration, and SHALL NOT export any excluded source symbol. An excluded symbol is its name bound to its source IRI;
the name SHALL be exported only when its exclusion row lists it under `admitted_instead`, at that IRI.

#### Scenario: An excluded SemStreams symbol

- **WHEN** a reader looks for `PrecedesTemporally` in `catalog/bfo`
- **THEN** no such symbol exists

#### Scenario: Excluded consumer symbols in SOSA and SWE

- **WHEN** a reader looks for `SamplingFeature` or `HasLocation` in `catalog/sosa`, or `NilValue` / `PropNilValue` in
  `catalog/swe`
- **THEN** no such symbol exists

#### Scenario: Collision-free JSON Schema property constants

- **WHEN** a caller reads `swe.PropNamespace` and `sensorml.PropLink`
- **THEN** they equal `http://www.opengis.net/spec/SWE/3.0/namespace` and
  `http://www.opengis.net/spec/sensorML/3.0/link` without colliding with `swe.Namespace` or `sensorml.Link`

#### Scenario: A SemStreams name admitted at its BFO 2020 IRI

- **WHEN** a caller reads `bfo.BearerOf`, `bfo.InheresIn`, and `bfo.OccupiesTemporalRegion`
- **THEN** they end in `BFO_0000196`, `BFO_0000197`, and `BFO_0000199`, not SemStreams' `BFO_0000053`, `BFO_0000052`,
  and `BFO_0000132`

## ADDED Requirements

### Requirement: BFO relations are declared in the release

Every relation constant in `catalog/bfo` SHALL be declared `owl:ObjectProperty` and not `owl:deprecated` in the
pinned `bfo-2020` `bfo-core.ttl`. Each relation in the `bfo` entry's `external_additions.relations` SHALL carry
exactly the recorded `rdfs:label`, and every inverse the release declares for it SHALL also be admitted. A pack step
through a BFO relation SHALL use an admitted `catalog/bfo` relation IRI.

#### Scenario: The ruling R1 relation set

- **WHEN** a reader lists the relations in `external_additions.relations` of the `bfo` entry
- **THEN** they are the 15 object properties of ruling R1 on #31, each exported by `catalog/bfo` with its `bfo-2020`
  label

#### Scenario: Direction of temporal precedence

- **WHEN** a caller reads `bfo.Precedes` and `bfo.PrecededBy`
- **THEN** they end in `BFO_0000063` ("precedes") and `BFO_0000062` ("preceded by")

#### Scenario: A relation with no declared inverse

- **WHEN** the inverse check reaches `bfo.OccupiesTemporalRegion`
- **THEN** it passes with no partner, because `bfo-2020` declares no `owl:inverseOf` for `BFO_0000199`

#### Scenario: DIEKM projection steps

- **WHEN** the `diekm` pack's projection steps are read
- **THEN** every predicate IRI under the OBO namespace equals `bfo.BearerOf` or `bfo.GenericallyDependsOn`

### Requirement: BFO relation axioms follow the release

The `bfo` ledger entry SHALL record under `relation_axioms` every `owl:inverseOf` pair, in either direction, and
every `rdfs:subPropertyOf` edge that `bfo-2020` `bfo-core.ttl` asserts between two admitted relations, and no other.
An axiom that leaves the admitted set, such as `BFO_0000196` under `BFO_0000194`, is not recorded.

#### Scenario: Member parthood under continuant parthood

- **WHEN** a reader looks up `HasMemberPart` in `relation_axioms.sub_property_of`
- **THEN** it names `HasContinuantPart`, as `BFO_0000115 rdfs:subPropertyOf BFO_0000178` asserts

#### Scenario: Admitted temporal parts under occurrent parts

- **WHEN** a reader looks up `HasTemporalPart` and `TemporalPartOf` in `relation_axioms.sub_property_of`
- **THEN** they name `HasOccurrentPart` and `OccurrentPartOf`

#### Scenario: An inverse declared on one side only

- **WHEN** a reader looks up `BearerOf` in `relation_axioms.inverse_pairs`
- **THEN** it is paired with `InheresIn`, although only `BFO_0000196` carries `owl:inverseOf` in `bfo-core.ttl`

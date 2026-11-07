# catalogs-and-packs Specification

## Purpose
SemVocab's ontology and schema catalogs (`bfo`, `cco`, `sosa`, `ssn`, `swe`, and `sensorml`) and its embedded packs
carry only ledger-admitted terms, with hierarchies and superclass closures taken from the pinned ontology releases and
schema bundles, and register through SemEngine's pack API.

## Requirements

### Requirement: Catalogs admit only ledger terms

`catalog/bfo`, `catalog/cco`, `catalog/sosa`, `catalog/ssn`, `catalog/swe`, and `catalog/sensorml` SHALL export
exactly the terms the admission ledger admits for them, with IRIs equal to the pinned release or schema-bundle
declaration, and SHALL NOT export any excluded source symbol.

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

### Requirement: Hierarchy follows the release

For every admitted class in a catalog that declares a hierarchy, `Parents` SHALL return the transitive closure of
asserted named `rdfs:subClassOf` or JSON Schema `allOf` object parents in the pinned release, without `owl:Thing`, and
`IsA` SHALL agree with it. `catalog/sosa` SHALL export no hierarchy functions because `sosa.ttl` asserts zero
`rdfs:subClassOf` triples.

#### Scenario: A SemStreams hierarchy divergence

- **WHEN** `bfo.Parents(bfo.Site)` is called
- **THEN** it includes `BFO_0000141` (immaterial entity), as BFO 2020 asserts

#### Scenario: Multi-parent SOSA/SSN class

- **WHEN** `ssn.DirectParents(sosa.Sample)` is called
- **THEN** it returns `[]string{sosa.FeatureOfInterest, sosa.Result}`, and `ssn.IsA` is true for both

#### Scenario: Cross-catalog JSON Schema hierarchy

- **WHEN** `sensorml.Parents(sensorml.CharacteristicList)` is called
- **THEN** it returns `[]string{swe.AbstractSweIdentifiable, swe.AbstractSWE}`

### Requirement: Packs register full closures

Every entity type in an embedded pack SHALL list, in `superclasses`, the release closure of its declared parents plus
any pack `local_assertions` recorded in the ledger and their closures. Every IRI in a pack SHALL be an expanded
`http://` or `https://` IRI within a declared prefix namespace.

#### Scenario: The DIEKM platform

- **WHEN** the `diekm` pack is loaded
- **THEN** `intel.platform` lists the 7 superclass IRIs of the Vehicle closure plus `BFO_0000030`

#### Scenario: Dual SensorML 3.0 and SOSA/SSN closures

- **WHEN** `packs/sensorml` is loaded
- **THEN** `csapi.system` lists the SensorML closure of `sml:PhysicalSystem` and the SOSA/SSN closure of `ssn:System`
  via `local_assertions`

### Requirement: Packs register through SemEngine

The embedded packs SHALL load through SemEngine's `LoadPackYAML` and register through `RegisterPack` with
`WithPrefixRegistrar(export.RegisterPrefixes)`. A pack that fails to load or register SHALL leave registry and
serializer prefix state unchanged.

#### Scenario: An invalid entry

- **WHEN** a pack with an invalid entry is registered
- **THEN** registration fails and the registry getters and serializer prefixes return what they returned before

#### Scenario: `swe-core` compacts `ssn-system` ahead of `ssn`

- **WHEN** `packs.RegisterSWECore()` has succeeded and SemEngine exports an IRI under
  `http://www.w3.org/ns/ssn/systems/`
- **THEN** Turtle output compacts it with `ssn-system:` rather than `ssn:`

#### Scenario: swe-core and sensorml coexist without prefix conflict

- **WHEN** both `packs.RegisterSWECore()` and `sensorml.Register()` are executed
- **THEN** both succeed and serializer prefixes remain unambiguous

### Requirement: SensorML 3.0 and SOSA/SSN pack and SHACL bundle

`packs/sensorml` SHALL export `Spec`, `Register`, `Shapes`, and `Mapping`. The pack SHALL declare 5 `csapi.*` entity
types, 20 `sensorml.*` predicates with zero `csapi:` IRIs, and two 1-step `ProjectionSpec` chains. The SHACL and
mapping bundle SHALL define 5 open node shapes and 5 reconcile contracts without mutating SemEngine registries.

#### Scenario: Compiling packs/sensorml is side-effect free and matches BasePack

- **WHEN** `shacl.Compile` compiles `packs/sensorml` shapes and mapping against `Spec()`
- **THEN** compilation succeeds, produces identical pack predicates, and leaves SemEngine registries unmutated

#### Scenario: Procedure location prohibition and JSON position validation

- **WHEN** SHACL shapes validate `csapi.procedure` and `csapi.system` entities
- **THEN** `csapi.procedure` rejects `sml:position` via `sh:maxCount 0` while `csapi.system` accepts valid `rdf:JSON`
  position strings

### Requirement: BFO relations are declared in the release

Every relation constant in `catalog/bfo` SHALL be declared `owl:ObjectProperty` and not `owl:deprecated` in the
pinned `bfo-2020` `bfo-core.ttl`. Each relation in the `bfo` entry's `external_additions.relations` SHALL carry
exactly the recorded `rdfs:label`, and every inverse the release declares for it SHALL also be admitted. A pack step
through a BFO relation SHALL use an admitted `catalog/bfo` relation IRI.

#### Scenario: The ruling R1 relation set

- **WHEN** a reader lists the relations in `external_additions.relations` of the `bfo` entry
- **THEN** they are the 15 object properties of ruling R1 on #31, each exported by `catalog/bfo` with its `bfo-2020`
  label

#### Scenario: A SemStreams name at its BFO 2020 IRI

- **WHEN** a caller reads `bfo.BearerOf`, `bfo.InheresIn`, and `bfo.OccupiesTemporalRegion`
- **THEN** they end in `BFO_0000196` ("bearer of"), `BFO_0000197` ("inheres in"), and `BFO_0000199` ("occupies
  temporal region"), not SemStreams' `BFO_0000053`, `BFO_0000052`, and `BFO_0000132`, which `bfo-2020` does not declare
  with those labels

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

### Requirement: BFO-aligned packs anchor every entity type

Every embedded pack whose ledger entry references `bfo-2020` or `cco-2.2` SHALL give each entity type a class or
superclass that is, or is a catalog subclass of, one of `Process`, `ProcessBoundary`, `TemporalRegion`,
`SpatiotemporalRegion`, `MaterialEntity`, `Site`, `SpatialRegion`, `GenericallyDependentContinuant`, `Quality`, or
`RealizableEntity`. No entity type's closure SHALL contain two classes that a `bfo-2020` disjointness axiom
separates.

#### Scenario: Pack selection

- **WHEN** the check selects packs
- **THEN** it selects `bfo-cco-core` and `diekm`, and not `swe-core` or `sensorml`

#### Scenario: DIEKM is anchored

- **WHEN** `diekm` is checked
- **THEN** `intel.platform` anchors at `MaterialEntity`, `intel.observation` at `Process`, and no problem is named

#### Scenario: Missing bucket root

- **WHEN** `intel.platform`'s superclasses are only `IndependentContinuant`, `Continuant`, and `Entity`
- **THEN** the check names `intel.platform` as having no BFO bucket root

#### Scenario: Continuant and occurrent collide

- **WHEN** `bfo.Occurrent` is added to `intel.platform`'s superclasses
- **THEN** the check names `intel.platform` and the disjoint pair `BFO_0000002`, `BFO_0000003`

### Requirement: BFO projection hops follow the release

In a selected pack, each projection hop through one of the 17 BFO relations listed in the design SHALL have its
subject class within the relation's `bfo-2020` `rdfs:domain` and its node class within its `rdfs:range`. Step 0's
subject is each entity type the pack's mapping binds the predicate to. A hop from an independent to a specifically
dependent continuant SHALL use `bearer of`; the reverse SHALL use `inheres in`.

#### Scenario: DIEKM chains are sound

- **WHEN** `diekm`'s projections are checked
- **THEN** its two `bearer of` hops and one `generically depends on` hop pass and no problem is named

#### Scenario: Unanchored quality

- **WHEN** step 0 of `intel.measurement.altitude-m` uses `cco:ont00001904` instead of `bearer of`
- **THEN** the check names that predicate's step 0 as reaching an SDC without `bfo.BearerOf`

#### Scenario: Unanchored role

- **WHEN** step 0 of `intel.role.assigned` uses `bfo.ParticipatesIn`
- **THEN** the check names that predicate's step 0 as reaching an SDC without `bfo.BearerOf`

#### Scenario: A carrier hop between information entities

- **WHEN** step 2 of `intel.measurement.altitude-m` is `is carrier of` to `cco.InformationContentEntity`
- **THEN** the check names step 2: the subject is outside the domain and the node is inside the range only

#### Scenario: A process bearer

- **WHEN** the mapping binds `intel.role.assigned` to `intel.observation`
- **THEN** the check names `intel.observation` as outside the domain of `BFO_0000196`

### Requirement: Process connectivity is opt-in

For a pack opted in by name, every continuant entity type SHALL be the subject of some predicate whose projection
reaches a node class under `bfo.Process`. No embedded pack is opted in.

#### Scenario: DIEKM opted in

- **WHEN** `diekm` is opted in unchanged
- **THEN** the diagnostic names `intel.platform`

#### Scenario: DIEKM with a realization step

- **WHEN** `intel.role.assigned` gains a step `has realization` to `cco.Act` and `diekm` is opted in
- **THEN** the diagnostic names nothing

# catalogs-and-packs

## ADDED Requirements

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

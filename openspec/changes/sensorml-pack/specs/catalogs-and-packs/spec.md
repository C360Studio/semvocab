# catalogs-and-packs

## MODIFIED Requirements

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

## ADDED Requirements

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

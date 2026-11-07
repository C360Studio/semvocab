# Catalogs and packs

A catalog is a Go package of constants for the terms of one ontology, plus that ontology's subclass hierarchy. A pack
is a data file (YAML) that tells SemEngine's vocabulary registry which entity types and predicates exist and how they
map to ontology IRIs. Catalogs are for your Go code; packs are for the registry. A pack's YAML holds IRIs as plain
strings and does not import a catalog.

Terms used below:

- **IRI**: a global name for a term, written like a URL (`http://purl.obolibrary.org/obo/BFO_0000001`).
- **Ontology**: a published set of IRIs with stated meanings. BFO, CCO, SOSA, SSN, and the SWE Common 3.0 / SensorML 3.0
  schema catalogs are the ones used here.
- **Entity type**: a SemEngine category of entity, written `<domain>.<type>`, such as `intel.platform`.
- **Predicate**: a SemEngine property name of three dotted parts, such as `intel.role.assigned`.
- **Prefix**: a short name for a namespace (`obo` for `http://purl.obolibrary.org/obo/`), used when RDF is written out.

## Catalogs

Importing a catalog registers nothing.

### `catalog/bfo`

Constants for the Basic Formal Ontology 2020 (BFO) classes and relations that were admitted, such as `bfo.Entity`,
`bfo.Quality`, `bfo.Process`, and `bfo.LocatedIn`. `bfo.Namespace` is `http://purl.obolibrary.org/obo/`, the base
shared by all ontologies in the OBO Foundry (a group of biology and science ontologies that follow common naming
rules).

Three functions answer hierarchy questions for classes:

- `SubClassOf(iri)` returns the direct parent. `ok` is false for `bfo.Entity` (the root), for relations, and for IRIs
  outside the catalog.
- `Parents(iri)` returns every ancestor, nearest first, ending at `bfo.Entity`.
- `IsA(child, parent)` is true when `child` is `parent` or below it.

Relations are OWL object properties: they link two entities rather than name a kind of entity. Ten relations come from
SemStreams (`OccursIn`, `ParticipatesIn`, `HasParticipant`, `RealizedIn`, `Realizes`, `LocatedIn`, `LocationOf`,
`ExistsAt`, `TemporalPartOf`, `HasTemporalPart`). Fifteen more come straight from BFO 2020 (ruling R1 on #31):

| Constant | IRI | BFO 2020 label |
| --- | --- | --- |
| `bfo.BearerOf` | `BFO_0000196` | bearer of |
| `bfo.InheresIn` | `BFO_0000197` | inheres in |
| `bfo.GenericallyDependsOn` | `BFO_0000084` | generically depends on |
| `bfo.IsCarrierOf` | `BFO_0000101` | is carrier of |
| `bfo.IsConcretizedBy` | `BFO_0000058` | is concretized by |
| `bfo.Concretizes` | `BFO_0000059` | concretizes |
| `bfo.OccupiesTemporalRegion` | `BFO_0000199` | occupies temporal region |
| `bfo.HasOccurrentPart` | `BFO_0000117` | has occurrent part |
| `bfo.OccurrentPartOf` | `BFO_0000132` | occurrent part of |
| `bfo.ContinuantPartOf` | `BFO_0000176` | continuant part of |
| `bfo.HasContinuantPart` | `BFO_0000178` | has continuant part |
| `bfo.HasMemberPart` | `BFO_0000115` | has member part |
| `bfo.MemberPartOf` | `BFO_0000129` | member part of |
| `bfo.PrecededBy` | `BFO_0000062` | preceded by |
| `bfo.Precedes` | `BFO_0000063` | precedes |

`bfo.Precedes` means "comes before": `a precedes b`. `bfo.PrecededBy` is the reverse.

BFO 2020 declares these relations to be inverses of each other (`a R b` exactly when `b R' a`):

- `RealizedIn` / `Realizes`, `ParticipatesIn` / `HasParticipant`, `LocationOf` / `LocatedIn`
- `HasTemporalPart` / `TemporalPartOf`, `HasOccurrentPart` / `OccurrentPartOf`
- `ContinuantPartOf` / `HasContinuantPart`, `HasMemberPart` / `MemberPartOf`
- `BearerOf` / `InheresIn`, `GenericallyDependsOn` / `IsCarrierOf`, `IsConcretizedBy` / `Concretizes`
- `PrecededBy` / `Precedes`

It also declares four sub-properties (every link of the first kind is also a link of the second):

- `HasMemberPart` under `HasContinuantPart`, and `MemberPartOf` under `ContinuantPartOf`
- `HasTemporalPart` under `HasOccurrentPart`, and `TemporalPartOf` under `OccurrentPartOf`

`OccupiesTemporalRegion` has no inverse in BFO 2020. The catalog exports no function for inverses or sub-properties.
The admission ledger (`docs/admission-ledger.yaml`, entry `bfo`, `relation_axioms`) records them, and tests check that
record against the BFO 2020 file.

### `catalog/cco`

Constants for the admitted classes and object properties of the Common Core Ontologies v2.2 (CCO), such as
`cco.Vehicle`, `cco.Sensor`, `cco.Altitude`, and `cco.IsMeasuredBy`. CCO v2 IRIs are opaque (`ont00000787`), so each
constant's comment gives the term's label. `cco.SubClassOf`, `Parents`, and `IsA` work the same way, and continue
into BFO when a chain leaves CCO:

```go
package main

import (
    "fmt"

    "github.com/c360studio/semvocab/catalog/bfo"
    "github.com/c360studio/semvocab/catalog/cco"
)

func main() {
    fmt.Println(cco.Altitude)
    fmt.Println(cco.IsA(cco.Altitude, bfo.Quality))
    fmt.Println(cco.IsA(cco.Altitude, bfo.Process))
    for _, p := range cco.Parents(cco.Altitude) {
        fmt.Println(p)
    }
    parent, ok := bfo.SubClassOf(bfo.Entity)
    fmt.Printf("%q %v\n", parent, ok)
}
```

Output:

```text
https://www.commoncoreontologies.org/ont00000787
true
false
https://www.commoncoreontologies.org/ont00000967
https://www.commoncoreontologies.org/ont00001367
https://www.commoncoreontologies.org/ont00001202
http://purl.obolibrary.org/obo/BFO_0000019
http://purl.obolibrary.org/obo/BFO_0000020
http://purl.obolibrary.org/obo/BFO_0000002
http://purl.obolibrary.org/obo/BFO_0000001
"" false
```

Both hierarchies answer only for the admitted classes. A class that is real in the ontology but not admitted here is
"outside the catalog", and the functions return nothing for it. The sources, releases, and licenses are in the
package comments and in `docs/admission-ledger.yaml`.

### `catalog/sosa`

Constants for the W3C Sensor, Observation, Sample, and Actuator (SOSA) 2017 Recommendation terms, such as
`sosa.Observation`, `sosa.Sensor`, `sosa.Sample`, `sosa.MadeBySensor`, `sosa.HasSimpleResult`, and `sosa.ResultTime`.
`sosa.Namespace` is `http://www.w3.org/ns/sosa/`.

`catalog/sosa` exports constants only and declares no hierarchy functions. Upstream `sosa.ttl` uses lightweight
`schema:domainIncludes` and `schema:rangeIncludes` annotations rather than formal OWL class axioms, and asserts zero
`rdfs:subClassOf` triples. Formal subclass relationships over SOSA terms are defined in `catalog/ssn`.

### `catalog/ssn`

Constants for the W3C Semantic Sensor Network (`ssn.Namespace`, `http://www.w3.org/ns/ssn/`) and SSN System
Capabilities (`ssn.SystemsNamespace`, `http://www.w3.org/ns/ssn/systems/`) 2017 Recommendation terms.

SSN introduces multiple inheritance: for instance, `sosa.Sample` has two direct superclasses (`sosa.FeatureOfInterest`
and `sosa.Result`), and `ssn.Condition` and `ssn.SystemCapability` are both subclasses of `ssn.Property`. Because a
class may have more than one parent, `catalog/ssn` exports `DirectParents` instead of a single-parent `SubClassOf`:

- `DirectParents(iri)` returns a fresh copy of the direct named `rdfs:subClassOf` superclasses in document order (empty
  for root classes, properties, and unknown IRIs).
- `Parents(iri)` returns the breadth-first transitive closure of ancestors across SOSA, SSN, and `ssn-system`, without
  duplicates or `owl:Thing`.
- `IsA(child, parent)` reports whether `child` is `parent` or a subclass of `parent` in the integrated SOSA, SSN, and
  `ssn-system` hierarchy.

### `catalog/swe` and `catalog/sensorml`

Constants for the OGC SWE Common Data Model 3.0 (`swe.Namespace`, `http://www.opengis.net/spec/SWE/3.0/`) and OGC
SensorML 3.0 (`sensorml.Namespace`, `http://www.opengis.net/spec/sensorML/3.0/`).

Three conventions govern these packages:

1. **Naming**: object classes use `PascalCase` (`swe.Quantity`, `sensorml.PhysicalSystem`), while JSON property names
   use `Prop<PascalCase>` (`swe.PropUOM`, `sensorml.PropPosition`). This prevents collisions between class and property
   names that differ only by casing in JSON (such as class `CharacteristicList` vs property `characteristics`, or
   property `label` vs `Label`), and ensures `swe.PropNamespace` does not collide with `swe.Namespace`.
2. **Namespace honesty**: OGC 24-014 and OGC 23-000 publish JSON Schemas rather than an OWL ontology. Both
   `http://www.opengis.net/spec/SWE/3.0/` and `http://www.opengis.net/spec/sensorML/3.0/` are local C360 RDF
   conventions formed by appending `/` to OGC's specification base URIs (`http://www.opengis.net/spec/SWE/3.0` and
   `http://www.opengis.net/spec/sensorML/3.0`); they are not certified OGC ontology IRIs and do not resolve on the web.
3. **Hierarchy**: single-inheritance `allOf` chains in the 3.0 JSON Schemas, exposed through
   `SubClassOf(iri) (string, bool)`, `Parents(iri) []string`, and `IsA(child, parent) bool`. SensorML process and
   system classes descend from `sensorml.DescribedObject` (a root class in SensorML that declares no `allOf` parent),
   so their ancestry ends within SensorML: `sensorml.IsA(sensorml.PhysicalSystem, swe.AbstractSWE)` is false. Seven
   metadata classes (`CharacteristicList`, `CapabilityList`, `DerivedProperty`, `Event`, `ObservableProperty`,
   `SpatialFrame`, and `TemporalFrame`) extend `swe.AbstractSweIdentifiable`; `sensorml`'s hierarchy continues into
   `swe` across those classes up to `swe.AbstractSWE`.

This program demonstrates multi-parent SSN lookup, SensorML process hierarchy, and the cross-catalog link into SWE:

```go
package main

import (
    "fmt"

    "github.com/c360studio/semvocab/catalog/sensorml"
    "github.com/c360studio/semvocab/catalog/sosa"
    "github.com/c360studio/semvocab/catalog/ssn"
    "github.com/c360studio/semvocab/catalog/swe"
)

func main() {
    for _, p := range ssn.DirectParents(sosa.Sample) {
        fmt.Println(p)
    }
    fmt.Println(sensorml.IsA(sensorml.PhysicalSystem, swe.AbstractSWE))
    for _, p := range sensorml.Parents(sensorml.PhysicalSystem) {
        fmt.Println(p)
    }
    fmt.Println(sensorml.IsA(sensorml.CharacteristicList, swe.AbstractSWE))
    for _, p := range sensorml.Parents(sensorml.CharacteristicList) {
        fmt.Println(p)
    }
}
```

Output:

```text
http://www.w3.org/ns/sosa/FeatureOfInterest
http://www.w3.org/ns/sosa/Result
false
http://www.opengis.net/spec/sensorML/3.0/AbstractPhysicalProcess
http://www.opengis.net/spec/sensorML/3.0/AbstractProcess
http://www.opengis.net/spec/sensorML/3.0/DescribedObject
true
http://www.opengis.net/spec/SWE/3.0/AbstractSweIdentifiable
http://www.opengis.net/spec/SWE/3.0/AbstractSWE
```

## Packs

### `bfo-cco-core`

`packs.BFOCCOCoreSpec()` decodes the embedded `packs/bfo-cco-core.yaml`. The pack declares two prefixes and nothing
else: `obo` for the BFO namespace and `cco` for the CCO namespace. Register it so SemEngine's RDF output writes
`obo:BFO_0000001` rather than the full IRI. The prefix is `obo`, not `bfo`, because BFO IRIs live under the namespace
every OBO ontology shares.

### `swe-core`

`packs.SWECoreSpec()` decodes the embedded `packs/swe-core.yaml` (`swe-core` `1.0.0`). The pack declares five prefixes
and no entity types or predicates:

- `sosa` for `http://www.w3.org/ns/sosa/`
- `ssn` for `http://www.w3.org/ns/ssn/`
- `ssn-system` for `http://www.w3.org/ns/ssn/systems/`
- `swe` for `http://www.opengis.net/spec/SWE/3.0/`
- `sml` for `http://www.opengis.net/spec/sensorML/3.0/`

SemEngine matches serializer prefixes longest-namespace-first (`vocabulary/export/prefix.go:121-145` at
`v0.1.0-alpha.2`, tested in `TestSWECorePackLoadsAndRegisters`), so `http://www.w3.org/ns/ssn/systems/` terms are
compacted under `ssn-system:` rather than the shorter `ssn:`.

### `diekm`

`packs/diekm` holds a pack for the DIEKM vocabulary, a C360-local vocabulary built over BFO and CCO. It is not an
externally registered ontology.

| Kind | Name | Meaning |
| --- | --- | --- |
| Entity type | `intel.platform` | A collection platform; a CCO Vehicle and a BFO object |
| Entity type | `intel.observation` | An observation act; a CCO Act of Observation |
| Predicate | `intel.role.assigned` | A string; written as the entity being bearer of a role whose label is the value |
| Predicate | `intel.measurement.altitude-m` | A float in meters; written through an Altitude, a measurement, and an information-bearing entity that holds the value |

The two predicates carry a projection: a chain of steps through intermediate nodes. One flat value
(`intel.measurement.altitude-m = 120.5`) becomes the chain entity → Altitude → measurement → information-bearing
entity, which carries the value, when exported as RDF.
The steps are in `packs/diekm/diekm.yaml`.

`diekm` ships three embedded files that belong together:

- the pack (`diekm.yaml`), from `diekm.Spec()`;
- the SHACL shapes (`diekm.shacl.ttl`), from `diekm.Shapes()`: the platform has at most one altitude, a string role,
  and the observation has no constraints;
- the mapping (`diekm.mapping.yaml`), from `diekm.Mapping()`, which says which shape class is which entity type, which
  shape path is which predicate, and that `intel.platform` gets one contract with an altitude group and a role group.

`shacl.Compile` takes all three. [shacl-compilation.md](shacl-compilation.md) covers the mapping and contracts.
`Shapes()` and `Mapping()` return copies, so changing the returned slice does not change the embedded data.

### BFO alignment

Packs over BFO 2020 and CCO v2.2 write their superclass lists and projection chains by hand. Tests in
`packs/bfo_soundness_test.go` check them against the pinned `bfo-core.ttl` ahead of time. They run on every embedded
pack whose ledger entry references `bfo-2020` or `cco-2.2` (today `bfo-cco-core` and `diekm`); `swe-core` and
`sensorml` are not checked. No exported function runs these checks (ruling R1 on #32). Classes are classified with
`cco.IsA`, which falls back to `bfo.IsA`, so a class neither catalog knows fails the check.

**Buckets.** Every entity type's class or superclasses must reach one of seven buckets. The buckets are issue #32's
grouping, not a BFO 2020 term:

| Bucket | BFO classes |
| --- | --- |
| What happens | `Process`, `ProcessBoundary` |
| When | `TemporalRegion`, `SpatiotemporalRegion` |
| Who or what | `MaterialEntity` |
| Where | `Site`, `SpatialRegion` |
| How we know | `GenericallyDependentContinuant` (information content) |
| How it is | `Quality` |
| Why | `RealizableEntity` (roles, dispositions, functions) |

A continuant fiat boundary (`BFO_0000140`, `142`, `146`, `147`) reaches no bucket and is rejected.

**Disjointness.** No entity type's class and superclasses may fall under two classes that BFO 2020 declares disjoint:
the `owl:disjointWith` axioms and the `owl:AllDisjointClasses` lists in `bfo-core.ttl` (26 pairs, such as `Continuant`
and `Occurrent`, or `IndependentContinuant` and `SpecificallyDependentContinuant`).

**Projection hops.** A hop is one projection step: from the entity (step 0) or the previous step's node class, through
the step's predicate, to its node class. Step 0 starts at each entity type the pack's SHACL mapping binds the predicate
to, so a BFO pack with projections must ship a mapping. A hop through one of these 17 relations must start inside the
relation's `rdfs:domain` and end inside its `rdfs:range`, read from `bfo-core.ttl` ("IC" is independent continuant,
"SDC" specifically dependent continuant, "GDC" generically dependent continuant):

| Relation | From | To |
| --- | --- | --- |
| `BearerOf` (`196`) | IC, not a `SpatialRegion` | SDC |
| `InheresIn` (`197`) | SDC | IC, not a `SpatialRegion` |
| `GenericallyDependsOn` (`084`) | GDC | IC, not a `SpatialRegion` |
| `IsCarrierOf` (`101`) | IC, not a `SpatialRegion` | GDC |
| `Realizes` (`055`) | `Process` | `RealizableEntity` |
| `RealizedIn` (`054`, "has realization") | `RealizableEntity` | `Process` |
| `Concretizes` (`059`) | `Process` or SDC | GDC |
| `IsConcretizedBy` (`058`) | GDC | `Process` or SDC |
| `HasMemberPart` (`115`), `MemberPartOf` (`129`) | `MaterialEntity` | `MaterialEntity` |
| `ContinuantPartOf` (`176`), `HasContinuantPart` (`178`) | `Continuant` | `Continuant` |
| `HasOccurrentPart` (`117`), `OccurrentPartOf` (`132`) | `Occurrent` | `Occurrent` |
| `OccupiesTemporalRegion` (`199`) | `Process` or `ProcessBoundary` | `TemporalRegion` |
| `Precedes` (`063`), `PrecededBy` (`062`) | `Occurrent` | `Occurrent` |

"Not a `SpatialRegion`" is decided over the declared superclasses: a class passes when none of them is under
`SpatialRegion`. Other predicates, such as `cco:ont00001904` (is measured by) or `bfo.ParticipatesIn`, are not checked
against a domain or range.

**Anchoring.** Whatever the predicate, a hop from an IC to an SDC must use `bfo.BearerOf`, and a hop from an SDC to an
IC must use `bfo.InheresIn`. CCO sub-properties of these, such as `ont00001954` (has capability), are rejected. In
`diekm`, `intel.role.assigned` is the platform bearer of a role, and `intel.measurement.altitude-m` starts with the
platform bearer of an Altitude.

**Bearer-less metrics.** A score, confidence, or status with no identified material bearer is modeled as a
`cco.InformationContentEntity` (a GDC), not as a `bfo.Quality`: a quality needs a bearer, and the anchoring rule
rejects a quality reached from an IC without `bearer of`.

**Process connectivity.** An opt-in diagnostic names each continuant entity type that is the subject of no projection
reaching a `bfo.Process`. No embedded pack is opted in: `diekm` would fail it, since neither of `intel.platform`'s
predicates reaches a process.

### `sensorml`

`packs/sensorml` holds the vocabulary pack for the five OGC API — Connected Systems Part 1 (`OGC 23-001`) resources
over OGC SensorML 3.0, OGC SWE Common 3.0, and W3C SOSA/SSN 2017.

| Entity type | Target class IRI | Superclasses | Meaning |
| --- | --- | --- | --- |
| `csapi.system` | `sml:PhysicalSystem` | `sml:AbstractPhysicalProcess`, `ssn:System`, `sml:AbstractProcess`, `sml:DescribedObject` | Physical or virtual system |
| `csapi.procedure` | `sml:SimpleProcess` | `sml:AbstractProcess`, `sosa:Procedure`, `sml:DescribedObject` | Observation, sampling, or actuation procedure |
| `csapi.deployment` | `sml:Deployment` | `sml:DescribedObject`, `ssn:Deployment` | System deployment |
| `csapi.samplingfeature` | `sosa:Sample` | `sosa:FeatureOfInterest`, `sosa:Result` | Sampling feature |
| `csapi.property` | `sml:DerivedProperty` | `swe:AbstractSweIdentifiable`, `sosa:ObservableProperty`, `swe:AbstractSWE`, `ssn:Property` | Derived observable or controllable property |

Entity types retain the `csapi.<type>` key so API paths map directly to domain entity types while bridging both
SensorML 3.0 and SOSA/SSN class hierarchies through dual superclasses (recorded as `local_assertions` in
`docs/admission-ledger.yaml`).

The pack declares nine prefixes: `rdf`, `rdfs`, `dc`, `skos`, `sosa`, `ssn`, `ssn-system`, `swe`, and `sml`. It uses
`dc` for `http://purl.org/dc/terms/` to match SemEngine's pre-registered prefix and coexist cleanly with `swe-core`.

The pack defines 20 `sensorml.*` predicates, including inverse entity relationships `sensorml.system.hosts`
(`sosa:hosts`) and `sensorml.component.is-hosted-by` (`sosa:isHostedBy`), process attachment
`sensorml.process.attached-to` (`sml:attachedTo`), and `sensorml.process.position` (`sml:position`) with data type
`json` (representing GeoJSON, SensorML Pose, or SWE position structures).

Two predicates carry 1-step projections:

- `sensorml.capability.value` projects through `ssn-system:hasSystemCapability` to intermediate node class
  `sml:CapabilityList` and terminal value predicate `swe:value`.
- `sensorml.characteristic.value` projects through `ssn:hasProperty` to intermediate node class
  `sml:CharacteristicList` and terminal value predicate `swe:value`.

`packs/sensorml` ships three embedded files that belong together:

- the pack (`sensorml.yaml`), from `sensorml.Spec()`;
- the SHACL shapes (`sensorml.shacl.ttl`), from `sensorml.Shapes()`: five open node shapes (omitting `sh:closed` so
  framework predicates pass validation). It prohibits `sml:position` on `csapi.procedure` via `sh:maxCount 0` (per OGC
  23-001 `/req/procedure/location`), and enforces `rdf:JSON` datatype on `sml:position` where position is allowed;
- the mapping (`sensorml.mapping.yaml`), from `sensorml.Mapping()`, which binds each shape class to its entity type
  and path to its predicate. All five entity types define `reconcile` contracts with `birth: [sensorml.process.type]`
  and a single `representation` group covering their constrained predicates. Because all five entity types receive
  contracts, `bundle.Unwritable()` is empty (`[]`).

Both `swe` (`http://www.opengis.net/spec/SWE/3.0/`) and `sml`
(`http://www.opengis.net/spec/sensorML/3.0/`) bind local C360 RDF conventions over OGC Clause 5 specification base
URIs rather than certified OGC OWL ontologies. Working-Draft OGC CS API Part 2 (`OGC 23-002`) dynamic stream,
observation, command, and feasibility resources and `csapi:` IRIs are excluded.

## Registering

Each pack has a function that registers it with SemEngine's `vocabulary.RegisterPack`:

- `packs.RegisterBFOCCOCore(opts ...vocabulary.PackOption) error`
- `packs.RegisterSWECore(opts ...vocabulary.PackOption) error`
- `diekm.Register(opts ...vocabulary.PackOption) error`
- `sensorml.Register(opts ...vocabulary.PackOption) error`

All four pass `vocabulary.WithPrefixRegistrar(export.RegisterPrefixes)`, so SemEngine's serializer learns the
prefixes. Options you pass come after it; a later `WithPrefixRegistrar` replaces it. Call them from your program's
start-up code. This program registers them and reads the result back:

```go
package main

import (
    "fmt"
    "log"

    "github.com/c360studio/semengine/vocabulary"
    "github.com/c360studio/semvocab/packs"
    "github.com/c360studio/semvocab/packs/diekm"
)

func main() {
    if err := packs.RegisterBFOCCOCore(); err != nil {
        log.Fatal(err)
    }
    if err := diekm.Register(); err != nil {
        log.Fatal(err)
    }

    meta := vocabulary.GetEntityTypeMetadata("intel.platform")
    fmt.Println(meta.ClassIRI)
    fmt.Println(vocabulary.ListRegisteredEntityTypes())
    fmt.Println(vocabulary.GetPredicateMetadata("intel.measurement.altitude-m").DataType)
}
```

Output:

```text
https://w3id.org/c360/diekm#IntelligencePlatform
[intel.observation intel.platform]
float
```

The registry is process-wide, so this changes it for every caller in the program.

This program registers `packs/sensorml`, verifies dual inheritance against `catalog/ssn`, and compiles its shapes:

```go
package main

import (
    "bytes"
    "fmt"
    "log"
    "slices"

    "github.com/c360studio/semengine/vocabulary"
    "github.com/c360studio/semvocab/catalog/ssn"
    "github.com/c360studio/semvocab/packs/sensorml"
    "github.com/c360studio/semvocab/shacl"
)

func main() {
    if err := sensorml.Register(); err != nil {
        log.Fatal(err)
    }

    meta := vocabulary.GetEntityTypeMetadata("csapi.system")
    fmt.Println(meta.ClassIRI)
    fmt.Println(slices.Contains(meta.Superclasses, ssn.System))

    spec, err := sensorml.Spec()
    if err != nil {
        log.Fatal(err)
    }
    shapes, err := shacl.ParseTurtle(bytes.NewReader(sensorml.Shapes()))
    if err != nil {
        log.Fatal(err)
    }
    mapping, err := shacl.DecodeMappingYAML(bytes.NewReader(sensorml.Mapping()))
    if err != nil {
        log.Fatal(err)
    }

    bundle, err := shacl.Compile(spec, shapes, mapping)
    if err != nil {
        log.Fatal(err)
    }
    for _, c := range bundle.Contracts() {
        fmt.Println("contract:", c.EntityPattern)
    }
    fmt.Println("unwritable:", bundle.Unwritable())
}
```

Output:

```text
http://www.opengis.net/spec/sensorML/3.0/PhysicalSystem
true
contract: *.*.*.csapi.deployment.*
contract: *.*.*.csapi.procedure.*
contract: *.*.*.csapi.property.*
contract: *.*.*.csapi.samplingfeature.*
contract: *.*.*.csapi.system.*
unwritable: []
```

If a registration fails, the registry and the serializer's prefixes are left as they were: SemEngine checks the whole
pack before it changes anything (`TestPackAtomicFailure`). Failures the tests cover are an unknown field in the YAML,
an invalid predicate name, an invalid superclass IRI, a projection without a final value predicate, and a prefix
already bound to a different namespace (`TestPackConflictingRegistration`).

Registering the same pack twice is allowed and changes nothing (`TestPackRepeatedRegistration`). Registering a pack
that reuses an entity-type or predicate name the registry already holds, with a different IRI or description, is not
refused: the later value wins, and superclass lists are merged (`TestPackConflictingRegistration`). Rebinding a prefix
is refused.

## Using your own pack

The functions above are the pattern. Write a pack YAML, decode it with `vocabulary.LoadPackYAML`, and register it with
`vocabulary.RegisterPack(spec, vocabulary.WithPrefixRegistrar(export.RegisterPrefixes))`. The pack format belongs to
SemEngine. To build a pack and validator from SHACL shapes instead of writing the pack by hand, see
[shacl-compilation.md](shacl-compilation.md).
A pack built over BFO 2020 or CCO v2.2 that is embedded here is added to the `embedded` list in
`packs/packs_test.go` and to the ledger, and the [BFO alignment](#bfo-alignment) checks then apply to it.

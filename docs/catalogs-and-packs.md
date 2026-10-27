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

## Registering

Each pack has a function that registers it with SemEngine's `vocabulary.RegisterPack`:

- `packs.RegisterBFOCCOCore(opts ...vocabulary.PackOption) error`
- `packs.RegisterSWECore(opts ...vocabulary.PackOption) error`
- `diekm.Register(opts ...vocabulary.PackOption) error`

All three pass `vocabulary.WithPrefixRegistrar(export.RegisterPrefixes)`, so SemEngine's serializer learns the
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

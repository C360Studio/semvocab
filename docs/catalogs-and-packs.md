# Catalogs and packs

A catalog is a Go package of constants for the terms of one ontology, plus that ontology's subclass hierarchy. A pack
is a data file (YAML) that tells SemEngine's vocabulary registry which entity types and predicates exist and how they
map to ontology IRIs. Catalogs are for your Go code; packs are for the registry. A pack's YAML holds IRIs as plain
strings and does not import a catalog.

Terms used below:

- **IRI**: a global name for a term, written like a URL (`http://purl.obolibrary.org/obo/BFO_0000001`).
- **Ontology**: a published set of IRIs with stated meanings. BFO and CCO are the two used here.
- **Entity type**: a SemEngine category of entity, written `<domain>.<type>`, such as `intel.platform`.
- **Predicate**: a SemEngine property name of three dotted parts, such as `intel.role.assigned`.
- **Prefix**: a short name for a namespace (`obo` for `http://purl.obolibrary.org/obo/`), used when RDF is written out.

## Catalogs

Importing a catalog registers nothing.

### `catalog/bfo`

Constants for the Basic Formal Ontology 2020 (BFO) classes and relations that were admitted, such as `bfo.Entity`,
`bfo.Quality`, `bfo.Process`, and `bfo.LocatedIn`. `bfo.Namespace` is `http://purl.obolibrary.org/obo/`, the base all
OBO Foundry ontologies share.

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

## Packs

### `bfo-cco-core`

`packs.BFOCCOCoreSpec()` decodes the embedded `packs/bfo-cco-core.yaml`. The pack declares two prefixes and nothing
else: `obo` for the BFO namespace and `cco` for the CCO namespace. Register it so SemEngine's RDF output writes
`obo:BFO_0000001` rather than the full IRI. The prefix is `obo`, not `bfo`, because BFO IRIs live under the namespace
every OBO ontology shares.

### `diekm`

`packs/diekm` holds a pack for the DIEKM vocabulary, a C360-local vocabulary built over BFO and CCO. It is not an
externally registered ontology.

| Kind | Name | Meaning |
| --- | --- | --- |
| Entity type | `intel.platform` | A collection platform; a CCO Vehicle and a BFO object |
| Entity type | `intel.observation` | An observation act; a CCO Act of Observation |
| Predicate | `intel.role.assigned` | A string; written as the entity being bearer of a role whose label is the value |
| Predicate | `intel.measurement.altitude-m` | A float in meters; written through an Altitude, a measurement, and the value |

The two predicates carry a projection: a chain of steps through intermediate nodes. One flat value
(`intel.measurement.altitude-m = 120.5`) becomes the chain entity, altitude, measurement, value when exported as RDF.
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
- `diekm.Register(opts ...vocabulary.PackOption) error`

Both also pass `vocabulary.WithPrefixRegistrar(export.RegisterPrefixes)`, so SemEngine's serializer learns the
prefixes. Options you pass come after it; a later `WithPrefixRegistrar` replaces it. Call them from your program's
start-up code. This program registers both and reads the result back:

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

Registering the same pack twice is allowed and changes nothing (`TestPackRepeatedRegistration`). Registering a
different pack that redefines a class or predicate IRI the registry already holds is not refused: the later scalar
value wins, and superclass lists are merged (`TestPackConflictingRegistration`). Rebinding a prefix is refused.

## Using your own pack

The functions above are the pattern. Write a pack YAML, decode it with `vocabulary.LoadPackYAML`, and register it with
`vocabulary.RegisterPack(spec, vocabulary.WithPrefixRegistrar(export.RegisterPrefixes))`. The pack format belongs to
SemEngine. To build a pack and validator from SHACL shapes instead of writing the pack by hand, see
[shacl-compilation.md](shacl-compilation.md).

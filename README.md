# SemVocab

SemVocab is a Go library of ontology terms and ready-made vocabulary packs for
[SemEngine](https://github.com/C360Studio/semengine), plus a compiler that turns a SHACL shapes document into the
pack and validator SemEngine needs.

It is early software: the latest tag is `v0.2.2`, and nothing here is promised to stay the same before 1.0.

## What problem it solves

SemEngine stores facts about things (entities) as triples: an entity ID, a predicate name such as
`intel.measurement.altitude-m`, and a value. Its vocabulary registry records which predicates and entity types exist,
and can map each one to an ontology IRI so that exported RDF uses standard terms. An ontology is a published set of
terms with agreed meanings; an IRI is a term's global name, written like a URL
(`https://www.commoncoreontologies.org/ont00000787`); RDF is the W3C data format for triples.

Someone has to write those registrations, and they have to be right: the IRIs must come from a real ontology release,
and the entity types must sit under the right ontology classes. SemVocab holds that work so you do not repeat it:

- the IRI constants and class hierarchies for six ontologies and schemas: BFO 2020, CCO v2.2, W3C SOSA and SSN (2017
  Recommendation), and OGC SWE Common 3.0 and SensorML 3.0 (catalogs);
- vocabulary packs, which are files that register entity types and predicates with SemEngine in one step;
- a compiler that reads SHACL shapes (a W3C format for stating what values an entity's properties may hold) and
  produces a pack, contracts, and a validator for one entity's complete state. A contract tells SemEngine which
  predicates on an entity type one program is allowed to write, and whether each write replaces or adds to the
  stored values.

SemEngine owns the registry and the RDF export. SemVocab fills the registry. SemSPARQL
([C360Studio/semsparql](https://github.com/C360Studio/semsparql)) owns query expansion, exporting shapes to an
external validator, and checking this repository's validator against one; none of that is here.

## What is in the repository

| Package | What it holds |
| --- | --- |
| `catalog/bfo` | IRI constants for the admitted Basic Formal Ontology (BFO 2020) classes and relations, and `SubClassOf`, `Parents`, `IsA` |
| `catalog/cco` | The same for the admitted Common Core Ontologies (CCO v2.2) classes and object properties; its hierarchy continues into `catalog/bfo` |
| `catalog/sosa` | IRI constants for the W3C Sensor, Observation, Sample, and Actuator (SOSA, 2017) classes and properties (constants only; SOSA declares no subclass axioms) |
| `catalog/ssn` | IRI constants for the W3C Semantic Sensor Network (`ssn:`) and SSN System Capabilities (`ssn-system:`) 2017 ontologies, and `DirectParents`, `Parents`, `IsA` (continuing into `catalog/sosa`) |
| `catalog/swe` | IRI constants for the OGC SWE Common Data Model 3.0 object classes and JSON properties (`Prop*`), and `SubClassOf`, `Parents`, `IsA` |
| `catalog/sensorml` | The same for OGC SensorML 3.0 object classes and JSON properties (`Prop*`); its hierarchy continues into `catalog/swe` |
| `packs` | `bfo-cco-core` (`RegisterBFOCCOCore`, `obo` and `cco` prefixes) and `swe-core` (`RegisterSWECore`, `sosa`, `ssn`, `ssn-system`, `swe`, and `sml` prefixes) |
| `packs/diekm` | The DIEKM pack (`intel.platform`, `intel.observation`, and two predicates), its SHACL shapes, and the mapping that binds them |
| `shacl` | Parses shapes (Turtle, a text format for RDF; JSON; YAML), compiles them with a mapping, validates an entity's state |

Importing any of these registers nothing. A program registers a pack only by calling its `Register…` function.

## Install

SemVocab needs Go 1.26.9 or later (`go.mod`).

```bash
go get github.com/c360studio/semvocab@v0.2.2
```

Your own module also needs SemEngine (`github.com/c360studio/semengine`) if it imports the registry; `go mod tidy`
adds it.

## Use

Register the built-in packs at program start, then use SemEngine's registry:

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

To compile a shapes document, give `shacl.Compile` a pack, parsed shapes, and a mapping (a small YAML or JSON file
that says which pack entity type and predicate each shape class and path stands for). This compiles the DIEKM files
that ship with the repository:

```go
package main

import (
    "bytes"
    "fmt"
    "log"

    "github.com/c360studio/semvocab/packs/diekm"
    "github.com/c360studio/semvocab/shacl"
)

func main() {
    spec, err := diekm.Spec()
    if err != nil {
        log.Fatal(err)
    }
    shapes, err := shacl.ParseTurtle(bytes.NewReader(diekm.Shapes()))
    if err != nil {
        log.Fatal(err)
    }
    mapping, err := shacl.DecodeMappingYAML(bytes.NewReader(diekm.Mapping()))
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
    fmt.Println("no contract:", bundle.Unwritable())
}
```

Output:

```text
contract: *.*.*.intel.platform.*
no contract: [intel.observation]
```

Entity IDs in SemEngine have six dotted parts, `org.platform.system.domain.type.instance`, so the pattern
`*.*.*.intel.platform.*` means "every `intel.platform` entity". `intel.observation` has no contract because its shape
constrains no predicate. Compiling registers nothing; the
bundle's `Pack()` is yours to register. [docs/shacl-compilation.md](docs/shacl-compilation.md) shows validating an
entity against the shapes.

## Status and limits

- Pre-1.0. The API can change between releases.
- The catalogs are not complete ontologies. Each holds only the terms that were admitted from a named release; the
  provenance is in [docs/catalog-ownership.md](docs/catalog-ownership.md) and `docs/admission-ledger.yaml`.
- `https://w3id.org/c360/diekm#` is a vocabulary local to C360. It is not a registered or certified ontology, and its
  IRIs do not resolve.
- `http://www.opengis.net/spec/SWE/3.0/` and `http://www.opengis.net/spec/sensorML/3.0/` are local C360 RDF conventions
  built over OGC's specification base URIs (`http://www.opengis.net/spec/SWE/3.0` and
  `http://www.opengis.net/spec/sensorML/3.0`) because OGC 24-014 and OGC 23-000 publish JSON Schemas rather than OWL
  ontologies; they are not registered or certified OGC ontologies and their IRIs do not resolve.
- A SensorML domain pack (entity types, three-part predicates, and SHACL shapes over these catalogs) is not yet
  present.
- The SHACL compiler reads a fixed part of SHACL and rejects everything else by name, so a shapes file written for
  another tool may not compile. The accepted and rejected terms are listed in
  [docs/shacl-compilation.md](docs/shacl-compilation.md).
- `sh:rule` is rejected. Translating rules is not present; it is tracked in
  [issue #21](https://github.com/C360Studio/semvocab/issues/21), which waits until the upstream rule-processing APIs
  are available.
- Exporting shapes to an external validator, and checking the validator here against one, are not present here; they
  belong to SemSPARQL.
- SemVocab depends on SemEngine `v0.1.0-alpha.2`, also pre-release.

## Where to go next

- [docs/catalogs-and-packs.md](docs/catalogs-and-packs.md): the catalogs, the packs, registering them, and DIEKM.
- [docs/shacl-compilation.md](docs/shacl-compilation.md): the `shacl` package in full.
- [docs/catalog-ownership.md](docs/catalog-ownership.md): which terms come from SemEngine and which from here.

## License

MIT, see [LICENSE](LICENSE). The upstream terms keep their own licenses: BFO 2020 (CC BY 4.0), CCO v2.2 (BSD 3-Clause),
W3C SOSA and SSN 2017 (W3C 2015 Software and Document License and OGC Software License), and OGC SWE Common 3.0 and
SensorML 3.0 (OGC Software License). Notices are in the package
comments of `catalog/bfo`, `catalog/cco`, `catalog/sosa`, `catalog/ssn`, `catalog/swe`, and `catalog/sensorml`, and in
`internal/oracle/testdata/*/NOTICE`.

## Contributing

Most contributors here are coding agents. [AGENTS.md](AGENTS.md) lists the commands and rules, and
[.agents/protocol.md](.agents/protocol.md) describes how work is claimed, reviewed, and merged. Run `task verify`
before pushing. Open an issue before starting anything large.

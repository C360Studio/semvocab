# SemVocab

SemVocab is a Go library of ontology terms and ready-made vocabulary packs for
[SemEngine](https://github.com/C360Studio/semengine), plus a compiler that turns a SHACL shapes document into the
pack and validator SemEngine needs.

It is early software: the latest tag is `v0.2.2`, and nothing here is promised to stay the same before 1.0.

## What problem it solves

SemEngine stores facts about things (entities) as triples: an entity ID, a predicate name such as
`intel.measurement.altitude-m`, and a value. Its vocabulary registry records which predicates and entity types exist,
and can map each one to an ontology IRI (a global name, such as `https://www.commoncoreontologies.org/ont00000787`) so
that exported RDF uses standard terms.

Someone has to write those registrations, and they have to be right: the IRIs must come from a real ontology release,
and the entity types must sit under the right ontology classes. SemVocab holds that work so you do not repeat it:

- the IRI constants and class hierarchy for two ontologies, BFO 2020 and CCO v2.2 (catalogs);
- vocabulary packs, which are files that register entity types and predicates with SemEngine in one step;
- a compiler that reads SHACL shapes (a W3C format for stating what values an entity's properties may hold) and
  produces a pack, rules for which predicates a writer owns, and a validator for one entity's complete state.

SemEngine owns the registry and the RDF export. SemVocab fills the registry. SemSPARQL
([C360Studio/semsparql](https://github.com/C360Studio/semsparql)) owns query expansion, exporting shapes to an
external validator, and checking this repository's validator against one; none of that is here.

## What is in the repository

| Package | What it holds |
| --- | --- |
| `catalog/bfo` | IRI constants for the admitted Basic Formal Ontology (BFO 2020) classes and relations, and `SubClassOf`, `Parents`, `IsA` |
| `catalog/cco` | The same for the admitted Common Core Ontologies (CCO v2.2) classes and object properties; its hierarchy continues into `catalog/bfo` |
| `packs` | `bfo-cco-core`, a pack that declares the `obo` and `cco` prefixes; `RegisterBFOCCOCore` |
| `packs/diekm` | The DIEKM pack (`intel.platform`, `intel.observation`, and two predicates), its SHACL shapes, and the mapping that binds them |
| `shacl` | Parses shapes (Turtle, JSON, YAML), compiles them with a mapping, validates an entity's state |

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

`intel.observation` has no contract because its shape constrains no predicate. Compiling registers nothing; the
bundle's `Pack()` is yours to register. [docs/shacl-compilation.md](docs/shacl-compilation.md) shows validating an
entity against the shapes.

## Status and limits

- Pre-1.0. The API can change between releases.
- The catalogs are not complete ontologies. Each holds only the terms that were admitted from a named release; the
  provenance is in [docs/catalog-ownership.md](docs/catalog-ownership.md) and `docs/admission-ledger.yaml`.
- `https://w3id.org/c360/diekm#` is a vocabulary local to C360. It is not a registered or certified ontology, and its
  IRIs do not resolve.
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

MIT, see [LICENSE](LICENSE). The BFO 2020 and CCO v2.2 terms keep their own licenses (CC BY 4.0 and BSD 3-Clause);
the notices are in the package comments of `catalog/bfo` and `catalog/cco`.

## Contributing

Most contributors here are coding agents. [AGENTS.md](AGENTS.md) lists the commands and rules, and
[.agents/protocol.md](.agents/protocol.md) describes how work is claimed, reviewed, and merged. Run `task verify`
before pushing. Open an issue before starting anything large.

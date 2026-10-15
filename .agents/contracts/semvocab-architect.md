# SemVocab Architect Agent Contract

## Purpose and authority

The architect designs nontrivial SemVocab work: catalog admission, pack layout, the SHACL compiler profile, and any
boundary with SemEngine or SemSPARQL. It is read-only. It produces inventories and design drafts as text; it does
not write files, open issues, or make binding rulings. Binding rulings stay with the owner, on the issue. Adapted from
`C360Studio/semengine` `.agents/contracts/semengine-architect.md`; where a rule differs, this file is authoritative.

Read `AGENTS.md` ("What this is for"), issue #1 (`gh issue view 1 --comments`), the [shared protocol](../protocol.md),
and the active OpenSpec change before any design.

## Required workflow

1. Name the question and the repositories the inventory reads. SemEngine (at the commit pinned in `go.mod`, or the
   commit the issue names before a pin exists), the SemStreams pin `8b99efe9c66a4faa4fa509f9f62cc6bad8392128`, and
   SemSPARQL are read-only sources. Read them from a local clone or `gh api`/`git show <sha>:<path>`; never from a
   moving branch without naming the commit.
2. Produce the ownership inventory (below) before any target state.
3. Produce the adopter seam inventory for any surface reached from outside this repository.
4. Draft the design: observable behaviour and the test that proves it, not mechanism. List every open PR (drafts
   included) that overlaps the change and state the merge order.
5. A design gets three review rounds. Findings still open after the third go to the owner on the issue, with the
   choice stated.

## Ownership inventory (mandatory first deliverable)

SemEngine remains the source of truth for standard constants and generic vocabulary behaviour it already exports.
For each proposed `catalog/<domain>` package, pack, or compiler surface, record:

- **SemEngine overlap, symbol by symbol:** read `vocabulary/iris.go`, `standards.go`, `predicates.go`, `labels.go`,
  `relationships.go`, `pack.go`, `projection.go`, and `vocabulary/export/prefix.go` at the pinned commit. Distinguish
  exported constants from built-in serializer prefix mappings and documentation examples; a namespace mention is not
  evidence of a public catalog API.
- **Disposition, one of:** use SemEngine's exported symbol directly (no SemVocab API); a thin, consumer-justified
  re-export referencing that symbol, with equality tests and no copied IRI string; new ontology content absent from
  SemEngine, with its own source, release, and license; or deferred with a stated reason.
- **Provenance:** exact source revision, file paths, and symbols at the SemStreams pin, or the authoritative external
  ontology and release when no pinned source exists. Whether the work is a direct port, an extraction from shared
  files, new material, or deferred. License and attribution. The test oracle.
- **Namespace and release policy:** BFO/CCO release choices and the provenance of the C360 DIEKM namespace. A local
  mapping is never presented as an externally certified ontology. Any intentional divergence from SemEngine needs an
  explicit mapping and compatibility rationale.

Do not invent source packages, infer terms from a numeric range or from namespace support, or duplicate SemEngine
registry code.

## Adopter seam inventory (mandatory second deliverable)

For any surface reached from outside this repository (a consumer registering a pack, SemSPARQL reading a projection,
SemEngine's loader reading our YAML), record who calls it, at which commit, what they pass, and what they rely on.
Prefer observation to prediction: cite the caller's code at a named commit, or say no caller exists yet and name the
consumer that will.

## Design discipline

- **Upstream gaps are SemEngine's.** When a design needs a SemEngine API that does not exist (as the pure
  `PackSpec.Validate` seam did, semengine#256), the design names the upstream issue and the SemVocab task carries a
  `Hold:` on it. It never designs a local copy of a private symbol.
- **State behaviour, not mechanism.** Each design commitment names the observable behaviour and its proving test.
- **Cite the pin.** Every statement on how SemEngine, the SemStreams pin, or a standard (W3C Turtle, JSON-LD, SHACL
  Core, SHACL-AF) behaves cites `file:line` at the commit or the spec section, or says the behaviour is not promised.
- **Bounded scope.** Optional maintenance findings are presented as optional ([protocol](../protocol.md),
  "Bounded port scope").
- **Reusable primitives.** A change establishing a reusable primitive lists who should adopt it.
- **No RDF or SHACL engine in SemEngine's hot paths.** A design that needs one is out of scope for this repository.

## Handoff

Return the inventories and design as text the orchestrating session can commit. Name every open question for the
owner and the choice each needs. State what the design does not cover.

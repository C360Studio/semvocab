# Notes: Tasks 3.1-3.6 (choices the design does not spell out)

All checks live in `packs/bfo_soundness_test.go` (design D1 (a)).

## Classification

- An entity type's class set is its `class_iri` followed by its `superclasses`. A class set is "in" a class when some
  member `cco.IsA` it (`cco.IsA` falls back to `bfo.IsA`). A node class's set is the node class alone.
- `knownClass` is true for `bfo.Entity` and for any IRI `cco.SubClassOf` answers. A node class that is not known fails
  closed ("unknown to catalog/bfo and catalog/cco") and ends that chain; an entity type whose class set has no known
  member fails closed the same way.

## Disjointness

`disjointPairs` reads `owl:disjointWith` between named BFO subjects and every pair of each `owl:AllDisjointClasses`
`owl:members` list (blank subjects, enumerated with `SubjectsWithPrefix("_:")`). A non-IRI member fails closed.
`TestBFODisjointPairsFromRelease` pins 26 pairs, counted by hand (5 + 3 + 3 + 6 + 6 + 3), so a reader that drops a
source fails without a pack fault.

## Hop check

- The IC/SDC anchoring rule applies to every hop, whatever the predicate; the domain/range rule applies only to the 17
  relations. This is what lets the "unanchored quality" (`cco:ont00001904`) and "unanchored role"
  (`bfo.ParticipatesIn`) scenarios fire.
- Stricter than design D3: a projected predicate bound to no entity type is reported whatever its step 0 predicate is,
  not only when step 0 is one of the 17, because the anchoring rule also needs a step-0 subject (owner answer 2,
  2026-11-07: fail closed without a mapping). No embedded pack is affected.
- Several `rdfs:domain` (or `rdfs:range`) objects on one relation must all hold; a relation declaring none fails closed.
  All 17 declare exactly one of each in `bfo-2020`.
- `hopProblems` also returns the hops it checked against domain and range, and `TestBFOProjectionHops` pins diekm's
  three, so a check that silently matches no relation fails.

## Interpreter

`readClassExpr` accepts a named class, `owl:unionOf` of named classes, and `owl:intersectionOf` of exactly one named
class with `owl:complementOf` named classes. It takes an `objectReader` interface (`Objects(s, p)`), which
`*oracle.Graph` satisfies, so `TestBFORelationBoundsFailClosed` can feed it Turtle snippets (an `owl:Restriction`, a
nested union, two named classes in an intersection, a bare complement, a literal, a missing domain) parsed with
`internal/turtle`.

## Mapping and opt-in

- `bfoPackMappings` maps a pack name to its embedded mapping (`diekm.Mapping`). `packMapping` fails closed for a
  selected pack with a projected predicate and no entry; a pack with no projection (`bfo-cco-core`) needs none.
- `processConnectedPacks` is empty. `TestBFOProjectionHops` runs `connectivityProblems` for each selected pack opted in
  there, so opting a pack in is a one-line change.

## Extra twins

Beyond tasks 3.2 and 3.5: a fiat-boundary entity type (`BFO_0000142` closure) is named as having no bucket root (owner
answer 3); the carrier-hop twin also asserts the range is not named; and the seeded realization step of the
connectivity scenario passes the hop check.

## Twins added on review (PR #39)

The first cut had no twin for the range diagnostic or for the SDC to IC rule (mutants 8 and 9 in `mutate.md` survived).
`TestBFOProjectionHopsSensitivity` now seeds `intel.role.assigned` as `bearer of` a `cco.Act`: the subject is inside
the domain and no SDC is reached, so only the range is named. It also extends the role chain with a Role to
`cco.Vehicle` hop (moving the terminal value predicate to the new last step): through `bfo.ParticipatesIn` it is named
"without bfo.InheresIn"; through `bfo.InheresIn` the chain has no problem.

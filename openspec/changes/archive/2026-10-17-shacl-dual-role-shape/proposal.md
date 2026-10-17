# shacl-dual-role-shape

## Why

Issue #10: `shacl.ParseTurtle` builds an IRI in two roles when it is a node shape (it has a target, or is typed
`sh:NodeShape`, ruling Q27) and also the object of another shape's `sh:property`. It builds a `NodeShape` and a
`PropertyShape` from the same triples. Compilation already fails, but only through unrelated-looking records against
the IRI (`sh:minCount`, `sh:path`, `rdf:type`, `sh:targetClass`). None of them says the IRI has two roles (measured at
`8442f12` on #10).

The owner ruled on 2026-10-17 (Q32, on #10): reject the input with one error that names both roles. Q27 allows targets
only on node shapes, with property shapes nested under them, so a property shape with its own target is outside the
profile. Rejected alternative: accept both roles, which would let the two meanings of one IRI drift apart.

## What Changes

- The parser finds each IRI that is a node shape and also the object of `sh:property` from a node shape, whatever the
  statement order. It builds that IRI in neither role.
- Instead of that IRI's per-triple records, the parser records exactly one rejection. It names the IRI, what makes it a
  node shape (its target predicates, or its `sh:NodeShape` type), and every node shape that refers to it. `Compile`
  reports it as one issue.
- A referring node shape is still built with its other property shapes. `Compile` still fails as a whole and returns
  no bundle.
- `docs/shacl-compilation.md` says so.

## Impact

`shacl/parser.go`, `shacl/compile.go` (the message of the new record), and their tests; one line in
`docs/shacl-compilation.md`. The exported API does not change. Inputs that compiled before still compile; inputs in
both roles still fail, now with one issue in place of several. Closes #10.

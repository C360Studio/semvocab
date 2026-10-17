# Tasks: shacl-dual-role-shape

## 1. Reject a shape in both roles

- [x] 1.1 A regression test compiles issue #10's example through `ParseTurtle` and `Compile` and fails at `8442f12`
- [x] 1.2 The parser builds a dual-role IRI in neither role and records one rejection naming the IRI, its node-shape
      role, and every referring node shape, in any statement order. Tests cover two referring shapes, each target
      predicate and the `sh:NodeShape` type, and the referring shape's other properties; a seeded change is shown to
      fail them
- [x] 1.3 The W3C rejected-family and path cases and the Rapid properties still pass
- [ ] 1.4 `docs/shacl-compilation.md` states the rule
- [ ] 1.5 `task verify` passes on a clean tree

## 2. Archive

- [ ] 2.1 The change is archived and `openspec/specs/` is synced; `task spec:check` passes

# Tasks: shacl-json-utf8-refusal

## 1. Refuse text the JSON decoders would replace

- [ ] 1.1 Regression tests feed each JSON decoder invalid UTF-8 inside a string and a lone surrogate escape, and fail
      at `2a0ae84`
- [ ] 1.2 Both JSON decoders refuse invalid UTF-8 and lone surrogate escapes with a byte offset, before
      `encoding/json` reads the input; a valid surrogate pair and escaped backslashes still decode. A seeded change
      is shown to fail the tests
- [ ] 1.3 `FuzzDecodeShapesJSON` asserts that an accepted input is valid UTF-8
- [ ] 1.4 `docs/shacl-compilation.md` states the rule
- [ ] 1.5 `task verify` passes on a clean tree

## 2. Archive

- [ ] 2.1 The change is archived and `openspec/specs/` is synced; `task spec:check` passes

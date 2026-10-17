# Tasks: shacl-decoder-size-cap

## 1. Cap the JSON and YAML documents

- [x] 1.1 A test feeds each of the four decoders a document of exactly 1 MiB, which decodes, and one of 1 MiB plus
      one byte, which fails with the limit and its byte offset; the over-limit cases fail at `f94f8ab`
- [x] 1.2 One read helper enforces the cap for `ParseTurtle` and the four decoders; `TestParseTurtleSizeLimit` still
      passes unchanged. A seeded change is shown to fail the tests
- [x] 1.3 The JSON and YAML fuzz targets accept the size-limit error as a named refusal
- [x] 1.4 `docs/shacl-compilation.md` states the cap for the JSON and YAML forms
- [ ] 1.5 `task verify` passes on a clean tree

## 2. Archive

- [ ] 2.1 The change is archived and `openspec/specs/` is synced; `task spec:check` passes

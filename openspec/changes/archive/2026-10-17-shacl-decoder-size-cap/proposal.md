# shacl-decoder-size-cap

## Why

Issue #17: `shacl.ParseTurtle` refuses input over 1 MiB (design D2, ruling Q25), but `DecodeShapesJSON`,
`DecodeShapesYAML`, `DecodeMappingJSON`, and `DecodeMappingYAML` read their whole input with no bound. A caller that
passes shape or mapping documents it did not write gets a memory bound from one format and not the other two.
SemSPARQL now requires `v0.2.1`, so these decoders are a consumer-facing API.

The owner ruled on 2026-10-17 (Q33, on #17): the four decoders get the same 1 MiB cap, failing with the byte offset
where the limit was crossed, as Turtle does. Real documents are far below it: the DIEKM shapes are about 700 bytes in
JSON and YAML, and its mapping about 1.3 KB.

## What Changes

- One read helper caps every shapes and mapping document at 1 MiB (1,048,576 bytes). `ParseTurtle` and the four
  decoders use it. Exactly 1 MiB is accepted; one byte more fails with
  `input exceeds the 1048576-byte limit at byte offset 1048576`, the message `ParseTurtle` already gives.
- The constant `maxTurtleBytes` becomes `maxDocumentBytes`, since it no longer caps only Turtle.
- `docs/shacl-compilation.md` states the cap for the JSON and YAML forms.

## Impact

`shacl/parser.go`, `shacl/decode.go`, and their tests; lines in `docs/shacl-compilation.md`. The exported API does
not change. Documents up to 1 MiB decode as before; larger JSON and YAML documents, which decoded before, now fail.
Closes #17.

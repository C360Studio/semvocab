# Notes: task group 1 (the 1 MiB cap on the JSON and YAML forms)

Written 2026-10-17 with go1.26.9 darwin/arm64, at `f94f8ab` plus this change. The ruling is Q33 on issue #17.

## Choices

- **One helper.** `readDocument` (`shacl/parser.go`) reads through `io.LimitReader` at the cap plus one byte, as
  `ParseTurtle` did, so no format reads more than 1 MiB + 1 byte into memory. `ParseTurtle`, `decodeJSON`, and
  `decodeYAML` all call it; `ParseTurtle`'s error text is unchanged (`shacl: input exceeds …`, `shacl: read: …`).
- **Error text.** The decoders wrap it as `shacl: shapes: input exceeds the 1048576-byte limit at byte offset
  1048576` (or `shacl: mapping: …`), the same words as Turtle under each decoder's existing prefix.
- **Rename.** `maxTurtleBytes` became `maxDocumentBytes`; its only other uses were in `shacl/fuzz_test.go`.
- **Padding.** The tests and fuzz seeds pad with trailing newlines, which JSON and YAML both ignore, so the
  at-limit document decodes to the same value as the small one.

## Evidence

- **1.1:** `TestDecodeSizeLimit` (`shacl/parser_test.go`). At `f94f8ab` all four over-limit cases fail with
  `err = <nil>`; the at-limit cases pass there too, since nothing capped them.
- **1.2:** four seeded changes, each reverted, each failing the tests: `>` to `>=` (all five size tests, Turtle
  included), the limit reader without the extra byte (all five), the YAML helper reading uncapped (both YAML cases),
  and the JSON helper reading uncapped (both JSON cases). `TestParseTurtleSizeLimit` is unchanged and passes.
- **1.3:** `FuzzDecodeShapesJSON` and `FuzzDecodeShapesYAML` add at-limit and one-over seeds under plain `go test`
  (as `FuzzParseTurtle` does) and accept `shapesSizeLimit` as a named refusal. With the pattern removed, both
  targets fail on the one-over seed.

# shacl-json-utf8-refusal

## Why

Issue #18: `DecodeShapesJSON` and `DecodeMappingJSON` decode through `encoding/json`. When it unmarshals a quoted
string, it replaces invalid UTF-8, and an invalid UTF-16 surrogate escape, with U+FFFD and reports no error
(`encoding/json/decode.go:98-100` at go1.26.9). So a shapes or mapping document with a malformed byte in an `in`
member or a `pattern` decodes into different text without error. The YAML form and the Turtle reader refuse the same
bytes with a byte offset. Silent replacement breaks the repository rule that failure paths fail closed (`AGENTS.md`
§ 1), and ruling Q31 already chose refusal over U+FFFD for candidate values in the validator.

## What Changes

- Before `encoding/json` reads it, the JSON decoders refuse input that is not valid UTF-8, with the byte offset of
  the first bad byte.
- They also refuse a `\u` escape that names a lone UTF-16 surrogate (a high surrogate not followed by a low one, or a
  low surrogate on its own), with the escape's byte offset. `encoding/json` would write that as U+FFFD too. A valid
  surrogate pair still decodes.
- `docs/shacl-compilation.md` says so.

## Impact

`shacl/decode.go` and its tests; a line in `docs/shacl-compilation.md`. The exported API does not change. JSON input
that decoded before still decodes unless it held one of the refused byte sequences, which used to be replaced
silently. Closes #18.

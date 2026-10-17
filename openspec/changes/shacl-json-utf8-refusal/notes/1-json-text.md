# Notes: task group 1 (text the JSON decoders would replace)

Written 2026-10-17 with go1.26.9 darwin/arm64, at `2a0ae84` plus this change.

## Choices

- **Lone surrogates are in scope.** Issue #18 quotes `encoding/json`'s doc comment, which names both invalid UTF-8
  and invalid UTF-16 surrogate pairs. A `\ud800` escape is valid UTF-8 on the wire but decodes to U+FFFD, the same
  silent replacement, so both are refused.
- **One pass over the bytes before any decode.** `checkJSONText` runs before `scanJSON` and the struct decode, so the
  error carries the offset of the first bad byte or escape and nothing downstream sees replaced text. It does not
  track whether it is inside a string: a backslash outside a string is a syntax error either way, and an escaped
  backslash (`\\`) is skipped as a pair, so `\\ud800` is the text `\ud800`, not an escape.
- **Error text.** `byte offset N: the input is not valid UTF-8` and `byte offset N: \uXXXX is a lone UTF-16
  surrogate`. Both match the fuzz harness's `jsonLocated` shape.

## Evidence

- **1.1:** `TestDecodeJSONRefusesReplacedText` (`shacl/parser_test.go`). At `2a0ae84` all eight refusal cases fail
  with `err = <nil>`. The offsets were computed independently of the code from the length of each test document's
  prefix (67 bytes for the shape form, 43 for the mapping form).
- **1.2:** five seeded changes to `shacl/decode.go`, each reverted afterwards, and each fails the test: dropping the
  UTF-8 check (3 cases fail), accepting a lone low surrogate (2), accepting an unpaired high surrogate (3), stepping
  one byte past a non-`\u` escape (the escaped-backslash acceptance fails), and removing the call (all 8).
- **1.3:** `FuzzDecodeShapesJSON` asserts `utf8.Valid` on every accepted input; a local run of 20 s (about 1.27 M
  executions) found no failure.

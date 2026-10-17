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
- **1.5:** `task verify` passed on a clean tree at `3fb4f25` (spec, docs, fmt, tidy, build, vet, lint, vuln,
  test:unit, test:repeat).

## Review fix (gemini review of `ad62de2`, MEDIUM)

- **Finding:** on a backslash not starting a `\uXXXX` escape, `checkJSONText` stepped two bytes. A backslash followed
  by a non-ASCII byte skipped that byte's UTF-8 check: `\` + `0xff` passed the check, and `\é` was misreported as
  invalid UTF-8 at the continuation byte. Both inputs are invalid JSON and `encoding/json` refused them, so nothing
  was replaced, but the errors were wrong.
- **Fix:** step two bytes only over an escaped backslash (`\\`), so a following `u` is never read as an escape;
  otherwise step one byte.
- **Evidence:** two cases in `TestDecodeJSONRefusesReplacedText` ("shapes invalid byte escaped" and "shapes escaped
  multibyte") fail at `ad62de2` and pass with the fix. Seeding `i++` in place of `i += 2` for the escaped backslash
  fails the test's `\\ud800` acceptance. A 20 s `FuzzDecodeShapesJSON` run found no failure.

## Archive-check fix (claude archive check of `92baf95`, MEDIUM)

- **Finding:** no test fed either decoder an escaped surrogate pair. The acceptance checks were meant to use
  `\ud83d\ude00`, but the file held the raw character, so the pair-accepting path was never run. The case written as
  `\ud83d\u0041` had become `\ud83dA`. The cause was the tool that wrote the file: it decoded `\u` escapes in the text
  before writing. Task 1.2 and this note's 1.2 evidence claimed the pair was covered; it was not, at `3fb4f25` through
  `92baf95`.
- **Fix:** both acceptance checks now use the escaped pair `\ud83d\ude00`; the `\ud83d\u0041` case is restored and
  `\ud83dA` kept as its own case; a new case `\ud83d\ue000` covers an escape above the low-surrogate range. The two spec
  scenario copies (`openspec/specs/shacl-compilation/spec.md` and this change's delta) also had the raw character in
  place of `\ud83d\ude00` and are corrected. Every file in the branch diff was scanned for the raw character and for
  missing `\u` escapes.
- **Evidence:** seeding "refuse every high surrogate" fails the shapes acceptance check (`\ud83d is a lone UTF-16
  surrogate`); dropping the `lo > 0xDFFF` bound fails "shapes high then above low"; dropping the `lo < 0xDC00` bound
  fails "shapes high then non-low".

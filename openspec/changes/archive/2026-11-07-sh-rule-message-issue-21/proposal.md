# Proposal: sh-rule-message-issue-21

Approved-by: Coby Leuschke (2026-11-07)

Track: quick (one error message, its test, two guide lines, and one spec scenario).

## Why

Issue #23: the `sh:rule` rejection message (`shacl/profile.go:65` at `c69bb28`) says rule translation is deferred to
"issue #1 Phase 4". #1 is closed, and rule translation is now tracked in #21 (ruling Q34 on #1), so the message sends
a reader to a closed issue.

## What Changes

- `reasonRule` reads "SHACL rules are not compiled: rule translation is deferred to issue #21".
- `TestRejectedConstructsReportedTogether` asserts the new wording and cites the requirement it checks.
- `docs/shacl-compilation.md:569,856` quote the new message.
- The "A shape with a rule" scenario names #21.

## Impact

Only the wording of one error message changes; no behaviour changes. A caller that matched the old text on
"issue #1 Phase 4" no longer matches. Closes #23.

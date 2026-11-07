# Tasks: sh-rule-message-issue-21

## 1. Repoint the message

- [x] 1.1 `reasonRule` (`shacl/profile.go:65`) names issue #21; `TestRejectedConstructsReportedTogether`
      (`shacl/reject_test.go:100`) asserts it and fails against the old wording
- [x] 1.2 `docs/shacl-compilation.md:569,856` quote the new message; `task docs:check` passes
- [x] 1.3 `task verify` passes on a clean tree

## 2. Archive

- [x] 2.1 The change is archived and `openspec/specs/` is synced; `task spec:check` passes

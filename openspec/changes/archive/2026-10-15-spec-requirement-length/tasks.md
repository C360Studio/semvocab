# Tasks: spec-requirement-length

## 1. Split the long requirements

- [x] 1.1 agent-setup "Thin role adapters" and catalog-provenance "One disposition per candidate" are each under 500
      characters, and their moved SHALL statements appear unchanged in "Adapter tool and model pins" and
      "Disposition fields"
- [x] 1.2 `openspec validate spec-requirement-length --strict` passes, and `task verify` passes on a clean tree

## 2. Archive

- [x] 2.1 The change is archived and `openspec/specs/` is synced; `task spec:check` passes under OpenSpec 1.14.0 and
      1.14.1

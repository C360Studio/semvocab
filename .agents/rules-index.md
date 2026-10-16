# SemVocab rule index

The full enforcement audit matrix behind the domain-grouped summary in `AGENTS.md`. Each numbered section below has
the same title and the same number of rows, in the same order, as its section in `AGENTS.md`;
`TestAgentSetupParity` (`internal/harness/contract/agentsetup_test.go`) checks the titles and row counts, and the
one-to-one meaning of each row is review only. "Enforced by" names the command, test, or contract section that
catches a break; "Gaps" names what stays review only. A rule whose only enforcement is review says so: a planned check
is listed as a gap until it exists.

## 1. Go and package boundaries

| Rule | Enforced by | Gaps |
| --- | --- | --- |
| Production code imports no test library and nothing under `internal/harness` | Review: developer § Required workflow; reviewer § Required review workflow; `TestTestOnlyImports` fails a non-test file importing `pgregory.net/rapid` | No guard for other test libraries or `internal/harness` yet; SemEngine's `TestProductionImports` is the model when production packages land |
| Public packages' exported signatures name no `internal/` type | Review: developer, reviewer | No signature guard yet (SemEngine `signatures_test.go` is the model) |
| Importing a package registers nothing; `Register…` functions, called only from a composition root | Review: developer § Registries and global state; reviewer § Registries and global state | No `init()` guard yet |
| Validation and compilation never mutate SemEngine's global registries, and never snapshot/clear/register/restore them | Review: developer, reviewer; #1 Phase 2 acceptance (state-independence tests) | Proven per package by its own tests once `shacl/` exists |
| Failure paths fail closed; any skip, drop, or degrade is declared | Review: developer § Required workflow | Review only |
| The pinned revive passes | `task lint` (in `task verify`) | None |

## 2. Vocabulary, packs, and catalogs

| Rule | Enforced by | Gaps |
| --- | --- | --- |
| SemEngine is the source of truth for constants and behaviour it exports; no copied IRI string or second registry | Review: architect § Ownership inventory; reviewer § Ownership and provenance | Parity/equality tests per re-export (#1 `TestCatalogPortParity`) once catalogs land |
| Each `catalog/<domain>` records one disposition before code | Review: architect § Ownership inventory | Review only |
| Predicate, entity-type key, and entity ID shapes | Review: developer, reviewer; SemEngine's own validators at registration | SemEngine's validators run only when a pack is registered |
| Packs register only through SemEngine's pinned `RegisterPack` and prefix registrar; a rejected load leaves state unchanged | Review: developer, reviewer; #1 Phase 1 acceptance tests | Review only until those tests exist |
| Ontology terms come from a named source and release; no inference from numeric ranges or namespace support; no false certification claim | Review: architect, developer, reviewer | Review only |
| The SHACL compiler rejects `sh:rule` and every out-of-profile construct, naming it | Review: architect, developer, reviewer; #1 Phase 2 acceptance | Review only until `shacl/` tests exist |

## 3. Testing and verification

| Rule | Enforced by | Gaps |
| --- | --- | --- |
| Tests bind no fixed address or port | Review: developer, reviewer | SemEngine's `lint-test-ports.sh` is not carried (no networked tests yet) |
| No `time.Sleep` in a test file or under `internal/harness/` | Review: developer, reviewer | No guard yet |
| No `Skip`; no build tag other than `integration` | Review: developer, reviewer | No guard yet |
| Unit tests once under `-race` at one CPU, and five times at one CPU without `-race`, shuffled | `task test:unit`, `task test:repeat` (in `task verify`) | The flags are not pinned by a contract test yet |
| Independent oracle; tests shown able to fail; structural guards carry sensitivity tests | `TestAgentSetupParitySensitivity`, `TestMergeCheck*`, `TestSpecQueue*` for the carried guards; review for the rest | Review only for product tests |
| A test that touches SemEngine's global registries asserts state, never restores it by hand | Review: developer, reviewer | Review only |
| Generated checks record seed and replayable failure; survivors reported as survivors | Review: developer, reviewer | Review only |

## 4. Porting, provenance, and dependencies

| Rule | Enforced by | Gaps |
| --- | --- | --- |
| Ported or extracted material has an admission-ledger row with pin/source, paths, symbols, license, disposition | Review: architect § Provenance; reviewer § Ownership and provenance | `docs/admission-ledger.yaml` and its schema check land with the first port (#1 Phase 0); SemEngine's `ledger:check` is the model |
| SemEngine pinned in `go.mod` to a merged commit; behaviour statements cite the pin | Review: architect, reviewer; `task tidy:check` for module consistency | Review only for citations |
| Upstream gaps filed on SemEngine and linked, never forked or copied | Review: architect, developer, reviewer | Review only |
| New surface needs a caller or contract line and test; no test-only back door | Review: architect, developer, reviewer | Review only |
| Library/standard behaviour statements cite a probe or spec section, or say not promised | Review: architect, developer, reviewer, technical writer | Review only |
| Sister repositories are read-only from here | Review: protocol § Work lifecycle | Review only |

## 5. Workflow, review, and merge gates

| Rule | Enforced by | Gaps |
| --- | --- | --- |
| OpenSpec changes and specs well formed and truthful | `task spec:check` (structure); review (truth against code) | Truth is review only |
| Every task tickable in or before archive; holds as `Hold:` inside the task | `scripts/openspec-queue.sh --check` in `task spec:check`; `TestSpecQueue*` | Whether a task asserts a post-merge fact is review only |
| Steps end on a local `task verify`; reviewer `PASS` cites commit and local run, never CI | `scripts/verify.sh` (fails on tracked-file change); review | Citation is review only |
| A merge needs CI green on an up-to-date head | `Required` check on the `main` ruleset; `merge-check` verifies the up-to-date setting | None |
| No merge while a `class:flake` issue is open unless the PR closes all | `scripts/merge-check.sh` (CI `merge-check`, `task merge:check`); `TestMergeCheckKnownFlake` | None |
| Code PRs need cross-agent review lines | `scripts/merge-check.sh`; `TestMergeCheckReview` | Truth of the lines, existence and verdict of the record (protocol § Cross-agent review) |
| Unfixed defects in repo code are filed as issues | Review: protocol § Work lifecycle "File"; reviewer | Review only |
| Unchosen-condition findings are `BLOCKING`/`HIGH` only with a real-world trigger | Review: reviewer § Finding and verdict format | Review only |
| A new rule names its enforcer and adds rows to both files | Review: reviewer; `TestAgentSetupParity` (row counts) | Meaning of each row is review only |
| Clarify gate before claiming, mutating, or spawning | Review: protocol § Work lifecycle; `.agents/README.md` | Review only |
| Briefs carry intent and artifacts; resume vs fresh | Review: `.agents/README.md` § Orchestrating role agents | Review only |
| Model tier and effort routing | `TestAgentSetupParity` (Claude model, Codex model and effort) | Gemini routing is review only |
| `AGENTS.md` size, rules-index parity, adapter parity | `TestAgentSetupParity`, `TestAgentSetupParitySensitivity` | One-to-one row meaning is review only |
| Claim branches update by merge, never rebase or force-push | Review: protocol § Work lifecycle "Land" | Review only |
| Cited evidence can be opened elsewhere; local-only paths named | Review: handoff and preflight skills | Review only |
| Docs written for a working developer | Review: technical writer, reviewer | Review only |

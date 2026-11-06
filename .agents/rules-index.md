# SemVocab rule index

The full enforcement audit matrix behind the domain-grouped summary in `AGENTS.md`. Each numbered section below has
the same title and the same number of rows, in the same order, as its section in `AGENTS.md`;
`TestAgentSetupParity` (`internal/harness/contract/agentsetup_test.go`) and `ateam doctor --check` verify the titles
and row counts, and the one-to-one meaning of each row is review only. "Enforced by" names the command, test, or
contract section that catches a break; "Gaps" names what stays review only.

## 1. Go and package boundaries

| Rule | Enforced by | Gaps |
| --- | --- | --- |
| Production code imports no test library and nothing under `internal/harness` | Review: developer § Required workflow; reviewer § Required review workflow; `TestTestOnlyImports` fails a non-test file importing `pgregory.net/rapid` | No guard for other test libraries or `internal/harness` yet |
| Public packages' exported signatures name no `internal/` type | Review: developer, reviewer | No signature guard yet (SemEngine `signatures_test.go` is the model) |
| Importing a package registers nothing; `Register…` functions, called only from a composition root | Review: developer § Registries and global state; reviewer § Registries and global state | No `init()` guard yet |
| Validation and compilation never mutate SemEngine's global registries, and never snapshot/clear/register/restore them | Review: developer, reviewer; state-independence tests in `shacl/` and `packs/` | New packages rely on review and their own tests |
| Failure paths fail closed; any skip, drop, or degrade is declared, and any code shortcut carries a `// shortcut:` marker linked to an issue | `ateam state pickup` surfaces untracked `// shortcut:` markers; review: developer § Required workflow | Undeclared shortcuts without markers are review only |
| The pinned revive passes | `task lint` (in `task verify`) | None |

## 2. Vocabulary, packs, and catalogs

| Rule | Enforced by | Gaps |
| --- | --- | --- |
| SemEngine is the source of truth for constants and behaviour it exports; no copied IRI string or second registry | Review: architect § Ownership inventory; reviewer § Ownership and provenance; catalog parity tests | New catalog strings rely on review + tests |
| Each `catalog/<domain>` records one disposition before code | Review: architect § Ownership inventory | Review only |
| Predicate, entity-type key, and entity ID shapes | Review: developer, reviewer; SemEngine's own validators at registration | SemEngine's validators run when a pack is validated or registered |
| Packs register only through SemEngine's pinned `RegisterPack` and prefix registrar; a rejected load leaves state unchanged | Review: developer, reviewer; pack registration tests | New pack loaders rely on review + tests |
| Ontology terms come from a named source and release; no inference from numeric ranges or namespace support; no false certification claim | Review: architect, developer, reviewer; `docs/admission-ledger.yaml` | Semantic fidelity to external source is review + fixture tests |
| The SHACL compiler rejects `sh:rule` and every out-of-profile construct, naming it | `shacl` rejection tests; review: architect, developer, reviewer | New out-of-profile constructs rely on review + tests |

## 3. Testing and verification

| Rule | Enforced by | Gaps |
| --- | --- | --- |
| Tests bind no fixed address or port | Review: developer, reviewer | SemEngine's `lint-test-ports.sh` is not carried (no networked tests) |
| No `time.Sleep` in a test file or under `internal/harness/` | Review: developer, reviewer | No static AST guard yet |
| No `Skip`; no build tag other than `integration` | Review: developer, reviewer | No static AST guard yet |
| Unit tests once under `-race` at one CPU, and five times at one CPU without `-race`, shuffled | `task test:unit`, `task test:repeat` (in `task verify`); `TestUnitInvocationsPinned` | None |
| Independent oracle; tests shown able to fail via `task mutate:check` (`ateam mutate`); structural guards carry sensitivity tests | `TestAgentSetupParitySensitivity`, `TestAteamTemplatesUnchangedSensitivity`, `TestMergeGatePinnedSensitivity`, `TestMergeCheckWrapperSensitivity`, `TestSpecTaskWiring` for harness guards; `task mutate:check` + review for product tests | Running `mutate:check` on new product checks is enforced in review |
| Every new or modified Go `Test`, `Fuzz`, or `Example` since the merge base carries a valid `// spec:` citation | `task test:spec` (`ateam test-spec --base`, in `task verify`) | Semantic adequacy of the cited Requirement is review only; untouched pre-adoption tests are exempt |
| A test that touches SemEngine's global registries asserts state, never restores it by hand | Review: developer, reviewer | Review only |
| Generated checks record seed and replayable failure; survivors reported as survivors | Review: developer, reviewer | Review only |

## 4. Porting, provenance, and dependencies

| Rule | Enforced by | Gaps |
| --- | --- | --- |
| Ported or extracted material has an admission-ledger row with pin/source, paths, symbols, license, disposition | `docs/admission-ledger.yaml`; fixture sha256 tests; review: architect § Provenance, reviewer § Ownership and provenance | Completeness across newly added files is review + test |
| SemEngine pinned in `go.mod` to a merged commit; behaviour statements cite the pin | `task tidy:check` for module consistency; review: architect, reviewer | Prose citation accuracy is review only |
| Upstream gaps filed on SemEngine and linked, never forked or copied | Review: architect, developer, reviewer | Review only |
| New surface needs a caller or contract line and test; no test-only back door | Review: architect, developer, reviewer | Review only |
| Library/standard behaviour statements cite a probe or spec section, or say not promised | Review: architect, developer, reviewer, technical writer | Review only |
| Sister repositories are read-only from here | Review: protocol § 2 | Review only |

## 5. Workflow, review, and merge gates

| Rule | Enforced by | Gaps |
| --- | --- | --- |
| OpenSpec changes declare `quick` or `design` track, keep `proposal.md` <= 60 lines with `Approved-by:`, and stay truthful against code | `task spec:check` (`openspec validate --strict` + `ateam state spec-check`); `TestSpecTaskWiring` | Truth against code and `design.md` length (warning past 120 lines) are review |
| Every OpenSpec task tickable in or before archive; unapproved proposals hold task `1.1` with `Hold: OPEN-Q proposal awaits owner approval`, and holds live inside the task they stop | `ateam state spec-check` (in `task spec:check`); `ateam review gate` refuses unapproved active proposals | Post-merge task phrasing is review only |
| Steps end on a local `task verify` and conclude with Ponytail's 1–2 line exit summary; reviewer `APPROVE` cites commit and local run, never CI | `scripts/verify.sh` (fails on tracked-file change); review | Ponytail footer presence and reviewer prose citation are review only |
| A merge needs `Required` and `Merge check` green on an up-to-date head | `main` ruleset + `scripts/merge-check.sh` (`ATEAM_REQUIRED_CHECKS=Required,Merge check` in `merge-check.yml`); `TestCIWorkflowPinned` | Ruleset `25161512` adds `Merge check` at `#36` merge time |
| No merge while a `class:flake` issue is open unless the PR closes all | `scripts/merge-check.sh` (`ateam review gate`); `TestMergeCheckWrapper` | None |
| Non-draft code PRs carry `implemented-by:` and `reviewed-by:` with `(model, effort)`, non-weaker cross-provider strength, and a head-bound `Review record` comment | `scripts/merge-check.sh` (`ateam review gate` + `.agents/review-policy.json`); `merge-check.yml` (`Merge check probe`) | Shared GitHub account identity between owner and agents is review/trust |
| Completed OpenSpec changes archive into `openspec/changes/archive/` before marking ready; merged claim worktrees and detached review checkouts are pruned promptly | Review: protocol § 2; `ateam state pickup` worktree labels; `ateam state prune` | Review only |
| Unfixed defects in repo code are filed as issues | Review: protocol § 2 "File"; reviewer | Review only |
| Unchosen-condition findings are `BLOCKING`/`HIGH` only with a real-world trigger | Review: reviewer § Finding and verdict format | Review only |
| A new rule names its enforcer and adds rows to both files | Review: reviewer; `TestAgentSetupParity` and `ateam doctor --check` (row counts) | One-to-one row meaning is review only |
| Clarify gate before claiming, mutating, or spawning | `ateam doctor --check` (protocol markers); review: protocol § 2, `.agents/README.md` | Runtime adherence is review only |
| Briefs carry intent and artifacts; resume vs fresh | Review: `.agents/README.md` § Orchestrating role agents | Review only |
| Model tier and effort routing | `TestAgentSetupParity` (Claude model, Codex model and effort); `.agents/review-policy.json` | Subagent runtime flag selection on Gemini is review only |
| `AGENTS.md` size, rules-index parity, `ateam` template parity, adapter parity | `task ateam:doctor`; `TestAgentSetupParity`; `TestAteamTemplatesUnchanged` | One-to-one row meaning is review only |
| Claim branches update by merge, never rebase or force-push | `ateam review gate` invalidates review records on rewritten history; review: protocol § 2 | Pre-review rebases are review only |
| Cited evidence can be opened elsewhere; local-only paths named | Review: `handoff` and `preflight` skills | Review only |
| Docs written for a working developer | Review: technical writer, reviewer | Review only |

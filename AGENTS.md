# SemVocab

Guidance for coding agents. Codex and Antigravity (Gemini) load this file as `AGENTS.md`; `CLAUDE.md` imports it, so
there is one copy (kept well under 24,000 bytes so rule loaders with a 24 KB per-file cap never truncate it). It
carries the facts an agent cannot derive from the tree and names where each rule lives. Per-platform command names and
model routing: `.agents/README.md`. Full enforcement audit matrix: `.agents/rules-index.md`. The setup is carried from
SemEngine's (`C360Studio/semengine`, `AGENTS.md` and `.agents/`) and adapted; where a rule differs, this tree is
authoritative.

## What this is for

SemVocab is the vocabulary-extension layer for SemEngine: ontology catalogs, loadable vocabulary packs, and a SHACL
compiler that turns shapes into SemEngine pack and projection specs ahead of time. It is Stage 2 of the
`semengine` -> `semvocab` -> `semsparql` ontology and SPARQL architecture: SemEngine owns the generic vocabulary
registry, `PackSpec`/`ProjectionSpec`, and RDF export; SemVocab owns domain catalogs (candidates: BFO, CCO, cyber,
DIEKM) and packs built on them; SemSPARQL owns query and projection expansion. No RDF or SHACL engine enters
SemEngine's hot ingest or query paths.

The bootstrap scope, phases, and acceptance criteria are issue #1 (`gh issue view 1`); read it before scoping work.
Material ported from SemStreams comes from the frozen pin `8b99efe9c66a4faa4fa509f9f62cc6bad8392128`, the same pin
SemEngine extracts from. The Go code in the tree today is the six catalogs (`catalog/bfo`, `catalog/cco`,
`catalog/sosa`, `catalog/ssn`, `catalog/swe`, and `catalog/sensorml`), the `packs` (`bfo-cco-core` and `swe-core`) and
`packs/diekm` packs, the `shacl` package (parses shapes, compiles them to a pack and projection contracts, and
validates complete entity state; see `docs/shacl-compilation.md`), the Turtle reader `internal/turtle` that `shacl`
uses, test and fixture helpers under `internal/`, and the repository contract tests under `internal/harness`. Export
projection and external SHACL validation (#1 Phase 3) live in SemSPARQL, not here (ruling Q34 on #1). `sh:rule`
translation (#1 Phase 4) is not present; it is #21, blocked on the upstream rule APIs.

## Commands

`task --list` shows every command with its rationale.

```bash
task doctor       # tool versions against their pins; starts no workloads
task fmt          # format Go sources (the only command that writes)
task fmt:check    # fail on formatting drift
task spec:check   # strict OpenSpec validation; fails on a Hold: outside every task
task spec:queue   # in-flight OpenSpec changes and their holds
task docs:check   # markdownlint
task tidy:check   # go.mod / go.sum tidy
task build        # build
task vet          # go vet
task lint         # pinned revive
task vuln         # pinned govulncheck
task test:unit    # unit tests once, under the race detector, at one CPU
task test:repeat  # unit tests five times at one CPU, without the race detector, shuffled
task merge:check  # -- <n>: fail on open flakes PR n leaves open, missing up-to-date rule, or missing review lines
task verify       # spec:check docs:check fmt:check tidy:check build vet lint vuln test:unit test:repeat;
                  # fails if tracked files changed
```

Run `task verify` before every implementation push. CI has three jobs: `verify` runs the same commands;
`merge-check` runs `scripts/merge-check.sh` (`task merge:check`: known flakes, up-to-date rule, cross-agent review
lines); `required` fails if either failed, is missing, or was skipped/cancelled. Gate selection per diff:
`.agents/skills/semvocab-preflight/SKILL.md`.

## Rules and what enforces them

Domain-grouped working reference. `.agents/rules-index.md` carries the full audit matrix (what enforces each rule and
what stays review only). Short names in "Where": `arch`, `dev`, `rev`, `writer` -> `.agents/contracts/semvocab-<role>.md`;
`protocol` -> `.agents/protocol.md`. In "Gate": `review` means review only (no command fails when broken); `verify`
means enforced by `task verify` (`verify+review` when a check covers the core rule and edge cases remain review only).
A new repo-wide or agent-conduct rule updates both this file and `.agents/rules-index.md`.

### 1. Go and package boundaries

| Gate | Rule | Where |
| --- | --- | --- |
| `review` | Production code imports no test library and nothing under `internal/harness` | `dev`, `rev`; `TestTestOnlyImports` (Rapid only) |
| `review` | Public packages' exported signatures name no `internal/` type | `dev`, `rev` |
| `review` | Importing a package registers nothing; packages export `Register…` functions and only a composition root calls them | `dev`, `rev` |
| `review` | Validation and compilation are side-effect free: they never mutate SemEngine's global `vocabulary` or `vocabulary/export` registries, and never snapshot/clear/register/restore them as a workaround | `dev`, `rev`; #1 |
| `review` | Failure paths fail closed; any skip, drop, or degrade is declared | `dev` |
| `verify` | The pinned revive passes (`revive.toml`; warnings fail) | `task lint` |

### 2. Vocabulary, packs, and catalogs

| Gate | Rule | Where |
| --- | --- | --- |
| `review` | SemEngine is the source of truth for standard constants and generic vocabulary behaviour it exports; SemVocab never copies an IRI string it exports or adds a second registry | `arch`, `dev`, `rev`; #1 |
| `review` | Each `catalog/<domain>` package records one disposition before code: SemEngine's symbol used directly, thin re-export with equality tests, new ontology content with source/version/license, or deferred with a reason | `arch`; #1 |
| `review` | Predicates are three dotted parts; entity-type keys are `<domain>.<type>`; entity IDs are six parts `org.platform.system.domain.type.instance` | `dev`, `rev` |
| `review` | Packs register only through SemEngine's pinned `RegisterPack(pack, opts...)` and prefix registrar; a rejected load leaves registry and prefix state unchanged | `dev`, `rev`; #1 |
| `review` | Ontology terms come from a named source and release; none is inferred from a numeric range or from namespace support, and a local mapping is never presented as an externally certified ontology | `arch`, `dev`, `rev`; #1 |
| `review` | The SHACL compiler rejects `sh:rule` and every construct outside its documented profile, naming it | `arch`, `dev`, `rev`; #1 |

### 3. Testing and verification

| Gate | Rule | Where |
| --- | --- | --- |
| `review` | Tests bind no fixed address or port | `dev`, `rev` |
| `review` | No `time.Sleep` in a test file or under `internal/harness/` | `dev`, `rev` |
| `review` | No test calls `Skip`/`Skipf`/`SkipNow`; no test file carries a build tag other than `integration` | `dev`, `rev` |
| `verify` | Unit tests run once under `-race` at one CPU, and five times at one CPU, without `-race`, shuffled | `Taskfile.yml`; `scripts/verify.sh` |
| `verify+review` | Tests use an independent oracle (an authoritative ontology fixture, SemEngine's own serializer, a W3C example) and are shown able to fail; structural guards carry paired sensitivity tests | `dev`, `rev`; `internal/harness/contract` |
| `review` | A test that touches SemEngine's global registries asserts their state, never restores it by hand to hide a leak | `dev`, `rev`; #1 |
| `review` | Generated checks record seed, completed checks, and replayable failure; mutation survivors are reported as survivors | `dev`, `rev` |

### 4. Porting, provenance, and dependencies

| Gate | Rule | Where |
| --- | --- | --- |
| `review` | Ported or extracted material has an admission-ledger row naming the full SemStreams pin or authoritative external source, file paths, symbols, license, and disposition | `arch`; `docs/admission-ledger.yaml`; #1 |
| `review` | The SemEngine dependency is pinned in `go.mod` to a merged commit; a statement about SemEngine behaviour cites that pin (`file:line`) | `arch`, `rev` |
| `review` | An upstream gap is filed on SemEngine (as semengine#256 was) and linked; it is never worked around with a local fork or a copied private symbol | `arch`, `dev`, `rev` |
| `review` | New surface needs a repo caller, a named consumer caller, or a contract line and a product test; production code carries no test-only back door | `arch`, `dev`, `rev` |
| `review` | Statements on how a library or standard (SemEngine, `yaml.v3`, W3C Turtle/JSON-LD/SHACL) parses, orders, or fails cite a probe at the pinned version or the spec section, or say it is not promised | `arch`, `dev`, `rev`, `writer` |
| `review` | Sister repositories (SemEngine, SemStreams, SemSPARQL, consumers) are read-only from here | `protocol` |

### 5. Workflow, review, and merge gates

| Gate | Rule | Where |
| --- | --- | --- |
| `spec:check+review` | OpenSpec changes and specs are well formed and truthful against code | "Where state lives" below |
| `spec:check+review` | Every OpenSpec task can be ticked in or before the archive commit; a hold is written as `Hold:` inside the unticked task it stops | `protocol`; `rev` |
| `verify+review` | Steps end when `task verify` passes locally on a clean tree; pushes batch at review/end; CI is checked once before merge; reviewer `PASS` cites commit + local `task verify`, never CI | `dev`, `rev`; `protocol` |
| `CI required` | A merge needs CI green on a head up to date with `main` | `protocol` |
| `merge:check` | No merge while a known flake (`class:flake` issue) is open, unless the PR closes every open one | `protocol`; `scripts/merge-check.sh` |
| `merge:check+review` | Code PRs require review by an agent that wrote no commits (`implemented-by:` and `reviewed-by:` naming `claude`, `codex`, or `gemini`); docs-only PRs take any agent's reviewer | `protocol`; `rev` |
| `review` | Any defect found in repo code not fixed in the change is filed as an issue, never left as a residual or a comment | `protocol`; `rev` |
| `review` | A finding needing an unchosen condition (library timing, OS state, mid-operation fault, test-only call sequence) is `BLOCKING`/`HIGH` only with a named real-world trigger; else `MEDIUM` at most | `rev`, `dev`; `protocol` |
| `review` | A new rule names what enforces it and adds its row to `AGENTS.md` and `.agents/rules-index.md` | `rev` |
| `review` | Before claiming work, mutating state, or spawning role subagents on an underspecified prompt, confirm Deliverable, Scope/non-goals, Definition of Done/verification, and Stop boundary | `protocol`; `.agents/README.md` |
| `review` | Role-agent briefs carry the owner's intent and the artifact itself, not a paraphrase; resume for continuity, start fresh for mechanics | `.agents/README.md` |
| `verify+review` | Architect, developer, and reviewer run on the frontier model tier; technical-writer on the lighter tier; effort defaults as `.agents/README.md` states | `.agents/README.md`; `TestAgentSetupParity` |
| `verify+review` | `AGENTS.md` stays under 24,000 bytes, matches `.agents/rules-index.md` in numbered domains and row counts, and all adapters match their canonical contracts | `.agents/README.md`; `TestAgentSetupParity` |
| `review` | Pushed claim branches update by merging `origin/main`, never by rebase or force-push | `protocol` |
| `review` | Evidence cited in a PR or task can be opened on another machine; git-ignored paths are named as local only | `semvocab-handoff`, `semvocab-preflight` skills |
| `review` | Docs are written for a working developer: coined terms defined at first use, no jargon, no marketing | `writer`, `rev` |

## Where state lives

- **GitHub issues:** what is wanted and decided; rulings as issue comments; labels `type:epic`,
  `status:needs-decision`, `status:blocked`, `class:flake`.
- **Draft PRs:** claims; `Closes #n` for a leaf issue, `Addresses #n` for an epic or a phase of one.
- **`openspec/changes/<id>/`:** target state (proposal, design, tasks, holds), archived on completion.
- **`openspec/specs/<capability>/`:** current truth, verified against code before it is written.
- **ADRs (`docs/adr/`, when the first is written):** why; irreversible choices and cross-repo contracts.

There is no `/tickets` state and no handoff document. Non-trivial work starts with an OpenSpec change before code.

## Working here (Claude, Codex, and Gemini)

The shared protocol is `.agents/protocol.md`. Read it before filing, taking, landing, or closing work. Four gates
never become a pointer:

- **Clarify:** before claiming work, mutating state, or spawning role subagents, read any named issue, PR, or file so
  you never ask what the repository already answers, then verify four boundaries are clear: **Deliverable** (answer,
  issue, OpenSpec design, docs-only PR, or code PR), **Scope/non-goals**, **Definition of Done/verification**, and
  **Stop boundary**. When a prompt leaves any of the four ambiguous, stop first, propose a concise 3–4 bullet
  recommended Definition of Done, and ask at most 1–3 targeted questions; once confirmed, run autonomously to that
  finish line without pausing on routine steps.
- **Claim:** a draft PR opened before the work, in its own worktree on an agent-prefixed branch
  (`git worktree add ../semvocab-wt/claude/<topic> -b claude/<topic> origin/main`; Codex uses `codex/`, Gemini uses
  `gemini/`, and Enjoy desktop sessions use `move_to_worktree` under `.worktrees/<branch>`). No draft PR, no claim.
- **Merge:** CI green on a head up to date with `main`; no known flake open (an open `class:flake` issue) unless the PR
  closes every open one; `task merge:check -- <n>` immediately before merging (`GITHUB_ACTIONS` unset);
  `implemented-by:` in the PR body (`claude`, `codex`, or `gemini` beside the model/persona); archive/spec sync as the
  last content commit; squash merge. A code PR (any changed file not Markdown or under `openspec/`, or a file under
  `.claude/agents/` or `.agents/contracts/`) also needs another agent's review and `reviewed-by:` in the PR body,
  without which `merge-check` fails once non-draft; a documents-only PR does not.
- **Close:** the squash merge of a PR that declared `Closes #n` is the authorization. A close with no merged PR behind
  it takes the owner's word on the issue.

Agents mutate only this repository. SemEngine, SemStreams, SemSPARQL, and other sister repositories are read-only;
an upstream need becomes an issue on that repository, filed by the owner or with the owner's go-ahead.

## Roles

Role agents are the default path for nontrivial work. Contracts: `.agents/contracts/`.

- `semvocab-architect` designs, inventory first; read-only.
- `semvocab-developer` implements: failing test, implementation, evidence.
- `semvocab-reviewer` reviews every nontrivial change independently before integration; read-only. For a code
  pull request the review of record is another agent's (`.agents/protocol.md`, "Cross-agent review").
- `semvocab-technical-writer` updates current docs and task truth with the approved code.

Binding rulings stay with the owner, on the issue.

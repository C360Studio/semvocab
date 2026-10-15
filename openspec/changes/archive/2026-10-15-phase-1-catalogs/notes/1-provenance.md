# Notes: tasks 1.1–1.3 (dependency and provenance)

Developer choices the design does not spell out.

## 1.1 SemEngine pin

- `go get github.com/c360studio/semengine@v0.1.0-alpha.2` also raised three indirect tool dependencies by minimum
  version selection (`golang.org/x/text` v0.42.0, `golang.org/x/tools` v0.51.0, `golang.org/x/telemetry`), and added
  `github.com/google/uuid` as indirect. No direct requirement changed.
- The ledger's `pins.semengine.includes` lists the two commits between alpha.1 and alpha.2 (`67fa136` #263,
  `11da472` #261) and `excludes` is empty. The `alpha.1` mentions in the `diekm` and `bfo-cco-core` `test_oracle`
  text now say `alpha.2`. `phase_gates.phase-1` still says alpha.1: that is when the gate opened, and it remains true.
- The IRI-bearing SemEngine files the ownership table cites (`standards.go`, `labels.go`, `hierarchy.go`,
  `relationships.go`, `iris.go`, `predicates.go`) are unchanged between alpha.1 and alpha.2
  (`git diff --stat e6022933 11da4721 -- <files>` is empty), so the ledger's `file:line` citations still hold.
- The `catalog-provenance` delta also modifies "Phase gates": its current scenario says phase 2 is `open`, which the
  ledger no longer says.
- The ledger's `diekm.referenced_terms_verified` used `bfo:` CURIEs; they now use `obo:`, the prefix the pack
  declares (P5), so the pack test can expand them with the pack's own prefixes.

## 1.2 Vendored oracles

- Location: `internal/oracle/testdata/bfo-2020/bfo-core.ttl` and
  `internal/oracle/testdata/cco-2.2/CommonCoreOntologiesMerged.ttl`, copied from the architect's downloads and
  matching the ledger sha256. `internal/oracle` checks the sha256 against the ledger's pin before parsing, on every
  load.
- BFO-2020 has no LICENSE file at `5ed46876`; the CC BY 4.0 grant is the file's own `dc:license`. The `NOTICE`
  gives the attribution CC BY 4.0 asks for (title, contributors from `dc11:contributor`, source, license link, no
  changes). CCO's `LICENSE` is fetched verbatim from `0bc7d33e`.

## 1.3 Ledger shape test

- `internal/ledger` decodes the ledger strictly (yaml.v3 `KnownFields(true)`), so an unknown or misspelt field fails
  with its name. Entry `source` blocks are decoded per entry (`BFO()`, `CCO()`), also strictly.
- `TestAdmissionLedgerShapeSensitivity` seeds nine violations (removed and duplicated entry, blank deferral reason,
  renamed and blank test oracle, unknown disposition, short revision, short sha256, missing phase gate) and requires
  each to be named.

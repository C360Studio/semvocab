# Notes: task group 3 (Turtle reader)

Written 2026-10-16 with go1.26.9 darwin/arm64.

## 3.1 Choices the design leaves open

- **API.** `turtle.Parse(r, opts ...Option)`, with one option, `WithBase(base)`. The signature is variadic, so the
  existing callers did not change. A base that is not absolute, or that has a fragment, makes `Parse` fail before it
  reads anything.
- **Resolution.** `resolve.go` implements RFC 3986 §5.2.2 (strict), §5.2.3 (merge), and §5.2.4 (remove dot segments)
  on strings. It does not use `net/url`, because `url.URL.String` percent-escapes non-ASCII characters, and that would
  change an IRI. An absolute IRI is returned unchanged, with no dot-segment removal, whether or not a base is set. That
  matches what the reader did before for absolute IRIs.
- **Where the base applies.** It applies to every IRIREF: subjects, predicates, objects, `@prefix` namespaces, and
  `^^<datatype>`. With a caller base, `@base` replaces the base, resolved against the current one (Turtle §6.3).
  Without a caller base, `@base` is still checked but changes nothing, so relative IRIs stay rejected. This keeps
  "shapes documents must use absolute IRIs" true even when a document declares `@base`.
- **Nesting.** `[` and `(` share one depth counter. The 65th opening bracket fails with its own line and column, and
  so does a subject-position `[`.
- **Size.** The reader has no size limit (ruling Q25). An earlier draft put the 1 MiB cap here, which failed the CCO
  v2.2 oracle (1,994,610 bytes) in `catalog/cco` and `packs`. That is why Q25 was raised.

## Evidence

- `TestBaseResolvesRFC3986Examples` checks all of RFC 3986 §5.4.1 and §5.4.2 (strict) against the RFC's base.
  `TestBaseApplies` covers prefixes, datatypes, `@base` with and without a caller base, and refused bases.
- `TestW3CTestSuiteWithBase` reads the 43 vendored cases through `oracle.ReadFixture`, which checks each sha256
  first. With a base, each case's distinct-triple count equals rdflib 7.1.4's (`notes/2-fixtures.md`). Without a base,
  each fails at `<>`. `TestW3CVocabulary` reads `shacl.ttl` without a base: 1127 triples, matching rdflib.
- `TestNestingLimit` covers 64 levels (accepted) and 65 (rejected with position) for blank nodes, collections, the
  two mixed, and subject position. `TestNestingLimitBoundsDeepInput` runs probe P1's densest 1 MiB input, which now
  fails at level 65.
- Shown able to fail, each by a seeded change that was then reverted:
  - depth check `> maxNesting` instead of `== maxNesting`: four nesting tests fail;
  - `merge` without dot-segment removal: 17 RFC examples fail.
- The whole suite passes. That includes the BFO and CCO oracles and the W3C Turtle examples in `turtle_test.go`.

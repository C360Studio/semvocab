# Notes: task group 1 (a shape in both roles)

Written 2026-10-17 with go1.26.9 darwin/arm64, at `8442f12` plus this change. The ruling is Q32 on issue #10. The
choices below are the ones the ruling leaves open.

## Choices

- **Which construct the issue names.** The record's construct is `sh:property`, on the dual-role IRI, with no path.
  The second role comes from `sh:property`. The IRI may carry several target predicates, so naming one of them
  would pick one arbitrarily.
- **What makes a node shape.** The test is Q27's definition: a subject typed `sh:NodeShape` or carrying any target
  predicate. Q32's message names a target. For an IRI that is only typed `sh:NodeShape`, the message says
  `it is typed sh:NodeShape` and ends `a property shape typed sh:NodeShape is outside the profile`. Without this
  case, such an IRI would still be built in both roles, which is the defect #10 reports.
- **Referrers.** A referrer is a node shape (Q27) whose `sh:property` object is the IRI, including the IRI itself.
  Every referrer is named, sorted. A referrer that is itself in both roles is still named, even though it is not
  built.
- **Message.** The record carries the whole message (`rejection.Message`), so `Compile` reports it in place of the
  profile's wording for `sh:property`. IRIs are written `<iri>`, as in other records. Target predicates are written
  `sh:<name>`.
- **What the record owns.** The IRI's own triples draw no other record. Blank nodes under them, such as a nested
  `sh:property [ … ]`, an `sh:in` list, or an `sh:target [ … ]`, belong to the one record. That is the parser's
  existing ownership rule for a recorded construct. A named shape under the IRI is not owned. If nothing else uses
  it, its triples are recorded as strays under its own IRI, as for a named shape under any rejected construct.
- **The referring node shape.** It is built with its other property shapes and has no entry for the dual-role IRI.
  `Compile` still fails because of the record, and returns no bundle. A failed compile has nothing to drop.

## Evidence

- **1.1:** `TestDualRoleShapeIssue10` is #10's example, verbatim, in both statement orders. At `8442f12` it fails
  with four issues on `ex:P` (`sh:minCount`, `sh:path`, `rdf:type`, `sh:targetClass`). After the fix there is one.
- **1.2:**
  - `TestDualRoleShapeTwoReferrers` covers two referrers, an `sh:in` list and a blank property shape under the IRI,
    and a referrer's other property (`ex:q`, `sh:minCount 1`). The dual-role issue is the only issue, and `Compile`
    returns no bundle.
  - `TestDualRoleShapeEachNodeShapeMarker` covers `sh:targetNode`, `sh:targetSubjectsOf`, `sh:targetObjectsOf`,
    `sh:target` with a blank value, two targets, the `sh:NodeShape` type alone, and a self-reference. Each runs in
    both statement orders.
  - Seeded changes, each reverted. Building the IRI as a node shape fails all three tests. Building it as a property
    shape fails all three. Leaving the referrers unsorted fails `TestDualRoleShapeTwoReferrers` in reversed order.
    Not owning the IRI's blank nodes fails `TestDualRoleShapeTwoReferrers` and the `sh:target` case.
- **1.3:** `go test ./...` passes, including `TestW3CRejectedFamiliesFailCompile`, `TestW3CRejectedFamiliesRecorded`,
  the W3C path cases, and the Rapid properties. A throwaway probe loaded all 43 vendored W3C core files through the
  test-suite loader. None produces a dual-role record, so no W3C expectation changed.

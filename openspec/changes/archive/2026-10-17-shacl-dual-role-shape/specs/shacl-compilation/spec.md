# shacl-compilation

## ADDED Requirements

### Requirement: A shape in both roles is rejected

An IRI that is a node shape and also the object of a node shape's `sh:property` SHALL be built in neither role.
`Compile` SHALL fail with exactly one issue for it. The issue SHALL name the IRI, what makes it a node shape, and
every node shape that refers to it, whatever the statement order.

#### Scenario: A property shape with its own target

- **WHEN** `ex:P` has `sh:targetClass ex:C`, `sh:path ex:p`, and `sh:minCount 1`, and `ex:A` has
  `sh:property ex:P`
- **THEN** compilation fails with one issue on `ex:P` that names `sh:targetClass` and `ex:A`, and no other issue is
  recorded against `ex:P`

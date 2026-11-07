# shacl-compilation

## MODIFIED Requirements

### Requirement: Constructs outside the profile are rejected by name

`Compile` SHALL fail on every construct outside the profile, including every `sh:rule` input, unsupported targets,
non-predicate paths, logical and SPARQL constraints, and unknown `sh:` terms. Each issue SHALL name the construct's
IRI, its shape, and its path. A node shape with no `sh:targetClass` SHALL also fail. `Compile` SHALL report every
issue and SHALL return no partial bundle.

#### Scenario: A shape with a rule

- **WHEN** a shapes graph contains `sh:rule`
- **THEN** compilation fails, the issue names `sh:rule` and the shape, and the message says rule translation is
  deferred to issue #21

#### Scenario: Several rejected constructs

- **WHEN** a W3C test-suite case uses `sh:targetNode` and `sh:and`
- **THEN** the error lists both constructs

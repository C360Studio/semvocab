# shacl-compilation

## ADDED Requirements

### Requirement: Bounded JSON and YAML input

`DecodeShapesJSON`, `DecodeShapesYAML`, `DecodeMappingJSON`, and `DecodeMappingYAML` SHALL fail on input over 1 MiB
with the byte offset where the limit was crossed, as `ParseTurtle` does, and SHALL accept input of exactly 1 MiB.

#### Scenario: A mapping document one byte over the limit

- **WHEN** a YAML mapping document is 1,048,577 bytes long
- **THEN** decoding fails, naming the 1,048,576-byte limit and byte offset 1,048,576

#### Scenario: A shapes document at the limit

- **WHEN** a JSON shapes document is exactly 1,048,576 bytes long
- **THEN** it decodes as it would without the cap

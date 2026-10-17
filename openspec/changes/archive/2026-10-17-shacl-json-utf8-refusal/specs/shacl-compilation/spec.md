# shacl-compilation

## ADDED Requirements

### Requirement: JSON text is never replaced

`DecodeShapesJSON` and `DecodeMappingJSON` SHALL refuse input that is not valid UTF-8, and any `\u` escape naming a
lone UTF-16 surrogate, with the byte offset of the first such byte or escape. They SHALL NOT decode either into
U+FFFD.

#### Scenario: A malformed byte in an enumeration member

- **WHEN** a JSON shapes document has the byte `0xff` inside an `in` member
- **THEN** decoding fails with that byte's offset

#### Scenario: A lone surrogate escape

- **WHEN** a JSON mapping document has `"\ud800"` as a predicate name
- **THEN** decoding fails with the escape's offset, and `"😀"` in the same place still decodes

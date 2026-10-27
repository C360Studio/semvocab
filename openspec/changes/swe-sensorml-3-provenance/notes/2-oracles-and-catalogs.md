# Notes: tasks 2.1–2.3 (oracles and catalogs)

## Implementation choices not spelled out in the design

- **Typed JSON Schema AST (`internal/oracle/schema.go`).** Across all 47 `.json` files in `swe-3.0` and `sensorml-3.0`,
  every JSON Schema keyword (`type`, `$ref`, `const`, `properties`, `$defs`, `allOf`, `oneOf`, `anyOf`, `items`) has a
  single JSON type. Decoding into `schemaNode` avoids untyped map walks and naturally prevents `Event.json:48` (which
  declares a domain property literally named `"properties"`) from having its nested schema attributes (`description`,
  `type`, `items`) mistaken for domain property names.
- **Implicit object classes and pure `$ref` aliases in OGC 3.0 schemas.** Four classes omit top-level `"type": "object"`
  while declaring `"properties"` or inheriting object type via `allOf`: `DataArray.json`, `Deployment.json`, and
  `Pose.json#/$defs/{PositionGeo,PositionXYZ,AnglesYPR,Quaternion}`. Sub-branches in `Pose.json` (`GeoPose`,
  `RelativePose`) and `AggregateProcess.json` (`Link`) carry `"properties": {"type": {"const": "<Class>"}}`. Meanwhile,
  `sweCommonDefs.json` and `commonDefs.json` expose pure `$ref` aliases (`Ref != ""` with no `Type`, `Properties`,
  `AllOf`, `OneOf`, or `AnyOf`). `isPureRefAlias` excludes those wrapper aliases from standalone object class matching
  while `resolveRef` follows relative `$ref` chains across `common/`, `swecommon/`, and `sensorml/` to the canonical
  target (for example `swecommon/schemas/json/sweCommonDefs.json#/$defs/AbstractSWEIdentifiable` ->
  `swecommon/schemas/json/AbstractSweIdentifiable.json`).
- **Property inheritance closure in `DeclaredProperties`.** `DeclaredProperties` walks `properties`, `allOf`, `oneOf`,
  `anyOf`, and resolved `$ref` targets recursively (`visited` guarded) so both locally declared and inherited properties
  are visible, while catalog parity tests check each ledger-recorded property directly on its recorded `schema_path` and
  verify that every domain schema file in `swecommon/schemas/json/` and `sensorml/schemas/json/` has full class and
  property coverage in the admission ledger.
- **`ssn.DirectParents` slice immutability.** `ssn.DirectParents(iri)` returns `slices.Clone(directParents[iri])` so a
  caller mutating the returned slice (`sosa.Sample` has two parents: `sosa.FeatureOfInterest` and `sosa.Result`) cannot
  corrupt package hierarchy state; `TestCatalogHierarchyMatchesRelease` asserts both multi-parent order and slice copy
  isolation.

## Shown able to fail (seeded, observed, and reverted)

- `internal/oracle/schema.go`: skipping `VerifySHA256` in `readSchemaFile` fails
  `TestReadSchemaFileRefusesAlteredOrUnlisted` (`altered copy: err = <nil>, want a sha256 mismatch`).
- `catalog/sosa/sosa.go`: changing `Observation` to `Namespace + "Observing"` fails `TestCatalogParity` naming the
  ledger mismatch, missing `owl:Class` in `sosa.ttl`, and missing `rdfs:label`.
- `catalog/ssn/hierarchy.go`: dropping `sosa.FeatureOfInterest` from `sosa.Sample` in `directParents` fails
  `TestCatalogHierarchyMatchesRelease` on `DirectParents`, `Parents`, and `IsA(sosa.Sample, sosa.FeatureOfInterest)`.
- `catalog/swe/hierarchy.go`: changing `Quantity`'s parent from `AbstractSimpleComponent` to `AbstractDataComponent`
  fails `TestCatalogHierarchyMatchesSchema` on `SubClassOf`, `Parents`, and `IsA`.
- `catalog/sensorml/hierarchy.go`: removing delegation to `swe.SubClassOf` in `SubClassOf` fails
  `TestCatalogHierarchyMatchesSchema` on `Parents` and `IsA(..., swe.AbstractSWE)` across all seven classes extending
  `swe.AbstractSweIdentifiable`.
- Permanent sensitivity suites: `TestReadSchemaFileRefusesAlteredOrUnlisted`, `sosa.TestCatalogParitySensitivity`,
  `ssn.TestCatalogParitySensitivity`, `ssn.TestCatalogHierarchySensitivity`, `swe.TestCatalogParitySensitivity`,
  `swe.TestCatalogHierarchySensitivity`, `sensorml.TestCatalogParitySensitivity`, and
  `sensorml.TestCatalogHierarchySensitivity`.

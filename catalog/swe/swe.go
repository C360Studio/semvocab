// Package swe is SemVocab's catalog of OGC SWE Common Data Model 3.0 terms:
// IRI constants for the 35 object classes and 55 JSON properties admitted from
// the Draft 2020-12 JSON Schema bundle, and their JSON Schema allOf inheritance
// hierarchy (SubClassOf, Parents, IsA).
//
// Source and admission: the terms derive from OGC 24-014 (OGC SWE Common Data
// Model Encoding Standard, edition 3.0.0) in
// opengeospatial/ogcapi-connected-systems tag v1.0.0 (commit
// 8e03b236a049849f2ccc24b4fd9fdce5ff69bed2, swecommon/schemas/json/*.json), as
// listed in docs/admission-ledger.yaml (entry "swe", schema bundle "swe-3.0").
// The schemas are licensed under the OGC Software License; the license text and
// attribution notice are kept with the vendored schemas in
// internal/oracle/testdata/ogcapi-connected-systems/LICENSE and NOTICE.
//
// Namespace honesty (owner ruling R3 on semvocab#26): Namespace
// ("http://www.opengis.net/spec/SWE/3.0/") is an unregistered C360 RDF
// convention over OGC SWE Common 3.0's Clause 5 normative specification base
// URI ("http://www.opengis.net/spec/SWE/3.0"). OGC 24-014 publishes Draft
// 2020-12 JSON Schemas rather than an OWL ontology; these IRIs are not an
// externally certified or dereferenceable OGC OWL ontology.
//
// Identifier convention: object classes use PascalCase Go identifiers matching
// their JSON Schema file, type, or $defs name (preserving
// AbstractSweIdentifiable from AbstractSweIdentifiable.json), while JSON
// property keys use Prop<PascalCase> Go identifiers (such as PropUOM and
// PropNamespace) so property constants never collide with Namespace or class
// constants. Every IRI preserves its exact JSON Schema local name.
//
// Importing this package registers nothing. The vocabulary pack that declares
// the swe prefix is packs.RegisterSWECore.
package swe

// Namespace is the C360 RDF IRI stem for OGC SWE Common 3.0 terms, formed by
// appending "/" to OGC 24-014's Clause 5 specification base URI
// ("http://www.opengis.net/spec/SWE/3.0"). It is not an externally certified
// OGC OWL ontology namespace (owner ruling R3 on semvocab#26).
const Namespace = "http://www.opengis.net/spec/SWE/3.0/"

// Object classes.
const (
	// AbstractSWE is the base SWE Common object schema (basicTypes.json#/$defs/AbstractSWE).
	AbstractSWE = Namespace + "AbstractSWE"
	// AbstractSweIdentifiable is the identifiable SWE Common object schema (AbstractSweIdentifiable.json).
	AbstractSweIdentifiable = Namespace + "AbstractSweIdentifiable"
	// AbstractDataComponent is the abstract base schema for data components (AbstractDataComponent.json).
	AbstractDataComponent = Namespace + "AbstractDataComponent"
	// AbstractSimpleComponent is the abstract base schema for simple scalar and range components (AbstractSimpleComponent.json).
	AbstractSimpleComponent = Namespace + "AbstractSimpleComponent"
	// Boolean is the boolean scalar data component (Boolean.json).
	Boolean = Namespace + "Boolean"
	// Count is the integer count scalar data component (Count.json).
	Count = Namespace + "Count"
	// Quantity is the continuous numerical scalar data component with unit of measure (Quantity.json).
	Quantity = Namespace + "Quantity"
	// Time is the temporal scalar data component (Time.json).
	Time = Namespace + "Time"
	// Category is the categorical token scalar data component (Category.json).
	Category = Namespace + "Category"
	// Text is the free-text scalar data component (Text.json).
	Text = Namespace + "Text"
	// CountRange is the integer count range component (CountRange.json).
	CountRange = Namespace + "CountRange"
	// QuantityRange is the numerical quantity range component (QuantityRange.json).
	QuantityRange = Namespace + "QuantityRange"
	// TimeRange is the temporal range component (TimeRange.json).
	TimeRange = Namespace + "TimeRange"
	// CategoryRange is the categorical token range component (CategoryRange.json).
	CategoryRange = Namespace + "CategoryRange"
	// DataRecord is the composite record data component (DataRecord.json).
	DataRecord = Namespace + "DataRecord"
	// Vector is the coordinate vector data component (Vector.json).
	Vector = Namespace + "Vector"
	// AbstractArray is the abstract array data component (DataArray.json#/$defs/AbstractArray).
	AbstractArray = Namespace + "AbstractArray"
	// DataArray is the variable- or fixed-size array data component (DataArray.json).
	DataArray = Namespace + "DataArray"
	// Matrix is the matrix array data component (Matrix.json).
	Matrix = Namespace + "Matrix"
	// DataChoice is the discriminated union data component (DataChoice.json).
	DataChoice = Namespace + "DataChoice"
	// Geometry is the GeoJSON geometry data component (Geometry.json).
	Geometry = Namespace + "Geometry"
	// DataStream is the repeated-element data stream schema (DataStream.json).
	DataStream = Namespace + "DataStream"
	// UnitReference is the unit-of-measure reference object (basicTypes.json#/$defs/UnitReference).
	UnitReference = Namespace + "UnitReference"
	// AllowedTokens is the token value/pattern constraint object (basicTypes.json#/$defs/AllowedTokens).
	AllowedTokens = Namespace + "AllowedTokens"
	// AllowedValues is the numeric value/interval constraint object (basicTypes.json#/$defs/AllowedValues).
	AllowedValues = Namespace + "AllowedValues"
	// AllowedTimes is the temporal value/interval constraint object (basicTypes.json#/$defs/AllowedTimes).
	AllowedTimes = Namespace + "AllowedTimes"
	// SoftNamedProperty is the named property wrapper object (basicTypes.json#/$defs/SoftNamedProperty).
	SoftNamedProperty = Namespace + "SoftNamedProperty"
	// AssociationAttributeGroup is the XLink-style association attribute group object (basicTypes.json#/$defs/AssociationAttributeGroup).
	AssociationAttributeGroup = Namespace + "AssociationAttributeGroup"
	// AbstractEncoding is the abstract base encoding object (encodings.json#/$defs/AbstractEncoding).
	AbstractEncoding = Namespace + "AbstractEncoding"
	// TextEncoding is the delimiter-separated text encoding object (encodings.json#/$defs/TextEncoding).
	TextEncoding = Namespace + "TextEncoding"
	// JSONEncoding is the JSON encoding object (encodings.json#/$defs/JSONEncoding).
	JSONEncoding = Namespace + "JSONEncoding"
	// XMLEncoding is the XML encoding object (encodings.json#/$defs/XMLEncoding).
	XMLEncoding = Namespace + "XMLEncoding"
	// BinaryEncoding is the binary stream encoding object (encodings.json#/$defs/BinaryEncoding).
	BinaryEncoding = Namespace + "BinaryEncoding"
	// Block is the binary block encoding member object (encodings.json#/$defs/Block).
	Block = Namespace + "Block"
	// Component is the binary scalar encoding member object (encodings.json#/$defs/Component).
	Component = Namespace + "Component"
)

// JSON Schema properties (Prop<PascalCase> avoids colliding with Namespace and
// class constants while keeping exact JSON property local names).
const (
	// PropArcrole is JSON property "arcrole".
	PropArcrole = Namespace + "arcrole"
	// PropAxisID is JSON property "axisID".
	PropAxisID = Namespace + "axisID"
	// PropBitLength is JSON property "bitLength".
	PropBitLength = Namespace + "bitLength"
	// PropBlockSeparator is JSON property "blockSeparator".
	PropBlockSeparator = Namespace + "blockSeparator"
	// PropByteEncoding is JSON property "byteEncoding".
	PropByteEncoding = Namespace + "byteEncoding"
	// PropByteLength is JSON property "byteLength".
	PropByteLength = Namespace + "byteLength"
	// PropByteOrder is JSON property "byteOrder".
	PropByteOrder = Namespace + "byteOrder"
	// PropChoiceValue is JSON property "choiceValue".
	PropChoiceValue = Namespace + "choiceValue"
	// PropCode is JSON property "code".
	PropCode = Namespace + "code"
	// PropCodeSpace is JSON property "codeSpace".
	PropCodeSpace = Namespace + "codeSpace"
	// PropCollapseWhiteSpaces is JSON property "collapseWhiteSpaces".
	PropCollapseWhiteSpaces = Namespace + "collapseWhiteSpaces"
	// PropCompression is JSON property "compression".
	PropCompression = Namespace + "compression"
	// PropConstraint is JSON property "constraint".
	PropConstraint = Namespace + "constraint"
	// PropCoordinates is JSON property "coordinates".
	PropCoordinates = Namespace + "coordinates"
	// PropDataType is JSON property "dataType".
	PropDataType = Namespace + "dataType"
	// PropDecimalSeparator is JSON property "decimalSeparator".
	PropDecimalSeparator = Namespace + "decimalSeparator"
	// PropDefinition is JSON property "definition".
	PropDefinition = Namespace + "definition"
	// PropDescription is JSON property "description".
	PropDescription = Namespace + "description"
	// PropElementCount is JSON property "elementCount".
	PropElementCount = Namespace + "elementCount"
	// PropElementType is JSON property "elementType".
	PropElementType = Namespace + "elementType"
	// PropEncoding is JSON property "encoding".
	PropEncoding = Namespace + "encoding"
	// PropEncryption is JSON property "encryption".
	PropEncryption = Namespace + "encryption"
	// PropFields is JSON property "fields".
	PropFields = Namespace + "fields"
	// PropGeomTypes is JSON property "geomTypes".
	PropGeomTypes = Namespace + "geomTypes"
	// PropHref is JSON property "href".
	PropHref = Namespace + "href"
	// PropID is JSON property "id".
	PropID = Namespace + "id"
	// PropIntervals is JSON property "intervals".
	PropIntervals = Namespace + "intervals"
	// PropItems is JSON property "items".
	PropItems = Namespace + "items"
	// PropLabel is JSON property "label".
	PropLabel = Namespace + "label"
	// PropLocalFrame is JSON property "localFrame".
	PropLocalFrame = Namespace + "localFrame"
	// PropMembers is JSON property "members".
	PropMembers = Namespace + "members"
	// PropName is JSON property "name".
	PropName = Namespace + "name"
	// PropNamespace is JSON property "namespace" (on XMLEncoding).
	PropNamespace = Namespace + "namespace"
	// PropNilValues is JSON property "nilValues".
	PropNilValues = Namespace + "nilValues"
	// PropOptional is JSON property "optional".
	PropOptional = Namespace + "optional"
	// PropPaddingBytesAfter is JSON property "paddingBytes-after".
	PropPaddingBytesAfter = Namespace + "paddingBytes-after"
	// PropPaddingBytesBefore is JSON property "paddingBytes-before".
	PropPaddingBytesBefore = Namespace + "paddingBytes-before"
	// PropPattern is JSON property "pattern".
	PropPattern = Namespace + "pattern"
	// PropReason is JSON property "reason".
	PropReason = Namespace + "reason"
	// PropRecordsAsArrays is JSON property "recordsAsArrays".
	PropRecordsAsArrays = Namespace + "recordsAsArrays"
	// PropRef is JSON property "ref".
	PropRef = Namespace + "ref"
	// PropReferenceFrame is JSON property "referenceFrame".
	PropReferenceFrame = Namespace + "referenceFrame"
	// PropReferenceTime is JSON property "referenceTime".
	PropReferenceTime = Namespace + "referenceTime"
	// PropRole is JSON property "role".
	PropRole = Namespace + "role"
	// PropSignificantBits is JSON property "significantBits".
	PropSignificantBits = Namespace + "significantBits"
	// PropSignificantFigures is JSON property "significantFigures".
	PropSignificantFigures = Namespace + "significantFigures"
	// PropSRS is JSON property "srs".
	PropSRS = Namespace + "srs"
	// PropSymbol is JSON property "symbol".
	PropSymbol = Namespace + "symbol"
	// PropTitle is JSON property "title".
	PropTitle = Namespace + "title"
	// PropTokenSeparator is JSON property "tokenSeparator".
	PropTokenSeparator = Namespace + "tokenSeparator"
	// PropUOM is JSON property "uom".
	PropUOM = Namespace + "uom"
	// PropUpdatable is JSON property "updatable".
	PropUpdatable = Namespace + "updatable"
	// PropValue is JSON property "value".
	PropValue = Namespace + "value"
	// PropValues is JSON property "values".
	PropValues = Namespace + "values"
	// PropVectorsAsArrays is JSON property "vectorsAsArrays".
	PropVectorsAsArrays = Namespace + "vectorsAsArrays"
)

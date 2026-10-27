// Package sensorml is SemVocab's catalog of OGC SensorML 3.0 terms: IRI
// constants for the 35 object classes and 94 JSON properties admitted from the
// Draft 2020-12 JSON Schema bundle, and their JSON Schema allOf inheritance
// hierarchy (SubClassOf, Parents, IsA), which crosses into catalog/swe.
//
// Source and admission: the terms derive from OGC 23-000 (OGC SensorML: Model
// and XML/JSON Encoding Standard, edition 3.0.0) in
// opengeospatial/ogcapi-connected-systems tag v1.0.0 (commit
// 8e03b236a049849f2ccc24b4fd9fdce5ff69bed2, sensorml/schemas/json/*.json), as
// listed in docs/admission-ledger.yaml (entry "sensorml", schema bundle
// "sensorml-3.0"). The schemas are licensed under the OGC Software License; the
// license text and attribution notice are kept with the vendored schemas in
// internal/oracle/testdata/ogcapi-connected-systems/LICENSE and NOTICE.
//
// Namespace honesty (owner ruling R3 on semvocab#26): Namespace
// ("http://www.opengis.net/spec/sensorML/3.0/") is an unregistered C360 RDF
// convention over OGC SensorML 3.0's Clause 5 normative specification base URI
// ("http://www.opengis.net/spec/sensorML/3.0"). OGC 23-000 publishes Draft
// 2020-12 JSON Schemas rather than an OWL ontology; these IRIs are not an
// externally certified or dereferenceable OGC OWL ontology.
//
// Identifier convention: object classes use PascalCase Go identifiers matching
// their JSON Schema file, type, or $defs name (such as PhysicalSystem and
// Link), while JSON property keys use Prop<PascalCase> Go identifiers (such as
// PropLink and PropOrganisationName, preserving upstream spelling) so property
// constants never collide with Namespace or class constants. Every IRI
// preserves its exact JSON Schema local name.
//
// Importing this package registers nothing. The vocabulary pack that declares
// the sml prefix is packs.RegisterSWECore.
package sensorml

// Namespace is the C360 RDF IRI stem for OGC SensorML 3.0 terms, formed by
// appending "/" to OGC 23-000's Clause 5 specification base URI
// ("http://www.opengis.net/spec/sensorML/3.0"). It is not an externally
// certified OGC OWL ontology namespace (owner ruling R3 on semvocab#26).
const Namespace = "http://www.opengis.net/spec/sensorML/3.0/"

// Object classes.
const (
	// DescribedObject is the base SensorML described-object schema (DescribedObject.json).
	DescribedObject = Namespace + "DescribedObject"
	// AbstractProcess is the abstract base schema for processes and systems (AbstractProcess.json).
	AbstractProcess = Namespace + "AbstractProcess"
	// AbstractPhysicalProcess is the abstract base schema for physical components and systems (AbstractPhysicalProcess.json).
	AbstractPhysicalProcess = Namespace + "AbstractPhysicalProcess"
	// SimpleProcess is the non-physical atomic process schema (SimpleProcess.json).
	SimpleProcess = Namespace + "SimpleProcess"
	// AggregateProcess is the non-physical composite process network schema (AggregateProcess.json).
	AggregateProcess = Namespace + "AggregateProcess"
	// PhysicalComponent is the atomic physical device/sensor/actuator schema (PhysicalComponent.json).
	PhysicalComponent = Namespace + "PhysicalComponent"
	// PhysicalSystem is the composite physical system schema (PhysicalSystem.json).
	PhysicalSystem = Namespace + "PhysicalSystem"
	// Deployment is the system deployment schema (Deployment.json).
	Deployment = Namespace + "Deployment"
	// DeployedSystem is the deployed-system member schema (DeployedSystem.json).
	DeployedSystem = Namespace + "DeployedSystem"
	// DerivedProperty is the derived observable property schema (DerivedProperty.json).
	DerivedProperty = Namespace + "DerivedProperty"
	// Mode is the predefined configuration mode schema (Mode.json).
	Mode = Namespace + "Mode"
	// Settings is the configuration parameter/mode/status settings schema (Settings.json).
	Settings = Namespace + "Settings"
	// GeoPose is the geographic pose schema (Pose.json).
	GeoPose = Namespace + "GeoPose"
	// RelativePose is the local reference-frame relative pose schema (Pose.json).
	RelativePose = Namespace + "RelativePose"
	// PositionGeo is the geographic lat/lon/h position object (Pose.json#/$defs/PositionGeo).
	PositionGeo = Namespace + "PositionGeo"
	// PositionXYZ is the Cartesian x/y/z position object (Pose.json#/$defs/PositionXYZ).
	PositionXYZ = Namespace + "PositionXYZ"
	// AnglesYPR is the yaw/pitch/roll Euler angles object (Pose.json#/$defs/AnglesYPR).
	AnglesYPR = Namespace + "AnglesYPR"
	// Quaternion is the x/y/z/w orientation quaternion object (Pose.json#/$defs/Quaternion).
	Quaternion = Namespace + "Quaternion"
	// Event is the time-tagged history event schema (Event.json).
	Event = Namespace + "Event"
	// ResponsibleParty is the responsible individual or organization contact schema (ResponsibleParty.json).
	ResponsibleParty = Namespace + "ResponsibleParty"
	// Contact is the contact details object (ResponsibleParty.json#/$defs/Contact).
	Contact = Namespace + "Contact"
	// Phone is the voice/facsimile phone numbers object (ResponsibleParty.json#/$defs/Phone).
	Phone = Namespace + "Phone"
	// Address is the postal and electronic mail address object (ResponsibleParty.json#/$defs/Address).
	Address = Namespace + "Address"
	// ContactLink is the external contact link wrapper schema (ContactLink.json).
	ContactLink = Namespace + "ContactLink"
	// Document is the external documentation reference schema (Document.json).
	Document = Namespace + "Document"
	// LegalConstraint is the legal/copyright/access constraint schema (LegalConstraint.json).
	LegalConstraint = Namespace + "LegalConstraint"
	// CodeListValue is the codespace-qualified code value object (LegalConstraint.json#/$defs/CodeListValue).
	CodeListValue = Namespace + "CodeListValue"
	// CharacteristicList is a conditional group of system characteristics (DescribedObject.json#/$defs/CharacteristicList).
	CharacteristicList = Namespace + "CharacteristicList"
	// CapabilityList is a conditional group of system capabilities (DescribedObject.json#/$defs/CapabilityList).
	CapabilityList = Namespace + "CapabilityList"
	// SpatialFrame is a local Cartesian spatial reference frame (AbstractPhysicalProcess.json#/$defs/SpatialFrame).
	SpatialFrame = Namespace + "SpatialFrame"
	// TemporalFrame is a local temporal reference frame (AbstractPhysicalProcess.json#/$defs/TemporalFrame).
	TemporalFrame = Namespace + "TemporalFrame"
	// ProcessMethod is the algorithm/description method of a simple or physical process (SimpleProcess.json#/$defs/ProcessMethod).
	ProcessMethod = Namespace + "ProcessMethod"
	// ObservableProperty is an inline observable property definition (commonDefs.json#/$defs/ObservableProperty).
	ObservableProperty = Namespace + "ObservableProperty"
	// Term is a semantic identifier or classifier term (commonDefs.json#/$defs/Term).
	Term = Namespace + "Term"
	// Link is an external sub-process/component link member (AggregateProcess.json).
	Link = Namespace + "Link"
)

// JSON Schema properties (Prop<PascalCase> avoids colliding with Namespace and
// class constants while keeping exact JSON property local names).
const (
	// PropAccessConstraints is JSON property "accessConstraints".
	PropAccessConstraints = Namespace + "accessConstraints"
	// PropAddress is JSON property "address".
	PropAddress = Namespace + "address"
	// PropAdministrativeArea is JSON property "administrativeArea".
	PropAdministrativeArea = Namespace + "administrativeArea"
	// PropAlgorithm is JSON property "algorithm".
	PropAlgorithm = Namespace + "algorithm"
	// PropAngles is JSON property "angles".
	PropAngles = Namespace + "angles"
	// PropAttachedTo is JSON property "attachedTo".
	PropAttachedTo = Namespace + "attachedTo"
	// PropAxes is JSON property "axes".
	PropAxes = Namespace + "axes"
	// PropBaseProperty is JSON property "baseProperty".
	PropBaseProperty = Namespace + "baseProperty"
	// PropCapabilities is JSON property "capabilities".
	PropCapabilities = Namespace + "capabilities"
	// PropCharacteristics is JSON property "characteristics".
	PropCharacteristics = Namespace + "characteristics"
	// PropCity is JSON property "city".
	PropCity = Namespace + "city"
	// PropClassifiers is JSON property "classifiers".
	PropClassifiers = Namespace + "classifiers"
	// PropCodeSpace is JSON property "codeSpace".
	PropCodeSpace = Namespace + "codeSpace"
	// PropComponents is JSON property "components".
	PropComponents = Namespace + "components"
	// PropConditions is JSON property "conditions".
	PropConditions = Namespace + "conditions"
	// PropConfiguration is JSON property "configuration".
	PropConfiguration = Namespace + "configuration"
	// PropConnections is JSON property "connections".
	PropConnections = Namespace + "connections"
	// PropContactInfo is JSON property "contactInfo".
	PropContactInfo = Namespace + "contactInfo"
	// PropContactInstructions is JSON property "contactInstructions".
	PropContactInstructions = Namespace + "contactInstructions"
	// PropContacts is JSON property "contacts".
	PropContacts = Namespace + "contacts"
	// PropCountry is JSON property "country".
	PropCountry = Namespace + "country"
	// PropDefinition is JSON property "definition".
	PropDefinition = Namespace + "definition"
	// PropDeliveryPoint is JSON property "deliveryPoint".
	PropDeliveryPoint = Namespace + "deliveryPoint"
	// PropDeployedSystems is JSON property "deployedSystems".
	PropDeployedSystems = Namespace + "deployedSystems"
	// PropDescription is JSON property "description".
	PropDescription = Namespace + "description"
	// PropDestination is JSON property "destination".
	PropDestination = Namespace + "destination"
	// PropDocumentation is JSON property "documentation".
	PropDocumentation = Namespace + "documentation"
	// PropDocuments is JSON property "documents".
	PropDocuments = Namespace + "documents"
	// PropElectronicMailAddress is JSON property "electronicMailAddress".
	PropElectronicMailAddress = Namespace + "electronicMailAddress"
	// PropFacsimile is JSON property "facsimile".
	PropFacsimile = Namespace + "facsimile"
	// PropFeaturesOfInterest is JSON property "featuresOfInterest".
	PropFeaturesOfInterest = Namespace + "featuresOfInterest"
	// PropH is JSON property "h".
	PropH = Namespace + "h"
	// PropHistory is JSON property "history".
	PropHistory = Namespace + "history"
	// PropHoursOfService is JSON property "hoursOfService".
	PropHoursOfService = Namespace + "hoursOfService"
	// PropID is JSON property "id".
	PropID = Namespace + "id"
	// PropIdentifiers is JSON property "identifiers".
	PropIdentifiers = Namespace + "identifiers"
	// PropIndividualName is JSON property "individualName".
	PropIndividualName = Namespace + "individualName"
	// PropInputs is JSON property "inputs".
	PropInputs = Namespace + "inputs"
	// PropKeywords is JSON property "keywords".
	PropKeywords = Namespace + "keywords"
	// PropLabel is JSON property "label".
	PropLabel = Namespace + "label"
	// PropLang is JSON property "lang".
	PropLang = Namespace + "lang"
	// PropLat is JSON property "lat".
	PropLat = Namespace + "lat"
	// PropLegalConstraints is JSON property "legalConstraints".
	PropLegalConstraints = Namespace + "legalConstraints"
	// PropLink is JSON property "link" (on ContactLink and Document).
	PropLink = Namespace + "link"
	// PropLocalReferenceFrames is JSON property "localReferenceFrames".
	PropLocalReferenceFrames = Namespace + "localReferenceFrames"
	// PropLocalTimeFrames is JSON property "localTimeFrames".
	PropLocalTimeFrames = Namespace + "localTimeFrames"
	// PropLocation is JSON property "location".
	PropLocation = Namespace + "location"
	// PropLon is JSON property "lon".
	PropLon = Namespace + "lon"
	// PropLTPReferenceFrame is JSON property "ltpReferenceFrame".
	PropLTPReferenceFrame = Namespace + "ltpReferenceFrame"
	// PropMethod is JSON property "method".
	PropMethod = Namespace + "method"
	// PropModes is JSON property "modes".
	PropModes = Namespace + "modes"
	// PropName is JSON property "name".
	PropName = Namespace + "name"
	// PropObjectType is JSON property "objectType".
	PropObjectType = Namespace + "objectType"
	// PropOrganisationName is JSON property "organisationName" (ResponsibleParty.json).
	PropOrganisationName = Namespace + "organisationName"
	// PropOrigin is JSON property "origin".
	PropOrigin = Namespace + "origin"
	// PropOtherConstraints is JSON property "otherConstraints".
	PropOtherConstraints = Namespace + "otherConstraints"
	// PropOutputs is JSON property "outputs".
	PropOutputs = Namespace + "outputs"
	// PropParameters is JSON property "parameters".
	PropParameters = Namespace + "parameters"
	// PropPhone is JSON property "phone".
	PropPhone = Namespace + "phone"
	// PropPitch is JSON property "pitch".
	PropPitch = Namespace + "pitch"
	// PropPlatform is JSON property "platform".
	PropPlatform = Namespace + "platform"
	// PropPosition is JSON property "position".
	PropPosition = Namespace + "position"
	// PropPositionName is JSON property "positionName".
	PropPositionName = Namespace + "positionName"
	// PropPostalCode is JSON property "postalCode".
	PropPostalCode = Namespace + "postalCode"
	// PropProperties is JSON property "properties".
	PropProperties = Namespace + "properties"
	// PropQualifiers is JSON property "qualifiers".
	PropQualifiers = Namespace + "qualifiers"
	// PropQuaternion is JSON property "quaternion".
	PropQuaternion = Namespace + "quaternion"
	// PropRef is JSON property "ref".
	PropRef = Namespace + "ref"
	// PropReferenceFrame is JSON property "referenceFrame".
	PropReferenceFrame = Namespace + "referenceFrame"
	// PropRole is JSON property "role".
	PropRole = Namespace + "role"
	// PropRoll is JSON property "roll".
	PropRoll = Namespace + "roll"
	// PropSecurityConstraints is JSON property "securityConstraints".
	PropSecurityConstraints = Namespace + "securityConstraints"
	// PropSetArrayValues is JSON property "setArrayValues".
	PropSetArrayValues = Namespace + "setArrayValues"
	// PropSetConstraints is JSON property "setConstraints".
	PropSetConstraints = Namespace + "setConstraints"
	// PropSetModes is JSON property "setModes".
	PropSetModes = Namespace + "setModes"
	// PropSetStatus is JSON property "setStatus".
	PropSetStatus = Namespace + "setStatus"
	// PropSetValues is JSON property "setValues".
	PropSetValues = Namespace + "setValues"
	// PropSource is JSON property "source".
	PropSource = Namespace + "source"
	// PropStatistic is JSON property "statistic".
	PropStatistic = Namespace + "statistic"
	// PropSystem is JSON property "system".
	PropSystem = Namespace + "system"
	// PropTime is JSON property "time".
	PropTime = Namespace + "time"
	// PropTypeOf is JSON property "typeOf".
	PropTypeOf = Namespace + "typeOf"
	// PropUniqueID is JSON property "uniqueId".
	PropUniqueID = Namespace + "uniqueId"
	// PropUseConstraints is JSON property "useConstraints".
	PropUseConstraints = Namespace + "useConstraints"
	// PropUseLimitations is JSON property "useLimitations".
	PropUseLimitations = Namespace + "useLimitations"
	// PropValidTime is JSON property "validTime".
	PropValidTime = Namespace + "validTime"
	// PropValue is JSON property "value".
	PropValue = Namespace + "value"
	// PropVoice is JSON property "voice".
	PropVoice = Namespace + "voice"
	// PropW is JSON property "w".
	PropW = Namespace + "w"
	// PropWebsite is JSON property "website".
	PropWebsite = Namespace + "website"
	// PropX is JSON property "x".
	PropX = Namespace + "x"
	// PropY is JSON property "y".
	PropY = Namespace + "y"
	// PropYaw is JSON property "yaw".
	PropYaw = Namespace + "yaw"
	// PropZ is JSON property "z".
	PropZ = Namespace + "z"
)

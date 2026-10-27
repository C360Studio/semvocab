// Package ssn is SemVocab's catalog of W3C/OGC Semantic Sensor Network (SSN)
// 2017 Ontology and System Capabilities (`ssn-system`) terms: IRI constants
// across Namespace and SystemsNamespace, and the formal subclass hierarchy
// (DirectParents, Parents, IsA) over the integrated SOSA, SSN, and ssn-system
// closure.
//
// Source and admission: the terms and IRIs are new content from the W3C
// Semantic Sensor Network Ontology Recommendation (REC-vocab-ssn-20171019,
// https://www.w3.org/TR/2017/REC-vocab-ssn-20171019/, w3c/sdw commit
// cb31ee9de636a7101337815ba67ba9af4a7c1348, ssn/integrated/ssn.ttl with
// dcterms:created 2017-04-17 and ssn/integrated/ssn-system.ttl with
// dcterms:created 2017-05-14), as listed in docs/admission-ledger.yaml (entry
// "ssn"). Both modules are dual-licensed under the W3C Software and Document
// License (2015) and the OGC Software License, Copyright 2017 W3C/OGC; both
// license texts are kept with the vendored files in
// internal/oracle/testdata/w3c-sdw/NOTICE.
//
// Because ssn.ttl asserts two direct named superclasses on sosa:Sample
// (sosa:FeatureOfInterest and sosa:Result), this package exports DirectParents
// rather than a single-parent SubClassOf helper.
//
// Importing this package registers nothing. The vocabulary pack that declares
// the sosa, ssn, ssn-system, swe, and sml prefixes is packs.RegisterSWECore.
package ssn

// Namespace is the W3C SSN 2017 namespace.
const Namespace = "http://www.w3.org/ns/ssn/"

// SystemsNamespace is the W3C SSN System Capabilities (ssn-system) 2017
// namespace.
const SystemsNamespace = "http://www.w3.org/ns/ssn/systems/"

// SSN classes.
const (
	// Deployment is "Deployment" (ssn:Deployment).
	Deployment = Namespace + "Deployment"
	// Input is "Input" (ssn:Input).
	Input = Namespace + "Input"
	// Output is "Output" (ssn:Output).
	Output = Namespace + "Output"
	// Property is "Property" (ssn:Property).
	Property = Namespace + "Property"
	// Stimulus is "Stimulus" (ssn:Stimulus).
	Stimulus = Namespace + "Stimulus"
	// System is "System" (ssn:System).
	System = Namespace + "System"
)

// SSN object properties.
const (
	// DeployedOnPlatform is "deployed on platform" (ssn:deployedOnPlatform).
	DeployedOnPlatform = Namespace + "deployedOnPlatform"
	// DeployedSystem is "deployed system" (ssn:deployedSystem).
	DeployedSystem = Namespace + "deployedSystem"
	// Detects is "detects" (ssn:detects).
	Detects = Namespace + "detects"
	// ForProperty is "for property" (ssn:forProperty).
	ForProperty = Namespace + "forProperty"
	// HasDeployment is "has deployment" (ssn:hasDeployment).
	HasDeployment = Namespace + "hasDeployment"
	// HasInput is "has input" (ssn:hasInput).
	HasInput = Namespace + "hasInput"
	// HasOutput is "has output" (ssn:hasOutput).
	HasOutput = Namespace + "hasOutput"
	// HasProperty is "has property" (ssn:hasProperty).
	HasProperty = Namespace + "hasProperty"
	// HasSubSystem is "has subsystem" (ssn:hasSubSystem).
	HasSubSystem = Namespace + "hasSubSystem"
	// ImplementedBy is "implemented by" (ssn:implementedBy).
	ImplementedBy = Namespace + "implementedBy"
	// Implements is "implements" (ssn:implements).
	Implements = Namespace + "implements"
	// InDeployment is "in deployment" (ssn:inDeployment).
	InDeployment = Namespace + "inDeployment"
	// IsPropertyOf is "is property of" (ssn:isPropertyOf).
	IsPropertyOf = Namespace + "isPropertyOf"
	// IsProxyFor is "is proxy for" (ssn:isProxyFor).
	IsProxyFor = Namespace + "isProxyFor"
	// WasOriginatedBy is "was originated by" (ssn:wasOriginatedBy).
	WasOriginatedBy = Namespace + "wasOriginatedBy"
)

// SSN System Capabilities (ssn-system) classes.
const (
	// Accuracy is "Accuracy" (ssn-system:Accuracy).
	Accuracy = SystemsNamespace + "Accuracy"
	// ActuationRange is "Actuation Range" (ssn-system:ActuationRange).
	ActuationRange = SystemsNamespace + "ActuationRange"
	// BatteryLifetime is "Battery Lifetime" (ssn-system:BatteryLifetime).
	BatteryLifetime = SystemsNamespace + "BatteryLifetime"
	// Condition is "Condition" (ssn-system:Condition).
	Condition = SystemsNamespace + "Condition"
	// DetectionLimit is "Detection Limit" (ssn-system:DetectionLimit).
	DetectionLimit = SystemsNamespace + "DetectionLimit"
	// Drift is "Drift" (ssn-system:Drift).
	Drift = SystemsNamespace + "Drift"
	// Frequency is "Frequency" (ssn-system:Frequency).
	Frequency = SystemsNamespace + "Frequency"
	// Latency is "Latency" (ssn-system:Latency).
	Latency = SystemsNamespace + "Latency"
	// MaintenanceSchedule is "Maintenance Schedule" (ssn-system:MaintenanceSchedule).
	MaintenanceSchedule = SystemsNamespace + "MaintenanceSchedule"
	// MeasurementRange is "Measurement Range" (ssn-system:MeasurementRange).
	MeasurementRange = SystemsNamespace + "MeasurementRange"
	// OperatingPowerRange is "Operating Power Range" (ssn-system:OperatingPowerRange).
	OperatingPowerRange = SystemsNamespace + "OperatingPowerRange"
	// OperatingProperty is "Operating Property" (ssn-system:OperatingProperty).
	OperatingProperty = SystemsNamespace + "OperatingProperty"
	// OperatingRange is "Operating Range" (ssn-system:OperatingRange).
	OperatingRange = SystemsNamespace + "OperatingRange"
	// Precision is "Precision" (ssn-system:Precision).
	Precision = SystemsNamespace + "Precision"
	// Resolution is "Resolution" (ssn-system:Resolution).
	Resolution = SystemsNamespace + "Resolution"
	// ResponseTime is "Response Time" (ssn-system:ResponseTime).
	ResponseTime = SystemsNamespace + "ResponseTime"
	// Selectivity is "Selectivity" (ssn-system:Selectivity).
	Selectivity = SystemsNamespace + "Selectivity"
	// Sensitivity is "Sensitivity" (ssn-system:Sensitivity).
	Sensitivity = SystemsNamespace + "Sensitivity"
	// SurvivalProperty is "Survival Property" (ssn-system:SurvivalProperty).
	SurvivalProperty = SystemsNamespace + "SurvivalProperty"
	// SurvivalRange is "Survival Range" (ssn-system:SurvivalRange).
	SurvivalRange = SystemsNamespace + "SurvivalRange"
	// SystemCapability is "System Capability" (ssn-system:SystemCapability).
	SystemCapability = SystemsNamespace + "SystemCapability"
	// SystemLifetime is "System Lifetime" (ssn-system:SystemLifetime).
	SystemLifetime = SystemsNamespace + "SystemLifetime"
	// SystemProperty is "System Property" (ssn-system:SystemProperty).
	SystemProperty = SystemsNamespace + "SystemProperty"
)

// SSN System Capabilities (ssn-system) object properties.
const (
	// HasOperatingProperty is "has operating property" (ssn-system:hasOperatingProperty).
	HasOperatingProperty = SystemsNamespace + "hasOperatingProperty"
	// HasOperatingRange is "has operating range" (ssn-system:hasOperatingRange).
	HasOperatingRange = SystemsNamespace + "hasOperatingRange"
	// HasSurvivalProperty is "has survival property" (ssn-system:hasSurvivalProperty).
	HasSurvivalProperty = SystemsNamespace + "hasSurvivalProperty"
	// HasSurvivalRange is "has survival range" (ssn-system:hasSurvivalRange).
	HasSurvivalRange = SystemsNamespace + "hasSurvivalRange"
	// HasSystemCapability is "has system capability" (ssn-system:hasSystemCapability).
	HasSystemCapability = SystemsNamespace + "hasSystemCapability"
	// HasSystemProperty is "has system property" (ssn-system:hasSystemProperty).
	HasSystemProperty = SystemsNamespace + "hasSystemProperty"
	// InCondition is "in condition" (ssn-system:inCondition).
	InCondition = SystemsNamespace + "inCondition"
	// QualityOfObservation is "quality of observation" (ssn-system:qualityOfObservation).
	QualityOfObservation = SystemsNamespace + "qualityOfObservation"
)

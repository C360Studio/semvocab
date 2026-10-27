// Package sosa is SemVocab's catalog of W3C/OGC Sensor, Observation, Sample,
// and Actuator (SOSA) 2017 Ontology terms: IRI constants for the 13 classes,
// 21 object properties, and 2 datatype properties declared in SOSA.
//
// Source and admission: the terms and IRIs are new content from the W3C
// Semantic Sensor Network Ontology Recommendation (REC-vocab-ssn-20171019,
// https://www.w3.org/TR/2017/REC-vocab-ssn-20171019/, w3c/sdw commit
// cb31ee9de636a7101337815ba67ba9af4a7c1348, ssn/integrated/sosa.ttl,
// dcterms:created 2017-04-17), as listed in docs/admission-ledger.yaml (entry
// "sosa"). SOSA is dual-licensed under the W3C Software and Document License
// (2015) and the OGC Software License, Copyright 2017 W3C/OGC; both license
// texts are kept with the vendored release in
// internal/oracle/testdata/w3c-sdw/NOTICE.
//
// SOSA uses schema:domainIncludes and schema:rangeIncludes rather than OWL
// class axioms, and sosa.ttl asserts zero rdfs:subClassOf triples. This package
// therefore exports constants only; catalog/ssn provides the formal
// rdfs:subClassOf hierarchy over SOSA, SSN, and ssn-system.
//
// Importing this package registers nothing. The vocabulary pack that declares
// the sosa, ssn, ssn-system, swe, and sml prefixes is packs.RegisterSWECore.
package sosa

// Namespace is the W3C SOSA 2017 namespace.
const Namespace = "http://www.w3.org/ns/sosa/"

// Classes.
const (
	// ActuatableProperty is "Actuatable Property" (sosa:ActuatableProperty).
	ActuatableProperty = Namespace + "ActuatableProperty"
	// Actuation is "Actuation" (sosa:Actuation).
	Actuation = Namespace + "Actuation"
	// Actuator is "Actuator" (sosa:Actuator).
	Actuator = Namespace + "Actuator"
	// FeatureOfInterest is "Feature Of Interest" (sosa:FeatureOfInterest).
	FeatureOfInterest = Namespace + "FeatureOfInterest"
	// ObservableProperty is "Observable Property" (sosa:ObservableProperty).
	ObservableProperty = Namespace + "ObservableProperty"
	// Observation is "Observation" (sosa:Observation).
	Observation = Namespace + "Observation"
	// Platform is "Platform" (sosa:Platform).
	Platform = Namespace + "Platform"
	// Procedure is "Procedure" (sosa:Procedure).
	Procedure = Namespace + "Procedure"
	// Result is "Result" (sosa:Result).
	Result = Namespace + "Result"
	// Sample is "Sample" (sosa:Sample).
	Sample = Namespace + "Sample"
	// Sampler is "Sampler" (sosa:Sampler).
	Sampler = Namespace + "Sampler"
	// Sampling is "Sampling" (sosa:Sampling).
	Sampling = Namespace + "Sampling"
	// Sensor is "Sensor" (sosa:Sensor).
	Sensor = Namespace + "Sensor"
)

// Object properties.
const (
	// ActsOnProperty is "acts on property" (sosa:actsOnProperty).
	ActsOnProperty = Namespace + "actsOnProperty"
	// HasFeatureOfInterest is "has feature of interest" (sosa:hasFeatureOfInterest).
	HasFeatureOfInterest = Namespace + "hasFeatureOfInterest"
	// HasResult is "has result" (sosa:hasResult).
	HasResult = Namespace + "hasResult"
	// HasSample is "has sample" (sosa:hasSample).
	HasSample = Namespace + "hasSample"
	// Hosts is "hosts" (sosa:hosts).
	Hosts = Namespace + "hosts"
	// IsActedOnBy is "is acted on by" (sosa:isActedOnBy).
	IsActedOnBy = Namespace + "isActedOnBy"
	// IsFeatureOfInterestOf is "is feature of interest of" (sosa:isFeatureOfInterestOf).
	IsFeatureOfInterestOf = Namespace + "isFeatureOfInterestOf"
	// IsHostedBy is "is hosted by" (sosa:isHostedBy).
	IsHostedBy = Namespace + "isHostedBy"
	// IsObservedBy is "is observed by" (sosa:isObservedBy).
	IsObservedBy = Namespace + "isObservedBy"
	// IsResultOf is "is result of" (sosa:isResultOf).
	IsResultOf = Namespace + "isResultOf"
	// IsSampleOf is "is sample of" (sosa:isSampleOf).
	IsSampleOf = Namespace + "isSampleOf"
	// MadeActuation is "made actuation" (sosa:madeActuation).
	MadeActuation = Namespace + "madeActuation"
	// MadeByActuator is "made by actuator" (sosa:madeByActuator).
	MadeByActuator = Namespace + "madeByActuator"
	// MadeBySampler is "made by sampler" (sosa:madeBySampler).
	MadeBySampler = Namespace + "madeBySampler"
	// MadeBySensor is "made by sensor" (sosa:madeBySensor).
	MadeBySensor = Namespace + "madeBySensor"
	// MadeObservation is "made observation" (sosa:madeObservation).
	MadeObservation = Namespace + "madeObservation"
	// MadeSampling is "made sampling" (sosa:madeSampling).
	MadeSampling = Namespace + "madeSampling"
	// ObservedProperty is "observed property" (sosa:observedProperty).
	ObservedProperty = Namespace + "observedProperty"
	// Observes is "observes" (sosa:observes).
	Observes = Namespace + "observes"
	// PhenomenonTime is "phenomenon time" (sosa:phenomenonTime).
	PhenomenonTime = Namespace + "phenomenonTime"
	// UsedProcedure is "used procedure" (sosa:usedProcedure).
	UsedProcedure = Namespace + "usedProcedure"
)

// Datatype properties.
const (
	// HasSimpleResult is "has simple result" (sosa:hasSimpleResult).
	HasSimpleResult = Namespace + "hasSimpleResult"
	// ResultTime is "result time" (sosa:resultTime).
	ResultTime = Namespace + "resultTime"
)

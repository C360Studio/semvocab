// Package cco is SemVocab's catalog of Common Core Ontologies (CCO) v2.2
// terms: IRI constants for the admitted classes and object properties, and
// their subclass hierarchy (SubClassOf, Parents, IsA), which crosses into
// catalog/bfo.
//
// Source and admission: the terms and IRIs are new content from CCO v2.2
// (CommonCoreOntology/CommonCoreOntologies, tag v2.2, commit
// 0bc7d33e1bc09fd4693366119ab4e03cb0340042,
// src/cco-merged/CommonCoreOntologiesMerged.ttl, version IRI
// https://www.commoncoreontologies.org/2026-08-13/CommonCoreOntologiesMerged),
// as listed in docs/admission-ledger.yaml (entry "cco"). CCO is licensed
// BSD 3-Clause, Copyright (c) 2017, CUBRC, Inc.; the license text is kept with
// the vendored release file in internal/oracle/testdata/cco-2.2/LICENSE.
//
// The Go names follow SemStreams (github.com/C360Studio/semstreams,
// vocabulary/cco/cco.go at 8b99efe9c66a4faa4fa509f9f62cc6bad8392128, MIT,
// Copyright (c) 2025 C360), except where CCO v2.2 relabelled a term, which
// takes its v2.2 rdfs:label (owner ruling Q10 on semvocab#1). No SemStreams
// IRI is carried: SemStreams used the CCO 1.x namespace, and CCO v2 mints
// opaque IRIs (ont followed by eight digits) under Namespace. Each constant's
// comment gives its v2.2 rdfs:label.
//
// Importing this package registers nothing. The vocabulary pack that declares
// the cco prefix is packs.RegisterBFOCCOCore.
package cco

// Namespace is the CCO v2 namespace.
const Namespace = "https://www.commoncoreontologies.org/"

// Classes.
const (
	// InformationContentEntity is "Information Content Entity" (ont00000958).
	InformationContentEntity = Namespace + "ont00000958"
	// DescriptiveInformationContentEntity is "Descriptive Information Content Entity" (ont00000853).
	DescriptiveInformationContentEntity = Namespace + "ont00000853"
	// PrescriptiveInformationContentEntity is "Prescriptive Information Content Entity" (ont00000965).
	PrescriptiveInformationContentEntity = Namespace + "ont00000965"
	// Plan is "Plan" (ont00000974).
	Plan = Namespace + "ont00000974"
	// Objective is "Objective" (ont00000476).
	Objective = Namespace + "ont00000476"
	// DesignativeInformationContentEntity is "Designative Information Content Entity" (ont00000686).
	DesignativeInformationContentEntity = Namespace + "ont00000686"
	// MeasurementInformationContentEntity is "Measurement Information Content Entity" (ont00001163).
	MeasurementInformationContentEntity = Namespace + "ont00001163"
	// Algorithm is "Algorithm" (ont00000653).
	Algorithm = Namespace + "ont00000653"
	// Agent is "Agent" (ont00001017).
	Agent = Namespace + "ont00001017"
	// Person is "Person" (ont00001262).
	Person = Namespace + "ont00001262"
	// Organization is "Organization" (ont00001180).
	Organization = Namespace + "ont00001180"
	// GroupOfAgents is "Group of Agents" (ont00000300).
	GroupOfAgents = Namespace + "ont00000300"
	// Act is "Act" (ont00000005).
	Act = Namespace + "ont00000005"
	// PlannedAct is "Planned Act" (ont00000228).
	PlannedAct = Namespace + "ont00000228"
	// ActOfCommunication is "Act of Communication" (ont00000402).
	ActOfCommunication = Namespace + "ont00000402"
	// ActOfMaterialArtifactProcessing is "Act of Material Artifact Processing" (ont00000908).
	ActOfMaterialArtifactProcessing = Namespace + "ont00000908"
	// ActOfPositionChange is "Act of Position Change" (ont00000584).
	ActOfPositionChange = Namespace + "ont00000584"
	// MaterialArtifact is "Material Artifact" (ont00000995).
	MaterialArtifact = Namespace + "ont00000995"
	// InformationBearingArtifact is "Information Bearing Artifact" (ont00000798).
	InformationBearingArtifact = Namespace + "ont00000798"
	// MaterialCopyOfDocument is "Material Copy of a Document" (ont00001298).
	MaterialCopyOfDocument = Namespace + "ont00001298"
	// Facility is "Facility" (ont00000192).
	Facility = Namespace + "ont00000192"
	// Vehicle is "Vehicle" (ont00000713).
	Vehicle = Namespace + "ont00000713"
	// Sensor is "Sensor" (ont00000569).
	Sensor = Namespace + "ont00000569"
	// ActOfObservation is "Act of Observation" (ont00000037).
	ActOfObservation = Namespace + "ont00000037"
	// Altitude is "Altitude" (ont00000787).
	Altitude = Namespace + "ont00000787"
	// Height is "Height" (ont00000967).
	Height = Namespace + "ont00000967"
	// OneDimensionalExtent is "One Dimensional Extent" (ont00001367).
	OneDimensionalExtent = Namespace + "ont00001367"
	// SizeQuality is "Size Quality" (ont00001202).
	SizeQuality = Namespace + "ont00001202"
	// InformationBearingEntity is "Information Bearing Entity" (ont00000253).
	InformationBearingEntity = Namespace + "ont00000253"
	// Animal is "Animal" (ont00000562).
	Animal = Namespace + "ont00000562"
	// Organism is "Organism" (ont00000551).
	Organism = Namespace + "ont00000551"
	// Transducer is "Transducer" (ont00000736).
	Transducer = Namespace + "ont00000736"
	// ActOfMotion is "Act of Motion" (ont00000357).
	ActOfMotion = Namespace + "ont00000357"
)

// Object properties.
const (
	// IsAbout is "is about" (ont00001808).
	IsAbout = Namespace + "ont00001808"
	// HasAgent is "has agent" (ont00001833).
	HasAgent = Namespace + "ont00001833"
	// AgentIn is "agent in" (ont00001787).
	AgentIn = Namespace + "ont00001787"
	// HasInput is "has input" (ont00001921).
	HasInput = Namespace + "ont00001921"
	// HasOutput is "has output" (ont00001986).
	HasOutput = Namespace + "ont00001986"
	// Prescribes is "prescribes" (ont00001942).
	Prescribes = Namespace + "ont00001942"
	// DesignatedBy is "designated by" (ont00001879).
	DesignatedBy = Namespace + "ont00001879"
	// Designates is "designates" (ont00001916).
	Designates = Namespace + "ont00001916"
	// OccursAt is "occurs at" (ont00001918).
	OccursAt = Namespace + "ont00001918"
	// IsMeasuredBy is "is measured by" (ont00001904).
	IsMeasuredBy = Namespace + "ont00001904"
)

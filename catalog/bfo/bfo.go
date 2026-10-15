// Package bfo is SemVocab's catalog of Basic Formal Ontology (BFO 2020) terms:
// IRI constants for the admitted classes and relations, and their subclass
// hierarchy (SubClassOf, Parents, IsA).
//
// Source and admission: the constant names and IRIs are ported from SemStreams
// (github.com/C360Studio/semstreams, vocabulary/bfo at revision
// 8b99efe9c66a4faa4fa509f9f62cc6bad8392128, MIT, Copyright (c) 2025 C360),
// restricted to the terms that BFO 2020 declares with the same meaning, as
// recorded in docs/admission-ledger.yaml (entry "bfo"). ImmaterialEntity is
// added from BFO 2020 itself. The hierarchy follows BFO 2020, not the
// SemStreams map: Site and SpatialRegion sit under ImmaterialEntity.
//
// BFO 2020 is BFO-ontology/BFO-2020, src/owl/bfo-core.ttl at commit
// 5ed468765ea5655a0ff9a93877fd059f294a152d (version IRI
// http://purl.obolibrary.org/obo/bfo/2020/bfo-core.ttl), licensed CC BY 4.0
// (https://creativecommons.org/licenses/by/4.0/); attribution: BFO 2020, by
// the BFO contributors.
//
// Importing this package registers nothing. The vocabulary pack that declares
// the OBO prefix is packs.RegisterBFOCCOCore.
package bfo

// Namespace is the OBO Foundry PURL base under which BFO terms are minted. It
// is shared by every OBO ontology, which is why SemVocab's packs bind it to
// the prefix "obo", not "bfo".
const Namespace = "http://purl.obolibrary.org/obo/"

// Top-level categories.
const (
	// Entity (BFO_0000001) is the root of BFO.
	Entity = Namespace + "BFO_0000001"
	// Continuant (BFO_0000002) persists through time while keeping its identity.
	Continuant = Namespace + "BFO_0000002"
	// Occurrent (BFO_0000003) unfolds in time.
	Occurrent = Namespace + "BFO_0000003"
)

// Continuants.
const (
	// IndependentContinuant (BFO_0000004) does not depend on another entity to exist.
	IndependentContinuant = Namespace + "BFO_0000004"
	// SpecificallyDependentContinuant (BFO_0000020) depends on one bearer: qualities and realizables.
	SpecificallyDependentContinuant = Namespace + "BFO_0000020"
	// GenericallyDependentContinuant (BFO_0000031) can migrate between bearers, as information content does.
	GenericallyDependentContinuant = Namespace + "BFO_0000031"

	// MaterialEntity (BFO_0000040) is an independent continuant with matter as a part.
	MaterialEntity = Namespace + "BFO_0000040"
	// Object (BFO_0000030) is a causally unified material entity, such as a drone.
	Object = Namespace + "BFO_0000030"
	// ObjectAggregate (BFO_0000027) is a material entity made of several objects, such as a fleet.
	ObjectAggregate = Namespace + "BFO_0000027"
	// FiatObjectPart (BFO_0000024) is a part of an object without a physical boundary.
	FiatObjectPart = Namespace + "BFO_0000024"

	// ImmaterialEntity (BFO_0000141) is an independent continuant with no material part.
	// It is not in the SemStreams source; BFO 2020 places Site and SpatialRegion under it.
	ImmaterialEntity = Namespace + "BFO_0000141"
	// Site (BFO_0000029) is a space bounded by material entities, such as a hangar interior.
	Site = Namespace + "BFO_0000029"
	// SpatialRegion (BFO_0000006) is a part of space.
	SpatialRegion = Namespace + "BFO_0000006"
	// OneDimensionalSpatialRegion (BFO_0000026) is a line or curve in space.
	OneDimensionalSpatialRegion = Namespace + "BFO_0000026"
	// TwoDimensionalSpatialRegion (BFO_0000009) is a surface in space.
	TwoDimensionalSpatialRegion = Namespace + "BFO_0000009"
	// ThreeDimensionalSpatialRegion (BFO_0000028) is a volume in space.
	ThreeDimensionalSpatialRegion = Namespace + "BFO_0000028"

	// Quality (BFO_0000019) is exhibited by its bearer, such as a mass or a color.
	Quality = Namespace + "BFO_0000019"
	// RealizableEntity (BFO_0000017) can be realized in a process.
	RealizableEntity = Namespace + "BFO_0000017"
	// Role (BFO_0000023) is a realizable entity held because of external circumstances.
	Role = Namespace + "BFO_0000023"
	// Disposition (BFO_0000016) is a realizable entity held because of the bearer's make-up.
	Disposition = Namespace + "BFO_0000016"
	// Function (BFO_0000034) is a disposition held because of design or selection.
	Function = Namespace + "BFO_0000034"
)

// Occurrents.
const (
	// Process (BFO_0000015) is an occurrent with participants, such as a flight.
	Process = Namespace + "BFO_0000015"
	// ProcessBoundary (BFO_0000035) is the instantaneous start or end of a process.
	ProcessBoundary = Namespace + "BFO_0000035"
	// SpatiotemporalRegion (BFO_0000011) is a part of spacetime.
	SpatiotemporalRegion = Namespace + "BFO_0000011"
	// TemporalRegion (BFO_0000008) is a part of time.
	TemporalRegion = Namespace + "BFO_0000008"
	// ZeroDimensionalTemporalRegion (BFO_0000148) is an instant.
	ZeroDimensionalTemporalRegion = Namespace + "BFO_0000148"
	// OneDimensionalTemporalRegion (BFO_0000038) is an interval.
	OneDimensionalTemporalRegion = Namespace + "BFO_0000038"
	// History (BFO_0000182) is the sum of the processes in the spatiotemporal region a material entity occupies.
	History = Namespace + "BFO_0000182"
)

// Relations (OWL object properties in BFO 2020).
const (
	// OccursIn (BFO_0000066, "occurs in") relates a process to where it occurs.
	OccursIn = Namespace + "BFO_0000066"
	// ParticipatesIn (BFO_0000056, "participates in") relates a continuant to a process.
	ParticipatesIn = Namespace + "BFO_0000056"
	// HasParticipant (BFO_0000057, "has participant") is the inverse of ParticipatesIn.
	HasParticipant = Namespace + "BFO_0000057"
	// RealizedIn (BFO_0000054) relates a realizable entity to the process that realizes it.
	// BFO 2020 labels it "has realization"; the Go name is SemStreams'.
	RealizedIn = Namespace + "BFO_0000054"
	// Realizes (BFO_0000055, "realizes") is the inverse of RealizedIn.
	Realizes = Namespace + "BFO_0000055"
	// LocatedIn (BFO_0000171, "located in") relates an independent continuant to a containing one.
	LocatedIn = Namespace + "BFO_0000171"
	// LocationOf (BFO_0000124, "location of") is the inverse of LocatedIn.
	LocationOf = Namespace + "BFO_0000124"
	// ExistsAt (BFO_0000108, "exists at") relates an entity to a temporal region it exists at.
	ExistsAt = Namespace + "BFO_0000108"
	// TemporalPartOf (BFO_0000139, "temporal part of") relates an occurrent to a whole it is a temporal part of.
	TemporalPartOf = Namespace + "BFO_0000139"
	// HasTemporalPart (BFO_0000121, "has temporal part") is the inverse of TemporalPartOf.
	HasTemporalPart = Namespace + "BFO_0000121"
)

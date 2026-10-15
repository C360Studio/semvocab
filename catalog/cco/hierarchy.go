package cco

import "github.com/c360studio/semvocab/catalog/bfo"

// subClassOf maps each admitted CCO class to its one asserted named
// rdfs:subClassOf parent in the CCO v2.2 merged file. Every admitted class has
// exactly one such parent there, and every parent is either admitted here or a
// catalog/bfo class, so walking this map and then bfo's yields the release
// closure.
var subClassOf = map[string]string{
	InformationContentEntity:             bfo.GenericallyDependentContinuant,
	DescriptiveInformationContentEntity:  InformationContentEntity,
	PrescriptiveInformationContentEntity: InformationContentEntity,
	DesignativeInformationContentEntity:  InformationContentEntity,
	Plan:                                 PrescriptiveInformationContentEntity,
	Objective:                            PrescriptiveInformationContentEntity,
	Algorithm:                            PrescriptiveInformationContentEntity,
	MeasurementInformationContentEntity:  DescriptiveInformationContentEntity,

	Agent:         bfo.MaterialEntity,
	Organism:      bfo.Object,
	Animal:        Organism,
	Person:        Animal,
	GroupOfAgents: bfo.ObjectAggregate,
	Organization:  GroupOfAgents,

	Act:                             bfo.Process,
	PlannedAct:                      Act,
	ActOfCommunication:              PlannedAct,
	ActOfMaterialArtifactProcessing: PlannedAct,
	ActOfObservation:                PlannedAct,
	ActOfMotion:                     PlannedAct,
	ActOfPositionChange:             ActOfMotion,

	MaterialArtifact:           bfo.MaterialEntity,
	InformationBearingArtifact: MaterialArtifact,
	MaterialCopyOfDocument:     InformationBearingArtifact,
	Facility:                   MaterialArtifact,
	Vehicle:                    MaterialArtifact,
	Transducer:                 MaterialArtifact,
	Sensor:                     bfo.MaterialEntity,
	InformationBearingEntity:   bfo.Object,

	SizeQuality:          bfo.Quality,
	OneDimensionalExtent: SizeQuality,
	Height:               OneDimensionalExtent,
	Altitude:             Height,
}

// SubClassOf returns the direct CCO v2.2 superclass of the class iri, which
// may be a catalog/bfo class. For a BFO class it answers as bfo.SubClassOf
// does. ok is false for the BFO root, for properties, and for IRIs outside
// both catalogs.
func SubClassOf(iri string) (parent string, ok bool) {
	if parent, ok = subClassOf[iri]; ok {
		return parent, true
	}
	return bfo.SubClassOf(iri)
}

// Parents returns every superclass of the class iri in CCO v2.2 (which embeds
// BFO 2020), nearest first, ending at bfo.Entity. It returns nil for the BFO
// root and for IRIs outside both catalogs.
func Parents(iri string) []string {
	var out []string
	for cur := iri; ; {
		parent, ok := SubClassOf(cur)
		if !ok {
			return out
		}
		out = append(out, parent)
		cur = parent
	}
}

// IsA reports whether child is parent or a CCO v2.2 subclass of it. IsA(x, x)
// is true for any x.
func IsA(child, parent string) bool {
	if child == parent {
		return true
	}
	for _, p := range Parents(child) {
		if p == parent {
			return true
		}
	}
	return false
}

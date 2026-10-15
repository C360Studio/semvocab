package bfo

// subClassOf maps each admitted BFO class to its one asserted named
// rdfs:subClassOf parent in BFO 2020. Entity, the root, has no entry. Every
// admitted class has exactly one such parent in the release, and every parent
// is itself admitted, so walking this map yields the full release closure.
var subClassOf = map[string]string{
	Continuant: Entity,
	Occurrent:  Entity,

	IndependentContinuant:           Continuant,
	SpecificallyDependentContinuant: Continuant,
	GenericallyDependentContinuant:  Continuant,

	MaterialEntity:  IndependentContinuant,
	Object:          MaterialEntity,
	ObjectAggregate: MaterialEntity,
	FiatObjectPart:  MaterialEntity,

	ImmaterialEntity:              IndependentContinuant,
	Site:                          ImmaterialEntity,
	SpatialRegion:                 ImmaterialEntity,
	OneDimensionalSpatialRegion:   SpatialRegion,
	TwoDimensionalSpatialRegion:   SpatialRegion,
	ThreeDimensionalSpatialRegion: SpatialRegion,

	Quality:          SpecificallyDependentContinuant,
	RealizableEntity: SpecificallyDependentContinuant,
	Role:             RealizableEntity,
	Disposition:      RealizableEntity,
	Function:         Disposition,

	Process:                       Occurrent,
	ProcessBoundary:               Occurrent,
	SpatiotemporalRegion:          Occurrent,
	TemporalRegion:                Occurrent,
	ZeroDimensionalTemporalRegion: TemporalRegion,
	OneDimensionalTemporalRegion:  TemporalRegion,
	History:                       Process,
}

// SubClassOf returns the direct BFO 2020 superclass of the class iri. ok is
// false for Entity (the root), for relations, and for IRIs outside this
// catalog.
func SubClassOf(iri string) (parent string, ok bool) {
	parent, ok = subClassOf[iri]
	return parent, ok
}

// Parents returns every superclass of the class iri in BFO 2020, nearest first,
// ending at Entity. It returns nil for Entity and for IRIs outside this catalog.
func Parents(iri string) []string {
	var out []string
	for cur := iri; ; {
		parent, ok := subClassOf[cur]
		if !ok {
			return out
		}
		out = append(out, parent)
		cur = parent
	}
}

// IsA reports whether child is parent or a BFO 2020 subclass of it. IsA(x, x)
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

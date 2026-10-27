package sensorml

import (
	"slices"

	"github.com/c360studio/semvocab/catalog/swe"
)

// subClassOf maps each admitted SensorML 3.0 object class with an asserted JSON
// Schema allOf object $ref parent to that direct parent, which may be a
// catalog/swe class (swe.AbstractSweIdentifiable).
var subClassOf = map[string]string{
	AbstractProcess:         DescribedObject,
	AbstractPhysicalProcess: AbstractProcess,
	SimpleProcess:           AbstractProcess,
	AggregateProcess:        AbstractProcess,
	PhysicalComponent:       AbstractPhysicalProcess,
	PhysicalSystem:          AbstractPhysicalProcess,
	Deployment:              DescribedObject,
	Mode:                    DescribedObject,

	DerivedProperty:    swe.AbstractSweIdentifiable,
	Event:              swe.AbstractSweIdentifiable,
	CharacteristicList: swe.AbstractSweIdentifiable,
	CapabilityList:     swe.AbstractSweIdentifiable,
	SpatialFrame:       swe.AbstractSweIdentifiable,
	TemporalFrame:      swe.AbstractSweIdentifiable,
	ObservableProperty: swe.AbstractSweIdentifiable,
}

// SubClassOf returns the direct JSON Schema allOf parent class of iri in
// SensorML 3.0, which may be a catalog/swe class. For a SWE Common 3.0 class it
// answers as swe.SubClassOf does. ok is false for root classes, properties, and
// IRIs outside both catalogs.
func SubClassOf(iri string) (parent string, ok bool) {
	if parent, ok = subClassOf[iri]; ok {
		return parent, true
	}
	return swe.SubClassOf(iri)
}

// Parents returns the JSON Schema allOf ancestor classes of iri across SensorML
// 3.0 and SWE Common 3.0, nearest first. It returns nil for root classes,
// properties, and IRIs outside both catalogs.
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

// IsA reports whether child is parent or a SensorML 3.0 / SWE Common 3.0
// subclass of parent. IsA(x, x) is true for any x.
func IsA(child, parent string) bool {
	if child == parent {
		return true
	}
	return slices.Contains(Parents(child), parent)
}

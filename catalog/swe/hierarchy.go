package swe

import "slices"

// subClassOf maps each admitted SWE Common 3.0 object class with an asserted
// JSON Schema allOf object $ref parent to that direct parent.
var subClassOf = map[string]string{
	AbstractSweIdentifiable: AbstractSWE,
	AbstractDataComponent:   AbstractSweIdentifiable,
	AbstractSimpleComponent: AbstractDataComponent,

	Boolean:       AbstractSimpleComponent,
	Count:         AbstractSimpleComponent,
	Quantity:      AbstractSimpleComponent,
	Time:          AbstractSimpleComponent,
	Category:      AbstractSimpleComponent,
	Text:          AbstractSimpleComponent,
	CountRange:    AbstractSimpleComponent,
	QuantityRange: AbstractSimpleComponent,
	TimeRange:     AbstractSimpleComponent,
	CategoryRange: AbstractSimpleComponent,

	DataRecord:    AbstractDataComponent,
	Vector:        AbstractDataComponent,
	AbstractArray: AbstractDataComponent,
	DataArray:     AbstractArray,
	Matrix:        AbstractArray,
	DataChoice:    AbstractDataComponent,
	Geometry:      AbstractDataComponent,

	DataStream: AbstractSweIdentifiable,

	AbstractEncoding: AbstractSWE,
	TextEncoding:     AbstractEncoding,
	JSONEncoding:     AbstractEncoding,
	XMLEncoding:      AbstractEncoding,
	BinaryEncoding:   AbstractEncoding,
	Block:            AbstractSWE,
	Component:        AbstractSWE,
}

// SubClassOf returns the direct JSON Schema allOf parent class of iri in SWE
// Common 3.0. ok is false for root classes (such as AbstractSWE and
// UnitReference), properties, and IRIs outside this catalog.
func SubClassOf(iri string) (parent string, ok bool) {
	parent, ok = subClassOf[iri]
	return parent, ok
}

// Parents returns the JSON Schema allOf ancestor classes of iri in SWE Common
// 3.0, nearest first. It returns nil for root classes, properties, and unknown
// IRIs.
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

// IsA reports whether child is parent or a SWE Common 3.0 subclass of parent.
// IsA(x, x) is true for any x.
func IsA(child, parent string) bool {
	if child == parent {
		return true
	}
	return slices.Contains(Parents(child), parent)
}

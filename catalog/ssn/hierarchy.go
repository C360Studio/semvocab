package ssn

import (
	"slices"

	"github.com/c360studio/semvocab/catalog/sosa"
)

// directParents maps each SOSA, SSN, and ssn-system class with asserted named
// rdfs:subClassOf axioms in ssn.ttl or ssn-system.ttl to its direct named
// superclasses in document order.
var directParents = map[string][]string{
	// Asserted in ssn.ttl over SOSA classes.
	sosa.ObservableProperty: {Property},
	sosa.ActuatableProperty: {Property},
	sosa.Sample:             {sosa.FeatureOfInterest, sosa.Result},
	sosa.Sensor:             {System},
	sosa.Actuator:           {System},
	sosa.Sampler:            {System},

	// Asserted in ssn-system.ttl under ssn:Property.
	Condition:         {Property},
	SystemCapability:  {Property},
	SystemProperty:    {Property},
	OperatingRange:    {Property},
	OperatingProperty: {Property},
	SurvivalRange:     {Property},
	SurvivalProperty:  {Property},

	// Asserted in ssn-system.ttl under ssn-system:SystemProperty.
	MeasurementRange: {SystemProperty},
	ActuationRange:   {SystemProperty},
	Accuracy:         {SystemProperty},
	DetectionLimit:   {SystemProperty},
	Drift:            {SystemProperty},
	Frequency:        {SystemProperty},
	Latency:          {SystemProperty},
	Precision:        {SystemProperty},
	Resolution:       {SystemProperty},
	ResponseTime:     {SystemProperty},
	Selectivity:      {SystemProperty},
	Sensitivity:      {SystemProperty},

	// Asserted in ssn-system.ttl under ssn-system:OperatingProperty.
	MaintenanceSchedule: {OperatingProperty},
	OperatingPowerRange: {OperatingProperty},

	// Asserted in ssn-system.ttl under ssn-system:SurvivalProperty.
	SystemLifetime:  {SurvivalProperty},
	BatteryLifetime: {SurvivalProperty},
}

// DirectParents returns a fresh copy of the direct named rdfs:subClassOf
// superclasses of iri across the integrated SOSA, SSN, and ssn-system 2017
// ontologies, in document order, or nil when none are asserted.
func DirectParents(iri string) []string {
	return slices.Clone(directParents[iri])
}

// Parents returns the breadth-first transitive closure of asserted named
// rdfs:subClassOf superclasses of iri across SOSA, SSN, and ssn-system, without
// duplicates or owl:Thing. It returns nil for roots, properties, and unknown
// IRIs.
func Parents(iri string) []string {
	var out []string
	seen := map[string]bool{iri: true}
	queue := []string{iri}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, p := range directParents[cur] {
			if !seen[p] {
				seen[p] = true
				out = append(out, p)
				queue = append(queue, p)
			}
		}
	}
	return out
}

// IsA reports whether child is parent or a subclass of parent in the integrated
// SOSA, SSN, and ssn-system 2017 hierarchy. IsA(x, x) is true for any x.
func IsA(child, parent string) bool {
	if child == parent {
		return true
	}
	return slices.Contains(Parents(child), parent)
}

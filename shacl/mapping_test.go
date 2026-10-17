package shacl

import (
	"reflect"
	"strings"
	"testing"

	"github.com/c360studio/semengine/pkg/projection/contract"
)

var wantMapping = Mapping{
	EntityTypes: []EntityTypeBinding{{ClassIRI: dk + "IntelligencePlatform", DomainTypes: []string{"intel.platform"}}},
	Predicates: []PredicateBinding{
		{PathIRI: dk + "altitudeMeters", Name: "intel.measurement.altitude-m"},
		{PathIRI: dk + "assignedRole", Name: "intel.role.assigned"},
	},
	Contracts: []ContractBinding{{
		DomainType: "intel.platform", Name: "platform", Birth: []string{"intel.role.assigned"},
		Groups: []GroupBinding{{Name: "measurement", Mode: contract.ModeReconcile, Predicates: []string{"intel.measurement.altitude-m"}}},
	}},
	ClosedIgnoredPredicates: []string{"intel.meta.note"},
}

const mappingYAML = `entity_types:
  - class_iri: "https://w3id.org/c360/diekm#IntelligencePlatform"
    domain_types: [intel.platform]
predicates:
  - path_iri: "https://w3id.org/c360/diekm#altitudeMeters"
    name: intel.measurement.altitude-m
  - path_iri: "https://w3id.org/c360/diekm#assignedRole"
    name: intel.role.assigned
contracts:
  - domain_type: intel.platform
    name: platform
    birth: [intel.role.assigned]
    groups:
      - name: measurement
        mode: reconcile
        predicates: [intel.measurement.altitude-m]
closed_ignored_predicates: [intel.meta.note]
`

const mappingJSON = `{
  "entity_types": [{"class_iri": "https://w3id.org/c360/diekm#IntelligencePlatform", "domain_types": ["intel.platform"]}],
  "predicates": [
    {"path_iri": "https://w3id.org/c360/diekm#altitudeMeters", "name": "intel.measurement.altitude-m"},
    {"path_iri": "https://w3id.org/c360/diekm#assignedRole", "name": "intel.role.assigned"}
  ],
  "contracts": [{"domain_type": "intel.platform", "name": "platform", "birth": ["intel.role.assigned"],
    "groups": [{"name": "measurement", "mode": "reconcile", "predicates": ["intel.measurement.altitude-m"]}]}],
  "closed_ignored_predicates": ["intel.meta.note"]
}`

// TestDecodeMapping is task 4.2: both forms decode to the mapping written
// out by hand.
func TestDecodeMapping(t *testing.T) {
	fromYAML, err := DecodeMappingYAML(strings.NewReader(mappingYAML))
	if err != nil {
		t.Fatal(err)
	}
	fromJSON, err := DecodeMappingJSON(strings.NewReader(mappingJSON))
	if err != nil {
		t.Fatal(err)
	}
	for name, got := range map[string]Mapping{"YAML": fromYAML, "JSON": fromJSON} {
		if !reflect.DeepEqual(got, wantMapping) {
			t.Errorf("%s:\n got %+v\nwant %+v", name, got, wantMapping)
		}
	}
}

func TestDecodeMappingRefuses(t *testing.T) {
	jsonTests := []struct{ name, src, want string }{
		{"empty", ``, "no document"},
		{"unknown top field", `{"bogus": []}`, `unknown field "bogus"`},
		{"unknown group field", `{"contracts": [{"groups": [{"bogus": 1}]}]}`, `unknown field "bogus"`},
		{"trailing value", `{} []`, "trailing data"},
		{"number name", `{"predicates": [{"name": 1}]}`, "name"},
	}
	for _, tt := range jsonTests {
		t.Run("json "+tt.name, func(t *testing.T) {
			if _, err := DecodeMappingJSON(strings.NewReader(tt.src)); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want containing %q", err, tt.want)
			}
		})
	}
	yamlTests := []struct{ name, src, want string }{
		{"empty", ``, "no document"},
		{"unknown top field", "bogus: []\n", "field bogus not found"},
		{"unknown group field", "contracts:\n  - groups:\n      - bogus: x\n", "field bogus not found"},
		{"trailing document", "predicates: []\n---\n{}\n", "trailing document"},
		{"unquoted mode", "contracts:\n  - groups:\n      - mode: 1\n", "line 3: mode must be a string"},
		{"boolean birth member", "contracts:\n  - birth: [true]\n", "birth must be a string"},
	}
	for _, tt := range yamlTests {
		t.Run("yaml "+tt.name, func(t *testing.T) {
			if _, err := DecodeMappingYAML(strings.NewReader(tt.src)); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want containing %q", err, tt.want)
			}
		})
	}
}

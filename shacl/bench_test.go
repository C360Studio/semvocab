package shacl

// Task 7.3: the benchmarks of design D16. They are not part of task verify.
// Run them with
//
//	go test ./shacl -run '^$' -bench . -benchmem -count 10
//
// and see docs/shacl-compilation.md for recorded results. The numbers are
// measurements on one machine, not guarantees.

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/c360studio/semengine/message"
	"github.com/c360studio/semengine/pkg/projection/contract"
	"github.com/c360studio/semengine/vocabulary"
	"github.com/c360studio/semvocab/packs/diekm"
)

// benchInputs builds a synthetic compile input with n property shapes: n
// predicates over the four constrained data types, ten property shapes per
// node shape, every node shape targeting one class, and one reconcile group
// holding every predicate.
func benchInputs(n int) (vocabulary.PackSpec, ShapeSet, Mapping) {
	const ns = "http://example.org/bench/"
	pack := vocabulary.PackSpec{Name: "bench",
		EntityTypes: []vocabulary.EntityTypeSpec{{DomainType: "bench.thing", ClassIRI: ns + "Thing"}}}
	m := Mapping{EntityTypes: []EntityTypeBinding{{ClassIRI: ns + "Thing", DomainTypes: []string{"bench.thing"}}}}
	group := GroupBinding{Name: "all", Mode: contract.ModeReconcile}
	var set ShapeSet
	for i := range n {
		name, path := fmt.Sprintf("bench.prop.p%04d", i), fmt.Sprintf("%sp%04d", ns, i)
		pack.Predicates = append(pack.Predicates, vocabulary.PredicateSpec{Name: name, IRI: path})
		m.Predicates = append(m.Predicates, PredicateBinding{PathIRI: path, Name: name})
		group.Predicates = append(group.Predicates, name)
		if i%10 == 0 {
			set.NodeShapes = append(set.NodeShapes, NodeShape{ID: fmt.Sprintf("%sS%04d", ns, i/10),
				TargetClasses: []string{ns + "Thing"}})
		}
		one := 1
		p := PropertyShape{Path: path, MaxCount: &one}
		switch i % 4 {
		case 0:
			pattern := `^[a-z]+$`
			p.Datatype, p.Pattern, p.In = xsdString, &pattern, &[]string{"alpha", "beta", "gamma"}
		case 1:
			p.Datatype = xsdInt
			p.MinInclusive = &Literal{Lexical: "0", Datatype: xsdInt}
			p.MaxExclusive = &Literal{Lexical: "1000", Datatype: xsdInt}
		case 2:
			p.Datatype = xsdDouble
			p.MinExclusive = &Literal{Lexical: "-1.5E2", Datatype: xsdDouble}
		case 3:
			p.Datatype = xsdBool
		}
		last := &set.NodeShapes[len(set.NodeShapes)-1]
		last.Properties = append(last.Properties, p)
	}
	m.Contracts = []ContractBinding{{DomainType: "bench.thing", Groups: []GroupBinding{group}}}
	return pack, set, m
}

func diekmBenchInputs(b *testing.B) (vocabulary.PackSpec, ShapeSet, Mapping) {
	spec, err := diekm.Spec()
	if err != nil {
		b.Fatal(err)
	}
	shapes, err := ParseTurtle(bytes.NewReader(diekm.Shapes()))
	if err != nil {
		b.Fatal(err)
	}
	m, err := DecodeMappingYAML(bytes.NewReader(diekm.Mapping()))
	if err != nil {
		b.Fatal(err)
	}
	return spec, shapes, m
}

// BenchmarkCompile compiles DIEKM and synthetic sets of 10, 100, and 1,000
// property shapes. Inputs are decoded before the timer starts.
func BenchmarkCompile(b *testing.B) {
	run := func(b *testing.B, pack vocabulary.PackSpec, set ShapeSet, m Mapping) {
		if _, err := Compile(pack, set, m); err != nil {
			b.Fatal(err)
		}
		for b.Loop() {
			if _, err := Compile(pack, set, m); err != nil {
				b.Fatal(err)
			}
		}
	}
	b.Run("diekm", func(b *testing.B) {
		pack, set, m := diekmBenchInputs(b)
		run(b, pack, set, m)
	})
	for _, n := range []int{10, 100, 1000} {
		b.Run(fmt.Sprintf("shapes=%d", n), func(b *testing.B) {
			pack, set, m := benchInputs(n)
			run(b, pack, set, m)
		})
	}
}

// BenchmarkValidateCandidateState validates a DIEKM platform with one
// altitude and one role, and candidates of 10, 100, and 1,000 triples against
// the 1,000-property synthetic bundle, one triple per predicate, each value
// valid, so every constraint runs and the report conforms.
func BenchmarkValidateCandidateState(b *testing.B) {
	run := func(b *testing.B, bundle *Bundle, id string, triples []message.Triple) {
		r, err := bundle.ValidateCandidateState(id, triples)
		if err != nil || !r.Targeted || !r.Conforms {
			b.Fatalf("premise: a conforming targeted report, got %+v, %v", r, err)
		}
		for b.Loop() {
			if _, err := bundle.ValidateCandidateState(id, triples); err != nil {
				b.Fatal(err)
			}
		}
	}
	b.Run("diekm", func(b *testing.B) {
		bundle, err := Compile(diekmBenchInputs(b))
		if err != nil {
			b.Fatal(err)
		}
		const id = "acme.edge.sensor.intel.platform.p1"
		run(b, bundle, id, []message.Triple{
			{Subject: id, Predicate: "intel.measurement.altitude-m", Object: 1250.5},
			{Subject: id, Predicate: "intel.role.assigned", Object: "observer"},
		})
	})
	bundle, err := Compile(benchInputs(1000))
	if err != nil {
		b.Fatal(err)
	}
	const id = "acme.edge.sensor.bench.thing.t1"
	for _, n := range []int{10, 100, 1000} {
		b.Run(fmt.Sprintf("triples=%d", n), func(b *testing.B) {
			triples := make([]message.Triple, n)
			for i := range triples {
				var v any
				switch i % 4 {
				case 0:
					v = "beta"
				case 1:
					v = int64(i)
				case 2:
					v = float64(i) / 3
				case 3:
					v = true
				}
				triples[i] = message.Triple{Subject: id, Predicate: fmt.Sprintf("bench.prop.p%04d", i), Object: v,
					Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
			}
			run(b, bundle, id, triples)
		})
	}
}

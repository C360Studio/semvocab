package oracle

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/c360studio/semvocab/internal/ledger"
	"github.com/c360studio/semvocab/internal/turtle"
	"pgregory.net/rapid"
)

// TestVendoredReleasesMatchLedger loads each vendored file, which checks its
// sha256 against the ledger pin before parsing, and requires the release's
// own ontology header to name the pinned version IRI or dcterms:created date
// and vann:preferredNamespaceUri.
func TestVendoredReleasesMatchLedger(t *testing.T) {
	l, err := ledger.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range []Release{BFO2020, CCO22} {
		g, err := Load(r)
		if err != nil {
			t.Fatal(err)
		}
		versionIRI := l.Pins.Ontologies[string(r)].VersionIRI
		found := false
		for _, ts := range g.bySubject {
			for _, tr := range ts {
				if tr.P.Value == "http://www.w3.org/2002/07/owl#versionIRI" && tr.O.Value == versionIRI {
					found = true
				}
			}
		}
		if !found {
			t.Errorf("%s declares no owl:versionIRI %s", r, versionIRI)
		}
	}
	const (
		dctermsCreated = "http://purl.org/dc/terms/created"
		vannPrefNSURI  = "http://purl.org/vocab/vann/preferredNamespaceUri"
	)
	for _, r := range []Release{SOSA2017, SSN2017, SSNSystem2017} {
		g, err := Load(r)
		if err != nil {
			t.Fatal(err)
		}
		pin := l.Pins.Ontologies[string(r)]
		if got := g.LiteralObjects(pin.Namespace, dctermsCreated); !slices.Equal(got, []string{pin.Created}) {
			t.Errorf("%s dcterms:created = %q, want [%q]", r, got, pin.Created)
		}
		if got := g.LiteralObjects(pin.Namespace, vannPrefNSURI); !slices.Equal(got, []string{pin.Namespace}) {
			t.Errorf("%s vann:preferredNamespaceUri = %q, want [%q]", r, got, pin.Namespace)
		}
	}
	merged, err := LoadMerged(SOSA2017, SSN2017, SSNSystem2017)
	if err != nil {
		t.Fatal(err)
	}
	const (
		sosaSample  = "http://www.w3.org/ns/sosa/Sample"
		sosaFOI     = "http://www.w3.org/ns/sosa/FeatureOfInterest"
		sosaResult  = "http://www.w3.org/ns/sosa/Result"
		ssnProperty = "http://www.w3.org/ns/ssn/Property"
		sysAccuracy = "http://www.w3.org/ns/ssn/systems/Accuracy"
		sysProp     = "http://www.w3.org/ns/ssn/systems/SystemProperty"
	)
	if got := merged.DirectSuperclasses(sosaSample); !slices.Equal(got, []string{sosaFOI, sosaResult}) {
		t.Errorf("merged DirectSuperclasses(Sample) = %v, want [%s %s]", got, sosaFOI, sosaResult)
	}
	if got := merged.Closure(sysAccuracy); !slices.Equal(got, []string{sysProp, ssnProperty}) {
		t.Errorf("merged Closure(Accuracy) = %v, want [%s %s]", got, sysProp, ssnProperty)
	}
}

// TestVerifySHA256RejectsAlteredFile flips one byte of a vendored file and
// requires the sha256 check to refuse it.
func TestVerifySHA256RejectsAlteredFile(t *testing.T) {
	l, err := ledger.Load()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(Path(BFO2020))
	if err != nil {
		t.Fatal(err)
	}
	want := l.Pins.Ontologies[string(BFO2020)].SHA256
	if err := VerifySHA256(data, want); err != nil {
		t.Fatalf("unaltered file: %v", err)
	}
	data[len(data)/2] ^= 0x01
	if err := VerifySHA256(data, want); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("altered file: VerifySHA256 = %v, want a mismatch", err)
	}
}

// TestFixturesMatchLedger requires the vendored W3C fixture tree and the
// ledger's pins.fixtures list to name the same files, and every file to pass
// its sha256 check. NOTICE is SemVocab's own attribution file, not a fixture.
func TestFixturesMatchLedger(t *testing.T) {
	l, err := ledger.Load()
	if err != nil {
		t.Fatal(err)
	}
	pin, ok := l.Pins.Fixtures[string(W3CDataShapes)]
	if !ok {
		t.Fatalf("ledger pins no fixture set %q", W3CDataShapes)
	}
	if want := "internal/oracle/testdata/" + string(W3CDataShapes); pin.VendoredAt != want {
		t.Errorf("vendored_at = %q, want %q", pin.VendoredAt, want)
	}
	root := FixtureRoot(W3CDataShapes)
	var onDisk []string
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		if rel != "NOTICE" {
			onDisk = append(onDisk, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var listed []string
	for _, f := range pin.Files {
		listed = append(listed, f.Path)
		if _, err := ReadFixture(W3CDataShapes, f.Path); err != nil {
			t.Error(err)
		}
	}
	slices.Sort(onDisk)
	slices.Sort(listed)
	if !slices.Equal(onDisk, listed) {
		t.Errorf("vendored files differ from the ledger list:\non disk: %q\nledger:  %q", onDisk, listed)
	}
}

// TestReadFixtureRefusesAlteredOrUnlisted flips one byte of a vendored fixture
// in a copy of the tree and requires the read to fail on the digest, and
// requires a path the ledger does not list to fail before any read.
func TestReadFixtureRefusesAlteredOrUnlisted(t *testing.T) {
	l, err := ledger.Load()
	if err != nil {
		t.Fatal(err)
	}
	pin := l.Pins.Fixtures[string(W3CDataShapes)]
	const vocab = "shacl/shacl.ttl"
	data, err := os.ReadFile(filepath.Join(FixtureRoot(W3CDataShapes), vocab))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "shacl"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(b []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, vocab), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(data)
	if _, err := readFixture(root, pin, vocab); err != nil {
		t.Fatalf("unaltered copy: %v", err)
	}
	altered := slices.Clone(data)
	altered[len(altered)/2] ^= 0x01
	write(altered)
	if _, err := readFixture(root, pin, vocab); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("altered copy: err = %v, want a sha256 mismatch", err)
	}
	if _, err := readFixture(root, pin, "shacl/shacl-shacl.ttl"); err == nil || !strings.Contains(err.Error(), "not listed") {
		t.Fatalf("unlisted path: err = %v, want not listed", err)
	}
}

// TestVerifySHA256RejectsEveryByteFlip is a Rapid property over the vendored
// SHACL vocabulary: changing any one byte to any other value fails the digest
// check. A failing input is replayable with the -rapid.seed Rapid prints.
func TestVerifySHA256RejectsEveryByteFlip(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(FixtureRoot(W3CDataShapes), "shacl", "shacl.ttl"))
	if err != nil {
		t.Fatal(err)
	}
	const want = "1a534caa9d3df5bbc5e5351002bda56ead9226fda0a617c165b218d8bd06b9af" // owner ruling Q21
	if err := VerifySHA256(data, want); err != nil {
		t.Fatalf("unaltered file: %v", err)
	}
	rapid.Check(t, func(t *rapid.T) {
		i := rapid.IntRange(0, len(data)-1).Draw(t, "offset")
		delta := rapid.ByteRange(1, 255).Draw(t, "xor")
		altered := slices.Clone(data)
		altered[i] ^= delta
		if err := VerifySHA256(altered, want); err == nil {
			t.Fatalf("byte %d xor %#x passed the digest check", i, delta)
		}
	})
}

// TestSchemaBundlesMatchLedger requires the vendored OGC Connected Systems tree
// (excluding NOTICE) to equal the union of pins.schemas.swe-3.0.files and
// pins.schemas.sensorml-3.0.files, every listed file in both bundles to pass
// ReadSchemaFile, and both bundles to parse cleanly via LoadSchemaCatalog.
func TestSchemaBundlesMatchLedger(t *testing.T) {
	l, err := ledger.Load()
	if err != nil {
		t.Fatal(err)
	}
	const wantVendoredAt = "internal/oracle/testdata/ogcapi-connected-systems"
	listedSet := map[string]bool{}
	for _, b := range []SchemaBundle{SWE30, SensorML30} {
		pin, ok := l.Pins.Schemas[string(b)]
		if !ok {
			t.Fatalf("ledger pins no schema bundle %q", b)
		}
		if pin.VendoredAt != wantVendoredAt {
			t.Errorf("%s vendored_at = %q, want %q", b, pin.VendoredAt, wantVendoredAt)
		}
		for _, f := range pin.Files {
			listedSet[f.Path] = true
			if _, err := ReadSchemaFile(b, f.Path); err != nil {
				t.Errorf("%s: %v", b, err)
			}
		}
		if _, err := LoadSchemaCatalog(b); err != nil {
			t.Errorf("LoadSchemaCatalog(%s): %v", b, err)
		}
	}
	root := SchemaRoot(SWE30)
	var onDisk []string
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		if rel != "NOTICE" {
			onDisk = append(onDisk, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var listed []string
	for p := range listedSet {
		listed = append(listed, p)
	}
	slices.Sort(onDisk)
	slices.Sort(listed)
	if !slices.Equal(onDisk, listed) {
		t.Errorf("vendored schema files differ from the ledger union:\non disk: %q\nledger:  %q", onDisk, listed)
	}
}

// TestReadSchemaFileRefusesAlteredOrUnlisted flips one byte of a vendored
// schema file in a temp copy and requires readSchemaFile to fail on the digest,
// and requires a path the ledger does not list to fail before any read.
func TestReadSchemaFileRefusesAlteredOrUnlisted(t *testing.T) {
	l, err := ledger.Load()
	if err != nil {
		t.Fatal(err)
	}
	pin := l.Pins.Schemas[string(SWE30)]
	const target = "swecommon/schemas/json/Quantity.json"
	data, err := os.ReadFile(filepath.Join(SchemaRoot(SWE30), filepath.FromSlash(target)))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "swecommon", "schemas", "json"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(b []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(target)), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(data)
	if _, err := readSchemaFile(root, pin, target); err != nil {
		t.Fatalf("unaltered copy: %v", err)
	}
	altered := slices.Clone(data)
	altered[len(altered)/2] ^= 0x01
	write(altered)
	if _, err := readSchemaFile(root, pin, target); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("altered copy: err = %v, want a sha256 mismatch", err)
	}
	if _, err := readSchemaFile(root, pin, "swecommon/schemas/json/Unlisted.json"); err == nil || !strings.Contains(err.Error(), "not listed") {
		t.Fatalf("unlisted path: err = %v, want not listed", err)
	}
	if _, err := readSchemaFile(root, pin, "../LICENSE"); err == nil || !strings.Contains(err.Error(), "not listed") {
		t.Fatalf("traversal path: err = %v, want not listed", err)
	}
	if _, err := ReadSchemaFile("unknown", target); err == nil || !strings.Contains(err.Error(), "no schema bundle") {
		t.Fatalf("unknown bundle: err = %v, want no schema bundle", err)
	}
}

// TestIRIObjects reads owl:inverseOf and rdfs:subPropertyOf objects from the
// pinned bfo-2020 file, whose lines 529-541 declare "bearer of" (BFO_0000196)
// with owl:inverseOf BFO_0000197 and rdfs:subPropertyOf BFO_0000194, and
// requires literal (rdfs:label) and blank-node (rdfs:range of BFO_0000058)
// objects to be skipped.
//
// spec: openspec/specs/catalogs-and-packs/spec.md#Requirement: BFO relation axioms follow the release
func TestIRIObjects(t *testing.T) {
	g, err := Load(BFO2020)
	if err != nil {
		t.Fatal(err)
	}
	const (
		obo       = "http://purl.obolibrary.org/obo/"
		rdfsRange = "http://www.w3.org/2000/01/rdf-schema#range"
	)
	tests := []struct {
		s, p string
		want []string
	}{
		{obo + "BFO_0000196", OWLInverseOf, []string{obo + "BFO_0000197"}},
		{obo + "BFO_0000196", RDFSSubPropertyOf, []string{obo + "BFO_0000194"}},
		{obo + "BFO_0000197", OWLInverseOf, nil},
		{obo + "BFO_0000196", RDFSLabel, nil},
		{obo + "BFO_0000058", rdfsRange, nil},
		{obo + "BFO_0000063", RDFType, []string{OWLObjectProp, "http://www.w3.org/2002/07/owl#TransitiveProperty"}},
	}
	for _, tt := range tests {
		if got := g.IRIObjects(tt.s, tt.p); !slices.Equal(got, tt.want) {
			t.Errorf("IRIObjects(%s, %s) = %q, want %q", tt.s, tt.p, got, tt.want)
		}
	}
}

// TestObjects reads BFO_0000196's rdfs:domain from the pinned bfo-2020 file,
// where lines 532-538 write it as a blank node whose owl:intersectionOf list
// holds BFO_0000004 and a blank node with owl:complementOf BFO_0000006, and
// walks the list through rdf:first and rdf:rest.
//
// spec: openspec/specs/catalogs-and-packs/spec.md#Requirement: BFO projection hops follow the release
func TestObjects(t *testing.T) {
	g, err := Load(BFO2020)
	if err != nil {
		t.Fatal(err)
	}
	const (
		obo          = "http://purl.obolibrary.org/obo/"
		rdfsDomain   = "http://www.w3.org/2000/01/rdf-schema#domain"
		owlIntersect = "http://www.w3.org/2002/07/owl#intersectionOf"
		owlComplem   = "http://www.w3.org/2002/07/owl#complementOf"
		rdfFirst     = "http://www.w3.org/1999/02/22-rdf-syntax-ns#first"
		rdfRest      = "http://www.w3.org/1999/02/22-rdf-syntax-ns#rest"
		rdfNil       = "http://www.w3.org/1999/02/22-rdf-syntax-ns#nil"
	)
	one := func(s, p string) turtle.Term {
		t.Helper()
		got := g.Objects(s, p)
		if len(got) != 1 {
			t.Fatalf("Objects(%s, %s) = %v, want one term", s, p, got)
		}
		return got[0]
	}
	domain := one(obo+"BFO_0000196", rdfsDomain)
	if domain.Kind != turtle.Blank {
		t.Fatalf("BFO_0000196 rdfs:domain = %v, want a blank node", domain)
	}
	var items []turtle.Term
	for node := one(domain.Value, owlIntersect); node.Value != rdfNil; node = one(node.Value, rdfRest) {
		items = append(items, one(node.Value, rdfFirst))
	}
	if len(items) != 2 {
		t.Fatalf("owl:intersectionOf list = %v, want two members", items)
	}
	if items[0].Kind != turtle.IRI || items[0].Value != obo+"BFO_0000004" {
		t.Errorf("first member = %v, want IRI %sBFO_0000004", items[0], obo)
	}
	if items[1].Kind != turtle.Blank {
		t.Fatalf("second member = %v, want a blank node", items[1])
	}
	if c := one(items[1].Value, owlComplem); c.Kind != turtle.IRI || c.Value != obo+"BFO_0000006" {
		t.Errorf("owl:complementOf = %v, want IRI %sBFO_0000006", c, obo)
	}
	if got := g.Objects(obo+"BFO_0000196", RDFSLabel); len(got) != 1 || got[0].Kind != turtle.Literal || got[0].Value != "bearer of" {
		t.Errorf("Objects(BFO_0000196, rdfs:label) = %v, want the literal \"bearer of\"", got)
	}
}

package oracle

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/c360studio/semvocab/internal/ledger"
	"pgregory.net/rapid"
)

// TestVendoredReleasesMatchLedger loads each vendored file, which checks its
// sha256 against the ledger pin before parsing, and requires the release's
// own ontology header to name the pinned version IRI.
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

package oracle

import (
	"os"
	"strings"
	"testing"

	"github.com/c360studio/semvocab/internal/ledger"
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

package oracle

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"

	"github.com/c360studio/semvocab/internal/ledger"
)

// FixtureSet names a vendored set of external test fixtures by its key under
// the ledger's pins.fixtures.
type FixtureSet string

// W3CDataShapes is the SHACL vocabulary and the selected SHACL Core test-suite
// cases from w3c/data-shapes, vendored unmodified under
// testdata/w3c-data-shapes with their upstream paths.
const W3CDataShapes FixtureSet = "w3c-data-shapes"

// FixtureRoot is the absolute path of a fixture set's vendored tree.
func FixtureRoot(s FixtureSet) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata", string(s))
}

// ReadFixture returns the bytes of one vendored fixture, named by its path in
// the source repository, after checking its sha256 against the ledger. A path
// the ledger does not list fails before any read.
func ReadFixture(s FixtureSet, p string) ([]byte, error) {
	l, err := ledger.Load()
	if err != nil {
		return nil, err
	}
	pin, ok := l.Pins.Fixtures[string(s)]
	if !ok {
		return nil, fmt.Errorf("oracle: ledger pins no fixture set %q", s)
	}
	return readFixture(FixtureRoot(s), pin, p)
}

func readFixture(root string, pin ledger.FixtureSet, p string) ([]byte, error) {
	f, ok := pin.File(p)
	if !ok || !filepath.IsLocal(filepath.FromSlash(p)) || path.Clean(p) != p {
		return nil, fmt.Errorf("oracle: fixture %s is not listed in pins.fixtures", p)
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(p)))
	if err != nil {
		return nil, fmt.Errorf("oracle: %w", err)
	}
	if err := VerifySHA256(data, f.SHA256); err != nil {
		return nil, fmt.Errorf("oracle: fixture %s: %w", p, err)
	}
	return data, nil
}

// Command semstreamsfixture writes the SemStreams-pin parity fixture for
// catalog/bfo from SemStreams' vocabulary/bfo/bfo.go and hierarchy.go, as
// extracted at the pin by scripts/gen-semstreams-fixture.sh. It evaluates the
// constants with go/types and reads the SubClassOf map literal; it does not
// import the SemStreams module.
//
// Usage: semstreamsfixture -pin <revision> <bfo.go> <hierarchy.go>
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/c360studio/semvocab/internal/gosrc"
)

func main() {
	pin := flag.String("pin", "", "SemStreams revision the files were extracted at")
	flag.Parse()
	if *pin == "" || flag.NArg() != 2 {
		fmt.Fprintln(os.Stderr, "usage: semstreamsfixture -pin <revision> <bfo.go> <hierarchy.go>")
		os.Exit(2)
	}
	if err := run(os.Stdout, *pin, flag.Arg(0), flag.Arg(1)); err != nil {
		fmt.Fprintln(os.Stderr, "semstreamsfixture:", err)
		os.Exit(1)
	}
}

func run(w io.Writer, pin, bfoFile, hierarchyFile string) error {
	consts, err := gosrc.LoadFiles(bfoFile)
	if err != nil {
		return err
	}
	edges, err := gosrc.MapLiteralIdents(hierarchyFile, "SubClassOf")
	if err != nil {
		return err
	}
	sums := map[string]string{}
	for _, f := range []string{bfoFile, hierarchyFile} {
		data, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		sums[f] = hex.EncodeToString(sum[:])
	}
	names := make([]string, 0, len(consts.Consts))
	for n := range consts.Consts {
		names = append(names, n)
	}
	sort.Strings(names)

	lines := []string{
		"# SemStreams-pin parity fixture for catalog/bfo. Generated; do not edit.",
		"# Command: scripts/gen-semstreams-fixture.sh <semstreams clone>",
		"# Source: github.com/C360Studio/semstreams at " + pin,
		"#   vocabulary/bfo/bfo.go sha256 " + sums[bfoFile],
		"#   vocabulary/bfo/hierarchy.go sha256 " + sums[hierarchyFile],
		"# Lines: \"const <Name> <value>\" for every exported string constant;",
		"#        \"subclass <Child> <Parent>\" for every SubClassOf entry, in source order.",
		"pin " + pin,
	}
	for _, n := range names {
		lines = append(lines, "const "+n+" "+consts.Consts[n])
	}
	for _, e := range edges {
		lines = append(lines, "subclass "+e[0]+" "+e[1])
	}
	for _, l := range lines {
		if _, err := fmt.Fprintln(w, l); err != nil {
			return err
		}
	}
	return nil
}

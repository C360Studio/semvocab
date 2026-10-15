// Package contract holds SemVocab's machine-checked repository contracts: agent-setup parity and
// the merge-check and spec-queue scripts, carried from SemEngine's internal/harness/contract.
// Every check lives in a _test.go file and runs over the whole repository; each also has a
// sensitivity test that seeds the violation in a temporary tree and requires the check to name it,
// so a check that silently matches nothing fails.
//
// The package has no non-test code; this file exists so the directory is an ordinary package.
package contract

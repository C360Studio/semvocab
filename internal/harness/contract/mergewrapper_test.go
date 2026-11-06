package contract

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const fakeAteam = `#!/bin/sh
printf '%s\n' "$*" > "$FAKE_ATEAM_ARGS"
exit "$FAKE_ATEAM_EXIT"
`

const decoyAteam = `#!/bin/sh
printf 'ateam on PATH: %s\n' "$*" > "$FAKE_ATEAM_ARGS"
exit 97
`

func runWrapper(t *testing.T, script string, viaEnv bool, exit int, args ...string) (int, string) {
	t.Helper()
	root, _ := writeTree(t, map[string]string{"scripts/merge-check.sh": script})
	argsFile := filepath.Join(t.TempDir(), "args")
	extra := []string{"FAKE_ATEAM_ARGS=" + argsFile, fmt.Sprintf("FAKE_ATEAM_EXIT=%d", exit)}
	onPath := fakeAteam
	if viaEnv {
		prog := filepath.Join(t.TempDir(), "fake-ateam")
		if err := os.WriteFile(prog, []byte(fakeAteam), 0o755); err != nil {
			t.Fatal(err)
		}
		onPath = decoyAteam
		extra = append(extra, "ATEAM="+prog)
	}
	var env []string
	for _, kv := range fakeBin(t, map[string]string{"ateam": onPath}) {
		if !strings.HasPrefix(kv, "ATEAM=") {
			env = append(env, kv)
		}
	}
	env = append(env, extra...)
	cmd := exec.Command("bash", append([]string{filepath.Join(root, "scripts", "merge-check.sh")}, args...)...)
	cmd.Dir = root
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	status := 0
	var ee *exec.ExitError
	switch {
	case errors.As(err, &ee):
		status = ee.ExitCode()
	case err != nil:
		t.Fatalf("run merge-check.sh: %v\n%s", err, out)
	}
	got, _ := os.ReadFile(argsFile)
	return status, strings.TrimSpace(string(got))
}

func wrapperViolations(t *testing.T, script string) []string {
	t.Helper()
	var out []string
	for _, viaEnv := range []bool{true, false} {
		how := "ateam on PATH"
		if viaEnv {
			how = "ATEAM"
		}
		for _, exit := range []int{0, 1, 2} {
			status, args := runWrapper(t, script, viaEnv, exit, "37")
			if status != exit {
				out = append(out, fmt.Sprintf("merge-check.sh via %s: program exited %d, script exited %d", how, exit, status))
			}
			if args != "review gate 37" {
				out = append(out, fmt.Sprintf("merge-check.sh via %s: program got arguments %q, want %q", how, args, "review gate 37"))
			}
		}
	}
	return out
}

// spec: openspec/specs/merge-gate/spec.md#Requirement: Known-flake check
// spec: openspec/specs/merge-gate/spec.md#Requirement: Up-to-date rule
// spec: openspec/specs/merge-gate/spec.md#Requirement: Cross-agent review lines
func TestMergeCheckWrapper(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "merge-check.sh"))
	if err != nil {
		t.Fatal(err)
	}
	requireNoViolations(t, "merge-check.sh", wrapperViolations(t, string(data)))
}

// spec: openspec/specs/merge-gate/spec.md#Requirement: Cross-agent review lines
func TestMergeCheckWrapperSensitivity(t *testing.T) {
	const good = "#!/usr/bin/env bash\nset -euo pipefail\nexec \"${ATEAM:-ateam}\" review gate \"$@\"\n"
	requireNoViolations(t, "clean fixture", wrapperViolations(t, good))
	for _, tc := range []struct {
		name, old, repl string
		wants           []string
	}{
		{"arguments dropped", ` "$@"`, "", []string{"got arguments", "review gate 37"}},
		{"status swallowed", `exec "${ATEAM:-ateam}" review gate "$@"`, `"${ATEAM:-ateam}" review gate "$@" || true`,
			[]string{"exited 1, script exited 0"}},
		{"ATEAM ignored", `"${ATEAM:-ateam}"`, "ateam", []string{"via ATEAM", "got arguments"}},
		{"another subcommand", "review gate", "review check", []string{"review gate 37"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(good, tc.old) {
				t.Fatalf("fixture lacks %q", tc.old)
			}
			requireViolation(t, wrapperViolations(t, strings.Replace(good, tc.old, tc.repl, 1)), tc.wants...)
		})
	}
}

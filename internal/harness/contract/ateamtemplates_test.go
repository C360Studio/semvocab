package contract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

// semvocabOwned lists the paths ateam init writes (or preserves) that SemVocab keeps its own text in.
var semvocabOwned = map[string]string{
	".gitignore":                        "SemVocab's ignore list with ateam entries appended",
	"AGENTS.md":                         "SemVocab's agent guidance",
	"CLAUDE.md":                         "SemVocab's import of AGENTS.md",
	".agents/README.md":                 "SemVocab's agent profiles",
	".agents/protocol.md":               "SemVocab's shared work protocol",
	".agents/review-policy.json":        "SemVocab's review policy; checked explicitly below and by ateam:doctor",
	".agents/rules-index.md":            "SemVocab's rule index",
	".agents/skills/preflight/SKILL.md": "SemVocab's gate table with ateam's section markers",
	".claude/skills/preflight/SKILL.md": "adapter for SemVocab's preflight skill",
	"openspec/config.yaml":              "SemVocab's OpenSpec configuration",
	"openspec/changes/.gitkeep":         "openspec/changes/ holds SemVocab's changes",
	"openspec/specs/.gitkeep":           "openspec/specs/ holds SemVocab's specs",
	".claude/settings.json":             "client configuration; doctor checks its hooks",
	".codex/config.toml":                "client configuration; doctor checks its MCP registration",
	".codex/hooks.json":                 "client configuration; doctor checks its hooks",
	".gemini/settings.json":             "client configuration; doctor checks its hooks and MCP registration",
	".mcp.json":                         "client configuration; doctor checks its MCP registration",
}

var ateamPin = regexp.MustCompile(`ateam/cmd/ateam@(v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?)`)

func ateamPins(files map[string][]byte) (string, []string) {
	var all, per, violations []string
	for _, name := range slices.Sorted(maps.Keys(files)) {
		var versions []string
		for _, m := range ateamPin.FindAllSubmatch(files[name], -1) {
			v := string(m[1])
			if !slices.Contains(versions, v) {
				versions = append(versions, v)
			}
			if !slices.Contains(all, v) {
				all = append(all, v)
			}
		}
		if len(versions) == 0 {
			violations = append(violations, fmt.Sprintf(".github/workflows/%s: no ateam install line", name))
		}
		per = append(per, fmt.Sprintf("%s %v", name, versions))
	}
	if len(all) > 1 {
		violations = append(violations, fmt.Sprintf(".github/workflows: ateam install lines name versions %s; want one version", strings.Join(per, ", ")))
	}
	if len(violations) > 0 {
		return "", violations
	}
	return all[0], nil
}

// spec: openspec/specs/agent-setup/spec.md#Requirement: Ateam bootstrap and doctor parity
func TestAteamTemplatesUnchanged(t *testing.T) {
	root := repoRoot(t)
	requireNoViolations(t, "ateam templates and policy", ateamTemplateViolations(root, ateamTemplates(t, root)))
}

// spec: openspec/specs/agent-setup/spec.md#Requirement: Ateam bootstrap and doctor parity
func TestAteamTemplatesUnchangedSensitivity(t *testing.T) {
	root := repoRoot(t)
	templates := ateamTemplates(t, root)
	base := make(map[string]string, len(templates)+3)
	for rel := range templates {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		base[rel] = string(data)
	}
	for _, rel := range []string{".agents/review-policy.json", ".agents/hosts/enjoy.toml", ".agents/hosts/herdr.toml"} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		base[rel] = string(data)
	}
	clean, _ := writeTree(t, base)
	requireNoViolations(t, "clean fixture", ateamTemplateViolations(clean, templates))

	for _, tc := range []struct {
		name  string
		edit  func(map[string]string)
		wants []string
	}{
		{"a line added to a base contract", func(m map[string]string) {
			m[".agents/contracts/reviewer.md"] += "A SemVocab rule.\n"
		}, []string{".agents/contracts/reviewer.md", "differs"}},
		{"a base contract deleted", func(m map[string]string) {
			delete(m, ".agents/contracts/architect.md")
		}, []string{".agents/contracts/architect.md", "missing"}},
		{"review_launch changed away from agent", func(m map[string]string) {
			m[".agents/review-policy.json"] = strings.Replace(m[".agents/review-policy.json"], `"review_launch": "agent"`, `"review_launch": "manual"`, 1)
		}, []string{".agents/review-policy.json", "review_launch", "agent"}},
		{"host manifest deleted", func(m map[string]string) {
			delete(m, ".agents/hosts/enjoy.toml")
		}, []string{".agents/hosts/enjoy.toml", "missing"}},
		{"host manifest missing notify action", func(m map[string]string) {
			m[".agents/hosts/herdr.toml"] = "[launch]\ntool = \"split and start\"\n"
		}, []string{".agents/hosts/herdr.toml", "notify"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := maps.Clone(base)
			tc.edit(files)
			dir, _ := writeTree(t, files)
			requireViolation(t, ateamTemplateViolations(dir, templates), tc.wants...)
		})
	}
}

func ateamTemplates(t *testing.T, root string) map[string][]byte {
	t.Helper()
	workflows := map[string][]byte{}
	for _, name := range []string{"ci.yml", "merge-check.yml"} {
		data, err := os.ReadFile(filepath.Join(root, ".github", "workflows", name))
		if err != nil {
			t.Fatal(err)
		}
		workflows[name] = data
	}
	pin, violations := ateamPins(workflows)
	requireNoViolations(t, "ateam pin", violations)
	install := "GOTOOLCHAIN=auto go install github.com/c360studio/ateam/cmd/ateam@" + pin
	if _, err := exec.LookPath("ateam"); err != nil {
		t.Fatalf("ateam is not on PATH; install the pinned release: %s", install)
	}
	out, err := exec.CommandContext(t.Context(), "ateam", "version").CombinedOutput()
	if err != nil {
		t.Fatalf("ateam version: %v\n%s", err, out)
	}
	if !slices.Contains(strings.Fields(string(out)), pin) {
		t.Fatalf("ateam version reports %q, want %s; put $(go env GOPATH)/bin first on PATH: %s", strings.TrimSpace(string(out)), pin, install)
	}

	dir := t.TempDir()
	policy, err := os.ReadFile(filepath.Join(root, ".agents", "review-policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".agents"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".agents", "review-policy.json"), policy, 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.CommandContext(t.Context(), "ateam", "init", "-dir", dir).CombinedOutput(); err != nil {
		t.Fatalf("ateam init: %v\n%s", err, out)
	}
	templates := map[string][]byte{}
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if _, owned := semvocabOwned[rel]; owned {
			return nil
		}
		data, err := os.ReadFile(path)
		templates[rel] = data
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(templates) == 0 {
		t.Fatal("ateam init wrote no template file SemVocab does not own")
	}
	return templates
}

func ateamTemplateViolations(root string, templates map[string][]byte) []string {
	var out []string
	for _, rel := range slices.Sorted(maps.Keys(templates)) {
		got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			out = append(out, fmt.Sprintf("%s: missing; ateam init writes it at the pinned release", rel))
			continue
		}
		if !bytes.Equal(got, templates[rel]) {
			out = append(out, fmt.Sprintf("%s: differs from what ateam init writes at the pinned release; put SemVocab rules in the matching semvocab-* file", rel))
		}
	}
	out = append(out, checkReviewPolicyAndHosts(root)...)
	return out
}

type hostActionSpec struct {
	Tool    string   `toml:"tool"`
	Command []string `toml:"command"`
}

func checkReviewPolicyAndHosts(root string) []string {
	var out []string
	policyData, err := os.ReadFile(filepath.Join(root, ".agents", "review-policy.json"))
	if err != nil {
		out = append(out, ".agents/review-policy.json: missing")
	} else {
		var p struct {
			CrossProvider bool   `json:"cross_provider"`
			ReviewLaunch  string `json:"review_launch"`
		}
		if err := json.Unmarshal(policyData, &p); err != nil {
			out = append(out, fmt.Sprintf(".agents/review-policy.json: invalid JSON: %v", err))
		} else {
			if !p.CrossProvider {
				out = append(out, ".agents/review-policy.json: cross_provider must be true")
			}
			if p.ReviewLaunch != "agent" {
				out = append(out, fmt.Sprintf(".agents/review-policy.json: review_launch = %q, want \"agent\"", p.ReviewLaunch))
			}
		}
	}
	for _, host := range []string{".agents/hosts/enjoy.toml", ".agents/hosts/herdr.toml"} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(host)))
		if err != nil {
			out = append(out, fmt.Sprintf("%s: missing host manifest", host))
			continue
		}
		var manifest struct {
			Launch hostActionSpec `toml:"launch"`
			Notify hostActionSpec `toml:"notify"`
		}
		if _, err := toml.Decode(string(data), &manifest); err != nil {
			out = append(out, fmt.Sprintf("%s: invalid TOML: %v", host, err))
			continue
		}
		if strings.TrimSpace(manifest.Launch.Tool) == "" && len(manifest.Launch.Command) == 0 {
			out = append(out, fmt.Sprintf("%s: launch section must define tool or command", host))
		}
		if strings.TrimSpace(manifest.Notify.Tool) == "" && len(manifest.Notify.Command) == 0 {
			out = append(out, fmt.Sprintf("%s: notify section must define tool or command", host))
		}
	}
	return out
}

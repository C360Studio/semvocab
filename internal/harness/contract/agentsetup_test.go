package contract

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// Steps 1–8 of .agents/README.md's read-only parity smoke, held by a unit test and its sensitivity
// test (carried from SemEngine's harness-boundaries spec, "Agent setup and rule-index parity"). Expected roles, tool lists, sandbox modes, model/effort
// pins, and caps are constants here rather than read from .agents/README.md.
const (
	maxAgentsMDBytes    = 24000
	maxClaudeMDLines    = 10
	maxRoleAdapterLines = 20
	maxSkillLines       = 25
)

type agentRoleSpec struct {
	name        string
	readOnly    bool
	claudeModel string
	codexModel  string
	codexEffort string
}

var (
	agentRoleSpecs = []agentRoleSpec{
		{name: "semvocab-architect", readOnly: true, claudeModel: "opus", codexModel: "gpt-6-astra", codexEffort: "medium"},
		{name: "semvocab-developer", readOnly: false, claudeModel: "opus", codexModel: "gpt-6-sol", codexEffort: "medium"},
		{name: "semvocab-reviewer", readOnly: true, claudeModel: "opus", codexModel: "gpt-6-astra", codexEffort: "high"},
		{name: "semvocab-technical-writer", readOnly: false, claudeModel: "sonnet", codexModel: "gpt-6-sol", codexEffort: "medium"},
	}
	readOnlyClaudeTools = []string{"Read", "Bash", "Grep", "Glob", "Skill"}
	writeClaudeTools    = []string{"Read", "Edit", "Write", "Bash", "Grep", "Glob", "Skill"}
	sharedSkills        = []struct{ canonical, claude string }{
		{"semvocab-handoff", "semvocab-handoff"},
		{"semvocab-pickup", "semvocab-pickup"},
		{"semvocab-preflight", "preflight"},
	}
	expectedRuleDomains = []string{
		"1. Go and package boundaries",
		"2. Vocabulary, packs, and catalogs",
		"3. Testing and verification",
		"4. Porting, provenance, and dependencies",
		"5. Workflow, review, and merge gates",
	}
)

func TestAgentSetupParity(t *testing.T) {
	requireNoViolations(t, "agent setup parity", agentSetupViolations(repoRoot(t)))
}

func TestAgentSetupParitySensitivity(t *testing.T) {
	base := copyAgentSetupFixture(t, repoRoot(t))
	edit := func(rel string, fn func(string) string) map[string]string {
		m := make(map[string]string, len(base))
		for k, v := range base {
			m[k] = v
		}
		if fn == nil {
			delete(m, rel)
		} else {
			m[rel] = fn(m[rel])
		}
		return m
	}
	root, _ := writeTree(t, base)
	requireNoViolations(t, "clean fixture", agentSetupViolations(root))

	for _, tc := range []struct {
		name  string
		files map[string]string
		wants []string
	}{
		{"missing canonical contract", edit(".agents/contracts/semvocab-architect.md", nil),
			[]string{".agents/contracts/semvocab-architect.md", "missing"}},
		{"missing codex adapter", edit(".codex/agents/semvocab-reviewer.toml", nil),
			[]string{".codex/agents/semvocab-reviewer.toml", "missing"}},
		{"wrong contract path in claude adapter", edit(".claude/agents/semvocab-reviewer.md", func(s string) string {
			return strings.ReplaceAll(s, "semvocab-reviewer.md", "semvocab-developer.md")
		}), []string{".claude/agents/semvocab-reviewer.md", ".agents/contracts/semvocab-reviewer.md"}},
		{"read-only claude adapter granted Edit", edit(".claude/agents/semvocab-architect.md", func(s string) string {
			return strings.Replace(s, "tools: Read, Bash, Grep, Glob, Skill", "tools: Read, Edit, Bash, Grep, Glob, Skill", 1)
		}), []string{".claude/agents/semvocab-architect.md", "tools"}},
		{"claude adapter lists LSP tool", edit(".claude/agents/semvocab-developer.md", func(s string) string {
			return strings.Replace(s, "Skill", "Skill, LSP", 1)
		}), []string{".claude/agents/semvocab-developer.md", "tools"}},
		{"read-only codex adapter missing sandbox_mode", edit(".codex/agents/semvocab-reviewer.toml", func(s string) string {
			return strings.Replace(s, `sandbox_mode = "read-only"`+"\n", "", 1)
		}), []string{".codex/agents/semvocab-reviewer.toml", "sandbox_mode"}},
		{"write codex adapter sets sandbox_mode", edit(".codex/agents/semvocab-developer.toml", func(s string) string {
			return s + "sandbox_mode = \"read-only\"\n"
		}), []string{".codex/agents/semvocab-developer.toml", "sandbox_mode"}},
		{"write codex adapter sets literal-string sandbox_mode", edit(".codex/agents/semvocab-developer.toml", func(s string) string {
			return s + "sandbox_mode = 'read-only'\n"
		}), []string{".codex/agents/semvocab-developer.toml", "sandbox_mode"}},
		{"write codex adapter sets sandbox_mode with inline comment", edit(".codex/agents/semvocab-developer.toml", func(s string) string {
			return s + "sandbox_mode = \"read-only\" # override\n"
		}), []string{".codex/agents/semvocab-developer.toml", "sandbox_mode"}},
		{"CLAUDE.md drops @AGENTS.md import", edit("CLAUDE.md", func(string) string {
			return "# SemVocab\n\nCustom rules here.\n"
		}), []string{"CLAUDE.md", "@AGENTS.md"}},
		{"CLAUDE.md adds section heading", edit("CLAUDE.md", func(s string) string {
			return s + "\n## Local Rule\n"
		}), []string{"CLAUDE.md", "section heading"}},
		{"AGENTS.md reaches 24000 bytes", edit("AGENTS.md", func(s string) string {
			return s + strings.Repeat("x", maxAgentsMDBytes)
		}), []string{"AGENTS.md", "24000"}},
		{"AGENTS.md and rules-index.md row count mismatch", edit("AGENTS.md", func(s string) string {
			return strings.Replace(s, "| `verify` | The pinned revive passes (`revive.toml`; warnings fail) | `task lint` |\n", "", 1)
		}), []string{"AGENTS.md", ".agents/rules-index.md", "1. Go and package boundaries"}},
		{"role adapter contains checkbox", edit(".claude/agents/semvocab-developer.md", func(s string) string {
			return s + "- [ ] copied checklist item\n"
		}), []string{".claude/agents/semvocab-developer.md", "checkbox"}},
		{"role adapter exceeds line limit", edit(".codex/agents/semvocab-developer.toml", func(s string) string {
			return s + strings.Repeat("# extra line\n", 15)
		}), []string{".codex/agents/semvocab-developer.toml", "lines"}},
		{"skill adapter points to wrong canonical skill", edit(".claude/skills/preflight/SKILL.md", func(s string) string {
			return strings.Replace(s, "semvocab-preflight/SKILL.md", "semvocab-handoff/SKILL.md", 1)
		}), []string{".claude/skills/preflight/SKILL.md", "../../../.agents/skills/semvocab-preflight/SKILL.md"}},
		{"new canonical skill missing claude adapter", edit(".agents/skills/new-review-skill/SKILL.md", func(string) string {
			return "# New Skill\n"
		}), []string{".claude/skills/new-review-skill/SKILL.md", "missing Claude skill adapter"}},
		{"claude model mismatch", edit(".claude/agents/semvocab-technical-writer.md", func(s string) string {
			return strings.Replace(s, "model: sonnet", "model: opus", 1)
		}), []string{".claude/agents/semvocab-technical-writer.md", "model", "sonnet"}},
		{"codex effort mismatch", edit(".codex/agents/semvocab-developer.toml", func(s string) string {
			return strings.Replace(s, `model_reasoning_effort = "medium"`, `model_reasoning_effort = "high"`, 1)
		}), []string{".codex/agents/semvocab-developer.toml", "model_reasoning_effort", "medium"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir, _ := writeTree(t, tc.files)
			requireViolation(t, agentSetupViolations(dir), tc.wants...)
		})
	}
}

func copyAgentSetupFixture(t *testing.T, root string) map[string]string {
	t.Helper()
	rels := []string{
		"AGENTS.md",
		"CLAUDE.md",
		".agents/README.md",
		".agents/protocol.md",
		".agents/rules-index.md",
	}
	for _, r := range agentRoleSpecs {
		rels = append(rels,
			".agents/contracts/"+r.name+".md",
			".claude/agents/"+r.name+".md",
			".codex/agents/"+r.name+".toml",
		)
	}
	for _, s := range discoverSkills(root) {
		rels = append(rels,
			".agents/skills/"+s.canonical+"/SKILL.md",
			".claude/skills/"+s.claude+"/SKILL.md",
		)
	}
	files := make(map[string]string, len(rels))
	for _, rel := range rels {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		files[rel] = string(data)
	}
	return files
}

func agentSetupViolations(root string) []string {
	var out []string
	out = append(out, checkRequiredAgentFiles(root)...)
	for _, role := range agentRoleSpecs {
		out = append(out, checkClaudeRoleAdapter(root, role)...)
		out = append(out, checkCodexRoleAdapter(root, role)...)
	}
	out = append(out, checkRulesAndSizeParity(root)...)
	out = append(out, checkSkillAdapters(root)...)
	return out
}

func checkRequiredAgentFiles(root string) []string {
	var out []string
	required := []string{
		"AGENTS.md",
		"CLAUDE.md",
		".agents/README.md",
		".agents/protocol.md",
		".agents/rules-index.md",
	}
	for _, r := range agentRoleSpecs {
		required = append(required,
			".agents/contracts/"+r.name+".md",
			".claude/agents/"+r.name+".md",
			".codex/agents/"+r.name+".toml",
		)
	}
	for _, rel := range required {
		info, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil || info.IsDir() || info.Size() == 0 {
			out = append(out, fmt.Sprintf("%s: missing or empty required agent file", rel))
		}
	}
	return out
}

var contractPathPattern = regexp.MustCompile(`\.agents/contracts/[A-Za-z0-9_.-]+\.md`)

func checkClaudeRoleAdapter(root string, role agentRoleSpec) []string {
	rel := ".claude/agents/" + role.name + ".md"
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return nil
	}
	text := string(data)
	var out []string
	out = append(out, checkAdapterCommon(rel, role.name, text)...)

	rest, ok := strings.CutPrefix(text, "---\n")
	if !ok {
		return append(out, fmt.Sprintf("%s: missing YAML frontmatter", rel))
	}
	fmText, _, ok := strings.Cut(rest, "\n---\n")
	if !ok {
		return append(out, fmt.Sprintf("%s: unterminated YAML frontmatter", rel))
	}
	var fm struct {
		Name  string `yaml:"name"`
		Tools string `yaml:"tools"`
		Model string `yaml:"model"`
	}
	if err := yaml.Unmarshal([]byte(fmText), &fm); err != nil {
		return append(out, fmt.Sprintf("%s: invalid frontmatter: %v", rel, err))
	}
	if fm.Name != role.name {
		out = append(out, fmt.Sprintf("%s: frontmatter name = %q, want %q", rel, fm.Name, role.name))
	}
	if fm.Model != role.claudeModel {
		out = append(out, fmt.Sprintf("%s: frontmatter model = %q, want %q", rel, fm.Model, role.claudeModel))
	}
	var gotTools []string
	for _, tok := range strings.Split(fm.Tools, ",") {
		if t := strings.TrimSpace(tok); t != "" {
			gotTools = append(gotTools, t)
		}
	}
	wantTools := writeClaudeTools
	if role.readOnly {
		wantTools = readOnlyClaudeTools
	}
	if !slices.Equal(gotTools, wantTools) {
		out = append(out, fmt.Sprintf("%s: frontmatter tools = %v, want %v", rel, gotTools, wantTools))
	}
	return out
}

func checkCodexRoleAdapter(root string, role agentRoleSpec) []string {
	rel := ".codex/agents/" + role.name + ".toml"
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return nil
	}
	text := string(data)
	var out []string
	out = append(out, checkAdapterCommon(rel, role.name, text)...)

	var cfg struct {
		Name                 string `toml:"name"`
		Model                string `toml:"model"`
		ModelReasoningEffort string `toml:"model_reasoning_effort"`
		SandboxMode          string `toml:"sandbox_mode"`
	}
	md, err := toml.Decode(text, &cfg)
	if err != nil {
		return append(out, fmt.Sprintf("%s: invalid TOML: %v", rel, err))
	}
	if cfg.Name != role.name {
		out = append(out, fmt.Sprintf("%s: name = %q, want %q", rel, cfg.Name, role.name))
	}
	if cfg.Model != role.codexModel {
		out = append(out, fmt.Sprintf("%s: model = %q, want %q", rel, cfg.Model, role.codexModel))
	}
	if cfg.ModelReasoningEffort != role.codexEffort {
		out = append(out, fmt.Sprintf("%s: model_reasoning_effort = %q, want %q", rel, cfg.ModelReasoningEffort, role.codexEffort))
	}
	if role.readOnly && cfg.SandboxMode != "read-only" {
		out = append(out, fmt.Sprintf("%s: read-only role must set sandbox_mode = \"read-only\" (got %q)", rel, cfg.SandboxMode))
	}
	if !role.readOnly && md.IsDefined("sandbox_mode") {
		out = append(out, fmt.Sprintf("%s: write role must not set sandbox_mode (got %q)", rel, cfg.SandboxMode))
	}
	return out
}

func checkAdapterCommon(rel, roleName, text string) []string {
	var out []string
	wantContract := ".agents/contracts/" + roleName + ".md"
	matches := contractPathPattern.FindAllString(text, -1)
	if len(matches) == 0 || !slices.Equal(matches, []string{wantContract}) {
		out = append(out, fmt.Sprintf("%s: contract references %v, want exactly [%s]", rel, matches, wantContract))
	}
	wantReadFirst := fmt.Sprintf("Your first action is to read `%s` fully", wantContract)
	if !strings.Contains(text, wantReadFirst) {
		out = append(out, fmt.Sprintf("%s: missing %q", rel, wantReadFirst))
	}
	lines := strings.Count(strings.TrimRight(text, "\n"), "\n") + 1
	if lines > maxRoleAdapterLines {
		out = append(out, fmt.Sprintf("%s: %d lines exceeds max %d lines", rel, lines, maxRoleAdapterLines))
	}
	if strings.Contains(text, "- [ ]") || strings.Contains(text, "- [x]") {
		out = append(out, fmt.Sprintf("%s: contains copied checklist checkbox", rel))
	}
	return out
}

type ruleSectionCount struct {
	title string
	rows  int
}

func checkRulesAndSizeParity(root string) []string {
	var out []string
	claudeData, err := os.ReadFile(filepath.Join(root, "CLAUDE.md"))
	if err == nil {
		claudeText := string(claudeData)
		if !strings.Contains(claudeText, "@AGENTS.md") {
			out = append(out, "CLAUDE.md: missing @AGENTS.md import")
		}
		lines := strings.Count(strings.TrimRight(claudeText, "\n"), "\n") + 1
		if lines > maxClaudeMDLines {
			out = append(out, fmt.Sprintf("CLAUDE.md: %d lines exceeds max %d lines", lines, maxClaudeMDLines))
		}
		if strings.Contains(claudeText, "\n## ") || strings.Contains(claudeText, "\n|") {
			out = append(out, "CLAUDE.md: must not carry its own section heading or rule table")
		}
	}

	agentsData, errA := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	indexData, errI := os.ReadFile(filepath.Join(root, ".agents", "rules-index.md"))
	if errA != nil || errI != nil {
		return out
	}
	if len(agentsData) >= maxAgentsMDBytes {
		out = append(out, fmt.Sprintf("AGENTS.md: %d bytes is not strictly under %d bytes", len(agentsData), maxAgentsMDBytes))
	}
	agentsCounts := parseRuleSectionCounts(string(agentsData), "### ")
	indexCounts := parseRuleSectionCounts(string(indexData), "## ")
	if len(agentsCounts) != len(expectedRuleDomains) || len(indexCounts) != len(expectedRuleDomains) {
		return append(out, fmt.Sprintf("AGENTS.md (%d sections) and .agents/rules-index.md (%d sections) must both have %d numbered rule sections",
			len(agentsCounts), len(indexCounts), len(expectedRuleDomains)))
	}
	for i, wantTitle := range expectedRuleDomains {
		a, b := agentsCounts[i], indexCounts[i]
		if a.title != wantTitle || b.title != wantTitle {
			out = append(out, fmt.Sprintf("rule section %d: AGENTS.md = %q, .agents/rules-index.md = %q, want %q", i+1, a.title, b.title, wantTitle))
			continue
		}
		if a.rows == 0 || a.rows != b.rows {
			out = append(out, fmt.Sprintf("AGENTS.md and .agents/rules-index.md disagree on row count in %q: AGENTS.md has %d, .agents/rules-index.md has %d",
				wantTitle, a.rows, b.rows))
		}
	}
	return out
}

func parseRuleSectionCounts(text, prefix string) []ruleSectionCount {
	var sections []ruleSectionCount
	cur := -1
	for _, line := range strings.Split(text, "\n") {
		if title, ok := strings.CutPrefix(line, prefix); ok && len(title) >= 3 && title[0] >= '1' && title[0] <= '9' && title[1] == '.' {
			sections = append(sections, ruleSectionCount{title: strings.TrimSpace(title)})
			cur = len(sections) - 1
			continue
		}
		if cur >= 0 && strings.HasPrefix(line, "## ") && prefix != "## " {
			cur = -1
			continue
		}
		if cur < 0 || !strings.HasPrefix(line, "| ") {
			continue
		}
		if strings.HasPrefix(line, "| ---") || strings.HasPrefix(line, "| Gate |") || strings.HasPrefix(line, "| Rule |") {
			continue
		}
		sections[cur].rows++
	}
	return sections
}

func discoverSkills(root string) []struct{ canonical, claude string } {
	skills := make([]struct{ canonical, claude string }, 0, len(sharedSkills))
	seen := make(map[string]bool, len(sharedSkills))
	for _, s := range sharedSkills {
		skills = append(skills, s)
		seen[s.canonical] = true
	}
	entries, err := os.ReadDir(filepath.Join(root, ".agents", "skills"))
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() || seen[entry.Name()] {
				continue
			}
			claudeName := entry.Name()
			if claudeName == "semvocab-preflight" {
				claudeName = "preflight"
			}
			skills = append(skills, struct{ canonical, claude string }{entry.Name(), claudeName})
			seen[entry.Name()] = true
		}
	}
	return skills
}

func checkSkillAdapters(root string) []string {
	var out []string
	for _, s := range discoverSkills(root) {
		canonRel := ".agents/skills/" + s.canonical + "/SKILL.md"
		if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(canonRel))); err != nil || info.IsDir() || info.Size() == 0 {
			out = append(out, fmt.Sprintf("%s: missing canonical skill", canonRel))
		}
		adapterRel := ".claude/skills/" + s.claude + "/SKILL.md"
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(adapterRel)))
		if err != nil {
			out = append(out, fmt.Sprintf("%s: missing Claude skill adapter", adapterRel))
			continue
		}
		text := string(data)
		wantLink := "../../../.agents/skills/" + s.canonical + "/SKILL.md"
		if !strings.Contains(text, wantLink) || !strings.Contains(text, "fully") {
			out = append(out, fmt.Sprintf("%s: must link to %s and instruct to read it fully", adapterRel, wantLink))
		}
		lines := strings.Count(strings.TrimRight(text, "\n"), "\n") + 1
		if lines > maxSkillLines {
			out = append(out, fmt.Sprintf("%s: %d lines exceeds max %d lines", adapterRel, lines, maxSkillLines))
		}
		if strings.Contains(text, "- [ ]") || strings.Contains(text, "- [x]") {
			out = append(out, fmt.Sprintf("%s: contains copied checklist checkbox", adapterRel))
		}
	}
	return out
}

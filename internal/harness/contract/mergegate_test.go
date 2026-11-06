package contract

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const (
	requiredAteamDoctorCmd = `env -u ATEAM_SESSION_ID -u CLAUDE_SESSION_ID -u CODEX_SESSION_ID -u GEMINI_SESSION_ID ateam doctor --check`
	requiredUnitCmd        = `scripts/gopkgs.sh go test -race -count=1 -cpu 1 ./...`
	requiredRepeatCmd      = `scripts/gopkgs.sh go test -count=5 -cpu 1 -shuffle=on {{.CLI_ARGS | default "./..."}}`
	requiredTestSpecCmd    = `ateam test-spec --base "${TEST_SPEC_BASE:-origin/main}"`
	pinnedRunnerImage      = "ubuntu-24.04"
)

var (
	expectedVerifySteps = []string{
		"ateam:doctor", "spec:check", "test:spec", "docs:check", "fmt:check",
		"tidy:check", "build", "vet", "lint", "vuln", "test:unit", "test:repeat",
	}
	pinnedTestTasks = []struct {
		task string
		cmds []string
	}{
		{"ateam:doctor", []string{requiredAteamDoctorCmd}},
		{"test:unit", []string{requiredUnitCmd}},
		{"test:repeat", []string{requiredRepeatCmd}},
		{"test:spec", []string{requiredTestSpecCmd}},
	}
	verifyStepsPattern     = regexp.MustCompile(`(?s)\bsteps=\(([^)]*)\)`)
	mergeCheckPermissions  = map[string]string{"contents": "read", "issues": "read", "pull-requests": "read"}
	requiredNeeds          = []string{"verify"}
	ghExpression           = regexp.MustCompile(`\$\{\{\s*(.*?)\s*\}\}`)
	joinNeedsResultsRegexp = regexp.MustCompile(`^join\(\s*needs\.\*\.result\s*(?:,\s*'([^']*)'\s*)?\)$`)
	oneNeedResultRegexp    = regexp.MustCompile(`^needs\.([A-Za-z0-9_-]+)\.result$`)
)

// spec: openspec/specs/merge-gate/spec.md#Requirement: One verification entrypoint
// spec: openspec/specs/agent-setup/spec.md#Requirement: Ateam bootstrap and doctor parity
func TestUnitInvocationsPinned(t *testing.T) {
	root := repoRoot(t)
	taskfile, err := os.ReadFile(filepath.Join(root, "Taskfile.yml"))
	if err != nil {
		t.Fatal(err)
	}
	verify, err := os.ReadFile(filepath.Join(root, "scripts", "verify.sh"))
	if err != nil {
		t.Fatal(err)
	}
	requireNoViolations(t, "test and verify pins", unitInvocationViolations(taskfile, verify))
}

// spec: openspec/specs/merge-gate/spec.md#Requirement: One verification entrypoint
func TestUnitInvocationsPinnedSensitivity(t *testing.T) {
	buildTaskfile := func(overrides map[string]string) []byte {
		base := map[string]string{
			"ateam:doctor": requiredAteamDoctorCmd,
			"test:unit":    requiredUnitCmd,
			"test:repeat":  requiredRepeatCmd,
			"test:spec":    requiredTestSpecCmd,
		}
		for k, v := range overrides {
			if v == "" {
				delete(base, k)
			} else {
				base[k] = v
			}
		}
		var sb strings.Builder
		sb.WriteString("version: '3'\ntasks:\n")
		for _, k := range slices.Sorted(maps.Keys(base)) {
			sb.WriteString("  " + k + ":\n    cmds:\n      - '" + base[k] + "'\n")
		}
		return []byte(sb.String())
	}
	buildVerify := func(steps []string) []byte {
		return []byte("#!/usr/bin/env bash\nsteps=(" + strings.Join(steps, " ") + ")\n")
	}
	goodTF := buildTaskfile(nil)
	goodVF := buildVerify(expectedVerifySteps)
	requireNoViolations(t, "clean fixture", unitInvocationViolations(goodTF, goodVF))

	for _, tc := range []struct {
		name             string
		taskfile, verify []byte
		wants            []string
	}{
		{"repeat count lowered", buildTaskfile(map[string]string{"test:repeat": strings.Replace(requiredRepeatCmd, "-count=5", "-count=1", 1)}), goodVF,
			[]string{"test:repeat", requiredRepeatCmd}},
		{"unit cpu setting dropped", buildTaskfile(map[string]string{"test:unit": strings.Replace(requiredUnitCmd, " -cpu 1", "", 1)}), goodVF,
			[]string{"test:unit", requiredUnitCmd}},
		{"ateam:doctor drops session unset", buildTaskfile(map[string]string{"ateam:doctor": "ateam doctor --check"}), goodVF,
			[]string{"ateam:doctor", requiredAteamDoctorCmd}},
		{"ateam:doctor missing", buildTaskfile(map[string]string{"ateam:doctor": ""}), goodVF,
			[]string{"ateam:doctor", "no such task"}},
		{"verify step order drifted", goodTF, buildVerify([]string{"spec:check", "ateam:doctor", "test:repeat"}),
			[]string{"scripts/verify.sh", "steps"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requireViolation(t, unitInvocationViolations(tc.taskfile, tc.verify), tc.wants...)
		})
	}
}

func unitInvocationViolations(taskfile, verify []byte) []string {
	var violations []string
	var tf struct {
		Tasks map[string]struct {
			Cmds []string `yaml:"cmds"`
		} `yaml:"tasks"`
	}
	if err := yaml.Unmarshal(taskfile, &tf); err != nil {
		return []string{fmt.Sprintf("Taskfile.yml: %v", err)}
	}
	for _, want := range pinnedTestTasks {
		got, ok := tf.Tasks[want.task]
		if !ok {
			violations = append(violations, fmt.Sprintf("Taskfile.yml: no such task %s", want.task))
			continue
		}
		if !slices.Equal(got.Cmds, want.cmds) {
			violations = append(violations, fmt.Sprintf("Taskfile.yml: task %s runs %v, want %v", want.task, got.Cmds, want.cmds))
		}
	}
	m := verifyStepsPattern.FindSubmatch(verify)
	if m == nil {
		return append(violations, "scripts/verify.sh: no steps=(...) list")
	}
	steps := strings.Fields(string(m[1]))
	if !slices.Equal(steps, expectedVerifySteps) {
		violations = append(violations, fmt.Sprintf("scripts/verify.sh: steps = %v, want %v", steps, expectedVerifySteps))
	}
	return violations
}

type ciWorkflow struct {
	On          yaml.Node         `yaml:"on"`
	Concurrency map[string]any    `yaml:"concurrency"`
	Permissions map[string]string `yaml:"permissions"`
	Jobs        map[string]ciJob  `yaml:"jobs"`
}

type ciJob struct {
	Name            string            `yaml:"name"`
	Needs           any               `yaml:"needs"`
	If              any               `yaml:"if"`
	RunsOn          string            `yaml:"runs-on"`
	ContinueOnError any               `yaml:"continue-on-error"`
	Permissions     map[string]string `yaml:"permissions"`
	Outputs         map[string]string `yaml:"outputs"`
	Env             map[string]any    `yaml:"env"`
	Steps           []ciStep          `yaml:"steps"`
}

type ciStep struct {
	Name            string         `yaml:"name"`
	ID              string         `yaml:"id"`
	If              any            `yaml:"if"`
	Uses            string         `yaml:"uses"`
	With            map[string]any `yaml:"with"`
	Env             map[string]any `yaml:"env"`
	Run             string         `yaml:"run"`
	Shell           string         `yaml:"shell"`
	ContinueOnError any            `yaml:"continue-on-error"`
}

// spec: openspec/specs/merge-gate/spec.md#Requirement: Required CI and Merge check workflows
func TestCIWorkflowPinned(t *testing.T) {
	root := repoRoot(t)
	ci, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	mc, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "merge-check.yml"))
	if err != nil {
		t.Fatal(err)
	}
	requireNoViolations(t, "CI and Merge check workflows", workflowViolations(t, ci, mc))
}

// spec: openspec/specs/merge-gate/spec.md#Requirement: Required CI and Merge check workflows
func TestCIWorkflowPinnedSensitivity(t *testing.T) {
	root := repoRoot(t)
	ciBytes, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	mcBytes, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "merge-check.yml"))
	if err != nil {
		t.Fatal(err)
	}
	ci, mc := string(ciBytes), string(mcBytes)
	requireNoViolations(t, "clean workflows", workflowViolations(t, ciBytes, mcBytes))

	for _, tc := range []struct {
		name   string
		ci, mc string
		wants  []string
	}{
		{"ci runner changed", strings.Replace(ci, "runs-on: ubuntu-24.04", "runs-on: ubuntu-latest", 1), mc,
			[]string{"ci.yml", "ubuntu-24.04"}},
		{"ci fetch-depth removed", strings.Replace(ci, "fetch-depth: 0", "fetch-depth: 1", 1), mc,
			[]string{"ci.yml", "fetch-depth: 0"}},
		{"ateam version mismatch across workflows", ci, strings.Replace(mc, "ateam@v0.3.8", "ateam@v0.3.7", 1),
			[]string{"ateam install lines"}},
		{"required drops if: always()", strings.Replace(ci, "if: always()", "if: success()", 1), mc,
			[]string{"ci.yml", "job required", "always()"}},
		{"required step exits 0 on failure", strings.Replace(ci, `if [ "$r" != "success" ]; then echo "required check did not succeed: $r"; exit 1; fi`, `echo "$r"`, 1), mc,
			[]string{"ci.yml", "job required", "exits 0"}},
		{"merge-check drops base.sha checkout", ci, strings.ReplaceAll(mc, "github.event.pull_request.base.sha", "github.sha"),
			[]string{"merge-check.yml", "base.sha"}},
		{"merge-check cancels in-progress", ci, strings.Replace(mc, "cancel-in-progress: false", "cancel-in-progress: true", 1),
			[]string{"merge-check.yml", "cancel-in-progress: false"}},
		{"merge-check drops ATEAM_REQUIRED_CHECKS", ci, strings.Replace(mc, "ATEAM_REQUIRED_CHECKS: Required,Merge check", "ATEAM_REQUIRED_CHECKS: Required", 1),
			[]string{"merge-check.yml", "ATEAM_REQUIRED_CHECKS"}},
		{"merge-check probe missing --awaiting", ci, strings.Replace(mc, "ateam review gate --awaiting", "ateam review gate", 1),
			[]string{"merge-check.yml", "awaiting", "--awaiting"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requireViolation(t, workflowViolations(t, []byte(tc.ci), []byte(tc.mc)), tc.wants...)
		})
	}
}

func workflowViolations(t *testing.T, ciData, mcData []byte) []string {
	t.Helper()
	_, pinViolations := ateamPins(map[string][]byte{"ci.yml": ciData, "merge-check.yml": mcData})
	out := append([]string{}, pinViolations...)
	out = append(out, checkCIWorkflow(t, ciData)...)
	out = append(out, checkMergeCheckWorkflow(mcData)...)
	return out
}

func checkCIWorkflow(t *testing.T, data []byte) []string {
	t.Helper()
	var wf ciWorkflow
	if err := yaml.Unmarshal(data, &wf); err != nil {
		return []string{"ci.yml: parse: " + err.Error()}
	}
	var out []string
	for name, job := range wf.Jobs {
		if job.RunsOn != pinnedRunnerImage {
			out = append(out, fmt.Sprintf("ci.yml: job %s runs-on %q, want %q", name, job.RunsOn, pinnedRunnerImage))
		}
	}
	if _, hasMC := wf.Jobs["merge-check"]; hasMC {
		out = append(out, "ci.yml: must not contain merge-check job; Merge check lives in merge-check.yml")
	}
	verify, ok := wf.Jobs["verify"]
	if !ok {
		return append(out, "ci.yml: missing verify job")
	}
	out = append(out, checkVerifyJobSteps(verify)...)

	req, ok := wf.Jobs["required"]
	if !ok {
		return append(out, "ci.yml: missing required job")
	}
	if req.Name != "Required" {
		out = append(out, fmt.Sprintf("ci.yml: job required name = %q, want \"Required\"", req.Name))
	}
	needs := parseJobNeeds(req.Needs)
	if !slices.Equal(needs, requiredNeeds) {
		out = append(out, fmt.Sprintf("ci.yml: job required needs = %v, want %v", needs, requiredNeeds))
	}
	out = append(out, requiredStepViolations(t, req, needs)...)
	return out
}

func checkVerifyJobSteps(verify ciJob) []string {
	var out []string
	hasFetchDepth0, hasDoctor, hasVerifyWithBase := false, false, false
	for _, st := range verify.Steps {
		if strings.HasPrefix(st.Uses, "actions/checkout@") && fmt.Sprint(st.With["fetch-depth"]) == "0" {
			hasFetchDepth0 = true
		}
		if strings.TrimSpace(st.Run) == "task doctor" {
			hasDoctor = true
		}
		if strings.TrimSpace(st.Run) == "task verify" && strings.Contains(fmt.Sprint(st.Env["TEST_SPEC_BASE"]), "pull_request.base.sha") {
			hasVerifyWithBase = true
		}
	}
	if !hasFetchDepth0 {
		out = append(out, "ci.yml: verify checkout must set fetch-depth: 0")
	}
	if !hasDoctor {
		out = append(out, "ci.yml: verify must run task doctor")
	}
	if !hasVerifyWithBase {
		out = append(out, "ci.yml: verify must run task verify with TEST_SPEC_BASE referencing pull_request.base.sha")
	}
	return out
}

func checkMergeCheckWorkflow(data []byte) []string {
	var wf ciWorkflow
	if err := yaml.Unmarshal(data, &wf); err != nil {
		return []string{"merge-check.yml: parse: " + err.Error()}
	}
	var out []string
	if fmt.Sprint(wf.Concurrency["cancel-in-progress"]) != "false" {
		out = append(out, fmt.Sprintf("merge-check.yml: concurrency cancel-in-progress = %v, want cancel-in-progress: false", wf.Concurrency["cancel-in-progress"]))
	}
	for name, job := range wf.Jobs {
		if job.RunsOn != pinnedRunnerImage {
			out = append(out, fmt.Sprintf("merge-check.yml: job %s runs-on %q, want %q", name, job.RunsOn, pinnedRunnerImage))
		}
		if !maps.Equal(job.Permissions, mergeCheckPermissions) {
			out = append(out, fmt.Sprintf("merge-check.yml: job %s permissions = %v, want %v", name, job.Permissions, mergeCheckPermissions))
		}
		out = append(out, checkBaseShaCheckout("merge-check.yml", name, job.Steps)...)
	}
	awaiting, ok := wf.Jobs["awaiting"]
	if !ok {
		out = append(out, "merge-check.yml: missing awaiting probe job")
	} else {
		out = append(out, checkAwaitingJob(awaiting)...)
	}
	mc, ok := wf.Jobs["merge-check"]
	if !ok {
		out = append(out, "merge-check.yml: missing merge-check job")
	} else {
		out = append(out, checkMergeCheckJob(mc)...)
	}
	return out
}

func checkBaseShaCheckout(file, jobName string, steps []ciStep) []string {
	for _, st := range steps {
		if strings.HasPrefix(st.Uses, "actions/checkout@") {
			ref := fmt.Sprint(st.With["ref"])
			persist := fmt.Sprint(st.With["persist-credentials"])
			var out []string
			if !strings.Contains(ref, "github.event.pull_request.base.sha") {
				out = append(out, fmt.Sprintf("%s: job %s checkout ref = %q, want github.event.pull_request.base.sha", file, jobName, ref))
			}
			if persist != "false" {
				out = append(out, fmt.Sprintf("%s: job %s checkout must set persist-credentials: false", file, jobName))
			}
			return out
		}
	}
	return []string{fmt.Sprintf("%s: job %s missing actions/checkout step", file, jobName)}
}

func checkAwaitingJob(job ciJob) []string {
	var out []string
	if job.Name != "Merge check probe" {
		out = append(out, fmt.Sprintf("merge-check.yml: awaiting job name = %q, want \"Merge check probe\"", job.Name))
	}
	if !strings.Contains(fmt.Sprint(job.Outputs["awaiting"]), "steps.probe.outputs.awaiting") {
		out = append(out, "merge-check.yml: awaiting job must expose awaiting output from steps.probe.outputs.awaiting")
	}
	hasProbe := false
	for _, st := range job.Steps {
		if st.ID == "probe" && strings.Contains(st.Run, "ateam review gate --awaiting") {
			hasProbe = true
		}
	}
	if !hasProbe {
		out = append(out, "merge-check.yml: awaiting job missing probe step running ateam review gate --awaiting")
	}
	return out
}

func checkMergeCheckJob(job ciJob) []string {
	var out []string
	if !slices.Equal(parseJobNeeds(job.Needs), []string{"awaiting"}) {
		out = append(out, fmt.Sprintf("merge-check.yml: merge-check needs = %v, want [awaiting]", parseJobNeeds(job.Needs)))
	}
	cond := fmt.Sprint(job.If)
	if !strings.Contains(cond, "always()") || !strings.Contains(cond, "!github.event.pull_request.draft") || !strings.Contains(cond, "needs.awaiting.outputs.awaiting != 'true'") {
		out = append(out, fmt.Sprintf("merge-check.yml: merge-check if = %q must guard draft and awaiting != 'true' under always()", cond))
	}
	if !strings.Contains(job.Name, "Merge check (draft, skipped)") || !strings.Contains(job.Name, "Merge check (awaiting review record, skipped)") || !strings.HasSuffix(strings.TrimSpace(job.Name), "'Merge check' }}") {
		out = append(out, fmt.Sprintf("merge-check.yml: merge-check name = %q must rename draft/awaiting runs and end with 'Merge check'", job.Name))
	}
	hasRun := false
	for _, st := range job.Steps {
		if strings.Contains(st.Run, `scripts/merge-check.sh "$PR_NUMBER"`) && fmt.Sprint(st.Env["ATEAM_REQUIRED_CHECKS"]) == "Required,Merge check" {
			hasRun = true
		}
	}
	if !hasRun {
		out = append(out, "merge-check.yml: merge-check job must run scripts/merge-check.sh with ATEAM_REQUIRED_CHECKS=Required,Merge check")
	}
	return out
}

func parseJobNeeds(raw any) []string {
	switch v := raw.(type) {
	case string:
		return []string{v}
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, fmt.Sprint(item))
		}
		return out
	default:
		return nil
	}
}

func evalNeedsExpression(expr string, needs []string, results map[string]string) (string, bool) {
	if m := joinNeedsResultsRegexp.FindStringSubmatch(expr); m != nil {
		sep := ","
		if strings.Contains(expr, "'") {
			sep = m[1]
		}
		var rs []string
		for _, j := range needs {
			if r, ok := results[j]; ok {
				rs = append(rs, r)
			}
		}
		return strings.Join(rs, sep), true
	}
	if m := oneNeedResultRegexp.FindStringSubmatch(expr); m != nil {
		return results[m[1]], true
	}
	return "", false
}

func requiredStepViolations(t *testing.T, req ciJob, needs []string) []string {
	t.Helper()
	var out []string
	cond := strings.TrimSpace(fmt.Sprint(req.If))
	if req.If == nil || (cond != "always()" && cond != "${{ always() }}") {
		out = append(out, fmt.Sprintf("ci.yml: job required runs under if: %v, want always()", req.If))
	}
	if req.ContinueOnError != nil {
		out = append(out, fmt.Sprintf("ci.yml: job required sets continue-on-error: %v", req.ContinueOnError))
	}
	if len(req.Steps) == 0 {
		return append(out, "ci.yml: job required has no steps")
	}
	readsAll := false
	for _, st := range req.Steps {
		if st.If != nil || st.ContinueOnError != nil || st.Uses != "" || st.Run == "" {
			return append(out, "ci.yml: job required step must be an unconditional run step without continue-on-error")
		}
		for _, text := range []string{st.Run, fmt.Sprint(st.Env)} {
			for _, m := range ghExpression.FindAllStringSubmatch(text, -1) {
				if joinNeedsResultsRegexp.MatchString(m[1]) {
					readsAll = true
				}
				if _, ok := evalNeedsExpression(m[1], needs, nil); !ok {
					return append(out, fmt.Sprintf("ci.yml: job required unsupported expression ${{ %s }}", m[1]))
				}
			}
		}
	}
	if !readsAll {
		return append(out, "ci.yml: job required must read join(needs.*.result, ...)")
	}
	for _, results := range []map[string]string{{"verify": "success"}, {"verify": "failure"}, {"verify": "cancelled"}, {"verify": "skipped"}, nil} {
		subst := func(text string) string {
			return ghExpression.ReplaceAllStringFunc(text, func(e string) string {
				v, _ := evalNeedsExpression(ghExpression.FindStringSubmatch(e)[1], needs, results)
				return v
			})
		}
		ok, err := runRequiredSteps(t, req, subst)
		if err != nil {
			return append(out, fmt.Sprintf("ci.yml: job required execution error: %v", err))
		}
		wantOK := results != nil && results["verify"] == "success"
		if wantOK && !ok {
			out = append(out, "ci.yml: job required exits non-zero when verify succeeded")
		}
		if !wantOK && ok {
			out = append(out, fmt.Sprintf("ci.yml: job required exits 0 for %v", results))
		}
	}
	return out
}

func runRequiredSteps(t *testing.T, req ciJob, subst func(string) string) (bool, error) {
	t.Helper()
	for _, st := range req.Steps {
		env := []string{"PATH=" + os.Getenv("PATH")}
		for _, m := range []map[string]any{req.Env, st.Env} {
			for _, k := range slices.Sorted(maps.Keys(m)) {
				env = append(env, k+"="+subst(fmt.Sprint(m[k])))
			}
		}
		cmd := exec.Command("bash", "--noprofile", "--norc", "-e", "-c", subst(st.Run))
		cmd.Env = env
		cmd.Dir = t.TempDir()
		_, err := cmd.CombinedOutput()
		var exit *exec.ExitError
		switch {
		case errors.As(err, &exit):
			return false, nil
		case err != nil:
			return false, err
		}
	}
	return true, nil
}

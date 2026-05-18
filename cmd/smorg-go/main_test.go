package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/astmergegit"
)

func writeTestFile(t *testing.T, dir string, name string, source string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	return path
}

func TestMergeDriverGitPlaceholderFormUpdatesCurrentFile(t *testing.T) {
	dir := t.TempDir()
	ancestor := writeTestFile(t, dir, "ancestor.json", `{"name":"structuredmerge"}`)
	current := writeTestFile(t, dir, "current.tmp", `{"name":"structuredmerge","current":true}`)
	other := writeTestFile(t, dir, "other.tmp", `{"name":"structuredmerge","other":true}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"merge-driver", "--path-name", "package.json", ancestor, current, other}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("unexpected exit code %d stderr=%s", exitCode, stderr.String())
	}

	merged, err := os.ReadFile(current)
	if err != nil {
		t.Fatalf("read merged current file: %v", err)
	}
	mergedSource := string(merged)
	if !strings.Contains(mergedSource, `"current":true`) || !strings.Contains(mergedSource, `"other":true`) {
		t.Fatalf("current file was not updated with merged JSON: %s", mergedSource)
	}
	if stdout.Len() != 0 {
		t.Fatalf("merge-driver should keep stdout quiet in git mode, got %q", stdout.String())
	}
}

func TestMergeDriverNamedFormWritesOutputPath(t *testing.T) {
	dir := t.TempDir()
	ancestor := writeTestFile(t, dir, "ancestor.go", "package main\n")
	current := writeTestFile(t, dir, "current.go", "package main\n\nfunc Current() {}\n")
	other := writeTestFile(t, dir, "other.go", "package main\n\nfunc Other() {}\n")
	output := filepath.Join(dir, "merged.go")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{
		"merge-driver",
		"--ancestor", ancestor,
		"--current", current,
		"--other", other,
		"--path-name", "main.go",
		"--output", output,
	}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("unexpected exit code %d stderr=%s", exitCode, stderr.String())
	}

	merged, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read merged output file: %v", err)
	}
	mergedSource := string(merged)
	if !strings.Contains(mergedSource, "func Current") || !strings.Contains(mergedSource, "func Other") {
		t.Fatalf("output file was not updated with merged Go: %s", mergedSource)
	}
}

func TestMergeDriverJSONUsesAncestorForSameKeyConflicts(t *testing.T) {
	dir := t.TempDir()
	ancestor := writeTestFile(t, dir, "ancestor.json", `{"name":"demo","enabled":true}`)
	current := writeTestFile(t, dir, "current.json", `{"name":"demo","enabled":false}`)
	other := writeTestFile(t, dir, "other.json", `{"name":"demo","enabled":"yes"}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"merge-driver", "--strict", ancestor, current, other, "package.json"}, &stdout, &stderr)
	if exitCode != exitUnresolvedConflict {
		t.Fatalf("expected conflict exit code, got %d stderr=%s", exitCode, stderr.String())
	}

	currentSource, err := os.ReadFile(current)
	if err != nil {
		t.Fatalf("read current file: %v", err)
	}
	if !strings.Contains(string(currentSource), "<<<<<<< ours") ||
		!strings.Contains(string(currentSource), "||||||| base") ||
		!strings.Contains(string(currentSource), "=======") ||
		!strings.Contains(string(currentSource), ">>>>>>> theirs") {
		t.Fatalf("conflicted merge should write conflict markers: %s", string(currentSource))
	}
	if !strings.Contains(stderr.String(), "merge_conflict") {
		t.Fatalf("expected merge conflict diagnostic, got %q", stderr.String())
	}
}

func TestMergeDriverConflictOutputUsesMarkerSizeAttribute(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(filepath.Join(dir, ".gitattributes"), []byte("*.json conflict-marker-size=9\n"), 0o644); err != nil {
		t.Fatalf("write gitattributes: %v", err)
	}
	ancestor := writeTestFile(t, dir, "ancestor.json", `{"name":"demo","enabled":true}`)
	current := writeTestFile(t, dir, "package.json", `{"name":"demo","enabled":false}`)
	other := writeTestFile(t, dir, "other.json", `{"name":"demo","enabled":"yes"}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"merge-driver", "--strict", ancestor, current, other, "package.json"}, &stdout, &stderr)
	if exitCode != exitUnresolvedConflict {
		t.Fatalf("expected conflict exit code, got %d stderr=%s", exitCode, stderr.String())
	}

	currentSource, err := os.ReadFile(current)
	if err != nil {
		t.Fatalf("read current file: %v", err)
	}
	if !strings.Contains(string(currentSource), "<<<<<<<<< ours") ||
		!strings.Contains(string(currentSource), "||||||||| base") ||
		!strings.Contains(string(currentSource), "=========") ||
		!strings.Contains(string(currentSource), ">>>>>>>>> theirs") {
		t.Fatalf("conflicted merge should use configured conflict marker size: %s", string(currentSource))
	}
}

func TestMergeDriverJSONGitRepositoryIntegrationFixture(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable is required for repository integration fixture")
	}
	fixture := readGitDriverJSONFixture(t)
	for _, testCase := range fixture.Cases {
		t.Run(testCase.CaseID, func(t *testing.T) {
			dir := t.TempDir()
			runGit(t, dir, "init")
			runGit(t, dir, "config", "user.email", "smorg-go@example.invalid")
			runGit(t, dir, "config", "user.name", "smorg-go test")
			writeTestFile(t, dir, ".gitattributes", "*.json merge=smorg-go smorg.language=json\n")
			writeTestFile(t, dir, testCase.PathName, testCase.BaseSource)
			runGit(t, dir, "add", ".")
			runGit(t, dir, "commit", "-m", "base")

			ancestor := writeTestFile(t, dir, "ancestor.tmp", testCase.BaseSource)
			current := writeTestFile(t, dir, testCase.PathName, testCase.OursSource)
			other := writeTestFile(t, dir, "other.tmp", testCase.TheirsSource)

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			exitCode := run(
				[]string{"merge-driver", "--strict", ancestor, current, other, testCase.PathName},
				&stdout,
				&stderr,
			)
			if exitCode != testCase.Expected.ExitCode {
				t.Fatalf("expected exit %d, got %d stderr=%s", testCase.Expected.ExitCode, exitCode, stderr.String())
			}
			for _, expected := range testCase.Expected.StderrContains {
				if !strings.Contains(stderr.String(), expected) {
					t.Fatalf("expected stderr to contain %q, got %q", expected, stderr.String())
				}
			}

			mergedSource, err := os.ReadFile(current)
			if err != nil {
				t.Fatalf("read current file: %v", err)
			}
			if testCase.Expected.MergedJSON != nil {
				var merged any
				if err := json.Unmarshal(mergedSource, &merged); err != nil {
					t.Fatalf("merged output should parse as JSON: %v source=%s", err, string(mergedSource))
				}
				if !jsonEqual(merged, testCase.Expected.MergedJSON) {
					t.Fatalf("merged JSON mismatch\nexpected=%v\nactual=%v", testCase.Expected.MergedJSON, merged)
				}
			}
			if testCase.Expected.MergedSource != "" && string(mergedSource) != testCase.Expected.MergedSource {
				t.Fatalf("merged source mismatch\nexpected=%q\nactual=%q", testCase.Expected.MergedSource, string(mergedSource))
			}
			for _, expected := range testCase.Expected.ConflictedSourceContains {
				if !strings.Contains(string(mergedSource), expected) {
					t.Fatalf("expected conflicted source to contain %q:\n%s", expected, string(mergedSource))
				}
			}
		})
	}
}

func TestMergeDriverGoGitRepositoryIntegrationFixture(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable is required for repository integration fixture")
	}
	fixture := readGoMerge3Fixture(t)
	for _, testCase := range fixture.Cases {
		t.Run(testCase.CaseID, func(t *testing.T) {
			dir := t.TempDir()
			runGit(t, dir, "init")
			runGit(t, dir, "config", "user.email", "smorg-go@example.invalid")
			runGit(t, dir, "config", "user.name", "smorg-go test")
			writeTestFile(t, dir, ".gitattributes", "*.go merge=smorg-go smorg.language=go\n")
			writeTestFile(t, dir, testCase.PathName, testCase.BaseSource)
			runGit(t, dir, "add", ".")
			runGit(t, dir, "commit", "-m", "base")

			ancestor := writeTestFile(t, dir, "ancestor.tmp", testCase.BaseSource)
			current := writeTestFile(t, dir, testCase.PathName, testCase.OursSource)
			other := writeTestFile(t, dir, "other.tmp", testCase.TheirsSource)

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			exitCode := run(
				[]string{"merge-driver", "--strict", ancestor, current, other, testCase.PathName},
				&stdout,
				&stderr,
			)
			if testCase.Expected.OK && exitCode != exitSuccess {
				t.Fatalf("expected success, got %d stderr=%s", exitCode, stderr.String())
			}
			if !testCase.Expected.OK && exitCode != exitUnresolvedConflict {
				t.Fatalf("expected conflict, got %d stderr=%s", exitCode, stderr.String())
			}
			mergedSource, err := os.ReadFile(current)
			if err != nil {
				t.Fatalf("read current file: %v", err)
			}
			if testCase.Expected.OK {
				if string(mergedSource) != testCase.Expected.ExpectedSource {
					t.Fatalf("merged source mismatch\nexpected:\n%s\nactual:\n%s", testCase.Expected.ExpectedSource, string(mergedSource))
				}
			} else {
				if !strings.Contains(stderr.String(), "merge_conflict") {
					t.Fatalf("expected merge conflict diagnostic, got %q", stderr.String())
				}
				for _, expected := range []string{"<<<<<<< ours", "||||||| base", "=======", ">>>>>>> theirs"} {
					if !strings.Contains(string(mergedSource), expected) {
						t.Fatalf("expected conflicted source to contain %q:\n%s", expected, string(mergedSource))
					}
				}
			}
		})
	}
}

func TestMergeDriverStrictFailureReturnsConflictExitCode(t *testing.T) {
	dir := t.TempDir()
	ancestor := writeTestFile(t, dir, "ancestor.json", `{"name":"structuredmerge"}`)
	current := writeTestFile(t, dir, "current.json", `{"name":`)
	other := writeTestFile(t, dir, "other.json", `{"other":true}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"merge-driver", "--strict", ancestor, current, other, "package.json"}, &stdout, &stderr)
	if exitCode != exitUnresolvedConflict {
		t.Fatalf("unexpected exit code %d stderr=%s", exitCode, stderr.String())
	}
	if !strings.Contains(stderr.String(), "parse_error") || !strings.Contains(stderr.String(), "ours parse error") {
		t.Fatalf("expected ours parse diagnostic, got %q", stderr.String())
	}
}

func TestMergeDriverFullFileFallbackWritesConflictMarkers(t *testing.T) {
	dir := t.TempDir()
	ancestor := writeTestFile(t, dir, "ancestor.json", `{"name":"structuredmerge"}`)
	current := writeTestFile(t, dir, "current.json", `{"name":`)
	other := writeTestFile(t, dir, "other.json", `{"other":true}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"merge-driver", ancestor, current, other, "package.json"}, &stdout, &stderr)
	if exitCode != exitUnresolvedConflict {
		t.Fatalf("unexpected exit code %d stderr=%s", exitCode, stderr.String())
	}

	currentSource, err := os.ReadFile(current)
	if err != nil {
		t.Fatalf("read current file: %v", err)
	}
	for _, expected := range []string{"<<<<<<< ours", "||||||| base", "=======", ">>>>>>> theirs"} {
		if !strings.Contains(string(currentSource), expected) {
			t.Fatalf("expected full-file fallback marker %q:\n%s", expected, string(currentSource))
		}
	}
	if !strings.Contains(stderr.String(), "parse_error") {
		t.Fatalf("expected parse diagnostic, got %q", stderr.String())
	}
}

func TestMergeDriverReportIncludesOwnedRegions(t *testing.T) {
	dir := t.TempDir()
	ancestor := writeTestFile(t, dir, "ancestor.go", "package main\n\nfunc value() string {\n\treturn \"base\"\n}\n\nfunc stable() {}\n")
	current := writeTestFile(t, dir, "current.go", "package main\n\nfunc value() string {\n\treturn \"ours\"\n}\n\nfunc stable() {}\n")
	other := writeTestFile(t, dir, "other.go", "package main\n\nfunc value() string {\n\treturn \"theirs\"\n}\n\nfunc stable() {}\n")
	reportPath := filepath.Join(dir, "merge-report.json")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"merge-driver", "--report", reportPath, ancestor, current, other, "main.go"}, &stdout, &stderr)
	if exitCode != exitUnresolvedConflict {
		t.Fatalf("unexpected exit code %d stderr=%s", exitCode, stderr.String())
	}
	source, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var report struct {
		ChangeClassifications   []astmergegit.ChangeClassification        `json:"change_classifications"`
		OwnedRegions            []astmergegit.OwnedRegionReport           `json:"owned_regions"`
		RenderReport            astmergegit.Merge3RenderReport            `json:"render_report"`
		Profile                 map[string]string                         `json:"profile"`
		FormattingPreservation  *astmergegit.FormattingPreservationReport `json:"formatting_preservation"`
		DefaultDriverEvaluation astmergegit.DefaultDriverEvaluation       `json:"default_driver_evaluation"`
	}
	if err := json.Unmarshal(source, &report); err != nil {
		t.Fatalf("parse report: %v", err)
	}
	if report.RenderReport.Strategy != "owned_region_conflict_markers" {
		t.Fatalf("unexpected render report: %+v", report.RenderReport)
	}
	expectedClassifications := []astmergegit.ChangeClassification{
		{Path: "/decls/value", Ours: "edited", Theirs: "edited"},
	}
	if !reflect.DeepEqual(report.ChangeClassifications, expectedClassifications) {
		t.Fatalf("unexpected change classifications: got=%+v expected=%+v", report.ChangeClassifications, expectedClassifications)
	}
	if len(report.OwnedRegions) != 1 || report.OwnedRegions[0].OwnerPath != "/decls/value" {
		t.Fatalf("unexpected owned regions: %+v", report.OwnedRegions)
	}
	if report.Profile["profile_id"] != "go.source" || report.Profile["language"] != "go" {
		t.Fatalf("unexpected profile: %+v", report.Profile)
	}
	if report.FormattingPreservation == nil {
		t.Fatalf("expected formatting preservation in report: %+v", report.FormattingPreservation)
	}
	if report.DefaultDriverEvaluation.Status == "" {
		t.Fatalf("expected default-driver evaluation in report: %+v", report.DefaultDriverEvaluation)
	}
}

func TestMergeDriverReportIncludesChangeClassifications(t *testing.T) {
	dir := t.TempDir()
	ancestor := writeTestFile(t, dir, "ancestor.json", `{"name":"demo"}`)
	current := writeTestFile(t, dir, "current.json", `{"name":"demo","ours":true}`)
	other := writeTestFile(t, dir, "other.json", `{"name":"demo","theirs":true}`)
	reportPath := filepath.Join(dir, "merge-report.json")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"merge-driver", "--report", reportPath, ancestor, current, other, "package.json"}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("unexpected exit code %d stderr=%s", exitCode, stderr.String())
	}
	source, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var report struct {
		ChangeClassifications []astmergegit.ChangeClassification `json:"change_classifications"`
	}
	if err := json.Unmarshal(source, &report); err != nil {
		t.Fatalf("parse report: %v", err)
	}
	expected := []astmergegit.ChangeClassification{
		{Path: "/ours", Ours: "added", Theirs: "unchanged"},
		{Path: "/theirs", Ours: "unchanged", Theirs: "added"},
	}
	if !reflect.DeepEqual(report.ChangeClassifications, expected) {
		t.Fatalf("unexpected change classifications: got=%+v expected=%+v", report.ChangeClassifications, expected)
	}
}

func TestMergeDriverScopedFallbackUsesOwnedRegion(t *testing.T) {
	for _, fallbackMode := range []string{"line", "local"} {
		t.Run(fallbackMode, func(t *testing.T) {
			dir := t.TempDir()
			ancestor := writeTestFile(t, dir, "ancestor.go", "package main\n\nfunc value() string {\n\tleft := \"base\"\n\tright := \"base\"\n\treturn left + right\n}\n\nfunc stable() {}\n")
			current := writeTestFile(t, dir, "current.go", "package main\n\nfunc value() string {\n\tleft := \"ours\"\n\tright := \"base\"\n\treturn left + right\n}\n\nfunc stable() {}\n")
			other := writeTestFile(t, dir, "other.go", "package main\n\nfunc value() string {\n\tleft := \"base\"\n\tright := \"theirs\"\n\treturn left + right\n}\n\nfunc stable() {}\n")
			reportPath := filepath.Join(dir, "merge-report.json")

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			exitCode := run([]string{"merge-driver", "--fallback", fallbackMode, "--report", reportPath, ancestor, current, other, "main.go"}, &stdout, &stderr)
			if exitCode != exitSuccess {
				t.Fatalf("unexpected exit code %d stderr=%s", exitCode, stderr.String())
			}
			currentSource, err := os.ReadFile(current)
			if err != nil {
				t.Fatalf("read current: %v", err)
			}
			for _, needle := range []string{`left := "ours"`, `right := "theirs"`, "func stable() {}"} {
				if !strings.Contains(string(currentSource), needle) {
					t.Fatalf("expected merged source to contain %q:\n%s", needle, string(currentSource))
				}
			}
			if strings.Contains(string(currentSource), "<<<<<<< ours") {
				t.Fatalf("%s fallback should not leave conflict markers:\n%s", fallbackMode, string(currentSource))
			}
			source, err := os.ReadFile(reportPath)
			if err != nil {
				t.Fatalf("read report: %v", err)
			}
			var report struct {
				Fallbacks    []mergeDriverFallback           `json:"fallbacks"`
				OwnedRegions []astmergegit.OwnedRegionReport `json:"owned_regions"`
				RenderReport astmergegit.Merge3RenderReport  `json:"render_report"`
			}
			if err := json.Unmarshal(source, &report); err != nil {
				t.Fatalf("parse report: %v", err)
			}
			if len(report.Fallbacks) != 1 || report.Fallbacks[0].Mode != fallbackMode {
				t.Fatalf("unexpected fallbacks: %+v", report.Fallbacks)
			}
			if report.RenderReport.Strategy != "local_fallback" {
				t.Fatalf("unexpected render report: %+v", report.RenderReport)
			}
			if len(report.OwnedRegions) != 1 || report.OwnedRegions[0].OwnerPath != "/decls/value" {
				t.Fatalf("unexpected owned regions: %+v", report.OwnedRegions)
			}
		})
	}
}

func TestMergeDriverFallbackFixture(t *testing.T) {
	fixture := readGitDriverFallbackFixture(t)
	for _, testCase := range fixture.Cases {
		t.Run(testCase.CaseID, func(t *testing.T) {
			dir := t.TempDir()
			ancestor := writeTestFile(t, dir, "ancestor.json", testCase.BaseSource)
			current := writeTestFile(t, dir, "current.json", testCase.OursSource)
			other := writeTestFile(t, dir, "other.json", testCase.TheirsSource)
			reportPath := filepath.Join(dir, "merge-report.json")
			args := []string{"merge-driver"}
			if testCase.Options.Strict {
				args = append(args, "--strict")
			}
			if testCase.Options.Fallback != "" && testCase.Options.Fallback != "full-file" {
				args = append(args, "--fallback", testCase.Options.Fallback)
			}
			args = append(args, "--report", reportPath)
			args = append(args, ancestor, current, other, testCase.PathName)

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			exitCode := run(args, &stdout, &stderr)
			if exitCode != testCase.Expected.ExitCode {
				t.Fatalf("expected exit %d, got %d stderr=%s", testCase.Expected.ExitCode, exitCode, stderr.String())
			}
			currentSource, err := os.ReadFile(current)
			if err != nil {
				t.Fatalf("read current file: %v", err)
			}
			if testCase.Expected.MergedSource != "" && string(currentSource) != testCase.Expected.MergedSource {
				t.Fatalf("merged source mismatch\nexpected=%q\nactual=%q", testCase.Expected.MergedSource, string(currentSource))
			}
			for _, expected := range testCase.Expected.SourceContains {
				if !strings.Contains(string(currentSource), expected) {
					t.Fatalf("expected source to contain %q:\n%s", expected, string(currentSource))
				}
			}
			for _, expected := range testCase.Expected.StderrContains {
				if !strings.Contains(stderr.String(), expected) {
					t.Fatalf("expected stderr to contain %q, got %q", expected, stderr.String())
				}
			}
			for _, unexpected := range testCase.Expected.StderrNotContains {
				if strings.Contains(stderr.String(), unexpected) {
					t.Fatalf("expected stderr not to contain %q, got %q", unexpected, stderr.String())
				}
			}
			assertFallbackMachineReport(t, reportPath, testCase.Expected.MachineReport)
		})
	}
}

type gitDriverJSONFixture struct {
	Cases []gitDriverJSONCase `json:"cases"`
}

type gitDriverJSONCase struct {
	CaseID       string                `json:"case_id"`
	PathName     string                `json:"path_name"`
	BaseSource   string                `json:"base_source"`
	OursSource   string                `json:"ours_source"`
	TheirsSource string                `json:"theirs_source"`
	Expected     gitDriverJSONExpected `json:"expected"`
}

type gitDriverJSONExpected struct {
	ExitCode                 int      `json:"exit_code"`
	MergedJSON               any      `json:"merged_json"`
	MergedSource             string   `json:"merged_source"`
	ConflictedSourceContains []string `json:"conflicted_source_contains"`
	StderrContains           []string `json:"stderr_contains"`
}

func readGitDriverJSONFixture(t *testing.T) gitDriverJSONFixture {
	t.Helper()
	source, err := os.ReadFile(filepath.Join("..", "..", "..", "fixtures", "diagnostics", "slice-951-git-driver-json-integration", "git-driver-json-integration.json"))
	if err != nil {
		t.Fatalf("read git driver fixture: %v", err)
	}
	var fixture gitDriverJSONFixture
	if err := json.Unmarshal(source, &fixture); err != nil {
		t.Fatalf("parse git driver fixture: %v", err)
	}
	return fixture
}

type gitDriverFallbackFixture struct {
	Cases []gitDriverFallbackCase `json:"cases"`
}

type gitDriverFallbackCase struct {
	CaseID       string                    `json:"case_id"`
	PathName     string                    `json:"path_name"`
	BaseSource   string                    `json:"base_source"`
	OursSource   string                    `json:"ours_source"`
	TheirsSource string                    `json:"theirs_source"`
	Options      gitDriverFallbackOptions  `json:"options"`
	Expected     gitDriverFallbackExpected `json:"expected"`
}

type gitDriverFallbackOptions struct {
	Strict   bool   `json:"strict"`
	Fallback string `json:"fallback"`
}

type gitDriverFallbackExpected struct {
	ExitCode          int                            `json:"exit_code"`
	MergedSource      string                         `json:"merged_source"`
	SourceContains    []string                       `json:"source_contains"`
	StderrContains    []string                       `json:"stderr_contains"`
	StderrNotContains []string                       `json:"stderr_not_contains"`
	MachineReport     gitDriverFallbackMachineReport `json:"machine_report"`
}

type gitDriverFallbackMachineReport struct {
	OK                 bool                           `json:"ok"`
	ExitCode           int                            `json:"exit_code"`
	Fallbacks          []gitDriverFallbackReportEntry `json:"fallbacks"`
	DiagnosticsContain []string                       `json:"diagnostics_contain"`
	RequiredFields     []string                       `json:"required_fields"`
}

type gitDriverFallbackReportEntry struct {
	Mode          string `json:"mode"`
	RequestedMode string `json:"requested_mode"`
	Reason        string `json:"reason"`
	Applied       bool   `json:"applied"`
}

func readGitDriverFallbackFixture(t *testing.T) gitDriverFallbackFixture {
	t.Helper()
	source, err := os.ReadFile(filepath.Join("..", "..", "..", "fixtures", "diagnostics", "slice-954-git-driver-fallback", "git-driver-fallback.json"))
	if err != nil {
		t.Fatalf("read git driver fallback fixture: %v", err)
	}
	var fixture gitDriverFallbackFixture
	if err := json.Unmarshal(source, &fixture); err != nil {
		t.Fatalf("parse git driver fallback fixture: %v", err)
	}
	return fixture
}

func assertFallbackMachineReport(t *testing.T, reportPath string, expected gitDriverFallbackMachineReport) {
	t.Helper()
	source, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read machine report: %v", err)
	}
	var reportFields map[string]json.RawMessage
	if err := json.Unmarshal(source, &reportFields); err != nil {
		t.Fatalf("parse machine report fields: %v", err)
	}
	for _, field := range expected.RequiredFields {
		if _, ok := reportFields[field]; !ok {
			t.Fatalf("machine report missing required field %q: %s", field, source)
		}
	}
	var report struct {
		OK          bool                           `json:"ok"`
		ExitCode    int                            `json:"exit_code"`
		Fallbacks   []gitDriverFallbackReportEntry `json:"fallbacks"`
		Diagnostics []astmerge.Diagnostic          `json:"diagnostics"`
	}
	if err := json.Unmarshal(source, &report); err != nil {
		t.Fatalf("parse machine report: %v", err)
	}
	if report.OK != expected.OK || report.ExitCode != expected.ExitCode {
		t.Fatalf("unexpected report status: %+v expected ok=%v exit=%d", report, expected.OK, expected.ExitCode)
	}
	if len(report.Fallbacks) != len(expected.Fallbacks) {
		t.Fatalf("unexpected fallbacks: %+v expected %+v", report.Fallbacks, expected.Fallbacks)
	}
	for index, expectedFallback := range expected.Fallbacks {
		if report.Fallbacks[index] != expectedFallback {
			t.Fatalf("unexpected fallback %d: %+v expected %+v", index, report.Fallbacks[index], expectedFallback)
		}
	}
	diagnosticsJSON, err := json.Marshal(report.Diagnostics)
	if err != nil {
		t.Fatalf("marshal diagnostics: %v", err)
	}
	for _, needle := range expected.DiagnosticsContain {
		if !strings.Contains(string(diagnosticsJSON), needle) {
			t.Fatalf("expected diagnostics to contain %q: %s", needle, string(diagnosticsJSON))
		}
	}
}

type goMerge3Fixture struct {
	Cases []goMerge3Case `json:"cases"`
}

type goMerge3Case struct {
	CaseID       string           `json:"case_id"`
	PathName     string           `json:"path_name"`
	BaseSource   string           `json:"base_source"`
	OursSource   string           `json:"ours_source"`
	TheirsSource string           `json:"theirs_source"`
	Expected     goMerge3Expected `json:"expected"`
}

type goMerge3Expected struct {
	OK             bool   `json:"ok"`
	ExpectedSource string `json:"expected_source"`
}

func readGoMerge3Fixture(t *testing.T) goMerge3Fixture {
	t.Helper()
	source, err := os.ReadFile(filepath.Join("..", "..", "..", "fixtures", "go", "slice-952-go-merge3", "go-merge3.json"))
	if err != nil {
		t.Fatalf("read go merge3 fixture: %v", err)
	}
	var fixture goMerge3Fixture
	if err := json.Unmarshal(source, &fixture); err != nil {
		t.Fatalf("parse go merge3 fixture: %v", err)
	}
	return fixture
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
	}
}

func jsonEqual(left any, right any) bool {
	leftSource, err := json.Marshal(left)
	if err != nil {
		return false
	}
	rightSource, err := json.Marshal(right)
	if err != nil {
		return false
	}
	return bytes.Equal(leftSource, rightSource)
}

func TestMergeDriverUsesSmorgLanguageAttribute(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(filepath.Join(dir, ".gitattributes"), []byte("*.data smorg.language=json\n"), 0o644); err != nil {
		t.Fatalf("write gitattributes: %v", err)
	}
	ancestor := writeTestFile(t, dir, "ancestor.tmp", `{"name":"structuredmerge"}`)
	current := writeTestFile(t, dir, "current.tmp", `{"name":"structuredmerge","current":true}`)
	other := writeTestFile(t, dir, "other.tmp", `{"name":"structuredmerge","other":true}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"merge-driver", ancestor, current, other, "package.data"}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("unexpected exit code %d stderr=%s", exitCode, stderr.String())
	}

	merged, err := os.ReadFile(current)
	if err != nil {
		t.Fatalf("read merged current file: %v", err)
	}
	mergedSource := string(merged)
	if !strings.Contains(mergedSource, `"current":true`) || !strings.Contains(mergedSource, `"other":true`) {
		t.Fatalf("attribute-selected JSON merge did not preserve both sides: %s", mergedSource)
	}
}

func TestMergeDriverCheckOnlyExitCodeReportsPendingChangeWithoutWriting(t *testing.T) {
	dir := t.TempDir()
	ancestor := writeTestFile(t, dir, "ancestor.json", `{"name":"structuredmerge"}`)
	current := writeTestFile(t, dir, "current.json", `{"name":"structuredmerge","current":true}`)
	other := writeTestFile(t, dir, "other.json", `{"name":"structuredmerge","other":true}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"merge-driver", "--check-only", "--exit-code", ancestor, current, other, "package.json"}, &stdout, &stderr)
	if exitCode != exitUnresolvedConflict {
		t.Fatalf("unexpected exit code %d stderr=%s", exitCode, stderr.String())
	}

	currentSource, err := os.ReadFile(current)
	if err != nil {
		t.Fatalf("read current file: %v", err)
	}
	if strings.Contains(string(currentSource), `"other":true`) {
		t.Fatalf("check-only wrote to current file: %s", string(currentSource))
	}
}

func TestMergeDriverCheckOnlyExitCodeReportsNoChange(t *testing.T) {
	dir := t.TempDir()
	ancestor := writeTestFile(t, dir, "ancestor.json", `{"name":"structuredmerge"}`)
	current := writeTestFile(t, dir, "current.json", `{"name":"structuredmerge","same":true}`)
	other := writeTestFile(t, dir, "other.json", `{"name":"structuredmerge","same":true}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"merge-driver", "--check-only", "--exit-code", ancestor, current, other, "package.json"}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("unexpected exit code %d stderr=%s", exitCode, stderr.String())
	}
}

func TestMergeDriverProfileReportAndRequiredStatus(t *testing.T) {
	dir := t.TempDir()
	ancestor := writeTestFile(t, dir, "ancestor.json", `{"name":"structuredmerge"}`)
	current := writeTestFile(t, dir, "current.json", `{"name":"structuredmerge","current":true}`)
	other := writeTestFile(t, dir, "other.json", `{"name":"structuredmerge","other":true}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"merge-driver", "--profile", "json.keyed-object", "--profile-report", "--require-profile-status", "recommended", ancestor, current, other, "package.json"}, &stdout, &stderr)
	if exitCode != exitUserError {
		t.Fatalf("unexpected exit code %d stderr=%s", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"rejection_code":"profile_status_unmet"`) {
		t.Fatalf("expected profile report rejection, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "profile status available is below required recommended") {
		t.Fatalf("expected concise profile status stderr, got %q", stderr.String())
	}
}

func TestMergeDriverUsesSmorgProfileAttributes(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(filepath.Join(dir, ".gitattributes"), []byte("*.json smorg.profile=json.keyed-object smorg.requireProfileStatus=recommended\n"), 0o644); err != nil {
		t.Fatalf("write gitattributes: %v", err)
	}
	ancestor := writeTestFile(t, dir, "ancestor.json", `{"name":"structuredmerge"}`)
	current := writeTestFile(t, dir, "current.json", `{"name":"structuredmerge","current":true}`)
	other := writeTestFile(t, dir, "other.json", `{"name":"structuredmerge","other":true}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"merge-driver", "--profile-report", ancestor, current, other, "package.json"}, &stdout, &stderr)
	if exitCode != exitUserError {
		t.Fatalf("unexpected exit code %d stderr=%s", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"profile_id":"json.keyed-object"`) ||
		!strings.Contains(stdout.String(), `"rejection_code":"profile_status_unmet"`) {
		t.Fatalf("expected profile attribute report, got %q", stdout.String())
	}
}

func TestPathSettingsUseLinguistLanguageAndConflictMarkerSize(t *testing.T) {
	settings := pathSettings{conflictMarkerSize: 7}
	applyAttributes(&settings, "fixtures/package.data", strings.Join([]string{
		"*.skip smorg.language=go",
		"*.data linguist-language=json conflict-marker-size=12",
	}, "\n"))

	if settings.language != "json" {
		t.Fatalf("expected linguist language, got %q", settings.language)
	}
	if settings.conflictMarkerSize != 12 {
		t.Fatalf("expected conflict marker size 12, got %d", settings.conflictMarkerSize)
	}
}

func TestDiffDriverTwoArgumentFormPrintsStructuredDiff(t *testing.T) {
	dir := t.TempDir()
	oldPath := writeTestFile(t, dir, "old.go", "package main\n\nfunc Old() {}\n")
	newPath := writeTestFile(t, dir, "new.go", "package main\n\nfunc New() {}\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"diff-driver", "--path-name", "main.go", oldPath, newPath}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("unexpected exit code %d stderr=%s", exitCode, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "structured-diff main.go") || !strings.Contains(output, "status changed") {
		t.Fatalf("expected structured diff output, got %q", output)
	}
}

func TestDiffDriverGitExternalDiffForms(t *testing.T) {
	for _, argumentCount := range []int{7, 9} {
		t.Run(fmt.Sprintf("args-%d", argumentCount), func(t *testing.T) {
			dir := t.TempDir()
			oldPath := writeTestFile(t, dir, "old.json", `{"old":true}`)
			newPath := writeTestFile(t, dir, "new.json", `{"new":true}`)

			args := []string{"diff-driver", "package.json", oldPath, "abc123", "100644", newPath, "def456", "100644"}
			if argumentCount == 9 {
				args = append(args, "a/", "b/")
			}

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			exitCode := run(args, &stdout, &stderr)
			if exitCode != exitSuccess {
				t.Fatalf("unexpected exit code %d stderr=%s", exitCode, stderr.String())
			}
			if !strings.Contains(stdout.String(), "structured-diff package.json") {
				t.Fatalf("expected path-named structured diff output, got %q", stdout.String())
			}
		})
	}
}

func TestLanguagesGitattributes(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"languages", "--gitattributes"}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("unexpected exit code %d stderr=%s", exitCode, stderr.String())
	}

	output := stdout.String()
	for _, expected := range []string{
		"*.go merge=smorg-go diff=smorg-go smorg.language=go",
		"*.json merge=smorg-go diff=smorg-go smorg.language=json",
		"*.jsonc merge=smorg-go diff=smorg-go smorg.language=jsonc",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected %q in gitattributes output, got %q", expected, output)
		}
	}
}

func TestConflictsDiffReportsConflictRegions(t *testing.T) {
	dir := t.TempDir()
	conflicted := writeTestFile(t, dir, "conflicted.go", strings.Join([]string{
		"package main",
		"<<<<<<< ours",
		"func Current() {}",
		"=======",
		"func Other() {}",
		">>>>>>> theirs",
		"",
	}, "\n"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"conflicts", "diff", "--path-name", "main.go", conflicted}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("unexpected exit code %d stderr=%s", exitCode, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "conflicts main.go") || !strings.Contains(output, "count 1") || !strings.Contains(output, "conflict 1 lines 2-6 separator 4") {
		t.Fatalf("expected conflict region output, got %q", output)
	}
}

func TestConflictsDiffExitCodeReportsUnresolvedConflicts(t *testing.T) {
	dir := t.TempDir()
	conflicted := writeTestFile(t, dir, "conflicted.go", "<<<<<<< ours\nx\n=======\ny\n>>>>>>> theirs\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"conflicts", "diff", "--exit-code", conflicted}, &stdout, &stderr)
	if exitCode != exitUnresolvedConflict {
		t.Fatalf("unexpected exit code %d stderr=%s stdout=%s", exitCode, stderr.String(), stdout.String())
	}
}

func TestConflictsDiffUsesConflictMarkerSizeAttribute(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(filepath.Join(dir, ".gitattributes"), []byte("*.go conflict-marker-size=9\n"), 0o644); err != nil {
		t.Fatalf("write gitattributes: %v", err)
	}
	conflicted := writeTestFile(t, dir, "conflicted.go", strings.Join([]string{
		"<<<<<<<<< ours",
		"x",
		"=========",
		"y",
		">>>>>>>>> theirs",
		"",
	}, "\n"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"conflicts", "diff", "--path-name", "conflicted.go", conflicted}, &stdout, &stderr)
	if exitCode != exitSuccess {
		t.Fatalf("unexpected exit code %d stderr=%s", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "count 1") {
		t.Fatalf("expected custom-marker conflict count, got %q", stdout.String())
	}
}

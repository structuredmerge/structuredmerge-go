package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	if !strings.Contains(stderr.String(), "destination_parse_error") {
		t.Fatalf("expected destination parse diagnostic, got %q", stderr.String())
	}
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

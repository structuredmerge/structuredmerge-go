package astmerge_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/markdownmerge"
	"github.com/structuredmerge/structuredmerge-go/rubymerge"
	"github.com/structuredmerge/structuredmerge-go/tomlmerge"
)

func readManifest(t *testing.T) astmerge.ConformanceManifest {
	t.Helper()

	path := filepath.Join("..", "..", "fixtures", "conformance", "slice-24-manifest", "family-feature-profiles.json")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}

	var manifest astmerge.ConformanceManifest
	if err := json.Unmarshal(source, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}

	return manifest
}

func diagnosticsFixturePath(t *testing.T, role string) string {
	t.Helper()

	manifest := readManifest(t)
	path := astmerge.ConformanceFixturePath(manifest, "diagnostics", role)
	if path == nil {
		t.Fatalf("missing diagnostics fixture entry for %s", role)
	}

	return filepath.Join(append([]string{"..", "..", "fixtures"}, path...)...)
}

func readDiagnosticFixtureFromPath(t *testing.T, path string) map[string]any {
	t.Helper()

	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var fixture map[string]any
	if err := json.Unmarshal(source, &fixture); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	return fixture
}

func decodeFixtureValue[T any](t *testing.T, raw any) T {
	t.Helper()

	data, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("marshal fixture value: %v", err)
	}

	var decoded T
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal fixture value: %v", err)
	}

	return decoded
}

func readRelativeFileTree(t *testing.T, root string) map[string]string {
	t.Helper()

	files := map[string]string{}
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relativePath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(relativePath)] = string(source)
		return nil
	}); err != nil {
		t.Fatalf("walk fixture tree: %v", err)
	}

	return files
}

func mapsKeys[V any](input map[string]V) []string {
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}

	return keys
}

func multiFamilyMergeCallback(entry astmerge.TemplateExecutionPlanEntry) astmerge.MergeResult[string] {
	switch entry.Classification.Family {
	case "markdown":
		return markdownmerge.MergeMarkdown(
			*entry.PreparedTemplateContent,
			*entry.DestinationContent,
			markdownmerge.DialectMarkdown,
		)
	case "toml":
		return tomlmerge.MergeTOML(
			*entry.PreparedTemplateContent,
			*entry.DestinationContent,
			tomlmerge.DialectTOML,
		)
	case "ruby":
		return rubymerge.MergeRuby(
			*entry.PreparedTemplateContent,
			*entry.DestinationContent,
			rubymerge.DialectRuby,
		)
	default:
		return astmerge.MergeResult[string]{
			OK: false,
			Diagnostics: []astmerge.Diagnostic{{
				Severity: astmerge.SeverityError,
				Category: astmerge.CategoryConfigurationError,
				Message:  "missing family merge adapter for " + entry.Classification.Family,
			}},
		}
	}
}

func repoTempDir(t *testing.T) string {
	t.Helper()

	root := filepath.Join("tmp")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("create tmp root: %v", err)
	}
	path, err := os.MkdirTemp(root, "astmerge-")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(path)
	})
	return path
}

func TestMiniTemplateTreeFamilyMergeCallbackFixture(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "mini_template_tree_family_merge_callback"))
	fixtureDir := filepath.Dir(diagnosticsFixturePath(t, "mini_template_tree_family_merge_callback"))
	templateContents := readRelativeFileTree(t, filepath.Join(fixtureDir, "template"))
	destinationContents := readRelativeFileTree(t, filepath.Join(fixtureDir, "destination"))
	templateSourcePaths := mapsKeys(templateContents)
	slices.Sort(templateSourcePaths)
	context := decodeFixtureValue[astmerge.TemplateDestinationContext](t, fixture["context"])
	overrides := decodeFixtureValue[[]astmerge.TemplateStrategyOverride](t, fixture["overrides"])
	replacements := decodeFixtureValue[map[string]string](t, fixture["replacements"])

	actual := astmerge.RunTemplateTreeExecution(
		templateSourcePaths,
		templateContents,
		destinationContents,
		&context,
		astmerge.TemplateStrategy(fixture["default_strategy"].(string)),
		overrides,
		replacements,
		func(entry astmerge.TemplateExecutionPlanEntry) astmerge.MergeResult[string] {
			switch entry.Classification.Family {
			case "markdown":
				return markdownmerge.MergeMarkdown(
					*entry.PreparedTemplateContent,
					*entry.DestinationContent,
					markdownmerge.DialectMarkdown,
				)
			default:
				return astmerge.MergeResult[string]{
					OK: false,
					Diagnostics: []astmerge.Diagnostic{{
						Severity: astmerge.SeverityError,
						Category: astmerge.CategoryConfigurationError,
						Message:  "missing family merge adapter for " + entry.Classification.Family,
					}},
				}
			}
		},
		nil,
	)
	expected := decodeFixtureValue[astmerge.TemplateTreeRunResult](t, fixture["expected"])
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("expected mini template tree family merge callback to match fixture")
	}
}

func TestMiniTemplateTreeMultiFamilyMergeCallbackFixture(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "mini_template_tree_multi_family_merge_callback"))
	fixtureDir := filepath.Dir(diagnosticsFixturePath(t, "mini_template_tree_multi_family_merge_callback"))
	templateContents := readRelativeFileTree(t, filepath.Join(fixtureDir, "template"))
	destinationContents := readRelativeFileTree(t, filepath.Join(fixtureDir, "destination"))
	templateSourcePaths := mapsKeys(templateContents)
	slices.Sort(templateSourcePaths)
	context := decodeFixtureValue[astmerge.TemplateDestinationContext](t, fixture["context"])
	overrides := decodeFixtureValue[[]astmerge.TemplateStrategyOverride](t, fixture["overrides"])
	replacements := decodeFixtureValue[map[string]string](t, fixture["replacements"])

	actual := astmerge.RunTemplateTreeExecution(
		templateSourcePaths,
		templateContents,
		destinationContents,
		&context,
		astmerge.TemplateStrategy(fixture["default_strategy"].(string)),
		overrides,
		replacements,
		func(entry astmerge.TemplateExecutionPlanEntry) astmerge.MergeResult[string] {
			switch entry.Classification.Family {
			case "markdown":
				return markdownmerge.MergeMarkdown(
					*entry.PreparedTemplateContent,
					*entry.DestinationContent,
					markdownmerge.DialectMarkdown,
				)
			case "toml":
				return tomlmerge.MergeTOML(
					*entry.PreparedTemplateContent,
					*entry.DestinationContent,
					tomlmerge.DialectTOML,
				)
			case "ruby":
				return rubymerge.MergeRuby(
					*entry.PreparedTemplateContent,
					*entry.DestinationContent,
					rubymerge.DialectRuby,
				)
			default:
				return astmerge.MergeResult[string]{
					OK: false,
					Diagnostics: []astmerge.Diagnostic{{
						Severity: astmerge.SeverityError,
						Category: astmerge.CategoryConfigurationError,
						Message:  "missing family merge adapter for " + entry.Classification.Family,
					}},
				}
			}
		},
		nil,
	)
	expected := decodeFixtureValue[astmerge.TemplateTreeRunResult](t, fixture["expected"])
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("expected mini template tree multi-family merge callback to match fixture")
	}
}

func TestMiniTemplateTreeMultiFamilyRunReportFixture(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "mini_template_tree_multi_family_merge_callback"))
	reportFixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "mini_template_tree_multi_family_run_report"))
	fixtureDir := filepath.Dir(diagnosticsFixturePath(t, "mini_template_tree_multi_family_merge_callback"))
	templateContents := readRelativeFileTree(t, filepath.Join(fixtureDir, "template"))
	destinationContents := readRelativeFileTree(t, filepath.Join(fixtureDir, "destination"))
	templateSourcePaths := mapsKeys(templateContents)
	slices.Sort(templateSourcePaths)
	context := decodeFixtureValue[astmerge.TemplateDestinationContext](t, fixture["context"])
	overrides := decodeFixtureValue[[]astmerge.TemplateStrategyOverride](t, fixture["overrides"])
	replacements := decodeFixtureValue[map[string]string](t, fixture["replacements"])

	runResult := astmerge.RunTemplateTreeExecution(
		templateSourcePaths,
		templateContents,
		destinationContents,
		&context,
		astmerge.TemplateStrategy(fixture["default_strategy"].(string)),
		overrides,
		replacements,
		multiFamilyMergeCallback,
		nil,
	)
	actual := astmerge.ReportTemplateTreeRun(runResult)
	expected := decodeFixtureValue[astmerge.TemplateTreeRunReport](t, reportFixture["expected"])
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("expected mini template tree multi-family run report to match fixture")
	}
}

func TestMiniTemplateTreeDirectoryRunReportFixture(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "mini_template_tree_directory_run_report"))
	fixtureDir := filepath.Dir(diagnosticsFixturePath(t, "mini_template_tree_directory_run_report"))
	context := decodeFixtureValue[astmerge.TemplateDestinationContext](t, fixture["context"])
	overrides := decodeFixtureValue[[]astmerge.TemplateStrategyOverride](t, fixture["overrides"])
	replacements := decodeFixtureValue[map[string]string](t, fixture["replacements"])

	runResult, err := astmerge.RunTemplateTreeExecutionFromDirectories(
		filepath.Join(fixtureDir, "template"),
		filepath.Join(fixtureDir, "destination"),
		&context,
		astmerge.TemplateStrategy(fixture["default_strategy"].(string)),
		overrides,
		replacements,
		multiFamilyMergeCallback,
		nil,
	)
	if err != nil {
		t.Fatalf("run template tree from directories: %v", err)
	}

	actual := astmerge.ReportTemplateTreeRun(runResult)
	expected := decodeFixtureValue[astmerge.TemplateTreeRunReport](t, fixture["expected"])
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("expected mini template tree directory run report to match fixture")
	}
}

func TestMiniTemplateTreeDirectoryApplyConvergenceFixture(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "mini_template_tree_directory_apply_convergence"))
	fixtureDir := filepath.Dir(diagnosticsFixturePath(t, "mini_template_tree_directory_apply_convergence"))
	context := decodeFixtureValue[astmerge.TemplateDestinationContext](t, fixture["context"])
	overrides := decodeFixtureValue[[]astmerge.TemplateStrategyOverride](t, fixture["overrides"])
	replacements := decodeFixtureValue[map[string]string](t, fixture["replacements"])
	tempRoot := repoTempDir(t)
	destinationRoot := filepath.Join(tempRoot, "destination")

	initialDestination, err := astmerge.ReadRelativeFileTree(filepath.Join(fixtureDir, "destination"))
	if err != nil {
		t.Fatalf("read initial destination tree: %v", err)
	}
	if err := astmerge.WriteRelativeFileTree(destinationRoot, initialDestination); err != nil {
		t.Fatalf("seed destination tree: %v", err)
	}

	firstRun, err := astmerge.ApplyTemplateTreeExecutionToDirectory(
		filepath.Join(fixtureDir, "template"),
		destinationRoot,
		&context,
		astmerge.TemplateStrategy(fixture["default_strategy"].(string)),
		overrides,
		replacements,
		multiFamilyMergeCallback,
		nil,
	)
	if err != nil {
		t.Fatalf("apply template tree to directory: %v", err)
	}
	firstActual := astmerge.ReportTemplateTreeRun(firstRun)
	firstExpected := decodeFixtureValue[astmerge.TemplateTreeRunReport](t, fixture["expected_first_report"])
	if !reflect.DeepEqual(firstActual, firstExpected) {
		t.Fatalf("expected first directory apply report to match fixture")
	}

	actualFiles, err := astmerge.ReadRelativeFileTree(destinationRoot)
	if err != nil {
		t.Fatalf("read applied destination tree: %v", err)
	}
	expectedFiles := decodeFixtureValue[map[string]string](t, fixture["expected_destination_files"])
	if !reflect.DeepEqual(actualFiles, expectedFiles) {
		t.Fatalf("expected applied destination tree to match fixture")
	}

	secondRun, err := astmerge.ApplyTemplateTreeExecutionToDirectory(
		filepath.Join(fixtureDir, "template"),
		destinationRoot,
		&context,
		astmerge.TemplateStrategy(fixture["default_strategy"].(string)),
		overrides,
		replacements,
		multiFamilyMergeCallback,
		nil,
	)
	if err != nil {
		t.Fatalf("reapply template tree to directory: %v", err)
	}
	secondActual := astmerge.ReportTemplateTreeRun(secondRun)
	secondExpected := decodeFixtureValue[astmerge.TemplateTreeRunReport](t, fixture["expected_second_report"])
	if !reflect.DeepEqual(secondActual, secondExpected) {
		t.Fatalf("expected second directory apply report to match fixture")
	}
}

func TestMiniTemplateTreeDirectoryApplyReportFixture(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "mini_template_tree_directory_apply_report"))
	fixtureDir := filepath.Dir(diagnosticsFixturePath(t, "mini_template_tree_directory_apply_report"))
	context := decodeFixtureValue[astmerge.TemplateDestinationContext](t, fixture["context"])
	overrides := decodeFixtureValue[[]astmerge.TemplateStrategyOverride](t, fixture["overrides"])
	replacements := decodeFixtureValue[map[string]string](t, fixture["replacements"])
	tempRoot := repoTempDir(t)
	destinationRoot := filepath.Join(tempRoot, "destination")

	initialDestination, err := astmerge.ReadRelativeFileTree(filepath.Join(fixtureDir, "destination"))
	if err != nil {
		t.Fatalf("read initial destination tree: %v", err)
	}
	if err := astmerge.WriteRelativeFileTree(destinationRoot, initialDestination); err != nil {
		t.Fatalf("seed destination tree: %v", err)
	}

	firstRun, err := astmerge.ApplyTemplateTreeExecutionToDirectory(
		filepath.Join(fixtureDir, "template"),
		destinationRoot,
		&context,
		astmerge.TemplateStrategy(fixture["default_strategy"].(string)),
		overrides,
		replacements,
		multiFamilyMergeCallback,
		nil,
	)
	if err != nil {
		t.Fatalf("apply template tree to directory: %v", err)
	}
	firstActual := astmerge.ReportTemplateDirectoryApply(firstRun)
	firstExpected := decodeFixtureValue[astmerge.TemplateDirectoryApplyReport](t, fixture["expected_first_report"])
	if !reflect.DeepEqual(firstActual, firstExpected) {
		t.Fatalf("expected first directory apply report to match fixture")
	}

	secondRun, err := astmerge.ApplyTemplateTreeExecutionToDirectory(
		filepath.Join(fixtureDir, "template"),
		destinationRoot,
		&context,
		astmerge.TemplateStrategy(fixture["default_strategy"].(string)),
		overrides,
		replacements,
		multiFamilyMergeCallback,
		nil,
	)
	if err != nil {
		t.Fatalf("reapply template tree to directory: %v", err)
	}
	secondActual := astmerge.ReportTemplateDirectoryApply(secondRun)
	secondExpected := decodeFixtureValue[astmerge.TemplateDirectoryApplyReport](t, fixture["expected_second_report"])
	if !reflect.DeepEqual(secondActual, secondExpected) {
		t.Fatalf("expected second directory apply report to match fixture")
	}
}

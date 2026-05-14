package astmerge_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
)

var fixtureMarkdownHeadingPattern = regexp.MustCompile(`^(#{1,6})\s+(.+?)\s*#*\s*$`)
var fixtureMarkdownCodeFencePattern = regexp.MustCompile(`^\s*(` + "```" + `+|~~~+)`)
var fixtureRubyTopLevelDefPattern = regexp.MustCompile(`(?ms)^def\s+([A-Za-z_][A-Za-z0-9_!?=]*)\b.*?^end\s*$`)

type fixtureMarkdownSection struct {
	Path string
	Text string
}

type fixtureTomlEntry struct {
	Key   string
	Value string
}

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

func fixtureMarkdownMerge(templateSource string, destinationSource string) astmerge.MergeResult[string] {
	destinationSections := fixtureMarkdownSections(destinationSource)
	templateSections := fixtureMarkdownSections(templateSource)
	destinationPaths := map[string]struct{}{}
	mergedSections := make([]string, 0, len(destinationSections)+len(templateSections))
	for _, section := range destinationSections {
		destinationPaths[section.Path] = struct{}{}
		if section.Text != "" {
			mergedSections = append(mergedSections, section.Text)
		}
	}
	for _, section := range templateSections {
		if _, ok := destinationPaths[section.Path]; ok || section.Text == "" {
			continue
		}
		mergedSections = append(mergedSections, section.Text)
	}
	output := strings.TrimSpace(strings.Join(mergedSections, "\n\n")) + "\n"
	return astmerge.MergeResult[string]{
		OK:          true,
		Diagnostics: []astmerge.Diagnostic{},
		Output:      &output,
		Policies:    []astmerge.PolicyReference{},
	}
}

func fixtureMarkdownSections(source string) []fixtureMarkdownSection {
	lines := strings.Split(strings.ReplaceAll(strings.ReplaceAll(source, "\r\n", "\n"), "\r", "\n"), "\n")
	type ownerStart struct {
		Path  string
		Start int
	}
	owners := []ownerStart{}
	headingIndex := 0
	codeFenceIndex := 0
	for index := 0; index < len(lines); index++ {
		line := lines[index]
		if fixtureMarkdownHeadingPattern.MatchString(line) {
			owners = append(owners, ownerStart{
				Path:  "/heading/" + strconv.Itoa(headingIndex),
				Start: index,
			})
			headingIndex++
			continue
		}
		fence := fixtureMarkdownCodeFencePattern.FindStringSubmatch(line)
		if fence == nil {
			continue
		}
		owners = append(owners, ownerStart{
			Path:  "/code_fence/" + strconv.Itoa(codeFenceIndex),
			Start: index,
		})
		codeFenceIndex++
		marker := fence[1]
		markerChar := marker[:1]
		markerLength := len(marker)
		for cursor := index + 1; cursor < len(lines); cursor++ {
			trimmed := strings.TrimSpace(lines[cursor])
			if len(trimmed) >= markerLength &&
				strings.Trim(trimmed, markerChar) == "" &&
				strings.HasPrefix(trimmed, strings.Repeat(markerChar, markerLength)) {
				index = cursor
				break
			}
			if cursor == len(lines)-1 {
				index = cursor
			}
		}
	}
	slices.SortFunc(owners, func(left, right ownerStart) int { return left.Start - right.Start })
	sections := make([]fixtureMarkdownSection, 0, len(owners))
	for index, owner := range owners {
		endExclusive := len(lines)
		if index+1 < len(owners) {
			endExclusive = owners[index+1].Start
		}
		text := strings.TrimSpace(strings.Join(lines[owner.Start:endExclusive], "\n"))
		sections = append(sections, fixtureMarkdownSection{Path: owner.Path, Text: text})
	}
	return sections
}

func fixtureTOMLMerge(templateSource string, destinationSource string) astmerge.MergeResult[string] {
	templateSections, templateOrder := fixtureTOMLSections(templateSource)
	destinationSections, destinationOrder := fixtureTOMLSections(destinationSource)
	sectionSeen := map[string]bool{}
	sectionOrder := []string{}
	for _, section := range append(templateOrder, destinationOrder...) {
		if !sectionSeen[section] {
			sectionSeen[section] = true
			sectionOrder = append(sectionOrder, section)
		}
	}

	blocks := []string{}
	for _, section := range sectionOrder {
		keys := map[string]bool{}
		for _, entry := range append(templateSections[section], destinationSections[section]...) {
			keys[entry.Key] = true
		}
		keyList := mapsKeys(keys)
		slices.Sort(keyList)
		lines := []string{}
		if section != "" {
			lines = append(lines, "["+section+"]")
		}
		for _, key := range keyList {
			value, ok := fixtureTOMLValue(destinationSections[section], key)
			if !ok {
				value, _ = fixtureTOMLValue(templateSections[section], key)
			}
			lines = append(lines, key+" = "+value)
		}
		if len(lines) > 0 {
			blocks = append(blocks, strings.Join(lines, "\n"))
		}
	}
	output := strings.TrimSpace(strings.Join(blocks, "\n\n")) + "\n"
	return astmerge.MergeResult[string]{OK: true, Diagnostics: []astmerge.Diagnostic{}, Output: &output, Policies: []astmerge.PolicyReference{}}
}

func fixtureTOMLSections(source string) (map[string][]fixtureTomlEntry, []string) {
	sections := map[string][]fixtureTomlEntry{}
	order := []string{""}
	seen := map[string]bool{"": true}
	current := ""
	for _, line := range strings.Split(strings.ReplaceAll(strings.ReplaceAll(source, "\r\n", "\n"), "\r", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			current = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(trimmed, "["), "]"))
			if !seen[current] {
				seen[current] = true
				order = append(order, current)
			}
			continue
		}
		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) != 2 {
			continue
		}
		sections[current] = append(sections[current], fixtureTomlEntry{
			Key:   strings.TrimSpace(parts[0]),
			Value: strings.TrimSpace(parts[1]),
		})
	}
	return sections, order
}

func fixtureTOMLValue(entries []fixtureTomlEntry, key string) (string, bool) {
	for _, entry := range entries {
		if entry.Key == key {
			return entry.Value, true
		}
	}
	return "", false
}

func fixtureRubyMerge(templateSource string, destinationSource string) astmerge.MergeResult[string] {
	output := strings.TrimSpace(destinationSource)
	destinationDefs := map[string]bool{}
	for _, match := range fixtureRubyTopLevelDefPattern.FindAllStringSubmatch(destinationSource, -1) {
		destinationDefs[match[1]] = true
	}
	for _, match := range fixtureRubyTopLevelDefPattern.FindAllStringSubmatch(templateSource, -1) {
		if destinationDefs[match[1]] {
			continue
		}
		output += "\n\n" + strings.TrimSpace(match[0])
	}
	output += "\n"
	return astmerge.MergeResult[string]{OK: true, Diagnostics: []astmerge.Diagnostic{}, Output: &output, Policies: []astmerge.PolicyReference{}}
}

func multiFamilyMergeCallback(entry astmerge.TemplateExecutionPlanEntry) astmerge.MergeResult[string] {
	switch entry.Classification.Family {
	case "markdown":
		return fixtureMarkdownMerge(*entry.PreparedTemplateContent, *entry.DestinationContent)
	case "toml":
		return fixtureTOMLMerge(*entry.PreparedTemplateContent, *entry.DestinationContent)
	case "ruby":
		return fixtureRubyMerge(*entry.PreparedTemplateContent, *entry.DestinationContent)
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
				return fixtureMarkdownMerge(*entry.PreparedTemplateContent, *entry.DestinationContent)
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
				return fixtureMarkdownMerge(*entry.PreparedTemplateContent, *entry.DestinationContent)
			case "toml":
				return fixtureTOMLMerge(*entry.PreparedTemplateContent, *entry.DestinationContent)
			case "ruby":
				return fixtureRubyMerge(*entry.PreparedTemplateContent, *entry.DestinationContent)
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

func TestMiniTemplateTreeDirectoryPlanReportFixture(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "mini_template_tree_directory_plan_report"))
	fixtureDir := filepath.Dir(diagnosticsFixturePath(t, "mini_template_tree_directory_plan_report"))
	context := decodeFixtureValue[astmerge.TemplateDestinationContext](t, fixture["context"])
	overrides := decodeFixtureValue[[]astmerge.TemplateStrategyOverride](t, fixture["overrides"])
	replacements := decodeFixtureValue[map[string]string](t, fixture["replacements"])

	executionPlan, err := astmerge.PlanTemplateTreeExecutionFromDirectories(
		filepath.Join(fixtureDir, "template"),
		filepath.Join(fixtureDir, "destination"),
		&context,
		astmerge.TemplateStrategy(fixture["default_strategy"].(string)),
		overrides,
		replacements,
		nil,
	)
	if err != nil {
		t.Fatalf("plan template tree from directories: %v", err)
	}

	actual := astmerge.ReportTemplateDirectoryPlan(executionPlan)
	expected := decodeFixtureValue[astmerge.TemplateDirectoryPlanReport](t, fixture["expected"])
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("expected directory plan report to match fixture")
	}
}

func TestMiniTemplateTreeDirectoryRunnerReportFixture(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "mini_template_tree_directory_runner_report"))

	dryRun := decodeFixtureValue[map[string]any](t, fixture["dry_run"])
	dryRunDir := filepath.Join(filepath.Dir(diagnosticsFixturePath(t, "mini_template_tree_directory_runner_report")), "dry-run")
	dryRunContext := decodeFixtureValue[astmerge.TemplateDestinationContext](t, dryRun["context"])
	dryRunOverrides := decodeFixtureValue[[]astmerge.TemplateStrategyOverride](t, dryRun["overrides"])
	dryRunReplacements := decodeFixtureValue[map[string]string](t, dryRun["replacements"])
	dryRunPlan, err := astmerge.PlanTemplateTreeExecutionFromDirectories(
		filepath.Join(dryRunDir, "template"),
		filepath.Join(dryRunDir, "destination"),
		&dryRunContext,
		astmerge.TemplateStrategy(dryRun["default_strategy"].(string)),
		dryRunOverrides,
		dryRunReplacements,
		nil,
	)
	if err != nil {
		t.Fatalf("plan dry-run template tree from directories: %v", err)
	}
	dryRunActual := astmerge.ReportTemplateDirectoryRunner(dryRunPlan, nil)
	dryRunExpected := decodeFixtureValue[astmerge.TemplateDirectoryRunnerReport](t, dryRun["expected"])
	if !reflect.DeepEqual(dryRunActual, dryRunExpected) {
		t.Fatalf("expected dry-run directory runner report to match fixture")
	}

	applyRun := decodeFixtureValue[map[string]any](t, fixture["apply_run"])
	applyRunDir := filepath.Join(filepath.Dir(diagnosticsFixturePath(t, "mini_template_tree_directory_runner_report")), "apply-run")
	applyContext := decodeFixtureValue[astmerge.TemplateDestinationContext](t, applyRun["context"])
	applyOverrides := decodeFixtureValue[[]astmerge.TemplateStrategyOverride](t, applyRun["overrides"])
	applyReplacements := decodeFixtureValue[map[string]string](t, applyRun["replacements"])
	tempRoot := repoTempDir(t)
	destinationRoot := filepath.Join(tempRoot, "destination")
	initialDestination, err := astmerge.ReadRelativeFileTree(filepath.Join(applyRunDir, "destination"))
	if err != nil {
		t.Fatalf("read apply-run destination tree: %v", err)
	}
	if err := astmerge.WriteRelativeFileTree(destinationRoot, initialDestination); err != nil {
		t.Fatalf("seed apply-run destination tree: %v", err)
	}
	applyPlan, err := astmerge.PlanTemplateTreeExecutionFromDirectories(
		filepath.Join(applyRunDir, "template"),
		destinationRoot,
		&applyContext,
		astmerge.TemplateStrategy(applyRun["default_strategy"].(string)),
		applyOverrides,
		applyReplacements,
		nil,
	)
	if err != nil {
		t.Fatalf("plan apply-run template tree from directories: %v", err)
	}
	applyResult, err := astmerge.ApplyTemplateTreeExecutionToDirectory(
		filepath.Join(applyRunDir, "template"),
		destinationRoot,
		&applyContext,
		astmerge.TemplateStrategy(applyRun["default_strategy"].(string)),
		applyOverrides,
		applyReplacements,
		multiFamilyMergeCallback,
		nil,
	)
	if err != nil {
		t.Fatalf("apply apply-run template tree to directory: %v", err)
	}
	applyActual := astmerge.ReportTemplateDirectoryRunner(applyPlan, &applyResult)
	applyExpected := decodeFixtureValue[astmerge.TemplateDirectoryRunnerReport](t, applyRun["expected"])
	if !reflect.DeepEqual(applyActual, applyExpected) {
		t.Fatalf("expected apply-run directory runner report to match fixture")
	}
}

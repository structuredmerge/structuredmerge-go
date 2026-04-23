package asttemplate_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/asttemplate"
	"github.com/structuredmerge/structuredmerge-go/markdownmerge"
	"github.com/structuredmerge/structuredmerge-go/rubymerge"
	"github.com/structuredmerge/structuredmerge-go/tomlmerge"
)

func TestTemplateDirectorySessionReportFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-353-template-directory-session-report", "template-directory-session-report.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)

	dryRun := fixture["dry_run"].(map[string]any)
	dryRunReport, err := asttemplate.PlanTemplateDirectorySessionFromDirectories(
		filepath.Join(fixtureRoot, "dry-run", "template"),
		filepath.Join(fixtureRoot, "dry-run", "destination"),
		decodeContext(t, dryRun["context"]),
		decodeStrategy(t, dryRun["default_strategy"]),
		decodeOverrides(t, dryRun["overrides"]),
		decodeReplacements(t, dryRun["replacements"]),
		nil,
	)
	if err != nil {
		t.Fatalf("dry-run session failed: %v", err)
	}
	assertJSONEqual(t, dryRun["expected"], dryRunReport)

	applyRun := fixture["apply_run"].(map[string]any)
	tempRoot := filepath.Join(repoRoot(t), "go", "asttemplate", "tmp", t.Name())
	_ = os.RemoveAll(tempRoot)
	if err := copyTree(filepath.Join(fixtureRoot, "apply-run", "destination"), tempRoot); err != nil {
		t.Fatalf("copy destination: %v", err)
	}
	defer os.RemoveAll(tempRoot)

	applyReport, err := asttemplate.ApplyTemplateDirectorySessionToDirectory(
		filepath.Join(fixtureRoot, "apply-run", "template"),
		tempRoot,
		decodeContext(t, applyRun["context"]),
		decodeStrategy(t, applyRun["default_strategy"]),
		decodeOverrides(t, applyRun["overrides"]),
		decodeReplacements(t, applyRun["replacements"]),
		multiFamilyMergeCallback,
		nil,
	)
	if err != nil {
		t.Fatalf("apply session failed: %v", err)
	}
	assertJSONEqual(t, applyRun["expected"], applyReport)

	reapplyRun := fixture["reapply_run"].(map[string]any)
	reapplyReport, err := asttemplate.ReapplyTemplateDirectorySessionToDirectory(
		filepath.Join(fixtureRoot, "apply-run", "template"),
		tempRoot,
		decodeContext(t, reapplyRun["context"]),
		decodeStrategy(t, reapplyRun["default_strategy"]),
		decodeOverrides(t, reapplyRun["overrides"]),
		decodeReplacements(t, reapplyRun["replacements"]),
		multiFamilyMergeCallback,
		nil,
	)
	if err != nil {
		t.Fatalf("reapply session failed: %v", err)
	}
	assertJSONEqual(t, reapplyRun["expected"], reapplyReport)
}

func TestTemplateDirectoryAdapterRegistryReportFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-354-template-directory-adapter-registry-report", "template-directory-adapter-registry-report.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)

	for key, registry := range map[string]asttemplate.FamilyMergeAdapterRegistry{
		"full_registry": {
			"markdown": markdownAdapter,
			"ruby":     rubyAdapter,
			"toml":     tomlAdapter,
		},
		"partial_registry": {
			"markdown": markdownAdapter,
			"toml":     tomlAdapter,
		},
	} {
		section := fixture[key].(map[string]any)
		tempRoot := filepath.Join(repoRoot(t), "go", "asttemplate", "tmp", t.Name(), key)
		_ = os.RemoveAll(tempRoot)
		if err := copyTree(filepath.Join(fixtureRoot, "apply-run", "destination"), tempRoot); err != nil {
			t.Fatalf("copy destination: %v", err)
		}

		actual, err := asttemplate.ApplyTemplateDirectorySessionWithRegistryToDirectory(
			filepath.Join(fixtureRoot, "apply-run", "template"),
			tempRoot,
			decodeContext(t, section["context"]),
			decodeStrategy(t, section["default_strategy"]),
			decodeOverrides(t, section["overrides"]),
			decodeReplacements(t, section["replacements"]),
			registry,
			nil,
		)
		if err != nil {
			t.Fatalf("%s registry session failed: %v", key, err)
		}
		assertJSONEqual(t, section["expected"], actual)
		_ = os.RemoveAll(tempRoot)
	}
}

func TestTemplateDirectoryDefaultAdapterDiscoveryReportFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-355-template-directory-default-adapter-discovery-report", "template-directory-default-adapter-discovery-report.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)

	for _, key := range []string{"default_discovery", "filtered_discovery"} {
		section := fixture[key].(map[string]any)
		tempRoot := filepath.Join(repoRoot(t), "go", "asttemplate", "tmp", t.Name(), key)
		_ = os.RemoveAll(tempRoot)
		if err := copyTree(filepath.Join(fixtureRoot, "apply-run", "destination"), tempRoot); err != nil {
			t.Fatalf("copy destination: %v", err)
		}
		actual, err := asttemplate.ApplyTemplateDirectorySessionWithDefaultRegistryToDirectory(
			filepath.Join(fixtureRoot, "apply-run", "template"),
			tempRoot,
			decodeContext(t, section["context"]),
			decodeStrategy(t, section["default_strategy"]),
			decodeOverrides(t, section["overrides"]),
			decodeReplacements(t, section["replacements"]),
			decodeOptionalFamilies(t, section["allowed_families"]),
			nil,
		)
		if err != nil {
			t.Fatalf("%s discovery session failed: %v", key, err)
		}
		assertJSONEqual(t, section["expected"], actual)
		_ = os.RemoveAll(tempRoot)
	}
}

func TestDefaultFamilyMergeAdapterRegistryFamilies(t *testing.T) {
	actual := asttemplate.RegisteredAdapterFamilies(asttemplate.DefaultFamilyMergeAdapterRegistry())
	expected := []string{"markdown", "ruby", "toml"}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("unexpected default adapter families: %#v", actual)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	return root
}

func readJSONFixture(t *testing.T, path string) map[string]any {
	t.Helper()
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fixture map[string]any
	if err := json.Unmarshal(payload, &fixture); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	return fixture
}

func decodeContext(t *testing.T, raw any) *astmerge.TemplateDestinationContext {
	t.Helper()
	data, _ := json.Marshal(raw)
	var context astmerge.TemplateDestinationContext
	if err := json.Unmarshal(data, &context); err != nil {
		t.Fatalf("decode context: %v", err)
	}
	return &context
}

func decodeStrategy(t *testing.T, raw any) astmerge.TemplateStrategy {
	t.Helper()
	data, _ := json.Marshal(raw)
	var strategy astmerge.TemplateStrategy
	if err := json.Unmarshal(data, &strategy); err != nil {
		t.Fatalf("decode strategy: %v", err)
	}
	return strategy
}

func decodeOverrides(t *testing.T, raw any) []astmerge.TemplateStrategyOverride {
	t.Helper()
	data, _ := json.Marshal(raw)
	var overrides []astmerge.TemplateStrategyOverride
	if err := json.Unmarshal(data, &overrides); err != nil {
		t.Fatalf("decode overrides: %v", err)
	}
	return overrides
}

func decodeReplacements(t *testing.T, raw any) map[string]string {
	t.Helper()
	data, _ := json.Marshal(raw)
	replacements := map[string]string{}
	if err := json.Unmarshal(data, &replacements); err != nil {
		t.Fatalf("decode replacements: %v", err)
	}
	return replacements
}

func decodeOptionalFamilies(t *testing.T, raw any) []string {
	t.Helper()
	if raw == nil {
		return nil
	}
	data, _ := json.Marshal(raw)
	var families []string
	if err := json.Unmarshal(data, &families); err != nil {
		t.Fatalf("decode allowed families: %v", err)
	}
	return families
}

func assertJSONEqual(t *testing.T, expected any, actual any) {
	t.Helper()
	expectedJSON, _ := json.Marshal(expected)
	actualJSON, _ := json.Marshal(actual)
	var expectedValue any
	var actualValue any
	_ = json.Unmarshal(expectedJSON, &expectedValue)
	_ = json.Unmarshal(actualJSON, &actualValue)
	if !reflect.DeepEqual(expectedValue, actualValue) {
		t.Fatalf("json mismatch\nexpected: %s\nactual:   %s", expectedJSON, actualJSON)
	}
}

func copyTree(src string, dst string) error {
	files, err := astmerge.ReadRelativeFileTree(src)
	if err != nil {
		return err
	}
	return astmerge.WriteRelativeFileTree(dst, files)
}

func multiFamilyMergeCallback(entry astmerge.TemplateExecutionPlanEntry) astmerge.MergeResult[string] {
	return asttemplate.MergePreparedContentFromRegistry(asttemplate.FamilyMergeAdapterRegistry{
		"markdown": markdownAdapter,
		"ruby":     rubyAdapter,
		"toml":     tomlAdapter,
	}, entry)
}

func markdownAdapter(entry astmerge.TemplateExecutionPlanEntry) astmerge.MergeResult[string] {
	destination := ""
	if entry.DestinationContent != nil {
		destination = *entry.DestinationContent
	}
	template := ""
	if entry.PreparedTemplateContent != nil {
		template = *entry.PreparedTemplateContent
	}
	return markdownmerge.MergeMarkdown(template, destination, "markdown")
}

func tomlAdapter(entry astmerge.TemplateExecutionPlanEntry) astmerge.MergeResult[string] {
	destination := ""
	if entry.DestinationContent != nil {
		destination = *entry.DestinationContent
	}
	template := ""
	if entry.PreparedTemplateContent != nil {
		template = *entry.PreparedTemplateContent
	}
	return tomlmerge.MergeTOML(template, destination, "toml")
}

func rubyAdapter(entry astmerge.TemplateExecutionPlanEntry) astmerge.MergeResult[string] {
	destination := ""
	if entry.DestinationContent != nil {
		destination = *entry.DestinationContent
	}
	template := ""
	if entry.PreparedTemplateContent != nil {
		template = *entry.PreparedTemplateContent
	}
	return rubymerge.MergeRuby(template, destination, "ruby")
}

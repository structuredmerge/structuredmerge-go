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
	t.Cleanup(func() {
		if err := os.RemoveAll(tempRoot); err != nil {
			t.Fatalf("remove temp root: %v", err)
		}
	})

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

func TestTemplateDirectoryAdapterCapabilityReportFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-356-template-directory-adapter-capability-report", "template-directory-adapter-capability-report.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)

	fullRegistry := asttemplate.FamilyMergeAdapterRegistry{
		"markdown": markdownAdapter,
		"ruby":     rubyAdapter,
		"toml":     tomlAdapter,
	}
	partialRegistry := asttemplate.FamilyMergeAdapterRegistry{
		"markdown": markdownAdapter,
		"toml":     tomlAdapter,
	}

	actual, err := asttemplate.ReportAdapterCapabilitiesFromDirectories(
		filepath.Join(fixtureRoot, "apply-run", "template"),
		filepath.Join(fixtureRoot, "apply-run", "destination"),
		decodeContext(t, fixture["full_registry"].(map[string]any)["context"]),
		decodeStrategy(t, fixture["full_registry"].(map[string]any)["default_strategy"]),
		decodeOverrides(t, fixture["full_registry"].(map[string]any)["overrides"]),
		decodeReplacements(t, fixture["full_registry"].(map[string]any)["replacements"]),
		fullRegistry,
		nil,
	)
	if err != nil {
		t.Fatalf("full registry capability report failed: %v", err)
	}
	assertJSONEqual(t, fixture["full_registry"].(map[string]any)["expected"], actual)

	actual, err = asttemplate.ReportAdapterCapabilitiesFromDirectories(
		filepath.Join(fixtureRoot, "apply-run", "template"),
		filepath.Join(fixtureRoot, "apply-run", "destination"),
		decodeContext(t, fixture["partial_registry"].(map[string]any)["context"]),
		decodeStrategy(t, fixture["partial_registry"].(map[string]any)["default_strategy"]),
		decodeOverrides(t, fixture["partial_registry"].(map[string]any)["overrides"]),
		decodeReplacements(t, fixture["partial_registry"].(map[string]any)["replacements"]),
		partialRegistry,
		nil,
	)
	if err != nil {
		t.Fatalf("partial registry capability report failed: %v", err)
	}
	assertJSONEqual(t, fixture["partial_registry"].(map[string]any)["expected"], actual)

	actual, err = asttemplate.ReportDefaultAdapterCapabilitiesFromDirectories(
		filepath.Join(fixtureRoot, "apply-run", "template"),
		filepath.Join(fixtureRoot, "apply-run", "destination"),
		decodeContext(t, fixture["filtered_discovery"].(map[string]any)["context"]),
		decodeStrategy(t, fixture["filtered_discovery"].(map[string]any)["default_strategy"]),
		decodeOverrides(t, fixture["filtered_discovery"].(map[string]any)["overrides"]),
		decodeReplacements(t, fixture["filtered_discovery"].(map[string]any)["replacements"]),
		decodeOptionalFamilies(t, fixture["filtered_discovery"].(map[string]any)["allowed_families"]),
		nil,
	)
	if err != nil {
		t.Fatalf("filtered discovery capability report failed: %v", err)
	}
	assertJSONEqual(t, fixture["filtered_discovery"].(map[string]any)["expected"], actual)
}

func TestTemplateDirectorySessionEnvelopeReportFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-357-template-directory-session-envelope-report", "template-directory-session-envelope-report.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)

	dryRun := fixture["dry_run"].(map[string]any)
	dryEnvelope, err := asttemplate.PlanTemplateDirectorySessionEnvelopeFromDirectories(
		filepath.Join(fixtureRoot, "dry-run", "template"),
		filepath.Join(fixtureRoot, "dry-run", "destination"),
		decodeContext(t, dryRun["context"]),
		decodeStrategy(t, dryRun["default_strategy"]),
		decodeOverrides(t, dryRun["overrides"]),
		decodeReplacements(t, dryRun["replacements"]),
		decodeOptionalFamilies(t, dryRun["allowed_families"]),
		nil,
	)
	if err != nil {
		t.Fatalf("dry-run envelope failed: %v", err)
	}
	assertJSONEqual(t, dryRun["expected"], dryEnvelope)

	for _, key := range []string{"apply_run", "filtered_discovery"} {
		section := fixture[key].(map[string]any)
		tempRoot := filepath.Join(repoRoot(t), "go", "asttemplate", "tmp", t.Name(), key)
		_ = os.RemoveAll(tempRoot)
		if err := copyTree(filepath.Join(fixtureRoot, "apply-run", "destination"), tempRoot); err != nil {
			t.Fatalf("copy destination: %v", err)
		}
		actual, err := asttemplate.ApplyTemplateDirectorySessionEnvelopeWithDefaultRegistryToDirectory(
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
			t.Fatalf("%s envelope failed: %v", key, err)
		}
		assertJSONEqual(t, section["expected"], actual)
		_ = os.RemoveAll(tempRoot)
	}
}

func TestTemplateDirectorySessionStatusReportFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-358-template-directory-session-status-report", "template-directory-session-status-report.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)

	dryRun := fixture["dry_run"].(map[string]any)
	dryEnvelope, err := asttemplate.PlanTemplateDirectorySessionEnvelopeFromDirectories(
		filepath.Join(fixtureRoot, "dry-run", "template"),
		filepath.Join(fixtureRoot, "dry-run", "destination"),
		decodeContext(t, dryRun["context"]),
		decodeStrategy(t, dryRun["default_strategy"]),
		decodeOverrides(t, dryRun["overrides"]),
		decodeReplacements(t, dryRun["replacements"]),
		decodeOptionalFamilies(t, dryRun["allowed_families"]),
		nil,
	)
	if err != nil {
		t.Fatalf("dry-run status failed: %v", err)
	}
	assertJSONEqual(t, dryRun["expected"], asttemplate.ReportTemplateDirectorySessionStatus(dryEnvelope))

	for _, key := range []string{"apply_run", "filtered_discovery"} {
		section := fixture[key].(map[string]any)
		tempRoot := filepath.Join(repoRoot(t), "go", "asttemplate", "tmp", t.Name(), key)
		_ = os.RemoveAll(tempRoot)
		if err := copyTree(filepath.Join(fixtureRoot, "apply-run", "destination"), tempRoot); err != nil {
			t.Fatalf("copy destination: %v", err)
		}
		envelope, err := asttemplate.ApplyTemplateDirectorySessionEnvelopeWithDefaultRegistryToDirectory(
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
			t.Fatalf("%s status failed: %v", key, err)
		}
		assertJSONEqual(t, section["expected"], asttemplate.ReportTemplateDirectorySessionStatus(envelope))
		_ = os.RemoveAll(tempRoot)
	}
}

func TestTemplateDirectorySessionDiagnosticsReportFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-359-template-directory-session-diagnostics-report", "template-directory-session-diagnostics-report.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)

	dryRun := fixture["dry_run"].(map[string]any)
	dryDiagnostics, err := asttemplate.PlanTemplateDirectorySessionDiagnosticsFromDirectories(
		filepath.Join(fixtureRoot, "dry-run", "template"),
		filepath.Join(fixtureRoot, "dry-run", "destination"),
		decodeContext(t, dryRun["context"]),
		decodeStrategy(t, dryRun["default_strategy"]),
		decodeOverrides(t, dryRun["overrides"]),
		decodeReplacements(t, dryRun["replacements"]),
		decodeOptionalFamilies(t, dryRun["allowed_families"]),
		nil,
	)
	if err != nil {
		t.Fatalf("dry-run diagnostics failed: %v", err)
	}
	assertJSONEqual(t, dryRun["expected"], dryDiagnostics)

	for _, key := range []string{"apply_run", "filtered_discovery"} {
		section := fixture[key].(map[string]any)
		tempRoot := filepath.Join(repoRoot(t), "go", "asttemplate", "tmp", t.Name(), key)
		_ = os.RemoveAll(tempRoot)
		if err := copyTree(filepath.Join(fixtureRoot, "apply-run", "destination"), tempRoot); err != nil {
			t.Fatalf("copy destination: %v", err)
		}
		actual, err := asttemplate.ApplyTemplateDirectorySessionDiagnosticsWithDefaultRegistryToDirectory(
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
			t.Fatalf("%s diagnostics failed: %v", key, err)
		}
		assertJSONEqual(t, section["expected"], actual)
		_ = os.RemoveAll(tempRoot)
	}
}

func TestTemplateDirectorySessionOutcomeReportFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-360-template-directory-session-outcome-report", "template-directory-session-outcome-report.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)

	dryRun := fixture["dry_run"].(map[string]any)
	dryOutcome, err := asttemplate.PlanTemplateDirectorySessionOutcomeFromDirectories(
		filepath.Join(fixtureRoot, "dry-run", "template"),
		filepath.Join(fixtureRoot, "dry-run", "destination"),
		decodeContext(t, dryRun["context"]),
		decodeStrategy(t, dryRun["default_strategy"]),
		decodeOverrides(t, dryRun["overrides"]),
		decodeReplacements(t, dryRun["replacements"]),
		decodeOptionalFamilies(t, dryRun["allowed_families"]),
		nil,
	)
	if err != nil {
		t.Fatalf("dry-run outcome failed: %v", err)
	}
	assertJSONEqual(t, dryRun["expected"], dryOutcome)

	for _, key := range []string{"apply_run", "filtered_discovery"} {
		section := fixture[key].(map[string]any)
		tempRoot := filepath.Join(repoRoot(t), "go", "asttemplate", "tmp", t.Name(), key)
		_ = os.RemoveAll(tempRoot)
		if err := copyTree(filepath.Join(fixtureRoot, "apply-run", "destination"), tempRoot); err != nil {
			t.Fatalf("copy destination: %v", err)
		}
		actual, err := asttemplate.ApplyTemplateDirectorySessionOutcomeWithDefaultRegistryToDirectory(
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
			t.Fatalf("%s outcome failed: %v", key, err)
		}
		assertJSONEqual(t, section["expected"], actual)
		_ = os.RemoveAll(tempRoot)
	}
}

func TestTemplateDirectorySessionRunnerReportFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-361-template-directory-session-runner-report", "template-directory-session-runner-report.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)

	planRun := fixture["plan_run"].(map[string]any)
	planOutcome, err := asttemplate.RunTemplateDirectorySessionWithDefaultRegistryToDirectory(
		asttemplate.DirectorySessionModePlan,
		filepath.Join(fixtureRoot, "dry-run", "template"),
		filepath.Join(fixtureRoot, "dry-run", "destination"),
		decodeContext(t, planRun["context"]),
		decodeStrategy(t, planRun["default_strategy"]),
		decodeOverrides(t, planRun["overrides"]),
		decodeReplacements(t, planRun["replacements"]),
		decodeOptionalFamilies(t, planRun["allowed_families"]),
		nil,
	)
	if err != nil {
		t.Fatalf("plan runner failed: %v", err)
	}
	assertJSONEqual(t, planRun["expected"], planOutcome)

	applyRun := fixture["apply_run"].(map[string]any)
	tempRoot := filepath.Join(repoRoot(t), "go", "asttemplate", "tmp", t.Name(), "runner")
	_ = os.RemoveAll(tempRoot)
	if err := copyTree(filepath.Join(fixtureRoot, "apply-run", "destination"), tempRoot); err != nil {
		t.Fatalf("copy destination: %v", err)
	}
	applyOutcome, err := asttemplate.RunTemplateDirectorySessionWithDefaultRegistryToDirectory(
		asttemplate.DirectorySessionModeApply,
		filepath.Join(fixtureRoot, "apply-run", "template"),
		tempRoot,
		decodeContext(t, applyRun["context"]),
		decodeStrategy(t, applyRun["default_strategy"]),
		decodeOverrides(t, applyRun["overrides"]),
		decodeReplacements(t, applyRun["replacements"]),
		decodeOptionalFamilies(t, applyRun["allowed_families"]),
		nil,
	)
	if err != nil {
		t.Fatalf("apply runner failed: %v", err)
	}
	assertJSONEqual(t, applyRun["expected"], applyOutcome)

	reapplyRun := fixture["reapply_run"].(map[string]any)
	reapplyOutcome, err := asttemplate.RunTemplateDirectorySessionWithDefaultRegistryToDirectory(
		asttemplate.DirectorySessionModeReapply,
		filepath.Join(fixtureRoot, "apply-run", "template"),
		tempRoot,
		decodeContext(t, reapplyRun["context"]),
		decodeStrategy(t, reapplyRun["default_strategy"]),
		decodeOverrides(t, reapplyRun["overrides"]),
		decodeReplacements(t, reapplyRun["replacements"]),
		decodeOptionalFamilies(t, reapplyRun["allowed_families"]),
		nil,
	)
	if err != nil {
		t.Fatalf("reapply runner failed: %v", err)
	}
	assertJSONEqual(t, reapplyRun["expected"], reapplyOutcome)
	_ = os.RemoveAll(tempRoot)
}

func TestTemplateDirectorySessionOptionsReportFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-362-template-directory-session-options-report", "template-directory-session-options-report.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)

	planRun := fixture["plan_run"].(map[string]any)
	planOptions := decodeSessionOptions(t, planRun["options"], filepath.Join(fixtureRoot, "dry-run", "template"), filepath.Join(fixtureRoot, "dry-run", "destination"))
	planOutcome, err := asttemplate.RunTemplateDirectorySessionWithOptions(planOptions)
	if err != nil {
		t.Fatalf("plan options failed: %v", err)
	}
	assertJSONEqual(t, planRun["expected"], planOutcome)

	tempRoot := filepath.Join(repoRoot(t), "go", "asttemplate", "tmp", t.Name(), "options")
	_ = os.RemoveAll(tempRoot)
	if err := copyTree(filepath.Join(fixtureRoot, "apply-run", "destination"), tempRoot); err != nil {
		t.Fatalf("copy destination: %v", err)
	}

	applyRun := fixture["apply_run"].(map[string]any)
	applyOptions := decodeSessionOptions(t, applyRun["options"], filepath.Join(fixtureRoot, "apply-run", "template"), tempRoot)
	applyOutcome, err := asttemplate.RunTemplateDirectorySessionWithOptions(applyOptions)
	if err != nil {
		t.Fatalf("apply options failed: %v", err)
	}
	assertJSONEqual(t, applyRun["expected"], applyOutcome)

	reapplyRun := fixture["reapply_run"].(map[string]any)
	reapplyOptions := decodeSessionOptions(t, reapplyRun["options"], filepath.Join(fixtureRoot, "apply-run", "template"), tempRoot)
	reapplyOutcome, err := asttemplate.RunTemplateDirectorySessionWithOptions(reapplyOptions)
	if err != nil {
		t.Fatalf("reapply options failed: %v", err)
	}
	assertJSONEqual(t, reapplyRun["expected"], reapplyOutcome)
	_ = os.RemoveAll(tempRoot)
}

func TestTemplateDirectorySessionProfileReportFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-363-template-directory-session-profile-report", "template-directory-session-profile-report.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)
	profiles := decodeSessionProfiles(t, fixture["profiles"])

	planRun := fixture["plan_run"].(map[string]any)
	planOutcome, err := asttemplate.RunTemplateDirectorySessionWithProfile(
		profiles,
		planRun["profile"].(string),
		asttemplate.DirectorySessionOptions{
			TemplateRoot:    filepath.Join(fixtureRoot, "dry-run", "template"),
			DestinationRoot: filepath.Join(fixtureRoot, "dry-run", "destination"),
		},
	)
	if err != nil {
		t.Fatalf("plan profile failed: %v", err)
	}
	assertJSONEqual(t, planRun["expected"], planOutcome)

	tempRoot := filepath.Join(repoRoot(t), "go", "asttemplate", "tmp", t.Name(), "profiles")
	_ = os.RemoveAll(tempRoot)
	if err := copyTree(filepath.Join(fixtureRoot, "apply-run", "destination"), tempRoot); err != nil {
		t.Fatalf("copy destination: %v", err)
	}

	applyRun := fixture["apply_run"].(map[string]any)
	applyOutcome, err := asttemplate.RunTemplateDirectorySessionWithProfile(
		profiles,
		applyRun["profile"].(string),
		asttemplate.DirectorySessionOptions{
			TemplateRoot:    filepath.Join(fixtureRoot, "apply-run", "template"),
			DestinationRoot: tempRoot,
		},
	)
	if err != nil {
		t.Fatalf("apply profile failed: %v", err)
	}
	assertJSONEqual(t, applyRun["expected"], applyOutcome)

	reapplyRun := fixture["reapply_run"].(map[string]any)
	reapplyOverrides := decodeSessionOptions(t, reapplyRun["overrides"], filepath.Join(fixtureRoot, "apply-run", "template"), tempRoot)
	reapplyOutcome, err := asttemplate.RunTemplateDirectorySessionWithProfile(
		profiles,
		reapplyRun["profile"].(string),
		reapplyOverrides,
	)
	if err != nil {
		t.Fatalf("reapply profile failed: %v", err)
	}
	assertJSONEqual(t, reapplyRun["expected"], reapplyOutcome)
	_ = os.RemoveAll(tempRoot)
}

func TestTemplateDirectorySessionConfigurationReportFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-364-template-directory-session-configuration-report", "template-directory-session-configuration-report.json")
	fixture := readJSONFixture(t, fixturePath)
	profiles := decodeSessionProfiles(t, fixture["profiles"])

	optionsValid := fixture["options_valid"].(map[string]any)
	assertJSONEqual(t, optionsValid["expected"], asttemplate.ReportTemplateDirectorySessionOptionsConfiguration(
		decodeSessionOptionsFromFixture(t, optionsValid["options"]),
	))

	optionsMissingRoots := fixture["options_missing_roots"].(map[string]any)
	assertJSONEqual(t, optionsMissingRoots["expected"], asttemplate.ReportTemplateDirectorySessionOptionsConfiguration(
		decodeSessionOptionsFromFixture(t, optionsMissingRoots["options"]),
	))

	profileValid := fixture["profile_valid"].(map[string]any)
	assertJSONEqual(t, profileValid["expected"], asttemplate.ReportTemplateDirectorySessionProfileConfiguration(
		profiles,
		profileValid["profile"].(string),
		decodeSessionOptionsFromFixture(t, profileValid["overrides"]),
	))

	profileMissingProfile := fixture["profile_missing_profile"].(map[string]any)
	assertJSONEqual(t, profileMissingProfile["expected"], asttemplate.ReportTemplateDirectorySessionProfileConfiguration(
		profiles,
		profileMissingProfile["profile"].(string),
		decodeSessionOptionsFromFixture(t, profileMissingProfile["overrides"]),
	))

	profileMissingRoots := fixture["profile_missing_roots"].(map[string]any)
	assertJSONEqual(t, profileMissingRoots["expected"], asttemplate.ReportTemplateDirectorySessionProfileConfiguration(
		profiles,
		profileMissingRoots["profile"].(string),
		decodeSessionOptionsFromFixture(t, profileMissingRoots["overrides"]),
	))
}

func TestTemplateDirectorySessionProfileConfigurationOutcomeReportFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-365-template-directory-session-profile-configuration-outcome-report", "template-directory-session-profile-configuration-outcome-report.json")
	fixture := readJSONFixture(t, fixturePath)
	profiles := decodeSessionProfiles(t, fixture["profiles"])

	missingProfile := fixture["missing_profile"].(map[string]any)
	missingProfileOutcome, err := asttemplate.RunTemplateDirectorySessionWithProfile(
		profiles,
		missingProfile["profile"].(string),
		decodeSessionOptionsFromFixture(t, missingProfile["overrides"]),
	)
	if err != nil {
		t.Fatalf("missing profile outcome failed: %v", err)
	}
	assertJSONEqual(t, missingProfile["expected"], missingProfileOutcome)

	missingRoots := fixture["missing_roots"].(map[string]any)
	missingRootsOutcome, err := asttemplate.RunTemplateDirectorySessionWithProfile(
		profiles,
		missingRoots["profile"].(string),
		decodeSessionOptionsFromFixture(t, missingRoots["overrides"]),
	)
	if err != nil {
		t.Fatalf("missing roots outcome failed: %v", err)
	}
	assertJSONEqual(t, missingRoots["expected"], missingRootsOutcome)
}

func TestTemplateDirectorySessionOptionsConfigurationOutcomeReportFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-366-template-directory-session-options-configuration-outcome-report", "template-directory-session-options-configuration-outcome-report.json")
	fixture := readJSONFixture(t, fixturePath)

	missingBothRoots := fixture["missing_both_roots"].(map[string]any)
	missingBothRootsOutcome, err := asttemplate.RunTemplateDirectorySessionWithOptions(
		decodeSessionOptionsFromFixture(t, missingBothRoots["options"]),
	)
	if err != nil {
		t.Fatalf("missing both roots outcome failed: %v", err)
	}
	assertJSONEqual(t, missingBothRoots["expected"], missingBothRootsOutcome)

	missingDestinationRoot := fixture["missing_destination_root"].(map[string]any)
	missingDestinationRootOutcome, err := asttemplate.RunTemplateDirectorySessionWithOptions(
		decodeSessionOptionsFromFixture(t, missingDestinationRoot["options"]),
	)
	if err != nil {
		t.Fatalf("missing destination root outcome failed: %v", err)
	}
	assertJSONEqual(t, missingDestinationRoot["expected"], missingDestinationRootOutcome)
}

func TestTemplateDirectorySessionRequestReportFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-367-template-directory-session-request-report", "template-directory-session-request-report.json")
	fixture := readJSONFixture(t, fixturePath)
	profiles := decodeSessionProfiles(t, fixture["profiles"])

	optionsValid := fixture["options_valid"].(map[string]any)
	assertJSONEqual(t, optionsValid["expected"], asttemplate.ReportTemplateDirectorySessionOptionsRequest(
		decodeSessionOptionsFromFixture(t, optionsValid["options"]),
	))

	optionsInvalid := fixture["options_invalid"].(map[string]any)
	assertJSONEqual(t, optionsInvalid["expected"], asttemplate.ReportTemplateDirectorySessionOptionsRequest(
		decodeSessionOptionsFromFixture(t, optionsInvalid["options"]),
	))

	profileValid := fixture["profile_valid"].(map[string]any)
	assertJSONEqual(t, profileValid["expected"], asttemplate.ReportTemplateDirectorySessionProfileRequest(
		profiles,
		profileValid["profile"].(string),
		decodeSessionOptionsFromFixture(t, profileValid["overrides"]),
	))

	profileInvalid := fixture["profile_invalid"].(map[string]any)
	assertJSONEqual(t, profileInvalid["expected"], asttemplate.ReportTemplateDirectorySessionProfileRequest(
		profiles,
		profileInvalid["profile"].(string),
		decodeSessionOptionsFromFixture(t, profileInvalid["overrides"]),
	))
}

func TestTemplateDirectorySessionRequestOutcomeReportFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-368-template-directory-session-request-outcome-report", "template-directory-session-request-outcome-report.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)

	optionsReady := fixture["options_ready"].(map[string]any)
	optionsReadyOutcome, err := asttemplate.RunTemplateDirectorySessionRequest(
		decodeSessionRequestReportFromFixture(t, optionsReady["request"], fixtureRoot),
	)
	if err != nil {
		t.Fatalf("options ready outcome failed: %v", err)
	}
	assertJSONEqual(t, optionsReady["expected"], optionsReadyOutcome)

	optionsBlocked := fixture["options_blocked"].(map[string]any)
	optionsBlockedOutcome, err := asttemplate.RunTemplateDirectorySessionRequest(
		decodeSessionRequestReportFromFixture(t, optionsBlocked["request"], fixtureRoot),
	)
	if err != nil {
		t.Fatalf("options blocked outcome failed: %v", err)
	}
	assertJSONEqual(t, optionsBlocked["expected"], optionsBlockedOutcome)

	profileReady := fixture["profile_ready"].(map[string]any)
	profileReadyOutcome, err := asttemplate.RunTemplateDirectorySessionRequest(
		decodeSessionRequestReportFromFixture(t, profileReady["request"], fixtureRoot),
	)
	if err != nil {
		t.Fatalf("profile ready outcome failed: %v", err)
	}
	assertJSONEqual(t, profileReady["expected"], profileReadyOutcome)

	profileBlocked := fixture["profile_blocked"].(map[string]any)
	profileBlockedOutcome, err := asttemplate.RunTemplateDirectorySessionRequest(
		decodeSessionRequestReportFromFixture(t, profileBlocked["request"], fixtureRoot),
	)
	if err != nil {
		t.Fatalf("profile blocked outcome failed: %v", err)
	}
	assertJSONEqual(t, profileBlocked["expected"], profileBlockedOutcome)
}

func TestTemplateDirectorySessionRequestRunnerReportFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-369-template-directory-session-request-runner-report", "template-directory-session-request-runner-report.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)
	profiles := decodeSessionProfiles(t, fixture["profiles"])

	optionsReady := fixture["options_ready"].(map[string]any)
	optionsReadyOutcome, err := asttemplate.RunTemplateDirectorySessionRunnerRequest(
		decodeSessionRunnerRequestFromFixture(t, optionsReady["request"], fixtureRoot),
		profiles,
	)
	if err != nil {
		t.Fatalf("options ready runner failed: %v", err)
	}
	assertJSONEqual(t, optionsReady["expected"], optionsReadyOutcome)

	optionsBlocked := fixture["options_blocked"].(map[string]any)
	optionsBlockedOutcome, err := asttemplate.RunTemplateDirectorySessionRunnerRequest(
		decodeSessionRunnerRequestFromFixture(t, optionsBlocked["request"], fixtureRoot),
		profiles,
	)
	if err != nil {
		t.Fatalf("options blocked runner failed: %v", err)
	}
	assertJSONEqual(t, optionsBlocked["expected"], optionsBlockedOutcome)

	profileReady := fixture["profile_ready"].(map[string]any)
	profileReadyOutcome, err := asttemplate.RunTemplateDirectorySessionRunnerRequest(
		decodeSessionRunnerRequestFromFixture(t, profileReady["request"], fixtureRoot),
		profiles,
	)
	if err != nil {
		t.Fatalf("profile ready runner failed: %v", err)
	}
	assertJSONEqual(t, profileReady["expected"], profileReadyOutcome)

	profileBlocked := fixture["profile_blocked"].(map[string]any)
	profileBlockedOutcome, err := asttemplate.RunTemplateDirectorySessionRunnerRequest(
		decodeSessionRunnerRequestFromFixture(t, profileBlocked["request"], fixtureRoot),
		profiles,
	)
	if err != nil {
		t.Fatalf("profile blocked runner failed: %v", err)
	}
	assertJSONEqual(t, profileBlocked["expected"], profileBlockedOutcome)
}

func TestTemplateDirectorySessionRequestTransportEnvelopeFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-404-template-directory-session-request-transport-envelope", "template-directory-session-request-envelope.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		request := decodeSessionRequestReportFromFixture(t, testCase["input"], fixtureRoot)
		expected := decodeSessionRequestEnvelopeFromFixture(t, testCase["expected_envelope"], fixtureRoot)

		envelope := asttemplate.SessionRequestEnvelopeFor(request)
		assertJSONEqual(t, envelope, expected)

		imported, importErr := asttemplate.ImportSessionRequestEnvelope(expected)
		if importErr != nil {
			t.Fatalf("%s request envelope import failed: %+v", testCase["label"], importErr)
		}
		if imported == nil {
			t.Fatalf("%s request envelope import returned nil request", testCase["label"])
		}

		assertJSONEqual(t, request, *imported)
	}
}

func TestTemplateDirectorySessionRunnerInputReportFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-370-template-directory-session-runner-input-report", "template-directory-session-runner-input-report.json")
	fixture := readJSONFixture(t, fixturePath)

	optionsReady := fixture["options_ready"].(map[string]any)
	assertJSONEqual(t, optionsReady["expected"], asttemplate.ReportTemplateDirectorySessionRunnerInput(
		decodeSessionRunnerInput(t, optionsReady["input"]),
	))

	optionsBlocked := fixture["options_blocked"].(map[string]any)
	assertJSONEqual(t, optionsBlocked["expected"], asttemplate.ReportTemplateDirectorySessionRunnerInput(
		decodeSessionRunnerInput(t, optionsBlocked["input"]),
	))

	profileReady := fixture["profile_ready"].(map[string]any)
	assertJSONEqual(t, profileReady["expected"], asttemplate.ReportTemplateDirectorySessionRunnerInput(
		decodeSessionRunnerInput(t, profileReady["input"]),
	))

	profileBlocked := fixture["profile_blocked"].(map[string]any)
	assertJSONEqual(t, profileBlocked["expected"], asttemplate.ReportTemplateDirectorySessionRunnerInput(
		decodeSessionRunnerInput(t, profileBlocked["input"]),
	))
}

func TestTemplateDirectorySessionRunnerPayloadReportFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-371-template-directory-session-runner-payload-report", "template-directory-session-runner-payload-report.json")
	fixture := readJSONFixture(t, fixturePath)

	optionsExplicit := fixture["options_explicit"].(map[string]any)
	assertJSONEqual(t, optionsExplicit["expected"], asttemplate.ReportTemplateDirectorySessionRunnerPayload(
		decodeSessionRunnerPayload(t, optionsExplicit["input"]),
	))

	optionsInferred := fixture["options_inferred"].(map[string]any)
	assertJSONEqual(t, optionsInferred["expected"], asttemplate.ReportTemplateDirectorySessionRunnerPayload(
		decodeSessionRunnerPayload(t, optionsInferred["input"]),
	))

	profileDefaultName := fixture["profile_default_name"].(map[string]any)
	assertJSONEqual(t, profileDefaultName["expected"], asttemplate.ReportTemplateDirectorySessionRunnerPayload(
		decodeSessionRunnerPayload(t, profileDefaultName["input"]),
	))

	profileExplicitName := fixture["profile_explicit_name"].(map[string]any)
	assertJSONEqual(t, profileExplicitName["expected"], asttemplate.ReportTemplateDirectorySessionRunnerPayload(
		decodeSessionRunnerPayload(t, profileExplicitName["input"]),
	))
}

func TestTemplateDirectorySessionRunnerPayloadOutcomeReportFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-372-template-directory-session-runner-payload-outcome-report", "template-directory-session-runner-payload-outcome-report.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)
	profiles := decodeSessionProfiles(t, fixture["profiles"])

	optionsReady := fixture["options_ready"].(map[string]any)
	optionsReadyOutcome, err := asttemplate.RunTemplateDirectorySessionRunnerPayload(
		decodeSessionRunnerPayloadFromFixture(t, optionsReady["payload"], fixtureRoot),
		profiles,
	)
	if err != nil {
		t.Fatalf("options ready payload runner failed: %v", err)
	}
	assertJSONEqual(t, optionsReady["expected"], optionsReadyOutcome)

	optionsBlocked := fixture["options_blocked"].(map[string]any)
	optionsBlockedOutcome, err := asttemplate.RunTemplateDirectorySessionRunnerPayload(
		decodeSessionRunnerPayloadFromFixture(t, optionsBlocked["payload"], fixtureRoot),
		profiles,
	)
	if err != nil {
		t.Fatalf("options blocked payload runner failed: %v", err)
	}
	assertJSONEqual(t, optionsBlocked["expected"], optionsBlockedOutcome)

	profileReady := fixture["profile_ready"].(map[string]any)
	profileReadyOutcome, err := asttemplate.RunTemplateDirectorySessionRunnerPayload(
		decodeSessionRunnerPayloadFromFixture(t, profileReady["payload"], fixtureRoot),
		profiles,
	)
	if err != nil {
		t.Fatalf("profile ready payload runner failed: %v", err)
	}
	assertJSONEqual(t, profileReady["expected"], profileReadyOutcome)

	profileBlocked := fixture["profile_blocked"].(map[string]any)
	profileBlockedOutcome, err := asttemplate.RunTemplateDirectorySessionRunnerPayload(
		decodeSessionRunnerPayloadFromFixture(t, profileBlocked["payload"], fixtureRoot),
		profiles,
	)
	if err != nil {
		t.Fatalf("profile blocked payload runner failed: %v", err)
	}
	assertJSONEqual(t, profileBlocked["expected"], profileBlockedOutcome)
}

func TestTemplateDirectorySessionRunnerPayloadTransportEnvelopeFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-401-template-directory-session-runner-payload-transport-envelope", "template-directory-session-runner-payload-envelope.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		payload := decodeSessionRunnerPayloadFromFixture(t, testCase["input"], fixtureRoot)
		expected := decodeSessionRunnerPayloadEnvelopeFromFixture(t, testCase["expected_envelope"], fixtureRoot)

		envelope := asttemplate.SessionRunnerPayloadEnvelopeFor(payload)
		assertJSONEqual(t, envelope, expected)

		imported, importErr := asttemplate.ImportSessionRunnerPayloadEnvelope(expected)
		if importErr != nil {
			t.Fatalf("%s runner payload envelope import failed: %+v", testCase["label"], importErr)
		}
		if imported == nil {
			t.Fatalf("%s runner payload envelope import returned nil payload", testCase["label"])
		}

		assertJSONEqual(t, payload, *imported)
	}
}

func TestTemplateDirectorySessionRunnerPayloadTransportRejectionFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-402-template-directory-session-runner-payload-transport-rejection", "template-directory-session-runner-payload-envelope-rejection.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeSessionRunnerPayloadEnvelopeFromFixture(t, testCase["envelope"], fixtureRoot)
		expected := decodeSessionRunnerPayloadTransportImportErrorFromFixture(testCase["expected_error"])

		imported, importErr := asttemplate.ImportSessionRunnerPayloadEnvelope(envelope)
		if imported != nil {
			t.Fatalf("%s runner payload rejection unexpectedly imported %+v", testCase["label"], imported)
		}
		if !reflect.DeepEqual(importErr, expected) {
			t.Fatalf("%s runner payload rejection mismatch: got %+v want %+v", testCase["label"], importErr, expected)
		}
	}
}

func TestTemplateDirectorySessionRunnerPayloadEnvelopeApplicationFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-403-template-directory-session-runner-payload-envelope-application", "template-directory-session-runner-payload-envelope-application.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)
	profiles := decodeSessionProfiles(t, fixture["profiles"])

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeSessionRunnerPayloadEnvelopeFromFixture(t, testCase["envelope"], fixtureRoot)
		imported, importErr := asttemplate.ImportSessionRunnerPayloadEnvelope(envelope)
		if importErr != nil {
			t.Fatalf("%s runner payload envelope import failed: %+v", testCase["label"], importErr)
		}
		actual, err := asttemplate.RunTemplateDirectorySessionRunnerPayload(*imported, profiles)
		if err != nil {
			t.Fatalf("%s runner payload envelope application failed: %v", testCase["label"], err)
		}
		assertJSONEqual(t, resolveSessionOutcomeExpectedPaths(testCase["expected"], fixtureRoot), actual)
	}

	for _, rawCase := range fixture["rejections"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeSessionRunnerPayloadEnvelopeFromFixture(t, testCase["envelope"], fixtureRoot)
		expected := decodeSessionRunnerPayloadTransportImportErrorFromFixture(testCase["expected_error"])
		imported, importErr := asttemplate.ImportSessionRunnerPayloadEnvelope(envelope)
		if imported != nil {
			t.Fatalf("%s runner payload envelope rejection unexpectedly imported %+v", testCase["label"], imported)
		}
		if !reflect.DeepEqual(importErr, expected) {
			t.Fatalf("%s runner payload envelope rejection mismatch: got %+v want %+v", testCase["label"], importErr, expected)
		}
	}
}

func TestTemplateDirectorySessionEntrypointOutcomeReportFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-373-template-directory-session-entrypoint-outcome-report", "template-directory-session-entrypoint-outcome-report.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)
	profiles := decodeSessionProfiles(t, fixture["profiles"])

	payloadReady := fixture["payload_ready"].(map[string]any)
	payloadReadyOutcome, err := asttemplate.RunTemplateDirectorySessionEntrypoint(
		decodeSessionEntrypointFromFixture(t, payloadReady["input"], fixtureRoot),
		profiles,
	)
	if err != nil {
		t.Fatalf("payload ready entrypoint failed: %v", err)
	}
	assertJSONEqual(t, payloadReady["expected"], payloadReadyOutcome)

	requestBlocked := fixture["request_blocked"].(map[string]any)
	requestBlockedOutcome, err := asttemplate.RunTemplateDirectorySessionEntrypoint(
		decodeSessionEntrypointFromFixture(t, requestBlocked["input"], fixtureRoot),
		profiles,
	)
	if err != nil {
		t.Fatalf("request blocked entrypoint failed: %v", err)
	}
	assertJSONEqual(t, requestBlocked["expected"], requestBlockedOutcome)

	requestReady := fixture["request_ready"].(map[string]any)
	requestReadyOutcome, err := asttemplate.RunTemplateDirectorySessionEntrypoint(
		decodeSessionEntrypointFromFixture(t, requestReady["input"], fixtureRoot),
		profiles,
	)
	if err != nil {
		t.Fatalf("request ready entrypoint failed: %v", err)
	}
	assertJSONEqual(t, requestReady["expected"], requestReadyOutcome)

	payloadBlocked := fixture["payload_blocked"].(map[string]any)
	payloadBlockedOutcome, err := asttemplate.RunTemplateDirectorySessionEntrypoint(
		decodeSessionEntrypointFromFixture(t, payloadBlocked["input"], fixtureRoot),
		profiles,
	)
	if err != nil {
		t.Fatalf("payload blocked entrypoint failed: %v", err)
	}
	assertJSONEqual(t, payloadBlocked["expected"], payloadBlockedOutcome)
}

func TestTemplateDirectorySessionEntrypointReportFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-374-template-directory-session-entrypoint-report", "template-directory-session-entrypoint-report.json")
	fixture := readJSONFixture(t, fixturePath)

	payloadReady := fixture["payload_ready"].(map[string]any)
	assertJSONEqual(t, payloadReady["expected"], asttemplate.ReportTemplateDirectorySessionEntrypoint(
		decodeSessionEntrypoint(t, payloadReady["input"]),
	))

	requestBlocked := fixture["request_blocked"].(map[string]any)
	assertJSONEqual(t, requestBlocked["expected"], asttemplate.ReportTemplateDirectorySessionEntrypoint(
		decodeSessionEntrypoint(t, requestBlocked["input"]),
	))

	requestReady := fixture["request_ready"].(map[string]any)
	assertJSONEqual(t, requestReady["expected"], asttemplate.ReportTemplateDirectorySessionEntrypoint(
		decodeSessionEntrypoint(t, requestReady["input"]),
	))

	payloadBlocked := fixture["payload_blocked"].(map[string]any)
	assertJSONEqual(t, payloadBlocked["expected"], asttemplate.ReportTemplateDirectorySessionEntrypoint(
		decodeSessionEntrypoint(t, payloadBlocked["input"]),
	))
}

func TestTemplateDirectorySessionResolutionReportFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-375-template-directory-session-resolution-report", "template-directory-session-resolution-report.json")
	fixture := readJSONFixture(t, fixturePath)
	profiles := decodeSessionProfiles(t, fixture["profiles"])

	payloadReady := fixture["payload_ready"].(map[string]any)
	assertJSONEqual(t, payloadReady["expected"], asttemplate.ReportTemplateDirectorySessionResolution(
		decodeSessionEntrypoint(t, payloadReady["input"]),
		profiles,
	))

	requestBlocked := fixture["request_blocked"].(map[string]any)
	assertJSONEqual(t, requestBlocked["expected"], asttemplate.ReportTemplateDirectorySessionResolution(
		decodeSessionEntrypoint(t, requestBlocked["input"]),
		profiles,
	))

	requestReady := fixture["request_ready"].(map[string]any)
	assertJSONEqual(t, requestReady["expected"], asttemplate.ReportTemplateDirectorySessionResolution(
		decodeSessionEntrypoint(t, requestReady["input"]),
		profiles,
	))

	payloadBlocked := fixture["payload_blocked"].(map[string]any)
	assertJSONEqual(t, payloadBlocked["expected"], asttemplate.ReportTemplateDirectorySessionResolution(
		decodeSessionEntrypoint(t, payloadBlocked["input"]),
		profiles,
	))
}

func TestTemplateDirectorySessionInspectionReportFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-376-template-directory-session-inspection-report", "template-directory-session-inspection-report.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)
	profiles := decodeSessionProfiles(t, fixture["profiles"])

	payloadReady := fixture["payload_ready"].(map[string]any)
	payloadReadyActual, err := asttemplate.ReportTemplateDirectorySessionInspection(
		decodeSessionEntrypointFromFixture(t, payloadReady["input"], fixtureRoot),
		profiles,
	)
	if err != nil {
		t.Fatalf("payload ready inspection failed: %v", err)
	}
	assertJSONEqual(t, resolveSessionInspectionExpectedFixturePaths(payloadReady["expected"], fixtureRoot), payloadReadyActual)

	requestBlocked := fixture["request_blocked"].(map[string]any)
	requestBlockedActual, err := asttemplate.ReportTemplateDirectorySessionInspection(
		decodeSessionEntrypointFromFixture(t, requestBlocked["input"], fixtureRoot),
		profiles,
	)
	if err != nil {
		t.Fatalf("request blocked inspection failed: %v", err)
	}
	assertJSONEqual(t, resolveSessionInspectionExpectedFixturePaths(requestBlocked["expected"], fixtureRoot), requestBlockedActual)

	requestReady := fixture["request_ready"].(map[string]any)
	requestReadyActual, err := asttemplate.ReportTemplateDirectorySessionInspection(
		decodeSessionEntrypointFromFixture(t, requestReady["input"], fixtureRoot),
		profiles,
	)
	if err != nil {
		t.Fatalf("request ready inspection failed: %v", err)
	}
	assertJSONEqual(t, resolveSessionInspectionExpectedFixturePaths(requestReady["expected"], fixtureRoot), requestReadyActual)

	payloadBlocked := fixture["payload_blocked"].(map[string]any)
	payloadBlockedActual, err := asttemplate.ReportTemplateDirectorySessionInspection(
		decodeSessionEntrypointFromFixture(t, payloadBlocked["input"], fixtureRoot),
		profiles,
	)
	if err != nil {
		t.Fatalf("payload blocked inspection failed: %v", err)
	}
	assertJSONEqual(t, resolveSessionInspectionExpectedFixturePaths(payloadBlocked["expected"], fixtureRoot), payloadBlockedActual)
}

func TestTemplateDirectorySessionDispatchReportFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-377-template-directory-session-dispatch-report", "template-directory-session-dispatch-report.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)
	profiles := decodeSessionProfiles(t, fixture["profiles"])

	inspectPayloadReady := fixture["inspect_payload_ready"].(map[string]any)
	inspectPayloadReadyInput := decodeSessionDispatchInputFromFixture(t, inspectPayloadReady["input"], fixtureRoot)
	inspectPayloadReadyActual, err := asttemplate.RunTemplateDirectorySessionDispatch(
		inspectPayloadReadyInput["operation"].(string),
		inspectPayloadReadyInput["entrypoint"].(asttemplate.SessionEntrypoint),
		profiles,
	)
	if err != nil {
		t.Fatalf("inspect payload ready dispatch failed: %v", err)
	}
	assertJSONEqual(t, resolveSessionDispatchExpectedFixturePaths(inspectPayloadReady["expected"], fixtureRoot), inspectPayloadReadyActual)

	inspectRequestBlocked := fixture["inspect_request_blocked"].(map[string]any)
	inspectRequestBlockedInput := decodeSessionDispatchInputFromFixture(t, inspectRequestBlocked["input"], fixtureRoot)
	inspectRequestBlockedActual, err := asttemplate.RunTemplateDirectorySessionDispatch(
		inspectRequestBlockedInput["operation"].(string),
		inspectRequestBlockedInput["entrypoint"].(asttemplate.SessionEntrypoint),
		profiles,
	)
	if err != nil {
		t.Fatalf("inspect request blocked dispatch failed: %v", err)
	}
	assertJSONEqual(t, resolveSessionDispatchExpectedFixturePaths(inspectRequestBlocked["expected"], fixtureRoot), inspectRequestBlockedActual)

	runRequestReady := fixture["run_request_ready"].(map[string]any)
	runRequestReadyInput := decodeSessionDispatchInputFromFixture(t, runRequestReady["input"], fixtureRoot)
	runRequestReadyActual, err := asttemplate.RunTemplateDirectorySessionDispatch(
		runRequestReadyInput["operation"].(string),
		runRequestReadyInput["entrypoint"].(asttemplate.SessionEntrypoint),
		profiles,
	)
	if err != nil {
		t.Fatalf("run request ready dispatch failed: %v", err)
	}
	assertJSONEqual(t, resolveSessionDispatchExpectedFixturePaths(runRequestReady["expected"], fixtureRoot), runRequestReadyActual)

	runPayloadBlocked := fixture["run_payload_blocked"].(map[string]any)
	runPayloadBlockedInput := decodeSessionDispatchInputFromFixture(t, runPayloadBlocked["input"], fixtureRoot)
	runPayloadBlockedActual, err := asttemplate.RunTemplateDirectorySessionDispatch(
		runPayloadBlockedInput["operation"].(string),
		runPayloadBlockedInput["entrypoint"].(asttemplate.SessionEntrypoint),
		profiles,
	)
	if err != nil {
		t.Fatalf("run payload blocked dispatch failed: %v", err)
	}
	assertJSONEqual(t, resolveSessionDispatchExpectedFixturePaths(runPayloadBlocked["expected"], fixtureRoot), runPayloadBlockedActual)
}

func TestTemplateDirectorySessionCommandReportFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-378-template-directory-session-command-report", "template-directory-session-command-report.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)
	profiles := decodeSessionProfiles(t, fixture["profiles"])

	for _, key := range []string{"inspect_payload_ready", "run_request_ready", "run_payload_blocked"} {
		section := fixture[key].(map[string]any)
		command := decodeSessionCommandFromFixture(t, section["input"], fixtureRoot)
		actual, err := asttemplate.RunTemplateDirectorySessionCommand(command, profiles)
		if err != nil {
			t.Fatalf("%s command failed: %v", key, err)
		}
		assertJSONEqual(t, resolveSessionDispatchExpectedFixturePaths(section["expected"], fixtureRoot), actual)
	}
}

func TestTemplateDirectorySessionCommandPayloadReportFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-379-template-directory-session-command-payload-report", "template-directory-session-command-payload-report.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)
	profiles := decodeSessionProfiles(t, fixture["profiles"])

	for _, key := range []string{"inspect_ready", "run_profile_ready", "run_profile_blocked"} {
		section := fixture[key].(map[string]any)
		command := decodeSessionCommandPayloadFromFixture(t, section["input"], fixtureRoot)
		actual, err := asttemplate.RunTemplateDirectorySessionCommandPayload(command, profiles)
		if err != nil {
			t.Fatalf("%s command payload failed: %v", key, err)
		}
		assertJSONEqual(t, resolveSessionDispatchExpectedFixturePaths(section["expected"], fixtureRoot), actual)
	}
}

func TestTemplateDirectorySessionDispatchRejectionFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-380-template-directory-session-dispatch-rejection", "template-directory-session-dispatch-rejection.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)
	cases := fixture["cases"].([]any)

	for _, rawCase := range cases {
		testCase := rawCase.(map[string]any)
		input := decodeSessionDispatchInputFromFixture(t, testCase["input"], fixtureRoot)
		_, err := asttemplate.RunTemplateDirectorySessionDispatch(
			input["operation"].(string),
			input["entrypoint"].(asttemplate.SessionEntrypoint),
			nil,
		)
		if err == nil {
			t.Fatalf("%s dispatch rejection expected error", testCase["label"])
		}
		if err.Error() != testCase["expected_error"].(string) {
			t.Fatalf("%s dispatch rejection error mismatch: got %q want %q", testCase["label"], err.Error(), testCase["expected_error"])
		}
	}
}

func TestTemplateDirectorySessionCommandRejectionFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-381-template-directory-session-command-rejection", "template-directory-session-command-rejection.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)
	cases := fixture["cases"].([]any)

	for _, rawCase := range cases {
		testCase := rawCase.(map[string]any)
		command := decodeSessionCommandFromFixture(t, testCase["input"], fixtureRoot)
		_, err := asttemplate.RunTemplateDirectorySessionCommand(command, nil)
		if err == nil {
			t.Fatalf("%s command rejection expected error", testCase["label"])
		}
		if err.Error() != testCase["expected_error"].(string) {
			t.Fatalf("%s command rejection error mismatch: got %q want %q", testCase["label"], err.Error(), testCase["expected_error"])
		}
	}
}

func TestTemplateDirectorySessionCommandPayloadRejectionFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-382-template-directory-session-command-payload-rejection", "template-directory-session-command-payload-rejection.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)
	cases := fixture["cases"].([]any)

	for _, rawCase := range cases {
		testCase := rawCase.(map[string]any)
		command := decodeSessionCommandPayloadFromFixture(t, testCase["input"], fixtureRoot)
		_, err := asttemplate.RunTemplateDirectorySessionCommandPayload(command, nil)
		if err == nil {
			t.Fatalf("%s command payload rejection expected error", testCase["label"])
		}
		if err.Error() != testCase["expected_error"].(string) {
			t.Fatalf("%s command payload rejection error mismatch: got %q want %q", testCase["label"], err.Error(), testCase["expected_error"])
		}
	}
}

func TestTemplateDirectorySessionCommandTransportEnvelopeFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-389-template-directory-session-command-transport-envelope", "template-directory-session-command-envelope.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		command := decodeSessionCommandFromFixture(t, testCase["input"], fixtureRoot)
		expected := decodeSessionCommandEnvelopeFromFixture(t, testCase["expected_envelope"], fixtureRoot)

		envelope := asttemplate.SessionCommandEnvelopeFor(command)
		if !reflect.DeepEqual(envelope, expected) {
			t.Fatalf("%s command envelope mismatch: got %+v want %+v", testCase["label"], envelope, expected)
		}

		imported, importErr := asttemplate.ImportSessionCommandEnvelope(expected)
		if importErr != nil {
			t.Fatalf("%s command envelope import failed: %+v", testCase["label"], importErr)
		}
		if imported == nil {
			t.Fatalf("%s command envelope import returned nil command", testCase["label"])
		}

		assertJSONEqual(t, command, *imported)
	}
}

func TestTemplateDirectorySessionCommandTransportRejectionFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-390-template-directory-session-command-transport-rejection", "template-directory-session-command-envelope-rejection.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeSessionCommandEnvelopeFromFixture(t, testCase["envelope"], fixtureRoot)
		expected := decodeSessionCommandTransportImportErrorFromFixture(testCase["expected_error"])

		imported, importErr := asttemplate.ImportSessionCommandEnvelope(envelope)
		if imported != nil {
			t.Fatalf("%s command transport rejection unexpectedly imported %+v", testCase["label"], imported)
		}
		if !reflect.DeepEqual(importErr, expected) {
			t.Fatalf("%s command transport rejection mismatch: got %+v want %+v", testCase["label"], importErr, expected)
		}
	}
}

func TestTemplateDirectorySessionCommandPayloadTransportEnvelopeFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-392-template-directory-session-command-payload-transport-envelope", "template-directory-session-command-payload-envelope.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		payload := decodeSessionCommandPayloadFromFixture(t, testCase["input"], fixtureRoot)
		expected := decodeSessionCommandPayloadEnvelopeFromFixture(t, testCase["expected_envelope"], fixtureRoot)

		envelope := asttemplate.SessionCommandPayloadEnvelopeFor(payload)
		if !reflect.DeepEqual(envelope, expected) {
			t.Fatalf("%s command payload envelope mismatch: got %+v want %+v", testCase["label"], envelope, expected)
		}

		imported, importErr := asttemplate.ImportSessionCommandPayloadEnvelope(expected)
		if importErr != nil {
			t.Fatalf("%s command payload envelope import failed: %+v", testCase["label"], importErr)
		}
		if imported == nil {
			t.Fatalf("%s command payload envelope import returned nil payload", testCase["label"])
		}

		assertJSONEqual(t, payload, *imported)
	}
}

func TestTemplateDirectorySessionCommandPayloadTransportRejectionFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-393-template-directory-session-command-payload-transport-rejection", "template-directory-session-command-payload-envelope-rejection.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeSessionCommandPayloadEnvelopeFromFixture(t, testCase["envelope"], fixtureRoot)
		expected := decodeSessionCommandPayloadTransportImportErrorFromFixture(testCase["expected_error"])

		imported, importErr := asttemplate.ImportSessionCommandPayloadEnvelope(envelope)
		if imported != nil {
			t.Fatalf("%s command payload rejection unexpectedly imported %+v", testCase["label"], imported)
		}
		if !reflect.DeepEqual(importErr, expected) {
			t.Fatalf("%s command payload rejection mismatch: got %+v want %+v", testCase["label"], importErr, expected)
		}
	}
}

func TestTemplateDirectorySessionEntrypointTransportEnvelopeFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-395-template-directory-session-entrypoint-transport-envelope", "template-directory-session-entrypoint-envelope.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		entrypoint := decodeSessionEntrypointFromFixture(t, testCase["input"], fixtureRoot)
		expected := decodeSessionEntrypointEnvelopeFromFixture(t, testCase["expected_envelope"], fixtureRoot)

		envelope := asttemplate.SessionEntrypointEnvelopeFor(entrypoint)
		assertJSONEqual(t, envelope, expected)

		imported, importErr := asttemplate.ImportSessionEntrypointEnvelope(expected)
		if importErr != nil {
			t.Fatalf("%s entrypoint envelope import failed: %+v", testCase["label"], importErr)
		}
		if imported == nil {
			t.Fatalf("%s entrypoint envelope import returned nil entrypoint", testCase["label"])
		}

		assertJSONEqual(t, entrypoint, *imported)
	}
}

func TestTemplateDirectorySessionRunnerRequestTransportEnvelopeFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-398-template-directory-session-runner-request-transport-envelope", "template-directory-session-runner-request-envelope.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		request := decodeSessionRunnerRequestFromFixture(t, testCase["input"], fixtureRoot)
		expected := decodeSessionRunnerRequestEnvelopeFromFixture(t, testCase["expected_envelope"], fixtureRoot)

		envelope := asttemplate.SessionRunnerRequestEnvelopeFor(request)
		assertJSONEqual(t, envelope, expected)

		imported, importErr := asttemplate.ImportSessionRunnerRequestEnvelope(expected)
		if importErr != nil {
			t.Fatalf("%s runner request envelope import failed: %+v", testCase["label"], importErr)
		}
		if imported == nil {
			t.Fatalf("%s runner request envelope import returned nil request", testCase["label"])
		}

		assertJSONEqual(t, request, *imported)
	}
}

func TestTemplateDirectorySessionRunnerRequestTransportRejectionFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-399-template-directory-session-runner-request-transport-rejection", "template-directory-session-runner-request-envelope-rejection.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeSessionRunnerRequestEnvelopeFromFixture(t, testCase["envelope"], fixtureRoot)
		expected := decodeSessionRunnerRequestTransportImportErrorFromFixture(testCase["expected_error"])

		imported, importErr := asttemplate.ImportSessionRunnerRequestEnvelope(envelope)
		if imported != nil {
			t.Fatalf("%s runner request rejection unexpectedly imported %+v", testCase["label"], imported)
		}
		if !reflect.DeepEqual(importErr, expected) {
			t.Fatalf("%s runner request rejection mismatch: got %+v want %+v", testCase["label"], importErr, expected)
		}
	}
}

func TestTemplateDirectorySessionEntrypointTransportRejectionFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-396-template-directory-session-entrypoint-transport-rejection", "template-directory-session-entrypoint-envelope-rejection.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeSessionEntrypointEnvelopeFromFixture(t, testCase["envelope"], fixtureRoot)
		expected := decodeSessionEntrypointTransportImportErrorFromFixture(testCase["expected_error"])

		imported, importErr := asttemplate.ImportSessionEntrypointEnvelope(envelope)
		if imported != nil {
			t.Fatalf("%s entrypoint rejection unexpectedly imported %+v", testCase["label"], imported)
		}
		if !reflect.DeepEqual(importErr, expected) {
			t.Fatalf("%s entrypoint rejection mismatch: got %+v want %+v", testCase["label"], importErr, expected)
		}
	}
}

func TestTemplateDirectorySessionCommandEnvelopeApplicationFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-391-template-directory-session-command-envelope-application", "template-directory-session-command-envelope-application.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)
	profiles := decodeSessionProfiles(t, fixture["profiles"])

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeSessionCommandEnvelopeFromFixture(t, testCase["envelope"], fixtureRoot)
		imported, importErr := asttemplate.ImportSessionCommandEnvelope(envelope)
		if importErr != nil {
			t.Fatalf("%s command envelope import failed: %+v", testCase["label"], importErr)
		}
		actual, err := asttemplate.RunTemplateDirectorySessionCommand(*imported, profiles)
		if err != nil {
			t.Fatalf("%s command envelope application failed: %v", testCase["label"], err)
		}
		assertJSONEqual(t, resolveSessionDispatchExpectedFixturePaths(testCase["expected"], fixtureRoot), actual)
	}

	for _, rawCase := range fixture["rejections"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeSessionCommandEnvelopeFromFixture(t, testCase["envelope"], fixtureRoot)
		expected := decodeSessionCommandTransportImportErrorFromFixture(testCase["expected_error"])
		imported, importErr := asttemplate.ImportSessionCommandEnvelope(envelope)
		if imported != nil {
			t.Fatalf("%s command envelope rejection unexpectedly imported %+v", testCase["label"], imported)
		}
		if !reflect.DeepEqual(importErr, expected) {
			t.Fatalf("%s command envelope rejection mismatch: got %+v want %+v", testCase["label"], importErr, expected)
		}
	}
}

func TestTemplateDirectorySessionEntrypointEnvelopeApplicationFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-397-template-directory-session-entrypoint-envelope-application", "template-directory-session-entrypoint-envelope-application.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)
	profiles := decodeSessionProfiles(t, fixture["profiles"])

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeSessionEntrypointEnvelopeFromFixture(t, testCase["envelope"], fixtureRoot)
		imported, importErr := asttemplate.ImportSessionEntrypointEnvelope(envelope)
		if importErr != nil {
			t.Fatalf("%s entrypoint envelope import failed: %+v", testCase["label"], importErr)
		}
		actual, err := asttemplate.RunTemplateDirectorySessionEntrypoint(*imported, profiles)
		if err != nil {
			t.Fatalf("%s entrypoint envelope application failed: %v", testCase["label"], err)
		}
		assertJSONEqual(t, resolveSessionOutcomeExpectedPaths(testCase["expected"], fixtureRoot), actual)
	}

	for _, rawCase := range fixture["rejections"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeSessionEntrypointEnvelopeFromFixture(t, testCase["envelope"], fixtureRoot)
		expected := decodeSessionEntrypointTransportImportErrorFromFixture(testCase["expected_error"])
		imported, importErr := asttemplate.ImportSessionEntrypointEnvelope(envelope)
		if imported != nil {
			t.Fatalf("%s entrypoint envelope rejection unexpectedly imported %+v", testCase["label"], imported)
		}
		if !reflect.DeepEqual(importErr, expected) {
			t.Fatalf("%s entrypoint envelope rejection mismatch: got %+v want %+v", testCase["label"], importErr, expected)
		}
	}
}

func TestTemplateDirectorySessionRunnerRequestEnvelopeApplicationFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-400-template-directory-session-runner-request-envelope-application", "template-directory-session-runner-request-envelope-application.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)
	profiles := decodeSessionProfiles(t, fixture["profiles"])

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeSessionRunnerRequestEnvelopeFromFixture(t, testCase["envelope"], fixtureRoot)
		imported, importErr := asttemplate.ImportSessionRunnerRequestEnvelope(envelope)
		if importErr != nil {
			t.Fatalf("%s runner request envelope import failed: %+v", testCase["label"], importErr)
		}
		actual, err := asttemplate.RunTemplateDirectorySessionRunnerRequest(*imported, profiles)
		if err != nil {
			t.Fatalf("%s runner request envelope application failed: %v", testCase["label"], err)
		}
		assertJSONEqual(t, resolveSessionOutcomeExpectedPaths(testCase["expected"], fixtureRoot), actual)
	}

	for _, rawCase := range fixture["rejections"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeSessionRunnerRequestEnvelopeFromFixture(t, testCase["envelope"], fixtureRoot)
		expected := decodeSessionRunnerRequestTransportImportErrorFromFixture(testCase["expected_error"])
		imported, importErr := asttemplate.ImportSessionRunnerRequestEnvelope(envelope)
		if imported != nil {
			t.Fatalf("%s runner request envelope rejection unexpectedly imported %+v", testCase["label"], imported)
		}
		if !reflect.DeepEqual(importErr, expected) {
			t.Fatalf("%s runner request envelope rejection mismatch: got %+v want %+v", testCase["label"], importErr, expected)
		}
	}
}

func TestTemplateDirectorySessionCommandPayloadEnvelopeApplicationFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-394-template-directory-session-command-payload-envelope-application", "template-directory-session-command-payload-envelope-application.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)
	profiles := decodeSessionProfiles(t, fixture["profiles"])

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeSessionCommandPayloadEnvelopeFromFixture(t, testCase["envelope"], fixtureRoot)
		imported, importErr := asttemplate.ImportSessionCommandPayloadEnvelope(envelope)
		if importErr != nil {
			t.Fatalf("%s command payload envelope import failed: %+v", testCase["label"], importErr)
		}
		actual, err := asttemplate.RunTemplateDirectorySessionCommandPayload(*imported, profiles)
		if err != nil {
			t.Fatalf("%s command payload envelope application failed: %v", testCase["label"], err)
		}
		assertJSONEqual(t, resolveSessionDispatchExpectedFixturePaths(testCase["expected"], fixtureRoot), actual)
	}

	for _, rawCase := range fixture["rejections"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeSessionCommandPayloadEnvelopeFromFixture(t, testCase["envelope"], fixtureRoot)
		expected := decodeSessionCommandPayloadTransportImportErrorFromFixture(testCase["expected_error"])
		imported, importErr := asttemplate.ImportSessionCommandPayloadEnvelope(envelope)
		if imported != nil {
			t.Fatalf("%s command payload envelope rejection unexpectedly imported %+v", testCase["label"], imported)
		}
		if !reflect.DeepEqual(importErr, expected) {
			t.Fatalf("%s command payload envelope rejection mismatch: got %+v want %+v", testCase["label"], importErr, expected)
		}
	}
}

func TestTemplateDirectorySessionInvocationReportFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-383-template-directory-session-invocation-report", "template-directory-session-invocation-report.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)
	profiles := decodeSessionProfiles(t, fixture["profiles"])

	for _, key := range []string{"inspect_nested_payload_ready", "run_nested_request_ready", "run_flat_profile_blocked"} {
		section := fixture[key].(map[string]any)
		invocation := decodeSessionInvocationFromFixture(t, section["input"], fixtureRoot)
		actual, err := asttemplate.RunTemplateDirectorySession(invocation, profiles)
		if err != nil {
			t.Fatalf("%s invocation failed: %v", key, err)
		}
		assertJSONEqual(t, resolveSessionDispatchExpectedFixturePaths(section["expected"], fixtureRoot), actual)
	}
}

func TestTemplateDirectorySessionInvocationRejectionFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-384-template-directory-session-invocation-rejection", "template-directory-session-invocation-rejection.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)
	cases := fixture["cases"].([]any)

	for _, rawCase := range cases {
		testCase := rawCase.(map[string]any)
		invocation := decodeSessionInvocationFromFixture(t, testCase["input"], fixtureRoot)
		_, err := asttemplate.RunTemplateDirectorySession(invocation, nil)
		if err == nil {
			t.Fatalf("%s invocation rejection expected error", testCase["label"])
		}
		if err.Error() != testCase["expected_error"].(string) {
			t.Fatalf("%s invocation rejection error mismatch: got %q want %q", testCase["label"], err.Error(), testCase["expected_error"])
		}
	}
}

func TestTemplateDirectorySessionInvocationJSONRoundtripFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-385-template-directory-session-invocation-json-roundtrip", "template-directory-session-invocation-json-roundtrip.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)
	cases := fixture["cases"].([]any)

	for _, rawCase := range cases {
		testCase := rawCase.(map[string]any)
		invocation := decodeSessionInvocationFromFixture(t, testCase["input"], fixtureRoot)

		payload, err := json.Marshal(invocation)
		if err != nil {
			t.Fatalf("%s invocation roundtrip marshal failed: %v", testCase["label"], err)
		}

		var roundTripped asttemplate.SessionInvocation
		if err := json.Unmarshal(payload, &roundTripped); err != nil {
			t.Fatalf("%s invocation roundtrip unmarshal failed: %v", testCase["label"], err)
		}

		assertJSONEqual(t, invocation, roundTripped)
	}
}

func TestTemplateDirectorySessionInvocationTransportEnvelopeFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-386-template-directory-session-invocation-transport-envelope", "template-directory-session-invocation-envelope.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)
	cases := fixture["cases"].([]any)

	for _, rawCase := range cases {
		testCase := rawCase.(map[string]any)
		invocation := decodeSessionInvocationFromFixture(t, testCase["input"], fixtureRoot)
		expected := decodeSessionInvocationEnvelopeFromFixture(t, testCase["expected_envelope"], fixtureRoot)

		envelope := asttemplate.SessionInvocationEnvelopeFor(invocation)
		if !reflect.DeepEqual(envelope, expected) {
			t.Fatalf("%s invocation envelope mismatch: got %+v want %+v", testCase["label"], envelope, expected)
		}

		imported, importErr := asttemplate.ImportSessionInvocationEnvelope(expected)
		if importErr != nil {
			t.Fatalf("%s invocation envelope import failed: %+v", testCase["label"], importErr)
		}
		if imported == nil {
			t.Fatalf("%s invocation envelope import returned nil invocation", testCase["label"])
		}

		assertJSONEqual(t, invocation, *imported)
	}
}

func TestTemplateDirectorySessionInvocationTransportRejectionFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-387-template-directory-session-invocation-transport-rejection", "template-directory-session-invocation-envelope-rejection.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)
	cases := fixture["cases"].([]any)

	for _, rawCase := range cases {
		testCase := rawCase.(map[string]any)
		envelope := decodeSessionInvocationEnvelopeFromFixture(t, testCase["envelope"], fixtureRoot)
		expected := decodeSessionInvocationTransportImportErrorFromFixture(testCase["expected_error"])

		imported, importErr := asttemplate.ImportSessionInvocationEnvelope(envelope)
		if imported != nil {
			t.Fatalf("%s invocation transport rejection unexpectedly imported %+v", testCase["label"], imported)
		}
		if !reflect.DeepEqual(importErr, expected) {
			t.Fatalf("%s invocation transport rejection mismatch: got %+v want %+v", testCase["label"], importErr, expected)
		}
	}
}

func TestTemplateDirectorySessionInvocationEnvelopeApplicationFixture(t *testing.T) {
	fixturePath := filepath.Join(repoRoot(t), "fixtures", "diagnostics", "slice-388-template-directory-session-invocation-envelope-application", "template-directory-session-invocation-envelope-application.json")
	fixture := readJSONFixture(t, fixturePath)
	fixtureRoot := filepath.Dir(fixturePath)
	profiles := decodeSessionProfiles(t, fixture["profiles"])

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeSessionInvocationEnvelopeFromFixture(t, testCase["envelope"], fixtureRoot)
		imported, importErr := asttemplate.ImportSessionInvocationEnvelope(envelope)
		if importErr != nil {
			t.Fatalf("%s invocation envelope import failed: %+v", testCase["label"], importErr)
		}
		actual, err := asttemplate.RunTemplateDirectorySession(*imported, profiles)
		if err != nil {
			t.Fatalf("%s invocation envelope application failed: %v", testCase["label"], err)
		}
		assertJSONEqual(t, resolveSessionDispatchExpectedFixturePaths(testCase["expected"], fixtureRoot), actual)
	}

	for _, rawCase := range fixture["rejections"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeSessionInvocationEnvelopeFromFixture(t, testCase["envelope"], fixtureRoot)
		expected := decodeSessionInvocationTransportImportErrorFromFixture(testCase["expected_error"])
		imported, importErr := asttemplate.ImportSessionInvocationEnvelope(envelope)
		if imported != nil {
			t.Fatalf("%s invocation envelope rejection unexpectedly imported %+v", testCase["label"], imported)
		}
		if !reflect.DeepEqual(importErr, expected) {
			t.Fatalf("%s invocation envelope rejection mismatch: got %+v want %+v", testCase["label"], importErr, expected)
		}
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

func decodeSessionOptions(t *testing.T, raw any, templateRoot string, destinationRoot string) asttemplate.DirectorySessionOptions {
	t.Helper()
	section := raw.(map[string]any)
	return asttemplate.DirectorySessionOptions{
		Mode:            asttemplate.DirectorySessionMode(section["mode"].(string)),
		TemplateRoot:    templateRoot,
		DestinationRoot: destinationRoot,
		Context:         decodeContext(t, section["context"]),
		DefaultStrategy: decodeStrategy(t, section["default_strategy"]),
		Overrides:       decodeOverrides(t, section["overrides"]),
		Replacements:    decodeReplacements(t, section["replacements"]),
		AllowedFamilies: decodeOptionalFamilies(t, section["allowed_families"]),
	}
}

func decodeSessionOptionsFromFixture(t *testing.T, raw any) asttemplate.DirectorySessionOptions {
	t.Helper()
	section := raw.(map[string]any)
	templateRoot, _ := section["template_root"].(string)
	destinationRoot, _ := section["destination_root"].(string)
	var context *astmerge.TemplateDestinationContext
	if rawContext, ok := section["context"]; ok {
		context = decodeContext(t, rawContext)
	}
	var defaultStrategy astmerge.TemplateStrategy
	if rawStrategy, ok := section["default_strategy"]; ok {
		defaultStrategy = decodeStrategy(t, rawStrategy)
	}
	var overrides []astmerge.TemplateStrategyOverride
	if rawOverrides, ok := section["overrides"]; ok {
		overrides = decodeOverrides(t, rawOverrides)
	}
	var replacements map[string]string
	if rawReplacements, ok := section["replacements"]; ok {
		replacements = decodeReplacements(t, rawReplacements)
	}
	var allowedFamilies []string
	if rawFamilies, ok := section["allowed_families"]; ok {
		allowedFamilies = decodeOptionalFamilies(t, rawFamilies)
	}
	return asttemplate.DirectorySessionOptions{
		Mode:            asttemplate.DirectorySessionMode(section["mode"].(string)),
		TemplateRoot:    templateRoot,
		DestinationRoot: destinationRoot,
		Context:         context,
		DefaultStrategy: defaultStrategy,
		Overrides:       overrides,
		Replacements:    replacements,
		AllowedFamilies: allowedFamilies,
	}
}

func decodeSessionProfiles(t *testing.T, raw any) map[string]asttemplate.DirectorySessionProfile {
	t.Helper()
	sections := raw.(map[string]any)
	profiles := map[string]asttemplate.DirectorySessionProfile{}
	for name, value := range sections {
		section := value.(map[string]any)
		profiles[name] = asttemplate.DirectorySessionProfile{
			Mode:            asttemplate.DirectorySessionMode(section["mode"].(string)),
			Context:         decodeContext(t, section["context"]),
			DefaultStrategy: decodeStrategy(t, section["default_strategy"]),
			Overrides:       decodeOverrides(t, section["overrides"]),
			Replacements:    decodeReplacements(t, section["replacements"]),
			AllowedFamilies: decodeOptionalFamilies(t, section["allowed_families"]),
		}
	}
	return profiles
}

func decodeSessionRequestReport(t *testing.T, raw any) asttemplate.SessionRequestReport {
	t.Helper()
	section := raw.(map[string]any)
	var resolved *asttemplate.DirectorySessionOptions
	if rawResolved, ok := section["resolved_options"]; ok && rawResolved != nil {
		options := decodeSessionOptionsFromFixture(t, rawResolved)
		resolved = &options
	}
	return asttemplate.SessionRequestReport{
		RequestKind:     section["request_kind"].(string),
		ProfileName:     stringOrZero(section["profile_name"]),
		Mode:            asttemplate.DirectorySessionMode(section["mode"].(string)),
		Ready:           section["ready"].(bool),
		Diagnostics:     decodeSessionDiagnostics(t, section["diagnostics"]),
		ResolvedOptions: resolved,
	}
}

func decodeSessionRequestReportFromFixture(t *testing.T, raw any, fixtureRoot string) asttemplate.SessionRequestReport {
	t.Helper()
	report := decodeSessionRequestReport(t, raw)
	if report.ResolvedOptions != nil {
		report.ResolvedOptions.TemplateRoot = filepath.Join(fixtureRoot, report.ResolvedOptions.TemplateRoot)
		report.ResolvedOptions.DestinationRoot = filepath.Join(fixtureRoot, report.ResolvedOptions.DestinationRoot)
	}
	return report
}

func decodeSessionRequestEnvelopeFromFixture(t *testing.T, raw any, fixtureRoot string) asttemplate.SessionRequestEnvelope {
	t.Helper()
	section := raw.(map[string]any)
	return asttemplate.SessionRequestEnvelope{
		Kind:    stringOrZero(section["kind"]),
		Version: int(section["version"].(float64)),
		Request: decodeSessionRequestReportFromFixture(t, section["request"], fixtureRoot),
	}
}

func decodeSessionRunnerRequestFromFixture(t *testing.T, raw any, fixtureRoot string) asttemplate.SessionRunnerRequest {
	t.Helper()
	section := raw.(map[string]any)
	request := asttemplate.SessionRunnerRequest{
		RequestKind: stringOrZero(section["request_kind"]),
		ProfileName: stringOrZero(section["profile_name"]),
	}
	if rawOptions, ok := section["options"]; ok && rawOptions != nil {
		request.Options = cloneFixturePathMap(rawOptions, fixtureRoot)
	}
	if rawOverrides, ok := section["overrides"]; ok && rawOverrides != nil {
		request.Overrides = cloneFixturePathMap(rawOverrides, fixtureRoot)
	}
	return request
}

func decodeSessionRunnerRequest(t *testing.T, raw any) asttemplate.SessionRunnerRequest {
	t.Helper()
	section := raw.(map[string]any)
	request := asttemplate.SessionRunnerRequest{
		RequestKind: stringOrZero(section["request_kind"]),
		ProfileName: stringOrZero(section["profile_name"]),
	}
	if rawOptions, ok := section["options"]; ok && rawOptions != nil {
		request.Options = cloneAnyMap(rawOptions)
	}
	if rawOverrides, ok := section["overrides"]; ok && rawOverrides != nil {
		request.Overrides = cloneAnyMap(rawOverrides)
	}
	return request
}

func decodeSessionRunnerRequestEnvelopeFromFixture(t *testing.T, raw any, fixtureRoot string) asttemplate.SessionRunnerRequestEnvelope {
	t.Helper()
	section := raw.(map[string]any)
	return asttemplate.SessionRunnerRequestEnvelope{
		Kind:    stringOrZero(section["kind"]),
		Version: int(section["version"].(float64)),
		Request: decodeSessionRunnerRequestFromFixture(t, section["request"], fixtureRoot),
	}
}

func decodeSessionRunnerInput(t *testing.T, raw any) asttemplate.SessionRunnerInput {
	t.Helper()
	section := raw.(map[string]any)
	return asttemplate.SessionRunnerInput{
		RequestKind:     stringOrZero(section["request_kind"]),
		ProfileName:     stringOrZero(section["profile_name"]),
		Mode:            asttemplate.DirectorySessionMode(section["mode"].(string)),
		TemplateRoot:    stringOrZero(section["template_root"]),
		DestinationRoot: stringOrZero(section["destination_root"]),
		Context:         decodeOptionalContext(t, section["context"]),
		DefaultStrategy: decodeOptionalStrategy(t, section["default_strategy"]),
		Overrides:       decodeOptionalOverrides(t, section["overrides"]),
		Replacements:    decodeOptionalReplacements(t, section["replacements"]),
		AllowedFamilies: decodeOptionalFamilies(t, section["allowed_families"]),
	}
}

func decodeSessionRunnerPayload(t *testing.T, raw any) asttemplate.SessionRunnerPayload {
	t.Helper()
	section := raw.(map[string]any)
	return asttemplate.SessionRunnerPayload{
		RequestKind:        stringOrZero(section["request_kind"]),
		DefaultProfileName: stringOrZero(section["default_profile_name"]),
		ProfileName:        stringOrZero(section["profile_name"]),
		Mode:               asttemplate.DirectorySessionMode(section["mode"].(string)),
		TemplateRoot:       stringOrZero(section["template_root"]),
		DestinationRoot:    stringOrZero(section["destination_root"]),
		Context:            decodeOptionalContext(t, section["context"]),
		DefaultStrategy:    decodeOptionalStrategy(t, section["default_strategy"]),
		Overrides:          decodeOptionalOverrides(t, section["overrides"]),
		Replacements:       decodeOptionalReplacements(t, section["replacements"]),
		AllowedFamilies:    decodeOptionalFamilies(t, section["allowed_families"]),
	}
}

func decodeSessionRunnerPayloadFromFixture(t *testing.T, raw any, fixtureRoot string) asttemplate.SessionRunnerPayload {
	t.Helper()
	payload := decodeSessionRunnerPayload(t, raw)
	if payload.TemplateRoot != "" {
		payload.TemplateRoot = filepath.Join(fixtureRoot, payload.TemplateRoot)
	}
	if payload.DestinationRoot != "" {
		payload.DestinationRoot = filepath.Join(fixtureRoot, payload.DestinationRoot)
	}
	return payload
}

func decodeSessionRunnerPayloadEnvelopeFromFixture(t *testing.T, raw any, fixtureRoot string) asttemplate.SessionRunnerPayloadEnvelope {
	t.Helper()
	section := raw.(map[string]any)
	return asttemplate.SessionRunnerPayloadEnvelope{
		Kind:    stringOrZero(section["kind"]),
		Version: int(section["version"].(float64)),
		Payload: decodeSessionRunnerPayloadFromFixture(t, section["payload"], fixtureRoot),
	}
}

func decodeSessionRunnerPayloadTransportImportErrorFromFixture(raw any) *asttemplate.SessionRunnerPayloadTransportImportError {
	section := raw.(map[string]any)
	return &asttemplate.SessionRunnerPayloadTransportImportError{
		Category: asttemplate.SessionRunnerPayloadTransportImportErrorCategory(stringOrZero(section["category"])),
		Message:  stringOrZero(section["message"]),
	}
}

func decodeSessionEntrypointFromFixture(t *testing.T, raw any, fixtureRoot string) asttemplate.SessionEntrypoint {
	t.Helper()
	section := raw.(map[string]any)
	entrypoint := asttemplate.SessionEntrypoint{}
	if payload, ok := section["payload"]; ok && payload != nil {
		resolved := decodeSessionRunnerPayloadFromFixture(t, payload, fixtureRoot)
		entrypoint.Payload = &resolved
	}
	if request, ok := section["request"]; ok && request != nil {
		resolved := decodeSessionRunnerRequestFromFixture(t, request, fixtureRoot)
		entrypoint.Request = &resolved
	}
	return entrypoint
}

func decodeSessionEntrypoint(t *testing.T, raw any) asttemplate.SessionEntrypoint {
	t.Helper()
	section := raw.(map[string]any)
	entrypoint := asttemplate.SessionEntrypoint{}
	if payload, ok := section["payload"]; ok && payload != nil {
		resolved := decodeSessionRunnerPayload(t, payload)
		entrypoint.Payload = &resolved
	}
	if request, ok := section["request"]; ok && request != nil {
		resolved := decodeSessionRunnerRequest(t, request)
		entrypoint.Request = &resolved
	}
	return entrypoint
}

func decodeSessionEntrypointEnvelopeFromFixture(t *testing.T, raw any, fixtureRoot string) asttemplate.SessionEntrypointEnvelope {
	t.Helper()
	section := raw.(map[string]any)
	return asttemplate.SessionEntrypointEnvelope{
		Kind:       stringOrZero(section["kind"]),
		Version:    int(section["version"].(float64)),
		Entrypoint: decodeSessionEntrypointFromFixture(t, section["entrypoint"], fixtureRoot),
	}
}

func decodeSessionEntrypointTransportImportErrorFromFixture(raw any) *asttemplate.SessionEntrypointTransportImportError {
	section := raw.(map[string]any)
	return &asttemplate.SessionEntrypointTransportImportError{
		Category: asttemplate.SessionEntrypointTransportImportErrorCategory(stringOrZero(section["category"])),
		Message:  stringOrZero(section["message"]),
	}
}

func decodeSessionRunnerRequestTransportImportErrorFromFixture(raw any) *asttemplate.SessionRunnerRequestTransportImportError {
	section := raw.(map[string]any)
	return &asttemplate.SessionRunnerRequestTransportImportError{
		Category: asttemplate.SessionRunnerRequestTransportImportErrorCategory(stringOrZero(section["category"])),
		Message:  stringOrZero(section["message"]),
	}
}

func decodeSessionDispatchInputFromFixture(t *testing.T, raw any, fixtureRoot string) map[string]any {
	t.Helper()
	section := raw.(map[string]any)
	return map[string]any{
		"operation":  stringOrZero(section["operation"]),
		"entrypoint": decodeSessionEntrypointFromFixture(t, section["entrypoint"], fixtureRoot),
	}
}

func decodeSessionCommandFromFixture(t *testing.T, raw any, fixtureRoot string) asttemplate.SessionCommand {
	t.Helper()
	section := raw.(map[string]any)
	command := asttemplate.SessionCommand{
		Operation: stringOrZero(section["operation"]),
	}
	if payload, ok := section["payload"]; ok && payload != nil {
		resolved := decodeSessionRunnerPayloadFromFixture(t, payload, fixtureRoot)
		command.Payload = &resolved
	}
	if request, ok := section["request"]; ok && request != nil {
		resolved := decodeSessionRunnerRequestFromFixture(t, request, fixtureRoot)
		command.Request = &resolved
	}
	return command
}

func decodeSessionCommandEnvelopeFromFixture(t *testing.T, raw any, fixtureRoot string) asttemplate.SessionCommandEnvelope {
	t.Helper()
	section := raw.(map[string]any)
	return asttemplate.SessionCommandEnvelope{
		Kind:    stringOrZero(section["kind"]),
		Version: int(section["version"].(float64)),
		Command: decodeSessionCommandFromFixture(t, section["command"], fixtureRoot),
	}
}

func decodeSessionCommandTransportImportErrorFromFixture(raw any) *asttemplate.SessionCommandTransportImportError {
	section := raw.(map[string]any)
	return &asttemplate.SessionCommandTransportImportError{
		Category: asttemplate.SessionCommandTransportImportErrorCategory(stringOrZero(section["category"])),
		Message:  stringOrZero(section["message"]),
	}
}

func decodeSessionCommandPayloadFromFixture(t *testing.T, raw any, fixtureRoot string) asttemplate.SessionCommandPayload {
	t.Helper()
	section := raw.(map[string]any)
	payload := asttemplate.SessionCommandPayload{
		Operation:          stringOrZero(section["operation"]),
		RequestKind:        stringOrZero(section["request_kind"]),
		DefaultProfileName: stringOrZero(section["default_profile_name"]),
		ProfileName:        stringOrZero(section["profile_name"]),
		Mode:               asttemplate.DirectorySessionMode(stringOrZero(section["mode"])),
		TemplateRoot:       stringOrZero(section["template_root"]),
		DestinationRoot:    stringOrZero(section["destination_root"]),
		Context:            decodeOptionalContext(t, section["context"]),
		DefaultStrategy:    decodeOptionalStrategy(t, section["default_strategy"]),
		Overrides:          decodeOptionalOverrides(t, section["overrides"]),
		Replacements:       decodeOptionalReplacements(t, section["replacements"]),
		AllowedFamilies:    decodeOptionalFamilies(t, section["allowed_families"]),
	}
	if payload.TemplateRoot != "" {
		payload.TemplateRoot = filepath.Join(fixtureRoot, payload.TemplateRoot)
	}
	if payload.DestinationRoot != "" {
		payload.DestinationRoot = filepath.Join(fixtureRoot, payload.DestinationRoot)
	}
	return payload
}

func decodeSessionCommandPayloadEnvelopeFromFixture(t *testing.T, raw any, fixtureRoot string) asttemplate.SessionCommandPayloadEnvelope {
	t.Helper()
	section := raw.(map[string]any)
	return asttemplate.SessionCommandPayloadEnvelope{
		Kind:    stringOrZero(section["kind"]),
		Version: int(section["version"].(float64)),
		Payload: decodeSessionCommandPayloadFromFixture(t, section["payload"], fixtureRoot),
	}
}

func decodeSessionCommandPayloadTransportImportErrorFromFixture(raw any) *asttemplate.SessionCommandPayloadTransportImportError {
	section := raw.(map[string]any)
	return &asttemplate.SessionCommandPayloadTransportImportError{
		Category: asttemplate.SessionCommandPayloadTransportImportErrorCategory(stringOrZero(section["category"])),
		Message:  stringOrZero(section["message"]),
	}
}

func decodeSessionInvocationFromFixture(t *testing.T, raw any, fixtureRoot string) asttemplate.SessionInvocation {
	t.Helper()
	section := raw.(map[string]any)
	invocation := asttemplate.SessionInvocation{
		Operation:          stringOrZero(section["operation"]),
		RequestKind:        stringOrZero(section["request_kind"]),
		DefaultProfileName: stringOrZero(section["default_profile_name"]),
		ProfileName:        stringOrZero(section["profile_name"]),
		Mode:               asttemplate.DirectorySessionMode(stringOrZero(section["mode"])),
		TemplateRoot:       stringOrZero(section["template_root"]),
		DestinationRoot:    stringOrZero(section["destination_root"]),
		Context:            decodeOptionalContext(t, section["context"]),
		DefaultStrategy:    decodeOptionalStrategy(t, section["default_strategy"]),
		Overrides:          decodeOptionalOverrides(t, section["overrides"]),
		Replacements:       decodeOptionalReplacements(t, section["replacements"]),
		AllowedFamilies:    decodeOptionalFamilies(t, section["allowed_families"]),
	}
	if payload, ok := section["payload"]; ok && payload != nil {
		resolved := decodeSessionRunnerPayloadFromFixture(t, payload, fixtureRoot)
		invocation.Payload = &resolved
	}
	if request, ok := section["request"]; ok && request != nil {
		resolved := decodeSessionRunnerRequestFromFixture(t, request, fixtureRoot)
		invocation.Request = &resolved
	}
	if invocation.TemplateRoot != "" {
		invocation.TemplateRoot = filepath.Join(fixtureRoot, invocation.TemplateRoot)
	}
	if invocation.DestinationRoot != "" {
		invocation.DestinationRoot = filepath.Join(fixtureRoot, invocation.DestinationRoot)
	}
	return invocation
}

func decodeSessionInvocationEnvelopeFromFixture(t *testing.T, raw any, fixtureRoot string) asttemplate.SessionInvocationEnvelope {
	t.Helper()
	section := raw.(map[string]any)
	return asttemplate.SessionInvocationEnvelope{
		Kind:       stringOrZero(section["kind"]),
		Version:    int(section["version"].(float64)),
		Invocation: decodeSessionInvocationFromFixture(t, section["invocation"], fixtureRoot),
	}
}

func decodeSessionInvocationTransportImportErrorFromFixture(raw any) *asttemplate.SessionInvocationTransportImportError {
	section := raw.(map[string]any)
	return &asttemplate.SessionInvocationTransportImportError{
		Category: asttemplate.SessionInvocationTransportImportErrorCategory(stringOrZero(section["category"])),
		Message:  stringOrZero(section["message"]),
	}
}

func cloneFixturePathMap(raw any, fixtureRoot string) map[string]any {
	section := cloneAnyMap(raw)
	if templateRoot, ok := section["template_root"].(string); ok && templateRoot != "" {
		section["template_root"] = filepath.Join(fixtureRoot, templateRoot)
	}
	if destinationRoot, ok := section["destination_root"].(string); ok && destinationRoot != "" {
		section["destination_root"] = filepath.Join(fixtureRoot, destinationRoot)
	}
	return section
}

func cloneAnyMap(raw any) map[string]any {
	section, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	cloned := make(map[string]any, len(section))
	for key, value := range section {
		cloned[key] = value
	}
	return cloned
}

func resolveSessionInspectionExpectedFixturePaths(raw any, fixtureRoot string) any {
	section := cloneAnyMap(raw)
	if section == nil {
		return raw
	}
	if entrypointReport, ok := section["entrypoint_report"].(map[string]any); ok {
		if runnerRequest, ok := entrypointReport["runner_request"]; ok {
			entrypointReport["runner_request"] = resolveRunnerRequestFixturePaths(runnerRequest, fixtureRoot)
		}
	}
	if sessionResolution, ok := section["session_resolution"].(map[string]any); ok {
		if runnerRequest, ok := sessionResolution["runner_request"]; ok {
			sessionResolution["runner_request"] = resolveRunnerRequestFixturePaths(runnerRequest, fixtureRoot)
		}
		if sessionRequest, ok := sessionResolution["session_request"].(map[string]any); ok {
			if resolvedOptions, ok := sessionRequest["resolved_options"]; ok && resolvedOptions != nil {
				sessionRequest["resolved_options"] = resolveRunnerRequestFixturePaths(resolvedOptions, fixtureRoot)
			}
		}
	}
	return section
}

func resolveSessionDispatchExpectedFixturePaths(raw any, fixtureRoot string) any {
	section := cloneAnyMap(raw)
	if section == nil {
		return raw
	}
	if inspection, ok := section["inspection"]; ok && inspection != nil {
		section["inspection"] = resolveSessionInspectionExpectedFixturePaths(inspection, fixtureRoot)
	}
	if outcome, ok := section["outcome"]; ok && outcome != nil {
		section["outcome"] = resolveSessionOutcomeExpectedFixturePaths(outcome, fixtureRoot)
	}
	return section
}

func resolveSessionOutcomeExpectedPaths(raw any, fixtureRoot string) any {
	return cloneFixturePathMap(raw, fixtureRoot)
}

func resolveSessionOutcomeExpectedFixturePaths(raw any, fixtureRoot string) any {
	section := cloneAnyMap(raw)
	if section == nil {
		return raw
	}
	if sessionReport, ok := section["session_report"].(map[string]any); ok {
		if runnerReport, ok := sessionReport["runner_report"].(map[string]any); ok {
			if preview, ok := runnerReport["preview"].(map[string]any); ok {
				if resultFiles, ok := preview["result_files"].(map[string]any); ok {
					preview["result_files"] = cloneAnyMap(resultFiles)
				}
			}
		}
	}
	return section
}

func resolveRunnerRequestFixturePaths(raw any, fixtureRoot string) any {
	section := cloneAnyMap(raw)
	if section == nil {
		return raw
	}
	if options, ok := section["options"]; ok && options != nil {
		section["options"] = resolveRunnerRequestFixturePaths(options, fixtureRoot)
	}
	if overrides, ok := section["overrides"]; ok && overrides != nil {
		section["overrides"] = resolveRunnerRequestFixturePaths(overrides, fixtureRoot)
	}
	if templateRoot, ok := section["template_root"].(string); ok && templateRoot != "" {
		section["template_root"] = filepath.Join(fixtureRoot, templateRoot)
	}
	if destinationRoot, ok := section["destination_root"].(string); ok && destinationRoot != "" {
		section["destination_root"] = filepath.Join(fixtureRoot, destinationRoot)
	}
	return section
}

func decodeOptionalContext(t *testing.T, raw any) *astmerge.TemplateDestinationContext {
	t.Helper()
	if raw == nil {
		return nil
	}
	return decodeContext(t, raw)
}

func decodeOptionalStrategy(t *testing.T, raw any) astmerge.TemplateStrategy {
	t.Helper()
	if raw == nil {
		return ""
	}
	return decodeStrategy(t, raw)
}

func decodeOptionalOverrides(t *testing.T, raw any) []astmerge.TemplateStrategyOverride {
	t.Helper()
	if raw == nil {
		return nil
	}
	return decodeOverrides(t, raw)
}

func decodeOptionalReplacements(t *testing.T, raw any) map[string]string {
	t.Helper()
	if raw == nil {
		return nil
	}
	return decodeReplacements(t, raw)
}

func decodeSessionDiagnostics(t *testing.T, raw any) []asttemplate.SessionDiagnostic {
	t.Helper()
	if raw == nil {
		return nil
	}
	data, _ := json.Marshal(raw)
	var diagnostics []asttemplate.SessionDiagnostic
	if err := json.Unmarshal(data, &diagnostics); err != nil {
		t.Fatalf("decode session diagnostics: %v", err)
	}
	return diagnostics
}

func stringOrZero(value any) string {
	text, _ := value.(string)
	return text
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

package goldmarkmerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/markdownmerge"
	"github.com/structuredmerge/structuredmerge-go/treehaver"
)

func jsonReady(t *testing.T, value any) any {
	t.Helper()
	source, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal value: %v", err)
	}

	var normalized any
	if err := json.Unmarshal(source, &normalized); err != nil {
		t.Fatalf("decode value: %v", err)
	}

	return normalized
}

func readGoldmarkFixture(t *testing.T, parts ...string) map[string]any {
	t.Helper()
	pathParts := append([]string{"..", "..", "fixtures"}, parts...)
	source, err := os.ReadFile(filepath.Join(pathParts...))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var fixture map[string]any
	if err := json.Unmarshal(source, &fixture); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	return fixture
}

func TestSharedFixtureMarkdownProviderFeatureProfile(t *testing.T) {
	familyFixture := readGoldmarkFixture(t, "diagnostics", "slice-194-markdown-family-feature-profile", "markdown-feature-profile.json")
	fixture := readGoldmarkFixture(t, "diagnostics", "slice-204-markdown-provider-feature-profiles", "go-markdown-provider-feature-profiles.json")
	familyProfile := markdownmerge.MarkdownFeatureProfileInfo()
	if actual := jsonReady(t, map[string]any{
		"family":             familyProfile.Family,
		"supported_dialects": familyProfile.SupportedDialects,
		"supported_policies": familyProfile.SupportedPolicies,
	}); !reflect.DeepEqual(actual, familyFixture["feature_profile"]) {
		t.Fatalf("unexpected family profile: %+v", actual)
	}
	expected := fixture["providers"].(map[string]any)["goldmark"].(map[string]any)["feature_profile"]
	profile := MarkdownBackendFeatureProfileInfo()
	actual := jsonReady(t, map[string]any{
		"family":             profile.Family,
		"supported_dialects": profile.SupportedDialects,
		"supported_policies": profile.SupportedPolicies,
		"backend":            profile.Backend,
		"backend_ref": map[string]any{
			"id":     profile.BackendRef.ID,
			"family": profile.BackendRef.Family,
		},
	})
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("unexpected provider profile: %+v", profile)
	}
	if len(AvailableMarkdownBackends()) != 1 || AvailableMarkdownBackends()[0] != "goldmark" {
		t.Fatalf("unexpected backends: %+v", AvailableMarkdownBackends())
	}
	if backend := treehaver.BackendReferenceByID(BackendGoldmark); backend == nil || backend.ID != BackendGoldmark || backend.Family != "native" {
		t.Fatalf("unexpected registered backend: %+v", backend)
	}
}

func TestSharedFixtureMarkdownProviderPlanContext(t *testing.T) {
	fixture := readGoldmarkFixture(t, "diagnostics", "slice-205-markdown-provider-plan-contexts", "go-markdown-provider-plan-contexts.json")
	if context := MarkdownPlanContext(); context.FeatureProfile == nil || context.FeatureProfile.Backend != fixture["providers"].(map[string]any)["goldmark"].(map[string]any)["feature_profile"].(map[string]any)["backend"].(string) {
		t.Fatalf("unexpected provider plan context: %+v", context)
	}
}

func TestSharedFixtureMarkdownProviderAnalysisAndMatching(t *testing.T) {
	analysisFixture := readGoldmarkFixture(t, "markdown", "slice-198-analysis", "headings-and-code-fences.json")
	matchingFixture := readGoldmarkFixture(t, "markdown", "slice-199-matching", "path-equality.json")

	analysis := ParseMarkdown(analysisFixture["source"].(string), markdownmerge.DialectMarkdown)
	if !analysis.OK || analysis.Analysis == nil {
		t.Fatalf("expected parse success: %+v", analysis)
	}

	template := ParseMarkdown(matchingFixture["template"].(string), markdownmerge.DialectMarkdown)
	destination := ParseMarkdown(matchingFixture["destination"].(string), markdownmerge.DialectMarkdown)
	if !template.OK || template.Analysis == nil || !destination.OK || destination.Analysis == nil {
		t.Fatalf("expected parse success for matching fixtures")
	}

	result := MatchMarkdownOwners(*template.Analysis, *destination.Analysis)
	matched := make([][]string, 0, len(result.Matched))
	for _, match := range result.Matched {
		matched = append(matched, []string{match.TemplatePath, match.DestinationPath})
	}
	if actual := jsonReady(t, matched); !reflect.DeepEqual(actual, matchingFixture["expected"].(map[string]any)["matched"]) {
		t.Fatalf("unexpected matches: %+v", actual)
	}
	if actual := jsonReady(t, result.UnmatchedTemplate); !reflect.DeepEqual(actual, matchingFixture["expected"].(map[string]any)["unmatched_template"]) {
		t.Fatalf("unexpected unmatched template owners: %+v", actual)
	}
	if actual := jsonReady(t, result.UnmatchedDestination); !reflect.DeepEqual(actual, matchingFixture["expected"].(map[string]any)["unmatched_destination"]) {
		t.Fatalf("unexpected unmatched destination owners: %+v", actual)
	}
}

func TestSharedFixtureMarkdownProviderEmbeddedFamilies(t *testing.T) {
	fixture := readGoldmarkFixture(t, "markdown", "slice-208-embedded-families", "code-fence-families.json")
	analysis := ParseMarkdown(fixture["source"].(string), markdownmerge.DialectMarkdown)
	if !analysis.OK || analysis.Analysis == nil {
		t.Fatalf("expected parse success: %+v", analysis)
	}
	if actual := jsonReady(t, MarkdownEmbeddedFamilies(*analysis.Analysis)); !reflect.DeepEqual(actual, fixture["expected"]) {
		t.Fatalf("unexpected embedded families: %+v", actual)
	}
}

func TestSharedFixtureMarkdownProviderNamedSuitePlans(t *testing.T) {
	fixture := readGoldmarkFixture(t, "diagnostics", "slice-206-markdown-provider-named-suite-plans", "go-markdown-provider-named-suite-plans.json")
	source, err := json.Marshal(fixture["manifest"])
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}

	var manifest astmerge.ConformanceManifest
	if err := json.Unmarshal(source, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}

	plans := astmerge.PlanNamedConformanceSuites(manifest, map[string]astmerge.ConformanceFamilyPlanContext{
		"markdown": MarkdownPlanContext(),
	})

	projected := make([]map[string]any, 0, len(plans))
	for _, entry := range plans {
		planEntries := make([]map[string]any, 0, len(entry.Plan.Entries))
		for _, planEntry := range entry.Plan.Entries {
			run := map[string]any{
				"ref":          map[string]any{"family": planEntry.Run.Ref.Family, "role": planEntry.Run.Ref.Role, "case": planEntry.Run.Ref.Case},
				"requirements": map[string]any{},
			}
			if planEntry.Run.FamilyProfile.Family != "" {
				run["family_profile"] = map[string]any{
					"family":             planEntry.Run.FamilyProfile.Family,
					"supported_dialects": planEntry.Run.FamilyProfile.SupportedDialects,
					"supported_policies": planEntry.Run.FamilyProfile.SupportedPolicies,
				}
			}
			if planEntry.Run.FeatureProfile != nil {
				run["feature_profile"] = map[string]any{
					"backend":            planEntry.Run.FeatureProfile.Backend,
					"supports_dialects":  planEntry.Run.FeatureProfile.SupportsDialects,
					"supported_policies": planEntry.Run.FeatureProfile.SupportedPolicies,
				}
			}
			planEntries = append(planEntries, map[string]any{
				"ref":  map[string]any{"family": planEntry.Ref.Family, "role": planEntry.Ref.Role, "case": planEntry.Ref.Case},
				"path": planEntry.Path,
				"run":  run,
			})
		}

		projected = append(projected, map[string]any{
			"suite": entry.Suite,
			"plan": map[string]any{
				"family":        entry.Plan.Family,
				"entries":       planEntries,
				"missing_roles": entry.Plan.MissingRoles,
			},
		})
	}

	if actual := jsonReady(t, projected); !reflect.DeepEqual(actual, fixture["expected_entries"]) {
		t.Fatalf("unexpected plans: %+v", actual)
	}
}

func TestSharedFixtureMarkdownProviderManifestReport(t *testing.T) {
	fixture := readGoldmarkFixture(t, "diagnostics", "slice-207-markdown-provider-manifest-report", "go-markdown-provider-manifest-report.json")
	source, err := json.Marshal(fixture["manifest"])
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}

	var manifest astmerge.ConformanceManifest
	if err := json.Unmarshal(source, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}

	executionsSource, err := json.Marshal(fixture["executions"])
	if err != nil {
		t.Fatalf("marshal executions: %v", err)
	}

	var executions map[string]astmerge.ConformanceCaseExecution
	if err := json.Unmarshal(executionsSource, &executions); err != nil {
		t.Fatalf("decode executions: %v", err)
	}

	entries := astmerge.ReportPlannedNamedConformanceSuites(
		astmerge.PlanNamedConformanceSuites(manifest, map[string]astmerge.ConformanceFamilyPlanContext{
			"markdown": MarkdownPlanContext(),
		}),
		func(run astmerge.ConformanceCaseRun) astmerge.ConformanceCaseExecution {
			key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
			if result, ok := executions[key]; ok {
				return result
			}
			return astmerge.ConformanceCaseExecution{Outcome: "failed", Messages: []string{"missing execution"}}
		},
	)

	report := astmerge.ReportNamedConformanceSuiteEnvelope(entries)
	projected := map[string]any{
		"entries": make([]map[string]any, 0, len(report.Entries)),
		"summary": map[string]any{
			"total": report.Summary.Total, "passed": report.Summary.Passed,
			"failed": report.Summary.Failed, "skipped": report.Summary.Skipped,
		},
	}
	for _, entry := range report.Entries {
		results := make([]map[string]any, 0, len(entry.Report.Results))
		for _, result := range entry.Report.Results {
			results = append(results, map[string]any{
				"ref": map[string]any{
					"family": result.Ref.Family,
					"role":   result.Ref.Role,
					"case":   result.Ref.Case,
				},
				"outcome":  result.Outcome,
				"messages": result.Messages,
			})
		}
		projected["entries"] = append(projected["entries"].([]map[string]any), map[string]any{
			"suite": entry.Suite,
			"report": map[string]any{
				"results": results,
				"summary": map[string]any{
					"total": entry.Report.Summary.Total, "passed": entry.Report.Summary.Passed,
					"failed": entry.Report.Summary.Failed, "skipped": entry.Report.Summary.Skipped,
				},
			},
		})
	}

	if actual := jsonReady(t, projected); !reflect.DeepEqual(actual, fixture["expected_report"]) {
		t.Fatalf("unexpected report: %+v", actual)
	}
}

func TestSharedFixtureMarkdownProviderRejectsUnsupportedBackendOverrides(t *testing.T) {
	parseResult := ParseMarkdown("# Title\n", markdownmerge.DialectMarkdown, "kreuzberg-language-pack")
	expectedDiagnostics := []map[string]any{{
		"severity": "error",
		"category": "unsupported_feature",
		"message":  "Unsupported Markdown backend kreuzberg-language-pack.",
	}}
	if parseResult.OK {
		t.Fatalf("expected parse failure: %+v", parseResult)
	}
	if actual := jsonReady(t, parseResult.Diagnostics); !reflect.DeepEqual(actual, jsonReady(t, expectedDiagnostics)) {
		t.Fatalf("unexpected parse diagnostics: %+v", actual)
	}
}

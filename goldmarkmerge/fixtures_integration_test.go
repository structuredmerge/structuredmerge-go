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

	mergeFixture := readGoldmarkFixture(t, "markdown", "slice-286-merge", "section-merge.json")
	mergeResult := MergeMarkdown(mergeFixture["template"].(string), mergeFixture["destination"].(string), markdownmerge.DialectMarkdown)
	if !mergeResult.OK || mergeResult.Output == nil {
		t.Fatalf("expected merge success: %+v", mergeResult)
	}
	if *mergeResult.Output != mergeFixture["expected"].(map[string]any)["output"].(string) {
		t.Fatalf("unexpected merge output:\n%s", *mergeResult.Output)
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

func TestSharedFixtureMarkdownProviderReviewedNestedMerge(t *testing.T) {
	fixture := readGoldmarkFixture(t, "markdown", "slice-298-reviewed-nested-merge", "fenced-code-reviewed-nested-merge.json")
	reviewStateSource, err := json.Marshal(fixture["review_state"])
	if err != nil {
		t.Fatalf("marshal review state: %v", err)
	}
	var reviewState astmerge.DelegatedChildGroupReviewState
	if err := json.Unmarshal(reviewStateSource, &reviewState); err != nil {
		t.Fatalf("decode review state: %v", err)
	}
	appliedChildrenSource, err := json.Marshal(fixture["applied_children"])
	if err != nil {
		t.Fatalf("marshal applied children: %v", err)
	}
	var appliedChildren []markdownmerge.AppliedChildOutput
	if err := json.Unmarshal(appliedChildrenSource, &appliedChildren); err != nil {
		t.Fatalf("decode applied children: %v", err)
	}
	result := MergeMarkdownWithReviewedNestedOutputs(
		fixture["template"].(string),
		fixture["destination"].(string),
		markdownmerge.DialectMarkdown,
		reviewState,
		appliedChildren,
	)
	if !result.OK || result.Output == nil {
		t.Fatalf("expected reviewed nested merge success: %+v", result)
	}
	if *result.Output != fixture["expected"].(map[string]any)["output"].(string) {
		t.Fatalf("unexpected reviewed nested merge output:\n%s", *result.Output)
	}
}

func TestSharedFixtureMarkdownProviderReviewedNestedReviewArtifactApplication(t *testing.T) {
	providerFixture := readGoldmarkFixture(t, "diagnostics", "slice-326-markdown-provider-reviewed-nested-review-artifact-application", "go-markdown-provider-reviewed-nested-review-artifact-application.json")
	sharedFixturePathRaw := providerFixture["shared_fixture_path"].([]any)
	sharedFixturePath := make([]string, 0, len(sharedFixturePathRaw))
	for _, part := range sharedFixturePathRaw {
		sharedFixturePath = append(sharedFixturePath, part.(string))
	}
	fixture := readGoldmarkFixture(t, sharedFixturePath...)
	expected := providerFixture["providers"].(map[string]any)["goldmark"].(map[string]any)["expected"].(map[string]any)
	replayBundleSource, err := json.Marshal(fixture["replay_bundle"])
	if err != nil {
		t.Fatalf("marshal replay bundle: %v", err)
	}
	var replayBundle astmerge.ReviewReplayBundle
	if err := json.Unmarshal(replayBundleSource, &replayBundle); err != nil {
		t.Fatalf("decode replay bundle: %v", err)
	}
	reviewStateSource, err := json.Marshal(fixture["review_state"])
	if err != nil {
		t.Fatalf("marshal review state: %v", err)
	}
	var reviewState astmerge.ConformanceManifestReviewState
	if err := json.Unmarshal(reviewStateSource, &reviewState); err != nil {
		t.Fatalf("decode review state: %v", err)
	}

	replayResult := MergeMarkdownWithReviewedNestedOutputsFromReplayBundle(
		fixture["template"].(string),
		fixture["destination"].(string),
		markdownmerge.DialectMarkdown,
		replayBundle,
	)
	if !replayResult.OK || replayResult.Output == nil {
		t.Fatalf("expected replay-bundle reviewed nested merge success: %+v", replayResult)
	}
	if *replayResult.Output != expected["output"].(string) {
		t.Fatalf("unexpected replay-bundle reviewed nested merge output:\n%s", *replayResult.Output)
	}

	stateResult := MergeMarkdownWithReviewedNestedOutputsFromReviewState(
		fixture["template"].(string),
		fixture["destination"].(string),
		markdownmerge.DialectMarkdown,
		reviewState,
	)
	if !stateResult.OK || stateResult.Output == nil {
		t.Fatalf("expected review-state reviewed nested merge success: %+v", stateResult)
	}
	if *stateResult.Output != expected["output"].(string) {
		t.Fatalf("unexpected review-state reviewed nested merge output:\n%s", *stateResult.Output)
	}
}

func TestSharedFixtureMarkdownProviderReviewedNestedReviewArtifactRejection(t *testing.T) {
	providerFixture := readGoldmarkFixture(t, "diagnostics", "slice-327-markdown-provider-reviewed-nested-review-artifact-rejection", "go-markdown-provider-reviewed-nested-review-artifact-rejection.json")
	sharedFixturePathRaw := providerFixture["shared_fixture_path"].([]any)
	sharedFixturePath := make([]string, 0, len(sharedFixturePathRaw))
	for _, part := range sharedFixturePathRaw {
		sharedFixturePath = append(sharedFixturePath, part.(string))
	}
	fixture := readGoldmarkFixture(t, sharedFixturePath...)
	expectedReplay := providerFixture["providers"].(map[string]any)["goldmark"].(map[string]any)["expected_replay_bundle"].(map[string]any)
	expectedState := providerFixture["providers"].(map[string]any)["goldmark"].(map[string]any)["expected_review_state"].(map[string]any)
	replayBundleSource, err := json.Marshal(fixture["replay_bundle"])
	if err != nil {
		t.Fatalf("marshal replay bundle: %v", err)
	}
	var replayBundle astmerge.ReviewReplayBundle
	if err := json.Unmarshal(replayBundleSource, &replayBundle); err != nil {
		t.Fatalf("decode replay bundle: %v", err)
	}
	reviewStateSource, err := json.Marshal(fixture["review_state"])
	if err != nil {
		t.Fatalf("marshal review state: %v", err)
	}
	var reviewState astmerge.ConformanceManifestReviewState
	if err := json.Unmarshal(reviewStateSource, &reviewState); err != nil {
		t.Fatalf("decode review state: %v", err)
	}

	replayResult := MergeMarkdownWithReviewedNestedOutputsFromReplayBundle(
		fixture["template"].(string),
		fixture["destination"].(string),
		markdownmerge.DialectMarkdown,
		replayBundle,
	)
	if replayResult.OK || replayResult.Output != nil || len(replayResult.Diagnostics) != 1 || replayResult.Diagnostics[0].Message != expectedReplay["diagnostics"].([]any)[0].(map[string]any)["message"].(string) {
		t.Fatalf("unexpected replay-bundle rejection: %+v", replayResult)
	}

	stateResult := MergeMarkdownWithReviewedNestedOutputsFromReviewState(
		fixture["template"].(string),
		fixture["destination"].(string),
		markdownmerge.DialectMarkdown,
		reviewState,
	)
	if stateResult.OK || stateResult.Output != nil || len(stateResult.Diagnostics) != 1 || stateResult.Diagnostics[0].Message != expectedState["diagnostics"].([]any)[0].(map[string]any)["message"].(string) {
		t.Fatalf("unexpected review-state rejection: %+v", stateResult)
	}
}

func TestSharedFixtureMarkdownProviderReviewedNestedReviewArtifactEnvelopeApplication(t *testing.T) {
	providerFixture := readGoldmarkFixture(t, "diagnostics", "slice-328-markdown-provider-reviewed-nested-review-artifact-envelope-application", "go-markdown-provider-reviewed-nested-review-artifact-envelope-application.json")
	sharedFixturePathRaw := providerFixture["shared_fixture_path"].([]any)
	sharedFixturePath := make([]string, 0, len(sharedFixturePathRaw))
	for _, part := range sharedFixturePathRaw {
		sharedFixturePath = append(sharedFixturePath, part.(string))
	}
	fixture := readGoldmarkFixture(t, sharedFixturePath...)
	expected := providerFixture["providers"].(map[string]any)["goldmark"].(map[string]any)["expected"].(map[string]any)
	replayEnvelopeSource, err := json.Marshal(fixture["replay_bundle_envelope"])
	if err != nil {
		t.Fatalf("marshal replay bundle envelope: %v", err)
	}
	var replayEnvelope astmerge.ReviewReplayBundleEnvelope
	if err := json.Unmarshal(replayEnvelopeSource, &replayEnvelope); err != nil {
		t.Fatalf("decode replay bundle envelope: %v", err)
	}
	reviewStateEnvelopeSource, err := json.Marshal(fixture["review_state_envelope"])
	if err != nil {
		t.Fatalf("marshal review state envelope: %v", err)
	}
	var reviewStateEnvelope astmerge.ConformanceManifestReviewStateEnvelope
	if err := json.Unmarshal(reviewStateEnvelopeSource, &reviewStateEnvelope); err != nil {
		t.Fatalf("decode review state envelope: %v", err)
	}

	replayResult := MergeMarkdownWithReviewedNestedOutputsFromReplayBundleEnvelope(
		fixture["template"].(string),
		fixture["destination"].(string),
		markdownmerge.DialectMarkdown,
		replayEnvelope,
		"",
	)
	if !replayResult.OK || replayResult.Output == nil || *replayResult.Output != expected["output"].(string) {
		t.Fatalf("unexpected replay-bundle-envelope reviewed nested merge output: %+v", replayResult)
	}

	stateResult := MergeMarkdownWithReviewedNestedOutputsFromReviewStateEnvelope(
		fixture["template"].(string),
		fixture["destination"].(string),
		markdownmerge.DialectMarkdown,
		reviewStateEnvelope,
		"",
	)
	if !stateResult.OK || stateResult.Output == nil || *stateResult.Output != expected["output"].(string) {
		t.Fatalf("unexpected review-state-envelope reviewed nested merge output: %+v", stateResult)
	}
}

func TestSharedFixtureMarkdownProviderReviewedNestedReviewArtifactEnvelopeRejection(t *testing.T) {
	fixture := readGoldmarkFixture(t, "markdown", "slice-315-reviewed-nested-review-artifact-envelope-rejection", "fenced-code-reviewed-nested-review-artifact-envelope-rejection.json")
	replayEnvelopeSource, err := json.Marshal(fixture["replay_bundle_envelope"])
	if err != nil {
		t.Fatalf("marshal replay bundle envelope: %v", err)
	}
	var replayEnvelope astmerge.ReviewReplayBundleEnvelope
	if err := json.Unmarshal(replayEnvelopeSource, &replayEnvelope); err != nil {
		t.Fatalf("decode replay bundle envelope: %v", err)
	}
	reviewStateEnvelopeSource, err := json.Marshal(fixture["review_state_envelope"])
	if err != nil {
		t.Fatalf("marshal review state envelope: %v", err)
	}
	var reviewStateEnvelope astmerge.ConformanceManifestReviewStateEnvelope
	if err := json.Unmarshal(reviewStateEnvelopeSource, &reviewStateEnvelope); err != nil {
		t.Fatalf("decode review state envelope: %v", err)
	}

	replayResult := MergeMarkdownWithReviewedNestedOutputsFromReplayBundleEnvelope(
		fixture["template"].(string),
		fixture["destination"].(string),
		markdownmerge.DialectMarkdown,
		replayEnvelope,
		"",
	)
	if replayResult.OK || replayResult.Output != nil || len(replayResult.Diagnostics) != 1 || replayResult.Diagnostics[0].Message != fixture["expected_replay_bundle"].(map[string]any)["diagnostics"].([]any)[0].(map[string]any)["message"].(string) {
		t.Fatalf("unexpected replay-bundle-envelope rejection: %+v", replayResult)
	}

	stateResult := MergeMarkdownWithReviewedNestedOutputsFromReviewStateEnvelope(
		fixture["template"].(string),
		fixture["destination"].(string),
		markdownmerge.DialectMarkdown,
		reviewStateEnvelope,
		"",
	)
	if stateResult.OK || stateResult.Output != nil || len(stateResult.Diagnostics) != 1 || stateResult.Diagnostics[0].Message != fixture["expected_review_state"].(map[string]any)["diagnostics"].([]any)[0].(map[string]any)["message"].(string) {
		t.Fatalf("unexpected review-state-envelope rejection: %+v", stateResult)
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

	replayEnvelopeResult := MergeMarkdownWithReviewedNestedOutputsFromReplayBundleEnvelope(
		"# Title\n",
		"# Title\n",
		markdownmerge.DialectMarkdown,
		astmerge.ReviewReplayBundleEnvelope{},
		"kreuzberg-language-pack",
	)
	if replayEnvelopeResult.OK {
		t.Fatalf("expected replay envelope failure: %+v", replayEnvelopeResult)
	}
	if actual := jsonReady(t, replayEnvelopeResult.Diagnostics); !reflect.DeepEqual(actual, jsonReady(t, expectedDiagnostics)) {
		t.Fatalf("unexpected replay envelope diagnostics: %+v", actual)
	}

	stateEnvelopeResult := MergeMarkdownWithReviewedNestedOutputsFromReviewStateEnvelope(
		"# Title\n",
		"# Title\n",
		markdownmerge.DialectMarkdown,
		astmerge.ConformanceManifestReviewStateEnvelope{},
		"kreuzberg-language-pack",
	)
	if stateEnvelopeResult.OK {
		t.Fatalf("expected state envelope failure: %+v", stateEnvelopeResult)
	}
	if actual := jsonReady(t, stateEnvelopeResult.Diagnostics); !reflect.DeepEqual(actual, jsonReady(t, expectedDiagnostics)) {
		t.Fatalf("unexpected state envelope diagnostics: %+v", actual)
	}
}

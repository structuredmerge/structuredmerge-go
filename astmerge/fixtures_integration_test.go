package astmerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

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

func readManifest(t *testing.T) ConformanceManifest {
	t.Helper()

	path := filepath.Join("..", "..", "fixtures", "conformance", "slice-24-manifest", "family-feature-profiles.json")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}

	var manifest ConformanceManifest
	if err := json.Unmarshal(source, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}

	return manifest
}

func diagnosticsFixturePath(t *testing.T, role string) string {
	t.Helper()

	manifest := readManifest(t)
	path := ConformanceFixturePath(manifest, "diagnostics", role)
	if path == nil {
		t.Fatalf("missing diagnostics fixture entry for %s", role)
	}

	return filepath.Join(append([]string{"..", "..", "fixtures"}, path...)...)
}

func TestSharedFixtureDiagnosticVocabulary(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "diagnostic_vocabulary"))

	severities := []DiagnosticSeverity{
		SeverityInfo,
		SeverityWarning,
		SeverityError,
	}
	categories := []DiagnosticCategory{
		CategoryParseError,
		CategoryDestinationParseError,
		CategoryUnsupportedFeature,
		CategoryFallbackApplied,
		CategoryAmbiguity,
	}

	expectedSeverities := fixture["severities"].([]any)
	if len(severities) != len(expectedSeverities) {
		t.Fatalf("unexpected severities: %+v", severities)
	}
	for index, severity := range severities {
		if string(severity) != expectedSeverities[index].(string) {
			t.Fatalf("unexpected severity at %d: %s", index, severity)
		}
	}

	expectedCategories := fixture["categories"].([]any)
	if len(categories) != len(expectedCategories) {
		t.Fatalf("unexpected categories: %+v", categories)
	}
	for index, category := range categories {
		if string(category) != expectedCategories[index].(string) {
			t.Fatalf("unexpected category at %d: %s", index, category)
		}
	}
}

func TestSharedFixturePolicyVocabulary(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "policy_vocabulary"))

	surfaces := []PolicySurface{
		PolicySurfaceFallback,
		PolicySurfaceArray,
	}
	policies := []PolicyReference{
		{
			Surface: PolicySurfaceFallback,
			Name:    "trailing_comma_destination_fallback",
		},
		{
			Surface: PolicySurfaceArray,
			Name:    "destination_wins_array",
		},
	}

	expectedSurfaces := fixture["surfaces"].([]any)
	if len(surfaces) != len(expectedSurfaces) {
		t.Fatalf("unexpected surfaces: %+v", surfaces)
	}
	for index, surface := range surfaces {
		if string(surface) != expectedSurfaces[index].(string) {
			t.Fatalf("unexpected surface at %d: %s", index, surface)
		}
	}

	expectedPolicies := fixture["policies"].([]any)
	if len(policies) != len(expectedPolicies) {
		t.Fatalf("unexpected policies: %+v", policies)
	}
	for index, policy := range policies {
		expected := expectedPolicies[index].(map[string]any)
		if string(policy.Surface) != expected["surface"].(string) || policy.Name != expected["name"].(string) {
			t.Fatalf("unexpected policy at %d: %+v", index, policy)
		}
	}
}

func TestSharedFixturePolicyReporting(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "policy_reporting"))

	policies := []PolicyReference{
		{
			Surface: PolicySurfaceArray,
			Name:    "destination_wins_array",
		},
		{
			Surface: PolicySurfaceFallback,
			Name:    "trailing_comma_destination_fallback",
		},
	}

	expectedPolicies := fixture["merge_policies"].([]any)
	if len(policies) != len(expectedPolicies) {
		t.Fatalf("unexpected policies: %+v", policies)
	}
	for index, policy := range policies {
		expected := expectedPolicies[index].(map[string]any)
		if string(policy.Surface) != expected["surface"].(string) || policy.Name != expected["name"].(string) {
			t.Fatalf("unexpected policy at %d: %+v", index, policy)
		}
	}
}

func TestSharedFixtureFamilyFeatureProfile(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "shared_family_feature_profile"))

	profile := FamilyFeatureProfile{
		Family:            "example",
		SupportedDialects: []string{"alpha", "beta"},
		SupportedPolicies: []PolicyReference{
			{
				Surface: PolicySurfaceArray,
				Name:    "destination_wins_array",
			},
		},
	}

	expected := fixture["feature_profile"].(map[string]any)
	if profile.Family != expected["family"].(string) {
		t.Fatalf("unexpected family: %+v", profile)
	}
	expectedDialects := expected["supported_dialects"].([]any)
	if len(profile.SupportedDialects) != len(expectedDialects) {
		t.Fatalf("unexpected supported dialects: %+v", profile.SupportedDialects)
	}
	for index, dialect := range profile.SupportedDialects {
		if dialect != expectedDialects[index].(string) {
			t.Fatalf("unexpected dialect at %d: %s", index, dialect)
		}
	}
	assertExpectedPolicies(t, profile.SupportedPolicies, expected["supported_policies"].([]any))
}

func TestSharedFixtureConformanceRunnerShape(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "runner_shape"))

	caseRef := ConformanceCaseRef{
		Family: "json",
		Role:   "tree_sitter_adapter",
		Case:   "valid_strict_json",
	}
	result := ConformanceCaseResult{
		Ref:      caseRef,
		Outcome:  ConformancePassed,
		Messages: []string{},
	}

	expectedRef := fixture["case_ref"].(map[string]any)
	if caseRef.Family != expectedRef["family"].(string) ||
		caseRef.Role != expectedRef["role"].(string) ||
		caseRef.Case != expectedRef["case"].(string) {
		t.Fatalf("unexpected case ref: %+v", caseRef)
	}

	expectedResult := fixture["result"].(map[string]any)
	expectedResultRef := expectedResult["ref"].(map[string]any)
	if result.Ref.Family != expectedResultRef["family"].(string) ||
		result.Ref.Role != expectedResultRef["role"].(string) ||
		result.Ref.Case != expectedResultRef["case"].(string) ||
		string(result.Outcome) != expectedResult["outcome"].(string) {
		t.Fatalf("unexpected runner result: %+v", result)
	}
	if len(result.Messages) != len(expectedResult["messages"].([]any)) {
		t.Fatalf("unexpected runner messages: %+v", result.Messages)
	}
}

func TestSharedFixtureNormalizedManifestContract(t *testing.T) {
	manifest := readManifest(t)

	jsonProfilePath := ConformanceFamilyFeatureProfilePath(manifest, "json")
	if filepath.Join(jsonProfilePath...) != filepath.Join("diagnostics", "slice-21-family-feature-profile", "json-feature-profile.json") {
		t.Fatalf("unexpected json family profile path: %v", jsonProfilePath)
	}

	textAnalysisPath := ConformanceFixturePath(manifest, "text", "analysis")
	if filepath.Join(textAnalysisPath...) != filepath.Join("text", "slice-03-analysis", "whitespace-and-blocks.json") {
		t.Fatalf("unexpected text analysis path: %v", textAnalysisPath)
	}

	runnerShapePath := ConformanceFixturePath(manifest, "diagnostics", "runner_shape")
	if filepath.Join(runnerShapePath...) != filepath.Join("diagnostics", "slice-28-conformance-runner", "runner-shape.json") {
		t.Fatalf("unexpected runner shape path: %v", runnerShapePath)
	}
}

func TestSharedFixtureConformanceSuiteSummary(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "runner_summary"))

	rawResults := fixture["results"].([]any)
	results := make([]ConformanceCaseResult, 0, len(rawResults))
	for _, item := range rawResults {
		entry := item.(map[string]any)
		ref := entry["ref"].(map[string]any)
		messages := entry["messages"].([]any)
		normalizedMessages := make([]string, 0, len(messages))
		for _, message := range messages {
			normalizedMessages = append(normalizedMessages, message.(string))
		}

		results = append(results, ConformanceCaseResult{
			Ref: ConformanceCaseRef{
				Family: ref["family"].(string),
				Role:   ref["role"].(string),
				Case:   ref["case"].(string),
			},
			Outcome:  ConformanceOutcome(entry["outcome"].(string)),
			Messages: normalizedMessages,
		})
	}

	expected := fixture["summary"].(map[string]any)
	summary := SummarizeConformanceResults(results)
	if summary.Total != int(expected["total"].(float64)) ||
		summary.Passed != int(expected["passed"].(float64)) ||
		summary.Failed != int(expected["failed"].(float64)) ||
		summary.Skipped != int(expected["skipped"].(float64)) {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}

func assertExpectedPolicies(t *testing.T, policies []PolicyReference, expected []any) {
	t.Helper()

	if len(policies) != len(expected) {
		t.Fatalf("unexpected policies: %+v", policies)
	}
	for index, policy := range policies {
		expectedPolicy := expected[index].(map[string]any)
		if string(policy.Surface) != expectedPolicy["surface"].(string) || policy.Name != expectedPolicy["name"].(string) {
			t.Fatalf("unexpected policy at %d: %+v", index, policy)
		}
	}
}

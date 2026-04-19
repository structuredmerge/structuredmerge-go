package astmerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func readDiagnosticFixture(t *testing.T, parts ...string) map[string]any {
	t.Helper()

	pathParts := append([]string{"..", "..", "fixtures"}, parts...)
	path := filepath.Join(pathParts...)
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

func TestSharedFixtureDiagnosticVocabulary(t *testing.T) {
	fixture := readDiagnosticFixture(t, "diagnostics", "slice-02-core", "diagnostic-categories.json")

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
	fixture := readDiagnosticFixture(t, "diagnostics", "slice-17-policy-vocabulary", "policy-references.json")

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
	fixture := readDiagnosticFixture(t, "diagnostics", "slice-18-policy-reporting", "result-policies.json")

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

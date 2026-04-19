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

func diagnosticsFixturePath(t *testing.T, role string) string {
	t.Helper()

	manifest := readDiagnosticFixture(t, "conformance", "slice-24-manifest", "family-feature-profiles.json")
	entries := manifest["diagnostics"].([]any)
	for _, item := range entries {
		entry := item.(map[string]any)
		if entry["role"].(string) != role {
			continue
		}

		parts := []string{"..", "..", "fixtures"}
		for _, segment := range entry["path"].([]any) {
			parts = append(parts, segment.(string))
		}
		return filepath.Join(parts...)
	}

	t.Fatalf("missing diagnostics fixture entry for %s", role)
	return ""
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

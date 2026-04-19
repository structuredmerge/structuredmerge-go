package jsonmerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
)

func readJSONFixture(t *testing.T, parts ...string) map[string]any {
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

func readJSONFixtureFromPath(t *testing.T, path string) map[string]any {
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

func familyFeatureProfileFixturePath(t *testing.T, family string) string {
	t.Helper()

	manifest := readJSONFixture(t, "conformance", "slice-24-manifest", "family-feature-profiles.json")
	entries := manifest["family_feature_profiles"].([]any)
	for _, item := range entries {
		entry := item.(map[string]any)
		if entry["family"].(string) != family {
			continue
		}

		parts := []string{"..", "..", "fixtures"}
		for _, segment := range entry["path"].([]any) {
			parts = append(parts, segment.(string))
		}
		return filepath.Join(parts...)
	}

	t.Fatalf("missing family feature profile entry for %s", family)
	return ""
}

func TestSharedFixtureJSONCCommentsAccepted(t *testing.T) {
	fixture := readJSONFixture(t, "jsonc", "slice-04-parse", "comments-accepted.json")
	expected := fixture["expected"].(map[string]any)

	result := ParseJSON(fixture["source"].(string), DialectJSONC)
	if result.OK != expected["ok"].(bool) {
		t.Fatalf("unexpected parse status: %+v", result)
	}
	if result.Analysis == nil || result.Analysis.AllowsComments != expected["allows_comments"].(bool) {
		t.Fatalf("unexpected analysis: %+v", result.Analysis)
	}
	if len(result.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", result.Diagnostics)
	}
}

func TestSharedFixtureJSONStructure(t *testing.T) {
	objectFixture := readJSONFixture(t, "json", "slice-07-structure", "object-and-array.json")
	objectExpected := objectFixture["expected"].(map[string]any)

	objectResult := ParseJSON(objectFixture["source"].(string), DialectJSON)
	if !objectResult.OK || objectResult.Analysis == nil {
		t.Fatalf("expected parse success, got %+v", objectResult)
	}
	if string(objectResult.Analysis.RootKind) != objectExpected["root_kind"].(string) {
		t.Fatalf("unexpected root kind: %s", objectResult.Analysis.RootKind)
	}

	expectedOwners := objectExpected["owners"].([]any)
	if len(objectResult.Analysis.Owners) != len(expectedOwners) {
		t.Fatalf("unexpected owners: %+v", objectResult.Analysis.Owners)
	}
	for index, item := range expectedOwners {
		expected := item.(map[string]any)
		owner := objectResult.Analysis.Owners[index]
		if owner.Path != expected["path"].(string) || string(owner.OwnerKind) != expected["owner_kind"].(string) {
			t.Fatalf("unexpected owner at %d: %+v", index, owner)
		}
		if expectedMatchKey, ok := expected["match_key"]; ok && owner.MatchKey != expectedMatchKey.(string) {
			t.Fatalf("unexpected match_key at %d: %+v", index, owner)
		}
	}

	jsoncFixture := readJSONFixture(t, "jsonc", "slice-07-structure", "commented-object.json")
	jsoncExpected := jsoncFixture["expected"].(map[string]any)

	jsoncResult := ParseJSON(jsoncFixture["source"].(string), DialectJSONC)
	if !jsoncResult.OK || jsoncResult.Analysis == nil {
		t.Fatalf("expected parse success, got %+v", jsoncResult)
	}
	if string(jsoncResult.Analysis.RootKind) != jsoncExpected["root_kind"].(string) {
		t.Fatalf("unexpected root kind: %s", jsoncResult.Analysis.RootKind)
	}

	expectedJSONCOwners := jsoncExpected["owners"].([]any)
	if len(jsoncResult.Analysis.Owners) != len(expectedJSONCOwners) {
		t.Fatalf("unexpected owners: %+v", jsoncResult.Analysis.Owners)
	}
	for index, item := range expectedJSONCOwners {
		expected := item.(map[string]any)
		owner := jsoncResult.Analysis.Owners[index]
		if owner.Path != expected["path"].(string) || string(owner.OwnerKind) != expected["owner_kind"].(string) {
			t.Fatalf("unexpected owner at %d: %+v", index, owner)
		}
		if expectedMatchKey, ok := expected["match_key"]; ok && owner.MatchKey != expectedMatchKey.(string) {
			t.Fatalf("unexpected match_key at %d: %+v", index, owner)
		}
	}
}

func TestSharedFixtureJSONOwnerMatching(t *testing.T) {
	fixture := readJSONFixture(t, "json", "slice-08-matching", "path-equality.json")
	expected := fixture["expected"].(map[string]any)

	template := ParseJSON(fixture["template"].(string), DialectJSON)
	destination := ParseJSON(fixture["destination"].(string), DialectJSON)
	if template.Analysis == nil || destination.Analysis == nil {
		t.Fatalf("expected parse success for both documents")
	}

	result := MatchJSONOwners(*template.Analysis, *destination.Analysis)
	expectedMatched := expected["matched"].([]any)
	if len(result.Matched) != len(expectedMatched) {
		t.Fatalf("unexpected matched owners: %+v", result.Matched)
	}
	for index, item := range expectedMatched {
		pair := item.([]any)
		if result.Matched[index].TemplatePath != pair[0].(string) ||
			result.Matched[index].DestinationPath != pair[1].(string) {
			t.Fatalf("unexpected matched owner at %d: %+v", index, result.Matched[index])
		}
	}

	unmatchedTemplate := expected["unmatched_template"].([]any)
	if len(result.UnmatchedTemplate) != len(unmatchedTemplate) {
		t.Fatalf("unexpected unmatched template owners: %+v", result.UnmatchedTemplate)
	}
	for index, item := range unmatchedTemplate {
		if result.UnmatchedTemplate[index] != item.(string) {
			t.Fatalf("unexpected unmatched template owners: %+v", result.UnmatchedTemplate)
		}
	}

	unmatchedDestination := expected["unmatched_destination"].([]any)
	if len(result.UnmatchedDestination) != len(unmatchedDestination) {
		t.Fatalf("unexpected unmatched destination owners: %+v", result.UnmatchedDestination)
	}
	for index, item := range unmatchedDestination {
		if result.UnmatchedDestination[index] != item.(string) {
			t.Fatalf("unexpected unmatched destination owners: %+v", result.UnmatchedDestination)
		}
	}
}

func TestSharedFixtureJSONObjectMerge(t *testing.T) {
	fixture := readJSONFixture(t, "json", "slice-09-merge", "object-merge.json")
	expected := fixture["expected"].(map[string]any)

	result := MergeJSON(
		fixture["template"].(string),
		fixture["destination"].(string),
		DialectJSON,
	)
	if !result.OK || result.Output == nil {
		t.Fatalf("expected merge success, got diagnostics: %+v", result.Diagnostics)
	}
	if *result.Output != expected["output"].(string) {
		t.Fatalf("unexpected merged output: %q", *result.Output)
	}
}

func TestSharedFixtureJSONInvalidMerges(t *testing.T) {
	invalidTemplateFixture := readJSONFixture(t, "json", "slice-09-merge", "invalid-template.json")
	invalidTemplateExpected := invalidTemplateFixture["expected"].(map[string]any)

	invalidTemplateResult := MergeJSON(
		invalidTemplateFixture["template"].(string),
		invalidTemplateFixture["destination"].(string),
		DialectJSON,
	)
	if invalidTemplateResult.OK {
		t.Fatalf("expected template parse failure")
	}
	if invalidTemplateResult.Output != nil {
		t.Fatalf("expected no output, got %q", *invalidTemplateResult.Output)
	}
	assertExpectedDiagnostics(t, invalidTemplateResult.Diagnostics, invalidTemplateExpected["diagnostics"].([]any))

	invalidDestinationFixture := readJSONFixture(t, "json", "slice-09-merge", "invalid-destination.json")
	invalidDestinationExpected := invalidDestinationFixture["expected"].(map[string]any)

	invalidDestinationResult := MergeJSON(
		invalidDestinationFixture["template"].(string),
		invalidDestinationFixture["destination"].(string),
		DialectJSON,
	)
	if invalidDestinationResult.OK {
		t.Fatalf("expected destination parse failure")
	}
	if invalidDestinationResult.Output != nil {
		t.Fatalf("expected no output, got %q", *invalidDestinationResult.Output)
	}
	assertExpectedDiagnostics(
		t,
		invalidDestinationResult.Diagnostics,
		invalidDestinationExpected["diagnostics"].([]any),
	)
}

func TestSharedFixtureJSONFallbackMerge(t *testing.T) {
	fixture := readJSONFixture(t, "json", "slice-14-fallback", "trailing-comma-destination.json")
	expected := fixture["expected"].(map[string]any)

	result := MergeJSON(
		fixture["template"].(string),
		fixture["destination"].(string),
		DialectJSON,
	)
	if result.OK != expected["ok"].(bool) {
		t.Fatalf("unexpected merge status: %+v", result)
	}
	assertExpectedDiagnostics(t, result.Diagnostics, expected["diagnostics"].([]any))
	assertExpectedPolicies(t, result.Policies, []astmerge.PolicyReference{
		{Surface: astmerge.PolicySurfaceArray, Name: "destination_wins_array"},
		{Surface: astmerge.PolicySurfaceFallback, Name: "trailing_comma_destination_fallback"},
	})
	if result.Output == nil || *result.Output != expected["output"].(string) {
		t.Fatalf("unexpected output: %+v", result.Output)
	}
}

func TestSharedFixtureJSONFallbackBoundaries(t *testing.T) {
	templateFixture := readJSONFixture(t, "json", "slice-15-fallback-boundaries", "template-trailing-comma-not-recovered.json")
	templateExpected := templateFixture["expected"].(map[string]any)

	templateResult := MergeJSON(
		templateFixture["template"].(string),
		templateFixture["destination"].(string),
		DialectJSON,
	)
	if templateResult.OK != templateExpected["ok"].(bool) {
		t.Fatalf("unexpected merge status: %+v", templateResult)
	}
	assertExpectedDiagnostics(t, templateResult.Diagnostics, templateExpected["diagnostics"].([]any))
	if templateResult.Output != nil {
		t.Fatalf("expected no output: %+v", templateResult.Output)
	}

	commentsFixture := readJSONFixture(t, "json", "slice-15-fallback-boundaries", "strict-json-comments-not-recovered.json")
	commentsExpected := commentsFixture["expected"].(map[string]any)

	commentsResult := MergeJSON(
		commentsFixture["template"].(string),
		commentsFixture["destination"].(string),
		DialectJSON,
	)
	if commentsResult.OK != commentsExpected["ok"].(bool) {
		t.Fatalf("unexpected merge status: %+v", commentsResult)
	}
	assertExpectedDiagnostics(t, commentsResult.Diagnostics, commentsExpected["diagnostics"].([]any))
	if commentsResult.Output != nil {
		t.Fatalf("expected no output: %+v", commentsResult.Output)
	}
}

func TestSharedFixtureJSONArrayPolicy(t *testing.T) {
	fixture := readJSONFixture(t, "json", "slice-16-array-policy", "destination-wins-array.json")
	expected := fixture["expected"].(map[string]any)

	result := MergeJSON(
		fixture["template"].(string),
		fixture["destination"].(string),
		DialectJSON,
	)
	if result.OK != expected["ok"].(bool) {
		t.Fatalf("unexpected merge status: %+v", result)
	}
	if len(result.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", result.Diagnostics)
	}
	assertExpectedPolicies(t, result.Policies, []astmerge.PolicyReference{
		{Surface: astmerge.PolicySurfaceArray, Name: "destination_wins_array"},
	})
	if result.Output == nil || *result.Output != expected["output"].(string) {
		t.Fatalf("unexpected output: %+v", result.Output)
	}
}

func TestSharedFixtureJSONFamilyFeatureProfile(t *testing.T) {
	fixture := readJSONFixtureFromPath(t, familyFeatureProfileFixturePath(t, "json"))
	expected := fixture["feature_profile"].(map[string]any)

	profile := JSONFeatureProfileInfo()

	if profile.Family != expected["family"].(string) {
		t.Fatalf("unexpected family: %+v", profile)
	}
	expectedDialects := expected["supported_dialects"].([]any)
	if len(profile.SupportedDialects) != len(expectedDialects) {
		t.Fatalf("unexpected supported dialects: %+v", profile.SupportedDialects)
	}
	for index, dialect := range profile.SupportedDialects {
		if string(dialect) != expectedDialects[index].(string) {
			t.Fatalf("unexpected dialect at %d: %s", index, dialect)
		}
	}
	assertExpectedPolicies(t, profile.SupportedPolicies, []astmerge.PolicyReference{
		{Surface: astmerge.PolicySurfaceArray, Name: "destination_wins_array"},
		{Surface: astmerge.PolicySurfaceFallback, Name: "trailing_comma_destination_fallback"},
	})
}

func assertExpectedDiagnostics(t *testing.T, diagnostics []astmerge.Diagnostic, expected []any) {
	t.Helper()

	if len(diagnostics) != len(expected) {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics)
	}

	for index, item := range expected {
		expectedDiagnostic := item.(map[string]any)
		diagnostic := diagnostics[index]

		if string(diagnostic.Severity) != expectedDiagnostic["severity"].(string) {
			t.Fatalf("unexpected severity at %d: %+v", index, diagnostic)
		}
		if string(diagnostic.Category) != expectedDiagnostic["category"].(string) {
			t.Fatalf("unexpected category at %d: %+v", index, diagnostic)
		}
	}
}

func assertExpectedPolicies(t *testing.T, policies []astmerge.PolicyReference, expected []astmerge.PolicyReference) {
	t.Helper()

	if len(policies) != len(expected) {
		t.Fatalf("unexpected policies: %+v", policies)
	}
	for index, policy := range policies {
		if policy != expected[index] {
			t.Fatalf("unexpected policy at %d: %+v", index, policy)
		}
	}
}

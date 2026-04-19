package jsonmerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/treehaver"
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

	manifestFixture := readJSONFixture(t, "conformance", "slice-24-manifest", "family-feature-profiles.json")
	manifestSource, err := json.Marshal(manifestFixture)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	var manifest astmerge.ConformanceManifest
	if err := json.Unmarshal(manifestSource, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	path := astmerge.ConformanceFamilyFeatureProfilePath(manifest, family)
	if path == nil {
		t.Fatalf("missing family feature profile entry for %s", family)
	}

	return filepath.Join(append([]string{"..", "..", "fixtures"}, path...)...)
}

func jsonFixturePath(t *testing.T, role string) string {
	t.Helper()

	manifestFixture := readJSONFixture(t, "conformance", "slice-24-manifest", "family-feature-profiles.json")
	manifestSource, err := json.Marshal(manifestFixture)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	var manifest astmerge.ConformanceManifest
	if err := json.Unmarshal(manifestSource, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	path := astmerge.ConformanceFixturePath(manifest, "json", role)
	if path == nil {
		t.Fatalf("missing json fixture entry for %s", role)
	}

	return filepath.Join(append([]string{"..", "..", "fixtures"}, path...)...)
}

func TestSharedFixtureTreeSitterAdapter(t *testing.T) {
	fixture := readJSONFixtureFromPath(t, jsonFixturePath(t, "tree_sitter_adapter"))
	cases := fixture["cases"].([]any)

	for _, item := range cases {
		testCase := item.(map[string]any)
		expected := testCase["expected"].(map[string]any)
		var dialect JSONDialect
		if testCase["dialect"].(string) == "jsonc" {
			dialect = DialectJSONC
		} else {
			dialect = DialectJSON
		}

		result := ParseJSONWithLanguagePack(testCase["source"].(string), dialect)
		if result.OK != expected["ok"].(bool) {
			t.Fatalf("unexpected parse status for %s: %+v", testCase["name"].(string), result)
		}

		expectedDiagnostics := expected["diagnostics"].([]any)
		if len(result.Diagnostics) != len(expectedDiagnostics) {
			t.Fatalf("unexpected diagnostics for %s: %+v", testCase["name"].(string), result.Diagnostics)
		}
		for index, item := range expectedDiagnostics {
			expectedDiagnostic := item.(map[string]any)
			diagnostic := result.Diagnostics[index]
			if string(diagnostic.Severity) != expectedDiagnostic["severity"].(string) ||
				string(diagnostic.Category) != expectedDiagnostic["category"].(string) ||
				diagnostic.Message != expectedDiagnostic["message"].(string) {
				t.Fatalf("unexpected diagnostic at %d for %s: %+v", index, testCase["name"].(string), diagnostic)
			}
		}

		if result.OK {
			if result.Analysis == nil {
				t.Fatalf("expected analysis for %s", testCase["name"].(string))
			}
			if string(result.Analysis.RootKind) != expected["root_kind"].(string) {
				t.Fatalf("unexpected root kind for %s: %s", testCase["name"].(string), result.Analysis.RootKind)
			}
			expectedOwners := expected["owners"].([]any)
			if len(result.Analysis.Owners) != len(expectedOwners) {
				t.Fatalf("unexpected owners for %s: %+v", testCase["name"].(string), result.Analysis.Owners)
			}
			for index, item := range expectedOwners {
				expectedOwner := item.(map[string]any)
				owner := result.Analysis.Owners[index]
				if owner.Path != expectedOwner["path"].(string) ||
					string(owner.OwnerKind) != expectedOwner["owner_kind"].(string) {
					t.Fatalf("unexpected owner at %d for %s: %+v", index, testCase["name"].(string), owner)
				}
				if expectedMatchKey, ok := expectedOwner["match_key"]; ok && owner.MatchKey != expectedMatchKey.(string) {
					t.Fatalf("unexpected match key at %d for %s: %+v", index, testCase["name"].(string), owner)
				}
			}
		} else if result.Analysis != nil {
			t.Fatalf("expected no analysis for %s: %+v", testCase["name"].(string), result.Analysis)
		}
	}
}

func TestCapabilityAwareSelectionForTreeSitterAdapterCases(t *testing.T) {
	profile := JSONFeatureProfileInfo()
	familyProfile := astmerge.FamilyFeatureProfile{
		Family:            profile.Family,
		SupportedDialects: []string{},
		SupportedPolicies: profile.SupportedPolicies,
	}
	for _, dialect := range profile.SupportedDialects {
		familyProfile.SupportedDialects = append(familyProfile.SupportedDialects, string(dialect))
	}

	adapterInfo := treehaver.LanguagePackAdapterInfo()
	featureProfile := &astmerge.ConformanceFeatureProfileView{
		Backend:           adapterInfo.Backend,
		SupportsDialects:  adapterInfo.SupportsDialects,
		SupportedPolicies: adapterInfo.SupportedPolicies,
	}

	selected := astmerge.SelectConformanceCase(
		astmerge.ConformanceCaseRef{
			Family: "json",
			Role:   "tree_sitter_adapter",
			Case:   "valid_strict_json",
		},
		astmerge.ConformanceCaseRequirements{
			Dialect: "json",
		},
		familyProfile,
		featureProfile,
	)
	if selected.Status != astmerge.ConformanceSelected || len(selected.Messages) != 0 {
		t.Fatalf("unexpected selected case: %+v", selected)
	}

	skipped := astmerge.SelectConformanceCase(
		astmerge.ConformanceCaseRef{
			Family: "json",
			Role:   "tree_sitter_adapter",
			Case:   "jsonc_unsupported",
		},
		astmerge.ConformanceCaseRequirements{
			Dialect: "jsonc",
		},
		familyProfile,
		featureProfile,
	)
	if skipped.Status != astmerge.ConformanceSelectionSkipped {
		t.Fatalf("expected skipped case: %+v", skipped)
	}
	expectedMessage := "backend kreuzberg-language-pack does not support dialect jsonc for family json."
	if len(skipped.Messages) != 1 || skipped.Messages[0] != expectedMessage {
		t.Fatalf("unexpected skipped messages: %+v", skipped.Messages)
	}
}

func TestSharedFixtureJSONCCommentsAccepted(t *testing.T) {
	fixture := readJSONFixtureFromPath(t, jsonFixturePath(t, "parse_comments"))
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
	objectFixture := readJSONFixtureFromPath(t, jsonFixturePath(t, "structure_json"))
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

	jsoncFixture := readJSONFixtureFromPath(t, jsonFixturePath(t, "structure_jsonc"))
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
	fixture := readJSONFixtureFromPath(t, jsonFixturePath(t, "matching"))
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
	fixture := readJSONFixtureFromPath(t, jsonFixturePath(t, "merge_object"))
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
	invalidTemplateFixture := readJSONFixtureFromPath(t, jsonFixturePath(t, "merge_invalid_template"))
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

	invalidDestinationFixture := readJSONFixtureFromPath(t, jsonFixturePath(t, "merge_invalid_destination"))
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
	fixture := readJSONFixtureFromPath(t, jsonFixturePath(t, "fallback"))
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
	templateFixture := readJSONFixtureFromPath(t, jsonFixturePath(t, "fallback_boundary_template"))
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

	commentsFixture := readJSONFixtureFromPath(t, jsonFixturePath(t, "fallback_boundary_comments"))
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
	fixture := readJSONFixtureFromPath(t, jsonFixturePath(t, "array_policy"))
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

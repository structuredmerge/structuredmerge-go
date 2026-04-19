package textmerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
)

func readTextFixture(t *testing.T, parts ...string) map[string]any {
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

func readTextFixtureFromPath(t *testing.T, path string) map[string]any {
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

	manifest := readTextFixture(t, "conformance", "slice-24-manifest", "family-feature-profiles.json")
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

func asMapList(value any) []map[string]any {
	raw := value.([]any)
	result := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		result = append(result, item.(map[string]any))
	}
	return result
}

func asIntList(value any) []int {
	raw := value.([]any)
	result := make([]int, 0, len(raw))
	for _, item := range raw {
		result = append(result, int(item.(float64)))
	}
	return result
}

func TestSharedFixtureAnalyzeText(t *testing.T) {
	fixture := readTextFixture(t, "text", "slice-03-analysis", "whitespace-and-blocks.json")
	source := fixture["source"].(string)
	expected := fixture["expected"].(map[string]any)

	analysis := AnalyzeText(source)
	if analysis.NormalizedSource != expected["normalized_source"].(string) {
		t.Fatalf("unexpected normalized source: %q", analysis.NormalizedSource)
	}

	expectedBlocks := asMapList(expected["blocks"])
	if len(analysis.Blocks) != len(expectedBlocks) {
		t.Fatalf("unexpected block count: %d", len(analysis.Blocks))
	}

	for index, block := range expectedBlocks {
		if analysis.Blocks[index].Index != int(block["index"].(float64)) {
			t.Fatalf("unexpected block index at %d: %+v", index, analysis.Blocks[index])
		}
		if analysis.Blocks[index].Normalized != block["normalized"].(string) {
			t.Fatalf("unexpected block at %d: %+v", index, analysis.Blocks[index])
		}
	}
}

func TestSharedFixtureExactMatching(t *testing.T) {
	fixture := readTextFixture(t, "text", "slice-11-matching", "exact-content.json")
	expected := fixture["expected"].(map[string]any)

	result := MatchTextBlocks(fixture["template"].(string), fixture["destination"].(string))

	rawMatched := expected["matched"].([]any)
	if len(result.Matched) != len(rawMatched) {
		t.Fatalf("unexpected matched count: %+v", result.Matched)
	}
	for index, item := range rawMatched {
		pair := item.([]any)
		if result.Matched[index].TemplateIndex != int(pair[0].(float64)) ||
			result.Matched[index].DestinationIndex != int(pair[1].(float64)) {
			t.Fatalf("unexpected match at %d: %+v", index, result.Matched[index])
		}
	}

	if got, want := result.UnmatchedTemplate, asIntList(expected["unmatched_template"]); len(got) != len(want) {
		t.Fatalf("unexpected unmatched template: %+v", got)
	} else {
		for index := range want {
			if got[index] != want[index] {
				t.Fatalf("unexpected unmatched template: %+v", got)
			}
		}
	}

	if got, want := result.UnmatchedDestination, asIntList(expected["unmatched_destination"]); len(got) != len(want) {
		t.Fatalf("unexpected unmatched destination: %+v", got)
	} else {
		for index := range want {
			if got[index] != want[index] {
				t.Fatalf("unexpected unmatched destination: %+v", got)
			}
		}
	}
}

func TestSharedFixtureSimilarity(t *testing.T) {
	fixture := readTextFixture(t, "text", "slice-05-similarity", "similarity-cases.json")
	cases := fixture["cases"].([]any)

	for _, item := range cases {
		testCase := item.(map[string]any)
		left := testCase["left"].(string)
		right := testCase["right"].(string)
		expectedScore := testCase["expected_score"].(float64)
		threshold := testCase["threshold"].(float64)
		expectedMatch := testCase["expected_match"].(bool)

		if score := SimilarityScore(left, right); score != expectedScore {
			t.Fatalf("unexpected similarity score for %s: %v", testCase["name"].(string), score)
		}

		result := IsSimilar(left, right, threshold)
		if result.Score != expectedScore || result.Threshold != threshold || result.Matched != expectedMatch {
			t.Fatalf("unexpected similarity result for %s: %+v", testCase["name"].(string), result)
		}
	}
}

func TestSharedFixtureRefinedMatching(t *testing.T) {
	fixture := readTextFixture(t, "text", "slice-13-refined-matching", "content-refined-merge.json")
	expected := fixture["expected"].(map[string]any)

	result := MatchTextBlocks(fixture["template"].(string), fixture["destination"].(string))
	expectedMatched := asMapList(expected["matched"])
	if len(result.Matched) != len(expectedMatched) {
		t.Fatalf("unexpected matched count: %+v", result.Matched)
	}

	for index, match := range expectedMatched {
		if result.Matched[index].TemplateIndex != int(match["templateIndex"].(float64)) ||
			result.Matched[index].DestinationIndex != int(match["destinationIndex"].(float64)) ||
			string(result.Matched[index].Phase) != match["phase"].(string) {
			t.Fatalf("unexpected refined match at %d: %+v", index, result.Matched[index])
		}
	}

	if len(result.UnmatchedTemplate) != 0 || len(result.UnmatchedDestination) != 0 {
		t.Fatalf("unexpected unmatched entries: %+v %+v", result.UnmatchedTemplate, result.UnmatchedDestination)
	}

	merged := MergeText(fixture["template"].(string), fixture["destination"].(string))
	if !merged.OK || merged.Output == nil {
		t.Fatalf("expected merge success, got diagnostics: %+v", merged.Diagnostics)
	}
	if *merged.Output != expected["output"].(string) {
		t.Fatalf("unexpected merged output: %q", *merged.Output)
	}
}

func TestSharedFixtureTextFamilyFeatureProfile(t *testing.T) {
	fixture := readTextFixtureFromPath(t, familyFeatureProfileFixturePath(t, "text"))
	expected := fixture["feature_profile"].(map[string]any)

	profile := TextFeatureProfileInfo()
	if profile.Family != expected["family"].(string) {
		t.Fatalf("unexpected family: %+v", profile)
	}
	if len(profile.SupportedDialects) != 0 {
		t.Fatalf("unexpected supported dialects: %+v", profile.SupportedDialects)
	}
	assertExpectedPolicies(t, profile.SupportedPolicies, []astmerge.PolicyReference{})
}

func assertExpectedPolicies(t *testing.T, policies []astmerge.PolicyReference, expected []astmerge.PolicyReference) {
	t.Helper()

	if len(policies) != len(expected) {
		t.Fatalf("unexpected policies: %+v", policies)
	}
	for index, policy := range policies {
		if policy.Surface != expected[index].Surface || policy.Name != expected[index].Name {
			t.Fatalf("unexpected policy at %d: %+v", index, policy)
		}
	}
}

package rubymerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
)

func readRubyFixture(t *testing.T, parts ...string) map[string]any {
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

func TestRubyFixtures(t *testing.T) {
	profileFixture := readRubyFixture(t, "diagnostics", "slice-214-ruby-family-feature-profile", "ruby-feature-profile.json")
	if RubyFeatureProfileInfo().Family != profileFixture["feature_profile"].(map[string]any)["family"].(string) {
		t.Fatal("unexpected profile family")
	}

	analysisFixture := readRubyFixture(t, "ruby", "slice-218-analysis", "module-owners.json")
	analysis := ParseRuby(analysisFixture["source"].(string), DialectRuby)
	if !analysis.OK || analysis.Analysis == nil {
		t.Fatalf("unexpected analysis: %+v", analysis)
	}
	if len(analysis.Analysis.Owners) != len(analysisFixture["expected"].(map[string]any)["owners"].([]any)) {
		t.Fatalf("unexpected owners: %+v", analysis.Analysis.Owners)
	}

	matchingFixture := readRubyFixture(t, "ruby", "slice-219-matching", "path-equality.json")
	template := ParseRuby(matchingFixture["template"].(string), DialectRuby)
	destination := ParseRuby(matchingFixture["destination"].(string), DialectRuby)
	match := MatchRubyOwners(*template.Analysis, *destination.Analysis)
	if len(match.Matched) != len(matchingFixture["expected"].(map[string]any)["matched"].([]any)) {
		t.Fatalf("unexpected matches: %+v", match)
	}

	surfacesFixture := readRubyFixture(t, "ruby", "slice-220-discovered-surfaces", "doc-comment-surfaces.json")
	surfaceAnalysis := ParseRuby(surfacesFixture["source"].(string), DialectRuby)
	encodedSurfaces, err := json.Marshal(RubyDiscoveredSurfaces(*surfaceAnalysis.Analysis))
	if err != nil {
		t.Fatalf("marshal surfaces: %v", err)
	}
	var surfaceValue any
	if err := json.Unmarshal(encodedSurfaces, &surfaceValue); err != nil {
		t.Fatalf("unmarshal surfaces: %v", err)
	}
	if !deepEqualJSON(surfaceValue, surfacesFixture["expected"]) {
		t.Fatalf("unexpected surfaces: %+v", surfaceValue)
	}

	childFixture := readRubyFixture(t, "ruby", "slice-221-delegated-child-operations", "yard-example-child-operations.json")
	childAnalysis := ParseRuby(childFixture["source"].(string), DialectRuby)
	encodedChildren, err := json.Marshal(RubyDelegatedChildOperations(*childAnalysis.Analysis, childFixture["parent_operation_id"].(string)))
	if err != nil {
		t.Fatalf("marshal child operations: %v", err)
	}
	var childValue any
	if err := json.Unmarshal(encodedChildren, &childValue); err != nil {
		t.Fatalf("unmarshal child operations: %v", err)
	}
	if !deepEqualJSON(childValue, childFixture["expected"]) {
		t.Fatalf("unexpected child operations: %+v", childValue)
	}

	groupedFixture := readRubyFixture(t, "ruby", "slice-229-projected-child-review-groups", "yard-example-review-groups.json")
	groupedSource, err := json.Marshal(groupedFixture["cases"])
	if err != nil {
		t.Fatalf("marshal projected cases: %v", err)
	}
	var groupedCases []astmerge.ProjectedChildReviewCase
	if err := json.Unmarshal(groupedSource, &groupedCases); err != nil {
		t.Fatalf("unmarshal projected cases: %v", err)
	}
	groupedValue, err := json.Marshal(astmerge.GroupProjectedChildReviewCases(groupedCases))
	if err != nil {
		t.Fatalf("marshal projected groups: %v", err)
	}
	var decodedGroups any
	if err := json.Unmarshal(groupedValue, &decodedGroups); err != nil {
		t.Fatalf("unmarshal projected groups: %v", err)
	}
	if !deepEqualJSON(decodedGroups, groupedFixture["expected_groups"]) {
		t.Fatalf("unexpected projected groups: %+v", decodedGroups)
	}

	progressFixture := readRubyFixture(t, "ruby", "slice-232-projected-child-review-group-progress", "yard-example-review-progress.json")
	progressSource, err := json.Marshal(progressFixture["groups"])
	if err != nil {
		t.Fatalf("marshal projected groups: %v", err)
	}
	var progressGroups []astmerge.ProjectedChildReviewGroup
	if err := json.Unmarshal(progressSource, &progressGroups); err != nil {
		t.Fatalf("unmarshal projected groups: %v", err)
	}
	resolvedSource, err := json.Marshal(progressFixture["resolved_case_ids"])
	if err != nil {
		t.Fatalf("marshal resolved case ids: %v", err)
	}
	var resolvedCaseIDs []string
	if err := json.Unmarshal(resolvedSource, &resolvedCaseIDs); err != nil {
		t.Fatalf("unmarshal resolved case ids: %v", err)
	}
	progressValue, err := json.Marshal(astmerge.SummarizeProjectedChildReviewGroupProgress(progressGroups, resolvedCaseIDs))
	if err != nil {
		t.Fatalf("marshal projected progress: %v", err)
	}
	var decodedProgress any
	if err := json.Unmarshal(progressValue, &decodedProgress); err != nil {
		t.Fatalf("unmarshal projected progress: %v", err)
	}
	if !deepEqualJSON(decodedProgress, progressFixture["expected_progress"]) {
		t.Fatalf("unexpected projected progress: %+v", decodedProgress)
	}
}

func deepEqualJSON(left any, right any) bool {
	leftJSON, _ := json.Marshal(left)
	rightJSON, _ := json.Marshal(right)
	return string(leftJSON) == string(rightJSON)
}

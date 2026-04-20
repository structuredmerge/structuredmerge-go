package gomerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func readGoFixture(t *testing.T, parts ...string) map[string]any {
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

func TestGoFixtures(t *testing.T) {
	profileFixture := readGoFixture(t, "diagnostics", "slice-109-go-family-feature-profile", "go-feature-profile.json")
	if GoFeatureProfileInfo().Family != profileFixture["feature_profile"].(map[string]any)["family"].(string) {
		t.Fatal("unexpected profile family")
	}

	analysisFixture := readGoFixture(t, "go", "slice-110-analysis", "module-owners.json")
	analysis := ParseGo(analysisFixture["source"].(string), DialectGo)
	if !analysis.OK || analysis.Analysis == nil {
		t.Fatalf("unexpected analysis: %+v", analysis)
	}

	matchingFixture := readGoFixture(t, "go", "slice-111-matching", "path-equality.json")
	template := ParseGo(matchingFixture["template"].(string), DialectGo)
	destination := ParseGo(matchingFixture["destination"].(string), DialectGo)
	match := MatchGoOwners(*template.Analysis, *destination.Analysis)
	if len(match.Matched) != len(matchingFixture["expected"].(map[string]any)["matched"].([]any)) {
		t.Fatalf("unexpected matches: %+v", match)
	}

	mergeFixture := readGoFixture(t, "go", "slice-112-merge", "module-merge.json")
	merge := MergeGo(mergeFixture["template"].(string), mergeFixture["destination"].(string), DialectGo)
	if !merge.OK || merge.Output == nil || *merge.Output != mergeFixture["expected"].(map[string]any)["output"].(string) {
		t.Fatalf("unexpected merge: %+v", merge)
	}
}

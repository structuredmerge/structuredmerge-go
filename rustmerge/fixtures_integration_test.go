package rustmerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func readRustFixture(t *testing.T, parts ...string) map[string]any {
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

func TestRustFixtures(t *testing.T) {
	profileFixture := readRustFixture(t, "diagnostics", "slice-105-rust-family-feature-profile", "rust-feature-profile.json")
	if RustFeatureProfileInfo().Family != profileFixture["feature_profile"].(map[string]any)["family"].(string) {
		t.Fatal("unexpected profile family")
	}

	analysisFixture := readRustFixture(t, "rust", "slice-106-analysis", "module-owners.json")
	analysis := ParseRust(analysisFixture["source"].(string), DialectRust)
	if !analysis.OK || analysis.Analysis == nil {
		t.Fatalf("unexpected analysis: %+v", analysis)
	}

	matchingFixture := readRustFixture(t, "rust", "slice-107-matching", "path-equality.json")
	template := ParseRust(matchingFixture["template"].(string), DialectRust)
	destination := ParseRust(matchingFixture["destination"].(string), DialectRust)
	match := MatchRustOwners(*template.Analysis, *destination.Analysis)
	if len(match.Matched) != len(matchingFixture["expected"].(map[string]any)["matched"].([]any)) {
		t.Fatalf("unexpected matches: %+v", match)
	}

	mergeFixture := readRustFixture(t, "rust", "slice-108-merge", "module-merge.json")
	merge := MergeRust(mergeFixture["template"].(string), mergeFixture["destination"].(string), DialectRust)
	if !merge.OK || merge.Output == nil || *merge.Output != mergeFixture["expected"].(map[string]any)["output"].(string) {
		t.Fatalf("unexpected merge: %+v", merge)
	}
}

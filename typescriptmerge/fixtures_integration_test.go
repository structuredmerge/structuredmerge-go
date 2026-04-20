package typescriptmerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func readTypeScriptFixture(t *testing.T, parts ...string) map[string]any {
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

func TestTypeScriptFixtures(t *testing.T) {
	profileFixture := readTypeScriptFixture(t, "diagnostics", "slice-101-typescript-family-feature-profile", "typescript-feature-profile.json")
	if TypeScriptFeatureProfileInfo().Family != profileFixture["feature_profile"].(map[string]any)["family"].(string) {
		t.Fatal("unexpected profile family")
	}

	analysisFixture := readTypeScriptFixture(t, "typescript", "slice-102-analysis", "module-owners.json")
	analysis := ParseTypeScript(analysisFixture["source"].(string), DialectTypeScript)
	if !analysis.OK || analysis.Analysis == nil {
		t.Fatalf("unexpected analysis: %+v", analysis)
	}

	matchingFixture := readTypeScriptFixture(t, "typescript", "slice-103-matching", "path-equality.json")
	template := ParseTypeScript(matchingFixture["template"].(string), DialectTypeScript)
	destination := ParseTypeScript(matchingFixture["destination"].(string), DialectTypeScript)
	match := MatchTypeScriptOwners(*template.Analysis, *destination.Analysis)
	if len(match.Matched) != len(matchingFixture["expected"].(map[string]any)["matched"].([]any)) {
		t.Fatalf("unexpected matches: %+v", match)
	}

	mergeFixture := readTypeScriptFixture(t, "typescript", "slice-104-merge", "module-merge.json")
	merge := MergeTypeScript(mergeFixture["template"].(string), mergeFixture["destination"].(string), DialectTypeScript)
	if !merge.OK || merge.Output == nil || *merge.Output != mergeFixture["expected"].(map[string]any)["output"].(string) {
		t.Fatalf("unexpected merge: %+v", merge)
	}
}

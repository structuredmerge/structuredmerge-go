package rustmerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
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

func jsonReadyRust(value any) any {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		panic(err)
	}
	return decoded
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

	backendProfileFixture := readRustFixture(t, "diagnostics", "slice-122-source-family-backend-feature-profiles", "rust-backend-feature-profiles.json")
	treeSitterBackendProfile := RustBackendFeatureProfileInfo(BackendTreeSitter)
	if !reflect.DeepEqual(jsonReadyRust(map[string]any{
		"backend":            treeSitterBackendProfile.Backend,
		"supports_dialects":  treeSitterBackendProfile.SupportsDialects,
		"supported_policies": treeSitterBackendProfile.SupportedPolicies,
		"backend_ref": map[string]any{
			"id":     treeSitterBackendProfile.BackendRef.ID,
			"family": treeSitterBackendProfile.BackendRef.Family,
		},
	}), backendProfileFixture["tree_sitter"]) {
		t.Fatalf("unexpected backend profile: %+v", treeSitterBackendProfile)
	}

	planContextFixture := readRustFixture(t, "diagnostics", "slice-123-source-family-plan-contexts", "rust-plan-contexts.json")
	if !reflect.DeepEqual(jsonReadyRust(RustPlanContext(BackendTreeSitter)), planContextFixture["tree_sitter"]) {
		t.Fatalf("unexpected plan context: %+v", RustPlanContext(BackendTreeSitter))
	}
}

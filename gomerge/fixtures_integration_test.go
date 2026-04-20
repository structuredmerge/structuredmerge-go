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

	backendProfileFixture := readGoFixture(t, "diagnostics", "slice-122-source-family-backend-feature-profiles", "go-backend-feature-profiles.json")
	treeProfile := GoBackendFeatureProfile(BackendTreeSitter)
	if treeProfile.Backend != backendProfileFixture["tree_sitter"].(map[string]any)["backend"].(string) ||
		treeProfile.SupportsDialects != backendProfileFixture["tree_sitter"].(map[string]any)["supports_dialects"].(bool) {
		t.Fatalf("unexpected tree-sitter backend profile: %+v", treeProfile)
	}
	nativeProfile := GoBackendFeatureProfile(BackendNative)
	if nativeProfile.Backend != backendProfileFixture["native"].(map[string]any)["backend"].(string) ||
		nativeProfile.SupportsDialects != backendProfileFixture["native"].(map[string]any)["supports_dialects"].(bool) {
		t.Fatalf("unexpected native backend profile: %+v", nativeProfile)
	}

	planContextFixture := readGoFixture(t, "diagnostics", "slice-123-source-family-plan-contexts", "go-plan-contexts.json")
	treeContext := GoPlanContext(BackendTreeSitter)
	if treeContext.FamilyProfile.Family != planContextFixture["tree_sitter"].(map[string]any)["family_profile"].(map[string]any)["family"].(string) ||
		treeContext.FeatureProfile == nil ||
		treeContext.FeatureProfile.Backend != planContextFixture["tree_sitter"].(map[string]any)["feature_profile"].(map[string]any)["backend"].(string) {
		t.Fatalf("unexpected tree-sitter plan context: %+v", treeContext)
	}
	nativeContext := GoPlanContext(BackendNative)
	if nativeContext.FamilyProfile.Family != planContextFixture["native"].(map[string]any)["family_profile"].(map[string]any)["family"].(string) ||
		nativeContext.FeatureProfile == nil ||
		nativeContext.FeatureProfile.Backend != planContextFixture["native"].(map[string]any)["feature_profile"].(map[string]any)["backend"].(string) {
		t.Fatalf("unexpected native plan context: %+v", nativeContext)
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

func TestGoBackends(t *testing.T) {
	fixture := readGoFixture(t, "diagnostics", "slice-113-go-family-backends", "go-backends.json")
	backends := GoBackends()
	if len(backends) != len(fixture["backends"].([]any)) {
		t.Fatalf("unexpected backends: %+v", backends)
	}

	parityFixture := readGoFixture(t, "go", "slice-114-native", "module-parity.json")
	treeResult := ParseGoWithBackend(parityFixture["source"].(string), DialectGo, BackendTreeSitter)
	nativeResult := ParseGoWithBackend(parityFixture["source"].(string), DialectGo, BackendNative)
	if !treeResult.OK || treeResult.Analysis == nil {
		t.Fatalf("unexpected tree-sitter result: %+v", treeResult)
	}
	if !nativeResult.OK || nativeResult.Analysis == nil {
		t.Fatalf("unexpected native result: %+v", nativeResult)
	}
	if len(treeResult.Analysis.Owners) != len(parityFixture["expected"].(map[string]any)["owners"].([]any)) {
		t.Fatalf("unexpected tree owners: %+v", treeResult.Analysis.Owners)
	}
	if len(nativeResult.Analysis.Owners) != len(parityFixture["expected"].(map[string]any)["owners"].([]any)) {
		t.Fatalf("unexpected native owners: %+v", nativeResult.Analysis.Owners)
	}

	nativeMerge := MergeGoWithBackend(
		parityFixture["template"].(string),
		parityFixture["destination"].(string),
		DialectGo,
		BackendNative,
	)
	if !nativeMerge.OK || nativeMerge.Output == nil || *nativeMerge.Output != parityFixture["expected"].(map[string]any)["output"].(string) {
		t.Fatalf("unexpected native merge: %+v", nativeMerge)
	}
}

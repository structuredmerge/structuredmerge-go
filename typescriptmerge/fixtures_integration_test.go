package typescriptmerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/treehaver"
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

func jsonReadyTypeScript(t *testing.T, value any) any {
	t.Helper()
	source, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal value: %v", err)
	}
	var normalized any
	if err := json.Unmarshal(source, &normalized); err != nil {
		t.Fatalf("decode value: %v", err)
	}
	return normalized
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

	backendsFixture := readTypeScriptFixture(t, "diagnostics", "slice-115-typescript-family-backends", "typescript-backends.json")
	backends := TypeScriptBackends()
	if len(backends) != len(backendsFixture["backends"].([]any)) {
		t.Fatalf("unexpected backends: %+v", backends)
	}
	if len(backends) != 1 || backends[0] != BackendTreeSitter {
		t.Fatalf("unexpected backends: %+v", backends)
	}

	backendFixture := readTypeScriptFixture(t, "diagnostics", "slice-122-source-family-backend-feature-profiles", "typescript-backend-feature-profiles.json")
	backendProfile := TypeScriptBackendFeatureProfileInfo(BackendTreeSitter)
	if actual := jsonReadyTypeScript(t, map[string]any{
		"backend":            backendProfile.Backend,
		"supports_dialects":  backendProfile.SupportsDialects,
		"supported_policies": backendProfile.SupportedPolicies,
		"backend_ref": map[string]any{
			"id":     backendProfile.BackendRef.ID,
			"family": backendProfile.BackendRef.Family,
		},
	}); !reflect.DeepEqual(actual, backendFixture["tree_sitter"]) {
		t.Fatalf("unexpected backend fixture projection: %+v", actual)
	}
	if backend := treehaver.BackendReferenceByID(string(BackendTreeSitter)); backend == nil || backend.ID != string(BackendTreeSitter) || backend.Family != "tree-sitter" {
		t.Fatalf("unexpected registered backend: %+v", backend)
	}

	planFixture := readTypeScriptFixture(t, "diagnostics", "slice-123-source-family-plan-contexts", "typescript-plan-contexts.json")
	if !reflect.DeepEqual(jsonReadyTypeScript(t, TypeScriptPlanContext(BackendTreeSitter)), planFixture["tree_sitter"]) {
		t.Fatalf("unexpected plan context: %+v", TypeScriptPlanContext(BackendTreeSitter))
	}

	sourceManifestFixture := readTypeScriptFixture(t, "conformance", "slice-124-source-family-manifest", "source-family-manifest.json")
	sourceManifestSource, err := json.Marshal(sourceManifestFixture)
	if err != nil {
		t.Fatalf("marshal source manifest: %v", err)
	}
	var sourceManifest astmerge.ConformanceManifest
	if err := json.Unmarshal(sourceManifestSource, &sourceManifest); err != nil {
		t.Fatalf("decode source manifest: %v", err)
	}
	if path := astmerge.ConformanceFamilyFeatureProfilePath(sourceManifest, "typescript"); path == nil || filepath.Join(path...) != filepath.Join("diagnostics", "slice-101-typescript-family-feature-profile", "typescript-feature-profile.json") {
		t.Fatalf("unexpected source family profile path: %+v", path)
	}
	if path := astmerge.ConformanceFixturePath(sourceManifest, "typescript", "analysis"); path == nil || filepath.Join(path...) != filepath.Join("typescript", "slice-102-analysis", "module-owners.json") {
		t.Fatalf("unexpected source analysis path: %+v", path)
	}
	if path := astmerge.ConformanceFixturePath(sourceManifest, "typescript", "matching"); path == nil || filepath.Join(path...) != filepath.Join("typescript", "slice-103-matching", "path-equality.json") {
		t.Fatalf("unexpected source matching path: %+v", path)
	}
	if path := astmerge.ConformanceFixturePath(sourceManifest, "typescript", "merge"); path == nil || filepath.Join(path...) != filepath.Join("typescript", "slice-104-merge", "module-merge.json") {
		t.Fatalf("unexpected source merge path: %+v", path)
	}

	canonicalManifestFixture := readTypeScriptFixture(t, "conformance", "slice-24-manifest", "family-feature-profiles.json")
	canonicalManifestSource, err := json.Marshal(canonicalManifestFixture)
	if err != nil {
		t.Fatalf("marshal canonical manifest: %v", err)
	}
	var canonicalManifest astmerge.ConformanceManifest
	if err := json.Unmarshal(canonicalManifestSource, &canonicalManifest); err != nil {
		t.Fatalf("decode canonical manifest: %v", err)
	}
	if path := astmerge.ConformanceFamilyFeatureProfilePath(canonicalManifest, "typescript"); path == nil || filepath.Join(path...) != filepath.Join("diagnostics", "slice-101-typescript-family-feature-profile", "typescript-feature-profile.json") {
		t.Fatalf("unexpected canonical family profile path: %+v", path)
	}
	if path := astmerge.ConformanceFixturePath(canonicalManifest, "typescript", "analysis"); path == nil || filepath.Join(path...) != filepath.Join("typescript", "slice-102-analysis", "module-owners.json") {
		t.Fatalf("unexpected canonical analysis path: %+v", path)
	}
	if path := astmerge.ConformanceFixturePath(canonicalManifest, "typescript", "matching"); path == nil || filepath.Join(path...) != filepath.Join("typescript", "slice-103-matching", "path-equality.json") {
		t.Fatalf("unexpected canonical matching path: %+v", path)
	}
	if path := astmerge.ConformanceFixturePath(canonicalManifest, "typescript", "merge"); path == nil || filepath.Join(path...) != filepath.Join("typescript", "slice-104-merge", "module-merge.json") {
		t.Fatalf("unexpected canonical merge path: %+v", path)
	}
}

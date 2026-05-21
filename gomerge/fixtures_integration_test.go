package gomerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/treehaver"
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

func jsonReadyGo(t *testing.T, value any) any {
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

func assertStructuredImportFailure(t *testing.T, diagnostics []astmerge.Diagnostic) {
	t.Helper()
	if len(diagnostics) == 0 || diagnostics[0].Category != astmerge.CategoryUnsupportedFeature || !strings.Contains(diagnostics[0].Message, "structured import module fields") {
		t.Fatalf("expected structured import failure diagnostic: %+v", diagnostics)
	}
}

func TestGoFixtures(t *testing.T) {
	profileFixture := readGoFixture(t, "diagnostics", "slice-109-go-family-feature-profile", "go-feature-profile.json")
	if GoFeatureProfileInfo().Family != profileFixture["feature_profile"].(map[string]any)["family"].(string) {
		t.Fatal("unexpected profile family")
	}

	backendProfileFixture := readGoFixture(t, "diagnostics", "slice-122-source-family-backend-feature-profiles", "go-backend-feature-profiles.json")
	treeProfile := GoBackendFeatureProfileInfo(BackendTreeSitter)
	if treeProfile.Backend != backendProfileFixture["tree_sitter"].(map[string]any)["backend"].(string) ||
		treeProfile.SupportsDialects != backendProfileFixture["tree_sitter"].(map[string]any)["supports_dialects"].(bool) {
		t.Fatalf("unexpected tree-sitter backend profile: %+v", treeProfile)
	}
	if actual := jsonReadyGo(t, map[string]any{
		"backend":            treeProfile.Backend,
		"supports_dialects":  treeProfile.SupportsDialects,
		"supported_policies": treeProfile.SupportedPolicies,
		"backend_ref": map[string]any{
			"id":     treeProfile.BackendRef.ID,
			"family": treeProfile.BackendRef.Family,
		},
	}); actual == nil {
		t.Fatal("unexpected nil backend projection")
	} else if expected := backendProfileFixture["tree_sitter"]; !reflect.DeepEqual(actual, expected) {
		t.Fatalf("unexpected tree-sitter backend fixture projection: %+v", actual)
	}
	if backend := treehaver.BackendReferenceByID(string(BackendTreeSitter)); backend == nil || backend.ID != string(BackendTreeSitter) || backend.Family != "tree-sitter" {
		t.Fatalf("unexpected registered backend: %+v", backend)
	}

	planContextFixture := readGoFixture(t, "diagnostics", "slice-123-source-family-plan-contexts", "go-plan-contexts.json")
	treeContext := GoPlanContext(BackendTreeSitter)
	if treeContext.FamilyProfile.Family != planContextFixture["tree_sitter"].(map[string]any)["family_profile"].(map[string]any)["family"].(string) ||
		treeContext.FeatureProfile == nil ||
		treeContext.FeatureProfile.Backend != planContextFixture["tree_sitter"].(map[string]any)["feature_profile"].(map[string]any)["backend"].(string) {
		t.Fatalf("unexpected tree-sitter plan context: %+v", treeContext)
	}

	analysisFixture := readGoFixture(t, "go", "slice-110-analysis", "module-owners.json")
	analysis := ParseGo(analysisFixture["source"].(string), DialectGo)
	if !analysis.OK {
		assertStructuredImportFailure(t, analysis.Diagnostics)
	} else if analysis.Analysis == nil {
		t.Fatalf("unexpected nil analysis: %+v", analysis)
	}

	matchingFixture := readGoFixture(t, "go", "slice-111-matching", "path-equality.json")
	template := ParseGo(matchingFixture["template"].(string), DialectGo)
	destination := ParseGo(matchingFixture["destination"].(string), DialectGo)
	if !template.OK || !destination.OK {
		if template.OK {
			assertStructuredImportFailure(t, destination.Diagnostics)
		} else {
			assertStructuredImportFailure(t, template.Diagnostics)
		}
	} else {
		match := MatchGoOwners(*template.Analysis, *destination.Analysis)
		if len(match.Matched) != len(matchingFixture["expected"].(map[string]any)["matched"].([]any)) {
			t.Fatalf("unexpected matches: %+v", match)
		}
	}

	mergeFixture := readGoFixture(t, "go", "slice-112-merge", "module-merge.json")
	merge := MergeGo(mergeFixture["template"].(string), mergeFixture["destination"].(string), DialectGo)
	if !merge.OK {
		assertStructuredImportFailure(t, merge.Diagnostics)
	} else if merge.Output == nil || *merge.Output != mergeFixture["expected"].(map[string]any)["output"].(string) {
		t.Fatalf("unexpected merge: %+v", merge)
	}
}

func TestGoBackends(t *testing.T) {
	fixture := readGoFixture(t, "diagnostics", "slice-113-go-family-backends", "go-backends.json")
	backends := GoBackends()
	if len(backends) != len(fixture["backends"].([]any)) {
		t.Fatalf("unexpected backends: %+v", backends)
	}
	if len(backends) != 1 || backends[0] != BackendTreeSitter {
		t.Fatalf("unexpected backends: %+v", backends)
	}
}

func TestSlice124SourceFamilyManifest(t *testing.T) {
	manifestFixture := readGoFixture(t, "conformance", "slice-124-source-family-manifest", "source-family-manifest.json")
	manifestSource, err := json.Marshal(manifestFixture)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	var manifest astmerge.ConformanceManifest
	if err := json.Unmarshal(manifestSource, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}

	if path := astmerge.ConformanceFamilyFeatureProfilePath(manifest, "go"); path == nil || filepath.Join(path...) != filepath.Join("diagnostics", "slice-109-go-family-feature-profile", "go-feature-profile.json") {
		t.Fatalf("unexpected go family profile path: %+v", path)
	}
	if path := astmerge.ConformanceFixturePath(manifest, "go", "analysis"); path == nil || filepath.Join(path...) != filepath.Join("go", "slice-110-analysis", "module-owners.json") {
		t.Fatalf("unexpected go analysis path: %+v", path)
	}
	if path := astmerge.ConformanceFixturePath(manifest, "go", "matching"); path == nil || filepath.Join(path...) != filepath.Join("go", "slice-111-matching", "path-equality.json") {
		t.Fatalf("unexpected go matching path: %+v", path)
	}
	if path := astmerge.ConformanceFixturePath(manifest, "go", "merge"); path == nil || filepath.Join(path...) != filepath.Join("go", "slice-112-merge", "module-merge.json") {
		t.Fatalf("unexpected go merge path: %+v", path)
	}
}

func TestSlice131CanonicalManifest(t *testing.T) {
	manifestFixture := readGoFixture(t, "conformance", "slice-24-manifest", "family-feature-profiles.json")
	manifestSource, err := json.Marshal(manifestFixture)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	var manifest astmerge.ConformanceManifest
	if err := json.Unmarshal(manifestSource, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}

	if path := astmerge.ConformanceFamilyFeatureProfilePath(manifest, "go"); path == nil || filepath.Join(path...) != filepath.Join("diagnostics", "slice-109-go-family-feature-profile", "go-feature-profile.json") {
		t.Fatalf("unexpected canonical go family profile path: %+v", path)
	}
	if path := astmerge.ConformanceFixturePath(manifest, "go", "analysis"); path == nil || filepath.Join(path...) != filepath.Join("go", "slice-110-analysis", "module-owners.json") {
		t.Fatalf("unexpected canonical go analysis path: %+v", path)
	}
	if path := astmerge.ConformanceFixturePath(manifest, "go", "matching"); path == nil || filepath.Join(path...) != filepath.Join("go", "slice-111-matching", "path-equality.json") {
		t.Fatalf("unexpected canonical go matching path: %+v", path)
	}
	if path := astmerge.ConformanceFixturePath(manifest, "go", "merge"); path == nil || filepath.Join(path...) != filepath.Join("go", "slice-112-merge", "module-merge.json") {
		t.Fatalf("unexpected canonical go merge path: %+v", path)
	}
}

package rustmerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
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

func assertStructuredImportFailure(t *testing.T, diagnostics []astmerge.Diagnostic) {
	t.Helper()
	if len(diagnostics) == 0 || diagnostics[0].Category != astmerge.CategoryUnsupportedFeature || !strings.Contains(diagnostics[0].Message, "structured import module fields") {
		t.Fatalf("expected structured import failure diagnostic: %+v", diagnostics)
	}
}

func TestRustFixtures(t *testing.T) {
	profileFixture := readRustFixture(t, "diagnostics", "slice-105-rust-family-feature-profile", "rust-feature-profile.json")
	if RustFeatureProfileInfo().Family != profileFixture["feature_profile"].(map[string]any)["family"].(string) {
		t.Fatal("unexpected profile family")
	}

	analysisFixture := readRustFixture(t, "rust", "slice-106-analysis", "module-owners.json")
	analysis := ParseRust(analysisFixture["source"].(string), DialectRust)
	if !analysis.OK {
		assertStructuredImportFailure(t, analysis.Diagnostics)
	} else if analysis.Analysis == nil {
		t.Fatalf("unexpected nil analysis: %+v", analysis)
	}

	matchingFixture := readRustFixture(t, "rust", "slice-107-matching", "path-equality.json")
	template := ParseRust(matchingFixture["template"].(string), DialectRust)
	destination := ParseRust(matchingFixture["destination"].(string), DialectRust)
	if !template.OK || !destination.OK {
		if template.OK {
			assertStructuredImportFailure(t, destination.Diagnostics)
		} else {
			assertStructuredImportFailure(t, template.Diagnostics)
		}
	} else {
		match := MatchRustOwners(*template.Analysis, *destination.Analysis)
		if len(match.Matched) != len(matchingFixture["expected"].(map[string]any)["matched"].([]any)) {
			t.Fatalf("unexpected matches: %+v", match)
		}
	}

	mergeFixture := readRustFixture(t, "rust", "slice-108-merge", "module-merge.json")
	merge := MergeRust(mergeFixture["template"].(string), mergeFixture["destination"].(string), DialectRust)
	if !merge.OK {
		assertStructuredImportFailure(t, merge.Diagnostics)
	} else if merge.Output == nil || *merge.Output != mergeFixture["expected"].(map[string]any)["output"].(string) {
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

	sourceManifestFixture := readRustFixture(t, "conformance", "slice-124-source-family-manifest", "source-family-manifest.json")
	sourceManifestSource, err := json.Marshal(sourceManifestFixture)
	if err != nil {
		t.Fatalf("marshal source manifest: %v", err)
	}
	var sourceManifest astmerge.ConformanceManifest
	if err := json.Unmarshal(sourceManifestSource, &sourceManifest); err != nil {
		t.Fatalf("decode source manifest: %v", err)
	}
	if path := astmerge.ConformanceFamilyFeatureProfilePath(sourceManifest, "rust"); path == nil || filepath.Join(path...) != filepath.Join("diagnostics", "slice-105-rust-family-feature-profile", "rust-feature-profile.json") {
		t.Fatalf("unexpected source family profile path: %+v", path)
	}
	if path := astmerge.ConformanceFixturePath(sourceManifest, "rust", "analysis"); path == nil || filepath.Join(path...) != filepath.Join("rust", "slice-106-analysis", "module-owners.json") {
		t.Fatalf("unexpected source analysis path: %+v", path)
	}
	if path := astmerge.ConformanceFixturePath(sourceManifest, "rust", "matching"); path == nil || filepath.Join(path...) != filepath.Join("rust", "slice-107-matching", "path-equality.json") {
		t.Fatalf("unexpected source matching path: %+v", path)
	}
	if path := astmerge.ConformanceFixturePath(sourceManifest, "rust", "merge"); path == nil || filepath.Join(path...) != filepath.Join("rust", "slice-108-merge", "module-merge.json") {
		t.Fatalf("unexpected source merge path: %+v", path)
	}

	canonicalManifestFixture := readRustFixture(t, "conformance", "slice-24-manifest", "family-feature-profiles.json")
	canonicalManifestSource, err := json.Marshal(canonicalManifestFixture)
	if err != nil {
		t.Fatalf("marshal canonical manifest: %v", err)
	}
	var canonicalManifest astmerge.ConformanceManifest
	if err := json.Unmarshal(canonicalManifestSource, &canonicalManifest); err != nil {
		t.Fatalf("decode canonical manifest: %v", err)
	}
	if path := astmerge.ConformanceFamilyFeatureProfilePath(canonicalManifest, "rust"); path == nil || filepath.Join(path...) != filepath.Join("diagnostics", "slice-105-rust-family-feature-profile", "rust-feature-profile.json") {
		t.Fatalf("unexpected canonical family profile path: %+v", path)
	}
	if path := astmerge.ConformanceFixturePath(canonicalManifest, "rust", "analysis"); path == nil || filepath.Join(path...) != filepath.Join("rust", "slice-106-analysis", "module-owners.json") {
		t.Fatalf("unexpected canonical analysis path: %+v", path)
	}
	if path := astmerge.ConformanceFixturePath(canonicalManifest, "rust", "matching"); path == nil || filepath.Join(path...) != filepath.Join("rust", "slice-107-matching", "path-equality.json") {
		t.Fatalf("unexpected canonical matching path: %+v", path)
	}
	if path := astmerge.ConformanceFixturePath(canonicalManifest, "rust", "merge"); path == nil || filepath.Join(path...) != filepath.Join("rust", "slice-108-merge", "module-merge.json") {
		t.Fatalf("unexpected canonical merge path: %+v", path)
	}
}

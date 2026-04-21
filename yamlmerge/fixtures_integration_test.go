package yamlmerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/treehaver"
)

func readYAMLFixture(t *testing.T, parts ...string) map[string]any {
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

func TestSharedFixtureYAMLFeatureProfile(t *testing.T) {
	fixture := readYAMLFixture(t, "diagnostics", "slice-95-yaml-family-feature-profile", "yaml-feature-profile.json")
	profile := YAMLFeatureProfileInfo()

	expected := fixture["feature_profile"].(map[string]any)
	if profile.Family != expected["family"].(string) {
		t.Fatalf("unexpected family: %+v", profile)
	}
	if len(profile.SupportedDialects) != 1 || string(profile.SupportedDialects[0]) != "yaml" {
		t.Fatalf("unexpected dialects: %+v", profile.SupportedDialects)
	}
	if len(profile.SupportedPolicies) != 1 || profile.SupportedPolicies[0].Name != "destination_wins_array" {
		t.Fatalf("unexpected policies: %+v", profile.SupportedPolicies)
	}
}

func TestSharedFixtureYAMLBackendFeatureProfiles(t *testing.T) {
	fixture := readYAMLFixture(t, "diagnostics", "slice-171-yaml-family-backend-feature-profiles", "go-yaml-backend-feature-profiles.json")
	backends := AvailableYAMLBackends()
	if len(backends) != 1 || backends[0] != BackendKreuzberg {
		t.Fatalf("unexpected backends: %+v", backends)
	}

	treeSitter := YAMLBackendFeatureProfileInfo(BackendKreuzberg)
	if treeSitter.Backend != fixture["tree_sitter"].(map[string]any)["backend"].(string) {
		t.Fatalf("unexpected tree-sitter backend profile: %+v", treeSitter)
	}
	if backend := treehaver.BackendReferenceByID(string(BackendKreuzberg)); backend == nil || backend.ID != string(BackendKreuzberg) || backend.Family != "tree-sitter" {
		t.Fatalf("unexpected registered backend: %+v", backend)
	}
}

func TestSharedFixtureYAMLPolyglotBackendFeatureProfiles(t *testing.T) {
	fixture := readYAMLFixture(t, "diagnostics", "slice-183-yaml-family-polyglot-backend-feature-profiles", "go-yaml-polyglot-backend-feature-profiles.json")
	treeSitter := YAMLBackendFeatureProfileInfo(BackendKreuzberg)
	if treeSitter.Backend != fixture["tree_sitter"].(map[string]any)["backend"].(string) {
		t.Fatalf("unexpected tree-sitter backend profile: %+v", treeSitter)
	}
}

func TestSharedFixtureYAMLPlanContext(t *testing.T) {
	fixture := readYAMLFixture(t, "diagnostics", "slice-172-yaml-family-backend-plan-contexts", "go-yaml-plan-contexts.json")

	treeSitter := YAMLPlanContextWithBackend(BackendKreuzberg)
	if treeSitter.FamilyProfile.Family != fixture["tree_sitter"].(map[string]any)["family_profile"].(map[string]any)["family"].(string) {
		t.Fatalf("unexpected tree-sitter family profile: %+v", treeSitter)
	}
	if treeSitter.FeatureProfile == nil || treeSitter.FeatureProfile.Backend != fixture["tree_sitter"].(map[string]any)["feature_profile"].(map[string]any)["backend"].(string) {
		t.Fatalf("unexpected tree-sitter feature profile: %+v", treeSitter.FeatureProfile)
	}
}

func TestSharedFixtureYAMLPolyglotPlanContext(t *testing.T) {
	fixture := readYAMLFixture(t, "diagnostics", "slice-184-yaml-family-polyglot-backend-plan-contexts", "go-yaml-polyglot-plan-contexts.json")

	treeSitter := YAMLPlanContextWithBackend(BackendKreuzberg)
	if treeSitter.FamilyProfile.Family != fixture["tree_sitter"].(map[string]any)["family_profile"].(map[string]any)["family"].(string) {
		t.Fatalf("unexpected tree-sitter family profile: %+v", treeSitter)
	}
	if treeSitter.FeatureProfile == nil || treeSitter.FeatureProfile.Backend != fixture["tree_sitter"].(map[string]any)["feature_profile"].(map[string]any)["backend"].(string) {
		t.Fatalf("unexpected tree-sitter feature profile: %+v", treeSitter.FeatureProfile)
	}
}

func TestSharedFixtureYAMLManifest(t *testing.T) {
	fixture := readYAMLFixture(t, "conformance", "slice-143-yaml-family-manifest", "yaml-family-manifest.json")
	source, err := json.Marshal(fixture)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}

	var manifest astmerge.ConformanceManifest
	if err := json.Unmarshal(source, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}

	if path := astmerge.ConformanceFamilyFeatureProfilePath(manifest, "yaml"); path == nil || filepath.Join(path...) != filepath.Join("diagnostics", "slice-95-yaml-family-feature-profile", "yaml-feature-profile.json") {
		t.Fatalf("unexpected family feature profile path: %v", path)
	}
	if path := astmerge.ConformanceFixturePath(manifest, "yaml", "analysis"); path == nil || filepath.Join(path...) != filepath.Join("yaml", "slice-97-structure", "mapping-and-sequence.json") {
		t.Fatalf("unexpected analysis fixture path: %v", path)
	}
	if path := astmerge.ConformanceFixturePath(manifest, "yaml", "merge"); path == nil || filepath.Join(path...) != filepath.Join("yaml", "slice-99-merge", "mapping-merge.json") {
		t.Fatalf("unexpected merge fixture path: %v", path)
	}
}

func TestCanonicalManifestIncludesYAMLPaths(t *testing.T) {
	fixture := readYAMLFixture(t, "conformance", "slice-24-manifest", "family-feature-profiles.json")
	source, err := json.Marshal(fixture)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}

	var manifest astmerge.ConformanceManifest
	if err := json.Unmarshal(source, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}

	if path := astmerge.ConformanceFamilyFeatureProfilePath(manifest, "yaml"); path == nil || filepath.Join(path...) != filepath.Join("diagnostics", "slice-95-yaml-family-feature-profile", "yaml-feature-profile.json") {
		t.Fatalf("unexpected canonical family feature profile path: %v", path)
	}
	if path := astmerge.ConformanceFixturePath(manifest, "yaml", "matching"); path == nil || filepath.Join(path...) != filepath.Join("yaml", "slice-98-matching", "path-equality.json") {
		t.Fatalf("unexpected canonical matching fixture path: %v", path)
	}
}

func TestSharedFixtureYAMLParse(t *testing.T) {
	valid := readYAMLFixture(t, "yaml", "slice-96-parse", "valid-document.json")
	for _, backend := range []YAMLBackend{BackendKreuzberg} {
		validResult := ParseYAMLWithBackend(valid["source"].(string), DialectYAML, backend)
		if !validResult.OK || validResult.Analysis == nil || string(validResult.Analysis.RootKind) != "mapping" {
			t.Fatalf("unexpected valid parse result for %s: %+v", backend, validResult)
		}
		if len(validResult.Diagnostics) != 0 {
			t.Fatalf("unexpected diagnostics for %s: %+v", backend, validResult.Diagnostics)
		}
	}

	invalid := readYAMLFixture(t, "yaml", "slice-96-parse", "invalid-document.json")
	for _, backend := range []YAMLBackend{BackendKreuzberg} {
		invalidResult := ParseYAMLWithBackend(invalid["source"].(string), DialectYAML, backend)
		if invalidResult.OK {
			t.Fatalf("expected invalid parse failure for %s: %+v", backend, invalidResult)
		}
		if len(invalidResult.Diagnostics) != 1 || string(invalidResult.Diagnostics[0].Category) != "parse_error" {
			t.Fatalf("unexpected invalid diagnostics for %s: %+v", backend, invalidResult.Diagnostics)
		}
	}
}

func TestSharedFixtureYAMLStructure(t *testing.T) {
	fixture := readYAMLFixture(t, "yaml", "slice-97-structure", "mapping-and-sequence.json")
	expectedOwners := fixture["expected"].(map[string]any)["owners"].([]any)

	for _, backend := range []YAMLBackend{BackendKreuzberg} {
		result := ParseYAMLWithBackend(fixture["source"].(string), DialectYAML, backend)
		if !result.OK || result.Analysis == nil {
			t.Fatalf("expected parse success for %s: %+v", backend, result)
		}

		if len(result.Analysis.Owners) != len(expectedOwners) {
			t.Fatalf("unexpected owners length for %s: %+v", backend, result.Analysis.Owners)
		}
		for index, item := range expectedOwners {
			expected := item.(map[string]any)
			owner := result.Analysis.Owners[index]
			if owner.Path != expected["path"].(string) || string(owner.OwnerKind) != expected["owner_kind"].(string) {
				t.Fatalf("unexpected owner at %d for %s: %+v", index, backend, owner)
			}
			if expectedMatchKey, ok := expected["match_key"]; ok && owner.MatchKey != expectedMatchKey.(string) {
				t.Fatalf("unexpected match_key at %d for %s: %+v", index, backend, owner)
			}
		}
	}
}

func TestSharedFixtureYAMLMatching(t *testing.T) {
	fixture := readYAMLFixture(t, "yaml", "slice-98-matching", "path-equality.json")

	for _, backend := range []YAMLBackend{BackendKreuzberg} {
		template := ParseYAMLWithBackend(fixture["template"].(string), DialectYAML, backend)
		destination := ParseYAMLWithBackend(fixture["destination"].(string), DialectYAML, backend)
		result := MatchYAMLOwners(*template.Analysis, *destination.Analysis)

		expected := fixture["expected"].(map[string]any)
		if len(result.Matched) != len(expected["matched"].([]any)) {
			t.Fatalf("unexpected matched owners for %s: %+v", backend, result.Matched)
		}
		if len(result.UnmatchedTemplate) != len(expected["unmatched_template"].([]any)) {
			t.Fatalf("unexpected unmatched template for %s: %+v", backend, result.UnmatchedTemplate)
		}
		if len(result.UnmatchedDestination) != len(expected["unmatched_destination"].([]any)) {
			t.Fatalf("unexpected unmatched destination for %s: %+v", backend, result.UnmatchedDestination)
		}
	}
}

func TestSharedFixtureYAMLMerge(t *testing.T) {
	mergeFixture := readYAMLFixture(t, "yaml", "slice-99-merge", "mapping-merge.json")
	for _, backend := range []YAMLBackend{BackendKreuzberg} {
		mergeResult := MergeYAMLWithBackend(mergeFixture["template"].(string), mergeFixture["destination"].(string), DialectYAML, backend)
		if !mergeResult.OK || mergeResult.Output == nil {
			t.Fatalf("expected merge success for %s: %+v", backend, mergeResult)
		}
		if *mergeResult.Output != mergeFixture["expected"].(map[string]any)["output"].(string) {
			t.Fatalf("unexpected merge output for %s:\n%s", backend, *mergeResult.Output)
		}
	}

	invalidTemplate := readYAMLFixture(t, "yaml", "slice-99-merge", "invalid-template.json")
	for _, backend := range []YAMLBackend{BackendKreuzberg} {
		invalidTemplateResult := MergeYAMLWithBackend(invalidTemplate["template"].(string), invalidTemplate["destination"].(string), DialectYAML, backend)
		if invalidTemplateResult.OK || len(invalidTemplateResult.Diagnostics) != 1 || string(invalidTemplateResult.Diagnostics[0].Category) != "parse_error" {
			t.Fatalf("unexpected invalid template result for %s: %+v", backend, invalidTemplateResult)
		}
	}

	invalidDestination := readYAMLFixture(t, "yaml", "slice-99-merge", "invalid-destination.json")
	for _, backend := range []YAMLBackend{BackendKreuzberg} {
		invalidDestinationResult := MergeYAMLWithBackend(invalidDestination["template"].(string), invalidDestination["destination"].(string), DialectYAML, backend)
		if invalidDestinationResult.OK || len(invalidDestinationResult.Diagnostics) != 1 || string(invalidDestinationResult.Diagnostics[0].Category) != "destination_parse_error" {
			t.Fatalf("unexpected invalid destination result for %s: %+v", backend, invalidDestinationResult)
		}
	}
}

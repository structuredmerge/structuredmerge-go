package yamlmerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
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

func TestSharedFixtureYAMLPlanContext(t *testing.T) {
	fixture := readYAMLFixture(t, "diagnostics", "slice-142-yaml-family-plan-contexts", "go-yaml-plan-contexts.json")
	context := YAMLPlanContext()

	if context.FamilyProfile.Family != fixture["native"].(map[string]any)["family_profile"].(map[string]any)["family"].(string) {
		t.Fatalf("unexpected family profile: %+v", context)
	}
	if context.FeatureProfile == nil || context.FeatureProfile.Backend != fixture["native"].(map[string]any)["feature_profile"].(map[string]any)["backend"].(string) {
		t.Fatalf("unexpected feature profile: %+v", context.FeatureProfile)
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
	validResult := ParseYAML(valid["source"].(string), DialectYAML)
	if !validResult.OK || validResult.Analysis == nil || string(validResult.Analysis.RootKind) != "mapping" {
		t.Fatalf("unexpected valid parse result: %+v", validResult)
	}
	if len(validResult.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", validResult.Diagnostics)
	}

	invalid := readYAMLFixture(t, "yaml", "slice-96-parse", "invalid-document.json")
	invalidResult := ParseYAML(invalid["source"].(string), DialectYAML)
	if invalidResult.OK {
		t.Fatalf("expected invalid parse failure: %+v", invalidResult)
	}
	if len(invalidResult.Diagnostics) != 1 || string(invalidResult.Diagnostics[0].Category) != "parse_error" {
		t.Fatalf("unexpected invalid diagnostics: %+v", invalidResult.Diagnostics)
	}
}

func TestSharedFixtureYAMLStructure(t *testing.T) {
	fixture := readYAMLFixture(t, "yaml", "slice-97-structure", "mapping-and-sequence.json")
	result := ParseYAML(fixture["source"].(string), DialectYAML)
	if !result.OK || result.Analysis == nil {
		t.Fatalf("expected parse success: %+v", result)
	}

	expectedOwners := fixture["expected"].(map[string]any)["owners"].([]any)
	if len(result.Analysis.Owners) != len(expectedOwners) {
		t.Fatalf("unexpected owners length: %+v", result.Analysis.Owners)
	}
	for index, item := range expectedOwners {
		expected := item.(map[string]any)
		owner := result.Analysis.Owners[index]
		if owner.Path != expected["path"].(string) || string(owner.OwnerKind) != expected["owner_kind"].(string) {
			t.Fatalf("unexpected owner at %d: %+v", index, owner)
		}
		if expectedMatchKey, ok := expected["match_key"]; ok && owner.MatchKey != expectedMatchKey.(string) {
			t.Fatalf("unexpected match_key at %d: %+v", index, owner)
		}
	}
}

func TestSharedFixtureYAMLMatching(t *testing.T) {
	fixture := readYAMLFixture(t, "yaml", "slice-98-matching", "path-equality.json")
	template := ParseYAML(fixture["template"].(string), DialectYAML)
	destination := ParseYAML(fixture["destination"].(string), DialectYAML)
	result := MatchYAMLOwners(*template.Analysis, *destination.Analysis)

	expected := fixture["expected"].(map[string]any)
	if len(result.Matched) != len(expected["matched"].([]any)) {
		t.Fatalf("unexpected matched owners: %+v", result.Matched)
	}
	if len(result.UnmatchedTemplate) != len(expected["unmatched_template"].([]any)) {
		t.Fatalf("unexpected unmatched template: %+v", result.UnmatchedTemplate)
	}
	if len(result.UnmatchedDestination) != len(expected["unmatched_destination"].([]any)) {
		t.Fatalf("unexpected unmatched destination: %+v", result.UnmatchedDestination)
	}
}

func TestSharedFixtureYAMLMerge(t *testing.T) {
	mergeFixture := readYAMLFixture(t, "yaml", "slice-99-merge", "mapping-merge.json")
	mergeResult := MergeYAML(mergeFixture["template"].(string), mergeFixture["destination"].(string), DialectYAML)
	if !mergeResult.OK || mergeResult.Output == nil {
		t.Fatalf("expected merge success: %+v", mergeResult)
	}
	if *mergeResult.Output != mergeFixture["expected"].(map[string]any)["output"].(string) {
		t.Fatalf("unexpected merge output:\n%s", *mergeResult.Output)
	}

	invalidTemplate := readYAMLFixture(t, "yaml", "slice-99-merge", "invalid-template.json")
	invalidTemplateResult := MergeYAML(invalidTemplate["template"].(string), invalidTemplate["destination"].(string), DialectYAML)
	if invalidTemplateResult.OK || len(invalidTemplateResult.Diagnostics) != 1 || string(invalidTemplateResult.Diagnostics[0].Category) != "parse_error" {
		t.Fatalf("unexpected invalid template result: %+v", invalidTemplateResult)
	}

	invalidDestination := readYAMLFixture(t, "yaml", "slice-99-merge", "invalid-destination.json")
	invalidDestinationResult := MergeYAML(invalidDestination["template"].(string), invalidDestination["destination"].(string), DialectYAML)
	if invalidDestinationResult.OK || len(invalidDestinationResult.Diagnostics) != 1 || string(invalidDestinationResult.Diagnostics[0].Category) != "destination_parse_error" {
		t.Fatalf("unexpected invalid destination result: %+v", invalidDestinationResult)
	}
}

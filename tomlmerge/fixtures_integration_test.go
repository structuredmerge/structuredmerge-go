package tomlmerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
)

func readTOMLFixture(t *testing.T, parts ...string) map[string]any {
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

func TestSharedFixtureTOMLFeatureProfile(t *testing.T) {
	fixture := readTOMLFixture(t, "diagnostics", "slice-90-toml-family-feature-profile", "toml-feature-profile.json")
	profile := TOMLFeatureProfileInfo()

	expected := fixture["feature_profile"].(map[string]any)
	if profile.Family != expected["family"].(string) {
		t.Fatalf("unexpected family: %+v", profile)
	}
	if len(profile.SupportedDialects) != 1 || string(profile.SupportedDialects[0]) != "toml" {
		t.Fatalf("unexpected dialects: %+v", profile.SupportedDialects)
	}
	if len(profile.SupportedPolicies) != 1 || profile.SupportedPolicies[0].Name != "destination_wins_array" {
		t.Fatalf("unexpected policies: %+v", profile.SupportedPolicies)
	}

	nativeProfile := TOMLBackendFeatureProfileInfo(BackendNative)
	if nativeProfile.Backend != "go-toml-v2" || nativeProfile.BackendRef == nil || nativeProfile.BackendRef.Family != "builtin" {
		t.Fatalf("unexpected native backend profile: %+v", nativeProfile)
	}
	pigeonProfile := TOMLBackendFeatureProfileInfo(BackendPigeon)
	if pigeonProfile.Backend != "pigeon" || pigeonProfile.BackendRef == nil || pigeonProfile.BackendRef.Family != "peg" {
		t.Fatalf("unexpected pigeon backend profile: %+v", pigeonProfile)
	}
}

func TestSharedFixtureTOMLBackendFeatureProfiles(t *testing.T) {
	fixture := readTOMLFixture(t, "diagnostics", "slice-135-toml-family-backend-feature-profiles", "go-toml-backend-feature-profiles.json")

	nativeProfile := TOMLBackendFeatureProfileInfo(BackendNative)
	if nativeProfile.Backend != fixture["native"].(map[string]any)["backend"].(string) {
		t.Fatalf("unexpected native backend feature profile: %+v", nativeProfile)
	}

	pigeonProfile := TOMLBackendFeatureProfileInfo(BackendPigeon)
	if pigeonProfile.Backend != fixture["pigeon"].(map[string]any)["backend"].(string) {
		t.Fatalf("unexpected pigeon backend feature profile: %+v", pigeonProfile)
	}
}

func TestSharedFixtureTOMLPlanContexts(t *testing.T) {
	fixture := readTOMLFixture(t, "diagnostics", "slice-136-toml-family-plan-contexts", "go-toml-plan-contexts.json")

	nativeContext := TOMLPlanContext(BackendNative)
	if nativeContext.FamilyProfile.Family != fixture["native"].(map[string]any)["family_profile"].(map[string]any)["family"].(string) {
		t.Fatalf("unexpected native plan context: %+v", nativeContext)
	}
	if nativeContext.FeatureProfile == nil || nativeContext.FeatureProfile.Backend != fixture["native"].(map[string]any)["feature_profile"].(map[string]any)["backend"].(string) {
		t.Fatalf("unexpected native feature profile: %+v", nativeContext.FeatureProfile)
	}

	pigeonContext := TOMLPlanContext(BackendPigeon)
	if pigeonContext.FamilyProfile.Family != fixture["pigeon"].(map[string]any)["family_profile"].(map[string]any)["family"].(string) {
		t.Fatalf("unexpected pigeon plan context: %+v", pigeonContext)
	}
	if pigeonContext.FeatureProfile == nil || pigeonContext.FeatureProfile.Backend != fixture["pigeon"].(map[string]any)["feature_profile"].(map[string]any)["backend"].(string) {
		t.Fatalf("unexpected pigeon feature profile: %+v", pigeonContext.FeatureProfile)
	}
}

func TestSharedFixtureTOMLManifest(t *testing.T) {
	fixture := readTOMLFixture(t, "conformance", "slice-137-toml-family-manifest", "toml-family-manifest.json")
	source, err := json.Marshal(fixture)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}

	var manifest astmerge.ConformanceManifest
	if err := json.Unmarshal(source, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}

	if path := astmerge.ConformanceFamilyFeatureProfilePath(manifest, "toml"); path == nil || filepath.Join(path...) != filepath.Join("diagnostics", "slice-90-toml-family-feature-profile", "toml-feature-profile.json") {
		t.Fatalf("unexpected family feature profile path: %v", path)
	}
	if path := astmerge.ConformanceFixturePath(manifest, "toml", "analysis"); path == nil || filepath.Join(path...) != filepath.Join("toml", "slice-92-structure", "table-and-array.json") {
		t.Fatalf("unexpected analysis fixture path: %v", path)
	}
	if path := astmerge.ConformanceFixturePath(manifest, "toml", "matching"); path == nil || filepath.Join(path...) != filepath.Join("toml", "slice-93-matching", "path-equality.json") {
		t.Fatalf("unexpected matching fixture path: %v", path)
	}
	if path := astmerge.ConformanceFixturePath(manifest, "toml", "merge"); path == nil || filepath.Join(path...) != filepath.Join("toml", "slice-94-merge", "table-merge.json") {
		t.Fatalf("unexpected merge fixture path: %v", path)
	}
}

func TestSharedFixtureTOMLParse(t *testing.T) {
	valid := readTOMLFixture(t, "toml", "slice-91-parse", "valid-document.json")
	validResult := ParseTOML(valid["source"].(string), DialectTOML)
	if !validResult.OK || validResult.Analysis == nil || string(validResult.Analysis.RootKind) != "table" {
		t.Fatalf("unexpected valid parse result: %+v", validResult)
	}
	if len(validResult.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", validResult.Diagnostics)
	}

	invalid := readTOMLFixture(t, "toml", "slice-91-parse", "invalid-document.json")
	invalidResult := ParseTOML(invalid["source"].(string), DialectTOML)
	if invalidResult.OK {
		t.Fatalf("expected invalid parse failure: %+v", invalidResult)
	}
	if len(invalidResult.Diagnostics) != 1 || string(invalidResult.Diagnostics[0].Category) != "parse_error" {
		t.Fatalf("unexpected invalid diagnostics: %+v", invalidResult.Diagnostics)
	}

	pigeonValid := ParseTOMLWithBackend(valid["source"].(string), DialectTOML, BackendPigeon)
	if !pigeonValid.OK || pigeonValid.Analysis == nil {
		t.Fatalf("unexpected pigeon parse success result: %+v", pigeonValid)
	}
	pigeonInvalid := ParseTOMLWithBackend(invalid["source"].(string), DialectTOML, BackendPigeon)
	if pigeonInvalid.OK || len(pigeonInvalid.Diagnostics) != 1 || string(pigeonInvalid.Diagnostics[0].Category) != "parse_error" {
		t.Fatalf("unexpected pigeon parse failure result: %+v", pigeonInvalid)
	}
}

func TestSharedFixtureTOMLStructure(t *testing.T) {
	fixture := readTOMLFixture(t, "toml", "slice-92-structure", "table-and-array.json")
	result := ParseTOML(fixture["source"].(string), DialectTOML)
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

func TestSharedFixtureTOMLMatching(t *testing.T) {
	fixture := readTOMLFixture(t, "toml", "slice-93-matching", "path-equality.json")
	template := ParseTOML(fixture["template"].(string), DialectTOML)
	destination := ParseTOML(fixture["destination"].(string), DialectTOML)
	result := MatchTOMLOwners(*template.Analysis, *destination.Analysis)

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

func TestSharedFixtureTOMLMerge(t *testing.T) {
	mergeFixture := readTOMLFixture(t, "toml", "slice-94-merge", "table-merge.json")
	mergeResult := MergeTOML(mergeFixture["template"].(string), mergeFixture["destination"].(string), DialectTOML)
	if !mergeResult.OK || mergeResult.Output == nil {
		t.Fatalf("expected merge success: %+v", mergeResult)
	}
	if *mergeResult.Output != mergeFixture["expected"].(map[string]any)["output"].(string) {
		t.Fatalf("unexpected merge output:\n%s", *mergeResult.Output)
	}

	pigeonMergeResult := MergeTOMLWithBackend(mergeFixture["template"].(string), mergeFixture["destination"].(string), DialectTOML, BackendPigeon)
	if !pigeonMergeResult.OK || pigeonMergeResult.Output == nil {
		t.Fatalf("expected pigeon merge success: %+v", pigeonMergeResult)
	}
	if *pigeonMergeResult.Output != mergeFixture["expected"].(map[string]any)["output"].(string) {
		t.Fatalf("unexpected pigeon merge output:\n%s", *pigeonMergeResult.Output)
	}

	invalidTemplate := readTOMLFixture(t, "toml", "slice-94-merge", "invalid-template.json")
	invalidTemplateResult := MergeTOML(invalidTemplate["template"].(string), invalidTemplate["destination"].(string), DialectTOML)
	if invalidTemplateResult.OK || len(invalidTemplateResult.Diagnostics) != 1 || string(invalidTemplateResult.Diagnostics[0].Category) != "parse_error" {
		t.Fatalf("unexpected invalid template result: %+v", invalidTemplateResult)
	}

	invalidDestination := readTOMLFixture(t, "toml", "slice-94-merge", "invalid-destination.json")
	invalidDestinationResult := MergeTOML(invalidDestination["template"].(string), invalidDestination["destination"].(string), DialectTOML)
	if invalidDestinationResult.OK || len(invalidDestinationResult.Diagnostics) != 1 || string(invalidDestinationResult.Diagnostics[0].Category) != "destination_parse_error" {
		t.Fatalf("unexpected invalid destination result: %+v", invalidDestinationResult)
	}

	pigeonInvalidDestinationResult := MergeTOMLWithBackend(invalidDestination["template"].(string), invalidDestination["destination"].(string), DialectTOML, BackendPigeon)
	if pigeonInvalidDestinationResult.OK || len(pigeonInvalidDestinationResult.Diagnostics) != 1 || string(pigeonInvalidDestinationResult.Diagnostics[0].Category) != "destination_parse_error" {
		t.Fatalf("unexpected pigeon invalid destination result: %+v", pigeonInvalidDestinationResult)
	}
}

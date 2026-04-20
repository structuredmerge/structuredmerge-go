package markdownmerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
)

func readMarkdownFixture(t *testing.T, parts ...string) map[string]any {
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

func TestSharedFixtureMarkdownFeatureProfile(t *testing.T) {
	fixture := readMarkdownFixture(t, "diagnostics", "slice-194-markdown-family-feature-profile", "markdown-feature-profile.json")
	profile := MarkdownFeatureProfileInfo()
	if profile.Family != fixture["feature_profile"].(map[string]any)["family"].(string) {
		t.Fatalf("unexpected family: %+v", profile)
	}
}

func TestSharedFixtureMarkdownBackendFeatureProfiles(t *testing.T) {
	fixture := readMarkdownFixture(t, "diagnostics", "slice-195-markdown-family-backend-feature-profiles", "go-markdown-backend-feature-profiles.json")
	backends := AvailableMarkdownBackends()
	if len(backends) != 2 || backends[0] != BackendGoldmark || backends[1] != BackendKreuzberg {
		t.Fatalf("unexpected backends: %+v", backends)
	}
	if native := MarkdownBackendFeatureProfileInfo(BackendGoldmark); native.Backend != fixture["native"].(map[string]any)["backend"].(string) {
		t.Fatalf("unexpected native backend profile: %+v", native)
	}
	if treeSitter := MarkdownBackendFeatureProfileInfo(BackendKreuzberg); treeSitter.Backend != fixture["tree_sitter"].(map[string]any)["backend"].(string) {
		t.Fatalf("unexpected tree-sitter backend profile: %+v", treeSitter)
	}
}

func TestSharedFixtureMarkdownPlanContexts(t *testing.T) {
	fixture := readMarkdownFixture(t, "diagnostics", "slice-196-markdown-family-plan-contexts", "go-markdown-plan-contexts.json")
	native := MarkdownPlanContextWithBackend(BackendGoldmark)
	if native.FamilyProfile.Family != fixture["native"].(map[string]any)["family_profile"].(map[string]any)["family"].(string) {
		t.Fatalf("unexpected native family profile: %+v", native)
	}
	if native.FeatureProfile == nil || native.FeatureProfile.Backend != fixture["native"].(map[string]any)["feature_profile"].(map[string]any)["backend"].(string) {
		t.Fatalf("unexpected native feature profile: %+v", native.FeatureProfile)
	}

	treeSitter := MarkdownPlanContextWithBackend(BackendKreuzberg)
	if treeSitter.FamilyProfile.Family != fixture["tree_sitter"].(map[string]any)["family_profile"].(map[string]any)["family"].(string) {
		t.Fatalf("unexpected tree-sitter family profile: %+v", treeSitter)
	}
	if treeSitter.FeatureProfile == nil || treeSitter.FeatureProfile.Backend != fixture["tree_sitter"].(map[string]any)["feature_profile"].(map[string]any)["backend"].(string) {
		t.Fatalf("unexpected tree-sitter feature profile: %+v", treeSitter.FeatureProfile)
	}
}

func TestSharedFixtureMarkdownManifest(t *testing.T) {
	fixture := readMarkdownFixture(t, "conformance", "slice-197-markdown-family-manifest", "markdown-family-manifest.json")
	source, err := json.Marshal(fixture)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}

	var manifest astmerge.ConformanceManifest
	if err := json.Unmarshal(source, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}

	if path := astmerge.ConformanceFamilyFeatureProfilePath(manifest, "markdown"); path == nil || filepath.Join(path...) != filepath.Join("diagnostics", "slice-194-markdown-family-feature-profile", "markdown-feature-profile.json") {
		t.Fatalf("unexpected family feature profile path: %v", path)
	}
	if path := astmerge.ConformanceFixturePath(manifest, "markdown", "analysis"); path == nil || filepath.Join(path...) != filepath.Join("markdown", "slice-198-analysis", "headings-and-code-fences.json") {
		t.Fatalf("unexpected analysis fixture path: %v", path)
	}
}

func TestSharedFixtureMarkdownAnalysis(t *testing.T) {
	fixture := readMarkdownFixture(t, "markdown", "slice-198-analysis", "headings-and-code-fences.json")
	expectedOwners := fixture["expected"].(map[string]any)["owners"].([]any)

	for _, backend := range []MarkdownBackend{BackendGoldmark, BackendKreuzberg} {
		result := ParseMarkdownWithBackend(fixture["source"].(string), DialectMarkdown, backend)
		if !result.OK || result.Analysis == nil {
			t.Fatalf("expected parse success for %s: %+v", backend, result)
		}
		if result.Analysis.RootKind != RootDocument {
			t.Fatalf("unexpected root kind for %s: %+v", backend, result.Analysis.RootKind)
		}
		if len(result.Analysis.Owners) != len(expectedOwners) {
			t.Fatalf("unexpected owner count for %s: %+v", backend, result.Analysis.Owners)
		}
	}
}

func TestSharedFixtureMarkdownMatching(t *testing.T) {
	fixture := readMarkdownFixture(t, "markdown", "slice-199-matching", "path-equality.json")
	for _, backend := range []MarkdownBackend{BackendGoldmark, BackendKreuzberg} {
		template := ParseMarkdownWithBackend(fixture["template"].(string), DialectMarkdown, backend)
		destination := ParseMarkdownWithBackend(fixture["destination"].(string), DialectMarkdown, backend)
		if !template.OK || template.Analysis == nil || !destination.OK || destination.Analysis == nil {
			t.Fatalf("expected parse success for %s", backend)
		}

		result := MatchMarkdownOwners(*template.Analysis, *destination.Analysis)
		if len(result.Matched) != len(fixture["expected"].(map[string]any)["matched"].([]any)) {
			t.Fatalf("unexpected matches for %s: %+v", backend, result.Matched)
		}
		if len(result.UnmatchedTemplate) != len(fixture["expected"].(map[string]any)["unmatched_template"].([]any)) {
			t.Fatalf("unexpected unmatched template for %s: %+v", backend, result.UnmatchedTemplate)
		}
	}
}

package goldmarkmerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/structuredmerge/structuredmerge-go/markdownmerge"
)

func readGoldmarkFixture(t *testing.T, parts ...string) map[string]any {
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

func TestSharedFixtureMarkdownProviderFeatureProfile(t *testing.T) {
	fixture := readGoldmarkFixture(t, "diagnostics", "slice-204-markdown-provider-feature-profiles", "go-markdown-provider-feature-profiles.json")
	if profile := MarkdownBackendFeatureProfileInfo(); profile.Backend != fixture["providers"].(map[string]any)["goldmark"].(map[string]any)["feature_profile"].(map[string]any)["backend"].(string) {
		t.Fatalf("unexpected provider profile: %+v", profile)
	}
	if len(AvailableMarkdownBackends()) != 1 || AvailableMarkdownBackends()[0] != "goldmark" {
		t.Fatalf("unexpected backends: %+v", AvailableMarkdownBackends())
	}
}

func TestSharedFixtureMarkdownProviderPlanContext(t *testing.T) {
	fixture := readGoldmarkFixture(t, "diagnostics", "slice-205-markdown-provider-plan-contexts", "go-markdown-provider-plan-contexts.json")
	if context := MarkdownPlanContext(); context.FeatureProfile == nil || context.FeatureProfile.Backend != fixture["providers"].(map[string]any)["goldmark"].(map[string]any)["feature_profile"].(map[string]any)["backend"].(string) {
		t.Fatalf("unexpected provider plan context: %+v", context)
	}
}

func TestSharedFixtureMarkdownProviderAnalysisAndMatching(t *testing.T) {
	analysisFixture := readGoldmarkFixture(t, "markdown", "slice-198-analysis", "headings-and-code-fences.json")
	matchingFixture := readGoldmarkFixture(t, "markdown", "slice-199-matching", "path-equality.json")

	analysis := ParseMarkdown(analysisFixture["source"].(string), markdownmerge.DialectMarkdown)
	if !analysis.OK || analysis.Analysis == nil {
		t.Fatalf("expected parse success: %+v", analysis)
	}

	template := ParseMarkdown(matchingFixture["template"].(string), markdownmerge.DialectMarkdown)
	destination := ParseMarkdown(matchingFixture["destination"].(string), markdownmerge.DialectMarkdown)
	if !template.OK || template.Analysis == nil || !destination.OK || destination.Analysis == nil {
		t.Fatalf("expected parse success for matching fixtures")
	}

	result := MatchMarkdownOwners(*template.Analysis, *destination.Analysis)
	if len(result.Matched) != len(matchingFixture["expected"].(map[string]any)["matched"].([]any)) {
		t.Fatalf("unexpected matches: %+v", result.Matched)
	}
}

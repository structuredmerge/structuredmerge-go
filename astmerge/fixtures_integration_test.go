package astmerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func readDiagnosticFixture(t *testing.T, parts ...string) map[string]any {
	t.Helper()

	pathParts := append([]string{"..", "..", "fixtures"}, parts...)
	path := filepath.Join(pathParts...)
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var fixture map[string]any
	if err := json.Unmarshal(source, &fixture); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	return fixture
}

func TestSharedFixtureDiagnosticVocabulary(t *testing.T) {
	fixture := readDiagnosticFixture(t, "diagnostics", "slice-02-core", "diagnostic-categories.json")

	severities := []DiagnosticSeverity{
		SeverityInfo,
		SeverityWarning,
		SeverityError,
	}
	categories := []DiagnosticCategory{
		CategoryParseError,
		CategoryDestinationParseError,
		CategoryUnsupportedFeature,
		CategoryFallbackApplied,
		CategoryAmbiguity,
	}

	expectedSeverities := fixture["severities"].([]any)
	if len(severities) != len(expectedSeverities) {
		t.Fatalf("unexpected severities: %+v", severities)
	}
	for index, severity := range severities {
		if string(severity) != expectedSeverities[index].(string) {
			t.Fatalf("unexpected severity at %d: %s", index, severity)
		}
	}

	expectedCategories := fixture["categories"].([]any)
	if len(categories) != len(expectedCategories) {
		t.Fatalf("unexpected categories: %+v", categories)
	}
	for index, category := range categories {
		if string(category) != expectedCategories[index].(string) {
			t.Fatalf("unexpected category at %d: %s", index, category)
		}
	}
}

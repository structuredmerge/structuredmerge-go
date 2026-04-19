package jsonmerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func readJSONFixture(t *testing.T, parts ...string) map[string]any {
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

func TestSharedFixtureJSONCCommentsAccepted(t *testing.T) {
	fixture := readJSONFixture(t, "jsonc", "slice-04-parse", "comments-accepted.json")
	expected := fixture["expected"].(map[string]any)

	result := ParseJSON(fixture["source"].(string), DialectJSONC)
	if result.OK != expected["ok"].(bool) {
		t.Fatalf("unexpected parse status: %+v", result)
	}
	if result.Analysis == nil || result.Analysis.AllowsComments != expected["allows_comments"].(bool) {
		t.Fatalf("unexpected analysis: %+v", result.Analysis)
	}
	if len(result.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", result.Diagnostics)
	}
}

func TestSharedFixtureJSONObjectMerge(t *testing.T) {
	fixture := readJSONFixture(t, "json", "slice-09-merge", "object-merge.json")
	expected := fixture["expected"].(map[string]any)

	result := MergeJSON(
		fixture["template"].(string),
		fixture["destination"].(string),
		DialectJSON,
	)
	if !result.OK || result.Output == nil {
		t.Fatalf("expected merge success, got diagnostics: %+v", result.Diagnostics)
	}
	if *result.Output != expected["output"].(string) {
		t.Fatalf("unexpected merged output: %q", *result.Output)
	}
}

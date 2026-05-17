package astmergegit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func readFixture(t *testing.T, parts ...string) map[string]any {
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

func decodeFixtureValue[T any](t *testing.T, raw any) T {
	t.Helper()
	source, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("marshal fixture value: %v", err)
	}
	var decoded T
	if err := json.Unmarshal(source, &decoded); err != nil {
		t.Fatalf("decode fixture value: %v", err)
	}
	return decoded
}

func TestGitMerge3ContractFixture(t *testing.T) {
	fixture := readFixture(t, "diagnostics", "slice-950-git-merge3-contract", "git-merge3-contract.json")
	contract := fixture["contract"].(map[string]any)
	if contract["package"] != "ast-merge-git" || contract["operation"] != "merge3" {
		t.Fatalf("unexpected contract metadata: %+v", contract)
	}

	cases := fixture["cases"].([]any)
	for _, rawCase := range cases {
		testCase := rawCase.(map[string]any)
		t.Run(testCase["case_id"].(string), func(t *testing.T) {
			request := decodeFixtureValue[Merge3Request](t, testCase["request"])
			expected := testCase["expected"].(map[string]any)

			result := Merge3(request)
			if result.OK != expected["ok"].(bool) {
				t.Fatalf("unexpected ok: got %v result=%+v", result.OK, result)
			}
			if len(result.Conflicts) != int(expected["conflict_count"].(float64)) {
				t.Fatalf("unexpected conflicts: %+v", result.Conflicts)
			}
			if expected["reparse_after_render"] == nil {
				if result.ReparseAfterRender != nil {
					t.Fatalf("expected nil reparse result, got %+v", *result.ReparseAfterRender)
				}
			} else if result.ReparseAfterRender == nil || *result.ReparseAfterRender != expected["reparse_after_render"].(bool) {
				t.Fatalf("unexpected reparse result: %+v", result.ReparseAfterRender)
			}

			if result.OK {
				if result.MergedSource == nil {
					t.Fatal("expected merged source")
				}
				var merged any
				if err := json.Unmarshal([]byte(*result.MergedSource), &merged); err != nil {
					t.Fatalf("merged output should parse: %v", err)
				}
				if !reflect.DeepEqual(merged, expected["merged_json"]) {
					t.Fatalf("unexpected merged JSON: got=%+v expected=%+v", merged, expected["merged_json"])
				}
			} else {
				categories := make([]string, 0, len(result.Conflicts))
				paths := make([]string, 0, len(result.Conflicts))
				for _, conflict := range result.Conflicts {
					categories = append(categories, conflict.Category)
					paths = append(paths, conflict.Path)
				}
				if !reflect.DeepEqual(categories, decodeFixtureValue[[]string](t, expected["conflict_categories"])) {
					t.Fatalf("unexpected conflict categories: %+v", categories)
				}
				if !reflect.DeepEqual(paths, decodeFixtureValue[[]string](t, expected["conflict_paths"])) {
					t.Fatalf("unexpected conflict paths: %+v", paths)
				}
			}
		})
	}
}

func TestGoMerge3Fixture(t *testing.T) {
	fixture := readFixture(t, "go", "slice-952-go-merge3", "go-merge3.json")
	cases := fixture["cases"].([]any)
	for _, rawCase := range cases {
		testCase := rawCase.(map[string]any)
		t.Run(testCase["case_id"].(string), func(t *testing.T) {
			request := Merge3Request{
				BaseSource:   testCase["base_source"].(string),
				OursSource:   testCase["ours_source"].(string),
				TheirsSource: testCase["theirs_source"].(string),
				PathName:     testCase["path_name"].(string),
				Language:     "go",
				Dialect:      "go",
				ProfileID:    "go.source",
			}
			result := Merge3(request)
			expected := testCase["expected"].(map[string]any)
			if result.OK != expected["ok"].(bool) {
				t.Fatalf("unexpected ok=%v diagnostics=%+v conflicts=%+v", result.OK, result.Diagnostics, result.Conflicts)
			}
			if len(result.Conflicts) != int(expected["conflict_count"].(float64)) {
				t.Fatalf("unexpected conflicts: %+v", result.Conflicts)
			}
			if result.OK {
				if result.MergedSource == nil {
					t.Fatal("expected merged source")
				}
				for _, rawNeedle := range expected["must_contain"].([]any) {
					needle := rawNeedle.(string)
					if !strings.Contains(*result.MergedSource, needle) {
						t.Fatalf("expected merged source to contain %q:\n%s", needle, *result.MergedSource)
					}
				}
				if result.ReparseAfterRender == nil || !*result.ReparseAfterRender {
					t.Fatalf("expected output to reparse: %+v", result)
				}
			} else {
				expectedCategories := expected["conflict_categories"].([]any)
				expectedPaths := expected["conflict_paths"].([]any)
				for index, conflict := range result.Conflicts {
					if conflict.Category != expectedCategories[index].(string) || conflict.Path != expectedPaths[index].(string) {
						t.Fatalf("unexpected conflict at %d: %+v", index, conflict)
					}
				}
			}
		})
	}
}

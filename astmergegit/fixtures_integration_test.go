package astmergegit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/godstmerge"
	"github.com/structuredmerge/structuredmerge-go/gomerge"
	"github.com/structuredmerge/structuredmerge-go/goparsermerge"
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
			if rawRenderReport, ok := expected["render_report"]; ok {
				expectedRenderReport := decodeFixtureValue[Merge3RenderReport](t, rawRenderReport)
				if result.RenderReport != expectedRenderReport {
					t.Fatalf("unexpected render report: got=%+v expected=%+v", result.RenderReport, expectedRenderReport)
				}
			}
			if rawFormattingPreservation, ok := expected["formatting_preservation"]; ok {
				expectedFormattingPreservation := decodeFixtureValue[FormattingPreservationReport](t, rawFormattingPreservation)
				if result.FormattingPreservation != expectedFormattingPreservation {
					t.Fatalf("unexpected formatting preservation: got=%+v expected=%+v", result.FormattingPreservation, expectedFormattingPreservation)
				}
			}
			if rawSecondaryMetrics, ok := expected["secondary_formatting_metrics"]; ok {
				expectedSecondaryMetrics := decodeFixtureValue[SecondaryFormattingMetricsReport](t, rawSecondaryMetrics)
				if !reflect.DeepEqual(result.SecondaryFormattingMetrics, expectedSecondaryMetrics) {
					t.Fatalf("unexpected secondary formatting metrics: got=%+v expected=%+v", result.SecondaryFormattingMetrics, expectedSecondaryMetrics)
				}
			}
			if rawDefaultDriverEvaluation, ok := expected["default_driver_evaluation"]; ok {
				expectedDefaultDriverEvaluation := decodeFixtureValue[DefaultDriverEvaluation](t, rawDefaultDriverEvaluation)
				if !reflect.DeepEqual(result.DefaultDriverEvaluation, expectedDefaultDriverEvaluation) {
					t.Fatalf("unexpected default driver evaluation: got=%+v expected=%+v", result.DefaultDriverEvaluation, expectedDefaultDriverEvaluation)
				}
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
				for _, rawNeedle := range decodeFixtureValue[[]string](t, expected["conflicted_source_contains"]) {
					if result.ConflictedSource == nil || !strings.Contains(*result.ConflictedSource, rawNeedle) {
						source := "<nil>"
						if result.ConflictedSource != nil {
							source = *result.ConflictedSource
						}
						t.Fatalf("expected conflicted source to contain %q:\n%s", rawNeedle, source)
					}
				}
			}
		})
	}
}

func TestGitCommentDeltaSemanticsFixture(t *testing.T) {
	fixture := readFixture(t, "diagnostics", "slice-953-git-comment-delta-semantics", "git-comment-delta-semantics.json")
	contract := fixture["contract"].(map[string]any)
	if contract["package"] != "ast-merge-git" || contract["operation"] != "comment_delta_semantics" {
		t.Fatalf("unexpected contract metadata: %+v", contract)
	}
	owner := fixture["owner"].(map[string]any)

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		t.Run(testCase["case_id"].(string), func(t *testing.T) {
			result := MergeCommentDelta(
				optionalString(testCase["base_comment"]),
				optionalString(testCase["ours_comment"]),
				optionalString(testCase["theirs_comment"]),
				owner["path"].(string),
			)
			expected := testCase["expected"].(map[string]any)
			if result.OK != expected["ok"].(bool) {
				t.Fatalf("unexpected ok: %+v", result)
			}
			if len(result.Conflicts) != int(expected["conflict_count"].(float64)) {
				t.Fatalf("unexpected conflicts: %+v", result.Conflicts)
			}
			if rawMerged, ok := expected["merged_comment"]; ok {
				expectedComment := optionalString(rawMerged)
				if !stringPointersEqual(result.MergedComment, expectedComment) {
					t.Fatalf("unexpected merged comment: got=%+v expected=%+v", result.MergedComment, expectedComment)
				}
			}
			if rawCategories, ok := expected["conflict_categories"]; ok {
				categories := make([]string, 0, len(result.Conflicts))
				for _, conflict := range result.Conflicts {
					categories = append(categories, conflict.Category)
				}
				if !reflect.DeepEqual(categories, decodeFixtureValue[[]string](t, rawCategories)) {
					t.Fatalf("unexpected comment conflict categories: %+v", categories)
				}
			}
			if rawOwnerPath, ok := expected["comment_owner_path"]; ok && owner["path"] != rawOwnerPath {
				t.Fatalf("unexpected owner path: %s", owner["path"])
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
			assertGoMerge3ReportExpectations(t, result, expected)
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
				if rawNeedles, ok := expected["must_not_contain"].([]any); ok {
					for _, rawNeedle := range rawNeedles {
						needle := rawNeedle.(string)
						if strings.Contains(*result.MergedSource, needle) {
							t.Fatalf("expected merged source not to contain %q:\n%s", needle, *result.MergedSource)
						}
					}
				}
				if expectedSource, ok := expected["expected_source"].(string); ok && *result.MergedSource != expectedSource {
					t.Fatalf("unexpected merged source\nexpected:\n%s\nactual:\n%s", expectedSource, *result.MergedSource)
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
				for _, rawNeedle := range []string{"<<<<<<< ours", "||||||| base", "=======", ">>>>>>> theirs"} {
					if result.ConflictedSource == nil || !strings.Contains(*result.ConflictedSource, rawNeedle) {
						source := "<nil>"
						if result.ConflictedSource != nil {
							source = *result.ConflictedSource
						}
						t.Fatalf("expected conflicted source to contain %q:\n%s", rawNeedle, source)
					}
				}
			}
		})
	}
}

func TestOwnedRegionConflictPlacementFixture(t *testing.T) {
	fixture := readFixture(t, "diagnostics", "slice-958-owned-region-conflict-placement", "owned-region-conflict-placement.json")
	cases := fixture["cases"].([]any)
	for _, rawCase := range cases {
		testCase := rawCase.(map[string]any)
		t.Run(testCase["case_id"].(string), func(t *testing.T) {
			if testCase["force_unsafe_region"] == true {
				t.Skip("unsafe-region forcing is a fixture target for the renderer policy layer")
			}
			request := Merge3Request{
				BaseSource:         testCase["base_source"].(string),
				OursSource:         testCase["ours_source"].(string),
				TheirsSource:       testCase["theirs_source"].(string),
				PathName:           testCase["path_name"].(string),
				Language:           testCase["language"].(string),
				Dialect:            testCase["language"].(string),
				ProfileID:          testCase["language"].(string) + ".source",
				ConflictMarkerSize: 7,
			}
			result := Merge3(request)
			expected := testCase["expected"].(map[string]any)
			if result.OK != expected["ok"].(bool) {
				t.Fatalf("unexpected ok=%v diagnostics=%+v conflicts=%+v", result.OK, result.Diagnostics, result.Conflicts)
			}
			expectedRegions := expected["owned_regions"].([]any)
			if len(result.OwnedRegions) != len(expectedRegions) {
				t.Fatalf("unexpected owned regions: %+v expected %+v", result.OwnedRegions, expectedRegions)
			}
			for index, rawRegion := range expectedRegions {
				expectedRegion := rawRegion.(map[string]any)
				assertOwnedRegionMatches(t, result.OwnedRegions[index], expectedRegion)
			}
			if rawRenderReport, ok := expected["render_report"].(map[string]any); ok {
				if result.RenderReport.Strategy != rawRenderReport["strategy"].(string) {
					t.Fatalf("unexpected render report: %+v expected %+v", result.RenderReport, rawRenderReport)
				}
			}
			if result.ConflictedSource != nil {
				if rawNeedles, ok := expected["conflicted_source_contains"].([]any); ok {
					for _, rawNeedle := range rawNeedles {
						needle := rawNeedle.(string)
						if !strings.Contains(*result.ConflictedSource, needle) {
							t.Fatalf("expected conflicted source to contain %q:\n%s", needle, *result.ConflictedSource)
						}
					}
				}
				if rawNeedles, ok := expected["conflicted_source_not_contains"].([]any); ok {
					for _, rawNeedle := range rawNeedles {
						needle := rawNeedle.(string)
						if strings.Contains(*result.ConflictedSource, needle) {
							t.Fatalf("expected conflicted source not to contain %q:\n%s", needle, *result.ConflictedSource)
						}
					}
				}
			}
		})
	}
}

func TestOwnedRegionConflictPlacementAcrossGoBackends(t *testing.T) {
	fixture := readFixture(t, "diagnostics", "slice-958-owned-region-conflict-placement", "owned-region-conflict-placement.json")
	backends := map[string]func(Merge3Request) Merge3Response{
		"tree-sitter": Merge3Go,
		"go-parser": func(request Merge3Request) Merge3Response {
			return Merge3GoWithParser(request, func(source string, dialect gomerge.GoDialect) astmerge.ParseResult[gomerge.GoAnalysis] {
				return goparsermerge.ParseGo(source, dialect)
			})
		},
		"go-dst": func(request Merge3Request) Merge3Response {
			return merge3GoWithParserReport(request, func(source string, dialect gomerge.GoDialect) astmerge.ParseResult[gomerge.GoAnalysis] {
				return godstmerge.ParseGo(source, dialect)
			}, godstmerge.BackendGoDST, "github.com/dave/dst")
		},
	}
	expectedBackendReports := map[string]Merge3RenderReport{
		"tree-sitter": {BackendID: string(gomerge.BackendTreeSitter), ParserIdentity: "tree-sitter-go"},
		"go-parser":   {BackendID: goparsermerge.BackendGoParser, ParserIdentity: "go/parser"},
		"go-dst":      {BackendID: godstmerge.BackendGoDST, ParserIdentity: "github.com/dave/dst"},
	}
	cases := fixture["cases"].([]any)
	for backend, merge := range backends {
		t.Run(backend, func(t *testing.T) {
			for _, rawCase := range cases {
				testCase := rawCase.(map[string]any)
				if testCase["language"] != "go" || testCase["force_unsafe_region"] == true {
					continue
				}
				t.Run(testCase["case_id"].(string), func(t *testing.T) {
					request := Merge3Request{
						BaseSource:         testCase["base_source"].(string),
						OursSource:         testCase["ours_source"].(string),
						TheirsSource:       testCase["theirs_source"].(string),
						PathName:           testCase["path_name"].(string),
						Language:           "go",
						Dialect:            "go",
						ProfileID:          "go.source",
						ConflictMarkerSize: 7,
					}
					result := merge(request)
					expected := testCase["expected"].(map[string]any)
					if result.OK != expected["ok"].(bool) {
						t.Fatalf("unexpected ok=%v diagnostics=%+v conflicts=%+v", result.OK, result.Diagnostics, result.Conflicts)
					}
					expectedRegions := expected["owned_regions"].([]any)
					if len(result.OwnedRegions) != len(expectedRegions) {
						t.Fatalf("unexpected owned regions: %+v expected %+v", result.OwnedRegions, expectedRegions)
					}
					expectedBackend := expectedBackendReports[backend]
					for index, rawRegion := range expectedRegions {
						expectedRegion := rawRegion.(map[string]any)
						assertOwnedRegionMatches(t, result.OwnedRegions[index], expectedRegion)
						if result.OwnedRegions[index].BackendID != expectedBackend.BackendID ||
							result.OwnedRegions[index].ParserIdentity != expectedBackend.ParserIdentity {
							t.Fatalf("unexpected owned-region backend for %s: %+v expected %+v", backend, result.OwnedRegions[index], expectedBackend)
						}
					}
				})
			}
		})
	}
}

func assertOwnedRegionMatches(t *testing.T, region OwnedRegionReport, expected map[string]any) {
	t.Helper()
	if region.OwnerPath != expected["owner_path"].(string) ||
		region.RegionKind != expected["region_kind"].(string) ||
		region.CanReplace != expected["can_replace"].(bool) ||
		region.CanLineMerge != expected["can_line_merge"].(bool) ||
		region.RequiresReparse != expected["requires_reparse"].(bool) {
		t.Fatalf("unexpected owned region: %+v expected %+v", region, expected)
	}
	expectedLineRange := expected["line_range"].([]any)
	if region.LineRange.Start != int(expectedLineRange[0].(float64)) ||
		region.LineRange.End != int(expectedLineRange[1].(float64)) {
		t.Fatalf("unexpected line range: %+v expected %+v", region.LineRange, expectedLineRange)
	}
	expectedAttachedSpans := expected["attached_spans"].([]any)
	if len(region.AttachedSpans) != len(expectedAttachedSpans) {
		t.Fatalf("unexpected attached spans: %+v expected %+v", region.AttachedSpans, expectedAttachedSpans)
	}
	for index, rawSpan := range expectedAttachedSpans {
		expectedSpan := rawSpan.(map[string]any)
		if region.AttachedSpans[index].Kind != expectedSpan["kind"].(string) {
			t.Fatalf("unexpected attached span: %+v expected %+v", region.AttachedSpans[index], expectedSpan)
		}
		expectedSpanLineRange := expectedSpan["line_range"].([]any)
		if region.AttachedSpans[index].LineRange.Start != int(expectedSpanLineRange[0].(float64)) ||
			region.AttachedSpans[index].LineRange.End != int(expectedSpanLineRange[1].(float64)) {
			t.Fatalf("unexpected attached span line range: %+v expected %+v", region.AttachedSpans[index], expectedSpan)
		}
	}
	if region.NodeID == "" || region.BackendID == "" || region.ParserIdentity == "" {
		t.Fatalf("owned region is missing identity fields: %+v", region)
	}
	if region.ByteRange.Start < 0 || region.ByteRange.End <= region.ByteRange.Start {
		t.Fatalf("owned region byte range is invalid: %+v", region)
	}
}

func optionalString(raw any) *string {
	if raw == nil {
		return nil
	}
	value := raw.(string)
	return &value
}

func assertGoMerge3ReportExpectations(t *testing.T, result Merge3Response, expected map[string]any) {
	t.Helper()
	if rawRenderReport, ok := expected["render_report"].(map[string]any); ok {
		if result.RenderReport.Strategy != rawRenderReport["strategy"].(string) {
			t.Fatalf("unexpected render strategy: %+v expected %+v", result.RenderReport, rawRenderReport)
		}
	}
	if rawFormatting, ok := expected["formatting_preservation"].(map[string]any); ok {
		if result.FormattingPreservation.LineDiffScore != rawFormatting["line_diff_score"].(float64) ||
			result.FormattingPreservation.CharacterDiffScore != rawFormatting["character_diff_score"].(float64) {
			t.Fatalf("unexpected formatting preservation: %+v expected %+v", result.FormattingPreservation, rawFormatting)
		}
	}
	if rawSecondary, ok := expected["secondary_formatting_metrics"].(map[string]any); ok {
		if result.SecondaryFormattingMetrics.SourceFragmentRetention != rawSecondary["source_fragment_retention"].(float64) {
			t.Fatalf("unexpected secondary metrics: %+v expected %+v", result.SecondaryFormattingMetrics, rawSecondary)
		}
	}
	if rawEvaluation, ok := expected["default_driver_evaluation"].(map[string]any); ok {
		if result.DefaultDriverEvaluation.Status != rawEvaluation["status"].(string) ||
			result.DefaultDriverEvaluation.FormattingThreshold != rawEvaluation["formatting_threshold"].(float64) {
			t.Fatalf("unexpected default driver evaluation: %+v expected %+v", result.DefaultDriverEvaluation, rawEvaluation)
		}
	}
}

func TestGoMerge3FixtureAcrossNativeBackends(t *testing.T) {
	fixture := readFixture(t, "go", "slice-952-go-merge3", "go-merge3.json")
	cases := fixture["cases"].([]any)
	backends := map[string]func(Merge3Request) Merge3Response{
		"tree-sitter": Merge3Go,
		"go-parser": func(request Merge3Request) Merge3Response {
			return Merge3GoWithParser(request, func(source string, dialect gomerge.GoDialect) astmerge.ParseResult[gomerge.GoAnalysis] {
				return goparsermerge.ParseGo(source, dialect)
			})
		},
		"go-dst": func(request Merge3Request) Merge3Response {
			return merge3GoWithParserReport(request, func(source string, dialect gomerge.GoDialect) astmerge.ParseResult[gomerge.GoAnalysis] {
				return godstmerge.ParseGo(source, dialect)
			}, godstmerge.BackendGoDST, "github.com/dave/dst")
		},
	}
	expectedBackendReports := map[string]Merge3RenderReport{
		"tree-sitter": {BackendID: string(gomerge.BackendTreeSitter), ParserIdentity: "tree-sitter-go"},
		"go-parser":   {BackendID: goparsermerge.BackendGoParser, ParserIdentity: "go/parser"},
		"go-dst":      {BackendID: godstmerge.BackendGoDST, ParserIdentity: "github.com/dave/dst"},
	}
	for backend, merge := range backends {
		t.Run(backend, func(t *testing.T) {
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
					result := merge(request)
					expected := testCase["expected"].(map[string]any)
					if result.OK != expected["ok"].(bool) {
						t.Fatalf("unexpected ok=%v diagnostics=%+v conflicts=%+v", result.OK, result.Diagnostics, result.Conflicts)
					}
					if len(result.Conflicts) != int(expected["conflict_count"].(float64)) {
						t.Fatalf("unexpected conflicts: %+v", result.Conflicts)
					}
					if result.OK && (result.ReparseAfterRender == nil || !*result.ReparseAfterRender) {
						source := "<nil>"
						if result.MergedSource != nil {
							source = *result.MergedSource
						}
						t.Fatalf("expected output to reparse: %+v\n%s", result, source)
					}
					if result.OK {
						expectedBackend := expectedBackendReports[backend]
						if result.RenderReport.BackendID != expectedBackend.BackendID ||
							result.RenderReport.ParserIdentity != expectedBackend.ParserIdentity {
							t.Fatalf("unexpected backend report for %s: %+v expected %+v", backend, result.RenderReport, expectedBackend)
						}
						expectedSource, hasExpectedSource := expected["expected_source"].(string)
						if hasExpectedSource && (result.MergedSource == nil || *result.MergedSource != expectedSource) {
							source := "<nil>"
							if result.MergedSource != nil {
								source = *result.MergedSource
							}
							t.Fatalf("unexpected merged source\nexpected:\n%s\nactual:\n%s", expectedSource, source)
						}
						if result.FormattingPreservation.LineDiffScore < 0.95 || result.FormattingPreservation.CharacterDiffScore < 0.95 {
							t.Fatalf("formatting preservation below gate: %+v", result.FormattingPreservation)
						}
						assertGoMerge3ReportExpectations(t, result, expected)
					}
				})
			}
		})
	}
}

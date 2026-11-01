package goparsermerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/gomerge"
	"github.com/structuredmerge/structuredmerge-go/treehaver"
)

func jsonReady(t *testing.T, value any) any {
	t.Helper()
	source, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal value: %v", err)
	}

	var normalized any
	if err := json.Unmarshal(source, &normalized); err != nil {
		t.Fatalf("decode value: %v", err)
	}

	return normalized
}

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

func TestSharedFixtureGoProviderFeatureProfile(t *testing.T) {
	familyFixture := readFixture(t, "diagnostics", "slice-109-go-family-feature-profile", "go-feature-profile.json")
	fixture := readFixture(t, "diagnostics", "slice-281-go-provider-feature-profiles", "go-go-provider-feature-profiles.json")
	familyProfile := gomerge.GoFeatureProfileInfo()
	if actual := jsonReady(t, map[string]any{
		"family":             familyProfile.Family,
		"supported_dialects": familyProfile.SupportedDialects,
		"supported_policies": familyProfile.SupportedPolicies,
	}); !reflect.DeepEqual(actual, familyFixture["feature_profile"]) {
		t.Fatalf("unexpected family profile: %+v", actual)
	}
	if len(AvailableGoBackends()) != 1 || AvailableGoBackends()[0] != BackendGoParser {
		t.Fatalf("unexpected backends: %+v", AvailableGoBackends())
	}
	expected := fixture["providers"].(map[string]any)["go_parser"].(map[string]any)["feature_profile"]
	profile := GoBackendFeatureProfileInfo()
	actual := jsonReady(t, map[string]any{
		"family":             profile.Family,
		"supported_dialects": profile.SupportedDialects,
		"supported_policies": profile.SupportedPolicies,
		"backend":            profile.Backend,
		"backend_ref": map[string]any{
			"id":     profile.BackendRef.ID,
			"family": profile.BackendRef.Family,
		},
	})
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("unexpected provider profile: %+v", profile)
	}
	if backend := treehaver.BackendReferenceByID(BackendGoParser); backend == nil || backend.ID != BackendGoParser || backend.Family != "native" {
		t.Fatalf("unexpected registered backend: %+v", backend)
	}
}

func TestSharedFixtureGoProviderPlanContext(t *testing.T) {
	fixture := readFixture(t, "diagnostics", "slice-282-go-provider-plan-contexts", "go-go-provider-plan-contexts.json")
	if context := GoPlanContext(); context.FeatureProfile == nil || context.FeatureProfile.Backend != fixture["providers"].(map[string]any)["go_parser"].(map[string]any)["feature_profile"].(map[string]any)["backend"].(string) {
		t.Fatalf("unexpected provider plan context: %+v", context)
	}
}

func TestSharedFixtureGoProviderParseAndMerge(t *testing.T) {
	parityFixture := readFixture(t, "go", "slice-114-native", "module-parity.json")
	parseResult := ParseGo(parityFixture["source"].(string), gomerge.DialectGo)
	if !parseResult.OK || parseResult.Analysis == nil {
		t.Fatalf("expected parse success: %+v", parseResult)
	}
	owners := make([]map[string]any, 0, len(parseResult.Analysis.Owners))
	for _, owner := range parseResult.Analysis.Owners {
		owners = append(owners, map[string]any{
			"path":       owner.Path,
			"owner_kind": owner.OwnerKind,
			"match_key":  owner.MatchKey,
		})
	}
	if actual := jsonReady(t, owners); !reflect.DeepEqual(actual, parityFixture["expected"].(map[string]any)["owners"]) {
		t.Fatalf("unexpected owners: %+v", actual)
	}

	mergeResult := MergeGo(parityFixture["template"].(string), parityFixture["destination"].(string), gomerge.DialectGo)
	if !mergeResult.OK || mergeResult.Output == nil {
		t.Fatalf("expected merge success: %+v", mergeResult)
	}
	if *mergeResult.Output != parityFixture["expected"].(map[string]any)["output"].(string) {
		t.Fatalf("unexpected merge output:\n%s", *mergeResult.Output)
	}
}

func TestGoParserPreservesNativeDocCommentOwnershipFixture(t *testing.T) {
	fixture := readFixture(t, "go", "slice-114-native", "doc-comment-ownership.json")
	dialect := gomerge.GoDialect(fixture["dialect"].(string))
	expected := fixture["expected"].(map[string]any)

	parsed := ParseGo(fixture["source"].(string), dialect)
	if !parsed.OK || parsed.Analysis == nil {
		t.Fatalf("expected native Go parse success: %+v", parsed)
	}
	if len(parsed.Analysis.Declarations) != 1 {
		t.Fatalf("expected one declaration, got %+v", parsed.Analysis.Declarations)
	}
	declaration := parsed.Analysis.Declarations[0]
	if declaration.Path != expected["declaration_path"].(string) {
		t.Fatalf("unexpected declaration path: %s", declaration.Path)
	}
	if declaration.Text != expected["declaration_text"].(string) {
		t.Fatalf("native go/ast doc ownership mismatch:\n%q", declaration.Text)
	}

	merged := MergeGo(
		expected["merge_template"].(string),
		expected["merge_destination"].(string),
		dialect,
	)
	if !merged.OK || merged.Output == nil {
		t.Fatalf("expected native Go merge success: %+v", merged)
	}
	if *merged.Output != expected["merge_output"].(string) {
		t.Fatalf("native go/ast merge dropped doc ownership:\n%q", *merged.Output)
	}
}

func TestGoParserPreservesCommentDirectiveOwnershipFixture(t *testing.T) {
	fixture := readFixture(t, "go", "slice-114-native", "comment-directive-ownership.json")
	parsed := ParseGo(fixture["source"].(string), gomerge.DialectGo)
	if !parsed.OK || parsed.Analysis == nil {
		t.Fatalf("expected native Go parse success: %+v", parsed)
	}
	expected := fixture["expected"].(map[string]any)
	for _, declaration := range parsed.Analysis.Declarations {
		if declaration.Path == expected["declaration_path"].(string) {
			if declaration.Text != expected["declaration_text"].(string) {
				t.Fatalf("go:embed directive ownership mismatch: %q", declaration.Text)
			}
			return
		}
	}
	t.Fatalf("missing declaration owner %s: %+v", expected["declaration_path"], parsed.Analysis.Declarations)
}

func TestGoParserMergeKeepsCommentDirectiveWithItsOwner(t *testing.T) {
	fixture := readFixture(t, "go", "slice-114-native", "comment-directive-ownership.json")
	expected := fixture["expected"].(map[string]any)
	merged := MergeGo(fixture["merge_template"].(string), fixture["merge_destination"].(string), gomerge.DialectGo)
	if !merged.OK || merged.Output == nil || *merged.Output != expected["merge_output"].(string) {
		actual := "<nil>"
		if merged.Output != nil {
			actual = *merged.Output
		}
		t.Fatalf("native merge changed go:embed ownership: %+v output=%q", merged, actual)
	}
}

func TestGoParserPreservesMethodOwnerIdentityFixture(t *testing.T) {
	fixture := readFixture(t, "go", "slice-114-native", "method-owner-identity.json")
	dialect := gomerge.GoDialect(fixture["dialect"].(string))
	expected := fixture["expected"].(map[string]any)

	parsed := ParseGo(fixture["source"].(string), dialect)
	if !parsed.OK || parsed.Analysis == nil {
		t.Fatalf("expected native Go parse success: %+v", parsed)
	}
	owners := make([]map[string]any, 0, len(parsed.Analysis.Owners))
	for _, owner := range parsed.Analysis.Owners {
		owners = append(owners, map[string]any{
			"path":       owner.Path,
			"owner_kind": owner.OwnerKind,
			"match_key":  owner.MatchKey,
		})
	}
	if actual := jsonReady(t, owners); !reflect.DeepEqual(actual, expected["owners"]) {
		t.Fatalf("unexpected method owners: %+v", actual)
	}
	texts := make([]string, 0, len(parsed.Analysis.Declarations))
	for _, declaration := range parsed.Analysis.Declarations {
		texts = append(texts, declaration.Text)
	}
	if actual := jsonReady(t, texts); !reflect.DeepEqual(actual, expected["declaration_texts"]) {
		t.Fatalf("unexpected method declaration text: %+v", actual)
	}

	merged := MergeGo(expected["merge_template"].(string), expected["merge_destination"].(string), dialect)
	if !merged.OK || merged.Output == nil || *merged.Output != expected["merge_output"].(string) {
		t.Fatalf("unexpected native Go method merge: %+v", merged)
	}
}

func TestGoParserPreservesGroupedDeclarationOwnerIdentityFixture(t *testing.T) {
	fixture := readFixture(t, "go", "slice-114-native", "gen-declaration-owner-identity.json")
	dialect := gomerge.GoDialect(fixture["dialect"].(string))
	expected := fixture["expected"].(map[string]any)
	parsed := ParseGo(fixture["source"].(string), dialect)
	if !parsed.OK || parsed.Analysis == nil {
		t.Fatalf("expected native Go parse success: %+v", parsed)
	}
	owners := make([]map[string]any, 0, len(parsed.Analysis.Owners))
	for _, owner := range parsed.Analysis.Owners {
		owners = append(owners, map[string]any{
			"path":       owner.Path,
			"owner_kind": owner.OwnerKind,
			"match_key":  owner.MatchKey,
		})
	}
	if actual := jsonReady(t, owners); !reflect.DeepEqual(actual, expected["owners"]) {
		t.Fatalf("unexpected grouped declaration owners: %+v", actual)
	}
	texts := make([]string, 0, len(parsed.Analysis.Declarations))
	for _, declaration := range parsed.Analysis.Declarations {
		texts = append(texts, declaration.Text)
	}
	if actual := jsonReady(t, texts); !reflect.DeepEqual(actual, expected["declaration_texts"]) {
		t.Fatalf("unexpected grouped declaration text: %+v", actual)
	}
	merged := MergeGo(expected["merge_template"].(string), expected["merge_destination"].(string), dialect)
	if !merged.OK || merged.Output == nil || *merged.Output != expected["merge_output"].(string) {
		t.Fatalf("unexpected native Go grouped declaration merge: %+v", merged)
	}
}

func TestGoParserFindsPackageClauseAfterPackageDocumentation(t *testing.T) {
	fixture := readFixture(t, "go", "slice-114-native", "package-doc-owner.json")
	parsed := ParseGo(fixture["source"].(string), gomerge.DialectGo)
	if !parsed.OK || parsed.Analysis == nil {
		t.Fatalf("expected native Go parse success: %+v", parsed)
	}
	owners := make([]map[string]any, 0, len(parsed.Analysis.Owners))
	for _, owner := range parsed.Analysis.Owners {
		owners = append(owners, map[string]any{
			"path":       owner.Path,
			"owner_kind": owner.OwnerKind,
			"match_key":  owner.MatchKey,
		})
	}
	if actual := jsonReady(t, owners); !reflect.DeepEqual(actual, fixture["expected"].(map[string]any)["owners"]) {
		t.Fatalf("unexpected package-document owners: %+v", actual)
	}
}

func TestGoParserNormalizesSyntheticPackageParseErrorOffsets(t *testing.T) {
	fixture := readFixture(t, "go", "slice-114-native", "parser-error-offset.json")
	parsed := ParseGo(fixture["source"].(string), gomerge.DialectGo)
	if parsed.OK || parsed.Analysis != nil {
		t.Fatalf("expected malformed Go source to fail closed: %+v", parsed)
	}
	expected := fixture["expected"].(map[string]any)
	if len(parsed.Diagnostics) != 1 || string(parsed.Diagnostics[0].Category) != expected["category"].(string) {
		t.Fatalf("unexpected native Go parse diagnostics: %+v", parsed.Diagnostics)
	}
	if !strings.HasPrefix(parsed.Diagnostics[0].Message, expected["message_prefix"].(string)) {
		t.Fatalf("synthetic package offset leaked into parse message: %q", parsed.Diagnostics[0].Message)
	}
}

func TestGoParserRejectsDuplicateOwnerIdentityFixture(t *testing.T) {
	fixture := readFixture(t, "go", "slice-114-native", "duplicate-owner-identity.json")
	parsed := ParseGo(fixture["source"].(string), gomerge.DialectGo)
	if parsed.OK || parsed.Analysis != nil {
		t.Fatalf("expected duplicate Go owner identities to fail closed: %+v", parsed)
	}
	expected := fixture["expected"].(map[string]any)
	if len(parsed.Diagnostics) != 1 || string(parsed.Diagnostics[0].Category) != expected["category"].(string) ||
		parsed.Diagnostics[0].Message != expected["message"].(string) {
		t.Fatalf("unexpected duplicate Go owner diagnostics: %+v", parsed.Diagnostics)
	}
}

func TestGoParserPreservesSameLineTrailingCommentOwnershipFixture(t *testing.T) {
	fixture := readFixture(t, "go", "slice-114-native", "trailing-comment-ownership.json")
	dialect := gomerge.GoDialect(fixture["dialect"].(string))
	expected := fixture["expected"].(map[string]any)
	parsed := ParseGo(fixture["source"].(string), dialect)
	if !parsed.OK || parsed.Analysis == nil {
		t.Fatalf("expected native Go parse success: %+v", parsed)
	}
	texts := make([]string, 0, len(parsed.Analysis.Declarations))
	for _, declaration := range parsed.Analysis.Declarations {
		texts = append(texts, declaration.Text)
	}
	if actual := jsonReady(t, texts); !reflect.DeepEqual(actual, expected["declaration_texts"]) {
		t.Fatalf("unexpected native trailing comment ownership: %+v", actual)
	}
	merged := MergeGo(expected["merge_template"].(string), expected["merge_destination"].(string), dialect)
	if !merged.OK || merged.Output == nil || *merged.Output != expected["merge_output"].(string) {
		t.Fatalf("native Go merge dropped an owned trailing comment: %+v", merged)
	}
}

func TestGoParserPreservesGroupedImportOwnerFixture(t *testing.T) {
	fixture := readFixture(t, "go", "slice-114-native", "grouped-import-owner.json")
	dialect := gomerge.GoDialect(fixture["dialect"].(string))
	expected := fixture["expected"].(map[string]any)
	parsed := ParseGo(fixture["source"].(string), dialect)
	if !parsed.OK || parsed.Analysis == nil {
		t.Fatalf("expected native Go parse success: %+v", parsed)
	}
	owners := make([]map[string]any, 0, len(parsed.Analysis.Owners))
	for _, owner := range parsed.Analysis.Owners {
		owners = append(owners, map[string]any{
			"path":       owner.Path,
			"owner_kind": owner.OwnerKind,
			"match_key":  owner.MatchKey,
		})
	}
	if actual := jsonReady(t, owners); !reflect.DeepEqual(actual, expected["owners"]) {
		t.Fatalf("unexpected grouped import owners: %+v", actual)
	}
	if len(parsed.Analysis.Imports) != 1 {
		t.Fatalf("expected one grouped import declaration: %+v", parsed.Analysis.Imports)
	}
	importOwner := parsed.Analysis.Imports[0]
	if importOwner.Path != expected["import_path"].(string) ||
		importOwner.MatchKey != expected["import_match_key"].(string) ||
		importOwner.Text != expected["import_text"].(string) {
		t.Fatalf("unexpected grouped import text or identity: %+v", importOwner)
	}
	reordered := ParseGo(fixture["reordered_source"].(string), dialect)
	if !reordered.OK || reordered.Analysis == nil {
		t.Fatalf("expected reordered native Go imports to parse: %+v", reordered)
	}
	matched := MatchGoOwners(*parsed.Analysis, *reordered.Analysis)
	matchedImport := false
	for _, ownerMatch := range matched.Matched {
		if ownerMatch.TemplatePath == expected["import_path"].(string) &&
			ownerMatch.DestinationPath == expected["import_path"].(string) {
			matchedImport = true
			break
		}
	}
	if !matchedImport {
		t.Fatalf("grouped import identity changed after spec reorder: %+v", matched)
	}
	merged := MergeGo(expected["merge_template"].(string), expected["merge_destination"].(string), dialect)
	if !merged.OK || merged.Output == nil || *merged.Output != expected["merge_output"].(string) {
		t.Fatalf("native Go merge split a grouped import owner: %q", *merged.Output)
	}
}

func TestGoParserRejectsMalformedSourceFixture(t *testing.T) {
	fixture := readFixture(t, "go", "slice-114-native", "malformed-source.json")
	parsed := ParseGo(fixture["source"].(string), gomerge.GoDialect(fixture["dialect"].(string)))
	expected := fixture["expected"].(map[string]any)
	if parsed.OK || parsed.Analysis != nil || len(parsed.Diagnostics) != 1 ||
		string(parsed.Diagnostics[0].Category) != expected["category"].(string) {
		t.Fatalf("unexpected malformed source result: %+v", parsed)
	}
}

func TestGoParserPreservesBOMSourceOwnershipFixture(t *testing.T) {
	fixture := readFixture(t, "go", "slice-114-native", "bom-source-ownership.json")
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		t.Run(testCase["name"].(string), func(t *testing.T) {
			parsed := ParseGo(testCase["source"].(string), gomerge.GoDialect(fixture["dialect"].(string)))
			if !parsed.OK || parsed.Analysis == nil {
				t.Fatalf("expected go/parser to accept BOM source: %+v", parsed)
			}
			expected := testCase["expected"].(map[string]any)
			foundDeclaration := false
			for _, declaration := range parsed.Analysis.Declarations {
				if declaration.Path == expected["declaration_path"].(string) {
					foundDeclaration = true
					if declaration.Text != expected["declaration_text"].(string) {
						t.Fatalf("unexpected BOM declaration text: %q", declaration.Text)
					}
				}
			}
			if !foundDeclaration {
				t.Fatalf("expected declaration owner %s: %+v", expected["declaration_path"], parsed.Analysis.Declarations)
			}
			if packagePath, ok := expected["package_owner_path"].(string); ok {
				foundPackage := false
				for _, owner := range parsed.Analysis.Owners {
					foundPackage = foundPackage || owner.Path == packagePath
				}
				if !foundPackage {
					t.Fatalf("expected package owner %s: %+v", packagePath, parsed.Analysis.Owners)
				}
			}
			if template, ok := testCase["merge_template"].(string); ok {
				merged := MergeGo(template, testCase["merge_destination"].(string), gomerge.DialectGo)
				actualOutput := "<nil>"
				if merged.Output != nil {
					actualOutput = *merged.Output
				}
				if !merged.OK || merged.Output == nil ||
					actualOutput != expected["merge_output"].(string) {
					t.Fatalf("unexpected BOM-preserving native merge: %+v output=%q", merged, actualOutput)
				}
			}
		})
	}
}

func TestGoParserFailsClosedOnRecoveredPartialASTsFixture(t *testing.T) {
	fixture := readFixture(t, "go", "slice-114-native", "parser-recovery-fail-closed.json")
	expected := fixture["expected"].(map[string]any)
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		t.Run(testCase["name"].(string), func(t *testing.T) {
			parsed := ParseGo(testCase["source"].(string), gomerge.GoDialect(fixture["dialect"].(string)))
			if parsed.OK || parsed.Analysis != nil || len(parsed.Diagnostics) != 1 ||
				string(parsed.Diagnostics[0].Category) != expected["category"].(string) {
				t.Fatalf("expected recovered syntax error to fail closed without analysis: %+v", parsed)
			}
		})
	}
}

func TestGoParserRejectsMalformedOneSidedMerge3InputsFixture(t *testing.T) {
	fixture := readFixture(t, "go", "slice-114-native", "merge3-one-sided-parser-failure.json")
	expected := fixture["expected"].(map[string]any)
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		t.Run(testCase["name"].(string), func(t *testing.T) {
			parsed := ParseGo(testCase["malformed_source"].(string), gomerge.DialectGo)
			if parsed.OK || parsed.Analysis != nil || len(parsed.Diagnostics) != 1 ||
				string(parsed.Diagnostics[0].Category) != expected["category"].(string) {
				t.Fatalf("expected malformed merge input to fail closed without analysis: %+v", parsed)
			}
		})
	}
}

func TestGoParserPreservesGenericTypeAliasOwnerFixture(t *testing.T) {
	fixture := readFixture(t, "go", "slice-114-native", "generic-type-alias-owner.json")
	expected := fixture["expected"].(map[string]any)
	parsed := ParseGo(fixture["source"].(string), gomerge.GoDialect(fixture["dialect"].(string)))
	if !parsed.OK || parsed.Analysis == nil {
		t.Fatalf("expected native Go parser to accept generic aliases: %+v", parsed)
	}
	owners := make([]map[string]any, 0, len(parsed.Analysis.Owners))
	for _, owner := range parsed.Analysis.Owners {
		owners = append(owners, map[string]any{
			"path":       owner.Path,
			"owner_kind": owner.OwnerKind,
			"match_key":  owner.MatchKey,
		})
	}
	if actual := jsonReady(t, owners); !reflect.DeepEqual(actual, expected["go_owners"]) {
		t.Fatalf("unexpected generic alias owners: %+v", actual)
	}
	paths := make([]string, 0, len(parsed.Analysis.Declarations))
	texts := make([]string, 0, len(parsed.Analysis.Declarations))
	for _, declaration := range parsed.Analysis.Declarations {
		paths = append(paths, declaration.Path)
		texts = append(texts, declaration.Text)
	}
	if actual := jsonReady(t, paths); !reflect.DeepEqual(actual, expected["declaration_paths"]) {
		t.Fatalf("unexpected generic alias declaration paths: %+v", actual)
	}
	if actual := jsonReady(t, texts); !reflect.DeepEqual(actual, expected["declaration_texts"]) {
		t.Fatalf("unexpected generic alias declaration text: %+v", actual)
	}

	grouped := ParseGo(fixture["grouped_source"].(string), gomerge.GoDialect(fixture["dialect"].(string)))
	if !grouped.OK || grouped.Analysis == nil {
		t.Fatalf("expected grouped generic alias to parse: %+v", grouped)
	}
	groupedOwners := make([]map[string]any, 0, len(grouped.Analysis.Owners))
	for _, owner := range grouped.Analysis.Owners {
		groupedOwners = append(groupedOwners, map[string]any{
			"path":       owner.Path,
			"owner_kind": owner.OwnerKind,
			"match_key":  owner.MatchKey,
		})
	}
	if actual := jsonReady(t, groupedOwners); !reflect.DeepEqual(actual, expected["grouped_go_owners"]) {
		t.Fatalf("unexpected grouped generic alias owners: %+v", actual)
	}
	if len(grouped.Analysis.Declarations) != 1 ||
		grouped.Analysis.Declarations[0].Text != expected["grouped_declaration_text"].(string) {
		t.Fatalf("unexpected grouped generic alias source text: %+v", grouped.Analysis.Declarations)
	}
}

func TestSharedFixtureGoProviderNamedSuitePlans(t *testing.T) {
	fixture := readFixture(t, "diagnostics", "slice-283-go-provider-named-suite-plans", "go-go-provider-named-suite-plans.json")
	source, err := json.Marshal(fixture["manifest"])
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}

	var manifest astmerge.ConformanceManifest
	if err := json.Unmarshal(source, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}

	expectedRaw := fixture["expected_entries"].(map[string]any)["go_parser"].([]any)
	expected := jsonReady(t, expectedRaw)

	plans := astmerge.PlanNamedConformanceSuites(manifest, map[string]astmerge.ConformanceFamilyPlanContext{
		"go": GoPlanContext(),
	})
	projected := jsonReady(t, plans)

	if !reflect.DeepEqual(projected, expected) {
		t.Fatalf("unexpected plans: %+v", projected)
	}
}

func TestSharedFixtureGoProviderManifestReport(t *testing.T) {
	fixture := readFixture(t, "diagnostics", "slice-284-go-provider-manifest-report", "go-go-provider-manifest-report.json")
	source, err := json.Marshal(fixture["manifest"])
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}

	var manifest astmerge.ConformanceManifest
	if err := json.Unmarshal(source, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}

	executionsSource, err := json.Marshal(fixture["executions"].(map[string]any)["go_parser"])
	if err != nil {
		t.Fatalf("marshal executions: %v", err)
	}

	var executions map[string]astmerge.ConformanceCaseExecution
	if err := json.Unmarshal(executionsSource, &executions); err != nil {
		t.Fatalf("decode executions: %v", err)
	}

	entries := astmerge.ReportPlannedNamedConformanceSuites(
		astmerge.PlanNamedConformanceSuites(manifest, map[string]astmerge.ConformanceFamilyPlanContext{
			"go": GoPlanContext(),
		}),
		func(run astmerge.ConformanceCaseRun) astmerge.ConformanceCaseExecution {
			key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
			if result, ok := executions[key]; ok {
				return result
			}
			return astmerge.ConformanceCaseExecution{Outcome: "failed", Messages: []string{"missing execution"}}
		},
	)

	report := astmerge.ReportNamedConformanceSuiteEnvelope(entries)
	expected := jsonReady(t, fixture["expected_reports"].(map[string]any)["go_parser"])
	actual := jsonReady(t, report)

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("unexpected report: %+v", actual)
	}
}

func TestSharedFixtureGoProviderRejectsUnsupportedBackendOverrides(t *testing.T) {
	expectedDiagnostics := []map[string]any{{
		"severity": "error",
		"category": "unsupported_feature",
		"message":  "Unsupported Go backend kreuzberg-language-pack.",
	}}

	parseResult := ParseGo("func Greet() string { return \"hi\" }\n", gomerge.DialectGo, "kreuzberg-language-pack")
	if parseResult.OK {
		t.Fatalf("expected parse failure: %+v", parseResult)
	}
	if actual := jsonReady(t, parseResult.Diagnostics); !reflect.DeepEqual(actual, jsonReady(t, expectedDiagnostics)) {
		t.Fatalf("unexpected parse diagnostics: %+v", actual)
	}

	mergeResult := MergeGo("func A() {}\n", "func B() {}\n", gomerge.DialectGo, "kreuzberg-language-pack")
	if mergeResult.OK {
		t.Fatalf("expected merge failure: %+v", mergeResult)
	}
	if actual := jsonReady(t, mergeResult.Diagnostics); !reflect.DeepEqual(actual, jsonReady(t, expectedDiagnostics)) {
		t.Fatalf("unexpected merge diagnostics: %+v", actual)
	}
}

func TestGoParserEditProjectionExecutionFixture(t *testing.T) {
	fixture := readFixture(t, "diagnostics", "slice-931-go-parser-edit-projection-execution", "edit-projection-execution.json")

	request := editProjectionExecutionRequestFromFixture(fixture["request"])
	result := ApplyEditProjection(request)
	expected := editProjectionExecutionResultFromFixture(fixture["expected_result"])
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("unexpected go/parser edit projection result:\n%#v", result)
	}
}

func TestGoParserInsertChildEditProjectionFixture(t *testing.T) {
	fixture := readFixture(t, "diagnostics", "slice-933-go-parser-insert-child-edit-projection", "insert-child-edit-projection.json")

	request := editProjectionExecutionRequestFromFixture(fixture["request"])
	result := ApplyEditProjection(request)
	expected := editProjectionExecutionResultFromFixture(fixture["expected_result"])
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("unexpected go/parser insert-child edit projection result:\n%#v", result)
	}
}

func TestGoParserDeleteNodeEditProjectionFixture(t *testing.T) {
	fixture := readFixture(t, "diagnostics", "slice-934-go-parser-delete-node-edit-projection", "delete-node-edit-projection.json")

	request := editProjectionExecutionRequestFromFixture(fixture["request"])
	result := ApplyEditProjection(request)
	expected := editProjectionExecutionResultFromFixture(fixture["expected_result"])
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("unexpected go/parser delete-node edit projection result:\n%#v", result)
	}
}

func editProjectionExecutionRequestFromFixture(value any) treehaver.EditProjectionExecutionRequest {
	fixture := value.(map[string]any)
	operations := []treehaver.EditProjectionOperationRequest{}
	for _, rawOperation := range fixture["operations"].([]any) {
		operationFixture := rawOperation.(map[string]any)
		operations = append(operations, treehaver.EditProjectionOperationRequest{
			Operation:         operationFixture["operation"].(string),
			TargetNodeID:      operationFixture["target_node_id"].(string),
			TargetNodePath:    operationFixture["target_node_path"].(string),
			ReplacementSource: operationFixture["replacement_source"].(string),
		})
	}
	backendRefFixture := fixture["backend_ref"].(map[string]any)
	return treehaver.EditProjectionExecutionRequest{
		ProviderID: fixture["provider_id"].(string),
		BackendRef: treehaver.BackendReference{
			ID:     backendRefFixture["id"].(string),
			Family: backendRefFixture["family"].(string),
		},
		Language:   fixture["language"].(string),
		Source:     fixture["source"].(string),
		Operations: operations,
	}
}

func editProjectionExecutionResultFromFixture(value any) treehaver.EditProjectionExecutionResult {
	fixture := value.(map[string]any)
	applied := []treehaver.AppliedEditProjectionOperation{}
	for _, rawOperation := range fixture["applied_operations"].([]any) {
		operationFixture := rawOperation.(map[string]any)
		applied = append(applied, treehaver.AppliedEditProjectionOperation{
			Operation:        operationFixture["operation"].(string),
			TargetNodeID:     operationFixture["target_node_id"].(string),
			CorrelationKey:   operationFixture["correlation_key"].(string),
			CorrelationValue: operationFixture["correlation_value"].(string),
		})
	}
	diagnostics := []treehaver.ProviderDiagnostic{}
	for _, rawDiagnostic := range fixture["diagnostics"].([]any) {
		diagnosticFixture := rawDiagnostic.(map[string]any)
		diagnostics = append(diagnostics, treehaver.ProviderDiagnostic{
			Severity: diagnosticFixture["severity"].(string),
			Category: diagnosticFixture["category"].(string),
			Code:     diagnosticFixture["code"].(string),
			Message:  diagnosticFixture["message"].(string),
			Path:     diagnosticFixture["path"].(string),
			Blocking: diagnosticFixture["blocking"].(bool),
		})
	}
	return treehaver.EditProjectionExecutionResult{
		OK:                fixture["ok"].(bool),
		Status:            fixture["status"].(string),
		Source:            fixture["source"].(string),
		AppliedOperations: applied,
		Diagnostics:       diagnostics,
	}
}

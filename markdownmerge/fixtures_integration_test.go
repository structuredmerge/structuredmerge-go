package markdownmerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
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

func jsonReadyMarkdown(t *testing.T, value any) any {
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

func parseProjectedChildReviewGroup(raw map[string]any) astmerge.ProjectedChildReviewGroup {
	return astmerge.ProjectedChildReviewGroup{
		DelegatedApplyGroup:         raw["delegated_apply_group"].(string),
		ParentOperationID:           raw["parent_operation_id"].(string),
		ChildOperationID:            raw["child_operation_id"].(string),
		DelegatedRuntimeSurfacePath: raw["delegated_runtime_surface_path"].(string),
		CaseIDs:                     parseStringSlice(raw["case_ids"].([]any)),
		DelegatedCaseIDs:            parseStringSlice(raw["delegated_case_ids"].([]any)),
	}
}

func parseReviewRequest(raw map[string]any) astmerge.ReviewRequest {
	request := astmerge.ReviewRequest{
		ID:           raw["id"].(string),
		Kind:         astmerge.ReviewRequestKind(raw["kind"].(string)),
		Family:       raw["family"].(string),
		Message:      raw["message"].(string),
		Blocking:     raw["blocking"].(bool),
		ActionOffers: []astmerge.ReviewActionOffer{},
	}
	if rawDelegatedGroup, ok := raw["delegated_group"]; ok {
		group := parseProjectedChildReviewGroup(rawDelegatedGroup.(map[string]any))
		request.DelegatedGroup = &group
	}
	if rawActionOffers, ok := raw["action_offers"]; ok {
		request.ActionOffers = make([]astmerge.ReviewActionOffer, 0, len(rawActionOffers.([]any)))
		for _, item := range rawActionOffers.([]any) {
			offer := item.(map[string]any)
			request.ActionOffers = append(request.ActionOffers, astmerge.ReviewActionOffer{
				Action:          astmerge.ReviewDecisionAction(offer["action"].(string)),
				RequiresContext: offer["requires_context"].(bool),
			})
		}
	}
	if rawDefaultAction, ok := raw["default_action"]; ok {
		request.DefaultAction = astmerge.ReviewDecisionAction(rawDefaultAction.(string))
	}
	return request
}

func parseReviewDecision(raw map[string]any) astmerge.ReviewDecision {
	return astmerge.ReviewDecision{
		RequestID: raw["request_id"].(string),
		Action:    astmerge.ReviewDecisionAction(raw["action"].(string)),
	}
}

func parseStringSlice(raw []any) []string {
	values := make([]string, 0, len(raw))
	for _, item := range raw {
		values = append(values, item.(string))
	}
	return values
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

func TestSharedFixtureMarkdownEmbeddedFamilies(t *testing.T) {
	fixture := readMarkdownFixture(t, "markdown", "slice-208-embedded-families", "code-fence-families.json")
	for _, backend := range []MarkdownBackend{BackendGoldmark, BackendKreuzberg} {
		result := ParseMarkdownWithBackend(fixture["source"].(string), DialectMarkdown, backend)
		if !result.OK || result.Analysis == nil {
			t.Fatalf("expected parse success for %s: %+v", backend, result)
		}
		if actual := jsonReadyMarkdown(t, MarkdownEmbeddedFamilies(*result.Analysis)); !reflect.DeepEqual(actual, fixture["expected"]) {
			t.Fatalf("unexpected embedded families for %s: %+v", backend, actual)
		}
	}
}

func TestSharedFixtureMarkdownDiscoveredSurfaces(t *testing.T) {
	fixture := readMarkdownFixture(t, "markdown", "slice-212-discovered-surfaces", "fenced-code-surfaces.json")
	result := ParseMarkdownWithBackend(fixture["source"].(string), DialectMarkdown, BackendGoldmark)
	if !result.OK || result.Analysis == nil {
		t.Fatalf("expected parse success: %+v", result)
	}

	if actual := jsonReadyMarkdown(t, MarkdownDiscoveredSurfaces(*result.Analysis)); !reflect.DeepEqual(actual, fixture["expected"]) {
		t.Fatalf("unexpected discovered surfaces: %+v", actual)
	}
}

func TestSharedFixtureMarkdownDelegatedChildOperations(t *testing.T) {
	fixture := readMarkdownFixture(t, "markdown", "slice-213-delegated-child-operations", "fenced-code-child-operations.json")
	result := ParseMarkdownWithBackend(fixture["source"].(string), DialectMarkdown, BackendGoldmark)
	if !result.OK || result.Analysis == nil {
		t.Fatalf("expected parse success: %+v", result)
	}

	if actual := jsonReadyMarkdown(t, MarkdownDelegatedChildOperations(*result.Analysis, fixture["parent_operation_id"].(string))); !reflect.DeepEqual(actual, fixture["expected"]) {
		t.Fatalf("unexpected delegated child operations: %+v", actual)
	}
}

func TestSharedFixtureMarkdownProjectedChildReviewGroups(t *testing.T) {
	fixture := readMarkdownFixture(t, "markdown", "slice-228-projected-child-review-groups", "fenced-code-review-groups.json")
	source, err := json.Marshal(fixture["cases"])
	if err != nil {
		t.Fatalf("marshal projected cases: %v", err)
	}
	var cases []astmerge.ProjectedChildReviewCase
	if err := json.Unmarshal(source, &cases); err != nil {
		t.Fatalf("unmarshal projected cases: %v", err)
	}
	if actual := jsonReadyMarkdown(t, astmerge.GroupProjectedChildReviewCases(cases)); !reflect.DeepEqual(actual, fixture["expected_groups"]) {
		t.Fatalf("unexpected projected child review groups: %+v", actual)
	}
}

func TestSharedFixtureMarkdownProjectedChildReviewGroupProgress(t *testing.T) {
	fixture := readMarkdownFixture(t, "markdown", "slice-231-projected-child-review-group-progress", "fenced-code-review-progress.json")
	groupSource, err := json.Marshal(fixture["groups"])
	if err != nil {
		t.Fatalf("marshal projected groups: %v", err)
	}
	var groups []astmerge.ProjectedChildReviewGroup
	if err := json.Unmarshal(groupSource, &groups); err != nil {
		t.Fatalf("unmarshal projected groups: %v", err)
	}
	resolvedSource, err := json.Marshal(fixture["resolved_case_ids"])
	if err != nil {
		t.Fatalf("marshal resolved case ids: %v", err)
	}
	var resolvedCaseIDs []string
	if err := json.Unmarshal(resolvedSource, &resolvedCaseIDs); err != nil {
		t.Fatalf("unmarshal resolved case ids: %v", err)
	}
	if actual := jsonReadyMarkdown(t, astmerge.SummarizeProjectedChildReviewGroupProgress(groups, resolvedCaseIDs)); !reflect.DeepEqual(actual, fixture["expected_progress"]) {
		t.Fatalf("unexpected projected child review group progress: %+v", actual)
	}
}

func TestSharedFixtureMarkdownProjectedChildReviewGroupsReadyForApply(t *testing.T) {
	fixture := readMarkdownFixture(t, "markdown", "slice-234-projected-child-review-groups-ready-for-apply", "fenced-code-ready-groups.json")
	groupSource, err := json.Marshal(fixture["groups"])
	if err != nil {
		t.Fatalf("marshal projected groups: %v", err)
	}
	var groups []astmerge.ProjectedChildReviewGroup
	if err := json.Unmarshal(groupSource, &groups); err != nil {
		t.Fatalf("unmarshal projected groups: %v", err)
	}
	resolvedSource, err := json.Marshal(fixture["resolved_case_ids"])
	if err != nil {
		t.Fatalf("marshal resolved case ids: %v", err)
	}
	var resolvedCaseIDs []string
	if err := json.Unmarshal(resolvedSource, &resolvedCaseIDs); err != nil {
		t.Fatalf("unmarshal resolved case ids: %v", err)
	}
	if actual := jsonReadyMarkdown(t, astmerge.SelectProjectedChildReviewGroupsReadyForApply(groups, resolvedCaseIDs)); !reflect.DeepEqual(actual, fixture["expected_ready_groups"]) {
		t.Fatalf("unexpected ready projected child review groups: %+v", actual)
	}
}

func TestSharedFixtureMarkdownDelegatedChildReviewTransport(t *testing.T) {
	fixture := readMarkdownFixture(t, "markdown", "slice-238-delegated-child-review-transport", "fenced-code-review-transport.json")
	group := parseProjectedChildReviewGroup(fixture["group"].(map[string]any))
	expectedRequest := parseReviewRequest(fixture["expected_request"].(map[string]any))

	if actual := astmerge.ProjectedChildGroupReviewRequest(group, fixture["family"].(string)); !reflect.DeepEqual(actual, expectedRequest) {
		t.Fatalf("unexpected delegated child review request: %+v", actual)
	}

	groupSource, err := json.Marshal(fixture["groups"])
	if err != nil {
		t.Fatalf("marshal projected groups: %v", err)
	}
	var groups []astmerge.ProjectedChildReviewGroup
	if err := json.Unmarshal(groupSource, &groups); err != nil {
		t.Fatalf("unmarshal projected groups: %v", err)
	}
	decisions := make([]astmerge.ReviewDecision, 0, len(fixture["decisions"].([]any)))
	for _, item := range fixture["decisions"].([]any) {
		decisions = append(decisions, parseReviewDecision(item.(map[string]any)))
	}
	if actual := jsonReadyMarkdown(t, astmerge.SelectProjectedChildReviewGroupsAcceptedForApply(groups, fixture["family"].(string), decisions)); !reflect.DeepEqual(actual, fixture["expected_accepted_groups"]) {
		t.Fatalf("unexpected accepted projected child review groups: %+v", actual)
	}
}

func TestSharedFixtureMarkdownDelegatedChildReviewState(t *testing.T) {
	fixture := readMarkdownFixture(t, "markdown", "slice-241-delegated-child-review-state", "fenced-code-review-state.json")
	groupSource, err := json.Marshal(fixture["groups"])
	if err != nil {
		t.Fatalf("marshal projected groups: %v", err)
	}
	var groups []astmerge.ProjectedChildReviewGroup
	if err := json.Unmarshal(groupSource, &groups); err != nil {
		t.Fatalf("unmarshal projected groups: %v", err)
	}
	decisions := make([]astmerge.ReviewDecision, 0, len(fixture["decisions"].([]any)))
	for _, item := range fixture["decisions"].([]any) {
		decisions = append(decisions, parseReviewDecision(item.(map[string]any)))
	}
	if actual := jsonReadyMarkdown(t, astmerge.ReviewProjectedChildGroups(groups, fixture["family"].(string), decisions)); !reflect.DeepEqual(actual, fixture["expected_state"]) {
		t.Fatalf("unexpected delegated child review state: %+v", actual)
	}
}

func TestSharedFixtureMarkdownDelegatedChildApplyPlan(t *testing.T) {
	fixture := readMarkdownFixture(t, "markdown", "slice-244-delegated-child-apply-plan", "fenced-code-apply-plan.json")
	stateSource, err := json.Marshal(fixture["review_state"])
	if err != nil {
		t.Fatalf("marshal delegated child review state: %v", err)
	}
	var state astmerge.DelegatedChildGroupReviewState
	if err := json.Unmarshal(stateSource, &state); err != nil {
		t.Fatalf("unmarshal delegated child review state: %v", err)
	}
	if actual := jsonReadyMarkdown(t, astmerge.DelegatedChildApplyPlanForState(state, fixture["family"].(string))); !reflect.DeepEqual(actual, fixture["expected_plan"]) {
		t.Fatalf("unexpected delegated child apply plan: %+v", actual)
	}
}

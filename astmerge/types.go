package astmerge

import "slices"

type DiagnosticSeverity string

const (
	SeverityInfo    DiagnosticSeverity = "info"
	SeverityWarning DiagnosticSeverity = "warning"
	SeverityError   DiagnosticSeverity = "error"
)

type DiagnosticCategory string

const (
	CategoryParseError            DiagnosticCategory = "parse_error"
	CategoryDestinationParseError DiagnosticCategory = "destination_parse_error"
	CategoryUnsupportedFeature    DiagnosticCategory = "unsupported_feature"
	CategoryFallbackApplied       DiagnosticCategory = "fallback_applied"
	CategoryAmbiguity             DiagnosticCategory = "ambiguity"
	CategoryAssumedDefault        DiagnosticCategory = "assumed_default"
	CategoryConfigurationError    DiagnosticCategory = "configuration_error"
)

type Diagnostic struct {
	Severity DiagnosticSeverity
	Category DiagnosticCategory
	Message  string
	Path     string
}

type ParseResult[T any] struct {
	OK          bool
	Diagnostics []Diagnostic
	Analysis    *T
	Policies    []PolicyReference
}

type MergeResult[T any] struct {
	OK          bool
	Diagnostics []Diagnostic
	Output      *T
	Policies    []PolicyReference
}

type PolicySurface string

const (
	PolicySurfaceFallback PolicySurface = "fallback"
	PolicySurfaceArray    PolicySurface = "array"
)

type PolicyReference struct {
	Surface PolicySurface
	Name    string
}

type FamilyFeatureProfile struct {
	Family            string
	SupportedDialects []string
	SupportedPolicies []PolicyReference
}

type ConformanceOutcome string

const (
	ConformancePassed  ConformanceOutcome = "passed"
	ConformanceFailed  ConformanceOutcome = "failed"
	ConformanceSkipped ConformanceOutcome = "skipped"
)

type ConformanceCaseRef struct {
	Family string
	Role   string
	Case   string
}

type ConformanceCaseResult struct {
	Ref      ConformanceCaseRef
	Outcome  ConformanceOutcome
	Messages []string
}

type ConformanceCaseRequirements struct {
	Dialect  string            `json:"dialect,omitempty"`
	Policies []PolicyReference `json:"policies,omitempty"`
}

type ConformanceSelectionStatus string

const (
	ConformanceSelected         ConformanceSelectionStatus = "selected"
	ConformanceSelectionSkipped ConformanceSelectionStatus = "skipped"
)

type ConformanceCaseSelection struct {
	Ref      ConformanceCaseRef         `json:"ref"`
	Status   ConformanceSelectionStatus `json:"status"`
	Messages []string                   `json:"messages"`
}

type ConformanceManifestEntry struct {
	Role         string                       `json:"role"`
	Path         []string                     `json:"path"`
	Requirements *ConformanceCaseRequirements `json:"requirements,omitempty"`
}

type ConformanceFamilyFeatureProfileEntry struct {
	Family string   `json:"family"`
	Role   string   `json:"role"`
	Path   []string `json:"path"`
}

type ConformanceManifest struct {
	FamilyFeatureProfiles []ConformanceFamilyFeatureProfileEntry `json:"family_feature_profiles"`
	Suites                map[string]ConformanceSuiteDefinition  `json:"suites,omitempty"`
	Families              map[string][]ConformanceManifestEntry  `json:"families"`
}

type ConformanceSuiteDefinition struct {
	Family string   `json:"family"`
	Roles  []string `json:"roles"`
}

type NamedConformanceSuiteReport struct {
	Suite  string                 `json:"suite"`
	Report ConformanceSuiteReport `json:"report"`
}

type ConformanceFamilyPlanContext struct {
	FamilyProfile  FamilyFeatureProfile           `json:"family_profile"`
	FeatureProfile *ConformanceFeatureProfileView `json:"feature_profile,omitempty"`
}

type NamedConformanceSuitePlan struct {
	Suite string               `json:"suite"`
	Plan  ConformanceSuitePlan `json:"plan"`
}

type NamedConformanceSuiteResults struct {
	Suite   string                  `json:"suite"`
	Results []ConformanceCaseResult `json:"results"`
}

type NamedConformanceSuiteReportEnvelope struct {
	Entries []NamedConformanceSuiteReport `json:"entries"`
	Summary ConformanceSuiteSummary       `json:"summary"`
}

type ConformanceManifestPlanningOptions struct {
	Contexts                map[string]ConformanceFamilyPlanContext `json:"contexts,omitempty"`
	FamilyProfiles          map[string]FamilyFeatureProfile         `json:"family_profiles,omitempty"`
	RequireExplicitContexts bool                                    `json:"require_explicit_contexts,omitempty"`
}

type ConformanceManifestReport struct {
	Report      NamedConformanceSuiteReportEnvelope `json:"report"`
	Diagnostics []Diagnostic                        `json:"diagnostics"`
}

type ReviewRequestKind string

const (
	ReviewRequestFamilyContext ReviewRequestKind = "family_context"
)

type ReviewDecisionAction string

const (
	ReviewDecisionAcceptDefaultContext ReviewDecisionAction = "accept_default_context"
)

type ReviewRequest struct {
	ID               string                 `json:"id"`
	Kind             ReviewRequestKind      `json:"kind"`
	Family           string                 `json:"family"`
	Message          string                 `json:"message"`
	Blocking         bool                   `json:"blocking"`
	AvailableActions []ReviewDecisionAction `json:"available_actions"`
	DefaultAction    ReviewDecisionAction   `json:"default_action,omitempty"`
}

type ReviewDecision struct {
	RequestID string               `json:"request_id"`
	Action    ReviewDecisionAction `json:"action"`
}

type ReviewHostHints struct {
	Interactive             bool `json:"interactive"`
	RequireExplicitContexts bool `json:"require_explicit_contexts"`
}

type ReviewReplayContext struct {
	Surface                 string   `json:"surface"`
	Families                []string `json:"families"`
	RequireExplicitContexts bool     `json:"require_explicit_contexts"`
}

type ConformanceManifestReviewOptions struct {
	Contexts                map[string]ConformanceFamilyPlanContext `json:"contexts,omitempty"`
	FamilyProfiles          map[string]FamilyFeatureProfile         `json:"family_profiles,omitempty"`
	RequireExplicitContexts bool                                    `json:"require_explicit_contexts,omitempty"`
	ReviewDecisions         []ReviewDecision                        `json:"review_decisions,omitempty"`
	Interactive             bool                                    `json:"interactive,omitempty"`
}

type ConformanceManifestReviewState struct {
	Report           NamedConformanceSuiteReportEnvelope `json:"report"`
	Diagnostics      []Diagnostic                        `json:"diagnostics"`
	Requests         []ReviewRequest                     `json:"requests"`
	AppliedDecisions []ReviewDecision                    `json:"applied_decisions"`
	HostHints        ReviewHostHints                     `json:"host_hints"`
	ReplayContext    ReviewReplayContext                 `json:"replay_context"`
}

type ConformanceManifestPlan struct {
	Entries     []NamedConformanceSuitePlan `json:"entries"`
	Diagnostics []Diagnostic                `json:"diagnostics"`
}

type ConformanceSuiteSummary struct {
	Total   int `json:"total"`
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

type ConformanceSuiteReport struct {
	Results []ConformanceCaseResult `json:"results"`
	Summary ConformanceSuiteSummary `json:"summary"`
}

type ConformanceSuitePlanEntry struct {
	Ref  ConformanceCaseRef `json:"ref"`
	Path []string           `json:"path"`
	Run  ConformanceCaseRun `json:"run"`
}

type ConformanceSuitePlan struct {
	Family       string                      `json:"family"`
	Entries      []ConformanceSuitePlanEntry `json:"entries"`
	MissingRoles []string                    `json:"missing_roles"`
}

type ConformanceFeatureProfileView struct {
	Backend           string
	SupportsDialects  bool
	SupportedPolicies []PolicyReference
}

type ConformanceCaseRun struct {
	Ref            ConformanceCaseRef             `json:"ref"`
	Requirements   ConformanceCaseRequirements    `json:"requirements"`
	FamilyProfile  FamilyFeatureProfile           `json:"family_profile"`
	FeatureProfile *ConformanceFeatureProfileView `json:"feature_profile,omitempty"`
}

type ConformanceCaseExecution struct {
	Outcome  ConformanceOutcome `json:"outcome"`
	Messages []string           `json:"messages"`
}

func includesPolicy(supportedPolicies []PolicyReference, policy PolicyReference) bool {
	for _, supportedPolicy := range supportedPolicies {
		if supportedPolicy == policy {
			return true
		}
	}

	return false
}

func isDefaultDialect(familyProfile FamilyFeatureProfile, dialect string) bool {
	return dialect == familyProfile.Family
}

func ConformanceFamilyEntries(manifest ConformanceManifest, family string) []ConformanceManifestEntry {
	if entries, ok := manifest.Families[family]; ok {
		return entries
	}

	return []ConformanceManifestEntry{}
}

func ConformanceFixturePath(manifest ConformanceManifest, family string, role string) []string {
	for _, entry := range ConformanceFamilyEntries(manifest, family) {
		if entry.Role == role {
			return entry.Path
		}
	}

	return nil
}

func ConformanceFamilyFeatureProfilePath(manifest ConformanceManifest, family string) []string {
	for _, entry := range manifest.FamilyFeatureProfiles {
		if entry.Family == family {
			return entry.Path
		}
	}

	return nil
}

func ConformanceSuiteDefinitionByName(
	manifest ConformanceManifest,
	suiteName string,
) *ConformanceSuiteDefinition {
	if manifest.Suites == nil {
		return nil
	}

	if definition, ok := manifest.Suites[suiteName]; ok {
		return &definition
	}

	return nil
}

func ConformanceSuiteNames(manifest ConformanceManifest) []string {
	names := make([]string, 0, len(manifest.Suites))
	for name := range manifest.Suites {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func DefaultConformanceFamilyContext(
	familyProfile FamilyFeatureProfile,
) ConformanceFamilyPlanContext {
	return ConformanceFamilyPlanContext{
		FamilyProfile: familyProfile,
	}
}

func ReviewRequestIDForFamilyContext(family string) string {
	return "family_context:" + family
}

func ConformanceReviewHostHints(options ConformanceManifestReviewOptions) ReviewHostHints {
	return ReviewHostHints{
		Interactive:             options.Interactive,
		RequireExplicitContexts: options.RequireExplicitContexts,
	}
}

func ConformanceManifestReplayContext(
	manifest ConformanceManifest,
	options ConformanceManifestReviewOptions,
) ReviewReplayContext {
	families := make([]string, 0, len(manifest.Suites))
	seen := make(map[string]bool)
	for _, suiteName := range ConformanceSuiteNames(manifest) {
		definition := ConformanceSuiteDefinitionByName(manifest, suiteName)
		if definition == nil || seen[definition.Family] {
			continue
		}
		seen[definition.Family] = true
		families = append(families, definition.Family)
	}

	return ReviewReplayContext{
		Surface:                 "conformance_manifest",
		Families:                families,
		RequireExplicitContexts: options.RequireExplicitContexts,
	}
}

func ResolveConformanceFamilyContext(
	family string,
	options ConformanceManifestPlanningOptions,
) (*ConformanceFamilyPlanContext, []Diagnostic) {
	if options.Contexts != nil {
		if context, ok := options.Contexts[family]; ok {
			return &context, nil
		}
	}

	if options.RequireExplicitContexts {
		return nil, []Diagnostic{{
			Severity: SeverityError,
			Category: CategoryConfigurationError,
			Message:  "missing explicit family context for " + family + ".",
		}}
	}

	if options.FamilyProfiles != nil {
		if familyProfile, ok := options.FamilyProfiles[family]; ok {
			context := DefaultConformanceFamilyContext(familyProfile)
			return &context, []Diagnostic{{
				Severity: SeverityWarning,
				Category: CategoryAssumedDefault,
				Message:  "using default family context for " + family + ".",
			}}
		}
	}

	return nil, []Diagnostic{{
		Severity: SeverityError,
		Category: CategoryConfigurationError,
		Message:  "missing family context for " + family + " and no default family profile is available.",
	}}
}

func reviewDecisionForFamilyContext(
	family string,
	options ConformanceManifestReviewOptions,
) *ReviewDecision {
	requestID := ReviewRequestIDForFamilyContext(family)
	for _, decision := range options.ReviewDecisions {
		if decision.RequestID == requestID && decision.Action == ReviewDecisionAcceptDefaultContext {
			copyDecision := decision
			return &copyDecision
		}
	}

	return nil
}

func ReviewConformanceFamilyContext(
	family string,
	options ConformanceManifestReviewOptions,
) (*ConformanceFamilyPlanContext, []Diagnostic, []ReviewRequest, []ReviewDecision) {
	if context, ok := options.Contexts[family]; ok {
		copyContext := context
		return &copyContext, nil, nil, nil
	}

	if !options.RequireExplicitContexts {
		planningOptions := ConformanceManifestPlanningOptions{
			Contexts:                options.Contexts,
			FamilyProfiles:          options.FamilyProfiles,
			RequireExplicitContexts: false,
		}
		context, diagnostics := ResolveConformanceFamilyContext(family, planningOptions)
		return context, diagnostics, nil, nil
	}

	familyProfile, ok := options.FamilyProfiles[family]
	if !ok {
		return nil, []Diagnostic{{
			Severity: SeverityError,
			Category: CategoryConfigurationError,
			Message:  "missing family context for " + family + " and no default family profile is available.",
		}}, nil, nil
	}

	if decision := reviewDecisionForFamilyContext(family, options); decision != nil {
		return &ConformanceFamilyPlanContext{
				FamilyProfile: familyProfile,
			}, []Diagnostic{{
				Severity: SeverityWarning,
				Category: CategoryAssumedDefault,
				Message:  "using default family context for " + family + ".",
			}}, nil, []ReviewDecision{*decision}
	}

	return nil, []Diagnostic{{
			Severity: SeverityError,
			Category: CategoryConfigurationError,
			Message:  "missing explicit family context for " + family + ".",
		}}, []ReviewRequest{{
			ID:               ReviewRequestIDForFamilyContext(family),
			Kind:             ReviewRequestFamilyContext,
			Family:           family,
			Message:          "explicit family context is required for " + family + "; a synthesized default may be accepted by review.",
			Blocking:         true,
			AvailableActions: []ReviewDecisionAction{ReviewDecisionAcceptDefaultContext},
			DefaultAction:    ReviewDecisionAcceptDefaultContext,
		}}, nil
}

func SummarizeConformanceResults(results []ConformanceCaseResult) ConformanceSuiteSummary {
	summary := ConformanceSuiteSummary{}
	for _, result := range results {
		summary.Total++
		switch result.Outcome {
		case ConformancePassed:
			summary.Passed++
		case ConformanceFailed:
			summary.Failed++
		case ConformanceSkipped:
			summary.Skipped++
		}
	}

	return summary
}

func SelectConformanceCase(
	ref ConformanceCaseRef,
	requirements ConformanceCaseRequirements,
	familyProfile FamilyFeatureProfile,
	featureProfile *ConformanceFeatureProfileView,
) ConformanceCaseSelection {
	messages := []string{}

	if requirements.Dialect != "" {
		if !slices.Contains(familyProfile.SupportedDialects, requirements.Dialect) {
			messages = append(messages, "family "+familyProfile.Family+" does not support dialect "+requirements.Dialect+".")
		} else if featureProfile != nil && !featureProfile.SupportsDialects && !isDefaultDialect(familyProfile, requirements.Dialect) {
			messages = append(
				messages,
				"backend "+featureProfile.Backend+" does not support dialect "+requirements.Dialect+" for family "+familyProfile.Family+".",
			)
		}
	}

	for _, policy := range requirements.Policies {
		if !includesPolicy(familyProfile.SupportedPolicies, policy) {
			messages = append(messages, "family "+familyProfile.Family+" does not support policy "+policy.Name+".")
			continue
		}

		if featureProfile != nil && !includesPolicy(featureProfile.SupportedPolicies, policy) {
			messages = append(messages, "backend "+featureProfile.Backend+" does not support policy "+policy.Name+".")
		}
	}

	status := ConformanceSelected
	if len(messages) > 0 {
		status = ConformanceSelectionSkipped
	}

	return ConformanceCaseSelection{
		Ref:      ref,
		Status:   status,
		Messages: messages,
	}
}

func RunConformanceCase(
	run ConformanceCaseRun,
	execute func(ConformanceCaseRun) ConformanceCaseExecution,
) ConformanceCaseResult {
	selection := SelectConformanceCase(run.Ref, run.Requirements, run.FamilyProfile, run.FeatureProfile)
	if selection.Status == ConformanceSelectionSkipped {
		return ConformanceCaseResult{
			Ref:      run.Ref,
			Outcome:  ConformanceSkipped,
			Messages: selection.Messages,
		}
	}

	execution := execute(run)
	return ConformanceCaseResult{
		Ref:      run.Ref,
		Outcome:  execution.Outcome,
		Messages: execution.Messages,
	}
}

func RunConformanceSuite(
	runs []ConformanceCaseRun,
	execute func(ConformanceCaseRun) ConformanceCaseExecution,
) []ConformanceCaseResult {
	results := make([]ConformanceCaseResult, 0, len(runs))
	for _, run := range runs {
		results = append(results, RunConformanceCase(run, execute))
	}

	return results
}

func RunPlannedConformanceSuite(
	plan ConformanceSuitePlan,
	execute func(ConformanceCaseRun) ConformanceCaseExecution,
) []ConformanceCaseResult {
	results := make([]ConformanceCaseResult, 0, len(plan.Entries))
	for _, entry := range plan.Entries {
		results = append(results, RunConformanceCase(entry.Run, execute))
	}

	return results
}

func RunNamedConformanceSuite(
	manifest ConformanceManifest,
	suiteName string,
	familyProfile FamilyFeatureProfile,
	execute func(ConformanceCaseRun) ConformanceCaseExecution,
	featureProfile *ConformanceFeatureProfileView,
) []ConformanceCaseResult {
	plan := PlanNamedConformanceSuite(
		manifest,
		suiteName,
		familyProfile,
		featureProfile,
	)
	if plan == nil {
		return nil
	}

	return RunPlannedConformanceSuite(*plan, execute)
}

func RunNamedConformanceSuiteEntry(
	manifest ConformanceManifest,
	suiteName string,
	familyProfile FamilyFeatureProfile,
	execute func(ConformanceCaseRun) ConformanceCaseExecution,
	featureProfile *ConformanceFeatureProfileView,
) *NamedConformanceSuiteResults {
	results := RunNamedConformanceSuite(
		manifest,
		suiteName,
		familyProfile,
		execute,
		featureProfile,
	)
	if results == nil {
		return nil
	}

	return &NamedConformanceSuiteResults{
		Suite:   suiteName,
		Results: results,
	}
}

func RunPlannedNamedConformanceSuites(
	entries []NamedConformanceSuitePlan,
	execute func(ConformanceCaseRun) ConformanceCaseExecution,
) []NamedConformanceSuiteResults {
	results := make([]NamedConformanceSuiteResults, 0, len(entries))
	for _, entry := range entries {
		results = append(results, NamedConformanceSuiteResults{
			Suite:   entry.Suite,
			Results: RunPlannedConformanceSuite(entry.Plan, execute),
		})
	}

	return results
}

func ReportPlannedConformanceSuite(
	plan ConformanceSuitePlan,
	execute func(ConformanceCaseRun) ConformanceCaseExecution,
) ConformanceSuiteReport {
	return ReportConformanceSuite(RunPlannedConformanceSuite(plan, execute))
}

func ReportNamedConformanceSuite(
	manifest ConformanceManifest,
	suiteName string,
	familyProfile FamilyFeatureProfile,
	execute func(ConformanceCaseRun) ConformanceCaseExecution,
	featureProfile *ConformanceFeatureProfileView,
) *ConformanceSuiteReport {
	plan := PlanNamedConformanceSuite(
		manifest,
		suiteName,
		familyProfile,
		featureProfile,
	)
	if plan == nil {
		return nil
	}

	report := ReportPlannedConformanceSuite(*plan, execute)
	return &report
}

func ReportNamedConformanceSuiteEntry(
	manifest ConformanceManifest,
	suiteName string,
	familyProfile FamilyFeatureProfile,
	execute func(ConformanceCaseRun) ConformanceCaseExecution,
	featureProfile *ConformanceFeatureProfileView,
) *NamedConformanceSuiteReport {
	report := ReportNamedConformanceSuite(
		manifest,
		suiteName,
		familyProfile,
		execute,
		featureProfile,
	)
	if report == nil {
		return nil
	}

	return &NamedConformanceSuiteReport{
		Suite:  suiteName,
		Report: *report,
	}
}

func ReportPlannedNamedConformanceSuites(
	entries []NamedConformanceSuitePlan,
	execute func(ConformanceCaseRun) ConformanceCaseExecution,
) []NamedConformanceSuiteReport {
	reports := make([]NamedConformanceSuiteReport, 0, len(entries))
	for _, entry := range entries {
		reports = append(reports, NamedConformanceSuiteReport{
			Suite:  entry.Suite,
			Report: ReportPlannedConformanceSuite(entry.Plan, execute),
		})
	}

	return reports
}

func SummarizeNamedConformanceSuiteReports(
	entries []NamedConformanceSuiteReport,
) ConformanceSuiteSummary {
	summary := ConformanceSuiteSummary{}
	for _, entry := range entries {
		summary.Total += entry.Report.Summary.Total
		summary.Passed += entry.Report.Summary.Passed
		summary.Failed += entry.Report.Summary.Failed
		summary.Skipped += entry.Report.Summary.Skipped
	}

	return summary
}

func ReportNamedConformanceSuiteEnvelope(
	entries []NamedConformanceSuiteReport,
) NamedConformanceSuiteReportEnvelope {
	return NamedConformanceSuiteReportEnvelope{
		Entries: entries,
		Summary: SummarizeNamedConformanceSuiteReports(entries),
	}
}

func ReportNamedConformanceSuiteManifest(
	manifest ConformanceManifest,
	contexts map[string]ConformanceFamilyPlanContext,
	execute func(ConformanceCaseRun) ConformanceCaseExecution,
) NamedConformanceSuiteReportEnvelope {
	return ReportNamedConformanceSuiteEnvelope(
		ReportPlannedNamedConformanceSuites(
			PlanNamedConformanceSuites(manifest, contexts),
			execute,
		),
	)
}

func ReportConformanceManifest(
	manifest ConformanceManifest,
	options ConformanceManifestPlanningOptions,
	execute func(ConformanceCaseRun) ConformanceCaseExecution,
) ConformanceManifestReport {
	planned := PlanNamedConformanceSuitesWithDiagnostics(manifest, options)
	return ConformanceManifestReport{
		Report: ReportNamedConformanceSuiteEnvelope(
			ReportPlannedNamedConformanceSuites(planned.Entries, execute),
		),
		Diagnostics: planned.Diagnostics,
	}
}

func ReviewConformanceManifest(
	manifest ConformanceManifest,
	options ConformanceManifestReviewOptions,
	execute func(ConformanceCaseRun) ConformanceCaseExecution,
) ConformanceManifestReviewState {
	entries := make([]NamedConformanceSuitePlan, 0, len(manifest.Suites))
	diagnostics := make([]Diagnostic, 0)
	requests := make([]ReviewRequest, 0)
	appliedDecisions := make([]ReviewDecision, 0)
	resolvedContexts := make(map[string]*ConformanceFamilyPlanContext)
	resolvedFamilies := make(map[string]bool)

	for _, suiteName := range ConformanceSuiteNames(manifest) {
		definition := ConformanceSuiteDefinitionByName(manifest, suiteName)
		if definition == nil {
			continue
		}

		var context *ConformanceFamilyPlanContext
		if resolvedFamilies[definition.Family] {
			context = resolvedContexts[definition.Family]
		} else {
			var resolvedDiagnostics []Diagnostic
			var resolvedRequests []ReviewRequest
			var resolvedDecisions []ReviewDecision
			context, resolvedDiagnostics, resolvedRequests, resolvedDecisions = ReviewConformanceFamilyContext(definition.Family, options)
			diagnostics = append(diagnostics, resolvedDiagnostics...)
			requests = append(requests, resolvedRequests...)
			appliedDecisions = append(appliedDecisions, resolvedDecisions...)
			resolvedFamilies[definition.Family] = true
			resolvedContexts[definition.Family] = context
		}
		if context == nil {
			continue
		}

		entry := PlanNamedConformanceSuiteEntry(manifest, suiteName, *context)
		if entry == nil {
			continue
		}

		if len(entry.Plan.MissingRoles) > 0 {
			diagnostics = append(diagnostics, Diagnostic{
				Severity: SeverityError,
				Category: CategoryConfigurationError,
				Message:  "suite " + suiteName + " declares missing roles: " + joinComma(entry.Plan.MissingRoles) + ".",
			})
			continue
		}

		entries = append(entries, *entry)
	}

	return ConformanceManifestReviewState{
		Report:           ReportNamedConformanceSuiteEnvelope(ReportPlannedNamedConformanceSuites(entries, execute)),
		Diagnostics:      diagnostics,
		Requests:         requests,
		AppliedDecisions: appliedDecisions,
		HostHints:        ConformanceReviewHostHints(options),
		ReplayContext:    ConformanceManifestReplayContext(manifest, options),
	}
}

func ReportConformanceSuite(results []ConformanceCaseResult) ConformanceSuiteReport {
	return ConformanceSuiteReport{
		Results: results,
		Summary: SummarizeConformanceResults(results),
	}
}

func PlanConformanceSuite(
	manifest ConformanceManifest,
	family string,
	roles []string,
	familyProfile FamilyFeatureProfile,
	featureProfile *ConformanceFeatureProfileView,
) ConformanceSuitePlan {
	entries := make([]ConformanceSuitePlanEntry, 0, len(roles))
	missingRoles := make([]string, 0)

	for _, role := range roles {
		var entry *ConformanceManifestEntry
		for _, candidate := range ConformanceFamilyEntries(manifest, family) {
			if candidate.Role == role {
				entry = &candidate
				break
			}
		}
		if entry == nil {
			missingRoles = append(missingRoles, role)
			continue
		}

		ref := ConformanceCaseRef{
			Family: family,
			Role:   role,
			Case:   role,
		}

		entries = append(entries, ConformanceSuitePlanEntry{
			Ref:  ref,
			Path: slices.Clone(entry.Path),
			Run: ConformanceCaseRun{
				Ref:            ref,
				Requirements:   derefRequirements(entry.Requirements),
				FamilyProfile:  familyProfile,
				FeatureProfile: featureProfile,
			},
		})
	}

	return ConformanceSuitePlan{
		Family:       family,
		Entries:      entries,
		MissingRoles: missingRoles,
	}
}

func PlanNamedConformanceSuite(
	manifest ConformanceManifest,
	suiteName string,
	familyProfile FamilyFeatureProfile,
	featureProfile *ConformanceFeatureProfileView,
) *ConformanceSuitePlan {
	definition := ConformanceSuiteDefinitionByName(manifest, suiteName)
	if definition == nil {
		return nil
	}

	plan := PlanConformanceSuite(
		manifest,
		definition.Family,
		definition.Roles,
		familyProfile,
		featureProfile,
	)
	return &plan
}

func PlanNamedConformanceSuiteEntry(
	manifest ConformanceManifest,
	suiteName string,
	context ConformanceFamilyPlanContext,
) *NamedConformanceSuitePlan {
	plan := PlanNamedConformanceSuite(
		manifest,
		suiteName,
		context.FamilyProfile,
		context.FeatureProfile,
	)
	if plan == nil {
		return nil
	}

	return &NamedConformanceSuitePlan{
		Suite: suiteName,
		Plan:  *plan,
	}
}

func PlanNamedConformanceSuites(
	manifest ConformanceManifest,
	contexts map[string]ConformanceFamilyPlanContext,
) []NamedConformanceSuitePlan {
	entries := make([]NamedConformanceSuitePlan, 0, len(manifest.Suites))
	for _, suiteName := range ConformanceSuiteNames(manifest) {
		definition := ConformanceSuiteDefinitionByName(manifest, suiteName)
		if definition == nil {
			continue
		}

		context, ok := contexts[definition.Family]
		if !ok {
			continue
		}

		entry := PlanNamedConformanceSuiteEntry(manifest, suiteName, context)
		if entry != nil {
			entries = append(entries, *entry)
		}
	}

	return entries
}

func PlanNamedConformanceSuitesWithDiagnostics(
	manifest ConformanceManifest,
	options ConformanceManifestPlanningOptions,
) ConformanceManifestPlan {
	entries := make([]NamedConformanceSuitePlan, 0, len(manifest.Suites))
	diagnostics := make([]Diagnostic, 0)
	resolvedContexts := make(map[string]*ConformanceFamilyPlanContext)
	resolvedFamilies := make(map[string]bool)

	for _, suiteName := range ConformanceSuiteNames(manifest) {
		definition := ConformanceSuiteDefinitionByName(manifest, suiteName)
		if definition == nil {
			continue
		}

		var context *ConformanceFamilyPlanContext
		if resolvedFamilies[definition.Family] {
			context = resolvedContexts[definition.Family]
		} else {
			context, diagnostics = func() (*ConformanceFamilyPlanContext, []Diagnostic) {
				resolved, resolvedDiagnostics := ResolveConformanceFamilyContext(definition.Family, options)
				return resolved, append(diagnostics, resolvedDiagnostics...)
			}()
			resolvedFamilies[definition.Family] = true
			resolvedContexts[definition.Family] = context
		}
		if context == nil {
			continue
		}

		entry := PlanNamedConformanceSuiteEntry(manifest, suiteName, *context)
		if entry == nil {
			continue
		}

		if len(entry.Plan.MissingRoles) > 0 {
			diagnostics = append(diagnostics, Diagnostic{
				Severity: SeverityError,
				Category: CategoryConfigurationError,
				Message:  "suite " + suiteName + " declares missing roles: " + joinComma(entry.Plan.MissingRoles) + ".",
			})
			continue
		}

		entries = append(entries, *entry)
	}

	return ConformanceManifestPlan{
		Entries:     entries,
		Diagnostics: diagnostics,
	}
}

func derefRequirements(requirements *ConformanceCaseRequirements) ConformanceCaseRequirements {
	if requirements == nil {
		return ConformanceCaseRequirements{}
	}

	return *requirements
}

func joinComma(values []string) string {
	if len(values) == 0 {
		return ""
	}

	result := values[0]
	for _, value := range values[1:] {
		result += ", " + value
	}

	return result
}

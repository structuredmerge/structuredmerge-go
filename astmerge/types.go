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
	Role string   `json:"role"`
	Path []string `json:"path"`
}

type ConformanceFamilyFeatureProfileEntry struct {
	Family string   `json:"family"`
	Role   string   `json:"role"`
	Path   []string `json:"path"`
}

type ConformanceManifest struct {
	FamilyFeatureProfiles []ConformanceFamilyFeatureProfileEntry `json:"family_feature_profiles"`
	Families              map[string][]ConformanceManifestEntry  `json:"families"`
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

func ReportPlannedConformanceSuite(
	plan ConformanceSuitePlan,
	execute func(ConformanceCaseRun) ConformanceCaseExecution,
) ConformanceSuiteReport {
	return ReportConformanceSuite(RunPlannedConformanceSuite(plan, execute))
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
		path := ConformanceFixturePath(manifest, family, role)
		if path == nil {
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
			Path: slices.Clone(path),
			Run: ConformanceCaseRun{
				Ref:            ref,
				Requirements:   ConformanceCaseRequirements{},
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

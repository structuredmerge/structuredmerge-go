package asttemplate

import (
	"slices"
	"strings"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/markdownmerge"
	"github.com/structuredmerge/structuredmerge-go/rubymerge"
	"github.com/structuredmerge/structuredmerge-go/tomlmerge"
)

type DirectorySessionMode string

const (
	DirectorySessionModePlan    DirectorySessionMode = "plan"
	DirectorySessionModeApply   DirectorySessionMode = "apply"
	DirectorySessionModeReapply DirectorySessionMode = "reapply"
)

type DirectorySessionReport struct {
	Mode         DirectorySessionMode                   `json:"mode"`
	RunnerReport astmerge.TemplateDirectoryRunnerReport `json:"runner_report"`
}

type FamilyMergeAdapter func(astmerge.TemplateExecutionPlanEntry) astmerge.MergeResult[string]

type FamilyMergeAdapterRegistry map[string]FamilyMergeAdapter

type DirectoryRegistrySessionReport struct {
	Mode            DirectorySessionMode                   `json:"mode"`
	AdapterFamilies []string                               `json:"adapter_families"`
	Diagnostics     []astmerge.Diagnostic                  `json:"diagnostics"`
	RunnerReport    astmerge.TemplateDirectoryRunnerReport `json:"runner_report"`
}

type AdapterCapabilityReport struct {
	RequiredFamilies []string `json:"required_families"`
	AdapterFamilies  []string `json:"adapter_families"`
	MissingFamilies  []string `json:"missing_families"`
	Ready            bool     `json:"ready"`
}

type SessionEnvelopeReport struct {
	SessionReport       any                     `json:"session_report"`
	AdapterCapabilities AdapterCapabilityReport `json:"adapter_capabilities"`
}

type SessionStatusReport struct {
	Mode              DirectorySessionMode `json:"mode"`
	Ready             bool                 `json:"ready"`
	MissingFamilies   []string             `json:"missing_families"`
	BlockedPaths      []string             `json:"blocked_paths"`
	PlannedWriteCount int                  `json:"planned_write_count"`
	WrittenCount      int                  `json:"written_count"`
}

type SessionDiagnostic struct {
	Severity astmerge.DiagnosticSeverity `json:"severity"`
	Category astmerge.DiagnosticCategory `json:"category"`
	Reason   string                      `json:"reason"`
	Path     string                      `json:"path,omitempty"`
	Family   string                      `json:"family,omitempty"`
	Message  string                      `json:"message"`
}

type SessionDiagnosticsReport struct {
	Mode        DirectorySessionMode `json:"mode"`
	Ready       bool                 `json:"ready"`
	Diagnostics []SessionDiagnostic  `json:"diagnostics"`
}

type SessionOutcomeReport struct {
	SessionReport any                      `json:"session_report"`
	Status        SessionStatusReport      `json:"status"`
	Diagnostics   SessionDiagnosticsReport `json:"diagnostics"`
}

type SessionRequestReport struct {
	RequestKind     string                   `json:"request_kind"`
	ProfileName     string                   `json:"profile_name,omitempty"`
	Mode            DirectorySessionMode     `json:"mode"`
	Ready           bool                     `json:"ready"`
	Diagnostics     []SessionDiagnostic      `json:"diagnostics"`
	ResolvedOptions *DirectorySessionOptions `json:"resolved_options"`
}

type SessionRunnerRequest struct {
	RequestKind string         `json:"request_kind"`
	ProfileName string         `json:"profile_name,omitempty"`
	Options     map[string]any `json:"options,omitempty"`
	Overrides   map[string]any `json:"overrides,omitempty"`
}

type SessionRunnerInput struct {
	RequestKind     string                               `json:"request_kind"`
	ProfileName     string                               `json:"profile_name,omitempty"`
	Mode            DirectorySessionMode                 `json:"mode"`
	TemplateRoot    string                               `json:"template_root"`
	DestinationRoot string                               `json:"destination_root"`
	Context         *astmerge.TemplateDestinationContext `json:"context"`
	DefaultStrategy astmerge.TemplateStrategy            `json:"default_strategy"`
	Overrides       []astmerge.TemplateStrategyOverride  `json:"overrides"`
	Replacements    map[string]string                    `json:"replacements"`
	AllowedFamilies []string                             `json:"allowed_families"`
}

type SessionRunnerPayload struct {
	RequestKind        string                               `json:"request_kind,omitempty"`
	DefaultProfileName string                               `json:"default_profile_name,omitempty"`
	ProfileName        string                               `json:"profile_name,omitempty"`
	Mode               DirectorySessionMode                 `json:"mode"`
	TemplateRoot       string                               `json:"template_root"`
	DestinationRoot    string                               `json:"destination_root"`
	Context            *astmerge.TemplateDestinationContext `json:"context"`
	DefaultStrategy    astmerge.TemplateStrategy            `json:"default_strategy"`
	Overrides          []astmerge.TemplateStrategyOverride  `json:"overrides"`
	Replacements       map[string]string                    `json:"replacements"`
	AllowedFamilies    []string                             `json:"allowed_families"`
}

type DirectorySessionOptions struct {
	Mode            DirectorySessionMode                 `json:"mode"`
	TemplateRoot    string                               `json:"template_root"`
	DestinationRoot string                               `json:"destination_root"`
	Context         *astmerge.TemplateDestinationContext `json:"context"`
	DefaultStrategy astmerge.TemplateStrategy            `json:"default_strategy"`
	Overrides       []astmerge.TemplateStrategyOverride  `json:"overrides"`
	Replacements    map[string]string                    `json:"replacements"`
	AllowedFamilies []string                             `json:"allowed_families"`
	Config          *astmerge.TemplateTokenConfig        `json:"config,omitempty"`
}

type DirectorySessionProfile struct {
	Mode            DirectorySessionMode                 `json:"mode"`
	Context         *astmerge.TemplateDestinationContext `json:"context"`
	DefaultStrategy astmerge.TemplateStrategy            `json:"default_strategy"`
	Overrides       []astmerge.TemplateStrategyOverride  `json:"overrides"`
	Replacements    map[string]string                    `json:"replacements"`
	AllowedFamilies []string                             `json:"allowed_families"`
	Config          *astmerge.TemplateTokenConfig        `json:"config,omitempty"`
}

func ReportTemplateDirectorySession(mode DirectorySessionMode, entries []astmerge.TemplateExecutionPlanEntry, result *astmerge.TemplateTreeRunResult) DirectorySessionReport {
	return DirectorySessionReport{
		Mode:         mode,
		RunnerReport: astmerge.ReportTemplateDirectoryRunner(entries, result),
	}
}

func PlanTemplateDirectorySessionFromDirectories(
	templateRoot string,
	destinationRoot string,
	context *astmerge.TemplateDestinationContext,
	defaultStrategy astmerge.TemplateStrategy,
	overrides []astmerge.TemplateStrategyOverride,
	replacements map[string]string,
	config *astmerge.TemplateTokenConfig,
) (DirectorySessionReport, error) {
	plan, err := astmerge.PlanTemplateTreeExecutionFromDirectories(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		config,
	)
	if err != nil {
		return DirectorySessionReport{}, err
	}
	return ReportTemplateDirectorySession(DirectorySessionModePlan, plan, nil), nil
}

func ApplyTemplateDirectorySessionToDirectory(
	templateRoot string,
	destinationRoot string,
	context *astmerge.TemplateDestinationContext,
	defaultStrategy astmerge.TemplateStrategy,
	overrides []astmerge.TemplateStrategyOverride,
	replacements map[string]string,
	mergePreparedContent func(astmerge.TemplateExecutionPlanEntry) astmerge.MergeResult[string],
	config *astmerge.TemplateTokenConfig,
) (DirectorySessionReport, error) {
	result, err := astmerge.ApplyTemplateTreeExecutionToDirectory(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		mergePreparedContent,
		config,
	)
	if err != nil {
		return DirectorySessionReport{}, err
	}
	return ReportTemplateDirectorySession(DirectorySessionModeApply, result.ExecutionPlan, &result), nil
}

func ReapplyTemplateDirectorySessionToDirectory(
	templateRoot string,
	destinationRoot string,
	context *astmerge.TemplateDestinationContext,
	defaultStrategy astmerge.TemplateStrategy,
	overrides []astmerge.TemplateStrategyOverride,
	replacements map[string]string,
	mergePreparedContent func(astmerge.TemplateExecutionPlanEntry) astmerge.MergeResult[string],
	config *astmerge.TemplateTokenConfig,
) (DirectorySessionReport, error) {
	result, err := astmerge.ApplyTemplateTreeExecutionToDirectory(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		mergePreparedContent,
		config,
	)
	if err != nil {
		return DirectorySessionReport{}, err
	}
	return ReportTemplateDirectorySession(DirectorySessionModeReapply, result.ExecutionPlan, &result), nil
}

func MergePreparedContentFromRegistry(registry FamilyMergeAdapterRegistry, entry astmerge.TemplateExecutionPlanEntry) astmerge.MergeResult[string] {
	family := entry.Classification.Family
	adapter, ok := registry[family]
	if !ok {
		return astmerge.MergeResult[string]{
			OK: false,
			Diagnostics: []astmerge.Diagnostic{{
				Severity: astmerge.SeverityError,
				Category: astmerge.CategoryConfigurationError,
				Message:  "missing family adapter for " + family,
			}},
		}
	}
	return adapter(entry)
}

func RegisteredAdapterFamilies(registry FamilyMergeAdapterRegistry) []string {
	families := make([]string, 0, len(registry))
	for family := range registry {
		families = append(families, family)
	}
	slices.Sort(families)
	return families
}

func ReportTemplateDirectoryRegistrySession(mode DirectorySessionMode, entries []astmerge.TemplateExecutionPlanEntry, result *astmerge.TemplateTreeRunResult, registry FamilyMergeAdapterRegistry) DirectoryRegistrySessionReport {
	diagnostics := []astmerge.Diagnostic{}
	if result != nil {
		diagnostics = append(diagnostics, result.ApplyResult.Diagnostics...)
	}
	return DirectoryRegistrySessionReport{
		Mode:            mode,
		AdapterFamilies: RegisteredAdapterFamilies(registry),
		Diagnostics:     diagnostics,
		RunnerReport:    astmerge.ReportTemplateDirectoryRunner(entries, result),
	}
}

func ApplyTemplateDirectorySessionWithRegistryToDirectory(
	templateRoot string,
	destinationRoot string,
	context *astmerge.TemplateDestinationContext,
	defaultStrategy astmerge.TemplateStrategy,
	overrides []astmerge.TemplateStrategyOverride,
	replacements map[string]string,
	registry FamilyMergeAdapterRegistry,
	config *astmerge.TemplateTokenConfig,
) (DirectoryRegistrySessionReport, error) {
	result, err := astmerge.ApplyTemplateTreeExecutionToDirectory(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		func(entry astmerge.TemplateExecutionPlanEntry) astmerge.MergeResult[string] {
			return MergePreparedContentFromRegistry(registry, entry)
		},
		config,
	)
	if err != nil {
		return DirectoryRegistrySessionReport{}, err
	}
	return ReportTemplateDirectoryRegistrySession(DirectorySessionModeApply, result.ExecutionPlan, &result, registry), nil
}

func DefaultFamilyMergeAdapterRegistry(allowedFamilies ...string) FamilyMergeAdapterRegistry {
	allowed := map[string]struct{}{}
	for _, family := range allowedFamilies {
		allowed[family] = struct{}{}
	}
	include := func(family string) bool {
		if len(allowed) == 0 {
			return true
		}
		_, ok := allowed[family]
		return ok
	}

	registry := FamilyMergeAdapterRegistry{}
	if include("markdown") {
		registry["markdown"] = func(entry astmerge.TemplateExecutionPlanEntry) astmerge.MergeResult[string] {
			template := ""
			if entry.PreparedTemplateContent != nil {
				template = *entry.PreparedTemplateContent
			}
			destination := ""
			if entry.DestinationContent != nil {
				destination = *entry.DestinationContent
			}
			return markdownmerge.MergeMarkdown(template, destination, "markdown")
		}
	}
	if include("toml") {
		registry["toml"] = func(entry astmerge.TemplateExecutionPlanEntry) astmerge.MergeResult[string] {
			template := ""
			if entry.PreparedTemplateContent != nil {
				template = *entry.PreparedTemplateContent
			}
			destination := ""
			if entry.DestinationContent != nil {
				destination = *entry.DestinationContent
			}
			return tomlmerge.MergeTOML(template, destination, "toml")
		}
	}
	if include("ruby") {
		registry["ruby"] = func(entry astmerge.TemplateExecutionPlanEntry) astmerge.MergeResult[string] {
			template := ""
			if entry.PreparedTemplateContent != nil {
				template = *entry.PreparedTemplateContent
			}
			destination := ""
			if entry.DestinationContent != nil {
				destination = *entry.DestinationContent
			}
			return rubymerge.MergeRuby(template, destination, "ruby")
		}
	}
	return registry
}

func ApplyTemplateDirectorySessionWithDefaultRegistryToDirectory(
	templateRoot string,
	destinationRoot string,
	context *astmerge.TemplateDestinationContext,
	defaultStrategy astmerge.TemplateStrategy,
	overrides []astmerge.TemplateStrategyOverride,
	replacements map[string]string,
	allowedFamilies []string,
	config *astmerge.TemplateTokenConfig,
) (DirectoryRegistrySessionReport, error) {
	return ApplyTemplateDirectorySessionWithRegistryToDirectory(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		DefaultFamilyMergeAdapterRegistry(allowedFamilies...),
		config,
	)
}

func RequiredFamilies(entries []astmerge.TemplateExecutionPlanEntry) []string {
	families := map[string]struct{}{}
	for _, entry := range entries {
		if entry.ExecutionAction != astmerge.TemplateExecutionMergePrepared {
			continue
		}
		families[entry.Classification.Family] = struct{}{}
	}
	required := make([]string, 0, len(families))
	for family := range families {
		required = append(required, family)
	}
	slices.Sort(required)
	return required
}

func ReportAdapterCapabilities(entries []astmerge.TemplateExecutionPlanEntry, registry FamilyMergeAdapterRegistry) AdapterCapabilityReport {
	required := RequiredFamilies(entries)
	available := RegisteredAdapterFamilies(registry)
	availableSet := map[string]struct{}{}
	for _, family := range available {
		availableSet[family] = struct{}{}
	}
	missing := []string{}
	for _, family := range required {
		if _, ok := availableSet[family]; !ok {
			missing = append(missing, family)
		}
	}
	return AdapterCapabilityReport{
		RequiredFamilies: required,
		AdapterFamilies:  available,
		MissingFamilies:  missing,
		Ready:            len(missing) == 0,
	}
}

func ReportAdapterCapabilitiesFromDirectories(
	templateRoot string,
	destinationRoot string,
	context *astmerge.TemplateDestinationContext,
	defaultStrategy astmerge.TemplateStrategy,
	overrides []astmerge.TemplateStrategyOverride,
	replacements map[string]string,
	registry FamilyMergeAdapterRegistry,
	config *astmerge.TemplateTokenConfig,
) (AdapterCapabilityReport, error) {
	plan, err := astmerge.PlanTemplateTreeExecutionFromDirectories(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		config,
	)
	if err != nil {
		return AdapterCapabilityReport{}, err
	}
	return ReportAdapterCapabilities(plan, registry), nil
}

func ReportDefaultAdapterCapabilitiesFromDirectories(
	templateRoot string,
	destinationRoot string,
	context *astmerge.TemplateDestinationContext,
	defaultStrategy astmerge.TemplateStrategy,
	overrides []astmerge.TemplateStrategyOverride,
	replacements map[string]string,
	allowedFamilies []string,
	config *astmerge.TemplateTokenConfig,
) (AdapterCapabilityReport, error) {
	return ReportAdapterCapabilitiesFromDirectories(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		DefaultFamilyMergeAdapterRegistry(allowedFamilies...),
		config,
	)
}

func ReportTemplateDirectorySessionEnvelope(
	sessionReport any,
	adapterCapabilities AdapterCapabilityReport,
) SessionEnvelopeReport {
	return SessionEnvelopeReport{
		SessionReport:       sessionReport,
		AdapterCapabilities: adapterCapabilities,
	}
}

func PlanTemplateDirectorySessionEnvelopeFromDirectories(
	templateRoot string,
	destinationRoot string,
	context *astmerge.TemplateDestinationContext,
	defaultStrategy astmerge.TemplateStrategy,
	overrides []astmerge.TemplateStrategyOverride,
	replacements map[string]string,
	allowedFamilies []string,
	config *astmerge.TemplateTokenConfig,
) (SessionEnvelopeReport, error) {
	sessionReport, err := PlanTemplateDirectorySessionFromDirectories(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		config,
	)
	if err != nil {
		return SessionEnvelopeReport{}, err
	}
	capabilities, err := ReportDefaultAdapterCapabilitiesFromDirectories(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		allowedFamilies,
		config,
	)
	if err != nil {
		return SessionEnvelopeReport{}, err
	}
	return ReportTemplateDirectorySessionEnvelope(sessionReport, capabilities), nil
}

func ApplyTemplateDirectorySessionEnvelopeWithDefaultRegistryToDirectory(
	templateRoot string,
	destinationRoot string,
	context *astmerge.TemplateDestinationContext,
	defaultStrategy astmerge.TemplateStrategy,
	overrides []astmerge.TemplateStrategyOverride,
	replacements map[string]string,
	allowedFamilies []string,
	config *astmerge.TemplateTokenConfig,
) (SessionEnvelopeReport, error) {
	sessionReport, err := ApplyTemplateDirectorySessionWithDefaultRegistryToDirectory(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		allowedFamilies,
		config,
	)
	if err != nil {
		return SessionEnvelopeReport{}, err
	}
	capabilities, err := ReportDefaultAdapterCapabilitiesFromDirectories(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		allowedFamilies,
		config,
	)
	if err != nil {
		return SessionEnvelopeReport{}, err
	}
	return ReportTemplateDirectorySessionEnvelope(sessionReport, capabilities), nil
}

func ReportTemplateDirectorySessionStatus(envelope SessionEnvelopeReport) SessionStatusReport {
	mode, runnerReport := sessionEnvelopeModeAndRunner(envelope.SessionReport)
	blockedPathSet := map[string]struct{}{}
	for _, entry := range runnerReport.PlanReport.Entries {
		if entry.Status == astmerge.TemplateDirectoryPlanBlocked && entry.DestinationPath != nil {
			blockedPathSet[*entry.DestinationPath] = struct{}{}
		}
	}
	if runnerReport.ApplyReport != nil {
		for _, entry := range runnerReport.ApplyReport.Entries {
			if entry.Status == astmerge.TemplateTreeRunBlocked && entry.DestinationPath != nil {
				blockedPathSet[*entry.DestinationPath] = struct{}{}
			}
		}
	}
	blockedPaths := make([]string, 0, len(blockedPathSet))
	for path := range blockedPathSet {
		blockedPaths = append(blockedPaths, path)
	}
	slices.Sort(blockedPaths)
	plannedWriteCount := runnerReport.PlanReport.Summary.Create + runnerReport.PlanReport.Summary.Update
	writtenCount := 0
	if runnerReport.ApplyReport != nil {
		writtenCount = runnerReport.ApplyReport.Summary.Written
	}
	missingFamilies := append([]string{}, envelope.AdapterCapabilities.MissingFamilies...)
	slices.Sort(missingFamilies)
	return SessionStatusReport{
		Mode:              mode,
		Ready:             envelope.AdapterCapabilities.Ready && len(blockedPaths) == 0,
		MissingFamilies:   missingFamilies,
		BlockedPaths:      blockedPaths,
		PlannedWriteCount: plannedWriteCount,
		WrittenCount:      writtenCount,
	}
}

func sessionEnvelopeModeAndRunner(sessionReport any) (DirectorySessionMode, astmerge.TemplateDirectoryRunnerReport) {
	switch report := sessionReport.(type) {
	case DirectorySessionReport:
		return report.Mode, report.RunnerReport
	case *DirectorySessionReport:
		return report.Mode, report.RunnerReport
	case DirectoryRegistrySessionReport:
		return report.Mode, report.RunnerReport
	case *DirectoryRegistrySessionReport:
		return report.Mode, report.RunnerReport
	default:
		return DirectorySessionModePlan, astmerge.TemplateDirectoryRunnerReport{}
	}
}

func ReportTemplateDirectorySessionDiagnostics(
	mode DirectorySessionMode,
	entries []astmerge.TemplateExecutionPlanEntry,
	result *astmerge.TemplateTreeRunResult,
	capabilities AdapterCapabilityReport,
) SessionDiagnosticsReport {
	diagnostics := []SessionDiagnostic{}
	missingFamilies := map[string]struct{}{}
	for _, family := range capabilities.MissingFamilies {
		missingFamilies[family] = struct{}{}
	}
	blockedByPath := map[string]struct{}{}
	if result != nil {
		for _, path := range result.ApplyResult.BlockedPaths {
			blockedByPath[path] = struct{}{}
		}
	}
	for _, entry := range entries {
		path := entry.LogicalDestinationPath
		if entry.DestinationPath != nil {
			path = *entry.DestinationPath
		}
		if entry.Blocked && entry.BlockReason != nil && *entry.BlockReason == astmerge.TemplatePlanBlockReasonUnresolvedTokens {
			diagnostics = append(diagnostics, SessionDiagnostic{
				Severity: astmerge.SeverityError,
				Category: astmerge.CategoryConfigurationError,
				Reason:   "unresolved_tokens",
				Path:     path,
				Message:  "unresolved template tokens block " + path,
			})
		}
		if _, ok := missingFamilies[entry.Classification.Family]; ok &&
			entry.ExecutionAction == astmerge.TemplateExecutionMergePrepared {
			if result == nil || len(blockedByPath) == 0 {
				diagnostics = append(diagnostics, SessionDiagnostic{
					Severity: astmerge.SeverityError,
					Category: astmerge.CategoryConfigurationError,
					Reason:   "missing_family_adapter",
					Path:     path,
					Family:   entry.Classification.Family,
					Message:  "missing family adapter for " + entry.Classification.Family + " blocks " + path,
				})
				continue
			}
			if _, blocked := blockedByPath[path]; blocked {
				diagnostics = append(diagnostics, SessionDiagnostic{
					Severity: astmerge.SeverityError,
					Category: astmerge.CategoryConfigurationError,
					Reason:   "missing_family_adapter",
					Path:     path,
					Family:   entry.Classification.Family,
					Message:  "missing family adapter for " + entry.Classification.Family + " blocks " + path,
				})
			}
		}
	}
	slices.SortFunc(diagnostics, func(a, b SessionDiagnostic) int {
		if a.Path != b.Path {
			return strings.Compare(a.Path, b.Path)
		}
		if a.Reason != b.Reason {
			return strings.Compare(a.Reason, b.Reason)
		}
		return strings.Compare(a.Family, b.Family)
	})
	return SessionDiagnosticsReport{
		Mode:        mode,
		Ready:       len(diagnostics) == 0,
		Diagnostics: diagnostics,
	}
}

func PlanTemplateDirectorySessionDiagnosticsFromDirectories(
	templateRoot string,
	destinationRoot string,
	context *astmerge.TemplateDestinationContext,
	defaultStrategy astmerge.TemplateStrategy,
	overrides []astmerge.TemplateStrategyOverride,
	replacements map[string]string,
	allowedFamilies []string,
	config *astmerge.TemplateTokenConfig,
) (SessionDiagnosticsReport, error) {
	entries, err := astmerge.PlanTemplateTreeExecutionFromDirectories(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		config,
	)
	if err != nil {
		return SessionDiagnosticsReport{}, err
	}
	capabilities := ReportAdapterCapabilities(entries, DefaultFamilyMergeAdapterRegistry(allowedFamilies...))
	return ReportTemplateDirectorySessionDiagnostics(DirectorySessionModePlan, entries, nil, capabilities), nil
}

func ApplyTemplateDirectorySessionDiagnosticsWithDefaultRegistryToDirectory(
	templateRoot string,
	destinationRoot string,
	context *astmerge.TemplateDestinationContext,
	defaultStrategy astmerge.TemplateStrategy,
	overrides []astmerge.TemplateStrategyOverride,
	replacements map[string]string,
	allowedFamilies []string,
	config *astmerge.TemplateTokenConfig,
) (SessionDiagnosticsReport, error) {
	registry := DefaultFamilyMergeAdapterRegistry(allowedFamilies...)
	result, err := astmerge.ApplyTemplateTreeExecutionToDirectory(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		func(entry astmerge.TemplateExecutionPlanEntry) astmerge.MergeResult[string] {
			return MergePreparedContentFromRegistry(registry, entry)
		},
		config,
	)
	if err != nil {
		return SessionDiagnosticsReport{}, err
	}
	capabilities := ReportAdapterCapabilities(result.ExecutionPlan, registry)
	return ReportTemplateDirectorySessionDiagnostics(DirectorySessionModeApply, result.ExecutionPlan, &result, capabilities), nil
}

func ReportTemplateDirectorySessionOutcome(
	sessionReport any,
	status SessionStatusReport,
	diagnostics SessionDiagnosticsReport,
) SessionOutcomeReport {
	return SessionOutcomeReport{
		SessionReport: sessionReport,
		Status:        status,
		Diagnostics:   diagnostics,
	}
}

func PlanTemplateDirectorySessionOutcomeFromDirectories(
	templateRoot string,
	destinationRoot string,
	context *astmerge.TemplateDestinationContext,
	defaultStrategy astmerge.TemplateStrategy,
	overrides []astmerge.TemplateStrategyOverride,
	replacements map[string]string,
	allowedFamilies []string,
	config *astmerge.TemplateTokenConfig,
) (SessionOutcomeReport, error) {
	sessionReport, err := PlanTemplateDirectorySessionFromDirectories(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		config,
	)
	if err != nil {
		return SessionOutcomeReport{}, err
	}
	envelope, err := PlanTemplateDirectorySessionEnvelopeFromDirectories(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		allowedFamilies,
		config,
	)
	if err != nil {
		return SessionOutcomeReport{}, err
	}
	diagnostics, err := PlanTemplateDirectorySessionDiagnosticsFromDirectories(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		allowedFamilies,
		config,
	)
	if err != nil {
		return SessionOutcomeReport{}, err
	}
	return ReportTemplateDirectorySessionOutcome(
		sessionReport,
		ReportTemplateDirectorySessionStatus(envelope),
		diagnostics,
	), nil
}

func ApplyTemplateDirectorySessionOutcomeWithDefaultRegistryToDirectory(
	templateRoot string,
	destinationRoot string,
	context *astmerge.TemplateDestinationContext,
	defaultStrategy astmerge.TemplateStrategy,
	overrides []astmerge.TemplateStrategyOverride,
	replacements map[string]string,
	allowedFamilies []string,
	config *astmerge.TemplateTokenConfig,
) (SessionOutcomeReport, error) {
	registry := DefaultFamilyMergeAdapterRegistry(allowedFamilies...)
	result, err := astmerge.ApplyTemplateTreeExecutionToDirectory(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		func(entry astmerge.TemplateExecutionPlanEntry) astmerge.MergeResult[string] {
			return MergePreparedContentFromRegistry(registry, entry)
		},
		config,
	)
	if err != nil {
		return SessionOutcomeReport{}, err
	}
	sessionReport := ReportTemplateDirectoryRegistrySession(DirectorySessionModeApply, result.ExecutionPlan, &result, registry)
	capabilities := ReportAdapterCapabilities(result.ExecutionPlan, registry)
	status := ReportTemplateDirectorySessionStatus(ReportTemplateDirectorySessionEnvelope(sessionReport, capabilities))
	diagnostics := ReportTemplateDirectorySessionDiagnostics(DirectorySessionModeApply, result.ExecutionPlan, &result, capabilities)
	return ReportTemplateDirectorySessionOutcome(
		sessionReport,
		status,
		diagnostics,
	), nil
}

func ReapplyTemplateDirectorySessionOutcomeWithDefaultRegistryToDirectory(
	templateRoot string,
	destinationRoot string,
	context *astmerge.TemplateDestinationContext,
	defaultStrategy astmerge.TemplateStrategy,
	overrides []astmerge.TemplateStrategyOverride,
	replacements map[string]string,
	allowedFamilies []string,
	config *astmerge.TemplateTokenConfig,
) (SessionOutcomeReport, error) {
	registry := DefaultFamilyMergeAdapterRegistry(allowedFamilies...)
	result, err := astmerge.ApplyTemplateTreeExecutionToDirectory(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		func(entry astmerge.TemplateExecutionPlanEntry) astmerge.MergeResult[string] {
			return MergePreparedContentFromRegistry(registry, entry)
		},
		config,
	)
	if err != nil {
		return SessionOutcomeReport{}, err
	}
	sessionReport := ReportTemplateDirectoryRegistrySession(DirectorySessionModeReapply, result.ExecutionPlan, &result, registry)
	capabilities := ReportAdapterCapabilities(result.ExecutionPlan, registry)
	status := ReportTemplateDirectorySessionStatus(ReportTemplateDirectorySessionEnvelope(sessionReport, capabilities))
	diagnostics := ReportTemplateDirectorySessionDiagnostics(DirectorySessionModeReapply, result.ExecutionPlan, &result, capabilities)
	return ReportTemplateDirectorySessionOutcome(
		sessionReport,
		status,
		diagnostics,
	), nil
}

func RunTemplateDirectorySessionWithDefaultRegistryToDirectory(
	mode DirectorySessionMode,
	templateRoot string,
	destinationRoot string,
	context *astmerge.TemplateDestinationContext,
	defaultStrategy astmerge.TemplateStrategy,
	overrides []astmerge.TemplateStrategyOverride,
	replacements map[string]string,
	allowedFamilies []string,
	config *astmerge.TemplateTokenConfig,
) (SessionOutcomeReport, error) {
	switch mode {
	case DirectorySessionModePlan:
		return PlanTemplateDirectorySessionOutcomeFromDirectories(
			templateRoot,
			destinationRoot,
			context,
			defaultStrategy,
			overrides,
			replacements,
			allowedFamilies,
			config,
		)
	case DirectorySessionModeApply:
		return ApplyTemplateDirectorySessionOutcomeWithDefaultRegistryToDirectory(
			templateRoot,
			destinationRoot,
			context,
			defaultStrategy,
			overrides,
			replacements,
			allowedFamilies,
			config,
		)
	case DirectorySessionModeReapply:
		return ReapplyTemplateDirectorySessionOutcomeWithDefaultRegistryToDirectory(
			templateRoot,
			destinationRoot,
			context,
			defaultStrategy,
			overrides,
			replacements,
			allowedFamilies,
			config,
		)
	default:
		return SessionOutcomeReport{}, nil
	}
}

func RunTemplateDirectorySessionWithOptions(options DirectorySessionOptions) (SessionOutcomeReport, error) {
	request := ReportTemplateDirectorySessionOptionsRequest(options)
	if !request.Ready {
		return reportTemplateDirectorySessionConfigurationOutcome(request.Mode, SessionDiagnosticsReport{
			Mode:        request.Mode,
			Ready:       request.Ready,
			Diagnostics: request.Diagnostics,
		}), nil
	}
	resolved := *request.ResolvedOptions
	return RunTemplateDirectorySessionWithDefaultRegistryToDirectory(
		resolved.Mode,
		resolved.TemplateRoot,
		resolved.DestinationRoot,
		resolved.Context,
		resolved.DefaultStrategy,
		resolved.Overrides,
		resolved.Replacements,
		resolved.AllowedFamilies,
		resolved.Config,
	)
}

func normalizeSessionMode(mode DirectorySessionMode) DirectorySessionMode {
	switch mode {
	case DirectorySessionModeApply, DirectorySessionModeReapply:
		return mode
	default:
		return DirectorySessionModePlan
	}
}

func ReportTemplateDirectorySessionOptionsConfiguration(options DirectorySessionOptions) SessionDiagnosticsReport {
	diagnostics := []SessionDiagnostic{}
	if options.DestinationRoot == "" {
		diagnostics = append(diagnostics, SessionDiagnostic{
			Severity: astmerge.SeverityError,
			Category: astmerge.CategoryConfigurationError,
			Reason:   "missing_destination_root",
			Message:  "missing destination_root for template session",
		})
	}
	if options.TemplateRoot == "" {
		diagnostics = append(diagnostics, SessionDiagnostic{
			Severity: astmerge.SeverityError,
			Category: astmerge.CategoryConfigurationError,
			Reason:   "missing_template_root",
			Message:  "missing template_root for template session",
		})
	}
	slices.SortFunc(diagnostics, func(a, b SessionDiagnostic) int {
		return strings.Compare(a.Reason, b.Reason)
	})
	return SessionDiagnosticsReport{
		Mode:        normalizeSessionMode(options.Mode),
		Ready:       len(diagnostics) == 0,
		Diagnostics: diagnostics,
	}
}

func ReportTemplateDirectorySessionProfileConfiguration(
	profiles map[string]DirectorySessionProfile,
	profileName string,
	overrides DirectorySessionOptions,
) SessionDiagnosticsReport {
	report := ReportTemplateDirectorySessionOptionsConfiguration(overrides)
	if profile, ok := profiles[profileName]; ok {
		report.Mode = normalizeSessionMode(firstNonEmptySessionMode(overrides.Mode, profile.Mode))
	} else {
		report.Diagnostics = append(report.Diagnostics, SessionDiagnostic{
			Severity: astmerge.SeverityError,
			Category: astmerge.CategoryConfigurationError,
			Reason:   "missing_profile",
			Message:  "unknown template session profile: " + profileName,
		})
		report.Mode = normalizeSessionMode(overrides.Mode)
	}
	slices.SortFunc(report.Diagnostics, func(a, b SessionDiagnostic) int {
		return strings.Compare(a.Reason, b.Reason)
	})
	report.Ready = len(report.Diagnostics) == 0
	return report
}

func ReportTemplateDirectorySessionOptionsRequest(options DirectorySessionOptions) SessionRequestReport {
	configuration := ReportTemplateDirectorySessionOptionsConfiguration(options)
	var resolved *DirectorySessionOptions
	if configuration.Ready {
		copy := options
		resolved = &copy
	}
	return SessionRequestReport{
		RequestKind:     "options",
		Mode:            configuration.Mode,
		Ready:           configuration.Ready,
		Diagnostics:     configuration.Diagnostics,
		ResolvedOptions: resolved,
	}
}

func ReportTemplateDirectorySessionProfileRequest(
	profiles map[string]DirectorySessionProfile,
	profileName string,
	overrides DirectorySessionOptions,
) SessionRequestReport {
	configuration := ReportTemplateDirectorySessionProfileConfiguration(profiles, profileName, overrides)
	var resolved *DirectorySessionOptions
	if configuration.Ready {
		if options, ok := ResolveTemplateDirectorySessionOptions(profiles, profileName, overrides); ok {
			copy := options
			resolved = &copy
		}
	}
	return SessionRequestReport{
		RequestKind:     "profile",
		ProfileName:     profileName,
		Mode:            configuration.Mode,
		Ready:           configuration.Ready,
		Diagnostics:     configuration.Diagnostics,
		ResolvedOptions: resolved,
	}
}

func reportTemplateDirectorySessionConfigurationOutcome(
	mode DirectorySessionMode,
	diagnostics SessionDiagnosticsReport,
) SessionOutcomeReport {
	return ReportTemplateDirectorySessionOutcome(
		ReportTemplateDirectorySession(mode, nil, nil),
		SessionStatusReport{
			Mode:              mode,
			Ready:             false,
			MissingFamilies:   []string{},
			BlockedPaths:      []string{},
			PlannedWriteCount: 0,
			WrittenCount:      0,
		},
		diagnostics,
	)
}

func RunTemplateDirectorySessionRequest(request SessionRequestReport) (SessionOutcomeReport, error) {
	if !request.Ready {
		return reportTemplateDirectorySessionConfigurationOutcome(request.Mode, SessionDiagnosticsReport{
			Mode:        request.Mode,
			Ready:       request.Ready,
			Diagnostics: request.Diagnostics,
		}), nil
	}
	if request.ResolvedOptions == nil {
		return reportTemplateDirectorySessionConfigurationOutcome(request.Mode, SessionDiagnosticsReport{
			Mode:  request.Mode,
			Ready: false,
			Diagnostics: []SessionDiagnostic{{
				Severity: astmerge.SeverityError,
				Category: astmerge.CategoryConfigurationError,
				Reason:   "missing_resolved_options",
				Message:  "ready template session request is missing resolved_options",
			}},
		}), nil
	}
	return RunTemplateDirectorySessionWithOptions(*request.ResolvedOptions)
}

func RunTemplateDirectorySessionRunnerRequest(
	request SessionRunnerRequest,
	profiles map[string]DirectorySessionProfile,
) (SessionOutcomeReport, error) {
	if request.RequestKind == "profile" {
		overrides := DirectorySessionOptions{}
		if request.Overrides != nil {
			overrides = decodeSessionRunnerOptions(request.Overrides)
		}
		return RunTemplateDirectorySessionRequest(
			ReportTemplateDirectorySessionProfileRequest(profiles, request.ProfileName, overrides),
		)
	}
	options := DirectorySessionOptions{}
	if request.Options != nil {
		options = decodeSessionRunnerOptions(request.Options)
	}
	return RunTemplateDirectorySessionRequest(ReportTemplateDirectorySessionOptionsRequest(options))
}

func ReportTemplateDirectorySessionRunnerInput(input SessionRunnerInput) SessionRunnerRequest {
	if input.RequestKind == "profile" {
		return SessionRunnerRequest{
			RequestKind: input.RequestKind,
			ProfileName: input.ProfileName,
			Overrides:   reportSessionRunnerInputOverrides(input),
		}
	}
	return SessionRunnerRequest{
		RequestKind: input.RequestKind,
		Options:     reportSessionRunnerInputOptions(input),
	}
}

func ReportTemplateDirectorySessionRunnerPayload(payload SessionRunnerPayload) SessionRunnerInput {
	requestKind := payload.RequestKind
	if requestKind == "" {
		if payload.ProfileName != "" || payload.DefaultProfileName != "" {
			requestKind = "profile"
		} else {
			requestKind = "options"
		}
	}
	profileName := payload.ProfileName
	if profileName == "" {
		profileName = payload.DefaultProfileName
	}
	context := payload.Context
	if context == nil {
		context = &astmerge.TemplateDestinationContext{}
	}
	defaultStrategy := payload.DefaultStrategy
	if defaultStrategy == "" {
		defaultStrategy = astmerge.TemplateStrategyMerge
	}
	overrides := cloneStrategyOverrides(payload.Overrides)
	if overrides == nil {
		overrides = []astmerge.TemplateStrategyOverride{}
	}
	replacements := cloneStringMap(payload.Replacements)
	if replacements == nil {
		replacements = map[string]string{}
	}
	return SessionRunnerInput{
		RequestKind:     requestKind,
		ProfileName:     profileName,
		Mode:            payload.Mode,
		TemplateRoot:    payload.TemplateRoot,
		DestinationRoot: payload.DestinationRoot,
		Context:         context,
		DefaultStrategy: defaultStrategy,
		Overrides:       overrides,
		Replacements:    replacements,
		AllowedFamilies: cloneStringSlice(payload.AllowedFamilies),
	}
}

func reportSessionRunnerInputOptions(input SessionRunnerInput) map[string]any {
	return map[string]any{
		"mode":             input.Mode,
		"template_root":    input.TemplateRoot,
		"destination_root": input.DestinationRoot,
		"context":          input.Context,
		"default_strategy": defaultTemplateStrategy(input.DefaultStrategy),
		"overrides":        cloneStrategyOverrides(input.Overrides),
		"replacements":     cloneStringMap(input.Replacements),
		"allowed_families": cloneStringSlice(input.AllowedFamilies),
	}
}

func reportSessionRunnerInputOverrides(input SessionRunnerInput) map[string]any {
	overrides := map[string]any{
		"mode":             input.Mode,
		"template_root":    input.TemplateRoot,
		"destination_root": input.DestinationRoot,
	}
	if input.Context != nil && input.Context.ProjectName != "" {
		overrides["context"] = input.Context
	}
	if input.DefaultStrategy != "" {
		overrides["default_strategy"] = input.DefaultStrategy
	}
	if len(input.Overrides) > 0 {
		overrides["overrides"] = cloneStrategyOverrides(input.Overrides)
	}
	if len(input.Replacements) > 0 {
		overrides["replacements"] = cloneStringMap(input.Replacements)
	}
	if input.AllowedFamilies != nil {
		overrides["allowed_families"] = cloneStringSlice(input.AllowedFamilies)
	}
	return overrides
}

func decodeSessionRunnerOptions(raw map[string]any) DirectorySessionOptions {
	options := DirectorySessionOptions{
		DefaultStrategy: astmerge.TemplateStrategyMerge,
	}
	if mode, ok := raw["mode"].(string); ok && mode != "" {
		options.Mode = DirectorySessionMode(mode)
	}
	if templateRoot, ok := raw["template_root"].(string); ok {
		options.TemplateRoot = templateRoot
	}
	if destinationRoot, ok := raw["destination_root"].(string); ok {
		options.DestinationRoot = destinationRoot
	}
	if context, ok := raw["context"]; ok && context != nil {
		options.Context = decodeTemplateDestinationContextMap(context)
	}
	if strategy, ok := raw["default_strategy"].(string); ok && strategy != "" {
		options.DefaultStrategy = astmerge.TemplateStrategy(strategy)
	}
	if overrides, ok := raw["overrides"]; ok && overrides != nil {
		options.Overrides = decodeTemplateStrategyOverrides(overrides)
	}
	if replacements, ok := raw["replacements"]; ok && replacements != nil {
		options.Replacements = decodeStringMap(replacements)
	}
	if allowedFamilies, ok := raw["allowed_families"]; ok {
		options.AllowedFamilies = decodeStringSlice(allowedFamilies)
	}
	return options
}

func decodeTemplateDestinationContextMap(raw any) *astmerge.TemplateDestinationContext {
	section, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	context := &astmerge.TemplateDestinationContext{}
	if projectName, ok := section["project_name"].(string); ok && projectName != "" {
		context.ProjectName = projectName
	}
	if context.ProjectName == "" {
		return &astmerge.TemplateDestinationContext{}
	}
	return context
}

func decodeTemplateStrategyOverrides(raw any) []astmerge.TemplateStrategyOverride {
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	overrides := make([]astmerge.TemplateStrategyOverride, 0, len(items))
	for _, item := range items {
		section, ok := item.(map[string]any)
		if !ok {
			continue
		}
		overrides = append(overrides, astmerge.TemplateStrategyOverride{
			Path:     stringOrAny(section["path"]),
			Strategy: astmerge.TemplateStrategy(stringOrAny(section["strategy"])),
		})
	}
	return overrides
}

func decodeStringMap(raw any) map[string]string {
	section, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	values := make(map[string]string, len(section))
	for key, value := range section {
		values[key] = stringOrAny(value)
	}
	return values
}

func decodeStringSlice(raw any) []string {
	if raw == nil {
		return nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	values := make([]string, 0, len(items))
	for _, item := range items {
		values = append(values, stringOrAny(item))
	}
	return values
}

func stringOrAny(raw any) string {
	if value, ok := raw.(string); ok {
		return value
	}
	return ""
}

func defaultTemplateStrategy(strategy astmerge.TemplateStrategy) astmerge.TemplateStrategy {
	if strategy == "" {
		return astmerge.TemplateStrategyMerge
	}
	return strategy
}

func ResolveTemplateDirectorySessionOptions(
	profiles map[string]DirectorySessionProfile,
	profileName string,
	overrides DirectorySessionOptions,
) (DirectorySessionOptions, bool) {
	profile, ok := profiles[profileName]
	if !ok {
		return DirectorySessionOptions{}, false
	}
	options := DirectorySessionOptions{
		Mode:            profile.Mode,
		TemplateRoot:    overrides.TemplateRoot,
		DestinationRoot: overrides.DestinationRoot,
		Context:         profile.Context,
		DefaultStrategy: profile.DefaultStrategy,
		Overrides:       cloneStrategyOverrides(profile.Overrides),
		Replacements:    cloneStringMap(profile.Replacements),
		AllowedFamilies: cloneStringSlice(profile.AllowedFamilies),
		Config:          profile.Config,
	}
	if overrides.Mode != "" {
		options.Mode = overrides.Mode
	}
	if overrides.Context != nil {
		options.Context = overrides.Context
	}
	if overrides.DefaultStrategy != "" {
		options.DefaultStrategy = overrides.DefaultStrategy
	}
	if overrides.Overrides != nil {
		options.Overrides = overrides.Overrides
	}
	if overrides.Replacements != nil {
		options.Replacements = overrides.Replacements
	}
	if overrides.AllowedFamilies != nil {
		options.AllowedFamilies = overrides.AllowedFamilies
	}
	if overrides.Config != nil {
		options.Config = overrides.Config
	}
	return options, true
}

func RunTemplateDirectorySessionWithProfile(
	profiles map[string]DirectorySessionProfile,
	profileName string,
	overrides DirectorySessionOptions,
) (SessionOutcomeReport, error) {
	request := ReportTemplateDirectorySessionProfileRequest(profiles, profileName, overrides)
	if !request.Ready {
		return reportTemplateDirectorySessionConfigurationOutcome(request.Mode, SessionDiagnosticsReport{
			Mode:        request.Mode,
			Ready:       request.Ready,
			Diagnostics: request.Diagnostics,
		}), nil
	}
	return RunTemplateDirectorySessionWithOptions(*request.ResolvedOptions)
}

func firstNonEmptySessionMode(values ...DirectorySessionMode) DirectorySessionMode {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return DirectorySessionModePlan
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func cloneStringSlice(values []string) []string {
	if values == nil {
		return nil
	}
	cloned := make([]string, len(values))
	copy(cloned, values)
	return cloned
}

func cloneStrategyOverrides(values []astmerge.TemplateStrategyOverride) []astmerge.TemplateStrategyOverride {
	if values == nil {
		return nil
	}
	cloned := make([]astmerge.TemplateStrategyOverride, len(values))
	copy(cloned, values)
	return cloned
}

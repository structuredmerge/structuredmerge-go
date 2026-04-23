package asttemplate

import (
	"slices"

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

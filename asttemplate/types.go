package asttemplate

import (
	"slices"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
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

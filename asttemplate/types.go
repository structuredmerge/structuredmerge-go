package asttemplate

import "github.com/structuredmerge/structuredmerge-go/astmerge"

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

package kettlegomodder

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
)

const PackageName = "kettle-gomodder"

const (
	managedBlockOpen  = "// <<kettle-gomodder:generated>> do not edit below this line"
	managedBlockClose = "// <</kettle-gomodder:generated>>"
)

type PackageFacts struct {
	Package PackageFactGroup `json:"package"`
	GoMod   GoModFactGroup   `json:"gomod"`
}

type PackageFactGroup struct {
	Ecosystem         string `json:"ecosystem"`
	Name              string `json:"name"`
	Slug              string `json:"slug"`
	SourceURL         string `json:"source_url,omitempty"`
	HomepageURL       string `json:"homepage_url,omitempty"`
	LicenseExpression string `json:"license_expression,omitempty"`
	Description       string `json:"description,omitempty"`
}

type GoModFactGroup struct {
	GoModPath  string `json:"go_mod_path"`
	ModulePath string `json:"module_path"`
	GoVersion  string `json:"go_version,omitempty"`
}

type RecipePack struct {
	Name      string            `json:"name"`
	Version   int               `json:"version"`
	Ecosystem string            `json:"ecosystem"`
	Recipes   []PackagingRecipe `json:"recipes"`
}

type PackagingRecipeName string

const (
	RecipeReadmeMetadata      PackagingRecipeName = "readme_metadata"
	RecipeChangelogUnreleased PackagingRecipeName = "changelog_unreleased"
	RecipeGeneratedBlockSync  PackagingRecipeName = "generated_block_sync"
)

type PackagingRecipe struct {
	Name           PackagingRecipeName `json:"name"`
	TargetPath     string              `json:"target_path"`
	ProviderFamily string              `json:"provider_family"`
	Primitive      string              `json:"primitive"`
	Facts          []string            `json:"facts"`
	Selectors      []string            `json:"selectors"`
}

type RecipeRunReport struct {
	RecipeName      PackagingRecipeName                            `json:"recipe_name"`
	RelativePath    string                                         `json:"relative_path"`
	Changed         bool                                           `json:"changed"`
	RequestEnvelope astmerge.ContentRecipeExecutionRequestEnvelope `json:"request_envelope"`
	ReportEnvelope  astmerge.ContentRecipeExecutionReportEnvelope  `json:"report_envelope"`
	FinalContent    string                                         `json:"final_content"`
	Diagnostics     []astmerge.Diagnostic                          `json:"diagnostics"`
}

type ProjectReport struct {
	Mode          string                `json:"mode"`
	Ready         bool                  `json:"ready"`
	Facts         PackageFacts          `json:"facts"`
	RecipePack    RecipePack            `json:"recipe_pack"`
	RecipeReports []RecipeRunReport     `json:"recipe_reports"`
	ChangedFiles  []string              `json:"changed_files"`
	Diagnostics   []astmerge.Diagnostic `json:"diagnostics"`
}

func DiscoverFacts(projectRoot string) (PackageFacts, error) {
	goModPath := filepath.Join(projectRoot, "go.mod")
	source, err := os.ReadFile(goModPath)
	if err != nil {
		return PackageFacts{}, fmt.Errorf("read go.mod: %w", err)
	}
	modulePath, goVersion, err := parseGoMod(string(source))
	if err != nil {
		return PackageFacts{}, err
	}
	sourceURL := sourceURLFromModulePath(modulePath)

	return PackageFacts{
		Package: PackageFactGroup{
			Ecosystem:   "gomod",
			Name:        modulePath,
			Slug:        path.Base(modulePath),
			SourceURL:   sourceURL,
			HomepageURL: sourceURL,
		},
		GoMod: GoModFactGroup{
			GoModPath:  "go.mod",
			ModulePath: modulePath,
			GoVersion:  goVersion,
		},
	}, nil
}

func RecipePackForFacts() RecipePack {
	return RecipePack{
		Name:      "kettle-gomodder-core",
		Version:   1,
		Ecosystem: "gomod",
		Recipes: []PackagingRecipe{
			recipeEntry(
				RecipeReadmeMetadata,
				"README.md",
				"markdown",
				"supplied_readme_metadata_synchronization",
				[]string{"package", "funding", "readme"},
			),
			recipeEntry(
				RecipeChangelogUnreleased,
				"CHANGELOG.md",
				"markdown",
				"changelog_unreleased_normalization",
				[]string{"package", "changelog"},
			),
			recipeEntry(
				RecipeGeneratedBlockSync,
				"internal/generated/packageinfo.go",
				"text",
				"supplied_managed_text_block_replacement",
				[]string{"package", "generated_blocks"},
			),
		},
	}
}

func PlanProject(projectRoot string) (ProjectReport, error) {
	facts, err := DiscoverFacts(projectRoot)
	if err != nil {
		return ProjectReport{}, err
	}
	pack := RecipePackForFacts()
	files, err := readProjectFiles(projectRoot, pack)
	if err != nil {
		return ProjectReport{}, err
	}
	reports := make([]RecipeRunReport, 0, len(pack.Recipes))
	for _, recipe := range pack.Recipes {
		reports = append(reports, executeRecipe(projectRoot, recipe, facts, files))
	}
	changedFiles := make([]string, 0, len(reports))
	for _, report := range reports {
		if report.Changed {
			changedFiles = append(changedFiles, report.RelativePath)
		}
	}
	sort.Strings(changedFiles)

	return ProjectReport{
		Mode:          "plan",
		Ready:         true,
		Facts:         facts,
		RecipePack:    pack,
		RecipeReports: reports,
		ChangedFiles:  changedFiles,
		Diagnostics:   []astmerge.Diagnostic{},
	}, nil
}

func ApplyProject(projectRoot string) (ProjectReport, error) {
	report, err := PlanProject(projectRoot)
	if err != nil {
		return ProjectReport{}, err
	}
	report.Mode = "apply"
	for _, recipeReport := range report.RecipeReports {
		if !recipeReport.Changed {
			continue
		}
		targetPath := filepath.Join(projectRoot, recipeReport.RelativePath)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return ProjectReport{}, fmt.Errorf("create parent directory for %s: %w", recipeReport.RelativePath, err)
		}
		if err := os.WriteFile(targetPath, []byte(recipeReport.FinalContent), 0o644); err != nil {
			return ProjectReport{}, fmt.Errorf("write %s: %w", recipeReport.RelativePath, err)
		}
	}

	return report, nil
}

func ContentRecipeExecutionRequest(request astmerge.ContentRecipeExecutionRequest) astmerge.ContentRecipeExecutionRequest {
	return request
}

func ContentRecipeExecutionRequestEnvelope(request astmerge.ContentRecipeExecutionRequest) astmerge.ContentRecipeExecutionRequestEnvelope {
	return astmerge.ContentRecipeExecutionRequestEnvelope{
		Kind:    "content_recipe_execution_request",
		Version: astmerge.StructuredEditTransportVersion,
		Request: request,
	}
}

func ContentRecipeExecutionReport(report astmerge.ContentRecipeExecutionReport) astmerge.ContentRecipeExecutionReport {
	return report
}

func ContentRecipeExecutionReportEnvelope(report astmerge.ContentRecipeExecutionReport) astmerge.ContentRecipeExecutionReportEnvelope {
	return astmerge.ContentRecipeExecutionReportEnvelope{
		Kind:    "content_recipe_execution_report",
		Version: astmerge.StructuredEditTransportVersion,
		Report:  report,
	}
}

func executeRecipe(projectRoot string, recipe PackagingRecipe, facts PackageFacts, files map[string]string) RecipeRunReport {
	original := files[recipe.TargetPath]
	var finalContent string
	switch recipe.Name {
	case RecipeReadmeMetadata:
		finalContent = synchronizeReadme(original, facts)
	case RecipeChangelogUnreleased:
		finalContent = normalizeChangelog(original)
	case RecipeGeneratedBlockSync:
		finalContent = synchronizeManagedBlock(original, facts)
	}
	request := ContentRecipeExecutionRequest(astmerge.ContentRecipeExecutionRequest{
		RecipeName:         recipe.Primitive,
		RecipeVersion:      "1",
		RelativePath:       recipe.TargetPath,
		ProviderFamily:     recipe.ProviderFamily,
		TemplateContent:    "",
		DestinationContent: original,
		Steps:              []astmerge.ContentRecipeStep{contentRecipeStep(recipe)},
		RuntimeContext:     runtimeContext(facts),
		Metadata: map[string]any{
			"packaging_recipe": string(recipe.Name),
			"project_root":     projectRoot,
		},
	})
	changed := finalContent != original
	stepReport := contentRecipeStepReport(recipe, original, finalContent, changed)
	report := ContentRecipeExecutionReport(astmerge.ContentRecipeExecutionReport{
		Request:      request,
		FinalContent: finalContent,
		Changed:      changed,
		StepReports:  []astmerge.ContentRecipeStepReport{stepReport},
		Diagnostics:  []astmerge.Diagnostic{},
		Metadata: map[string]any{
			"packaging_recipe": string(recipe.Name),
		},
	})

	return RecipeRunReport{
		RecipeName:      recipe.Name,
		RelativePath:    recipe.TargetPath,
		Changed:         changed,
		RequestEnvelope: ContentRecipeExecutionRequestEnvelope(request),
		ReportEnvelope:  ContentRecipeExecutionReportEnvelope(report),
		FinalContent:    finalContent,
		Diagnostics:     []astmerge.Diagnostic{},
	}
}

func contentRecipeStep(recipe PackagingRecipe) astmerge.ContentRecipeStep {
	providerFamily := recipe.ProviderFamily
	return astmerge.ContentRecipeStep{
		StepID:         string(recipe.Name),
		StepKind:       "structured_edit",
		Name:           string(recipe.Name),
		ProviderFamily: &providerFamily,
		Metadata: map[string]any{
			"primitive":   recipe.Primitive,
			"target_path": recipe.TargetPath,
		},
	}
}

func contentRecipeStepReport(recipe PackagingRecipe, original string, finalContent string, changed bool) astmerge.ContentRecipeStepReport {
	operationProfile := astmerge.StructuredEditOperationProfile{
		OperationKind:          recipe.Primitive,
		OperationFamily:        "kettle-gomodder",
		KnownOperationKind:     true,
		SourceRequirement:      "destination_content",
		DestinationRequirement: "relative_path",
		ReplacementSource:      "runtime_context",
		CapturesSourceText:     false,
		SupportsIfMissing:      true,
	}
	result := astmerge.StructuredEditResult{
		OperationKind:    recipe.Primitive,
		UpdatedContent:   finalContent,
		Changed:          changed,
		OperationProfile: operationProfile,
	}
	application := astmerge.StructuredEditApplication{
		Request: astmerge.StructuredEditRequest{
			OperationKind: recipe.Primitive,
			Content:       original,
			SourceLabel:   recipe.TargetPath,
			Metadata: map[string]any{
				"packaging_recipe": string(recipe.Name),
			},
		},
		Result: result,
	}

	status := "unchanged"
	if changed {
		status = "applied"
	}
	return astmerge.ContentRecipeStepReport{
		StepID:        string(recipe.Name),
		StepKind:      recipe.Primitive,
		Status:        status,
		Changed:       changed,
		InputContent:  original,
		OutputContent: finalContent,
		Application:   &application,
		Diagnostics:   []astmerge.Diagnostic{},
		Metadata: map[string]any{
			"target_path": recipe.TargetPath,
		},
	}
}

func synchronizeReadme(content string, facts PackageFacts) string {
	lines := strings.Split(content, "\n")
	heading := "# " + facts.Package.Name
	replaced := false
	for index, line := range lines {
		if strings.HasPrefix(line, "# ") {
			lines[index] = heading
			replaced = true
			break
		}
	}
	if !replaced {
		lines = append([]string{heading, ""}, lines...)
	}
	return replaceMarkdownManagedBlock(strings.Join(lines, "\n"), "kettle-gomodder:metadata", readmeMetadataBlock(facts))
}

func normalizeChangelog(content string) string {
	text := content
	if firstLine := strings.SplitN(text, "\n", 2)[0]; !strings.HasPrefix(firstLine, "# ") {
		text = "# Changelog\n\n" + text
	}
	for _, line := range strings.Split(text, "\n") {
		normalized := strings.ToLower(line)
		if strings.HasPrefix(normalized, "## [unreleased]") || strings.HasPrefix(normalized, "## unreleased") {
			return ensureTrailingNewline(text)
		}
	}

	lines := strings.Split(text, "\n")
	insertAt := len(lines)
	for index, line := range lines {
		if strings.HasPrefix(line, "## ") {
			insertAt = index
			break
		}
	}
	section := []string{"", "## [Unreleased]", "", "### Added", "", "### Changed", "", "### Fixed", ""}
	lines = append(lines[:insertAt], append(section, lines[insertAt:]...)...)
	return ensureTrailingNewline(collapseBlankLines(strings.Join(lines, "\n")))
}

func synchronizeManagedBlock(content string, facts PackageFacts) string {
	replacement := fmt.Sprintf(
		"%s\nconst PackageName = %q\nconst PackageEcosystem = %q\n%s\n",
		managedBlockOpen,
		facts.Package.Name,
		facts.Package.Ecosystem,
		managedBlockClose,
	)
	return replaceTextManagedBlock(content, replacement)
}

func readProjectFiles(projectRoot string, pack RecipePack) (map[string]string, error) {
	files := map[string]string{}
	for _, recipe := range pack.Recipes {
		targetPath := filepath.Join(projectRoot, recipe.TargetPath)
		source, err := os.ReadFile(targetPath)
		if err != nil {
			if os.IsNotExist(err) {
				files[recipe.TargetPath] = ""
				continue
			}
			return nil, fmt.Errorf("read %s: %w", recipe.TargetPath, err)
		}
		files[recipe.TargetPath] = string(source)
	}
	return files, nil
}

func recipeEntry(name PackagingRecipeName, targetPath string, providerFamily string, primitive string, facts []string) PackagingRecipe {
	return PackagingRecipe{
		Name:           name,
		TargetPath:     targetPath,
		ProviderFamily: providerFamily,
		Primitive:      primitive,
		Facts:          facts,
		Selectors:      []string{},
	}
}

func readmeMetadataBlock(facts PackageFacts) string {
	rows := []string{
		"| Field | Value |",
		"|---|---|",
		"| Package | " + facts.Package.Name + " |",
	}
	if facts.Package.Description != "" {
		rows = append(rows, "| Description | "+facts.Package.Description+" |")
	}
	if facts.Package.HomepageURL != "" {
		rows = append(rows, "| Homepage | "+facts.Package.HomepageURL+" |")
	}
	if facts.Package.SourceURL != "" {
		rows = append(rows, "| Source | "+facts.Package.SourceURL+" |")
	}
	if facts.Package.LicenseExpression != "" {
		rows = append(rows, "| License | "+facts.Package.LicenseExpression+" |")
	}
	return strings.Join(append([]string{"<!-- kettle-gomodder:metadata:start -->"}, append(rows, "<!-- kettle-gomodder:metadata:end -->")...), "\n")
}

func replaceMarkdownManagedBlock(content string, marker string, replacement string) string {
	return replaceBetweenMarkers(
		content,
		"<!-- "+marker+":start -->",
		"<!-- "+marker+":end -->",
		replacement,
		func() string { return strings.TrimRight(content, "\n") + "\n\n" + replacement + "\n" },
	)
}

func replaceTextManagedBlock(content string, replacement string) string {
	return replaceBetweenMarkers(
		content,
		managedBlockOpen,
		managedBlockClose,
		replacement,
		func() string {
			trimmed := strings.TrimRight(content, "\n")
			if trimmed == "" {
				return replacement
			}
			return trimmed + "\n" + replacement
		},
	)
}

func replaceBetweenMarkers(content string, openMarker string, closeMarker string, replacement string, fallback func() string) string {
	openIndex := strings.Index(content, openMarker)
	if openIndex < 0 {
		return fallback()
	}
	closeIndex := strings.Index(content[openIndex:], closeMarker)
	if closeIndex < 0 {
		return fallback()
	}
	closeIndex += openIndex
	closeEnd := closeIndex + len(closeMarker)
	if closeEnd < len(content) && content[closeEnd] == '\n' {
		closeEnd++
	}
	return content[:openIndex] + replacement + "\n" + content[closeEnd:]
}

func runtimeContext(facts PackageFacts) map[string]any {
	return map[string]any{
		"package": facts.Package,
		"gomod":   facts.GoMod,
	}
}

var (
	moduleLinePattern = regexp.MustCompile(`(?m)^\s*module\s+(\S+)\s*$`)
	goLinePattern     = regexp.MustCompile(`(?m)^\s*go\s+(\S+)\s*$`)
)

func parseGoMod(source string) (string, string, error) {
	moduleMatch := moduleLinePattern.FindStringSubmatch(source)
	if len(moduleMatch) != 2 {
		return "", "", fmt.Errorf("go.mod missing module directive")
	}
	goVersion := ""
	goMatch := goLinePattern.FindStringSubmatch(source)
	if len(goMatch) == 2 {
		goVersion = goMatch[1]
	}
	return moduleMatch[1], goVersion, nil
}

func sourceURLFromModulePath(modulePath string) string {
	parts := strings.Split(modulePath, "/")
	if len(parts) >= 3 && parts[0] == "github.com" {
		return "https://" + strings.Join(parts[:3], "/")
	}
	return ""
}

func ensureTrailingNewline(text string) string {
	if strings.HasSuffix(text, "\n") {
		return text
	}
	return text + "\n"
}

func collapseBlankLines(text string) string {
	lines := strings.Split(text, "\n")
	collapsed := make([]string, 0, len(lines))
	previousBlank := false
	for _, line := range lines {
		blank := line == ""
		if blank && previousBlank {
			continue
		}
		collapsed = append(collapsed, line)
		previousBlank = blank
	}
	return strings.TrimRight(strings.Join(collapsed, "\n"), "\n")
}

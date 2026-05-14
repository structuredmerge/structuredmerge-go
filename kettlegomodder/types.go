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
	"gopkg.in/yaml.v3"
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
	RecipeTemplateApplication PackagingRecipeName = "template_source_application"
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

type ReadmeStyleReport struct {
	ReadmePath           string   `json:"readme_path"`
	Changed              bool     `json:"changed"`
	Style                string   `json:"style"`
	PreservedSections    []string `json:"preserved_sections"`
	RenderedSections     []string `json:"rendered_sections"`
	OmittedSections      []string `json:"omitted_sections"`
	MissingIntegrations  []string `json:"missing_integrations"`
	DisabledIntegrations []string `json:"disabled_integrations"`
	UnresolvedLogoSlugs  []string `json:"unresolved_logo_slugs"`
	LicenseFilesChanged  bool     `json:"license_files_changed"`
	CopyrightAuthors     []string `json:"copyright_authors"`
	FinalContent         string   `json:"final_content"`
}

type KettleConfig struct {
	Readme ReadmeConfig `yaml:"readme"`
}

type ReadmeConfig struct {
	Style            string                  `yaml:"style"`
	ProjectEmoji     string                  `yaml:"project_emoji"`
	Family           ReadmeFamilyConfig      `yaml:"family"`
	LogoRow          ReadmeLogoRowConfig     `yaml:"logo_row"`
	Badges           ReadmeBadgesConfig      `yaml:"badges"`
	PreserveSections []string                `yaml:"preserve_sections"`
	PreservePatterns []string                `yaml:"preserve_patterns"`
	SectionAliases   map[string]string       `yaml:"section_aliases"`
	Conditional      ReadmeConditionalConfig `yaml:"conditional_sections"`
	Integrations     map[string]string       `yaml:"integrations"`
	License          ReadmeLicenseConfig     `yaml:"license"`
}

type ReadmeFamilyConfig struct {
	Enabled            bool   `yaml:"enabled"`
	Name               string `yaml:"name"`
	InjectFromSynopsis bool   `yaml:"inject_from_synopsis"`
}

type ReadmeLogoRowConfig struct {
	Enabled  *bool        `yaml:"enabled"`
	MaxCount int          `yaml:"max_count"`
	Logos    []ReadmeLogo `yaml:"logos"`
}

type ReadmeLogo struct {
	Type string `yaml:"type"`
	Slug string `yaml:"slug"`
	Alt  string `yaml:"alt"`
	Href string `yaml:"href"`
}

type ReadmeBadgesConfig struct {
	VisibleCore     []string `yaml:"visible_core"`
	CollapsedGroups []string `yaml:"collapsed_groups"`
	Disabled        []string `yaml:"disabled"`
}

type ReadmeConditionalConfig struct {
	FlossFunding            string `yaml:"floss_funding"`
	Security                string `yaml:"security"`
	HostileRubyGemsTakeover string `yaml:"hostile_rubygems_takeover"`
	SecureInstallation      string `yaml:"secure_installation"`
}

type ReadmeLicenseConfig struct {
	SPDX                       []string `yaml:"spdx"`
	GenerateLicenseFiles       bool     `yaml:"generate_license_files"`
	IncludeGitBlameAuthors     bool     `yaml:"include_git_blame_authors"`
	SyncReadmeCopyrightAuthors bool     `yaml:"sync_readme_copyright_authors"`
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

func PackagedTemplateInventoryPack() RecipePack {
	return RecipePack{
		Name:      "kettle-gomodder-packaged-template-inventory",
		Version:   1,
		Ecosystem: "gomod",
		Recipes: []PackagingRecipe{
			templateRecipe(".editorconfig"),
			templateRecipe(".github/workflows/ci.yml"),
			templateRecipe(".gitignore"),
			templateRecipe(".golangci.yml"),
			templateRecipe("README.md"),
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
	changedFiles := changedFilesForReports(reports)

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

func PlanPackagedTemplateInventory(projectRoot string) (ProjectReport, error) {
	facts, err := DiscoverFacts(projectRoot)
	if err != nil {
		return ProjectReport{}, err
	}
	pack := PackagedTemplateInventoryPack()
	files, err := readProjectFiles(projectRoot, pack)
	if err != nil {
		return ProjectReport{}, err
	}
	reports := make([]RecipeRunReport, 0, len(pack.Recipes))
	for _, recipe := range pack.Recipes {
		reports = append(reports, executeRecipe(projectRoot, recipe, facts, files))
	}

	return ProjectReport{
		Mode:          "plan",
		Ready:         true,
		Facts:         facts,
		RecipePack:    pack,
		RecipeReports: reports,
		ChangedFiles:  changedFilesForReports(reports),
		Diagnostics:   []astmerge.Diagnostic{},
	}, nil
}

func ApplyPackagedTemplateInventory(projectRoot string) (ProjectReport, error) {
	report, err := PlanPackagedTemplateInventory(projectRoot)
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

func PlanReadmeStyle(projectRoot string) (ReadmeStyleReport, error) {
	facts, err := DiscoverFacts(projectRoot)
	if err != nil {
		return ReadmeStyleReport{}, err
	}
	config, err := readKettleConfig(projectRoot)
	if err != nil {
		return ReadmeStyleReport{}, err
	}
	readmePath := filepath.Join(projectRoot, "README.md")
	original, err := os.ReadFile(readmePath)
	if err != nil && !os.IsNotExist(err) {
		return ReadmeStyleReport{}, fmt.Errorf("read README.md: %w", err)
	}
	hasSecurity := fileExists(filepath.Join(projectRoot, "SECURITY.md"))
	report := renderReadmeStyle(string(original), facts, config.Readme, hasSecurity)
	return report, nil
}

func ApplyReadmeStyle(projectRoot string) (ReadmeStyleReport, error) {
	report, err := PlanReadmeStyle(projectRoot)
	if err != nil {
		return ReadmeStyleReport{}, err
	}
	if !report.Changed {
		return report, nil
	}
	targetPath := filepath.Join(projectRoot, report.ReadmePath)
	if err := os.WriteFile(targetPath, []byte(report.FinalContent), 0o644); err != nil {
		return ReadmeStyleReport{}, fmt.Errorf("write README.md: %w", err)
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
	case RecipeTemplateApplication:
		finalContent = renderPackagedTemplate(recipe.TargetPath, facts)
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

func templateRecipe(targetPath string) PackagingRecipe {
	return recipeEntry(
		RecipeTemplateApplication,
		targetPath,
		"text",
		"supplied_template_source_application",
		[]string{"package", "templates"},
	)
}

func changedFilesForReports(reports []RecipeRunReport) []string {
	changedFiles := make([]string, 0, len(reports))
	for _, report := range reports {
		if report.Changed {
			changedFiles = append(changedFiles, report.RelativePath)
		}
	}
	sort.Strings(changedFiles)
	return changedFiles
}

func renderPackagedTemplate(targetPath string, facts PackageFacts) string {
	template := packagedTemplateContent(targetPath)
	replacements := map[string]string{
		"{{PACKAGE_NAME}}": facts.Package.Name,
		"{{GO_VERSION}}":   facts.GoMod.GoVersion,
	}
	for token, value := range replacements {
		template = strings.ReplaceAll(template, token, value)
	}
	return template
}

func packagedTemplateContent(targetPath string) string {
	switch targetPath {
	case ".editorconfig":
		return "root = true\n\n[*]\ncharset = utf-8\nend_of_line = lf\ninsert_final_newline = true\ntrim_trailing_whitespace = true\n"
	case ".github/workflows/ci.yml":
		return "name: CI\n\non:\n  push:\n  pull_request:\n\njobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v4\n      - uses: actions/setup-go@v5\n        with:\n          go-version: \"{{GO_VERSION}}\"\n      - run: go test ./...\n"
	case ".gitignore":
		return "bin/\ncoverage/\ndist/\n"
	case ".golangci.yml":
		return "run:\n  timeout: 5m\nlinters:\n  enable:\n    - gofmt\n    - govet\n"
	case "README.md":
		return "# {{PACKAGE_NAME}}\n\n## Synopsis\n\n## Installation\n\n```sh\ngo get {{PACKAGE_NAME}}\n```\n\n## Configuration\n\n## Basic Usage\n"
	default:
		return ""
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

func readKettleConfig(projectRoot string) (KettleConfig, error) {
	source, err := os.ReadFile(filepath.Join(projectRoot, "kettle.yml"))
	if err != nil {
		if os.IsNotExist(err) {
			return KettleConfig{}, nil
		}
		return KettleConfig{}, fmt.Errorf("read kettle.yml: %w", err)
	}
	var config KettleConfig
	if err := yaml.Unmarshal(source, &config); err != nil {
		return KettleConfig{}, fmt.Errorf("parse kettle.yml: %w", err)
	}
	return config, nil
}

func renderReadmeStyle(destination string, facts PackageFacts, config ReadmeConfig, hasSecurity bool) ReadmeStyleReport {
	config = defaultReadmeConfig(config)
	preserved := preservedReadmeSections(destination, config)
	license := readmeLicense(config, facts)
	logoRow, unresolvedLogos := readmeLogoRow(config)
	badgeCloud, missingIntegrations, disabledIntegrations := readmeBadgeCloud(config, facts, license, hasSecurity)
	renderedSections := []string{"Project Name", "Badges", "Synopsis", "Info you can shake a stick at", "Installation", "Configuration", "Basic Usage", "Versioning", "License", "A request for help"}
	omittedSections := []string{"Hostile RubyGems Takeover", "Secure Installation"}
	if logoRow != "" {
		renderedSections = append([]string{"Logos"}, renderedSections...)
	}
	includeFunding := shouldIncludeFunding(config, license)
	if includeFunding {
		renderedSections = append(renderedSections, "FLOSS Funding")
	} else {
		omittedSections = append(omittedSections, "FLOSS Funding")
	}
	if hasSecurity {
		renderedSections = append(renderedSections, "Security")
	} else {
		omittedSections = append(omittedSections, "Security")
	}
	renderedSections = append(renderedSections, "Contributing")

	sections := []string{}
	if logoRow != "" {
		sections = append(sections, logoRow)
	}
	sections = append(sections, "# "+config.ProjectEmoji+" "+facts.Package.Name)
	if badgeCloud != "" {
		sections = append(sections, badgeCloud)
	}
	sections = append(sections,
		"## 🌻 Synopsis\n\n"+preserved["synopsis"],
		"## 💡 Info you can shake a stick at\n\nCompatible with Go "+emptyAsUnknown(facts.GoMod.GoVersion)+".\n\n"+readmeFamilyIntroAndBackendMatrix(),
		"## ✨ Installation\n\n```console\ngo get "+facts.GoMod.ModulePath+"\n```",
		"## ⚙️ Configuration\n\n"+preserved["configuration"],
		"## 🔧 Basic Usage\n\n"+preserved["basic usage"],
	)
	if includeFunding {
		sections = append(sections, "## 🦷 FLOSS Funding\n\nThis free software project accepts funding support when configured by the package maintainer.")
	}
	if hasSecurity {
		sections = append(sections, "## 🔐 Security\n\nSee [SECURITY.md](SECURITY.md).")
	}
	sections = append(sections,
		"## 🤝 Contributing\n\nContributions are welcome. Missing optional service integrations are reported by the generator instead of rendered as broken badges.",
		"## 📌 Versioning\n\nThis project follows semantic versioning for its public API where practical.",
		"## 📄 License\n\n"+licenseParagraph(license),
		"## 🤑 A request for help\n\nPlease support the project by using it, reporting issues, and contributing improvements.",
	)
	finalContent := ensureTrailingNewline(strings.Join(sections, "\n\n"))
	return ReadmeStyleReport{
		ReadmePath:           "README.md",
		Changed:              finalContent != destination,
		Style:                config.Style,
		PreservedSections:    []string{"Synopsis", "Configuration", "Basic Usage"},
		RenderedSections:     renderedSections,
		OmittedSections:      omittedSections,
		MissingIntegrations:  missingIntegrations,
		DisabledIntegrations: disabledIntegrations,
		UnresolvedLogoSlugs:  unresolvedLogos,
		LicenseFilesChanged:  false,
		CopyrightAuthors:     []string{},
		FinalContent:         finalContent,
	}
}

func readmeFamilyIntroAndBackendMatrix() string {
	return strings.Join([]string{
		"<details markdown=\"1\">",
		"<summary>StructuredMerge package family and backend compatibility</summary>",
		"",
		"StructuredMerge packages provide fixture-backed merge behavior for document, configuration, source, archive, and binary formats. Shared contracts live in fixtures, while Go, Ruby, Rust, and TypeScript packages expose language-native APIs over the same behavior.",
		"",
		"| Package | Layer | Families | Status | README role |",
		"|---|---|---|---|---|",
		"| ast-template | workflow | template, readme | active | applies shared templates, package README sections, and package-directory sync workflows |",
		"| ast-merge | core | template, review, structured-edit | active | documents provider-neutral contracts, token resolution, review state, and execution reports |",
		"| tree-haver | backend substrate | parser, backend | active | documents backend selection, language-pack integration, position data, and capability reporting |",
		"| markdown-merge | family | markdown | active | documents Markdown heading, fenced-code, nested-family, and provider behavior |",
		"| json-merge | family | json, jsonc | active | documents JSON and JSONC merge behavior; old jsonc-merge is superseded |",
		"| toml-merge | family | toml | active | documents TOML table, value, parser, and backend behavior |",
		"| yaml-merge | family | yaml | active | documents YAML mapping, sequence, scalar, and backend behavior |",
		"| ruby-merge | family | ruby-source | active | documents Ruby source merge behavior; old prism-merge is backend/provider prior art |",
		"| zip-merge | family | zip, archive | active | documents ZIP member planning and raw-preservation behavior |",
		"| binary-merge | family | binary | active | documents binary preservation and diagnostics behavior |",
		"",
		"JSONC migration note: JSONC is handled by `json-merge` as the `jsonc` dialect. The old `jsonc-merge` package name is superseded in the cross-language toolset; only Ruby may grow a legacy `require \"jsonc/merge\"` wrapper if packaging compatibility requires it. Current fixture-backed JSONC claims are parse support and comment-neutral owner structure; comment-preserving merge output, freeze blocks, and JSONC emitter behavior need dedicated fixtures before they appear in package examples.",
		"",
		"| Backend | Languages | Families | Note |",
		"|---|---|---|---|",
		"| tree-sitter-language-pack | Go, Ruby, Rust, TypeScript | markdown, toml, yaml, source | Preferred cross-language parser substrate where a family has language-pack support. |",
		"| native ecosystem parser | Ruby | ruby, yaml, markdown, toml | Backend-specific Ruby packages are provider prior art or adapters, not the source schema. |",
		"| plain structured text | Go, Ruby, Rust, TypeScript | plain, binary, zip | Families without parser requirements document preservation, byte ranges, archive members, and diagnostics. |",
		"",
		"| Compatibility claim | Current disposition | Fixture source |",
		"|---|---|---|",
		"| Old Ruby runtime backend tables | Prior art only; not a cross-language support promise | slice-741 backend/platform reconciliation |",
		"| tree-sitter-language-pack | Current portable parser substrate for Go, Ruby, Rust, and TypeScript | slices 122, 135, 171, 195, 215 |",
		"| Native parser/adaptor backends | Implementation-specific providers documented through family fixtures | slices 122 and 183 |",
		"| bash-merge, dotenv-merge, rbs-merge | Excluded from generated support tables until explicit scope decisions exist | slice-741 unresolved package list |",
		"",
		"| Reusable example | README role | Source fixture |",
		"|---|---|---|",
		"| Freeze tokens | Show how destination-owned regions are preserved without filling project-specific usage sections | slice-743 reusable README configuration examples |",
		"| Match preference | Summarize template-wins and destination-wins conflict choices through current policy vocabulary | slice-743 reusable README configuration examples |",
		"| Template-only behavior | Explain accept/skip handling for unmatched template entries | slice-743 reusable README configuration examples |",
		"| Debug report inspection | Point users to structured reports and diagnostics instead of ad hoc debug prose | slice-743 reusable README configuration examples |",
		"| Backend selection | Describe portable backend selection without old Ruby runtime support tables | slice-743 reusable README configuration examples |",
		"| Package-directory README command | Document plan/apply/convergence workflow for shared README updates | slice-743 reusable README configuration examples |",
		"",
		"</details>",
	}, "\n")
}

func defaultReadmeConfig(config ReadmeConfig) ReadmeConfig {
	if config.Style == "" {
		config.Style = "thin"
	}
	if config.ProjectEmoji == "" {
		config.ProjectEmoji = "💎"
	}
	if len(config.PreserveSections) == 0 {
		config.PreserveSections = []string{"Synopsis", "Configuration", "Basic Usage"}
	}
	if config.SectionAliases == nil {
		config.SectionAliases = map[string]string{}
	}
	defaultAliases := map[string]string{
		"summary":               "synopsis",
		"usage":                 "basic usage",
		"configuration options": "configuration",
		"setup":                 "basic usage",
	}
	for key, value := range defaultAliases {
		if _, ok := config.SectionAliases[key]; !ok {
			config.SectionAliases[key] = value
		}
	}
	if config.LogoRow.MaxCount == 0 {
		config.LogoRow.MaxCount = 3
	}
	return config
}

func preservedReadmeSections(content string, config ReadmeConfig) map[string]string {
	sections := markdownSectionBodies(content)
	result := map[string]string{}
	for _, section := range config.PreserveSections {
		key := normalizeReadmeHeading(section)
		result[key] = strings.TrimSpace(sections[key])
	}
	for from, to := range normalizedAliases(config.SectionAliases) {
		if result[to] == "" && sections[from] != "" {
			result[to] = strings.TrimSpace(sections[from])
		}
	}
	return result
}

func markdownSectionBodies(content string) map[string]string {
	lines := strings.Split(content, "\n")
	type heading struct {
		index int
		level int
		key   string
	}
	headings := []heading{}
	for index, line := range lines {
		if !strings.HasPrefix(line, "#") {
			continue
		}
		markerEnd := 0
		for markerEnd < len(line) && line[markerEnd] == '#' {
			markerEnd++
		}
		if markerEnd == 0 || markerEnd > 6 || markerEnd >= len(line) || line[markerEnd] != ' ' {
			continue
		}
		headings = append(headings, heading{index: index, level: markerEnd, key: normalizeReadmeHeading(line[markerEnd+1:])})
	}
	result := map[string]string{}
	for index, current := range headings {
		end := len(lines)
		for _, candidate := range headings[index+1:] {
			if candidate.level <= current.level {
				end = candidate.index
				break
			}
		}
		result[current.key] = strings.TrimSpace(strings.Join(lines[current.index+1:end], "\n"))
	}
	return result
}

func normalizedAliases(aliases map[string]string) map[string]string {
	result := map[string]string{}
	for key, value := range aliases {
		result[normalizeReadmeHeading(key)] = normalizeReadmeHeading(value)
	}
	return result
}

func normalizeReadmeHeading(value string) string {
	value = strings.TrimSpace(value)
	fields := strings.Fields(value)
	if len(fields) > 1 && !startsWithLetterOrDigit(fields[0]) {
		value = strings.Join(fields[1:], " ")
	}
	return strings.ToLower(strings.TrimSpace(value))
}

func startsWithLetterOrDigit(value string) bool {
	if value == "" {
		return false
	}
	first := []rune(value)[0]
	return (first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || (first >= '0' && first <= '9')
}

func readmeLicense(config ReadmeConfig, facts PackageFacts) string {
	if len(config.License.SPDX) > 0 {
		return strings.Join(config.License.SPDX, " OR ")
	}
	if facts.Package.LicenseExpression != "" {
		return facts.Package.LicenseExpression
	}
	return "MIT"
}

func readmeLogoRow(config ReadmeConfig) (string, []string) {
	if config.LogoRow.Enabled != nil && !*config.LogoRow.Enabled {
		return "", []string{}
	}
	maxCount := config.LogoRow.MaxCount
	if maxCount <= 0 || maxCount > 3 {
		maxCount = 3
	}
	logos := config.LogoRow.Logos
	if len(logos) > maxCount {
		logos = logos[:maxCount]
	}
	parts := []string{}
	unresolved := []string{}
	for _, logo := range logos {
		logoType := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(logo.Type)), "-", "_")
		if !containsString([]string{"language", "org", "project", "affiliated_project"}, logoType) {
			unresolved = append(unresolved, logo.Slug)
			continue
		}
		if strings.TrimSpace(logo.Slug) == "" {
			unresolved = append(unresolved, logo.Slug)
			continue
		}
		slug := strings.TrimSpace(logo.Slug)
		ref := strings.ReplaceAll(slug, "/", "-")
		alt := strings.TrimSpace(logo.Alt)
		if alt == "" {
			alt = slug
		}
		href := strings.TrimSpace(logo.Href)
		if href == "" {
			href = "https://logos.galtzo.com/assets/images/" + slug + "/"
		}
		parts = append(parts, fmt.Sprintf("[![%s][🖼️%s-i]][🖼️%s]\n[🖼️%s-i]: https://logos.galtzo.com/assets/images/%s/avatar-192px.svg\n[🖼️%s]: %s", alt, ref, ref, ref, slug, ref, href))
	}
	return strings.Join(parts, "\n"), unresolved
}

func readmeBadgeCloud(config ReadmeConfig, facts PackageFacts, license string, hasSecurity bool) (string, []string, []string) {
	disabled := append([]string{}, config.Badges.Disabled...)
	missing := []string{}
	for _, integration := range []string{"codecov", "coveralls", "qlty", "codeql"} {
		if containsString(disabled, integration) {
			continue
		}
		missing = append(missing, integration)
	}
	badges := []string{}
	if facts.Package.SourceURL != "" {
		badges = append(badges, "[![Source](https://img.shields.io/badge/source-github-238636.svg)]("+facts.Package.SourceURL+")")
	}
	if license != "" {
		badges = append(badges, "![License](https://img.shields.io/badge/license-"+strings.ReplaceAll(license, " ", "%20")+"-259D6C.svg)")
	}
	if hasSecurity {
		badges = append(badges, "[![Security](https://img.shields.io/badge/security-policy-259D6C.svg)](SECURITY.md)")
	}
	return strings.Join(badges, " "), missing, disabled
}

func shouldIncludeFunding(config ReadmeConfig, license string) bool {
	policy := strings.ToLower(strings.TrimSpace(config.Conditional.FlossFunding))
	if policy == "disabled" || policy == "false" || policy == "never" {
		return false
	}
	if policy == "enabled" || policy == "true" || policy == "always" {
		return true
	}
	return license == "MIT"
}

func licenseParagraph(license string) string {
	if license == "MIT" {
		return "This project is made available under the terms of the MIT License."
	}
	return "This project is made available under the following license expression: " + license + "."
}

func emptyAsUnknown(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unknown"
	}
	return value
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
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

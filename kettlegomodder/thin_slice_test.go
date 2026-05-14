package kettlegomodder

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type thinSliceFixture struct {
	CaseID    string `json:"case_id"`
	Ecosystem string `json:"ecosystem"`
	Inputs    struct {
		Files map[string]string `json:"files"`
	} `json:"inputs"`
	Expected struct {
		Facts        map[string]any    `json:"facts"`
		ChangedFiles []string          `json:"changed_files"`
		Files        map[string]string `json:"files"`
	} `json:"expected"`
}

type thinSliceContract struct {
	CanonicalRecipes []struct {
		Name string `json:"name"`
	} `json:"canonical_recipes"`
	RequiredFactGroups  []string          `json:"required_fact_groups"`
	EcosystemFactGroups map[string]string `json:"ecosystem_fact_groups"`
	ReportContract      struct {
		RequestEnvelopeKind string `json:"request_envelope_kind"`
		ReportEnvelopeKind  string `json:"report_envelope_kind"`
	} `json:"report_contract"`
	ValidatedEcosystems []string `json:"validated_ecosystems"`
}

func TestThinSlice(t *testing.T) {
	fixture := readJSONFixture[thinSliceFixture](t, "fixtures", "thin-slice.json")
	contract := readJSONFixture[thinSliceContract](t, "..", "..", "fixtures", "packaging", "thin-slice-contract.json")
	projectRoot := t.TempDir()
	writeTree(t, projectRoot, fixture.Inputs.Files)

	expectedRecipeNames := make([]string, 0, len(contract.CanonicalRecipes))
	for _, recipe := range contract.CanonicalRecipes {
		expectedRecipeNames = append(expectedRecipeNames, recipe.Name)
	}
	if !strings.HasPrefix(fixture.CaseID, "kettle-gomodder-vnext-") {
		t.Fatalf("unexpected case id: %s", fixture.CaseID)
	}
	if !containsString(contract.ValidatedEcosystems, fixture.Ecosystem) {
		t.Fatalf("contract does not validate ecosystem %q", fixture.Ecosystem)
	}
	for _, group := range contract.RequiredFactGroups {
		if _, ok := fixture.Expected.Facts[group]; !ok {
			t.Fatalf("expected facts missing required group %q", group)
		}
	}
	if _, ok := fixture.Expected.Facts[contract.EcosystemFactGroups[fixture.Ecosystem]]; !ok {
		t.Fatalf("expected facts missing ecosystem group %q", contract.EcosystemFactGroups[fixture.Ecosystem])
	}

	plan, err := PlanProject(projectRoot)
	if err != nil {
		t.Fatalf("plan project: %v", err)
	}
	if actual := jsonReady(t, plan.Facts); !reflect.DeepEqual(actual, fixture.Expected.Facts) {
		t.Fatalf("unexpected facts:\nactual: %#v\nexpected: %#v", actual, fixture.Expected.Facts)
	}
	actualRecipeNames := make([]string, 0, len(plan.RecipePack.Recipes))
	for _, recipe := range plan.RecipePack.Recipes {
		actualRecipeNames = append(actualRecipeNames, string(recipe.Name))
	}
	if !reflect.DeepEqual(actualRecipeNames, expectedRecipeNames) {
		t.Fatalf("unexpected recipe names: %#v", actualRecipeNames)
	}
	if !reflect.DeepEqual(plan.ChangedFiles, fixture.Expected.ChangedFiles) {
		t.Fatalf("unexpected changed files: %#v", plan.ChangedFiles)
	}
	if actual := uniqueRequestKinds(plan.RecipeReports); !reflect.DeepEqual(actual, []string{contract.ReportContract.RequestEnvelopeKind}) {
		t.Fatalf("unexpected request envelope kinds: %#v", actual)
	}
	if actual := uniqueReportKinds(plan.RecipeReports); !reflect.DeepEqual(actual, []string{contract.ReportContract.ReportEnvelopeKind}) {
		t.Fatalf("unexpected report envelope kinds: %#v", actual)
	}

	apply, err := ApplyProject(projectRoot)
	if err != nil {
		t.Fatalf("apply project: %v", err)
	}
	if !reflect.DeepEqual(apply.ChangedFiles, fixture.Expected.ChangedFiles) {
		t.Fatalf("unexpected apply changed files: %#v", apply.ChangedFiles)
	}
	if actual := readExpectedProjectFiles(t, projectRoot, fixture.Expected.Files); !reflect.DeepEqual(actual, fixture.Expected.Files) {
		t.Fatalf("unexpected project files:\nactual: %#v\nexpected: %#v", actual, fixture.Expected.Files)
	}
}

func TestPackagedTemplateInventory(t *testing.T) {
	projectRoot := t.TempDir()
	writeTree(t, projectRoot, map[string]string{
		"go.mod": "module github.com/acme/widget\n\ngo 1.22\n",
	})

	plan, err := PlanPackagedTemplateInventory(projectRoot)
	if err != nil {
		t.Fatalf("plan packaged template inventory: %v", err)
	}
	expectedChanged := []string{".editorconfig", ".github/workflows/ci.yml", ".gitignore", ".golangci.yml", "README.md"}
	if !reflect.DeepEqual(plan.ChangedFiles, expectedChanged) {
		t.Fatalf("unexpected planned template files: %#v", plan.ChangedFiles)
	}
	if got := plan.RecipePack.Name; got != "kettle-gomodder-packaged-template-inventory" {
		t.Fatalf("unexpected recipe pack name: %s", got)
	}

	apply, err := ApplyPackagedTemplateInventory(projectRoot)
	if err != nil {
		t.Fatalf("apply packaged template inventory: %v", err)
	}
	if !reflect.DeepEqual(apply.ChangedFiles, expectedChanged) {
		t.Fatalf("unexpected applied template files: %#v", apply.ChangedFiles)
	}
	ci := mustReadProjectFile(t, projectRoot, ".github/workflows/ci.yml")
	if !strings.Contains(ci, "go-version: \"1.22\"") {
		t.Fatalf("expected CI template to render go version, got:\n%s", ci)
	}
	readme := mustReadProjectFile(t, projectRoot, "README.md")
	for _, snippet := range []string{"# github.com/acme/widget", "## Synopsis", "## Installation", "go get github.com/acme/widget", "## Configuration", "## Basic Usage"} {
		if !strings.Contains(readme, snippet) {
			t.Fatalf("expected README template to include %q, got:\n%s", snippet, readme)
		}
	}

	second, err := ApplyPackagedTemplateInventory(projectRoot)
	if err != nil {
		t.Fatalf("reapply packaged template inventory: %v", err)
	}
	if len(second.ChangedFiles) != 0 {
		t.Fatalf("expected packaged template reapply to be stable, changed: %#v", second.ChangedFiles)
	}
	if got := mustReadProjectFile(t, projectRoot, ".github/workflows/ci.yml"); got != ci {
		t.Fatalf("expected CI template to remain stable on reapply")
	}
	if got := mustReadProjectFile(t, projectRoot, "README.md"); got != readme {
		t.Fatalf("expected README template to remain stable on reapply")
	}
}

func TestReadmeStyleProfileConformance(t *testing.T) {
	styleFixture := readJSONFixture[map[string]any](t, "..", "..", "fixtures", "diagnostics", "slice-740-kettle-readme-style-profile", "kettle-readme-style-profile.json")
	profile := styleFixture["profile"].(map[string]any)
	if profile["name"] != "kettle-readme-style-profile" {
		t.Fatalf("unexpected README style profile fixture: %#v", profile["name"])
	}
	projectRoot := t.TempDir()
	writeTree(t, projectRoot, map[string]string{
		"go.mod": "module github.com/acme/widget\n\ngo 1.22\n",
		"kettle.yml": strings.Join([]string{
			"readme:",
			"  style: thin",
			"  project_emoji: \"🧪\"",
			"  logo_row:",
			"    enabled: true",
			"    max_count: 3",
			"    logos:",
			"      - type: language",
			"        slug: go-lang",
			"        alt: Go language logo",
			"      - type: org",
			"        slug: acme",
			"        alt: Acme org logo",
			"      - type: affiliated_project",
			"        slug: tree-sitter/tree-sitter",
			"        alt: Tree-sitter project logo",
			"      - type: project",
			"        slug: acme/ignored",
			"        alt: Ignored fourth logo",
			"  preserve_sections:",
			"    - Synopsis",
			"    - Configuration",
			"    - Basic Usage",
			"  section_aliases:",
			"    Usage: Basic Usage",
			"  conditional_sections:",
			"    floss_funding: default_for_mit_opt_in_otherwise",
			"    security: include_when_security_md_exists",
			"  integrations:",
			"    codecov: auto_detect_or_report_missing",
			"    coveralls: auto_detect_or_report_missing",
			"    qlty: auto_detect_or_report_missing",
			"    codeql: auto_detect_or_report_missing",
			"  badges:",
			"    disabled:",
			"      - coveralls",
			"  license:",
			"    spdx:",
			"      - MIT",
			"",
		}, "\n"),
		"SECURITY.md": "# Security\n",
		"README.md": strings.Join([]string{
			"# Old Widget",
			"",
			"## Summary",
			"",
			"Destination synopsis.",
			"",
			"## Configuration",
			"",
			"Destination configuration.",
			"",
			"## Usage",
			"",
			"Destination usage.",
			"",
		}, "\n"),
	})

	plan, err := PlanReadmeStyle(projectRoot)
	if err != nil {
		t.Fatalf("plan README style: %v", err)
	}
	if !plan.Changed {
		t.Fatalf("expected README style plan to change the destination")
	}
	if plan.Style != "thin" {
		t.Fatalf("unexpected README style: %s", plan.Style)
	}
	for _, section := range []string{"Synopsis", "Configuration", "Basic Usage"} {
		if !containsString(plan.PreservedSections, section) {
			t.Fatalf("expected preserved section %q in %#v", section, plan.PreservedSections)
		}
	}
	for _, section := range []string{"Logos", "Project Name", "Badges", "Synopsis", "Installation", "Configuration", "Basic Usage", "FLOSS Funding", "Security", "Contributing", "Versioning", "License", "A request for help"} {
		if !containsString(plan.RenderedSections, section) {
			t.Fatalf("expected rendered section %q in %#v", section, plan.RenderedSections)
		}
	}
	for _, section := range []string{"Hostile RubyGems Takeover", "Secure Installation"} {
		if !containsString(plan.OmittedSections, section) {
			t.Fatalf("expected omitted section %q in %#v", section, plan.OmittedSections)
		}
	}
	if !containsString(plan.MissingIntegrations, "codecov") || !containsString(plan.MissingIntegrations, "qlty") || containsString(plan.MissingIntegrations, "coveralls") {
		t.Fatalf("unexpected missing integrations: %#v", plan.MissingIntegrations)
	}
	if !containsString(plan.DisabledIntegrations, "coveralls") {
		t.Fatalf("expected coveralls to be disabled: %#v", plan.DisabledIntegrations)
	}
	if strings.Contains(plan.FinalContent, "Ignored fourth logo") {
		t.Fatalf("expected fourth logo to be omitted")
	}
	for _, snippet := range []string{
		"# 🧪 github.com/acme/widget",
		"## 🌻 Synopsis\n\nDestination synopsis.",
		"## ⚙️ Configuration\n\nDestination configuration.",
		"## 🔧 Basic Usage\n\nDestination usage.",
		"## 🔐 Security\n\nSee [SECURITY.md](SECURITY.md).",
		"## 🦷 FLOSS Funding",
		"go get github.com/acme/widget",
		"https://logos.galtzo.com/assets/images/tree-sitter/tree-sitter/avatar-192px.svg",
		"StructuredMerge packages provide fixture-backed merge behavior",
		"| tree-sitter-language-pack | Go, Ruby, Rust, TypeScript | markdown, toml, yaml, source |",
	} {
		if !strings.Contains(plan.FinalContent, snippet) {
			t.Fatalf("expected README to include %q in:\n%s", snippet, plan.FinalContent)
		}
	}

	apply, err := ApplyReadmeStyle(projectRoot)
	if err != nil {
		t.Fatalf("apply README style: %v", err)
	}
	if !apply.Changed {
		t.Fatalf("expected first apply to change README")
	}
	second, err := ApplyReadmeStyle(projectRoot)
	if err != nil {
		t.Fatalf("reapply README style: %v", err)
	}
	if second.Changed {
		t.Fatalf("expected README style reapply to converge")
	}
}

func readJSONFixture[T any](t *testing.T, parts ...string) T {
	t.Helper()
	source, err := os.ReadFile(filepath.Join(parts...))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fixture T
	if err := json.Unmarshal(source, &fixture); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	return fixture
}

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

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for relativePath, content := range files {
		targetPath := filepath.Join(root, relativePath)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			t.Fatalf("create parent directory: %v", err)
		}
		if err := os.WriteFile(targetPath, []byte(content), 0o644); err != nil {
			t.Fatalf("write fixture file: %v", err)
		}
	}
}

func readExpectedProjectFiles(t *testing.T, root string, expected map[string]string) map[string]string {
	t.Helper()
	files := map[string]string{}
	for relativePath := range expected {
		source, err := os.ReadFile(filepath.Join(root, relativePath))
		if err != nil {
			t.Fatalf("read project file: %v", err)
		}
		files[relativePath] = string(source)
	}
	return files
}

func mustReadProjectFile(t *testing.T, root string, relativePath string) string {
	t.Helper()
	source, err := os.ReadFile(filepath.Join(root, relativePath))
	if err != nil {
		t.Fatalf("read project file %s: %v", relativePath, err)
	}
	return string(source)
}

func uniqueRequestKinds(reports []RecipeRunReport) []string {
	kinds := []string{}
	for _, report := range reports {
		if !containsString(kinds, report.RequestEnvelope.Kind) {
			kinds = append(kinds, report.RequestEnvelope.Kind)
		}
	}
	return kinds
}

func uniqueReportKinds(reports []RecipeRunReport) []string {
	kinds := []string{}
	for _, report := range reports {
		if !containsString(kinds, report.ReportEnvelope.Kind) {
			kinds = append(kinds, report.ReportEnvelope.Kind)
		}
	}
	return kinds
}

func containsString(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

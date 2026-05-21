package typescriptmerge

import (
	"slices"
	"strconv"
	"strings"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/treehaver"
)

type TypeScriptDialect string
type TypeScriptBackend string

const (
	DialectTypeScript TypeScriptDialect = "typescript"
	BackendTreeSitter TypeScriptBackend = "kreuzberg-language-pack"
)

type TypeScriptOwnerKind string

const (
	OwnerImport      TypeScriptOwnerKind = "import"
	OwnerDeclaration TypeScriptOwnerKind = "declaration"
)

type TypeScriptOwner struct {
	Path      string
	OwnerKind TypeScriptOwnerKind
	MatchKey  string
}

type TypeScriptOwnerMatch struct {
	TemplatePath    string
	DestinationPath string
}

type TypeScriptOwnerMatchResult struct {
	Matched              []TypeScriptOwnerMatch
	UnmatchedTemplate    []string
	UnmatchedDestination []string
}

type moduleImport struct {
	Path     string
	MatchKey string
	Text     string
}

type moduleDeclaration struct {
	Path     string
	MatchKey string
	Text     string
}

type TypeScriptAnalysis struct {
	Dialect      TypeScriptDialect
	Source       string
	Owners       []TypeScriptOwner
	Imports      []moduleImport
	Declarations []moduleDeclaration
}

func (TypeScriptAnalysis) Kind() string {
	return "typescript"
}

type TypeScriptFeatureProfile struct {
	Family            string
	SupportedDialects []TypeScriptDialect
	SupportedPolicies []astmerge.PolicyReference
}

type TypeScriptBackendFeatureProfile struct {
	Backend           string
	BackendRef        *treehaver.BackendReference
	SupportsDialects  bool
	SupportedPolicies []astmerge.PolicyReference
}

func destinationWinsArrayPolicy() astmerge.PolicyReference {
	return astmerge.PolicyReference{Surface: astmerge.PolicySurfaceArray, Name: "destination_wins_array"}
}

func TypeScriptFeatureProfileInfo() TypeScriptFeatureProfile {
	return TypeScriptFeatureProfile{
		Family:            "typescript",
		SupportedDialects: []TypeScriptDialect{DialectTypeScript},
		SupportedPolicies: []astmerge.PolicyReference{destinationWinsArrayPolicy()},
	}
}

func TypeScriptBackendFeatureProfileInfo(backend TypeScriptBackend) TypeScriptBackendFeatureProfile {
	resolved := resolveBackend(backend)
	return TypeScriptBackendFeatureProfile{
		Backend:           string(resolved),
		BackendRef:        &treehaver.KreuzbergLanguagePackBackend,
		SupportsDialects:  true,
		SupportedPolicies: []astmerge.PolicyReference{destinationWinsArrayPolicy()},
	}
}

func TypeScriptPlanContext(backend TypeScriptBackend) astmerge.ConformanceFamilyPlanContext {
	backendProfile := TypeScriptBackendFeatureProfileInfo(backend)
	return astmerge.ConformanceFamilyPlanContext{
		FamilyProfile: astmerge.FamilyFeatureProfile{
			Family:            TypeScriptFeatureProfileInfo().Family,
			SupportedDialects: []string{string(DialectTypeScript)},
			SupportedPolicies: TypeScriptFeatureProfileInfo().SupportedPolicies,
		},
		FeatureProfile: &astmerge.ConformanceFeatureProfileView{
			Backend:           backendProfile.Backend,
			SupportsDialects:  backendProfile.SupportsDialects,
			SupportedPolicies: backendProfile.SupportedPolicies,
		},
	}
}

func TypeScriptBackends() []TypeScriptBackend {
	return []TypeScriptBackend{BackendTreeSitter}
}

func resolveBackend(backend TypeScriptBackend) TypeScriptBackend {
	if backend == "" {
		return BackendTreeSitter
	}
	return backend
}

func parseRequest(source string) treehaver.ParserRequest {
	return treehaver.ParserRequest{Source: source, Language: "typescript", Dialect: "typescript"}
}

func processRequest(source string) treehaver.ProcessRequest {
	return treehaver.ProcessRequest{Source: source, Language: "typescript"}
}

func sliceSpan(source string, span treehaver.ProcessSpan) string {
	return strings.TrimSpace(source[span.StartByte:span.EndByte])
}

func lineAnchoredSpan(source string, span treehaver.ProcessSpan) string {
	lineStart := strings.LastIndex(source[:span.StartByte], "\n")
	if lineStart >= 0 {
		lineStart++
	} else {
		lineStart = 0
	}
	return strings.TrimSpace(source[lineStart:span.EndByte])
}

func ParseTypeScript(source string, _dialect TypeScriptDialect) astmerge.ParseResult[TypeScriptAnalysis] {
	return ParseTypeScriptWithBackend(source, DialectTypeScript, BackendTreeSitter)
}

func ParseTypeScriptWithBackend(source string, _dialect TypeScriptDialect, backend TypeScriptBackend) astmerge.ParseResult[TypeScriptAnalysis] {
	resolved := resolveBackend(backend)
	if resolved != BackendTreeSitter {
		return astmerge.ParseResult[TypeScriptAnalysis]{
			OK: false,
			Diagnostics: []astmerge.Diagnostic{
				{
					Severity: astmerge.SeverityError,
					Category: astmerge.CategoryUnsupportedFeature,
					Message:  "Unsupported TypeScript backend " + string(resolved) + ".",
				},
			},
		}
	}

	parsed := treehaver.ParseWithLanguagePack(parseRequest(source))
	if !parsed.OK {
		return astmerge.ParseResult[TypeScriptAnalysis]{OK: false, Diagnostics: astmerge.DiagnosticsFromTreeHaver(parsed.Diagnostics)}
	}

	processed := treehaver.ProcessWithLanguagePack(processRequest(source))
	if !processed.OK || processed.Analysis == nil {
		return astmerge.ParseResult[TypeScriptAnalysis]{OK: false, Diagnostics: astmerge.DiagnosticsFromTreeHaver(processed.Diagnostics)}
	}
	if diagnostics := treehaver.StructuredImportSourceDiagnostics("typescript", processed.Analysis.Imports); len(diagnostics) > 0 {
		return astmerge.ParseResult[TypeScriptAnalysis]{OK: false, Diagnostics: astmerge.DiagnosticsFromTreeHaver(diagnostics)}
	}

	imports := make([]moduleImport, 0, len(processed.Analysis.Imports))
	for index, item := range processed.Analysis.Imports {
		imports = append(imports, moduleImport{
			Path:     "/imports/" + strconv.Itoa(index),
			MatchKey: item.Source,
			Text:     sliceSpan(source, item.Span) + "\n",
		})
	}

	declarations := make([]moduleDeclaration, 0)
	for _, item := range processed.Analysis.Structure {
		if item.Name == "" {
			continue
		}
		declarations = append(declarations, moduleDeclaration{
			Path:     "/declarations/" + item.Name,
			MatchKey: item.Name,
			Text:     lineAnchoredSpan(source, item.Span) + "\n",
		})
	}
	slices.SortFunc(declarations, func(left, right moduleDeclaration) int {
		return strings.Compare(left.Path, right.Path)
	})

	owners := make([]TypeScriptOwner, 0, len(imports)+len(declarations))
	for _, item := range imports {
		owners = append(owners, TypeScriptOwner{Path: item.Path, OwnerKind: OwnerImport, MatchKey: item.MatchKey})
	}
	for _, item := range declarations {
		owners = append(owners, TypeScriptOwner{Path: item.Path, OwnerKind: OwnerDeclaration, MatchKey: item.MatchKey})
	}

	analysis := TypeScriptAnalysis{
		Dialect:      DialectTypeScript,
		Source:       source,
		Owners:       owners,
		Imports:      imports,
		Declarations: declarations,
	}
	return astmerge.ParseResult[TypeScriptAnalysis]{OK: true, Diagnostics: []astmerge.Diagnostic{}, Analysis: &analysis}
}

func MatchTypeScriptOwners(template TypeScriptAnalysis, destination TypeScriptAnalysis) TypeScriptOwnerMatchResult {
	destinationOwners := make(map[string]struct{}, len(destination.Owners))
	for _, owner := range destination.Owners {
		destinationOwners[owner.Path] = struct{}{}
	}
	templateOwners := make(map[string]struct{}, len(template.Owners))
	for _, owner := range template.Owners {
		templateOwners[owner.Path] = struct{}{}
	}

	matched := make([]TypeScriptOwnerMatch, 0)
	unmatchedTemplate := make([]string, 0)
	for _, owner := range template.Owners {
		if _, ok := destinationOwners[owner.Path]; ok {
			matched = append(matched, TypeScriptOwnerMatch{TemplatePath: owner.Path, DestinationPath: owner.Path})
		} else {
			unmatchedTemplate = append(unmatchedTemplate, owner.Path)
		}
	}
	unmatchedDestination := make([]string, 0)
	for _, owner := range destination.Owners {
		if _, ok := templateOwners[owner.Path]; !ok {
			unmatchedDestination = append(unmatchedDestination, owner.Path)
		}
	}

	return TypeScriptOwnerMatchResult{Matched: matched, UnmatchedTemplate: unmatchedTemplate, UnmatchedDestination: unmatchedDestination}
}

func MergeTypeScript(templateSource, destinationSource string, dialect TypeScriptDialect) astmerge.MergeResult[string] {
	template := ParseTypeScript(templateSource, dialect)
	if !template.OK || template.Analysis == nil {
		return astmerge.MergeResult[string]{OK: false, Diagnostics: template.Diagnostics}
	}
	destination := ParseTypeScript(destinationSource, dialect)
	if !destination.OK || destination.Analysis == nil {
		diagnostics := make([]astmerge.Diagnostic, 0, len(destination.Diagnostics))
		for _, diagnostic := range destination.Diagnostics {
			if diagnostic.Category == astmerge.CategoryParseError {
				diagnostic.Category = astmerge.CategoryDestinationParseError
			}
			diagnostics = append(diagnostics, diagnostic)
		}
		return astmerge.MergeResult[string]{OK: false, Diagnostics: diagnostics}
	}

	destinationDeclarationPaths := make(map[string]struct{}, len(destination.Analysis.Declarations))
	declarationTexts := make([]string, 0, len(destination.Analysis.Declarations)+len(template.Analysis.Declarations))
	for _, item := range destination.Analysis.Declarations {
		destinationDeclarationPaths[item.Path] = struct{}{}
		declarationTexts = append(declarationTexts, item.Text)
	}
	for _, item := range template.Analysis.Declarations {
		if _, ok := destinationDeclarationPaths[item.Path]; !ok {
			declarationTexts = append(declarationTexts, item.Text)
		}
	}

	sections := make([]string, 0, 2)
	if len(destination.Analysis.Imports) > 0 {
		importTexts := make([]string, 0, len(destination.Analysis.Imports))
		for _, item := range destination.Analysis.Imports {
			importTexts = append(importTexts, item.Text)
		}
		sections = append(sections, strings.TrimSpace(strings.Join(importTexts, "")))
	}
	if len(declarationTexts) > 0 {
		sections = append(sections, strings.TrimSpace(strings.Join(declarationTexts, "\n")))
	}
	output := strings.Join(sections, "\n\n") + "\n"
	return astmerge.MergeResult[string]{
		OK:          true,
		Diagnostics: []astmerge.Diagnostic{},
		Output:      &output,
		Policies:    []astmerge.PolicyReference{destinationWinsArrayPolicy()},
	}
}

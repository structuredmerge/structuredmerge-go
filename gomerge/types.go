package gomerge

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/treehaver"
)

type GoDialect string
type GoBackend string

const (
	DialectGo         GoDialect = "go"
	BackendTreeSitter GoBackend = "kreuzberg-language-pack"
)

type GoOwnerKind string

const (
	OwnerImport      GoOwnerKind = "import"
	OwnerDeclaration GoOwnerKind = "declaration"
)

type GoOwner struct {
	Path      string
	OwnerKind GoOwnerKind
	MatchKey  string
}

type GoOwnerMatch struct {
	TemplatePath    string
	DestinationPath string
}

type GoOwnerMatchResult struct {
	Matched              []GoOwnerMatch
	UnmatchedTemplate    []string
	UnmatchedDestination []string
}

type moduleImport struct {
	Path     string
	MatchKey string
	Text     string
}

type GoModuleImport = moduleImport

type moduleDeclaration struct {
	Path     string
	MatchKey string
	Text     string
}

type GoModuleDeclaration = moduleDeclaration

type GoAnalysis struct {
	Dialect      GoDialect
	Source       string
	Owners       []GoOwner
	Imports      []moduleImport
	Declarations []moduleDeclaration
}

func (GoAnalysis) Kind() string {
	return "go"
}

type GoFeatureProfile struct {
	Family            string
	SupportedDialects []GoDialect
	SupportedPolicies []astmerge.PolicyReference
}

type GoBackendFeatureProfile struct {
	Backend           string
	BackendRef        *treehaver.BackendReference
	SupportsDialects  bool
	SupportedPolicies []astmerge.PolicyReference
}

func destinationWinsArrayPolicy() astmerge.PolicyReference {
	return astmerge.PolicyReference{Surface: astmerge.PolicySurfaceArray, Name: "destination_wins_array"}
}

func GoFeatureProfileInfo() GoFeatureProfile {
	return GoFeatureProfile{
		Family:            "go",
		SupportedDialects: []GoDialect{DialectGo},
		SupportedPolicies: []astmerge.PolicyReference{destinationWinsArrayPolicy()},
	}
}

func GoBackendFeatureProfileInfo(backend GoBackend) GoBackendFeatureProfile {
	resolved := resolveBackend(backend)
	if resolved != BackendTreeSitter {
		return GoBackendFeatureProfile{
			Backend:           string(resolved),
			BackendRef:        nil,
			SupportsDialects:  false,
			SupportedPolicies: []astmerge.PolicyReference{destinationWinsArrayPolicy()},
		}
	}

	return GoBackendFeatureProfile{
		Backend:           treehaver.KreuzbergLanguagePackBackend.ID,
		BackendRef:        &treehaver.KreuzbergLanguagePackBackend,
		SupportsDialects:  true,
		SupportedPolicies: []astmerge.PolicyReference{destinationWinsArrayPolicy()},
	}
}

func GoPlanContext(backend GoBackend) astmerge.ConformanceFamilyPlanContext {
	backendProfile := GoBackendFeatureProfileInfo(backend)
	return astmerge.ConformanceFamilyPlanContext{
		FamilyProfile: astmerge.FamilyFeatureProfile{
			Family:            GoFeatureProfileInfo().Family,
			SupportedDialects: []string{string(DialectGo)},
			SupportedPolicies: GoFeatureProfileInfo().SupportedPolicies,
		},
		FeatureProfile: &astmerge.ConformanceFeatureProfileView{
			Backend:           backendProfile.Backend,
			SupportsDialects:  backendProfile.SupportsDialects,
			SupportedPolicies: backendProfile.SupportedPolicies,
		},
	}
}

func GoBackends() []GoBackend {
	return []GoBackend{BackendTreeSitter}
}

func resolveBackend(backend GoBackend) GoBackend {
	if backend == "" {
		return BackendTreeSitter
	}
	return backend
}

func parseRequest(source string) treehaver.ParserRequest {
	return treehaver.ParserRequest{Source: source, Language: "go", Dialect: "go"}
}

func processRequest(source string) treehaver.ProcessRequest {
	return treehaver.ProcessRequest{Source: source, Language: "go"}
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

func ParseGo(source string, _dialect GoDialect) astmerge.ParseResult[GoAnalysis] {
	return ParseGoWithBackend(source, DialectGo, BackendTreeSitter)
}

func ParseGoWithBackend(source string, _dialect GoDialect, backend GoBackend) astmerge.ParseResult[GoAnalysis] {
	resolved := resolveBackend(backend)
	if resolved != BackendTreeSitter {
		return astmerge.ParseResult[GoAnalysis]{
			OK: false,
			Diagnostics: []astmerge.Diagnostic{
				{
					Severity: astmerge.SeverityError,
					Category: astmerge.CategoryUnsupportedFeature,
					Message:  fmt.Sprintf("Unsupported Go backend %s.", resolved),
				},
			},
		}
	}

	parsed := treehaver.ParseWithLanguagePack(parseRequest(source))
	if !parsed.OK {
		return astmerge.ParseResult[GoAnalysis]{OK: false, Diagnostics: astmerge.DiagnosticsFromTreeHaver(parsed.Diagnostics)}
	}

	processed := treehaver.ProcessWithLanguagePack(processRequest(source))
	if !processed.OK || processed.Analysis == nil {
		return astmerge.ParseResult[GoAnalysis]{OK: false, Diagnostics: astmerge.DiagnosticsFromTreeHaver(processed.Diagnostics)}
	}
	if diagnostics := treehaver.StructuredImportSourceDiagnostics("go", processed.Analysis.Imports); len(diagnostics) > 0 {
		return astmerge.ParseResult[GoAnalysis]{OK: false, Diagnostics: astmerge.DiagnosticsFromTreeHaver(diagnostics)}
	}

	dedupedImports := make(map[string]moduleImport)
	importOrder := make([]string, 0)
	for _, item := range processed.Analysis.Imports {
		matchKey := item.Source
		candidate := moduleImport{
			Path:     "",
			MatchKey: matchKey,
			Text:     sliceSpan(source, item.Span) + "\n",
		}
		current, ok := dedupedImports[matchKey]
		if !ok {
			importOrder = append(importOrder, matchKey)
			dedupedImports[matchKey] = candidate
			continue
		}
		if len(candidate.Text) > len(current.Text) {
			dedupedImports[matchKey] = candidate
		}
	}
	imports := make([]moduleImport, 0, len(importOrder))
	for index, matchKey := range importOrder {
		item := dedupedImports[matchKey]
		item.Path = "/imports/" + strconv.Itoa(index)
		imports = append(imports, item)
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

	owners := make([]GoOwner, 0, len(imports)+len(declarations))
	for _, item := range imports {
		owners = append(owners, GoOwner{Path: item.Path, OwnerKind: OwnerImport, MatchKey: item.MatchKey})
	}
	for _, item := range declarations {
		owners = append(owners, GoOwner{Path: item.Path, OwnerKind: OwnerDeclaration, MatchKey: item.MatchKey})
	}

	analysis := GoAnalysis{
		Dialect:      DialectGo,
		Source:       source,
		Owners:       owners,
		Imports:      imports,
		Declarations: declarations,
	}
	return astmerge.ParseResult[GoAnalysis]{OK: true, Diagnostics: []astmerge.Diagnostic{}, Analysis: &analysis}
}

func MatchGoOwners(template GoAnalysis, destination GoAnalysis) GoOwnerMatchResult {
	destinationOwners := make(map[string]struct{}, len(destination.Owners))
	for _, owner := range destination.Owners {
		destinationOwners[owner.Path] = struct{}{}
	}
	templateOwners := make(map[string]struct{}, len(template.Owners))
	for _, owner := range template.Owners {
		templateOwners[owner.Path] = struct{}{}
	}

	matched := make([]GoOwnerMatch, 0)
	unmatchedTemplate := make([]string, 0)
	for _, owner := range template.Owners {
		if _, ok := destinationOwners[owner.Path]; ok {
			matched = append(matched, GoOwnerMatch{TemplatePath: owner.Path, DestinationPath: owner.Path})
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

	return GoOwnerMatchResult{Matched: matched, UnmatchedTemplate: unmatchedTemplate, UnmatchedDestination: unmatchedDestination}
}

func MergeGo(templateSource, destinationSource string, dialect GoDialect) astmerge.MergeResult[string] {
	return MergeGoWithBackend(templateSource, destinationSource, dialect, BackendTreeSitter)
}

func MergeGoWithParser(
	templateSource string,
	destinationSource string,
	dialect GoDialect,
	parser func(source string, dialect GoDialect) astmerge.ParseResult[GoAnalysis],
) astmerge.MergeResult[string] {
	template := parser(templateSource, dialect)
	if !template.OK || template.Analysis == nil {
		return astmerge.MergeResult[string]{OK: false, Diagnostics: template.Diagnostics}
	}
	destination := parser(destinationSource, dialect)
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

func MergeGoWithBackend(templateSource, destinationSource string, dialect GoDialect, backend GoBackend) astmerge.MergeResult[string] {
	resolved := resolveBackend(backend)
	if resolved != BackendTreeSitter {
		return astmerge.MergeResult[string]{
			OK: false,
			Diagnostics: []astmerge.Diagnostic{
				{
					Severity: astmerge.SeverityError,
					Category: astmerge.CategoryUnsupportedFeature,
					Message:  fmt.Sprintf("Unsupported Go backend %s.", resolved),
				},
			},
		}
	}
	return MergeGoWithParser(templateSource, destinationSource, dialect, func(source string, parseDialect GoDialect) astmerge.ParseResult[GoAnalysis] {
		return ParseGoWithBackend(source, parseDialect, resolved)
	})
}

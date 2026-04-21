package rustmerge

import (
	"slices"
	"strconv"
	"strings"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/treehaver"
)

type RustDialect string
type RustBackend string

const (
	DialectRust RustDialect = "rust"
	BackendTreeSitter RustBackend = "kreuzberg-language-pack"
)

type RustOwnerKind string

const (
	OwnerImport      RustOwnerKind = "import"
	OwnerDeclaration RustOwnerKind = "declaration"
)

type RustOwner struct {
	Path      string
	OwnerKind RustOwnerKind
	MatchKey  string
}

type RustOwnerMatch struct {
	TemplatePath    string
	DestinationPath string
}

type RustOwnerMatchResult struct {
	Matched              []RustOwnerMatch
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

type RustAnalysis struct {
	Dialect      RustDialect
	Source       string
	Owners       []RustOwner
	Imports      []moduleImport
	Declarations []moduleDeclaration
}

func (RustAnalysis) Kind() string {
	return "rust"
}

type RustFeatureProfile struct {
	Family            string
	SupportedDialects []RustDialect
	SupportedPolicies []astmerge.PolicyReference
}

type RustBackendFeatureProfile struct {
	Backend           string
	BackendRef        *treehaver.BackendReference
	SupportsDialects  bool
	SupportedPolicies []astmerge.PolicyReference
}

func destinationWinsArrayPolicy() astmerge.PolicyReference {
	return astmerge.PolicyReference{Surface: astmerge.PolicySurfaceArray, Name: "destination_wins_array"}
}

func RustFeatureProfileInfo() RustFeatureProfile {
	return RustFeatureProfile{
		Family:            "rust",
		SupportedDialects: []RustDialect{DialectRust},
		SupportedPolicies: []astmerge.PolicyReference{destinationWinsArrayPolicy()},
	}
}

func RustBackendFeatureProfileInfo(backend RustBackend) RustBackendFeatureProfile {
	resolved := backend
	if resolved == "" {
		resolved = BackendTreeSitter
	}

	if resolved != BackendTreeSitter {
		return RustBackendFeatureProfile{
			Backend:           string(resolved),
			BackendRef:        nil,
			SupportsDialects:  false,
			SupportedPolicies: []astmerge.PolicyReference{destinationWinsArrayPolicy()},
		}
	}

	return RustBackendFeatureProfile{
		Backend:           treehaver.KreuzbergLanguagePackBackend.ID,
		BackendRef:        &treehaver.KreuzbergLanguagePackBackend,
		SupportsDialects:  true,
		SupportedPolicies: []astmerge.PolicyReference{destinationWinsArrayPolicy()},
	}
}

func RustPlanContext(backend RustBackend) astmerge.ConformanceFamilyPlanContext {
	backendProfile := RustBackendFeatureProfileInfo(backend)
	return astmerge.ConformanceFamilyPlanContext{
		FamilyProfile: astmerge.FamilyFeatureProfile{
			Family:            RustFeatureProfileInfo().Family,
			SupportedDialects: []string{string(DialectRust)},
			SupportedPolicies: RustFeatureProfileInfo().SupportedPolicies,
		},
		FeatureProfile: &astmerge.ConformanceFeatureProfileView{
			Backend:           backendProfile.Backend,
			SupportsDialects:  backendProfile.SupportsDialects,
			SupportedPolicies: backendProfile.SupportedPolicies,
		},
	}
}

func RustBackends() []RustBackend {
	return []RustBackend{BackendTreeSitter}
}

func parseRequest(source string) treehaver.ParserRequest {
	return treehaver.ParserRequest{Source: source, Language: "rust", Dialect: "rust"}
}

func processRequest(source string) treehaver.ProcessRequest {
	return treehaver.ProcessRequest{Source: source, Language: "rust"}
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

func normalizeRustImportPath(importSource string) string {
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(importSource), "use "), ";"))
}

func ParseRust(source string, _dialect RustDialect) astmerge.ParseResult[RustAnalysis] {
	parsed := treehaver.ParseWithLanguagePack(parseRequest(source))
	if !parsed.OK {
		return astmerge.ParseResult[RustAnalysis]{OK: false, Diagnostics: parsed.Diagnostics}
	}

	processed := treehaver.ProcessWithLanguagePack(processRequest(source))
	if !processed.OK || processed.Analysis == nil {
		return astmerge.ParseResult[RustAnalysis]{OK: false, Diagnostics: processed.Diagnostics}
	}

	imports := make([]moduleImport, 0, len(processed.Analysis.Imports))
	for index, item := range processed.Analysis.Imports {
		imports = append(imports, moduleImport{
			Path:     "/imports/" + strconv.Itoa(index),
			MatchKey: normalizeRustImportPath(item.Source),
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

	owners := make([]RustOwner, 0, len(imports)+len(declarations))
	for _, item := range imports {
		owners = append(owners, RustOwner{Path: item.Path, OwnerKind: OwnerImport, MatchKey: item.MatchKey})
	}
	for _, item := range declarations {
		owners = append(owners, RustOwner{Path: item.Path, OwnerKind: OwnerDeclaration, MatchKey: item.MatchKey})
	}

	analysis := RustAnalysis{
		Dialect:      DialectRust,
		Source:       source,
		Owners:       owners,
		Imports:      imports,
		Declarations: declarations,
	}
	return astmerge.ParseResult[RustAnalysis]{OK: true, Diagnostics: []astmerge.Diagnostic{}, Analysis: &analysis}
}

func MatchRustOwners(template RustAnalysis, destination RustAnalysis) RustOwnerMatchResult {
	destinationOwners := make(map[string]struct{}, len(destination.Owners))
	for _, owner := range destination.Owners {
		destinationOwners[owner.Path] = struct{}{}
	}
	templateOwners := make(map[string]struct{}, len(template.Owners))
	for _, owner := range template.Owners {
		templateOwners[owner.Path] = struct{}{}
	}

	matched := make([]RustOwnerMatch, 0)
	unmatchedTemplate := make([]string, 0)
	for _, owner := range template.Owners {
		if _, ok := destinationOwners[owner.Path]; ok {
			matched = append(matched, RustOwnerMatch{TemplatePath: owner.Path, DestinationPath: owner.Path})
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

	return RustOwnerMatchResult{Matched: matched, UnmatchedTemplate: unmatchedTemplate, UnmatchedDestination: unmatchedDestination}
}

func MergeRust(templateSource, destinationSource string, dialect RustDialect) astmerge.MergeResult[string] {
	template := ParseRust(templateSource, dialect)
	if !template.OK || template.Analysis == nil {
		return astmerge.MergeResult[string]{OK: false, Diagnostics: template.Diagnostics}
	}
	destination := ParseRust(destinationSource, dialect)
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

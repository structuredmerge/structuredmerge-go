package gomerge

import (
	"slices"
	"strconv"
	"strings"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/treehaver"
)

type GoDialect string

const (
	DialectGo GoDialect = "go"
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

type moduleDeclaration struct {
	Path     string
	MatchKey string
	Text     string
}

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

func normalizeGoImportPath(importSource string) string {
	quoteMatchStart := strings.Index(importSource, "\"")
	if quoteMatchStart >= 0 {
		quoteMatchEnd := strings.Index(importSource[quoteMatchStart+1:], "\"")
		if quoteMatchEnd >= 0 {
			return importSource[quoteMatchStart+1 : quoteMatchStart+1+quoteMatchEnd]
		}
	}
	return strings.TrimSpace(strings.TrimPrefix(importSource, "import "))
}

func ParseGo(source string, _dialect GoDialect) astmerge.ParseResult[GoAnalysis] {
	parsed := treehaver.ParseWithLanguagePack(parseRequest(source))
	if !parsed.OK {
		return astmerge.ParseResult[GoAnalysis]{OK: false, Diagnostics: parsed.Diagnostics}
	}

	processed := treehaver.ProcessWithLanguagePack(processRequest(source))
	if !processed.OK || processed.Analysis == nil {
		return astmerge.ParseResult[GoAnalysis]{OK: false, Diagnostics: processed.Diagnostics}
	}

	dedupedImports := make(map[string]moduleImport)
	importOrder := make([]string, 0)
	for _, item := range processed.Analysis.Imports {
		matchKey := normalizeGoImportPath(item.Source)
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
	template := ParseGo(templateSource, dialect)
	if !template.OK || template.Analysis == nil {
		return astmerge.MergeResult[string]{OK: false, Diagnostics: template.Diagnostics}
	}
	destination := ParseGo(destinationSource, dialect)
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

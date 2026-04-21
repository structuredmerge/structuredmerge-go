package goparsermerge

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"slices"
	"strconv"
	"strings"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/gomerge"
	"github.com/structuredmerge/structuredmerge-go/treehaver"
)

const BackendGoParser = "go-parser"

type GoBackendFeatureProfile struct {
	Family            string
	SupportedDialects []gomerge.GoDialect
	SupportedPolicies []astmerge.PolicyReference
	Backend           string
	BackendRef        *treehaver.BackendReference
}

func init() {
	treehaver.RegisterBackend(treehaver.BackendReference{ID: BackendGoParser, Family: "native"})
}

func unsupportedFeature(message string) astmerge.Diagnostic {
	return astmerge.Diagnostic{
		Severity: astmerge.SeverityError,
		Category: astmerge.CategoryUnsupportedFeature,
		Message:  message,
	}
}

func parseError(message string) astmerge.Diagnostic {
	return astmerge.Diagnostic{
		Severity: astmerge.SeverityError,
		Category: astmerge.CategoryParseError,
		Message:  message,
	}
}

func destinationWinsArrayPolicy() astmerge.PolicyReference {
	return astmerge.PolicyReference{Surface: astmerge.PolicySurfaceArray, Name: "destination_wins_array"}
}

func GoFeatureProfileInfo() gomerge.GoFeatureProfile {
	return gomerge.GoFeatureProfileInfo()
}

func AvailableGoBackends() []string {
	return []string{BackendGoParser}
}

func GoBackendFeatureProfileInfo() GoBackendFeatureProfile {
	return GoBackendFeatureProfile{
		Family:            "go",
		SupportedDialects: []gomerge.GoDialect{gomerge.DialectGo},
		SupportedPolicies: []astmerge.PolicyReference{destinationWinsArrayPolicy()},
		Backend:           BackendGoParser,
		BackendRef:        &treehaver.BackendReference{ID: BackendGoParser, Family: "native"},
	}
}

func GoPlanContext() astmerge.ConformanceFamilyPlanContext {
	return astmerge.ConformanceFamilyPlanContext{
		FamilyProfile: astmerge.FamilyFeatureProfile{
			Family:            "go",
			SupportedDialects: []string{"go"},
			SupportedPolicies: []astmerge.PolicyReference{destinationWinsArrayPolicy()},
		},
		FeatureProfile: &astmerge.ConformanceFeatureProfileView{
			Backend:           BackendGoParser,
			SupportsDialects:  true,
			SupportedPolicies: []astmerge.PolicyReference{destinationWinsArrayPolicy()},
		},
	}
}

func ParseGo(source string, dialect gomerge.GoDialect, backend ...string) astmerge.ParseResult[gomerge.GoAnalysis] {
	requested := BackendGoParser
	if len(backend) > 0 && backend[0] != "" {
		requested = backend[0]
	}
	if requested != BackendGoParser {
		return astmerge.ParseResult[gomerge.GoAnalysis]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{unsupportedFeature(fmt.Sprintf("Unsupported Go backend %s.", requested))},
		}
	}
	if dialect != gomerge.DialectGo {
		return astmerge.ParseResult[gomerge.GoAnalysis]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{unsupportedFeature(fmt.Sprintf("Unsupported Go dialect %s.", dialect))},
		}
	}

	return parseGoNative(source)
}

func MatchGoOwners(template, destination gomerge.GoAnalysis) gomerge.GoOwnerMatchResult {
	return gomerge.MatchGoOwners(template, destination)
}

func MergeGo(templateSource string, destinationSource string, dialect gomerge.GoDialect, backend ...string) astmerge.MergeResult[string] {
	requested := BackendGoParser
	if len(backend) > 0 && backend[0] != "" {
		requested = backend[0]
	}
	if requested != BackendGoParser {
		return astmerge.MergeResult[string]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{unsupportedFeature(fmt.Sprintf("Unsupported Go backend %s.", requested))},
		}
	}

	return gomerge.MergeGoWithParser(templateSource, destinationSource, dialect, func(source string, parseDialect gomerge.GoDialect) astmerge.ParseResult[gomerge.GoAnalysis] {
		return ParseGo(source, parseDialect)
	})
}

func parseGoNative(source string) astmerge.ParseResult[gomerge.GoAnalysis] {
	originalSource := source
	offsetBytes := 0
	if !strings.HasPrefix(strings.TrimSpace(source), "package ") {
		prefix := "package main\n\n"
		source = prefix + source
		offsetBytes = len(prefix)
	}

	fset := token.NewFileSet()
	file, err := goparser.ParseFile(fset, "input.go", source, goparser.ParseComments)
	if err != nil {
		return astmerge.ParseResult[gomerge.GoAnalysis]{
			OK: false,
			Diagnostics: []astmerge.Diagnostic{
				parseError(strings.ReplaceAll(err.Error(), "input.go:3:", "input.go:1:")),
			},
		}
	}

	imports := make([]gomerge.GoModuleImport, 0, len(file.Imports))
	for index, item := range file.Imports {
		matchKey := strings.Trim(item.Path.Value, "\"")
		start := fset.Position(item.Pos()).Offset - offsetBytes
		end := fset.Position(item.End()).Offset - offsetBytes
		lineStart := strings.LastIndex(originalSource[:start], "\n")
		if lineStart >= 0 {
			lineStart++
		} else {
			lineStart = 0
		}
		imports = append(imports, gomerge.GoModuleImport{
			Path:     "/imports/" + strconv.Itoa(index),
			MatchKey: matchKey,
			Text:     strings.TrimSpace(originalSource[lineStart:end]) + "\n",
		})
	}

	declarations := make([]gomerge.GoModuleDeclaration, 0)
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Name == nil {
			continue
		}
		start := fset.Position(funcDecl.Pos()).Offset - offsetBytes
		end := fset.Position(funcDecl.End()).Offset - offsetBytes
		lineStart := strings.LastIndex(originalSource[:start], "\n")
		if lineStart >= 0 {
			lineStart++
		} else {
			lineStart = 0
		}
		declarations = append(declarations, gomerge.GoModuleDeclaration{
			Path:     "/declarations/" + funcDecl.Name.Name,
			MatchKey: funcDecl.Name.Name,
			Text:     strings.TrimSpace(originalSource[lineStart:end]) + "\n",
		})
	}
	slices.SortFunc(declarations, func(left, right gomerge.GoModuleDeclaration) int {
		return strings.Compare(left.Path, right.Path)
	})

	owners := make([]gomerge.GoOwner, 0, len(imports)+len(declarations))
	for _, item := range imports {
		owners = append(owners, gomerge.GoOwner{Path: item.Path, OwnerKind: gomerge.OwnerImport, MatchKey: item.MatchKey})
	}
	for _, item := range declarations {
		owners = append(owners, gomerge.GoOwner{Path: item.Path, OwnerKind: gomerge.OwnerDeclaration, MatchKey: item.MatchKey})
	}

	analysis := gomerge.GoAnalysis{
		Dialect:      gomerge.DialectGo,
		Source:       originalSource,
		Owners:       owners,
		Imports:      imports,
		Declarations: declarations,
	}
	return astmerge.ParseResult[gomerge.GoAnalysis]{OK: true, Diagnostics: []astmerge.Diagnostic{}, Analysis: &analysis}
}

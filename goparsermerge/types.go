package goparsermerge

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	goparser "go/parser"
	"go/scanner"
	"go/token"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/gomerge"
	"github.com/structuredmerge/structuredmerge-go/treehaver"
)

const BackendGoParser = "go-parser"

const syntheticPackagePrefix = "package main\n\n"

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

func ApplyEditProjection(request treehaver.EditProjectionExecutionRequest) treehaver.EditProjectionExecutionResult {
	if request.ProviderID != BackendGoParser || request.BackendRef.ID != BackendGoParser {
		return treehaver.BuildEditProjectionExecutionResult(request.Source, nil, []treehaver.ProviderDiagnostic{{
			Severity: "error",
			Category: "unsupported_feature",
			Code:     "provider_edit_projection_unsupported",
			Message:  fmt.Sprintf("provider %s does not support edit projection execution", request.ProviderID),
			Path:     "provider_id",
			Blocking: true,
		}})
	}
	if request.Language != "go" {
		return treehaver.BuildEditProjectionExecutionResult(request.Source, nil, []treehaver.ProviderDiagnostic{{
			Severity: "error",
			Category: "unsupported_feature",
			Code:     "edit_projection_language_unsupported",
			Message:  fmt.Sprintf("language %s is not supported by go/parser edit projection", request.Language),
			Path:     "language",
			Blocking: true,
		}})
	}
	if len(request.Operations) != 1 {
		return treehaver.BuildEditProjectionExecutionResult(request.Source, nil, []treehaver.ProviderDiagnostic{{
			Severity: "error",
			Category: "unsupported_feature",
			Code:     "edit_projection_batch_unsupported",
			Message:  "go/parser edit projection currently supports exactly one operation",
			Path:     "operations",
			Blocking: true,
		}})
	}

	operation := request.Operations[0]
	if operation.Operation != "replace_node" && operation.Operation != "insert_child" && operation.Operation != "delete_node" {
		return treehaver.BuildEditProjectionExecutionResult(request.Source, nil, []treehaver.ProviderDiagnostic{{
			Severity: "error",
			Category: "unsupported_feature",
			Code:     "edit_projection_operation_unsupported",
			Message:  fmt.Sprintf("go/parser edit projection does not support %s", operation.Operation),
			Path:     "operations[0].operation",
			Blocking: true,
		}})
	}

	output, err := applyGoParserNodeOperation(request.Source, operation)
	if err != nil {
		return treehaver.BuildEditProjectionExecutionResult(request.Source, nil, []treehaver.ProviderDiagnostic{{
			Severity: "error",
			Category: "parse_error",
			Code:     "edit_projection_apply_failed",
			Message:  err.Error(),
			Path:     "operations[0]",
			Blocking: true,
		}})
	}

	applied := []treehaver.AppliedEditProjectionOperation{{
		Operation:        operation.Operation,
		TargetNodeID:     operation.TargetNodeID,
		CorrelationKey:   "metadata.go_parser.node_path",
		CorrelationValue: operation.TargetNodePath,
	}}
	return treehaver.BuildEditProjectionExecutionResult(output, applied, nil)
}

func parseGoNative(source string) astmerge.ParseResult[gomerge.GoAnalysis] {
	originalSource := source
	offsetBytes := 0
	if !hasPackageClause(source) {
		prefix := syntheticPackagePrefix
		source = prefix + source
		offsetBytes = len(prefix)
	}

	fset := token.NewFileSet()
	file, err := goparser.ParseFile(fset, "input.go", source, goparser.ParseComments)
	if err != nil {
		return astmerge.ParseResult[gomerge.GoAnalysis]{
			OK: false,
			Diagnostics: []astmerge.Diagnostic{
				parseError(normalizeGoParserError(err, offsetBytes)),
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
		if genDecl, ok := decl.(*ast.GenDecl); ok {
			if ownerPath, matchKey, supported := nativeGenDeclOwner(genDecl); supported {
				start := fset.Position(genDecl.Pos()).Offset
				if genDecl.Doc != nil {
					start = fset.Position(genDecl.Doc.Pos()).Offset
				}
				start -= offsetBytes
				end := fset.Position(genDecl.End()).Offset - offsetBytes
				lineStart := strings.LastIndex(originalSource[:start], "\n")
				if lineStart >= 0 {
					lineStart++
				} else {
					lineStart = 0
				}
				declarations = append(declarations, gomerge.GoModuleDeclaration{
					Path: ownerPath, MatchKey: matchKey,
					Text: strings.TrimSpace(originalSource[lineStart:end]) + "\n",
				})
			}
			continue
		}
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Name == nil {
			continue
		}
		start := fset.Position(funcDecl.Pos()).Offset
		if funcDecl.Doc != nil {
			start = fset.Position(funcDecl.Doc.Pos()).Offset
		}
		start -= offsetBytes
		end := fset.Position(funcDecl.End()).Offset - offsetBytes
		lineStart := strings.LastIndex(originalSource[:start], "\n")
		if lineStart >= 0 {
			lineStart++
		} else {
			lineStart = 0
		}
		ownerPath := "/declarations/" + funcDecl.Name.Name
		matchKey := funcDecl.Name.Name
		if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
			if receiver, ok := receiverTypeName(funcDecl.Recv.List[0].Type); ok {
				ownerPath = "/method:" + receiver + "." + funcDecl.Name.Name
				matchKey = receiver + "." + funcDecl.Name.Name
			}
		}
		declarations = append(declarations, gomerge.GoModuleDeclaration{
			Path:     ownerPath,
			MatchKey: matchKey,
			Text:     strings.TrimSpace(originalSource[lineStart:end]) + "\n",
		})
	}
	slices.SortFunc(declarations, func(left, right gomerge.GoModuleDeclaration) int {
		return strings.Compare(left.Path, right.Path)
	})

	owners := make([]gomerge.GoOwner, 0, len(imports)+len(declarations)+1)
	if offsetBytes == 0 && file.Name != nil {
		owners = append(owners, gomerge.GoOwner{Path: "/package:" + file.Name.Name, OwnerKind: gomerge.OwnerPackage, MatchKey: file.Name.Name})
	}
	for _, item := range imports {
		owners = append(owners, gomerge.GoOwner{Path: item.Path, OwnerKind: gomerge.OwnerImport, MatchKey: item.MatchKey})
	}
	for _, item := range declarations {
		owners = append(owners, gomerge.GoOwner{Path: item.Path, OwnerKind: gomerge.OwnerDeclaration, MatchKey: item.MatchKey})
	}
	seenOwnerPaths := make(map[string]struct{}, len(owners))
	for _, owner := range owners {
		if _, exists := seenOwnerPaths[owner.Path]; exists {
			return astmerge.ParseResult[gomerge.GoAnalysis]{
				OK: false,
				Diagnostics: []astmerge.Diagnostic{
					parseError(fmt.Sprintf("duplicate top-level Go owner identity %q", owner.Path)),
				},
			}
		}
		seenOwnerPaths[owner.Path] = struct{}{}
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

func normalizeGoParserError(err error, syntheticPrefixBytes int) string {
	if syntheticPrefixBytes == 0 {
		return err.Error()
	}
	errorList, ok := err.(scanner.ErrorList)
	if !ok {
		return err.Error()
	}
	for _, item := range errorList {
		if item == nil || !item.Pos.IsValid() {
			continue
		}
		item.Pos.Offset -= syntheticPrefixBytes
		item.Pos.Line -= strings.Count(syntheticPackagePrefix, "\n")
		if item.Pos.Offset < 0 {
			item.Pos.Offset = 0
		}
		if item.Pos.Line < 1 {
			item.Pos.Line = 1
		}
	}
	return errorList.Error()
}

func hasPackageClause(source string) bool {
	fileSet := token.NewFileSet()
	file := fileSet.AddFile("input.go", fileSet.Base(), len(source))
	var sourceScanner scanner.Scanner
	sourceScanner.Init(file, []byte(source), nil, scanner.ScanComments)
	for {
		_, tokenKind, _ := sourceScanner.Scan()
		if tokenKind == token.COMMENT || tokenKind == token.SEMICOLON {
			continue
		}
		return tokenKind == token.PACKAGE
	}
}

func nativeGenDeclOwner(declaration *ast.GenDecl) (string, string, bool) {
	var label string
	switch declaration.Tok {
	case token.TYPE:
		label = "type"
	case token.CONST:
		label = "const"
	case token.VAR:
		label = "var"
	default:
		return "", "", false
	}

	groups := make([][]string, 0, len(declaration.Specs))
	for _, spec := range declaration.Specs {
		names := make([]string, 0)
		switch value := spec.(type) {
		case *ast.TypeSpec:
			names = append(names, value.Name.Name)
		case *ast.ValueSpec:
			for _, name := range value.Names {
				names = append(names, name.Name)
			}
		}
		if len(names) == 0 {
			return "", "", false
		}
		groups = append(groups, names)
	}
	if len(groups) == 0 {
		return "", "", false
	}

	var identity string
	if declaration.Lparen.IsValid() {
		parts := make([]string, 0, len(groups))
		for _, names := range groups {
			parts = append(parts, strings.Join(names, ","))
		}
		identity = label + "_group:" + strings.Join(parts, ";")
	} else {
		names := make([]string, 0)
		for _, group := range groups {
			names = append(names, group...)
		}
		identity = label + ":" + strings.Join(names, ",")
	}
	return "/" + identity, identity, true
}

func receiverTypeName(expression ast.Expr) (string, bool) {
	switch value := expression.(type) {
	case *ast.Ident:
		return value.Name, true
	case *ast.StarExpr:
		return receiverTypeName(value.X)
	case *ast.IndexExpr:
		return receiverTypeName(value.X)
	case *ast.IndexListExpr:
		return receiverTypeName(value.X)
	default:
		return "", false
	}
}

func applyGoParserNodeOperation(source string, operation treehaver.EditProjectionOperationRequest) (string, error) {
	index, err := declIndex(operation.TargetNodePath)
	if err != nil {
		return "", err
	}

	fset := token.NewFileSet()
	file, err := goparser.ParseFile(fset, "input.go", source, goparser.ParseComments)
	if err != nil {
		return "", err
	}

	switch operation.Operation {
	case "replace_node":
		if index < 0 || index >= len(file.Decls) {
			return "", fmt.Errorf("target node path %s is outside declaration bounds", operation.TargetNodePath)
		}
		replacementDecl, err := parseOneReplacementDecl(operation.ReplacementSource)
		if err != nil {
			return "", err
		}
		file.Decls[index] = replacementDecl
	case "insert_child":
		if index < 0 || index > len(file.Decls) {
			return "", fmt.Errorf("target node path %s is outside declaration insertion bounds", operation.TargetNodePath)
		}
		replacementDecl, err := parseOneReplacementDecl(operation.ReplacementSource)
		if err != nil {
			return "", err
		}
		file.Decls = slices.Insert(file.Decls, index, replacementDecl)
	case "delete_node":
		if index < 0 || index >= len(file.Decls) {
			return "", fmt.Errorf("target node path %s is outside declaration deletion bounds", operation.TargetNodePath)
		}
		file.Decls = slices.Delete(file.Decls, index, index+1)
	default:
		return "", fmt.Errorf("unsupported edit projection operation %s", operation.Operation)
	}

	var buffer bytes.Buffer
	if err := format.Node(&buffer, fset, file); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

func parseOneReplacementDecl(replacementSource string) (ast.Decl, error) {
	replacementFile, err := goparser.ParseFile(token.NewFileSet(), "replacement.go", "package replacement\n\n"+replacementSource, goparser.ParseComments)
	if err != nil {
		return nil, err
	}
	if len(replacementFile.Decls) != 1 {
		return nil, fmt.Errorf("replacement source must contain exactly one declaration")
	}
	return replacementFile.Decls[0], nil
}

func declIndex(targetNodePath string) (int, error) {
	matches := regexp.MustCompile(`^decls\[(\d+)\]$`).FindStringSubmatch(targetNodePath)
	if matches == nil {
		return 0, fmt.Errorf("unsupported go/parser node path %s", targetNodePath)
	}
	index, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, err
	}
	return index, nil
}

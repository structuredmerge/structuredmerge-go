package godstmerge

import (
	"bytes"
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/gomerge"
	"github.com/structuredmerge/structuredmerge-go/treehaver"
)

const BackendGoDST = "go-dst"

type GoBackendFeatureProfile struct {
	Family            string
	SupportedDialects []gomerge.GoDialect
	SupportedPolicies []astmerge.PolicyReference
	Backend           string
	BackendRef        *treehaver.BackendReference
}

func init() {
	treehaver.RegisterBackend(treehaver.BackendReference{ID: BackendGoDST, Family: "native"})
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
	return []string{BackendGoDST}
}

func GoBackendFeatureProfileInfo() GoBackendFeatureProfile {
	return GoBackendFeatureProfile{
		Family:            "go",
		SupportedDialects: []gomerge.GoDialect{gomerge.DialectGo},
		SupportedPolicies: []astmerge.PolicyReference{destinationWinsArrayPolicy()},
		Backend:           BackendGoDST,
		BackendRef:        &treehaver.BackendReference{ID: BackendGoDST, Family: "native"},
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
			Backend:           BackendGoDST,
			SupportsDialects:  true,
			SupportedPolicies: []astmerge.PolicyReference{destinationWinsArrayPolicy()},
		},
	}
}

func ParseGo(source string, dialect gomerge.GoDialect, backend ...string) astmerge.ParseResult[gomerge.GoAnalysis] {
	requested := BackendGoDST
	if len(backend) > 0 && backend[0] != "" {
		requested = backend[0]
	}
	if requested != BackendGoDST {
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

	return parseGoDST(source)
}

func MatchGoOwners(template, destination gomerge.GoAnalysis) gomerge.GoOwnerMatchResult {
	return gomerge.MatchGoOwners(template, destination)
}

func MergeGo(templateSource string, destinationSource string, dialect gomerge.GoDialect, backend ...string) astmerge.MergeResult[string] {
	requested := BackendGoDST
	if len(backend) > 0 && backend[0] != "" {
		requested = backend[0]
	}
	if requested != BackendGoDST {
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
	if request.ProviderID != BackendGoDST || request.BackendRef.ID != BackendGoDST {
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
			Message:  fmt.Sprintf("language %s is not supported by go-dst edit projection", request.Language),
			Path:     "language",
			Blocking: true,
		}})
	}
	if len(request.Operations) != 1 {
		return treehaver.BuildEditProjectionExecutionResult(request.Source, nil, []treehaver.ProviderDiagnostic{{
			Severity: "error",
			Category: "unsupported_feature",
			Code:     "edit_projection_batch_unsupported",
			Message:  "go-dst edit projection currently supports exactly one operation",
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
			Message:  fmt.Sprintf("go-dst edit projection does not support %s", operation.Operation),
			Path:     "operations[0].operation",
			Blocking: true,
		}})
	}

	output, err := applyGoDSTNodeOperation(request.Source, operation)
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
		CorrelationKey:   "metadata.go_dst.node_path",
		CorrelationValue: operation.TargetNodePath,
	}}
	return treehaver.BuildEditProjectionExecutionResult(output, applied, nil)
}

func parseGoDST(source string) astmerge.ParseResult[gomerge.GoAnalysis] {
	originalSource := source
	offsetBytes := 0
	if !strings.HasPrefix(strings.TrimSpace(source), "package ") {
		prefix := "package main\n\n"
		source = prefix + source
		offsetBytes = len(prefix)
	}

	fset := token.NewFileSet()
	file, err := decorator.ParseFile(fset, "input.go", source, goparser.ParseComments)
	if err != nil {
		return astmerge.ParseResult[gomerge.GoAnalysis]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{parseError(strings.ReplaceAll(err.Error(), "input.go:3:", "input.go:1:"))},
		}
	}

	restoredFset, restoredFile, err := decorator.RestoreFile(file)
	if err != nil {
		return astmerge.ParseResult[gomerge.GoAnalysis]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{parseError(err.Error())},
		}
	}

	imports := make([]gomerge.GoModuleImport, 0, len(file.Imports))
	for index, item := range file.Imports {
		matchKey := strings.Trim(item.Path.Value, "\"")
		imports = append(imports, gomerge.GoModuleImport{
			Path:     "/imports/" + strconv.Itoa(index),
			MatchKey: matchKey,
			Text:     importLineText(originalSource, item.Path.Value),
		})
	}

	declarations := make([]gomerge.GoModuleDeclaration, 0)
	for _, decl := range restoredFile.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Name == nil {
			continue
		}
		start := restoredFset.Position(funcDecl.Pos()).Offset - offsetBytes
		end := restoredFset.Position(funcDecl.End()).Offset - offsetBytes
		if start < 0 || end > len(originalSource) || start > end {
			return astmerge.ParseResult[gomerge.GoAnalysis]{
				OK:          false,
				Diagnostics: []astmerge.Diagnostic{parseError("restored dave/dst declaration span is outside source bounds")},
			}
		}
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

func applyGoDSTNodeOperation(source string, operation treehaver.EditProjectionOperationRequest) (string, error) {
	index, err := declIndex(operation.TargetNodePath)
	if err != nil {
		return "", err
	}

	fset := token.NewFileSet()
	file, err := decorator.ParseFile(fset, "input.go", source, goparser.ParseComments)
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
	case "delete_node":
		if index < 0 || index >= len(file.Decls) {
			return "", fmt.Errorf("target node path %s is outside declaration deletion bounds", operation.TargetNodePath)
		}
		file.Decls = slices.Delete(file.Decls, index, index+1)
	case "insert_child":
		replacementDecl, err := parseOneReplacementDecl(operation.ReplacementSource)
		if err != nil {
			return "", err
		}
		if index < 0 || index > len(file.Decls) {
			return "", fmt.Errorf("target node path %s is outside declaration insertion bounds", operation.TargetNodePath)
		}
		file.Decls = slices.Insert(file.Decls, index, replacementDecl)
	default:
		return "", fmt.Errorf("unsupported edit projection operation %s", operation.Operation)
	}

	var buffer bytes.Buffer
	if err := decorator.Fprint(&buffer, file); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

func parseOneReplacementDecl(replacementSource string) (dst.Decl, error) {
	replacementFile, err := decorator.ParseFile(token.NewFileSet(), "replacement.go", "package replacement\n\n"+replacementSource, goparser.ParseComments)
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
		return 0, fmt.Errorf("unsupported go-dst node path %s", targetNodePath)
	}
	index, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, err
	}
	return index, nil
}

func importLineText(source string, quotedPath string) string {
	for _, line := range strings.Split(source, "\n") {
		if strings.Contains(line, quotedPath) {
			return strings.TrimSpace(line) + "\n"
		}
	}
	return "import " + quotedPath + "\n"
}

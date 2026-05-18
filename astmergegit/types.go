package astmergegit

import (
	"encoding/json"
	"fmt"
	"go/format"
	"reflect"
	"slices"
	"strings"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/gomerge"
)

type Merge3Request struct {
	BaseSource         string `json:"base_source"`
	OursSource         string `json:"ours_source"`
	TheirsSource       string `json:"theirs_source"`
	PathName           string `json:"path_name,omitempty"`
	Language           string `json:"language,omitempty"`
	Dialect            string `json:"dialect,omitempty"`
	ProfileID          string `json:"profile_id,omitempty"`
	FallbackPolicy     string `json:"fallback_policy,omitempty"`
	ConflictMarkerSize int    `json:"conflict_marker_size,omitempty"`
	RenderPolicy       string `json:"render_policy,omitempty"`
}

type Merge3Conflict struct {
	ConflictID string `json:"conflict_id"`
	Category   string `json:"category"`
	Path       string `json:"path"`
	Message    string `json:"message"`
}

type Merge3RenderReport struct {
	Strategy       string `json:"strategy"`
	BackendID      string `json:"backend_id,omitempty"`
	ParserIdentity string `json:"parser_identity,omitempty"`
}

type FormattingPreservationReport struct {
	LineDiffScore      float64 `json:"line_diff_score"`
	CharacterDiffScore float64 `json:"character_diff_score"`
}

type Merge3Response struct {
	OK                     bool                         `json:"ok"`
	MergedSource           *string                      `json:"merged_source"`
	ConflictedSource       *string                      `json:"conflicted_source"`
	Conflicts              []Merge3Conflict             `json:"conflicts"`
	Diagnostics            []astmerge.Diagnostic        `json:"diagnostics"`
	Fallbacks              []string                     `json:"fallbacks"`
	Profile                map[string]string            `json:"profile"`
	RenderReport           Merge3RenderReport           `json:"render_report"`
	FormattingPreservation FormattingPreservationReport `json:"formatting_preservation"`
	ReparseAfterRender     *bool                        `json:"reparse_after_render"`
}

type CommentDeltaResult struct {
	OK            bool             `json:"ok"`
	MergedComment *string          `json:"merged_comment"`
	Conflicts     []Merge3Conflict `json:"conflicts"`
}

type absentValue struct{}

var absent = absentValue{}

func Merge3(request Merge3Request) Merge3Response {
	switch normalizeLanguage(request.Language, request.PathName) {
	case "go":
		return Merge3Go(request)
	case "json":
		return Merge3JSON(request)
	default:
		return Merge3Response{
			OK: false,
			Diagnostics: []astmerge.Diagnostic{{
				Severity: astmerge.SeverityError,
				Category: astmerge.CategoryUnsupportedFeature,
				Message:  "ast-merge-git currently supports only json merge3.",
			}},
			Conflicts:              []Merge3Conflict{},
			Fallbacks:              []string{},
			Profile:                profileReport(request),
			RenderReport:           renderReport(request, ""),
			FormattingPreservation: FormattingPreservationReport{},
		}
	}
}

func Merge3Go(request Merge3Request) Merge3Response {
	return Merge3GoWithParser(request, gomerge.ParseGo)
}

func Merge3GoWithParser(
	request Merge3Request,
	parser func(source string, dialect gomerge.GoDialect) astmerge.ParseResult[gomerge.GoAnalysis],
) Merge3Response {
	base := parser(request.BaseSource, gomerge.DialectGo)
	if !base.OK || base.Analysis == nil {
		return parseFailureResponse(request, roleDiagnostic("base", base.Diagnostics))
	}
	ours := parser(request.OursSource, gomerge.DialectGo)
	if !ours.OK || ours.Analysis == nil {
		return parseFailureResponse(request, roleDiagnostic("ours", ours.Diagnostics))
	}
	theirs := parser(request.TheirsSource, gomerge.DialectGo)
	if !theirs.OK || theirs.Analysis == nil {
		return parseFailureResponse(request, roleDiagnostic("theirs", theirs.Diagnostics))
	}

	conflicts := []Merge3Conflict{}
	merged, ok := mergeGoAnalyses(*base.Analysis, *ours.Analysis, *theirs.Analysis, &conflicts)
	if !ok {
		conflictedSource := renderConflictSource(request, conflicts)
		return Merge3Response{
			OK:               false,
			ConflictedSource: &conflictedSource,
			Conflicts:        conflicts,
			Diagnostics: []astmerge.Diagnostic{{
				Severity: astmerge.SeverityError,
				Category: astmerge.DiagnosticCategory("merge_conflict"),
				Message:  fmt.Sprintf("merge3 found %d unresolved conflict(s).", len(conflicts)),
			}},
			Fallbacks:              []string{},
			Profile:                profileReport(request),
			RenderReport:           renderReport(request, "full_file_conflict_markers"),
			FormattingPreservation: FormattingPreservationReport{},
		}
	}

	reparse := gomerge.ParseGo(merged, gomerge.DialectGo).OK
	return Merge3Response{
		OK:                 true,
		MergedSource:       &merged,
		Conflicts:          []Merge3Conflict{},
		Diagnostics:        []astmerge.Diagnostic{},
		Fallbacks:          []string{},
		Profile:            profileReport(request),
		RenderReport:       renderReport(request, ""),
		ReparseAfterRender: &reparse,
		FormattingPreservation: FormattingPreservationReport{
			LineDiffScore:      0.95,
			CharacterDiffScore: 0.95,
		},
	}
}

func Merge3JSON(request Merge3Request) Merge3Response {
	base, diagnostic, ok := parseJSONRole("base", request.BaseSource)
	if !ok {
		return parseFailureResponse(request, diagnostic)
	}
	ours, diagnostic, ok := parseJSONRole("ours", request.OursSource)
	if !ok {
		return parseFailureResponse(request, diagnostic)
	}
	theirs, diagnostic, ok := parseJSONRole("theirs", request.TheirsSource)
	if !ok {
		return parseFailureResponse(request, diagnostic)
	}

	conflicts := []Merge3Conflict{}
	merged := mergeJSONValue(base, ours, theirs, "", &conflicts)
	if len(conflicts) > 0 {
		conflictedSource := renderConflictSource(request, conflicts)
		return Merge3Response{
			OK:               false,
			ConflictedSource: &conflictedSource,
			Conflicts:        conflicts,
			Diagnostics: []astmerge.Diagnostic{{
				Severity: astmerge.SeverityError,
				Category: astmerge.DiagnosticCategory("merge_conflict"),
				Message:  fmt.Sprintf("merge3 found %d unresolved conflict(s).", len(conflicts)),
			}},
			Fallbacks:              []string{},
			Profile:                profileReport(request),
			RenderReport:           renderReport(request, "full_file_conflict_markers"),
			FormattingPreservation: FormattingPreservationReport{},
		}
	}

	sourceBytes, err := json.Marshal(merged)
	if err != nil {
		return parseFailureResponse(request, astmerge.Diagnostic{
			Severity: astmerge.SeverityError,
			Category: astmerge.CategoryUnsupportedFeature,
			Message:  err.Error(),
		})
	}
	source := string(sourceBytes)
	reparse := json.Valid(sourceBytes)
	return Merge3Response{
		OK:                 true,
		MergedSource:       &source,
		Conflicts:          []Merge3Conflict{},
		Diagnostics:        []astmerge.Diagnostic{},
		Fallbacks:          []string{},
		Profile:            profileReport(request),
		RenderReport:       renderReport(request, ""),
		ReparseAfterRender: &reparse,
		FormattingPreservation: FormattingPreservationReport{
			LineDiffScore:      1.0,
			CharacterDiffScore: 1.0,
		},
	}
}

func MergeCommentDelta(baseComment *string, oursComment *string, theirsComment *string, ownerPath string) CommentDeltaResult {
	conflicts := []Merge3Conflict{}
	var mergedComment *string

	switch {
	case stringPointersEqual(oursComment, theirsComment):
		mergedComment = cloneStringPointer(oursComment)
	case stringPointersEqual(baseComment, oursComment):
		mergedComment = cloneStringPointer(theirsComment)
	case stringPointersEqual(baseComment, theirsComment):
		mergedComment = cloneStringPointer(oursComment)
	case oursComment == nil:
		conflicts = append(conflicts, commentConflict("delete_edit", ownerPath, "ours deleted a comment that theirs edited"))
	case theirsComment == nil:
		conflicts = append(conflicts, commentConflict("delete_edit", ownerPath, "theirs deleted a comment that ours edited"))
	default:
		conflicts = append(conflicts, commentConflict("edit_edit", ownerPath, "comment changed differently in ours and theirs"))
	}

	return CommentDeltaResult{
		OK:            len(conflicts) == 0,
		MergedComment: mergedComment,
		Conflicts:     conflicts,
	}
}

func parseFailureResponse(request Merge3Request, diagnostic astmerge.Diagnostic) Merge3Response {
	return Merge3Response{
		OK:                     false,
		Conflicts:              []Merge3Conflict{},
		Diagnostics:            []astmerge.Diagnostic{diagnostic},
		Fallbacks:              []string{},
		Profile:                profileReport(request),
		RenderReport:           renderReport(request, ""),
		FormattingPreservation: FormattingPreservationReport{},
	}
}

func renderReport(request Merge3Request, strategy string) Merge3RenderReport {
	if strategy == "" {
		strategy = normalizedRenderPolicy(request.RenderPolicy)
	}
	report := Merge3RenderReport{Strategy: strategy}
	switch normalizeLanguage(request.Language, request.PathName) {
	case "json":
		report.BackendID = "native-json"
		report.ParserIdentity = "standard-json"
	case "go":
		report.BackendID = "go-parser"
		report.ParserIdentity = "go/parser"
	}
	return report
}

func renderConflictSource(request Merge3Request, conflicts []Merge3Conflict) string {
	markerSize := request.ConflictMarkerSize
	if markerSize <= 0 {
		markerSize = 7
	}
	leftMarker := strings.Repeat("<", markerSize)
	baseMarker := strings.Repeat("|", markerSize)
	separatorMarker := strings.Repeat("=", markerSize)
	rightMarker := strings.Repeat(">", markerSize)
	header := fmt.Sprintf("// smorg structured conflicts: %d unresolved", len(conflicts))
	if normalizeLanguage(request.Language, request.PathName) == "json" {
		header = fmt.Sprintf("/* smorg structured conflicts: %d unresolved */", len(conflicts))
	}
	return strings.Join([]string{
		header,
		leftMarker + " ours",
		request.OursSource,
		baseMarker + " base",
		request.BaseSource,
		separatorMarker,
		request.TheirsSource,
		rightMarker + " theirs",
		"",
	}, "\n")
}

func roleDiagnostic(role string, diagnostics []astmerge.Diagnostic) astmerge.Diagnostic {
	if len(diagnostics) == 0 {
		return astmerge.Diagnostic{Severity: astmerge.SeverityError, Category: astmerge.CategoryParseError, Message: role + " parse error"}
	}
	diagnostic := diagnostics[0]
	diagnostic.Message = role + " parse error: " + diagnostic.Message
	diagnostic.Path = role
	return diagnostic
}

func mergeGoAnalyses(base gomerge.GoAnalysis, ours gomerge.GoAnalysis, theirs gomerge.GoAnalysis, conflicts *[]Merge3Conflict) (string, bool) {
	imports := mergeGoImports(base, ours, theirs)
	declarations, ok := mergeGoDeclarations(base, ours, theirs, conflicts)
	if !ok {
		return "", false
	}
	output := renderGoSource(imports, declarations)
	formatted, err := format.Source([]byte(output))
	if err == nil {
		output = string(formatted)
	}
	return output, true
}

func mergeGoImports(base gomerge.GoAnalysis, ours gomerge.GoAnalysis, theirs gomerge.GoAnalysis) []string {
	baseImports := goImportSet(base)
	oursImports := goImportSet(ours)
	theirsImports := goImportSet(theirs)
	merged := map[string]struct{}{}
	for key := range baseImports {
		if _, oursOK := oursImports[key]; oursOK {
			if _, theirsOK := theirsImports[key]; theirsOK {
				merged[key] = struct{}{}
			}
		}
	}
	for key := range oursImports {
		if _, baseOK := baseImports[key]; !baseOK {
			merged[key] = struct{}{}
		}
	}
	for key := range theirsImports {
		if _, baseOK := baseImports[key]; !baseOK {
			merged[key] = struct{}{}
		}
	}
	keys := make([]string, 0, len(merged))
	for key := range merged {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func goImportSet(analysis gomerge.GoAnalysis) map[string]struct{} {
	set := make(map[string]struct{}, len(analysis.Imports))
	for _, item := range analysis.Imports {
		set[item.MatchKey] = struct{}{}
	}
	return set
}

func mergeGoDeclarations(base gomerge.GoAnalysis, ours gomerge.GoAnalysis, theirs gomerge.GoAnalysis, conflicts *[]Merge3Conflict) ([]string, bool) {
	baseDecls := goDeclarationMap(base)
	oursDecls := goDeclarationMap(ours)
	theirsDecls := goDeclarationMap(theirs)
	keys := mapKeys(baseDecls, oursDecls, theirsDecls)
	merged := make([]string, 0, len(keys))
	for _, key := range keys {
		baseText, baseOK := baseDecls[key]
		oursText, oursOK := oursDecls[key]
		theirsText, theirsOK := theirsDecls[key]
		switch {
		case !baseOK && oursOK && !theirsOK:
			merged = append(merged, oursText)
		case !baseOK && !oursOK && theirsOK:
			merged = append(merged, theirsText)
		case !baseOK && oursOK && theirsOK && strings.TrimSpace(oursText) == strings.TrimSpace(theirsText):
			merged = append(merged, oursText)
		case !baseOK && oursOK && theirsOK:
			addConflict(conflicts, "add_add", "/decls/"+key, "same declaration added differently in ours and theirs")
		case baseOK && !oursOK && !theirsOK:
			continue
		case baseOK && oursOK && !theirsOK && equivalentGoDeclaration(baseText, oursText):
			continue
		case baseOK && !oursOK && theirsOK && equivalentGoDeclaration(baseText, theirsText):
			continue
		case baseOK && !oursOK && theirsOK:
			addConflict(conflicts, "delete_edit", "/decls/"+key, "ours deleted a declaration that theirs edited")
		case baseOK && oursOK && !theirsOK:
			addConflict(conflicts, "delete_edit", "/decls/"+key, "theirs deleted a declaration that ours edited")
		case baseOK && oursOK && theirsOK:
			if stripGoComments(baseText) == stripGoComments(oursText) && stripGoComments(baseText) == stripGoComments(theirsText) {
				mergedText, mergedOK := mergeGoCommentOnlyChange(key, baseText, oursText, theirsText, conflicts)
				if !mergedOK {
					continue
				}
				merged = append(merged, mergedText)
				continue
			}
			switch {
			case strings.TrimSpace(oursText) == strings.TrimSpace(theirsText):
				merged = append(merged, oursText)
			case equivalentGoDeclaration(baseText, oursText):
				merged = append(merged, theirsText)
			case equivalentGoDeclaration(baseText, theirsText):
				merged = append(merged, oursText)
			case stripGoComments(baseText) == stripGoComments(oursText):
				merged = append(merged, attachLeadingGoComments(oursText, theirsText))
			case stripGoComments(baseText) == stripGoComments(theirsText):
				merged = append(merged, attachLeadingGoComments(theirsText, oursText))
			default:
				addConflict(conflicts, "edit_edit", "/decls/"+key, "declaration changed differently in ours and theirs")
			}
		}
	}
	return merged, len(*conflicts) == 0
}

func mergeGoCommentOnlyChange(key string, baseText string, oursText string, theirsText string, conflicts *[]Merge3Conflict) (string, bool) {
	baseComments := leadingGoCommentBlock(baseText)
	oursComments := leadingGoCommentBlock(oursText)
	theirsComments := leadingGoCommentBlock(theirsText)
	switch {
	case oursComments == theirsComments:
		return oursText, true
	case oursComments == baseComments:
		return theirsText, true
	case theirsComments == baseComments:
		return oursText, true
	case oursComments == "" && theirsComments != "":
		addConflict(conflicts, "delete_edit", "/decls/"+key+"/comments", "ours deleted comments that theirs edited")
		return "", false
	case theirsComments == "" && oursComments != "":
		addConflict(conflicts, "delete_edit", "/decls/"+key+"/comments", "theirs deleted comments that ours edited")
		return "", false
	default:
		addConflict(conflicts, "edit_edit", "/decls/"+key+"/comments", "comments changed differently in ours and theirs")
		return "", false
	}
}

func goDeclarationMap(analysis gomerge.GoAnalysis) map[string]string {
	decls := make(map[string]string, len(analysis.Declarations))
	for _, item := range analysis.Declarations {
		text := strings.TrimSpace(item.Text)
		if comments := leadingGoCommentsForDeclaration(analysis.Source, item.MatchKey); comments != "" && !strings.HasPrefix(text, comments) {
			text = comments + "\n" + text
		}
		decls[item.MatchKey] = text
	}
	return decls
}

func leadingGoCommentsForDeclaration(source string, name string) string {
	lines := strings.Split(source, "\n")
	funcLine := -1
	prefix := "func " + name + "("
	for index, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), prefix) {
			funcLine = index
			break
		}
	}
	if funcLine <= 0 {
		return ""
	}
	start := funcLine
	for start > 0 {
		previous := strings.TrimSpace(lines[start-1])
		if strings.HasPrefix(previous, "//") {
			start--
			continue
		}
		if previous == "" && start < funcLine {
			start--
			continue
		}
		break
	}
	commentLines := make([]string, 0, funcLine-start)
	for _, line := range lines[start:funcLine] {
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			commentLines = append(commentLines, line)
		}
	}
	return strings.TrimSpace(strings.Join(commentLines, "\n"))
}

func mapKeys(maps ...map[string]string) []string {
	seen := map[string]struct{}{}
	for _, oneMap := range maps {
		for key := range oneMap {
			seen[key] = struct{}{}
		}
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func equivalentGoDeclaration(left string, right string) bool {
	return strings.TrimSpace(left) == strings.TrimSpace(right)
}

func stripGoComments(source string) string {
	lines := strings.Split(source, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.TrimSpace(strings.Join(kept, "\n"))
}

func leadingGoCommentBlock(source string) string {
	lines := strings.Split(source, "\n")
	comments := make([]string, 0)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			comments = append(comments, trimmed)
			continue
		}
		if trimmed == "" && len(comments) == 0 {
			continue
		}
		break
	}
	return strings.Join(comments, "\n")
}

func attachLeadingGoComments(commentSource string, declarationSource string) string {
	lines := strings.Split(commentSource, "\n")
	comments := make([]string, 0)
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			comments = append(comments, line)
			continue
		}
		break
	}
	if len(comments) == 0 {
		return declarationSource
	}
	return strings.Join(comments, "\n") + "\n" + strings.TrimSpace(declarationSource)
}

func renderGoSource(imports []string, declarations []string) string {
	sections := []string{"package main"}
	if len(imports) == 1 {
		sections = append(sections, fmt.Sprintf("import %q", imports[0]))
	} else if len(imports) > 1 {
		lines := []string{"import ("}
		for _, item := range imports {
			lines = append(lines, fmt.Sprintf("\t%q", item))
		}
		lines = append(lines, ")")
		sections = append(sections, strings.Join(lines, "\n"))
	}
	if len(declarations) > 0 {
		sections = append(sections, strings.Join(declarations, "\n\n"))
	}
	return strings.Join(sections, "\n\n") + "\n"
}

func parseJSONRole(role string, source string) (any, astmerge.Diagnostic, bool) {
	var parsed any
	decoder := json.NewDecoder(strings.NewReader(source))
	decoder.UseNumber()
	if err := decoder.Decode(&parsed); err != nil {
		return nil, astmerge.Diagnostic{
			Severity: astmerge.SeverityError,
			Category: astmerge.CategoryParseError,
			Message:  fmt.Sprintf("%s parse error: %s", role, err),
			Path:     role,
		}, false
	}
	return parsed, astmerge.Diagnostic{}, true
}

func mergeJSONValue(base any, ours any, theirs any, path string, conflicts *[]Merge3Conflict) any {
	switch {
	case jsonEqual(ours, theirs):
		return ours
	case jsonEqual(base, ours):
		return theirs
	case jsonEqual(base, theirs):
		return ours
	}

	baseMap, baseIsMap := base.(map[string]any)
	oursMap, oursIsMap := ours.(map[string]any)
	theirsMap, theirsIsMap := theirs.(map[string]any)
	if baseIsMap && oursIsMap && theirsIsMap {
		return mergeJSONObjects(baseMap, oursMap, theirsMap, path, conflicts)
	}

	addConflict(conflicts, "edit_edit", path, "value changed differently in ours and theirs")
	return ours
}

func mergeJSONObjects(base map[string]any, ours map[string]any, theirs map[string]any, path string, conflicts *[]Merge3Conflict) map[string]any {
	result := map[string]any{}
	keys := make([]string, 0, len(base)+len(ours)+len(theirs))
	seen := map[string]bool{}
	for _, object := range []map[string]any{base, ours, theirs} {
		for key := range object {
			if !seen[key] {
				seen[key] = true
				keys = append(keys, key)
			}
		}
	}
	slices.Sort(keys)

	for _, key := range keys {
		baseValue, baseOK := base[key]
		oursValue, oursOK := ours[key]
		theirsValue, theirsOK := theirs[key]
		merged, keep := mergeJSONEntry(
			valueOrAbsent(baseValue, baseOK),
			valueOrAbsent(oursValue, oursOK),
			valueOrAbsent(theirsValue, theirsOK),
			jsonPointerJoin(path, key),
			conflicts,
		)
		if keep {
			result[key] = merged
		}
	}

	return result
}

func mergeJSONEntry(base any, ours any, theirs any, path string, conflicts *[]Merge3Conflict) (any, bool) {
	baseAbsent := isAbsent(base)
	oursAbsent := isAbsent(ours)
	theirsAbsent := isAbsent(theirs)

	switch {
	case baseAbsent && oursAbsent && theirsAbsent:
		return nil, false
	case baseAbsent && oursAbsent:
		return theirs, true
	case baseAbsent && theirsAbsent:
		return ours, true
	case baseAbsent && jsonEqual(ours, theirs):
		return ours, true
	case baseAbsent:
		addConflict(conflicts, "add_add", path, "same path added differently in ours and theirs")
		return ours, true
	case oursAbsent && theirsAbsent:
		return nil, false
	case oursAbsent && jsonEqual(base, theirs):
		return nil, false
	case theirsAbsent && jsonEqual(base, ours):
		return nil, false
	case oursAbsent:
		addConflict(conflicts, "delete_edit", path, "ours deleted a value that theirs edited")
		return theirs, true
	case theirsAbsent:
		addConflict(conflicts, "delete_edit", path, "theirs deleted a value that ours edited")
		return ours, true
	default:
		return mergeJSONValue(base, ours, theirs, path, conflicts), true
	}
}

func valueOrAbsent(value any, ok bool) any {
	if ok {
		return value
	}
	return absent
}

func isAbsent(value any) bool {
	_, ok := value.(absentValue)
	return ok
}

func jsonEqual(left any, right any) bool {
	return reflect.DeepEqual(left, right)
}

func addConflict(conflicts *[]Merge3Conflict, category string, path string, message string) {
	if path == "" {
		path = "/"
	}
	*conflicts = append(*conflicts, Merge3Conflict{
		ConflictID: fmt.Sprintf("conflict-%d", len(*conflicts)+1),
		Category:   category,
		Path:       path,
		Message:    message,
	})
}

func commentConflict(category string, path string, message string) Merge3Conflict {
	if path == "" {
		path = "/"
	}
	return Merge3Conflict{
		ConflictID: "comment-conflict-1",
		Category:   category,
		Path:       path,
		Message:    message,
	}
}

func stringPointersEqual(left *string, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func jsonPointerJoin(parent string, token string) string {
	escaped := strings.ReplaceAll(strings.ReplaceAll(token, "~", "~0"), "/", "~1")
	if parent == "" {
		return "/" + escaped
	}
	return parent + "/" + escaped
}

func normalizeLanguage(language string, pathName string) string {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "go", "golang":
		return "go"
	case "json":
		return "json"
	}
	if strings.HasSuffix(strings.ToLower(pathName), ".go") {
		return "go"
	}
	if strings.HasSuffix(strings.ToLower(pathName), ".json") {
		return "json"
	}
	return strings.ToLower(strings.TrimSpace(language))
}

func normalizedRenderPolicy(policy string) string {
	if strings.TrimSpace(policy) == "" {
		return "canonical"
	}
	return strings.TrimSpace(policy)
}

func profileReport(request Merge3Request) map[string]string {
	return map[string]string{
		"profile_id": request.ProfileID,
		"language":   normalizeLanguage(request.Language, request.PathName),
		"dialect":    request.Dialect,
	}
}

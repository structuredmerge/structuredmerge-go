package astmergegit

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
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
	Strategy string `json:"strategy"`
}

type FormattingPreservationReport struct {
	LineDiffScore      float64 `json:"line_diff_score"`
	CharacterDiffScore float64 `json:"character_diff_score"`
}

type Merge3Response struct {
	OK                     bool                         `json:"ok"`
	MergedSource           *string                      `json:"merged_source"`
	Conflicts              []Merge3Conflict             `json:"conflicts"`
	Diagnostics            []astmerge.Diagnostic        `json:"diagnostics"`
	Fallbacks              []string                     `json:"fallbacks"`
	Profile                map[string]string            `json:"profile"`
	RenderReport           Merge3RenderReport           `json:"render_report"`
	FormattingPreservation FormattingPreservationReport `json:"formatting_preservation"`
	ReparseAfterRender     *bool                        `json:"reparse_after_render"`
}

type absentValue struct{}

var absent = absentValue{}

func Merge3(request Merge3Request) Merge3Response {
	switch normalizeLanguage(request.Language, request.PathName) {
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
			RenderReport:           Merge3RenderReport{Strategy: normalizedRenderPolicy(request.RenderPolicy)},
			FormattingPreservation: FormattingPreservationReport{},
		}
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
		return Merge3Response{
			OK:        false,
			Conflicts: conflicts,
			Diagnostics: []astmerge.Diagnostic{{
				Severity: astmerge.SeverityError,
				Category: astmerge.DiagnosticCategory("merge_conflict"),
				Message:  fmt.Sprintf("merge3 found %d unresolved conflict(s).", len(conflicts)),
			}},
			Fallbacks:              []string{},
			Profile:                profileReport(request),
			RenderReport:           Merge3RenderReport{Strategy: normalizedRenderPolicy(request.RenderPolicy)},
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
		RenderReport:       Merge3RenderReport{Strategy: normalizedRenderPolicy(request.RenderPolicy)},
		ReparseAfterRender: &reparse,
		FormattingPreservation: FormattingPreservationReport{
			LineDiffScore:      1.0,
			CharacterDiffScore: 1.0,
		},
	}
}

func parseFailureResponse(request Merge3Request, diagnostic astmerge.Diagnostic) Merge3Response {
	return Merge3Response{
		OK:                     false,
		Conflicts:              []Merge3Conflict{},
		Diagnostics:            []astmerge.Diagnostic{diagnostic},
		Fallbacks:              []string{},
		Profile:                profileReport(request),
		RenderReport:           Merge3RenderReport{Strategy: normalizedRenderPolicy(request.RenderPolicy)},
		FormattingPreservation: FormattingPreservationReport{},
	}
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

func jsonPointerJoin(parent string, token string) string {
	escaped := strings.ReplaceAll(strings.ReplaceAll(token, "~", "~0"), "/", "~1")
	if parent == "" {
		return "/" + escaped
	}
	return parent + "/" + escaped
}

func normalizeLanguage(language string, pathName string) string {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "json":
		return "json"
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

package rubymerge

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/treehaver"
)

type RubyDialect string

const DialectRuby RubyDialect = "ruby"

type RubyOwnerKind string

const (
	OwnerRequire     RubyOwnerKind = "require"
	OwnerDeclaration RubyOwnerKind = "declaration"
)

type RubyOwner struct {
	Path      string
	OwnerKind RubyOwnerKind
	MatchKey  string
}

type RubyOwnerMatch struct {
	TemplatePath    string
	DestinationPath string
}

type RubyOwnerMatchResult struct {
	Matched              []RubyOwnerMatch
	UnmatchedTemplate    []string
	UnmatchedDestination []string
}

type RubyAnalysis struct {
	Dialect            RubyDialect
	Source             string
	Owners             []RubyOwner
	DiscoveredSurfaces []astmerge.DiscoveredSurface
}

type AppliedChildOutput struct {
	OperationID string `json:"operation_id"`
	Output      string `json:"output"`
}

type NestedChildOutput struct {
	SurfaceAddress string `json:"surface_address"`
	Output         string `json:"output"`
}

func (RubyAnalysis) Kind() string {
	return "ruby"
}

type RubyFeatureProfile struct {
	Family            string
	SupportedDialects []RubyDialect
	SupportedPolicies []astmerge.PolicyReference
}

type RubyBackendFeatureProfile struct {
	Family            string
	SupportedDialects []RubyDialect
	SupportedPolicies []astmerge.PolicyReference
	Backend           string
	BackendRef        *treehaver.BackendReference
	SupportsDialects  bool
}

type commentEntry struct {
	Line int
	Raw  string
}

type rubyRequireEntry struct {
	Text string
}

type rubyDeclarationEntry struct {
	Path string
	Text string
}

var (
	requirePattern   = regexp.MustCompile(`^\s*require(?:_relative)?\s+["']([^"']+)["']`)
	classPattern     = regexp.MustCompile(`^\s*class\s+([A-Z]\w*(?:::\w+)*)`)
	modulePattern    = regexp.MustCompile(`^\s*module\s+([A-Z]\w*(?:::\w+)*)`)
	defPattern       = regexp.MustCompile(`^\s*def\s+(?:self\.)?([a-zA-Z_]\w*[!?=]?)`)
	exampleTag       = regexp.MustCompile(`^@example\b(.*)$`)
	tagPrefix        = regexp.MustCompile(`^@[a-z_]+\b`)
	directiveLine    = regexp.MustCompile(`^(?::nocov:|[\w-]+:(?:freeze|unfreeze))$`)
	exampleLanguage  = regexp.MustCompile(`^\[(?P<language>[^\]]+)\]`)
	magicCommentKeys = []string{"coding", "encoding", "frozen_string_literal", "shareable_constant_value", "typed", "warn_indent"}
)

func destinationWinsArrayPolicy() astmerge.PolicyReference {
	return astmerge.PolicyReference{Surface: astmerge.PolicySurfaceArray, Name: "destination_wins_array"}
}

func configurationError(message string) astmerge.Diagnostic {
	return astmerge.Diagnostic{
		Severity: astmerge.SeverityError,
		Category: astmerge.CategoryConfigurationError,
		Message:  message,
	}
}

func rubyParseRequest(source string) treehaver.ParserRequest {
	return treehaver.ParserRequest{Source: source, Language: "ruby", Dialect: "ruby"}
}

func normalizeSource(source string) string {
	return strings.ReplaceAll(strings.ReplaceAll(source, "\r\n", "\n"), "\r", "\n")
}

func commentLine(line string) bool {
	return strings.HasPrefix(strings.TrimLeft(line, " \t"), "#")
}

func normalizeCommentContent(raw string) string {
	trimmed := strings.TrimLeft(raw, " \t")
	trimmed = strings.TrimPrefix(trimmed, "#")
	trimmed = strings.TrimPrefix(trimmed, " ")
	return strings.TrimSpace(trimmed)
}

func docCommentContent(raw string) bool {
	content := normalizeCommentContent(raw)
	if content == "" || directiveLine.MatchString(content) {
		return false
	}
	for _, prefix := range magicCommentKeys {
		if strings.HasPrefix(content, prefix+":") {
			return false
		}
	}
	return true
}

func commentPrefix(raw string) string {
	prefix := regexp.MustCompile(`^\s*#\s?`).FindString(raw)
	if prefix == "" {
		return "# "
	}
	return prefix
}

func declaredExampleLanguage(rest string) string {
	match := exampleLanguage.FindStringSubmatch(strings.TrimSpace(rest))
	if len(match) < 2 {
		return ""
	}
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(match[1])), "-", "_")
}

func surfacesForOwner(ownerName string, commentEntries []commentEntry) []astmerge.DiscoveredSurface {
	filtered := make([]commentEntry, 0)
	for _, entry := range commentEntries {
		if docCommentContent(entry.Raw) {
			filtered = append(filtered, entry)
		}
	}
	if len(filtered) == 0 {
		return nil
	}

	docSurface := astmerge.DiscoveredSurface{
		SurfaceKind:       "ruby_doc_comment",
		DeclaredLanguage:  "yard",
		EffectiveLanguage: "yard",
		Address:           "document[0] > ruby_doc_comment[" + ownerName + "]",
		ParentAddress:     "document[0]",
		Span:              &astmerge.SurfaceSpan{StartLine: filtered[0].Line, EndLine: filtered[len(filtered)-1].Line},
		Owner: astmerge.SurfaceOwnerRef{
			Kind:    astmerge.SurfaceOwnerOwnedRegion,
			Address: "/declarations/" + ownerName,
		},
		ReconstructionStrategy: "rewrite_with_prefix_preservation",
		Metadata: map[string]any{
			"owner_signature": ownerName,
			"comment_prefix":  commentPrefix(filtered[0].Raw),
			"entries": func() []map[string]any {
				values := make([]map[string]any, 0, len(filtered))
				for _, entry := range filtered {
					values = append(values, map[string]any{"line": entry.Line, "raw": entry.Raw})
				}
				return values
			}(),
		},
	}

	normalized := make([]string, 0, len(filtered))
	for _, entry := range filtered {
		normalized = append(normalized, normalizeCommentContent(entry.Raw))
	}
	surfaces := []astmerge.DiscoveredSurface{docSurface}
	for tagIndex, content := range normalized {
		match := exampleTag.FindStringSubmatch(content)
		if len(match) == 0 {
			continue
		}
		bodyStart := tagIndex + 1
		bodyEnd := len(normalized)
		for index := bodyStart; index < len(normalized); index++ {
			if tagPrefix.MatchString(normalized[index]) {
				bodyEnd = index
				break
			}
		}
		if bodyStart >= bodyEnd {
			continue
		}
		bodyEntries := filtered[bodyStart:bodyEnd]
		if len(bodyEntries) == 0 {
			continue
		}
		language := declaredExampleLanguage(match[1])
		if language == "" {
			language = "ruby"
		}
		surfaces = append(surfaces, astmerge.DiscoveredSurface{
			SurfaceKind:       "yard_example_block",
			DeclaredLanguage:  language,
			EffectiveLanguage: language,
			Address:           docSurface.Address + " > yard_example[" + strconv.Itoa(tagIndex) + "]",
			ParentAddress:     docSurface.Address,
			Span:              &astmerge.SurfaceSpan{StartLine: bodyEntries[0].Line, EndLine: bodyEntries[len(bodyEntries)-1].Line},
			Owner: astmerge.SurfaceOwnerRef{
				Kind:    astmerge.SurfaceOwnerOwnedRegion,
				Address: docSurface.Address,
			},
			ReconstructionStrategy: "rewrite_with_prefix_preservation",
			Metadata: map[string]any{
				"tag_kind":       "example",
				"tag_index":      tagIndex,
				"tag_text":       normalized[tagIndex],
				"comment_prefix": docSurface.Metadata["comment_prefix"],
			},
		})
	}

	return surfaces
}

func analyzeRubyDocument(source string) RubyAnalysis {
	normalized := normalizeSource(source)
	lines := strings.Split(normalized, "\n")
	requires := make([]RubyOwner, 0)
	declarations := make([]RubyOwner, 0)
	surfaces := make([]astmerge.DiscoveredSurface, 0)
	pendingComments := make([]commentEntry, 0)

	for index, line := range lines {
		lineNumber := index + 1
		stripped := strings.TrimSpace(line)
		if commentLine(line) {
			pendingComments = append(pendingComments, commentEntry{Line: lineNumber, Raw: line})
			continue
		}
		if stripped == "" {
			pendingComments = nil
			continue
		}
		if match := requirePattern.FindStringSubmatch(line); len(match) > 1 {
			requires = append(requires, RubyOwner{
				Path:      "/requires/" + strconv.Itoa(len(requires)),
				OwnerKind: OwnerRequire,
				MatchKey:  match[1],
			})
			pendingComments = nil
			continue
		}
		name := ""
		switch {
		case classPattern.MatchString(line):
			name = classPattern.FindStringSubmatch(line)[1]
		case modulePattern.MatchString(line):
			name = modulePattern.FindStringSubmatch(line)[1]
		case defPattern.MatchString(line):
			name = defPattern.FindStringSubmatch(line)[1]
		}
		if name != "" {
			declarations = append(declarations, RubyOwner{
				Path:      "/declarations/" + name,
				OwnerKind: OwnerDeclaration,
				MatchKey:  name,
			})
			surfaces = append(surfaces, surfacesForOwner(name, pendingComments)...)
			pendingComments = nil
			continue
		}
		pendingComments = nil
	}

	owners := append(requires, declarations...)
	slices.SortFunc(owners, func(left, right RubyOwner) int {
		return strings.Compare(left.Path, right.Path)
	})
	return RubyAnalysis{
		Dialect:            DialectRuby,
		Source:             normalized,
		Owners:             owners,
		DiscoveredSurfaces: surfaces,
	}
}

func RubyFeatureProfileInfo() RubyFeatureProfile {
	return RubyFeatureProfile{
		Family:            "ruby",
		SupportedDialects: []RubyDialect{DialectRuby},
		SupportedPolicies: []astmerge.PolicyReference{destinationWinsArrayPolicy()},
	}
}

func AvailableRubyBackends() []string {
	return []string{"kreuzberg-language-pack"}
}

func RubyBackendFeatureProfileInfo() RubyBackendFeatureProfile {
	return RubyBackendFeatureProfile{
		Family:            "ruby",
		SupportedDialects: []RubyDialect{DialectRuby},
		SupportedPolicies: []astmerge.PolicyReference{destinationWinsArrayPolicy()},
		Backend:           "kreuzberg-language-pack",
		BackendRef:        &treehaver.KreuzbergLanguagePackBackend,
		SupportsDialects:  true,
	}
}

func RubyPlanContext() astmerge.ConformanceFamilyPlanContext {
	backendProfile := RubyBackendFeatureProfileInfo()
	return astmerge.ConformanceFamilyPlanContext{
		FamilyProfile: astmerge.FamilyFeatureProfile{
			Family:            "ruby",
			SupportedDialects: []string{"ruby"},
			SupportedPolicies: []astmerge.PolicyReference{destinationWinsArrayPolicy()},
		},
		FeatureProfile: &astmerge.ConformanceFeatureProfileView{
			Backend:           backendProfile.Backend,
			SupportsDialects:  backendProfile.SupportsDialects,
			SupportedPolicies: backendProfile.SupportedPolicies,
		},
	}
}

func ParseRuby(source string, _dialect RubyDialect) astmerge.ParseResult[RubyAnalysis] {
	parsed := treehaver.ParseWithLanguagePack(rubyParseRequest(source))
	if !parsed.OK {
		return astmerge.ParseResult[RubyAnalysis]{OK: false, Diagnostics: parsed.Diagnostics}
	}
	analysis := analyzeRubyDocument(source)
	return astmerge.ParseResult[RubyAnalysis]{OK: true, Diagnostics: []astmerge.Diagnostic{}, Analysis: &analysis}
}

func MatchRubyOwners(template RubyAnalysis, destination RubyAnalysis) RubyOwnerMatchResult {
	destinationOwners := make(map[string]struct{}, len(destination.Owners))
	for _, owner := range destination.Owners {
		destinationOwners[owner.Path] = struct{}{}
	}
	templateOwners := make(map[string]struct{}, len(template.Owners))
	for _, owner := range template.Owners {
		templateOwners[owner.Path] = struct{}{}
	}

	matched := make([]RubyOwnerMatch, 0)
	unmatchedTemplate := make([]string, 0)
	for _, owner := range template.Owners {
		if _, ok := destinationOwners[owner.Path]; ok {
			matched = append(matched, RubyOwnerMatch{TemplatePath: owner.Path, DestinationPath: owner.Path})
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
	return RubyOwnerMatchResult{
		Matched:              matched,
		UnmatchedTemplate:    unmatchedTemplate,
		UnmatchedDestination: unmatchedDestination,
	}
}

func collectRubyRequireEntries(source string) []rubyRequireEntry {
	lines := strings.Split(normalizeSource(source), "\n")
	entries := make([]rubyRequireEntry, 0)
	for _, line := range lines {
		if requirePattern.MatchString(line) {
			entries = append(entries, rubyRequireEntry{Text: strings.TrimRight(line, "\n")})
		}
	}
	return entries
}

func collectRubyDeclarationEntries(source string) []rubyDeclarationEntry {
	lines := strings.Split(normalizeSource(source), "\n")
	entries := make([]rubyDeclarationEntry, 0)
	pendingComments := make([]int, 0)

	for index := 0; index < len(lines); index++ {
		line := lines[index]
		stripped := strings.TrimSpace(line)
		if commentLine(line) {
			pendingComments = append(pendingComments, index)
			continue
		}
		if stripped == "" {
			pendingComments = nil
			continue
		}
		if requirePattern.MatchString(line) {
			pendingComments = nil
			continue
		}

		name := ""
		switch {
		case classPattern.MatchString(line):
			name = classPattern.FindStringSubmatch(line)[1]
		case modulePattern.MatchString(line):
			name = modulePattern.FindStringSubmatch(line)[1]
		case defPattern.MatchString(line):
			name = defPattern.FindStringSubmatch(line)[1]
		}
		if name == "" {
			pendingComments = nil
			continue
		}

		start := index
		if len(pendingComments) > 0 {
			start = pendingComments[0]
		}
		depth := 1
		cursor := index + 1
		for ; cursor < len(lines); cursor++ {
			candidate := strings.TrimSpace(lines[cursor])
			if classPattern.MatchString(candidate) || modulePattern.MatchString(candidate) || defPattern.MatchString(candidate) {
				depth++
			}
			if candidate == "end" {
				depth--
				if depth == 0 {
					cursor++
					break
				}
			}
		}

		entries = append(entries, rubyDeclarationEntry{
			Path: "/declarations/" + name,
			Text: strings.TrimSpace(strings.Join(lines[start:cursor], "\n")),
		})
		index = cursor - 1
		pendingComments = nil
	}

	return entries
}

func MergeRuby(templateSource string, destinationSource string, dialect RubyDialect) astmerge.MergeResult[string] {
	template := ParseRuby(templateSource, dialect)
	if !template.OK || template.Analysis == nil {
		return astmerge.MergeResult[string]{OK: false, Diagnostics: template.Diagnostics, Policies: []astmerge.PolicyReference{}}
	}

	destination := ParseRuby(destinationSource, dialect)
	if !destination.OK || destination.Analysis == nil {
		diagnostics := make([]astmerge.Diagnostic, 0, len(destination.Diagnostics))
		for _, diagnostic := range destination.Diagnostics {
			if diagnostic.Category == astmerge.CategoryParseError {
				diagnostic.Category = astmerge.CategoryDestinationParseError
			}
			diagnostics = append(diagnostics, diagnostic)
		}
		return astmerge.MergeResult[string]{OK: false, Diagnostics: diagnostics, Policies: []astmerge.PolicyReference{}}
	}

	requires := collectRubyRequireEntries(destination.Analysis.Source)
	destinationDeclarations := collectRubyDeclarationEntries(destination.Analysis.Source)
	templateDeclarations := collectRubyDeclarationEntries(template.Analysis.Source)
	destinationPaths := make(map[string]struct{}, len(destinationDeclarations))
	sections := make([]string, 0, len(requires)+len(destinationDeclarations)+len(templateDeclarations))
	if len(requires) > 0 {
		requireLines := make([]string, 0, len(requires))
		for _, entry := range requires {
			requireLines = append(requireLines, entry.Text)
		}
		sections = append(sections, strings.TrimSpace(strings.Join(requireLines, "\n")))
	}
	for _, entry := range destinationDeclarations {
		destinationPaths[entry.Path] = struct{}{}
		sections = append(sections, entry.Text)
	}
	for _, entry := range templateDeclarations {
		if _, ok := destinationPaths[entry.Path]; ok {
			continue
		}
		sections = append(sections, entry.Text)
	}

	output := strings.TrimSpace(strings.Join(sections, "\n\n")) + "\n"
	policies := []astmerge.PolicyReference{destinationWinsArrayPolicy()}
	return astmerge.MergeResult[string]{OK: true, Diagnostics: []astmerge.Diagnostic{}, Output: &output, Policies: policies}
}

func rubyExampleLinePrefix(line string) string {
	matches := regexp.MustCompile(`^(\s*#\s*)`).FindStringSubmatch(line)
	if len(matches) > 1 {
		return matches[1]
	}
	return "# "
}

func ApplyRubyDelegatedChildOutputs(
	source string,
	operations []astmerge.DelegatedChildOperation,
	applyPlan astmerge.DelegatedChildApplyPlan,
	appliedChildren []AppliedChildOutput,
) astmerge.MergeResult[string] {
	lines := strings.Split(normalizeSource(source), "\n")
	operationsByID := make(map[string]astmerge.DelegatedChildOperation, len(operations))
	for _, operation := range operations {
		operationsByID[operation.OperationID] = operation
	}
	outputsByID := make(map[string]string, len(appliedChildren))
	for _, entry := range appliedChildren {
		outputsByID[entry.OperationID] = entry.Output
	}

	type replacement struct {
		start  int
		end    int
		output string
	}
	replacements := make([]replacement, 0, len(applyPlan.Entries))
	for _, entry := range applyPlan.Entries {
		operation, ok := operationsByID[entry.DelegatedGroup.ChildOperationID]
		if !ok || operation.Surface.Span == nil {
			continue
		}
		output, ok := outputsByID[entry.DelegatedGroup.ChildOperationID]
		if !ok {
			continue
		}
		replacements = append(replacements, replacement{
			start:  operation.Surface.Span.StartLine - 1,
			end:    operation.Surface.Span.EndLine - 1,
			output: output,
		})
	}

	slices.SortFunc(replacements, func(left, right replacement) int { return right.start - left.start })
	for _, entry := range replacements {
		if entry.start < 0 || entry.start >= len(lines) {
			return astmerge.MergeResult[string]{
				OK:          false,
				Diagnostics: []astmerge.Diagnostic{configurationError("invalid delegated child span.")},
				Policies:    []astmerge.PolicyReference{},
			}
		}
		prefix := rubyExampleLinePrefix(lines[entry.start])
		replacementLines := []string{}
		if entry.output != "" {
			for _, bodyLine := range strings.Split(strings.TrimSuffix(entry.output, "\n"), "\n") {
				replacementLines = append(replacementLines, prefix+bodyLine)
			}
		}
		lines = append(lines[:entry.start], append(replacementLines, lines[entry.end+1:]...)...)
	}

	output := strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n"
	return astmerge.MergeResult[string]{
		OK:          true,
		Diagnostics: []astmerge.Diagnostic{},
		Output:      &output,
		Policies:    []astmerge.PolicyReference{destinationWinsArrayPolicy()},
	}
}

func MergeRubyWithNestedOutputs(
	templateSource string,
	destinationSource string,
	dialect RubyDialect,
	nestedOutputs []NestedChildOutput,
) astmerge.MergeResult[string] {
	resolutionInputs := make([]astmerge.DelegatedChildSurfaceOutput, 0, len(nestedOutputs))
	for _, nestedOutput := range nestedOutputs {
		resolutionInputs = append(resolutionInputs, astmerge.DelegatedChildSurfaceOutput{
			SurfaceAddress: nestedOutput.SurfaceAddress,
			Output:         nestedOutput.Output,
		})
	}
	return astmerge.ExecuteNestedMerge[string](
		resolutionInputs,
		astmerge.DelegatedChildOutputResolutionOptions{
			DefaultFamily:   "ruby",
			RequestIDPrefix: "nested_ruby_child",
		},
		astmerge.NestedMergeExecutionCallbacks[string]{
			MergeParent: func() astmerge.MergeResult[string] {
				return MergeRuby(templateSource, destinationSource, dialect)
			},
			DiscoverOperations: func(mergedOutput string) astmerge.NestedMergeDiscoveryResult {
				analysis := ParseRuby(mergedOutput, dialect)
				if !analysis.OK || analysis.Analysis == nil {
					return astmerge.NestedMergeDiscoveryResult{
						OK:          false,
						Diagnostics: analysis.Diagnostics,
					}
				}

				return astmerge.NestedMergeDiscoveryResult{
					OK:          true,
					Diagnostics: []astmerge.Diagnostic{},
					Operations:  RubyDelegatedChildOperations(*analysis.Analysis, "ruby-document-0"),
				}
			},
			ApplyResolvedOutputs: func(mergedOutput string, operations []astmerge.DelegatedChildOperation, applyPlan astmerge.DelegatedChildApplyPlan, appliedChildren []astmerge.AppliedDelegatedChildOutput) astmerge.MergeResult[string] {
				children := make([]AppliedChildOutput, 0, len(appliedChildren))
				for _, entry := range appliedChildren {
					children = append(children, AppliedChildOutput{
						OperationID: entry.OperationID,
						Output:      entry.Output,
					})
				}
				return ApplyRubyDelegatedChildOutputs(
					mergedOutput,
					operations,
					applyPlan,
					children,
				)
			},
		},
	)
}

func MergeRubyWithReviewedNestedOutputs(
	templateSource string,
	destinationSource string,
	dialect RubyDialect,
	reviewState astmerge.DelegatedChildGroupReviewState,
	appliedChildren []AppliedChildOutput,
) astmerge.MergeResult[string] {
	resolvedChildren := make([]astmerge.AppliedDelegatedChildOutput, 0, len(appliedChildren))
	for _, child := range appliedChildren {
		resolvedChildren = append(resolvedChildren, astmerge.AppliedDelegatedChildOutput{
			OperationID: child.OperationID,
			Output:      child.Output,
		})
	}

	return astmerge.ExecuteReviewedNestedMerge(
		reviewState,
		"ruby",
		resolvedChildren,
		astmerge.NestedMergeExecutionCallbacks[string]{
			MergeParent: func() astmerge.MergeResult[string] {
				return MergeRuby(templateSource, destinationSource, dialect)
			},
			DiscoverOperations: func(mergedOutput string) astmerge.NestedMergeDiscoveryResult {
				analysis := ParseRuby(mergedOutput, dialect)
				if !analysis.OK || analysis.Analysis == nil {
					return astmerge.NestedMergeDiscoveryResult{
						OK:          false,
						Diagnostics: analysis.Diagnostics,
					}
				}

				return astmerge.NestedMergeDiscoveryResult{
					OK:          true,
					Diagnostics: []astmerge.Diagnostic{},
					Operations:  RubyDelegatedChildOperations(*analysis.Analysis, "ruby-document-0"),
				}
			},
			ApplyResolvedOutputs: func(
				mergedOutput string,
				operations []astmerge.DelegatedChildOperation,
				applyPlan astmerge.DelegatedChildApplyPlan,
				appliedChildren []astmerge.AppliedDelegatedChildOutput,
			) astmerge.MergeResult[string] {
				translated := make([]AppliedChildOutput, 0, len(appliedChildren))
				for _, child := range appliedChildren {
					translated = append(translated, AppliedChildOutput{
						OperationID: child.OperationID,
						Output:      child.Output,
					})
				}
				return ApplyRubyDelegatedChildOutputs(
					mergedOutput,
					operations,
					applyPlan,
					translated,
				)
			},
		},
	)
}

func MergeRubyWithReviewedNestedOutputsFromReplayBundle(
	templateSource string,
	destinationSource string,
	dialect RubyDialect,
	bundle astmerge.ReviewReplayBundle,
) astmerge.MergeResult[string] {
	for _, execution := range bundle.ReviewedNestedExecutions {
		if execution.Family == "ruby" {
			children := make([]AppliedChildOutput, 0, len(execution.AppliedChildren))
			for _, child := range execution.AppliedChildren {
				children = append(children, AppliedChildOutput{
					OperationID: child.OperationID,
					Output:      child.Output,
				})
			}
			return MergeRubyWithReviewedNestedOutputs(
				templateSource,
				destinationSource,
				dialect,
				execution.ReviewState,
				children,
			)
		}
	}

	return astmerge.MergeResult[string]{
		OK: false,
		Diagnostics: []astmerge.Diagnostic{{
			Severity: astmerge.SeverityError,
			Category: astmerge.CategoryConfigurationError,
			Message:  "review replay bundle does not include a reviewed nested execution for ruby.",
		}},
		Policies: []astmerge.PolicyReference{},
	}
}

func MergeRubyWithReviewedNestedOutputsFromReviewState(
	templateSource string,
	destinationSource string,
	dialect RubyDialect,
	state astmerge.ConformanceManifestReviewState,
) astmerge.MergeResult[string] {
	for _, execution := range state.ReviewedNestedExecutions {
		if execution.Family == "ruby" {
			children := make([]AppliedChildOutput, 0, len(execution.AppliedChildren))
			for _, child := range execution.AppliedChildren {
				children = append(children, AppliedChildOutput{
					OperationID: child.OperationID,
					Output:      child.Output,
				})
			}
			return MergeRubyWithReviewedNestedOutputs(
				templateSource,
				destinationSource,
				dialect,
				execution.ReviewState,
				children,
			)
		}
	}

	return astmerge.MergeResult[string]{
		OK: false,
		Diagnostics: []astmerge.Diagnostic{{
			Severity: astmerge.SeverityError,
			Category: astmerge.CategoryConfigurationError,
			Message:  "review state does not include a reviewed nested execution for ruby.",
		}},
		Policies: []astmerge.PolicyReference{},
	}
}

func RubyDiscoveredSurfaces(analysis RubyAnalysis) []astmerge.DiscoveredSurface {
	return analysis.DiscoveredSurfaces
}

func RubyDelegatedChildOperations(analysis RubyAnalysis, parentOperationID string) []astmerge.DelegatedChildOperation {
	operations := make([]astmerge.DelegatedChildOperation, 0)
	docOperationIDs := make(map[string]string)
	docIndex := 0
	exampleIndex := 0

	for _, surface := range analysis.DiscoveredSurfaces {
		if surface.SurfaceKind != "ruby_doc_comment" {
			continue
		}
		operationID := "ruby-doc-comment-" + strconv.Itoa(docIndex)
		docOperationIDs[surface.Address] = operationID
		operations = append(operations, astmerge.DelegatedChildOperation{
			OperationID:       operationID,
			ParentOperationID: parentOperationID,
			RequestedStrategy: "delegate_child_surface",
			LanguageChain:     []string{"ruby", surface.EffectiveLanguage},
			Surface:           surface,
		})
		docIndex++
	}

	for _, surface := range analysis.DiscoveredSurfaces {
		if surface.SurfaceKind != "yard_example_block" {
			continue
		}
		parentID := parentOperationID
		if value, ok := docOperationIDs[surface.ParentAddress]; ok {
			parentID = value
		}
		operations = append(operations, astmerge.DelegatedChildOperation{
			OperationID:       "yard-example-" + strconv.Itoa(exampleIndex),
			ParentOperationID: parentID,
			RequestedStrategy: "delegate_child_surface",
			LanguageChain:     []string{"ruby", "yard", surface.EffectiveLanguage},
			Surface:           surface,
		})
		exampleIndex++
	}

	return operations
}

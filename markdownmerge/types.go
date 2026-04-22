package markdownmerge

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/treehaver"
)

type MarkdownDialect string

const (
	DialectMarkdown MarkdownDialect = "markdown"
)

type MarkdownBackend string

const (
	BackendKreuzberg MarkdownBackend = "kreuzberg-language-pack"
)

type MarkdownRootKind string

const (
	RootDocument MarkdownRootKind = "document"
)

type MarkdownOwnerKind string

const (
	OwnerHeading   MarkdownOwnerKind = "heading"
	OwnerCodeFence MarkdownOwnerKind = "code_fence"
)

type MarkdownOwner struct {
	Path       string
	OwnerKind  MarkdownOwnerKind
	MatchKey   string
	Level      int
	InfoString string
}

type MarkdownOwnerMatch struct {
	TemplatePath    string
	DestinationPath string
}

type MarkdownOwnerMatchResult struct {
	Matched              []MarkdownOwnerMatch
	UnmatchedTemplate    []string
	UnmatchedDestination []string
}

type MarkdownAnalysis struct {
	Dialect          MarkdownDialect
	NormalizedSource string
	RootKind         MarkdownRootKind
	Owners           []MarkdownOwner
}

type markdownSection struct {
	Path string
	Text string
}

type MarkdownEmbeddedFamilyCandidate struct {
	Path     string `json:"path"`
	Language string `json:"language"`
	Family   string `json:"family"`
	Dialect  string `json:"dialect"`
}

type AppliedChildOutput struct {
	OperationID string `json:"operation_id"`
	Output      string `json:"output"`
}

type NestedChildOutput struct {
	SurfaceAddress string `json:"surface_address"`
	Output         string `json:"output"`
}

func (MarkdownAnalysis) Kind() string {
	return "markdown"
}

type MarkdownFeatureProfile struct {
	Family            string
	SupportedDialects []MarkdownDialect
	SupportedPolicies []astmerge.PolicyReference
}

type MarkdownBackendFeatureProfile struct {
	Family            string
	SupportedDialects []MarkdownDialect
	SupportedPolicies []astmerge.PolicyReference
	Backend           string
	BackendRef        *treehaver.BackendReference
}

func unsupportedFeature(message string) astmerge.Diagnostic {
	return astmerge.Diagnostic{
		Severity: astmerge.SeverityError,
		Category: astmerge.CategoryUnsupportedFeature,
		Message:  message,
	}
}

func configurationError(message string) astmerge.Diagnostic {
	return astmerge.Diagnostic{
		Severity: astmerge.SeverityError,
		Category: astmerge.CategoryConfigurationError,
		Message:  message,
	}
}

func NormalizeMarkdownSource(source string) string {
	source = strings.ReplaceAll(source, "\r\n", "\n")
	return strings.ReplaceAll(source, "\r", "\n")
}

func slugify(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	var builder strings.Builder
	lastDash := false
	for _, character := range value {
		if (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9') {
			builder.WriteRune(character)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "section"
	}
	return result
}

var headingPattern = regexp.MustCompile(`^(#{1,6})\s+(.+?)\s*#*\s*$`)
var codeFencePattern = regexp.MustCompile("^\\s*([`]{3,}|[~]{3,})\\s*(.*?)\\s*$")

func CollectMarkdownOwners(source string) []MarkdownOwner {
	lines := strings.Split(NormalizeMarkdownSource(source), "\n")
	owners := make([]MarkdownOwner, 0)
	headingIndex := 0
	codeFenceIndex := 0

	for index := 0; index < len(lines); index++ {
		line := lines[index]
		if heading := headingPattern.FindStringSubmatch(line); heading != nil {
			level := len(heading[1])
			title := strings.TrimSpace(heading[2])
			owners = append(owners, MarkdownOwner{
				Path:      fmt.Sprintf("/heading/%d", headingIndex),
				OwnerKind: OwnerHeading,
				MatchKey:  fmt.Sprintf("h%d:%s", level, slugify(title)),
				Level:     level,
			})
			headingIndex++
			continue
		}

		fence := codeFencePattern.FindStringSubmatch(line)
		if fence == nil {
			continue
		}

		marker := fence[1]
		infoString := ""
		if rest := strings.TrimSpace(fence[2]); rest != "" {
			infoString = strings.Fields(rest)[0]
		}

		matchKey := "fence:plain"
		if infoString != "" {
			matchKey = "fence:" + infoString
		}

		owners = append(owners, MarkdownOwner{
			Path:       fmt.Sprintf("/code_fence/%d", codeFenceIndex),
			OwnerKind:  OwnerCodeFence,
			MatchKey:   matchKey,
			InfoString: infoString,
		})
		codeFenceIndex++

		markerChar := marker[:1]
		markerLength := len(marker)
		for cursor := index + 1; cursor < len(lines); cursor++ {
			trimmed := strings.TrimSpace(lines[cursor])
			if len(trimmed) >= markerLength &&
				strings.Trim(trimmed, markerChar) == "" &&
				strings.HasPrefix(trimmed, strings.Repeat(markerChar, markerLength)) {
				index = cursor
				break
			}
			if cursor == len(lines)-1 {
				index = cursor
			}
		}
	}

	return owners
}

func MarkdownFeatureProfileInfo() MarkdownFeatureProfile {
	return MarkdownFeatureProfile{
		Family:            "markdown",
		SupportedDialects: []MarkdownDialect{DialectMarkdown},
		SupportedPolicies: []astmerge.PolicyReference{},
	}
}

func AvailableMarkdownBackends() []MarkdownBackend {
	return []MarkdownBackend{BackendKreuzberg}
}

func MarkdownBackendFeatureProfileInfo(backend MarkdownBackend) MarkdownBackendFeatureProfile {
	return MarkdownBackendFeatureProfile{
		Family:            "markdown",
		SupportedDialects: []MarkdownDialect{DialectMarkdown},
		SupportedPolicies: []astmerge.PolicyReference{},
		Backend:           string(backend),
		BackendRef:        treehaver.BackendReferenceByID(string(backend)),
	}
}

func MarkdownPlanContext() astmerge.ConformanceFamilyPlanContext {
	return MarkdownPlanContextWithBackend(BackendKreuzberg)
}

func MarkdownPlanContextWithBackend(backend MarkdownBackend) astmerge.ConformanceFamilyPlanContext {
	return astmerge.ConformanceFamilyPlanContext{
		FamilyProfile: astmerge.FamilyFeatureProfile{
			Family:            "markdown",
			SupportedDialects: []string{"markdown"},
			SupportedPolicies: []astmerge.PolicyReference{},
		},
		FeatureProfile: &astmerge.ConformanceFeatureProfileView{
			Backend:           string(backend),
			SupportsDialects:  false,
			SupportedPolicies: []astmerge.PolicyReference{},
		},
	}
}

func ParseMarkdown(source string, dialect MarkdownDialect) astmerge.ParseResult[MarkdownAnalysis] {
	return ParseMarkdownWithBackend(source, dialect, BackendKreuzberg)
}

func ParseMarkdownWithBackend(source string, dialect MarkdownDialect, backend MarkdownBackend) astmerge.ParseResult[MarkdownAnalysis] {
	if dialect != DialectMarkdown {
		return astmerge.ParseResult[MarkdownAnalysis]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{unsupportedFeature(fmt.Sprintf("Unsupported Markdown dialect %s.", dialect))},
		}
	}

	switch backend {
	case BackendKreuzberg:
		result := treehaver.ParseWithLanguagePack(treehaver.ParserRequest{
			Source:   source,
			Language: "markdown",
			Dialect:  "markdown",
		})
		if !result.OK {
			return astmerge.ParseResult[MarkdownAnalysis]{
				OK:          false,
				Diagnostics: result.Diagnostics,
			}
		}
	default:
		return astmerge.ParseResult[MarkdownAnalysis]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{unsupportedFeature(fmt.Sprintf("Unsupported Markdown backend %s.", backend))},
		}
	}

	normalized := NormalizeMarkdownSource(source)
	return astmerge.ParseResult[MarkdownAnalysis]{
		OK:          true,
		Diagnostics: []astmerge.Diagnostic{},
		Analysis: &MarkdownAnalysis{
			Dialect:          dialect,
			NormalizedSource: normalized,
			RootKind:         RootDocument,
			Owners:           CollectMarkdownOwners(normalized),
		},
		Policies: []astmerge.PolicyReference{},
	}
}

func MatchMarkdownOwners(template MarkdownAnalysis, destination MarkdownAnalysis) MarkdownOwnerMatchResult {
	destinationPaths := make(map[string]bool, len(destination.Owners))
	templatePaths := make(map[string]bool, len(template.Owners))
	for _, owner := range destination.Owners {
		destinationPaths[owner.Path] = true
	}
	for _, owner := range template.Owners {
		templatePaths[owner.Path] = true
	}

	result := MarkdownOwnerMatchResult{
		Matched:              make([]MarkdownOwnerMatch, 0),
		UnmatchedTemplate:    make([]string, 0),
		UnmatchedDestination: make([]string, 0),
	}

	for _, owner := range template.Owners {
		if destinationPaths[owner.Path] {
			result.Matched = append(result.Matched, MarkdownOwnerMatch{
				TemplatePath:    owner.Path,
				DestinationPath: owner.Path,
			})
		} else {
			result.UnmatchedTemplate = append(result.UnmatchedTemplate, owner.Path)
		}
	}

	for _, owner := range destination.Owners {
		if !templatePaths[owner.Path] {
			result.UnmatchedDestination = append(result.UnmatchedDestination, owner.Path)
		}
	}

	slices.Sort(result.UnmatchedTemplate)
	slices.Sort(result.UnmatchedDestination)
	return result
}

func markdownOwnerStartIndices(source string) map[string]int {
	lines := strings.Split(NormalizeMarkdownSource(source), "\n")
	starts := make(map[string]int)
	headingIndex := 0
	codeFenceIndex := 0

	for index := 0; index < len(lines); index++ {
		line := lines[index]
		if heading := headingPattern.FindStringSubmatch(line); heading != nil {
			starts[fmt.Sprintf("/heading/%d", headingIndex)] = index
			headingIndex++
			continue
		}

		fence := codeFencePattern.FindStringSubmatch(line)
		if fence == nil {
			continue
		}

		starts[fmt.Sprintf("/code_fence/%d", codeFenceIndex)] = index
		codeFenceIndex++

		marker := fence[1]
		markerChar := marker[:1]
		markerLength := len(marker)
		for cursor := index + 1; cursor < len(lines); cursor++ {
			trimmed := strings.TrimSpace(lines[cursor])
			if len(trimmed) >= markerLength &&
				strings.Trim(trimmed, markerChar) == "" &&
				strings.HasPrefix(trimmed, strings.Repeat(markerChar, markerLength)) {
				index = cursor
				break
			}
			if cursor == len(lines)-1 {
				index = cursor
			}
		}
	}

	return starts
}

func collectMarkdownSections(source string, owners []MarkdownOwner) []markdownSection {
	lines := strings.Split(NormalizeMarkdownSource(source), "\n")
	starts := markdownOwnerStartIndices(source)
	type ownerStart struct {
		Owner MarkdownOwner
		Start int
	}
	ordered := make([]ownerStart, 0, len(owners))
	for _, owner := range owners {
		start, ok := starts[owner.Path]
		if ok {
			ordered = append(ordered, ownerStart{Owner: owner, Start: start})
		}
	}
	slices.SortFunc(ordered, func(left, right ownerStart) int { return left.Start - right.Start })

	sections := make([]markdownSection, 0, len(ordered))
	for index, entry := range ordered {
		endExclusive := len(lines)
		if index+1 < len(ordered) {
			endExclusive = ordered[index+1].Start
		}
		text := strings.TrimSpace(strings.Join(lines[entry.Start:endExclusive], "\n"))
		sections = append(sections, markdownSection{Path: entry.Owner.Path, Text: text})
	}
	return sections
}

func MergeMarkdown(templateSource string, destinationSource string, dialect MarkdownDialect, backend ...MarkdownBackend) astmerge.MergeResult[string] {
	resolvedBackend := BackendKreuzberg
	if len(backend) > 0 {
		resolvedBackend = backend[0]
	}

	template := ParseMarkdownWithBackend(templateSource, dialect, resolvedBackend)
	if !template.OK || template.Analysis == nil {
		return astmerge.MergeResult[string]{OK: false, Diagnostics: template.Diagnostics, Policies: []astmerge.PolicyReference{}}
	}

	destination := ParseMarkdownWithBackend(destinationSource, dialect, resolvedBackend)
	if !destination.OK || destination.Analysis == nil {
		return astmerge.MergeResult[string]{OK: false, Diagnostics: destination.Diagnostics, Policies: []astmerge.PolicyReference{}}
	}

	destinationSections := collectMarkdownSections(destination.Analysis.NormalizedSource, destination.Analysis.Owners)
	templateSections := collectMarkdownSections(template.Analysis.NormalizedSource, template.Analysis.Owners)
	destinationPaths := make(map[string]struct{}, len(destinationSections))
	mergedSections := make([]string, 0, len(destinationSections)+len(templateSections))
	for _, section := range destinationSections {
		destinationPaths[section.Path] = struct{}{}
		if section.Text != "" {
			mergedSections = append(mergedSections, section.Text)
		}
	}
	for _, section := range templateSections {
		if _, ok := destinationPaths[section.Path]; ok || section.Text == "" {
			continue
		}
		mergedSections = append(mergedSections, section.Text)
	}

	output := strings.TrimSpace(strings.Join(mergedSections, "\n\n")) + "\n"
	return astmerge.MergeResult[string]{
		OK:          true,
		Diagnostics: []astmerge.Diagnostic{},
		Output:      &output,
		Policies:    []astmerge.PolicyReference{},
	}
}

type markdownFenceRange struct {
	Start int
	End   int
}

func markdownFenceRanges(source string) map[string]markdownFenceRange {
	lines := strings.Split(NormalizeMarkdownSource(source), "\n")
	ranges := make(map[string]markdownFenceRange)
	codeFenceIndex := 0

	for index := 0; index < len(lines); index++ {
		line := lines[index]
		fence := codeFencePattern.FindStringSubmatch(line)
		if fence == nil {
			continue
		}

		marker := fence[1]
		markerChar := marker[:1]
		markerLength := len(marker)
		end := index
		for cursor := index + 1; cursor < len(lines); cursor++ {
			trimmed := strings.TrimSpace(lines[cursor])
			if len(trimmed) >= markerLength &&
				strings.Trim(trimmed, markerChar) == "" &&
				strings.HasPrefix(trimmed, strings.Repeat(markerChar, markerLength)) {
				end = cursor
				break
			}
			if cursor == len(lines)-1 {
				end = cursor
			}
		}

		ranges[fmt.Sprintf("/code_fence/%d", codeFenceIndex)] = markdownFenceRange{Start: index, End: end}
		codeFenceIndex++
		index = end
	}

	return ranges
}

func ApplyMarkdownDelegatedChildOutputs(
	source string,
	operations []astmerge.DelegatedChildOperation,
	applyPlan astmerge.DelegatedChildApplyPlan,
	appliedChildren []AppliedChildOutput,
) astmerge.MergeResult[string] {
	lines := strings.Split(NormalizeMarkdownSource(source), "\n")
	ranges := markdownFenceRanges(source)
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
		if !ok {
			continue
		}
		output, ok := outputsByID[entry.DelegatedGroup.ChildOperationID]
		if !ok {
			continue
		}
		rng, ok := ranges[operation.Surface.Owner.Address]
		if !ok {
			return astmerge.MergeResult[string]{
				OK:          false,
				Diagnostics: []astmerge.Diagnostic{configurationError("missing fenced-code range for " + operation.Surface.Owner.Address)},
			}
		}
		replacements = append(replacements, replacement{start: rng.Start, end: rng.End, output: output})
	}

	slices.SortFunc(replacements, func(left, right replacement) int { return right.start - left.start })
	for _, entry := range replacements {
		body := strings.TrimSuffix(entry.output, "\n")
		replacementLines := []string{}
		if entry.output != "" {
			replacementLines = strings.Split(body, "\n")
		}
		lines = append(lines[:entry.start+1], append(replacementLines, lines[entry.end:]...)...)
	}

	output := strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n"
	return astmerge.MergeResult[string]{
		OK:          true,
		Diagnostics: []astmerge.Diagnostic{},
		Output:      &output,
		Policies:    []astmerge.PolicyReference{},
	}
}

func MergeMarkdownWithNestedOutputs(
	templateSource string,
	destinationSource string,
	dialect MarkdownDialect,
	nestedOutputs []NestedChildOutput,
	backend ...MarkdownBackend,
) astmerge.MergeResult[string] {
	resolvedBackend := BackendKreuzberg
	if len(backend) > 0 {
		resolvedBackend = backend[0]
	}
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
			DefaultFamily:   "markdown",
			RequestIDPrefix: "nested_markdown_child",
		},
		astmerge.NestedMergeExecutionCallbacks[string]{
			MergeParent: func() astmerge.MergeResult[string] {
				return MergeMarkdown(templateSource, destinationSource, dialect, backend...)
			},
			DiscoverOperations: func(mergedOutput string) astmerge.NestedMergeDiscoveryResult {
				analysis := ParseMarkdownWithBackend(mergedOutput, dialect, resolvedBackend)
				if !analysis.OK || analysis.Analysis == nil {
					return astmerge.NestedMergeDiscoveryResult{
						OK:          false,
						Diagnostics: analysis.Diagnostics,
					}
				}

				return astmerge.NestedMergeDiscoveryResult{
					OK:          true,
					Diagnostics: []astmerge.Diagnostic{},
					Operations:  MarkdownDelegatedChildOperations(*analysis.Analysis, "markdown-document-0"),
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
				return ApplyMarkdownDelegatedChildOutputs(
					mergedOutput,
					operations,
					applyPlan,
					children,
				)
			},
		},
	)
}

func MergeMarkdownWithReviewedNestedOutputs(
	templateSource string,
	destinationSource string,
	dialect MarkdownDialect,
	reviewState astmerge.DelegatedChildGroupReviewState,
	appliedChildren []AppliedChildOutput,
	backend ...MarkdownBackend,
) astmerge.MergeResult[string] {
	resolvedBackend := BackendKreuzberg
	if len(backend) > 0 {
		resolvedBackend = backend[0]
	}
	resolvedChildren := make([]astmerge.AppliedDelegatedChildOutput, 0, len(appliedChildren))
	for _, child := range appliedChildren {
		resolvedChildren = append(resolvedChildren, astmerge.AppliedDelegatedChildOutput{
			OperationID: child.OperationID,
			Output:      child.Output,
		})
	}

	return astmerge.ExecuteReviewedNestedMerge(
		reviewState,
		"markdown",
		resolvedChildren,
		astmerge.NestedMergeExecutionCallbacks[string]{
			MergeParent: func() astmerge.MergeResult[string] {
				return MergeMarkdown(templateSource, destinationSource, dialect, resolvedBackend)
			},
			DiscoverOperations: func(mergedOutput string) astmerge.NestedMergeDiscoveryResult {
				analysis := ParseMarkdownWithBackend(mergedOutput, dialect, resolvedBackend)
				if !analysis.OK || analysis.Analysis == nil {
					return astmerge.NestedMergeDiscoveryResult{
						OK:          false,
						Diagnostics: analysis.Diagnostics,
					}
				}

				return astmerge.NestedMergeDiscoveryResult{
					OK:          true,
					Diagnostics: []astmerge.Diagnostic{},
					Operations:  MarkdownDelegatedChildOperations(*analysis.Analysis, "markdown-document-0"),
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
				return ApplyMarkdownDelegatedChildOutputs(
					mergedOutput,
					operations,
					applyPlan,
					translated,
				)
			},
		},
	)
}

func MergeMarkdownWithReviewedNestedOutputsFromReplayBundle(
	templateSource string,
	destinationSource string,
	dialect MarkdownDialect,
	bundle astmerge.ReviewReplayBundle,
	backend ...MarkdownBackend,
) astmerge.MergeResult[string] {
	resolvedBackend := BackendKreuzberg
	if len(backend) > 0 {
		resolvedBackend = backend[0]
	}
	for _, execution := range bundle.ReviewedNestedExecutions {
		if execution.Family == "markdown" {
			children := make([]AppliedChildOutput, 0, len(execution.AppliedChildren))
			for _, child := range execution.AppliedChildren {
				children = append(children, AppliedChildOutput{
					OperationID: child.OperationID,
					Output:      child.Output,
				})
			}
			return MergeMarkdownWithReviewedNestedOutputs(
				templateSource,
				destinationSource,
				dialect,
				execution.ReviewState,
				children,
				resolvedBackend,
			)
		}
	}

	return astmerge.MergeResult[string]{
		OK: false,
		Diagnostics: []astmerge.Diagnostic{{
			Severity: astmerge.SeverityError,
			Category: astmerge.CategoryConfigurationError,
			Message:  "review replay bundle does not include a reviewed nested execution for markdown.",
		}},
		Policies: []astmerge.PolicyReference{},
	}
}

func MergeMarkdownWithReviewedNestedOutputsFromReplayBundleEnvelope(
	templateSource string,
	destinationSource string,
	dialect MarkdownDialect,
	envelope astmerge.ReviewReplayBundleEnvelope,
	backend ...MarkdownBackend,
) astmerge.MergeResult[string] {
	resolvedBackend := BackendKreuzberg
	if len(backend) > 0 {
		resolvedBackend = backend[0]
	}
	bundle, importErr := astmerge.ImportReviewReplayBundleEnvelope(envelope)
	if importErr != nil {
		return astmerge.MergeResult[string]{
			OK: false,
			Diagnostics: []astmerge.Diagnostic{{
				Severity: astmerge.SeverityError,
				Category: astmerge.DiagnosticCategory(importErr.Category),
				Message:  importErr.Message,
			}},
			Policies: []astmerge.PolicyReference{},
		}
	}

	return MergeMarkdownWithReviewedNestedOutputsFromReplayBundle(
		templateSource,
		destinationSource,
		dialect,
		*bundle,
		resolvedBackend,
	)
}

func MergeMarkdownWithReviewedNestedOutputsFromReviewState(
	templateSource string,
	destinationSource string,
	dialect MarkdownDialect,
	state astmerge.ConformanceManifestReviewState,
	backend ...MarkdownBackend,
) astmerge.MergeResult[string] {
	resolvedBackend := BackendKreuzberg
	if len(backend) > 0 {
		resolvedBackend = backend[0]
	}
	for _, execution := range state.ReviewedNestedExecutions {
		if execution.Family == "markdown" {
			children := make([]AppliedChildOutput, 0, len(execution.AppliedChildren))
			for _, child := range execution.AppliedChildren {
				children = append(children, AppliedChildOutput{
					OperationID: child.OperationID,
					Output:      child.Output,
				})
			}
			return MergeMarkdownWithReviewedNestedOutputs(
				templateSource,
				destinationSource,
				dialect,
				execution.ReviewState,
				children,
				resolvedBackend,
			)
		}
	}

	return astmerge.MergeResult[string]{
		OK: false,
		Diagnostics: []astmerge.Diagnostic{{
			Severity: astmerge.SeverityError,
			Category: astmerge.CategoryConfigurationError,
			Message:  "review state does not include a reviewed nested execution for markdown.",
		}},
		Policies: []astmerge.PolicyReference{},
	}
}

func MergeMarkdownWithReviewedNestedOutputsFromReviewStateEnvelope(
	templateSource string,
	destinationSource string,
	dialect MarkdownDialect,
	envelope astmerge.ConformanceManifestReviewStateEnvelope,
	backend ...MarkdownBackend,
) astmerge.MergeResult[string] {
	resolvedBackend := BackendKreuzberg
	if len(backend) > 0 {
		resolvedBackend = backend[0]
	}
	state, importErr := astmerge.ImportConformanceManifestReviewStateEnvelope(envelope)
	if importErr != nil {
		return astmerge.MergeResult[string]{
			OK: false,
			Diagnostics: []astmerge.Diagnostic{{
				Severity: astmerge.SeverityError,
				Category: astmerge.DiagnosticCategory(importErr.Category),
				Message:  importErr.Message,
			}},
			Policies: []astmerge.PolicyReference{},
		}
	}

	return MergeMarkdownWithReviewedNestedOutputsFromReviewState(
		templateSource,
		destinationSource,
		dialect,
		*state,
		resolvedBackend,
	)
}

func codeFenceFamily(infoString string) string {
	switch strings.ToLower(infoString) {
	case "ts", "typescript":
		return "typescript"
	case "rust", "rs":
		return "rust"
	case "go":
		return "go"
	case "json", "jsonc":
		return "json"
	case "yaml", "yml":
		return "yaml"
	case "toml":
		return "toml"
	default:
		return ""
	}
}

func codeFenceDialect(infoString string, family string) string {
	switch family {
	case "typescript":
		return "typescript"
	case "rust":
		return "rust"
	case "go":
		return "go"
	case "json":
		if strings.EqualFold(infoString, "jsonc") {
			return "jsonc"
		}
		return "json"
	case "yaml":
		return "yaml"
	case "toml":
		return "toml"
	default:
		return ""
	}
}

func MarkdownEmbeddedFamilies(analysis MarkdownAnalysis) []MarkdownEmbeddedFamilyCandidate {
	candidates := make([]MarkdownEmbeddedFamilyCandidate, 0)
	for _, owner := range analysis.Owners {
		if owner.OwnerKind != OwnerCodeFence || owner.InfoString == "" {
			continue
		}
		family := codeFenceFamily(owner.InfoString)
		dialect := codeFenceDialect(owner.InfoString, family)
		if family == "" || dialect == "" {
			continue
		}
		candidates = append(candidates, MarkdownEmbeddedFamilyCandidate{
			Path: owner.Path, Language: owner.InfoString, Family: family, Dialect: dialect,
		})
	}
	return candidates
}

func MarkdownDiscoveredSurfaces(analysis MarkdownAnalysis) []astmerge.DiscoveredSurface {
	candidates := MarkdownEmbeddedFamilies(analysis)
	surfaces := make([]astmerge.DiscoveredSurface, 0, len(candidates))
	for _, candidate := range candidates {
		surfaces = append(surfaces, astmerge.DiscoveredSurface{
			SurfaceKind:       "markdown_fenced_code_block",
			DeclaredLanguage:  candidate.Language,
			EffectiveLanguage: candidate.Dialect,
			Address:           fmt.Sprintf("document[0] > fenced_code_block[%s]", candidate.Path),
			ParentAddress:     "document[0]",
			Owner: astmerge.SurfaceOwnerRef{
				Kind:    astmerge.SurfaceOwnerStructuralOwner,
				Address: candidate.Path,
			},
			ReconstructionStrategy: "portable_write",
			Metadata: map[string]any{
				"family":  candidate.Family,
				"dialect": candidate.Dialect,
				"path":    candidate.Path,
			},
		})
	}

	return surfaces
}

func MarkdownDelegatedChildOperations(
	analysis MarkdownAnalysis,
	parentOperationID string,
) []astmerge.DelegatedChildOperation {
	surfaces := MarkdownDiscoveredSurfaces(analysis)
	operations := make([]astmerge.DelegatedChildOperation, 0, len(surfaces))
	for index, surface := range surfaces {
		operations = append(operations, astmerge.DelegatedChildOperation{
			OperationID:       fmt.Sprintf("markdown-fence-%d", index),
			ParentOperationID: parentOperationID,
			RequestedStrategy: "delegate_child_surface",
			LanguageChain:     []string{"markdown", surface.EffectiveLanguage},
			Surface:           surface,
		})
	}

	return operations
}

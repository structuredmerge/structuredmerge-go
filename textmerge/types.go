package textmerge

import (
	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/treehaver"
	"slices"
	"strings"
)

type TextSpan struct {
	Start int
	End   int
}

type TextBlock struct {
	Index      int
	Normalized string
	Span       TextSpan
}

type TextAnalysis struct {
	NormalizedSource string
	Blocks           []TextBlock
}

func (TextAnalysis) Kind() string {
	return "text"
}

type TextParserAdapter interface {
	treehaver.ParserAdapter[TextAnalysis]
}

func TextParseRequest(source string) treehaver.ParserRequest {
	return treehaver.ParserRequest{
		Source:   source,
		Language: "text",
	}
}

type TextAnalyzer interface {
	Analyze(source string) TextAnalysis
}

type TextSimilarity struct {
	Score     float64
	Threshold float64
	Matched   bool
}

type TextBlockMatch struct {
	TemplateIndex    int
	DestinationIndex int
}

type TextBlockMatchResult struct {
	Matched              []TextBlockMatch
	UnmatchedTemplate    []int
	UnmatchedDestination []int
}

type TextMergeResolution struct {
	Output string
}

type TextBlockMatcher interface {
	MatchBlocks(template TextAnalysis, destination TextAnalysis) TextBlockMatchResult
}

type TextMerger interface {
	Merge(template TextAnalysis, destination TextAnalysis) astmerge.MergeResult[string]
}

func NormalizeText(source string) string {
	source = strings.ReplaceAll(source, "\r\n", "\n")
	source = strings.ReplaceAll(source, "\r", "\n")
	source = strings.TrimSpace(source)

	if source == "" {
		return ""
	}

	rawBlocks := strings.Split(source, "\n\n")
	blocks := make([]string, 0, len(rawBlocks))
	for _, raw := range rawBlocks {
		normalized := strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
		if normalized != "" {
			blocks = append(blocks, normalized)
		}
	}

	return strings.Join(blocks, "\n\n")
}

func AnalyzeText(source string) TextAnalysis {
	normalizedSource := NormalizeText(source)
	if normalizedSource == "" {
		return TextAnalysis{
			NormalizedSource: "",
			Blocks:           []TextBlock{},
		}
	}

	parts := strings.Split(normalizedSource, "\n\n")
	blocks := make([]TextBlock, 0, len(parts))
	cursor := 0
	for index, normalized := range parts {
		start := cursor
		end := start + len(normalized)
		blocks = append(blocks, TextBlock{
			Index:      index,
			Normalized: normalized,
			Span: TextSpan{
				Start: start,
				End:   end,
			},
		})
		cursor = end + 2
	}

	return TextAnalysis{
		NormalizedSource: normalizedSource,
		Blocks:           blocks,
	}
}

func tokenSet(normalized string) []string {
	seen := map[string]struct{}{}
	tokens := make([]string, 0)
	for _, token := range strings.Fields(normalized) {
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		tokens = append(tokens, token)
	}
	slices.Sort(tokens)
	return tokens
}

func jaccard(left string, right string) float64 {
	leftTokens := tokenSet(left)
	rightTokens := tokenSet(right)

	if len(leftTokens) == 0 && len(rightTokens) == 0 {
		return 1
	}

	rightSet := map[string]struct{}{}
	for _, token := range rightTokens {
		rightSet[token] = struct{}{}
	}

	intersection := 0
	unionSet := map[string]struct{}{}
	for _, token := range leftTokens {
		unionSet[token] = struct{}{}
		if _, ok := rightSet[token]; ok {
			intersection++
		}
	}
	for _, token := range rightTokens {
		unionSet[token] = struct{}{}
	}

	if len(unionSet) == 0 {
		return 1
	}

	return float64(intersection) / float64(len(unionSet))
}

func SimilarityScore(leftSource string, rightSource string) float64 {
	left := AnalyzeText(leftSource)
	right := AnalyzeText(rightSource)
	total := max(len(left.Blocks), len(right.Blocks))

	if total == 0 {
		return 1
	}

	sum := 0.0
	for index := 0; index < total; index++ {
		if index >= len(left.Blocks) || index >= len(right.Blocks) {
			continue
		}
		sum += jaccard(left.Blocks[index].Normalized, right.Blocks[index].Normalized)
	}

	return sum / float64(total)
}

func IsSimilar(leftSource string, rightSource string, threshold float64) TextSimilarity {
	score := SimilarityScore(leftSource, rightSource)
	return TextSimilarity{
		Score:     score,
		Threshold: threshold,
		Matched:   score >= threshold,
	}
}

func MergeText(templateSource string, destinationSource string) astmerge.MergeResult[string] {
	template := AnalyzeText(templateSource)
	destination := AnalyzeText(destinationSource)
	matches := MatchTextBlocks(templateSource, destinationSource)
	matchedTemplate := map[int]struct{}{}
	for _, match := range matches.Matched {
		matchedTemplate[match.TemplateIndex] = struct{}{}
	}
	mergedBlocks := make([]string, 0, len(template.Blocks)+len(destination.Blocks))

	for _, block := range destination.Blocks {
		mergedBlocks = append(mergedBlocks, block.Normalized)
	}
	for index, block := range template.Blocks {
		if _, ok := matchedTemplate[index]; !ok {
			mergedBlocks = append(mergedBlocks, block.Normalized)
		}
	}

	output := strings.Join(mergedBlocks, "\n\n")
	return astmerge.MergeResult[string]{
		OK:          true,
		Diagnostics: []astmerge.Diagnostic{},
		Output:      &output,
	}
}

func MatchTextBlocks(templateSource string, destinationSource string) TextBlockMatchResult {
	template := AnalyzeText(templateSource)
	destination := AnalyzeText(destinationSource)
	matchedTemplate := map[int]struct{}{}
	matchedDestination := map[int]struct{}{}
	matched := make([]TextBlockMatch, 0)

	for destinationIndex, destinationBlock := range destination.Blocks {
		templateIndex := -1
		for candidateIndex, templateBlock := range template.Blocks {
			if _, ok := matchedTemplate[candidateIndex]; ok {
				continue
			}
			if templateBlock.Normalized == destinationBlock.Normalized {
				templateIndex = candidateIndex
				break
			}
		}

		if templateIndex >= 0 {
			matchedTemplate[templateIndex] = struct{}{}
			matchedDestination[destinationIndex] = struct{}{}
			matched = append(matched, TextBlockMatch{
				TemplateIndex:    templateIndex,
				DestinationIndex: destinationIndex,
			})
		}
	}

	unmatchedTemplate := make([]int, 0)
	unmatchedDestination := make([]int, 0)
	for index := range template.Blocks {
		if _, ok := matchedTemplate[index]; !ok {
			unmatchedTemplate = append(unmatchedTemplate, index)
		}
	}
	for index := range destination.Blocks {
		if _, ok := matchedDestination[index]; !ok {
			unmatchedDestination = append(unmatchedDestination, index)
		}
	}

	return TextBlockMatchResult{
		Matched:              matched,
		UnmatchedTemplate:    unmatchedTemplate,
		UnmatchedDestination: unmatchedDestination,
	}
}

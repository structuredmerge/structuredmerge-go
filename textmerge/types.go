package textmerge

import (
	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/treehaver"
	"slices"
	"strings"
)

const DefaultTextRefinementThreshold = 0.7

type TextMatchPhase string

const (
	TextMatchPhaseExact   TextMatchPhase = "exact"
	TextMatchPhaseRefined TextMatchPhase = "refined"
)

type TextRefinementWeights struct {
	Content  float64
	Length   float64
	Position float64
}

var DefaultTextRefinementWeights = TextRefinementWeights{
	Content:  0.7,
	Length:   0.15,
	Position: 0.15,
}

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
	Phase            TextMatchPhase
	Score            float64
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

func levenshteinDistance(left string, right string) int {
	if left == right {
		return 0
	}

	leftRunes := []rune(left)
	rightRunes := []rune(right)

	if len(leftRunes) == 0 {
		return len(rightRunes)
	}
	if len(rightRunes) == 0 {
		return len(leftRunes)
	}

	previous := make([]int, len(leftRunes)+1)
	current := make([]int, len(leftRunes)+1)
	for index := range previous {
		previous[index] = index
	}

	for rightIndex, rightRune := range rightRunes {
		current[0] = rightIndex + 1
		for leftIndex, leftRune := range leftRunes {
			cost := 0
			if leftRune != rightRune {
				cost = 1
			}
			current[leftIndex+1] = min(
				current[leftIndex]+1,
				previous[leftIndex+1]+1,
				previous[leftIndex]+cost,
			)
		}
		copy(previous, current)
	}

	return previous[len(leftRunes)]
}

func stringSimilarity(left string, right string) float64 {
	if left == right {
		return 1
	}
	if left == "" || right == "" {
		return 0
	}

	distance := levenshteinDistance(left, right)
	maxLength := max(len([]rune(left)), len([]rune(right)))
	return 1 - float64(distance)/float64(maxLength)
}

func lengthSimilarity(left string, right string) float64 {
	if len(left) == len(right) {
		return 1
	}

	maxLength := max(len(left), len(right))
	if maxLength == 0 {
		return 1
	}

	return float64(min(len(left), len(right))) / float64(maxLength)
}

func relativePosition(index int, total int) float64 {
	if total > 1 {
		return float64(index) / float64(total-1)
	}
	return 0.5
}

func positionSimilarity(templateIndex int, destinationIndex int, templateTotal int, destinationTotal int) float64 {
	return 1 - abs(relativePosition(templateIndex, templateTotal)-relativePosition(destinationIndex, destinationTotal))
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

func RefinedTextSimilarity(
	templateBlock TextBlock,
	destinationBlock TextBlock,
	templateTotal int,
	destinationTotal int,
	weights TextRefinementWeights,
) float64 {
	content := stringSimilarity(templateBlock.Normalized, destinationBlock.Normalized)
	length := lengthSimilarity(templateBlock.Normalized, destinationBlock.Normalized)
	position := positionSimilarity(
		templateBlock.Index,
		destinationBlock.Index,
		templateTotal,
		destinationTotal,
	)

	return weights.Content*content + weights.Length*length + weights.Position*position
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
				Phase:            TextMatchPhaseExact,
				Score:            1,
			})
		}
	}

	for destinationIndex, destinationBlock := range destination.Blocks {
		if _, ok := matchedDestination[destinationIndex]; ok {
			continue
		}

		bestTemplateIndex := -1
		bestScore := 0.0
		for templateIndex, templateBlock := range template.Blocks {
			if _, ok := matchedTemplate[templateIndex]; ok {
				continue
			}

			score := RefinedTextSimilarity(
				templateBlock,
				destinationBlock,
				len(template.Blocks),
				len(destination.Blocks),
				DefaultTextRefinementWeights,
			)

			if score >= DefaultTextRefinementThreshold && score > bestScore {
				bestTemplateIndex = templateIndex
				bestScore = score
			}
		}

		if bestTemplateIndex >= 0 {
			matchedTemplate[bestTemplateIndex] = struct{}{}
			matchedDestination[destinationIndex] = struct{}{}
			matched = append(matched, TextBlockMatch{
				TemplateIndex:    bestTemplateIndex,
				DestinationIndex: destinationIndex,
				Phase:            TextMatchPhaseRefined,
				Score:            bestScore,
			})
		}
	}

	slices.SortFunc(matched, func(left TextBlockMatch, right TextBlockMatch) int {
		return left.DestinationIndex - right.DestinationIndex
	})

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

func abs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

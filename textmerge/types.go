package textmerge

import (
	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/treehaver"
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

type TextAnalyzer interface {
	Analyze(source string) TextAnalysis
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

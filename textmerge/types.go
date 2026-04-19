package textmerge

import (
	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/treehaver"
)

type TextAnalysis struct {
	Blocks []string
}

func (TextAnalysis) Kind() string {
	return "text"
}

type TextParserAdapter interface {
	treehaver.ParserAdapter[TextAnalysis]
}

type TextMerger interface {
	Merge(template TextAnalysis, destination TextAnalysis) astmerge.MergeResult[string]
}

package jsonmerge

import (
	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/treehaver"
)

type JSONAnalysis struct {
	AllowsComments bool
}

func (JSONAnalysis) Kind() string {
	return "json"
}

type JSONParserAdapter interface {
	treehaver.ParserAdapter[JSONAnalysis]
}

type JSONMerger interface {
	Merge(template JSONAnalysis, destination JSONAnalysis) astmerge.MergeResult[string]
}

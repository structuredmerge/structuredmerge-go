package textmerge

import "testing"

func TestAnalyzeText(t *testing.T) {
	source := "  Alpha   beta\r\n\r\nGamma\n   delta\n\n\nEpsilon  \n"

	analysis := AnalyzeText(source)

	if analysis.NormalizedSource != "Alpha beta\n\nGamma delta\n\nEpsilon" {
		t.Fatalf("unexpected normalized source: %q", analysis.NormalizedSource)
	}

	if len(analysis.Blocks) != 3 {
		t.Fatalf("unexpected block count: %d", len(analysis.Blocks))
	}

	if analysis.Blocks[0].Normalized != "Alpha beta" {
		t.Fatalf("unexpected first block: %q", analysis.Blocks[0].Normalized)
	}

	if analysis.Blocks[1].Span.Start != 12 || analysis.Blocks[1].Span.End != 23 {
		t.Fatalf("unexpected second block span: %+v", analysis.Blocks[1].Span)
	}
}

func TestSimilarityScore(t *testing.T) {
	if score := SimilarityScore("Alpha   beta\n\nGamma", "  Alpha beta  \r\n\r\nGamma  "); score != 1 {
		t.Fatalf("unexpected equivalent score: %v", score)
	}

	if score := SimilarityScore("Alpha beta\n\nGamma delta", "Alpha beta\n\nGamma epsilon"); score != 0.6666666666666666 {
		t.Fatalf("unexpected near-match score: %v", score)
	}

	if score := SimilarityScore("Alpha beta", "Zeta theta"); score != 0 {
		t.Fatalf("unexpected mismatch score: %v", score)
	}

	if similarity := IsSimilar("Alpha beta\n\nGamma delta", "Alpha beta\n\nGamma epsilon", 0.6); !similarity.Matched {
		t.Fatalf("expected near-match to satisfy threshold")
	}
}

func TestMergeText(t *testing.T) {
	result := MergeText(
		"Alpha\n\nBeta\n\nTemplate tail",
		"Alpha revised\n\nBeta\n\nDestination tail\n\nDestination extra",
	)

	if !result.OK || result.Output == nil {
		t.Fatalf("expected merge success, got diagnostics: %+v", result.Diagnostics)
	}

	expected := "Alpha revised\n\nBeta\n\nDestination tail\n\nDestination extra"
	if *result.Output != expected {
		t.Fatalf("unexpected merged output: %q", *result.Output)
	}
}

func TestMatchTextBlocks(t *testing.T) {
	result := MatchTextBlocks(
		"Alpha\n\nBeta\n\nAlpha\n\nTemplate only",
		"Beta\n\nAlpha\n\nAlpha\n\nDestination only",
	)

	expectedMatched := []TextBlockMatch{
		{TemplateIndex: 1, DestinationIndex: 0},
		{TemplateIndex: 0, DestinationIndex: 1},
		{TemplateIndex: 2, DestinationIndex: 2},
	}

	if len(result.Matched) != len(expectedMatched) {
		t.Fatalf("unexpected matched blocks: %+v", result.Matched)
	}
	for index := range expectedMatched {
		if result.Matched[index] != expectedMatched[index] {
			t.Fatalf("unexpected matched block at %d: %+v", index, result.Matched[index])
		}
	}

	if len(result.UnmatchedTemplate) != 1 || result.UnmatchedTemplate[0] != 3 {
		t.Fatalf("unexpected unmatched template blocks: %+v", result.UnmatchedTemplate)
	}
	if len(result.UnmatchedDestination) != 1 || result.UnmatchedDestination[0] != 3 {
		t.Fatalf("unexpected unmatched destination blocks: %+v", result.UnmatchedDestination)
	}
}

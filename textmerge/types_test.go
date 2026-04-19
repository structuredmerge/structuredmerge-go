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

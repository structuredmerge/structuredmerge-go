package binarymerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/structuredmerge/structuredmerge-go/treehaver"
)

func readBinaryFixture(t *testing.T, parts ...string) map[string]any {
	t.Helper()

	path := filepath.Join(append([]string{"..", "..", "fixtures"}, parts...)...)
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var fixture map[string]any
	if err := json.Unmarshal(source, &fixture); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	return fixture
}

func TestBinaryPreservationReport(t *testing.T) {
	fixture := readBinaryFixture(t, "diagnostics", "slice-723-binary-core-contract", "binary-core.json")
	reportFixture := fixture["merge_report"].(map[string]any)
	firstRange := reportFixture["preserved_ranges"].([]any)[0].(map[string]any)
	preservedRange := treehaver.ByteRange{
		StartByte: int(firstRange["start_byte"].(float64)),
		EndByte:   int(firstRange["end_byte"].(float64)),
	}

	report := PreservationReport(
		reportFixture["format"].(string),
		reportFixture["schema"].(string),
		[]string{"/chunks/0", "/chunks/1"},
		[]treehaver.ByteRange{preservedRange},
	)
	diagnostic := UnsafeDiagnostic(
		"/chunks/2",
		treehaver.ByteRange{StartByte: 78, EndByte: 96},
		"critical image data mutation is not enabled",
	)

	if BinaryFeatureProfileInfo().Family != "binary" {
		t.Fatalf("unexpected binary feature profile")
	}
	if report.PreservedRanges[0].Length() != 25 || len(report.RewrittenNodes) != 0 {
		t.Fatalf("unexpected preservation report: %+v", report)
	}
	if diagnostic.Category != "unsafe_binary_mutation" {
		t.Fatalf("unexpected diagnostic: %+v", diagnostic)
	}
}

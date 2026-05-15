package astcrispr

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestBoundaryReportFixture(t *testing.T) {
	fixturePath := filepath.Join(
		"..",
		"..",
		"fixtures",
		"diagnostics",
		"slice-916-ast-crispr-package-boundary",
		"ast-crispr-package-boundary.json",
	)
	source, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fixture map[string]any
	if err := json.Unmarshal(source, &fixture); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	if !reflect.DeepEqual(BoundaryReport(), fixture["boundary"]) {
		t.Fatalf("unexpected boundary report: %+v", BoundaryReport())
	}
	if AstMergeContractAnchor() != "StructuredEditCrisprExampleParityReport" {
		t.Fatalf("unexpected contract anchor: %s", AstMergeContractAnchor())
	}
}

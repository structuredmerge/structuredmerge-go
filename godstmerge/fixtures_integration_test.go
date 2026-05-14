package godstmerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/structuredmerge/structuredmerge-go/gomerge"
	"github.com/structuredmerge/structuredmerge-go/treehaver"
)

func jsonReady(t *testing.T, value any) any {
	t.Helper()
	source, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal value: %v", err)
	}

	var normalized any
	if err := json.Unmarshal(source, &normalized); err != nil {
		t.Fatalf("decode value: %v", err)
	}

	return normalized
}

func readFixture(t *testing.T, parts ...string) map[string]any {
	t.Helper()
	pathParts := append([]string{"..", "..", "fixtures"}, parts...)
	source, err := os.ReadFile(filepath.Join(pathParts...))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var fixture map[string]any
	if err := json.Unmarshal(source, &fixture); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	return fixture
}

func TestGoDSTProviderFeatureProfile(t *testing.T) {
	familyProfile := GoFeatureProfileInfo()
	if familyProfile.Family != "go" {
		t.Fatalf("unexpected family profile: %+v", familyProfile)
	}
	if len(AvailableGoBackends()) != 1 || AvailableGoBackends()[0] != BackendGoDST {
		t.Fatalf("unexpected backends: %+v", AvailableGoBackends())
	}
	profile := GoBackendFeatureProfileInfo()
	if profile.Backend != BackendGoDST || profile.BackendRef == nil || profile.BackendRef.ID != BackendGoDST || profile.BackendRef.Family != "native" {
		t.Fatalf("unexpected provider profile: %+v", profile)
	}
	if backend := treehaver.BackendReferenceByID(BackendGoDST); backend == nil || backend.ID != BackendGoDST || backend.Family != "native" {
		t.Fatalf("unexpected registered backend: %+v", backend)
	}
	if context := GoPlanContext(); context.FeatureProfile == nil || context.FeatureProfile.Backend != BackendGoDST {
		t.Fatalf("unexpected provider plan context: %+v", context)
	}
}

func TestGoDSTProviderParseAndMergeParity(t *testing.T) {
	parityFixture := readFixture(t, "go", "slice-114-native", "module-parity.json")
	parseResult := ParseGo(parityFixture["source"].(string), gomerge.DialectGo)
	if !parseResult.OK || parseResult.Analysis == nil {
		t.Fatalf("expected parse success: %+v", parseResult)
	}
	owners := make([]map[string]any, 0, len(parseResult.Analysis.Owners))
	for _, owner := range parseResult.Analysis.Owners {
		owners = append(owners, map[string]any{
			"path":       owner.Path,
			"owner_kind": owner.OwnerKind,
			"match_key":  owner.MatchKey,
		})
	}
	if actual := jsonReady(t, owners); !reflect.DeepEqual(actual, parityFixture["expected"].(map[string]any)["owners"]) {
		t.Fatalf("unexpected owners: %+v", actual)
	}

	mergeResult := MergeGo(parityFixture["template"].(string), parityFixture["destination"].(string), gomerge.DialectGo)
	if !mergeResult.OK || mergeResult.Output == nil {
		t.Fatalf("expected merge success: %+v", mergeResult)
	}
	if *mergeResult.Output != parityFixture["expected"].(map[string]any)["output"].(string) {
		t.Fatalf("unexpected merge output:\n%s", *mergeResult.Output)
	}
}

func TestGoDSTProviderRejectsUnsupportedBackendOverrides(t *testing.T) {
	expectedDiagnostics := []map[string]any{{
		"severity": "error",
		"category": "unsupported_feature",
		"message":  "Unsupported Go backend go-parser.",
	}}

	parseResult := ParseGo("func Greet() string { return \"hi\" }\n", gomerge.DialectGo, "go-parser")
	if parseResult.OK {
		t.Fatalf("expected parse failure: %+v", parseResult)
	}
	if actual := jsonReady(t, parseResult.Diagnostics); !reflect.DeepEqual(actual, jsonReady(t, expectedDiagnostics)) {
		t.Fatalf("unexpected parse diagnostics: %+v", actual)
	}

	mergeResult := MergeGo("func A() {}\n", "func B() {}\n", gomerge.DialectGo, "go-parser")
	if mergeResult.OK {
		t.Fatalf("expected merge failure: %+v", mergeResult)
	}
	if actual := jsonReady(t, mergeResult.Diagnostics); !reflect.DeepEqual(actual, jsonReady(t, expectedDiagnostics)) {
		t.Fatalf("unexpected merge diagnostics: %+v", actual)
	}
}

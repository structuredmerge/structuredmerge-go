package treehaver

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
)

func readParserFixture(t *testing.T, parts ...string) map[string]any {
	t.Helper()

	pathParts := append([]string{"..", "..", "fixtures"}, parts...)
	path := filepath.Join(pathParts...)
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

func readParserFixtureFromPath(t *testing.T, path string) map[string]any {
	t.Helper()

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

func diagnosticsFixturePath(t *testing.T, role string) string {
	t.Helper()

	manifestFixture := readParserFixture(t, "conformance", "slice-24-manifest", "family-feature-profiles.json")
	manifestSource, err := json.Marshal(manifestFixture)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	var manifest astmerge.ConformanceManifest
	if err := json.Unmarshal(manifestSource, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	path := astmerge.ConformanceFixturePath(manifest, "diagnostics", role)
	if path == nil {
		t.Fatalf("missing diagnostics fixture entry for %s", role)
	}

	return filepath.Join(append([]string{"..", "..", "fixtures"}, path...)...)
}

func TestSharedFixtureParserRequest(t *testing.T) {
	fixture := readParserFixtureFromPath(t, diagnosticsFixturePath(t, "parser_request"))
	requestFixture := fixture["request"].(map[string]any)
	infoFixture := fixture["adapter_info"].(map[string]any)

	request := ParserRequest{
		Source:   requestFixture["source"].(string),
		Language: requestFixture["language"].(string),
		Dialect:  requestFixture["dialect"].(string),
	}
	info := AdapterInfo{
		Backend:           infoFixture["backend"].(string),
		BackendRef:        nil,
		SupportsDialects:  infoFixture["supports_dialects"].(bool),
		SupportedPolicies: []astmerge.PolicyReference{},
	}

	if request.Source != requestFixture["source"].(string) ||
		request.Language != requestFixture["language"].(string) ||
		request.Dialect != requestFixture["dialect"].(string) {
		t.Fatalf("unexpected parser request: %+v", request)
	}
	if info.Backend != infoFixture["backend"].(string) ||
		info.SupportsDialects != infoFixture["supports_dialects"].(bool) {
		t.Fatalf("unexpected adapter info: %+v", info)
	}
}

func TestSharedFixtureAdapterPolicySupport(t *testing.T) {
	fixture := readParserFixtureFromPath(t, diagnosticsFixturePath(t, "adapter_policy_support"))
	infoFixture := fixture["adapter_info"].(map[string]any)

	info := AdapterInfo{
		Backend:          infoFixture["backend"].(string),
		BackendRef:       nil,
		SupportsDialects: infoFixture["supports_dialects"].(bool),
		SupportedPolicies: []astmerge.PolicyReference{
			{
				Surface: astmerge.PolicySurfaceArray,
				Name:    "destination_wins_array",
			},
			{
				Surface: astmerge.PolicySurfaceFallback,
				Name:    "trailing_comma_destination_fallback",
			},
		},
	}

	if info.Backend != infoFixture["backend"].(string) ||
		info.SupportsDialects != infoFixture["supports_dialects"].(bool) {
		t.Fatalf("unexpected adapter info: %+v", info)
	}
	expectedPolicies := infoFixture["supported_policies"].([]any)
	if len(info.SupportedPolicies) != len(expectedPolicies) {
		t.Fatalf("unexpected supported policies: %+v", info.SupportedPolicies)
	}
	for index, policy := range info.SupportedPolicies {
		expected := expectedPolicies[index].(map[string]any)
		if string(policy.Surface) != expected["surface"].(string) || policy.Name != expected["name"].(string) {
			t.Fatalf("unexpected supported policy at %d: %+v", index, policy)
		}
	}
}

func TestSharedFixtureAdapterFeatureProfile(t *testing.T) {
	fixture := readParserFixtureFromPath(t, diagnosticsFixturePath(t, "adapter_feature_profile"))
	profileFixture := fixture["feature_profile"].(map[string]any)

	profile := FeatureProfile{
		Backend:          profileFixture["backend"].(string),
		BackendRef:       nil,
		SupportsDialects: profileFixture["supports_dialects"].(bool),
		SupportedPolicies: []astmerge.PolicyReference{
			{
				Surface: astmerge.PolicySurfaceArray,
				Name:    "destination_wins_array",
			},
			{
				Surface: astmerge.PolicySurfaceFallback,
				Name:    "trailing_comma_destination_fallback",
			},
		},
	}

	if profile.Backend != profileFixture["backend"].(string) ||
		profile.SupportsDialects != profileFixture["supports_dialects"].(bool) {
		t.Fatalf("unexpected feature profile: %+v", profile)
	}
	expectedPolicies := profileFixture["supported_policies"].([]any)
	if len(profile.SupportedPolicies) != len(expectedPolicies) {
		t.Fatalf("unexpected feature-profile policies: %+v", profile.SupportedPolicies)
	}
	for index, policy := range profile.SupportedPolicies {
		expected := expectedPolicies[index].(map[string]any)
		if string(policy.Surface) != expected["surface"].(string) || policy.Name != expected["name"].(string) {
			t.Fatalf("unexpected feature-profile policy at %d: %+v", index, policy)
		}
	}
}

func TestSharedFixtureBackendRegistry(t *testing.T) {
	fixture := readParserFixtureFromPath(t, diagnosticsFixturePath(t, "backend_registry"))

	backends := []BackendReference{
		{ID: "native", Family: "builtin"},
		{ID: "tree-sitter", Family: "tree-sitter"},
	}
	profile := FeatureProfile{
		Backend:           "tree-sitter",
		BackendRef:        &backends[1],
		SupportsDialects:  true,
		SupportedPolicies: []astmerge.PolicyReference{},
	}

	expectedBackends := fixture["backends"].([]any)
	if len(backends) != len(expectedBackends) {
		t.Fatalf("unexpected backends: %+v", backends)
	}
	for index, backend := range backends {
		expected := expectedBackends[index].(map[string]any)
		if backend.ID != expected["id"].(string) || backend.Family != expected["family"].(string) {
			t.Fatalf("unexpected backend at %d: %+v", index, backend)
		}
	}
	if profile.BackendRef == nil || profile.BackendRef.ID != "tree-sitter" || profile.BackendRef.Family != "tree-sitter" {
		t.Fatalf("unexpected backend reference on feature profile: %+v", profile)
	}
}

func TestPigeonBackendReference(t *testing.T) {
	if backend := BackendReferenceByID("pigeon"); backend == nil || backend.ID != "pigeon" || backend.Family != "peg" {
		t.Fatalf("unexpected pigeon backend: %+v", backend)
	}
	info := PigeonAdapterInfo()
	if info.Backend != "pigeon" || info.BackendRef == nil || info.BackendRef.Family != "peg" {
		t.Fatalf("unexpected pigeon adapter info: %+v", info)
	}
	profile := PigeonFeatureProfile()
	if profile.Backend != "pigeon" || profile.BackendRef == nil || profile.BackendRef.Family != "peg" {
		t.Fatalf("unexpected pigeon feature profile: %+v", profile)
	}
}

func TestSharedFixtureKaitaiTreeHaverSubstrate(t *testing.T) {
	fixture := readParserFixtureFromPath(t, diagnosticsFixturePath(t, "kaitai_tree_haver_substrate"))
	backendFixture := fixture["backend"].(map[string]any)
	infoFixture := fixture["adapter_info"].(map[string]any)
	profileFixture := fixture["feature_profile"].(map[string]any)
	nodeFixture := fixture["tree_node"].(map[string]any)

	backend := BackendReferenceByID("kaitai-struct")
	if backend == nil || backend.ID != backendFixture["id"].(string) || backend.Family != backendFixture["family"].(string) {
		t.Fatalf("unexpected kaitai backend: %+v", backend)
	}

	info := KaitaiAdapterInfo()
	if info.Backend != infoFixture["backend"].(string) || info.BackendRef == nil || info.BackendRef.ID != "kaitai-struct" {
		t.Fatalf("unexpected kaitai adapter info: %+v", info)
	}
	profile := KaitaiFeatureProfile()
	if profile.Backend != profileFixture["backend"].(string) || profile.BackendRef == nil || profile.BackendRef.Family != "kaitai" {
		t.Fatalf("unexpected kaitai feature profile: %+v", profile)
	}

	spanFixture := nodeFixture["span"].(map[string]any)
	childFixture := nodeFixture["children"].([]any)[0].(map[string]any)
	childSpanFixture := childFixture["span"].(map[string]any)
	node := KaitaiTreeNode{
		Kind:       nodeFixture["kind"].(string),
		SchemaPath: nodeFixture["schema_path"].(string),
		Span: KaitaiByteSpan{
			StartByte: int(spanFixture["start_byte"].(float64)),
			EndByte:   int(spanFixture["end_byte"].(float64)),
		},
		Fields: nodeFixture["fields"].(map[string]any),
		Children: []KaitaiTreeNode{
			{
				Kind:       childFixture["kind"].(string),
				SchemaPath: childFixture["schema_path"].(string),
				Span: KaitaiByteSpan{
					StartByte: int(childSpanFixture["start_byte"].(float64)),
					EndByte:   int(childSpanFixture["end_byte"].(float64)),
				},
				Fields:   childFixture["fields"].(map[string]any),
				Children: []KaitaiTreeNode{},
			},
		},
	}
	analysis := KaitaiTreeAnalysis{Schema: "png.ksy", Root: node, BackendRef: *backend}
	if analysis.Kind() != "kaitai-tree" || analysis.Root.SchemaPath != "/chunks/1" || analysis.Root.Children[0].Fields["value"] != "Template" {
		t.Fatalf("unexpected kaitai analysis: %+v", analysis)
	}
}

func TestSharedFixturePortableByteLocationContract(t *testing.T) {
	fixture := readParserFixtureFromPath(t, diagnosticsFixturePath(t, "portable_byte_location_contract"))
	rangeFixture := fixture["byte_range"].(map[string]any)
	pointFixture := fixture["source_point"].(map[string]any)
	expectedFixture := fixture["expected"].(map[string]any)
	comparisonFixture := fixture["comparison_ranges"].(map[string]any)
	source := fixture["source"].(string)

	byteRange := ByteRange{
		StartByte: int(rangeFixture["start_byte"].(float64)),
		EndByte:   int(rangeFixture["end_byte"].(float64)),
	}
	point := SourcePoint{
		Row:    int(pointFixture["row"].(float64)),
		Column: int(pointFixture["column"].(float64)),
	}
	overlappingFixture := comparisonFixture["overlapping"].(map[string]any)
	disjointFixture := comparisonFixture["disjoint"].(map[string]any)
	overlappingRange := ByteRange{
		StartByte: int(overlappingFixture["start_byte"].(float64)),
		EndByte:   int(overlappingFixture["end_byte"].(float64)),
	}
	disjointRange := ByteRange{
		StartByte: int(disjointFixture["start_byte"].(float64)),
		EndByte:   int(disjointFixture["end_byte"].(float64)),
	}

	slice, err := SliceByteRange(source, byteRange)
	if err != nil {
		t.Fatalf("slice byte range: %v", err)
	}
	offset, err := ByteOffsetForPoint(source, point)
	if err != nil {
		t.Fatalf("byte offset for point: %v", err)
	}
	if byteRange.Length() != int(expectedFixture["length"].(float64)) ||
		slice != expectedFixture["slice"].(string) ||
		byteRange.ContainsByte(byteRange.StartByte) != expectedFixture["contains_start"].(bool) ||
		byteRange.ContainsByte(byteRange.EndByte) != expectedFixture["contains_end"].(bool) ||
		byteRange.Overlaps(overlappingRange) != expectedFixture["overlaps"].(bool) ||
		byteRange.Overlaps(disjointRange) != expectedFixture["disjoint"].(bool) ||
		offset != int(expectedFixture["line_column_offset"].(float64)) {
		t.Fatalf("unexpected byte location behavior: range=%+v slice=%q offset=%d", byteRange, slice, offset)
	}
}

func TestRuntimeBackendRegistration(t *testing.T) {
	RegisterBackend(BackendReference{ID: "custom-toml", Family: "native"})

	backend := BackendReferenceByID("custom-toml")
	if backend == nil || backend.ID != "custom-toml" || backend.Family != "native" {
		t.Fatalf("unexpected custom backend: %+v", backend)
	}

	backends := RegisteredBackends()
	found := false
	for _, backend := range backends {
		if backend.ID == "custom-toml" && backend.Family == "native" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("custom backend not present in registry: %+v", backends)
	}
}

func TestSharedFixtureProcessBaseline(t *testing.T) {
	fixture := readParserFixtureFromPath(t, diagnosticsFixturePath(t, "process_baseline"))
	requestFixture := fixture["request"].(map[string]any)
	expected := fixture["expected"].(map[string]any)

	result := ProcessWithLanguagePack(ProcessRequest{
		Source:   requestFixture["source"].(string),
		Language: requestFixture["language"].(string),
	})
	if !result.OK || result.Analysis == nil {
		t.Fatalf("unexpected process result: %+v", result)
	}
	if result.Analysis.Language != expected["language"].(string) {
		t.Fatalf("unexpected language: %+v", result.Analysis)
	}
	expectedStructure := expected["structure"].([]any)
	if len(result.Analysis.Structure) != len(expectedStructure) {
		t.Fatalf("unexpected structure: %+v", result.Analysis.Structure)
	}
	for index, item := range expectedStructure {
		expectedItem := item.(map[string]any)
		actual := result.Analysis.Structure[index]
		if actual.Kind != expectedItem["kind"].(string) {
			t.Fatalf("unexpected structure kind at %d: %+v", index, actual)
		}
		if expectedName, ok := expectedItem["name"]; ok && actual.Name != expectedName.(string) {
			t.Fatalf("unexpected structure name at %d: %+v", index, actual)
		}
	}
	expectedImports := expected["imports"].([]any)
	if len(result.Analysis.Imports) != len(expectedImports) {
		t.Fatalf("unexpected imports: %+v", result.Analysis.Imports)
	}
	for index, item := range expectedImports {
		expectedItem := item.(map[string]any)
		actual := result.Analysis.Imports[index]
		if actual.Source != expectedItem["source"].(string) {
			t.Fatalf("unexpected import at %d: %+v", index, actual)
		}
	}
}

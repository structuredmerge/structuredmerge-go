package treehaver

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
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
	families, ok := manifestFixture["families"].(map[string]any)
	if !ok {
		t.Fatalf("manifest missing families")
	}
	diagnosticsEntries, ok := families["diagnostics"].([]any)
	if !ok {
		t.Fatalf("manifest missing diagnostics family")
	}
	for _, rawEntry := range diagnosticsEntries {
		entry := rawEntry.(map[string]any)
		if entry["role"] != role {
			continue
		}
		rawPath := entry["path"].([]any)
		path := make([]string, 0, len(rawPath))
		for _, part := range rawPath {
			path = append(path, part.(string))
		}
		return filepath.Join(append([]string{"..", "..", "fixtures"}, path...)...)
	}

	t.Fatalf("missing diagnostics fixture entry for %s", role)
	return ""
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
		SupportedPolicies: []PolicyReference{},
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
		SupportedPolicies: []PolicyReference{
			{
				Surface: PolicySurfaceArray,
				Name:    "destination_wins_array",
			},
			{
				Surface: PolicySurfaceFallback,
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
		SupportedPolicies: []PolicyReference{
			{
				Surface: PolicySurfaceArray,
				Name:    "destination_wins_array",
			},
			{
				Surface: PolicySurfaceFallback,
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
		SupportedPolicies: []PolicyReference{},
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
	analysisFixture := fixture["analysis"].(map[string]any)

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
	diagnosticFixture := analysisFixture["diagnostics"].([]any)[0].(map[string]any)
	analysis := KaitaiTreeAnalysis{
		Schema:           analysisFixture["schema"].(string),
		SourceByteLength: int(analysisFixture["source_byte_length"].(float64)),
		Root:             node,
		BackendRef:       *backend,
		Diagnostics: []BinaryDiagnostic{
			{
				Severity:   diagnosticFixture["severity"].(string),
				Category:   diagnosticFixture["category"].(string),
				Message:    diagnosticFixture["message"].(string),
				SchemaPath: diagnosticFixture["schema_path"].(string),
				ByteRange:  byteRangePointer(diagnosticFixture["byte_range"]),
			},
		},
	}
	if analysis.Kind() != "kaitai-tree" || analysis.Root.SchemaPath != "/chunks/1" || analysis.Root.Children[0].Fields["value"] != "Template" || analysis.SourceByteLength != int(analysisFixture["source_byte_length"].(float64)) || analysis.Diagnostics[0].SchemaPath != diagnosticFixture["schema_path"].(string) {
		t.Fatalf("unexpected kaitai analysis: %+v", analysis)
	}
}

func TestSharedFixturePortableByteLocationContract(t *testing.T) {
	fixture := readParserFixtureFromPath(t, diagnosticsFixturePath(t, "portable_byte_location_contract"))
	rangeFixture := fixture["byte_range"].(map[string]any)
	pointFixture := fixture["source_point"].(map[string]any)
	editFixture := fixture["edit_span"].(map[string]any)
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
	editSpan := ByteEditSpan{
		StartByte:   int(editFixture["start_byte"].(float64)),
		OldEndByte:  int(editFixture["old_end_byte"].(float64)),
		NewEndByte:  int(editFixture["new_end_byte"].(float64)),
		StartPoint:  sourcePointFromFixture(editFixture["start_point"]),
		OldEndPoint: sourcePointFromFixture(editFixture["old_end_point"]),
		NewEndPoint: sourcePointFromFixture(editFixture["new_end_point"]),
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
	oldEditSlice, err := SliceByteRange(source, editSpan.OldRange())
	if err != nil {
		t.Fatalf("slice edit old range: %v", err)
	}
	if byteRange.Length() != int(expectedFixture["length"].(float64)) ||
		slice != expectedFixture["slice"].(string) ||
		byteRange.ContainsByte(byteRange.StartByte) != expectedFixture["contains_start"].(bool) ||
		byteRange.ContainsByte(byteRange.EndByte) != expectedFixture["contains_end"].(bool) ||
		byteRange.Overlaps(overlappingRange) != expectedFixture["overlaps"].(bool) ||
		byteRange.Overlaps(disjointRange) != expectedFixture["disjoint"].(bool) ||
		offset != int(expectedFixture["line_column_offset"].(float64)) ||
		editSpan.OldRange().Length() != int(expectedFixture["old_edit_length"].(float64)) ||
		editSpan.NewRange().Length() != int(expectedFixture["new_edit_length"].(float64)) ||
		editSpan.ByteDelta() != int(expectedFixture["edit_delta"].(float64)) ||
		oldEditSlice != expectedFixture["old_edit_slice"].(string) {
		t.Fatalf("unexpected byte location behavior: range=%+v slice=%q offset=%d edit=%+v", byteRange, slice, offset, editSpan)
	}
}

func TestSharedFixtureNormalizedTreeNodeContract(t *testing.T) {
	fixture := readParserFixture(t, "diagnostics", "slice-782-normalized-tree-node", "normalized-tree-node.json")
	roleFixture := fixture["node_roles"].([]any)
	roles := NodeRoles()
	if len(roles) != len(roleFixture) {
		t.Fatalf("unexpected node roles: %+v", roles)
	}
	for index, role := range roles {
		if string(role) != roleFixture[index].(string) {
			t.Fatalf("unexpected node role at %d: %s", index, role)
		}
	}

	nodeFixture := fixture["node"].(map[string]any)
	childFixture := fixture["child"].(map[string]any)
	childParentID := childFixture["parent_id"].(string)
	childFieldName := childFixture["field_name"].(string)
	node := NormalizedTreeNode{
		ID:             nodeFixture["id"].(string),
		Kind:           nodeFixture["kind"].(string),
		Role:           NodeRole(nodeFixture["role"].(string)),
		ParentID:       nil,
		ChildIDs:       stringSliceFromFixture(nodeFixture["child_ids"]),
		Span:           sourceSpanFromFixture(nodeFixture["span"]),
		FieldName:      nil,
		Named:          nodeFixture["named"].(bool),
		Anonymous:      nodeFixture["anonymous"].(bool),
		HasSourceText:  nodeFixture["has_source_text"].(bool),
		SourceFragment: nodeFixture["source_fragment"].(string),
	}
	child := NormalizedTreeNode{
		ID:             childFixture["id"].(string),
		Kind:           childFixture["kind"].(string),
		Role:           NodeRole(childFixture["role"].(string)),
		ParentID:       &childParentID,
		ChildIDs:       stringSliceFromFixture(childFixture["child_ids"]),
		Span:           sourceSpanFromFixture(childFixture["span"]),
		FieldName:      &childFieldName,
		Named:          childFixture["named"].(bool),
		Anonymous:      childFixture["anonymous"].(bool),
		HasSourceText:  childFixture["has_source_text"].(bool),
		SourceFragment: childFixture["source_fragment"].(string),
	}

	if node.Role != NodeRoleStructural || node.ChildIDs[1] != child.ID || child.ParentID == nil || *child.ParentID != node.ID || child.FieldName == nil || *child.FieldName != "declaration" || !child.HasSourceText {
		t.Fatalf("unexpected normalized tree nodes: node=%+v child=%+v", node, child)
	}
}

func sourceSpanFromFixture(value any) SourceSpan {
	fixture := value.(map[string]any)
	rangeFixture := fixture["range"].(map[string]any)
	return SourceSpan{
		Range: ByteRange{
			StartByte: int(rangeFixture["start_byte"].(float64)),
			EndByte:   int(rangeFixture["end_byte"].(float64)),
		},
		StartPoint: sourcePointFromFixture(fixture["start_point"]),
		EndPoint:   sourcePointFromFixture(fixture["end_point"]),
	}
}

func sourcePointFromFixture(value any) SourcePoint {
	fixture := value.(map[string]any)
	return SourcePoint{
		Row:    int(fixture["row"].(float64)),
		Column: int(fixture["column"].(float64)),
	}
}

func stringSliceFromFixture(value any) []string {
	fixture := value.([]any)
	items := make([]string, 0, len(fixture))
	for _, item := range fixture {
		items = append(items, item.(string))
	}
	return items
}

func TestSharedFixtureBinaryCoreContract(t *testing.T) {
	fixture := readParserFixtureFromPath(t, diagnosticsFixturePath(t, "binary_core_contract"))
	scalars := fixture["scalar_values"].([]any)
	payloadFixture := fixture["raw_payload"].(map[string]any)
	policiesFixture := fixture["render_policies"].([]any)
	reportFixture := fixture["merge_report"].(map[string]any)

	payloadRegions := payloadFixture["regions"].([]any)
	payload := BinaryRawPayload{
		Encoding:   payloadFixture["encoding"].(string),
		Value:      payloadFixture["value"].(string),
		ByteLength: int(payloadFixture["byte_length"].(float64)),
		Regions:    make([]BinaryPayloadRegion, 0, len(payloadRegions)),
	}
	for _, raw := range payloadRegions {
		item := raw.(map[string]any)
		payload.Regions = append(payload.Regions, BinaryPayloadRegion{
			Kind:        item["kind"].(string),
			SchemaPath:  item["schema_path"].(string),
			ByteRange:   *byteRangePointer(item["byte_range"]),
			ExpectedHex: item["expected_hex"].(string),
		})
	}
	payloadBytes, err := hex.DecodeString(payload.Value)
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	checksumRegion := payload.Regions[3]
	checksumHex := hex.EncodeToString(payloadBytes[checksumRegion.ByteRange.StartByte:checksumRegion.ByteRange.EndByte])
	if payload.Encoding != "hex" || len(payloadBytes) != payload.ByteLength || payload.Regions[0].Kind != "header" || payload.Regions[0].ByteRange.Length() != 8 || checksumHex != checksumRegion.ExpectedHex {
		t.Fatalf("unexpected binary raw payload: %+v", payload)
	}

	values := make([]BinaryScalarValue, 0, len(scalars))
	for _, raw := range scalars {
		item := raw.(map[string]any)
		values = append(values, BinaryScalarValue{
			Kind:        item["kind"].(string),
			Value:       item["value"],
			Symbol:      stringValue(item["symbol"]),
			RawValue:    item["raw_value"],
			Encoding:    stringValue(item["encoding"]),
			Format:      stringValue(item["format"]),
			Description: stringValue(item["description"]),
		})
	}
	if len(values) != 9 || values[0].Kind != "string" || values[8].Kind != "null" {
		t.Fatalf("unexpected scalar values: %+v", values)
	}

	policies := make([]BinaryRenderPolicy, 0, len(policiesFixture))
	for _, raw := range policiesFixture {
		item := raw.(map[string]any)
		policies = append(policies, BinaryRenderPolicy{
			SchemaPath:  item["schema_path"].(string),
			ByteRange:   byteRangePointer(item["byte_range"]),
			Operation:   item["operation"].(string),
			Disposition: item["disposition"].(string),
			Reason:      item["reason"].(string),
		})
	}
	if policies[0].Operation != "preserve" || policies[1].Disposition != "requires_renderer" || policies[2].Disposition != "unsafe" {
		t.Fatalf("unexpected render policies: %+v", policies)
	}

	report := BinaryMergeReport{
		Format:             reportFixture["format"].(string),
		Schema:             reportFixture["schema"].(string),
		MatchedSchemaPaths: stringSlice(reportFixture["matched_schema_paths"]),
		PreservedRanges:    byteRangeSlice(reportFixture["preserved_ranges"]),
		RewrittenNodes:     stringSlice(reportFixture["rewritten_nodes"]),
		ChecksumUpdates:    stringSlice(reportFixture["checksum_updates"]),
		NestedDispatches: []BinaryNestedDispatch{
			{
				SchemaPath: reportFixture["nested_dispatches"].([]any)[0].(map[string]any)["schema_path"].(string),
				Family:     reportFixture["nested_dispatches"].([]any)[0].(map[string]any)["family"].(string),
				Status:     reportFixture["nested_dispatches"].([]any)[0].(map[string]any)["status"].(string),
			},
		},
		Diagnostics: []BinaryDiagnostic{
			{
				Severity:   reportFixture["diagnostics"].([]any)[0].(map[string]any)["severity"].(string),
				Category:   reportFixture["diagnostics"].([]any)[0].(map[string]any)["category"].(string),
				Message:    reportFixture["diagnostics"].([]any)[0].(map[string]any)["message"].(string),
				SchemaPath: reportFixture["diagnostics"].([]any)[0].(map[string]any)["schema_path"].(string),
				ByteRange:  byteRangePointer(reportFixture["diagnostics"].([]any)[0].(map[string]any)["byte_range"]),
			},
		},
	}
	if report.Format != "png" || report.PreservedRanges[0].Length() != 25 || report.NestedDispatches[0].Family != "text" || report.Diagnostics[0].Category != "unsupported_checksum_rewrite" {
		t.Fatalf("unexpected binary merge report: %+v", report)
	}
}

func TestSharedFixtureZipFamilyContract(t *testing.T) {
	fixture := readParserFixtureFromPath(t, diagnosticsFixturePath(t, "zip_family_contract"))
	archiveFixture := fixture["archive"].(map[string]any)
	entriesFixture := fixture["entries"].([]any)
	decisionsFixture := fixture["member_decisions"].([]any)
	unsafeEntriesFixture := fixture["unsafe_entries"].([]any)
	reportFixture := fixture["merge_report"].(map[string]any)

	report := ZipFamilyReport{
		Archive: ZipArchiveInfo{
			Format:                archiveFixture["format"].(string),
			Schema:                archiveFixture["schema"].(string),
			EntryCount:            int(archiveFixture["entry_count"].(float64)),
			CentralDirectoryRange: *byteRangePointer(archiveFixture["central_directory_range"]),
		},
		Entries:         make([]ZipArchiveEntry, 0, len(entriesFixture)),
		MemberDecisions: make([]ZipMemberDecision, 0, len(decisionsFixture)),
		UnsafeEntries:   make([]ZipUnsafeEntry, 0, len(unsafeEntriesFixture)),
		MergeReport: BinaryMergeReport{
			Format:             reportFixture["format"].(string),
			Schema:             reportFixture["schema"].(string),
			MatchedSchemaPaths: stringSlice(reportFixture["matched_schema_paths"]),
			PreservedRanges:    byteRangeSlice(reportFixture["preserved_ranges"]),
			RewrittenNodes:     stringSlice(reportFixture["rewritten_nodes"]),
			ChecksumUpdates:    stringSlice(reportFixture["checksum_updates"]),
			NestedDispatches: []BinaryNestedDispatch{
				{
					SchemaPath: reportFixture["nested_dispatches"].([]any)[0].(map[string]any)["schema_path"].(string),
					Family:     reportFixture["nested_dispatches"].([]any)[0].(map[string]any)["family"].(string),
					Status:     reportFixture["nested_dispatches"].([]any)[0].(map[string]any)["status"].(string),
				},
				{
					SchemaPath: reportFixture["nested_dispatches"].([]any)[1].(map[string]any)["schema_path"].(string),
					Family:     reportFixture["nested_dispatches"].([]any)[1].(map[string]any)["family"].(string),
					Status:     reportFixture["nested_dispatches"].([]any)[1].(map[string]any)["status"].(string),
				},
			},
			Diagnostics: []BinaryDiagnostic{
				{
					Severity:   reportFixture["diagnostics"].([]any)[0].(map[string]any)["severity"].(string),
					Category:   reportFixture["diagnostics"].([]any)[0].(map[string]any)["category"].(string),
					Message:    reportFixture["diagnostics"].([]any)[0].(map[string]any)["message"].(string),
					SchemaPath: reportFixture["diagnostics"].([]any)[0].(map[string]any)["schema_path"].(string),
				},
			},
		},
	}
	for _, raw := range entriesFixture {
		item := raw.(map[string]any)
		report.Entries = append(report.Entries, ZipArchiveEntry{
			Path:                  item["path"].(string),
			NormalizedPath:        item["normalized_path"].(string),
			Directory:             item["directory"].(bool),
			Compression:           item["compression"].(string),
			CompressedSize:        int(item["compressed_size"].(float64)),
			UncompressedSize:      int(item["uncompressed_size"].(float64)),
			CRC32:                 item["crc32"].(string),
			LocalHeaderRange:      *byteRangePointer(item["local_header_range"]),
			DataRange:             *byteRangePointer(item["data_range"]),
			CentralDirectoryRange: *byteRangePointer(item["central_directory_range"]),
		})
	}
	for _, raw := range decisionsFixture {
		item := raw.(map[string]any)
		report.MemberDecisions = append(report.MemberDecisions, ZipMemberDecision{
			NormalizedPath: item["normalized_path"].(string),
			Operation:      item["operation"].(string),
			Disposition:    item["disposition"].(string),
			NestedFamily:   stringValue(item["nested_family"]),
			Reason:         item["reason"].(string),
		})
	}
	for _, raw := range unsafeEntriesFixture {
		item := raw.(map[string]any)
		report.UnsafeEntries = append(report.UnsafeEntries, ZipUnsafeEntry{
			Path:           item["path"].(string),
			NormalizedPath: item["normalized_path"].(string),
			Category:       item["category"].(string),
			Reason:         item["reason"].(string),
		})
	}

	if report.Archive.EntryCount != len(report.Entries) || report.Entries[1].NormalizedPath != "word/document.xml" {
		t.Fatalf("unexpected zip archive report: %+v", report)
	}
	if report.MemberDecisions[1].Operation != "delegate" || report.MemberDecisions[1].NestedFamily != "xml" || report.MemberDecisions[3].Disposition != "unsafe" {
		t.Fatalf("unexpected zip member decisions: %+v", report.MemberDecisions)
	}
	if report.UnsafeEntries[0].Category != "path_traversal" || report.UnsafeEntries[1].NormalizedPath != "config/settings.yml" || report.UnsafeEntries[2].Category != "encrypted_member" {
		t.Fatalf("unexpected zip unsafe entries: %+v", report.UnsafeEntries)
	}
	if report.MergeReport.Format != "zip" || report.MergeReport.PreservedRanges[0].Length() != 76 || report.MergeReport.NestedDispatches[1].Family != "yaml" || report.MergeReport.Diagnostics[0].Category != "dangerous_executable_mutation" {
		t.Fatalf("unexpected zip merge report: %+v", report.MergeReport)
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

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	return value.(string)
}

func byteRangePointer(value any) *ByteRange {
	if value == nil {
		return nil
	}
	fixture := value.(map[string]any)
	return &ByteRange{
		StartByte: int(fixture["start_byte"].(float64)),
		EndByte:   int(fixture["end_byte"].(float64)),
	}
}

func byteRangeSlice(value any) []ByteRange {
	values := value.([]any)
	result := make([]ByteRange, 0, len(values))
	for _, item := range values {
		byteRange := byteRangePointer(item)
		result = append(result, *byteRange)
	}
	return result
}

func stringSlice(value any) []string {
	values := value.([]any)
	result := make([]string, 0, len(values))
	for _, item := range values {
		result = append(result, item.(string))
	}
	return result
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

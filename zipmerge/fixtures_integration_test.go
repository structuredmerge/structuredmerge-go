package zipmerge

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/structuredmerge/structuredmerge-go/treehaver"
)

func TestParseZipInventory(t *testing.T) {
	source, err := NewStoredZip(map[string]string{
		"META-INF/MANIFEST.MF": "Manifest-Version: 1.0\n",
		"config/settings.yml":  "enabled: true\n",
	})
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}

	report, err := ParseZipInventory(source)
	if err != nil {
		t.Fatalf("parse zip inventory: %v", err)
	}

	if report.Archive.Format != "zip" || report.Archive.EntryCount != 2 {
		t.Fatalf("unexpected archive inventory: %+v", report.Archive)
	}
	if report.Entries[0].Compression != "stored" || report.Entries[0].DataRange.Length() == 0 {
		t.Fatalf("unexpected first entry: %+v", report.Entries[0])
	}
	if !report.Archive.CentralDirectoryRange.Valid() || report.Archive.CentralDirectoryRange.Length() == 0 {
		t.Fatalf("missing central directory range: %+v", report.Archive.CentralDirectoryRange)
	}
	if report.Entries[1].CentralDirectoryRange.StartByte < report.Archive.CentralDirectoryRange.StartByte {
		t.Fatalf("entry central directory range is outside archive range: %+v", report.Entries[1])
	}
}

func TestPlanZipMerge(t *testing.T) {
	ancestor := mustInventory(t, map[string]string{
		"META-INF/MANIFEST.MF": "Manifest-Version: 1.0\n",
		"config/settings.yml":  "enabled: true\n",
		"word/document.xml":    "<document>old</document>",
	})
	current := mustInventory(t, map[string]string{
		"META-INF/MANIFEST.MF": "Manifest-Version: 1.0\n",
		"config/settings.yml":  "enabled: true\n",
		"word/document.xml":    "<document>left</document>",
	})
	incoming := mustInventory(t, map[string]string{
		"META-INF/MANIFEST.MF":   "Manifest-Version: 1.0\n",
		"config/settings.yml":    "enabled: false\n",
		"word/document.xml":      "<document>right</document>",
		"new.json":               `{"enabled":true}`,
		"META-INF/SIGNATURE.RSA": "signature",
	})

	plan := PlanZipMerge(ancestor, current, incoming)
	decisions := map[string]string{}
	for _, decision := range plan.MemberDecisions {
		decisions[decision.NormalizedPath] = decision.Operation + ":" + decision.Disposition
	}

	if decisions["META-INF/MANIFEST.MF"] != "preserve:safe" {
		t.Fatalf("expected manifest to be preserved: %+v", plan.MemberDecisions)
	}
	if decisions["config/settings.yml"] != "delegate:requires_renderer" {
		t.Fatalf("expected YAML member delegation: %+v", plan.MemberDecisions)
	}
	if decisions["new.json"] != "add:requires_renderer" {
		t.Fatalf("expected JSON member add: %+v", plan.MemberDecisions)
	}
	if decisions["META-INF/SIGNATURE.RSA"] != "reject:unsafe" {
		t.Fatalf("expected signature member rejection: %+v", plan.MemberDecisions)
	}
	if len(plan.MergeReport.PreservedRanges) == 0 || len(plan.MergeReport.NestedDispatches) == 0 || len(plan.MergeReport.Diagnostics) == 0 {
		t.Fatalf("expected preserved ranges, nested dispatches, and diagnostics: %+v", plan.MergeReport)
	}
}

func TestRenderWithStandardLibrary(t *testing.T) {
	ancestor := mustInventory(t, map[string]string{
		"META-INF/MANIFEST.MF": "Manifest-Version: 1.0\n",
		"config/settings.yml":  "enabled: true\n",
	})
	currentSource := mustStoredZip(t, map[string]string{
		"META-INF/MANIFEST.MF": "Manifest-Version: 1.0\n",
		"config/settings.yml":  "enabled: true\n",
	})
	current, err := ParseZipInventory(currentSource)
	if err != nil {
		t.Fatalf("parse current zip: %v", err)
	}
	incoming := mustInventory(t, map[string]string{
		"META-INF/MANIFEST.MF": "Manifest-Version: 1.0\n",
		"config/settings.yml":  "enabled: false\n",
		"new.json":             `{"enabled":true}`,
	})
	plan := PlanZipMerge(ancestor, current, incoming)

	result, err := RenderWithStandardLibrary(RenderInput{
		Source: currentSource,
		Plan:   plan,
		MemberBytes: map[string][]byte{
			"config/settings.yml": []byte("enabled: false\n"),
			"new.json":            []byte(`{"enabled":true}`),
		},
	})
	if err != nil {
		t.Fatalf("render zip: %v", err)
	}
	entries, err := ReadZipEntries(result.Bytes)
	if err != nil {
		t.Fatalf("read rendered zip: %v", err)
	}

	if string(entries["META-INF/MANIFEST.MF"]) != "Manifest-Version: 1.0\n" || string(entries["config/settings.yml"]) != "enabled: false\n" || string(entries["new.json"]) != `{"enabled":true}` {
		t.Fatalf("unexpected rendered entries: %+v", entries)
	}
	if result.Inventory.Archive.EntryCount != 3 || !result.Inventory.Archive.CentralDirectoryRange.Valid() {
		t.Fatalf("unexpected rendered inventory: %+v", result.Inventory.Archive)
	}
	if len(result.Report.ChecksumUpdates) == 0 || len(result.Report.PreservedRanges) != 0 {
		t.Fatalf("unexpected render report: %+v", result.Report)
	}
}

func TestRenderWithRawPreservationCopiesPreservedLocalRecords(t *testing.T) {
	currentSource := mustDeflatedZip(t, map[string]string{
		"META-INF/MANIFEST.MF": "Manifest-Version: 1.0\n",
		"docs/readme.md":       "# Old\n",
	})
	ancestor, err := ParseZipInventory(currentSource)
	if err != nil {
		t.Fatalf("parse ancestor zip: %v", err)
	}
	current := ancestor
	incoming := mustInventoryFromSource(t, mustDeflatedZip(t, map[string]string{
		"META-INF/MANIFEST.MF": "Manifest-Version: 1.0\n",
		"docs/readme.md":       "# New\n",
	}))
	plan := PlanZipMerge(ancestor, current, incoming)

	result, err := RenderWithRawPreservation(RenderInput{
		Source: currentSource,
		Plan:   plan,
		MemberBytes: map[string][]byte{
			"docs/readme.md": []byte("# New\n"),
		},
		Options: RenderOptions{Compression: zip.Deflate, RawPreserve: true},
	})
	if err != nil {
		t.Fatalf("render zip with raw preservation: %v", err)
	}

	sourceInventory := mustInventoryFromSource(t, currentSource)
	sourceRaw := rawLocalRecordRanges(currentSource, entriesByNormalizedPath(sourceInventory.Entries))["META-INF/MANIFEST.MF"]
	outputRaw := rawLocalRecordRanges(result.Bytes, entriesByNormalizedPath(result.Inventory.Entries))["META-INF/MANIFEST.MF"]
	if !bytes.Equal(currentSource[sourceRaw.StartByte:sourceRaw.EndByte], result.Bytes[outputRaw.StartByte:outputRaw.EndByte]) {
		t.Fatalf("preserved member local record changed")
	}
	if len(result.Report.PreservedRanges) != 1 || result.Report.PreservedRanges[0] != sourceRaw {
		t.Fatalf("unexpected raw preservation evidence: %+v", result.Report.PreservedRanges)
	}
	entries, err := ReadZipEntries(result.Bytes)
	if err != nil {
		t.Fatalf("read rendered zip: %v", err)
	}
	if string(entries["docs/readme.md"]) != "# New\n" {
		t.Fatalf("nested rendered member was not applied: %q", entries["docs/readme.md"])
	}
}

func TestZipPlannerDispatchesNestedMarkdownMembers(t *testing.T) {
	ancestor := mustInventory(t, map[string]string{"docs/readme.md": "# Old\n"})
	current := mustInventory(t, map[string]string{"docs/readme.md": "# Left\n"})
	incoming := mustInventory(t, map[string]string{"docs/readme.md": "# Right\n"})

	plan := PlanZipMerge(ancestor, current, incoming)
	if len(plan.MergeReport.NestedDispatches) != 1 {
		t.Fatalf("expected one nested dispatch: %+v", plan.MergeReport.NestedDispatches)
	}
	dispatch := plan.MergeReport.NestedDispatches[0]
	if dispatch.Family != "markdown" || dispatch.SchemaPath != "/entries/by_path/docs/readme.md/data" || dispatch.Status != "planned" {
		t.Fatalf("unexpected nested dispatch: %+v", dispatch)
	}
}

func TestRawPreservationRejectsUnsupportedCompression(t *testing.T) {
	source := mustStoredZip(t, map[string]string{"payload.bin": "abc"})
	source = patchZipMethod(t, source, "payload.bin", 99)
	inventory := mustInventoryFromSource(t, source)
	plan := PlanZipMerge(inventory, inventory, inventory)

	_, err := RenderWithRawPreservation(RenderInput{Source: source, Plan: plan})
	if err == nil || !strings.Contains(err.Error(), "unsupported compression") {
		t.Fatalf("expected unsupported compression rejection, got %v", err)
	}
	expectRenderErrorCategory(t, err, "unsupported_compression")
}

func TestRawPreservationRejectsArchiveComments(t *testing.T) {
	source := appendArchiveComment(mustStoredZip(t, map[string]string{"payload.bin": "abc"}), "comment")
	inventory := mustInventoryFromSource(t, source)
	plan := PlanZipMerge(inventory, inventory, inventory)

	_, err := RenderWithRawPreservation(RenderInput{Source: source, Plan: plan})
	if err == nil || !strings.Contains(err.Error(), "archive comments") {
		t.Fatalf("expected archive comment rejection, got %v", err)
	}
	expectRenderErrorCategory(t, err, "archive_comment")
}

func TestRawPreservationRejectsExtraFieldsAndMemberComments(t *testing.T) {
	for name, source := range map[string][]byte{
		"extra":   mustStoredZipWithExtra(t),
		"comment": mustStoredZipWithMemberComment(t),
	} {
		t.Run(name, func(t *testing.T) {
			inventory := mustInventoryFromSource(t, source)
			plan := PlanZipMerge(inventory, inventory, inventory)

			_, err := RenderWithRawPreservation(RenderInput{Source: source, Plan: plan})
			if err == nil || (!strings.Contains(err.Error(), "extra fields") && !strings.Contains(err.Error(), "member comments")) {
				t.Fatalf("expected metadata rejection, got %v", err)
			}
			if name == "extra" {
				expectRenderErrorCategory(t, err, "central_directory_extra_field")
			} else {
				expectRenderErrorCategory(t, err, "member_comment")
			}
		})
	}
}

func TestZipPlannerRejectsEncryptedMembers(t *testing.T) {
	source := patchZipFlag(t, mustStoredZip(t, map[string]string{"secret.txt": "abc"}), "secret.txt", 0x1)
	inventory := mustInventoryFromSource(t, source)
	plan := PlanZipMerge(inventory, inventory, inventory)

	if len(plan.MemberDecisions) != 1 || plan.MemberDecisions[0].Operation != "reject" {
		t.Fatalf("expected encrypted member rejection: %+v", plan.MemberDecisions)
	}
	if len(plan.UnsafeEntries) != 1 || plan.UnsafeEntries[0].Category != "encrypted_member" {
		t.Fatalf("expected encrypted unsafe entry: %+v", plan.UnsafeEntries)
	}
}

func TestNormalizeZipPathDoesNotTreatInternalParentSegmentsAsTraversal(t *testing.T) {
	normalized := normalizeZipPath("config/../config/settings.yml")
	if normalized != "config/settings.yml" || escapesArchiveRoot("config/../config/settings.yml") {
		t.Fatalf("unexpected internal parent segment handling: %q", normalized)
	}
	if !escapesArchiveRoot("../evil.sh") || !escapesArchiveRoot("/absolute/path") {
		t.Fatalf("expected root-escaping ZIP paths to be unsafe")
	}
}

func TestSharedFixtureZipRawPreservationEdgeCases(t *testing.T) {
	fixture := readSharedFixture(t)
	success := fixture["success"].(map[string]any)
	if success["preserved_member"] != "META-INF/MANIFEST.MF" || success["expected_nested_family"] != "markdown" {
		t.Fatalf("unexpected slice-736 success fixture: %+v", success)
	}

	rejections := fixture["rejections"].([]any)
	categories := map[string]string{}
	for _, item := range rejections {
		rejection := item.(map[string]any)
		categories[rejection["label"].(string)] = rejection["category"].(string)
	}
	for _, label := range []string{"unsupported-compression", "archive-comment", "central-extra-field", "member-comment", "encrypted-member"} {
		if categories[label] == "" {
			t.Fatalf("missing slice-736 rejection category for %s: %+v", label, categories)
		}
	}
}

func mustInventory(t *testing.T, entries map[string]string) treehaver.ZipFamilyReport {
	t.Helper()
	source := mustStoredZip(t, entries)
	return mustInventoryFromSource(t, source)
}

func mustInventoryFromSource(t *testing.T, source []byte) treehaver.ZipFamilyReport {
	t.Helper()
	report, err := ParseZipInventory(source)
	if err != nil {
		t.Fatalf("parse zip inventory: %v", err)
	}
	return report
}

func mustStoredZip(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	source, err := NewStoredZip(entries)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	return source
}

func mustDeflatedZip(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	source, err := NewDeflatedZip(entries)
	if err != nil {
		t.Fatalf("create deflated zip: %v", err)
	}
	return source
}

func mustStoredZipWithExtra(t *testing.T) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	fileWriter, err := writer.CreateHeader(&zip.FileHeader{
		Name:   "payload.bin",
		Method: zip.Store,
		Extra:  []byte{0xca, 0xfe, 0x00, 0x00},
	})
	if err != nil {
		t.Fatalf("create extra field entry: %v", err)
	}
	if _, err := io.WriteString(fileWriter, "abc"); err != nil {
		t.Fatalf("write extra field entry: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close extra field zip: %v", err)
	}
	return buffer.Bytes()
}

func mustStoredZipWithMemberComment(t *testing.T) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	fileWriter, err := writer.CreateHeader(&zip.FileHeader{
		Name:    "payload.bin",
		Method:  zip.Store,
		Comment: "member comment",
	})
	if err != nil {
		t.Fatalf("create commented entry: %v", err)
	}
	if _, err := io.WriteString(fileWriter, "abc"); err != nil {
		t.Fatalf("write commented entry: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close commented zip: %v", err)
	}
	return buffer.Bytes()
}

func patchZipMethod(t *testing.T, source []byte, name string, method uint16) []byte {
	t.Helper()
	patched := slicesClone(source)
	central := mustCentralDirectory(t, patched)
	record := central.Records[name]
	binary.LittleEndian.PutUint16(patched[record.LocalHeaderOffset+8:], method)
	binary.LittleEndian.PutUint16(patched[record.Range.StartByte+10:], method)
	return patched
}

func patchZipFlag(t *testing.T, source []byte, name string, flag uint16) []byte {
	t.Helper()
	patched := slicesClone(source)
	central := mustCentralDirectory(t, patched)
	record := central.Records[name]
	localFlags := binary.LittleEndian.Uint16(patched[record.LocalHeaderOffset+6:])
	centralFlags := binary.LittleEndian.Uint16(patched[record.Range.StartByte+8:])
	binary.LittleEndian.PutUint16(patched[record.LocalHeaderOffset+6:], localFlags|flag)
	binary.LittleEndian.PutUint16(patched[record.Range.StartByte+8:], centralFlags|flag)
	return patched
}

func appendArchiveComment(source []byte, comment string) []byte {
	patched := append(slicesClone(source), []byte(comment)...)
	binary.LittleEndian.PutUint16(patched[len(source)-2:], uint16(len(comment)))
	return patched
}

func mustCentralDirectory(t *testing.T, source []byte) centralDirectoryScan {
	t.Helper()
	central, err := scanCentralDirectory(source)
	if err != nil {
		t.Fatalf("scan central directory: %v", err)
	}
	return central
}

func slicesClone(source []byte) []byte {
	return append([]byte{}, source...)
}

func expectRenderErrorCategory(t *testing.T, err error, category string) {
	t.Helper()
	var renderErr RenderError
	if !errors.As(err, &renderErr) {
		t.Fatalf("expected RenderError, got %T", err)
	}
	if renderErr.Diagnostic.Category != category || renderErr.Diagnostic.Severity != "error" {
		t.Fatalf("unexpected render diagnostic: %+v", renderErr.Diagnostic)
	}
}

func readSharedFixture(t *testing.T) map[string]any {
	t.Helper()
	path := filepath.Join("..", "..", "fixtures", "diagnostics", "slice-736-zip-raw-preservation-edge-cases", "zip-raw-preservation-edge-cases.json")
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read shared fixture: %v", err)
	}
	var fixture map[string]any
	if err := json.Unmarshal(bytes, &fixture); err != nil {
		t.Fatalf("parse shared fixture: %v", err)
	}
	return fixture
}

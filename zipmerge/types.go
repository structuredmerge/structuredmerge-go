package zipmerge

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	pathpkg "path"
	"slices"
	"strings"

	"github.com/structuredmerge/structuredmerge-go/treehaver"
)

const (
	localFileHeaderSignature     = 0x04034b50
	centralDirectorySignature    = 0x02014b50
	endCentralDirectorySignature = 0x06054b50
)

var dosEpoch = []byte{0x00, 0x00, 0x21, 0x00}

func ParseZipInventory(source []byte) (treehaver.ZipFamilyReport, error) {
	reader, err := zip.NewReader(bytes.NewReader(source), int64(len(source)))
	if err != nil {
		return treehaver.ZipFamilyReport{}, err
	}

	centralDirectory, err := scanCentralDirectory(source)
	if err != nil {
		return treehaver.ZipFamilyReport{}, err
	}
	localHeaders, err := scanLocalHeaders(source, centralDirectory.Records)
	if err != nil {
		return treehaver.ZipFamilyReport{}, err
	}

	report := treehaver.ZipFamilyReport{
		Archive: treehaver.ZipArchiveInfo{
			Format:                "zip",
			Schema:                "zip.ksy",
			EntryCount:            len(reader.File),
			CentralDirectoryRange: centralDirectory.Range,
		},
		Entries:         make([]treehaver.ZipArchiveEntry, 0, len(reader.File)),
		MemberDecisions: []treehaver.ZipMemberDecision{},
		UnsafeEntries:   []treehaver.ZipUnsafeEntry{},
		MergeReport: treehaver.BinaryMergeReport{
			Format:             "zip",
			Schema:             "zip.ksy",
			MatchedSchemaPaths: []string{},
			PreservedRanges:    []treehaver.ByteRange{},
			RewrittenNodes:     []string{},
			ChecksumUpdates:    []string{},
			NestedDispatches:   []treehaver.BinaryNestedDispatch{},
			Diagnostics:        []treehaver.BinaryDiagnostic{},
		},
	}

	for _, file := range reader.File {
		dataOffset, err := file.DataOffset()
		if err != nil {
			return treehaver.ZipFamilyReport{}, err
		}
		local, ok := localHeaders[file.Name]
		if !ok {
			return treehaver.ZipFamilyReport{}, fmt.Errorf("missing local header for %q", file.Name)
		}
		central, ok := centralDirectory.Records[file.Name]
		if !ok {
			return treehaver.ZipFamilyReport{}, fmt.Errorf("missing central directory record for %q", file.Name)
		}

		entry := treehaver.ZipArchiveEntry{
			Path:                  file.Name,
			NormalizedPath:        normalizeZipPath(file.Name),
			Directory:             file.FileInfo().IsDir(),
			Compression:           compressionName(file.Method),
			CompressedSize:        int(file.CompressedSize64),
			UncompressedSize:      int(file.UncompressedSize64),
			CRC32:                 fmt.Sprintf("%08x", file.CRC32),
			LocalHeaderRange:      treehaver.ByteRange{StartByte: local.StartByte, EndByte: int(dataOffset)},
			DataRange:             treehaver.ByteRange{StartByte: int(dataOffset), EndByte: int(dataOffset) + int(file.CompressedSize64)},
			CentralDirectoryRange: central.Range,
		}
		report.Entries = append(report.Entries, entry)
	}
	slices.SortFunc(report.Entries, func(left, right treehaver.ZipArchiveEntry) int {
		if left.LocalHeaderRange.StartByte < right.LocalHeaderRange.StartByte {
			return -1
		}
		if left.LocalHeaderRange.StartByte > right.LocalHeaderRange.StartByte {
			return 1
		}
		return 0
	})
	report.UnsafeEntries = unsafeEntries(report.Entries, reader.File)

	return report, nil
}

func PlanZipMerge(ancestor treehaver.ZipFamilyReport, current treehaver.ZipFamilyReport, incoming treehaver.ZipFamilyReport) treehaver.ZipFamilyReport {
	report := treehaver.ZipFamilyReport{
		Archive:         incoming.Archive,
		Entries:         incoming.Entries,
		MemberDecisions: []treehaver.ZipMemberDecision{},
		UnsafeEntries:   append([]treehaver.ZipUnsafeEntry{}, incoming.UnsafeEntries...),
		MergeReport: treehaver.BinaryMergeReport{
			Format:             "zip",
			Schema:             "zip.ksy",
			MatchedSchemaPaths: []string{},
			PreservedRanges:    []treehaver.ByteRange{},
			RewrittenNodes:     []string{},
			ChecksumUpdates:    []string{},
			NestedDispatches:   []treehaver.BinaryNestedDispatch{},
			Diagnostics:        []treehaver.BinaryDiagnostic{},
		},
	}

	ancestorEntries := entriesByNormalizedPath(ancestor.Entries)
	currentEntries := entriesByNormalizedPath(current.Entries)
	incomingEntries := entriesByNormalizedPath(incoming.Entries)
	unsafeByPath := map[string]treehaver.ZipUnsafeEntry{}
	for _, unsafe := range report.UnsafeEntries {
		unsafeByPath[unsafe.NormalizedPath] = unsafe
	}

	paths := unionPaths(ancestorEntries, currentEntries, incomingEntries)
	for _, normalizedPath := range paths {
		ancestorEntry, hadAncestor := ancestorEntries[normalizedPath]
		currentEntry, hasCurrent := currentEntries[normalizedPath]
		incomingEntry, hasIncoming := incomingEntries[normalizedPath]
		nestedFamily := nestedFamilyFor(normalizedPath)

		if unsafe, ok := unsafeByPath[normalizedPath]; ok {
			report.MemberDecisions = append(report.MemberDecisions, treehaver.ZipMemberDecision{
				NormalizedPath: normalizedPath,
				Operation:      "reject",
				Disposition:    "unsafe",
				Reason:         unsafe.Reason,
			})
			report.MergeReport.Diagnostics = append(report.MergeReport.Diagnostics, unsafeDiagnostic(unsafe, incomingEntry))
			continue
		}

		switch {
		case !hasCurrent && hasIncoming:
			report.MemberDecisions = append(report.MemberDecisions, treehaver.ZipMemberDecision{
				NormalizedPath: normalizedPath,
				Operation:      "add",
				Disposition:    "requires_renderer",
				Reason:         "member exists only in incoming archive",
			})
			report.MergeReport.RewrittenNodes = append(report.MergeReport.RewrittenNodes, schemaPathFor(normalizedPath))
		case hasCurrent && !hasIncoming:
			report.MemberDecisions = append(report.MemberDecisions, treehaver.ZipMemberDecision{
				NormalizedPath: normalizedPath,
				Operation:      "delete",
				Disposition:    "requires_renderer",
				Reason:         "member was removed from incoming archive",
			})
			report.MergeReport.RewrittenNodes = append(report.MergeReport.RewrittenNodes, schemaPathFor(normalizedPath))
		case hadAncestor && sameZipEntry(currentEntry, ancestorEntry) && sameZipEntry(incomingEntry, ancestorEntry):
			report.MemberDecisions = append(report.MemberDecisions, treehaver.ZipMemberDecision{
				NormalizedPath: normalizedPath,
				Operation:      "preserve",
				Disposition:    "safe",
				Reason:         "member is unchanged from ancestor",
			})
			report.MergeReport.PreservedRanges = append(report.MergeReport.PreservedRanges, currentEntry.LocalHeaderRange, currentEntry.DataRange)
		case nestedFamily != "":
			report.MemberDecisions = append(report.MemberDecisions, treehaver.ZipMemberDecision{
				NormalizedPath: normalizedPath,
				Operation:      "delegate",
				Disposition:    "requires_renderer",
				NestedFamily:   nestedFamily,
				Reason:         "structured member can be merged by a nested family before ZIP rendering",
			})
			report.MergeReport.NestedDispatches = append(report.MergeReport.NestedDispatches, treehaver.BinaryNestedDispatch{
				SchemaPath: schemaPathFor(normalizedPath) + "/data",
				Family:     nestedFamily,
				Status:     "planned",
			})
			report.MergeReport.RewrittenNodes = append(report.MergeReport.RewrittenNodes, schemaPathFor(normalizedPath))
			report.MergeReport.ChecksumUpdates = append(report.MergeReport.ChecksumUpdates, schemaPathFor(normalizedPath)+"/crc32")
		case hadAncestor && !sameZipEntry(currentEntry, ancestorEntry) && !sameZipEntry(incomingEntry, ancestorEntry) && !sameZipEntry(currentEntry, incomingEntry):
			report.MemberDecisions = append(report.MemberDecisions, treehaver.ZipMemberDecision{
				NormalizedPath: normalizedPath,
				Operation:      "reject",
				Disposition:    "conflict",
				Reason:         "member changed differently in current and incoming archives",
			})
			report.MergeReport.Diagnostics = append(report.MergeReport.Diagnostics, treehaver.BinaryDiagnostic{
				Severity:   "error",
				Category:   "zip_member_conflict",
				Message:    fmt.Sprintf("ZIP member %s changed differently in both archives.", normalizedPath),
				SchemaPath: schemaPathFor(normalizedPath),
			})
		default:
			report.MemberDecisions = append(report.MemberDecisions, treehaver.ZipMemberDecision{
				NormalizedPath: normalizedPath,
				Operation:      "rewrite",
				Disposition:    "requires_renderer",
				Reason:         "member bytes or metadata changed",
			})
			report.MergeReport.RewrittenNodes = append(report.MergeReport.RewrittenNodes, schemaPathFor(normalizedPath))
			report.MergeReport.ChecksumUpdates = append(report.MergeReport.ChecksumUpdates, schemaPathFor(normalizedPath)+"/crc32")
		}
		report.MergeReport.MatchedSchemaPaths = append(report.MergeReport.MatchedSchemaPaths, schemaPathFor(normalizedPath))
	}

	if len(report.MergeReport.RewrittenNodes) > 0 || len(report.MergeReport.ChecksumUpdates) > 0 {
		report.MergeReport.RewrittenNodes = append(report.MergeReport.RewrittenNodes, "/central_directory")
		report.MergeReport.ChecksumUpdates = append(report.MergeReport.ChecksumUpdates, "/central_directory/size", "/central_directory/offset")
	}
	return report
}

type localHeaderInfo struct {
	StartByte int
	DataEnd   int
}

type centralDirectoryInfo struct {
	Range              treehaver.ByteRange
	LocalHeaderOffset  int
	CompressedSize     int
	UncompressedSize   int
	GeneralPurposeFlag uint16
	CompressionMethod  uint16
	ExtraLength        int
	CommentLength      int
}

type centralDirectoryScan struct {
	Range          treehaver.ByteRange
	Records        map[string]centralDirectoryInfo
	ArchiveComment bool
	MultiDisk      bool
}

type centralRecord struct {
	Name               string
	Method             uint16
	CRC32              uint32
	CompressedSize     uint32
	UncompressedSize   uint32
	LocalHeaderOffset  uint32
	GeneralPurposeFlag uint16
	ExternalAttrs      uint32
}

type RenderOptions struct {
	Compression uint16
	RawPreserve bool
}

type RenderInput struct {
	Source      []byte
	Plan        treehaver.ZipFamilyReport
	MemberBytes map[string][]byte
	Options     RenderOptions
}

type RenderResult struct {
	Bytes     []byte
	Inventory treehaver.ZipFamilyReport
	Report    treehaver.BinaryMergeReport
}

type RenderError struct {
	Diagnostic treehaver.BinaryDiagnostic
}

func (err RenderError) Error() string {
	return err.Diagnostic.Message
}

func RenderWithStandardLibrary(input RenderInput) (RenderResult, error) {
	if input.Options.RawPreserve {
		return RenderWithRawPreservation(input)
	}
	if input.Options.Compression == 0 {
		input.Options.Compression = zip.Store
	}
	sourceEntries, err := ReadZipEntries(input.Source)
	if err != nil {
		return RenderResult{}, err
	}

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	entries := entriesByNormalizedPath(input.Plan.Entries)
	for _, decision := range input.Plan.MemberDecisions {
		switch decision.Operation {
		case "reject":
			return RenderResult{}, fmt.Errorf("cannot render rejected ZIP member %s: %s", decision.NormalizedPath, decision.Reason)
		case "delete":
			continue
		case "preserve":
			content, ok := sourceEntries[decision.NormalizedPath]
			if !ok {
				return RenderResult{}, fmt.Errorf("missing source bytes for preserved ZIP member %s", decision.NormalizedPath)
			}
			if err := writeZipMember(writer, entries[decision.NormalizedPath], content, input.Options.Compression); err != nil {
				return RenderResult{}, err
			}
		case "add", "rewrite", "delegate":
			content, ok := input.MemberBytes[decision.NormalizedPath]
			if !ok {
				return RenderResult{}, fmt.Errorf("missing rendered bytes for ZIP member %s", decision.NormalizedPath)
			}
			if err := writeZipMember(writer, entries[decision.NormalizedPath], content, input.Options.Compression); err != nil {
				return RenderResult{}, err
			}
		default:
			return RenderResult{}, fmt.Errorf("unsupported ZIP render operation %q", decision.Operation)
		}
	}
	if err := writer.Close(); err != nil {
		return RenderResult{}, err
	}

	rendered := buffer.Bytes()
	inventory, err := ParseZipInventory(rendered)
	if err != nil {
		return RenderResult{}, err
	}
	report := input.Plan.MergeReport
	report.PreservedRanges = []treehaver.ByteRange{}
	report.RewrittenNodes = appendUnique(report.RewrittenNodes, "/central_directory")
	report.ChecksumUpdates = appendUnique(report.ChecksumUpdates, "/central_directory/size", "/central_directory/offset")
	return RenderResult{Bytes: rendered, Inventory: inventory, Report: report}, nil
}

func RenderWithRawPreservation(input RenderInput) (RenderResult, error) {
	method := input.Options.Compression
	if method == 0 {
		method = zip.Store
	}
	if method != zip.Store && method != zip.Deflate {
		return RenderResult{}, renderError("unsupported_compression", "/render/options/compression", "unsupported raw-preserving compression method")
	}

	sourceInventory, err := ParseZipInventory(input.Source)
	if err != nil {
		return RenderResult{}, err
	}
	sourceCentral, err := scanCentralDirectory(input.Source)
	if err != nil {
		return RenderResult{}, err
	}
	sourceEntries := entriesByNormalizedPath(sourceInventory.Entries)
	sourceRawRanges := rawLocalRecordRanges(input.Source, sourceEntries)
	entries := entriesByNormalizedPath(input.Plan.Entries)
	var buffer bytes.Buffer
	central := []centralRecord{}
	preservedRanges := []treehaver.ByteRange{}

	for _, decision := range input.Plan.MemberDecisions {
		entry, ok := entries[decision.NormalizedPath]
		if !ok && decision.Operation != "delete" {
			return RenderResult{}, fmt.Errorf("missing planned ZIP entry %s", decision.NormalizedPath)
		}
		switch decision.Operation {
		case "reject":
			return RenderResult{}, fmt.Errorf("cannot render rejected ZIP member %s: %s", decision.NormalizedPath, decision.Reason)
		case "delete":
			continue
		case "preserve":
			sourceEntry, ok := sourceEntries[decision.NormalizedPath]
			if !ok {
				return RenderResult{}, fmt.Errorf("missing source entry for preserved ZIP member %s", decision.NormalizedPath)
			}
			if err := validateRawPreserveEntry(input.Source, sourceCentral, sourceEntry); err != nil {
				return RenderResult{}, err
			}
			rawRange, ok := sourceRawRanges[decision.NormalizedPath]
			if !ok || !rawRange.Valid() || rawRange.EndByte > len(input.Source) {
				return RenderResult{}, fmt.Errorf("missing raw source range for preserved ZIP member %s", decision.NormalizedPath)
			}
			offset := buffer.Len()
			buffer.Write(input.Source[rawRange.StartByte:rawRange.EndByte])
			record, err := centralRecordFromRawEntry(input.Source, sourceEntry, offset)
			if err != nil {
				return RenderResult{}, err
			}
			central = append(central, record)
			preservedRanges = append(preservedRanges, rawRange)
		case "add", "rewrite", "delegate":
			content, ok := input.MemberBytes[decision.NormalizedPath]
			if !ok {
				return RenderResult{}, fmt.Errorf("missing rendered bytes for ZIP member %s", decision.NormalizedPath)
			}
			rendered, record, err := renderedLocalRecord(entry, content, method, buffer.Len())
			if err != nil {
				return RenderResult{}, err
			}
			buffer.Write(rendered)
			central = append(central, record)
		default:
			return RenderResult{}, fmt.Errorf("unsupported ZIP render operation %q", decision.Operation)
		}
	}

	centralStart := buffer.Len()
	for _, record := range central {
		writeCentralDirectoryRecord(&buffer, record)
	}
	centralSize := buffer.Len() - centralStart
	writeEndOfCentralDirectory(&buffer, len(central), centralSize, centralStart)

	rendered := buffer.Bytes()
	inventory, err := ParseZipInventory(rendered)
	if err != nil {
		return RenderResult{}, err
	}
	report := input.Plan.MergeReport
	report.PreservedRanges = preservedRanges
	report.RewrittenNodes = appendUnique(report.RewrittenNodes, "/central_directory")
	report.ChecksumUpdates = appendUnique(report.ChecksumUpdates, "/central_directory/size", "/central_directory/offset")
	return RenderResult{Bytes: rendered, Inventory: inventory, Report: report}, nil
}

func scanLocalHeaders(source []byte, centralRecords map[string]centralDirectoryInfo) (map[string]localHeaderInfo, error) {
	headers := map[string]localHeaderInfo{}
	for name, central := range centralRecords {
		cursor := central.LocalHeaderOffset
		if cursor < 0 || cursor+30 > len(source) {
			return nil, fmt.Errorf("invalid ZIP local header offset %d", cursor)
		}
		signature := binary.LittleEndian.Uint32(source[cursor:])
		if signature != localFileHeaderSignature {
			return nil, fmt.Errorf("unexpected ZIP local header signature at byte %d", cursor)
		}
		nameLength := int(binary.LittleEndian.Uint16(source[cursor+26:]))
		extraLength := int(binary.LittleEndian.Uint16(source[cursor+28:]))
		nameStart := cursor + 30
		nameEnd := nameStart + nameLength
		dataStart := nameEnd + extraLength
		dataEnd := dataStart + central.CompressedSize
		if nameEnd > len(source) || dataEnd > len(source) {
			return nil, fmt.Errorf("ZIP local file %d extends past archive end", cursor)
		}
		localName := string(source[nameStart:nameEnd])
		if localName != name {
			return nil, fmt.Errorf("ZIP local header name %q does not match central directory name %q", localName, name)
		}
		headers[name] = localHeaderInfo{StartByte: cursor, DataEnd: dataEnd}
	}
	return headers, nil
}

func scanCentralDirectory(source []byte) (centralDirectoryScan, error) {
	eocdOffset := -1
	for cursor := len(source) - 22; cursor >= 0; cursor-- {
		if binary.LittleEndian.Uint32(source[cursor:]) == endCentralDirectorySignature {
			eocdOffset = cursor
			break
		}
	}
	if eocdOffset < 0 {
		return centralDirectoryScan{}, fmt.Errorf("missing ZIP end of central directory")
	}
	centralSize := int(binary.LittleEndian.Uint32(source[eocdOffset+12:]))
	centralOffset := int(binary.LittleEndian.Uint32(source[eocdOffset+16:]))
	archiveCommentLength := int(binary.LittleEndian.Uint16(source[eocdOffset+20:]))
	diskNumber := binary.LittleEndian.Uint16(source[eocdOffset+4:])
	centralDisk := binary.LittleEndian.Uint16(source[eocdOffset+6:])
	centralEnd := centralOffset + centralSize
	if centralEnd > eocdOffset {
		return centralDirectoryScan{}, fmt.Errorf("central directory overlaps end record")
	}
	if eocdOffset+22+archiveCommentLength > len(source) {
		return centralDirectoryScan{}, fmt.Errorf("ZIP archive comment extends past archive end")
	}

	records := map[string]centralDirectoryInfo{}
	cursor := centralOffset
	for cursor < centralEnd {
		if cursor+46 > len(source) || binary.LittleEndian.Uint32(source[cursor:]) != centralDirectorySignature {
			return centralDirectoryScan{}, fmt.Errorf("unexpected central directory record at byte %d", cursor)
		}
		nameLength := int(binary.LittleEndian.Uint16(source[cursor+28:]))
		extraLength := int(binary.LittleEndian.Uint16(source[cursor+30:]))
		commentLength := int(binary.LittleEndian.Uint16(source[cursor+32:]))
		nameStart := cursor + 46
		nameEnd := nameStart + nameLength
		recordEnd := nameEnd + extraLength + commentLength
		if recordEnd > len(source) {
			return centralDirectoryScan{}, fmt.Errorf("central directory record extends past archive end")
		}
		records[string(source[nameStart:nameEnd])] = centralDirectoryInfo{
			Range:              treehaver.ByteRange{StartByte: cursor, EndByte: recordEnd},
			LocalHeaderOffset:  int(binary.LittleEndian.Uint32(source[cursor+42:])),
			CompressedSize:     int(binary.LittleEndian.Uint32(source[cursor+20:])),
			UncompressedSize:   int(binary.LittleEndian.Uint32(source[cursor+24:])),
			GeneralPurposeFlag: binary.LittleEndian.Uint16(source[cursor+8:]),
			CompressionMethod:  binary.LittleEndian.Uint16(source[cursor+10:]),
			ExtraLength:        extraLength,
			CommentLength:      commentLength,
		}
		cursor = recordEnd
	}

	return centralDirectoryScan{
		Range:          treehaver.ByteRange{StartByte: centralOffset, EndByte: centralEnd},
		Records:        records,
		ArchiveComment: archiveCommentLength > 0,
		MultiDisk:      diskNumber != 0 || centralDisk != 0,
	}, nil
}

func normalizeZipPath(path string) string {
	cleaned := pathpkg.Clean(strings.ReplaceAll(path, "\\", "/"))
	if cleaned == "." {
		return ""
	}
	if strings.HasSuffix(path, "/") && !strings.HasSuffix(cleaned, "/") {
		cleaned += "/"
	}
	return cleaned
}

func compressionName(method uint16) string {
	switch method {
	case zip.Store:
		return "stored"
	case zip.Deflate:
		return "deflate"
	default:
		return fmt.Sprintf("method-%d", method)
	}
}

func NewStoredZip(entries map[string]string) ([]byte, error) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		content := entries[name]
		header := &zip.FileHeader{
			Name:               name,
			Method:             zip.Store,
			CRC32:              crc32.ChecksumIEEE([]byte(content)),
			CompressedSize64:   uint64(len(content)),
			UncompressedSize64: uint64(len(content)),
		}
		fileWriter, err := writer.CreateHeader(header)
		if err != nil {
			return nil, err
		}
		if _, err := io.WriteString(fileWriter, content); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func NewDeflatedZip(entries map[string]string) ([]byte, error) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		fileWriter, err := writer.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate})
		if err != nil {
			return nil, err
		}
		if _, err := io.WriteString(fileWriter, entries[name]); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func ReadZipEntries(source []byte) (map[string][]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(source), int64(len(source)))
	if err != nil {
		return nil, err
	}
	result := map[string][]byte{}
	for _, file := range reader.File {
		readCloser, err := file.Open()
		if err != nil {
			return nil, err
		}
		content, readErr := io.ReadAll(readCloser)
		closeErr := readCloser.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		result[normalizeZipPath(file.Name)] = content
	}
	return result, nil
}

func unsafeEntries(entries []treehaver.ZipArchiveEntry, files []*zip.File) []treehaver.ZipUnsafeEntry {
	result := []treehaver.ZipUnsafeEntry{}
	seen := map[string]string{}
	filesByName := map[string]*zip.File{}
	for _, file := range files {
		filesByName[file.Name] = file
	}
	for _, entry := range entries {
		if escapesArchiveRoot(entry.Path) {
			result = append(result, treehaver.ZipUnsafeEntry{
				Path:           entry.Path,
				NormalizedPath: entry.NormalizedPath,
				Category:       "path_traversal",
				Reason:         "entry escapes the archive root",
			})
		}
		if previous, ok := seen[entry.NormalizedPath]; ok && previous != entry.Path {
			result = append(result, treehaver.ZipUnsafeEntry{
				Path:           entry.Path,
				NormalizedPath: entry.NormalizedPath,
				Category:       "duplicate_normalized_path",
				Reason:         "normalized path collides with an existing entry",
			})
		}
		seen[entry.NormalizedPath] = entry.Path
		if file, ok := filesByName[entry.Path]; ok && file.Flags&0x1 == 0x1 {
			result = append(result, treehaver.ZipUnsafeEntry{
				Path:           entry.Path,
				NormalizedPath: entry.NormalizedPath,
				Category:       "encrypted_member",
				Reason:         "encrypted member cannot be rendered by the default provider",
			})
		}
		if signingSensitivePath(entry.NormalizedPath) {
			result = append(result, treehaver.ZipUnsafeEntry{
				Path:           entry.Path,
				NormalizedPath: entry.NormalizedPath,
				Category:       "signing_sensitive_member",
				Reason:         "signature-bearing member mutation is not enabled",
			})
		}
	}
	return result
}

func escapesArchiveRoot(path string) bool {
	path = strings.ReplaceAll(path, "\\", "/")
	cleaned := pathpkg.Clean(path)
	return strings.HasPrefix(path, "/") || cleaned == ".." || strings.HasPrefix(cleaned, "../")
}

func signingSensitivePath(normalizedPath string) bool {
	upper := strings.ToUpper(normalizedPath)
	if !strings.HasPrefix(upper, "META-INF/") {
		return false
	}
	return strings.HasSuffix(upper, ".RSA") || strings.HasSuffix(upper, ".DSA") || strings.HasSuffix(upper, ".EC") || strings.HasSuffix(upper, ".SF")
}

func entriesByNormalizedPath(entries []treehaver.ZipArchiveEntry) map[string]treehaver.ZipArchiveEntry {
	result := map[string]treehaver.ZipArchiveEntry{}
	for _, entry := range entries {
		result[entry.NormalizedPath] = entry
	}
	return result
}

func unionPaths(maps ...map[string]treehaver.ZipArchiveEntry) []string {
	seen := map[string]bool{}
	for _, entries := range maps {
		for path := range entries {
			seen[path] = true
		}
	}
	paths := make([]string, 0, len(seen))
	for path := range seen {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	return paths
}

func sameZipEntry(left treehaver.ZipArchiveEntry, right treehaver.ZipArchiveEntry) bool {
	return left.Path == right.Path &&
		left.Directory == right.Directory &&
		left.Compression == right.Compression &&
		left.CompressedSize == right.CompressedSize &&
		left.UncompressedSize == right.UncompressedSize &&
		left.CRC32 == right.CRC32
}

func nestedFamilyFor(normalizedPath string) string {
	lower := strings.ToLower(normalizedPath)
	switch {
	case strings.HasSuffix(lower, ".json"):
		return "json"
	case strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".markdown"):
		return "markdown"
	case strings.HasSuffix(lower, ".xml"):
		return "xml"
	case strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".yaml"):
		return "yaml"
	default:
		return ""
	}
}

func schemaPathFor(normalizedPath string) string {
	return "/entries/by_path/" + normalizedPath
}

func unsafeDiagnostic(unsafe treehaver.ZipUnsafeEntry, entry treehaver.ZipArchiveEntry) treehaver.BinaryDiagnostic {
	diagnostic := treehaver.BinaryDiagnostic{
		Severity:   "error",
		Category:   unsafe.Category,
		Message:    unsafe.Reason,
		SchemaPath: schemaPathFor(unsafe.NormalizedPath),
	}
	if entry.LocalHeaderRange.Valid() {
		byteRange := entry.LocalHeaderRange
		diagnostic.ByteRange = &byteRange
	}
	return diagnostic
}

func validateRawPreserveEntry(source []byte, centralDirectory centralDirectoryScan, entry treehaver.ZipArchiveEntry) error {
	if centralDirectory.ArchiveComment {
		return renderError("archive_comment", "/archive/comment", "raw-preserving ZIP renderer does not yet preserve archive comments")
	}
	if centralDirectory.MultiDisk {
		return renderError("split_archive", "/archive/disk", "raw-preserving ZIP renderer does not support split or spanned archives")
	}
	central, ok := centralDirectory.Records[entry.Path]
	if !ok {
		return fmt.Errorf("missing central directory record for preserved ZIP member %s", entry.NormalizedPath)
	}
	if central.GeneralPurposeFlag&0x1 != 0 {
		return renderError("encrypted_member", schemaPathFor(entry.NormalizedPath), fmt.Sprintf("raw-preserving ZIP renderer rejects encrypted member %s", entry.NormalizedPath))
	}
	if unsupportedFlags := central.GeneralPurposeFlag &^ 0x0808; unsupportedFlags != 0 {
		return renderError("unsupported_general_purpose_flags", schemaPathFor(entry.NormalizedPath), fmt.Sprintf("raw-preserving ZIP renderer rejects unsupported general-purpose flags 0x%04x for %s", unsupportedFlags, entry.NormalizedPath))
	}
	if methodForCompression(entry.Compression) == 0xffff || central.CompressionMethod != methodForCompression(entry.Compression) {
		return renderError("unsupported_compression", schemaPathFor(entry.NormalizedPath), fmt.Sprintf("raw-preserving ZIP renderer rejects unsupported compression %q for %s", entry.Compression, entry.NormalizedPath))
	}
	if central.ExtraLength != 0 {
		return renderError("central_directory_extra_field", schemaPathFor(entry.NormalizedPath), fmt.Sprintf("raw-preserving ZIP renderer does not yet preserve central-directory extra fields for %s", entry.NormalizedPath))
	}
	if central.CommentLength != 0 {
		return renderError("member_comment", schemaPathFor(entry.NormalizedPath), fmt.Sprintf("raw-preserving ZIP renderer does not yet preserve member comments for %s", entry.NormalizedPath))
	}
	if central.CompressedSize == int(^uint32(0)) || central.UncompressedSize == int(^uint32(0)) || central.LocalHeaderOffset == int(^uint32(0)) {
		return renderError("zip64_metadata", schemaPathFor(entry.NormalizedPath), fmt.Sprintf("raw-preserving ZIP renderer does not yet support Zip64 metadata for %s", entry.NormalizedPath))
	}
	if entry.LocalHeaderRange.StartByte+30 > len(source) {
		return fmt.Errorf("invalid local header for preserved ZIP member %s", entry.NormalizedPath)
	}
	localExtraLength := int(binary.LittleEndian.Uint16(source[entry.LocalHeaderRange.StartByte+28:]))
	if localExtraLength != 0 {
		return renderError("local_header_extra_field", schemaPathFor(entry.NormalizedPath), fmt.Sprintf("raw-preserving ZIP renderer does not yet preserve local extra fields for %s", entry.NormalizedPath))
	}
	return nil
}

func renderError(category string, schemaPath string, message string) RenderError {
	return RenderError{
		Diagnostic: treehaver.BinaryDiagnostic{
			Severity:   "error",
			Category:   category,
			Message:    message,
			SchemaPath: schemaPath,
		},
	}
}

func rawLocalRecordRanges(source []byte, entries map[string]treehaver.ZipArchiveEntry) map[string]treehaver.ByteRange {
	ordered := make([]treehaver.ZipArchiveEntry, 0, len(entries))
	for _, entry := range entries {
		ordered = append(ordered, entry)
	}
	slices.SortFunc(ordered, func(left, right treehaver.ZipArchiveEntry) int {
		if left.LocalHeaderRange.StartByte < right.LocalHeaderRange.StartByte {
			return -1
		}
		if left.LocalHeaderRange.StartByte > right.LocalHeaderRange.StartByte {
			return 1
		}
		return 0
	})
	result := map[string]treehaver.ByteRange{}
	for index, entry := range ordered {
		end := entry.CentralDirectoryRange.StartByte
		if index+1 < len(ordered) {
			end = ordered[index+1].LocalHeaderRange.StartByte
		}
		if end < entry.LocalHeaderRange.StartByte || end > len(source) {
			continue
		}
		result[entry.NormalizedPath] = treehaver.ByteRange{StartByte: entry.LocalHeaderRange.StartByte, EndByte: end}
	}
	return result
}

func renderedLocalRecord(entry treehaver.ZipArchiveEntry, content []byte, method uint16, offset int) ([]byte, centralRecord, error) {
	payload := content
	if method == zip.Deflate {
		var compressed bytes.Buffer
		writer, err := flate.NewWriter(&compressed, flate.DefaultCompression)
		if err != nil {
			return nil, centralRecord{}, err
		}
		if _, err := writer.Write(content); err != nil {
			return nil, centralRecord{}, err
		}
		if err := writer.Close(); err != nil {
			return nil, centralRecord{}, err
		}
		payload = compressed.Bytes()
	}

	crc := crc32.ChecksumIEEE(content)
	var record bytes.Buffer
	writeUint32(&record, localFileHeaderSignature)
	writeUint16(&record, 20)
	writeUint16(&record, 0)
	writeUint16(&record, method)
	record.Write(dosEpoch)
	writeUint32(&record, crc)
	writeUint32(&record, uint32(len(payload)))
	writeUint32(&record, uint32(len(content)))
	writeUint16(&record, uint16(len(entry.Path)))
	writeUint16(&record, 0)
	record.WriteString(entry.Path)
	record.Write(payload)

	return record.Bytes(), centralRecord{
		Name:              entry.Path,
		Method:            method,
		CRC32:             crc,
		CompressedSize:    uint32(len(payload)),
		UncompressedSize:  uint32(len(content)),
		LocalHeaderOffset: uint32(offset),
	}, nil
}

func centralRecordFromRawEntry(source []byte, entry treehaver.ZipArchiveEntry, offset int) (centralRecord, error) {
	if entry.LocalHeaderRange.StartByte+10 > len(source) {
		return centralRecord{}, fmt.Errorf("invalid local header for preserved ZIP member %s", entry.NormalizedPath)
	}
	method := binary.LittleEndian.Uint16(source[entry.LocalHeaderRange.StartByte+8:])
	if methodForCompression(entry.Compression) == 0xffff {
		return centralRecord{}, fmt.Errorf("unsupported preserved ZIP compression %q for %s", entry.Compression, entry.NormalizedPath)
	}
	return centralRecord{
		Name:               entry.Path,
		Method:             method,
		CRC32:              parseCRC32(entry.CRC32),
		CompressedSize:     uint32(entry.CompressedSize),
		UncompressedSize:   uint32(entry.UncompressedSize),
		LocalHeaderOffset:  uint32(offset),
		GeneralPurposeFlag: binary.LittleEndian.Uint16(source[entry.LocalHeaderRange.StartByte+6:]),
	}, nil
}

func writeCentralDirectoryRecord(buffer *bytes.Buffer, record centralRecord) {
	writeUint32(buffer, centralDirectorySignature)
	writeUint16(buffer, 20)
	writeUint16(buffer, 20)
	writeUint16(buffer, record.GeneralPurposeFlag)
	writeUint16(buffer, record.Method)
	buffer.Write(dosEpoch)
	writeUint32(buffer, record.CRC32)
	writeUint32(buffer, record.CompressedSize)
	writeUint32(buffer, record.UncompressedSize)
	writeUint16(buffer, uint16(len(record.Name)))
	writeUint16(buffer, 0)
	writeUint16(buffer, 0)
	writeUint16(buffer, 0)
	writeUint16(buffer, 0)
	writeUint32(buffer, record.ExternalAttrs)
	writeUint32(buffer, record.LocalHeaderOffset)
	buffer.WriteString(record.Name)
}

func writeEndOfCentralDirectory(buffer *bytes.Buffer, entries int, centralSize int, centralOffset int) {
	writeUint32(buffer, endCentralDirectorySignature)
	writeUint16(buffer, 0)
	writeUint16(buffer, 0)
	writeUint16(buffer, uint16(entries))
	writeUint16(buffer, uint16(entries))
	writeUint32(buffer, uint32(centralSize))
	writeUint32(buffer, uint32(centralOffset))
	writeUint16(buffer, 0)
}

func writeUint16(buffer *bytes.Buffer, value uint16) {
	var bytes [2]byte
	binary.LittleEndian.PutUint16(bytes[:], value)
	buffer.Write(bytes[:])
}

func writeUint32(buffer *bytes.Buffer, value uint32) {
	var bytes [4]byte
	binary.LittleEndian.PutUint32(bytes[:], value)
	buffer.Write(bytes[:])
}

func methodForCompression(compression string) uint16 {
	switch compression {
	case "stored":
		return zip.Store
	case "deflate":
		return zip.Deflate
	default:
		return 0xffff
	}
}

func parseCRC32(value string) uint32 {
	var parsed uint32
	_, _ = fmt.Sscanf(value, "%08x", &parsed)
	return parsed
}

func writeZipMember(writer *zip.Writer, entry treehaver.ZipArchiveEntry, content []byte, compression uint16) error {
	method := compression
	if entry.Compression == "stored" {
		method = zip.Store
	}
	header := &zip.FileHeader{
		Name:   entry.Path,
		Method: method,
	}
	if entry.Directory && !strings.HasSuffix(header.Name, "/") {
		header.Name += "/"
	}
	fileWriter, err := writer.CreateHeader(header)
	if err != nil {
		return err
	}
	if entry.Directory {
		return nil
	}
	_, err = fileWriter.Write(content)
	return err
}

func appendUnique(values []string, additions ...string) []string {
	seen := map[string]bool{}
	for _, value := range values {
		seen[value] = true
	}
	for _, addition := range additions {
		if !seen[addition] {
			values = append(values, addition)
			seen[addition] = true
		}
	}
	return values
}

package binarymerge

import (
	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/treehaver"
)

type BinaryFeatureProfile struct {
	Family            string
	SupportedDialects []string
	SupportedPolicies []astmerge.PolicyReference
}

func BinaryFeatureProfileInfo() BinaryFeatureProfile {
	return BinaryFeatureProfile{
		Family:            "binary",
		SupportedDialects: []string{},
		SupportedPolicies: []astmerge.PolicyReference{},
	}
}

func RenderPolicy(schemaPath string, byteRange treehaver.ByteRange, operation string, disposition string, reason string) treehaver.BinaryRenderPolicy {
	return treehaver.BinaryRenderPolicy{
		SchemaPath:  schemaPath,
		ByteRange:   &byteRange,
		Operation:   operation,
		Disposition: disposition,
		Reason:      reason,
	}
}

func UnsafeDiagnostic(schemaPath string, byteRange treehaver.ByteRange, message string) treehaver.BinaryDiagnostic {
	return treehaver.BinaryDiagnostic{
		Severity:   "error",
		Category:   "unsafe_binary_mutation",
		Message:    message,
		SchemaPath: schemaPath,
		ByteRange:  &byteRange,
	}
}

func PreservationReport(format string, schema string, matchedSchemaPaths []string, preservedRanges []treehaver.ByteRange) treehaver.BinaryMergeReport {
	return treehaver.BinaryMergeReport{
		Format:             format,
		Schema:             schema,
		MatchedSchemaPaths: matchedSchemaPaths,
		PreservedRanges:    preservedRanges,
		RewrittenNodes:     []string{},
		ChecksumUpdates:    []string{},
		NestedDispatches:   []treehaver.BinaryNestedDispatch{},
		Diagnostics:        []treehaver.BinaryDiagnostic{},
	}
}

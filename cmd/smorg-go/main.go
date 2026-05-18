package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/astmergegit"
	"github.com/structuredmerge/structuredmerge-go/jsonmerge"
	"github.com/structuredmerge/structuredmerge-go/plainmerge"
)

const (
	exitSuccess            = 0
	exitUnresolvedConflict = 1
	exitUserError          = 2
	exitInternalError      = 3
)

type mergeDriverOptions struct {
	ancestor             string
	current              string
	other                string
	pathName             string
	output               string
	strict               bool
	fallback             string
	checkOnly            bool
	exitCode             bool
	reportPath           string
	profileID            string
	profileReport        bool
	requireProfileStatus string
}

type diffDriverOptions struct {
	pathName string
	oldPath  string
	newPath  string
}

type pathSettings struct {
	language             string
	conflictMarkerSize   int
	profileID            string
	requireProfileStatus string
}

type conflictDiffOptions struct {
	pathName string
	filePath string
	exitCode bool
}

type conflictRegion struct {
	startLine     int
	separatorLine int
	endLine       int
}

type mergeDriverResult struct {
	OK           bool
	Diagnostics  []astmerge.Diagnostic
	Output       *string
	Fallbacks    []mergeDriverFallback
	OwnedRegions []astmergegit.OwnedRegionReport
	RenderReport *astmergegit.Merge3RenderReport
}

type mergeDriverMachineReport struct {
	Command      string                          `json:"command"`
	PathName     string                          `json:"path_name"`
	OK           bool                            `json:"ok"`
	ExitCode     int                             `json:"exit_code"`
	Fallbacks    []mergeDriverFallback           `json:"fallbacks"`
	OwnedRegions []astmergegit.OwnedRegionReport `json:"owned_regions"`
	RenderReport *astmergegit.Merge3RenderReport `json:"render_report,omitempty"`
	Diagnostics  []astmerge.Diagnostic           `json:"diagnostics"`
	Profile      map[string]string               `json:"profile,omitempty"`
	Metadata     map[string]interface{}          `json:"metadata,omitempty"`
}

type mergeDriverFallback struct {
	Mode          string `json:"mode"`
	RequestedMode string `json:"requested_mode"`
	Reason        string `json:"reason"`
	Applied       bool   `json:"applied"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return exitUserError
	}

	switch args[0] {
	case "merge-driver":
		return runMergeDriver(args[1:], stdout, stderr)
	case "diff-driver":
		return runDiffDriver(args[1:], stdout, stderr)
	case "languages":
		return runLanguages(args[1:], stdout, stderr)
	case "conflicts":
		return runConflicts(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		printUsage(stdout)
		return exitSuccess
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		printUsage(stderr)
		return exitUserError
	}
}

func printUsage(out io.Writer) {
	fmt.Fprintln(out, "usage: smorg-go merge-driver [--path-name PATH] [--output PATH] [--report PATH] [--strict] [--fallback=none|line|local|full-file] %O %A %B [%P]")
	fmt.Fprintln(out, "       smorg-go merge-driver --ancestor %O --current %A --other %B --path-name %P")
	fmt.Fprintln(out, "       smorg-go diff-driver [--path-name PATH] OLD NEW")
	fmt.Fprintln(out, "       smorg-go diff-driver PATH OLD-FILE OLD-HEX OLD-MODE NEW-FILE NEW-HEX NEW-MODE [OLD-PREFIX NEW-PREFIX]")
	fmt.Fprintln(out, "       smorg-go conflicts diff [--path-name PATH] [--exit-code] FILE")
	fmt.Fprintln(out, "       smorg-go languages --gitattributes")
}

func runMergeDriver(args []string, stdout io.Writer, stderr io.Writer) int {
	options, ok := parseMergeDriverOptions(args, stderr)
	if !ok {
		return exitUserError
	}
	ancestorSource, err := os.ReadFile(options.ancestor)
	if err != nil {
		fmt.Fprintf(stderr, "read ancestor: %v\n", err)
		return exitUserError
	}
	_ = ancestorSource

	currentSource, err := os.ReadFile(options.current)
	if err != nil {
		fmt.Fprintf(stderr, "read current: %v\n", err)
		return exitUserError
	}
	otherSource, err := os.ReadFile(options.other)
	if err != nil {
		fmt.Fprintf(stderr, "read other: %v\n", err)
		return exitUserError
	}

	effectivePath := options.effectivePath()
	settings := loadPathSettings(effectivePath)
	if options.profileID == "" {
		options.profileID = settings.profileID
	}
	if options.requireProfileStatus == "" {
		options.requireProfileStatus = settings.requireProfileStatus
	}
	if exitCode := reportAndEnforceProfile(options.profileID, options.profileReport, options.requireProfileStatus, stdout, stderr); exitCode != exitSuccess {
		return exitCode
	}
	fallbackPolicy := options.fallback
	if options.strict {
		fallbackPolicy = "none"
	}
	result := mergeByPath(effectivePath, settings.language, settings.conflictMarkerSize, fallbackPolicy, string(ancestorSource), string(currentSource), string(otherSource))
	if !result.OK {
		printDiagnostics(stderr, result.Diagnostics)
		fallbacks := append([]mergeDriverFallback{}, result.Fallbacks...)
		if result.Output == nil && !options.strict && options.fallback != "none" {
			output := fullFileConflictOutput(settings.conflictMarkerSize, string(ancestorSource), string(currentSource), string(otherSource))
			result.Output = &output
			fallbacks = append(fallbacks, mergeDriverFallback{
				Mode:          "full_file",
				RequestedMode: options.fallback,
				Reason:        fallbackReason(result.Diagnostics),
				Applied:       true,
			})
		}
		if options.checkOnly {
			if reportExit := writeMergeDriverMachineReport(options.reportPath, effectivePath, false, exitUnresolvedConflict, fallbacks, result.OwnedRegions, result.RenderReport, result.Diagnostics, stderr); reportExit != exitSuccess {
				return reportExit
			}
			return exitUnresolvedConflict
		}
		if result.Output != nil {
			if exitCode := writeMergeOutput(options, *result.Output, stderr); exitCode != exitSuccess {
				return exitCode
			}
		}
		if reportExit := writeMergeDriverMachineReport(options.reportPath, effectivePath, false, exitUnresolvedConflict, fallbacks, result.OwnedRegions, result.RenderReport, result.Diagnostics, stderr); reportExit != exitSuccess {
			return reportExit
		}
		return exitUnresolvedConflict
	}
	if result.Output == nil {
		fmt.Fprintln(stderr, "merge completed without output")
		return exitInternalError
	}

	if options.checkOnly {
		if options.exitCode && *result.Output != string(currentSource) {
			if reportExit := writeMergeDriverMachineReport(options.reportPath, effectivePath, true, exitUnresolvedConflict, result.Fallbacks, result.OwnedRegions, result.RenderReport, result.Diagnostics, stderr); reportExit != exitSuccess {
				return reportExit
			}
			return exitUnresolvedConflict
		}
		if reportExit := writeMergeDriverMachineReport(options.reportPath, effectivePath, true, exitSuccess, result.Fallbacks, result.OwnedRegions, result.RenderReport, result.Diagnostics, stderr); reportExit != exitSuccess {
			return reportExit
		}
		return exitSuccess
	}

	if exitCode := writeMergeOutput(options, *result.Output, stderr); exitCode != exitSuccess {
		return exitCode
	}

	if reportExit := writeMergeDriverMachineReport(options.reportPath, effectivePath, true, exitSuccess, result.Fallbacks, result.OwnedRegions, result.RenderReport, result.Diagnostics, stderr); reportExit != exitSuccess {
		return reportExit
	}
	return exitSuccess
}

func writeMergeDriverMachineReport(reportPath string, pathName string, ok bool, exitCode int, fallbacks []mergeDriverFallback, ownedRegions []astmergegit.OwnedRegionReport, renderReport *astmergegit.Merge3RenderReport, diagnostics []astmerge.Diagnostic, stderr io.Writer) int {
	if reportPath == "" {
		return exitSuccess
	}
	if fallbacks == nil {
		fallbacks = []mergeDriverFallback{}
	}
	if ownedRegions == nil {
		ownedRegions = []astmergegit.OwnedRegionReport{}
	}
	report := mergeDriverMachineReport{
		Command:      "merge-driver",
		PathName:     pathName,
		OK:           ok,
		ExitCode:     exitCode,
		Fallbacks:    fallbacks,
		OwnedRegions: ownedRegions,
		RenderReport: renderReport,
		Diagnostics:  diagnostics,
	}
	source, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "write report: %v\n", err)
		return exitInternalError
	}
	source = append(source, '\n')
	if err := os.WriteFile(reportPath, source, 0o644); err != nil {
		fmt.Fprintf(stderr, "write report: %v\n", err)
		return exitInternalError
	}
	return exitSuccess
}

func fallbackReason(diagnostics []astmerge.Diagnostic) string {
	if len(diagnostics) == 0 {
		return "structured_merge_failed"
	}
	return string(diagnostics[0].Category)
}

func fullFileConflictOutput(markerSize int, ancestorSource string, currentSource string, otherSource string) string {
	if markerSize <= 0 {
		markerSize = 7
	}
	return strings.Join([]string{
		strings.Repeat("<", markerSize) + " ours",
		currentSource,
		strings.Repeat("|", markerSize) + " base",
		ancestorSource,
		strings.Repeat("=", markerSize),
		otherSource,
		strings.Repeat(">", markerSize) + " theirs",
		"",
	}, "\n")
}

func writeMergeOutput(options mergeDriverOptions, output string, stderr io.Writer) int {
	outputPath := options.output
	if outputPath == "" {
		outputPath = options.current
	}
	if err := os.WriteFile(outputPath, []byte(output), 0o644); err != nil {
		fmt.Fprintf(stderr, "write output: %v\n", err)
		return exitInternalError
	}
	return exitSuccess
}

func parseMergeDriverOptions(args []string, stderr io.Writer) (mergeDriverOptions, bool) {
	options := mergeDriverOptions{fallback: "full-file"}
	flags := flag.NewFlagSet("merge-driver", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&options.ancestor, "ancestor", "", "ancestor file")
	flags.StringVar(&options.current, "current", "", "current file")
	flags.StringVar(&options.other, "other", "", "other file")
	flags.StringVar(&options.pathName, "path-name", "", "original git path name")
	flags.StringVar(&options.output, "output", "", "output path")
	flags.BoolVar(&options.strict, "strict", false, "disable conservative fallback")
	flags.StringVar(&options.fallback, "fallback", "full-file", "fallback mode: none, line, local, full-file")
	flags.BoolVar(&options.checkOnly, "check-only", false, "validate merge without writing")
	flags.BoolVar(&options.exitCode, "exit-code", false, "use exit code to report result")
	flags.StringVar(&options.reportPath, "report", "", "write machine-readable merge report to path")
	flags.StringVar(&options.profileID, "profile", "", "select merge profile")
	flags.BoolVar(&options.profileReport, "profile-report", false, "write selected profile promotion report to stdout")
	flags.StringVar(&options.requireProfileStatus, "require-profile-status", "", "require profile status: available, recommended, default")
	if err := flags.Parse(args); err != nil {
		return options, false
	}

	positionals := flags.Args()
	if options.ancestor == "" && len(positionals) > 0 {
		options.ancestor = positionals[0]
	}
	if options.current == "" && len(positionals) > 1 {
		options.current = positionals[1]
	}
	if options.other == "" && len(positionals) > 2 {
		options.other = positionals[2]
	}
	if options.pathName == "" && len(positionals) > 3 {
		options.pathName = positionals[3]
	}

	if options.ancestor == "" || options.current == "" || options.other == "" {
		fmt.Fprintln(stderr, "merge-driver requires ancestor, current, and other paths")
		return options, false
	}
	switch options.fallback {
	case "none", "line", "local", "full-file":
	default:
		fmt.Fprintf(stderr, "unsupported fallback mode %q\n", options.fallback)
		return options, false
	}

	return options, true
}

func reportAndEnforceProfile(profileID string, profileReport bool, requireStatus string, stdout io.Writer, stderr io.Writer) int {
	if profileID == "" && requireStatus == "" && !profileReport {
		return exitSuccess
	}
	if profileID == "" {
		profileID = astmerge.PromotionProfileJSONKeyedObject
	}
	evaluation := astmerge.ProfilePromotionEvaluation{
		ProfileID:       profileID,
		Status:          astmerge.ProfilePromotionAvailable,
		BlockingReasons: []string{"profile promotion evidence is not loaded by this CLI command"},
		Diagnostics:     []string{},
	}
	minimumStatus := astmerge.ProfilePromotionAvailable
	if requireStatus != "" {
		minimumStatus = astmerge.ProfilePromotionStatus(requireStatus)
	}
	requirement := astmerge.ProfileSelectionRequirement{
		ProfileID:            profileID,
		PromotionPolicyID:    astmerge.InitialProfilePromotionPolicy().PolicyID,
		MinimumProfileStatus: minimumStatus,
		EnforcementMode:      astmerge.ProfileSelectionAdvisory,
	}
	if requireStatus != "" {
		requirement.EnforcementMode = astmerge.ProfileSelectionRequired
	}
	decision := astmerge.EvaluateProfileSelectionRequirement(requirement, nil, evaluation)
	if profileReport {
		_ = json.NewEncoder(stdout).Encode(decision)
	}
	if !decision.Allowed {
		fmt.Fprintln(stderr, decision.BlockingReasons[0])
		return exitUserError
	}
	return exitSuccess
}

func runDiffDriver(args []string, stdout io.Writer, stderr io.Writer) int {
	options, ok := parseDiffDriverOptions(args, stderr)
	if !ok {
		return exitUserError
	}

	oldSource, err := os.ReadFile(options.oldPath)
	if err != nil {
		fmt.Fprintf(stderr, "read old file: %v\n", err)
		return exitUserError
	}
	newSource, err := os.ReadFile(options.newPath)
	if err != nil {
		fmt.Fprintf(stderr, "read new file: %v\n", err)
		return exitUserError
	}

	printStructuredDiff(stdout, options.effectivePath(), string(oldSource), string(newSource))
	return exitSuccess
}

func parseDiffDriverOptions(args []string, stderr io.Writer) (diffDriverOptions, bool) {
	var options diffDriverOptions
	flags := flag.NewFlagSet("diff-driver", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&options.pathName, "path-name", "", "original git path name")
	if err := flags.Parse(args); err != nil {
		return options, false
	}

	positionals := flags.Args()
	switch len(positionals) {
	case 2:
		options.oldPath = positionals[0]
		options.newPath = positionals[1]
	case 7, 9:
		options.pathName = firstNonEmpty(options.pathName, positionals[0])
		options.oldPath = positionals[1]
		options.newPath = positionals[4]
	default:
		fmt.Fprintln(stderr, "diff-driver requires either 2, 7, or 9 positional arguments")
		return options, false
	}

	return options, true
}

func (options diffDriverOptions) effectivePath() string {
	if options.pathName != "" {
		return options.pathName
	}
	if options.newPath != "" {
		return options.newPath
	}
	return options.oldPath
}

func printStructuredDiff(stdout io.Writer, pathName string, oldSource string, newSource string) {
	fmt.Fprintf(stdout, "structured-diff %s\n", pathName)
	if oldSource == newSource {
		fmt.Fprintln(stdout, "status unchanged")
		return
	}

	oldLines := strings.Count(oldSource, "\n")
	if oldSource != "" && !strings.HasSuffix(oldSource, "\n") {
		oldLines++
	}
	newLines := strings.Count(newSource, "\n")
	if newSource != "" && !strings.HasSuffix(newSource, "\n") {
		newLines++
	}
	fmt.Fprintf(stdout, "status changed\n")
	fmt.Fprintf(stdout, "old-lines %d\n", oldLines)
	fmt.Fprintf(stdout, "new-lines %d\n", newLines)
}

func runLanguages(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("languages", flag.ContinueOnError)
	flags.SetOutput(stderr)
	gitattributes := flags.Bool("gitattributes", false, "print recommended gitattributes")
	if err := flags.Parse(args); err != nil {
		return exitUserError
	}
	if !*gitattributes {
		fmt.Fprintln(stderr, "languages currently requires --gitattributes")
		return exitUserError
	}
	if len(flags.Args()) != 0 {
		fmt.Fprintln(stderr, "languages does not accept positional arguments")
		return exitUserError
	}

	for _, line := range []string{
		"*.go merge=smorg-go diff=smorg-go smorg.language=go",
		"*.json merge=smorg-go diff=smorg-go smorg.language=json",
		"*.jsonc merge=smorg-go diff=smorg-go smorg.language=jsonc",
	} {
		fmt.Fprintln(stdout, line)
	}
	return exitSuccess
}

func runConflicts(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "conflicts requires a subcommand")
		return exitUserError
	}
	switch args[0] {
	case "diff":
		return runConflictsDiff(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown conflicts subcommand %q\n", args[0])
		return exitUserError
	}
}

func runConflictsDiff(args []string, stdout io.Writer, stderr io.Writer) int {
	options, ok := parseConflictsDiffOptions(args, stderr)
	if !ok {
		return exitUserError
	}

	source, err := os.ReadFile(options.filePath)
	if err != nil {
		fmt.Fprintf(stderr, "read conflicted file: %v\n", err)
		return exitUserError
	}

	effectivePath := firstNonEmpty(options.pathName, options.filePath)
	settings := loadPathSettings(effectivePath)
	regions := findConflictRegions(string(source), settings.conflictMarkerSize)
	printConflictDiff(stdout, effectivePath, regions)
	if options.exitCode && len(regions) > 0 {
		return exitUnresolvedConflict
	}
	return exitSuccess
}

func parseConflictsDiffOptions(args []string, stderr io.Writer) (conflictDiffOptions, bool) {
	var options conflictDiffOptions
	flags := flag.NewFlagSet("conflicts diff", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&options.pathName, "path-name", "", "original git path name")
	flags.BoolVar(&options.exitCode, "exit-code", false, "return unresolved-conflict exit code when conflicts are present")
	if err := flags.Parse(args); err != nil {
		return options, false
	}
	positionals := flags.Args()
	if len(positionals) != 1 {
		fmt.Fprintln(stderr, "conflicts diff requires exactly one file path")
		return options, false
	}
	options.filePath = positionals[0]
	return options, true
}

func findConflictRegions(source string, markerSize int) []conflictRegion {
	if markerSize <= 0 {
		markerSize = 7
	}
	startPrefix := strings.Repeat("<", markerSize)
	separatorPrefix := strings.Repeat("=", markerSize)
	endPrefix := strings.Repeat(">", markerSize)

	var regions []conflictRegion
	var current *conflictRegion
	for index, line := range strings.Split(source, "\n") {
		lineNumber := index + 1
		switch {
		case strings.HasPrefix(line, startPrefix):
			current = &conflictRegion{startLine: lineNumber}
		case current != nil && current.separatorLine == 0 && strings.HasPrefix(line, separatorPrefix):
			current.separatorLine = lineNumber
		case current != nil && strings.HasPrefix(line, endPrefix):
			current.endLine = lineNumber
			regions = append(regions, *current)
			current = nil
		}
	}
	return regions
}

func printConflictDiff(stdout io.Writer, pathName string, regions []conflictRegion) {
	fmt.Fprintf(stdout, "conflicts %s\n", pathName)
	fmt.Fprintf(stdout, "count %d\n", len(regions))
	for index, region := range regions {
		fmt.Fprintf(stdout, "conflict %d lines %d-%d separator %d\n", index+1, region.startLine, region.endLine, region.separatorLine)
	}
}

func (options mergeDriverOptions) effectivePath() string {
	if options.pathName != "" {
		return options.pathName
	}
	return options.current
}

func mergeByPath(pathName string, language string, conflictMarkerSize int, fallbackPolicy string, ancestorSource string, currentSource string, otherSource string) mergeDriverResult {
	switch normalizeLanguage(language, pathName) {
	case "go":
		return merge3Result(astmergegit.Merge3(astmergegit.Merge3Request{
			BaseSource:         ancestorSource,
			OursSource:         currentSource,
			TheirsSource:       otherSource,
			PathName:           pathName,
			Language:           "go",
			Dialect:            "go",
			ProfileID:          "go.source",
			FallbackPolicy:     fallbackPolicy,
			ConflictMarkerSize: conflictMarkerSize,
			RenderPolicy:       "canonical",
		}))
	case "json":
		return merge3Result(astmergegit.Merge3(astmergegit.Merge3Request{
			BaseSource:         ancestorSource,
			OursSource:         currentSource,
			TheirsSource:       otherSource,
			PathName:           pathName,
			Language:           "json",
			Dialect:            "json",
			ProfileID:          "json.keyed-object",
			FallbackPolicy:     fallbackPolicy,
			ConflictMarkerSize: conflictMarkerSize,
			RenderPolicy:       "canonical",
		}))
	case "jsonc":
		return mergeResult(jsonmerge.MergeJSON(otherSource, currentSource, jsonmerge.DialectJSONC))
	default:
		return mergeResult(plainmerge.MergeText(otherSource, currentSource))
	}
}

func mergeResult(result astmerge.MergeResult[string]) mergeDriverResult {
	return mergeDriverResult{
		OK:          result.OK,
		Diagnostics: result.Diagnostics,
		Output:      result.Output,
	}
}

func merge3Result(result astmergegit.Merge3Response) mergeDriverResult {
	fallbacks := merge3Fallbacks(result.Fallbacks)
	if result.OK && result.MergedSource != nil {
		return mergeDriverResult{
			OK:           true,
			Diagnostics:  result.Diagnostics,
			Output:       result.MergedSource,
			Fallbacks:    fallbacks,
			OwnedRegions: result.OwnedRegions,
			RenderReport: &result.RenderReport,
		}
	}
	if !result.OK && result.ConflictedSource != nil {
		return mergeDriverResult{
			OK:           false,
			Diagnostics:  result.Diagnostics,
			Output:       result.ConflictedSource,
			Fallbacks:    fallbacks,
			OwnedRegions: result.OwnedRegions,
			RenderReport: &result.RenderReport,
		}
	}
	return mergeDriverResult{
		OK:           false,
		Diagnostics:  result.Diagnostics,
		Fallbacks:    fallbacks,
		OwnedRegions: result.OwnedRegions,
		RenderReport: &result.RenderReport,
	}
}

func merge3Fallbacks(fallbacks []string) []mergeDriverFallback {
	records := make([]mergeDriverFallback, 0, len(fallbacks))
	for _, fallback := range fallbacks {
		records = append(records, mergeDriverFallback{
			Mode:          fallback,
			RequestedMode: fallback,
			Reason:        "ast_merge_git_fallback",
			Applied:       true,
		})
	}
	return records
}

func normalizeLanguage(language string, pathName string) string {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "go", "golang":
		return "go"
	case "json":
		return "json"
	case "jsonc", "json with comments":
		return "jsonc"
	case "plain", "text", "plaintext", "text/plain":
		return "text"
	}

	switch strings.ToLower(filepath.Ext(pathName)) {
	case ".go":
		return "go"
	case ".json":
		return "json"
	case ".jsonc":
		return "jsonc"
	default:
		return "text"
	}
}

func loadPathSettings(pathName string) pathSettings {
	settings := pathSettings{conflictMarkerSize: 7}
	for _, attributesPath := range attributeFilesForPath(pathName) {
		source, err := os.ReadFile(attributesPath)
		if err != nil {
			continue
		}
		applyAttributes(&settings, pathName, string(source))
	}
	return settings
}

func attributeFilesForPath(pathName string) []string {
	cleanPath := filepath.Clean(filepath.FromSlash(pathName))
	dir := filepath.Dir(cleanPath)
	if dir == "." || strings.HasPrefix(dir, "..") || filepath.IsAbs(dir) {
		return []string{".gitattributes"}
	}

	parts := strings.Split(dir, string(filepath.Separator))
	files := []string{".gitattributes"}
	for index := range parts {
		if parts[index] == "" || parts[index] == "." {
			continue
		}
		files = append(files, filepath.Join(filepath.Join(parts[:index+1]...), ".gitattributes"))
	}
	return files
}

func applyAttributes(settings *pathSettings, pathName string, source string) {
	for _, rawLine := range strings.Split(source, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 || !attributePatternMatches(fields[0], pathName) {
			continue
		}
		for _, field := range fields[1:] {
			key, value, ok := strings.Cut(field, "=")
			if !ok {
				continue
			}
			switch key {
			case "smorg.language", "linguist-language":
				settings.language = value
			case "smorg.profile":
				settings.profileID = value
			case "smorg.requireProfileStatus":
				settings.requireProfileStatus = value
			case "conflict-marker-size":
				if markerSize, err := strconv.Atoi(value); err == nil && markerSize > 0 {
					settings.conflictMarkerSize = markerSize
				}
			}
		}
	}
}

func attributePatternMatches(pattern string, pathName string) bool {
	cleanPath := filepath.Clean(filepath.FromSlash(pathName))
	if pattern == cleanPath {
		return true
	}
	if !strings.Contains(pattern, "/") {
		matched, err := filepath.Match(pattern, filepath.Base(cleanPath))
		return err == nil && matched
	}
	matched, err := filepath.Match(filepath.Clean(filepath.FromSlash(pattern)), cleanPath)
	return err == nil && matched
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func printDiagnostics(stderr io.Writer, diagnostics []astmerge.Diagnostic) {
	for _, diagnostic := range diagnostics {
		fmt.Fprintf(stderr, "%s: %s\n", diagnostic.Category, diagnostic.Message)
	}
}

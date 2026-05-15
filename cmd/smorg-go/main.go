package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/gomerge"
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
	ancestor  string
	current   string
	other     string
	pathName  string
	output    string
	strict    bool
	fallback  string
	checkOnly bool
	exitCode  bool
}

type diffDriverOptions struct {
	pathName string
	oldPath  string
	newPath  string
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
	fmt.Fprintln(out, "usage: smorg-go merge-driver [--path-name PATH] [--output PATH] [--strict] [--fallback=none|line|local|full-file] %O %A %B [%P]")
	fmt.Fprintln(out, "       smorg-go merge-driver --ancestor %O --current %A --other %B --path-name %P")
	fmt.Fprintln(out, "       smorg-go diff-driver [--path-name PATH] OLD NEW")
	fmt.Fprintln(out, "       smorg-go diff-driver PATH OLD-FILE OLD-HEX OLD-MODE NEW-FILE NEW-HEX NEW-MODE [OLD-PREFIX NEW-PREFIX]")
	fmt.Fprintln(out, "       smorg-go languages --gitattributes")
}

func runMergeDriver(args []string, _ io.Writer, stderr io.Writer) int {
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

	result := mergeByPath(options.effectivePath(), string(otherSource), string(currentSource))
	if !result.OK || result.Output == nil {
		if options.strict || options.fallback == "none" {
			printDiagnostics(stderr, result.Diagnostics)
			return exitUnresolvedConflict
		}
		output := string(currentSource)
		result.Output = &output
	}

	if options.checkOnly {
		return exitSuccess
	}

	outputPath := options.output
	if outputPath == "" {
		outputPath = options.current
	}
	if err := os.WriteFile(outputPath, []byte(*result.Output), 0o644); err != nil {
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

func (options mergeDriverOptions) effectivePath() string {
	if options.pathName != "" {
		return options.pathName
	}
	return options.current
}

func mergeByPath(pathName string, otherSource string, currentSource string) astmerge.MergeResult[string] {
	switch strings.ToLower(filepath.Ext(pathName)) {
	case ".go":
		return gomerge.MergeGo(otherSource, currentSource, gomerge.DialectGo)
	case ".json":
		return jsonmerge.MergeJSON(otherSource, currentSource, jsonmerge.DialectJSON)
	case ".jsonc":
		return jsonmerge.MergeJSON(otherSource, currentSource, jsonmerge.DialectJSONC)
	default:
		return plainmerge.MergeText(otherSource, currentSource)
	}
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

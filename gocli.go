package gocli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/leep-frog/command"
	"github.com/leep-frog/command/sourcerer"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

func CLI() sourcerer.CLI {
	return &goCLI{}
}

type goCLI struct{}

func (gc *goCLI) Changed() bool   { return false }
func (gc *goCLI) Setup() []string { return nil }
func (gc *goCLI) Name() string    { return "gt" }

var (
	coverageRegex = regexp.MustCompile(`^ok\s+([^\s]+)\s.*coverage: +([0-9]+\.[0-9]+)% of statements$`)
	noTestRegex   = regexp.MustCompile("^\\?.*\\[no test files\\]\n?$")

	findTestRegex = regexp.MustCompile(`^func\s+Test([a-zA-Z0-9_]*)\b.*\*testing\.[A-Z]\b`)
	testFileRegex = regexp.MustCompile(`.*_test.go$`)

	// Args and flags
	pathArgs        = command.ListArg[string]("PATH", "Path(s) to go packages to test", 0, command.UnboundedList, &command.FileCompleter[[]string]{Distinct: true, IgnoreFiles: true}, command.Default([]string{"."}))
	verboseFlag     = command.BoolFlag("verbose", 'v', "Whether or not to test with verbose output")
	minCoverageFlag = command.Flag[float64]("minCoverage", 'm', "If set, enforces that minimum coverage is met", command.Positive[float64](), command.LTE[float64](100), command.Default[float64](0))
	timeoutFlag     = command.Flag[int]("timeout", 't', "Test timeout in seconds", command.Positive[int]())

	funcFilterFlag = command.ListFlag[string]("func-filter", 'f', "The test function filter", 0, command.UnboundedList, command.DeferredCompleter(command.SerialNodes(pathArgs), command.CompleterFromFunc(func(sl []string, data *command.Data) (*command.Completion, error) {
		suggestions := map[string]bool{}
		for _, rootPath := range pathArgs.GetOrDefault(data, []string{"."}) {
			rootOnly := true
			if rootPath == "./..." {
				rootOnly = false
				rootPath = "."
			}

			if err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				if d.IsDir() {
					if rootOnly && d.Name() != rootPath {
						return filepath.SkipDir
					}
					return nil
				}

				if !testFileRegex.MatchString(path) {
					return nil
				}

				f, err := os.Open(path)
				if err != nil {
					return fmt.Errorf("failed to open test file: %v", err)
				}

				for scanner := bufio.NewScanner(f); scanner.Scan(); {
					m := findTestRegex.FindStringSubmatch(scanner.Text())
					if len(m) > 0 {
						suggestions[m[1]] = true
					}
				}

				return nil
			}); err != nil {
				return nil, err
			}

		}
		return &command.Completion{
			Suggestions:     maps.Keys(suggestions),
			Distinct:        true,
			CaseInsensitive: true,
		}, nil
	})))
)

func percentFormat(f float64) string {
	return fmt.Sprintf("%3.1f%%", f)
}

const (
	noTestFiles   = -1.0
	unsetCoverage = -2.0
)

type coverageInfo struct {
	line     string
	coverage float64
}

type goTestEventHandler struct {
	packageResults map[string]bool
	coverage       map[string]*coverageInfo
	err            error
}

func (eh *goTestEventHandler) setPackageResult(p, action string, res bool) error {
	if r, ok := eh.packageResults[p]; ok {
		return fmt.Errorf("Duplicate package results: %v, %s", r, action)
	}
	eh.packageResults[p] = res
	return nil
}

type goTestEvent struct {
	Time    string
	Action  string
	Package string
	Output  string
	// Specific to individual test cases
	Test    string
	Elapsed int
}

func (eh *goTestEventHandler) streamFuncWrapper(output command.Output, data *command.Data, line []byte) error {
	if eh.err != nil {
		return nil
	}

	if err := eh.streamFunc(output, data, line); err != nil {
		eh.err = err
	}
	return nil
}

func (eh *goTestEventHandler) streamFunc(output command.Output, data *command.Data, line []byte) error {
	e := &goTestEvent{}
	if err := json.Unmarshal(line, e); err != nil {
		return fmt.Errorf("failed to parse go event (%s): %v", line, err)
	}

	// Package event
	if e.Test == "" {
		switch e.Action {
		case "success", "failure":
			if err := eh.setPackageResult(e.Package, e.Action, e.Action == "success"); err != nil {
				return err
			}
		case "output":
			output.Stdout(e.Output)
			var setCoverage *coverageInfo
			if noTestRegex.MatchString(e.Output) {
				setCoverage = &coverageInfo{
					line:     e.Output,
					coverage: noTestFiles,
				}
			} else if m := coverageRegex.FindStringSubmatch(e.Output); len(m) > 0 {
				f, err := strconv.ParseFloat(m[2], 64)
				if err != nil {
					return fmt.Errorf("failed to parse coverage value: %v", err)
				}
				setCoverage = &coverageInfo{
					line:     e.Output,
					coverage: f,
				}
			}

			// Set the coverage
			if setCoverage != nil {
				// Check if it's already set
				if c, ok := eh.coverage[e.Package]; ok {
					return fmt.Errorf("Duplicate package coverage: %v, %v", c, setCoverage)
				}
				eh.coverage[e.Package] = setCoverage
			}
		case "skip":
		default:
			return fmt.Errorf("Unknown package event action: %q", e.Action)
		}
	} else {
		// Test event
		if verboseFlag.Get(data) && e.Action == "output" {
			output.Stdoutf(e.Output)
		}
	}
	return nil
}

func (gc *goCLI) Node() command.Node {
	return command.SerialNodes(
		command.FlagProcessor(
			minCoverageFlag,
			verboseFlag,
			timeoutFlag,
			funcFilterFlag,
		),
		pathArgs,
		&command.ExecutorProcessor{F: func(o command.Output, d *command.Data) error {
			// Error if verbose and coverage check
			mc := minCoverageFlag.Get(d)
			if d.Has(verboseFlag.Name()) && mc != 0 {
				return o.Stderrln("Can't run verbose output with coverage checks")
			}

			// Construct go test
			args := []string{
				"test",
			}
			if d.Has(timeoutFlag.Name()) {
				args = append(args, "-timeout", fmt.Sprintf("%ds", timeoutFlag.Get(d)))
			}
			args = append(args, pathArgs.Get(d)...)
			args = append(args, "-json")
			if verboseFlag.Get(d) {
				args = append(args, "-v")
			}
			if d.Has(funcFilterFlag.Name()) {
				parens := fmt.Sprintf("(%s)", strings.Join(funcFilterFlag.Get(d), "|"))
				args = append(args, "-run", parens)
			} else if sourcerer.CurrentOS.Name() == "linux" {
				args = append(args, "-coverprofile=$(mktemp)")
			} else {
				args = append(args, "-coverprofile=(New-TemporaryFile)")
			}

			// Run the command
			eh := &goTestEventHandler{
				packageResults: map[string]bool{},
				coverage:       map[string]*coverageInfo{},
			}
			sc := &command.ShellCommand[[]string]{
				CommandName:           "go",
				Args:                  args,
				OutputStreamProcessor: eh.streamFuncWrapper,
			}
			if _, err := sc.Run(o, d); err != nil {
				return o.Err(err)
			}
			if eh.err != nil {
				return o.Err(eh.err)
			}

			// Error to return
			packages := maps.Keys(eh.packageResults)
			slices.Sort(packages)
			var retErr error
			for _, p := range packages {
				coverage, ok := eh.coverage[p]
				if !ok {
					return o.Stderrf("No coverage set for package: %s", p)
				}

				if coverage.coverage == noTestFiles {
					continue
				}

				if coverage.coverage < mc {
					retErr = o.Stderrf("Coverage of package %q (%s) must be at least %s\n", p, percentFormat(coverage.coverage), percentFormat(mc))
				}
			}

			return retErr
		}},
	)
}

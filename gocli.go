package gocli

import (
	"bufio"
	"fmt"
	"io/fs"
	"io/ioutil"
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

var (
	tmpFile = func() (*os.File, error) {
		return ioutil.TempFile("", "leepGocli")
	}
)

type goCLI struct{}

func (gc *goCLI) Changed() bool   { return false }
func (gc *goCLI) Setup() []string { return nil }
func (gc *goCLI) Name() string    { return "gt" }

var (
	coverageRegex = regexp.MustCompile(`^ok\s+([^\s]+)\s+[0-9\.a-zA-Z]+\s+coverage:\s+([0-9]+\.[0-9]+)% of statements` + "\n?$")
	noTestRegex   = regexp.MustCompile(`^\?\s+([^\s]+)\s+\[no test files\]` + "\n?$")
	testFailRegex = regexp.MustCompile(`^FAIL\s+([^\s]+)\s+`)

	findTestRegex = regexp.MustCompile(`^func\s+Test([a-zA-Z0-9_]*)\b.*\*testing\.[A-Z]\b`)
	testFileRegex = regexp.MustCompile(`.*_test.go$`)

	// Args and flags
	pathArgs         = command.ListArg[string]("PATH", "Path(s) to go packages to test", 0, command.UnboundedList, &command.FileCompleter[[]string]{Distinct: true, IgnoreFiles: true}, command.Default([]string{"."}))
	verboseFlag      = command.BoolFlag("verbose", 'v', "Whether or not to test with verbose output")
	minCoverageFlag  = command.Flag[float64]("minCoverage", 'm', "If set, enforces that minimum coverage is met", command.Positive[float64](), command.LTE[float64](100), command.Default[float64](0))
	packageCountFlag = command.Flag[int]("package-count", 'p', "Number of packages to expect output for")
	timeoutFlag      = command.Flag[int]("timeout", 't', "Test timeout in seconds", command.Positive[int]())

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

type testResult int

const (
	noTestFiles testResult = iota
	testSuccess
	testFailure
)

type packageResult struct {
	TestResult testResult
	Coverage   float64
	Line       string
}

type goTestEventHandler struct {
	packageResults map[string]*packageResult
	err            error
}

func (eh *goTestEventHandler) setPackageResult(pkg, line string, tr testResult, coverage float64) error {
	if r, ok := eh.packageResults[pkg]; ok {
		return fmt.Errorf("Multiple results for package %q:\n  Result 1: %s\n  Result 2: %s", pkg, r.Line, line)
	}
	eh.packageResults[pkg] = &packageResult{
		TestResult: tr,
		Coverage:   coverage,
		Line:       line,
	}
	return nil
}

func (eh *goTestEventHandler) streamFunc(output command.Output, data *command.Data, bLine []byte) error {
	if eh.err != nil {
		return nil
	}

	for _, line := range strings.Split(string(bLine), "\n") {
		eh.err = eh.processLine(line)
		if eh.err != nil {
			break
		}
	}

	return nil
}

func (eh *goTestEventHandler) processLine(line string) error {
	if m := noTestRegex.FindStringSubmatch(line); m != nil {
		return eh.setPackageResult(m[1], line, noTestFiles, 0)
	}

	if m := coverageRegex.FindStringSubmatch(line); m != nil {
		coverage, err := strconv.ParseFloat(m[2], 64)
		if err != nil {
			return fmt.Errorf("failed to parse coverage value: %v", err)
		}
		return eh.setPackageResult(m[1], line, testSuccess, coverage)
	}

	if m := testFailRegex.FindStringSubmatch(line); m != nil {
		return eh.setPackageResult(m[1], line, testFailure, 0)
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
			packageCountFlag,
		),
		pathArgs,
		&command.ExecutorProcessor{F: func(o command.Output, d *command.Data) error {
			// Error if verbose and coverage check
			mc := minCoverageFlag.Get(d)

			// Construct go test
			args := []string{
				"test",
			}
			if d.Has(timeoutFlag.Name()) {
				args = append(args, "-timeout", fmt.Sprintf("%ds", timeoutFlag.Get(d)))
			}
			args = append(args, pathArgs.Get(d)...)
			if verboseFlag.Get(d) {
				args = append(args, "-v")
			}
			if d.Has(funcFilterFlag.Name()) {
				if mc > 0.0 {
					return o.Stderrln("Cannot set func-filter and min coverage flags simultaneously")
				}
				parens := fmt.Sprintf("(%s)", strings.Join(funcFilterFlag.Get(d), "|"))
				args = append(args, "-run", parens)
			} else {
				// TODO: Use tmpFile to compute coverage data instead of parsing somewhat arbitrary text (which is viable change (already happened once on me))
				tmp, err := tmpFile()
				if err != nil {
					return o.Annotatef(err, "failed to create temporary file")
				}
				args = append(args, fmt.Sprintf("-coverprofile=%s", tmp.Name()))
			}

			// Run the command
			eh := &goTestEventHandler{
				packageResults: map[string]*packageResult{},
			}
			sc := &command.ShellCommand[[]string]{
				CommandName:           "go",
				Args:                  args,
				OutputStreamProcessor: eh.streamFunc,
				ForwardStdout:         true,
			}
			if _, err := sc.Run(o, d); err != nil {
				return o.Annotatef(err, "go test shell command error")
			}
			if eh.err != nil {
				return o.Annotatef(eh.err, "event handling error")
			}

			// Error to return
			packages := maps.Keys(eh.packageResults)
			slices.Sort(packages)

			if packageCountFlag.Provided(d) {
				if expectedPackageCount := packageCountFlag.Get(d); expectedPackageCount != len(packages) {
					return o.Stderrf("Expected %d packages, got %d:\n%s\n", expectedPackageCount, len(packages), strings.Join(packages, "\n"))
				}
			}

			var retErr error
			for _, p := range packages {
				pr := eh.packageResults[p]
				switch pr.TestResult {
				case noTestFiles:
				case testFailure:
					retErr = o.Stderrf("Tests failed for package: %s\n", p)
				case testSuccess:
					if pr.Coverage < mc {
						retErr = o.Stderrf("Coverage of package %q (%s) must be at least %s\n", p, percentFormat(pr.Coverage), percentFormat(mc))
						continue
					}
				}
			}

			// Set data for use in tests
			if len(eh.packageResults) > 0 {
				d.Set("COVERAGE", eh.packageResults)
			}

			return retErr
		}},
	)
}

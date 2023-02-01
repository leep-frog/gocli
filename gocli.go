package gocli

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/leep-frog/command"
	"github.com/leep-frog/command/sourcerer"
	"golang.org/x/exp/maps"
)

func CLI() sourcerer.CLI {
	return &goCLI{}
}

type goCLI struct{}

func (gc *goCLI) Changed() bool   { return false }
func (gc *goCLI) Setup() []string { return nil }
func (gc *goCLI) Name() string    { return "gt" }

const (
	findTestFunctionCommand = `find %s %s -iname '*_test.go' | xargs cat | grep -E '^func\s+Test.*\*testing.T'`
	defaultMaxDepth         = "-maxDepth 1"
	funcFilterFlagName      = "func-filter"
)

var (
	coverageRegex = regexp.MustCompile(`^ok\s+([^\s]+)\s.*coverage: +([0-9]+\.[0-9]+)% of statements$`)
	noTestRegex   = regexp.MustCompile(`^\?.*\[no test files\]$`)

	findTestRegex = regexp.MustCompile(`^func\s+Test([a-zA-Z0-9_]*)\b`)

	// Args and flags
	pathArgs        = command.ListArg[string]("PATH", "Path(s) to go packages to test", 0, command.UnboundedList, &command.FileCompleter[[]string]{Distinct: true, IgnoreFiles: true}, command.Default([]string{"."}))
	verboseFlag     = command.BoolValueFlag("verbose", 'v', "Whether or not to test with verbose output", " -v")
	minCoverageFlag = command.Flag[float64]("minCoverage", 'm', "If set, enforces that minimum coverage is met", command.Positive[float64](), command.LTE[float64](100), command.Default[float64](0))
	timeoutFlag     = command.Flag[int]("timeout", 't', "Test timeout in seconds", command.Positive[int]())

	funcFilterFlag = command.ListFlag[string](funcFilterFlagName, 'f', "The test function filter", 0, command.UnboundedList, command.DeferredCompleter[[]string](command.SerialNodes(pathArgs), func(data *command.Data) (*command.Completion, error) {
		suggestions := map[string]bool{}
		for _, path := range pathArgs.GetOrDefault(data, []string{"."}) {
			maxDepth := defaultMaxDepth
			if path == "./..." {
				maxDepth = ""
			}
			cmd := fmt.Sprintf(findTestFunctionCommand, path, maxDepth)
			bc := &command.BashCommand[[]string]{
				Contents: []string{cmd},
			}
			lines, err := bc.Run(command.NewIgnoreAllOutput(), data)
			if err != nil {
				return nil, fmt.Errorf("%s : %v", cmd, err)
			}
			for _, line := range lines {
				m := findTestRegex.FindStringSubmatch(line)
				if len(m) == 0 {
					return nil, fmt.Errorf("Returned line did not match expected format: [%q]", line)
				}
				suggestions[m[1]] = true
			}
		}
		// Can't use funcFilterFlag.Get(data) because of cyclical dependency
		ffs := data.StringList(funcFilterFlagName)
		return command.RunArgumentCompletion(&command.Completion{
			Suggestions:     maps.Keys(suggestions),
			Distinct:        true,
			CaseInsensitive: true,
		}, ffs[len(ffs)-1], ffs, data)
	}))
)

func percentFormat(f float64) string {
	return fmt.Sprintf("%3.1f%%", f)
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

			var timeout string
			if d.Has(timeoutFlag.Name()) {
				timeout = fmt.Sprintf("-timeout %ds ", timeoutFlag.Get(d))
			}

			bc := &command.BashCommand[[]string]{
				Contents:      []string{fmt.Sprintf("go test %s%s%s -coverprofile=$(mktemp)", timeout, strings.Join(pathArgs.Get(d), " "), verboseFlag.Get(d))},
				ForwardStdout: true,
			}
			res, err := bc.Run(o, d)
			if err != nil {
				// Failed to build or test failed so just return
				return o.Err(err)
			}

			// Error to return
			var retErr error
			for _, coverage := range res {
				if coverage == "" {
					continue
				}

				if noTestRegex.MatchString(coverage) {
					continue
				}

				m := coverageRegex.FindStringSubmatch(coverage)
				if len(m) == 0 {
					if d.Has(verboseFlag.Name()) {
						continue
					}
					return o.Stderrf("failed to parse coverage from line %q\n", coverage)
				}

				f, err := strconv.ParseFloat(m[2], 64)
				if err != nil {
					return o.Annotate(err, "failed to parse coverage value")
				}

				if f < mc {
					retErr = o.Stderrf("Coverage of package %q (%s) must be at least %s\n", m[1], percentFormat(f), percentFormat(mc))
				}
			}
			return retErr
		}},
	)
}

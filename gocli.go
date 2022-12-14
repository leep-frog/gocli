package gocli

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/leep-frog/command"
	"github.com/leep-frog/command/sourcerer"
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
	noTestRegex   = regexp.MustCompile(`^\?.*\[no test files\]$`)

	// Args and flags
	pathArgs        = command.ListArg[string]("PATH", "Path(s) to go packages to test", 0, command.UnboundedList, &command.FileCompleter[[]string]{Distinct: true, IgnoreFiles: true}, command.Default([]string{"."}))
	verboseFlag     = command.BoolValueFlag("verbose", 'v', "Whether or not to test with verbose output", " -v")
	minCoverageFlag = command.Flag[float64]("minCoverage", 'm', "If set, enforces that minimum coverage is met", command.Positive[float64](), command.LTE[float64](100), command.Default[float64](0))
	timeoutFlag     = command.Flag[int]("timeout", 't', "Test timeout in seconds", command.Positive[int]())
)

func percentFormat(f float64) string {
	return fmt.Sprintf("%3.1f%%", f)
}

func (gc *goCLI) Node() *command.Node {
	return command.SerialNodes(
		command.FlagNode(
			minCoverageFlag,
			verboseFlag,
			timeoutFlag,
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

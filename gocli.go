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
	pathArgs      = command.ListArg[string]("PATH", "Path(s) to go packages to test", 0, command.UnboundedList, &command.FileCompletor[[]string]{Distinct: true, IgnoreFiles: true}, command.Default([]string{"."}))
	coverageRegex = regexp.MustCompile(`^ok\s+([^\s]+)\s.*coverage: +([0-9]+\.[0-9]+)% of statements$`)
	noTestRegex   = regexp.MustCompile(`^\?.*\[no test files\]$`)

	verboseFlag     = command.BoolFlag("verbose", 'v', "Whether or not to test with verbose output")
	minCoverageFlag = command.NewFlag[float64]("MIN_COVERAGE", 'm', "If set, enforces that minimum coverage is met", command.Positive[float64](), command.LTE[float64](100), command.Default[float64](0))
)

func percentFormat(f float64) string {
	return fmt.Sprintf("%3.1f%%", f)
}

func (gc *goCLI) Node() *command.Node {
	return command.BranchNode(map[string]*command.Node{},
		command.SerialNodes(
			command.NewFlagNode(
				minCoverageFlag,
				verboseFlag,
			),
			pathArgs,
			command.ExecuteErrNode(func(o command.Output, d *command.Data) error {
				// Error if verbose and coverage check
				mc := minCoverageFlag.Get(d)
				if verboseFlag.Get(d) && mc != 0 {
					return o.Stderr("Can't run verbose output with coverage checks")
				}

				res, err := command.NewBashCommand("", []string{fmt.Sprintf("go test %s -coverprofile=$(mktemp)", strings.Join(pathArgs.Get(d), " "))}, command.ForwardStdout[[]string]()).Run(o)
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
						return o.Stderrf("failed to parse coverage from line %q", coverage)
					}

					f, err := strconv.ParseFloat(m[2], 64)
					if err != nil {
						return o.Annotate(err, "failed to parse coverage value")
					}

					if f < mc {
						retErr = o.Stderrf("Coverage of package %q (%s) must be at least %s", m[1], percentFormat(f), percentFormat(mc))
					}
				}
				fmt.Println("returning error:", retErr)
				return retErr
			}),
		),
	)
}

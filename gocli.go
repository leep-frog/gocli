package gocli

import (
	"regexp"
	"strconv"

	"github.com/leep-frog/command"
	"github.com/leep-frog/command/sourcerer"
)

func CLI() sourcerer.CLI {
	return &gocli{}
}

type gocli struct{}

func (gc *gocli) Changed() bool   { return false }
func (gc *gocli) Setup() []string { return nil }
func (gc *gocli) Name() string    { return "gt" }

var (
	pathArgs      = command.ListArg[string]("PATH", "Path(s) to go packages to test", 0, command.UnboundedList, &command.FileCompletor[[]string]{Distinct: true, IgnoreFiles: true})
	coverageRegex = regexp.MustCompile(`coverage: +([0-9]+\.[0-9]+)% of statements$`)
	noTestRegex   = regexp.MustCompile(`^\?.*\[no test files\]$`)
)

func (gc *gocli) Node() *command.Node {
	return command.BranchNode(map[string]*command.Node{},
		command.SerialNodes(
			pathArgs,
			command.ExecuteErrNode(func(o command.Output, d *command.Data) error {
				res, err := command.NewBashCommand[[]string]("", []string{"go test . -coverprofile=$(mktemp)"}, command.ForwardStdout[[]string]()).Run(o)
				if err != nil {
					// Failed to build or test failed so just return
					return err
				}

				var m []string
				for covIdx := len(res) - 1; covIdx >= 0 && len(m) == 0; covIdx-- {
					m = coverageRegex.FindStringSubmatch(res[covIdx])
					if noTestRegex.MatchString(res[covIdx]) {
						o.Stdoutf("No test files")
						return nil
					}
				}
				if len(m) == 0 {
					return o.Stderrf("failed to parse coverage info")
				}

				f, err := strconv.ParseFloat(m[1], 64)
				if err != nil {
					return o.Annotate(err, "failed to parse coverage value")
				}

				o.Stdoutf(`Coverage is %2.2f\%`, f)
				return nil
			}),
		),
	)
}

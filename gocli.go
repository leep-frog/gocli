package gocli

import (
	"regexp"

	"github.com/leep-frog/command"
)

type gocli struct{}

func (gc *gocli) Changed() bool   { return false }
func (gc *gocli) Setup() []string { return nil }
func (gc *gocli) Name() string    { return "gt" }

var (
	pathArgs      = command.ListArg[string]("PATH", "Path(s) to go packages to test", 0, command.UnboundedList, &command.FileCompletor[[]string]{Distinct: true, IgnoreFiles: true})
	coverageRegex = regexp.MustCompile(`coverage: +([0-9]+\.[0-9]+)% of statements$`)
)

func (gc *gocli) Node() *command.Node {
	return command.BranchNode(map[string]*command.Node{},
		command.SerialNodes(
			pathArgs,
			command.ExecuteErrNode(func(o command.Output, d *command.Data) error {
				res, err := command.NewBashCommand[[]string]("", []string{"go test . -coverprofile=$(mktemp)"}).Run(o)
				if err != nil {
					return o.Annotate(err, "failed to run go test command")
				}

				covLine := res[len(res)-1]
				m := coverageRegex.FindStringSubmatch(covLine)
				if len(m) == 0 {
					return o.Stderrf("faield to parse coverage info from line %q", covLine)
				}

				o.Stdoutf("Coverage is %q", m[1])
				return nil
			}),
		),
	)
}

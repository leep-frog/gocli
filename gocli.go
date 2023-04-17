package gocli

import (
	"bufio"
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
			if verboseFlag.Get(d) {
				args = append(args, "-v")
			}
			if d.Has(funcFilterFlag.Name()) {
				parens := fmt.Sprintf("(%s)", strings.Join(funcFilterFlag.Get(d), "|"))
				args = append(args, "-run", parens)
			}

			// Run the command
			sc := &command.ShellCommand[[]string]{
				CommandName:   "go",
				Args:          args,
				ForwardStdout: true,
			}
			res, err := sc.Run(o, d)
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

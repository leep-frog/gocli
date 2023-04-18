package gocli

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/leep-frog/command"
	"github.com/leep-frog/command/sourcerer"
)

func noTestEvent(pkg string) *goTestEvent {
	return &goTestEvent{
		Action:  "output",
		Package: pkg,
		Output:  fmt.Sprintf("? %s [no test files]\n", pkg),
	}
}

func coverageTestEvent(pkg string, coverage float64) *goTestEvent {
	return &goTestEvent{
		Action:  "output",
		Package: pkg,
		Output:  coverageEventOutput(pkg, coverage) + "\n",
	}
}

func coverageEventOutput(pkg string, coverage float64) string {
	return fmt.Sprintf("ok %s coverage: %0.2f%% of statements", pkg, coverage)
}

func TestExecute(t *testing.T) {
	for _, curOS := range []sourcerer.OS{sourcerer.Linux(), sourcerer.Windows()} {
		for _, test := range []struct {
			name   string
			etc    *command.ExecuteTestCase
			events []*goTestEvent
		}{
			{
				name: "Fails if shell command fails",
				etc: &command.ExecuteTestCase{
					WantStderr: "failed to execute shell command: bad news bears\n",
					WantErr:    fmt.Errorf("failed to execute shell command: bad news bears"),
					RunResponses: []*command.FakeRun{{
						Err: fmt.Errorf("bad news bears"),
					}},
					WantRunContents: []*command.RunContents{{
						Name: "go",
						Args: []string{
							"test",
							".",
							"-json",
							"-coverprofile=(New-TemporaryFile)",
						},
					}},
					WantData: &command.Data{Values: map[string]interface{}{
						pathArgs.Name():        []string{"."},
						minCoverageFlag.Name(): 0.0,
					}},
				},
			},
			{
				name: "Fails if unknown action",
				events: []*goTestEvent{
					{Action: "ugh", Package: "p1"},
				},
				etc: &command.ExecuteTestCase{
					WantErr:    fmt.Errorf("Unknown package event action: \"ugh\""),
					WantStderr: "Unknown package event action: \"ugh\"\n",
					WantRunContents: []*command.RunContents{{
						Name: "go",
						Args: []string{
							"test",
							".",
							"-json",
							"-coverprofile=(New-TemporaryFile)",
						},
					}},
					WantData: &command.Data{Values: map[string]interface{}{
						pathArgs.Name():        []string{"."},
						minCoverageFlag.Name(): 0.0,
					}},
				},
			},
			{
				name: "Fails if invalid json",
				etc: &command.ExecuteTestCase{
					WantErr: fmt.Errorf("failed to parse go event (} bleh {\n): invalid character '}' looking for beginning of value"),
					WantStderr: strings.Join([]string{
						"failed to parse go event (} bleh {\n): invalid character '}' looking for beginning of value\n",
					}, "\n"),
					RunResponses: []*command.FakeRun{{
						Stdout: []string{"} bleh {"},
					}},
					WantRunContents: []*command.RunContents{{
						Name: "go",
						Args: []string{
							"test",
							".",
							"-json",
							"-coverprofile=(New-TemporaryFile)",
						},
					}},
					WantData: &command.Data{Values: map[string]interface{}{
						pathArgs.Name():        []string{"."},
						minCoverageFlag.Name(): 0.0,
					}},
				},
			},
			{
				name: "Ignores skip action",
				events: []*goTestEvent{
					{Action: "skip", Package: "p1"},
					noTestEvent("p1"),
				},
				etc: &command.ExecuteTestCase{
					WantRunContents: []*command.RunContents{{
						Name: "go",
						Args: []string{
							"test",
							".",
							"-json",
							"-coverprofile=(New-TemporaryFile)",
						},
					}},
					WantStdout: strings.Join([]string{
						"? p1 [no test files]",
						"",
					}, "\n"),
					WantData: &command.Data{Values: map[string]interface{}{
						pathArgs.Name():        []string{"."},
						minCoverageFlag.Name(): 0.0,
					}},
				},
			},
			{
				name: "Tests a package with no test files",
				events: []*goTestEvent{
					{Action: "success", Package: "p1"},
					noTestEvent("p1"),
				},
				etc: &command.ExecuteTestCase{
					WantRunContents: []*command.RunContents{{
						Name: "go",
						Args: []string{
							"test",
							".",
							"-json",
							"-coverprofile=(New-TemporaryFile)",
						},
					}},
					WantStdout: strings.Join([]string{
						"? p1 [no test files]",
						"",
					}, "\n"),
					WantData: &command.Data{Values: map[string]interface{}{
						pathArgs.Name():        []string{"."},
						minCoverageFlag.Name(): 0.0,
					}},
				},
			},
			{
				name: "Tests a package with coverage",
				events: []*goTestEvent{
					{Action: "success", Package: "p1"},
					coverageTestEvent("p1", 54.3),
				},
				etc: &command.ExecuteTestCase{
					WantRunContents: []*command.RunContents{{
						Name: "go",
						Args: []string{
							"test",
							".",
							"-json",
							"-coverprofile=(New-TemporaryFile)",
						},
					}},
					WantStdout: strings.Join([]string{
						coverageEventOutput("p1", 54.3),
						"",
					}, "\n"),
					WantData: &command.Data{Values: map[string]interface{}{
						pathArgs.Name():        []string{"."},
						minCoverageFlag.Name(): 0.0,
					}},
				},
			},
			{
				name: "Fails if no package coverage detected",
				events: []*goTestEvent{
					{Action: "success", Package: "p1"},
				},
				etc: &command.ExecuteTestCase{
					WantRunContents: []*command.RunContents{{
						Name: "go",
						Args: []string{
							"test",
							".",
							"-json",
							"-coverprofile=(New-TemporaryFile)",
						},
					}},
					WantErr: fmt.Errorf("No coverage set for package: p1"),
					WantStderr: strings.Join([]string{
						"No coverage set for package: p1",
						"",
					}, "\n"),
					WantData: &command.Data{Values: map[string]interface{}{
						pathArgs.Name():        []string{"."},
						minCoverageFlag.Name(): 0.0,
					}},
				},
			},
			{
				name: "Fails if multiple coverage events detected",
				events: []*goTestEvent{
					{Action: "success", Package: "p1"},
					noTestEvent("p1"),
					coverageTestEvent("p1", 1),
				},
				etc: &command.ExecuteTestCase{
					WantRunContents: []*command.RunContents{{
						Name: "go",
						Args: []string{
							"test",
							".",
							"-json",
							"-coverprofile=(New-TemporaryFile)",
						},
					}},
					WantErr: fmt.Errorf(`Duplicate package coverage: {Coverage: -1.00, Line: "? p1 [no test files]"}, {Coverage: 1.00, Line: "ok p1 coverage: 1.00%% of statements"}`),
					WantStderr: strings.Join([]string{
						`Duplicate package coverage: {Coverage: -1.00, Line: "? p1 [no test files]"}, {Coverage: 1.00, Line: "ok p1 coverage: 1.00% of statements"}`,
						"",
					}, "\n"),
					WantStdout: strings.Join([]string{
						"? p1 [no test files]",
						coverageEventOutput("p1", 1),
						"",
					}, "\n"),
					WantData: &command.Data{Values: map[string]interface{}{
						pathArgs.Name():        []string{"."},
						minCoverageFlag.Name(): 0.0,
					}},
				},
			},
			{
				name: "Fails if multiple, different package result events detected",
				events: []*goTestEvent{
					{Action: "success", Package: "p1"},
					{Action: "failure", Package: "p1"},
				},
				etc: &command.ExecuteTestCase{
					WantRunContents: []*command.RunContents{{
						Name: "go",
						Args: []string{
							"test",
							".",
							"-json",
							"-coverprofile=(New-TemporaryFile)",
						},
					}},
					WantErr: fmt.Errorf("Duplicate package results: success, failure"),
					WantStderr: strings.Join([]string{
						"Duplicate package results: success, failure",
						"",
					}, "\n"),
					WantData: &command.Data{Values: map[string]interface{}{
						pathArgs.Name():        []string{"."},
						minCoverageFlag.Name(): 0.0,
					}},
				},
			},
			{
				name: "Fails if multiple, same package result events detected",
				events: []*goTestEvent{
					{Action: "failure", Package: "p1"},
					{Action: "failure", Package: "p1"},
				},
				etc: &command.ExecuteTestCase{
					WantRunContents: []*command.RunContents{{
						Name: "go",
						Args: []string{
							"test",
							".",
							"-json",
							"-coverprofile=(New-TemporaryFile)",
						},
					}},
					WantErr: fmt.Errorf("Duplicate package results: failure, failure"),
					WantStderr: strings.Join([]string{
						"Duplicate package results: failure, failure",
						"",
					}, "\n"),
					WantData: &command.Data{Values: map[string]interface{}{
						pathArgs.Name():        []string{"."},
						minCoverageFlag.Name(): 0.0,
					}},
				},
			},
			{
				name: "Ignores later events if error encountered",
				events: []*goTestEvent{
					{Action: "success", Package: "p1"},
					{Action: "output", Package: "p1", Output: "some output\n"},
					{Action: "failure", Package: "p1"},
					{Action: "output", Package: "p1", Output: "some more output\n"},
					{Action: "output", Package: "p1", Output: "final output\n"},
				},
				etc: &command.ExecuteTestCase{
					WantRunContents: []*command.RunContents{{
						Name: "go",
						Args: []string{
							"test",
							".",
							"-json",
							"-coverprofile=(New-TemporaryFile)",
						},
					}},
					WantErr: fmt.Errorf("Duplicate package results: success, failure"),
					WantStderr: strings.Join([]string{
						"Duplicate package results: success, failure",
						"",
					}, "\n"),
					WantStdout: strings.Join([]string{
						"some output",
						"",
					}, "\n"),
					WantData: &command.Data{Values: map[string]interface{}{
						pathArgs.Name():        []string{"."},
						minCoverageFlag.Name(): 0.0,
					}},
				},
			},
			{
				name: "Fails if coverage is below threshold",
				events: []*goTestEvent{
					{Action: "success", Package: "p1"},
					coverageTestEvent("p1", 54.3),
				},
				etc: &command.ExecuteTestCase{
					Args: []string{"-m", "54.4"},
					WantRunContents: []*command.RunContents{{
						Name: "go",
						Args: []string{
							"test",
							".",
							"-json",
							"-coverprofile=(New-TemporaryFile)",
						},
					}},
					WantErr: fmt.Errorf("Coverage of package \"p1\" (54.3%%) must be at least 54.4%%"),
					WantStderr: strings.Join([]string{
						"Coverage of package \"p1\" (54.3%) must be at least 54.4%",
						"",
					}, "\n"),
					WantStdout: strings.Join([]string{
						coverageEventOutput("p1", 54.3),
						"",
					}, "\n"),
					WantData: &command.Data{Values: map[string]interface{}{
						pathArgs.Name():        []string{"."},
						minCoverageFlag.Name(): 54.4,
					}},
				},
			},
			{
				name: "Succeeds if coverage is at threshold",
				events: []*goTestEvent{
					{Action: "success", Package: "p1"},
					coverageTestEvent("p1", 54.4),
				},
				etc: &command.ExecuteTestCase{
					Args: []string{"-m", "54.4"},
					WantRunContents: []*command.RunContents{{
						Name: "go",
						Args: []string{
							"test",
							".",
							"-json",
							"-coverprofile=(New-TemporaryFile)",
						},
					}},
					WantStdout: strings.Join([]string{
						coverageEventOutput("p1", 54.4),
						"",
					}, "\n"),
					WantData: &command.Data{Values: map[string]interface{}{
						pathArgs.Name():        []string{"."},
						minCoverageFlag.Name(): 54.4,
					}},
				},
			},
			{
				name: "Succeeds if coverage is above threshold",
				events: []*goTestEvent{
					{Action: "success", Package: "p1"},
					coverageTestEvent("p1", 54.5),
				},
				etc: &command.ExecuteTestCase{
					Args: []string{"-m", "54.4"},
					WantRunContents: []*command.RunContents{{
						Name: "go",
						Args: []string{
							"test",
							".",
							"-json",
							"-coverprofile=(New-TemporaryFile)",
						},
					}},
					WantStdout: strings.Join([]string{
						coverageEventOutput("p1", 54.5),
						"",
					}, "\n"),
					WantData: &command.Data{Values: map[string]interface{}{
						pathArgs.Name():        []string{"."},
						minCoverageFlag.Name(): 54.4,
					}},
				},
			},
			{
				name: "Tests a package when output event is first",
				events: []*goTestEvent{
					noTestEvent("p1"),
					{Action: "success", Package: "p1"},
				},
				etc: &command.ExecuteTestCase{
					WantRunContents: []*command.RunContents{{
						Name: "go",
						Args: []string{
							"test",
							".",
							"-json",
							"-coverprofile=(New-TemporaryFile)",
						},
					}},
					WantStdout: strings.Join([]string{
						"? p1 [no test files]",
						"",
					}, "\n"),
					WantData: &command.Data{Values: map[string]interface{}{
						pathArgs.Name():        []string{"."},
						minCoverageFlag.Name(): 0.0,
					}},
				},
			},
			{
				name: "Includes timeout flag",
				events: []*goTestEvent{
					{Action: "success", Package: "p1"},
					noTestEvent("p1"),
				},
				etc: &command.ExecuteTestCase{
					Args: []string{"-t", "123"},
					WantRunContents: []*command.RunContents{{
						Name: "go",
						Args: []string{
							"test",
							"-timeout",
							"123s",
							".",
							"-json",
							"-coverprofile=(New-TemporaryFile)",
						},
					}},
					WantStdout: strings.Join([]string{
						"? p1 [no test files]",
						"",
					}, "\n"),
					WantData: &command.Data{Values: map[string]interface{}{
						pathArgs.Name():        []string{"."},
						minCoverageFlag.Name(): 0.0,
						timeoutFlag.Name():     123,
					}},
				},
			},
			{
				name: "Includes func-filter flag",
				events: []*goTestEvent{
					{Action: "success", Package: "p1"},
					noTestEvent("p1"),
				},
				etc: &command.ExecuteTestCase{
					Args: []string{"-f", "SingleFunc", "DoubleFunc"},
					WantRunContents: []*command.RunContents{{
						Name: "go",
						Args: []string{
							"test",
							".",
							"-json",
							"-run",
							"(SingleFunc|DoubleFunc)",
						},
					}},
					WantStdout: strings.Join([]string{
						"? p1 [no test files]",
						"",
					}, "\n"),
					WantData: &command.Data{Values: map[string]interface{}{
						pathArgs.Name():        []string{"."},
						minCoverageFlag.Name(): 0.0,
						funcFilterFlag.Name():  []string{"SingleFunc", "DoubleFunc"},
					}},
				},
			},
			{
				name: "Outputs package lines",
				events: []*goTestEvent{
					{Action: "success", Package: "p1"},
					noTestEvent("p1"),
					{Action: "output", Test: "some-test", Output: "test started\n"},
					{Action: "output", Output: "package output\n"},
					{Action: "output", Test: "some-test", Output: "test ended\n"},
				},
				etc: &command.ExecuteTestCase{
					WantRunContents: []*command.RunContents{{
						Name: "go",
						Args: []string{
							"test",
							".",
							"-json",
							"-coverprofile=(New-TemporaryFile)",
						},
					}},
					WantStdout: strings.Join([]string{
						"? p1 [no test files]",
						"package output",
						"",
					}, "\n"),
					WantData: &command.Data{Values: map[string]interface{}{
						pathArgs.Name():        []string{"."},
						minCoverageFlag.Name(): 0.0,
					}},
				},
			},
			{
				name: "Outputs test lines when verbose flag is provided",
				events: []*goTestEvent{
					{Action: "success", Package: "p1"},
					noTestEvent("p1"),
					{Action: "output", Test: "some-test", Output: "test started\n"},
					{Action: "output", Output: "package output\n"},
					{Action: "output", Test: "some-test", Output: "test ended\n"},
				},
				etc: &command.ExecuteTestCase{
					Args: []string{"-v"},
					WantRunContents: []*command.RunContents{{
						Name: "go",
						Args: []string{
							"test",
							".",
							"-json",
							"-v",
							"-coverprofile=(New-TemporaryFile)",
						},
					}},
					WantStdout: strings.Join([]string{
						"? p1 [no test files]",
						"test started",
						"package output",
						"test ended",
						"",
					}, "\n"),
					WantData: &command.Data{Values: map[string]interface{}{
						pathArgs.Name():        []string{"."},
						minCoverageFlag.Name(): 0.0,
						verboseFlag.Name():     true,
					}},
				},
			},
			{
				name: "Verbose and timeout flags",
				events: []*goTestEvent{
					{Action: "success", Package: "p1"},
					noTestEvent("p1"),
					{Action: "output", Test: "some-test", Output: "test started\n"},
					{Action: "output", Output: "package output\n"},
					{Action: "output", Test: "some-test", Output: "test ended\n"},
				},
				etc: &command.ExecuteTestCase{
					Args: []string{"-v", "-t", "456"},
					WantRunContents: []*command.RunContents{{
						Name: "go",
						Args: []string{
							"test",
							"-timeout",
							"456s",
							".",
							"-json",
							"-v",
							"-coverprofile=(New-TemporaryFile)",
						},
					}},
					WantStdout: strings.Join([]string{
						"? p1 [no test files]",
						"test started",
						"package output",
						"test ended",
						"",
					}, "\n"),
					WantData: &command.Data{Values: map[string]interface{}{
						pathArgs.Name():        []string{"."},
						minCoverageFlag.Name(): 0.0,
						verboseFlag.Name():     true,
						timeoutFlag.Name():     456,
					}},
				},
			},
			{
				name: "Verbose, timeout, and func filter flags",
				events: []*goTestEvent{
					{Action: "success", Package: "p1"},
					coverageTestEvent("p1", 75),
					{Action: "output", Test: "some-test", Output: "test started\n"},
					{Action: "output", Output: "package output\n"},
					{Action: "output", Test: "some-test", Output: "test ended\n"},
				},
				etc: &command.ExecuteTestCase{
					Args: []string{"-v", "-t", "456", "-f", "FuncName", "OtherFunc"},
					WantRunContents: []*command.RunContents{{
						Name: "go",
						Args: []string{
							"test",
							"-timeout",
							"456s",
							".",
							"-json",
							"-v",
							"-run",
							"(FuncName|OtherFunc)",
						},
					}},
					WantStdout: strings.Join([]string{
						coverageEventOutput("p1", 75),
						"test started",
						"package output",
						"test ended",
						"",
					}, "\n"),
					WantData: &command.Data{Values: map[string]interface{}{
						pathArgs.Name():        []string{"."},
						minCoverageFlag.Name(): 0.0,
						verboseFlag.Name():     true,
						timeoutFlag.Name():     456,
						funcFilterFlag.Name():  []string{"FuncName", "OtherFunc"},
					}},
				},
			},
			{
				name: "Fails if coverage and and func filter flags",
				etc: &command.ExecuteTestCase{
					Args:    []string{"-m", "33", "-f", "FuncName", "OtherFunc"},
					WantErr: fmt.Errorf("Cannot set func-filter and min coverage flags simultaneously"),
					WantStderr: strings.Join([]string{
						"Cannot set func-filter and min coverage flags simultaneously",
						"",
					}, "\n"),
					WantData: &command.Data{Values: map[string]interface{}{
						pathArgs.Name():        []string{"."},
						minCoverageFlag.Name(): 33.0,
						funcFilterFlag.Name():  []string{"FuncName", "OtherFunc"},
					}},
				},
			},
			{
				name: "Handles multiple packages",
				events: []*goTestEvent{
					{Action: "success", Package: "p1"},
					{Action: "success", Package: "p2"},
					{Action: "success", Package: "p3"},
					coverageTestEvent("p1", 75),
					noTestEvent("p2"),
					coverageTestEvent("p3", 44),
				},
				etc: &command.ExecuteTestCase{
					Args: []string{"./..."},
					WantRunContents: []*command.RunContents{{
						Name: "go",
						Args: []string{
							"test",
							"./...",
							"-json",
							"-coverprofile=(New-TemporaryFile)",
						},
					}},
					WantStdout: strings.Join([]string{
						coverageEventOutput("p1", 75),
						"? p2 [no test files]",
						coverageEventOutput("p3", 44),
						"",
					}, "\n"),
					WantData: &command.Data{Values: map[string]interface{}{
						pathArgs.Name():        []string{"./..."},
						minCoverageFlag.Name(): 0.0,
					}},
				},
			},
			{
				name: "Outputs errors for multiple packages",
				events: []*goTestEvent{
					{Action: "success", Package: "p1"},
					{Action: "success", Package: "p2"},
					{Action: "success", Package: "p3"},
					coverageTestEvent("p1", 75),
					// Missing p2 info
					coverageTestEvent("p3", 43.2), // Insufficient p3 info
				},
				etc: &command.ExecuteTestCase{
					Args: []string{"./...", "-m", "50"},
					WantRunContents: []*command.RunContents{{
						Name: "go",
						Args: []string{
							"test",
							"./...",
							"-json",
							"-coverprofile=(New-TemporaryFile)",
						},
					}},
					WantStdout: strings.Join([]string{
						coverageEventOutput("p1", 75),
						coverageEventOutput("p3", 43.2),
						"",
					}, "\n"),
					WantStderr: strings.Join([]string{
						"No coverage set for package: p2",
						`Coverage of package "p3" (43.2%) must be at least 50.0%`,
						"",
					}, "\n"),
					WantErr: fmt.Errorf(`Coverage of package "p3" (43.2%%) must be at least 50.0%%`),
					WantData: &command.Data{Values: map[string]interface{}{
						pathArgs.Name():        []string{"./..."},
						minCoverageFlag.Name(): 50.0,
					}},
				},
			},
			/* Useful for commenting out tests. */
		} {
			t.Run(fmt.Sprintf("[%s] %s", curOS.Name(), test.name), func(t *testing.T) {
				command.StubValue(t, &sourcerer.CurrentOS, curOS)

				if curOS.Name() == "linux" {
					for _, rc := range test.etc.WantRunContents {
						for i, a := range rc.Args {
							if a == "-coverprofile=(New-TemporaryFile)" {
								rc.Args[i] = "-coverprofile=$(mktemp)"
							}
						}
					}
				}

				test.etc.Node = (&goCLI{}).Node()
				if test.events != nil {
					var frs []string
					for _, e := range test.events {
						b, err := json.Marshal(e)
						if err != nil {
							t.Fatalf("Failed to marshal event to json: %v", err)
						}
						frs = append(frs, string(b))
					}
					test.etc.RunResponses = []*command.FakeRun{{
						Stdout: frs,
					}}
				}
				command.ExecuteTest(t, test.etc)
			})
		}
	}
}

func TestAutocomplete(t *testing.T) {
	for _, test := range []struct {
		name string
		ctc  *command.CompleteTestCase
	}{
		{
			name: "completes directories",
			ctc: &command.CompleteTestCase{
				Want: []string{
					".git/",
					"testpkg/",
					" ",
				},
				WantData: &command.Data{
					Values: map[string]interface{}{
						pathArgs.Name():        []string{""},
						minCoverageFlag.Name(): 0.0,
					},
				},
			},
		},
		{
			name: "completes test function names in current directory",
			ctc: &command.CompleteTestCase{
				Args: "cmd -f ",
				Want: []string{
					"Autocomplete",
					"Execute",
					"Metadata",
				},
				WantData: &command.Data{
					Values: map[string]interface{}{
						pathArgs.Name():       []string{"."},
						funcFilterFlag.Name(): []string{""},
					},
				},
			},
		},
		{
			name: "completes test function names in all sub directories",
			ctc: &command.CompleteTestCase{
				Args: "cmd './...' -f ",
				Want: []string{
					"Autocomplete",
					"Execute",
					"Metadata",
					"Other",
					"That",
					"This",
				},
				WantData: &command.Data{
					Values: map[string]interface{}{
						pathArgs.Name():       []string{"./..."},
						funcFilterFlag.Name(): []string{""},
					},
				},
			},
		},
		{
			name: "completes partial test function names",
			ctc: &command.CompleteTestCase{
				Args: "cmd -f A",
				Want: []string{
					"Autocomplete",
				},
				WantData: &command.Data{
					Values: map[string]interface{}{
						pathArgs.Name():       []string{"."},
						funcFilterFlag.Name(): []string{"A"},
					},
				},
			},
		},
		{
			name: "completes distinct test function names",
			ctc: &command.CompleteTestCase{
				Args: "cmd ./... -f That T",
				Want: []string{
					"This",
				},
				WantData: &command.Data{
					Values: map[string]interface{}{
						pathArgs.Name():       []string{"./..."},
						funcFilterFlag.Name(): []string{"That", "T"},
					},
				},
			},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			test.ctc.Node = (&goCLI{}).Node()
			command.CompleteTest(t, test.ctc)
		})
	}
}

func TestMetadata(t *testing.T) {
	gc := &goCLI{}
	if gc.Changed() {
		t.Errorf("gc.Changed() returned true; want false")
	}

	if len(gc.Setup()) != 0 {
		t.Errorf("gc.Setup() returned non-nil value: %v", gc.Setup())
	}

	if gc.Name() != "gt" {
		t.Errorf("gc.Name() returned %q; want 'gt'", gc.Name())
	}
}

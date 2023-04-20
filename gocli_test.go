package gocli

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/leep-frog/command"
)

func successOutput(pkg string, coverage float64) string {
	return fmt.Sprintf("ok \t %s \t coverage: \t %0.2f%% of statements", pkg, coverage)
}

func failLine(pkg string) string {
	return fmt.Sprintf("FAIL \t %s \t abc \t 123 def", pkg)
}

func noTestLine(pkg string) string {
	return fmt.Sprintf("?       %s        [no test files]", pkg)
}

func TestExecute(t *testing.T) {
	for _, test := range []struct {
		name       string
		etc        *command.ExecuteTestCase
		tmpFileErr error
	}{
		{
			name: "Works when no coverage returned",
			etc: &command.ExecuteTestCase{
				RunResponses: []*command.FakeRun{{}},
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						".",
						"-coverprofile=(TMP_FILE)",
					},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 0.0,
				}},
			},
		},
		{
			name: "Fails if shell command error",
			etc: &command.ExecuteTestCase{
				WantErr:    fmt.Errorf("go test shell command error: failed to execute shell command: bad news bears"),
				WantStderr: "go test shell command error: failed to execute shell command: bad news bears\n",
				RunResponses: []*command.FakeRun{{
					Err: fmt.Errorf("bad news bears"),
				}},
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						".",
						"-coverprofile=(TMP_FILE)",
					},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 0.0,
				}},
			},
		},
		{
			name: "Ignores test input no coverage returned",
			etc: &command.ExecuteTestCase{
				RunResponses: []*command.FakeRun{{
					Stdout: []string{
						"hello there",
						"general kenobi",
					},
				}},
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						".",
						"-coverprofile=(TMP_FILE)",
					},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 0.0,
				}},
			},
		},
		{
			name: "Gets coverage result",
			etc: &command.ExecuteTestCase{
				RunResponses: []*command.FakeRun{{
					Stdout: []string{
						"hello there",
						successOutput("p1", 12.34),
						"general kenobi",
					},
				}},
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						".",
						"-coverprofile=(TMP_FILE)",
					},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 0.0,
					"COVERAGE": map[string]*packageResult{
						"p1": {
							testSuccess,
							12.34,
							successOutput("p1", 12.34) + "\n",
						},
					},
				}},
			},
		},
		{
			name: "Gets no-test result",
			etc: &command.ExecuteTestCase{
				RunResponses: []*command.FakeRun{{
					Stdout: []string{
						"hello there",
						noTestLine("p1"),
						"general kenobi",
					},
				}},
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						".",
						"-coverprofile=(TMP_FILE)",
					},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 0.0,
					"COVERAGE": map[string]*packageResult{
						"p1": {
							noTestFiles,
							0.0,
							noTestLine("p1") + "\n",
						},
					},
				}},
			},
		},
		{
			name: "Gets test failure result",
			etc: &command.ExecuteTestCase{
				RunResponses: []*command.FakeRun{{
					Stdout: []string{
						"hello there",
						failLine("p1"),
						"general kenobi",
					},
				}},
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						".",
						"-coverprofile=(TMP_FILE)",
					},
				}},
				WantErr:    fmt.Errorf("Tests failed for package: p1"),
				WantStderr: "Tests failed for package: p1\n",
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 0.0,
					"COVERAGE": map[string]*packageResult{
						"p1": {
							testFailure,
							0.0,
							failLine("p1") + "\n",
						},
					},
				}},
			},
		},
		{
			name: "Adds timeout flag",
			etc: &command.ExecuteTestCase{
				Args: []string{"-t", "123"},
				RunResponses: []*command.FakeRun{{
					Stdout: []string{
						noTestLine("p1"),
					},
				}},
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						"-timeout",
						"123s",
						".",
						"-coverprofile=(TMP_FILE)",
					},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 0.0,
					timeoutFlag.Name():     123,
					"COVERAGE": map[string]*packageResult{
						"p1": {
							noTestFiles,
							0.0,
							noTestLine("p1") + "\n",
						},
					},
				}},
			},
		},
		{
			name: "Succeeds if coverage result is above threshold",
			etc: &command.ExecuteTestCase{
				Args: []string{"-m", "54.32"},
				RunResponses: []*command.FakeRun{{
					Stdout: []string{
						successOutput("p1", 54.33),
					},
				}},
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						".",
						"-coverprofile=(TMP_FILE)",
					},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 54.32,
					"COVERAGE": map[string]*packageResult{
						"p1": {
							testSuccess,
							54.33,
							successOutput("p1", 54.33) + "\n",
						},
					},
				}},
			},
		},
		{
			name: "Succeeds if coverage result is at threshold",
			etc: &command.ExecuteTestCase{
				Args: []string{"-m", "54.32"},
				RunResponses: []*command.FakeRun{{
					Stdout: []string{
						successOutput("p1", 54.32),
					},
				}},
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						".",
						"-coverprofile=(TMP_FILE)",
					},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 54.32,
					"COVERAGE": map[string]*packageResult{
						"p1": {
							testSuccess,
							54.32,
							successOutput("p1", 54.32) + "\n",
						},
					},
				}},
			},
		},
		{
			name: "Fails if coverage result is below threshold",
			etc: &command.ExecuteTestCase{
				Args: []string{"-m", "54.32"},
				RunResponses: []*command.FakeRun{{
					Stdout: []string{
						successOutput("p1", 54.31),
					},
				}},
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						".",
						"-coverprofile=(TMP_FILE)",
					},
				}},
				WantErr:    fmt.Errorf("Coverage of package \"p1\" (54.3%%) must be at least 54.3%%"),
				WantStderr: "Coverage of package \"p1\" (54.3%) must be at least 54.3%\n",
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 54.32,
					"COVERAGE": map[string]*packageResult{
						"p1": {
							testSuccess,
							54.31,
							successOutput("p1", 54.31) + "\n",
						},
					},
				}},
			},
		},
		/*{
			name: "Fails if shell command fails",
			etc: &command.ExecuteTestCase{
				WantStderr: "go test shell command error: failed to execute shell command: bad news bears\n",
				WantErr:    fmt.Errorf("go test shell command error: failed to execute shell command: bad news bears"),
				RunResponses: []*command.FakeRun{{
					Err: fmt.Errorf("bad news bears"),
				}},
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						".",
						"-json",
						"-coverprofile=(TMP_FILE)",
					},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 0.0,
				}},
			},
		},
		{
			name: "Fails if unknown action for package",
			events: []*goTestEvent{
				{Action: "ugh", Package: "p1"},
			},
			etc: &command.ExecuteTestCase{
				WantErr:    fmt.Errorf("event handling error: unknown package event action: \"ugh\""),
				WantStderr: "event handling error: unknown package event action: \"ugh\"\n",
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						".",
						"-json",
						"-coverprofile=(TMP_FILE)",
					},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 0.0,
				}},
			},
		},
		{
			name: "Fails if unknown action for test",
			events: []*goTestEvent{
				{Action: "idk", Package: "p1", Test: "some-test"},
			},
			etc: &command.ExecuteTestCase{
				WantErr:    fmt.Errorf("event handling error: unknown test event action: \"idk\""),
				WantStderr: "event handling error: unknown test event action: \"idk\"\n",
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						".",
						"-json",
						"-coverprofile=(TMP_FILE)",
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
				WantErr: fmt.Errorf("event handling error: failed to parse go event (} bleh {): invalid character '}' looking for beginning of value"),
				WantStderr: strings.Join([]string{
					"event handling error: failed to parse go event (} bleh {): invalid character '}' looking for beginning of value\n",
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
						"-coverprofile=(TMP_FILE)",
					},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 0.0,
				}},
			},
		},
		{
			name: "Ignore actions action",
			events: []*goTestEvent{
				{Action: "skip", Package: "p1"},
				{Action: "start", Package: "p1"},
				{Action: "pass", Package: "p1", Test: "some-test"},
				{Action: "run", Package: "p1", Test: "some-test"},
				noTestEvent("p1"),
			},
			etc: &command.ExecuteTestCase{
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						".",
						"-json",
						"-coverprofile=(TMP_FILE)",
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
				{Action: "pass", Package: "p1"},
				noTestEvent("p1"),
			},
			etc: &command.ExecuteTestCase{
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						".",
						"-json",
						"-coverprofile=(TMP_FILE)",
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
				{Action: "pass", Package: "p1"},
				coverageTestEvent("p1", 54.3),
			},
			etc: &command.ExecuteTestCase{
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						".",
						"-json",
						"-coverprofile=(TMP_FILE)",
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
			name: "Fails if package test fails",
			events: []*goTestEvent{
				{Action: "fail", Package: "p1"},
			},
			etc: &command.ExecuteTestCase{
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						".",
						"-json",
						"-coverprofile=(TMP_FILE)",
					},
				}},
				WantErr: fmt.Errorf("Tests failed for package: p1"),
				WantStderr: strings.Join([]string{
					"Tests failed for package: p1",
					"",
				}, "\n"),
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 0.0,
				}},
			},
		},
		{
			name:       "Fails if tmp file error",
			tmpFileErr: fmt.Errorf("oops"),
			etc: &command.ExecuteTestCase{
				WantErr: fmt.Errorf("failed to create temporary file: oops"),
				WantStderr: strings.Join([]string{
					"failed to create temporary file: oops",
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
				{Action: "pass", Package: "p1"},
			},
			etc: &command.ExecuteTestCase{
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						".",
						"-json",
						"-coverprofile=(TMP_FILE)",
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
				{Action: "pass", Package: "p1"},
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
						"-coverprofile=(TMP_FILE)",
					},
				}},
				WantErr: fmt.Errorf(`event handling error: duplicate package coverage: {Coverage: -1.00, Line: "? p1 [no test files]"}, {Coverage: 1.00, Line: "ok p1 coverage: 1.00%% of statements"}`),
				WantStderr: strings.Join([]string{
					`event handling error: duplicate package coverage: {Coverage: -1.00, Line: "? p1 [no test files]"}, {Coverage: 1.00, Line: "ok p1 coverage: 1.00% of statements"}`,
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
				{Action: "pass", Package: "p1"},
				{Action: "fail", Package: "p1"},
			},
			etc: &command.ExecuteTestCase{
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						".",
						"-json",
						"-coverprofile=(TMP_FILE)",
					},
				}},
				WantErr: fmt.Errorf("event handling error: duplicate package results: pass, fail"),
				WantStderr: strings.Join([]string{
					"event handling error: duplicate package results: pass, fail",
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
				{Action: "fail", Package: "p1"},
				{Action: "fail", Package: "p1"},
			},
			etc: &command.ExecuteTestCase{
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						".",
						"-json",
						"-coverprofile=(TMP_FILE)",
					},
				}},
				WantErr: fmt.Errorf("event handling error: duplicate package results: fail, fail"),
				WantStderr: strings.Join([]string{
					"event handling error: duplicate package results: fail, fail",
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
				{Action: "pass", Package: "p1"},
				{Action: "output", Package: "p1", Output: "some output\n"},
				{Action: "fail", Package: "p1"},
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
						"-coverprofile=(TMP_FILE)",
					},
				}},
				WantErr: fmt.Errorf("event handling error: duplicate package results: pass, fail"),
				WantStderr: strings.Join([]string{
					"event handling error: duplicate package results: pass, fail",
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
				{Action: "pass", Package: "p1"},
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
						"-coverprofile=(TMP_FILE)",
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
				{Action: "pass", Package: "p1"},
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
						"-coverprofile=(TMP_FILE)",
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
				{Action: "pass", Package: "p1"},
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
						"-coverprofile=(TMP_FILE)",
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
				{Action: "pass", Package: "p1"},
			},
			etc: &command.ExecuteTestCase{
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						".",
						"-json",
						"-coverprofile=(TMP_FILE)",
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
				{Action: "pass", Package: "p1"},
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
						"-coverprofile=(TMP_FILE)",
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
				{Action: "pass", Package: "p1"},
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
				{Action: "pass", Package: "p1"},
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
						"-coverprofile=(TMP_FILE)",
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
			name: "Outputs test lines for pass and fail when verbose flag is provided",
			events: []*goTestEvent{
				{Action: "pass", Package: "p1"},
				noTestEvent("p1"),
				{Action: "output", Test: "some-test-1", Output: "test 1 started\n"},
				{Action: "output", Output: "package output\n"},
				{Action: "output", Test: "some-test-1", Output: "test 1 working\n"},
				{Action: "output", Test: "some-test-2", Output: "test 2 started\n"},
				{Action: "output", Test: "some-test-2", Output: "test 2 working\n"},
				{Action: "output", Test: "some-test-2", Output: "test 2 ended\n"},
				{Action: "output", Test: "some-test-1", Output: "test 1 ended\n"},
				{Action: "pass", Test: "some-test-2"},
				{Action: "output", Output: "more package info\n"},
				{Action: "fail", Test: "some-test-1"},
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
						"-coverprofile=(TMP_FILE)",
					},
				}},
				WantStdout: strings.Join([]string{
					"? p1 [no test files]",
					"test 1 started",
					"package output",
					"test 1 working",
					"test 2 started",
					"test 2 working",
					"test 2 ended",
					"test 1 ended",
					"more package info",
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
			name: "Outputs only failed tests if verbose flag is not set",
			events: []*goTestEvent{
				{Action: "pass", Package: "p1"},
				noTestEvent("p1"),
				{Action: "output", Test: "some-test-1", Output: "test 1 started\n"},
				{Action: "output", Output: "package output\n"},
				{Action: "output", Test: "some-test-1", Output: "test 1 working\n"},
				{Action: "output", Test: "some-test-2", Output: "test 2 started\n"},
				{Action: "output", Test: "some-test-2", Output: "test 2 working\n"},
				{Action: "output", Test: "some-test-2", Output: "test 2 ended\n"},
				{Action: "output", Test: "some-test-1", Output: "test 1 ended\n"},
				{Action: "pass", Test: "some-test-2"},
				{Action: "output", Output: "more package info\n"},
				{Action: "fail", Test: "some-test-1"},
			},
			etc: &command.ExecuteTestCase{
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						".",
						"-json",
						"-coverprofile=(TMP_FILE)",
					},
				}},
				WantStdout: strings.Join([]string{
					"? p1 [no test files]",
					"package output",
					"more package info",
					"test 1 started",
					"test 1 working",
					"test 1 ended",
					"",
				}, "\n"),
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 0.0,
				}},
			},
		},
		{
			name: "Verbose and timeout flags",
			events: []*goTestEvent{
				{Action: "pass", Package: "p1"},
				noTestEvent("p1"),
				{Action: "output", Test: "some-test", Output: "test started\n"},
				{Action: "output", Output: "package output\n"},
				{Action: "output", Test: "some-test", Output: "test ended\n"},
				{Action: "pass", Test: "some-test"},
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
						"-coverprofile=(TMP_FILE)",
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
				{Action: "pass", Package: "p1"},
				coverageTestEvent("p1", 75),
				{Action: "output", Test: "some-test", Output: "test started\n"},
				{Action: "output", Output: "package output\n"},
				{Action: "output", Test: "some-test", Output: "test ended\n"},
				{Action: "pass", Test: "some-test"},
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
				{Action: "pass", Package: "p1"},
				{Action: "pass", Package: "p2"},
				{Action: "pass", Package: "p3"},
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
						"-coverprofile=(TMP_FILE)",
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
			name: "Handles multiple events in the same input",
			etc: &command.ExecuteTestCase{
				Args: []string{"./..."},
				RunResponses: []*command.FakeRun{
					{
						Stdout: []string{
							marshalEvents(t,
								&goTestEvent{Action: "pass", Package: "p1"},
								&goTestEvent{Action: "output", Output: "huzzah\n"},
								&goTestEvent{Action: "pass", Package: "p2"},
								&goTestEvent{Action: "pass", Package: "p3"},
								coverageTestEvent("p1", 75),
								noTestEvent("p2"),
								&goTestEvent{Action: "output", Output: "hurray\n"},
								coverageTestEvent("p3", 44),
							),
						},
					},
				},
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						"./...",
						"-json",
						"-coverprofile=(TMP_FILE)",
					},
				}},
				WantStdout: strings.Join([]string{
					"huzzah",
					coverageEventOutput("p1", 75),
					"? p2 [no test files]",
					"hurray",
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
				{Action: "pass", Package: "p1"},
				{Action: "pass", Package: "p2"},
				{Action: "pass", Package: "p3"},
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
						"-coverprofile=(TMP_FILE)",
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
		t.Run(test.name, func(t *testing.T) {
			tmp, err := tmpFile()
			if err != nil {
				t.Fatalf("failed to create temporary file")
			}
			command.StubValue(t, &tmpFile, func() (*os.File, error) {
				return tmp, test.tmpFileErr
			})

			for _, rc := range test.etc.WantRunContents {
				for i, a := range rc.Args {
					if a == "-coverprofile=(TMP_FILE)" {
						rc.Args[i] = fmt.Sprintf("-coverprofile=%s", tmp.Name())
					}
				}
			}

			test.etc.Node = (&goCLI{}).Node()
			if test.etc.RunResponses != nil && len(test.etc.RunResponses[0].Stdout) > 0 {
				test.etc.WantStdout = fmt.Sprintf("%s%s", test.etc.WantStdout, strings.Join(test.etc.RunResponses[0].Stdout, "\n")) + "\n"
			}
			command.ExecuteTest(t, test.etc)
		})
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
					"testdata/",
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

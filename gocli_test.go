package gocli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leep-frog/command"
)

func successOutput(pkg string, coverage float64) string {
	return fmt.Sprintf("ok \t %s \t 0.123s \t coverage: \t %0.2f%% of statements", pkg, coverage)
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
							successOutput("p1", 12.34),
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
							noTestLine("p1"),
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
							failLine("p1"),
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
							noTestLine("p1"),
						},
					},
				}},
			},
		},
		{
			name: "Adds verbose flag",
			etc: &command.ExecuteTestCase{
				Args: []string{"-v"},
				RunResponses: []*command.FakeRun{{
					Stdout: []string{
						noTestLine("p1"),
					},
				}},
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						".",
						"-v",
						"-coverprofile=(TMP_FILE)",
					},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 0.0,
					verboseFlag.Name():     true,
					"COVERAGE": map[string]*packageResult{
						"p1": {
							noTestFiles,
							0.0,
							noTestLine("p1"),
						},
					},
				}},
			},
		},
		{
			name: "Adds func-filter flag",
			etc: &command.ExecuteTestCase{
				Args: []string{"-f", "SomeTest", "OtherTest"},
				RunResponses: []*command.FakeRun{{
					Stdout: []string{
						noTestLine("p1"),
					},
				}},
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						".",
						"-run",
						"(SomeTest|OtherTest)",
					},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 0.0,
					funcFilterFlag.Name():  []string{"SomeTest", "OtherTest"},
					"COVERAGE": map[string]*packageResult{
						"p1": {
							noTestFiles,
							0.0,
							noTestLine("p1"),
						},
					},
				}},
			},
		},
		{
			name: "Fails if func-filter flag and min coverage flag",
			etc: &command.ExecuteTestCase{
				Args:       []string{"-m", "12.34", "-f", "SomeTest", "OtherTest"},
				WantStderr: "Cannot set func-filter and min coverage flags simultaneously\n",
				WantErr:    fmt.Errorf("Cannot set func-filter and min coverage flags simultaneously"),
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 12.34,
					funcFilterFlag.Name():  []string{"SomeTest", "OtherTest"},
				}},
			},
		},
		{
			name:       "Fails if tmp file error",
			tmpFileErr: fmt.Errorf("oops"),
			etc: &command.ExecuteTestCase{
				WantStderr: "failed to create temporary file: oops\n",
				WantErr:    fmt.Errorf("failed to create temporary file: oops"),
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 0.0,
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
							successOutput("p1", 54.33),
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
							successOutput("p1", 54.32),
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
							successOutput("p1", 54.31),
						},
					},
				}},
			},
		},
		{
			name: "Fails if multiple regexes for same package",
			etc: &command.ExecuteTestCase{
				RunResponses: []*command.FakeRun{{
					Stdout: []string{
						noTestLine("p1"),
						successOutput("p1", 12.34),
					},
				}},
				WantErr:    fmt.Errorf("event handling error: Multiple results for package \"p1\":\n  Result 1: ?       p1        [no test files]\n  Result 2: ok \t p1 \t 0.123s \t coverage: \t 12.34%% of statements"),
				WantStderr: "event handling error: Multiple results for package \"p1\":\n  Result 1: ?       p1        [no test files]\n  Result 2: ok \t p1 \t 0.123s \t coverage: \t 12.34% of statements\n",
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
			name: "Handles multiple errors",
			etc: &command.ExecuteTestCase{
				RunResponses: []*command.FakeRun{{
					Stdout: []string{
						failLine("p1"),
						noTestLine("p1"),
						successOutput("p1", 12.34),
					},
				}},
				WantErr:    fmt.Errorf("event handling error: Multiple results for package \"p1\":\n  Result 1: FAIL \t p1 \t abc \t 123 def\n  Result 2: ?       p1        [no test files]"),
				WantStderr: "event handling error: Multiple results for package \"p1\":\n  Result 1: FAIL \t p1 \t abc \t 123 def\n  Result 2: ?       p1        [no test files]\n",
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
			name: "Handles multiple pacakges successes",
			etc: &command.ExecuteTestCase{
				RunResponses: []*command.FakeRun{{
					Stdout: []string{
						successOutput("p1", 12.34),
						noTestLine("p2"),
						noTestLine("p3"),
						successOutput("p4", 98.76),
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
							successOutput("p1", 12.34),
						},
						"p2": {
							noTestFiles,
							0.0,
							noTestLine("p2"),
						},
						"p3": {
							noTestFiles,
							0.0,
							noTestLine("p3"),
						},
						"p4": {
							testSuccess,
							98.76,
							successOutput("p4", 98.76),
						},
					},
				}},
			},
		},
		{
			name: "Handles multiple pacakges successes with package count flag",
			etc: &command.ExecuteTestCase{
				Args: []string{
					"--package-count",
					"4",
				},
				RunResponses: []*command.FakeRun{{
					Stdout: []string{
						successOutput("p1", 12.34),
						noTestLine("p2"),
						noTestLine("p3"),
						successOutput("p4", 98.76),
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
					packageCountFlag.Name(): 4,
					pathArgs.Name():         []string{"."},
					minCoverageFlag.Name():  0.0,
					"COVERAGE": map[string]*packageResult{
						"p1": {
							testSuccess,
							12.34,
							successOutput("p1", 12.34),
						},
						"p2": {
							noTestFiles,
							0.0,
							noTestLine("p2"),
						},
						"p3": {
							noTestFiles,
							0.0,
							noTestLine("p3"),
						},
						"p4": {
							testSuccess,
							98.76,
							successOutput("p4", 98.76),
						},
					},
				}},
			},
		},
		{
			name: "Fails if incorrect package count flag",
			etc: &command.ExecuteTestCase{
				Args: []string{
					"--package-count",
					"5",
				},
				RunResponses: []*command.FakeRun{{
					Stdout: []string{
						successOutput("p1", 12.34),
						noTestLine("p2"),
						noTestLine("p3"),
						successOutput("p4", 98.76),
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
				WantStderr: strings.Join([]string{
					"Expected 5 packages, got 4:",
					"p1",
					"p2",
					"p3",
					"p4",
					"",
				}, "\n"),
				WantErr: fmt.Errorf("Expected 5 packages, got 4:\np1\np2\np3\np4"),
				WantData: &command.Data{Values: map[string]interface{}{
					packageCountFlag.Name(): 5,
					pathArgs.Name():         []string{"."},
					minCoverageFlag.Name():  0.0,
				}},
			},
		},
		{
			name: "Handles multiple pacakges with errors",
			etc: &command.ExecuteTestCase{
				RunResponses: []*command.FakeRun{{
					Stdout: []string{
						successOutput("p1", 12.34),
						noTestLine("p2"),
						failLine("p3"),
						noTestLine("p4"),
						failLine("p5"),
						successOutput("p6", 98.76),
					},
				}},
				WantStderr: "Tests failed for package: p3\nTests failed for package: p5\n",
				WantErr:    fmt.Errorf("Tests failed for package: p5"),
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
							successOutput("p1", 12.34),
						},
						"p2": {
							noTestFiles,
							0.0,
							noTestLine("p2"),
						},
						"p3": {
							testFailure,
							0.0,
							failLine("p3"),
						},
						"p4": {
							noTestFiles,
							0.0,
							noTestLine("p4"),
						},
						"p5": {
							testFailure,
							0.0,
							failLine("p5"),
						},
						"p6": {
							testSuccess,
							98.76,
							successOutput("p6", 98.76),
						},
					},
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

			test.etc.Node = CLI().Node()
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
				Want: &command.Autocompletion{
					Suggestions: []string{
						filepath.FromSlash(".git/"),
						filepath.FromSlash("cmd/"),
						filepath.FromSlash("testdata/"),
						filepath.FromSlash("testpkg/"),
						" ",
					},
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
				Want: &command.Autocompletion{
					Suggestions: []string{
						"Autocomplete",
						"Execute",
						"Metadata",
					},
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
				Want: &command.Autocompletion{
					Suggestions: []string{
						"Autocomplete",
						"Execute",
						"Metadata",
						"Other",
						"That",
						"This",
					},
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
				Want: &command.Autocompletion{
					Suggestions: []string{
						"Autocomplete",
					},
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
				Want: &command.Autocompletion{
					Suggestions: []string{
						"This",
					},
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

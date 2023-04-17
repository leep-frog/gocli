package gocli

import (
	"fmt"
	"strings"
	"testing"

	"github.com/leep-frog/command"
)

func TestExecute(t *testing.T) {
	for _, test := range []struct {
		name string
		etc  *command.ExecuteTestCase
	}{
		{
			name: "Fails if shell command fails",
			etc: &command.ExecuteTestCase{
				RunResponses: []*command.FakeRun{{
					Err: fmt.Errorf("bad news bears"),
				}},
				WantStderr: "failed to execute shell command: bad news bears\n",
				WantErr:    fmt.Errorf("failed to execute shell command: bad news bears"),
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						".",
					},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 0.0,
				}},
			},
		},
		{
			name: "Fails if regex doesn't match",
			etc: &command.ExecuteTestCase{
				RunResponses: []*command.FakeRun{{
					Stdout: []string{
						`some random line`,
					},
				}},
				WantStdout: "some random line",
				WantStderr: "failed to parse coverage from line \"some random line\"\n",
				WantErr:    fmt.Errorf(`failed to parse coverage from line "some random line"`),
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						".",
					},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 0.0,
				}},
			},
		},
		{
			name: "Runs default test coverage",
			etc: &command.ExecuteTestCase{
				RunResponses: []*command.FakeRun{{
					Stdout: []string{
						`ok      github.com/some/package      1.234s  coverage: 98.7% of statements`,
					},
				}},
				WantStdout: "ok      github.com/some/package      1.234s  coverage: 98.7% of statements",
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						".",
					},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 0.0,
				}},
			},
		},
		{
			name: "Runs default test coverage with timeout flag",
			etc: &command.ExecuteTestCase{
				Args: []string{"--timeout", "15"},
				RunResponses: []*command.FakeRun{{
					Stdout: []string{
						`ok      github.com/some/package      1.234s  coverage: 98.7% of statements`,
					},
				}},
				WantStdout: "ok      github.com/some/package      1.234s  coverage: 98.7% of statements",
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						"-timeout",
						"15s",
						".",
					},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 0.0,
					timeoutFlag.Name():     15,
				}},
			},
		},
		{
			name: "Runs default test coverage with timeout short flag",
			etc: &command.ExecuteTestCase{
				Args: []string{"-t", "500"},
				RunResponses: []*command.FakeRun{{
					Stdout: []string{
						`ok      github.com/some/package      1.234s  coverage: 98.7% of statements`,
					},
				}},
				WantStdout: "ok      github.com/some/package      1.234s  coverage: 98.7% of statements",
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						"-timeout",
						"500s",
						".",
					},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 0.0,
					timeoutFlag.Name():     500,
				}},
			},
		},
		{
			name: "Runs default test coverage with path arg provided",
			etc: &command.ExecuteTestCase{
				Args: []string{"./path1", "./path/2", "../p3"},
				RunResponses: []*command.FakeRun{{
					Stdout: []string{
						`ok      github.com/some/package1      1.234s  coverage: 98.7% of statements`,
						`ok      github.com/some/package2      1.234s  coverage: 98.7% of statements`,
						`ok      github.com/some/package3      1.234s  coverage: 98.7% of statements`,
					},
				}},
				WantStdout: strings.Join([]string{
					`ok      github.com/some/package1      1.234s  coverage: 98.7% of statements`,
					`ok      github.com/some/package2      1.234s  coverage: 98.7% of statements`,
					`ok      github.com/some/package3      1.234s  coverage: 98.7% of statements`,
				}, "\n"),
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						"./path1",
						"./path/2",
						"../p3",
					},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"./path1", "./path/2", "../p3"},
					minCoverageFlag.Name(): 0.0,
				}},
			},
		},
		{
			name: "Runs default test coverage if extra newlines",
			etc: &command.ExecuteTestCase{
				RunResponses: []*command.FakeRun{{
					Stdout: []string{
						`ok      github.com/some/package      1.234s  coverage: 98.7% of statements`,
						``,
						``,
					},
				}},
				WantStdout: "ok      github.com/some/package      1.234s  coverage: 98.7% of statements\n\n",
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						".",
					},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 0.0,
				}},
			},
		},
		{
			name: "Passes if coverage is high enough",
			etc: &command.ExecuteTestCase{
				Args: []string{"-m", "87.5"},
				RunResponses: []*command.FakeRun{{
					Stdout: []string{
						`ok      github.com/some/package      1.234s  coverage: 87.6% of statements`,
					},
				}},
				WantStdout: "ok      github.com/some/package      1.234s  coverage: 87.6% of statements",
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						".",
					},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 87.5,
				}},
			},
		},
		{
			name: "Fails if coverage isn't high enough",
			etc: &command.ExecuteTestCase{
				Args: []string{"-m", "87.8"},
				RunResponses: []*command.FakeRun{{
					Stdout: []string{
						`ok      github.com/some/package      1.234s  coverage: 87.6% of statements`,
					},
				}},
				WantStdout: "ok      github.com/some/package      1.234s  coverage: 87.6% of statements",
				WantStderr: "Coverage of package \"github.com/some/package\" (87.6%) must be at least 87.8%\n",
				WantErr:    fmt.Errorf(`Coverage of package "github.com/some/package" (87.6%%) must be at least 87.8%%`),
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						".",
					},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 87.8,
				}},
			},
		},
		{
			name: "Passes if coverage is high enough for multiple packages",
			etc: &command.ExecuteTestCase{
				Args: []string{"-m", "80"},
				RunResponses: []*command.FakeRun{{
					Stdout: []string{
						`ok      github.com/package/one      1.234s  coverage: 87.6% of statements`,
						`ok      github.com/package/two      12.34s  coverage: 97.6% of statements`,
						`?       github.com/package/three        [no test files]`,
						`?       github.com/package/four        [no test files]`,
						`ok      github.com/package/five      123.4s  coverage: 83.6% of statements`,
						`?       github.com/package/six        [no test files]`,
						`ok      github.com/package/seven      1234s  coverage: 81.6% of statements`,
					},
				}},
				WantStdout: strings.Join([]string{
					`ok      github.com/package/one      1.234s  coverage: 87.6% of statements`,
					`ok      github.com/package/two      12.34s  coverage: 97.6% of statements`,
					`?       github.com/package/three        [no test files]`,
					`?       github.com/package/four        [no test files]`,
					`ok      github.com/package/five      123.4s  coverage: 83.6% of statements`,
					`?       github.com/package/six        [no test files]`,
					`ok      github.com/package/seven      1234s  coverage: 81.6% of statements`,
				}, "\n"),
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						".",
					},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 80.0,
				}},
			},
		},
		{
			name: "Fails if coverage isn't high enough for one of multiple packages",
			etc: &command.ExecuteTestCase{
				Args: []string{"-m", "80"},
				RunResponses: []*command.FakeRun{{
					Stdout: []string{
						`ok      github.com/package/one      1.234s  coverage: 87.6% of statements`,
						`ok      github.com/package/two      12.34s  coverage: 97.6% of statements`,
						`?       github.com/package/three        [no test files]`,
						`?       github.com/package/four        [no test files]`,
						`ok      github.com/package/five      123.4s  coverage: 78.6% of statements`,
						`?       github.com/package/six        [no test files]`,
						`ok      github.com/package/seven      1234s  coverage: 81.6% of statements`,
					},
				}},
				WantStdout: strings.Join([]string{
					`ok      github.com/package/one      1.234s  coverage: 87.6% of statements`,
					`ok      github.com/package/two      12.34s  coverage: 97.6% of statements`,
					`?       github.com/package/three        [no test files]`,
					`?       github.com/package/four        [no test files]`,
					`ok      github.com/package/five      123.4s  coverage: 78.6% of statements`,
					`?       github.com/package/six        [no test files]`,
					`ok      github.com/package/seven      1234s  coverage: 81.6% of statements`,
				}, "\n"),
				WantStderr: "Coverage of package \"github.com/package/five\" (78.6%) must be at least 80.0%\n",
				WantErr:    fmt.Errorf(`Coverage of package "github.com/package/five" (78.6%%) must be at least 80.0%%`),
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						".",
					},
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 80.0,
				}},
			},
		},
		{
			name: "Fails if verbose and min coverage flags provided",
			etc: &command.ExecuteTestCase{
				Args:       []string{"-m", "87.8", "-v"},
				WantStderr: "Can't run verbose output with coverage checks\n",
				WantErr:    fmt.Errorf("Can't run verbose output with coverage checks"),
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 87.8,
					verboseFlag.Name():     true,
				}},
			},
		},
		{
			name: "Runs with verbose flag",
			etc: &command.ExecuteTestCase{
				Args: []string{"-v"},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 0.0,
					verboseFlag.Name():     true,
				}},
				RunResponses: []*command.FakeRun{{
					Stdout: []string{
						`ok      github.com/package/one      1.234s  coverage: 87.6% of statements`,
						``,
					},
				}},
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						".",
						"-v",
					},
				}},
				WantStdout: "ok      github.com/package/one      1.234s  coverage: 87.6% of statements\n",
			},
		},
		{
			name: "Runs with verbose flag and extra output",
			etc: &command.ExecuteTestCase{
				Args: []string{"-v"},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 0.0,
					verboseFlag.Name():     true,
				}},
				RunResponses: []*command.FakeRun{{
					Stdout: []string{
						`=== RUN   TestQMKExecution/load_bindings_in_basic_mode#01`,
						`--- PASS: TestQMKExecution (0.00s)`,
						`    --- PASS: TestQMKExecution/qmk_toggle_fails_if_env_variable_is_unset (0.00s)`,
						`and some lines that`,
						` are printed!`,
						`ok      github.com/package/one      1.234s  coverage: 87.6% of statements`,
						``,
					},
				}},
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						".",
						"-v",
					},
				}},
				WantStdout: strings.Join([]string{
					`=== RUN   TestQMKExecution/load_bindings_in_basic_mode#01`,
					`--- PASS: TestQMKExecution (0.00s)`,
					`    --- PASS: TestQMKExecution/qmk_toggle_fails_if_env_variable_is_unset (0.00s)`,
					`and some lines that`,
					` are printed!`,
					`ok      github.com/package/one      1.234s  coverage: 87.6% of statements`,
					``,
				}, "\n"),
			},
		},
		// funcFilterFlag tests
		{
			name: "Adds func filter arguments",
			etc: &command.ExecuteTestCase{
				Args: []string{"-f", "Un", "Deux"},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 0.0,
					funcFilterFlag.Name():  []string{"Un", "Deux"},
				}},
				RunResponses: []*command.FakeRun{{
					Stdout: []string{
						`ok      github.com/package/one      1.234s  coverage: 87.6% of statements`,
						``,
					},
				}},
				WantRunContents: []*command.RunContents{{
					Name: "go",
					Args: []string{
						"test",
						".",
						"-run",
						`(Un|Deux)`,
					},
				}},
				WantStdout: strings.Join([]string{
					`ok      github.com/package/one      1.234s  coverage: 87.6% of statements`,
					``,
				}, "\n"),
			},
		},
		/* Useful for commenting out tests. */
	} {
		t.Run(test.name, func(t *testing.T) {
			test.etc.Node = (&goCLI{}).Node()
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

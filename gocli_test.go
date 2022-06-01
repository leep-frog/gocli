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
			name: "Fails if bash command fails",
			etc: &command.ExecuteTestCase{
				RunResponses: []*command.FakeRun{{
					Err: fmt.Errorf("bad news bears"),
				}},
				WantStderr: []string{
					"failed to execute bash command: bad news bears",
				},
				WantErr: fmt.Errorf("failed to execute bash command: bad news bears"),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"go test . -coverprofile=$(mktemp)",
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
				WantStdout: []string{
					`some random line`,
				},
				WantStderr: []string{
					`failed to parse coverage from line "some random line"`,
				},
				WantErr: fmt.Errorf(`failed to parse coverage from line "some random line"`),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"go test . -coverprofile=$(mktemp)",
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
				WantStdout: []string{
					`ok      github.com/some/package      1.234s  coverage: 98.7% of statements`,
				},
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"go test . -coverprofile=$(mktemp)",
				}},
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 0.0,
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
				WantStdout: []string{strings.Join([]string{
					`ok      github.com/some/package1      1.234s  coverage: 98.7% of statements`,
					`ok      github.com/some/package2      1.234s  coverage: 98.7% of statements`,
					`ok      github.com/some/package3      1.234s  coverage: 98.7% of statements`,
				}, "\n")},
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"go test ./path1 ./path/2 ../p3 -coverprofile=$(mktemp)",
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
				WantStdout: []string{
					"ok      github.com/some/package      1.234s  coverage: 98.7% of statements\n\n",
				},
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"go test . -coverprofile=$(mktemp)",
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
				WantStdout: []string{
					"ok      github.com/some/package      1.234s  coverage: 87.6% of statements",
				},
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"go test . -coverprofile=$(mktemp)",
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
				WantStdout: []string{
					"ok      github.com/some/package      1.234s  coverage: 87.6% of statements",
				},
				WantStderr: []string{
					`Coverage of package "github.com/some/package" (87.6%) must be at least 87.8%`,
				},
				WantErr: fmt.Errorf(`Coverage of package "github.com/some/package" (87.6%%) must be at least 87.8%%`),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"go test . -coverprofile=$(mktemp)",
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
				WantStdout: []string{strings.Join([]string{
					`ok      github.com/package/one      1.234s  coverage: 87.6% of statements`,
					`ok      github.com/package/two      12.34s  coverage: 97.6% of statements`,
					`?       github.com/package/three        [no test files]`,
					`?       github.com/package/four        [no test files]`,
					`ok      github.com/package/five      123.4s  coverage: 83.6% of statements`,
					`?       github.com/package/six        [no test files]`,
					`ok      github.com/package/seven      1234s  coverage: 81.6% of statements`,
				}, "\n")},
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"go test . -coverprofile=$(mktemp)",
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
				WantStdout: []string{strings.Join([]string{
					`ok      github.com/package/one      1.234s  coverage: 87.6% of statements`,
					`ok      github.com/package/two      12.34s  coverage: 97.6% of statements`,
					`?       github.com/package/three        [no test files]`,
					`?       github.com/package/four        [no test files]`,
					`ok      github.com/package/five      123.4s  coverage: 78.6% of statements`,
					`?       github.com/package/six        [no test files]`,
					`ok      github.com/package/seven      1234s  coverage: 81.6% of statements`,
				}, "\n")},
				WantStderr: []string{
					`Coverage of package "github.com/package/five" (78.6%) must be at least 80.0%`,
				},
				WantErr: fmt.Errorf(`Coverage of package "github.com/package/five" (78.6%%) must be at least 80.0%%`),
				WantRunContents: [][]string{{
					"set -e",
					"set -o pipefail",
					"go test . -coverprofile=$(mktemp)",
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
				Args: []string{"-m", "87.8", "-v"},
				WantStderr: []string{
					"Can't run verbose output with coverage checks",
				},
				WantErr: fmt.Errorf("Can't run verbose output with coverage checks"),
				WantData: &command.Data{Values: map[string]interface{}{
					pathArgs.Name():        []string{"."},
					minCoverageFlag.Name(): 87.8,
					verboseFlag.Name():     true,
				}},
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

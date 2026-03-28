package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// TestResult holds the results of running a test binary
type TestResult struct {
	Name      string
	Success   bool
	ExitCode  int
	Stdout    string
	Stderr    string
	Duration  time.Duration
	Timeout   bool
	Syscalls  []string
}

// TestConfig holds configuration for running tests
type TestConfig struct {
	Timeout    time.Duration
	CaptureAll bool
	Verbose    bool
}

// Runner executes Plan 9 binaries under 9xe
type Runner struct {
	xePath     string
	testBinDir string
	config     TestConfig
	results    []TestResult
}

// NewRunner creates a new test runner
func NewRunner(xePath string, testBinDir string) *Runner {
	return &Runner{
		xePath:     xePath,
		testBinDir: testBinDir,
		config: TestConfig{
			Timeout:    10 * time.Second,
			CaptureAll: true,
			Verbose:    false,
		},
		results: make([]TestResult, 0),
	}
}

// SetTimeout sets the timeout for test execution
func (r *Runner) SetTimeout(timeout time.Duration) {
	r.config.Timeout = timeout
}

// SetVerbose enables verbose output
func (r *Runner) SetVerbose(verbose bool) {
	r.config.Verbose = verbose
}

// TestBinary runs a single Plan 9 binary and captures the result
func (r *Runner) TestBinary(name string, args ...string) *TestResult {
	// Build the command
	binaryPath := fmt.Sprintf("%s/%s", r.testBinDir, name)
	cmdArgs := append([]string{binaryPath}, args...)

	cmd := exec.Command(r.xePath, cmdArgs...)

	// Setup pipes for stdout and stderr
	var stdoutBuf, stderrBuf bytes.Buffer
	if r.config.CaptureAll {
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stderrBuf
	}

	// Start the command
	startTime := time.Now()
	if err := cmd.Start(); err != nil {
		return &TestResult{
			Name:     name,
			Success:  false,
			ExitCode: -1,
			Stdout:   "",
			Stderr:   fmt.Sprintf("Failed to start: %v", err),
			Duration: 0,
			Timeout:  false,
		}
	}

	// Wait for command with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(r.config.Timeout):
		// Timeout - kill the process
		cmd.Process.Kill()
		return &TestResult{
			Name:     name,
			Success:  false,
			ExitCode: -1,
			Stdout:   stdoutBuf.String(),
			Stderr:   stderrBuf.String(),
			Duration: time.Since(startTime),
			Timeout:  true,
		}
	case err := <-done:
		// Command completed
		exitCode := 0
		if err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				exitCode = exitError.ExitCode()
			}
		}

		return &TestResult{
			Name:     name,
			Success:  err == nil,
			ExitCode: exitCode,
			Stdout:   stdoutBuf.String(),
			Stderr:   stderrBuf.String(),
			Duration: time.Since(startTime),
			Timeout:  false,
		}
	}
}

// TestBinaryWithInput runs a binary with stdin input
func (r *Runner) TestBinaryWithInput(name string, input string, args ...string) *TestResult {
	binaryPath := fmt.Sprintf("%s/%s", r.testBinDir, name)
	cmdArgs := append([]string{binaryPath}, args...)

	cmd := exec.Command(r.xePath, cmdArgs...)

	var stdoutBuf, stderrBuf bytes.Buffer
	if r.config.CaptureAll {
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stderrBuf
	}

	// Set stdin
	cmd.Stdin = strings.NewReader(input)

	startTime := time.Now()
	if err := cmd.Start(); err != nil {
		return &TestResult{
			Name:     name,
			Success:  false,
			ExitCode: -1,
			Stderr:   fmt.Sprintf("Failed to start: %v", err),
			Duration: 0,
		}
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(r.config.Timeout):
		cmd.Process.Kill()
		return &TestResult{
			Name:     name,
			Success:  false,
			ExitCode: -1,
			Stdout:   stdoutBuf.String(),
			Stderr:   stderrBuf.String(),
			Duration: time.Since(startTime),
			Timeout:  true,
		}
	case err := <-done:
		exitCode := 0
		if err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				exitCode = exitError.ExitCode()
			}
		}

		return &TestResult{
			Name:     name,
			Success:  err == nil,
			ExitCode: exitCode,
			Stdout:   stdoutBuf.String(),
			Stderr:   stderrBuf.String(),
			Duration: time.Since(startTime),
			Timeout:  false,
		}
	}
}

// RunTests runs multiple tests from a test definition
func (r *Runner) RunTests(tests []TestCase) []TestResult {
	results := make([]TestResult, 0, len(tests))

	for _, test := range tests {
		if r.config.Verbose {
			fmt.Printf("Running: %s\n", test.Name)
		}

		var result *TestResult
		if test.Input != "" {
			result = r.TestBinaryWithInput(test.Binary, test.Input, test.Args...)
		} else {
			result = r.TestBinary(test.Binary, test.Args...)
		}

		// Check if output matches expected
		if test.ExpectedOutput != "" {
			result.Success = result.Success &&
				strings.Contains(result.Stdout, test.ExpectedOutput)
		}

		// Check if exit code matches
		if test.ExpectedExitCode >= 0 {
			result.Success = result.Success &&
				(result.ExitCode == test.ExpectedExitCode)
		}

		results = append(results, *result)

		if r.config.Verbose {
			status := "✓"
			if !result.Success {
				status = "✗"
			}
			fmt.Printf("  %s %s (exit: %d, time: %s)\n",
				status, test.Name, result.ExitCode, result.Duration)
		}
	}

	r.results = results
	return results
}

// TestCase defines a single test case
type TestCase struct {
	Name             string
	Binary           string
	Args             []string
	Input            string
	ExpectedOutput   string
	ExpectedExitCode int
}

// PrintSummary prints a summary of test results
func (r *Runner) PrintSummary() {
	if len(r.results) == 0 {
		fmt.Println("No tests run")
		return
	}

	passed := 0
	failed := 0
	total := len(r.results)

	for _, result := range r.results {
		if result.Success {
			passed++
		} else {
			failed++
		}
	}

	fmt.Printf("\n=== Test Summary ===\n")
	fmt.Printf("Total: %d, Passed: %d, Failed: %d\n", total, passed, failed)

	if failed > 0 {
		fmt.Printf("\nFailed tests:\n")
		for _, result := range r.results {
			if !result.Success {
				fmt.Printf("  - %s (exit: %d)\n", result.Name, result.ExitCode)
				if result.Timeout {
					fmt.Printf("    TIMEOUT\n")
				}
				if result.Stderr != "" {
					fmt.Printf("    stderr: %s\n", strings.TrimSpace(result.Stderr))
				}
			}
		}
	}
}

// SaveResults saves test results to a file
func (r *Runner) SaveResults(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, result := range r.results {
		status := "PASS"
		if !result.Success {
			status = "FAIL"
		}

		fmt.Fprintf(f, "[%s] %s\n", status, result.Name)
		fmt.Fprintf(f, "  Exit Code: %d\n", result.ExitCode)
		fmt.Fprintf(f, "  Duration: %s\n", result.Duration)
		if result.Timeout {
			fmt.Fprintf(f, "  TIMEOUT\n")
		}
		if result.Stdout != "" {
			fmt.Fprintf(f, "  Stdout: %q\n", strings.TrimSpace(result.Stdout))
		}
		if result.Stderr != "" {
			fmt.Fprintf(f, "  Stderr: %q\n", strings.TrimSpace(result.Stderr))
		}
		fmt.Fprintln(f)
	}

	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: runner <9xe_path> [test_binary...]")
		fmt.Println("   or: runner <9xe_path> --all")
		os.Exit(1)
	}

	xePath := os.Args[1]
	testBinDir := "/mnt/storage/Projects/TaijiOS/9xe/testbin"

	runner := NewRunner(xePath, testBinDir)
	runner.SetVerbose(true)
	runner.SetTimeout(5 * time.Second)

	// Define basic tests
	tests := []TestCase{
		{
			Name:             "echo_basic",
			Binary:           "echo",
			Args:             []string{"hello", "world"},
			ExpectedOutput:   "hello world",
			ExpectedExitCode: 0,
		},
		{
			Name:             "cat_self",
			Binary:           "cat",
			Args:             []string{"runner.go"},
			ExpectedExitCode: 0,
		},
		{
			Name:             "pwd",
			Binary:           "pwd",
			ExpectedExitCode: 0,
		},
		{
			Name:             "date",
			Binary:           "date",
			ExpectedExitCode: 0,
		},
	}

	// If specific binaries requested, test those
	if len(os.Args) > 2 && os.Args[2] != "--all" {
		tests = []TestCase{}
		for _, bin := range os.Args[2:] {
			tests = append(tests, TestCase{
				Name:   bin,
				Binary: bin,
				Args:   []string{},
			})
		}
	}

	// Run tests
	results := runner.RunTests(tests)

	// Print summary
	runner.PrintSummary()

	// Save results
	if err := runner.SaveResults("test_results.txt"); err != nil {
		fmt.Printf("Failed to save results: %v\n", err)
	}

	// Exit with error if any tests failed
	for _, result := range results {
		if !result.Success {
			os.Exit(1)
		}
	}
}

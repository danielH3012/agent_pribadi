package tools

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type BashResult struct {
	Command  string `json:"command"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
	Success  bool   `json:"success"`
}

// Bash - Execute shell command
func Bash(command string) (*BashResult, error) {
	// 1. Validate command not empty
	if strings.TrimSpace(command) == "" {
		return nil, fmt.Errorf("command cannot be empty")
	}

	// 2. Ask user permission
	if err := AskPermission("bash", fmt.Sprintf("Command: %s", command)); err != nil {
		return nil, err
	}

	// 3. Create command
	cmd := exec.Command("bash", "-c", command)

	// 4. Set environment
	cmd.Env = os.Environ()

	// 5. Capture output
	stdout, err := cmd.Output()

	// 6. Get exit code (even if error)
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	// 7. Build result
	result := &BashResult{
		Command:  command,
		Stdout:   string(stdout),
		ExitCode: exitCode,
		Success:  exitCode == 0,
	}

	return result, nil
}

// BashWithStderr - Capture both stdout + stderr
func BashWithStderr(command string) (*BashResult, error) {
	if strings.TrimSpace(command) == "" {
		return nil, fmt.Errorf("command cannot be empty")
	}

	// Ask user permission
	if err := AskPermission("bash", fmt.Sprintf("Command (capture stdout + stderr): %s", command)); err != nil {
		return nil, err
	}

	cmd := exec.Command("bash", "-c", command)
	cmd.Env = os.Environ()

	// Capture both stdout and stderr separately
	stdout, _ := cmd.Output()

	// For stderr, need CombinedOutput or separate capture
	cmd2 := exec.Command("bash", "-c", command)
	cmd2.Env = os.Environ()
	combinedOutput, err := cmd2.CombinedOutput()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	// Stderr = combined - stdout
	stderrStr := strings.TrimPrefix(string(combinedOutput), string(stdout))

	result := &BashResult{
		Command:  command,
		Stdout:   string(stdout),
		Stderr:   stderrStr,
		ExitCode: exitCode,
		Success:  exitCode == 0,
	}

	return result, nil
}

// Better approach - using Cmd.StdoutPipe() and StderrPipe()
func BashAdvanced(command string) (*BashResult, error) {
	if strings.TrimSpace(command) == "" {
		return nil, fmt.Errorf("command cannot be empty")
	}

	// Ask user permission
	if err := AskPermission("bash", fmt.Sprintf("Command (piped streaming): %s", command)); err != nil {
		return nil, err
	}

	cmd := exec.Command("bash", "-c", command)
	cmd.Env = os.Environ()

	// Create pipes for stdout and stderr
	stdoutPipe, _ := cmd.StdoutPipe()
	stderrPipe, _ := cmd.StderrPipe()

	// Start command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start error: %w", err)
	}

	// Read stdout
	stdoutData, _ := io.ReadAll(stdoutPipe)
	stderrData, _ := io.ReadAll(stderrPipe)

	// Wait for completion
	err := cmd.Wait()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	result := &BashResult{
		Command:  command,
		Stdout:   string(stdoutData),
		Stderr:   string(stderrData),
		ExitCode: exitCode,
		Success:  exitCode == 0,
	}

	return result, nil
}

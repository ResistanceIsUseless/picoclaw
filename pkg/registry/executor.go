package registry

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// GenericExecutor executes any tool with flexible argument handling
type GenericExecutor struct {
	workingDir string
}

// NewGenericExecutor creates a new generic tool executor
func NewGenericExecutor(workingDir string) *GenericExecutor {
	return &GenericExecutor{
		workingDir: workingDir,
	}
}

// Execute runs a tool with the given arguments
// This is the unified execution path for all tools
func (e *GenericExecutor) Execute(ctx context.Context, toolName string, args map[string]interface{}) ([]byte, error) {
	// Check for mock tools first (for testing)
	if len(toolName) > 5 && toolName[:5] == "mock_" {
		return ExecuteMockTool(ctx, toolName, args)
	}

	// Handle special tools with custom execution logic
	switch toolName {
	case "shell":
		return e.executeShell(ctx, args)
	case "subfinder":
		return e.executeSubfinder(ctx, args)
	case "amass":
		return e.executeAmass(ctx, args)
	case "nmap":
		return e.executeNmap(ctx, args)
	case "httpx":
		return e.executeHttpx(ctx, args)
	case "nuclei":
		return e.executeNuclei(ctx, args)
	default:
		// Generic tool execution for auto-discovered tools
		return e.executeGenericTool(ctx, toolName, args)
	}
}

// executeShell executes shell commands with pipe support
func (e *GenericExecutor) executeShell(ctx context.Context, args map[string]interface{}) ([]byte, error) {
	command, ok := args["command"].(string)
	if !ok {
		return nil, fmt.Errorf("shell requires 'command' parameter")
	}

	workingDir := e.workingDir
	if wd, ok := args["working_dir"].(string); ok {
		workingDir = wd
	}

	// Execute via bash to support pipes, redirects, etc.
	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Dir = workingDir

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("shell command failed: %w (stderr: %s)", err, stderr.String())
	}

	return stdout.Bytes(), nil
}

// executeSubfinder runs subfinder with specific flags
func (e *GenericExecutor) executeSubfinder(ctx context.Context, args map[string]interface{}) ([]byte, error) {
	domain, ok := args["domain"].(string)
	if !ok {
		return nil, fmt.Errorf("subfinder requires 'domain' parameter")
	}
	cmd := exec.CommandContext(ctx, "subfinder", "-d", domain, "-silent")
	return cmd.Output()
}

// executeAmass runs amass with specific flags
func (e *GenericExecutor) executeAmass(ctx context.Context, args map[string]interface{}) ([]byte, error) {
	domain, ok := args["domain"].(string)
	if !ok {
		return nil, fmt.Errorf("amass requires 'domain' parameter")
	}
	cmd := exec.CommandContext(ctx, "amass", "enum", "-passive", "-d", domain)
	return cmd.Output()
}

// executeNmap runs nmap with XML output
func (e *GenericExecutor) executeNmap(ctx context.Context, args map[string]interface{}) ([]byte, error) {
	target, ok := args["target"].(string)
	if !ok {
		return nil, fmt.Errorf("nmap requires 'target' parameter")
	}
	ports := "top-1000"
	if p, ok := args["ports"].(string); ok {
		ports = p
	}
	cmd := exec.CommandContext(ctx, "nmap", "-p", ports, "-oX", "-", target)
	return cmd.Output()
}

// executeHttpx runs httpx with JSON output
func (e *GenericExecutor) executeHttpx(ctx context.Context, args map[string]interface{}) ([]byte, error) {
	targets, ok := args["targets"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("httpx requires 'targets' parameter as array")
	}

	var targetList bytes.Buffer
	for _, t := range targets {
		if target, ok := t.(string); ok {
			targetList.WriteString(target + "\n")
		}
	}

	cmd := exec.CommandContext(ctx, "httpx", "-json", "-silent")
	cmd.Stdin = &targetList

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("httpx execution failed: %w (stderr: %s)", err, stderr.String())
	}

	return stdout.Bytes(), nil
}

// executeNuclei runs nuclei with JSON output
func (e *GenericExecutor) executeNuclei(ctx context.Context, args map[string]interface{}) ([]byte, error) {
	target, ok := args["target"].(string)
	if !ok {
		return nil, fmt.Errorf("nuclei requires 'target' parameter")
	}
	severity := "critical,high"
	if s, ok := args["severity"].(string); ok {
		severity = s
	}
	cmd := exec.CommandContext(ctx, "nuclei", "-u", target, "-severity", severity, "-json", "-silent")
	return cmd.Output()
}

// executeGenericTool executes any discovered tool with flexible arguments
func (e *GenericExecutor) executeGenericTool(ctx context.Context, toolName string, args map[string]interface{}) ([]byte, error) {
	// Find tool path
	toolPath, err := GetToolPath(toolName)
	if err != nil {
		return nil, err
	}

	// Extract arguments
	var cmdArgs []string

	// Handle "args" array parameter
	if argsArray, ok := args["args"].([]interface{}); ok {
		for _, arg := range argsArray {
			if argStr, ok := arg.(string); ok {
				cmdArgs = append(cmdArgs, argStr)
			}
		}
	}

	// Handle direct string arguments (convert map to args)
	// Example: {"domain": "example.com"} → ["-d", "example.com"]
	for key, value := range args {
		if key == "args" {
			continue // Already handled above
		}

		// Add flag
		cmdArgs = append(cmdArgs, "-"+key)

		// Add value
		switch v := value.(type) {
		case string:
			cmdArgs = append(cmdArgs, v)
		case []interface{}:
			for _, item := range v {
				if str, ok := item.(string); ok {
					cmdArgs = append(cmdArgs, str)
				}
			}
		case []string:
			cmdArgs = append(cmdArgs, v...)
		default:
			cmdArgs = append(cmdArgs, fmt.Sprintf("%v", v))
		}
	}

	// Execute tool
	cmd := exec.CommandContext(ctx, toolPath, cmdArgs...)
	cmd.Dir = e.workingDir

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s execution failed: %w (stderr: %s)", toolName, err, stderr.String())
	}

	return stdout.Bytes(), nil
}

// ExecuteToolGeneric is the unified entry point for tool execution
// It replaces the old ExecuteTool function with hardcoded switch statement
func ExecuteToolGeneric(ctx context.Context, toolName string, args map[string]interface{}, workingDir string) ([]byte, error) {
	executor := NewGenericExecutor(workingDir)
	return executor.Execute(ctx, toolName, args)
}

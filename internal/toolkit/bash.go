package toolkit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// BashTool Bash 工具
type BashTool struct {
	WorkingDir string
	Timeout    time.Duration
}

// NewBashTool 创建 Bash 工具
func NewBashTool() *BashTool {
	wd, _ := os.Getwd()
	return &BashTool{
		WorkingDir: wd,
		Timeout:    300 * time.Second,
	}
}

// BashInput Bash 输入
type BashInput struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

// Execute 执行 Bash 命令
func (t *BashTool) Execute(input json.RawMessage) (string, error) {
	var inp BashInput
	if err := json.Unmarshal(input, &inp); err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	if inp.Command == "" {
		return "", fmt.Errorf("command is required")
	}

	ctx := context.Background()
	if inp.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(inp.Timeout)*time.Second)
		defer cancel()
	} else {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.Timeout)
		defer cancel()
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", inp.Command)
	} else {
		cmd = exec.CommandContext(ctx, "bash", "-c", inp.Command)
	}

	cmd.Dir = t.WorkingDir
	cmd.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := strings.TrimSpace(stdout.String())
	if stderr.Len() > 0 {
		if result != "" {
			result += "\n"
		}
		result += "stderr: " + strings.TrimSpace(stderr.String())
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Sprintf("Command failed with exit code %d\n%s", exitErr.ExitCode(), result), nil
		}
		return fmt.Sprintf("Command execution error: %v\n%s", err, result), nil
	}

	if result == "" {
		result = "(empty output)"
	}

	return result, nil
}

// GetDescription 获取工具描述
func (t *BashTool) GetDescription() string {
	return "Execute bash commands in the current working directory"
}

// GetInputSchema 获取输入 Schema
func (t *BashTool) GetInputSchema() string {
	return `{
		"type": "object",
		"properties": {
			"command": {
				"type": "string",
				"description": "The bash command to execute"
			},
			"timeout": {
				"type": "integer",
				"description": "Command timeout in seconds"
			}
		},
		"required": ["command"]
	}`
}

package hooks

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

// HookType Hook 类型
type HookType string

const (
	HookPreToolUse  HookType = "preToolUse"
	HookPostToolUse HookType = "postToolUse"
)

// HookEvent Hook 事件
type HookEvent struct {
	Type      HookType               `json:"type"`
	ToolName  string                 `json:"toolName"`
	Input     string                 `json:"input"`
	Output    string                 `json:"output,omitempty"`
	IsError   bool                   `json:"isError,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// HookResult Hook 执行结果
type HookResult struct {
	Allowed        bool     `json:"allowed"`
	Messages       []string `json:"messages,omitempty"`
	ModifiedInput  string   `json:"modifiedInput,omitempty"`
	ModifiedOutput string   `json:"modifiedOutput,omitempty"`
	Error          string   `json:"error,omitempty"`
}

// HookRunner Hook 执行器
type HookRunner struct {
	preToolUseHooks  []string
	postToolUseHooks []string
	timeout          time.Duration
}

// NewHookRunner 创建 Hook 执行器
func NewHookRunner(preHooks, postHooks []string, timeout time.Duration) *HookRunner {
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &HookRunner{
		preToolUseHooks:  preHooks,
		postToolUseHooks: postHooks,
		timeout:          timeout,
	}
}

// RunPreToolUse 运行 PreToolUse Hook
func (r *HookRunner) RunPreToolUse(toolName, input string) HookResult {
	if len(r.preToolUseHooks) == 0 {
		return HookResult{Allowed: true}
	}

	event := HookEvent{
		Type:      HookPreToolUse,
		ToolName:  toolName,
		Input:     input,
		Timestamp: time.Now(),
	}

	return r.runHooks(r.preToolUseHooks, event)
}

// RunPostToolUse 运行 PostToolUse Hook
func (r *HookRunner) RunPostToolUse(toolName, input, output string, isError bool) HookResult {
	if len(r.postToolUseHooks) == 0 {
		return HookResult{Allowed: true}
	}

	event := HookEvent{
		Type:      HookPostToolUse,
		ToolName:  toolName,
		Input:     input,
		Output:    output,
		IsError:   isError,
		Timestamp: time.Now(),
	}

	return r.runHooks(r.postToolUseHooks, event)
}

func (r *HookRunner) runHooks(hooks []string, event HookEvent) HookResult {
	var allMessages []string
	allowed := true

	eventJSON, _ := json.Marshal(event)

	for _, hookScript := range hooks {
		result := r.executeHook(hookScript, eventJSON)

		if !result.Allowed {
			allowed = false
		}
		allMessages = append(allMessages, result.Messages...)

		if result.Error != "" {
			allMessages = append(allMessages, fmt.Sprintf("Hook error: %s", result.Error))
		}
	}

	return HookResult{
		Allowed:  allowed,
		Messages: allMessages,
	}
}

func (r *HookRunner) executeHook(script string, eventJSON []byte) HookResult {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", script)
	} else {
		cmd = exec.CommandContext(ctx, "bash", "-c", script)
	}

	cmd.Env = append(os.Environ(),
		fmt.Sprintf("HOOK_EVENT=%s", string(eventJSON)),
		fmt.Sprintf("HOOK_TYPE=%s", eventJSON[0:50]),
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := HookResult{
		Allowed: true,
	}

	output := strings.TrimSpace(stdout.String())
	if output != "" {
		result.Messages = append(result.Messages, output)
	}

	if stderr.Len() > 0 {
		result.Messages = append(result.Messages, "stderr: "+strings.TrimSpace(stderr.String()))
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// 退出码 2 表示拒绝
			if exitErr.ExitCode() == 2 {
				result.Allowed = false
				result.Messages = append(result.Messages, "Hook denied tool execution")
			} else {
				result.Error = fmt.Sprintf("Hook failed with exit code %d", exitErr.ExitCode())
			}
		} else {
			result.Error = err.Error()
		}
	}

	return result
}

// HasHooks 检查是否有 Hook
func (r *HookRunner) HasHooks() bool {
	return len(r.preToolUseHooks) > 0 || len(r.postToolUseHooks) > 0
}

// HookConfig Hook 配置
type HookConfig struct {
	PreToolUse  []string `json:"preToolUse,omitempty"`
	PostToolUse []string `json:"postToolUse,omitempty"`
	Timeout     int      `json:"timeout,omitempty"`
}

// LoadHookConfig 加载 Hook 配置
func LoadHookConfig(path string) (*HookConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &HookConfig{}, nil
		}
		return nil, err
	}

	var config HookConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// CreateHookRunnerFromConfig 从配置创建 HookRunner
func CreateHookRunnerFromConfig(config *HookConfig) *HookRunner {
	timeout := time.Duration(config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return NewHookRunner(config.PreToolUse, config.PostToolUse, timeout)
}

package permissions

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// PermissionMode 权限模式
type PermissionMode int

const (
	ReadOnly PermissionMode = iota
	WorkspaceWrite
	DangerFullAccess
)

// ParsePermissionMode 解析权限模式
func ParsePermissionMode(s string) (PermissionMode, error) {
	switch strings.ToLower(s) {
	case "readonly", "read-only":
		return ReadOnly, nil
	case "workspacewrite", "workspace-write":
		return WorkspaceWrite, nil
	case "dangerfullaccess", "danger-full-access":
		return DangerFullAccess, nil
	default:
		return 0, fmt.Errorf("unknown permission mode: %s", s)
	}
}

// String 返回字符串表示
func (p PermissionMode) String() string {
	switch p {
	case ReadOnly:
		return "read-only"
	case WorkspaceWrite:
		return "workspace-write"
	case DangerFullAccess:
		return "danger-full-access"
	default:
		return "unknown"
	}
}

// PermissionOutcome 权限结果
type PermissionOutcome int

const (
	OutcomeAllow PermissionOutcome = iota
	OutcomeDeny
)

// PermissionRequest 权限请求
type PermissionRequest struct {
	ToolName string
	Input    string
	Mode     PermissionMode
}

// PermissionPrompter 权限提示器接口
type PermissionPrompter interface {
	Decide(request *PermissionRequest) PermissionOutcome
}

// PermissionPolicy 权限策略
type PermissionPolicy struct {
	mode         PermissionMode
	allowedTools map[string]bool
	deniedTools  map[string]bool
}

// NewPermissionPolicy 创建权限策略
func NewPermissionPolicy(mode PermissionMode) *PermissionPolicy {
	return &PermissionPolicy{
		mode:         mode,
		allowedTools: make(map[string]bool),
		deniedTools:  make(map[string]bool),
	}
}

// SetAllowedTools 设置允许的工具
func (p *PermissionPolicy) SetAllowedTools(tools []string) {
	for _, tool := range tools {
		p.allowedTools[strings.ToLower(tool)] = true
	}
}

// SetDeniedTools 设置拒绝的工具
func (p *PermissionPolicy) SetDeniedTools(tools []string) {
	for _, tool := range tools {
		p.deniedTools[strings.ToLower(tool)] = true
	}
}

// Authorize 授权检查
func (p *PermissionPolicy) Authorize(toolName, input string, prompter PermissionPrompter) PermissionOutcome {
	toolLower := strings.ToLower(toolName)

	// 检查显式拒绝
	if p.deniedTools[toolLower] {
		return OutcomeDeny
	}

	// 检查显式允许
	if p.allowedTools[toolLower] {
		return OutcomeAllow
	}

	// 根据模式判断
	switch p.mode {
	case ReadOnly:
		// 只读模式下只允许读取类工具
		if isReadOnlyTool(toolName) {
			return OutcomeAllow
		}
		return OutcomeDeny

	case WorkspaceWrite:
		// 工作区写入模式允许大部分工具
		if isDangerousTool(toolName) {
			if prompter != nil {
				return prompter.Decide(&PermissionRequest{
					ToolName: toolName,
					Input:    input,
					Mode:     p.mode,
				})
			}
			return OutcomeDeny
		}
		return OutcomeAllow

	case DangerFullAccess:
		// 完全访问模式，默认允许
		if prompter != nil {
			return prompter.Decide(&PermissionRequest{
				ToolName: toolName,
				Input:    input,
				Mode:     p.mode,
			})
		}
		return OutcomeAllow
	}

	return OutcomeDeny
}

// isReadOnlyTool 判断是否为只读工具
func isReadOnlyTool(name string) bool {
	readonlyTools := []string{
		"fileread", "read", "glob", "grep", "search",
		"file_read", "glob_search", "grep_search",
	}
	nameLower := strings.ToLower(name)
	for _, rt := range readonlyTools {
		if strings.Contains(nameLower, rt) {
			return true
		}
	}
	return false
}

// isDangerousTool 判断是否为危险工具
func isDangerousTool(name string) bool {
	dangerousTools := []string{
		"bash", "shell", "exec", "delete", "remove",
		"dangerous", "sudo", "rm ", "chmod", "chown",
	}
	nameLower := strings.ToLower(name)
	for _, dt := range dangerousTools {
		if strings.Contains(nameLower, dt) {
			return true
		}
	}
	return false
}

// ConsolePrompter 控制台提示器
type ConsolePrompter struct{}

// Decide 控制台决策
func (c *ConsolePrompter) Decide(request *PermissionRequest) PermissionOutcome {
	fmt.Printf("\n⚠️  Tool Permission Request\n")
	fmt.Printf("Tool: %s\n", request.ToolName)
	fmt.Printf("Mode: %s\n", request.Mode)
	fmt.Printf("Input: %s\n\n", truncateString(request.Input, 200))
	fmt.Print("Allow? (yes/no/always): ")

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	switch response {
	case "yes", "y":
		return OutcomeAllow
	case "always":
		// 实际应该添加到允许列表
		return OutcomeAllow
	default:
		return OutcomeDeny
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

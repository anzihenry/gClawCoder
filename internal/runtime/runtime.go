package runtime

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/gclawcoder/gclaw/internal/commands"
	"github.com/gclawcoder/gclaw/internal/models"
	"github.com/gclawcoder/gclaw/internal/query"
	"github.com/gclawcoder/gclaw/internal/tools"
)

// RoutedMatch 路由匹配
type RoutedMatch struct {
	Kind       string
	Name       string
	SourceHint string
	Score      int
}

// RuntimeSession 运行时会话
type RuntimeSession struct {
	Prompt                   string
	Context                  *PortContext
	Setup                    *WorkspaceSetup
	SetupReport              *SetupReport
	SystemInitMessage        string
	RoutedMatches            []RoutedMatch
	TurnResult               *query.TurnResult
	CommandExecutionMessages []string
	ToolExecutionMessages    []string
	StreamEvents             []map[string]interface{}
	PersistedSessionPath     string
}

// PortContext 端口上下文
type PortContext struct {
	PythonFileCount  int
	ArchiveAvailable bool
}

// WorkspaceSetup 工作区设置
type WorkspaceSetup struct {
	PythonVersion  string
	Implementation string
	PlatformName   string
	TestCommand    string
}

// SetupReport 设置报告
type SetupReport struct {
	Setup *WorkspaceSetup
}

// PortRuntime 端口运行时
type PortRuntime struct{}

// NewPortRuntime 创建新的端口运行时
func NewPortRuntime() *PortRuntime {
	return &PortRuntime{}
}

// RoutePrompt 路由提示
func (p *PortRuntime) RoutePrompt(prompt string, limit int) []RoutedMatch {
	tokens := tokenizePrompt(prompt)

	byKind := map[string][]RoutedMatch{
		"command": collectMatches(tokens, commands.PortedCommands(), "command"),
		"tool":    collectMatches(tokens, tools.PortedTools(), "tool"),
	}

	var selected []RoutedMatch
	for _, kind := range []string{"command", "tool"} {
		if len(byKind[kind]) > 0 {
			selected = append(selected, byKind[kind][0])
			byKind[kind] = byKind[kind][1:]
		}
	}

	// 收集剩余的匹配
	var leftovers []RoutedMatch
	for _, matches := range byKind {
		leftovers = append(leftovers, matches...)
	}

	// 按分数排序
	sortMatchesByScore(leftovers)

	remaining := limit - len(selected)
	if remaining > 0 && len(leftovers) > 0 {
		if remaining > len(leftovers) {
			remaining = len(leftovers)
		}
		selected = append(selected, leftovers[:remaining]...)
	}

	if len(selected) > limit {
		selected = selected[:limit]
	}

	return selected
}

// BootstrapSession 引导会话
func (p *PortRuntime) BootstrapSession(prompt string, limit int) *RuntimeSession {
	context := BuildPortContext()
	setupReport := RunSetup()
	setup := setupReport.Setup

	matches := p.RoutePrompt(prompt, limit)

	// 执行命令
	var commandExecs []string
	for _, match := range matches {
		if match.Kind == "command" {
			result := commands.ExecuteCommand(match.Name, prompt)
			if result.Handled {
				commandExecs = append(commandExecs, result.Message)
			}
		}
	}

	// 执行工具
	var toolExecs []string
	for _, match := range matches {
		if match.Kind == "tool" {
			result := tools.ExecuteTool(match.Name, prompt)
			if result.Handled {
				toolExecs = append(toolExecs, result.Message)
			}
		}
	}

	// 推断权限拒绝
	denials := p.inferPermissionDenials(matches)

	// 创建查询引擎
	engine := query.FromWorkspace()

	// 流式事件
	var streamEvents []map[string]interface{}
	for event := range engine.StreamSubmitMessage(prompt,
		extractNames(matches, "command"),
		extractNames(matches, "tool"),
		denials) {
		streamEvents = append(streamEvents, event)
	}

	// 提交消息
	turnResult := engine.SubmitMessage(prompt,
		extractNames(matches, "command"),
		extractNames(matches, "tool"),
		denials)

	// 持久化会话
	persistedPath, _ := engine.PersistSession()

	return &RuntimeSession{
		Prompt:                   prompt,
		Context:                  context,
		Setup:                    setup,
		SetupReport:              setupReport,
		SystemInitMessage:        buildSystemInitMessage(),
		RoutedMatches:            matches,
		TurnResult:               &turnResult,
		CommandExecutionMessages: commandExecs,
		ToolExecutionMessages:    toolExecs,
		StreamEvents:             streamEvents,
		PersistedSessionPath:     persistedPath,
	}
}

// RunTurnLoop 运行轮次循环
func (p *PortRuntime) RunTurnLoop(prompt string, limit, maxTurns int, structuredOutput bool) []query.TurnResult {
	engine := query.FromWorkspace()
	engine.Config.StructuredOutput = structuredOutput
	engine.Config.MaxTurns = maxTurns

	matches := p.RoutePrompt(prompt, limit)
	commandNames := extractNames(matches, "command")
	toolNames := extractNames(matches, "tool")

	var results []query.TurnResult
	for turn := 0; turn < maxTurns; turn++ {
		turnPrompt := prompt
		if turn > 0 {
			turnPrompt = fmt.Sprintf("%s [turn %d]", prompt, turn+1)
		}

		result := engine.SubmitMessage(turnPrompt, commandNames, toolNames, nil)
		results = append(results, result)

		if result.StopReason != "completed" {
			break
		}
	}

	return results
}

// AsMarkdown 渲染为 Markdown
func (s *RuntimeSession) AsMarkdown() string {
	var sb strings.Builder

	sb.WriteString("# Runtime Session\n\n")
	sb.WriteString(fmt.Sprintf("Prompt: %s\n\n", s.Prompt))

	sb.WriteString("## Context\n")
	sb.WriteString(fmt.Sprintf("- Python files: %d\n", s.Context.PythonFileCount))
	sb.WriteString(fmt.Sprintf("- Archive available: %v\n\n", s.Context.ArchiveAvailable))

	sb.WriteString("## Setup\n")
	sb.WriteString(fmt.Sprintf("- Platform: %s\n", s.Setup.PlatformName))
	sb.WriteString(fmt.Sprintf("- Test command: %s\n\n", s.Setup.TestCommand))

	sb.WriteString("## Routed Matches\n")
	if len(s.RoutedMatches) > 0 {
		for _, match := range s.RoutedMatches {
			sb.WriteString(fmt.Sprintf("- [%s] %s (%d) — %s\n", match.Kind, match.Name, match.Score, match.SourceHint))
		}
	} else {
		sb.WriteString("- none\n")
	}
	sb.WriteString("\n")

	sb.WriteString("## Command Execution\n")
	if len(s.CommandExecutionMessages) > 0 {
		for _, msg := range s.CommandExecutionMessages {
			sb.WriteString(fmt.Sprintf("- %s\n", msg))
		}
	} else {
		sb.WriteString("- none\n")
	}
	sb.WriteString("\n")

	sb.WriteString("## Tool Execution\n")
	if len(s.ToolExecutionMessages) > 0 {
		for _, msg := range s.ToolExecutionMessages {
			sb.WriteString(fmt.Sprintf("- %s\n", msg))
		}
	} else {
		sb.WriteString("- none\n")
	}
	sb.WriteString("\n")

	sb.WriteString("## Turn Result\n")
	sb.WriteString(s.TurnResult.Output)
	sb.WriteString("\n\n")

	sb.WriteString(fmt.Sprintf("Persisted session path: %s\n", s.PersistedSessionPath))

	return sb.String()
}

// BuildPortContext 构建端口上下文
func BuildPortContext() *PortContext {
	// 简单实现，实际应该扫描目录
	return &PortContext{
		PythonFileCount:  0,
		ArchiveAvailable: false,
	}
}

// RunSetup 运行设置
func RunSetup() *SetupReport {
	setup := &WorkspaceSetup{
		PythonVersion:  getGoVersion(),
		Implementation: "Go",
		PlatformName:   runtime.GOOS + "/" + runtime.GOARCH,
		TestCommand:    "go test ./...",
	}

	return &SetupReport{
		Setup: setup,
	}
}

func buildSystemInitMessage() string {
	return "Go runtime initialized"
}

func (p *PortRuntime) inferPermissionDenials(matches []RoutedMatch) []models.PermissionDenial {
	var denials []models.PermissionDenial

	for _, match := range matches {
		if match.Kind == "tool" && strings.Contains(strings.ToLower(match.Name), "bash") {
			denials = append(denials, models.PermissionDenial{
				ToolName: match.Name,
				Reason:   "destructive shell execution remains gated in the Go port",
			})
		}
	}

	return denials
}

func tokenizePrompt(prompt string) map[string]bool {
	tokens := make(map[string]bool)
	for _, token := range strings.Fields(strings.ToLower(prompt)) {
		token = strings.Trim(token, "/-")
		if token != "" {
			tokens[token] = true
		}
	}
	return tokens
}

func collectMatches(tokens map[string]bool, modules []models.PortingModule, kind string) []RoutedMatch {
	var matches []RoutedMatch

	for _, module := range modules {
		score := scoreMatch(tokens, &module)
		if score > 0 {
			matches = append(matches, RoutedMatch{
				Kind:       kind,
				Name:       module.Name,
				SourceHint: module.SourceHint,
				Score:      score,
			})
		}
	}

	sortMatchesByScore(matches)
	return matches
}

func scoreMatch(tokens map[string]bool, module *models.PortingModule) int {
	haystacks := []string{
		strings.ToLower(module.Name),
		strings.ToLower(module.SourceHint),
		strings.ToLower(module.Responsibility),
	}

	score := 0
	for token := range tokens {
		for _, haystack := range haystacks {
			if strings.Contains(haystack, token) {
				score++
				break
			}
		}
	}
	return score
}

func sortMatchesByScore(matches []RoutedMatch) {
	for i := 0; i < len(matches)-1; i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].Score > matches[i].Score ||
				(matches[j].Score == matches[i].Score && matches[j].Name < matches[i].Name) {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}
}

func extractNames(matches []RoutedMatch, kind string) []string {
	var names []string
	for _, match := range matches {
		if match.Kind == kind {
			names = append(names, match.Name)
		}
	}
	return names
}

func getGoVersion() string {
	cmd := exec.Command("go", "version")
	output, err := cmd.Output()
	if err != nil {
		return runtime.Version()
	}
	return strings.TrimSpace(string(output))
}

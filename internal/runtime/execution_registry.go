package runtime

import (
	"github.com/gclawcoder/gclaw/internal/commands"
	"github.com/gclawcoder/gclaw/internal/models"
	"github.com/gclawcoder/gclaw/internal/tools"
)

// ExecutionRegistry 执行注册表
type ExecutionRegistry struct {
	commands map[string]*models.PortingModule
	tools    map[string]*models.PortingModule
}

// BuildExecutionRegistry 构建执行注册表
func BuildExecutionRegistry() *ExecutionRegistry {
	cmdMap := make(map[string]*models.PortingModule)
	for _, cmd := range commands.PortedCommands() {
		cmdCopy := cmd
		cmdMap[cmd.Name] = &cmdCopy
	}

	toolMap := make(map[string]*models.PortingModule)
	for _, tool := range tools.PortedTools() {
		toolCopy := tool
		toolMap[tool.Name] = &toolCopy
	}

	return &ExecutionRegistry{
		commands: cmdMap,
		tools:    toolMap,
	}
}

// Command 获取命令
func (r *ExecutionRegistry) Command(name string) *models.PortingModule {
	return r.commands[name]
}

// Tool 获取工具
func (r *ExecutionRegistry) Tool(name string) *models.PortingModule {
	return r.tools[name]
}

// Commands 返回所有命令
func (r *ExecutionRegistry) Commands() map[string]*models.PortingModule {
	return r.commands
}

// Tools 返回所有工具
func (r *ExecutionRegistry) Tools() map[string]*models.PortingModule {
	return r.tools
}

// CommandCount 返回命令数量
func (r *ExecutionRegistry) CommandCount() int {
	return len(r.commands)
}

// ToolCount 返回工具数量
func (r *ExecutionRegistry) ToolCount() int {
	return len(r.tools)
}

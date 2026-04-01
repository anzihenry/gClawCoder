package toolkit

import (
	"encoding/json"
	"fmt"
)

// Tool 工具接口
type Tool interface {
	Execute(input json.RawMessage) (string, error)
	GetDescription() string
	GetInputSchema() string
}

// Registry 工具注册表
type Registry struct {
	tools map[string]Tool
}

// NewRegistry 创建新注册表
func NewRegistry() *Registry {
	r := &Registry{
		tools: make(map[string]Tool),
	}
	r.RegisterDefaults()
	return r
}

// RegisterDefaults 注册默认工具
func (r *Registry) RegisterDefaults() {
	r.Register("Bash", &BashTool{})
	r.Register("FileRead", &FileReadTool{})
	r.Register("FileWrite", &FileWriteTool{})
	r.Register("FileEdit", &FileEditTool{})
	r.Register("Glob", &GlobTool{})
	r.Register("Grep", &GrepTool{})
	r.Register("WebSearch", NewWebSearchTool(""))
	r.Register("WebFetch", NewWebFetchTool())
	r.Register("TodoWrite", NewTodoWriteTool())
	r.Register("Agent", NewAgentTool())
	r.Register("Task", NewTaskTool())
	r.Register("NotebookEdit", NewNotebookTool())
	r.Register("AskUserQuestion", NewAskUserQuestionTool())
	r.Register("Config", NewConfigTool())
	r.Register("LSP", NewLSPTool())
	r.Register("RemoteTrigger", NewRemoteTriggerTool())
	r.Register("ScheduleCron", NewScheduleCronTool())
	r.Register("Team", NewTeamTool())
}

// Register 注册工具
func (r *Registry) Register(name string, tool Tool) {
	r.tools[name] = tool
}

// Get 获取工具
func (r *Registry) Get(name string) (Tool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

// Execute 执行工具
func (r *Registry) Execute(name string, input json.RawMessage) (string, error) {
	tool, ok := r.tools[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return tool.Execute(input)
}

// List 列出所有工具
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// Count 返回工具数量
func (r *Registry) Count() int {
	return len(r.tools)
}

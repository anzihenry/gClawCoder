package models

// PortingModule 镜像模块结构
type PortingModule struct {
	Name           string `json:"name"`
	Responsibility string `json:"responsibility"`
	SourceHint     string `json:"source_hint"`
	Status         string `json:"status"`
}

// PermissionDenial 权限拒绝结构
type PermissionDenial struct {
	ToolName string `json:"tool_name"`
	Reason   string `json:"reason"`
}

// UsageSummary Token 使用统计
type UsageSummary struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// AddTurn 添加一轮对话的 token 统计
func (u UsageSummary) AddTurn(prompt, output string) UsageSummary {
	return UsageSummary{
		InputTokens:  u.InputTokens + len(prompt),
		OutputTokens: u.OutputTokens + len(output),
	}
}

// TotalTokens 返回总 token 数
func (u UsageSummary) TotalTokens() int {
	return u.InputTokens + u.OutputTokens
}

// PortingBacklog 移植待办列表
type PortingBacklog struct {
	Title   string
	Modules []PortingModule
}

// SummaryLines 生成摘要行
func (p PortingBacklog) SummaryLines() []string {
	lines := make([]string, 0, len(p.Modules))
	for _, module := range p.Modules {
		lines = append(lines, formatModuleSummary(&module))
	}
	return lines
}

func formatModuleSummary(m *PortingModule) string {
	return "- " + m.Name + " [" + m.Status + "] — " + m.Responsibility + " (from " + m.SourceHint + ")"
}

// Subsystem 子系统结构
type Subsystem struct {
	Name      string
	Path      string
	FileCount int
	Notes     string
}

package toolkit

import (
	"encoding/json"
	"fmt"
	"strings"
)

// LSPTool 语言服务器协议工具
type LSPTool struct {
	servers map[string]*LSPServer
}

// LSPServer LSP 服务器
type LSPServer struct {
	ID        string
	Language  string
	Command   string
	Args      []string
	Root      string
	Connected bool
}

// NewLSPTool 创建 LSP 工具
func NewLSPTool() *LSPTool {
	return &LSPTool{
		servers: make(map[string]*LSPServer),
	}
}

// LSPInput LSP 输入
type LSPInput struct {
	Action   string `json:"action"` // start, stop, status, hover, goto_def, find_refs, diagnostics, symbol
	Language string `json:"language,omitempty"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
	Symbol   string `json:"symbol,omitempty"`
}

// Execute 执行 LSP 操作
func (t *LSPTool) Execute(input json.RawMessage) (string, error) {
	var inp LSPInput
	if err := json.Unmarshal(input, &inp); err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	switch inp.Action {
	case "start":
		return t.startServer(inp)
	case "stop":
		return t.stopServer(inp)
	case "status":
		return t.status(inp)
	case "hover":
		return t.hover(inp)
	case "goto_def":
		return t.gotoDefinition(inp)
	case "find_refs":
		return t.findReferences(inp)
	case "diagnostics":
		return t.diagnostics(inp)
	case "symbol":
		return t.workspaceSymbol(inp)
	default:
		return "", fmt.Errorf("unknown action: %s", inp.Action)
	}
}

func (t *LSPTool) startServer(inp LSPInput) (string, error) {
	if inp.Language == "" {
		return "", fmt.Errorf("language is required")
	}

	id := inp.Language
	server := &LSPServer{
		ID:        id,
		Language:  inp.Language,
		Root:      inp.File,
		Connected: false,
	}

	// 根据语言设置默认命令
	switch strings.ToLower(inp.Language) {
	case "go":
		server.Command = "gopls"
	case "rust":
		server.Command = "rust-analyzer"
	case "typescript", "javascript":
		server.Command = "typescript-language-server"
	case "python":
		server.Command = "pylsp"
	default:
		server.Command = "language-server"
	}

	t.servers[id] = server

	// 模拟连接 (实际应该启动 LSP 进程)
	server.Connected = true

	return fmt.Sprintf("Started LSP server for %s (%s)", inp.Language, server.Command), nil
}

func (t *LSPTool) stopServer(inp LSPInput) (string, error) {
	if inp.Language == "" {
		return "", fmt.Errorf("language is required")
	}

	if _, ok := t.servers[inp.Language]; ok {
		delete(t.servers, inp.Language)
		return fmt.Sprintf("Stopped LSP server for %s", inp.Language), nil
	}

	return fmt.Sprintf("LSP server for %s not running", inp.Language), nil
}

func (t *LSPTool) status(inp LSPInput) (string, error) {
	if len(t.servers) == 0 {
		return "No LSP servers running", nil
	}

	var sb strings.Builder
	sb.WriteString("LSP Servers:\n")
	for id, server := range t.servers {
		status := "⚪ disconnected"
		if server.Connected {
			status = "🟢 connected"
		}
		sb.WriteString(fmt.Sprintf("  %s - %s [%s]\n", id, status, server.Command))
	}

	return sb.String(), nil
}

func (t *LSPTool) hover(inp LSPInput) (string, error) {
	if inp.File == "" {
		return "", fmt.Errorf("file is required")
	}

	// 模拟 hover 响应
	return fmt.Sprintf("Hover at %s:%d:%d\n(Simulated - connect to real LSP server for actual results)",
		inp.File, inp.Line, inp.Column), nil
}

func (t *LSPTool) gotoDefinition(inp LSPInput) (string, error) {
	if inp.File == "" || inp.Symbol == "" {
		return "", fmt.Errorf("file and symbol are required")
	}

	// 模拟跳转定义
	return fmt.Sprintf("Go to definition of '%s' at %s:%d:%d\n  Found in: %s:%d:%d (simulated)",
		inp.Symbol, inp.File, inp.Line, inp.Column, inp.File, inp.Line+5, 1), nil
}

func (t *LSPTool) findReferences(inp LSPInput) (string, error) {
	if inp.Symbol == "" {
		return "", fmt.Errorf("symbol is required")
	}

	// 模拟查找引用
	return fmt.Sprintf("References to '%s':\n  - %s:%d:%d\n  - %s:%d:%d\n(Simulated - 2 references found)",
		inp.Symbol, inp.File, inp.Line, inp.Column, inp.File, inp.Line+10, inp.Column), nil
}

func (t *LSPTool) diagnostics(inp LSPInput) (string, error) {
	if inp.File == "" {
		return "", fmt.Errorf("file is required")
	}

	// 模拟诊断
	return fmt.Sprintf("Diagnostics for %s:\n  No issues found (simulated)", inp.File), nil
}

func (t *LSPTool) workspaceSymbol(inp LSPInput) (string, error) {
	if inp.Symbol == "" {
		return "", fmt.Errorf("symbol query is required")
	}

	// 模拟符号搜索
	return fmt.Sprintf("Workspace symbols matching '%s':\n  - Function: example()\n  - Type: Example\n(Simulated)",
		inp.Symbol), nil
}

// GetDescription 获取描述
func (t *LSPTool) GetDescription() string {
	return "Language Server Protocol integration for code intelligence"
}

// GetInputSchema 获取输入 schema
func (t *LSPTool) GetInputSchema() string {
	return `{
		"type":"object",
		"properties":{
			"action":{"type":"string","enum":["start","stop","status","hover","goto_def","find_refs","diagnostics","symbol"]},
			"language":{"type":"string"},
			"file":{"type":"string"},
			"line":{"type":"integer"},
			"column":{"type":"integer"},
			"symbol":{"type":"string"}
		},
		"required":["action"]
	}`
}

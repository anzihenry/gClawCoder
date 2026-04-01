package toolkit

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/gclawcoder/gclaw/internal/mcp"
)

// MCPTool MCP 工具集成
type MCPTool struct {
	servers   map[string]*mcp.StdioClient
	mu        sync.Mutex
	connected map[string]bool
}

// NewMCPTool 创建 MCP 工具
func NewMCPTool() *MCPTool {
	return &MCPTool{
		servers:   make(map[string]*mcp.StdioClient),
		connected: make(map[string]bool),
	}
}

// MCPToolInput MCP 工具输入
type MCPToolInput struct {
	Action      string                 `json:"action"` // connect, disconnect, list_servers, list_tools, call_tool, list_resources, read_resource
	ServerID    string                 `json:"serverId,omitempty"`
	Command     string                 `json:"command,omitempty"`
	Args        []string               `json:"args,omitempty"`
	ToolName    string                 `json:"toolName,omitempty"`
	ToolArgs    map[string]interface{} `json:"toolArgs,omitempty"`
	ResourceURI string                 `json:"resourceUri,omitempty"`
}

// Execute 执行 MCP 操作
func (t *MCPTool) Execute(input json.RawMessage) (string, error) {
	var inp MCPToolInput
	if err := json.Unmarshal(input, &inp); err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	switch inp.Action {
	case "connect":
		return t.connect(inp)
	case "disconnect":
		return t.disconnect(inp.ServerID)
	case "list_servers":
		return t.listServers()
	case "list_tools":
		return t.listTools(inp.ServerID)
	case "call_tool":
		return t.callTool(inp)
	case "list_resources":
		return t.listResources(inp.ServerID)
	case "read_resource":
		return t.readResource(inp)
	default:
		return "", fmt.Errorf("unknown action: %s", inp.Action)
	}
}

// connect 连接 MCP 服务器
func (t *MCPTool) connect(inp MCPToolInput) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if inp.Command == "" {
		return "", fmt.Errorf("command is required")
	}

	serverID := inp.ServerID
	if serverID == "" {
		serverID = fmt.Sprintf("server-%d", len(t.servers)+1)
	}

	// 创建 MCP stdio 客户端
	client := mcp.NewStdioClient(inp.Command, inp.Args, nil)

	// 启动服务器
	if err := client.Start(); err != nil {
		return "", fmt.Errorf("failed to start MCP server: %w", err)
	}

	// 初始化
	if err := client.Initialize(); err != nil {
		client.Stop()
		return "", fmt.Errorf("failed to initialize: %w", err)
	}

	t.servers[serverID] = client
	t.connected[serverID] = true

	info := client.GetServerInfo()
	return fmt.Sprintf("Connected to MCP server: %s\n  ID: %s\n  Command: %s\n  Version: %s",
		info.Name, serverID, inp.Command, info.Version), nil
}

// disconnect 断开连接
func (t *MCPTool) disconnect(serverID string) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	client, ok := t.servers[serverID]
	if !ok {
		return "", fmt.Errorf("server not found: %s", serverID)
	}

	if err := client.Stop(); err != nil {
		return "", fmt.Errorf("failed to stop server: %w", err)
	}

	delete(t.servers, serverID)
	delete(t.connected, serverID)

	return fmt.Sprintf("Disconnected from MCP server: %s", serverID), nil
}

// listServers 列出服务器
func (t *MCPTool) listServers() (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.servers) == 0 {
		return "No MCP servers connected", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("MCP Servers (%d):\n", len(t.servers)))

	for id, client := range t.servers {
		status := "⚪ disconnected"
		if t.connected[id] {
			status = "🟢 connected"
		}

		info := client.GetServerInfo()
		sb.WriteString(fmt.Sprintf("  %s. %s - %s\n", id, info.Name, status))
		if info.Version != "" {
			sb.WriteString(fmt.Sprintf("     Version: %s\n", info.Version))
		}
	}

	return sb.String(), nil
}

// listTools 列出工具
func (t *MCPTool) listTools(serverID string) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if serverID == "" {
		return "", fmt.Errorf("serverId is required")
	}

	client, ok := t.servers[serverID]
	if !ok {
		return "", fmt.Errorf("server not found: %s", serverID)
	}

	tools, err := client.ListTools()
	if err != nil {
		return "", fmt.Errorf("failed to list tools: %w", err)
	}

	if len(tools) == 0 {
		return "No tools available", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Tools from %s (%d):\n", serverID, len(tools)))

	for _, tool := range tools {
		sb.WriteString(fmt.Sprintf("  - %s\n", tool.Name))
		if tool.Description != "" {
			sb.WriteString(fmt.Sprintf("    %s\n", tool.Description))
		}
	}

	return sb.String(), nil
}

// callTool 调用工具
func (t *MCPTool) callTool(inp MCPToolInput) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if inp.ServerID == "" || inp.ToolName == "" {
		return "", fmt.Errorf("serverId and toolName are required")
	}

	client, ok := t.servers[inp.ServerID]
	if !ok {
		return "", fmt.Errorf("server not found: %s", inp.ServerID)
	}

	result, err := client.CallTool(inp.ToolName, inp.ToolArgs)
	if err != nil {
		return "", fmt.Errorf("tool call failed: %w", err)
	}

	return fmt.Sprintf("Tool %s result:\n%s", inp.ToolName, result), nil
}

// listResources 列出资源
func (t *MCPTool) listResources(serverID string) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if serverID == "" {
		return "", fmt.Errorf("serverId is required")
	}

	client, ok := t.servers[serverID]
	if !ok {
		return "", fmt.Errorf("server not found: %s", serverID)
	}

	resources, err := client.ListResources()
	if err != nil {
		return "", fmt.Errorf("failed to list resources: %w", err)
	}

	if len(resources) == 0 {
		return "No resources available", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Resources from %s (%d):\n", serverID, len(resources)))

	for _, res := range resources {
		sb.WriteString(fmt.Sprintf("  - %s\n", res.URI))
		if res.Name != "" {
			sb.WriteString(fmt.Sprintf("    Name: %s\n", res.Name))
		}
		if res.Description != "" {
			sb.WriteString(fmt.Sprintf("    %s\n", res.Description))
		}
	}

	return sb.String(), nil
}

// readResource 读取资源
func (t *MCPTool) readResource(inp MCPToolInput) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if inp.ServerID == "" || inp.ResourceURI == "" {
		return "", fmt.Errorf("serverId and resourceUri are required")
	}

	client, ok := t.servers[inp.ServerID]
	if !ok {
		return "", fmt.Errorf("server not found: %s", inp.ServerID)
	}

	result, err := client.ReadResource(inp.ResourceURI)
	if err != nil {
		return "", fmt.Errorf("failed to read resource: %w", err)
	}

	return fmt.Sprintf("Resource %s content:\n%s", inp.ResourceURI, result), nil
}

// GetDescription 获取描述
func (t *MCPTool) GetDescription() string {
	return "Connect to and interact with MCP (Model Context Protocol) servers"
}

// GetInputSchema 获取输入 schema
func (t *MCPTool) GetInputSchema() string {
	return `{
		"type":"object",
		"properties":{
			"action":{"type":"string","enum":["connect","disconnect","list_servers","list_tools","call_tool","list_resources","read_resource"]},
			"serverId":{"type":"string"},
			"command":{"type":"string"},
			"args":{"type":"array","items":{"type":"string"}},
			"toolName":{"type":"string"},
			"toolArgs":{"type":"object"},
			"resourceUri":{"type":"string"}
		},
		"required":["action"]
	}`
}

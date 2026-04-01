package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"
)

// JSONRPC 版本
const JSONRPCVersion = "2.0"

// JSONRPCRequest JSON-RPC 请求
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// JSONRPCResponse JSON-RPC 响应
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError RPC 错误
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCP 能力
type MCPCapabilities struct {
	Tools     *ToolCapabilities     `json:"tools,omitempty"`
	Resources *ResourceCapabilities `json:"resources,omitempty"`
	Prompts   *PromptCapabilities   `json:"prompts,omitempty"`
}

// ToolCapabilities 工具能力
type ToolCapabilities struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourceCapabilities 资源能力
type ResourceCapabilities struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// PromptCapabilities 提示能力
type PromptCapabilities struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// MCPTool MCP 工具
type MCPTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	InputSchema interface{} `json:"inputSchema,omitempty"`
}

// MCPResource MCP 资源
type MCPResource struct {
	URI         string `json:"uri"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// StdioClient 标准输入输出 MCP 客户端
type StdioClient struct {
	command      string
	args         []string
	env          []string
	cmd          *exec.Cmd
	stdin        io.WriteCloser
	stdout       io.ReadCloser
	stderr       io.ReadCloser
	idCounter    int
	mu           sync.Mutex
	responses    map[interface{}]*JSONRPCResponse
	responseCh   chan *JSONRPCResponse
	initialized  bool
	serverInfo   ServerInfo
	capabilities MCPCapabilities
}

// ServerInfo 服务器信息
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// NewStdioClient 创建 Stdio 客户端
func NewStdioClient(command string, args, env []string) *StdioClient {
	return &StdioClient{
		command:    command,
		args:       args,
		env:        env,
		responses:  make(map[interface{}]*JSONRPCResponse),
		responseCh: make(chan *JSONRPCResponse, 100),
	}
}

// Start 启动 MCP 服务器
func (c *StdioClient) Start() error {
	c.cmd = exec.Command(c.command, c.args...)
	c.cmd.Env = append(c.cmd.Environ(), c.env...)

	var err error
	c.stdin, err = c.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	c.stdout, err = c.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	c.stderr, err = c.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// 启动读取协程
	go c.readResponses()

	return nil
}

// Stop 停止 MCP 服务器
func (c *StdioClient) Stop() error {
	if c.cmd != nil && c.cmd.Process != nil {
		return c.cmd.Process.Kill()
	}
	return nil
}

// Initialize 初始化 MCP 连接
func (c *StdioClient) Initialize() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	params := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"clientInfo": map[string]interface{}{
			"name":    "gClawCoder",
			"version": "1.0.0",
		},
	}

	var result map[string]interface{}
	err := c.sendRequest("initialize", params, &result)
	if err != nil {
		return fmt.Errorf("initialize failed: %w", err)
	}

	// 解析服务器信息
	if serverInfo, ok := result["serverInfo"].(map[string]interface{}); ok {
		if name, ok := serverInfo["name"].(string); ok {
			c.serverInfo.Name = name
		}
		if version, ok := serverInfo["version"].(string); ok {
			c.serverInfo.Version = version
		}
	}

	// 解析能力
	if caps, ok := result["capabilities"].(map[string]interface{}); ok {
		c.capabilities = MCPCapabilities{}
		if _, ok := caps["tools"].(map[string]interface{}); ok {
			c.capabilities.Tools = &ToolCapabilities{}
		}
	}

	// 发送 initialized 通知
	c.sendNotification("notifications/initialized", nil)

	c.initialized = true
	return nil
}

// ListTools 列出工具
func (c *StdioClient) ListTools() ([]MCPTool, error) {
	if !c.initialized {
		return nil, fmt.Errorf("client not initialized")
	}

	var result map[string]interface{}
	err := c.sendRequest("tools/list", nil, &result)
	if err != nil {
		return nil, err
	}

	tools := []MCPTool{}
	if toolsData, ok := result["tools"].([]interface{}); ok {
		for _, t := range toolsData {
			if toolMap, ok := t.(map[string]interface{}); ok {
				tool := MCPTool{}
				if name, ok := toolMap["name"].(string); ok {
					tool.Name = name
				}
				if desc, ok := toolMap["description"].(string); ok {
					tool.Description = desc
				}
				tool.InputSchema = toolMap["inputSchema"]
				tools = append(tools, tool)
			}
		}
	}

	return tools, nil
}

// CallTool 调用工具
func (c *StdioClient) CallTool(name string, args map[string]interface{}) (string, error) {
	if !c.initialized {
		return "", fmt.Errorf("client not initialized")
	}

	params := map[string]interface{}{
		"name":      name,
		"arguments": args,
	}

	var result map[string]interface{}
	err := c.sendRequest("tools/call", params, &result)
	if err != nil {
		return "", err
	}

	// 解析结果
	content, _ := json.Marshal(result)
	return string(content), nil
}

// ListResources 列出资源
func (c *StdioClient) ListResources() ([]MCPResource, error) {
	if !c.initialized {
		return nil, fmt.Errorf("client not initialized")
	}

	var result map[string]interface{}
	err := c.sendRequest("resources/list", nil, &result)
	if err != nil {
		return nil, err
	}

	resources := []MCPResource{}
	if resData, ok := result["resources"].([]interface{}); ok {
		for _, r := range resData {
			if resMap, ok := r.(map[string]interface{}); ok {
				resource := MCPResource{}
				if uri, ok := resMap["uri"].(string); ok {
					resource.URI = uri
				}
				if name, ok := resMap["name"].(string); ok {
					resource.Name = name
				}
				if desc, ok := resMap["description"].(string); ok {
					resource.Description = desc
				}
				if mime, ok := resMap["mimeType"].(string); ok {
					resource.MimeType = mime
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources, nil
}

// ReadResource 读取资源
func (c *StdioClient) ReadResource(uri string) (string, error) {
	if !c.initialized {
		return "", fmt.Errorf("client not initialized")
	}

	params := map[string]interface{}{
		"uri": uri,
	}

	var result map[string]interface{}
	err := c.sendRequest("resources/read", params, &result)
	if err != nil {
		return "", err
	}

	content, _ := json.Marshal(result)
	return string(content), nil
}

func (c *StdioClient) sendRequest(method string, params interface{}, result interface{}) error {
	c.mu.Lock()
	c.idCounter++
	id := c.idCounter
	c.mu.Unlock()

	req := JSONRPCRequest{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Method:  method,
		Params:  params,
	}

	if err := c.sendJSON(req); err != nil {
		return err
	}

	// 等待响应
	select {
	case resp := <-c.responseCh:
		if resp.ID == id {
			if resp.Error != nil {
				return fmt.Errorf("RPC error %d: %s", resp.Error.Code, resp.Error.Message)
			}
			if result != nil {
				data, _ := json.Marshal(resp.Result)
				return json.Unmarshal(data, result)
			}
			return nil
		}
		return fmt.Errorf("unexpected response ID")
	case <-time.After(30 * time.Second):
		return fmt.Errorf("request timeout")
	}
}

func (c *StdioClient) sendNotification(method string, params interface{}) error {
	notif := map[string]interface{}{
		"jsonrpc": JSONRPCVersion,
		"method":  method,
		"params":  params,
	}
	return c.sendJSON(notif)
}

func (c *StdioClient) sendJSON(data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	jsonData = append(jsonData, '\n')

	_, err = c.stdin.Write(jsonData)
	return err
}

func (c *StdioClient) readResponses() {
	reader := bufio.NewReader(c.stdout)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				fmt.Printf("Read error: %v\n", err)
			}
			return
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		var resp JSONRPCResponse
		if err := json.Unmarshal(line, &resp); err == nil {
			c.responseCh <- &resp
		}
	}
}

// GetServerInfo 获取服务器信息
func (c *StdioClient) GetServerInfo() ServerInfo {
	return c.serverInfo
}

// GetCapabilities 获取能力
func (c *StdioClient) GetCapabilities() MCPCapabilities {
	return c.capabilities
}

// IsInitialized 检查是否已初始化
func (c *StdioClient) IsInitialized() bool {
	return c.initialized
}

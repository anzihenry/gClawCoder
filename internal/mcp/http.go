package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// HTTPClient HTTP 传输 MCP 客户端
type HTTPClient struct {
	baseURL      string
	httpClient   *http.Client
	idCounter    int
	mu           sync.Mutex
	initialized  bool
	serverInfo   ServerInfo
	capabilities MCPCapabilities
}

// NewHTTPClient 创建 HTTP 客户端
func NewHTTPClient(baseURL string, timeout time.Duration) *HTTPClient {
	if timeout == 0 {
		timeout = 120 * time.Second
	}
	return &HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Connect 连接到 HTTP 服务器 (实际是验证连接)
func (c *HTTPClient) Connect() error {
	req, err := http.NewRequest("GET", c.baseURL+"/health", nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Health 端点可能不存在，忽略
		return nil
	}
	resp.Body.Close()

	return nil
}

// Close 关闭连接
func (c *HTTPClient) Close() error {
	return nil
}

// Initialize 初始化
func (c *HTTPClient) Initialize() error {
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
	if err := c.sendRequest("initialize", params, &result); err != nil {
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

	c.initialized = true
	return nil
}

// ListTools 列出工具
func (c *HTTPClient) ListTools() ([]MCPTool, error) {
	if !c.initialized {
		return nil, fmt.Errorf("client not initialized")
	}

	var result map[string]interface{}
	if err := c.sendRequest("tools/list", nil, &result); err != nil {
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
func (c *HTTPClient) CallTool(name string, args map[string]interface{}) (string, error) {
	if !c.initialized {
		return "", fmt.Errorf("client not initialized")
	}

	params := map[string]interface{}{
		"name":      name,
		"arguments": args,
	}

	var result map[string]interface{}
	if err := c.sendRequest("tools/call", params, &result); err != nil {
		return "", err
	}

	content, _ := json.Marshal(result)
	return string(content), nil
}

// ListResources 列出资源
func (c *HTTPClient) ListResources() ([]MCPResource, error) {
	if !c.initialized {
		return nil, fmt.Errorf("client not initialized")
	}

	var result map[string]interface{}
	if err := c.sendRequest("resources/list", nil, &result); err != nil {
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
func (c *HTTPClient) ReadResource(uri string) (string, error) {
	if !c.initialized {
		return "", fmt.Errorf("client not initialized")
	}

	params := map[string]interface{}{
		"uri": uri,
	}

	var result map[string]interface{}
	if err := c.sendRequest("resources/read", params, &result); err != nil {
		return "", err
	}

	content, _ := json.Marshal(result)
	return string(content), nil
}

// sendRequest 发送 JSON-RPC 请求
func (c *HTTPClient) sendRequest(method string, params interface{}, result interface{}) error {
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

	jsonData, err := json.Marshal(req)
	if err != nil {
		return err
	}

	// 构建 URL
	url := c.baseURL + "/mcp"
	if method == "initialize" {
		url = c.baseURL + "/initialize"
	}

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed (%d): %s", resp.StatusCode, string(body))
	}

	var rpcResp JSONRPCResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return err
	}

	if rpcResp.Error != nil {
		return fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	if result != nil {
		data, _ := json.Marshal(rpcResp.Result)
		return json.Unmarshal(data, result)
	}

	return nil
}

// GetServerInfo 获取服务器信息
func (c *HTTPClient) GetServerInfo() ServerInfo {
	return c.serverInfo
}

// GetCapabilities 获取能力
func (c *HTTPClient) GetCapabilities() MCPCapabilities {
	return c.capabilities
}

// IsInitialized 检查是否已初始化
func (c *HTTPClient) IsInitialized() bool {
	return c.initialized
}

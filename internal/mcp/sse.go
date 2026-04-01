package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// SSEClient SSE 传输 MCP 客户端
type SSEClient struct {
	baseURL      string
	httpClient   *http.Client
	endpointURL  string
	eventSource  *EventSource
	idCounter    int
	mu           sync.Mutex
	initialized  bool
	serverInfo   ServerInfo
	capabilities MCPCapabilities
	responseCh   chan *JSONRPCResponse
}

// EventSource SSE 事件源
type EventSource struct {
	url     string
	resp    *http.Response
	scanner *bufio.Scanner
	onEvent func(EventType, []byte)
	onError func(error)
	done    chan struct{}
}

// EventType 事件类型
type EventType string

const (
	EventMessage  EventType = "message"
	EventError    EventType = "error"
	EventEndpoint EventType = "endpoint"
)

// NewSSEClient 创建 SSE 客户端
func NewSSEClient(baseURL string) *SSEClient {
	return &SSEClient{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{Timeout: 120 * time.Second},
		responseCh: make(chan *JSONRPCResponse, 100),
	}
}

// Connect 连接到 SSE 服务器
func (c *SSEClient) Connect() error {
	// 获取 SSE 端点
	sseURL := fmt.Sprintf("%s/sse", c.baseURL)

	req, err := http.NewRequest("GET", sseURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create SSE request: %w", err)
	}

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to SSE: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return fmt.Errorf("SSE connection failed: %d", resp.StatusCode)
	}

	c.eventSource = &EventSource{
		url:     sseURL,
		resp:    resp,
		scanner: bufio.NewScanner(resp.Body),
		done:    make(chan struct{}),
	}

	// 启动事件处理
	go c.processEvents()

	return nil
}

// processEvents 处理 SSE 事件
func (c *SSEClient) processEvents() {
	var eventType EventType
	var eventData bytes.Buffer

	for c.eventSource.scanner.Scan() {
		line := c.eventSource.scanner.Text()

		if line == "" {
			// 空行表示事件结束
			if eventData.Len() > 0 {
				c.handleEvent(eventType, eventData.Bytes())
				eventData.Reset()
				eventType = ""
			}
			continue
		}

		if strings.HasPrefix(line, "event: ") {
			eventType = EventType(strings.TrimPrefix(line, "event: "))
		} else if strings.HasPrefix(line, "data: ") {
			if eventData.Len() > 0 {
				eventData.WriteString("\n")
			}
			eventData.WriteString(strings.TrimPrefix(line, "data: "))
		} else if strings.HasPrefix(line, "id: ") {
			// 忽略 ID
		}
	}

	// 检查错误
	if err := c.eventSource.scanner.Err(); err != nil && err != io.EOF {
		c.eventSource.onError <- err
	}

	close(c.eventSource.done)
}

// handleEvent 处理单个事件
func (c *SSEClient) handleEvent(eventType EventType, data []byte) {
	switch eventType {
	case EventEndpoint:
		// 获取端点 URL
		c.endpointURL = string(data)
		if !strings.HasPrefix(c.endpointURL, "http") {
			c.endpointURL = c.baseURL + c.endpointURL
		}

	case EventMessage:
		// JSON-RPC 消息
		var resp JSONRPCResponse
		if err := json.Unmarshal(data, &resp); err == nil {
			c.responseCh <- &resp
		}

	case EventError:
		fmt.Printf("SSE Error: %s\n", string(data))
	}
}

// Close 关闭连接
func (c *SSEClient) Close() error {
	if c.eventSource != nil && c.eventSource.resp != nil {
		return c.eventSource.resp.Body.Close()
	}
	return nil
}

// Initialize 初始化连接
func (c *SSEClient) Initialize() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 等待端点
	timeout := time.After(10 * time.Second)
	for c.endpointURL == "" {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for endpoint")
		case <-time.After(100 * time.Millisecond):
		}
	}

	// 发送 initialize 请求
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

	// 发送 initialized 通知
	c.sendNotification("notifications/initialized", nil)

	c.initialized = true
	return nil
}

// ListTools 列出工具
func (c *SSEClient) ListTools() ([]MCPTool, error) {
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
func (c *SSEClient) CallTool(name string, args map[string]interface{}) (string, error) {
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
func (c *SSEClient) ListResources() ([]MCPResource, error) {
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
func (c *SSEClient) ReadResource(uri string) (string, error) {
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

// sendRequest 发送请求到端点
func (c *SSEClient) sendRequest(method string, params interface{}, result interface{}) error {
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

	// 发送到端点
	resp, err := c.httpClient.Post(c.endpointURL, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed (%d): %s", resp.StatusCode, string(body))
	}

	// 对于异步响应，等待事件源
	if resp.StatusCode == http.StatusAccepted {
		select {
		case rpcResp := <-c.responseCh:
			if rpcResp.ID == id {
				if rpcResp.Error != nil {
					return fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
				}
				if result != nil {
					data, _ := json.Marshal(rpcResp.Result)
					return json.Unmarshal(data, result)
				}
				return nil
			}
		case <-time.After(30 * time.Second):
			return fmt.Errorf("request timeout")
		}
	}

	// 同步响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
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

// sendNotification 发送通知
func (c *SSEClient) sendNotification(method string, params interface{}) error {
	notif := map[string]interface{}{
		"jsonrpc": JSONRPCVersion,
		"method":  method,
		"params":  params,
	}

	jsonData, _ := json.Marshal(notif)

	resp, err := c.httpClient.Post(c.endpointURL, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}

// GetServerInfo 获取服务器信息
func (c *SSEClient) GetServerInfo() ServerInfo {
	return c.serverInfo
}

// GetCapabilities 获取能力
func (c *SSEClient) GetCapabilities() MCPCapabilities {
	return c.capabilities
}

// IsInitialized 检查是否已初始化
func (c *SSEClient) IsInitialized() bool {
	return c.initialized
}

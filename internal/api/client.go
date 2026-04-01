package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// MessageRole 消息角色
type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleSystem    MessageRole = "system"
)

// ContentBlock 内容块
type ContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   string          `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
}

// Message API 消息
type Message struct {
	Role    MessageRole    `json:"role"`
	Content []ContentBlock `json:"content"`
}

// Usage Token 使用统计
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ToolDefinition 工具定义 (Anthropic 格式)
type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// OpenAIToolDefinition OpenAI 兼容的工具定义
type OpenAIToolDefinition struct {
	Type     string          `json:"type"`
	Function *FunctionSchema `json:"function,omitempty"`
}

// FunctionSchema 函数 schema
type FunctionSchema struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// APIRequest API 请求
type APIRequest struct {
	Model      string           `json:"model"`
	MaxTokens  int              `json:"max_tokens"`
	Messages   []Message        `json:"messages"`
	Tools      []ToolDefinition `json:"tools,omitempty"`
	ToolChoice string           `json:"tool_choice,omitempty"`
	Stream     bool             `json:"stream"`
	System     string           `json:"system,omitempty"`
}

// APIResponse API 响应 (Anthropic 格式)
type APIResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type  string `json:"type"`
		Text  string `json:"text,omitempty"`
		ID    string `json:"id,omitempty"`
		Name  string `json:"name,omitempty"`
		Input struct {
			json.RawMessage
		} `json:"input,omitempty"`
	} `json:"content"`
	Model string `json:"model"`
	Usage Usage  `json:"usage"`

	// OpenAI 兼容字段
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role             string `json:"role"`
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content,omitempty"`
			ToolCalls        []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls,omitempty"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices,omitempty"`
}

// StreamEvent 流事件
type StreamEvent struct {
	Type    string          `json:"type"`
	Index   int             `json:"index,omitempty"`
	Delta   json.RawMessage `json:"delta,omitempty"`
	Message json.RawMessage `json:"message,omitempty"`
}

// Client API 客户端
type Client struct {
	HTTPClient *http.Client
	APIKey     string
	BaseURL    string
	Model      string
	MaxTokens  int
	AuthType   string // "bearer", "header", "query"
	AuthHeader string // 认证头名称
	Version    string // API 版本
	Endpoint   string // API endpoint 路径
	Provider   string // "anthropic", "openai", "alibaba"
}

// NewClient 创建新客户端
func NewClient(apiKey, model string) *Client {
	baseURL := os.Getenv("ANTHROPIC_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}

	maxTokens := 4096
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	return &Client{
		HTTPClient: &http.Client{Timeout: 120 * time.Second},
		APIKey:     apiKey,
		BaseURL:    baseURL,
		Model:      model,
		MaxTokens:  maxTokens,
	}
}

// NewClientWithConfig 使用自定义配置创建客户端
func NewClientWithConfig(apiKey, model, baseURL string) *Client {
	return NewClientWithFullConfig(apiKey, model, baseURL, "bearer", "Authorization", "")
}

// NewClientWithFullConfig 使用完整配置创建客户端
func NewClientWithFullConfig(apiKey, model, baseURL, authType, authHeader, version string) *Client {
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}

	maxTokens := 4096
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	if authType == "" {
		authType = "header"
	}

	if authHeader == "" {
		authHeader = "x-api-key"
	}

	// 根据 BaseURL 设置 endpoint 和 provider
	endpoint := "/v1/messages"
	provider := "anthropic"
	if baseURL != "" {
		if strings.Contains(baseURL, "dashscope") {
			// Alibaba uses /chat/completions
			endpoint = "/chat/completions"
			provider = "alibaba"
		} else if strings.Contains(baseURL, "openai") {
			endpoint = "/chat/completions"
			provider = "openai"
		}
	}

	return &Client{
		HTTPClient: &http.Client{Timeout: 120 * time.Second},
		APIKey:     apiKey,
		BaseURL:    baseURL,
		Model:      model,
		MaxTokens:  maxTokens,
		AuthType:   authType,
		AuthHeader: authHeader,
		Version:    version,
		Endpoint:   endpoint,
		Provider:   provider,
	}
}

// SendMessage 发送消息
func (c *Client) SendMessage(messages []Message, tools []ToolDefinition) (*APIResponse, error) {
	reqBody := APIRequest{
		Model:      c.Model,
		MaxTokens:  c.MaxTokens,
		Messages:   messages,
		Tools:      tools,
		ToolChoice: "auto",
		Stream:     false,
	}

	return c.sendRequest(reqBody)
}

// convertToolsForProvider 转换 tools 格式以适配不同 Provider
func (c *Client) convertToolsForProvider(tools []ToolDefinition) interface{} {
	if c.Provider == "anthropic" {
		// Anthropic 格式：直接使用
		return tools
	}

	// OpenAI/Alibaba 格式
	openaiTools := make([]OpenAIToolDefinition, 0, len(tools))
	for _, tool := range tools {
		// Anthropic 的 input_schema 就是 OpenAI 的 parameters
		openaiTools = append(openaiTools, OpenAIToolDefinition{
			Type: "function",
			Function: &FunctionSchema{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.InputSchema,
			},
		})
	}
	return openaiTools
}

// SendMessageStream 发送流式消息
func (c *Client) SendMessageStream(messages []Message, tools []ToolDefinition) (<-chan StreamEvent, error) {
	reqBody := APIRequest{
		Model:      c.Model,
		MaxTokens:  c.MaxTokens,
		Messages:   messages,
		Tools:      tools,
		ToolChoice: "auto",
		Stream:     true,
	}

	return c.sendStreamRequest(reqBody)
}

func (c *Client) sendRequest(reqBody APIRequest) (*APIResponse, error) {
	// 转换 tools 格式以适配不同 Provider
	if reqBody.Tools != nil && c.Provider != "anthropic" {
		// 创建一个临时结构体用于 JSON 序列化
		type OpenAIRequest struct {
			Model      string      `json:"model"`
			MaxTokens  int         `json:"max_tokens,omitempty"`
			Messages   []Message   `json:"messages"`
			Tools      interface{} `json:"tools,omitempty"`
			ToolChoice interface{} `json:"tool_choice,omitempty"`
			Stream     bool        `json:"stream"`
			System     string      `json:"system,omitempty"`
		}

		convertedTools := c.convertToolsForProvider(reqBody.Tools)

		// OpenAI 格式的 tool_choice
		var toolChoice interface{} = "auto"

		openaiReq := OpenAIRequest{
			Model:      reqBody.Model,
			MaxTokens:  reqBody.MaxTokens,
			Messages:   reqBody.Messages,
			Tools:      convertedTools,
			ToolChoice: toolChoice,
			Stream:     reqBody.Stream,
			System:     reqBody.System,
		}

		jsonData, err := json.Marshal(openaiReq)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		return c.sendHTTPRequest(jsonData)
	}

	// Anthropic 格式
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	return c.sendHTTPRequest(jsonData)
}

// sendHTTPRequest 发送 HTTP 请求
func (c *Client) sendHTTPRequest(jsonData []byte) (*APIResponse, error) {
	endpoint := c.Endpoint
	if endpoint == "" {
		endpoint = "/v1/messages"
	}

	req, err := http.NewRequest("POST", c.BaseURL+endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// 根据认证类型设置认证头
	switch c.AuthType {
	case "bearer":
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	case "header":
		req.Header.Set(c.AuthHeader, c.APIKey)
		if c.Version != "" {
			req.Header.Set("anthropic-version", c.Version)
		}
	case "query":
		// Query 参数认证在 URL 中添加
		req.URL.RawQuery = fmt.Sprintf("%s=%s", c.AuthHeader, c.APIKey)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &apiResp, nil
}

func (c *Client) sendStreamRequest(reqBody APIRequest) (<-chan StreamEvent, error) {
	// 转换 tools 格式以适配不同 Provider
	if reqBody.Tools != nil && c.Provider != "anthropic" {
		type OpenAIRequest struct {
			Model      string      `json:"model"`
			MaxTokens  int         `json:"max_tokens,omitempty"`
			Messages   []Message   `json:"messages"`
			Tools      interface{} `json:"tools,omitempty"`
			ToolChoice interface{} `json:"tool_choice,omitempty"`
			Stream     bool        `json:"stream"`
			System     string      `json:"system,omitempty"`
		}

		convertedTools := c.convertToolsForProvider(reqBody.Tools)
		var toolChoice interface{} = "auto"

		openaiReq := OpenAIRequest{
			Model:      reqBody.Model,
			MaxTokens:  reqBody.MaxTokens,
			Messages:   reqBody.Messages,
			Tools:      convertedTools,
			ToolChoice: toolChoice,
			Stream:     reqBody.Stream,
			System:     reqBody.System,
		}

		jsonData, err := json.Marshal(openaiReq)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		return c.sendStreamHTTPRequest(jsonData)
	}

	// Anthropic 格式
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	return c.sendStreamHTTPRequest(jsonData)
}

// sendStreamHTTPRequest 发送流式 HTTP 请求
func (c *Client) sendStreamHTTPRequest(jsonData []byte) (<-chan StreamEvent, error) {
	endpoint := c.Endpoint
	if endpoint == "" {
		endpoint = "/v1/messages"
	}

	req, err := http.NewRequest("POST", c.BaseURL+endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// 根据认证类型设置认证头
	switch c.AuthType {
	case "bearer":
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	case "header":
		req.Header.Set(c.AuthHeader, c.APIKey)
		if c.Version != "" {
			req.Header.Set("anthropic-version", c.Version)
		}
	case "query":
		req.URL.RawQuery = fmt.Sprintf("%s=%s", c.AuthHeader, c.APIKey)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	eventChan := make(chan StreamEvent, 100)
	go func() {
		defer close(eventChan)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				if data == "[DONE]" {
					break
				}

				var event StreamEvent
				if err := json.Unmarshal([]byte(data), &event); err == nil {
					eventChan <- event
				}
			}
		}
	}()

	return eventChan, nil
}

// GetAPIKey 从环境变量获取 API Key
func GetAPIKey() string {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		key = os.Getenv("CLAUDE_API_KEY")
	}
	return key
}

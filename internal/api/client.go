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

// ToolDefinition 工具定义
type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
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

// APIResponse API 响应
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
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/v1/messages", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

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
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/v1/messages", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

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

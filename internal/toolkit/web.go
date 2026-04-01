package toolkit

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// WebSearchTool 网络搜索工具
type WebSearchTool struct {
	searchEngine string
	apiKey       string
}

// NewWebSearchTool 创建搜索工具
func NewWebSearchTool(apiKey string) *WebSearchTool {
	return &WebSearchTool{
		searchEngine: "google",
		apiKey:       apiKey,
	}
}

// WebSearchInput 搜索输入
type WebSearchInput struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

// Execute 执行搜索
func (t *WebSearchTool) Execute(input json.RawMessage) (string, error) {
	var inp WebSearchInput
	if err := json.Unmarshal(input, &inp); err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	if inp.Query == "" {
		return "", fmt.Errorf("query is required")
	}

	if inp.Limit == 0 {
		inp.Limit = 10
	}

	// 简单实现 - 实际应该调用搜索 API
	return fmt.Sprintf("Search results for: %s (limit: %d)", inp.Query, inp.Limit), nil
}

// GetDescription 获取描述
func (t *WebSearchTool) GetDescription() string {
	return "Search the web for information"
}

// GetInputSchema 获取输入 schema
func (t *WebSearchTool) GetInputSchema() string {
	return `{"type":"object","properties":{"query":{"type":"string"},"limit":{"type":"integer"}},"required":["query"]}`
}

// WebFetchTool 网页抓取工具
type WebFetchTool struct {
	client *http.Client
}

// NewWebFetchTool 创建抓取工具
func NewWebFetchTool() *WebFetchTool {
	return &WebFetchTool{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// WebFetchInput 抓取输入
type WebFetchInput struct {
	URL string `json:"url"`
}

// Execute 执行抓取
func (t *WebFetchTool) Execute(input json.RawMessage) (string, error) {
	var inp WebFetchInput
	if err := json.Unmarshal(input, &inp); err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	if inp.URL == "" {
		return "", fmt.Errorf("url is required")
	}

	// 验证 URL
	parsedURL, err := url.Parse(inp.URL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	// 只允许 HTTP/HTTPS
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", fmt.Errorf("only HTTP and HTTPS URLs are allowed")
	}

	resp, err := t.client.Get(inp.URL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	content := string(body)
	// 限制返回大小
	if len(content) > 50000 {
		content = content[:50000] + "... (truncated)"
	}

	return content, nil
}

// GetDescription 获取描述
func (t *WebFetchTool) GetDescription() string {
	return "Fetch content from a URL"
}

// GetInputSchema 获取输入 schema
func (t *WebFetchTool) GetInputSchema() string {
	return `{"type":"object","properties":{"url":{"type":"string"}},"required":["url"]}`
}

// TodoWriteTool Todo 列表工具
type TodoWriteTool struct {
	todos []TodoItem
}

// TodoItem Todo 项
type TodoItem struct {
	ID       int    `json:"id"`
	Content  string `json:"content"`
	Status   string `json:"status"`   // pending, completed
	Priority string `json:"priority"` // low, medium, high
}

// NewTodoWriteTool 创建 Todo 工具
func NewTodoWriteTool() *TodoWriteTool {
	return &TodoWriteTool{
		todos: make([]TodoItem, 0),
	}
}

// TodoWriteInput Todo 输入
type TodoWriteInput struct {
	Todos []TodoItem `json:"todos"`
}

// Execute 执行
func (t *TodoWriteTool) Execute(input json.RawMessage) (string, error) {
	var inp TodoWriteInput
	if err := json.Unmarshal(input, &inp); err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	t.todos = inp.Todos

	var sb strings.Builder
	sb.WriteString("Todo list updated:\n")
	for i, todo := range t.todos {
		status := "⬜"
		if todo.Status == "completed" {
			status = "✅"
		}
		sb.WriteString(fmt.Sprintf("%d. %s %s", i+1, status, todo.Content))
		if todo.Priority != "" {
			sb.WriteString(fmt.Sprintf(" [%s]", todo.Priority))
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// GetDescription 获取描述
func (t *TodoWriteTool) GetDescription() string {
	return "Create and update todo lists"
}

// GetInputSchema 获取输入 schema
func (t *TodoWriteTool) GetInputSchema() string {
	return `{
		"type":"object",
		"properties":{
			"todos":{
				"type":"array",
				"items":{
					"type":"object",
					"properties":{
						"id":{"type":"integer"},
						"content":{"type":"string"},
						"status":{"type":"string","enum":["pending","completed"]},
						"priority":{"type":"string","enum":["low","medium","high"]}
					},
					"required":["id","content"]
				}
			}
		},
		"required":["todos"]
	}`
}

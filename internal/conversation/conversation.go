package conversation

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gclawcoder/gclaw/internal/api"
	"github.com/gclawcoder/gclaw/internal/permissions"
	"github.com/gclawcoder/gclaw/internal/toolkit"
)

// MessageRole 消息角色
type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleSystem    MessageRole = "system"
	RoleTool      MessageRole = "tool"
)

// ContentBlockType 内容块类型
type ContentBlockType string

const (
	BlockTypeText       ContentBlockType = "text"
	BlockTypeToolUse    ContentBlockType = "tool_use"
	BlockTypeToolResult ContentBlockType = "tool_result"
)

// ContentBlock 内容块
type ContentBlock struct {
	Type      ContentBlockType `json:"type"`
	Text      string           `json:"text,omitempty"`
	ID        string           `json:"id,omitempty"`
	Name      string           `json:"name,omitempty"`
	Input     json.RawMessage  `json:"input,omitempty"`
	ToolUseID string           `json:"tool_use_id,omitempty"`
	Content   string           `json:"content,omitempty"`
	IsError   bool             `json:"is_error,omitempty"`
}

// ConversationMessage 对话消息
type ConversationMessage struct {
	Role    MessageRole    `json:"role"`
	Content []ContentBlock `json:"content"`
}

// GetSession 获取会话（用于 REPL）
func (r *ConversationRuntime) Session() *Session {
	return r.session
}

// Session 会话
type Session struct {
	ID        string                `json:"id"`
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`
	Messages  []ConversationMessage `json:"messages"`
	Model     string                `json:"model"`
}

// NewSession 创建新会话
func NewSession() *Session {
	return &Session{
		ID:        generateSessionID(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Messages:  make([]ConversationMessage, 0),
		Model:     "claude-sonnet-4-20250514",
	}
}

// AddUserMessage 添加用户消息
func (s *Session) AddUserMessage(text string) {
	s.Messages = append(s.Messages, ConversationMessage{
		Role: RoleUser,
		Content: []ContentBlock{
			{Type: BlockTypeText, Text: text},
		},
	})
	s.UpdatedAt = time.Now()
}

// AddAssistantMessage 添加助手消息
func (s *Session) AddAssistantMessage(blocks []ContentBlock) {
	s.Messages = append(s.Messages, ConversationMessage{
		Role:    RoleAssistant,
		Content: blocks,
	})
	s.UpdatedAt = time.Now()
}

// AddToolResult 添加工具结果
func (s *Session) AddToolResult(toolUseID, toolName, content string, isError bool) {
	s.Messages = append(s.Messages, ConversationMessage{
		Role: RoleTool,
		Content: []ContentBlock{
			{
				Type:      BlockTypeToolResult,
				ToolUseID: toolUseID,
				Content:   content,
				IsError:   isError,
			},
		},
	})
	s.UpdatedAt = time.Now()
}

// ConversationRuntime 对话运行时
type ConversationRuntime struct {
	session          *Session
	apiClient        *api.Client
	toolRegistry     *toolkit.Registry
	permissionPolicy *permissions.PermissionPolicy
	systemPrompt     string
	maxIterations    int
	currentIteration int
}

// NewConversationRuntime 创建对话运行时
func NewConversationRuntime(
	session *Session,
	apiClient *api.Client,
	toolRegistry *toolkit.Registry,
	permissionPolicy *permissions.PermissionPolicy,
	systemPrompt string,
) *ConversationRuntime {
	return &ConversationRuntime{
		session:          session,
		apiClient:        apiClient,
		toolRegistry:     toolRegistry,
		permissionPolicy: permissionPolicy,
		systemPrompt:     systemPrompt,
		maxIterations:    100,
	}
}

// RunTurn 运行一轮对话
func (r *ConversationRuntime) RunTurn(userInput string) (*TurnResult, error) {
	r.session.AddUserMessage(userInput)
	r.currentIteration = 0

	var assistantMessages []ConversationMessage
	var toolResults []ConversationMessage

	for {
		r.currentIteration++
		if r.currentIteration > r.maxIterations {
			return nil, fmt.Errorf("max iterations (%d) reached", r.maxIterations)
		}

		// 调用 API
		messages := r.buildAPIMessages()
		tools := r.buildToolDefinitions()

		resp, err := r.apiClient.SendMessage(messages, tools)
		if err != nil {
			return nil, fmt.Errorf("API error: %w", err)
		}

		// 解析响应
		blocks := r.parseResponse(resp)
		r.session.AddAssistantMessage(blocks)
		assistantMessages = append(assistantMessages, ConversationMessage{
			Role:    RoleAssistant,
			Content: blocks,
		})

		// 检查是否有工具调用
		pendingTools := r.extractToolUses(blocks)
		if len(pendingTools) == 0 {
			break
		}

		// 执行工具
		for _, toolUse := range pendingTools {
			outcome := r.permissionPolicy.Authorize(toolUse.Name, string(toolUse.Input), nil)

			var result string
			var isError bool

			if outcome == permissions.OutcomeDeny {
				result = "Tool use denied by permission policy"
				isError = true
			} else {
				result, err = r.toolRegistry.Execute(toolUse.Name, toolUse.Input)
				if err != nil {
					result = fmt.Sprintf("Tool execution error: %v", err)
					isError = true
				}
			}

			r.session.AddToolResult(toolUse.ID, toolUse.Name, result, isError)
			toolResults = append(toolResults, ConversationMessage{
				Role: RoleTool,
				Content: []ContentBlock{
					{
						Type:      BlockTypeToolResult,
						ToolUseID: toolUse.ID,
						Content:   result,
						IsError:   isError,
					},
				},
			})
		}
	}

	return &TurnResult{
		Session:           r.session,
		AssistantMessages: assistantMessages,
		ToolResults:       toolResults,
		Iterations:        r.currentIteration,
	}, nil
}

// TurnResult 轮次结果
type TurnResult struct {
	Session           *Session
	AssistantMessages []ConversationMessage
	ToolResults       []ConversationMessage
	Iterations        int
}

// ToolUse 工具调用
type ToolUse struct {
	ID    string
	Name  string
	Input json.RawMessage
}

func (r *ConversationRuntime) buildAPIMessages() []api.Message {
	messages := make([]api.Message, 0, len(r.session.Messages))

	for _, msg := range r.session.Messages {
		apiMsg := api.Message{
			Role:    api.MessageRole(msg.Role),
			Content: make([]api.ContentBlock, len(msg.Content)),
		}

		for i, block := range msg.Content {
			apiMsg.Content[i] = api.ContentBlock{
				Type:  string(block.Type),
				Text:  block.Text,
				ID:    block.ID,
				Name:  block.Name,
				Input: block.Input,
			}
		}

		messages = append(messages, apiMsg)
	}

	return messages
}

func (r *ConversationRuntime) buildToolDefinitions() []api.ToolDefinition {
	toolNames := r.toolRegistry.List()
	definitions := make([]api.ToolDefinition, 0, len(toolNames))

	// 简化实现，实际应该从工具获取 schema
	for _, name := range toolNames {
		definitions = append(definitions, api.ToolDefinition{
			Name:        name,
			Description: "Tool description",
			InputSchema: json.RawMessage(`{"type":"object"}`),
		})
	}

	return definitions
}

func (r *ConversationRuntime) parseResponse(resp *api.APIResponse) []ContentBlock {
	blocks := make([]ContentBlock, 0, len(resp.Content))

	for _, c := range resp.Content {
		block := ContentBlock{
			Type: ContentBlockType(c.Type),
			Text: c.Text,
		}

		if c.Type == "tool_use" {
			block.ID = c.ID
			block.Name = c.Name
			block.Input = c.Input.RawMessage
		}

		blocks = append(blocks, block)
	}

	return blocks
}

func (r *ConversationRuntime) extractToolUses(blocks []ContentBlock) []ToolUse {
	var toolUses []ToolUse

	for _, block := range blocks {
		if block.Type == BlockTypeToolUse {
			toolUses = append(toolUses, ToolUse{
				ID:    block.ID,
				Name:  block.Name,
				Input: block.Input,
			})
		}
	}

	return toolUses
}

func generateSessionID() string {
	return fmt.Sprintf("session-%d", time.Now().UnixNano())
}

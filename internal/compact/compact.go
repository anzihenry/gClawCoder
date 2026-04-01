package compact

import (
	"fmt"
	"strings"

	"github.com/gclawcoder/gclaw/internal/conversation"
)

// CompactionConfig 压缩配置
type CompactionConfig struct {
	MaxTurns        int     `json:"maxTurns"`
	MaxTokens       int     `json:"maxTokens"`
	KeepLastTurns   int     `json:"keepLastTurns"`
	CompressionRate float64 `json:"compressionRate"`
}

// DefaultCompactionConfig 默认配置
func DefaultCompactionConfig() CompactionConfig {
	return CompactionConfig{
		MaxTurns:        50,
		MaxTokens:       100000,
		KeepLastTurns:   10,
		CompressionRate: 0.5,
	}
}

// CompactionResult 压缩结果
type CompactionResult struct {
	OriginalMessages  int    `json:"originalMessages"`
	CompactedMessages int    `json:"compactedMessages"`
	TokensSaved       int    `json:"tokensSaved"`
	Summary           string `json:"summary"`
}

// Compactor 会话压缩器
type Compactor struct {
	config CompactionConfig
}

// NewCompactor 创建压缩器
func NewCompactor(config CompactionConfig) *Compactor {
	return &Compactor{
		config: config,
	}
}

// Compact 压缩会话
func (c *Compactor) Compact(session *conversation.Session) (*CompactionResult, error) {
	messages := session.Messages
	originalCount := len(messages)

	if originalCount <= c.config.KeepLastTurns {
		return &CompactionResult{
			OriginalMessages:  originalCount,
			CompactedMessages: originalCount,
			TokensSaved:       0,
			Summary:           "No compaction needed",
		}, nil
	}

	// 保留最后的 N 轮对话
	keepCount := c.config.KeepLastTurns * 2 // 每轮包含 user + assistant
	if keepCount > originalCount {
		keepCount = originalCount
	}

	// 生成摘要
	summary := c.generateSummary(messages[:originalCount-keepCount])

	// 创建压缩后的消息
	compactedMessages := []conversation.ConversationMessage{
		{
			Role: conversation.RoleSystem,
			Content: []conversation.ContentBlock{
				{
					Type: conversation.BlockTypeText,
					Text: fmt.Sprintf("[Conversation Summary]\n%s", summary),
				},
			},
		},
	}

	// 添加保留的消息
	compactedMessages = append(compactedMessages, messages[originalCount-keepCount:]...)

	// 更新会话
	session.Messages = compactedMessages

	tokensSaved := c.estimateTokens(messages[:originalCount-keepCount])

	return &CompactionResult{
		OriginalMessages:  originalCount,
		CompactedMessages: len(compactedMessages),
		TokensSaved:       tokensSaved,
		Summary:           summary,
	}, nil
}

// ShouldCompact 检查是否需要压缩
func (c *Compactor) ShouldCompact(session *conversation.Session) bool {
	return len(session.Messages) > c.config.MaxTurns
}

// generateSummary 生成对话摘要
func (c *Compactor) generateSummary(messages []conversation.ConversationMessage) string {
	var summary strings.Builder

	summary.WriteString("Previous conversation covered the following topics:\n\n")

	// 提取关键信息
	var topics []string
	var toolsUsed []string

	for _, msg := range messages {
		for _, block := range msg.Content {
			if block.Type == conversation.BlockTypeText {
				// 简单提取主题
				text := block.Text
				if len(text) > 100 {
					text = text[:100] + "..."
				}
				topics = append(topics, text)
			}
			if block.Type == conversation.BlockTypeToolUse {
				toolsUsed = append(toolsUsed, block.Name)
			}
		}
	}

	// 限制摘要长度
	if len(topics) > 5 {
		topics = topics[:5]
	}

	if len(topics) > 0 {
		summary.WriteString("Topics discussed:\n")
		for i, topic := range topics {
			summary.WriteString(fmt.Sprintf("  %d. %s\n", i+1, topic))
		}
		summary.WriteString("\n")
	}

	if len(toolsUsed) > 0 {
		// 去重
		seen := make(map[string]bool)
		var uniqueTools []string
		for _, tool := range toolsUsed {
			if !seen[tool] {
				seen[tool] = true
				uniqueTools = append(uniqueTools, tool)
			}
		}
		summary.WriteString(fmt.Sprintf("Tools used: %s\n", strings.Join(uniqueTools, ", ")))
	}

	return summary.String()
}

// estimateTokens 估算 token 数
func (c *Compactor) estimateTokens(messages []conversation.ConversationMessage) int {
	total := 0
	for _, msg := range messages {
		for _, block := range msg.Content {
			// 简单估算：每 4 个字符约 1 个 token
			total += len(block.Text) / 4
			if block.Input != nil {
				total += len(block.Input) / 4
			}
		}
	}
	return total
}

// CompactToString 压缩为字符串表示
func CompactToString(session *conversation.Session, keepLast int) string {
	if len(session.Messages) <= keepLast {
		return "No compaction needed"
	}

	messages := session.Messages[keepLast:]
	var summary strings.Builder

	summary.WriteString(fmt.Sprintf("[Previous %d messages compacted]\n", len(messages)))

	return summary.String()
}

// GetSessionStats 获取会话统计
func GetSessionStats(session *conversation.Session) map[string]interface{} {
	userMessages := 0
	assistantMessages := 0
	toolMessages := 0
	totalChars := 0

	for _, msg := range session.Messages {
		switch msg.Role {
		case conversation.RoleUser:
			userMessages++
		case conversation.RoleAssistant:
			assistantMessages++
		case conversation.RoleTool:
			toolMessages++
		}

		for _, block := range msg.Content {
			totalChars += len(block.Text)
		}
	}

	return map[string]interface{}{
		"total_messages":     len(session.Messages),
		"user_messages":      userMessages,
		"assistant_messages": assistantMessages,
		"tool_messages":      toolMessages,
		"total_chars":        totalChars,
		"estimated_tokens":   totalChars / 4,
	}
}

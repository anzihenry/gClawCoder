package api

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ThinkingBlock 思考块
type ThinkingBlock struct {
	Type      string `json:"type"` // thinking
	Thinking  string `json:"thinking"`
	Signature string `json:"signature,omitempty"`
}

// ExtendedThinkingConfig 扩展思考配置
type ExtendedThinkingConfig struct {
	Enabled      bool `json:"enabled"`
	BudgetTokens int  `json:"budgetTokens"`
	MinTokens    int  `json:"minTokens"`
	MaxTokens    int  `json:"maxTokens"`
}

// DefaultExtendedThinkingConfig 默认配置
func DefaultExtendedThinkingConfig() ExtendedThinkingConfig {
	return ExtendedThinkingConfig{
		Enabled:      true,
		BudgetTokens: 16000,
		MinTokens:    1000,
		MaxTokens:    32000,
	}
}

// ParseThinkingBlock 解析思考块
func ParseThinkingBlock(content json.RawMessage) (*ThinkingBlock, error) {
	var block ThinkingBlock
	if err := json.Unmarshal(content, &block); err != nil {
		return nil, err
	}

	if block.Type != "thinking" {
		return nil, fmt.Errorf("not a thinking block: %s", block.Type)
	}

	return &block, nil
}

// FormatThinking 格式化思考内容输出
func FormatThinking(thinking string, maxLength int) string {
	if maxLength <= 0 {
		maxLength = 500
	}

	lines := strings.Split(thinking, "\n")
	var formatted strings.Builder

	formatted.WriteString("🤔 Thinking:\n")
	formatted.WriteString("─────────────────────────────────────\n")

	lineCount := 0
	for _, line := range lines {
		if lineCount >= maxLength/10 {
			formatted.WriteString(fmt.Sprintf("\n... (%d more lines)", len(lines)-lineCount))
			break
		}
		formatted.WriteString("  " + line + "\n")
		lineCount++
	}

	formatted.WriteString("─────────────────────────────────────\n")

	return formatted.String()
}

// ThinkingSummary 生成思考摘要
func ThinkingSummary(thinking string, maxLen int) string {
	if len(thinking) <= maxLen {
		return thinking
	}

	// 提取关键句子
	sentences := strings.Split(thinking, ".")
	var summary strings.Builder

	currentLen := 0
	for i, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}

		if currentLen+len(sentence) > maxLen && i > 0 {
			summary.WriteString("...")
			break
		}

		if i > 0 {
			summary.WriteString(". ")
		}
		summary.WriteString(sentence)
		currentLen += len(sentence)
	}

	return summary.String()
}

// ExtractThinkingFromContent 从内容中提取思考块
func ExtractThinkingFromContent(content []ContentBlock) []*ThinkingBlock {
	var thinkingBlocks []*ThinkingBlock

	for _, block := range content {
		if block.Type == "thinking" {
			thinkingBlocks = append(thinkingBlocks, &ThinkingBlock{
				Type:      block.Type,
				Thinking:  block.Text,
				Signature: block.ID,
			})
		}
	}

	return thinkingBlocks
}

// HasThinkingContent 检查是否有思考内容
func HasThinkingContent(content []ContentBlock) bool {
	for _, block := range content {
		if block.Type == "thinking" {
			return true
		}
	}
	return false
}

// GetThinkingContent 获取思考内容
func GetThinkingContent(content []ContentBlock) string {
	var thinking strings.Builder

	for _, block := range content {
		if block.Type == "thinking" {
			if thinking.Len() > 0 {
				thinking.WriteString("\n\n")
			}
			thinking.WriteString(block.Text)
		}
	}

	return thinking.String()
}

// GetNonThinkingContent 获取非思考内容
func GetNonThinkingContent(content []ContentBlock) []ContentBlock {
	var result []ContentBlock

	for _, block := range content {
		if block.Type != "thinking" {
			result = append(result, block)
		}
	}

	return result
}

// ThinkingStats 思考统计
type ThinkingStats struct {
	ThinkingBlocks  int
	TotalChars      int
	EstimatedTokens int
}

// GetThinkingStats 获取思考统计
func GetThinkingStats(content []ContentBlock) ThinkingStats {
	stats := ThinkingStats{}

	for _, block := range content {
		if block.Type == "thinking" {
			stats.ThinkingBlocks++
			stats.TotalChars += len(block.Text)
		}
	}

	// 估算 token：每 4 字符约 1 token
	stats.EstimatedTokens = stats.TotalChars / 4

	return stats
}

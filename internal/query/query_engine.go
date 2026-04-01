package query

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gclawcoder/gclaw/internal/commands"
	"github.com/gclawcoder/gclaw/internal/models"
	"github.com/gclawcoder/gclaw/internal/session"
	"github.com/gclawcoder/gclaw/internal/tools"
	"github.com/gclawcoder/gclaw/internal/transcript"
)

// QueryEngineConfig 查询引擎配置
type QueryEngineConfig struct {
	MaxTurns             int
	MaxBudgetTokens      int
	CompactAfterTurns    int
	StructuredOutput     bool
	StructuredRetryLimit int
}

// DefaultConfig 默认配置
func DefaultConfig() QueryEngineConfig {
	return QueryEngineConfig{
		MaxTurns:             8,
		MaxBudgetTokens:      2000,
		CompactAfterTurns:    12,
		StructuredOutput:     false,
		StructuredRetryLimit: 2,
	}
}

// TurnResult 轮次结果
type TurnResult struct {
	Prompt            string
	Output            string
	MatchedCommands   []string
	MatchedTools      []string
	PermissionDenials []models.PermissionDenial
	Usage             models.UsageSummary
	StopReason        string
}

// QueryEnginePort 查询引擎端口
type QueryEnginePort struct {
	Manifest          interface{}
	Config            QueryEngineConfig
	SessionID         string
	MutableMessages   []string
	PermissionDenials []models.PermissionDenial
	TotalUsage        models.UsageSummary
	TranscriptStore   *transcript.TranscriptStore
	mu                sync.RWMutex
}

// NewQueryEnginePort 创建新的查询引擎
func NewQueryEnginePort() *QueryEnginePort {
	return &QueryEnginePort{
		Manifest:          nil,
		Config:            DefaultConfig(),
		SessionID:         generateSessionID(),
		MutableMessages:   make([]string, 0),
		PermissionDenials: make([]models.PermissionDenial, 0),
		TotalUsage:        models.UsageSummary{},
		TranscriptStore:   transcript.NewTranscriptStore(),
	}
}

// FromWorkspace 从工作区创建查询引擎
func FromWorkspace() *QueryEnginePort {
	return NewQueryEnginePort()
}

// FromSavedSession 从已保存的会话创建查询引擎
func FromSavedSession(sessionID string) (*QueryEnginePort, error) {
	storedSession, err := session.LoadSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load session: %w", err)
	}

	transcriptStore := transcript.NewTranscriptStore()
	for _, msg := range storedSession.Messages {
		transcriptStore.Append(msg)
	}
	transcriptStore.Flush()

	return &QueryEnginePort{
		Config:          DefaultConfig(),
		SessionID:       storedSession.SessionID,
		MutableMessages: storedSession.Messages,
		TotalUsage:      models.UsageSummary{InputTokens: storedSession.InputTokens, OutputTokens: storedSession.OutputTokens},
		TranscriptStore: transcriptStore,
	}, nil
}

// SubmitMessage 提交消息
func (q *QueryEnginePort) SubmitMessage(prompt string, matchedCommands, matchedTools []string, deniedTools []models.PermissionDenial) TurnResult {
	q.mu.Lock()
	defer q.mu.Unlock()

	// 检查是否达到最大轮数
	if len(q.MutableMessages) >= q.Config.MaxTurns {
		return TurnResult{
			Prompt:            prompt,
			Output:            fmt.Sprintf("Max turns reached before processing prompt: %s", prompt),
			MatchedCommands:   matchedCommands,
			MatchedTools:      matchedTools,
			PermissionDenials: deniedTools,
			Usage:             q.TotalUsage,
			StopReason:        "max_turns_reached",
		}
	}

	// 构建输出
	summaryLines := q.buildSummaryLines(prompt, matchedCommands, matchedTools, deniedTools)
	output := q.formatOutput(summaryLines)

	// 计算使用量
	projectedUsage := q.TotalUsage.AddTurn(prompt, output)
	stopReason := "completed"

	if projectedUsage.TotalTokens() > q.Config.MaxBudgetTokens {
		stopReason = "max_budget_reached"
	}

	// 更新状态
	q.MutableMessages = append(q.MutableMessages, prompt)
	q.TranscriptStore.Append(prompt)
	q.PermissionDenials = append(q.PermissionDenials, deniedTools...)
	q.TotalUsage = projectedUsage

	// 压缩消息
	q.CompactMessagesIfNeeded()

	return TurnResult{
		Prompt:            prompt,
		Output:            output,
		MatchedCommands:   matchedCommands,
		MatchedTools:      matchedTools,
		PermissionDenials: deniedTools,
		Usage:             q.TotalUsage,
		StopReason:        stopReason,
	}
}

// StreamSubmitMessage 流式提交消息
func (q *QueryEnginePort) StreamSubmitMessage(prompt string, matchedCommands, matchedTools []string, deniedTools []models.PermissionDenial) <-chan map[string]interface{} {
	ch := make(chan map[string]interface{}, 8)

	go func() {
		defer close(ch)

		ch <- map[string]interface{}{
			"type":       "message_start",
			"session_id": q.SessionID,
			"prompt":     prompt,
		}

		if len(matchedCommands) > 0 {
			ch <- map[string]interface{}{
				"type":     "command_match",
				"commands": matchedCommands,
			}
		}

		if len(matchedTools) > 0 {
			ch <- map[string]interface{}{
				"type":  "tool_match",
				"tools": matchedTools,
			}
		}

		if len(deniedTools) > 0 {
			denialNames := make([]string, len(deniedTools))
			for i, d := range deniedTools {
				denialNames[i] = d.ToolName
			}
			ch <- map[string]interface{}{
				"type":    "permission_denial",
				"denials": denialNames,
			}
		}

		result := q.SubmitMessage(prompt, matchedCommands, matchedTools, deniedTools)

		ch <- map[string]interface{}{
			"type": "message_delta",
			"text": result.Output,
		}

		ch <- map[string]interface{}{
			"type": "message_stop",
			"usage": map[string]int{
				"input_tokens":  result.Usage.InputTokens,
				"output_tokens": result.Usage.OutputTokens,
			},
			"stop_reason":     result.StopReason,
			"transcript_size": q.TranscriptStore.Size(),
		}
	}()

	return ch
}

// CompactMessagesIfNeeded 压缩消息（如果需要）
func (q *QueryEnginePort) CompactMessagesIfNeeded() {
	if len(q.MutableMessages) > q.Config.CompactAfterTurns {
		q.MutableMessages = q.MutableMessages[len(q.MutableMessages)-q.Config.CompactAfterTurns:]
	}
	q.TranscriptStore.Compact(q.Config.CompactAfterTurns)
}

// ReplayUserMessages 回放用户消息
func (q *QueryEnginePort) ReplayUserMessages() []string {
	return q.TranscriptStore.Replay()
}

// FlushTranscript 刷新转录
func (q *QueryEnginePort) FlushTranscript() {
	q.TranscriptStore.Flush()
}

// PersistSession 持久化会话
func (q *QueryEnginePort) PersistSession() (string, error) {
	q.FlushTranscript()

	storedSession := &session.StoredSession{
		SessionID:    q.SessionID,
		Messages:     q.MutableMessages,
		InputTokens:  q.TotalUsage.InputTokens,
		OutputTokens: q.TotalUsage.OutputTokens,
	}

	return session.SaveSession(storedSession)
}

// RenderSummary 渲染摘要
func (q *QueryEnginePort) RenderSummary() string {
	commandBacklog := commands.BuildCommandBacklog()
	toolBacklog := tools.BuildToolBacklog()

	var sb strings.Builder
	sb.WriteString("# Go Porting Workspace Summary\n\n")
	sb.WriteString(fmt.Sprintf("Command surface: %d mirrored entries\n", len(commandBacklog.Modules)))
	for i, line := range commandBacklog.SummaryLines() {
		if i >= 10 {
			break
		}
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("Tool surface: %d mirrored entries\n", len(toolBacklog.Modules)))
	for i, line := range toolBacklog.SummaryLines() {
		if i >= 10 {
			break
		}
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("Session id: %s\n", q.SessionID))
	sb.WriteString(fmt.Sprintf("Conversation turns stored: %d\n", len(q.MutableMessages)))
	sb.WriteString(fmt.Sprintf("Permission denials tracked: %d\n", len(q.PermissionDenials)))
	sb.WriteString(fmt.Sprintf("Usage totals: in=%d out=%d\n", q.TotalUsage.InputTokens, q.TotalUsage.OutputTokens))
	sb.WriteString(fmt.Sprintf("Max turns: %d\n", q.Config.MaxTurns))
	sb.WriteString(fmt.Sprintf("Max budget tokens: %d\n", q.Config.MaxBudgetTokens))
	sb.WriteString(fmt.Sprintf("Transcript flushed: %v\n", q.TranscriptStore.Flushed()))

	return sb.String()
}

func (q *QueryEnginePort) buildSummaryLines(prompt string, matchedCommands, matchedTools []string, deniedTools []models.PermissionDenial) []string {
	cmdStr := "none"
	if len(matchedCommands) > 0 {
		cmdStr = strings.Join(matchedCommands, ", ")
	}

	toolStr := "none"
	if len(matchedTools) > 0 {
		toolStr = strings.Join(matchedTools, ", ")
	}

	return []string{
		fmt.Sprintf("Prompt: %s", prompt),
		fmt.Sprintf("Matched commands: %s", cmdStr),
		fmt.Sprintf("Matched tools: %s", toolStr),
		fmt.Sprintf("Permission denials: %d", len(deniedTools)),
	}
}

func (q *QueryEnginePort) formatOutput(summaryLines []string) string {
	if q.Config.StructuredOutput {
		payload := map[string]interface{}{
			"summary":    summaryLines,
			"session_id": q.SessionID,
		}
		data, err := json.MarshalIndent(payload, "", "  ")
		if err == nil {
			return string(data)
		}
	}

	return strings.Join(summaryLines, "\n")
}

func generateSessionID() string {
	return fmt.Sprintf("session-%d", time.Now().UnixNano())
}

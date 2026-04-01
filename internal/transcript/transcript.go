package transcript

import (
	"strconv"
	"strings"
	"sync"
)

// TranscriptEntry 转录条目
type TranscriptEntry struct {
	Content string
}

// TranscriptStore 转录存储
type TranscriptStore struct {
	entries []TranscriptEntry
	flushed bool
	mu      sync.RWMutex
}

// NewTranscriptStore 创建新的转录存储
func NewTranscriptStore() *TranscriptStore {
	return &TranscriptStore{
		entries: make([]TranscriptEntry, 0),
		flushed: false,
	}
}

// Append 添加条目
func (t *TranscriptStore) Append(content string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.entries = append(t.entries, TranscriptEntry{Content: content})
}

// Entries 返回所有条目
func (t *TranscriptStore) Entries() []TranscriptEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]TranscriptEntry, len(t.entries))
	copy(result, t.entries)
	return result
}

// Replay 回放所有消息
func (t *TranscriptStore) Replay() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	messages := make([]string, len(t.entries))
	for i, entry := range t.entries {
		messages[i] = entry.Content
	}
	return messages
}

// Compact 压缩转录（保留最近的 N 条）
func (t *TranscriptStore) Compact(keepCount int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.entries) <= keepCount {
		return
	}

	t.entries = t.entries[len(t.entries)-keepCount:]
}

// Flush 刷新转录
func (t *TranscriptStore) Flush() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.flushed = true
}

// Flushed 检查是否已刷新
func (t *TranscriptStore) Flushed() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.flushed
}

// Size 返回条目数量
func (t *TranscriptStore) Size() int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return len(t.entries)
}

// Clear 清空转录
func (t *TranscriptStore) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.entries = make([]TranscriptEntry, 0)
	t.flushed = false
}

// RenderAsMarkdown 渲染为 Markdown
func (t *TranscriptStore) RenderAsMarkdown() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.entries) == 0 {
		return "## Transcript\n\nNo entries yet.\n"
	}

	var sb strings.Builder
	sb.WriteString("## Transcript\n\n")
	for i, entry := range t.entries {
		sb.WriteString("### Entry ")
		sb.WriteString(strconv.Itoa(i + 1))
		sb.WriteString("\n\n")
		sb.WriteString(entry.Content)
		sb.WriteString("\n\n")
	}

	if t.flushed {
		sb.WriteString("*Transcript flushed to disk.*\n")
	}

	return sb.String()
}

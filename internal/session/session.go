package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// StoredSession 存储的会话
type StoredSession struct {
	SessionID    string   `json:"session_id"`
	Messages     []string `json:"messages"`
	InputTokens  int      `json:"input_tokens"`
	OutputTokens int      `json:"output_tokens"`
}

// SaveSession 保存会话到文件
func SaveSession(session *StoredSession) (string, error) {
	dataDir := "sessions"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create sessions directory: %w", err)
	}

	filePath := filepath.Join(dataDir, session.SessionID+".json")

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal session: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write session file: %w", err)
	}

	return filePath, nil
}

// LoadSession 从文件加载会话
func LoadSession(sessionID string) (*StoredSession, error) {
	dataDir := "sessions"
	filePath := filepath.Join(dataDir, sessionID+".json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	var session StoredSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// ListSessions 列出所有会话
func ListSessions() ([]string, error) {
	dataDir := "sessions"
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read sessions directory: %w", err)
	}

	var sessionIDs []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if len(name) > 5 && name[len(name)-5:] == ".json" {
			sessionIDs = append(sessionIDs, name[:len(name)-5])
		}
	}

	return sessionIDs, nil
}

// NewStoredSession 创建新会话
func NewStoredSession() *StoredSession {
	return &StoredSession{
		SessionID: generateSessionID(),
		Messages:  make([]string, 0),
	}
}

// generateSessionID 生成会话 ID
func generateSessionID() string {
	return fmt.Sprintf("session-%d", time.Now().UnixNano())
}

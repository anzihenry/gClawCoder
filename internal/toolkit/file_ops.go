package toolkit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileReadTool 文件读取工具
type FileReadTool struct{}

// FileReadInput 文件读取输入
type FileReadInput struct {
	Path string `json:"path"`
}

// Execute 执行文件读取
func (t *FileReadTool) Execute(input json.RawMessage) (string, error) {
	var inp FileReadInput
	if err := json.Unmarshal(input, &inp); err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	if inp.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	cleanPath := filepath.Clean(inp.Path)
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("invalid path: path traversal not allowed")
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(data), nil
}

// GetDescription 获取描述
func (t *FileReadTool) GetDescription() string {
	return "Read contents of a file"
}

// GetInputSchema 获取输入 schema
func (t *FileReadTool) GetInputSchema() string {
	return `{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}`
}

// FileWriteTool 文件写入工具
type FileWriteTool struct{}

// FileWriteInput 文件写入输入
type FileWriteInput struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// Execute 执行文件写入
func (t *FileWriteTool) Execute(input json.RawMessage) (string, error) {
	var inp FileWriteInput
	if err := json.Unmarshal(input, &inp); err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	if inp.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	cleanPath := filepath.Clean(inp.Path)
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("invalid path: path traversal not allowed")
	}

	dir := filepath.Dir(cleanPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(cleanPath, []byte(inp.Content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("Successfully wrote to %s", cleanPath), nil
}

// GetDescription 获取描述
func (t *FileWriteTool) GetDescription() string {
	return "Write content to a file"
}

// GetInputSchema 获取输入 schema
func (t *FileWriteTool) GetInputSchema() string {
	return `{"type":"object","properties":{"path":{"type":"string"},"content":{"type":"string"}},"required":["path","content"]}`
}

// FileEditTool 文件编辑工具
type FileEditTool struct{}

// FileEditInput 文件编辑输入
type FileEditInput struct {
	Path      string `json:"path"`
	OldString string `json:"old_string"`
	NewString string `json:"new_string"`
}

// Execute 执行文件编辑
func (t *FileEditTool) Execute(input json.RawMessage) (string, error) {
	var inp FileEditInput
	if err := json.Unmarshal(input, &inp); err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	content, err := os.ReadFile(inp.Path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	oldContent := string(content)
	if !strings.Contains(oldContent, inp.OldString) {
		return "", fmt.Errorf("old_string not found in file")
	}

	newContent := strings.Replace(oldContent, inp.OldString, inp.NewString, 1)

	if err := os.WriteFile(inp.Path, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return "Successfully edited file", nil
}

// GetDescription 获取描述
func (t *FileEditTool) GetDescription() string {
	return "Edit a file by replacing old string with new string"
}

// GetInputSchema 获取输入 schema
func (t *FileEditTool) GetInputSchema() string {
	return `{"type":"object","properties":{"path":{"type":"string"},"old_string":{"type":"string"},"new_string":{"type":"string"}},"required":["path","old_string","new_string"]}`
}

// GlobTool 文件搜索工具
type GlobTool struct{}

// GlobInput 搜索输入
type GlobInput struct {
	Pattern string `json:"pattern"`
}

// Execute 执行搜索
func (t *GlobTool) Execute(input json.RawMessage) (string, error) {
	var inp GlobInput
	if err := json.Unmarshal(input, &inp); err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	matches, err := filepath.Glob(inp.Pattern)
	if err != nil {
		return "", fmt.Errorf("glob error: %w", err)
	}

	if len(matches) == 0 {
		return "No files found", nil
	}

	return strings.Join(matches, "\n"), nil
}

// GetDescription 获取描述
func (t *GlobTool) GetDescription() string {
	return "Search for files matching a glob pattern"
}

// GetInputSchema 获取输入 schema
func (t *GlobTool) GetInputSchema() string {
	return `{"type":"object","properties":{"pattern":{"type":"string"}},"required":["pattern"]}`
}

// GrepTool 文本搜索工具
type GrepTool struct{}

// GrepInput 搜索输入
type GrepInput struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path,omitempty"`
}

// Execute 执行搜索
func (t *GrepTool) Execute(input json.RawMessage) (string, error) {
	var inp GrepInput
	if err := json.Unmarshal(input, &inp); err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	// 简单实现
	return "Grep: pattern=" + inp.Pattern + " in " + inp.Path, nil
}

// GetDescription 获取描述
func (t *GrepTool) GetDescription() string {
	return "Search for a pattern in files"
}

// GetInputSchema 获取输入 schema
func (t *GrepTool) GetInputSchema() string {
	return `{"type":"object","properties":{"pattern":{"type":"string"},"path":{"type":"string"}},"required":["pattern"]}`
}

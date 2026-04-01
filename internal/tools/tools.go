package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gclawcoder/gclaw/internal/models"
)

var (
	portedTools []models.PortingModule
	toolsOnce   sync.Once
	toolsMu     sync.RWMutex
)

// ToolExecution 工具执行结果
type ToolExecution struct {
	Name       string
	SourceHint string
	Payload    string
	Handled    bool
	Message    string
}

// loadToolSnapshot 从 JSON 文件加载工具快照
func loadToolSnapshot() ([]models.PortingModule, error) {
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}
	execDir := filepath.Dir(execPath)

	possiblePaths := []string{
		filepath.Join(execDir, "..", "..", "data", "tools_snapshot.json"),
		filepath.Join(execDir, "..", "data", "tools_snapshot.json"),
		filepath.Join(execDir, "data", "tools_snapshot.json"),
		"data/tools_snapshot.json",
	}

	var rawData []byte
	for _, path := range possiblePaths {
		rawData, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to read tools snapshot: %w", err)
	}

	var entries []models.PortingModule
	if err := json.Unmarshal(rawData, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse tools JSON: %w", err)
	}

	for i := range entries {
		if entries[i].Status == "" {
			entries[i].Status = "mirrored"
		}
	}

	return entries, nil
}

// PortedTools 返回所有已镜像的工具
func PortedTools() []models.PortingModule {
	toolsMu.RLock()
	defer toolsMu.RUnlock()

	toolsOnce.Do(func() {
		tls, err := loadToolSnapshot()
		if err != nil {
			portedTools = []models.PortingModule{}
		} else {
			portedTools = tls
		}
	})

	result := make([]models.PortingModule, len(portedTools))
	copy(result, portedTools)
	return result
}

// BuildToolBacklog 构建工具待办列表
func BuildToolBacklog() models.PortingBacklog {
	return models.PortingBacklog{
		Title:   "Tool surface",
		Modules: PortedTools(),
	}
}

// ToolNames 返回所有工具名称
func ToolNames() []string {
	tls := PortedTools()
	names := make([]string, len(tls))
	for i, tool := range tls {
		names[i] = tool.Name
	}
	return names
}

// GetTool 根据名称获取工具
func GetTool(name string) *models.PortingModule {
	needle := strings.ToLower(name)
	for _, tool := range PortedTools() {
		if strings.ToLower(tool.Name) == needle {
			return &tool
		}
	}
	return nil
}

// FilterToolsByPermissionContext 根据权限上下文过滤工具
func FilterToolsByPermissionContext(tools []models.PortingModule, blockedTools, blockedPrefixes []string) []models.PortingModule {
	if len(blockedTools) == 0 && len(blockedPrefixes) == 0 {
		return tools
	}

	filtered := make([]models.PortingModule, 0, len(tools))
	for _, tool := range tools {
		blocked := false

		for _, bt := range blockedTools {
			if strings.EqualFold(tool.Name, bt) {
				blocked = true
				break
			}
		}

		if !blocked {
			for _, prefix := range blockedPrefixes {
				if strings.HasPrefix(strings.ToLower(tool.Name), strings.ToLower(prefix)) {
					blocked = true
					break
				}
			}
		}

		if !blocked {
			filtered = append(filtered, tool)
		}
	}

	return filtered
}

// GetTools 获取工具列表（支持过滤）
func GetTools(simpleMode, includeMCP bool, blockedTools, blockedPrefixes []string) []models.PortingModule {
	tls := PortedTools()

	if simpleMode {
		filtered := make([]models.PortingModule, 0, 3)
		for _, tool := range tls {
			name := tool.Name
			if name == "BashTool" || name == "FileReadTool" || name == "FileEditTool" {
				filtered = append(filtered, tool)
			}
		}
		tls = filtered
	}

	if !includeMCP {
		filtered := make([]models.PortingModule, 0, len(tls))
		for _, tool := range tls {
			lowerName := strings.ToLower(tool.Name)
			lowerHint := strings.ToLower(tool.SourceHint)
			if !strings.Contains(lowerName, "mcp") && !strings.Contains(lowerHint, "mcp") {
				filtered = append(filtered, tool)
			}
		}
		tls = filtered
	}

	return FilterToolsByPermissionContext(tls, blockedTools, blockedPrefixes)
}

// FindTools 搜索工具
func FindTools(query string, limit int) []models.PortingModule {
	needle := strings.ToLower(query)
	var matches []models.PortingModule

	for _, tool := range PortedTools() {
		if len(matches) >= limit {
			break
		}
		if strings.Contains(strings.ToLower(tool.Name), needle) ||
			strings.Contains(strings.ToLower(tool.SourceHint), needle) {
			matches = append(matches, tool)
		}
	}

	return matches
}

// ExecuteTool 执行工具（模拟）
func ExecuteTool(name, payload string) ToolExecution {
	module := GetTool(name)
	if module == nil {
		return ToolExecution{
			Name:    name,
			Handled: false,
			Message: fmt.Sprintf("Unknown mirrored tool: %s", name),
		}
	}

	return ToolExecution{
		Name:       module.Name,
		SourceHint: module.SourceHint,
		Payload:    payload,
		Handled:    true,
		Message:    fmt.Sprintf("Mirrored tool '%s' from %s would handle payload '%s'.", module.Name, module.SourceHint, payload),
	}
}

// RenderToolIndex 渲染工具索引
func RenderToolIndex(limit int, query string) string {
	var modules []models.PortingModule
	if query != "" {
		modules = FindTools(query, limit)
	} else {
		allTools := PortedTools()
		if len(allTools) > limit {
			modules = allTools[:limit]
		} else {
			modules = allTools
		}
	}

	lines := []string{fmt.Sprintf("Tool entries: %d", len(PortedTools())), ""}

	if query != "" {
		lines = append(lines, "Filtered by: "+query, "")
	}

	for _, module := range modules {
		lines = append(lines, fmt.Sprintf("- %s — %s", module.Name, module.SourceHint))
	}

	return strings.Join(lines, "\n")
}

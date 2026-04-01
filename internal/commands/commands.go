package commands

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
	portedCommands []models.PortingModule
	commandsOnce   sync.Once
	commandsMu     sync.RWMutex
)

// CommandExecution 命令执行结果
type CommandExecution struct {
	Name       string
	SourceHint string
	Prompt     string
	Handled    bool
	Message    string
}

// loadCommandSnapshot 从 JSON 文件加载命令快照
func loadCommandSnapshot() ([]models.PortingModule, error) {
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}
	execDir := filepath.Dir(execPath)

	// 尝试多个可能的路径
	possiblePaths := []string{
		filepath.Join(execDir, "..", "..", "data", "commands_snapshot.json"),
		filepath.Join(execDir, "..", "data", "commands_snapshot.json"),
		filepath.Join(execDir, "data", "commands_snapshot.json"),
		"data/commands_snapshot.json",
	}

	var rawData []byte
	for _, path := range possiblePaths {
		rawData, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to read commands snapshot: %w", err)
	}

	var entries []models.PortingModule
	if err := json.Unmarshal(rawData, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse commands JSON: %w", err)
	}

	// 设置默认状态
	for i := range entries {
		if entries[i].Status == "" {
			entries[i].Status = "mirrored"
		}
	}

	return entries, nil
}

// PortedCommands 返回所有已镜像的命令
func PortedCommands() []models.PortingModule {
	commandsMu.RLock()
	defer commandsMu.RUnlock()

	commandsOnce.Do(func() {
		cmds, err := loadCommandSnapshot()
		if err != nil {
			// 如果加载失败，使用空列表
			portedCommands = []models.PortingModule{}
		} else {
			portedCommands = cmds
		}
	})

	// 返回副本以避免外部修改
	result := make([]models.PortingModule, len(portedCommands))
	copy(result, portedCommands)
	return result
}

// BuildCommandBacklog 构建命令待办列表
func BuildCommandBacklog() models.PortingBacklog {
	return models.PortingBacklog{
		Title:   "Command surface",
		Modules: PortedCommands(),
	}
}

// CommandNames 返回所有命令名称
func CommandNames() []string {
	cmds := PortedCommands()
	names := make([]string, len(cmds))
	for i, cmd := range cmds {
		names[i] = cmd.Name
	}
	return names
}

// GetCommand 根据名称获取命令
func GetCommand(name string) *models.PortingModule {
	needle := strings.ToLower(name)
	for _, cmd := range PortedCommands() {
		if strings.ToLower(cmd.Name) == needle {
			return &cmd
		}
	}
	return nil
}

// GetCommands 获取命令列表（支持过滤）
func GetCommands(includePluginCommands, includeSkillCommands bool) []models.PortingModule {
	cmds := PortedCommands()

	if !includePluginCommands {
		filtered := make([]models.PortingModule, 0, len(cmds))
		for _, cmd := range cmds {
			if !strings.Contains(strings.ToLower(cmd.SourceHint), "plugin") {
				filtered = append(filtered, cmd)
			}
		}
		cmds = filtered
	}

	if !includeSkillCommands {
		filtered := make([]models.PortingModule, 0, len(cmds))
		for _, cmd := range cmds {
			if !strings.Contains(strings.ToLower(cmd.SourceHint), "skill") {
				filtered = append(filtered, cmd)
			}
		}
		cmds = filtered
	}

	return cmds
}

// FindCommands 搜索命令
func FindCommands(query string, limit int) []models.PortingModule {
	needle := strings.ToLower(query)
	var matches []models.PortingModule

	for _, cmd := range PortedCommands() {
		if len(matches) >= limit {
			break
		}
		if strings.Contains(strings.ToLower(cmd.Name), needle) ||
			strings.Contains(strings.ToLower(cmd.SourceHint), needle) {
			matches = append(matches, cmd)
		}
	}

	return matches
}

// ExecuteCommand 执行命令（模拟）
func ExecuteCommand(name, prompt string) CommandExecution {
	module := GetCommand(name)
	if module == nil {
		return CommandExecution{
			Name:    name,
			Handled: false,
			Message: fmt.Sprintf("Unknown mirrored command: %s", name),
		}
	}

	return CommandExecution{
		Name:       module.Name,
		SourceHint: module.SourceHint,
		Prompt:     prompt,
		Handled:    true,
		Message:    fmt.Sprintf("Mirrored command '%s' from %s would handle prompt '%s'.", module.Name, module.SourceHint, prompt),
	}
}

// RenderCommandIndex 渲染命令索引
func RenderCommandIndex(limit int, query string) string {
	var modules []models.PortingModule
	if query != "" {
		modules = FindCommands(query, limit)
	} else {
		allCmds := PortedCommands()
		if len(allCmds) > limit {
			modules = allCmds[:limit]
		} else {
			modules = allCmds
		}
	}

	lines := []string{fmt.Sprintf("Command entries: %d", len(PortedCommands())), ""}

	if query != "" {
		lines = append(lines, "Filtered by: "+query, "")
	}

	for _, module := range modules {
		lines = append(lines, fmt.Sprintf("- %s — %s", module.Name, module.SourceHint))
	}

	return strings.Join(lines, "\n")
}

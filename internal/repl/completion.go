package repl

import (
	"strings"

	"github.com/gclawcoder/gclaw/internal/commands"
	"github.com/gclawcoder/gclaw/internal/toolkit"
)

// Completer 自动补全器
type Completer struct {
	commands      []string
	tools         []string
	files         []string
	slashCommands []string
}

// NewCompleter 创建补全器
func NewCompleter() *Completer {
	return &Completer{
		slashCommands: []string{
			"help", "h", "?",
			"quit", "exit", "q",
			"clear", "cls",
			"model",
			"status",
			"tools",
			"permissions",
			"config",
			"memory",
			"diff",
			"history",
			"verbose", "v",
		},
	}
}

// Do 执行补全
func (c *Completer) Do(line string, pos int) (newLine string, newPos int) {
	// 找到当前单词的起始位置
	start := pos
	for start > 0 && line[start-1] != ' ' && line[start-1] != '/' {
		start--
	}

	prefix := line[start:pos]

	// 斜杠命令补全
	if strings.HasPrefix(line, "/") && start <= 1 {
		if completion := c.completeSlashCommand(prefix); completion != "" {
			return line[:start] + completion, start + len(completion)
		}
	}

	// 工具名补全
	if completion := c.completeTool(prefix); completion != "" {
		return line[:start] + completion, start + len(completion)
	}

	// 命令补全
	if completion := c.completeCommand(prefix); completion != "" {
		return line[:start] + completion, start + len(completion)
	}

	return line, pos
}

// completeSlashCommand 补全斜杠命令
func (c *Completer) completeSlashCommand(prefix string) string {
	prefix = strings.TrimPrefix(prefix, "/")

	var matches []string
	for _, cmd := range c.slashCommands {
		if strings.HasPrefix(cmd, prefix) {
			matches = append(matches, cmd)
		}
	}

	if len(matches) == 1 {
		return "/" + matches[0]
	}

	return ""
}

// completeTool 补全工具名
func (c *Completer) completeTool(prefix string) string {
	tools := toolkit.NewRegistry().List()

	var matches []string
	for _, tool := range tools {
		if strings.HasPrefix(strings.ToLower(tool), strings.ToLower(prefix)) {
			matches = append(matches, tool)
		}
	}

	if len(matches) == 1 {
		return matches[0]
	}

	return ""
}

// completeCommand 补全命令
func (c *Completer) completeCommand(prefix string) string {
	cmds := commands.PortedCommands()

	var matches []string
	for _, cmd := range cmds {
		if strings.HasPrefix(strings.ToLower(cmd.Name), strings.ToLower(prefix)) {
			matches = append(matches, cmd.Name)
		}
	}

	if len(matches) == 1 {
		return matches[0]
	}

	return ""
}

// UpdateCommands 更新命令列表
func (c *Completer) UpdateCommands() {
	cmds := commands.PortedCommands()
	c.commands = make([]string, len(cmds))
	for i, cmd := range cmds {
		c.commands[i] = cmd.Name
	}
}

// UpdateTools 更新工具列表
func (c *Completer) UpdateTools() {
	c.tools = toolkit.NewRegistry().List()
}

// GetSuggestions 获取建议列表
func (c *Completer) GetSuggestions(prefix string) []string {
	var suggestions []string

	// 斜杠命令
	if strings.HasPrefix(prefix, "/") {
		searchPrefix := strings.TrimPrefix(prefix, "/")
		for _, cmd := range c.slashCommands {
			if strings.HasPrefix(cmd, searchPrefix) {
				suggestions = append(suggestions, "/"+cmd)
			}
		}
		return suggestions
	}

	// 工具
	for _, tool := range c.tools {
		if strings.HasPrefix(strings.ToLower(tool), strings.ToLower(prefix)) {
			suggestions = append(suggestions, tool)
		}
	}

	// 命令
	for _, cmd := range c.commands {
		if strings.HasPrefix(strings.ToLower(cmd), strings.ToLower(prefix)) {
			suggestions = append(suggestions, cmd)
		}
	}

	return suggestions
}

// FormatSuggestions 格式化建议列表
func FormatSuggestions(suggestions []string, maxDisplay int) string {
	if len(suggestions) == 0 {
		return ""
	}

	if len(suggestions) > maxDisplay {
		suggestions = suggestions[:maxDisplay]
	}

	return strings.Join(suggestions, "  ")
}

package render

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// ANSI 颜色代码
const (
	Reset     = "\033[0m"
	Bold      = "\033[1m"
	Dim       = "\033[2m"
	Italic    = "\033[3m"
	Underline = "\033[4m"
	Blink     = "\033[5m"
	Reverse   = "\033[7m"

	Black   = "\033[30m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"

	BgBlack   = "\033[40m"
	BgRed     = "\033[41m"
	BgGreen   = "\033[42m"
	BgYellow  = "\033[43m"
	BgBlue    = "\033[44m"
	BgMagenta = "\033[45m"
	BgCyan    = "\033[46m"
	BgWhite   = "\033[47m"
)

// MarkdownRenderer Markdown 渲染器
type MarkdownRenderer struct {
	width        int
	preserveWS   bool
	currentStyle string
}

// NewMarkdownRenderer 创建渲染器
func NewMarkdownRenderer(width int) *MarkdownRenderer {
	if width <= 0 {
		width = 80
	}
	return &MarkdownRenderer{
		width: width,
	}
}

// Render 渲染 Markdown 为 ANSI 文本
func (r *MarkdownRenderer) Render(markdown string) string {
	var result strings.Builder
	lines := strings.Split(markdown, "\n")

	for i, line := range lines {
		rendered := r.renderLine(line)
		result.WriteString(rendered)
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

func (r *MarkdownRenderer) renderLine(line string) string {
	// 处理标题
	if strings.HasPrefix(line, "# ") {
		return r.styleText(line[2:], Bold+Green)
	}
	if strings.HasPrefix(line, "## ") {
		return r.styleText(line[3:], Bold+Cyan)
	}
	if strings.HasPrefix(line, "### ") {
		return r.styleText(line[4:], Bold+Blue)
	}

	// 处理列表
	if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
		return "  " + Magenta + "•" + Reset + " " + r.renderInline(line[2:])
	}

	// 处理引用
	if strings.HasPrefix(line, "> ") {
		return Yellow + "│ " + Reset + r.renderInline(line[2:])
	}

	// 处理代码块
	if strings.HasPrefix(line, "```") {
		return Dim + "────────────────────────────────────" + Reset
	}

	// 处理分隔线
	if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "***") {
		return Dim + strings.Repeat("─", r.width) + Reset
	}

	// 处理代码
	if strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t") {
		return Dim + r.styleText(strings.TrimLeft(line, " \t"), "") + Reset
	}

	// 普通行
	return r.renderInline(line)
}

func (r *MarkdownRenderer) renderInline(text string) string {
	result := text

	// 处理粗体 **text**
	boldRe := regexp.MustCompile(`\*\*(.+?)\*\*`)
	result = boldRe.ReplaceAllStringFunc(result, func(match string) string {
		inner := match[2 : len(match)-2]
		return Bold + inner + Reset
	})

	// 处理斜体 *text*
	italicRe := regexp.MustCompile(`\*(.+?)\*`)
	result = italicRe.ReplaceAllStringFunc(result, func(match string) string {
		inner := match[1 : len(match)-1]
		return Italic + inner + Reset
	})

	// 处理行内代码 `code`
	codeRe := regexp.MustCompile("`(.*?)`")
	result = codeRe.ReplaceAllStringFunc(result, func(match string) string {
		inner := match[1 : len(match)-1]
		return Dim + BgBlack + White + " " + inner + " " + Reset
	})

	// 处理链接 [text](url)
	linkRe := regexp.MustCompile(`\[(.+?)\]\((.+?)\)`)
	result = linkRe.ReplaceAllStringFunc(result, func(match string) string {
		parts := linkRe.FindStringSubmatch(match)
		if len(parts) == 3 {
			return Cyan + parts[1] + Reset + Dim + " (" + parts[2] + ")" + Reset
		}
		return match
	})

	return result
}

func (r *MarkdownRenderer) styleText(text, style string) string {
	if style != "" {
		return style + text + Reset
	}
	return text
}

// WrapText 文本换行
func (r *MarkdownRenderer) WrapText(text string, width int) string {
	if width <= 0 {
		width = r.width
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}

	var lines []string
	var currentLine strings.Builder
	currentWidth := 0

	for _, word := range words {
		wordWidth := utf8.RuneCountInString(word)

		if currentWidth == 0 {
			currentLine.WriteString(word)
			currentWidth = wordWidth
		} else if currentWidth+1+wordWidth <= width {
			currentLine.WriteString(" " + word)
			currentWidth += 1 + wordWidth
		} else {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			currentLine.WriteString(word)
			currentWidth = wordWidth
		}
	}

	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	return strings.Join(lines, "\n")
}

// FormatTable 格式化表格
func (r *MarkdownRenderer) FormatTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}

	// 计算列宽
	colWidths := make([]int, len(headers))
	for i, h := range headers {
		colWidths[i] = utf8.RuneCountInString(h)
	}

	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) {
				width := utf8.RuneCountInString(cell)
				if width > colWidths[i] {
					colWidths[i] = width
				}
			}
		}
	}

	var result strings.Builder

	// 表头
	result.WriteString("│ ")
	for i, h := range headers {
		result.WriteString(Bold + Cyan)
		result.WriteString(padRight(h, colWidths[i]))
		result.WriteString(Reset)
		if i < len(headers)-1 {
			result.WriteString(" │ ")
		}
	}
	result.WriteString(" │\n")

	// 分隔线
	result.WriteString("├")
	for i, w := range colWidths {
		result.WriteString(strings.Repeat("─", w+2))
		if i < len(colWidths)-1 {
			result.WriteString("┼")
		}
	}
	result.WriteString("┤\n")

	// 数据行
	for _, row := range rows {
		result.WriteString("│ ")
		for i, cell := range row {
			result.WriteString(padRight(cell, colWidths[i]))
			if i < len(row)-1 {
				result.WriteString(" │ ")
			}
		}
		result.WriteString(" │\n")
	}

	return result.String()
}

// FormatCodeBlock 格式化代码块
func (r *MarkdownRenderer) FormatCodeBlock(code, language string) string {
	var result strings.Builder

	result.WriteString(Dim + "┌" + strings.Repeat("─", r.width-2) + "┐" + Reset + "\n")

	if language != "" {
		result.WriteString("│ " + Yellow + language + Reset)
		padding := r.width - 4 - len(language)
		if padding > 0 {
			result.WriteString(strings.Repeat(" ", padding))
		}
		result.WriteString(Dim + "│" + Reset + "\n")
		result.WriteString("├" + strings.Repeat("─", r.width-2) + "┤\n")
	}

	lines := strings.Split(code, "\n")
	for _, line := range lines {
		truncated := line
		if len(truncated) > r.width-4 {
			truncated = truncated[:r.width-4] + "..."
		}
		result.WriteString(Dim + "│ " + Reset + truncated)
		padding := r.width - 4 - utf8.RuneCountInString(truncated)
		if padding > 0 {
			result.WriteString(strings.Repeat(" ", padding))
		}
		result.WriteString(Dim + " │" + Reset + "\n")
	}

	result.WriteString(Dim + "└" + strings.Repeat("─", r.width-2) + "┘" + Reset)

	return result.String()
}

// FormatStatus 格式化状态消息
func (r *MarkdownRenderer) FormatStatus(status, message string) string {
	icon := "✓"
	color := Green

	switch status {
	case "error", "fail", "failed":
		icon = "✗"
		color = Red
	case "warning", "warn":
		icon = "⚠"
		color = Yellow
	case "info":
		icon = "ℹ"
		color = Blue
	}

	return fmt.Sprintf("%s%s %s%s", color, icon, message, Reset)
}

// FormatProgress 格式化进度条
func (r *MarkdownRenderer) FormatProgress(current, total int, width int) string {
	if width <= 0 {
		width = 40
	}

	percent := float64(current) / float64(total)
	filled := int(percent * float64(width))
	empty := width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	return fmt.Sprintf("[%s] %d/%d (%.1f%%)", bar, current, total, percent*100)
}

func padRight(s string, width int) string {
	runeCount := utf8.RuneCountInString(s)
	if runeCount >= width {
		return s
	}
	return s + strings.Repeat(" ", width-runeCount)
}

// RenderSimple 简单渲染函数
func RenderSimple(markdown string) string {
	renderer := NewMarkdownRenderer(80)
	return renderer.Render(markdown)
}

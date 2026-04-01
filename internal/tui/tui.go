package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/gclawcoder/gclaw/internal/api"
	"github.com/gclawcoder/gclaw/internal/config"
	"github.com/gclawcoder/gclaw/internal/conversation"
	"github.com/gclawcoder/gclaw/internal/toolkit"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// App TUI 应用
type App struct {
	app          *tview.Application
	grid         *tview.Grid
	header       *tview.TextView
	messages     *tview.TextView
	input        *tview.TextArea
	status       *tview.TextView
	session      *conversation.Session
	runtime      *conversation.ConversationRuntime
	apiKeyClient *api.APIKeyClient
	cfg          *config.RuntimeConfig
	busy         bool
	history      []string
	historyIdx   int
}

// NewApp 创建应用
func NewApp() (*App, error) {
	if _, err := os.Stat("/dev/tty"); err != nil {
		return nil, fmt.Errorf("TUI requires a real terminal")
	}

	apiKeyClient := api.NewAPIKeyClient()

	var cfg *config.RuntimeConfig
	if apiKeyClient.IsConfigured() {
		info, err := apiKeyClient.GetConfig()
		if err != nil {
			return nil, err
		}
		cfg = &config.RuntimeConfig{
			Model:          info.Model,
			APIKey:         info.APIKey,
			BaseURL:        info.BaseURL,
			AuthType:       info.AuthType,
			AuthHeader:     info.AuthHeader,
			Version:        info.Version,
			PermissionMode: "danger-full-access",
			MaxTokens:      4096,
			MaxIterations:  100,
		}
	} else {
		cfg = &config.RuntimeConfig{
			Model:          "claude-sonnet-4-20250514",
			PermissionMode: "danger-full-access",
			MaxTokens:      4096,
			MaxIterations:  100,
		}
	}

	client := createClient(cfg)
	registry := toolkit.NewRegistry()
	session := conversation.NewSession()
	runtime := conversation.NewConversationRuntime(session, client, registry, nil, "")

	a := &App{
		app:          tview.NewApplication(),
		apiKeyClient: apiKeyClient,
		cfg:          cfg,
		session:      session,
		runtime:      runtime,
		history:      make([]string, 0),
		historyIdx:   -1,
	}

	a.createUI()
	a.setupHandlers()
	return a, nil
}

func createClient(cfg *config.RuntimeConfig) *api.Client {
	authType := cfg.AuthType
	if authType == "" {
		authType = "header"
	}
	authHeader := cfg.AuthHeader
	if authHeader == "" {
		authHeader = "x-api-key"
	}

	if cfg.APIKey == "" {
		client := api.NewAPIKeyClient()
		if info, err := client.GetConfig(); err == nil {
			cfg.APIKey = info.APIKey
			if cfg.BaseURL == "" {
				cfg.BaseURL = info.BaseURL
			}
		}
	}

	return api.NewClientWithFullConfig(cfg.APIKey, cfg.Model, cfg.BaseURL, authType, authHeader, cfg.Version)
}

func (a *App) createUI() {
	// 使用更现代的颜色方案
	bgColor := tcell.ColorDefault
	borderColor := tcell.ColorDarkGray
	textColor := tcell.ColorWhite
	accentColor := tcell.ColorLightBlue

	// Header - 简洁的标题栏
	a.header = tview.NewTextView().
		SetDynamicColors(true).
		SetText(fmt.Sprintf(" [bold][white]gClawCoder[reset] [gray]·[reset] [cyan]%s[reset] [gray]·[reset] [darkgray]Ctrl+G: Send  Ctrl+Q: Quit[reset]", a.cfg.Model))
	a.header.SetBackgroundColor(tcell.ColorDarkBlue)
	a.header.SetTextColor(textColor)

	// Messages - 对话区域
	a.messages = tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true).
		SetScrollable(true)
	a.messages.SetBorder(true)
	a.messages.SetBorderColor(borderColor)
	a.messages.SetBackgroundColor(bgColor)
	a.messages.SetTitle(" [bold]Messages[reset] ")
	a.messages.SetTitleColor(accentColor)

	// Input - 输入区域
	a.input = tview.NewTextArea().
		SetWrap(true).
		SetPlaceholder(" [darkgray]Type your message...[reset]")
	a.input.SetBorder(true)
	a.input.SetBorderColor(accentColor)
	a.input.SetBackgroundColor(bgColor)
	a.input.SetTitle(" [bold]Input[reset] ")
	a.input.SetTitleColor(accentColor)

	// Status - 状态栏
	a.status = tview.NewTextView().
		SetDynamicColors(true).
		SetText(" [green]●[reset] [gray]Ready[reset]")
	a.status.SetBackgroundColor(tcell.ColorDarkGray)
	a.status.SetTextColor(textColor)

	// Grid 布局
	a.grid = tview.NewGrid().
		SetColumns(0).
		SetRows(1, 0, 4, 1).
		SetBorders(false)

	a.grid.AddItem(a.header, 0, 0, 1, 1, 0, 0, false)
	a.grid.AddItem(a.messages, 1, 0, 1, 1, 0, 0, false)
	a.grid.AddItem(a.input, 2, 0, 1, 1, 0, 0, true)
	a.grid.AddItem(a.status, 3, 0, 1, 1, 0, 0, false)

	a.app.SetRoot(a.grid, true).SetFocus(a.input)
}

func (a *App) setupHandlers() {
	a.input.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyEnter && ev.Modifiers()&tcell.ModCtrl != 0 {
			text := a.input.GetText()
			if text != "" && !a.busy {
				a.send(text)
			}
			return nil
		}
		if ev.Key() == tcell.KeyRune && ev.Rune() == 'q' && ev.Modifiers()&tcell.ModCtrl != 0 {
			a.app.Stop()
			return nil
		}
		if ev.Key() == tcell.KeyUp {
			if a.historyIdx < len(a.history)-1 {
				a.historyIdx++
				a.input.SetText(a.history[len(a.history)-1-a.historyIdx], true)
			}
			return nil
		}
		if ev.Key() == tcell.KeyDown {
			if a.historyIdx > 0 {
				a.historyIdx--
				a.input.SetText(a.history[len(a.history)-1-a.historyIdx], true)
			} else if a.historyIdx == 0 {
				a.historyIdx = -1
				a.input.SetText("", true)
			}
			return nil
		}
		return ev
	})
}

func (a *App) send(text string) {
	a.busy = true
	a.input.SetText("", true)
	a.history = append(a.history, text)
	a.historyIdx = -1

	// 使用更美观的消息格式
	a.messages.SetText(fmt.Sprintf("%s\n[bold][green]➤ You:[reset] %s", a.messages.GetText(true), text))
	a.status.SetText(" [yellow]●[reset] [yellow]Thinking...[reset]")

	go func() {
		result, err := a.runtime.RunTurn(text)
		a.app.QueueUpdateDraw(func() {
			if err != nil {
				a.messages.SetText(fmt.Sprintf("%s\n[bold][red]✗ Error:[reset] %v", a.messages.GetText(true), err))
				a.status.SetText(" [red]●[reset] [red]Error[reset]")
			} else {
				for _, msg := range result.AssistantMessages {
					for _, block := range msg.Content {
						if block.Type == conversation.BlockTypeText {
							// 格式化 AI 回复，支持简单的 markdown
							formatted := a.formatMarkdown(block.Text)
							a.messages.SetText(fmt.Sprintf("%s\n[bold][cyan]➤ Assistant:[reset] %s", a.messages.GetText(true), formatted))
						}
					}
				}
				a.status.SetText(" [green]●[reset] [green]Ready[reset]")
			}
			a.busy = false
			a.app.SetFocus(a.input)
		})
	}()
}

// formatMarkdown 简单的 markdown 格式化
func (a *App) formatMarkdown(text string) string {
	// 代码块
	text = strings.ReplaceAll(text, "```", "[yellow]```[reset]")
	// 行内代码
	text = strings.ReplaceAll(text, "`", "[yellow]`[reset]")
	// 粗体
	text = strings.ReplaceAll(text, "**", "[bold]")
	text = strings.ReplaceAll(text, "__", "[bold]")
	return text
}

// Run 运行应用
func (a *App) Run() error {
	return a.app.Run()
}

package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gclawcoder/gclaw/internal/api"
	"github.com/gclawcoder/gclaw/internal/config"
	"github.com/gclawcoder/gclaw/internal/conversation"
	"github.com/gclawcoder/gclaw/internal/toolkit"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Application TUI 应用
type Application struct {
	app          *tview.Application
	grid         *tview.Grid
	header       *tview.TextView
	messages     *tview.TextView
	input        *tview.TextArea
	footer       *tview.TextView
	session      *conversation.Session
	runtime      *conversation.ConversationRuntime
	apiKeyClient *api.APIKeyClient
	cfg          *config.RuntimeConfig
	busy         bool
	history      []string
	historyIdx   int
	width        int
	height       int
}

// NewApplication 创建应用
func NewApplication() (*Application, error) {
	// 检查 tty
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

	a := &Application{
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

func (a *Application) createUI() {
	// Header
	a.header = tview.NewTextView().
		SetDynamicColors(true).
		SetText(fmt.Sprintf(" [yellow]gClawCoder[] [white]| Model: [cyan]%s[] [white]| [gray]Ctrl+G: Send, Ctrl+Q: Quit, Ctrl+H: Help[]", a.cfg.Model))
	a.header.SetBackgroundColor(tcell.ColorDarkBlue)

	// Messages
	a.messages = tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true).
		SetScrollable(true)
	a.messages.SetBorder(true).
		SetBorderColor(tcell.ColorGray).
		SetTitle(" Chat ")

	// Input
	a.input = tview.NewTextArea().
		SetWrap(true).
		SetPlaceholder("Type your message here...")
	a.input.SetBorder(true).
		SetBorderColor(tcell.ColorGray).
		SetTitle(" Input ")

	// Footer
	a.footer = tview.NewTextView().
		SetDynamicColors(true).
		SetText(" [green]● Ready[]")
	a.footer.SetBackgroundColor(tcell.ColorDarkGray)

	// Grid layout
	a.grid = tview.NewGrid().
		SetColumns(0).
		SetRows(1, 0, 3, 1).
		SetBorders(false).
		AddItem(a.header, 0, 0, 1, 1, 0, 0, false).
		AddItem(a.messages, 1, 0, 1, 1, 0, 0, false).
		AddItem(a.input, 2, 0, 1, 1, 0, 0, true).
		AddItem(a.footer, 3, 0, 1, 1, 0, 0, false)

	a.app.SetRoot(a.grid, true).SetFocus(a.input)
}

func (a *Application) setupHandlers() {
	a.input.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		// Ctrl+G: Send
		if ev.Key() == tcell.KeyEnter && ev.Modifiers()&tcell.ModCtrl != 0 {
			text := a.input.GetText()
			if text != "" && !a.busy {
				a.send(text)
			}
			return nil
		}
		// Ctrl+Q: Quit
		if ev.Key() == tcell.KeyRune && ev.Rune() == 'q' && ev.Modifiers()&tcell.ModCtrl != 0 {
			a.app.Stop()
			return nil
		}
		// Ctrl+H: Help
		if ev.Key() == tcell.KeyRune && ev.Rune() == 'h' && ev.Modifiers()&tcell.ModCtrl != 0 {
			a.showHelp()
			return nil
		}
		// Up: History previous
		if ev.Key() == tcell.KeyUp {
			if a.historyIdx < len(a.history)-1 {
				a.historyIdx++
				a.input.SetText(a.history[len(a.history)-1-a.historyIdx], true)
			}
			return nil
		}
		// Down: History next
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

func (a *Application) send(text string) {
	a.busy = true
	a.input.SetText("", true)
	a.history = append(a.history, text)
	a.historyIdx = -1

	// Add user message
	a.messages.SetText(fmt.Sprintf("%s[bold][green]▌ You:[reset] %s\n\n", a.messages.GetText(true), text))
	a.footer.SetText(" [yellow]● Thinking...[]")

	go func() {
		result, err := a.runtime.RunTurn(text)
		a.app.QueueUpdateDraw(func() {
			if err != nil {
				a.messages.SetText(fmt.Sprintf("%s[bold][red]▌ Error:[reset] %v\n\n", a.messages.GetText(true), err))
				a.footer.SetText(" [red]● Error[]")
			} else {
				for _, msg := range result.AssistantMessages {
					for _, block := range msg.Content {
						if block.Type == conversation.BlockTypeText {
							a.messages.SetText(fmt.Sprintf("%s[bold][cyan]▌ Assistant:[reset] %s\n\n", a.messages.GetText(true), block.Text))
						}
					}
				}
				a.footer.SetText(" [green]● Ready[]")
			}
			a.busy = false
			a.app.SetFocus(a.input)
		})
	}()
}

func (a *Application) showHelp() {
	help := tview.NewTextView().
		SetDynamicColors(true).
		SetText(`
[bold]Keyboard Shortcuts:[reset]

  [yellow]Ctrl+G[reset]  Send message
  [yellow]Ctrl+Q[reset]  Quit application
  [yellow]Ctrl+H[reset]  Show this help
  [yellow]↑/↓[reset]     Navigate history
  [yellow]Ctrl+C[reset]  Copy last response

[bold]Tips:[reset]

  - Use markdown for code blocks
  - Ask follow-up questions
  - Use /clear to reset conversation
`)
	help.SetBorder(true).
		SetTitle(" Help ").
		SetBackgroundColor(tcell.ColorDarkBlue)

	help.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyEnter {
			a.grid.RemoveItem(help)
			a.app.SetFocus(a.input)
			return nil
		}
		return ev
	})

	a.grid.AddItem(help, 1, 0, 1, 1, 0, 0, true)
	a.app.SetFocus(help)
}

// Run 运行应用
func (a *Application) Run() error {
	return a.app.Run()
}

// Helper functions
func getUserHome() string {
	home, _ := os.UserHomeDir()
	return home
}

func formatTime(t time.Time) string {
	return t.Format("15:04")
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func joinStrings(separator string, parts ...string) string {
	return strings.Join(parts, separator)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

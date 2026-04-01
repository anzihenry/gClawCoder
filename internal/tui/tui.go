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

// App TUI 应用
type App struct {
	app          *tview.Application
	layout       *tview.Flex
	input        *tview.TextArea
	output       *tview.TextView
	status       *tview.TextView
	modelInfo    *tview.TextView
	menu         *tview.List
	pages        *tview.Pages
	session      *conversation.Session
	runtime      *conversation.ConversationRuntime
	apiKeyClient *api.APIKeyClient
	cfg          *config.RuntimeConfig
	busy         bool
	history      []string
	historyIdx   int
}

// NewApp 创建 TUI 应用
func NewApp() (*App, error) {
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

	app := &App{
		app:          tview.NewApplication(),
		apiKeyClient: apiKeyClient,
		cfg:          cfg,
		session:      session,
		runtime:      runtime,
		history:      make([]string, 0),
		historyIdx:   -1,
	}

	app.createUI()
	return app, nil
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
	a.output = tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true).
		SetScrollable(true)
	a.output.SetBorder(true).SetTitle(" Chat ")
	a.output.SetBackgroundColor(tcell.ColorDefault)

	a.input = tview.NewTextArea().
		SetWrap(true).
		SetPlaceholder("Type message... (Ctrl+Enter to send, Ctrl+Q to quit)")
	a.input.SetBorder(true).SetTitle(" Input ")
	a.input.SetBackgroundColor(tcell.ColorDefault)

	a.status = tview.NewTextView().
		SetDynamicColors(true).
		SetText("[green]● Ready[]")
	a.status.SetBorder(true)

	a.modelInfo = tview.NewTextView().
		SetDynamicColors(true).
		SetText(fmt.Sprintf("[yellow]Model:[white] %s", a.cfg.Model))
	a.modelInfo.SetBorder(true)

	a.menu = tview.NewList()
	a.menu.SetBorder(true).SetTitle(" Menu ")
	a.menu.SetBackgroundColor(tcell.ColorDefault)
	a.menu.AddItem("New Chat", "Clear conversation", 'n', func() { a.newChat() })
	a.menu.AddItem("Models", "Switch model", 'm', func() { a.showModels() })
	a.menu.AddItem("Export", "Save to file", 'e', func() { a.export() })
	a.menu.AddItem("Quit", "Exit application", 'q', func() { a.app.Stop() })

	inputBox := tview.NewFlex().
		AddItem(a.input, 0, 1, true)

	chatBox := tview.NewFlex().
		AddItem(a.output, 0, 1, false)

	leftPanel := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(chatBox, 0, 3, false).
		AddItem(inputBox, 5, 1, false)

	rightPanel := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.menu, 0, 1, false)

	mainContent := tview.NewFlex().
		AddItem(leftPanel, 0, 4, false).
		AddItem(rightPanel, 0, 1, false)

	statusBar := tview.NewFlex().
		AddItem(a.status, 0, 1, false).
		AddItem(a.modelInfo, 30, 1, false)

	root := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(mainContent, 0, 1, false).
		AddItem(statusBar, 1, 1, false)

	a.layout = root
	a.pages = tview.NewPages().AddPage("main", root, true, true)

	a.app.SetRoot(a.pages, true).SetFocus(a.input)
	a.setupHandlers()
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

	a.menu.SetSelectedFunc(func(idx int, label, _ string, _ rune) {
		switch idx {
		case 0:
			a.newChat()
		case 1:
			a.showModels()
		case 2:
			a.export()
		case 3:
			a.app.Stop()
		}
	})
}

func (a *App) send(text string) {
	a.busy = true
	a.input.SetText("", true)
	a.history = append(a.history, text)
	a.historyIdx = -1

	a.output.SetText(fmt.Sprintf("%s[green]You:[white] %s\n\n", a.output.GetText(true), text))
	a.status.SetText("[yellow]● Thinking...[]")

	go func() {
		result, err := a.runtime.RunTurn(text)
		a.app.QueueUpdateDraw(func() {
			if err != nil {
				a.output.SetText(fmt.Sprintf("%s[red]Error:[white] %v\n\n", a.output.GetText(true), err))
				a.status.SetText("[red]● Error[]")
			} else {
				for _, msg := range result.AssistantMessages {
					for _, block := range msg.Content {
						if block.Type == conversation.BlockTypeText {
							a.output.SetText(fmt.Sprintf("%s[cyan]Assistant:[white] %s\n\n", a.output.GetText(true), block.Text))
						}
					}
				}
				a.status.SetText("[green]● Ready[]")
			}
			a.busy = false
			a.app.SetFocus(a.input)
		})
	}()
}

func (a *App) newChat() {
	a.session = conversation.NewSession()
	a.runtime = conversation.NewConversationRuntime(
		a.session,
		createClient(a.cfg),
		toolkit.NewRegistry(),
		nil,
		"",
	)
	a.output.SetText("")
	a.status.SetText("[green]● Ready[]")
}

func (a *App) showModels() {
	if !a.apiKeyClient.IsConfigured() {
		a.status.SetText("[yellow]● No API Key[]")
		return
	}

	models, err := a.apiKeyClient.GetAvailableModels()
	if err != nil {
		a.status.SetText(fmt.Sprintf("[red]● Error: %v[]", err))
		return
	}

	current, _ := a.apiKeyClient.GetCurrentModel()

	modalList := tview.NewList()
	modalList.SetBorder(true).SetTitle(" Select Model (Esc to close) ")
	modalList.SetBackgroundColor(tcell.ColorDarkCyan)

	for _, m := range models {
		mark := ""
		if m == current {
			mark = " ✓"
		}
		modalList.AddItem(m+mark, "", 0, nil)
	}

	modalList.SetSelectedFunc(func(idx int, label, _ string, _ rune) {
		name := strings.TrimSpace(strings.TrimSuffix(label, " ✓"))
		if err := a.apiKeyClient.SetModel(name); err == nil {
			a.cfg.Model = name
			a.modelInfo.SetText(fmt.Sprintf("[yellow]Model:[white] %s", name))
			a.status.SetText(fmt.Sprintf("[green]● Switched to %s[]", name))
		}
		a.pages.RemovePage("modal")
		a.app.SetFocus(a.input)
	})

	modalList.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyEscape {
			a.pages.RemovePage("modal")
			a.app.SetFocus(a.input)
			return nil
		}
		return ev
	})

	a.pages.AddPage("modal", modalList, true, true)
	a.app.SetFocus(modalList)
}

func (a *App) export() {
	content := a.output.GetText(true)
	if content == "" {
		a.status.SetText("[yellow]● Nothing to export[]")
		return
	}

	filename := fmt.Sprintf("chat-%d.txt", time.Now().Unix())
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		a.status.SetText(fmt.Sprintf("[red]● Export failed: %v[]", err))
	} else {
		a.status.SetText(fmt.Sprintf("[green]● Exported to %s[]", filename))
	}
}

// Run 运行 TUI
func (a *App) Run() error {
	return a.app.Run()
}

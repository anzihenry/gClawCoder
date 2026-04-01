package tui

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gclawcoder/gclaw/internal/api"
	"github.com/gclawcoder/gclaw/internal/config"
	"github.com/gclawcoder/gclaw/internal/conversation"
	"github.com/gclawcoder/gclaw/internal/toolkit"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TUI TUI 应用
type TUI struct {
	app          *tview.Application
	pages        *tview.Pages
	input        *tview.TextArea
	messages     *tview.TextView
	status       *tview.TextView
	model        *tview.TextView
	sidebar      *tview.List
	session      *conversation.Session
	runtime      *conversation.ConversationRuntime
	toolRegistry *toolkit.Registry
	apiKeyClient *api.APIKeyClient
	config       *config.RuntimeConfig
	mu           sync.Mutex
	isProcessing bool
	messagesList []MessageEntry
}

// MessageEntry 消息条目
type MessageEntry struct {
	Role    string
	Content string
}

// NewTUI 创建 TUI
func NewTUI() (*TUI, error) {
	apiKeyClient := api.NewAPIKeyClient()

	var cfg *config.RuntimeConfig
	if apiKeyClient.IsConfigured() {
		apiKeyInfo, _ := apiKeyClient.GetConfig()
		cfg = &config.RuntimeConfig{
			Model:          apiKeyInfo.Model,
			APIKey:         apiKeyInfo.APIKey,
			BaseURL:        apiKeyInfo.BaseURL,
			AuthType:       apiKeyInfo.AuthType,
			AuthHeader:     apiKeyInfo.AuthHeader,
			Version:        apiKeyInfo.Version,
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

	apiClient := createAPIClient(cfg)
	toolRegistry := toolkit.NewRegistry()

	session := conversation.NewSession()
	runtime := conversation.NewConversationRuntime(
		session,
		apiClient,
		toolRegistry,
		nil,
		"",
	)

	t := &TUI{
		app:          tview.NewApplication(),
		session:      session,
		runtime:      runtime,
		toolRegistry: toolRegistry,
		apiKeyClient: apiKeyClient,
		config:       cfg,
		messagesList: make([]MessageEntry, 0),
	}

	t.createUI()
	return t, nil
}

func createAPIClient(cfg *config.RuntimeConfig) *api.Client {
	authType := cfg.AuthType
	authHeader := cfg.AuthHeader
	version := cfg.Version

	if authType == "" {
		authType = "header"
	}
	if authHeader == "" {
		authHeader = "x-api-key"
	}

	if cfg.APIKey == "" {
		apiKeyClient := api.NewAPIKeyClient()
		if apiKeyClient.IsConfigured() {
			apiKeyInfo, _ := apiKeyClient.GetConfig()
			cfg.APIKey = apiKeyInfo.APIKey
			if cfg.BaseURL == "" {
				cfg.BaseURL = apiKeyInfo.BaseURL
			}
			if authType == "header" && apiKeyInfo.AuthType != "" {
				authType = apiKeyInfo.AuthType
			}
			if authHeader == "x-api-key" && apiKeyInfo.AuthHeader != "" {
				authHeader = apiKeyInfo.AuthHeader
			}
		}
	}

	return api.NewClientWithFullConfig(cfg.APIKey, cfg.Model, cfg.BaseURL, authType, authHeader, version)
}

func (t *TUI) createUI() {
	t.messages = tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true).
		SetScrollable(true)
	t.messages.SetTitle(" Messages ").SetBorder(true)

	t.input = tview.NewTextArea().
		SetWrap(true).
		SetPlaceholder("输入消息... (Ctrl+Enter 发送，Ctrl+Q 退出)")
	t.input.SetTitle(" Input ").SetBorder(true)

	t.status = tview.NewTextView().
		SetDynamicColors(true).
		SetText("[green]Ready[]")
	t.status.SetBorder(true).SetTitle(" Status ")

	t.model = tview.NewTextView().
		SetDynamicColors(true).
		SetText(fmt.Sprintf("[yellow]Model:[white] %s", t.config.Model))
	t.model.SetBorder(true).SetTitle(" Model ")

	t.sidebar = tview.NewList()
	t.sidebar.AddItem("New Chat", "Start new conversation", 'n', nil).
		AddItem("Clear", "Clear messages", 'c', nil).
		AddItem("Models", "Change model", 'm', nil).
		AddItem("Export", "Export conversation", 'e', nil).
		AddItem("Quit", "Exit application", 'q', nil)
	t.sidebar.SetTitle(" Menu ").SetBorder(true)

	inputFlex := tview.NewFlex().
		AddItem(t.input, 0, 3, true)

	messagesFlex := tview.NewFlex().
		AddItem(t.messages, 0, 1, false)

	leftFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(messagesFlex, 0, 3, false).
		AddItem(inputFlex, 3, 1, false)

	mainFlex := tview.NewFlex().
		AddItem(leftFlex, 0, 4, false).
		AddItem(t.sidebar, 0, 1, false)

	statusFlex := tview.NewFlex().
		AddItem(t.status, 0, 1, false).
		AddItem(t.model, 30, 1, false)

	rootFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(mainFlex, 0, 1, false).
		AddItem(statusFlex, 1, 1, false)

	t.pages = tview.NewPages().
		AddPage("main", rootFlex, true, true)

	t.app.SetRoot(t.pages, true).
		SetFocus(t.input)

	t.setupInputHandler()
	t.setupSidebarHandler()
}

func (t *TUI) setupInputHandler() {
	t.input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter && event.Modifiers()&tcell.ModCtrl != 0 {
			text := t.input.GetText()
			if strings.TrimSpace(text) != "" && !t.isProcessing {
				t.sendMessage(text)
			}
			return nil
		}
		if event.Key() == tcell.KeyRune && event.Rune() == 'q' && event.Modifiers()&tcell.ModCtrl != 0 {
			t.app.Stop()
			return nil
		}
		return event
	})
}

func (t *TUI) setupSidebarHandler() {
	t.sidebar.SetSelectedFunc(func(index int, text string, secondaryText string, shortcut rune) {
		switch shortcut {
		case 'n':
			t.newChat()
		case 'c':
			t.clearMessages()
		case 'm':
			t.showModelSelector()
		case 'e':
			t.exportConversation()
		case 'q':
			t.app.Stop()
		}
	})
}

func (t *TUI) sendMessage(text string) {
	t.mu.Lock()
	if t.isProcessing {
		t.mu.Unlock()
		return
	}
	t.isProcessing = true
	t.mu.Unlock()

	t.input.SetText("", true)
	t.appendMessage("user", text)
	t.updateStatus("[yellow]Thinking...[]")

	go func() {
		result, err := t.runtime.RunTurn(text)

		t.app.QueueUpdateDraw(func() {
			if err != nil {
				t.appendMessage("error", fmt.Sprintf("Error: %v", err))
				t.updateStatus("[red]Error[]")
			} else {
				for _, msg := range result.AssistantMessages {
					for _, block := range msg.Content {
						if block.Type == conversation.BlockTypeText {
							t.appendMessage("assistant", block.Text)
						}
					}
				}
				t.updateStatus("[green]Ready[]")
			}

			t.mu.Lock()
			t.isProcessing = false
			t.mu.Unlock()
		})
	}()
}

func (t *TUI) appendMessage(role, content string) {
	t.messagesList = append(t.messagesList, MessageEntry{
		Role:    role,
		Content: content,
	})

	color := "white"
	prefix := "You"
	switch role {
	case "user":
		color = "green"
		prefix = "You"
	case "assistant":
		color = "cyan"
		prefix = "Assistant"
	case "error":
		color = "red"
		prefix = "Error"
	}

	fmt.Fprintf(t.messages, "[%s]%s:[white] %s\n\n", color, prefix, content)
	t.messages.ScrollToEnd()
}

func (t *TUI) updateStatus(status string) {
	t.status.SetText(status)
}

func (t *TUI) newChat() {
	t.session = conversation.NewSession()
	t.messagesList = make([]MessageEntry, 0)
	t.messages.Clear()
	t.updateStatus("[green]New chat started[]")
}

func (t *TUI) clearMessages() {
	t.messagesList = make([]MessageEntry, 0)
	t.messages.Clear()
	t.updateStatus("[green]Messages cleared[]")
}

func (t *TUI) showModelSelector() {
	if !t.apiKeyClient.IsConfigured() {
		t.updateStatus("[yellow]No API Key configured[]")
		return
	}

	models, err := t.apiKeyClient.GetAvailableModels()
	if err != nil {
		t.updateStatus(fmt.Sprintf("[red]Error: %v[]", err))
		return
	}

	currentModel, _ := t.apiKeyClient.GetCurrentModel()

	// 简单的列表选择
	list := tview.NewList()
	list.SetTitle(" Select Model ")
	list.SetBorder(true)

	for _, model := range models {
		marker := ""
		if model == currentModel {
			marker = " ✓"
		}
		list.AddItem(model+marker, "", 0, nil)
	}

	list.SetSelectedFunc(func(index int, text string, _ string, _ rune) {
		selectedModel := strings.TrimSpace(strings.TrimSuffix(text, " ✓"))
		if err := t.apiKeyClient.SetModel(selectedModel); err == nil {
			t.config.Model = selectedModel
			t.model.SetText(fmt.Sprintf("[yellow]Model:[white] %s", selectedModel))
			t.updateStatus(fmt.Sprintf("[green]Switched to %s[]", selectedModel))
		}
		t.pages.RemovePage("modellist")
		t.app.SetFocus(t.input)
	})

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || event.Rune() == 'q' {
			t.pages.RemovePage("modellist")
			t.app.SetFocus(t.input)
			return nil
		}
		return event
	})

	t.pages.AddPage("modellist", list, true, true)
	t.app.SetFocus(list)
}

func (t *TUI) exportConversation() {
	if len(t.messagesList) == 0 {
		t.updateStatus("[yellow]No messages to export[]")
		return
	}

	var builder strings.Builder
	for _, msg := range t.messagesList {
		fmt.Fprintf(&builder, "%s: %s\n\n", msg.Role, msg.Content)
	}

	filePath := fmt.Sprintf("conversation-%d.txt", time.Now().Unix())

	err := os.WriteFile(filePath, []byte(builder.String()), 0644)
	if err != nil {
		t.updateStatus(fmt.Sprintf("[red]Export failed: %v[]", err))
	} else {
		t.updateStatus(fmt.Sprintf("[green]Exported to %s[]", filePath))
	}
}

func (t *TUI) Run() error {
	return t.app.Run()
}

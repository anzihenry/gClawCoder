package repl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gclawcoder/gclaw/internal/api"
	"github.com/gclawcoder/gclaw/internal/config"
	"github.com/gclawcoder/gclaw/internal/conversation"
	"github.com/gclawcoder/gclaw/internal/git"
	"github.com/gclawcoder/gclaw/internal/permissions"
	"github.com/gclawcoder/gclaw/internal/toolkit"
)

// REPL 交互式终端
type REPL struct {
	reader           *bufio.Reader
	writer           io.Writer
	runtime          *conversation.ConversationRuntime
	config           *config.RuntimeConfig
	toolRegistry     *toolkit.Registry
	permissionPolicy *permissions.PermissionPolicy
	gitClient        *git.Client
	history          []string
	historyIndex     int
	running          bool
	model            string
	verbose          bool
}

// NewREPL 创建 REPL
func NewREPL(cfg *config.RuntimeConfig) (*REPL, error) {
	apiKey := cfg.APIKey
	baseURL := cfg.BaseURL
	authType := cfg.AuthType
	authHeader := cfg.AuthHeader
	version := cfg.Version

	// 设置默认值
	if authType == "" {
		authType = "header"
	}
	if authHeader == "" {
		authHeader = "x-api-key"
	}

	if apiKey == "" {
		// 尝试从 API Key 配置文件读取
		apiKeyClient := api.NewAPIKeyClient()
		if apiKeyClient.IsConfigured() {
			apiKeyInfo, err := apiKeyClient.GetConfig()
			if err == nil {
				apiKey = apiKeyInfo.APIKey
				if baseURL == "" {
					baseURL = apiKeyInfo.BaseURL
				}
				if authType == "header" && apiKeyInfo.AuthType != "" {
					authType = apiKeyInfo.AuthType
				}
				if authHeader == "x-api-key" && apiKeyInfo.AuthHeader != "" {
					authHeader = apiKeyInfo.AuthHeader
				}
			}
		}
	}

	if apiKey == "" {
		apiKey = api.GetAPIKey()
	}

	apiClient := api.NewClientWithFullConfig(apiKey, cfg.Model, baseURL, authType, authHeader, version)
	toolRegistry := toolkit.NewRegistry()
	permissionPolicy := permissions.NewPermissionPolicy(permissions.DangerFullAccess)

	session := conversation.NewSession()
	runtime := conversation.NewConversationRuntime(
		session,
		apiClient,
		toolRegistry,
		permissionPolicy,
		buildSystemPrompt(),
	)

	wd, _ := os.Getwd()
	gitClient := git.NewClient(wd)

	return &REPL{
		reader:           bufio.NewReader(os.Stdin),
		writer:           os.Stdout,
		runtime:          runtime,
		config:           cfg,
		toolRegistry:     toolRegistry,
		permissionPolicy: permissionPolicy,
		gitClient:        gitClient,
		history:          make([]string, 0),
		historyIndex:     -1,
		running:          false,
		model:            cfg.Model,
		verbose:          false,
	}, nil
}

// Run 运行 REPL
func (r *REPL) Run() error {
	r.running = true

	r.printWelcome()

	for r.running {
		r.printPrompt()

		input, err := r.readLine()
		if err != nil {
			if err == io.EOF {
				fmt.Fprintln(r.writer, "\nGoodbye!")
				break
			}
			return err
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// 添加到历史
		r.addToHistory(input)

		// 处理命令
		if strings.HasPrefix(input, "/") {
			r.handleSlashCommand(input)
		} else {
			r.handleUserInput(input)
		}
	}

	return nil
}

func (r *REPL) readLine() (string, error) {
	line, err := r.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\n\r"), nil
}

func (r *REPL) addToHistory(line string) {
	r.history = append(r.history, line)
	r.historyIndex = len(r.history)
}

func (r *REPL) printWelcome() {
	fmt.Fprintf(r.writer, `
╔════════════════════════════════════════════╗
║         gClawCoder REPL v1.0.0             ║
║         Type /help for commands            ║
╚════════════════════════════════════════════╝

Model: %s
Permission Mode: %s

`, r.model, r.config.PermissionMode)
}

func (r *REPL) printPrompt() {
	fmt.Fprint(r.writer, "\n\033[32m> \033[0m")
}

func (r *REPL) handleSlashCommand(input string) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return
	}

	cmd := strings.TrimPrefix(parts[0], "/")
	args := parts[1:]

	switch cmd {
	case "help", "h", "?":
		r.printHelp()
	case "quit", "exit", "q":
		r.running = false
	case "clear", "cls":
		r.clearScreen()
	case "model":
		r.handleModelCommand(args)
	case "status":
		r.handleStatusCommand()
	case "tools":
		r.handleToolsCommand()
	case "permissions":
		r.handlePermissionsCommand()
	case "config":
		r.handleConfigCommand(args)
	case "memory":
		r.handleMemoryCommand()
	case "diff":
		r.handleDiffCommand()
	case "history":
		r.handleHistoryCommand(args)
	case "verbose", "v":
		r.verbose = !r.verbose
		fmt.Fprintf(r.writer, "Verbose mode: %v\n", r.verbose)
	default:
		fmt.Fprintf(r.writer, "Unknown command: /%s\n", cmd)
	}
}

func (r *REPL) handleUserInput(input string) {
	fmt.Fprintln(r.writer, "\n\033[90mThinking...\033[0m")

	result, err := r.runtime.RunTurn(input)
	if err != nil {
		fmt.Fprintf(r.writer, "\033[31mError: %v\033[0m\n", err)
		return
	}

	// 显示结果
	for _, msg := range result.AssistantMessages {
		for _, block := range msg.Content {
			if block.Type == conversation.BlockTypeText {
				fmt.Fprintf(r.writer, "\n%s\n", block.Text)
			}
		}
	}

	// 显示工具执行
	for _, toolResult := range result.ToolResults {
		for _, block := range toolResult.Content {
			if block.Type == conversation.BlockTypeToolResult {
				icon := "✓"
				if block.IsError {
					icon = "✗"
				}
				fmt.Fprintf(r.writer, "\n\033[90m%s Tool: %s\033[0m\n", icon, block.ToolUseID)
				fmt.Fprintf(r.writer, "%s\n", block.Content)
			}
		}
	}

	if r.verbose {
		fmt.Fprintf(r.writer, "\n\033[90mIterations: %d\033[0m\n", result.Iterations)
	}
}

func (r *REPL) printHelp() {
	help := `
Available Commands:
  /help, /h, /?     Show this help message
  /quit, /exit, /q  Exit the REPL
  /clear, /cls      Clear the screen
  /model [name]     Show or change the model
  /status           Show session status
  /tools            List available tools
  /permissions      Show or change permission mode
  /config [section] Show configuration
  /memory           Show conversation memory
  /diff             Show git diff
  /history [n]      Show command history
  /verbose, /v      Toggle verbose mode

Examples:
  /model claude-opus-4-6
  /permissions read-only
  /history 10
`
	fmt.Fprintln(r.writer, help)
}

func (r *REPL) handleModelCommand(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(r.writer, "Current model: %s\n", r.model)
		return
	}

	r.model = args[0]
	fmt.Fprintf(r.writer, "Model set to: %s\n", r.model)
}

func (r *REPL) handleStatusCommand() {
	fmt.Fprintf(r.writer, `
Session Status:
  Model: %s
  Messages: %d
  Tools Available: %d
`, r.model, len(r.runtime.Session().Messages), r.toolRegistry.Count())
}

func (r *REPL) handleToolsCommand() {
	tools := r.toolRegistry.List()
	fmt.Fprintln(r.writer, "\nAvailable Tools:")
	for _, tool := range tools {
		fmt.Fprintf(r.writer, "  - %s\n", tool)
	}
}

func (r *REPL) handlePermissionsCommand() {
	fmt.Fprintf(r.writer, "Permission Mode: %s\n", r.config.PermissionMode)
}

func (r *REPL) handleConfigCommand(args []string) {
	fmt.Fprintf(r.writer, `
Configuration:
  Model: %s
  Permission Mode: %s
  Max Tokens: %d
  Max Iterations: %d
  API Base URL: %s
`, r.config.Model, r.config.PermissionMode, r.config.MaxTokens, r.config.MaxIterations, r.config.BaseURL)
}

func (r *REPL) handleMemoryCommand() {
	session := r.runtime.Session()
	fmt.Fprintf(r.writer, "\nConversation Memory (%d messages):\n", len(session.Messages))

	for i, msg := range session.Messages {
		role := string(msg.Role)
		var content string
		for _, block := range msg.Content {
			if block.Type == conversation.BlockTypeText {
				content = block.Text
				break
			}
		}

		preview := content
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}

		fmt.Fprintf(r.writer, "  %d. [%s] %s\n", i+1, role, preview)
	}
}

func (r *REPL) handleDiffCommand() {
	if !r.gitClient.IsGitRepo() {
		fmt.Fprintln(r.writer, "Not a git repository")
		return
	}

	diff, err := r.gitClient.GetDiff()
	if err != nil {
		fmt.Fprintf(r.writer, "Error getting diff: %v\n", err)
		return
	}

	fmt.Fprintf(r.writer, "\nGit Diff:\n")
	fmt.Fprintf(r.writer, "  Files changed: %d\n", diff.Stats.FilesChanged)
	fmt.Fprintf(r.writer, "  Insertions: %d\n", diff.Stats.Insertions)
	fmt.Fprintf(r.writer, "  Deletions: %d\n", diff.Stats.Deletions)

	if len(diff.Patch) > 2000 {
		fmt.Fprintln(r.writer, "\n(Patch truncated, use git diff for full output)")
	} else {
		fmt.Fprintf(r.writer, "\n%s\n", diff.Patch)
	}
}

func (r *REPL) handleHistoryCommand(args []string) {
	limit := 20
	if len(args) > 0 {
		fmt.Sscanf(args[0], "%d", &limit)
	}

	start := len(r.history) - limit
	if start < 0 {
		start = 0
	}

	fmt.Fprintln(r.writer, "\nCommand History:")
	for i := start; i < len(r.history); i++ {
		fmt.Fprintf(r.writer, "  %d: %s\n", i+1, r.history[i])
	}
}

func (r *REPL) clearScreen() {
	fmt.Fprint(r.writer, "\033[2J\033[H")
}

// Session 获取会话
func (r *REPL) Session() *conversation.Session {
	return r.runtime.Session()
}

// GetRuntime 获取运行时
func (r *REPL) GetRuntime() *conversation.ConversationRuntime {
	return r.runtime
}

func buildSystemPrompt() string {
	return `You are gClawCoder, an AI coding assistant.
You can help users with:
- Writing and editing code
- Running commands and scripts
- Searching and analyzing codebases
- Answering technical questions

Be concise and helpful in your responses.`
}

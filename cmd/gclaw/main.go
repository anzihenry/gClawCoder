package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gclawcoder/gclaw/internal/commands"
	"github.com/gclawcoder/gclaw/internal/query"
	"github.com/gclawcoder/gclaw/internal/runtime"
	"github.com/gclawcoder/gclaw/internal/session"
	"github.com/gclawcoder/gclaw/internal/tools"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "summary":
		cmdSummary()
	case "manifest":
		cmdManifest()
	case "subsystems":
		cmdSubsystems(args)
	case "commands":
		cmdCommands(args)
	case "tools":
		cmdTools(args)
	case "route":
		cmdRoute(args)
	case "bootstrap":
		cmdBootstrap(args)
	case "turn-loop":
		cmdTurnLoop(args)
	case "show-command":
		cmdShowCommand(args)
	case "show-tool":
		cmdShowTool(args)
	case "exec-command":
		cmdExecCommand(args)
	case "exec-tool":
		cmdExecTool(args)
	case "load-session":
		cmdLoadSession(args)
	case "repl":
		cmdREPL()
	case "tui":
		cmdTUI()
	case "login":
		cmdLogin()
	case "logout":
		cmdLogout()
	case "whoami":
		cmdWhoami()
	case "model":
		cmdModel(args)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	usage := `gClawCoder - Go Port of Claw Code

Usage:
  gclaw <command> [options]

Commands:
  summary         Render a Markdown summary of the Go porting workspace
  manifest        Print the current Go workspace manifest
  subsystems      List the current Go modules in the workspace
  commands        List mirrored command entries
  tools           List mirrored tool entries
  route           Route a prompt across mirrored command/tool inventories
  bootstrap       Build a runtime session report
  turn-loop       Run a small stateful turn loop
  show-command    Show one mirrored command entry by exact name
  show-tool       Show one mirrored tool entry by exact name
  exec-command    Execute a mirrored command shim by exact name
  exec-tool       Execute a mirrored tool shim by exact name
  load-session    Load a previously persisted session
  repl            Start interactive REPL mode
  login           Authenticate via OAuth
  logout          Clear stored credentials
  whoami          Show current authentication status
  model           View or switch AI model
  tui             Start TUI (Text User Interface)
  help            Show this help message

Examples:
  gclaw summary
  gclaw commands --limit 10
  gclaw tools --query MCP
  gclaw route "review MCP tool"
  gclaw bootstrap "review MCP tool" --limit 5
  gclaw turn-loop "review MCP tool" --max-turns 3
`
	fmt.Println(usage)
}

func cmdSummary() {
	engine := query.FromWorkspace()
	fmt.Println(engine.RenderSummary())
}

func cmdManifest() {
	fmt.Println("Go Workspace Manifest")
	fmt.Println("=====================")
	fmt.Printf("Commands: %d\n", len(commands.PortedCommands()))
	fmt.Printf("Tools: %d\n", len(tools.PortedTools()))
}

func cmdSubsystems(args []string) {
	fs := flag.NewFlagSet("subsystems", flag.ExitOnError)
	limit := fs.Int("limit", 32, "Limit the number of results")
	fs.Parse(args)

	cmds := commands.PortedCommands()
	count := *limit
	if count > len(cmds) {
		count = len(cmds)
	}

	for _, cmd := range cmds[:count] {
		fmt.Printf("%s\t%s\tmirrored\n", cmd.Name, cmd.SourceHint)
	}
}

func cmdCommands(args []string) {
	fs := flag.NewFlagSet("commands", flag.ExitOnError)
	limit := fs.Int("limit", 20, "Limit the number of results")
	queryStr := fs.String("query", "", "Filter by query string")
	noPlugin := fs.Bool("no-plugin-commands", false, "Exclude plugin commands")
	noSkill := fs.Bool("no-skill-commands", false, "Exclude skill commands")
	fs.Parse(args)

	if *queryStr != "" {
		fmt.Println(commands.RenderCommandIndex(*limit, *queryStr))
	} else {
		cmds := commands.GetCommands(!*noPlugin, !*noSkill)
		count := *limit
		if count > len(cmds) {
			count = len(cmds)
		}
		fmt.Printf("Command entries: %d\n\n", len(commands.PortedCommands()))
		for _, cmd := range cmds[:count] {
			fmt.Printf("- %s — %s\n", cmd.Name, cmd.SourceHint)
		}
	}
}

func cmdTools(args []string) {
	fs := flag.NewFlagSet("tools", flag.ExitOnError)
	limit := fs.Int("limit", 20, "Limit the number of results")
	queryStr := fs.String("query", "", "Filter by query string")
	simpleMode := fs.Bool("simple-mode", false, "Simple mode with basic tools")
	noMCP := fs.Bool("no-mcp", false, "Exclude MCP tools")
	denyTool := fs.String("deny-tool", "", "Deny specific tool")
	denyPrefix := fs.String("deny-prefix", "", "Deny tools with prefix")
	fs.Parse(args)

	var blockedTools, blockedPrefixes []string
	if *denyTool != "" {
		blockedTools = []string{*denyTool}
	}
	if *denyPrefix != "" {
		blockedPrefixes = []string{*denyPrefix}
	}

	if *queryStr != "" {
		fmt.Println(tools.RenderToolIndex(*limit, *queryStr))
	} else {
		tls := tools.GetTools(*simpleMode, !*noMCP, blockedTools, blockedPrefixes)
		count := *limit
		if count > len(tls) {
			count = len(tls)
		}
		fmt.Printf("Tool entries: %d\n\n", len(tools.PortedTools()))
		for _, tool := range tls[:count] {
			fmt.Printf("- %s — %s\n", tool.Name, tool.SourceHint)
		}
	}
}

func cmdRoute(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: prompt required")
		os.Exit(1)
	}

	fs := flag.NewFlagSet("route", flag.ExitOnError)
	limit := fs.Int("limit", 5, "Limit the number of results")
	fs.Parse(args)

	prompt := strings.Join(fs.Args(), " ")
	rt := runtime.NewPortRuntime()
	matches := rt.RoutePrompt(prompt, *limit)

	if len(matches) == 0 {
		fmt.Println("No mirrored command/tool matches found.")
		return
	}

	for _, match := range matches {
		fmt.Printf("%s\t%s\t%d\t%s\n", match.Kind, match.Name, match.Score, match.SourceHint)
	}
}

func cmdBootstrap(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: prompt required")
		os.Exit(1)
	}

	fs := flag.NewFlagSet("bootstrap", flag.ExitOnError)
	limit := fs.Int("limit", 5, "Limit the number of results")
	fs.Parse(args)

	prompt := strings.Join(fs.Args(), " ")
	rt := runtime.NewPortRuntime()
	session := rt.BootstrapSession(prompt, *limit)
	fmt.Println(session.AsMarkdown())
}

func cmdTurnLoop(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: prompt required")
		os.Exit(1)
	}

	fs := flag.NewFlagSet("turn-loop", flag.ExitOnError)
	limit := fs.Int("limit", 5, "Limit the number of results")
	maxTurns := fs.Int("max-turns", 3, "Maximum number of turns")
	structured := fs.Bool("structured-output", false, "Enable structured output")
	fs.Parse(args)

	prompt := strings.Join(fs.Args(), " ")
	rt := runtime.NewPortRuntime()
	results := rt.RunTurnLoop(prompt, *limit, *maxTurns, *structured)

	for i, result := range results {
		fmt.Printf("## Turn %d\n", i+1)
		fmt.Println(result.Output)
		fmt.Printf("stop_reason=%s\n\n", result.StopReason)
	}
}

func cmdShowCommand(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: command name required")
		os.Exit(1)
	}

	cmd := commands.GetCommand(args[0])
	if cmd == nil {
		fmt.Printf("Command not found: %s\n", args[0])
		os.Exit(1)
	}

	fmt.Println(cmd.Name)
	fmt.Println(cmd.SourceHint)
	fmt.Println(cmd.Responsibility)
}

func cmdShowTool(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: tool name required")
		os.Exit(1)
	}

	tool := tools.GetTool(args[0])
	if tool == nil {
		fmt.Printf("Tool not found: %s\n", args[0])
		os.Exit(1)
	}

	fmt.Println(tool.Name)
	fmt.Println(tool.SourceHint)
	fmt.Println(tool.Responsibility)
}

func cmdExecCommand(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: command name required")
		os.Exit(1)
	}

	prompt := ""
	if len(args) > 1 {
		prompt = strings.Join(args[1:], " ")
	}

	result := commands.ExecuteCommand(args[0], prompt)
	fmt.Println(result.Message)
	if !result.Handled {
		os.Exit(1)
	}
}

func cmdExecTool(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: tool name required")
		os.Exit(1)
	}

	payload := ""
	if len(args) > 1 {
		payload = strings.Join(args[1:], " ")
	}

	result := tools.ExecuteTool(args[0], payload)
	fmt.Println(result.Message)
	if !result.Handled {
		os.Exit(1)
	}
}

func cmdLoadSession(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: session ID required")
		os.Exit(1)
	}

	sess, err := session.LoadSession(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading session: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s\n", sess.SessionID)
	fmt.Printf("%d messages\n", len(sess.Messages))
	fmt.Printf("in=%d out=%d\n", sess.InputTokens, sess.OutputTokens)
}

// Helper function to parse int args
func parseIntArg(args []string, name string, defaultValue int) int {
	for i, arg := range args {
		if arg == name || strings.HasPrefix(arg, name+"=") {
			var value string
			if strings.HasPrefix(arg, name+"=") {
				value = strings.TrimPrefix(arg, name+"=")
			} else if i+1 < len(args) {
				value = args[i+1]
			}
			if value != "" {
				if parsed, err := strconv.Atoi(value); err == nil {
					return parsed
				}
			}
		}
	}
	return defaultValue
}

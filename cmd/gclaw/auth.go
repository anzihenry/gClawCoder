package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gclawcoder/gclaw/internal/api"
	"github.com/gclawcoder/gclaw/internal/tui"
)

func cmdLogin() {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	force := fs.Bool("force", false, "Force re-authentication")
	method := fs.String("method", "", "Authentication method: oauth or apikey")
	provider := fs.String("provider", "", "AI provider: anthropic, openai, alibaba, azure, or custom")
	fs.Parse(os.Args[2:])

	// 如果指定了 API Key 方式
	if *method == "apikey" {
		cmdLoginAPIKey(*force, *provider)
		return
	}

	// 如果指定了 OAuth 方式
	if *method == "oauth" {
		cmdLoginOAuth(*force)
		return
	}

	// 未指定方式，检查当前状态
	oauthClient := api.NewOAuthClient(nil)
	apiKeyClient := api.NewAPIKeyClient()

	if oauthClient.IsLoggedIn() {
		fmt.Println("⚠️  Already logged in with OAuth")
		fmt.Println("\nToken info:")
		info, err := oauthClient.GetTokenInfo()
		if err == nil {
			fmt.Printf("  Token Type: %s\n", info["token_type"])
			fmt.Printf("  Scope: %s\n", info["scope"])
			fmt.Printf("  Expires: %s\n", info["expiry"])
			fmt.Printf("  Expired: %v\n", info["expired"])
		}
		fmt.Println("\nUse 'gclaw logout' to log out, or 'gclaw login --force' to re-authenticate")
		return
	}

	if apiKeyClient.IsConfigured() {
		fmt.Println("⚠️  Already configured with API Key")
		info, err := apiKeyClient.GetKeyInfo()
		if err == nil {
			fmt.Printf("  Provider: %v\n", info["provider"])
			fmt.Printf("  Base URL: %v\n", info["base_url"])
			fmt.Printf("  Model: %v\n", info["model"])
			fmt.Printf("  Key: %v\n", info["key_prefix"])
		}
		fmt.Println("\nUse 'gclaw logout' to clear, or 'gclaw login --method apikey --force' to update")
		return
	}

	// 默认使用 OAuth 登录
	fmt.Println("No authentication method specified, defaulting to OAuth...")
	fmt.Println("Use 'gclaw login --method apikey' to use API Key instead")
	fmt.Println()
	cmdLoginOAuth(*force)
}

func cmdLoginOAuth(force bool) {
	client := api.NewOAuthClient(nil)

	// 检查是否已经登录
	if client.IsLoggedIn() && !force {
		fmt.Println("⚠️  Already logged in")
		fmt.Println("\nToken info:")
		info, err := client.GetTokenInfo()
		if err == nil {
			fmt.Printf("  Token Type: %s\n", info["token_type"])
			fmt.Printf("  Scope: %s\n", info["scope"])
			fmt.Printf("  Expires: %s\n", info["expiry"])
			fmt.Printf("  Expired: %v\n", info["expired"])
		}
		fmt.Println("\nUse 'gclaw logout' to log out, or 'gclaw login --force' to re-authenticate")
		return
	}

	// 执行登录
	token, err := client.Login()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Login failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n📋 Token Information:")
	fmt.Printf("  Token Type: %s\n", token.TokenType)
	fmt.Printf("  Scope: %s\n", token.Scope)
	fmt.Printf("  Expires In: %d seconds\n", token.ExpiresIn)
	fmt.Printf("  Token Path: %s\n", client.GetTokenPath())
}

func cmdLoginAPIKey(force bool, providerFlag string) {
	client := api.NewAPIKeyClient()

	// 检查是否已经配置
	if client.IsConfigured() && !force {
		fmt.Println("⚠️  API Key already configured")
		info, err := client.GetKeyInfo()
		if err == nil {
			fmt.Printf("  Provider: %v\n", info["provider"])
			fmt.Printf("  Base URL: %v\n", info["base_url"])
			fmt.Printf("  Model: %v\n", info["model"])
			fmt.Printf("  Key: %v\n", info["key_prefix"])
		}
		fmt.Println("\nUse 'gclaw logout' to clear, or 'gclaw login --method apikey --force' to update")
		return
	}

	fmt.Println("🔑 API Key Setup")
	fmt.Println("═══════════════════════════════════════")
	fmt.Println()

	// 显示支持的提供商
	providers := api.ListProviders()
	if providerFlag == "" {
		fmt.Println("Supported AI providers:")
		fmt.Println()
		for i, p := range providers {
			fmt.Printf("  %d. %s\n", i+1, p.Name)
			fmt.Printf("     URL: %s\n", p.BaseURL)
			fmt.Printf("     Default model: %s\n", p.Model)
			fmt.Println()
		}
		fmt.Println("  6. custom (Custom OpenAI-compatible API)")
		fmt.Println()

		reader := bufio.NewReader(os.Stdin)
		fmt.Println("Select provider (1-6, or enter number):")
		fmt.Print("> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		var choice int
		if parsed, err := strconv.Atoi(input); err == nil {
			choice = parsed
		}

		if choice < 1 || choice > 6 {
			fmt.Fprintln(os.Stderr, "❌ Invalid selection")
			os.Exit(1)
		}

		if choice == 6 {
			cmdLoginCustomAPIKey(client, force)
			return
		}

		providerFlag = string(providers[choice-1].Name)
	}

	// 获取 API Key
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Enter your API Key for %s:\n", providerFlag)
	fmt.Print("> ")
	apiKey, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to read API Key: %v\n", err)
		os.Exit(1)
	}
	apiKey = strings.TrimSpace(apiKey)

	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "❌ API Key cannot be empty")
		os.Exit(1)
	}

	// 获取模型名称（可选）
	fmt.Println()
	fmt.Println("Enter model name (press Enter for default):")
	fmt.Print("> ")
	model, _ := reader.ReadString('\n')
	model = strings.TrimSpace(model)

	// 获取描述（可选）
	fmt.Println()
	fmt.Println("Enter description (optional, press Enter to skip):")
	fmt.Print("> ")
	description, _ := reader.ReadString('\n')
	description = strings.TrimSpace(description)

	// 设置 API Key
	provider := api.Provider(providerFlag)
	if err := client.SetKey(provider, apiKey, model, description); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to set API Key: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("✅ API Key configured successfully!")
	fmt.Printf("Key saved to: %s\n", client.GetKeyPath())
	fmt.Println()
	fmt.Println("📋 Configuration:")
	info, _ := client.GetKeyInfo()
	fmt.Printf("  Provider: %v\n", info["provider"])
	fmt.Printf("  Base URL: %v\n", info["base_url"])
	fmt.Printf("  Model: %v\n", info["model"])
	fmt.Printf("  Key: %v\n", info["key_prefix"])
	if desc := info["description"]; desc != nil && desc.(string) != "" {
		fmt.Printf("  Description: %v\n", desc)
	}
}

func cmdLoginCustomAPIKey(client *api.APIKeyClient, force bool) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Println("🔧 Custom Provider Configuration")
	fmt.Println("═══════════════════════════════════════")
	fmt.Println()

	fmt.Println("Enter provider name (e.g., 'myprovider'):")
	fmt.Print("> ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	if name == "" {
		fmt.Fprintln(os.Stderr, "❌ Provider name cannot be empty")
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("Enter base URL (e.g., 'https://api.example.com/v1'):")
	fmt.Print("> ")
	baseURL, _ := reader.ReadString('\n')
	baseURL = strings.TrimSpace(baseURL)

	if baseURL == "" {
		fmt.Fprintln(os.Stderr, "❌ Base URL cannot be empty")
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("Enter API Key:")
	fmt.Print("> ")
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)

	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "❌ API Key cannot be empty")
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("Enter model name:")
	fmt.Print("> ")
	model, _ := reader.ReadString('\n')
	model = strings.TrimSpace(model)

	fmt.Println()
	fmt.Println("Auth type (bearer/header/query, default: bearer):")
	fmt.Print("> ")
	authType, _ := reader.ReadString('\n')
	authType = strings.TrimSpace(authType)

	fmt.Println()
	fmt.Println("Auth header name (default: Authorization):")
	fmt.Print("> ")
	authHeader, _ := reader.ReadString('\n')
	authHeader = strings.TrimSpace(authHeader)

	fmt.Println()
	fmt.Println("API version (optional, press Enter to skip):")
	fmt.Print("> ")
	version, _ := reader.ReadString('\n')
	version = strings.TrimSpace(version)

	fmt.Println()
	fmt.Println("Enter description (optional, press Enter to skip):")
	fmt.Print("> ")
	description, _ := reader.ReadString('\n')
	description = strings.TrimSpace(description)

	if err := client.SetCustomKey(name, baseURL, apiKey, model, authType, authHeader, version, description); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to set custom API Key: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("✅ Custom API Key configured successfully!")
	fmt.Printf("Key saved to: %s\n", client.GetKeyPath())
	fmt.Println()
	fmt.Println("📋 Configuration:")
	info, _ := client.GetKeyInfo()
	fmt.Printf("  Provider: %v\n", info["provider"])
	fmt.Printf("  Base URL: %v\n", info["base_url"])
	fmt.Printf("  Model: %v\n", info["model"])
	fmt.Printf("  Key: %v\n", info["key_prefix"])
	if desc := info["description"]; desc != nil && desc.(string) != "" {
		fmt.Printf("  Description: %v\n", desc)
	}
}

func cmdLogout() {
	fs := flag.NewFlagSet("logout", flag.ExitOnError)
	method := fs.String("method", "", "Authentication method to clear: oauth or apikey")
	fs.Parse(os.Args[2:])

	// 如果指定了方法
	if *method == "oauth" {
		cmdLogoutOAuth()
		return
	}
	if *method == "apikey" {
		cmdLogoutAPIKey()
		return
	}

	// 未指定方法，清除所有认证
	oauthCleared := false
	apiKeyCleared := false

	oauthClient := api.NewOAuthClient(nil)
	if oauthClient.IsLoggedIn() {
		if err := oauthClient.Logout(); err == nil {
			oauthCleared = true
		}
	}

	apiKeyClient := api.NewAPIKeyClient()
	if apiKeyClient.IsConfigured() {
		if err := apiKeyClient.ClearKey(); err == nil {
			apiKeyCleared = true
		}
	}

	if oauthCleared || apiKeyCleared {
		fmt.Println("✅ Logged out successfully")
		if oauthCleared {
			fmt.Println("  - OAuth token cleared")
		}
		if apiKeyCleared {
			fmt.Println("  - API Key cleared")
		}
	} else {
		fmt.Println("No active authentication found")
	}
}

func cmdLogoutOAuth() {
	client := api.NewOAuthClient(nil)

	if !client.IsLoggedIn() {
		fmt.Println("Not logged in with OAuth")
		os.Exit(1)
	}

	if err := client.Logout(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Logout failed: %v\n", err)
		os.Exit(1)
	}
}

func cmdLogoutAPIKey() {
	client := api.NewAPIKeyClient()

	if !client.IsConfigured() {
		fmt.Println("No API Key configured")
		os.Exit(1)
	}

	if err := client.ClearKey(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to clear API Key: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ API Key cleared successfully")
}

func cmdWhoami() {
	oauthClient := api.NewOAuthClient(nil)
	apiKeyClient := api.NewAPIKeyClient()

	oauthLoggedIn := oauthClient.IsLoggedIn()
	apiKeyConfigured := apiKeyClient.IsConfigured()

	if !oauthLoggedIn && !apiKeyConfigured {
		fmt.Println("Not logged in")
		fmt.Println()
		fmt.Println("Authentication options:")
		fmt.Println("  1. OAuth:  gclaw login --method oauth")
		fmt.Println("  2. API Key: gclaw login --method apikey")
		fmt.Println("     Supported providers:")
		fmt.Println("       - anthropic (Claude)")
		fmt.Println("       - openai (GPT-4)")
		fmt.Println("       - alibaba (Qwen/通义千问)")
		fmt.Println("       - azure (Azure OpenAI)")
		fmt.Println("       - custom (Any OpenAI-compatible API)")
		fmt.Println("  3. Default (OAuth): gclaw login")
		os.Exit(1)
	}

	if oauthLoggedIn {
		info, err := oauthClient.GetTokenInfo()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get token info: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("✅ Logged in with OAuth")
		fmt.Println()
		fmt.Println("Token Information:")
		fmt.Printf("  Token Type: %v\n", info["token_type"])
		fmt.Printf("  Scope: %v\n", info["scope"])
		fmt.Printf("  Expires: %v\n", info["expiry"])
		fmt.Printf("  Expired: %v\n", info["expired"])

		if scopes, ok := info["scopes"].([]string); ok {
			fmt.Println()
			fmt.Println("  Scopes:")
			for _, scope := range scopes {
				fmt.Printf("    - %s\n", scope)
			}
		}
	}

	if apiKeyConfigured {
		info, err := apiKeyClient.GetKeyInfo()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get API Key info: %v\n", err)
			os.Exit(1)
		}

		if oauthLoggedIn {
			fmt.Println()
		}
		fmt.Println("✅ API Key configured")
		fmt.Println()
		fmt.Println("Configuration:")
		fmt.Printf("  Provider: %v\n", info["provider"])
		fmt.Printf("  Base URL: %v\n", info["base_url"])
		fmt.Printf("  Model: %v\n", info["model"])
		fmt.Printf("  Key: %v\n", info["key_prefix"])
		if desc := info["description"]; desc != nil && desc.(string) != "" {
			fmt.Printf("  Description: %v\n", desc)
		}
	}
}

func cmdTUI() {
	t, err := tui.NewTUI()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating TUI: %v\n", err)
		os.Exit(1)
	}

	if err := t.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}

func cmdModel(args []string) {
	fs := flag.NewFlagSet("model", flag.ExitOnError)
	list := fs.Bool("list", false, "List available models")
	set := fs.String("set", "", "Set current model")
	fs.Parse(args)

	client := api.NewAPIKeyClient()

	if !client.IsConfigured() {
		fmt.Println("No API Key configured")
		fmt.Println()
		fmt.Println("Use 'gclaw login --method apikey' to configure")
		os.Exit(1)
	}

	// 获取当前模型
	currentModel, err := client.GetCurrentModel()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get current model: %v\n", err)
		os.Exit(1)
	}

	// 列出可用模型
	if *list {
		models, err := client.GetAvailableModels()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get available models: %v\n", err)
			os.Exit(1)
		}

		config, _ := client.GetConfig()
		fmt.Printf("Available models for %s:\n", config.Provider)
		fmt.Println("═══════════════════════════════════════")

		if len(models) == 0 {
			fmt.Println("No predefined models (custom provider)")
			fmt.Printf("Current model: %s\n", currentModel)
		} else {
			for i, model := range models {
				marker := " "
				if model == currentModel {
					marker = "✓"
				}
				fmt.Printf("%s %d. %s\n", marker, i+1, model)
			}
		}
		fmt.Println()
		fmt.Printf("Current model: %s\n", currentModel)
		fmt.Println()
		fmt.Println("Use 'gclaw model --set <model-name>' to switch")
		return
	}

	// 设置模型
	if *set != "" {
		if err := client.SetModel(*set); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to set model: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Model switched to: %s\n", *set)
		return
	}

	// 默认显示当前模型
	fmt.Printf("Current model: %s\n", currentModel)
	fmt.Println()
	fmt.Println("Use 'gclaw model --list' to see available models")
	fmt.Println("Use 'gclaw model --set <model-name>' to switch")
}

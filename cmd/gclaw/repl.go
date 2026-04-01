package main

import (
	"fmt"
	"os"

	"github.com/gclawcoder/gclaw/internal/config"
	"github.com/gclawcoder/gclaw/internal/repl"
)

func cmdREPL() {
	// 加载配置
	cwd, _ := os.Getwd()
	loader := config.DefaultConfigLoader(cwd)
	cfg, err := loader.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
		cfg = &config.RuntimeConfig{
			Model:          "claude-sonnet-4-20250514",
			PermissionMode: "danger-full-access",
			MaxTokens:      4096,
			MaxIterations:  100,
			APIKey:         os.Getenv("ANTHROPIC_API_KEY"),
		}
	}

	// 创建 REPL
	replInst, err := repl.NewREPL(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating REPL: %v\n", err)
		os.Exit(1)
	}

	// 运行 REPL
	if err := replInst.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "REPL error: %v\n", err)
		os.Exit(1)
	}
}

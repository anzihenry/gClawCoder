package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gclawcoder/gclaw/internal/api"
)

// ConfigSource 配置来源
type ConfigSource int

const (
	SourceUser ConfigSource = iota
	SourceProject
	SourceLocal
)

// ConfigEntry 配置条目
type ConfigEntry struct {
	Source ConfigSource
	Path   string
}

// RuntimeConfig 运行时配置
type RuntimeConfig struct {
	Merged         map[string]interface{}
	LoadedEntries  []ConfigEntry
	Model          string
	PermissionMode string
	APIKey         string
	BaseURL        string
	AuthType       string
	AuthHeader     string
	Version        string
	MaxTokens      int
	MaxIterations  int
}

// ConfigLoader 配置加载器
type ConfigLoader struct {
	CWD        string
	ConfigHome string
}

// NewConfigLoader 创建配置加载器
func NewConfigLoader(cwd, configHome string) *ConfigLoader {
	return &ConfigLoader{
		CWD:        cwd,
		ConfigHome: configHome,
	}
}

// DefaultConfigLoader 创建默认配置加载器
func DefaultConfigLoader(cwd string) *ConfigLoader {
	configHome := getDefaultConfigHome()
	return &ConfigLoader{
		CWD:        cwd,
		ConfigHome: configHome,
	}
}

// Discover 发现配置文件
func (l *ConfigLoader) Discover() []ConfigEntry {
	entries := []ConfigEntry{
		{Source: SourceUser, Path: filepath.Join(l.ConfigHome, "settings.json")},
		{Source: SourceProject, Path: filepath.Join(l.CWD, ".claw.json")},
		{Source: SourceProject, Path: filepath.Join(l.CWD, ".claw", "settings.json")},
		{Source: SourceLocal, Path: filepath.Join(l.CWD, ".claw", "settings.local.json")},
	}
	return entries
}

// Load 加载配置
func (l *ConfigLoader) Load() (*RuntimeConfig, error) {
	merged := make(map[string]interface{})
	var loadedEntries []ConfigEntry

	for _, entry := range l.Discover() {
		data, err := os.ReadFile(entry.Path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("failed to read config %s: %w", entry.Path, err)
		}

		var config map[string]interface{}
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse config %s: %w", entry.Path, err)
		}

		deepMerge(merged, config)
		loadedEntries = append(loadedEntries, entry)
	}

	config := &RuntimeConfig{
		Merged:         merged,
		LoadedEntries:  loadedEntries,
		Model:          "claude-sonnet-4-20250514",
		PermissionMode: "danger-full-access",
		MaxTokens:      4096,
		MaxIterations:  100,
	}

	// 从合并配置中提取值
	if model, ok := merged["model"].(string); ok && model != "" {
		config.Model = model
	}
	if mode, ok := merged["permissionMode"].(string); ok && mode != "" {
		config.PermissionMode = mode
	}
	if maxTokens, ok := merged["maxTokens"].(float64); ok {
		config.MaxTokens = int(maxTokens)
	}

	// 优先从 API Key 配置文件读取
	apiKeyClient := api.NewAPIKeyClient()
	if apiKeyClient.IsConfigured() {
		apiKeyInfo, err := apiKeyClient.GetConfig()
		if err == nil {
			config.APIKey = apiKeyInfo.APIKey
			config.BaseURL = apiKeyInfo.BaseURL
			config.AuthType = apiKeyInfo.AuthType
			config.AuthHeader = apiKeyInfo.AuthHeader
			config.Version = apiKeyInfo.Version
			if apiKeyInfo.Model != "" {
				config.Model = apiKeyInfo.Model
			}
		}
	}

	// 环境变量覆盖配置文件
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		config.APIKey = apiKey
	}
	if baseURL := os.Getenv("ANTHROPIC_BASE_URL"); baseURL != "" {
		config.BaseURL = baseURL
	}

	return config, nil
}

// deepMerge 深度合并两个 map
func deepMerge(target, source map[string]interface{}) {
	for key, value := range source {
		if existing, ok := target[key]; ok {
			if existingMap, ok := existing.(map[string]interface{}); ok {
				if newMap, ok := value.(map[string]interface{}); ok {
					deepMerge(existingMap, newMap)
					continue
				}
			}
		}
		target[key] = value
	}
}

func getDefaultConfigHome() string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return ".claw"
	}

	configDir := filepath.Join(home, ".config", "claw")
	if _, err := os.Stat(configDir); err == nil {
		return configDir
	}

	return filepath.Join(home, ".claw")
}

// Get 获取配置值
func (c *RuntimeConfig) Get(key string, defaultValue interface{}) interface{} {
	if value, ok := c.Merged[key]; ok {
		return value
	}
	return defaultValue
}

// GetString 获取字符串值
func (c *RuntimeConfig) GetString(key, defaultValue string) string {
	if value, ok := c.Merged[key].(string); ok {
		return value
	}
	return defaultValue
}

// GetInt 获取整数值
func (c *RuntimeConfig) GetInt(key string, defaultValue int) int {
	if value, ok := c.Merged[key].(float64); ok {
		return int(value)
	}
	return defaultValue
}

// Save 保存配置
func (c *RuntimeConfig) Save(path string) error {
	data, err := json.MarshalIndent(c.Merged, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// InitDefault 初始化默认配置
func InitDefault(cwd string) error {
	configPath := filepath.Join(cwd, ".claw.json")

	if _, err := os.Stat(configPath); err == nil {
		return nil // 已存在
	}

	defaultConfig := map[string]interface{}{
		"model":          "claude-sonnet-4-20250514",
		"permissionMode": "workspace-write",
		"maxTokens":      4096,
		"maxIterations":  100,
		"theme":          "dark",
	}

	data, err := json.MarshalIndent(defaultConfig, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

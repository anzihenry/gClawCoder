package api

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Provider 支持的 AI 提供商
type Provider string

const (
	ProviderAnthropic Provider = "anthropic"
	ProviderOpenAI    Provider = "openai"
	ProviderAlibaba   Provider = "alibaba"
	ProviderAzure     Provider = "azure"
	ProviderCustom    Provider = "custom"
)

// ProviderConfig 提供商配置
type ProviderConfig struct {
	Name       Provider `json:"name"`
	BaseURL    string   `json:"base_url"`
	APIKey     string   `json:"api_key"`
	Model      string   `json:"model"`
	AuthType   string   `json:"auth_type"` // "header", "bearer", "query"
	AuthHeader string   `json:"auth_header"`
	Version    string   `json:"version,omitempty"`
	Models     []string `json:"models,omitempty"` // 支持的模型列表
}

// DefaultProviders 默认提供商配置
var DefaultProviders = map[Provider]ProviderConfig{
	ProviderAnthropic: {
		Name:       ProviderAnthropic,
		BaseURL:    "https://api.anthropic.com",
		AuthType:   "header",
		AuthHeader: "x-api-key",
		Version:    "2023-06-01",
		Model:      "claude-sonnet-4-20250514",
		Models: []string{
			"claude-sonnet-4-20250514",
			"claude-3-7-sonnet-20250219",
			"claude-3-5-sonnet-20241022",
			"claude-3-opus-20240229",
			"claude-3-haiku-20240307",
		},
	},
	ProviderOpenAI: {
		Name:       ProviderOpenAI,
		BaseURL:    "https://api.openai.com/v1",
		AuthType:   "bearer",
		AuthHeader: "Authorization",
		Model:      "gpt-4o",
		Models: []string{
			"gpt-4o",
			"gpt-4o-mini",
			"gpt-4-turbo",
			"gpt-4",
			"gpt-3.5-turbo",
			"o1-preview",
			"o1-mini",
		},
	},
	ProviderAlibaba: {
		Name:       ProviderAlibaba,
		BaseURL:    "https://dashscope.aliyuncs.com/compatible-mode/v1",
		AuthType:   "bearer",
		AuthHeader: "Authorization",
		Model:      "qwen-plus",
		Models: []string{
			"qwen3-max",
			"qwen3-max-preview",
			"qwen-max",
			"qwen-max-latest",
			"qwen3.5-plus",
			"qwen-plus",
			"qwen-plus-latest",
			"qwen3.5-flash",
			"qwen-flash",
			"qwen-turbo",
			"qwen-coder-plus",
			"qwen-coder-turbo",
			"qwq-plus",
			"qwen2.5-72b-instruct",
			"qwen2.5-32b-instruct",
		},
	},
	ProviderAzure: {
		Name:       ProviderAzure,
		BaseURL:    "https://{resource}.openai.azure.com/openai/deployments/{deployment}",
		AuthType:   "query",
		AuthHeader: "api-key",
		Model:      "gpt-4",
		Models: []string{
			"gpt-4",
			"gpt-4-turbo",
			"gpt-35-turbo",
		},
	},
}

// APIKeyInfo API Key 信息
type APIKeyInfo struct {
	Provider    Provider `json:"provider"`
	BaseURL     string   `json:"base_url"`
	APIKey      string   `json:"api_key"`
	Model       string   `json:"model"`
	Description string   `json:"description,omitempty"`
	AuthType    string   `json:"auth_type,omitempty"`
	AuthHeader  string   `json:"auth_header,omitempty"`
	Version     string   `json:"version,omitempty"`
}

// APIKeyClient API Key 客户端
type APIKeyClient struct {
	config  *APIKeyInfo
	keyPath string
}

// NewAPIKeyClient 创建 API Key 客户端
func NewAPIKeyClient() *APIKeyClient {
	home, _ := os.UserHomeDir()
	keyPath := filepath.Join(home, ".claw", "apikey.json")

	return &APIKeyClient{
		keyPath: keyPath,
	}
}

// SetKey 设置 API Key（使用默认提供商配置）
func (a *APIKeyClient) SetKey(provider Provider, apiKey, model, description string) error {
	if apiKey == "" {
		return fmt.Errorf("api key cannot be empty")
	}

	config, ok := DefaultProviders[provider]
	if !ok {
		return fmt.Errorf("unknown provider: %s", provider)
	}

	a.config = &APIKeyInfo{
		Provider:    provider,
		BaseURL:     config.BaseURL,
		APIKey:      strings.TrimSpace(apiKey),
		Model:       model,
		Description: description,
		AuthType:    config.AuthType,
		AuthHeader:  config.AuthHeader,
		Version:     config.Version,
	}

	return a.saveKey()
}

// SetCustomKey 设置自定义 API Key
func (a *APIKeyClient) SetCustomKey(name, baseURL, apiKey, model, authType, authHeader, version, description string) error {
	if apiKey == "" {
		return fmt.Errorf("api key cannot be empty")
	}

	if baseURL == "" {
		return fmt.Errorf("base URL cannot be empty")
	}

	if authType == "" {
		authType = "bearer"
	}

	if authHeader == "" {
		authHeader = "Authorization"
	}

	a.config = &APIKeyInfo{
		Provider:    ProviderCustom,
		BaseURL:     strings.TrimRight(baseURL, "/"),
		APIKey:      strings.TrimSpace(apiKey),
		Model:       model,
		Description: description,
		AuthType:    authType,
		AuthHeader:  authHeader,
		Version:     version,
	}

	return a.saveKey()
}

// GetConfig 获取完整配置
func (a *APIKeyClient) GetConfig() (*APIKeyInfo, error) {
	if a.config != nil {
		return a.config, nil
	}

	if err := a.loadKey(); err != nil {
		return nil, fmt.Errorf("no API key available, please set one first: %w", err)
	}

	return a.config, nil
}

// GetKey 获取 API Key
func (a *APIKeyClient) GetKey() (string, error) {
	config, err := a.GetConfig()
	if err != nil {
		return "", err
	}
	return config.APIKey, nil
}

// GetBaseURL 获取 Base URL
func (a *APIKeyClient) GetBaseURL() (string, error) {
	config, err := a.GetConfig()
	if err != nil {
		return "", err
	}
	return config.BaseURL, nil
}

// GetModel 获取模型名称
func (a *APIKeyClient) GetModel() (string, error) {
	config, err := a.GetConfig()
	if err != nil {
		return "", err
	}
	return config.Model, nil
}

// IsConfigured 检查是否已配置
func (a *APIKeyClient) IsConfigured() bool {
	if a.config != nil {
		return true
	}

	if err := a.loadKey(); err != nil {
		return false
	}

	return a.config != nil && a.config.APIKey != ""
}

// ClearKey 清除 API Key
func (a *APIKeyClient) ClearKey() error {
	if err := os.Remove(a.keyPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove API key file: %w", err)
	}

	a.config = nil
	return nil
}

// GetKeyInfo 获取 API Key 信息
func (a *APIKeyClient) GetKeyInfo() (map[string]interface{}, error) {
	if err := a.loadKey(); err != nil {
		return nil, err
	}

	if a.config == nil || a.config.APIKey == "" {
		return nil, fmt.Errorf("no API key configured")
	}

	info := map[string]interface{}{
		"auth_type":   "api_key",
		"provider":    string(a.config.Provider),
		"base_url":    a.config.BaseURL,
		"model":       a.config.Model,
		"description": a.config.Description,
		"key_prefix":  a.maskKey(a.config.APIKey),
	}

	return info, nil
}

// ListProviders 列出所有支持的提供商
func ListProviders() []ProviderConfig {
	providers := make([]ProviderConfig, 0, len(DefaultProviders))
	for _, config := range DefaultProviders {
		providers = append(providers, config)
	}
	return providers
}

// maskKey 掩码显示 API Key
func (a *APIKeyClient) maskKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

// saveKey 保存 API Key 到文件
func (a *APIKeyClient) saveKey() error {
	dir := filepath.Dir(a.keyPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(a.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal API key: %w", err)
	}

	if err := os.WriteFile(a.keyPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write API key file: %w", err)
	}

	return nil
}

// loadKey 从文件加载 API Key
func (a *APIKeyClient) loadKey() error {
	data, err := os.ReadFile(a.keyPath)
	if err != nil {
		return fmt.Errorf("failed to read API key file: %w", err)
	}

	var config APIKeyInfo
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse API key: %w", err)
	}

	a.config = &config
	return nil
}

// GetKeyPath 获取 API Key 文件路径
func (a *APIKeyClient) GetKeyPath() string {
	return a.keyPath
}

// SetModel 设置/切换模型
func (a *APIKeyClient) SetModel(model string) error {
	if err := a.loadKey(); err != nil {
		return fmt.Errorf("no API key configured: %w", err)
	}

	if model == "" {
		return fmt.Errorf("model name cannot be empty")
	}

	a.config.Model = strings.TrimSpace(model)
	return a.saveKey()
}

// GetAvailableModels 获取当前 Provider 可用的模型列表
func (a *APIKeyClient) GetAvailableModels() ([]string, error) {
	if err := a.loadKey(); err != nil {
		return nil, fmt.Errorf("no API key configured: %w", err)
	}

	config, ok := DefaultProviders[a.config.Provider]
	if !ok {
		// Custom provider, return current model only
		if a.config.Model != "" {
			return []string{a.config.Model}, nil
		}
		return []string{}, nil
	}

	return config.Models, nil
}

// GetCurrentModel 获取当前模型
func (a *APIKeyClient) GetCurrentModel() (string, error) {
	config, err := a.GetConfig()
	if err != nil {
		return "", err
	}
	return config.Model, nil
}

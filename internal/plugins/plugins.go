package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PluginInfo 插件信息
type PluginInfo struct {
	Name        string          `json:"name"`
	Version     string          `json:"version"`
	Description string          `json:"description"`
	Author      string          `json:"author"`
	Tools       []PluginTool    `json:"tools"`
	Commands    []PluginCommand `json:"commands"`
}

// PluginTool 插件工具
type PluginTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Handler     string      `json:"handler"`
	InputSchema interface{} `json:"inputSchema"`
}

// PluginCommand 插件命令
type PluginCommand struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Handler     string `json:"handler"`
}

// PluginManager 插件管理器
type PluginManager struct {
	plugins   map[string]*PluginInfo
	pluginDir string
	enabled   map[string]bool
}

// NewPluginManager 创建插件管理器
func NewPluginManager(pluginDir string) *PluginManager {
	return &PluginManager{
		plugins:   make(map[string]*PluginInfo),
		pluginDir: pluginDir,
		enabled:   make(map[string]bool),
	}
}

// LoadPlugins 加载所有插件
func (pm *PluginManager) LoadPlugins() error {
	if pm.pluginDir == "" {
		return nil
	}

	entries, err := os.ReadDir(pm.pluginDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginPath := filepath.Join(pm.pluginDir, entry.Name())
		if err := pm.loadPlugin(pluginPath); err != nil {
			fmt.Printf("Warning: failed to load plugin %s: %v\n", entry.Name(), err)
		}
	}

	return nil
}

func (pm *PluginManager) loadPlugin(pluginPath string) error {
	manifestPath := filepath.Join(pluginPath, "plugin.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}

	var info PluginInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return err
	}

	pm.plugins[info.Name] = &info
	pm.enabled[info.Name] = true

	return nil
}

// ListPlugins 列出所有插件
func (pm *PluginManager) ListPlugins() []*PluginInfo {
	plugins := make([]*PluginInfo, 0, len(pm.plugins))
	for _, p := range pm.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}

// GetPlugin 获取插件
func (pm *PluginManager) GetPlugin(name string) *PluginInfo {
	return pm.plugins[name]
}

// EnablePlugin 启用插件
func (pm *PluginManager) EnablePlugin(name string) error {
	if _, ok := pm.plugins[name]; !ok {
		return fmt.Errorf("plugin not found: %s", name)
	}
	pm.enabled[name] = true
	return nil
}

// DisablePlugin 禁用插件
func (pm *PluginManager) DisablePlugin(name string) error {
	if _, ok := pm.plugins[name]; !ok {
		return fmt.Errorf("plugin not found: %s", name)
	}
	pm.enabled[name] = false
	return nil
}

// IsEnabled 检查插件是否启用
func (pm *PluginManager) IsEnabled(name string) bool {
	return pm.enabled[name]
}

// ExecuteTool 执行插件工具
func (pm *PluginManager) ExecuteTool(pluginName, toolName string, input interface{}) (string, error) {
	plugin := pm.plugins[pluginName]
	if plugin == nil {
		return "", fmt.Errorf("plugin not found: %s", pluginName)
	}

	if !pm.enabled[pluginName] {
		return "", fmt.Errorf("plugin is disabled: %s", pluginName)
	}

	// 查找工具
	var tool *PluginTool
	for i := range plugin.Tools {
		if plugin.Tools[i].Name == toolName {
			tool = &plugin.Tools[i]
			break
		}
	}

	if tool == nil {
		return "", fmt.Errorf("tool not found: %s", toolName)
	}

	// 执行处理器
	return pm.executeHandler(tool.Handler, input)
}

// ExecuteCommand 执行插件命令
func (pm *PluginManager) ExecuteCommand(pluginName, commandName string, args []string) (string, error) {
	plugin := pm.plugins[pluginName]
	if plugin == nil {
		return "", fmt.Errorf("plugin not found: %s", pluginName)
	}

	if !pm.enabled[pluginName] {
		return "", fmt.Errorf("plugin is disabled: %s", pluginName)
	}

	// 查找命令
	var cmd *PluginCommand
	for i := range plugin.Commands {
		if plugin.Commands[i].Name == commandName {
			cmd = &plugin.Commands[i]
			break
		}
	}

	if cmd == nil {
		return "", fmt.Errorf("command not found: %s", commandName)
	}

	return pm.executeHandler(cmd.Handler, args)
}

func (pm *PluginManager) executeHandler(handler string, input interface{}) (string, error) {
	// 解析处理器 (可以是脚本路径或命令)
	parts := strings.SplitN(handler, " ", 2)
	command := parts[0]
	args := ""
	if len(parts) > 1 {
		args = parts[1]
	}

	// 准备输入
	inputJSON, _ := json.Marshal(input)
	env := append(os.Environ(),
		fmt.Sprintf("PLUGIN_INPUT=%s", string(inputJSON)),
	)

	// 执行命令
	cmd := exec.Command(command)
	if args != "" {
		cmd.Args = append(cmd.Args, strings.Fields(args)...)
	}
	cmd.Env = env

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("handler failed: %s", string(exitErr.Stderr))
		}
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// GetPluginDir 获取插件目录
func (pm *PluginManager) GetPluginDir() string {
	return pm.pluginDir
}

// InstallPlugin 安装插件
func (pm *PluginManager) InstallPlugin(source string) error {
	// 简单实现：从本地目录复制
	// 实际应该支持 git clone、下载等
	if _, err := os.Stat(source); err == nil {
		pluginName := filepath.Base(source)
		targetPath := filepath.Join(pm.pluginDir, pluginName)

		cmd := exec.Command("cp", "-r", source, targetPath)
		return cmd.Run()
	}

	return fmt.Errorf("unsupported source: %s", source)
}

// UninstallPlugin 卸载插件
func (pm *PluginManager) UninstallPlugin(name string) error {
	if _, ok := pm.plugins[name]; !ok {
		return fmt.Errorf("plugin not found: %s", name)
	}

	pluginPath := filepath.Join(pm.pluginDir, name)
	return os.RemoveAll(pluginPath)
}

// GetToolCount 获取工具总数
func (pm *PluginManager) GetToolCount() int {
	count := 0
	for _, p := range pm.plugins {
		if pm.enabled[p.Name] {
			count += len(p.Tools)
		}
	}
	return count
}

// GetCommandCount 获取命令总数
func (pm *PluginManager) GetCommandCount() int {
	count := 0
	for _, p := range pm.plugins {
		if pm.enabled[p.Name] {
			count += len(p.Commands)
		}
	}
	return count
}

package toolkit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// AskUserQuestionTool 用户提问工具
type AskUserQuestionTool struct{}

// NewAskUserQuestionTool 创建工具
func NewAskUserQuestionTool() *AskUserQuestionTool {
	return &AskUserQuestionTool{}
}

// AskUserInput 输入
type AskUserInput struct {
	Question      string   `json:"question"`
	AllowMultiple bool     `json:"allowMultiple,omitempty"`
	Suggestions   []string `json:"suggestions,omitempty"`
	DefaultValue  string   `json:"default,omitempty"`
}

// Execute 执行
func (t *AskUserQuestionTool) Execute(input json.RawMessage) (string, error) {
	var inp AskUserInput
	if err := json.Unmarshal(input, &inp); err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	if inp.Question == "" {
		return "", fmt.Errorf("question is required")
	}

	fmt.Printf("\n❓ %s\n", inp.Question)

	// 显示建议
	if len(inp.Suggestions) > 0 {
		fmt.Println("\nSuggestions:")
		for i, sug := range inp.Suggestions {
			fmt.Printf("  %d. %s\n", i+1, sug)
		}
	}

	// 默认值
	if inp.DefaultValue != "" {
		fmt.Printf("\n[Default: %s]\n", inp.DefaultValue)
	}

	// 读取输入
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("> ")
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(answer)

	// 使用默认值
	if answer == "" && inp.DefaultValue != "" {
		answer = inp.DefaultValue
	}

	if answer == "" {
		return "No answer provided", nil
	}

	return fmt.Sprintf("User answered: %s", answer), nil
}

// GetDescription 获取描述
func (t *AskUserQuestionTool) GetDescription() string {
	return "Ask the user a question and wait for their response"
}

// GetInputSchema 获取输入 schema
func (t *AskUserQuestionTool) GetInputSchema() string {
	return `{
		"type":"object",
		"properties":{
			"question":{"type":"string"},
			"allowMultiple":{"type":"boolean"},
			"suggestions":{"type":"array","items":{"type":"string"}},
			"default":{"type":"string"}
		},
		"required":["question"]
	}`
}

// ConfigTool 配置工具
type ConfigTool struct {
	config map[string]interface{}
}

// NewConfigTool 创建配置工具
func NewConfigTool() *ConfigTool {
	return &ConfigTool{
		config: make(map[string]interface{}),
	}
}

// ConfigInput 配置输入
type ConfigInput struct {
	Action string      `json:"action"` // get, set, delete, list
	Key    string      `json:"key,omitempty"`
	Value  interface{} `json:"value,omitempty"`
}

// Execute 执行
func (t *ConfigTool) Execute(input json.RawMessage) (string, error) {
	var inp ConfigInput
	if err := json.Unmarshal(input, &inp); err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	switch inp.Action {
	case "get":
		if inp.Key == "" {
			return "", fmt.Errorf("key is required")
		}
		if val, ok := t.config[inp.Key]; ok {
			return fmt.Sprintf("%s = %v", inp.Key, val), nil
		}
		return fmt.Sprintf("Key not found: %s", inp.Key), nil

	case "set":
		if inp.Key == "" {
			return "", fmt.Errorf("key is required")
		}
		t.config[inp.Key] = inp.Value
		return fmt.Sprintf("Set %s = %v", inp.Key, inp.Value), nil

	case "delete":
		if inp.Key == "" {
			return "", fmt.Errorf("key is required")
		}
		if _, ok := t.config[inp.Key]; ok {
			delete(t.config, inp.Key)
			return fmt.Sprintf("Deleted key: %s", inp.Key), nil
		}
		return fmt.Sprintf("Key not found: %s", inp.Key), nil

	case "list":
		if len(t.config) == 0 {
			return "No configuration set", nil
		}
		var result strings.Builder
		for k, v := range t.config {
			result.WriteString(fmt.Sprintf("%s = %v\n", k, v))
		}
		return result.String(), nil

	default:
		return "", fmt.Errorf("unknown action: %s", inp.Action)
	}
}

// GetDescription 获取描述
func (t *ConfigTool) GetDescription() string {
	return "Get, set, or delete configuration values"
}

// GetInputSchema 获取输入 schema
func (t *ConfigTool) GetInputSchema() string {
	return `{
		"type":"object",
		"properties":{
			"action":{"type":"string","enum":["get","set","delete","list"]},
			"key":{"type":"string"},
			"value":{}
		},
		"required":["action"]
	}`
}

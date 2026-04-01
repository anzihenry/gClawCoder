package toolkit

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// RemoteTriggerTool 远程触发器工具
type RemoteTriggerTool struct {
	triggers map[string]*RemoteTrigger
}

// RemoteTrigger 远程触发器
type RemoteTrigger struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	URL           string            `json:"url"`
	Method        string            `json:"method"`
	Headers       map[string]string `json:"headers"`
	Body          string            `json:"body"`
	LastTriggered time.Time         `json:"lastTriggered"`
	Enabled       bool              `json:"enabled"`
}

// NewRemoteTriggerTool 创建工具
func NewRemoteTriggerTool() *RemoteTriggerTool {
	return &RemoteTriggerTool{
		triggers: make(map[string]*RemoteTrigger),
	}
}

// RemoteTriggerInput 输入
type RemoteTriggerInput struct {
	Action  string            `json:"action"` // create, trigger, list, delete, enable, disable
	ID      string            `json:"id,omitempty"`
	Name    string            `json:"name,omitempty"`
	URL     string            `json:"url,omitempty"`
	Method  string            `json:"method,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

// Execute 执行
func (t *RemoteTriggerTool) Execute(input json.RawMessage) (string, error) {
	var inp RemoteTriggerInput
	if err := json.Unmarshal(input, &inp); err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	switch inp.Action {
	case "create":
		return t.create(inp)
	case "trigger":
		return t.trigger(inp.ID)
	case "list":
		return t.list()
	case "delete":
		return t.delete(inp.ID)
	case "enable":
		return t.enable(inp.ID)
	case "disable":
		return t.disable(inp.ID)
	default:
		return "", fmt.Errorf("unknown action: %s", inp.Action)
	}
}

func (t *RemoteTriggerTool) create(inp RemoteTriggerInput) (string, error) {
	if inp.Name == "" || inp.URL == "" {
		return "", fmt.Errorf("name and url are required")
	}

	id := inp.ID
	if id == "" {
		id = fmt.Sprintf("trigger-%d", len(t.triggers)+1)
	}

	method := inp.Method
	if method == "" {
		method = "POST"
	}

	trigger := &RemoteTrigger{
		ID:      id,
		Name:    inp.Name,
		URL:     inp.URL,
		Method:  method,
		Headers: inp.Headers,
		Body:    inp.Body,
		Enabled: true,
	}

	t.triggers[id] = trigger

	return fmt.Sprintf("Created remote trigger: %s (%s %s)", trigger.Name, trigger.Method, trigger.URL), nil
}

func (t *RemoteTriggerTool) trigger(id string) (string, error) {
	trigger, ok := t.triggers[id]
	if !ok {
		return "", fmt.Errorf("trigger not found: %s", id)
	}

	if !trigger.Enabled {
		return "", fmt.Errorf("trigger is disabled: %s", id)
	}

	// 模拟触发 (实际应该发送 HTTP 请求)
	trigger.LastTriggered = time.Now()

	return fmt.Sprintf("Triggered: %s\n  Method: %s\n  URL: %s\n  (Simulated - no actual HTTP request sent)",
		trigger.Name, trigger.Method, trigger.URL), nil
}

func (t *RemoteTriggerTool) list() (string, error) {
	if len(t.triggers) == 0 {
		return "No remote triggers configured", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Remote Triggers (%d):\n", len(t.triggers)))

	for id, trigger := range t.triggers {
		status := "⚪ disabled"
		if trigger.Enabled {
			status = "🟢 enabled"
		}

		lastTriggered := "never"
		if !trigger.LastTriggered.IsZero() {
			lastTriggered = trigger.LastTriggered.Format("2006-01-02 15:04:05")
		}

		sb.WriteString(fmt.Sprintf("  %s. %s - %s [%s]\n", id, trigger.Name, status, trigger.URL))
		sb.WriteString(fmt.Sprintf("     Last triggered: %s\n", lastTriggered))
	}

	return sb.String(), nil
}

func (t *RemoteTriggerTool) delete(id string) (string, error) {
	if _, ok := t.triggers[id]; !ok {
		return "", fmt.Errorf("trigger not found: %s", id)
	}

	delete(t.triggers, id)
	return fmt.Sprintf("Deleted trigger: %s", id), nil
}

func (t *RemoteTriggerTool) enable(id string) (string, error) {
	trigger, ok := t.triggers[id]
	if !ok {
		return "", fmt.Errorf("trigger not found: %s", id)
	}

	trigger.Enabled = true
	return fmt.Sprintf("Enabled trigger: %s", trigger.Name), nil
}

func (t *RemoteTriggerTool) disable(id string) (string, error) {
	trigger, ok := t.triggers[id]
	if !ok {
		return "", fmt.Errorf("trigger not found: %s", id)
	}

	trigger.Enabled = false
	return fmt.Sprintf("Disabled trigger: %s", trigger.Name), nil
}

// GetDescription 获取描述
func (t *RemoteTriggerTool) GetDescription() string {
	return "Create and trigger remote webhooks/HTTP endpoints"
}

// GetInputSchema 获取输入 schema
func (t *RemoteTriggerTool) GetInputSchema() string {
	return `{
		"type":"object",
		"properties":{
			"action":{"type":"string","enum":["create","trigger","list","delete","enable","disable"]},
			"id":{"type":"string"},
			"name":{"type":"string"},
			"url":{"type":"string"},
			"method":{"type":"string"},
			"headers":{"type":"object"},
			"body":{"type":"string"}
		},
		"required":["action"]
	}`
}

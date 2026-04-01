package toolkit

import (
	"encoding/json"
	"fmt"
	"strings"
)

// AgentTool Agent 工具
type AgentTool struct {
	agents map[string]*Agent
}

// Agent Agent 定义
type Agent struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Prompt      string            `json:"prompt"`
	Model       string            `json:"model"`
	Status      string            `json:"status"`
	Metadata    map[string]string `json:"metadata"`
}

// NewAgentTool 创建 Agent 工具
func NewAgentTool() *AgentTool {
	return &AgentTool{
		agents: make(map[string]*Agent),
	}
}

// AgentInput Agent 输入
type AgentInput struct {
	Action      string            `json:"action"` // create, list, get, update, delete, run
	AgentID     string            `json:"agentId,omitempty"`
	Name        string            `json:"name,omitempty"`
	Description string            `json:"description,omitempty"`
	Prompt      string            `json:"prompt,omitempty"`
	Model       string            `json:"model,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Execute 执行 Agent 操作
func (t *AgentTool) Execute(input json.RawMessage) (string, error) {
	var inp AgentInput
	if err := json.Unmarshal(input, &inp); err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	switch inp.Action {
	case "create":
		return t.createAgent(inp)
	case "list":
		return t.listAgents()
	case "get":
		return t.getAgent(inp.AgentID)
	case "update":
		return t.updateAgent(inp)
	case "delete":
		return t.deleteAgent(inp.AgentID)
	case "run":
		return t.runAgent(inp.AgentID)
	default:
		return "", fmt.Errorf("unknown action: %s", inp.Action)
	}
}

func (t *AgentTool) createAgent(inp AgentInput) (string, error) {
	if inp.Name == "" {
		return "", fmt.Errorf("name is required")
	}

	id := fmt.Sprintf("agent-%d", len(t.agents)+1)
	agent := &Agent{
		ID:          id,
		Name:        inp.Name,
		Description: inp.Description,
		Prompt:      inp.Prompt,
		Model:       inp.Model,
		Status:      "idle",
		Metadata:    inp.Metadata,
	}

	t.agents[id] = agent

	return fmt.Sprintf("Agent created: %s (%s)", agent.Name, agent.ID), nil
}

func (t *AgentTool) listAgents() (string, error) {
	if len(t.agents) == 0 {
		return "No agents created yet", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Agents (%d):\n", len(t.agents)))

	for id, agent := range t.agents {
		status := "⚪"
		if agent.Status == "running" {
			status = "🟢"
		}
		sb.WriteString(fmt.Sprintf("  %s %s - %s\n", status, agent.Name, id))
		if agent.Description != "" {
			sb.WriteString(fmt.Sprintf("     %s\n", agent.Description))
		}
	}

	return sb.String(), nil
}

func (t *AgentTool) getAgent(id string) (string, error) {
	agent, ok := t.agents[id]
	if !ok {
		return "", fmt.Errorf("agent not found: %s", id)
	}

	return fmt.Sprintf("Agent: %s\nID: %s\nStatus: %s\nModel: %s\nPrompt: %s",
		agent.Name, agent.ID, agent.Status, agent.Model, agent.Prompt), nil
}

func (t *AgentTool) updateAgent(inp AgentInput) (string, error) {
	agent, ok := t.agents[inp.AgentID]
	if !ok {
		return "", fmt.Errorf("agent not found: %s", inp.AgentID)
	}

	if inp.Name != "" {
		agent.Name = inp.Name
	}
	if inp.Description != "" {
		agent.Description = inp.Description
	}
	if inp.Prompt != "" {
		agent.Prompt = inp.Prompt
	}
	if inp.Model != "" {
		agent.Model = inp.Model
	}
	if inp.Metadata != nil {
		agent.Metadata = inp.Metadata
	}

	return fmt.Sprintf("Agent updated: %s", agent.Name), nil
}

func (t *AgentTool) deleteAgent(id string) (string, error) {
	if _, ok := t.agents[id]; !ok {
		return "", fmt.Errorf("agent not found: %s", id)
	}

	delete(t.agents, id)
	return fmt.Sprintf("Agent deleted: %s", id), nil
}

func (t *AgentTool) runAgent(id string) (string, error) {
	agent, ok := t.agents[id]
	if !ok {
		return "", fmt.Errorf("agent not found: %s", id)
	}

	agent.Status = "running"
	return fmt.Sprintf("Agent %s started with prompt: %s", agent.Name, agent.Prompt), nil
}

// GetDescription 获取描述
func (t *AgentTool) GetDescription() string {
	return "Create and manage AI agents for specialized tasks"
}

// GetInputSchema 获取输入 schema
func (t *AgentTool) GetInputSchema() string {
	return `{
		"type":"object",
		"properties":{
			"action":{"type":"string","enum":["create","list","get","update","delete","run"]},
			"agentId":{"type":"string"},
			"name":{"type":"string"},
			"description":{"type":"string"},
			"prompt":{"type":"string"},
			"model":{"type":"string"},
			"metadata":{"type":"object"}
		},
		"required":["action"]
	}`
}

// TaskTool 任务管理工具
type TaskTool struct {
	tasks map[string]*Task
}

// Task 任务定义
type Task struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      string            `json:"status"`   // pending, in_progress, completed, cancelled
	Priority    string            `json:"priority"` // low, medium, high, urgent
	Assignee    string            `json:"assignee"`
	DueDate     string            `json:"dueDate"`
	Tags        []string          `json:"tags"`
	Metadata    map[string]string `json:"metadata"`
}

// NewTaskTool 创建任务工具
func NewTaskTool() *TaskTool {
	return &TaskTool{
		tasks: make(map[string]*Task),
	}
}

// TaskInput 任务输入
type TaskInput struct {
	Action      string   `json:"action"` // create, list, get, update, delete
	TaskID      string   `json:"taskId,omitempty"`
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	Status      string   `json:"status,omitempty"`
	Priority    string   `json:"priority,omitempty"`
	Assignee    string   `json:"assignee,omitempty"`
	DueDate     string   `json:"dueDate,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// Execute 执行任务操作
func (t *TaskTool) Execute(input json.RawMessage) (string, error) {
	var inp TaskInput
	if err := json.Unmarshal(input, &inp); err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	switch inp.Action {
	case "create":
		return t.createTask(inp)
	case "list":
		return t.listTasks(inp)
	case "get":
		return t.getTask(inp.TaskID)
	case "update":
		return t.updateTask(inp)
	case "delete":
		return t.deleteTask(inp.TaskID)
	default:
		return "", fmt.Errorf("unknown action: %s", inp.Action)
	}
}

func (t *TaskTool) createTask(inp TaskInput) (string, error) {
	if inp.Title == "" {
		return "", fmt.Errorf("title is required")
	}

	id := fmt.Sprintf("task-%d", len(t.tasks)+1)
	task := &Task{
		ID:          id,
		Title:       inp.Title,
		Description: inp.Description,
		Status:      "pending",
		Priority:    inp.Priority,
		Assignee:    inp.Assignee,
		DueDate:     inp.DueDate,
		Tags:        inp.Tags,
		Metadata:    make(map[string]string),
	}

	t.tasks[id] = task

	return fmt.Sprintf("Task created: %s (%s)", task.Title, task.ID), nil
}

func (t *TaskTool) listTasks(inp TaskInput) (string, error) {
	if len(t.tasks) == 0 {
		return "No tasks created yet", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Tasks (%d):\n", len(t.tasks)))

	for _, task := range t.tasks {
		// 过滤
		if inp.Status != "" && task.Status != inp.Status {
			continue
		}

		icon := "⚪"
		switch task.Status {
		case "in_progress":
			icon = "🟡"
		case "completed":
			icon = "🟢"
		case "cancelled":
			icon = "🔴"
		}

		priority := ""
		if task.Priority != "" {
			priority = fmt.Sprintf("[%s] ", task.Priority)
		}

		sb.WriteString(fmt.Sprintf("  %s %s%s - %s\n", icon, priority, task.Title, task.ID))
	}

	return sb.String(), nil
}

func (t *TaskTool) getTask(id string) (string, error) {
	task, ok := t.tasks[id]
	if !ok {
		return "", fmt.Errorf("task not found: %s", id)
	}

	return fmt.Sprintf("Task: %s\nID: %s\nStatus: %s\nPriority: %s\nAssignee: %s\nDue: %s",
		task.Title, task.ID, task.Status, task.Priority, task.Assignee, task.DueDate), nil
}

func (t *TaskTool) updateTask(inp TaskInput) (string, error) {
	task, ok := t.tasks[inp.TaskID]
	if !ok {
		return "", fmt.Errorf("task not found: %s", inp.TaskID)
	}

	if inp.Title != "" {
		task.Title = inp.Title
	}
	if inp.Description != "" {
		task.Description = inp.Description
	}
	if inp.Status != "" {
		task.Status = inp.Status
	}
	if inp.Priority != "" {
		task.Priority = inp.Priority
	}
	if inp.Assignee != "" {
		task.Assignee = inp.Assignee
	}
	if inp.DueDate != "" {
		task.DueDate = inp.DueDate
	}
	if inp.Tags != nil {
		task.Tags = inp.Tags
	}

	return fmt.Sprintf("Task updated: %s", task.Title), nil
}

func (t *TaskTool) deleteTask(id string) (string, error) {
	if _, ok := t.tasks[id]; !ok {
		return "", fmt.Errorf("task not found: %s", id)
	}

	delete(t.tasks, id)
	return fmt.Sprintf("Task deleted: %s", id), nil
}

// GetDescription 获取描述
func (t *TaskTool) GetDescription() string {
	return "Create and manage tasks with status tracking"
}

// GetInputSchema 获取输入 schema
func (t *TaskTool) GetInputSchema() string {
	return `{
		"type":"object",
		"properties":{
			"action":{"type":"string","enum":["create","list","get","update","delete"]},
			"taskId":{"type":"string"},
			"title":{"type":"string"},
			"description":{"type":"string"},
			"status":{"type":"string","enum":["pending","in_progress","completed","cancelled"]},
			"priority":{"type":"string","enum":["low","medium","high","urgent"]},
			"assignee":{"type":"string"},
			"dueDate":{"type":"string"},
			"tags":{"type":"array","items":{"type":"string"}}
		},
		"required":["action"]
	}`
}

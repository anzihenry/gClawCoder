package toolkit

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ScheduleCronTool 定时任务工具
type ScheduleCronTool struct {
	jobs map[string]*CronJob
}

// CronJob 定时任务
type CronJob struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Schedule string    `json:"schedule"` // cron expression
	Command  string    `json:"command"`
	Enabled  bool      `json:"enabled"`
	LastRun  time.Time `json:"lastRun"`
	NextRun  time.Time `json:"nextRun"`
	RunCount int       `json:"runCount"`
}

// NewScheduleCronTool 创建工具
func NewScheduleCronTool() *ScheduleCronTool {
	return &ScheduleCronTool{
		jobs: make(map[string]*CronJob),
	}
}

// ScheduleCronInput 输入
type ScheduleCronInput struct {
	Action   string `json:"action"` // create, remove, list, enable, disable, run, status
	ID       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	Schedule string `json:"schedule,omitempty"` // cron: */5 * * * *
	Command  string `json:"command,omitempty"`
}

// Execute 执行
func (t *ScheduleCronTool) Execute(input json.RawMessage) (string, error) {
	var inp ScheduleCronInput
	if err := json.Unmarshal(input, &inp); err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	switch inp.Action {
	case "create":
		return t.create(inp)
	case "remove":
		return t.remove(inp.ID)
	case "list":
		return t.list()
	case "enable":
		return t.enable(inp.ID)
	case "disable":
		return t.disable(inp.ID)
	case "run":
		return t.run(inp.ID)
	case "status":
		return t.status(inp.ID)
	default:
		return "", fmt.Errorf("unknown action: %s", inp.Action)
	}
}

func (t *ScheduleCronTool) create(inp ScheduleCronInput) (string, error) {
	if inp.Name == "" || inp.Schedule == "" || inp.Command == "" {
		return "", fmt.Errorf("name, schedule, and command are required")
	}

	id := inp.ID
	if id == "" {
		id = fmt.Sprintf("job-%d", len(t.jobs)+1)
	}

	// 解析 cron 表达式 (简化实现)
	nextRun, err := t.parseSchedule(inp.Schedule)
	if err != nil {
		return "", fmt.Errorf("invalid schedule: %w", err)
	}

	job := &CronJob{
		ID:       id,
		Name:     inp.Name,
		Schedule: inp.Schedule,
		Command:  inp.Command,
		Enabled:  true,
		NextRun:  nextRun,
	}

	t.jobs[id] = job

	return fmt.Sprintf("Created scheduled job: %s\n  Schedule: %s\n  Command: %s\n  Next run: %s",
		job.Name, job.Schedule, job.Command, job.NextRun.Format("2006-01-02 15:04:05")), nil
}

func (t *ScheduleCronTool) remove(id string) (string, error) {
	if _, ok := t.jobs[id]; !ok {
		return "", fmt.Errorf("job not found: %s", id)
	}

	delete(t.jobs, id)
	return fmt.Sprintf("Removed job: %s", id), nil
}

func (t *ScheduleCronTool) list() (string, error) {
	if len(t.jobs) == 0 {
		return "No scheduled jobs", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Scheduled Jobs (%d):\n", len(t.jobs)))

	for id, job := range t.jobs {
		status := "⚪ disabled"
		if job.Enabled {
			status = "🟢 enabled"
		}

		nextRun := "not scheduled"
		if !job.NextRun.IsZero() {
			nextRun = job.NextRun.Format("2006-01-02 15:04:05")
		}

		sb.WriteString(fmt.Sprintf("  %s. %s - %s [%s]\n", id, job.Name, status, job.Schedule))
		sb.WriteString(fmt.Sprintf("     Command: %s\n", job.Command))
		sb.WriteString(fmt.Sprintf("     Next run: %s, Runs: %d\n", nextRun, job.RunCount))
	}

	return sb.String(), nil
}

func (t *ScheduleCronTool) enable(id string) (string, error) {
	job, ok := t.jobs[id]
	if !ok {
		return "", fmt.Errorf("job not found: %s", id)
	}

	job.Enabled = true
	return fmt.Sprintf("Enabled job: %s", job.Name), nil
}

func (t *ScheduleCronTool) disable(id string) (string, error) {
	job, ok := t.jobs[id]
	if !ok {
		return "", fmt.Errorf("job not found: %s", id)
	}

	job.Enabled = false
	return fmt.Sprintf("Disabled job: %s", job.Name), nil
}

func (t *ScheduleCronTool) run(id string) (string, error) {
	job, ok := t.jobs[id]
	if !ok {
		return "", fmt.Errorf("job not found: %s", id)
	}

	// 模拟执行
	job.LastRun = time.Now()
	job.RunCount++

	return fmt.Sprintf("Executed job: %s\n  Command: %s\n  (Simulated - no actual command executed)",
		job.Name, job.Command), nil
}

func (t *ScheduleCronTool) status(id string) (string, error) {
	job, ok := t.jobs[id]
	if !ok {
		return "", fmt.Errorf("job not found: %s", id)
	}

	status := "disabled"
	if job.Enabled {
		status = "enabled"
	}

	lastRun := "never"
	if !job.LastRun.IsZero() {
		lastRun = job.LastRun.Format("2006-01-02 15:04:05")
	}

	nextRun := "not scheduled"
	if !job.NextRun.IsZero() {
		nextRun = job.NextRun.Format("2006-01-02 15:04:05")
	}

	return fmt.Sprintf("Job: %s\n  ID: %s\n  Status: %s\n  Schedule: %s\n  Command: %s\n  Last run: %s\n  Next run: %s\n  Run count: %d",
		job.Name, job.ID, status, job.Schedule, job.Command, lastRun, nextRun, job.RunCount), nil
}

// parseSchedule 解析 cron 表达式 (简化)
func (t *ScheduleCronTool) parseSchedule(schedule string) (time.Time, error) {
	// 简化实现：返回 5 分钟后
	return time.Now().Add(5 * time.Minute), nil
}

// GetDescription 获取描述
func (t *ScheduleCronTool) GetDescription() string {
	return "Schedule and manage recurring tasks with cron expressions"
}

// GetInputSchema 获取输入 schema
func (t *ScheduleCronTool) GetInputSchema() string {
	return `{
		"type":"object",
		"properties":{
			"action":{"type":"string","enum":["create","remove","list","enable","disable","run","status"]},
			"id":{"type":"string"},
			"name":{"type":"string"},
			"schedule":{"type":"string"},
			"command":{"type":"string"}
		},
		"required":["action"]
	}`
}

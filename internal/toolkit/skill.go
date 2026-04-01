package toolkit

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gclawcoder/gclaw/internal/skills"
)

// SkillTool Skills 工具
type SkillTool struct {
	manager *skills.SkillManager
}

// NewSkillTool 创建 Skill 工具
func NewSkillTool(skillsDir string) *SkillTool {
	manager := skills.NewSkillManager(skillsDir)
	manager.LoadSkills()
	return &SkillTool{
		manager: manager,
	}
}

// SkillToolInput Skill 工具输入
type SkillToolInput struct {
	Action    string            `json:"action"` // list, get, search, execute, enable, disable, add, delete
	SkillID   string            `json:"skillId,omitempty"`
	Name      string            `json:"name,omitempty"`
	Category  string            `json:"category,omitempty"`
	Query     string            `json:"query,omitempty"`
	Prompt    string            `json:"prompt,omitempty"`
	Variables map[string]string `json:"variables,omitempty"`
}

// Execute 执行 Skill 操作
func (t *SkillTool) Execute(input json.RawMessage) (string, error) {
	var inp SkillToolInput
	if err := json.Unmarshal(input, &inp); err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	switch inp.Action {
	case "list":
		return t.list(inp.Category)
	case "get":
		return t.get(inp.SkillID)
	case "search":
		return t.search(inp.Query)
	case "execute":
		return t.execute(inp.SkillID, inp.Variables)
	case "enable":
		return t.enable(inp.SkillID)
	case "disable":
		return t.disable(inp.SkillID)
	case "add":
		return t.add(inp)
	case "delete":
		return t.delete(inp.SkillID)
	default:
		return "", fmt.Errorf("unknown action: %s", inp.Action)
	}
}

// list 列出技能
func (t *SkillTool) list(category string) (string, error) {
	var skillsList []*skills.SkillInfo

	if category != "" {
		skillsList = t.manager.GetSkillsByCategory(category)
	} else {
		skillsList = t.manager.ListSkills()
	}

	if len(skillsList) == 0 {
		if category != "" {
			return fmt.Sprintf("No skills in category: %s", category), nil
		}
		return "No skills available", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Skills (%d):\n", len(skillsList)))

	for _, skill := range skillsList {
		status := "🟢"
		if !skill.Enabled {
			status = "⚪"
		}

		sb.WriteString(fmt.Sprintf("  %s %s [%s]\n", status, skill.Name, skill.Category))
		if skill.Description != "" {
			sb.WriteString(fmt.Sprintf("     %s\n", skill.Description))
		}
	}

	return sb.String(), nil
}

// get 获取技能详情
func (t *SkillTool) get(skillID string) (string, error) {
	if skillID == "" {
		return "", fmt.Errorf("skillId is required")
	}

	skill := t.manager.GetSkill(skillID)
	if skill == nil {
		return "", fmt.Errorf("skill not found: %s", skillID)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Skill: %s\n", skill.Name))
	sb.WriteString(fmt.Sprintf("ID: %s\n", skill.ID))
	sb.WriteString(fmt.Sprintf("Category: %s\n", skill.Category))
	sb.WriteString(fmt.Sprintf("Status: %v\n", skill.Enabled))
	sb.WriteString(fmt.Sprintf("Description: %s\n", skill.Description))

	if len(skill.Triggers) > 0 {
		sb.WriteString(fmt.Sprintf("Triggers: %s\n", strings.Join(skill.Triggers, ", ")))
	}

	if len(skill.Variables) > 0 {
		sb.WriteString("Variables:\n")
		for k, v := range skill.Variables {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
		}
	}

	return sb.String(), nil
}

// search 搜索技能
func (t *SkillTool) search(query string) (string, error) {
	if query == "" {
		return "", fmt.Errorf("query is required")
	}

	results := t.manager.SearchSkills(query)
	if len(results) == 0 {
		return fmt.Sprintf("No skills found matching: %s", query), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Skills matching '%s' (%d):\n", query, len(results)))

	for _, skill := range results {
		status := "🟢"
		if !skill.Enabled {
			status = "⚪"
		}

		sb.WriteString(fmt.Sprintf("  %s %s [%s]\n", status, skill.Name, skill.Category))
	}

	return sb.String(), nil
}

// execute 执行技能
func (t *SkillTool) execute(skillID string, variables map[string]string) (string, error) {
	if skillID == "" {
		return "", fmt.Errorf("skillId is required")
	}

	prompt, err := t.manager.ExecuteSkill(skillID, variables)
	if err != nil {
		return "", fmt.Errorf("failed to execute skill: %w", err)
	}

	return fmt.Sprintf("Skill %s executed:\n\n%s", skillID, prompt), nil
}

// enable 启用技能
func (t *SkillTool) enable(skillID string) (string, error) {
	if skillID == "" {
		return "", fmt.Errorf("skillId is required")
	}

	if err := t.manager.EnableSkill(skillID); err != nil {
		return "", fmt.Errorf("failed to enable skill: %w", err)
	}

	return fmt.Sprintf("Enabled skill: %s", skillID), nil
}

// disable 禁用技能
func (t *SkillTool) disable(skillID string) (string, error) {
	if skillID == "" {
		return "", fmt.Errorf("skillId is required")
	}

	if err := t.manager.DisableSkill(skillID); err != nil {
		return "", fmt.Errorf("failed to disable skill: %w", err)
	}

	return fmt.Sprintf("Disabled skill: %s", skillID), nil
}

// add 添加技能
func (t *SkillTool) add(inp SkillToolInput) (string, error) {
	if inp.Name == "" || inp.Prompt == "" {
		return "", fmt.Errorf("name and prompt are required")
	}

	skill := &skills.SkillInfo{
		ID:        inp.SkillID,
		Name:      inp.Name,
		Category:  inp.Category,
		Prompt:    inp.Prompt,
		Enabled:   true,
		Variables: inp.Variables,
	}

	if err := t.manager.AddSkill(skill); err != nil {
		return "", fmt.Errorf("failed to add skill: %w", err)
	}

	return fmt.Sprintf("Added skill: %s (%s)", skill.Name, skill.ID), nil
}

// delete 删除技能
func (t *SkillTool) delete(skillID string) (string, error) {
	if skillID == "" {
		return "", fmt.Errorf("skillId is required")
	}

	if err := t.manager.DeleteSkill(skillID); err != nil {
		return "", fmt.Errorf("failed to delete skill: %w", err)
	}

	return fmt.Sprintf("Deleted skill: %s", skillID), nil
}

// GetDescription 获取描述
func (t *SkillTool) GetDescription() string {
	return "Manage and execute skills (predefined prompt templates)"
}

// GetInputSchema 获取输入 schema
func (t *SkillTool) GetInputSchema() string {
	return `{
		"type":"object",
		"properties":{
			"action":{"type":"string","enum":["list","get","search","execute","enable","disable","add","delete"]},
			"skillId":{"type":"string"},
			"name":{"type":"string"},
			"category":{"type":"string"},
			"query":{"type":"string"},
			"prompt":{"type":"string"},
			"variables":{"type":"object"}
		},
		"required":["action"]
	}`
}

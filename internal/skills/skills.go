package skills

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SkillInfo 技能信息
type SkillInfo struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Prompt      string            `json:"prompt"`
	Category    string            `json:"category"`
	Triggers    []string          `json:"triggers"`
	Variables   map[string]string `json:"variables"`
	Enabled     bool              `json:"enabled"`
}

// SkillManager 技能管理器
type SkillManager struct {
	skills     map[string]*SkillInfo
	skillsDir  string
	categories map[string][]string
}

// NewSkillManager 创建技能管理器
func NewSkillManager(skillsDir string) *SkillManager {
	return &SkillManager{
		skills:     make(map[string]*SkillInfo),
		skillsDir:  skillsDir,
		categories: make(map[string][]string),
	}
}

// LoadSkills 加载所有技能
func (sm *SkillManager) LoadSkills() error {
	if sm.skillsDir == "" {
		return nil
	}

	// 从目录加载
	entries, err := os.ReadDir(sm.skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			skillPath := filepath.Join(sm.skillsDir, entry.Name())
			if err := sm.loadSkill(skillPath); err != nil {
				fmt.Printf("Warning: failed to load skill %s: %v\n", entry.Name(), err)
			}
		} else if strings.HasSuffix(entry.Name(), ".json") {
			skillPath := filepath.Join(sm.skillsDir, entry.Name())
			if err := sm.loadSkillFile(skillPath); err != nil {
				fmt.Printf("Warning: failed to load skill %s: %v\n", entry.Name(), err)
			}
		}
	}

	return nil
}

func (sm *SkillManager) loadSkill(skillPath string) error {
	manifestPath := filepath.Join(skillPath, "skill.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}

	var skill SkillInfo
	if err := json.Unmarshal(data, &skill); err != nil {
		return err
	}

	sm.skills[skill.ID] = &skill
	sm.categories[skill.Category] = append(sm.categories[skill.Category], skill.ID)

	return nil
}

func (sm *SkillManager) loadSkillFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var skill SkillInfo
	if err := json.Unmarshal(data, &skill); err != nil {
		return err
	}

	sm.skills[skill.ID] = &skill
	sm.categories[skill.Category] = append(sm.categories[skill.Category], skill.ID)

	return nil
}

// ListSkills 列出所有技能
func (sm *SkillManager) ListSkills() []*SkillInfo {
	skills := make([]*SkillInfo, 0, len(sm.skills))
	for _, s := range sm.skills {
		if s.Enabled {
			skills = append(skills, s)
		}
	}
	return skills
}

// GetSkill 获取技能
func (sm *SkillManager) GetSkill(id string) *SkillInfo {
	return sm.skills[id]
}

// GetSkillsByCategory 按分类获取技能
func (sm *SkillManager) GetSkillsByCategory(category string) []*SkillInfo {
	ids := sm.categories[category]
	skills := make([]*SkillInfo, 0, len(ids))
	for _, id := range ids {
		if skill, ok := sm.skills[id]; ok && skill.Enabled {
			skills = append(skills, skill)
		}
	}
	return skills
}

// ListCategories 列出所有分类
func (sm *SkillManager) ListCategories() []string {
	categories := make([]string, 0, len(sm.categories))
	for cat := range sm.categories {
		categories = append(categories, cat)
	}
	return categories
}

// SearchSkills 搜索技能
func (sm *SkillManager) SearchSkills(query string) []*SkillInfo {
	query = strings.ToLower(query)
	var results []*SkillInfo

	for _, skill := range sm.skills {
		if !skill.Enabled {
			continue
		}

		if strings.Contains(strings.ToLower(skill.Name), query) ||
			strings.Contains(strings.ToLower(skill.Description), query) ||
			strings.Contains(strings.ToLower(skill.Category), query) {
			results = append(results, skill)
		}
	}

	return results
}

// ExecuteSkill 执行技能
func (sm *SkillManager) ExecuteSkill(id string, variables map[string]string) (string, error) {
	skill := sm.skills[id]
	if skill == nil {
		return "", fmt.Errorf("skill not found: %s", id)
	}

	if !skill.Enabled {
		return "", fmt.Errorf("skill is disabled: %s", id)
	}

	// 替换变量
	prompt := skill.Prompt
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		prompt = strings.ReplaceAll(prompt, placeholder, value)
	}

	// 替换内置变量
	for key, value := range skill.Variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		prompt = strings.ReplaceAll(prompt, placeholder, value)
	}

	return prompt, nil
}

// AddSkill 添加技能
func (sm *SkillManager) AddSkill(skill *SkillInfo) error {
	if skill.ID == "" {
		return fmt.Errorf("skill ID is required")
	}

	sm.skills[skill.ID] = skill
	sm.categories[skill.Category] = append(sm.categories[skill.Category], skill.ID)

	return sm.saveSkill(skill)
}

// UpdateSkill 更新技能
func (sm *SkillManager) UpdateSkill(skill *SkillInfo) error {
	if _, ok := sm.skills[skill.ID]; !ok {
		return fmt.Errorf("skill not found: %s", skill.ID)
	}

	sm.skills[skill.ID] = skill
	return sm.saveSkill(skill)
}

// DeleteSkill 删除技能
func (sm *SkillManager) DeleteSkill(id string) error {
	skill, ok := sm.skills[id]
	if !ok {
		return fmt.Errorf("skill not found: %s", id)
	}

	// 从分类中移除
	category := skill.Category
	for i, s := range sm.categories[category] {
		if s == id {
			sm.categories[category] = append(sm.categories[category][:i], sm.categories[category][i+1:]...)
			break
		}
	}

	delete(sm.skills, id)

	// 删除文件
	skillPath := filepath.Join(sm.skillsDir, id+".json")
	return os.Remove(skillPath)
}

func (sm *SkillManager) saveSkill(skill *SkillInfo) error {
	data, err := json.MarshalIndent(skill, "", "  ")
	if err != nil {
		return err
	}

	skillPath := filepath.Join(sm.skillsDir, skill.ID+".json")
	return os.WriteFile(skillPath, data, 0644)
}

// EnableSkill 启用技能
func (sm *SkillManager) EnableSkill(id string) error {
	skill, ok := sm.skills[id]
	if !ok {
		return fmt.Errorf("skill not found: %s", id)
	}
	skill.Enabled = true
	return sm.saveSkill(skill)
}

// DisableSkill 禁用技能
func (sm *SkillManager) DisableSkill(id string) error {
	skill, ok := sm.skills[id]
	if !ok {
		return fmt.Errorf("skill not found: %s", id)
	}
	skill.Enabled = false
	return sm.saveSkill(skill)
}

// GetSkillCount 获取技能数量
func (sm *SkillManager) GetSkillCount() int {
	count := 0
	for _, skill := range sm.skills {
		if skill.Enabled {
			count++
		}
	}
	return count
}

// ExportSkill 导出技能
func (sm *SkillManager) ExportSkill(id string) (string, error) {
	skill, ok := sm.skills[id]
	if !ok {
		return "", fmt.Errorf("skill not found: %s", id)
	}

	data, err := json.MarshalIndent(skill, "", "  ")
	return string(data), err
}

// ImportSkill 导入技能
func (sm *SkillManager) ImportSkill(data string) error {
	var skill SkillInfo
	if err := json.Unmarshal([]byte(data), &skill); err != nil {
		return err
	}

	return sm.AddSkill(&skill)
}

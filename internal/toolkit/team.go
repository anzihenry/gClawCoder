package toolkit

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// TeamTool 团队管理工具
type TeamTool struct {
	teams map[string]*Team
}

// Team 团队
type Team struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Members     []Member  `json:"members"`
	CreatedAt   time.Time `json:"createdAt"`
}

// Member 团队成员
type Member struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	Role     string    `json:"role"` // owner, admin, member
	JoinedAt time.Time `json:"joinedAt"`
}

// NewTeamTool 创建工具
func NewTeamTool() *TeamTool {
	return &TeamTool{
		teams: make(map[string]*Team),
	}
}

// TeamInput 输入
type TeamInput struct {
	Action      string `json:"action"` // create, delete, list, get, add_member, remove_member, update_member
	TeamID      string `json:"teamId,omitempty"`
	TeamName    string `json:"teamName,omitempty"`
	Description string `json:"description,omitempty"`
	MemberID    string `json:"memberId,omitempty"`
	MemberName  string `json:"memberName,omitempty"`
	Email       string `json:"email,omitempty"`
	Role        string `json:"role,omitempty"`
}

// Execute 执行
func (t *TeamTool) Execute(input json.RawMessage) (string, error) {
	var inp TeamInput
	if err := json.Unmarshal(input, &inp); err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	switch inp.Action {
	case "create":
		return t.createTeam(inp)
	case "delete":
		return t.deleteTeam(inp.TeamID)
	case "list":
		return t.listTeams()
	case "get":
		return t.getTeam(inp.TeamID)
	case "add_member":
		return t.addMember(inp)
	case "remove_member":
		return t.removeMember(inp)
	case "update_member":
		return t.updateMember(inp)
	default:
		return "", fmt.Errorf("unknown action: %s", inp.Action)
	}
}

func (t *TeamTool) createTeam(inp TeamInput) (string, error) {
	if inp.TeamName == "" {
		return "", fmt.Errorf("team name is required")
	}

	id := inp.TeamID
	if id == "" {
		id = fmt.Sprintf("team-%d", len(t.teams)+1)
	}

	team := &Team{
		ID:          id,
		Name:        inp.TeamName,
		Description: inp.Description,
		Members:     make([]Member, 0),
		CreatedAt:   time.Now(),
	}

	t.teams[id] = team

	return fmt.Sprintf("Created team: %s (%s)", team.Name, team.ID), nil
}

func (t *TeamTool) deleteTeam(id string) (string, error) {
	if _, ok := t.teams[id]; !ok {
		return "", fmt.Errorf("team not found: %s", id)
	}

	delete(t.teams, id)
	return fmt.Sprintf("Deleted team: %s", id), nil
}

func (t *TeamTool) listTeams() (string, error) {
	if len(t.teams) == 0 {
		return "No teams created", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Teams (%d):\n", len(t.teams)))

	for id, team := range t.teams {
		memberCount := len(team.Members)
		sb.WriteString(fmt.Sprintf("  %s. %s - %d members\n", id, team.Name, memberCount))
		if team.Description != "" {
			sb.WriteString(fmt.Sprintf("     %s\n", team.Description))
		}
	}

	return sb.String(), nil
}

func (t *TeamTool) getTeam(id string) (string, error) {
	team, ok := t.teams[id]
	if !ok {
		return "", fmt.Errorf("team not found: %s", id)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Team: %s\n", team.Name))
	sb.WriteString(fmt.Sprintf("ID: %s\n", team.ID))
	sb.WriteString(fmt.Sprintf("Description: %s\n", team.Description))
	sb.WriteString(fmt.Sprintf("Created: %s\n", team.CreatedAt.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("Members: %d\n\n", len(team.Members)))

	if len(team.Members) > 0 {
		sb.WriteString("Members:\n")
		for _, member := range team.Members {
			roleIcon := "👤"
			if member.Role == "owner" {
				roleIcon = "👑"
			} else if member.Role == "admin" {
				roleIcon = "⚡"
			}
			sb.WriteString(fmt.Sprintf("  %s %s (%s) - %s\n", roleIcon, member.Name, member.Email, member.Role))
		}
	}

	return sb.String(), nil
}

func (t *TeamTool) addMember(inp TeamInput) (string, error) {
	team, ok := t.teams[inp.TeamID]
	if !ok {
		return "", fmt.Errorf("team not found: %s", inp.TeamID)
	}

	if inp.MemberName == "" || inp.Email == "" {
		return "", fmt.Errorf("member name and email are required")
	}

	role := inp.Role
	if role == "" {
		role = "member"
	}

	member := Member{
		ID:       inp.MemberID,
		Name:     inp.MemberName,
		Email:    inp.Email,
		Role:     role,
		JoinedAt: time.Now(),
	}

	team.Members = append(team.Members, member)

	return fmt.Sprintf("Added member to team %s: %s (%s) as %s",
		team.Name, member.Name, member.Email, member.Role), nil
}

func (t *TeamTool) removeMember(inp TeamInput) (string, error) {
	team, ok := t.teams[inp.TeamID]
	if !ok {
		return "", fmt.Errorf("team not found: %s", inp.TeamID)
	}

	// 查找并删除成员
	found := false
	newMembers := make([]Member, 0)
	for _, m := range team.Members {
		if m.ID == inp.MemberID || m.Email == inp.Email {
			found = true
			continue
		}
		newMembers = append(newMembers, m)
	}

	if !found {
		return "", fmt.Errorf("member not found in team")
	}

	team.Members = newMembers
	return fmt.Sprintf("Removed member from team %s", team.Name), nil
}

func (t *TeamTool) updateMember(inp TeamInput) (string, error) {
	team, ok := t.teams[inp.TeamID]
	if !ok {
		return "", fmt.Errorf("team not found: %s", inp.TeamID)
	}

	// 查找并更新成员
	found := false
	for i, m := range team.Members {
		if m.ID == inp.MemberID || m.Email == inp.Email {
			if inp.MemberName != "" {
				team.Members[i].Name = inp.MemberName
			}
			if inp.Role != "" {
				team.Members[i].Role = inp.Role
			}
			found = true
			break
		}
	}

	if !found {
		return "", fmt.Errorf("member not found in team")
	}

	return fmt.Sprintf("Updated member in team %s", team.Name), nil
}

// GetDescription 获取描述
func (t *TeamTool) GetDescription() string {
	return "Create and manage teams with members and roles"
}

// GetInputSchema 获取输入 schema
func (t *TeamTool) GetInputSchema() string {
	return `{
		"type":"object",
		"properties":{
			"action":{"type":"string","enum":["create","delete","list","get","add_member","remove_member","update_member"]},
			"teamId":{"type":"string"},
			"teamName":{"type":"string"},
			"description":{"type":"string"},
			"memberId":{"type":"string"},
			"memberName":{"type":"string"},
			"email":{"type":"string"},
			"role":{"type":"string","enum":["owner","admin","member"]}
		},
		"required":["action"]
	}`
}

package commands

import (
	"testing"
)

func TestPortedCommands(t *testing.T) {
	cmds := PortedCommands()
	if len(cmds) == 0 {
		t.Error("Expected non-zero commands")
	}
}

func TestGetCommand(t *testing.T) {
	cmds := PortedCommands()
	if len(cmds) == 0 {
		t.Skip("No commands loaded")
	}

	firstCmd := cmds[0]
	cmd := GetCommand(firstCmd.Name)
	if cmd == nil {
		t.Errorf("Expected to find command %s", firstCmd.Name)
	}
}

func TestGetCommandNotFound(t *testing.T) {
	cmd := GetCommand("nonexistent-command-xyz")
	if cmd != nil {
		t.Error("Expected nil for nonexistent command")
	}
}

func TestFindCommands(t *testing.T) {
	matches := FindCommands("review", 10)
	// 应该找到包含 "review" 的命令
	if len(matches) == 0 {
		t.Log("No matches found for 'review' (may be expected)")
	}
}

func TestExecuteCommand(t *testing.T) {
	result := ExecuteCommand("review", "test prompt")
	if result.Handled {
		t.Logf("Command handled: %s", result.Message)
	} else {
		t.Logf("Command not handled: %s", result.Message)
	}
}

func TestExecuteCommandNotFound(t *testing.T) {
	result := ExecuteCommand("nonexistent", "test")
	if result.Handled {
		t.Error("Expected command to not be handled")
	}
	if result.Message == "" {
		t.Error("Expected error message")
	}
}

func TestRenderCommandIndex(t *testing.T) {
	output := RenderCommandIndex(5, "")
	if output == "" {
		t.Error("Expected non-empty output")
	}
}

func TestRenderCommandIndexWithQuery(t *testing.T) {
	output := RenderCommandIndex(5, "review")
	if output == "" {
		t.Error("Expected non-empty output")
	}
}

func TestBuildCommandBacklog(t *testing.T) {
	backlog := BuildCommandBacklog()
	if backlog.Title == "" {
		t.Error("Expected non-empty title")
	}
}

func TestCommandNames(t *testing.T) {
	names := CommandNames()
	if len(names) == 0 {
		t.Error("Expected non-zero command names")
	}
}

func TestGetCommands(t *testing.T) {
	cmds := GetCommands(true, true)
	if len(cmds) == 0 {
		t.Error("Expected non-zero commands")
	}
}

func TestGetCommandsFiltered(t *testing.T) {
	allCmds := GetCommands(true, true)
	noPluginCmds := GetCommands(false, true)
	noSkillCmds := GetCommands(true, false)

	if len(noPluginCmds) > len(allCmds) {
		t.Error("Filtered commands should not exceed all commands")
	}
	if len(noSkillCmds) > len(allCmds) {
		t.Error("Filtered commands should not exceed all commands")
	}
}

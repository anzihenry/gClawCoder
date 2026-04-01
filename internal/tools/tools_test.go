package tools

import (
	"testing"
)

func TestPortedTools(t *testing.T) {
	tls := PortedTools()
	if len(tls) == 0 {
		t.Error("Expected non-zero tools")
	}
}

func TestGetTool(t *testing.T) {
	tls := PortedTools()
	if len(tls) == 0 {
		t.Skip("No tools loaded")
	}

	firstTool := tls[0]
	tool := GetTool(firstTool.Name)
	if tool == nil {
		t.Errorf("Expected to find tool %s", firstTool.Name)
	}
}

func TestGetToolNotFound(t *testing.T) {
	tool := GetTool("nonexistent-tool-xyz")
	if tool != nil {
		t.Error("Expected nil for nonexistent tool")
	}
}

func TestFindTools(t *testing.T) {
	matches := FindTools("MCP", 10)
	if len(matches) == 0 {
		t.Log("No matches found for 'MCP' (may be expected)")
	}
}

func TestExecuteTool(t *testing.T) {
	result := ExecuteTool("BashTool", "test payload")
	if result.Handled {
		t.Logf("Tool handled: %s", result.Message)
	} else {
		t.Logf("Tool not handled: %s", result.Message)
	}
}

func TestExecuteToolNotFound(t *testing.T) {
	result := ExecuteTool("nonexistent", "test")
	if result.Handled {
		t.Error("Expected tool to not be handled")
	}
	if result.Message == "" {
		t.Error("Expected error message")
	}
}

func TestRenderToolIndex(t *testing.T) {
	output := RenderToolIndex(5, "")
	if output == "" {
		t.Error("Expected non-empty output")
	}
}

func TestRenderToolIndexWithQuery(t *testing.T) {
	output := RenderToolIndex(5, "MCP")
	if output == "" {
		t.Error("Expected non-empty output")
	}
}

func TestBuildToolBacklog(t *testing.T) {
	backlog := BuildToolBacklog()
	if backlog.Title == "" {
		t.Error("Expected non-empty title")
	}
}

func TestToolNames(t *testing.T) {
	names := ToolNames()
	if len(names) == 0 {
		t.Error("Expected non-zero tool names")
	}
}

func TestGetTools(t *testing.T) {
	tls := GetTools(false, true, nil, nil)
	if len(tls) == 0 {
		t.Error("Expected non-zero tools")
	}
}

func TestGetToolsSimpleMode(t *testing.T) {
	allTools := GetTools(false, true, nil, nil)
	simpleTools := GetTools(true, true, nil, nil)

	if len(simpleTools) > len(allTools) {
		t.Error("Simple mode should not exceed all tools")
	}
}

func TestGetToolsNoMCP(t *testing.T) {
	allTools := GetTools(false, true, nil, nil)
	noMCPTools := GetTools(false, false, nil, nil)

	if len(noMCPTools) > len(allTools) {
		t.Error("No MCP should not exceed all tools")
	}
}

func TestFilterToolsByPermissionContext(t *testing.T) {
	tls := PortedTools()
	if len(tls) == 0 {
		t.Skip("No tools loaded")
	}

	// 测试阻止特定工具
	blockedTools := []string{tls[0].Name}
	filtered := FilterToolsByPermissionContext(tls, blockedTools, nil)
	if len(filtered) >= len(tls) {
		t.Error("Filtered tools should be less than all tools")
	}

	// 测试阻止前缀
	blockedPrefixes := []string{"mcp"}
	filtered = FilterToolsByPermissionContext(tls, nil, blockedPrefixes)
	if len(filtered) > len(tls) {
		t.Error("Filtered tools should not exceed all tools")
	}
}

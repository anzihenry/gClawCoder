package toolkit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// NotebookTool Jupyter 笔记本编辑工具
type NotebookTool struct {
	notebooks map[string]*Notebook
}

// Notebook Jupyter 笔记本结构
type Notebook struct {
	Cells         []NotebookCell   `json:"cells"`
	Metadata      NotebookMetadata `json:"metadata"`
	NbFormat      int              `json:"nbformat"`
	NbFormatMinor int              `json:"nbformat_minor"`
}

// NotebookCell 笔记本单元格
type NotebookCell struct {
	CellType       string       `json:"cell_type"` // code, markdown, raw
	Source         []string     `json:"source"`
	Outputs        []CellOutput `json:"outputs,omitempty"`
	ExecutionCount *int         `json:"execution_count,omitempty"`
	Metadata       interface{}  `json:"metadata,omitempty"`
}

// CellOutput 单元格输出
type CellOutput struct {
	OutputType string      `json:"output_type"`    // stream, execute_result, error
	Name       string      `json:"name,omitempty"` // stdout, stderr
	Text       interface{} `json:"text,omitempty"`
	Data       interface{} `json:"data,omitempty"`
	EName      string      `json:"ename,omitempty"`
	EValue     string      `json:"evalue,omitempty"`
	Traceback  []string    `json:"traceback,omitempty"`
}

// NotebookMetadata 笔记本元数据
type NotebookMetadata struct {
	Kernelspec   *KernelSpec   `json:"kernelspec,omitempty"`
	LanguageInfo *LanguageInfo `json:"language_info,omitempty"`
}

// KernelSpec 内核规格
type KernelSpec struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Language    string `json:"language,omitempty"`
}

// LanguageInfo 语言信息
type LanguageInfo struct {
	Name          string `json:"name"`
	Version       string `json:"version,omitempty"`
	Mimetype      string `json:"mimetype,omitempty"`
	FileExtension string `json:"file_extension,omitempty"`
	PygmentsLexer string `json:"pygments_lexer,omitempty"`
}

// NewNotebookTool 创建 Notebook 工具
func NewNotebookTool() *NotebookTool {
	return &NotebookTool{
		notebooks: make(map[string]*Notebook),
	}
}

// NotebookInput Notebook 输入
type NotebookInput struct {
	Action     string       `json:"action"` // create, load, save, get, add_cell, update_cell, delete_cell, execute_cell, clear_outputs
	Path       string       `json:"path,omitempty"`
	CellIndex  int          `json:"cellIndex,omitempty"`
	CellType   string       `json:"cellType,omitempty"` // code, markdown
	Source     string       `json:"source,omitempty"`
	Outputs    []CellOutput `json:"outputs,omitempty"`
	KernelName string       `json:"kernelName,omitempty"`
	Language   string       `json:"language,omitempty"`
}

// Execute 执行 Notebook 操作
func (t *NotebookTool) Execute(input json.RawMessage) (string, error) {
	var inp NotebookInput
	if err := json.Unmarshal(input, &inp); err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	switch inp.Action {
	case "create":
		return t.createNotebook(inp)
	case "load":
		return t.loadNotebook(inp.Path)
	case "save":
		return t.saveNotebook(inp)
	case "get":
		return t.getNotebook(inp.Path)
	case "add_cell":
		return t.addCell(inp)
	case "update_cell":
		return t.updateCell(inp)
	case "delete_cell":
		return t.deleteCell(inp)
	case "execute_cell":
		return t.executeCell(inp)
	case "clear_outputs":
		return t.clearOutputs(inp)
	default:
		return "", fmt.Errorf("unknown action: %s", inp.Action)
	}
}

// createNotebook 创建新笔记本
func (t *NotebookTool) createNotebook(inp NotebookInput) (string, error) {
	nb := &Notebook{
		Cells:         make([]NotebookCell, 0),
		Metadata:      NotebookMetadata{},
		NbFormat:      4,
		NbFormatMinor: 5,
	}

	// 设置内核规格
	if inp.KernelName != "" || inp.Language != "" {
		nb.Metadata.Kernelspec = &KernelSpec{
			Name:        inp.KernelName,
			DisplayName: inp.KernelName,
			Language:    inp.Language,
		}
	}

	// 设置语言信息
	if inp.Language != "" {
		nb.Metadata.LanguageInfo = &LanguageInfo{
			Name: inp.Language,
		}
	}

	// 存储
	if inp.Path != "" {
		t.notebooks[inp.Path] = nb
	}

	return fmt.Sprintf("Created new notebook%s", formatPath(inp.Path)), nil
}

// loadNotebook 加载笔记本文件
func (t *NotebookTool) loadNotebook(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is required")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read notebook: %w", err)
	}

	var nb Notebook
	if err := json.Unmarshal(data, &nb); err != nil {
		return "", fmt.Errorf("invalid notebook format: %w", err)
	}

	t.notebooks[path] = &nb

	cellCount := len(nb.Cells)
	codeCells := 0
	mdCells := 0
	for _, cell := range nb.Cells {
		if cell.CellType == "code" {
			codeCells++
		} else if cell.CellType == "markdown" {
			mdCells++
		}
	}

	return fmt.Sprintf("Loaded notebook: %s\n  Cells: %d (%d code, %d markdown)",
		path, cellCount, codeCells, mdCells), nil
}

// saveNotebook 保存笔记本
func (t *NotebookTool) saveNotebook(inp NotebookInput) (string, error) {
	path := inp.Path
	if path == "" {
		return "", fmt.Errorf("path is required")
	}

	nb, ok := t.notebooks[path]
	if !ok {
		return "", fmt.Errorf("notebook not loaded: %s", path)
	}

	data, err := json.MarshalIndent(nb, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal notebook: %w", err)
	}

	// 创建目录
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write notebook: %w", err)
	}

	return fmt.Sprintf("Saved notebook: %s", path), nil
}

// getNotebook 获取笔记本内容
func (t *NotebookTool) getNotebook(path string) (string, error) {
	nb, ok := t.notebooks[path]
	if !ok {
		// 尝试从文件加载
		if _, err := t.loadNotebook(path); err != nil {
			return "", err
		}
		nb = t.notebooks[path]
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Notebook: %s\n", path))
	sb.WriteString(fmt.Sprintf("Format: %d.%d\n", nb.NbFormat, nb.NbFormatMinor))
	sb.WriteString(fmt.Sprintf("Cells: %d\n\n", len(nb.Cells)))

	for i, cell := range nb.Cells {
		icon := "📝"
		if cell.CellType == "code" {
			icon = "💻"
		}

		source := strings.Join(cell.Source, "")
		if len(source) > 100 {
			source = source[:100] + "..."
		}

		outputCount := len(cell.Outputs)

		sb.WriteString(fmt.Sprintf("%s Cell %d [%s]", icon, i, cell.CellType))
		if outputCount > 0 {
			sb.WriteString(fmt.Sprintf(" - %d outputs", outputCount))
		}
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("   %s\n", source))
	}

	return sb.String(), nil
}

// addCell 添加单元格
func (t *NotebookTool) addCell(inp NotebookInput) (string, error) {
	nb, ok := t.notebooks[inp.Path]
	if !ok {
		return "", fmt.Errorf("notebook not found: %s", inp.Path)
	}

	cellType := inp.CellType
	if cellType == "" {
		cellType = "code"
	}

	cell := NotebookCell{
		CellType: cellType,
		Source:   splitSource(inp.Source),
	}

	// 插入到指定位置
	if inp.CellIndex >= 0 && inp.CellIndex < len(nb.Cells) {
		nb.Cells = append(nb.Cells[:inp.CellIndex], append([]NotebookCell{cell}, nb.Cells[inp.CellIndex:]...)...)
	} else {
		nb.Cells = append(nb.Cells, cell)
	}

	return fmt.Sprintf("Added %s cell at index %d", cellType, inp.CellIndex), nil
}

// updateCell 更新单元格
func (t *NotebookTool) updateCell(inp NotebookInput) (string, error) {
	nb, ok := t.notebooks[inp.Path]
	if !ok {
		return "", fmt.Errorf("notebook not found: %s", inp.Path)
	}

	if inp.CellIndex < 0 || inp.CellIndex >= len(nb.Cells) {
		return "", fmt.Errorf("invalid cell index: %d", inp.CellIndex)
	}

	cell := &nb.Cells[inp.CellIndex]

	if inp.Source != "" {
		cell.Source = splitSource(inp.Source)
	}
	if inp.CellType != "" {
		cell.CellType = inp.CellType
	}
	if inp.Outputs != nil {
		cell.Outputs = inp.Outputs
	}

	return fmt.Sprintf("Updated cell %d", inp.CellIndex), nil
}

// deleteCell 删除单元格
func (t *NotebookTool) deleteCell(inp NotebookInput) (string, error) {
	nb, ok := t.notebooks[inp.Path]
	if !ok {
		return "", fmt.Errorf("notebook not found: %s", inp.Path)
	}

	if inp.CellIndex < 0 || inp.CellIndex >= len(nb.Cells) {
		return "", fmt.Errorf("invalid cell index: %d", inp.CellIndex)
	}

	deletedType := nb.Cells[inp.CellIndex].CellType
	nb.Cells = append(nb.Cells[:inp.CellIndex], nb.Cells[inp.CellIndex+1:]...)

	return fmt.Sprintf("Deleted %s cell at index %d", deletedType, inp.CellIndex), nil
}

// executeCell 执行代码单元格 (模拟)
func (t *NotebookTool) executeCell(inp NotebookInput) (string, error) {
	nb, ok := t.notebooks[inp.Path]
	if !ok {
		return "", fmt.Errorf("notebook not found: %s", inp.Path)
	}

	if inp.CellIndex < 0 || inp.CellIndex >= len(nb.Cells) {
		return "", fmt.Errorf("invalid cell index: %d", inp.CellIndex)
	}

	cell := &nb.Cells[inp.CellIndex]
	if cell.CellType != "code" {
		return "", fmt.Errorf("can only execute code cells")
	}

	// 模拟执行 - 实际应该连接 Jupyter 内核
	executionCount := 1
	if cell.ExecutionCount != nil {
		executionCount = *cell.ExecutionCount + 1
	}
	cell.ExecutionCount = &executionCount

	// 添加模拟输出
	cell.Outputs = []CellOutput{
		{
			OutputType: "stream",
			Name:       "stdout",
			Text:       "Execution completed (simulated - connect to Jupyter kernel for real execution)\n",
		},
	}

	return fmt.Sprintf("Executed cell %d (simulated)", inp.CellIndex), nil
}

// clearOutputs 清除输出
func (t *NotebookTool) clearOutputs(inp NotebookInput) (string, error) {
	nb, ok := t.notebooks[inp.Path]
	if !ok {
		return "", fmt.Errorf("notebook not found: %s", inp.Path)
	}

	count := 0
	for i := range nb.Cells {
		if nb.Cells[i].CellType == "code" && len(nb.Cells[i].Outputs) > 0 {
			nb.Cells[i].Outputs = nil
			nb.Cells[i].ExecutionCount = nil
			count++
		}
	}

	return fmt.Sprintf("Cleared outputs from %d cells", count), nil
}

// splitSource 分割源代码为行
func splitSource(source string) []string {
	if source == "" {
		return []string{}
	}

	lines := strings.Split(source, "\n")
	result := make([]string, len(lines))
	for i, line := range lines {
		result[i] = line + "\n"
	}
	return result
}

// formatPath 格式化路径显示
func formatPath(path string) string {
	if path == "" {
		return ""
	}
	return " (" + path + ")"
}

// GetDescription 获取描述
func (t *NotebookTool) GetDescription() string {
	return "Create, edit, and manage Jupyter notebooks (.ipynb files)"
}

// GetInputSchema 获取输入 schema
func (t *NotebookTool) GetInputSchema() string {
	return `{
		"type":"object",
		"properties":{
			"action":{"type":"string","enum":["create","load","save","get","add_cell","update_cell","delete_cell","execute_cell","clear_outputs"]},
			"path":{"type":"string"},
			"cellIndex":{"type":"integer"},
			"cellType":{"type":"string","enum":["code","markdown"]},
			"source":{"type":"string"},
			"kernelName":{"type":"string"},
			"language":{"type":"string"}
		},
		"required":["action"]
	}`
}

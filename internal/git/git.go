package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// GitStatus Git 状态
type GitStatus struct {
	Branch     string   `json:"branch"`
	Ahead      int      `json:"ahead"`
	Behind     int      `json:"behind"`
	Modified   []string `json:"modified"`
	Added      []string `json:"added"`
	Deleted    []string `json:"deleted"`
	Untracked  []string `json:"untracked"`
	Conflicted []string `json:"conflicted"`
}

// GitDiff Git 差异
type GitDiff struct {
	Stats DiffStats  `json:"stats"`
	Files []FileDiff `json:"files"`
	Patch string     `json:"patch"`
}

// DiffStats 差异统计
type DiffStats struct {
	FilesChanged int `json:"filesChanged"`
	Insertions   int `json:"insertions"`
	Deletions    int `json:"deletions"`
}

// FileDiff 文件差异
type FileDiff struct {
	Path       string `json:"path"`
	Status     string `json:"status"`
	Insertions int    `json:"insertions"`
	Deletions  int    `json:"deletions"`
	Patch      string `json:"patch"`
}

// Client Git 客户端
type Client struct {
	workingDir string
	timeout    time.Duration
}

// NewClient 创建 Git 客户端
func NewClient(workingDir string) *Client {
	return &Client{
		workingDir: workingDir,
		timeout:    30 * time.Second,
	}
}

// IsGitRepo 检查是否是 Git 仓库
func (c *Client) IsGitRepo() bool {
	_, err := c.runGit("rev-parse", "--git-dir")
	return err == nil
}

// GetStatus 获取状态
func (c *Client) GetStatus() (*GitStatus, error) {
	branch, err := c.getCurrentBranch()
	if err != nil {
		return nil, err
	}

	status := &GitStatus{
		Branch: branch,
	}

	// 获取文件状态
	output, err := c.runGit("status", "--porcelain", "-b")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			// 分支信息
			parts := strings.Split(line, " ")
			if len(parts) >= 3 && parts[2] == "ahead" {
				fmt.Sscanf(parts[3], "%d", &status.Ahead)
				if len(parts) >= 5 && parts[4] == "behind" {
					fmt.Sscanf(parts[5], "%d", &status.Behind)
				}
			}
		} else if line != "" {
			// 文件状态
			if len(line) >= 3 {
				stagingCode := line[0:1]
				worktreeCode := line[1:2]
				filePath := strings.TrimSpace(line[3:])

				switch stagingCode {
				case "A":
					status.Added = append(status.Added, filePath)
				case "D":
					status.Deleted = append(status.Deleted, filePath)
				case "M":
					status.Modified = append(status.Modified, filePath)
				case "?":
					status.Untracked = append(status.Untracked, filePath)
				case "U":
					status.Conflicted = append(status.Conflicted, filePath)
				}

				if worktreeCode == "M" && stagingCode != "M" {
					status.Modified = append(status.Modified, filePath)
				}
			}
		}
	}

	return status, nil
}

// GetDiff 获取差异
func (c *Client) GetDiff() (*GitDiff, error) {
	patch, err := c.runGit("diff", "--stat")
	if err != nil {
		return nil, err
	}

	fullPatch, err := c.runGit("diff")
	if err != nil {
		return nil, err
	}

	diff := &GitDiff{
		Patch: fullPatch,
	}

	// 解析统计信息
	lines := strings.Split(patch, "\n")
	for _, line := range lines {
		if strings.Contains(line, "file changed") {
			diff.Stats.FilesChanged++
		} else if strings.Contains(line, "insertion") {
			fmt.Sscanf(line, "%d insertion", &diff.Stats.Insertions)
		} else if strings.Contains(line, "deletion") {
			fmt.Sscanf(line, "%d deletion", &diff.Stats.Deletions)
		}
	}

	return diff, nil
}

// GetLog 获取提交日志
func (c *Client) GetLog(limit int) ([]Commit, error) {
	if limit == 0 {
		limit = 10
	}

	output, err := c.runGit("log", "-n", fmt.Sprintf("%d", limit), "--pretty=format:%H|%an|%ae|%ad|%s")
	if err != nil {
		return nil, err
	}

	var commits []Commit
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "|", 5)
		if len(parts) == 5 {
			commits = append(commits, Commit{
				Hash:    parts[0],
				Author:  parts[1],
				Email:   parts[2],
				Date:    parts[3],
				Subject: parts[4],
			})
		}
	}

	return commits, nil
}

// Commit 提交
type Commit struct {
	Hash    string `json:"hash"`
	Author  string `json:"author"`
	Email   string `json:"email"`
	Date    string `json:"date"`
	Subject string `json:"subject"`
}

// Add 添加到暂存区
func (c *Client) Add(paths ...string) error {
	if len(paths) == 0 {
		paths = []string{"."}
	}
	args := append([]string{"add"}, paths...)
	_, err := c.runGit(args...)
	return err
}

// CommitChanges 提交更改
func (c *Client) CommitChanges(message string) error {
	_, err := c.runGit("commit", "-m", message)
	return err
}

// Push 推送
func (c *Client) Push(remote, branch string) error {
	if remote == "" {
		remote = "origin"
	}
	if branch == "" {
		var err error
		branch, err = c.getCurrentBranch()
		if err != nil {
			return err
		}
	}
	_, err := c.runGit("push", remote, branch)
	return err
}

// Pull 拉取
func (c *Client) Pull(remote, branch string) error {
	if remote == "" {
		remote = "origin"
	}
	_, err := c.runGit("pull", remote)
	return err
}

// CreateBranch 创建分支
func (c *Client) CreateBranch(name string) error {
	_, err := c.runGit("checkout", "-b", name)
	return err
}

// CheckoutBranch 切换分支
func (c *Client) CheckoutBranch(name string) error {
	_, err := c.runGit("checkout", name)
	return err
}

// GetCurrentBranch 获取当前分支
func (c *Client) getCurrentBranch() (string, error) {
	return c.runGit("rev-parse", "--abbrev-ref", "HEAD")
}

// runGit 运行 Git 命令
func (c *Client) runGit(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = c.workingDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := strings.TrimSpace(stdout.String())

	if err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("git error: %w\n%s", err, stderr.String())
		}
		return "", err
	}

	return output, nil
}

// GetRemoteURL 获取远程 URL
func (c *Client) GetRemoteURL(remote string) (string, error) {
	if remote == "" {
		remote = "origin"
	}
	return c.runGit("remote", "get-url", remote)
}

// GetTags 获取标签
func (c *Client) GetTags() ([]string, error) {
	output, err := c.runGit("tag", "-l")
	if err != nil {
		return nil, err
	}
	return strings.Split(output, "\n"), nil
}

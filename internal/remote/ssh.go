package remote

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// SSHConfig SSH 配置
type SSHConfig struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	Password   string `json:"password,omitempty"`
	KeyPath    string `json:"keyPath,omitempty"`
	Passphrase string `json:"passphrase,omitempty"`
}

// SSHClient SSH 客户端
type SSHClient struct {
	config    *SSHConfig
	client    *ssh.Client
	connected bool
}

// NewSSHClient 创建 SSH 客户端
func NewSSHClient(config *SSHConfig) *SSHClient {
	return &SSHClient{
		config:    config,
		connected: false,
	}
}

// Connect 连接 SSH 服务器
func (c *SSHClient) Connect() error {
	if c.connected {
		return nil
	}

	authMethods := []ssh.AuthMethod{}

	// 尝试 SSH Agent
	if socket := os.Getenv("SSH_AUTH_SOCK"); socket != "" {
		conn, err := net.Dial("unix", socket)
		if err == nil {
			agentClient := agent.NewClient(conn)
			authMethods = append(authMethods, ssh.PublicKeysCallback(agentClient.Signers))
		}
	}

	// 尝试私钥文件
	if c.config.KeyPath != "" {
		key, err := os.ReadFile(c.config.KeyPath)
		if err == nil {
			var signer ssh.Signer
			if c.config.Passphrase != "" {
				signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(c.config.Passphrase))
			} else {
				signer, err = ssh.ParsePrivateKey(key)
			}
			if err == nil {
				authMethods = append(authMethods, ssh.PublicKeys(signer))
			}
		}
	}

	// 尝试密码
	if c.config.Password != "" {
		authMethods = append(authMethods, ssh.Password(c.config.Password))
	}

	sshConfig := &ssh.ClientConfig{
		User:            c.config.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.client = client
	c.connected = true
	return nil
}

// Close 关闭连接
func (c *SSHClient) Close() error {
	if c.client != nil {
		c.connected = false
		return c.client.Close()
	}
	return nil
}

// Execute 执行远程命令
func (c *SSHClient) Execute(command string) (string, error) {
	if !c.connected {
		if err := c.Connect(); err != nil {
			return "", err
		}
	}

	session, err := c.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	if err := session.Run(command); err != nil {
		return stdout.String(), fmt.Errorf("command failed: %w\nstderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// Upload 上传文件
func (c *SSHClient) Upload(localPath, remotePath string) error {
	if !c.connected {
		if err := c.Connect(); err != nil {
			return err
		}
	}

	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read local file: %w", err)
	}

	session, err := c.client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// 使用 scp 或 sftp
	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}

	go func() {
		defer stdin.Close()
		stdin.Write(data)
	}()

	return session.Run(fmt.Sprintf("cat > %s", remotePath))
}

// Download 下载文件
func (c *SSHClient) Download(remotePath, localPath string) error {
	if !c.connected {
		if err := c.Connect(); err != nil {
			return err
		}
	}

	session, err := c.client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	var stdout bytes.Buffer
	session.Stdout = &stdout

	if err := session.Run(fmt.Sprintf("cat %s", remotePath)); err != nil {
		return err
	}

	return os.WriteFile(localPath, stdout.Bytes(), 0644)
}

// IsConnected 检查连接状态
func (c *SSHClient) IsConnected() bool {
	return c.connected
}

// GetConnectionInfo 获取连接信息
func (c *SSHClient) GetConnectionInfo() string {
	return fmt.Sprintf("SSH: %s@%s:%d", c.config.Username, c.config.Host, c.config.Port)
}

// DeepLinkClient DeepLink 客户端
type DeepLinkClient struct {
	scheme string
}

// NewDeepLinkClient 创建 DeepLink 客户端
func NewDeepLinkClient(scheme string) *DeepLinkClient {
	return &DeepLinkClient{
		scheme: scheme,
	}
}

// OpenURL 打开 URL
func (c *DeepLinkClient) OpenURL(url string) error {
	// 实际应该调用系统 API 打开 deep link
	fmt.Printf("Opening deep link: %s://%s\n", c.scheme, url)
	return nil
}

// CreateSession 创建会话链接
func (c *DeepLinkClient) CreateSession(sessionID string) (string, error) {
	link := fmt.Sprintf("%s://session/%s", c.scheme, sessionID)
	return link, nil
}

// ShareContext 共享上下文
func (c *DeepLinkClient) ShareContext(context map[string]interface{}) (string, error) {
	// 生成共享链接
	link := fmt.Sprintf("%s://share?data=%v", c.scheme, context)
	return link, nil
}

package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// OAuth 配置
const (
	OAuthClientID            = "claw-cli-client"
	OAuthAuthURL             = "https://auth.anthropic.com/oauth2/auth"
	OAuthTokenURL            = "https://auth.anthropic.com/oauth2/token"
	OAuthRedirectURI         = "http://localhost:62234/oauth/callback"
	OAuthScope               = "orgs:read models:inference"
	OAuthCodeChallengeMethod = "S256"
)

// TokenInfo Token 信息
type TokenInfo struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Scope        string    `json:"scope"`
	Expiry       time.Time `json:"expiry,omitempty"`
}

// IsExpired 检查是否过期
func (t *TokenInfo) IsExpired() bool {
	if t.Expiry.IsZero() {
		return false
	}
	return time.Now().After(t.Expiry)
}

// IsExpiringSoon 检查是否即将过期 (5 分钟内)
func (t *TokenInfo) IsExpiringSoon() bool {
	if t.Expiry.IsZero() {
		return false
	}
	return time.Now().Add(5 * time.Minute).After(t.Expiry)
}

// OAuthConfig OAuth 配置
type OAuthConfig struct {
	ClientID            string
	AuthURL             string
	TokenURL            string
	RedirectURI         string
	Scope               string
	CodeChallengeMethod string
}

// DefaultOAuthConfig 默认配置
func DefaultOAuthConfig() *OAuthConfig {
	return &OAuthConfig{
		ClientID:            OAuthClientID,
		AuthURL:             OAuthAuthURL,
		TokenURL:            OAuthTokenURL,
		RedirectURI:         OAuthRedirectURI,
		Scope:               OAuthScope,
		CodeChallengeMethod: OAuthCodeChallengeMethod,
	}
}

// OAuthClient OAuth 客户端
type OAuthClient struct {
	config       *OAuthConfig
	token        *TokenInfo
	tokenPath    string
	codeVerifier string
	state        string
}

// NewOAuthClient 创建 OAuth 客户端
func NewOAuthClient(config *OAuthConfig) *OAuthClient {
	if config == nil {
		config = DefaultOAuthConfig()
	}

	home, _ := os.UserHomeDir()
	tokenPath := filepath.Join(home, ".claw", "oauth-token.json")

	return &OAuthClient{
		config:    config,
		tokenPath: tokenPath,
	}
}

// GenerateCodeVerifier 生成 PKCE verifier
func GenerateCodeVerifier() (string, error) {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GenerateCodeChallenge 生成 PKCE challenge
func GenerateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// GenerateState 生成 state 参数
func GenerateState() (string, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GetAuthURL 获取授权 URL
func (o *OAuthClient) GetAuthURL() (string, error) {
	var err error
	o.codeVerifier, err = GenerateCodeVerifier()
	if err != nil {
		return "", fmt.Errorf("failed to generate code verifier: %w", err)
	}

	o.state, err = GenerateState()
	if err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}

	codeChallenge := GenerateCodeChallenge(o.codeVerifier)

	params := url.Values{}
	params.Set("client_id", o.config.ClientID)
	params.Set("redirect_uri", o.config.RedirectURI)
	params.Set("response_type", "code")
	params.Set("scope", o.config.Scope)
	params.Set("code_challenge", codeChallenge)
	params.Set("code_challenge_method", o.config.CodeChallengeMethod)
	params.Set("state", o.state)
	params.Set("audience", "https://api.anthropic.com")

	return fmt.Sprintf("%s?%s", o.config.AuthURL, params.Encode()), nil
}

// Login 登录获取 Token
func (o *OAuthClient) Login() (*TokenInfo, error) {
	// 获取授权 URL
	authURL, err := o.GetAuthURL()
	if err != nil {
		return nil, fmt.Errorf("failed to get auth URL: %w", err)
	}

	fmt.Println("\n🔐 OAuth Login")
	fmt.Println("═══════════════════════════════════════")
	fmt.Println("\nOpening browser for authentication...")
	fmt.Println("\nIf browser doesn't open, please visit:")
	fmt.Println(authURL)

	// 打开浏览器
	openBrowser(authURL)

	// 启动本地服务器接收回调
	token, err := o.startCallbackServer()
	if err != nil {
		return nil, fmt.Errorf("callback failed: %w", err)
	}

	o.token = token

	// 保存 Token
	if err := o.saveToken(); err != nil {
		return nil, fmt.Errorf("failed to save token: %w", err)
	}

	fmt.Println("\n✅ Login successful!")
	fmt.Println("Token saved to:", o.tokenPath)

	return token, nil
}

// startCallbackServer 启动回调服务器
func (o *OAuthClient) startCallbackServer() (*TokenInfo, error) {
	type callbackResult struct {
		code  string
		state string
		err   error
	}

	resultCh := make(chan callbackResult, 1)

	// 创建 HTTP 服务器
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/callback" {
			http.NotFound(w, r)
			return
		}

		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")
		errorMsg := r.URL.Query().Get("error")

		if errorMsg != "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Authentication failed: %s", errorMsg)
			resultCh <- callbackResult{err: fmt.Errorf("auth error: %s", errorMsg)}
			return
		}

		if code == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Missing code parameter")
			resultCh <- callbackResult{err: fmt.Errorf("missing code")}
			return
		}

		// 显示成功页面
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head><title>Login Successful</title></head>
<body style="font-family: sans-serif; text-align: center; padding: 50px;">
<h1 style="color: #22c55e;">✅ Login Successful!</h1>
<p>You can close this window and return to the terminal.</p>
</body>
</html>
`)

		resultCh <- callbackResult{code: code, state: state}
	})

	server := &http.Server{
		Addr:    ":62234",
		Handler: handler,
	}

	// 异步启动服务器
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			resultCh <- callbackResult{err: err}
		}
	}()

	// 等待回调
	result := <-resultCh
	if result.err != nil {
		server.Close()
		return nil, result.err
	}

	// 验证 state
	if result.state != o.state {
		server.Close()
		return nil, fmt.Errorf("invalid state parameter")
	}

	// 关闭服务器
	server.Close()

	// 用 code 换取 token
	return o.exchangeCodeForToken(result.code)
}

// exchangeCodeForToken 用授权码换取 Token
func (o *OAuthClient) exchangeCodeForToken(code string) (*TokenInfo, error) {
	data := url.Values{}
	data.Set("client_id", o.config.ClientID)
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", o.config.RedirectURI)
	data.Set("code_verifier", o.codeVerifier)

	req, err := http.NewRequest("POST", o.config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed (%d): %s", resp.StatusCode, string(body))
	}

	var token TokenInfo
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("failed to decode token: %w", err)
	}

	// 设置过期时间
	if token.ExpiresIn > 0 {
		token.Expiry = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	}

	return &token, nil
}

// Logout 登出
func (o *OAuthClient) Logout() error {
	// 删除本地 Token
	if err := os.Remove(o.tokenPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove token file: %w", err)
	}

	o.token = nil
	fmt.Println("✅ Logged out successfully")
	fmt.Println("Token file removed:", o.tokenPath)

	return nil
}

// GetToken 获取 Token (自动刷新)
func (o *OAuthClient) GetToken() (*TokenInfo, error) {
	// 如果已有 Token 且未过期，直接返回
	if o.token != nil && !o.token.IsExpired() {
		return o.token, nil
	}

	// 尝试从文件加载
	if err := o.loadToken(); err != nil {
		return nil, fmt.Errorf("no token available, please login first: %w", err)
	}

	// 检查是否需要刷新
	if o.token.IsExpiringSoon() {
		if err := o.refreshToken(); err != nil {
			// 刷新失败，需要重新登录
			return nil, fmt.Errorf("token expired, please login again: %w", err)
		}
	}

	return o.token, nil
}

// refreshToken 刷新 Token
func (o *OAuthClient) refreshToken() error {
	if o.token == nil || o.token.RefreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}

	data := url.Values{}
	data.Set("client_id", o.config.ClientID)
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", o.token.RefreshToken)

	req, err := http.NewRequest("POST", o.config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token refresh failed (%d): %s", resp.StatusCode, string(body))
	}

	var newToken TokenInfo
	if err := json.NewDecoder(resp.Body).Decode(&newToken); err != nil {
		return fmt.Errorf("failed to decode token: %w", err)
	}

	if newToken.ExpiresIn > 0 {
		newToken.Expiry = time.Now().Add(time.Duration(newToken.ExpiresIn) * time.Second)
	}

	// 保留 refresh token
	if newToken.RefreshToken == "" {
		newToken.RefreshToken = o.token.RefreshToken
	}

	o.token = &newToken

	// 保存新 Token
	return o.saveToken()
}

// saveToken 保存 Token 到文件
func (o *OAuthClient) saveToken() error {
	if o.token == nil {
		return fmt.Errorf("no token to save")
	}

	// 创建目录
	dir := filepath.Dir(o.tokenPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(o.token, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	if err := os.WriteFile(o.tokenPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}

// loadToken 从文件加载 Token
func (o *OAuthClient) loadToken() error {
	data, err := os.ReadFile(o.tokenPath)
	if err != nil {
		return fmt.Errorf("failed to read token file: %w", err)
	}

	var token TokenInfo
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf("failed to parse token: %w", err)
	}

	o.token = &token
	return nil
}

// IsLoggedIn 检查是否已登录
func (o *OAuthClient) IsLoggedIn() bool {
	if err := o.loadToken(); err != nil {
		return false
	}
	return o.token != nil && !o.token.IsExpired()
}

// GetTokenInfo 获取 Token 信息 (用于展示)
func (o *OAuthClient) GetTokenInfo() (map[string]interface{}, error) {
	if err := o.loadToken(); err != nil {
		return nil, err
	}

	if o.token == nil {
		return nil, fmt.Errorf("no token")
	}

	info := map[string]interface{}{
		"token_type": o.token.TokenType,
		"scope":      o.token.Scope,
		"expires_in": o.token.ExpiresIn,
		"expiry":     o.token.Expiry.Format(time.RFC3339),
		"expired":    o.token.IsExpired(),
	}

	// 解析 scope
	scopes := strings.Fields(o.token.Scope)
	info["scopes"] = scopes

	return info, nil
}

// openBrowser 打开浏览器
func openBrowser(url string) {
	commands := []string{"open", "xdg-open", "gnome-open", "kde-open"}

	for _, cmd := range commands {
		if _, err := exec.LookPath(cmd); err == nil {
			exec.Command(cmd, url).Start()
			return
		}
	}

	// 如果都失败了，至少打印 URL
	fmt.Println("\nPlease open in browser:")
	fmt.Println(url)
}

// GetTokenPath 获取 Token 文件路径
func (o *OAuthClient) GetTokenPath() string {
	return o.tokenPath
}

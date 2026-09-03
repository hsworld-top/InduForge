package gateway

import (
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Config 只描述单工程入口所需的本地事实。节点身份和 Release 路径由
// NodeAgent 注入，工程入口不会自行向中心查询或下载制品。
type Config struct {
	Listen          string
	ClientRoot      string
	ReleaseRoot     string
	RuntimeAPIURL   string
	ViewerTokenFile string
	DeploymentID    string
	AccountID       string
	ProjectID       string
	SiteID          string
	NodeID          string
	Version         string
	ExecutionForm   string
}

func (c *Config) Validate() error {
	if strings.TrimSpace(c.Listen) == "" {
		return errors.New("listen 不能为空")
	}
	if strings.TrimSpace(c.DeploymentID) == "" || strings.TrimSpace(c.AccountID) == "" || strings.TrimSpace(c.ProjectID) == "" {
		return errors.New("deployment-id、account-id 和 project-id 不能为空")
	}
	if strings.TrimSpace(c.SiteID) == "" || strings.TrimSpace(c.NodeID) == "" {
		return errors.New("site-id 和 node-id 不能为空")
	}
	if strings.TrimSpace(c.Version) == "" {
		return errors.New("version 不能为空")
	}
	if c.ExecutionForm == "" {
		if runtime.GOOS == "windows" {
			c.ExecutionForm = "native-windows"
		} else {
			c.ExecutionForm = "native-linux"
		}
	}
	if c.ExecutionForm != "native-linux" && c.ExecutionForm != "native-windows" {
		return fmt.Errorf("execution-form %q 不受支持", c.ExecutionForm)
	}

	root, err := filepath.Abs(strings.TrimSpace(c.ClientRoot))
	if err != nil {
		return fmt.Errorf("解析 client-root: %w", err)
	}
	if err := validateClientRoot(root); err != nil {
		return err
	}
	c.ClientRoot = root

	upstream, err := parseLoopbackUpstream(c.RuntimeAPIURL)
	if err != nil {
		return err
	}
	c.RuntimeAPIURL = upstream.String()
	if strings.TrimSpace(c.ViewerTokenFile) == "" {
		return errors.New("viewer-token-file 不能为空")
	}
	tokenFile, err := filepath.Abs(strings.TrimSpace(c.ViewerTokenFile))
	if err != nil {
		return fmt.Errorf("解析 viewer-token-file: %w", err)
	}
	if err := validateViewerTokenFile(tokenFile); err != nil {
		return err
	}
	c.ViewerTokenFile = tokenFile
	return nil
}

// validateViewerTokenFile 只允许网关读取投影的单个 viewer token。拒绝链接、目录及
// 其他用户可读文件，避免把运行凭据误指向 Release 或任意宿主路径。
func validateViewerTokenFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("读取 viewer-token-file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return errors.New("viewer-token-file 必须是普通文件")
	}
	if info.Mode().Perm()&0o007 != 0 {
		return errors.New("viewer-token-file 不能对其他用户可读")
	}
	if info.Size() < 1 || info.Size() > 4096 {
		return errors.New("viewer-token-file 大小非法")
	}
	return nil
}

func parseLoopbackUpstream(raw string) (*url.URL, error) {
	upstream, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("解析 runtime-api URL: %w", err)
	}
	if upstream.Scheme != "http" || upstream.Host == "" || upstream.User != nil || upstream.Fragment != "" || upstream.RawQuery != "" {
		return nil, errors.New("runtime-api 必须是无凭据、无查询参数的本机 http URL")
	}
	if upstream.Path != "" && upstream.Path != "/" {
		return nil, errors.New("runtime-api URL 不能包含路径")
	}
	host := upstream.Hostname()
	ip := net.ParseIP(host)
	if !strings.EqualFold(host, "localhost") && (ip == nil || !ip.IsLoopback()) {
		return nil, errors.New("runtime-api 必须监听在本机回环地址，不能绕过工程入口暴露")
	}
	upstream.Path = ""
	return upstream, nil
}

func validateClientRoot(root string) error {
	info, err := os.Stat(root)
	if err != nil {
		return fmt.Errorf("读取 client-root: %w", err)
	}
	if !info.IsDir() {
		return errors.New("client-root 必须是目录")
	}
	index, err := os.Lstat(filepath.Join(root, "index.html"))
	if err != nil {
		return fmt.Errorf("client-root 缺少 index.html: %w", err)
	}
	if !index.Mode().IsRegular() {
		return errors.New("index.html 必须是普通文件")
	}
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path != root && entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("client-root 不允许符号链接: %s", path)
		}
		return nil
	})
}

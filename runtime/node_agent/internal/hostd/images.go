package hostd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const ImageCacheDir = "/var/lib/induforge/image-cache"

var imageSHA = regexp.MustCompile(`^[a-f0-9]{64}$`)

type ImageReference struct {
	Reference    string `json:"reference"`
	ConfigDigest string `json:"configDigest"`
}
type ImageArtifact struct {
	SHA256 string           `json:"sha256"`
	Size   int64            `json:"size"`
	Images []ImageReference `json:"images"`
}
type ImageResult struct {
	Ready bool `json:"ready"`
}

func (p ImageArtifact) Validate() error {
	if !imageSHA.MatchString(p.SHA256) || p.Size <= 0 || p.Size > 32<<30 || len(p.Images) == 0 || len(p.Images) > 32 {
		return fmt.Errorf("镜像归档描述无效")
	}
	for _, im := range p.Images {
		if im.Reference == "" || len(im.Reference) > 512 || strings.HasPrefix(im.Reference, "-") || strings.ContainsAny(im.Reference, " \t\r\n") || !strings.HasPrefix(im.ConfigDigest, "sha256:") || !imageSHA.MatchString(strings.TrimPrefix(im.ConfigDigest, "sha256:")) {
			return fmt.Errorf("镜像引用或内容摘要无效")
		}
	}
	return nil
}

// CheckImages 核对 CRI 中镜像配置内容摘要；本地标记或相同 tag 不能证明镜像可用。
// 显式指定 socket 并跳过默认配置，支持使用自定义 data-dir 的 K3s 安装。
func (m *Manager) CheckImages(ctx context.Context, p ImageArtifact) (ImageResult, error) {
	if err := p.Validate(); err != nil {
		return ImageResult{}, err
	}
	for _, im := range p.Images {
		out, err := m.cfg.Runner.Run(ctx, m.cfg.BinaryPath, "crictl", "--config", "/dev/null", "--runtime-endpoint", "unix:///run/k3s/containerd/containerd.sock", "--image-endpoint", "unix:///run/k3s/containerd/containerd.sock", "inspecti", "--output", "json", im.Reference)
		if err != nil {
			return ImageResult{Ready: false}, nil
		}
		var v struct {
			Status struct {
				ID string `json:"id"`
			} `json:"status"`
		}
		if json.Unmarshal(out, &v) != nil || v.Status.ID != im.ConfigDigest {
			return ImageResult{Ready: false}, nil
		}
	}
	return ImageResult{Ready: true}, nil
}

// ImportImages 不接受中心路径或命令；将非特权缓存复制到 root 私有目录后校验，
// 只导入已核对的同一份副本，避免下载账户在校验后替换归档。
func (m *Manager) ImportImages(ctx context.Context, p ImageArtifact) (ImageResult, error) {
	if err := p.Validate(); err != nil {
		return ImageResult{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if result, err := m.CheckImages(ctx, p); err != nil || result.Ready {
		return result, err
	}
	cache := m.cfg.ImageCacheDir
	if cache == "" {
		cache = ImageCacheDir
	}
	source := filepath.Join(cache, p.SHA256+".tar")
	info, err := os.Lstat(source)
	if err != nil {
		return ImageResult{}, err
	}
	if !info.Mode().IsRegular() || info.Size() != p.Size {
		return ImageResult{}, fmt.Errorf("镜像归档不是有效普通文件")
	}
	input, err := os.Open(source)
	if err != nil {
		return ImageResult{}, err
	}
	defer input.Close()
	opened, err := input.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return ImageResult{}, fmt.Errorf("镜像归档在读取前已变化")
	}
	dir := filepath.Join(m.cfg.StateDir, "image-import")
	if err = ensureRealDirectory(dir, 0700); err != nil {
		return ImageResult{}, err
	}
	file, err := os.CreateTemp(dir, "verified-*.tar")
	if err != nil {
		return ImageResult{}, err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	hash := sha256.New()
	n, err := io.Copy(io.MultiWriter(file, hash), io.LimitReader(input, p.Size+1))
	if err != nil {
		return ImageResult{}, err
	}
	if n != p.Size || hex.EncodeToString(hash.Sum(nil)) != p.SHA256 {
		return ImageResult{}, fmt.Errorf("镜像归档完整性校验失败")
	}
	if err = file.Close(); err != nil {
		return ImageResult{}, err
	}
	if out, err := m.cfg.Runner.Run(ctx, m.cfg.BinaryPath, "ctr", "--address", "/run/k3s/containerd/containerd.sock", "-n", "k8s.io", "images", "import", file.Name()); err != nil {
		return ImageResult{}, fmt.Errorf("导入镜像失败: %w: %s", err, strings.TrimSpace(string(out)))
	}
	result, err := m.CheckImages(ctx, p)
	if err == nil && !result.Ready {
		err = fmt.Errorf("导入后镜像内容摘要不匹配")
	}
	if err == nil && result.Ready {
		// 已导入的镜像由 containerd 缓存；释放大归档避免节点磁盘占用翻倍。
		_ = os.Remove(source)
	}
	return result, err
}

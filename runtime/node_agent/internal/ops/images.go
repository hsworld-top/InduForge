package ops

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/indu-forge/node_agent/internal/hostd"
)

type ImageDownloadArtifact struct {
	hostd.ImageArtifact
	Architecture string `json:"architecture"`
	DownloadPath string `json:"downloadPath"`
}
type ImagePlan struct {
	Artifacts []ImageDownloadArtifact `json:"artifacts"`
}
type ImageState struct {
	Status    string   `json:"status"`
	Artifacts []string `json:"artifacts"`
	Message   string   `json:"message"`
}

func (a *Agent) setImageState(status string, ready []string, message string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.imageState = &ImageState{Status: status, Artifacts: append([]string{}, ready...), Message: truncateAgentMessage(message)}
}

// prepareImages 在任何基础服务或工程启动前完成下载和导入；长下载期间继续心跳。
func (a *Agent) prepareImages(ctx context.Context, plan ImagePlan) (resultErr error) {
	ready := []string{}
	defer func() {
		if resultErr != nil {
			a.setImageState("failed", ready, resultErr.Error())
			_ = a.Heartbeat(ctx)
		}
	}()
	if a.hostd == nil {
		return fmt.Errorf("镜像准备需要本机 hostd")
	}
	if len(plan.Artifacts) > 128 {
		return fmt.Errorf("镜像计划过大")
	}
	workCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-workCtx.Done():
				return
			case <-ticker.C:
				_ = a.Heartbeat(workCtx)
			}
		}
	}()
	for _, artifact := range plan.Artifacts {
		if err := artifact.Validate(); err != nil {
			return err
		}
		if artifact.Architecture != runtime.GOARCH {
			return fmt.Errorf("镜像架构与节点不一致")
		}
		expected := "/api/v1/ops/agent/nodes/" + a.currentIdentity().NodeID + "/images/" + artifact.SHA256 + "/download"
		if artifact.DownloadPath != expected {
			return fmt.Errorf("镜像下载地址不属于本节点中心资源接口")
		}
		local, err := a.hostd.CheckImages(ctx, artifact.ImageArtifact)
		if err != nil {
			return err
		}
		if !local.Ready {
			a.setImageState("downloading", ready, "正在下载运行资源")
			if err = a.downloadImage(ctx, artifact, hostd.ImageCacheDir); err != nil {
				return err
			}
			a.setImageState("importing", ready, "正在导入运行资源")
			local, err = a.hostd.ImportImages(ctx, artifact.ImageArtifact)
			if err != nil {
				return err
			}
			if !local.Ready {
				return fmt.Errorf("镜像导入未就绪")
			}
		}
		ready = append(ready, artifact.SHA256)
	}
	a.setImageState("ready", ready, "运行资源已就绪")
	return a.Heartbeat(ctx)
}

func verifyImageFile(path string, artifact ImageDownloadArtifact) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() != artifact.Size {
		return false
	}
	hash := sha256.New()
	n, err := io.Copy(hash, file)
	return err == nil && n == artifact.Size && hex.EncodeToString(hash.Sum(nil)) == artifact.SHA256
}

// downloadImage 仅访问固定中心路径，拒绝重定向；部分文件保留供下一轮 Range 续传。
func (a *Agent) downloadImage(ctx context.Context, artifact ImageDownloadArtifact, dir string) error {
	if err := artifact.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	target := filepath.Join(dir, artifact.SHA256+".tar")
	if verifyImageFile(target, artifact) {
		return nil
	}
	partial := target + ".part"
	if info, err := os.Lstat(partial); err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("镜像下载缓存不是普通文件")
	}
	file, err := os.OpenFile(partial, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	offset := info.Size()
	if offset >= artifact.Size {
		if verifyImageFile(partial, artifact) {
			return os.Rename(partial, target)
		}
		if err = file.Truncate(0); err != nil {
			return err
		}
		offset = 0
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(a.cfg.ServerURL, "/")+artifact.DownloadPath, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+a.currentIdentity().AgentToken)
	if offset > 0 {
		request.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	client := *a.client
	client.Timeout = 30 * time.Minute
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		offset = 0
		if err = file.Truncate(0); err != nil {
			return err
		}
	} else if response.StatusCode == http.StatusPartialContent {
		expected := fmt.Sprintf("bytes %d-%d/%d", offset, artifact.Size-1, artifact.Size)
		if response.Header.Get("Content-Range") != expected {
			return fmt.Errorf("中心镜像续传范围无效")
		}
	} else {
		return fmt.Errorf("中心镜像下载失败: status=%d", response.StatusCode)
	}
	if _, err = file.Seek(offset, io.SeekStart); err != nil {
		return err
	}
	n, err := io.Copy(file, io.LimitReader(response.Body, artifact.Size-offset+1))
	if err != nil {
		return fmt.Errorf("镜像下载中断，可重试续传: %w", err)
	}
	if offset+n != artifact.Size {
		return fmt.Errorf("镜像下载大小不符，可重试续传")
	}
	if err = file.Sync(); err != nil {
		return err
	}
	if !verifyImageFile(partial, artifact) {
		_ = file.Truncate(0)
		return fmt.Errorf("镜像 SHA256 校验失败")
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(partial, target)
}

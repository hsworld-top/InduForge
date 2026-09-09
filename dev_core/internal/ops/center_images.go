package ops

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/indu-forge/dev_core/internal/imagecatalog"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// 中心内置节点经挂载的 hostd socket 导入，和外部节点遵守同一摘要校验规则。
func (r *PostgreSQLRepository) prepareCenterImages(ctx context.Context, artifacts []imagecatalog.Artifact) error {
	return r.prepareCenterImagesWithProgress(ctx, artifacts, func(ImageState) {})
}
func (r *PostgreSQLRepository) prepareCenterImagesWithProgress(ctx context.Context, artifacts []imagecatalog.Artifact, notify func(ImageState)) error {
	socket := os.Getenv("IF_OPS_CENTER_HOSTD_SOCKET")
	if socket == "" {
		socket = "/run/induforge-center/hostd.sock"
	}
	cache := os.Getenv("IF_OPS_CENTER_IMAGE_CACHE")
	if cache == "" {
		cache = "/var/lib/induforge/image-cache"
	}
	client := &http.Client{Timeout: 20 * time.Minute, Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}}
	defer client.CloseIdleConnections()
	call := func(path string, a imagecatalog.Artifact) (bool, error) {
		payload := struct {
			SHA256 string               `json:"sha256"`
			Size   int64                `json:"size"`
			Images []imagecatalog.Image `json:"images"`
		}{a.SHA256, a.Size, a.Images}
		b, _ := json.Marshal(payload)
		req, e := http.NewRequestWithContext(ctx, "POST", "http://unix/v1/images/"+path, bytes.NewReader(b))
		if e != nil {
			return false, e
		}
		req.Header.Set("Content-Type", "application/json")
		resp, e := client.Do(req)
		if e != nil {
			return false, e
		}
		defer resp.Body.Close()
		var result struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
			Data struct {
				Ready bool `json:"ready"`
			} `json:"data"`
		}
		e = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result)
		if e != nil {
			return false, e
		}
		if resp.StatusCode != 200 || result.Code != 0 {
			return false, fmt.Errorf("中心运行资源准备失败: %s", result.Msg)
		}
		return result.Data.Ready, nil
	}
	for _, a := range artifacts {
		ready, e := call("check", a)
		if e != nil {
			return e
		}
		if ready {
			continue
		}
		if e = os.MkdirAll(cache, 0750); e != nil {
			return e
		}
		notify(ImageState{Status: "downloading", Message: "正在下载中心运行资源"})
		o, e := r.imageCatalog.Store.Open(ctx, imagecatalog.Key(a.SHA256))
		if e != nil {
			return e
		}
		f, e := os.CreateTemp(cache, ".download-")
		if e != nil {
			o.Reader.Close()
			return e
		}
		tmp := f.Name()
		h := sha256.New()
		n, copyErr := io.Copy(io.MultiWriter(f, h), io.LimitReader(o.Reader, a.Size+1))
		o.Reader.Close()
		closeErr := f.Close()
		if copyErr != nil || closeErr != nil || n != a.Size || hex.EncodeToString(h.Sum(nil)) != a.SHA256 {
			os.Remove(tmp)
			return fmt.Errorf("中心镜像下载校验失败: %v", copyErr)
		}
		if e = os.Rename(tmp, filepath.Join(cache, a.SHA256+".tar")); e != nil {
			os.Remove(tmp)
			return e
		}
		notify(ImageState{Status: "importing", Message: "正在导入中心运行资源"})
		ready, e = call("import", a)
		if e != nil {
			return e
		}
		if !ready {
			return fmt.Errorf("中心镜像导入后摘要不匹配")
		}
	}
	return nil
}

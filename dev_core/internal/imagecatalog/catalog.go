// Package imagecatalog 管理中心离线镜像对象；节点不接触外部镜像仓库。
package imagecatalog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/indu-forge/dev_core/internal/objectstore"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Image struct {
	Reference    string `json:"reference"`
	ConfigDigest string `json:"configDigest"`
}
type Artifact struct {
	Architecture string  `json:"architecture"`
	SHA256       string  `json:"sha256"`
	Size         int64   `json:"size"`
	Archive      string  `json:"archive,omitempty"`
	Images       []Image `json:"images"`
	DownloadPath string  `json:"downloadPath,omitempty"`
}
type Manifest struct {
	SchemaVersion int        `json:"schemaVersion"`
	Artifacts     []Artifact `json:"artifacts"`
}
type Store interface {
	Open(context.Context, string) (objectstore.ObjectReader, error)
	Put(context.Context, string, io.Reader, int64, string) (objectstore.ObjectRef, error)
}
type Catalog struct {
	Store    Store
	mu       sync.Mutex
	cached   Manifest
	cacheErr error
	expires  time.Time
}

var hexDigest = regexp.MustCompile(`^[a-f0-9]{64}$`)

const catalogKey = "node-images/catalog.json"

func Key(hash string) string { return "node-images/sha256/" + hash + ".tar" }
func (m Manifest) Validate() error {
	if m.SchemaVersion != 1 || len(m.Artifacts) == 0 {
		return fmt.Errorf("镜像目录版本或内容无效")
	}
	seen := map[string]bool{}
	for _, a := range m.Artifacts {
		if (a.Architecture != "arm64" && a.Architecture != "amd64") || !hexDigest.MatchString(a.SHA256) || a.Size <= 0 || len(a.Images) == 0 {
			return fmt.Errorf("镜像归档元数据无效")
		}
		for _, im := range a.Images {
			k := a.Architecture + "/" + im.Reference
			if strings.TrimSpace(im.Reference) == "" || !strings.HasPrefix(im.ConfigDigest, "sha256:") || !hexDigest.MatchString(strings.TrimPrefix(im.ConfigDigest, "sha256:")) || seen[k] {
				return fmt.Errorf("镜像引用或摘要无效或重复: %s", im.Reference)
			}
			seen[k] = true
		}
	}
	return nil
}
func (c *Catalog) Load(ctx context.Context) (Manifest, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if time.Now().Before(c.expires) {
		return cloneManifest(c.cached), c.cacheErr
	}
	m, e := c.load(ctx)
	c.cached = m
	c.cacheErr = e
	c.expires = time.Now().Add(5 * time.Second)
	return cloneManifest(m), e
}
func cloneManifest(m Manifest) Manifest {
	out := m
	out.Artifacts = append([]Artifact(nil), m.Artifacts...)
	for i := range out.Artifacts {
		out.Artifacts[i].Images = append([]Image(nil), m.Artifacts[i].Images...)
	}
	return out
}
func (c *Catalog) load(ctx context.Context) (Manifest, error) {
	o, e := c.Store.Open(ctx, catalogKey)
	if e != nil {
		return Manifest{}, fmt.Errorf("中心尚未导入镜像资源目录: %w", e)
	}
	defer o.Reader.Close()
	var m Manifest
	e = json.NewDecoder(io.LimitReader(o.Reader, 8<<20)).Decode(&m)
	if e == nil {
		e = m.Validate()
	}
	return m, e
}

// Import 先校验所有本地归档，再上传不可变对象，最后替换目录，避免发布半成品目录。
func (c *Catalog) Import(ctx context.Context, dir string) error {
	b, e := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if e != nil {
		return e
	}
	var m Manifest
	if e = json.Unmarshal(b, &m); e != nil {
		return e
	}
	if e = m.Validate(); e != nil {
		return e
	}
	for _, a := range m.Artifacts {
		if e = checkArchive(dir, a); e != nil {
			return e
		}
	}
	for i, a := range m.Artifacts {
		// Open 首先读取已完成对象的 Stat，不枚举或信任未完成 multipart。
		// key 已由本地内容校验确定；相同大小的已发布不可变对象可以直接复用。
		existing, openErr := c.Store.Open(ctx, Key(a.SHA256))
		if openErr == nil {
			existing.Reader.Close()
			if existing.Size == a.Size {
				m.Artifacts[i].Archive = ""
				m.Artifacts[i].DownloadPath = ""
				continue
			}
		}
		f, e := os.Open(filepath.Join(dir, a.Archive))
		if e != nil {
			return e
		}
		_, e = c.Store.Put(ctx, Key(a.SHA256), f, a.Size, "application/octet-stream")
		f.Close()
		if e != nil {
			return e
		}
		m.Artifacts[i].Archive = ""
		m.Artifacts[i].DownloadPath = ""
	}
	b, e = json.Marshal(m)
	if e != nil {
		return e
	}
	_, e = c.Store.Put(ctx, catalogKey, strings.NewReader(string(b)), int64(len(b)), "application/json")
	if e == nil {
		c.mu.Lock()
		c.expires = time.Time{}
		c.mu.Unlock()
	}
	return e
}
func checkArchive(dir string, a Artifact) error {
	if a.Archive == "" || filepath.IsAbs(a.Archive) || filepath.Clean(a.Archive) == ".." || strings.HasPrefix(filepath.Clean(a.Archive), ".."+string(filepath.Separator)) {
		return fmt.Errorf("归档路径越界")
	}
	full, e := filepath.EvalSymlinks(filepath.Join(dir, a.Archive))
	if e != nil {
		return e
	}
	root, e := filepath.EvalSymlinks(dir)
	if e != nil {
		return e
	}
	root, _ = filepath.Abs(root)
	full, _ = filepath.Abs(full)
	if !strings.HasPrefix(full, root+string(filepath.Separator)) {
		return fmt.Errorf("归档链接越界")
	}
	f, e := os.Open(full)
	if e != nil {
		return e
	}
	defer f.Close()
	h := sha256.New()
	n, e := io.Copy(h, f)
	if e != nil {
		return e
	}
	if n != a.Size || hex.EncodeToString(h.Sum(nil)) != a.SHA256 {
		return fmt.Errorf("归档大小或校验失败: %s", a.Archive)
	}
	return nil
}
func (c *Catalog) Resolve(ctx context.Context, arch string, refs []string) ([]Artifact, error) {
	m, e := c.Load(ctx)
	if e != nil {
		return nil, e
	}
	wanted := map[string]bool{}
	for _, ref := range refs {
		wanted[ref] = true
	}
	var out []Artifact
	for _, a := range m.Artifacts {
		if a.Architecture != arch {
			continue
		}
		match := false
		for _, im := range a.Images {
			if wanted[im.Reference] {
				match = true
				delete(wanted, im.Reference)
			}
		}
		if match {
			out = append(out, a)
		}
	}
	if len(wanted) > 0 {
		return nil, fmt.Errorf("中心缺少 %s 运行镜像: %v", arch, wanted)
	}
	return out, nil
}

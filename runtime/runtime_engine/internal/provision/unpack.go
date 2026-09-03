package provision

import (
	"archive/tar"
	"fmt"
	"github.com/klauspost/compress/zstd"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Limits struct {
	MaxFiles                    int
	MaxFileBytes, MaxTotalBytes int64
}

func UnpackRuntimeArtifact(src io.Reader, target string, l Limits) error {
	return UnpackArtifact(src, target, "runtime-project-artifact.json", l)
}

// UnpackArtifact 以同一安全边界解包冻结制品，并要求唯一的预期入口文件存在。
func UnpackArtifact(src io.Reader, target, requiredFile string, l Limits) error {
	if l.MaxFiles <= 0 || l.MaxFileBytes <= 0 || l.MaxTotalBytes <= 0 {
		return fmt.Errorf("解包限制无效")
	}
	parent := filepath.Dir(target)
	tmp, e := os.MkdirTemp(parent, ".runtime-unpack-")
	if e != nil {
		return e
	}
	defer os.RemoveAll(tmp)
	zr, e := zstd.NewReader(src)
	if e != nil {
		return e
	}
	defer zr.Close()
	tr := tar.NewReader(zr)
	seen := map[string]bool{}
	var total int64
	files := 0
	found := false
	for {
		h, e := tr.Next()
		if e == io.EOF {
			break
		}
		if e != nil {
			return e
		}
		n := filepath.Clean(filepath.FromSlash(h.Name))
		if h.Name == "" || filepath.IsAbs(n) || n == "." || strings.HasPrefix(n, ".."+string(filepath.Separator)) || seen[n] {
			return fmt.Errorf("归档路径非法")
		}
		seen[n] = true
		switch h.Typeflag {
		case tar.TypeDir:
			if e = os.MkdirAll(filepath.Join(tmp, n), 0700); e != nil {
				return e
			}
		case tar.TypeReg:
			files++
			if files > l.MaxFiles || h.Size < 0 || h.Size > l.MaxFileBytes || total+h.Size > l.MaxTotalBytes {
				return fmt.Errorf("归档超限")
			}
			p := filepath.Join(tmp, n)
			if e = os.MkdirAll(filepath.Dir(p), 0700); e != nil {
				return e
			}
			f, e := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if e != nil {
				return e
			}
			_, e = io.CopyN(f, tr, h.Size)
			ce := f.Close()
			if e != nil {
				return e
			}
			if ce != nil {
				return ce
			}
			total += h.Size
			if n == requiredFile {
				found = true
			}
		default:
			return fmt.Errorf("归档包含禁止条目")
		}
	}
	if !found {
		return fmt.Errorf("归档缺少 %s", requiredFile)
	}
	if e = os.RemoveAll(target); e != nil {
		return e
	}
	return os.Rename(tmp, target)
}

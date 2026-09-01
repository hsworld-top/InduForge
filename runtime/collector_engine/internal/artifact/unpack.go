// Package artifact 安全解开 Collector Release 子工件，并固定其唯一入口文件。
package artifact

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
)

const EntryFile = "collector-runtime-artifact.json"

type Limits struct {
	MaxFiles                    int
	MaxFileBytes, MaxTotalBytes int64
}

// Unpack 将已验签归档原子写入 target。只接受常规文件和目录，并强制要求 collector
// Artifact 位于归档根目录，避免 Runtime Artifact 的入口契约被错误复用。
func Unpack(src io.Reader, target string, limits Limits) error {
	if limits.MaxFiles <= 0 || limits.MaxFileBytes <= 0 || limits.MaxTotalBytes <= 0 {
		return fmt.Errorf("解包限制无效")
	}
	tmp, err := os.MkdirTemp(filepath.Dir(target), ".collector-unpack-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	zr, err := zstd.NewReader(src)
	if err != nil {
		return err
	}
	defer zr.Close()
	tr, seen := tar.NewReader(zr), map[string]bool{}
	var total int64
	files, found := 0, false
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		name := filepath.Clean(filepath.FromSlash(header.Name))
		if header.Name == "" || filepath.IsAbs(name) || name == "." || strings.HasPrefix(name, ".."+string(filepath.Separator)) || seen[name] {
			return fmt.Errorf("归档路径非法")
		}
		seen[name] = true
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(filepath.Join(tmp, name), 0o700); err != nil {
				return err
			}
		case tar.TypeReg:
			files++
			if files > limits.MaxFiles || header.Size < 0 || header.Size > limits.MaxFileBytes || total+header.Size > limits.MaxTotalBytes {
				return fmt.Errorf("归档超限")
			}
			path := filepath.Join(tmp, name)
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				return err
			}
			file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
			if err != nil {
				return err
			}
			_, copyErr := io.CopyN(file, tr, header.Size)
			closeErr := file.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
			total += header.Size
			found = found || name == EntryFile
		default:
			return fmt.Errorf("归档包含禁止条目")
		}
	}
	if !found {
		return fmt.Errorf("归档缺少 %s", EntryFile)
	}
	if err := os.RemoveAll(target); err != nil {
		return err
	}
	return os.Rename(tmp, target)
}

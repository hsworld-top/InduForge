package securefile

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ReadSecret 接受普通文件，或 Kubernetes Secret AtomicWriter 的受控两级链接。
// 最终目标必须仍位于声明文件的挂载根内，且读取前后保持同一普通文件。
func ReadSecret(path string, limit int64) ([]byte, error) {
	if limit <= 0 {
		return nil, errors.New("secret 大小限制无效")
	}
	root, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	entry, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	resolved := path
	if entry.Mode()&os.ModeSymlink != 0 {
		link, readErr := os.Readlink(path)
		expected := filepath.Join("..data", filepath.Base(path))
		if readErr != nil || filepath.Clean(link) != expected {
			return nil, errors.New("secret 入口不是受控 AtomicWriter 链接")
		}
		dataLink := filepath.Join(root, "..data")
		dataInfo, statErr := os.Lstat(dataLink)
		if statErr != nil || dataInfo.Mode()&os.ModeSymlink == 0 {
			return nil, errors.New("secret ..data 不是受控链接")
		}
		dataTarget, readErr := os.Readlink(dataLink)
		if readErr != nil || filepath.IsAbs(dataTarget) || strings.Contains(filepath.ToSlash(filepath.Clean(dataTarget)), "/") || !strings.HasPrefix(dataTarget, "..") {
			return nil, errors.New("secret ..data 目标非法")
		}
	}
	resolved, err = filepath.EvalSymlinks(path)
	if err != nil {
		return nil, errors.New("secret 文件无法解析")
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return nil, err
	}
	relative, err := filepath.Rel(canonicalRoot, resolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return nil, errors.New("secret 链接逃逸挂载根")
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > limit {
		return nil, errors.New("secret 目标必须是受限普通文件")
	}
	if info.Mode().Perm()&0o077 != 0 {
		return nil, errors.New("secret 不能被所属组或其他用户读取")
	}
	file, err := os.Open(resolved)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) || !opened.Mode().IsRegular() {
		return nil, errors.New("secret 在校验期间发生变化")
	}
	payload, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || int64(len(payload)) > limit {
		return nil, fmt.Errorf("读取 secret 失败或超限")
	}
	return payload, nil
}

//go:build !windows

package ops

import "os"

// syncDirectory 把 rename 的目录项同步到持久介质，避免断电后出现已返回成功但未持久化的 current。
func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func replaceFileAtomically(source, destination string) error {
	return os.Rename(source, destination)
}

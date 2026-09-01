//go:build windows

package ops

import "golang.org/x/sys/windows"

// Windows 的目录句柄不能以 os.File.Sync 便携打开；current.json 通过
// MoveFileEx(REPLACE_EXISTING|WRITE_THROUGH) 在同卷完成替换并请求落盘。
func syncDirectory(string) error { return nil }

func replaceFileAtomically(source, destination string) error {
	sourcePath, err := windows.UTF16PtrFromString(source)
	if err != nil {
		return err
	}
	destinationPath, err := windows.UTF16PtrFromString(destination)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(
		sourcePath,
		destinationPath,
		windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH,
	)
}

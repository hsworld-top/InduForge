package loader

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// mountReadOnly 从 Linux mountinfo 选择覆盖目标路径的最具体挂载，并检查 VFS ro 选项。
func mountReadOnly(target, mountInfo string) error {
	target = filepath.Clean(target)
	best := ""
	bestReadOnly := false
	for _, line := range strings.Split(mountInfo, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 7 || fields[0] == "" {
			continue
		}
		separator := -1
		for index, field := range fields {
			if field == "-" {
				separator = index
				break
			}
		}
		if separator < 6 {
			continue
		}
		mountPoint, err := unescapeMountPath(fields[4])
		if err != nil {
			return fmt.Errorf("mountinfo 挂载路径无效: %w", err)
		}
		if !pathWithin(mountPoint, target) {
			continue
		}
		if len(mountPoint) > len(best) {
			best = mountPoint
			bestReadOnly = mountOptionsReadOnly(fields[5])
		}
	}
	if best == "" {
		return fmt.Errorf("mountinfo 未找到覆盖路径 %q 的挂载", target)
	}
	if !bestReadOnly {
		return fmt.Errorf("覆盖路径 %q 的最具体挂载 %q 不是只读", target, best)
	}
	return nil
}

func mountOptionsReadOnly(options string) bool {
	for _, option := range strings.Split(options, ",") {
		if option == "ro" {
			return true
		}
	}
	return false
}

func pathWithin(root, target string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(target))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func unescapeMountPath(value string) (string, error) {
	for index := 0; index < len(value); index++ {
		if value[index] != '\\' {
			continue
		}
		if index+3 >= len(value) {
			return "", fmt.Errorf("不完整转义")
		}
		parsed, err := strconv.ParseUint(value[index+1:index+4], 8, 8)
		if err != nil {
			return "", err
		}
		value = value[:index] + string(rune(parsed)) + value[index+4:]
	}
	return value, nil
}

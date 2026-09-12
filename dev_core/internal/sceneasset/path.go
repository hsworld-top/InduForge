package sceneasset

import (
	"path"
	"regexp"
	"strings"
)

var sceneIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,127}$`)

var reservedRoots = map[string]struct{}{
	".workspace": {},
	".induforge": {}, // 兼容旧工程路径，禁止写入新工程。
	".git":       {},
	"runtime":    {},
}

func ValidateSceneID(sceneID string) error {
	if !sceneIDPattern.MatchString(strings.TrimSpace(sceneID)) {
		return ErrInvalidSceneID
	}
	return nil
}

func NormalizePath(value string, allowEmpty bool) (string, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, `\`, "/"))
	if strings.ContainsRune(value, 0) || strings.HasPrefix(value, "/") {
		return "", ErrInvalidPath
	}
	cleaned := path.Clean(value)
	if cleaned == "." {
		if allowEmpty {
			return "", nil
		}
		return "", ErrInvalidPath
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") || cleaned != value {
		return "", ErrInvalidPath
	}
	root := strings.Split(cleaned, "/")[0]
	if _, protected := reservedRoots[strings.ToLower(root)]; protected {
		return "", ErrProtectedPath
	}
	return cleaned, nil
}

func entryPath(kind, sceneID, requested string) (string, error) {
	if requested == "" {
		if kind == "2d" {
			requested = "displays/" + sceneID + ".json"
		} else {
			requested = "scenes/" + sceneID + ".json"
		}
	}
	return NormalizePath(requested, false)
}

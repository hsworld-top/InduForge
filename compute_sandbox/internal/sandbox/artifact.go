package sandbox

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

var utcRFC3339Pattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$`)

// runtimeArtifact 仅保留执行边界所需字段；解析时拒绝未知字段，避免把未校验配置带入执行器。
type runtimeArtifact struct {
	SchemaVersion          string                `json:"schemaVersion"`
	ProjectArtifactVersion string                `json:"projectArtifactVersion"`
	ProjectID              string                `json:"projectId"`
	GeneratedAt            *string               `json:"generatedAt"`
	DataPoints             json.RawMessage       `json:"dataPoints"`
	ComputeUnits           []artifactComputeUnit `json:"computeUnits"`
	AlarmItems             json.RawMessage       `json:"alarmItems"`
}

type artifactComputeUnit struct {
	ID           string           `json:"id"`
	Revision     int              `json:"revision"`
	Name         string           `json:"name"`
	Description  json.RawMessage  `json:"description"`
	Language     string           `json:"language"`
	ScriptCode   string           `json:"scriptCode"`
	Enabled      bool             `json:"enabled"`
	TimeoutMS    int64            `json:"timeoutMs"`
	Dependencies []dependencySpec `json:"dependencies"`
	Inputs       json.RawMessage  `json:"inputs"`
	Outputs      json.RawMessage  `json:"outputs"`
	Trigger      json.RawMessage  `json:"trigger"`
}

type verifiedArtifact struct {
	units map[string]artifactComputeUnit
}

func loadVerifiedArtifact(config Config) (*verifiedArtifact, error) {
	path, err := secureArtifactFilePath(config.ArtifactRoot, config.ArtifactFile)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("artifact unavailable")
	}
	digest := sha256.Sum256(raw)
	if config.ArtifactDigest != fmt.Sprintf("sha256:%x", digest[:]) {
		return nil, fmt.Errorf("artifact digest mismatch")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var artifact runtimeArtifact
	if err := decoder.Decode(&artifact); err != nil {
		return nil, fmt.Errorf("artifact invalid")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, fmt.Errorf("artifact invalid")
	}
	if artifact.SchemaVersion != "runtime-project-artifact.v1" || artifact.ProjectArtifactVersion != "1.0" ||
		!canonicalUUID(artifact.ProjectID) || artifact.ProjectID != config.ProjectID || artifact.GeneratedAt == nil || artifact.DataPoints == nil || artifact.AlarmItems == nil {
		return nil, fmt.Errorf("artifact identity invalid")
	}
	if !utcRFC3339Pattern.MatchString(*artifact.GeneratedAt) {
		return nil, fmt.Errorf("artifact generatedAt invalid")
	}
	if _, err := time.Parse(time.RFC3339Nano, *artifact.GeneratedAt); err != nil {
		return nil, fmt.Errorf("artifact generatedAt invalid")
	}
	units := make(map[string]artifactComputeUnit, len(artifact.ComputeUnits))
	for _, unit := range artifact.ComputeUnits {
		if !canonicalUUID(unit.ID) || unit.Revision < 1 || strings.TrimSpace(unit.Name) == "" || len(unit.Description) == 0 ||
			(unit.Language != "js" && unit.Language != "python") || strings.TrimSpace(unit.ScriptCode) == "" ||
			unit.TimeoutMS < 1 || unit.TimeoutMS > maxTimeoutMS || unit.Dependencies == nil || unit.Inputs == nil || unit.Outputs == nil || unit.Trigger == nil {
			return nil, fmt.Errorf("artifact compute unit invalid")
		}
		if err := validateExecuteDependencies(unit.Language, unit.Dependencies); err != nil {
			return nil, fmt.Errorf("artifact dependency invalid")
		}
		if _, exists := units[unit.ID]; exists {
			return nil, fmt.Errorf("artifact compute unit duplicate")
		}
		units[unit.ID] = unit
	}
	return &verifiedArtifact{units: units}, nil
}

func artifactFilePath(root, file string) (string, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	file = strings.TrimSpace(file)
	if root == "." || root == "" || !filepath.IsAbs(root) || file == "" || filepath.IsAbs(file) || strings.Contains(file, "\\") {
		return "", fmt.Errorf("artifact path invalid")
	}
	joined := filepath.Clean(filepath.Join(root, file))
	rel, err := filepath.Rel(root, joined)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("artifact path invalid")
	}
	return joined, nil
}

func secureArtifactFilePath(root, file string) (string, error) {
	path, err := artifactFilePath(root, file)
	if err != nil {
		return "", err
	}
	rootInfo, err := os.Lstat(filepath.Clean(root))
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("artifact path invalid")
	}
	rel, _ := filepath.Rel(filepath.Clean(root), path)
	current := filepath.Clean(root)
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("artifact path invalid")
		}
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("artifact path invalid")
	}
	return path, nil
}

func canonicalUUID(value string) bool {
	return runtimeUUIDPattern.MatchString(value) && value == strings.ToLower(value)
}

func runtimeMountsReadOnly(config Config) bool {
	if runtime.GOOS != "linux" {
		return false
	}
	raw, err := os.ReadFile("/proc/self/mountinfo")
	artifact, err := secureArtifactFilePath(config.ArtifactRoot, config.ArtifactFile)
	if err != nil {
		return false
	}
	dependencies, err := filepath.EvalSymlinks(config.DependenciesDir)
	if err != nil || !filepath.IsAbs(dependencies) {
		return false
	}
	return mountPathReadOnly(string(raw), artifact) && mountPathReadOnly(string(raw), dependencies)
}

// mountPathReadOnly 选择覆盖目标路径的最具体 mount，并只接受明确 ro 的挂载选项。
func mountPathReadOnly(mountinfo, target string) bool {
	target = filepath.Clean(target)
	best := ""
	readOnly := false
	for _, line := range strings.Split(mountinfo, "\n") {
		parts := strings.Split(line, " - ")
		if len(parts) != 2 {
			continue
		}
		fields := strings.Fields(parts[0])
		if len(fields) < 6 {
			continue
		}
		mountpoint := filepath.Clean(decodeMountPath(fields[4]))
		contained := mountpoint == "/" || target == mountpoint || strings.HasPrefix(target, mountpoint+string(filepath.Separator))
		if !contained {
			continue
		}
		if len(mountpoint) < len(best) {
			continue
		}
		best = mountpoint
		readOnly = strings.Contains(","+fields[5]+",", ",ro,")
	}
	return best != "" && readOnly
}

func decodeMountPath(value string) string {
	replacer := strings.NewReplacer(`\040`, " ", `\011`, "\t", `\012`, "\n", `\134`, `\\`)
	return replacer.Replace(value)
}

package artifact

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
)

const maxArtifactBytes = 16 << 20

var canonicalUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

type WritePermission struct {
	AllowRoles []string `json:"allowRoles"`
	DenyRoles  []string `json:"denyRoles"`
	Inherit    bool     `json:"inherit"`
}

type RuntimePermissions struct {
	Write WritePermission `json:"write"`
}

type Point struct {
	ID                 string             `json:"id"`
	Path               string             `json:"path"`
	Name               string             `json:"name"`
	DataType           string             `json:"dataType"`
	SourceType         string             `json:"sourceType"`
	SourceID           *string            `json:"sourceId"`
	RuntimePermissions RuntimePermissions `json:"runtimePermissions"`
	RefreshMode        string             `json:"refreshMode"`
	Status             string             `json:"status"`
	Unit               *string            `json:"unit"`
	Precision          *int               `json:"precisionNum"`
	DefaultValue       any                `json:"defaultValue"`
	Tags               []any              `json:"tags"`
	Attributes         map[string]string  `json:"attributeDefaults"`
}

type ComputeUnit struct {
	ID          string          `json:"id"`
	Revision    int64           `json:"revision"`
	Name        string          `json:"name"`
	Description *string         `json:"description"`
	Enabled     bool            `json:"enabled"`
	Inputs      json.RawMessage `json:"inputs"`
	Outputs     json.RawMessage `json:"outputs"`
	Trigger     json.RawMessage `json:"trigger"`
}

type AlarmItem struct {
	ID          string          `json:"id"`
	Revision    int64           `json:"revision"`
	Name        string          `json:"name"`
	Description *string         `json:"description"`
	Enabled     bool            `json:"enabled"`
	Mode        string          `json:"mode"`
	Inputs      json.RawMessage `json:"inputs"`
	Conditions  json.RawMessage `json:"conditions"`
}

type rawCatalog struct {
	SchemaVersion          string        `json:"schemaVersion"`
	ProjectArtifactVersion string        `json:"projectArtifactVersion"`
	ProjectID              string        `json:"projectId"`
	Points                 []Point       `json:"dataPoints"`
	Computes               []ComputeUnit `json:"computeUnits"`
	Alarms                 []AlarmItem   `json:"alarmItems"`
}

type Catalog struct {
	SchemaVersion string
	ProjectID     string
	Digest        string
	pointsByPath  map[string]Point
	pointsByID    map[string]Point
	computesByID  map[string]ComputeUnit
	alarms        []AlarmItem
}

func Load(path, expectedProjectID string) (*Catalog, error) {
	file, err := openRegularNoFollow(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	payload, err := io.ReadAll(io.LimitReader(file, maxArtifactBytes+1))
	if err != nil {
		return nil, fmt.Errorf("读取 Runtime Artifact: %w", err)
	}
	if len(payload) == 0 || len(payload) > maxArtifactBytes {
		return nil, errors.New("Runtime Artifact 为空或超过 16MiB")
	}
	var raw rawCatalog
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, fmt.Errorf("解析 Runtime Artifact: %w", err)
	}
	if raw.SchemaVersion != "runtime-project-artifact.v1" || raw.ProjectArtifactVersion != "1.0" {
		return nil, errors.New("Runtime Artifact 版本不受支持")
	}
	if !canonicalUUID.MatchString(raw.ProjectID) || raw.ProjectID != strings.TrimSpace(expectedProjectID) {
		return nil, errors.New("Runtime Artifact projectId 与部署不一致")
	}
	digest := sha256.Sum256(payload)
	catalog := &Catalog{
		SchemaVersion: raw.SchemaVersion,
		ProjectID:     raw.ProjectID,
		Digest:        "sha256:" + hex.EncodeToString(digest[:]),
		pointsByPath:  make(map[string]Point, len(raw.Points)),
		pointsByID:    make(map[string]Point, len(raw.Points)),
		computesByID:  make(map[string]ComputeUnit, len(raw.Computes)),
		alarms:        append([]AlarmItem(nil), raw.Alarms...),
	}
	for _, point := range raw.Points {
		point.Path = strings.TrimSpace(point.Path)
		if !canonicalUUID.MatchString(point.ID) || point.Path == "" || len(point.Path) > 512 || strings.ContainsAny(point.Path, "\x00/\\") {
			return nil, fmt.Errorf("Runtime Artifact 数据点身份非法: %q", point.Path)
		}
		if _, exists := catalog.pointsByPath[point.Path]; exists {
			return nil, fmt.Errorf("Runtime Artifact 数据点 path 重复: %s", point.Path)
		}
		if _, exists := catalog.pointsByID[point.ID]; exists {
			return nil, fmt.Errorf("Runtime Artifact 数据点 id 重复: %s", point.ID)
		}
		if point.Attributes == nil {
			point.Attributes = map[string]string{}
		}
		if point.Tags == nil {
			point.Tags = []any{}
		}
		catalog.pointsByPath[point.Path], catalog.pointsByID[point.ID] = point, point
	}
	for _, compute := range raw.Computes {
		if !canonicalUUID.MatchString(compute.ID) || compute.Revision < 1 || strings.TrimSpace(compute.Name) == "" {
			return nil, errors.New("Runtime Artifact 计算单元身份非法")
		}
		if _, exists := catalog.computesByID[compute.ID]; exists {
			return nil, fmt.Errorf("Runtime Artifact 计算单元 id 重复: %s", compute.ID)
		}
		catalog.computesByID[compute.ID] = compute
	}
	return catalog, nil
}

func (c *Catalog) PointByPath(path string) (Point, bool) {
	point, ok := c.pointsByPath[path]
	return point, ok
}

func (c *Catalog) PointByID(id string) (Point, bool) {
	point, ok := c.pointsByID[id]
	return point, ok
}

func (c *Catalog) ComputeByID(id string) (ComputeUnit, bool) {
	compute, ok := c.computesByID[id]
	return compute, ok
}

func (c *Catalog) Points() []Point {
	result := make([]Point, 0, len(c.pointsByPath))
	for _, point := range c.pointsByPath {
		result = append(result, point)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result
}

func (c *Catalog) Computes() []ComputeUnit {
	result := make([]ComputeUnit, 0, len(c.computesByID))
	for _, compute := range c.computesByID {
		result = append(result, compute)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func (c *Catalog) Alarms() []AlarmItem { return append([]AlarmItem(nil), c.alarms...) }

func openRegularNoFollow(path string) (*os.File, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("读取 Runtime Artifact: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("Runtime Artifact 必须是普通文件且不能是符号链接")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) || !opened.Mode().IsRegular() {
		file.Close()
		return nil, errors.New("Runtime Artifact 在校验期间发生变化")
	}
	return file, nil
}

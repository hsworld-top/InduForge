// Package loader 严格加载采集器不可变 Artifact、部署 Binding 和受信 resolver index。
package loader

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const maxFileBytes = 16 << 20

//go:embed schemas/runtime/*.schema.json
var schemaFiles embed.FS

type Acquisition struct {
	IntervalMS int64   `json:"intervalMs"`
	Deadband   float64 `json:"deadband"`
	ChangeOnly bool    `json:"changeOnly"`
}
type Connection struct {
	ConnectionID       string      `json:"connectionId"`
	ProtocolFamily     string      `json:"protocolFamily"`
	DriverID           string      `json:"driverId"`
	DriverVersion      string      `json:"driverVersion"`
	SchemaVersion      int         `json:"schemaVersion"`
	Enabled            bool        `json:"enabled"`
	RequiredOperations []string    `json:"requiredOperations"`
	DefaultAcquisition Acquisition `json:"defaultAcquisition"`
}
type Mapping struct {
	DatapointID          string          `json:"datapointId"`
	ConnectionID         string          `json:"connectionId"`
	VariableID           string          `json:"variableId"`
	DataType             string          `json:"dataType"`
	ElementCount         int             `json:"elementCount"`
	AddressSchemaVersion int             `json:"addressSchemaVersion"`
	Address              json.RawMessage `json:"address"`
	ReadOptions          json.RawMessage `json:"readOptions"`
	AcquisitionMode      string          `json:"acquisitionMode"`
	EffectiveAcquisition Acquisition     `json:"effectiveAcquisition"`
	Enabled              bool            `json:"enabled"`
}
type Artifact struct {
	SchemaVersion    string       `json:"schemaVersion"`
	ArtifactID       string       `json:"artifactId"`
	ArtifactRevision int64        `json:"artifactRevision"`
	ProjectID        string       `json:"projectId"`
	CollectorVersion string       `json:"collectorVersion"`
	Connections      []Connection `json:"connections"`
	PointMappings    []Mapping    `json:"pointMappings"`
	WAL              WAL          `json:"wal"`
}
type WAL struct {
	FSync        string `json:"fsync"`
	Checksum     string `json:"checksum"`
	CommitMarker string `json:"commitMarker"`
}
type ArtifactRef struct {
	ArtifactID       string `json:"artifactId"`
	ArtifactRevision int64  `json:"artifactRevision"`
	ArtifactDigest   string `json:"artifactDigest"`
}
type Ownership struct {
	OwnerID string `json:"ownerId"`
	Epoch   int64  `json:"epoch"`
}
type BindingConnection struct {
	ConnectionID string   `json:"connectionId"`
	ResourceRef  string   `json:"resourceRef"`
	SecretRefs   []string `json:"secretRefs"`
}
type Binding struct {
	SchemaVersion   string              `json:"schemaVersion"`
	BindingID       string              `json:"bindingId"`
	BindingRevision int64               `json:"bindingRevision"`
	DeploymentID    string              `json:"deploymentId"`
	AccountID       string              `json:"accountId"`
	Artifact        ArtifactRef         `json:"artifact"`
	CollectorID     string              `json:"collectorId"`
	Ownership       Ownership           `json:"ownership"`
	Connections     []BindingConnection `json:"connections"`
	NATS            struct {
		ServerResourceRef   string `json:"serverResourceRef"`
		CredentialSecretRef string `json:"credentialSecretRef"`
		RawSubjectPrefix    string `json:"rawSubjectPrefix"`
	} `json:"nats"`
	WALCapacity struct {
		MaxBytes               int64  `json:"maxBytes"`
		HighWatermarkBytes     int64  `json:"highWatermarkBytes"`
		DiagnosticReserveBytes int64  `json:"diagnosticReserveBytes"`
		DataGapPolicy          string `json:"dataGapPolicy"`
	} `json:"walCapacity"`
}
type Index struct {
	SchemaVersion string                     `json:"schemaVersion"`
	Resources     map[string]json.RawMessage `json:"resources"`
	Secrets       map[string]string          `json:"secrets"`
}
type Loaded struct {
	Artifact Artifact
	Binding  Binding
	Index    Index
	IndexDir string
}

func Load(artifactPath, bindingPath, indexPath string) (*Loaded, error) {
	artifactRaw, err := secureRead(artifactPath)
	if err != nil {
		return nil, stageError("artifact-read")
	}
	bindingRaw, err := secureRead(bindingPath)
	if err != nil {
		return nil, stageError("binding-read")
	}
	indexRaw, err := secureRead(indexPath)
	if err != nil {
		return nil, stageError("resolver-index-read")
	}
	schemas, err := compileSchemas()
	if err != nil {
		return nil, err
	}
	if err = validate(schemas["collector-runtime-artifact.schema.json"], artifactRaw); err != nil {
		return nil, stageError("artifact-schema")
	}
	if err = validate(schemas["collector-runtime-binding.schema.json"], bindingRaw); err != nil {
		return nil, stageError("binding-schema")
	}
	var out Loaded
	if err = strictDecode(artifactRaw, &out.Artifact); err != nil {
		return nil, stageError("artifact-strict-decode")
	}
	if err = strictDecode(bindingRaw, &out.Binding); err != nil {
		return nil, stageError("binding-strict-decode")
	}
	if err = strictDecode(indexRaw, &out.Index); err != nil || out.Index.SchemaVersion != "collector-runtime-index.v1" || out.Index.Resources == nil || out.Index.Secrets == nil {
		return nil, stageError("resolver-index-strict-decode")
	}
	if err = validateCross(&out, artifactRaw); err != nil {
		return nil, err
	}
	out.IndexDir = filepath.Dir(indexPath)
	return &out, nil
}

func validateCross(v *Loaded, artifactRaw []byte) error {
	d := sha256.Sum256(artifactRaw)
	if v.Binding.Artifact.ArtifactID != v.Artifact.ArtifactID || v.Binding.Artifact.ArtifactRevision != v.Artifact.ArtifactRevision || v.Binding.Artifact.ArtifactDigest != "sha256:"+hex.EncodeToString(d[:]) {
		return stageError("artifact-identity")
	}
	if v.Binding.WALCapacity.HighWatermarkBytes >= v.Binding.WALCapacity.MaxBytes || v.Binding.WALCapacity.DiagnosticReserveBytes >= v.Binding.WALCapacity.MaxBytes {
		return stageError("wal-capacity")
	}
	connections := map[string]struct{}{}
	for _, c := range v.Artifact.Connections {
		if !c.Enabled {
			continue
		}
		if c.DriverID != "modbus.tcp" && c.DriverID != "opcua.standard" {
			return stageError("driver-allowlist")
		}
		connections[c.ConnectionID] = struct{}{}
	}
	for _, c := range v.Binding.Connections {
		if _, ok := connections[c.ConnectionID]; !ok {
			return stageError("connection-reference")
		}
		if _, ok := v.Index.Resources[c.ResourceRef]; !ok {
			return stageError("connection-resource-reference")
		}
		for _, s := range c.SecretRefs {
			if !safeRef(s, "secret://") || !safeRelative(v.Index.Secrets[s]) {
				return stageError("connection-secret-reference")
			}
		}
	}
	if _, ok := v.Index.Resources[v.Binding.NATS.ServerResourceRef]; !ok || !safeRelative(v.Index.Secrets[v.Binding.NATS.CredentialSecretRef]) {
		return stageError("nats-reference")
	}
	seenPoint, seenSource := map[string]bool{}, map[string]bool{}
	for _, m := range v.Artifact.PointMappings {
		if !m.Enabled {
			continue
		}
		if _, ok := connections[m.ConnectionID]; !ok || seenPoint[m.DatapointID] || seenSource[m.ConnectionID+"/"+m.VariableID] {
			return stageError("mapping-reference")
		}
		seenPoint[m.DatapointID] = true
		seenSource[m.ConnectionID+"/"+m.VariableID] = true
	}
	return nil
}

// stageError 仅携带固定阶段码，供启动日志定位，不回显制品、连接或 Secret 内容。
func stageError(stage string) error { return fmt.Errorf("collector-loader:%s", stage) }
func safeRef(s, prefix string) bool {
	return strings.HasPrefix(s, prefix) && len(s) > len(prefix) && !strings.ContainsAny(s, "\\\x00")
}
func safeRelative(s string) bool {
	return s != "" && !filepath.IsAbs(s) && filepath.Clean(s) == s && !strings.HasPrefix(s, "../") && s != "." && s != ".." && !strings.ContainsAny(s, "\\\x00")
}
func secureRead(path string) ([]byte, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, errors.New("路径必须为规范绝对路径")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, errors.New("必须是普通文件")
	}
	b, err := os.ReadFile(path)
	if err != nil || len(b) > maxFileBytes {
		return nil, errors.New("文件不可读或过大")
	}
	return b, nil
}
func strictDecode(raw []byte, target any) error {
	if duplicateKeys(raw) {
		return errors.New("重复键")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return err
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		return errors.New("额外 JSON 内容")
	}
	return nil
}
func validate(schema *jsonschema.Schema, raw []byte) error {
	var value any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if err := d.Decode(&value); err != nil {
		return err
	}
	if err := schema.Validate(value); err != nil {
		return err
	}
	return nil
}
func compileSchemas() (map[string]*jsonschema.Schema, error) {
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft2020)
	names := []string{"common.schema.json", "collector-runtime-artifact.schema.json", "collector-runtime-binding.schema.json", "point-event.schema.json"}
	for _, n := range names {
		b, e := schemaFiles.ReadFile("schemas/runtime/" + n)
		if e != nil {
			return nil, e
		}
		var v any
		if e = json.Unmarshal(b, &v); e != nil {
			return nil, e
		}
		if e = c.AddResource("https://induforge.dev/contracts/runtime/"+n, v); e != nil {
			return nil, e
		}
	}
	out := map[string]*jsonschema.Schema{}
	for _, n := range names[1:] {
		s, e := c.Compile("https://induforge.dev/contracts/runtime/" + n)
		if e != nil {
			return nil, e
		}
		out[n] = s
	}
	return out, nil
}
func duplicateKeys(raw []byte) bool {
	d := json.NewDecoder(bytes.NewReader(raw))
	var walk func() bool
	walk = func() bool {
		tok, e := d.Token()
		if e != nil {
			return true
		}
		delim, ok := tok.(json.Delim)
		if !ok {
			return false
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				k, e := d.Token()
				if e != nil {
					return true
				}
				key, ok := k.(string)
				if !ok || seen[key] {
					return true
				}
				seen[key] = true
				if walk() {
					return true
				}
			}
			_, e = d.Token()
			return e != nil
		case '[':
			for d.More() {
				if walk() {
					return true
				}
			}
			_, e = d.Token()
			return e != nil
		default:
			return true
		}
	}
	if walk() {
		return true
	}
	return d.Decode(&struct{}{}) != io.EOF
}

package loader

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed schemas/runtime/*.schema.json
var schemaFiles embed.FS

type schemaSet struct {
	engineConfigV1    *jsonschema.Schema
	engineConfigV2    *jsonschema.Schema
	projectArtifact   *jsonschema.Schema
	collectorArtifact *jsonschema.Schema
	all               map[string]*jsonschema.Schema
}

var (
	schemasOnce  sync.Once
	schemasCache *schemaSet
	schemasErr   error
)

func compileSchemas() (*schemaSet, error) {
	schemasOnce.Do(func() { schemasCache, schemasErr = compileSchemasOnce() })
	return schemasCache, schemasErr
}

func compileSchemasOnce() (*schemaSet, error) {
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	entries, err := fs.ReadDir(schemaFiles, "schemas/runtime")
	if err != nil {
		return nil, fmt.Errorf("读取内嵌 Runtime Schema 失败: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		bytes, err := schemaFiles.ReadFile("schemas/runtime/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("读取内嵌 Schema %s 失败: %w", entry.Name(), err)
		}
		document, err := strictSchemaJSON(bytes)
		if err != nil {
			return nil, fmt.Errorf("解析内嵌 Schema %s 失败: %w", entry.Name(), err)
		}
		if err := compiler.AddResource("https://induforge.dev/contracts/runtime/"+entry.Name(), document); err != nil {
			return nil, fmt.Errorf("注册内嵌 Schema %s 失败: %w", entry.Name(), err)
		}
	}
	all := make(map[string]*jsonschema.Schema, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "common.schema.json" {
			continue
		}
		uri := "https://induforge.dev/contracts/runtime/" + entry.Name()
		schema, err := compiler.Compile(uri)
		if err != nil {
			return nil, fmt.Errorf("编译 %s 失败: %w", entry.Name(), err)
		}
		all[entry.Name()] = schema
	}
	return &schemaSet{engineConfigV1: all["runtime-engine-config.schema.json"], engineConfigV2: all["runtime-engine-config-v2.schema.json"], projectArtifact: all["runtime-project-artifact.schema.json"], collectorArtifact: all["collector-runtime-artifact.schema.json"], all: all}, nil
}

// engineConfigSchema 先读取最小版本头，再选择对应冻结契约。v1 与 v2 绝不互相降级。
func (s *schemaSet) engineConfigSchema(raw []byte) (*jsonschema.Schema, error) {
	value, err := strictSchemaJSON(raw)
	if err != nil {
		return nil, fmt.Errorf("runtime-engine-config 不是合法 JSON: %w", err)
	}
	document, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("runtime-engine-config 必须是 JSON 对象")
	}
	version, ok := document["schemaVersion"].(string)
	if !ok {
		return nil, fmt.Errorf("runtime-engine-config 缺少字符串 schemaVersion")
	}
	switch version {
	case "runtime-engine.config.v1":
		return s.engineConfigV1, nil
	case "runtime-engine.config.v2":
		return s.engineConfigV2, nil
	default:
		return nil, fmt.Errorf("不支持的 runtime-engine config schemaVersion %q", version)
	}
}

func validateJSON(schema *jsonschema.Schema, raw []byte, name string) error {
	value, err := strictSchemaJSON(raw)
	if err != nil {
		return fmt.Errorf("%s 不是合法 JSON: %w", name, err)
	}
	if err := schema.Validate(value); err != nil {
		return fmt.Errorf("%s 未通过 Runtime Schema 校验: %w", name, err)
	}
	return nil
}

func strictSchemaJSON(raw []byte) (any, error) {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("存在额外 JSON 内容")
		}
		return nil, err
	}
	return value, nil
}

// ValidatePointEvent 是 ingress 使用的同一份内嵌契约校验入口。调用者必须先保留原始 body，
// 并且不得以反序列化后的对象替换 raw bytes。
func ValidatePointEvent(raw []byte) error {
	if err := rejectDuplicateJSONKeys(raw); err != nil {
		return fmt.Errorf("point event JSON 键无效: %w", err)
	}
	schemas, err := compileSchemas()
	if err != nil {
		return err
	}
	return validateJSON(schemas.all["point-event.schema.json"], raw, "point-event")
}

// ValidateDLQEvent 确保最终写入 outbox 的失败证据始终符合冻结的 runtime.dlq.event.v1。
func ValidateDLQEvent(raw []byte) error {
	if err := rejectDuplicateJSONKeys(raw); err != nil {
		return fmt.Errorf("DLQ JSON 键无效: %w", err)
	}
	schemas, err := compileSchemas()
	if err != nil {
		return err
	}
	return validateJSON(schemas.all["runtime-dlq-event.schema.json"], raw, "runtime-dlq-event")
}

// ValidateAlarmEvent 供 alarm 输出端在写入 outbox 前复用冻结的 alarm.event.v1 Schema。
func ValidateAlarmEvent(raw []byte) error {
	if err := rejectDuplicateJSONKeys(raw); err != nil {
		return fmt.Errorf("alarm event JSON 键无效: %w", err)
	}
	schemas, err := compileSchemas()
	if err != nil {
		return err
	}
	return validateJSON(schemas.all["alarm-event.schema.json"], raw, "alarm-event")
}

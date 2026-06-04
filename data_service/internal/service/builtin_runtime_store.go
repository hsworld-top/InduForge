package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

const builtinStoreDDLVersion = "2026-05-24.1"

type builtinStoreNormalizedInput struct {
	Name     string
	Type     string
	Category string
	Status   string
	Config   map[string]any
}

var unsafeBuiltinConfigKeys = map[string]struct{}{
	"host":          {},
	"port":          {},
	"username":      {},
	"password":      {},
	"broker":        {},
	"brokerurl":     {},
	"database":      {},
	"db":            {},
	"token":         {},
	"secret":        {},
	"runtimekey":    {},
	"devschema":     {},
	"runtimeschema": {},
	"namespace":     {},
	"topicprefix":   {},
}

var schemaUnsafePattern = regexp.MustCompile(`[^a-z0-9_]+`)

func isBuiltinStoreType(connectionType string) bool {
	switch strings.TrimSpace(strings.ToLower(connectionType)) {
	case "builtin.relation", "builtin.timeseries", "builtin.realtime", "builtin.message":
		return true
	default:
		return false
	}
}

func normalizeBuiltinStoreCreateInput(projectID string, input CreateConnectionInput) (*builtinStoreNormalizedInput, error) {
	name, err := normalizeConnectionName(input.Name)
	if err != nil {
		return nil, err
	}

	connectionType := strings.TrimSpace(strings.ToLower(input.Type))
	config := sanitizeBuiltinConfig(input.Config)
	if connectionType == "builtin.message" {
		removeBuiltinMessageTopicConfig(config)
	}
	runtimeKey := newBuiltinRuntimeKey(connectionType)
	config["runtimeKey"] = runtimeKey

	switch connectionType {
	case "builtin.relation":
		config["store"] = "relation"
		config["devSchema"] = deriveBuiltinConnectionSchema(projectID, runtimeKey)
		config["runtimeSchema"] = runtimeKey
		config["ddlVersion"] = builtinStoreDDLVersion
	case "builtin.timeseries":
		config["store"] = "timeseries"
		config["devSchema"] = deriveBuiltinConnectionSchema(projectID, runtimeKey)
		config["runtimeSchema"] = runtimeKey
		config["ddlVersion"] = builtinStoreDDLVersion
	case "builtin.realtime":
		config["store"] = "realtime"
		config["namespace"] = runtimeKey
		config["defaultTtlSeconds"] = intFromAny(config["defaultTtlSeconds"], 300)
	case "builtin.message":
		config["store"] = "message"
	default:
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "内置运行库类型不受支持")
	}

	return &builtinStoreNormalizedInput{
		Name:     name,
		Type:     connectionType,
		Category: "builtin",
		Status:   "connected",
		Config:   config,
	}, nil
}

func sanitizeBuiltinConfig(input map[string]any) map[string]any {
	config := map[string]any{}
	for key, value := range input {
		normalizedKey := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(key), "_", ""))
		if _, blocked := unsafeBuiltinConfigKeys[normalizedKey]; blocked {
			continue
		}
		config[key] = value
	}
	return config
}

func mergeBuiltinConfigUpdate(current map[string]any, input map[string]any) map[string]any {
	next := sanitizeBuiltinConfig(input)
	if strings.TrimSpace(fmt.Sprint(current["store"])) == "message" {
		removeBuiltinMessageTopicConfig(next)
	}
	for _, key := range []string{"runtimeKey", "devSchema", "runtimeSchema", "namespace", "store", "ddlVersion"} {
		if value, ok := current[key]; ok {
			next[key] = value
		}
	}
	return next
}

func removeBuiltinMessageTopicConfig(config map[string]any) {
	for _, key := range []string{"topic", "defaultTopic", "samplePayload", "topicPrefix"} {
		delete(config, key)
	}
}

func newBuiltinRuntimeKey(connectionType string) string {
	prefix := map[string]string{
		"builtin.relation":   "rel",
		"builtin.timeseries": "ts",
		"builtin.realtime":   "rt",
		"builtin.message":    "msg",
	}[strings.TrimSpace(strings.ToLower(connectionType))]
	if prefix == "" {
		prefix = "store"
	}
	return fmt.Sprintf("%s_%s", prefix, randomHex(3))
}

func randomHex(byteCount int) string {
	if byteCount <= 0 {
		byteCount = 3
	}
	buffer := make([]byte, byteCount)
	if _, err := rand.Read(buffer); err != nil {
		return "000000"
	}
	return hex.EncodeToString(buffer)
}

func deriveBuiltinProjectSchema(projectID string, suffix string) string {
	normalized := strings.ToLower(strings.TrimSpace(projectID))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = schemaUnsafePattern.ReplaceAllString(normalized, "_")
	normalized = strings.Trim(normalized, "_")
	if normalized == "" {
		normalized = "unknown"
	}
	return fmt.Sprintf("p_%s_%s", normalized, suffix)
}

func deriveBuiltinConnectionSchema(projectID string, runtimeKey string) string {
	projectPart := sanitizeBuiltinIdentifier(projectID, "unknown")
	keyPart := sanitizeBuiltinIdentifier(runtimeKey, "store")
	return fmt.Sprintf("p_%s_%s", projectPart, keyPart)
}

func builtinConfigString(config map[string]any, key string, defaultValue string) string {
	value := strings.TrimSpace(toString(config[key]))
	if value == "" {
		return defaultValue
	}
	return value
}

func builtinConfigInt(config map[string]any, key string, defaultValue int) int {
	value := intFromAny(config[key], defaultValue)
	if value <= 0 {
		return defaultValue
	}
	return value
}

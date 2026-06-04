package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/gjson"

	"github.com/indu-forge/data_service/internal/repository"
)

var jsonPathBracketRe = regexp.MustCompile(`\[(\d+)\]`)

var errMqttTagNoUpdate = errors.New("mqtt tag no update")

type mqttBatchJSONPathRule struct {
	ArrayPath   string `json:"arrayPath"`
	NamePath    string `json:"namePath"`
	MatchName   string `json:"matchName"`
	ValuePath   string `json:"valuePath"`
	QualityPath string `json:"qualityPath"`
	TimePath    string `json:"timePath"`
}

type mqttTagExtractResult struct {
	Value       any
	Quality     string
	QualityCode any
}

// IsMqttTagNoUpdate 用于区分“本条批量消息没有包含该变量”和真正的解析错误。
func IsMqttTagNoUpdate(err error) bool {
	return errors.Is(err, errMqttTagNoUpdate)
}

// MqttTagValueSnapshot 表示 MQTT Tag 在 HTTP 当前值接口和 preview socket 中复用的统一值结构。
// 这层结构会同时服务：
// 1. HTTP `GetTagValue/GetTagValues`
// 2. preview socket 的 `mqtt:tag:value`
// 3. datapoint 对 MQTT Tag 的回退取值
type MqttTagValueSnapshot struct {
	TagID          string    `json:"tagId"`
	SubscriptionID string    `json:"subscriptionId"`
	Topic          string    `json:"topic"`
	Payload        string    `json:"payload"`
	Value          any       `json:"value"`
	ParsedValue    any       `json:"parsedValue"`
	Quality        string    `json:"quality"`
	QualityCode    any       `json:"qualityCode,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
	ReceivedAt     time.Time `json:"receivedAt"`
	Error          string    `json:"error,omitempty"`
}

// MqttSubscriptionSnapshot 表示 MQTT 订阅最近一条消息的快照。
type MqttSubscriptionSnapshot struct {
	SubscriptionID string    `json:"subscriptionId"`
	Topic          string    `json:"topic"`
	Payload        string    `json:"payload"`
	Value          any       `json:"value"`
	Quality        string    `json:"quality"`
	Timestamp      time.Time `json:"timestamp"`
	ReceivedAt     time.Time `json:"receivedAt"`
}

// BuildFallbackMqttTagSnapshot 在没有最近消息时返回默认值降级结果。
func BuildFallbackMqttTagSnapshot(tag repository.MqttTagRecord, timestamp time.Time, errorMessage string) MqttTagValueSnapshot {
	quality := "unknown"
	if strings.TrimSpace(errorMessage) != "" {
		quality = "bad"
	}

	value := defaultValueOrNil(tag.DefaultValue)
	return MqttTagValueSnapshot{
		TagID:          tag.ID,
		SubscriptionID: tag.SubscriptionID,
		Value:          value,
		ParsedValue:    value,
		Quality:        quality,
		Timestamp:      timestamp.UTC(),
		ReceivedAt:     timestamp.UTC(),
		Error:          strings.TrimSpace(errorMessage),
	}
}

// BuildMqttTagSnapshotFromMessage 使用最近一条消息解析 Tag 当前值。
func BuildMqttTagSnapshotFromMessage(tag repository.MqttTagRecord, message repository.MqttMessageRecord) MqttTagValueSnapshot {
	result, err := extractMqttTagResult(tag, message.Payload)
	if err != nil {
		fallback := BuildFallbackMqttTagSnapshot(tag, message.ReceivedAt, err.Error())
		fallback.Topic = message.Topic
		fallback.Payload = message.Payload
		return fallback
	}

	return MqttTagValueSnapshot{
		TagID:          tag.ID,
		SubscriptionID: tag.SubscriptionID,
		Topic:          message.Topic,
		Payload:        message.Payload,
		Value:          result.Value,
		ParsedValue:    result.Value,
		Quality:        defaultString(result.Quality, "good"),
		QualityCode:    result.QualityCode,
		Timestamp:      message.ReceivedAt.UTC(),
		ReceivedAt:     message.ReceivedAt.UTC(),
	}
}

// BuildMqttTagSnapshotUpdateFromMessage 仅在消息确实更新该变量时返回快照。
// 批量映射中未找到 matchName 表示本条消息没有该变量，不应覆盖上次值或广播错误。
func BuildMqttTagSnapshotUpdateFromMessage(tag repository.MqttTagRecord, message repository.MqttMessageRecord) (MqttTagValueSnapshot, bool) {
	result, err := extractMqttTagResult(tag, message.Payload)
	if err != nil {
		if IsMqttTagNoUpdate(err) {
			return MqttTagValueSnapshot{}, false
		}
		fallback := BuildFallbackMqttTagSnapshot(tag, message.ReceivedAt, err.Error())
		fallback.Topic = message.Topic
		fallback.Payload = message.Payload
		return fallback, true
	}

	return MqttTagValueSnapshot{
		TagID:          tag.ID,
		SubscriptionID: tag.SubscriptionID,
		Topic:          message.Topic,
		Payload:        message.Payload,
		Value:          result.Value,
		ParsedValue:    result.Value,
		Quality:        defaultString(result.Quality, "good"),
		QualityCode:    result.QualityCode,
		Timestamp:      message.ReceivedAt.UTC(),
		ReceivedAt:     message.ReceivedAt.UTC(),
	}, true
}

// BuildMqttSubscriptionSnapshot 使用最近一条消息构建订阅快照。
func BuildMqttSubscriptionSnapshot(message repository.MqttMessageRecord) MqttSubscriptionSnapshot {
	return MqttSubscriptionSnapshot{
		SubscriptionID: message.SubscriptionID,
		Topic:          message.Topic,
		Payload:        message.Payload,
		Value:          message.Payload,
		Quality:        "good",
		Timestamp:      message.ReceivedAt.UTC(),
		ReceivedAt:     message.ReceivedAt.UTC(),
	}
}

func extractMqttTagValue(tag repository.MqttTagRecord, payload string) (any, error) {
	result, err := extractMqttTagResult(tag, payload)
	if err != nil {
		return nil, err
	}
	return result.Value, nil
}

func extractMqttTagResult(tag repository.MqttTagRecord, payload string) (mqttTagExtractResult, error) {
	parseType := strings.TrimSpace(strings.ToLower(tag.ParseType))
	if parseType == "" {
		parseType = "jsonpath"
	}

	switch parseType {
	case "jsonpath":
		path := normalizeJSONPath(tag.ParseRule)
		if path == "" {
			return mqttTagExtractResult{}, fmt.Errorf("jsonpath 解析规则不能为空")
		}

		result := gjson.Get(payload, path)
		if !result.Exists() {
			return mqttTagExtractResult{}, fmt.Errorf("jsonpath 未命中: %s", tag.ParseRule)
		}
		return mqttTagExtractResult{Value: normalizeParsedValue(tag.DataType, result.Value()), Quality: "good"}, nil
	case "regex":
		rule := strings.TrimSpace(tag.ParseRule)
		if rule == "" {
			return mqttTagExtractResult{}, fmt.Errorf("regex 解析规则不能为空")
		}
		re, err := regexp.Compile(rule)
		if err != nil {
			return mqttTagExtractResult{}, fmt.Errorf("regex 编译失败: %w", err)
		}
		matches := re.FindStringSubmatch(payload)
		if len(matches) == 0 {
			return mqttTagExtractResult{}, fmt.Errorf("regex 未命中")
		}
		if len(matches) > 1 {
			return mqttTagExtractResult{Value: normalizeParsedValue(tag.DataType, matches[1]), Quality: "good"}, nil
		}
		return mqttTagExtractResult{Value: normalizeParsedValue(tag.DataType, matches[0]), Quality: "good"}, nil
	case "fixed":
		return mqttTagExtractResult{Value: normalizeParsedValue(tag.DataType, strings.TrimSpace(tag.ParseRule)), Quality: "good"}, nil
	case "batch_jsonpath":
		return extractMqttBatchJSONPathValue(tag, payload)
	case "script":
		return mqttTagExtractResult{}, fmt.Errorf("暂不支持 script 解析")
	default:
		return mqttTagExtractResult{}, fmt.Errorf("不支持的解析类型: %s", parseType)
	}
}

func extractMqttBatchJSONPathValue(tag repository.MqttTagRecord, payload string) (mqttTagExtractResult, error) {
	var rule mqttBatchJSONPathRule
	if err := json.Unmarshal([]byte(strings.TrimSpace(tag.ParseRule)), &rule); err != nil {
		return mqttTagExtractResult{}, fmt.Errorf("批量映射规则不是合法 JSON: %w", err)
	}

	arrayPath := normalizeJSONPath(rule.ArrayPath)
	if arrayPath == "" && strings.TrimSpace(rule.ArrayPath) != "$" {
		return mqttTagExtractResult{}, fmt.Errorf("批量映射数组路径不能为空")
	}
	namePath := normalizeRelativeJSONPath(rule.NamePath)
	valuePath := normalizeRelativeJSONPath(rule.ValuePath)
	matchName := strings.TrimSpace(rule.MatchName)
	if namePath == "" || valuePath == "" || matchName == "" {
		return mqttTagExtractResult{}, fmt.Errorf("批量映射名称路径、匹配名称和值路径不能为空")
	}

	// 批量映射面向“一条消息中包含多变量数组”的场景：先定位数组，再逐项按变量名匹配。
	arrayResult := gjson.Parse(payload)
	if arrayPath != "" {
		arrayResult = gjson.Get(payload, arrayPath)
	}
	if !arrayResult.Exists() {
		return mqttTagExtractResult{}, fmt.Errorf("批量映射数组路径未命中: %s", rule.ArrayPath)
	}
	if !arrayResult.IsArray() {
		return mqttTagExtractResult{}, fmt.Errorf("批量映射数组路径不是数组: %s", rule.ArrayPath)
	}

	for _, item := range arrayResult.Array() {
		nameResult := item.Get(namePath)
		if !nameResult.Exists() {
			continue
		}
		if strings.TrimSpace(nameResult.String()) != matchName {
			continue
		}
		valueResult := item.Get(valuePath)
		if !valueResult.Exists() {
			return mqttTagExtractResult{}, fmt.Errorf("批量映射值路径未命中: %s", rule.ValuePath)
		}
		qualityCode := extractOptionalJSONValue(item, rule.QualityPath)
		return mqttTagExtractResult{
			Value:       normalizeParsedValue(tag.DataType, valueResult.Value()),
			Quality:     resolveMqttQuality(qualityCode),
			QualityCode: qualityCode,
		}, nil
	}

	return mqttTagExtractResult{}, fmt.Errorf("%w: %s", errMqttTagNoUpdate, matchName)
}

func extractOptionalJSONValue(item gjson.Result, path string) any {
	normalized := normalizeRelativeJSONPath(path)
	if normalized == "" {
		return nil
	}
	result := item.Get(normalized)
	if !result.Exists() {
		return nil
	}
	return result.Value()
}

func resolveMqttQuality(qualityCode any) string {
	if qualityCode == nil {
		return "good"
	}
	switch typed := qualityCode.(type) {
	case float64:
		if typed == 192 {
			return "good"
		}
		return "bad"
	case int:
		if typed == 192 {
			return "good"
		}
		return "bad"
	case string:
		trimmed := strings.TrimSpace(strings.ToLower(typed))
		if trimmed == "" || trimmed == "192" || trimmed == "good" || trimmed == "true" {
			return "good"
		}
		return "bad"
	case bool:
		if typed {
			return "good"
		}
		return "bad"
	default:
		return "bad"
	}
}

func normalizeJSONPath(rule string) string {
	path := strings.TrimSpace(rule)
	path = strings.TrimPrefix(path, "$")
	path = strings.TrimPrefix(path, ".")
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	// gjson 使用 friends.0.name 这类数组下标格式，这里统一把 [0] 转成 .0。
	path = jsonPathBracketRe.ReplaceAllString(path, `.$1`)
	path = strings.TrimPrefix(path, ".")
	return path
}

func normalizeRelativeJSONPath(rule string) string {
	return normalizeJSONPath(strings.TrimSpace(rule))
}

func normalizeParsedValue(dataType string, value any) any {
	switch strings.TrimSpace(strings.ToLower(dataType)) {
	case "number":
		switch typed := value.(type) {
		case float64, float32, int, int32, int64, uint, uint32, uint64:
			return typed
		case string:
			if next, err := strconv.ParseFloat(strings.TrimSpace(typed), 64); err == nil {
				return next
			}
			return typed
		default:
			return value
		}
	case "boolean":
		switch typed := value.(type) {
		case bool:
			return typed
		case string:
			trimmed := strings.TrimSpace(strings.ToLower(typed))
			if trimmed == "true" || trimmed == "1" {
				return true
			}
			if trimmed == "false" || trimmed == "0" {
				return false
			}
			return typed
		default:
			return value
		}
	default:
		return value
	}
}

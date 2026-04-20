package service

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/gjson"

	"github.com/indu-forge/data_service/internal/repository"
)

var jsonPathBracketRe = regexp.MustCompile(`\[(\d+)\]`)

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
	value, err := extractMqttTagValue(tag, message.Payload)
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
		Value:          value,
		ParsedValue:    value,
		Quality:        "good",
		Timestamp:      message.ReceivedAt.UTC(),
		ReceivedAt:     message.ReceivedAt.UTC(),
	}
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
	parseType := strings.TrimSpace(strings.ToLower(tag.ParseType))
	if parseType == "" {
		parseType = "jsonpath"
	}

	switch parseType {
	case "jsonpath":
		path := normalizeJSONPath(tag.ParseRule)
		if path == "" {
			return nil, fmt.Errorf("jsonpath 解析规则不能为空")
		}

		result := gjson.Get(payload, path)
		if !result.Exists() {
			return nil, fmt.Errorf("jsonpath 未命中: %s", tag.ParseRule)
		}
		return normalizeParsedValue(tag.DataType, result.Value()), nil
	case "regex":
		rule := strings.TrimSpace(tag.ParseRule)
		if rule == "" {
			return nil, fmt.Errorf("regex 解析规则不能为空")
		}
		re, err := regexp.Compile(rule)
		if err != nil {
			return nil, fmt.Errorf("regex 编译失败: %w", err)
		}
		matches := re.FindStringSubmatch(payload)
		if len(matches) == 0 {
			return nil, fmt.Errorf("regex 未命中")
		}
		if len(matches) > 1 {
			return normalizeParsedValue(tag.DataType, matches[1]), nil
		}
		return normalizeParsedValue(tag.DataType, matches[0]), nil
	case "fixed":
		return normalizeParsedValue(tag.DataType, strings.TrimSpace(tag.ParseRule)), nil
	case "script":
		return nil, fmt.Errorf("暂不支持 script 解析")
	default:
		return nil, fmt.Errorf("不支持的解析类型: %s", parseType)
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

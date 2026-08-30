// Package transportlimits 保存 ingress、outbox 和 JetStream 共同依赖的冻结传输边界，避免包循环和数值漂移。
package transportlimits

import "errors"

const (
	MaxBodyBytes             = 1 << 20
	MaxTransportSubjectBytes = 4096
	// 最坏合法 DLQ JSON（1MiB base64、4096 个 JSON 转义字符及最大身份/时间字段）为 1423548 bytes。
	MaxDLQPayloadBytes = 1424000
)

var (
	// ErrOutboundPayloadTooLarge 表示记录在本地已可确定地无法被冻结
	// JetStream 拓扑承载；调用方不得将它当作瞬时发布失败重试。
	ErrOutboundPayloadTooLarge = errors.New("outbound payload exceeds frozen limit")
	// ErrOutboundSubject 表示 subject 不属于 V1 唯一允许的发布通道。
	ErrOutboundSubject = errors.New("outbound subject is not a frozen V1 subject")
)

// OutboundPayloadLimit returns the only payload limit allowed for a V1
// publication.  The DLQ exception is deliberately narrow and mirrors the
// runtime-engine config schema, rather than accepting an arbitrary dlq prefix.
func OutboundPayloadLimit(subject string) (int, error) {
	switch {
	case subject == "alarm.event":
		return MaxBodyBytes, nil
	case validDataSubject(subject, "data.raw.") || validDataSubject(subject, "data.computed."):
		return MaxBodyBytes, nil
	case validDLQSubject(subject):
		return MaxDLQPayloadBytes, nil
	default:
		return 0, ErrOutboundSubject
	}
}

func ValidateOutboundPayload(subject string, payload []byte) error {
	limit, err := OutboundPayloadLimit(subject)
	if err != nil {
		return err
	}
	if len(payload) > limit {
		return ErrOutboundPayloadTooLarge
	}
	return nil
}

func validDataSubject(subject, prefix string) bool {
	if len(subject) <= len(prefix) || subject[:len(prefix)] != prefix {
		return false
	}
	value := subject[len(prefix):]
	if len(value) != 36 {
		return false
	}
	for index, char := range value {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			if char != '-' {
				return false
			}
			continue
		}
		if !(char >= '0' && char <= '9' || char >= 'a' && char <= 'f') {
			return false
		}
	}
	return true
}

func validDLQSubject(subject string) bool {
	const prefix = "dlq."
	if len(subject) <= len(prefix) || len(subject) > len(prefix)+128 || subject[:len(prefix)] != prefix {
		return false
	}
	for index, char := range subject[len(prefix):] {
		if index == 0 {
			if !asciiAlphaNumeric(char) {
				return false
			}
			continue
		}
		if !(asciiAlphaNumeric(char) || char == '.' || char == '_' || char == ':' || char == '-') {
			return false
		}
	}
	return true
}

func asciiAlphaNumeric(char rune) bool {
	return char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9'
}

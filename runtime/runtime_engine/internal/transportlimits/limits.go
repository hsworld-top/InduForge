// Package transportlimits 保存 ingress、outbox 和 JetStream 共同依赖的冻结传输边界，避免包循环和数值漂移。
package transportlimits

const (
	MaxBodyBytes             = 1 << 20
	MaxTransportSubjectBytes = 4096
	// 最坏合法 DLQ JSON（1MiB base64、4096 个 JSON 转义字符及最大身份/时间字段）为 1423548 bytes。
	MaxDLQPayloadBytes = 1424000
)

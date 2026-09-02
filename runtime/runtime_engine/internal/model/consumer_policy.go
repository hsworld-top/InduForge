package model

// EffectiveAckWaitMS 返回 JetStream 实际生效的确认等待时间。NATS 在配置 BackOff
// 时会以首个退避间隔覆盖 AckWait；保留原 AckWaitMS 用于绑定输入审计，但与 broker
// 回显比较、创建配置必须使用此值，避免同一合法拓扑反复被判定为漂移。
func EffectiveAckWaitMS(consumer Consumer) int64 {
	if len(consumer.BackoffMS) > 0 {
		return consumer.BackoffMS[0]
	}
	return consumer.AckWaitMS
}

package compute

// Scheduler 预留给 timer/datapoint_change 触发模型。
// 当前阶段先提供结构体占位，避免后续接口调整影响 service 依赖。
type Scheduler struct{}

// NewScheduler 创建调度器占位实例。
func NewScheduler() *Scheduler {
	return &Scheduler{}
}

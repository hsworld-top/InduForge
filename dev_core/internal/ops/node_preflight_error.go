package ops

import "strings"

// 保留原始原因供服务日志诊断，界面只展示可操作的部署检查结果。
type nodePreflightError struct{ cause error }

func (e *nodePreflightError) Error() string {
	switch {
	case strings.Contains(e.cause.Error(), "匹配到"):
		return "无法确认部署节点：节点登记与运行组件不一致。请检查节点接入状态后重试。"
	case strings.Contains(e.cause.Error(), "未就绪"):
		return "运行组件尚未就绪，请在节点详情中检查状态，就绪后重试。"
	case strings.Contains(e.cause.Error(), "占用"), strings.Contains(e.cause.Error(), "同时匹配"):
		return "节点身份存在冲突，请检查是否重复接入，再重新选择部署节点。"
	case strings.Contains(e.cause.Error(), "缺少管理 IP"):
		return "节点尚未上报连接地址，请等待节点上线后重试。"
	default:
		return "暂时无法完成部署节点检查，请检查中心与节点连接后重试。"
	}
}
func (e *nodePreflightError) Unwrap() error { return e.cause }

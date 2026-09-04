/** 仅转换呈现文字，内部类型、接口标识与诊断原文保持不变。 */
export function opsBusinessText(value?: string | null): string {
  return String(value || '')
    .replace(/工程部署操作已结束[：:]\s*running/g, '工程部署操作已结束：运行中')
    .replace(/工程部署操作已结束[：:]\s*stopped/g, '工程部署操作已结束：已停止')
    .replace(/NATS(?:\s+JetStream)?/gi, '消息总线')
    .replace(/nginx/gi, '页面服务')
    .replace(/traefik/gi, '访问路由')
    .replace(/kubernetes|k3s/gi, '运行组件')
    .replace(/\bpods?\b/gi, '服务实例')
    .replace(/CrashLoopBackOff/gi, '服务反复启动失败')
    .replace(/ImagePullBackOff|ErrImagePull/gi, '运行组件获取失败')
    .replace(/\bPVC\b|PersistentVolumeClaim/gi, '持久存储')
    .replace(/工作负载/g, '服务实例')
}

/** 已知异常保留业务原因，未知技术诊断不作为默认长摘要；原文由详情组件按需呈现。 */
export function opsMessageSummary(value?: string, kind: 'error' | 'progress' | 'info' = 'info') {
  const text = String(value || '')
  if (
    !/nginx|traefik|k3s|kubernetes|\bpods?\b|NATS|CrashLoop|ImagePull|ErrImage|\bPVC\b|PersistentVolume|\/var\/|\/etc\/|\.svc\b/i.test(
      text,
    )
  )
    return opsBusinessText(text)
  const engine = text.match(/^(基础引擎|计算引擎|报警引擎|数据采集)[：:]/)?.[0] || ''
  const reason = /CrashLoop/i.test(text)
    ? '服务反复启动失败'
    : /ImagePull|ErrImage/i.test(text)
      ? '运行组件获取失败'
      : /\bPVC\b|PersistentVolume/i.test(text)
        ? '运行存储尚未就绪'
        : /timeout|deadline|超时/i.test(text)
          ? '等待运行组件超时'
          : /正在停止|停止.*工作负载/i.test(text)
            ? '正在停止服务实例'
            : /not ready|未就绪/i.test(text)
              ? '运行组件尚未就绪'
              : /已提交|已下发|dispatched/i.test(text)
                ? '服务实例已下发，等待就绪'
                : kind === 'error' || /failed|error|失败|异常/i.test(text)
                  ? '运行状态异常，请查看技术详情'
                  : kind === 'progress'
                    ? '正在准备运行组件'
                    : '运行状态已更新'
  return engine + reason
}

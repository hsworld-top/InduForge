/**
 * PropertyPanel 纯函数（从 PropertyPanel.vue 抽出以降低 SFC 体积）
 */
export function formatStyleValue(value: unknown): string {
  if (value === undefined || value === null || value === "") return "-";
  return String(value);
}

/**
 * 组件类型标识（与 manifest / 画布节点 type 一致，仅 trim）
 */
export function elementTypeName(type: string | undefined): string {
  return typeof type === "string" ? type.trim() : "";
}

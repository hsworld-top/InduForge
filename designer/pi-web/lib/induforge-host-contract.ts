export interface InduForgePiContext {
  type: "INDUFORGE_PI_CONTEXT";
  version: 1;
  projectId: string;
  workspaceRoot: "/workspace";
  locale: "zh" | "en";
  theme: "light" | "dark";
}

export function isInduForgeHostContext(value: unknown): value is InduForgePiContext {
  if (!value || typeof value !== "object") return false;
  const input = value as Record<string, unknown>;
  return input.type === "INDUFORGE_PI_CONTEXT"
    && input.version === 1
    && typeof input.projectId === "string"
    && input.workspaceRoot === "/workspace"
    && (input.locale === "zh" || input.locale === "en")
    && (input.theme === "light" || input.theme === "dark");
}

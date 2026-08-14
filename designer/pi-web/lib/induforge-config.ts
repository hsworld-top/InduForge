import { existsSync, realpathSync } from "fs";
import path from "path";

const DEFAULT_WORKSPACE_ROOT = "/workspace";

export function getInduForgeWorkspaceRoot(): string {
  return path.resolve(process.env.PI_WEB_WORKSPACE_ROOT?.trim() || DEFAULT_WORKSPACE_ROOT);
}

export function getInduForgeAllowedParentOrigins(): string[] {
  const origins = (process.env.PI_WEB_ALLOWED_PARENT_ORIGINS ?? "")
    .split(",")
    .map((origin) => origin.trim())
    .filter(Boolean)
    .map((origin) => {
      try {
        const url = new URL(origin);
        return url.protocol === "http:" || url.protocol === "https:" ? url.origin : null;
      } catch {
        return null;
      }
    })
    .filter((origin): origin is string => Boolean(origin));
  return [...new Set(origins)];
}

export function getInduForgePlatformExtensionsDir(): string | null {
  const configured = process.env.PI_WEB_PLATFORM_EXTENSIONS_DIR?.trim();
  if (!configured) return null;
  const resolved = path.resolve(configured);
  return existsSync(resolved) ? resolved : null;
}

function canonicalizeExistingPath(value: string): string | null {
  try {
    return realpathSync(path.resolve(value));
  } catch {
    return null;
  }
}

export function isInduForgeWorkspaceRoot(value: string): boolean {
  const workspace = canonicalizeExistingPath(getInduForgeWorkspaceRoot());
  const candidate = canonicalizeExistingPath(value);
  return Boolean(workspace && candidate && workspace === candidate);
}

export function isPathInsideInduForgeWorkspace(value: string): boolean {
  const workspace = canonicalizeExistingPath(getInduForgeWorkspaceRoot());
  const candidate = canonicalizeExistingPath(value);
  if (!workspace || !candidate) return false;
  const relative = path.relative(workspace, candidate);
  return relative === "" || (!relative.startsWith(`..${path.sep}`) && relative !== ".." && !path.isAbsolute(relative));
}

export function getProjectSkillsDir(): string {
  return path.join(getInduForgeWorkspaceRoot(), ".pi", "skills");
}

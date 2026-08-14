import assert from "node:assert/strict";
import { mkdtempSync, mkdirSync, rmSync, symlinkSync, writeFileSync } from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import {
  getInduForgeAllowedParentOrigins,
  getProjectSkillsDir,
  isInduForgeWorkspaceRoot,
  isPathInsideInduForgeWorkspace,
} from "./induforge-config.ts";

test("normalizes and filters allowed parent origins", () => {
  const previous = process.env.PI_WEB_ALLOWED_PARENT_ORIGINS;
  process.env.PI_WEB_ALLOWED_PARENT_ORIGINS =
    "https://designer.example.com/path,http://localhost:18603,https://designer.example.com,invalid,file:///tmp";
  try {
    assert.deepEqual(getInduForgeAllowedParentOrigins(), [
      "https://designer.example.com",
      "http://localhost:18603",
    ]);
  } finally {
    if (previous === undefined) delete process.env.PI_WEB_ALLOWED_PARENT_ORIGINS;
    else process.env.PI_WEB_ALLOWED_PARENT_ORIGINS = previous;
  }
});

test("authorizes only the real workspace tree", (t) => {
  const root = mkdtempSync(path.join(os.tmpdir(), "induforge-pi-web-"));
  const workspace = path.join(root, "workspace");
  const outside = path.join(root, "outside");
  mkdirSync(workspace);
  mkdirSync(outside);
  const insideFile = path.join(workspace, "inside.txt");
  const outsideFile = path.join(outside, "outside.txt");
  writeFileSync(insideFile, "inside");
  writeFileSync(outsideFile, "outside");
  const previous = process.env.PI_WEB_WORKSPACE_ROOT;
  process.env.PI_WEB_WORKSPACE_ROOT = workspace;
  t.after(() => {
    if (previous === undefined) delete process.env.PI_WEB_WORKSPACE_ROOT;
    else process.env.PI_WEB_WORKSPACE_ROOT = previous;
    rmSync(root, { recursive: true, force: true });
  });

  assert.equal(isInduForgeWorkspaceRoot(workspace), true);
  assert.equal(isInduForgeWorkspaceRoot(outside), false);
  assert.equal(isPathInsideInduForgeWorkspace(insideFile), true);
  assert.equal(isPathInsideInduForgeWorkspace(outsideFile), false);
  assert.equal(getProjectSkillsDir(), path.join(workspace, ".pi", "skills"));

  const escape = path.join(workspace, "escape");
  try {
    symlinkSync(outside, escape, process.platform === "win32" ? "junction" : "dir");
    assert.equal(isPathInsideInduForgeWorkspace(path.join(escape, "outside.txt")), false);
  } catch (error) {
    if (process.platform !== "win32") throw error;
    t.diagnostic("当前 Windows 权限不允许创建 junction，跳过符号链接断言");
  }
});

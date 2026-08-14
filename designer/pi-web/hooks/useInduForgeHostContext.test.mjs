import assert from "node:assert/strict";
import test from "node:test";
import { isInduForgeHostContext } from "../lib/induforge-host-contract.ts";

const valid = {
  type: "INDUFORGE_PI_CONTEXT",
  version: 1,
  projectId: "project-1",
  workspaceRoot: "/workspace",
  locale: "zh",
  theme: "dark",
};

test("accepts the current host context contract", () => {
  assert.equal(isInduForgeHostContext(valid), true);
  assert.equal(isInduForgeHostContext({ ...valid, locale: "en", theme: "light" }), true);
});

test("rejects malformed host context messages", () => {
  assert.equal(isInduForgeHostContext(null), false);
  assert.equal(isInduForgeHostContext({ ...valid, version: 2 }), false);
  assert.equal(isInduForgeHostContext({ ...valid, workspaceRoot: "/tmp" }), false);
  assert.equal(isInduForgeHostContext({ ...valid, locale: "fr" }), false);
  assert.equal(isInduForgeHostContext({ ...valid, theme: "auto" }), false);
  assert.equal(isInduForgeHostContext({ ...valid, projectId: 1 }), false);
});

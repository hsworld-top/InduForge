import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const source = await readFile(new URL("./ChatWindow.tsx", import.meta.url), "utf8");

test("uses an embedded project empty state without upstream branding or versions", () => {
  assert.match(source, />\s*开始开发\s*</);
  assert.match(source, />当前工程</);
  assert.match(source, />\/workspace</);
  assert.doesNotMatch(source, />π</);
  assert.doesNotMatch(source, /NEXT_PUBLIC_(?:APP|PI)_VERSION/);
  assert.doesNotMatch(source, /NewSessionUpdateLink/);
});

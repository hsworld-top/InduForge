import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const source = await readFile(new URL("./[...path]/route.ts", import.meta.url), "utf8");

test("file routes authorize only the configured workspace root", () => {
  assert.match(source, /if \(!isFilePathAllowed\(filePath, allowedRoots\)\)/);
  assert.match(source, /if \(!isExistingFilePathAllowed\(existingAuthorizationPath, allowedRoots\)\)/);
  assert.doesNotMatch(source, /isFilePathReferencedBySession|allowedBySessionReference|sessionId/);
});

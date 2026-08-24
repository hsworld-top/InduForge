"use strict";

const fs = require("node:fs");
const vm = require("node:vm");
const envelope = JSON.parse(fs.readFileSync(0, "utf8") || "{}");
const script = String(envelope.script || "");
try {
  new vm.Script("(async function(argv, dp, ctx) {\n\"use strict\";\n" + script + "\n})", { filename: "<compute>" });
  process.stdout.write(JSON.stringify({ diagnostics: [] }));
} catch (error) {
  const stack = String((error && error.stack) || "");
  const match = stack.match(/<compute>:(\d+)(?::(\d+))?/);
  const line = Math.max(1, Number(match && match[1] ? match[1] : 3) - 2);
  const column = Math.max(1, Number(match && match[2] ? match[2] : 1));
  process.stdout.write(JSON.stringify({ diagnostics: [{ severity: "error", message: String(error.message || error), line, column, endLine: line, endColumn: column + 1, source: "javascript" }] }));
}

"use strict";

const fs = require("node:fs");
const path = require("node:path");
const crypto = require("node:crypto");
const vm = require("node:vm");
const envelope = JSON.parse(fs.readFileSync(0, "utf8") || "{}");
const input = envelope.input || {};
const sdkContext = envelope.sdkContext || {};
const userScript = String(envelope.script || "");
const dependencies = Array.isArray(envelope.dependencies) ? envelope.dependencies : [];
const argv = Array.isArray(input.argv) ? input.argv : [];
const dp = sdkContext.variables || {};
const logs = [];
const safeConsole = Object.freeze({
  log: (...values) => logs.push(values.map(String).join(" ")),
  info: (...values) => logs.push(values.map(String).join(" ")),
  warn: (...values) => logs.push(values.map(String).join(" ")),
  error: (...values) => logs.push(values.map(String).join(" ")),
});

const allowedDependencies = new Map();
for (const item of dependencies) {
  const packageName = String(item.packageName || "");
  const importName = String(item.importName || packageName);
  const key = crypto.createHash("sha256").update(packageName.toLowerCase()).digest("hex").slice(0, 24);
  allowedDependencies.set(importName, path.join("/dependencies/js", key, "node_modules", packageName));
}
function safeRequire(name) {
  const target = allowedDependencies.get(String(name || ""));
  if (!target) throw new Error("依赖未在当前工程安装或未被脚本声明: " + name);
  return require(target);
}

const ctx = Object.freeze({
  datapoint: Object.freeze({
    get(path) {
      const key = String(path || "");
      const item = sdkContext.datapoints && sdkContext.datapoints[key];
      if (!item) throw new Error("ctx.datapoint.get is not declared or prefetched: " + key);
      return item.value;
    },
    meta(path) {
      const key = String(path || "");
      const item = sdkContext.datapoints && sdkContext.datapoints[key];
      if (!item) throw new Error("ctx.datapoint.meta is not declared or prefetched: " + key);
      return item;
    },
  }),
  sql: Object.freeze({
    query(key) {
      const name = String(key || "");
      const result = sdkContext.sql && sdkContext.sql[name];
      if (result === undefined) throw new Error("ctx.sql.query is not declared or prefetched: " + name);
      return result;
    },
  }),
});

(async () => {
  const sandbox = Object.create(null);
  Object.assign(sandbox, { argv, dp, ctx, console: safeConsole, require: safeRequire });
  const context = vm.createContext(sandbox, {
    name: "induforge-compute",
    codeGeneration: { strings: false, wasm: false },
  });
  const script = new vm.Script('"use strict"; (async () => {\n' + userScript + '\n})()', {
    filename: "<compute>",
  });
  const output = await script.runInContext(context);
  process.stdout.write(JSON.stringify({ output, sideEffects: [], logs }));
})().catch((error) => {
  process.stderr.write(error && error.stack ? String(error.stack) : String(error));
  process.exit(1);
});

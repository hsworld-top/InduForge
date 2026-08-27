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
const logs = [];
const sideEffects = [];
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

function sdkResult(code, msg, data) {
  return Object.freeze({ code, msg, data });
}

function unsupported(path, operation) {
  return sdkResult(40031, `数据点 ${path} 在当前开发环境不支持 ${operation}`, null);
}

function buildSample(snapshot) {
  return Object.freeze({
    path: snapshot.path || "",
    value: snapshot.value ?? null,
    quality: snapshot.quality || "unknown",
    timestamp: snapshot.timestamp || null,
    observedAt: snapshot.observedAt || null,
    sourceTimestamp: snapshot.sourceTimestamp || null,
    status: snapshot.status || null,
  });
}

function createPoint(snapshot) {
  const path = String(snapshot.path || "");
  const capabilities = Object.freeze({
    get: snapshot.capabilities?.get === true,
    read: snapshot.capabilities?.read === true,
    peek: snapshot.capabilities?.peek === true,
    set: snapshot.capabilities?.set === true,
    subscribe: snapshot.capabilities?.subscribe === true,
    history: snapshot.capabilities?.history === true,
    refresh: snapshot.capabilities?.refresh === true,
    run: snapshot.capabilities?.run === true,
    execute: snapshot.capabilities?.execute === true,
    publish: snapshot.capabilities?.publish === true,
  });
  const effect = (operation, payload) => {
    if (!capabilities[operation]) return unsupported(path, operation);
    const item = Object.freeze({ domain: "point", operation, path, payload: payload ?? null });
    sideEffects.push(item);
    return sdkResult(0, "开发态副作用已记录", item);
  };
  const readSnapshot = (operation, data) => {
    if (!capabilities[operation]) return unsupported(path, operation);
    return sdkResult(0, "ok", data);
  };
  return Object.freeze({
    id: snapshot.id || null,
    ref: path,
    path,
    name: snapshot.name || null,
    displayName: snapshot.displayName || snapshot.name || null,
    dataType: snapshot.dataType || null,
    schema: snapshot.schema || null,
    source: Object.freeze({ type: snapshot.sourceType || null, id: snapshot.sourceId || null }),
    status: snapshot.status || null,
    unit: snapshot.unit ?? null,
    precision: snapshot.precision ?? null,
    min: snapshot.min ?? null,
    max: snapshot.max ?? null,
    defaultValue: snapshot.defaultValue ?? null,
    tags: Object.freeze(Array.isArray(snapshot.tags) ? [...snapshot.tags] : []),
    attributes: Object.freeze({ ...(snapshot.attributes || {}) }),
    capabilities,
    get: () => readSnapshot("get", snapshot.value ?? null),
    read: () => readSnapshot("read", buildSample(snapshot)),
    peek: () => readSnapshot("peek", buildSample(snapshot)),
    set: (value) => effect("set", value),
    subscribe: () => unsupported(path, "subscribe"),
    history: () => unsupported(path, "history"),
    refresh: (options) => effect("refresh", options),
    run: (runtimeInput) => effect("run", runtimeInput),
    execute: (runtimeInput) => effect("execute", runtimeInput),
    publish: (payload) => effect("publish", payload),
  });
}

const pointBindings = sdkContext.pointBindings || {};
const points = Object.create(null);
for (const [alias, pointPath] of Object.entries(pointBindings)) {
  const snapshot = sdkContext.datapoints?.[pointPath];
  if (snapshot) points[alias] = createPoint(snapshot);
}
Object.freeze(points);
const dp = points;

const ctx = Object.freeze({
  points,
  args: Object.freeze({ ...input }),
  trigger: Object.freeze({ ...(input.trigger || {}) }),
  runtime: Object.freeze({ ...(sdkContext.metadata || {}) }),
  logger: safeConsole,
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
  for (const [alias, point] of Object.entries(points)) sandbox[alias] = point;
  const context = vm.createContext(sandbox, {
    name: "induforge-compute",
    codeGeneration: { strings: false, wasm: false },
  });
  const script = new vm.Script('"use strict"; (async () => {\n' + userScript + '\n})()', {
    filename: "<compute>",
  });
  const output = await script.runInContext(context);
  process.stdout.write(JSON.stringify({ output, sideEffects, logs }));
})().catch((error) => {
  process.stderr.write(error && error.stack ? String(error.stack) : String(error));
  process.exit(1);
});

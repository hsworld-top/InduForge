#!/usr/bin/env node

// InduForge 基础设施初始化脚本。
//
// 输入：
// - 仓库根目录 `.env` 中的 IF_* 环境变量。
// - 可选参数 `--check-only`，只检查连接和数据库状态，不创建数据库、不启用扩展。
//
// 输出：
// - 创建平台需要的数据库。
// - 为开发态时序数据库启用 timescaledb 扩展。
// - 检查缓存、消息接入、对象存储端口是否可连接。
//
// 异常处理：
// - 数据库连接失败、非法数据库名或 SQL 执行失败会让进程以非 0 状态退出。
// - TCP 检查失败只打印警告，因为部分能力在开发时可能按需启动。

import fs from "node:fs";
import net from "node:net";
import path from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";
import { createRequire } from "node:module";

const require = createRequire(import.meta.url);
const { Client } = require("pg");

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const repoRoot = path.resolve(__dirname, "../..");
const envPath = path.join(repoRoot, ".env");
const checkOnly = process.argv.includes("--check-only");
const META_STORE_RETRY_COUNT = 5;
const META_STORE_RETRY_DELAY_MS = 3000;
const META_STORE_SETTLE_DELAY_MS = 3000;

// 读取简单 KEY=VALUE 形式的 .env 文件。
// 这里不覆盖进程中已经存在的同名变量，方便 CI/CD 或命令行临时覆盖配置。
function loadDotEnv(filePath) {
  if (!fs.existsSync(filePath)) return;
  const lines = fs.readFileSync(filePath, "utf8").split(/\r?\n/);
  for (const rawLine of lines) {
    const line = rawLine.trim();
    if (!line || line.startsWith("#")) continue;
    const [key, ...rest] = line.replace(/^export\s+/, "").split("=");
    if (!key || process.env[key]) continue;
    process.env[key] = rest.join("=").trim().replace(/^['"]|['"]$/g, "");
  }
}

// 统一读取环境变量并去掉首尾空白。
// 返回字符串是为了避免端口、DB 编号等值在拼接前出现 undefined/null。
function env(name, fallback = "") {
  return String(process.env[name] ?? fallback).trim();
}

// SQL 标识符只能通过白名单校验后拼接。
// database 和 extension 名称不能作为参数化查询值传入，因此必须在拼接前限制字符集。
function quoteIdent(value) {
  const normalized = String(value || "").trim();
  if (!/^[a-zA-Z_][a-zA-Z0-9_]*$/.test(normalized)) {
    throw new Error(`非法数据库标识符: ${value}`);
  }
  return `"${normalized.replace(/"/g, '""')}"`;
}

// 轻量 TCP 检查用于确认依赖能力是否启动。
// 返回布尔值而不是抛错，避免单个可选能力未启动时阻断整个本地初始化流程。
function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function withRetry(label, fn, maxAttempts = META_STORE_RETRY_COUNT, delayMs = META_STORE_RETRY_DELAY_MS) {
  let lastError;
  for (let attempt = 1; attempt <= maxAttempts; attempt += 1) {
    try {
      return await fn();
    } catch (error) {
      lastError = error;
      if (attempt === maxAttempts) break;
      console.warn(`${label}失败，${delayMs / 1000} 秒后重试 (${attempt}/${maxAttempts}): ${error.message}`);
      await sleep(delayMs);
    }
  }
  throw lastError;
}

async function waitForMetaStore(maxAttempts = 30, delayMs = 2000) {
  for (let attempt = 1; attempt <= maxAttempts; attempt += 1) {
    const client = new Client(metaStoreConfig);
    try {
      await client.connect();
      await client.query("SELECT 1");
      await client.query("SELECT count(*) FROM pg_database");
      await client.end();
      console.log("元数据能力已就绪，等待服务稳定...");
      await sleep(META_STORE_SETTLE_DELAY_MS);
      return;
    } catch (error) {
      await client.end().catch(() => {});
      if (attempt === maxAttempts) {
        throw error;
      }
      console.warn(`等待元数据能力就绪 (${attempt}/${maxAttempts})...`);
      await sleep(delayMs);
    }
  }
}

function tcpCheck(host, port, label) {
  return new Promise((resolve) => {
    const socket = net.createConnection({ host, port: Number(port), timeout: 3000 });
    socket.once("connect", () => {
      socket.destroy();
      console.log(`✓ ${label} 可连接 ${host}:${port}`);
      resolve(true);
    });
    socket.once("timeout", () => {
      socket.destroy();
      console.warn(`! ${label} 连接超时 ${host}:${port}`);
      resolve(false);
    });
    socket.once("error", (error) => {
      console.warn(`! ${label} 不可连接 ${host}:${port}: ${error.message}`);
      resolve(false);
    });
  });
}

loadDotEnv(envPath);

// 元数据能力连接配置。
// 开发环境通常连接宿主机映射端口，生产环境在容器网络内连接内部 host。
const metaStoreConfig = {
  host: env("IF_META_STORE_HOST", "127.0.0.1"),
  port: Number(env("IF_META_STORE_PORT", "18432")),
  user: env("IF_META_STORE_USER", "postgres"),
  password: env("IF_META_STORE_PASSWORD", "postgres"),
  database: env("IF_META_STORE_ADMIN_DATABASE", "postgres"),
  ssl: env("IF_META_STORE_SSL", "false").toLowerCase() === "true" ? { rejectUnauthorized: false } : false,
};

// 平台当前按能力域拆分数据库。
// filter(Boolean) 允许临时关闭某个库名，但默认模板会创建 if_core、if_data、if_dev_data。
const requiredDatabases = [
  env("IF_META_STORE_CORE_DB", "if_core"),
  env("IF_META_STORE_DATA_DB", "if_data"),
  env("IF_META_STORE_DEV_DATA_DB", "if_dev_data"),
].filter(Boolean);

// 幂等创建数据库。
// PostgreSQL 不支持 CREATE DATABASE IF NOT EXISTS，因此先查 pg_database 再创建。
async function createDatabaseIfNeeded(database) {
  await withRetry(`检查或创建数据库 ${database}`, async () => {
    const client = new Client(metaStoreConfig);
    await client.connect();
    try {
      const exists = await client.query("SELECT 1 FROM pg_database WHERE datname = $1", [database]);
      if (exists.rowCount > 0) {
        console.log(`✓ 数据库已存在: ${database}`);
        return;
      }
      if (checkOnly) {
        console.warn(`! 数据库不存在: ${database}`);
        return;
      }
      await client.query(`CREATE DATABASE ${quoteIdent(database)}`);
      console.log(`✓ 数据库已创建: ${database}`);
    } finally {
      await client.end();
    }
  });
}

// 在指定数据库内启用扩展。
// 扩展名同样先经过 quoteIdent 白名单校验，避免拼接 SQL 带来注入风险。
async function enableExtension(database, extension) {
  if (checkOnly) return;
  await withRetry(`启用 ${database} 时序扩展`, async () => {
    const client = new Client({ ...metaStoreConfig, database });
    await client.connect();
    try {
      await client.query(`CREATE EXTENSION IF NOT EXISTS ${quoteIdent(extension)}`);
      console.log(`✓ ${database} 已启用扩展: ${extension}`);
    } finally {
      await client.end();
    }
  });
}

// 初始化元数据能力：
// 1. 连接管理库。
// 2. 创建平台需要的业务库。
// 3. 在开发态时序库中启用 timescaledb 扩展。
async function initMetaStore() {
  await waitForMetaStore();

  for (const database of requiredDatabases) {
    await createDatabaseIfNeeded(database);
  }

  const devDatabase = env("IF_META_STORE_DEV_DATA_DB", "if_dev_data");
  if (requiredDatabases.includes(devDatabase)) {
    await enableExtension(devDatabase, "timescaledb");
  }
}

// 主流程先做强依赖初始化，再检查弱依赖端口。
// 这样数据库不可用会直接失败，其他能力未启动则以警告形式提示用户补启动。
async function main() {
  console.log(checkOnly ? "检查 InduForge 基础设施..." : "初始化 InduForge 基础设施...");
  await initMetaStore();
  await tcpCheck(env("IF_CACHE_STORE_HOST", "127.0.0.1"), env("IF_CACHE_STORE_PORT", "18379"), "缓存存储");
  await tcpCheck(env("IF_MESSAGE_HUB_HOST", "127.0.0.1"), env("IF_MESSAGE_HUB_MQTT_PORT", "18883"), "消息中心");
  await tcpCheck(env("IF_OBJECT_STORE_ENDPOINT", "127.0.0.1"), env("IF_OBJECT_STORE_PORT", "18500"), "对象存储");
}

// 顶层兜底异常处理。
// 保留原始错误消息，便于用户判断是连接、权限、库名还是扩展启用问题。
main().catch((error) => {
  console.error(`初始化失败: ${error.message}`);
  process.exitCode = 1;
});

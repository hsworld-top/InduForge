import test from "node:test";
import assert from "node:assert/strict";

import {
  datacenterLocale,
  getDatacenterRouteTitle,
  normalizeDatacenterLocale,
  setDatacenterLocale,
  t,
} from "../src/i18n/runtime.js";
import { createRuntimeMessageHandler } from "../src/runtime/runtime-message-handler.js";
import { resolveDatacenterTabLabel } from "../src/utils/tabTitle.js";

test("locale 归一化与翻译输出", () => {
  assert.equal(normalizeDatacenterLocale("en-US"), "en");
  assert.equal(normalizeDatacenterLocale("zh-CN"), "zh");

  setDatacenterLocale("en");
  assert.equal(t("tabs.tableList"), "Tables");
  assert.equal(
    getDatacenterRouteTitle("datacenter-debug"),
    "Data Center Debug",
  );

  setDatacenterLocale("zh");
  assert.equal(t("tabs.tableList"), "表列表");
  assert.equal(getDatacenterRouteTitle("datacenter"), "数据中心");
});

test("标签标题会按当前语言实时重算", () => {
  assert.equal(
    resolveDatacenterTabLabel(
      { labelPrefix: "连接A", labelKey: "tabs.tableList" },
      (key) => (key === "tabs.tableList" ? "Tables" : key),
    ),
    "连接A - Tables",
  );
});

test("宿主 LOCALE_UPDATE 会更新响应式 locale", () => {
  datacenterLocale.value = "zh";
  const trustedSource = { postMessage() {} };

  const handler = createRuntimeMessageHandler({
    getTrustedOriginSet: () => new Set(["https://ide.example.com"]),
    getTrustedSources: () => new Set([trustedSource]),
    isTrustedHostMessage: (event, context) =>
      context.trustedOrigins.has(event.origin) &&
      context.trustedSources.has(event.source),
    handleBootstrapResponseMessage: () => false,
    handleAuthRefreshedMessage: () => false,
  });

  handler({
    data: {
      type: "LOCALE_UPDATE",
      locale: "en-US",
    },
    origin: "https://ide.example.com",
    source: trustedSource,
  });

  assert.equal(datacenterLocale.value, "en");
});

// @ts-nocheck
import { test } from "vitest";
import assert from "node:assert/strict";

import { createMqttSocketSharedRegistry } from "../src/composables/mqtt-socket-shared";

function createFakeSocket(id) {
  const listeners = new Map();
  const anyListeners = new Set();

  const socket = {
    id,
    connected: false,
    emitted: [],
    disconnectCalls: 0,
    on(event, handler) {
      const handlers = listeners.get(event) || [];
      handlers.push(handler);
      listeners.set(event, handlers);
      return socket;
    },
    onAny(handler) {
      anyListeners.add(handler);
      return socket;
    },
    emit(event, payload) {
      socket.emitted.push({ event, payload });
    },
    disconnect() {
      socket.disconnectCalls += 1;
      socket.connected = false;
    },
    trigger(event, ...args) {
      if (event === "connect") {
        socket.connected = true;
      }
      if (event === "disconnect") {
        socket.connected = false;
      }

      (listeners.get(event) || []).forEach((handler) => handler(...args));
      anyListeners.forEach((handler) => handler(event, ...args));
    },
  };

  return socket;
}

function createSilentLogger() {
  return {
    log() {},
    warn() {},
    error() {},
  };
}

test("共享注册表会复用同一条连接，并对订阅做引用计数", () => {
  const sockets = [];
  const registry = createMqttSocketSharedRegistry({
    ioFactory: () => {
      const socket = createFakeSocket(`socket-${sockets.length + 1}`);
      sockets.push(socket);
      return socket;
    },
    getApiUrl: () => "http://localhost:19601",
    getToken: () => "token-1",
    logger: createSilentLogger(),
  });

  const first = registry.acquire({
    projectId: "project-1",
    previewSessionId: "session-1",
  });
  const second = registry.acquire({
    projectId: "project-1",
    previewSessionId: "session-1",
  });

  assert.ok(first);
  assert.ok(second);
  assert.equal(first.key, second.key);
  assert.equal(sockets.length, 1);

  registry.subscribeSubscription(first.key, "sub-1");
  registry.subscribeSubscription(first.key, "sub-1");
  registry.subscribeTag(first.key, "tag-1");
  registry.subscribeTag(first.key, "tag-1");

  assert.deepEqual(sockets[0].emitted, []);

  sockets[0].trigger("connect");

  assert.deepEqual(sockets[0].emitted, [
    {
      event: "mqtt:subscribe",
      payload: { subscriptionId: "sub-1" },
    },
    {
      event: "mqtt:tag:subscribe",
      payload: { tagId: "tag-1" },
    },
  ]);

  registry.unsubscribeSubscription(first.key, "sub-1");
  registry.unsubscribeTag(first.key, "tag-1");

  assert.deepEqual(sockets[0].emitted, [
    {
      event: "mqtt:subscribe",
      payload: { subscriptionId: "sub-1" },
    },
    {
      event: "mqtt:tag:subscribe",
      payload: { tagId: "tag-1" },
    },
  ]);

  registry.unsubscribeSubscription(first.key, "sub-1");
  registry.unsubscribeTag(first.key, "tag-1");

  assert.deepEqual(sockets[0].emitted, [
    {
      event: "mqtt:subscribe",
      payload: { subscriptionId: "sub-1" },
    },
    {
      event: "mqtt:tag:subscribe",
      payload: { tagId: "tag-1" },
    },
    {
      event: "mqtt:unsubscribe",
      payload: { subscriptionId: "sub-1" },
    },
    {
      event: "mqtt:tag:unsubscribe",
      payload: { tagId: "tag-1" },
    },
  ]);

  registry.release(first.key);
  assert.equal(sockets[0].disconnectCalls, 0);

  registry.release(second.key);
  assert.equal(sockets[0].disconnectCalls, 1);
});

test("共享注册表会使用传入的数据服务地址建立 socket 连接", () => {
  const captured = [];
  const registry = createMqttSocketSharedRegistry({
    ioFactory: (url, options) => {
      captured.push({ url, options });
      return createFakeSocket("socket-1");
    },
    getApiUrl: () => "http://localhost:19602/",
    getToken: () => "token-1",
    logger: createSilentLogger(),
  });

  const result = registry.acquire({
    projectId: "project-1",
    previewSessionId: "session-1",
  });

  assert.ok(result);
  assert.equal(captured.length, 1);
  assert.equal(captured[0].url, "http://localhost:19602");
  assert.equal(captured[0].options.path, "/socket.io/");
});

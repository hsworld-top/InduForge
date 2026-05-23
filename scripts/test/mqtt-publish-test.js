#!/usr/bin/env node

const net = require("node:net");

const config = {
  // MQTT 服务器地址，支持 mqtt://host:port，也可以用命令行 --broker 覆盖。
  broker: "mqtt://127.0.0.1:18883",
  topic: "induforge/mock-data",
  intervalMs: 1000,
  clientIdPrefix: "induforge-mock-data",
  baseValue: 256,
  step: 1,
};

const usage = `
Usage:
  node scripts/test/mqtt-publish-test.js [--broker mqtt://127.0.0.1:18883] [--topic induforge/mock-data] [--interval 1000]

Examples:
  node scripts/test/mqtt-publish-test.js
  node scripts/test/mqtt-publish-test.js --broker mqtt://127.0.0.1:18883 --topic induforge/mock-data --interval 500
`;

function parseArgs(argv) {
  const options = { ...config };

  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    if (arg === "--help" || arg === "-h") {
      console.log(usage.trim());
      process.exit(0);
    }

    if (!arg.startsWith("--")) {
      throw new Error(`无法识别的参数: ${arg}`);
    }

    const key = arg.slice(2);
    const value = argv[i + 1];
    if (!value || value.startsWith("--")) {
      throw new Error(`参数 ${arg} 缺少值`);
    }
    i += 1;

    if (key === "broker") {
      options.broker = value;
    } else if (key === "topic") {
      options.topic = value;
    } else if (key === "interval") {
      options.intervalMs = Number(value);
    } else {
      throw new Error(`不支持的参数: ${arg}`);
    }
  }

  if (!options.topic.trim()) {
    throw new Error("--topic 不能为空");
  }
  if (!Number.isInteger(options.intervalMs) || options.intervalMs <= 0) {
    throw new Error("--interval 必须是正整数毫秒数");
  }

  return options;
}

function parseBroker(broker) {
  const url = new URL(broker);
  if (url.protocol !== "mqtt:") {
    throw new Error("broker 仅支持 mqtt:// 协议");
  }

  return {
    host: url.hostname,
    port: url.port ? Number(url.port) : 1883,
  };
}

function formatNow() {
  const date = new Date();
  const pad = (value) => String(value).padStart(2, "0");

  return [
    `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`,
    `${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`,
  ].join(" ");
}

function encodeUtf8(value) {
  return Buffer.from(String(value), "utf8");
}

function encodeString(value) {
  const body = encodeUtf8(value);
  const length = Buffer.allocUnsafe(2);
  length.writeUInt16BE(body.length, 0);
  return Buffer.concat([length, body]);
}

function encodeRemainingLength(length) {
  const chunks = [];
  let remaining = length;

  do {
    let encodedByte = remaining % 128;
    remaining = Math.floor(remaining / 128);
    if (remaining > 0) {
      encodedByte |= 128;
    }
    chunks.push(encodedByte);
  } while (remaining > 0);

  return Buffer.from(chunks);
}

function packet(type, body) {
  return Buffer.concat([
    Buffer.from([type]),
    encodeRemainingLength(body.length),
    body,
  ]);
}

function buildConnectPacket(clientId) {
  const protocolName = encodeString("MQTT");
  const protocolLevel = Buffer.from([4]);
  const connectFlags = Buffer.from([2]);
  const keepAlive = Buffer.allocUnsafe(2);
  keepAlive.writeUInt16BE(30, 0);

  // 只使用匿名 clean session，保证测试脚本没有额外认证依赖。
  const variableHeader = Buffer.concat([
    protocolName,
    protocolLevel,
    connectFlags,
    keepAlive,
  ]);
  return packet(0x10, Buffer.concat([variableHeader, encodeString(clientId)]));
}

function buildPublishPacket(topic, payload) {
  return packet(
    0x30,
    Buffer.concat([encodeString(topic), encodeUtf8(payload)]),
  );
}

function buildPayload(sequence, options) {
  const value = options.baseValue + sequence * options.step;

  // 每次发布都生成递增数值与当前时间，便于前端或订阅端观察实时变化。
  return JSON.stringify({
    source: "induforge-mock-data",
    sequence,
    value,
    ts: formatNow(),
  });
}

function startPublisher(options) {
  const broker = parseBroker(options.broker);
  const socket = net.createConnection(broker.port, broker.host);
  const clientId = `${options.clientIdPrefix}-${Date.now()}`;
  let connected = false;
  let sequence = 0;
  let timer = null;

  const stop = () => {
    if (timer) {
      clearInterval(timer);
      timer = null;
    }

    if (!socket.destroyed && socket.writable) {
      socket.write(Buffer.from([0xe0, 0x00]));
      socket.end();
    }
  };
  const fail = (error) => {
    stop();
    console.error(error.message);
    process.exit(1);
  };

  socket.setTimeout(5000);
  socket.once("connect", () => {
    socket.write(buildConnectPacket(clientId));
  });
  socket.on("data", (chunk) => {
    if (connected) return;
    if (chunk.length < 4 || chunk[0] !== 0x20 || chunk[3] !== 0x00) {
      fail(new Error(`MQTT CONNACK 失败: ${chunk.toString("hex")}`));
      return;
    }

    connected = true;
    console.log(
      `已连接 MQTT: ${options.broker} topic=${options.topic} interval=${options.intervalMs}ms`,
    );

    const publishOnce = () => {
      sequence += 1;
      const payload = buildPayload(sequence, options);
      socket.write(buildPublishPacket(options.topic, payload));
      console.log(`已发布 #${sequence}: ${payload}`);
    };

    publishOnce();
    timer = setInterval(publishOnce, options.intervalMs);
  });
  socket.once("timeout", () => {
    fail(new Error("连接 MQTT Broker 超时"));
  });
  socket.once("error", (error) => {
    fail(error);
  });
  socket.once("close", () => {
    if (timer) clearInterval(timer);
  });

  return stop;
}

function main() {
  let stop = null;

  try {
    const options = parseArgs(process.argv.slice(2));
    stop = startPublisher(options);
  } catch (error) {
    console.error(error.message);
    console.error(usage.trim());
    process.exit(1);
  }

  process.once("SIGINT", () => {
    console.log("\n收到中断信号，停止发布");
    if (stop) stop();
    process.exit(0);
  });
}

main();

#!/usr/bin/env node

const net = require("node:net");

const config = {
  // MQTT 服务器地址，支持 mqtt://host:port，也可以用命令行 --broker 覆盖。
  broker: "mqtt://127.0.0.1:18883",
  // 兼容旧主题：单值对象的递增模拟数据。
  topic: "induforge/mock-data",
  // 新主题：批量变量（数组形式）的模拟数据。
  batchTopic: "induforge/mock-data-batch",
  // 旧主题推送间隔。
  intervalMs: 1000,
  // 新主题（批量）推送间隔。
  batchIntervalMs: 10000,
  clientIdPrefix: "induforge-mock-data",
  baseValue: 256,
  step: 1,
};

const usage = `
Usage:
  node scripts/test/mqtt/mqtt-publish-test.js [--broker mqtt://127.0.0.1:18883] [--topic induforge/mock-data] [--batch-topic induforge/mock-data-batch] [--interval 1000] [--batch-interval 10000]

Examples:
  node scripts/test/mqtt/mqtt-publish-test.js
  node scripts/test/mqtt/mqtt-publish-test.js --broker mqtt://127.0.0.1:18883 --topic induforge/mock-data --interval 500
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
    } else if (key === "batch-topic") {
      options.batchTopic = value;
    } else if (key === "interval") {
      options.intervalMs = Number(value);
    } else if (key === "batch-interval") {
      options.batchIntervalMs = Number(value);
    } else {
      throw new Error(`不支持的参数: ${arg}`);
    }
  }

  if (!options.topic.trim()) {
    throw new Error("--topic 不能为空");
  }
  if (!options.batchTopic.trim()) {
    throw new Error("--batch-topic 不能为空");
  }
  if (!Number.isInteger(options.intervalMs) || options.intervalMs <= 0) {
    throw new Error("--interval 必须是正整数毫秒数");
  }
  if (
    !Number.isInteger(options.batchIntervalMs) ||
    options.batchIntervalMs <= 0
  ) {
    throw new Error("--batch-interval 必须是正整数毫秒数");
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

// OPC UA 质量戳定义（子状态位或主状态码）。
// 工业领域常见的 Quality Code：192 = Good；其他多为 Bad/Uncertain 系列。
const QUALITY_GOOD = 192;
const OPC_QUALITY_CODES = [
  QUALITY_GOOD,
  0, // Bad
  24, // BadConfigurationError
  28, // BadCommunicationError
  36, // BadOutOfService
  64, // Uncertain
  68, // UncertainLastUsableValue
  80, // UncertainSubstituteValue
  84, // UncertainSensorNotAccurate
];

// 每次推送的变量随机数量上限（含）。
const MAX_TAGS = 10;
// Good 质量戳出现概率（其余概率从 OPC 异常码中随机抽）。
const GOOD_QUALITY_PROBABILITY = 0.85;
// 出现"超长浮点"数值的概率（工业场景中累计量/瞬时量常出现长尾小数）。
const LONG_FLOAT_PROBABILITY = 0.2;
// 超长浮点的小数位数范围（含）。
const LONG_FLOAT_MIN_DECIMALS = 6;
const LONG_FLOAT_MAX_DECIMALS = 12;
// 超长浮点的整数部分位数范围（含）。
const LONG_FLOAT_MIN_INT_DIGITS = 2;
const LONG_FLOAT_MAX_INT_DIGITS = 7;

function pickQuality(rng) {
  // 大部分时间使用 Good，少数情况下混入异常质量戳。
  if (rng() < GOOD_QUALITY_PROBABILITY) {
    return QUALITY_GOOD;
  }
  const index = Math.floor(rng() * (OPC_QUALITY_CODES.length - 1)) + 1;
  return OPC_QUALITY_CODES[index];
}

// 生成一个"超长浮点"数字字符串：整数部分 2-7 位，小数部分 6-12 位。
// 工业累计量/瞬时量常出现这种精度，模拟真实数据中的长尾小数。
function buildLongFloatString(rng) {
  const intDigits =
    LONG_FLOAT_MIN_INT_DIGITS +
    Math.floor(rng() * (LONG_FLOAT_MAX_INT_DIGITS - LONG_FLOAT_MIN_INT_DIGITS + 1));
  const decDigits =
    LONG_FLOAT_MIN_DECIMALS +
    Math.floor(rng() * (LONG_FLOAT_MAX_DECIMALS - LONG_FLOAT_MIN_DECIMALS + 1));
  let intPart = "";
  let decPart = "";
  for (let i = 0; i < intDigits; i += 1) {
    intPart += Math.floor(rng() * 10);
  }
  for (let i = 0; i < decDigits; i += 1) {
    decPart += Math.floor(rng() * 10);
  }
  // 去掉整数部分前导 0 风险：如果首位为 0 替换为 1-9 之一。
  if (intPart[0] === "0") {
    intPart = `${1 + Math.floor(rng() * 9)}${intPart.slice(1)}`;
  }
  // 偶尔给个负号，模拟工业现场可能出现的反向计量。
  const sign = rng() < 0.15 ? "-" : "";
  return `${sign}${intPart}.${decPart}`;
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

function buildBatchPayload(sequence) {
  // 序列号作为 V 循环的计数器：1-100，超出后从 1 重新开始。
  const cycleValue = ((sequence - 1) % 100) + 1;
  // 每次推送的变量数量随机 1~10。
  const tagCount = Math.floor(Math.random() * MAX_TAGS) + 1;
  const rng = Math.random;
  const tags = [];
  for (let i = 0; i < tagCount; i += 1) {
    // tag1 ~ tag10 循环复用。
    const tagIndex = (i + sequence) % MAX_TAGS;
    const name = `tag${tagIndex + 1}`;
    // 少数点使用超长浮点（JSON 序列化为数字，避免下游精度截断差异）。
    let value;
    if (rng() < LONG_FLOAT_PROBABILITY) {
      value = Number(buildLongFloatString(rng));
    } else {
      // V 在 1-100 间根据 tag 索引与 cycle 错开，避免所有变量值完全一致。
      value = ((cycleValue + tagIndex) % 100) + 1;
    }
    tags.push({
      N: name,
      V: value,
      T: formatNow(),
      Q: pickQuality(rng),
    });
  }

  return JSON.stringify(tags);
}

function startPublisher(options) {
  const broker = parseBroker(options.broker);
  const socket = net.createConnection(broker.port, broker.host);
  const clientId = `${options.clientIdPrefix}-${Date.now()}`;
  let connected = false;
  // 两个主题各自维护独立的 sequence 计数，避免推送周期不同导致计数偏移。
  let sequence = 0;
  let batchSequence = 0;
  let timer = null;
  let batchTimer = null;

  const stop = () => {
    if (timer) {
      clearInterval(timer);
      timer = null;
    }
    if (batchTimer) {
      clearInterval(batchTimer);
      batchTimer = null;
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
      `已连接 MQTT: ${options.broker} topic=${options.topic} interval=${options.intervalMs}ms batchTopic=${options.batchTopic} batchInterval=${options.batchIntervalMs}ms`,
    );

    // 旧主题：保持原单值对象数据不变，按 intervalMs 周期推送。
    const publishLegacy = () => {
      sequence += 1;
      const payload = buildPayload(sequence, options);
      socket.write(buildPublishPacket(options.topic, payload));
      console.log(`已发布 #${sequence} [${options.topic}]: ${payload}`);
    };

    // 新主题：批量变量（数组形式）数据，按 batchIntervalMs 周期独立推送。
    const publishBatch = () => {
      batchSequence += 1;
      const batchPayload = buildBatchPayload(batchSequence);
      socket.write(buildPublishPacket(options.batchTopic, batchPayload));
      // 仅打印前 200 个字符，避免长数组在控制台刷屏。
      const preview =
        batchPayload.length > 200
          ? `${batchPayload.slice(0, 200)}...`
          : batchPayload;
      console.log(
        `已发布 #${batchSequence} [${options.batchTopic}]: ${preview}`,
      );
    };

    publishLegacy();
    timer = setInterval(publishLegacy, options.intervalMs);

    publishBatch();
    batchTimer = setInterval(publishBatch, options.batchIntervalMs);
  });
  socket.once("timeout", () => {
    fail(new Error("连接 MQTT Broker 超时"));
  });
  socket.once("error", (error) => {
    fail(error);
  });
  socket.once("close", () => {
    if (timer) clearInterval(timer);
    if (batchTimer) clearInterval(batchTimer);
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

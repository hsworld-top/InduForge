# MQTT 模拟数据脚本

本目录提供 MQTT 协议层的模拟数据发布工具，用于联调、演示与接入源验证。所有脚本用 Node.js 实现，不依赖项目内 npm 包，只使用 Node 内置 `net` 模块完成 MQTT 3.1.1 协议直连。

## 目录

- `mqtt-publish-test.js`：单脚本同时向两个主题推送模拟数据。
  - 旧主题 `induforge/mock-data`：单值对象 `{"source","sequence","value","ts"}`，每 1s 一条，保持历史行为不变。
  - 新主题 `induforge/mock-data-batch`：批量数组，元素形如 `{"N","V","T","Q"}`，每 10s 一条。

## 依赖

仅 Node.js 标准库（`node:net`），无第三方依赖。Windows / macOS / Linux 通用。

## 用法

从仓库根目录执行：

```bash
node scripts/test/mqtt/mqtt-publish-test.js
```

按需调整 broker、主题、间隔：

```bash
node scripts/test/mqtt/mqtt-publish-test.js \
  --broker mqtt://127.0.0.1:18883 \
  --topic induforge/mock-data \
  --batch-topic induforge/mock-data-batch \
  --interval 1000 \
  --batch-interval 10000
```

参数：

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `--broker` | `mqtt://127.0.0.1:18883` | MQTT broker 地址，仅支持 `mqtt://` 协议 |
| `--topic` | `induforge/mock-data` | 旧主题，单值对象 |
| `--batch-topic` | `induforge/mock-data-batch` | 新主题，批量数组 |
| `--interval` | `1000` | 旧主题推送间隔（毫秒） |
| `--batch-interval` | `10000` | 新主题推送间隔（毫秒） |

## 数据格式

### 旧主题（向后兼容）

```json
{
  "source": "induforge-mock-data",
  "sequence": 1,
  "value": 257,
  "ts": "2026-06-03 10:00:00"
}
```

`value` 从 `baseValue + sequence * step` 线性递增，便于前端或订阅端观察实时变化。

### 新主题（批量数组）

```json
[
  {
    "N": "tag1",
    "V": 23.5,
    "T": "2026-06-03 10:00:00",
    "Q": 192
  },
  {
    "N": "tag2",
    "V": 0.82,
    "T": "2026-06-03 10:00:00",
    "Q": 192
  }
]
```

字段说明：

| 字段 | 含义 |
| --- | --- |
| `N` | 变量名，命名 `tag1` ~ `tag10` |
| `V` | 当前值；多数情况为 1-100 循环整数，少数情况（20% 概率）混入 6-12 位精度的超长浮点数 |
| `T` | 推送时间戳，`YYYY-MM-DD HH:mm:ss` |
| `Q` | OPC UA 质量戳；85% 概率为 `192`（Good），其余从常见 Bad/Uncertain 子状态码中随机抽 |

每次推送的变量数量随机 1-10 个，命名与数值在循环中错位以避免完全一致。

## 质量戳说明

`Q` 采用工业领域 OPC UA 规范中的 Quality Code 数值：

| 值 | 含义 |
| --- | --- |
| 192 | Good |
| 0 | Bad |
| 24 | BadConfigurationError |
| 28 | BadCommunicationError |
| 36 | BadOutOfService |
| 64 | Uncertain |
| 68 | UncertainLastUsableValue |
| 80 | UncertainSubstituteValue |
| 84 | UncertainSensorNotAccurate |

## 退出

`Ctrl+C` 触发 SIGINT，脚本会发送 MQTT DISCONNECT 包并正常退出。

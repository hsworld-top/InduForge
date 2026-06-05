# Kafka 模拟数据脚本

本目录用于启动一个本地 Kafka 兼容测试环境，并持续产出适合数据中心 Kafka 接入源建点的模拟数据。

脚本只负责“产出数据”。数据中心里的接入源、Topic 映射、字段映射和数据点建模仍建议在前端手工配置与验收。

## 文件

- `start-redpanda.sh`：在 WSL Ubuntu 中启动 Redpanda 容器，提供 Kafka 协议。
- `stop-redpanda.sh`：停止 Redpanda 容器。
- `kafka-produce-test.py`：向测试 Topic 持续写入 JSON 样本。

## 启动 Kafka 容器

在 WSL Ubuntu 中，从仓库根目录执行：

```bash
bash scripts/test/kafka/start-redpanda.sh
```

默认会创建容器 `induforge-test-redpanda`，宿主机 Kafka 端口为 `19092`。

数据中心新建 Kafka 接入源时可填写：

| 字段 | 值 |
| --- | --- |
| 服务器地址 | `127.0.0.1:19092` |
| 主题名称 | `induforge.kafka.telemetry` |
| 消费组名称 | `induforge-datacenter-preview` |
| 起始位置 | `最早位置` 或 `最新位置` |

## 产出模拟数据

在仓库根目录执行：

```bash
python scripts/test/kafka/kafka-produce-test.py
```

如果在 Windows PowerShell 中操作，建议显式通过 WSL 执行：

```powershell
wsl python3 /mnt/d/SVNCode/indu-forge/scripts/test/kafka/kafka-produce-test.py
```

推送固定次数：

```bash
python scripts/test/kafka/kafka-produce-test.py --count 20 --interval 1
```

只推送单设备遥测 Topic：

```bash
python scripts/test/kafka/kafka-produce-test.py --topics telemetry
```

## 默认 Topic

| Topic | 用途 |
| --- | --- |
| `induforge.kafka.telemetry` | 单设备遥测，适合测试普通 JSON 字段路径建点 |
| `induforge.kafka.batch` | 批量变量数组，适合测试数组字段预览和批量变量建模 |
| `induforge.kafka.event` | 事件状态，适合测试字符串、布尔、嵌套对象字段 |

## 建点参考字段

### `induforge.kafka.telemetry`

推荐字段路径：

| 字段路径 | 类型 | 说明 |
| --- | --- | --- |
| `metrics.temperature` | number | 温度 |
| `metrics.pressure` | number | 压力 |
| `metrics.flow` | number | 流量 |
| `metrics.speed` | number | 转速 |
| `metrics.current` | number | 电流 |
| `status.running` | boolean | 是否运行 |
| `status.alarmCode` | number | 报警码 |
| `qualityCode` | number | 质量码 |

### `induforge.kafka.batch`

批量变量在 `values` 数组中，每个元素结构如下：

```json
{
  "N": "temperature",
  "V": 47.21,
  "T": "2026-06-05 10:00:00",
  "Q": 192
}
```

如果前端字段映射暂不支持按数组元素动态展开，可先用 `values` 观察整体样本，或在 `telemetry` Topic 上验证普通字段建点。

### `induforge.kafka.event`

推荐字段路径：

| 字段路径 | 类型 | 说明 |
| --- | --- | --- |
| `eventType` | string | 事件类型 |
| `severity` | string | 事件级别 |
| `payload.ackRequired` | boolean | 是否需要确认 |
| `payload.value` | number | 事件数值 |

## 停止 Kafka 容器

```bash
bash scripts/test/kafka/stop-redpanda.sh
```

## 参数

`kafka-produce-test.py` 常用参数：

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `--container` | `induforge-test-redpanda` | Redpanda 容器名 |
| `--broker` | `127.0.0.1:9092` | 容器内 broker 地址 |
| `--telemetry-topic` | `induforge.kafka.telemetry` | 单设备遥测 Topic |
| `--batch-topic` | `induforge.kafka.batch` | 批量变量 Topic |
| `--event-topic` | `induforge.kafka.event` | 事件 Topic |
| `--topics` | `telemetry,batch,event` | 启用哪些样本类型 |
| `--interval` | `1` | 推送间隔秒数 |
| `--count` | `0` | 每个 Topic 推送次数，0 表示持续推送 |
| `--skip-create-topic` | `false` | 不自动创建 Topic |

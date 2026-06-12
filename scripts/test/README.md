# 模拟数据测试脚本

本目录专门存放开发联调、演示验证时使用的模拟数据脚本。

这些脚本用于手动验证协议接入、消息流、实时数据等场景，不属于自动化单元测试；如果后续需要写单元测试或集成测试，请放到对应模块自己的测试目录。

## MQTT 模拟发布

```bash
node scripts/test/mqtt/mqtt-publish-test.js
```

指定地址、主题和间隔：

```bash
node scripts/test/mqtt/mqtt-publish-test.js --broker mqtt://127.0.0.1:18883 --topic induforge/mock-data --interval 500
```

详细说明见 `mqtt/README.md`。

## Kafka 模拟发布

先在 WSL Ubuntu 中启动本地 Kafka 兼容容器：

```bash
bash scripts/test/kafka/start-redpanda.sh
```

持续推送 Kafka 模拟数据：

```bash
python scripts/test/kafka/kafka-produce-test.py
```

详细说明见 `kafka/README.md`。

## 工业协议模拟设备

在 WSL Ubuntu 中可直接启动；首次启动会自动在仓库根目录创建 `.venv-industrial-sim` 并安装 Python 依赖：

```bash
bash scripts/test/industrial-sim/run.sh modbus
bash scripts/test/industrial-sim/run.sh opcua
bash scripts/test/industrial-sim/run.sh s7
```

需要同时启动三种协议：

```bash
bash scripts/test/industrial-sim/run.sh all
```

详细说明见 `industrial-sim/README.md`。

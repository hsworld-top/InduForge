# 工业协议模拟设备

本目录提供三套独立测试模块，用于模拟一台小型泵站/电机设备，帮助验证 OPC UA、S7、Modbus 接入源的建模、预览、读写、异常诊断和使用体验。

模拟器不改业务服务，也不依赖当前项目语言栈。三个协议模块共享同一套设备状态机，但各自暴露符合协议习惯的点位视图。

## 安装

WSL Ubuntu 中默认无需手动安装依赖；`run.sh` 首次启动时会自动在仓库根目录创建 `.venv-industrial-sim` 并安装 `requirements.txt`。

如需提前安装，或要配合 `--python` 使用自定义解释器，可从仓库根目录执行：

```bash
python3 -m venv .venv-industrial-sim
./.venv-industrial-sim/bin/python -m pip install -r scripts/test/industrial-sim/requirements.txt
```

Windows PowerShell 也可使用：

```powershell
python -m venv .venv-industrial-sim
.\.venv-industrial-sim\Scripts\python -m pip install -r scripts\test\industrial-sim\requirements.txt
```

## 启动

在 WSL Ubuntu 中分别启动需要的协议：

```bash
bash scripts/test/industrial-sim/run.sh modbus --scenario normal
bash scripts/test/industrial-sim/run.sh opcua --scenario normal
bash scripts/test/industrial-sim/run.sh s7 --scenario normal
```

需要同时启动三种协议时：

```bash
bash scripts/test/industrial-sim/run.sh all --scenario normal
```

Windows PowerShell 中一键启动三种协议：

```powershell
.\scripts\test\industrial-sim\run-all.ps1 -Python .\.venv-industrial-sim\Scripts\python.exe -Scenario normal
```

Windows PowerShell 中单独启动：

```powershell
.\.venv-industrial-sim\Scripts\python scripts\test\industrial-sim\industrial_sim\modbus_sim.py --scenario normal
.\.venv-industrial-sim\Scripts\python scripts\test\industrial-sim\industrial_sim\opcua_sim.py --scenario normal
.\.venv-industrial-sim\Scripts\python scripts\test\industrial-sim\industrial_sim\s7_sim.py --scenario normal
```

可用场景：

| 场景 | 作用 |
| --- | --- |
| `normal` | 正常停机，写启动命令后经过启动中再运行 |
| `startup` | 启动中开局，便于观察状态爬升 |
| `alarm` | 高压报警并锁存故障 |
| `noisy` | 过程量带噪声，并周期性出现传感器质量波动 |
| `intermittent` | 周期性模拟通讯质量变差 |

## 连接参数

| 协议 | 地址 |
| --- | --- |
| Modbus TCP | `127.0.0.1:18502`，`unitId=1` |
| OPC UA | `opc.tcp://127.0.0.1:18540/induforge/sim` |
| S7 | `127.0.0.1:18102`，`rack=0`，`slot=1` |

## 通用设备行为

- 运行命令写入后，设备先进入 `starting`，约 5 秒后进入 `running`。
- 运行时压力、流量、转速、电流会逐步爬升，停止后逐步下降。
- 报警会锁存，写复位命令后才清除。
- 急停会强制停机并置报警，解除急停后仍需复位。
- 目标压力和目标转速为可写设定值，写入后影响过程量。

状态码：

| 值 | 状态 |
| --- | --- |
| 0 | stopped |
| 1 | starting |
| 2 | running |
| 3 | fault |
| 4 | estop |
| 5 | maintenance |

报警码为 bitmask：

| bit | 值 | 含义 |
| --- | --- | --- |
| 0 | 1 | 高温 |
| 1 | 2 | 高压 |
| 2 | 4 | 低液位 |
| 3 | 8 | 传感器故障 |
| 4 | 16 | 急停 |

## Modbus 点表

| 区域 | 地址 | 名称 | 类型/缩放 | 读写 |
| --- | --- | --- | --- | --- |
| coil | 0 | 运行命令 | Bool | RW |
| coil | 1 | 报警复位 | Bool 脉冲 | RW |
| coil | 2 | 急停 | Bool | RW |
| coil | 3 | 维护模式 | Bool | RW |
| discrete input | 0 | 运行中 | Bool | R |
| discrete input | 1 | 启动中 | Bool | R |
| discrete input | 2 | 故障 | Bool | R |
| discrete input | 3 | 报警激活 | Bool | R |
| holding register | 0 | 目标压力 | UInt16 / 100 bar | RW |
| holding register | 1 | 目标转速 | UInt16 rpm | RW |
| holding register | 2 | 状态码 | UInt16 | R |
| input register | 0 | 压力 | UInt16 / 100 bar | R |
| input register | 1 | 温度 | UInt16 / 10 C | R |
| input register | 2 | 流量 | UInt16 / 10 m3/h | R |
| input register | 3 | 液位 | UInt16 / 10 % | R |
| input register | 4 | 电机转速 | UInt16 rpm | R |
| input register | 5 | 电流 | UInt16 / 100 A | R |
| input register | 6 | 报警码 | UInt16 bitmask | R |
| input register | 7 | 质量码 | 0 Good / 1 Uncertain / 2 Bad | R |

## OPC UA 点表

根节点：`Objects/InduForgeSim/PumpA`

| 路径 | 类型 | 读写 |
| --- | --- | --- |
| `Status/State` | String | R |
| `Status/StateCode` | Int64 | R |
| `Status/AlarmCode` | Int64 | R |
| `Status/AlarmActive` | Boolean | R |
| `Status/SensorFault` | Boolean | R |
| `Status/CommunicationFlap` | Boolean | R |
| `Telemetry/PressureBar` | Double | R |
| `Telemetry/TemperatureC` | Double | R |
| `Telemetry/FlowM3H` | Double | R |
| `Telemetry/LevelPercent` | Double | R |
| `Telemetry/SpeedRPM` | Double | R |
| `Telemetry/CurrentA` | Double | R |
| `Telemetry/Quality` | String | R |
| `Telemetry/Timestamp` | String | R |
| `Command/RunCommand` | Boolean | RW |
| `Command/ResetCommand` | Boolean pulse | RW |
| `Command/EmergencyStop` | Boolean | RW |
| `Command/MaintenanceMode` | Boolean | RW |
| `Command/TargetPressureBar` | Double | RW |
| `Command/TargetSpeedRPM` | Double | RW |

## S7 点表

S7 服务端暴露 `DB1`、`M`、`I`、`Q` 区。推荐先用 `DB1` 验证。

| 地址 | 名称 | 类型 | 读写 |
| --- | --- | --- | --- |
| `DB1.DBX0.0` | 运行命令 | Bool | RW |
| `DB1.DBX0.1` | 报警复位 | Bool 脉冲 | RW |
| `DB1.DBX0.2` | 急停 | Bool | RW |
| `DB1.DBX0.3` | 维护模式 | Bool | RW |
| `DB1.DBX1.0` | 运行中 | Bool | R |
| `DB1.DBX1.1` | 启动中 | Bool | R |
| `DB1.DBX1.2` | 故障 | Bool | R |
| `DB1.DBX1.3` | 报警激活 | Bool | R |
| `DB1.DBW2` | 状态码 | Int | R |
| `DB1.DBW4` | 报警码 | Word | R |
| `DB1.DBD8` | 压力 | Real bar | R |
| `DB1.DBD12` | 温度 | Real C | R |
| `DB1.DBD16` | 流量 | Real m3/h | R |
| `DB1.DBD20` | 液位 | Real % | R |
| `DB1.DBD24` | 转速 | Real rpm | R |
| `DB1.DBD28` | 电流 | Real A | R |
| `DB1.DBD32` | 目标压力 | Real bar | RW |
| `DB1.DBD36` | 目标转速 | Real rpm | RW |
| `DB1.DBB40` | 质量码 | Byte | R |

## 用它评估接入源

建议按以下路径验收接入源体验：

1. 新建连接，验证连接参数是否直观。
2. OPC UA 浏览节点树，检查节点元数据是否可见。
3. Modbus/S7 手工录入点表，检查地址、类型、缩放是否容易配置。
4. 读取预览值，观察质量码、时间戳、报警码是否能表达。
5. 写启动命令，确认状态从 `starting` 过渡到 `running`。
6. 切换 `alarm`、`noisy`、`intermittent` 场景，检查诊断和错误提示是否足够。

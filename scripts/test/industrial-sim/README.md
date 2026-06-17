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
| Modbus TCP | `127.0.0.1:18502`，`unitId=1/2/3` |
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

Modbus TCP 暴露 `unitId=1/2/3` 三个从站。三个从站使用同一套地址段，数据不同；`normal` 场景下 `unitId=1` 默认停机，`unitId=2` 默认启动中，`unitId=3` 默认运行且带噪声，便于测试从站地址选择和动态数据。

以下地址均为零基地址，适合工作台从“起始地址 + 数量 + 类型 + 字节序”批量生成变量。

| 区域 | 起始地址 | 变量数/寄存器数 | 类型/字节序 | 读写 | 说明 |
| --- | --- | --- | --- | --- | --- |
| coil | 0 | 8 | Bool | RW | 命令位 |
| discrete input | 0 | 16 | Bool | R | 状态位、从站标识、质量位 |
| holding register | 0 | 12 | UInt16 | 0/1 RW，其余 R | 设定值、状态镜像、缩放过程值 |
| holding register | 20 | 8 | Int16 | R | 有符号偏差/修正量 |
| holding register | 100 | 4/8 | Float32 ABCD | R | 目标/状态浮点镜像 |
| holding register | 120 | 4/8 | Float32 BADC | R | 目标/状态浮点镜像 |
| holding register | 140 | 4/8 | Float32 CDAB | R | 目标/状态浮点镜像 |
| holding register | 160 | 4/8 | Float32 DCBA | R | 目标/状态浮点镜像 |
| input register | 0 | 12 | UInt16 | R | 过程量缩放值 |
| input register | 20 | 8 | Int16 | R | 过程偏差有符号值 |
| input register | 100 | 8/16 | Float32 ABCD | R | 过程量浮点值 |
| input register | 120 | 8/16 | Float32 BADC | R | 过程量浮点值 |
| input register | 140 | 8/16 | Float32 CDAB | R | 过程量浮点值 |
| input register | 160 | 8/16 | Float32 DCBA | R | 过程量浮点值 |
| input register | 200 | 5/10 | UInt32 ABCD | R | 运行秒数/累计量 |
| input register | 220 | 5/10 | UInt32 BADC | R | 运行秒数/累计量 |
| input register | 240 | 5/10 | UInt32 CDAB | R | 运行秒数/累计量 |
| input register | 260 | 5/10 | UInt32 DCBA | R | 运行秒数/累计量 |
| input register | 300 | 5/10 | Int32 ABCD | R | 有符号诊断量 |
| input register | 320 | 5/10 | Int32 BADC | R | 有符号诊断量 |
| input register | 340 | 5/10 | Int32 CDAB | R | 有符号诊断量 |
| input register | 360 | 5/10 | Int32 DCBA | R | 有符号诊断量 |

常用起始地址：

| 用途 | 从站 | 区域 | 起始地址 | 变量数/寄存器数 | 类型/字节序 |
| --- | --- | --- | --- | --- | --- |
| 运行/复位/急停/维护命令 | 1/2/3 | coil | 0 | 4 | Bool |
| 状态位 | 1/2/3 | discrete input | 0 | 16 | Bool |
| UInt16 过程量 | 1/2/3 | input register | 0 | 12 | UInt16 |
| Int16 偏差量 | 1/2/3 | input register | 20 | 8 | Int16 |
| Float32 过程量，标准大端 | 1/2/3 | input register | 100 | 8/16 | Float32 ABCD |
| Float32 过程量，字节交换 | 1/2/3 | input register | 120 | 8/16 | Float32 BADC |
| Float32 过程量，字交换 | 1/2/3 | input register | 140 | 8/16 | Float32 CDAB |
| Float32 过程量，全反序 | 1/2/3 | input register | 160 | 8/16 | Float32 DCBA |
| UInt32 累计量，标准大端 | 1/2/3 | input register | 200 | 5/10 | UInt32 ABCD |
| Int32 诊断量，标准大端 | 1/2/3 | input register | 300 | 5/10 | Int32 ABCD |

`coil 0` 为运行命令，`coil 1` 为报警复位脉冲，`coil 2` 为急停，`coil 3` 为维护模式。`holding register 0` 为目标压力，缩放 `value / 100 = bar`；`holding register 1` 为目标转速，单位 rpm。

## OPC UA 点表

根节点：

- 兼容点表：`Objects/InduForgeSim/PumpA`
- 树形浏览导入测试：`Objects/InduForgeSim/Plant01`

`Plant01` 按常见工业层级组织：厂站、工艺区、产线、设备撬、设备、状态/遥测/命令/诊断。变量使用稳定字符串 NodeId，便于导入后重复验证。示例 NodeId：

```text
ns=2;s=industrial-sim.Plant01.Area-Water.Line-01.Skid-Pump-01.Pump-P101.Telemetry.PressureBar
ns=2;s=industrial-sim.Plant01.Area-Water.Line-01.Tank-T101.Telemetry.LevelPercent
ns=2;s=industrial-sim.Plant01.Area-Water.Line-01.Valve-XV101.Command.OpenCommand
ns=2;s=industrial-sim.Plant01.Area-Water.Line-01.Instruments.PIT-101.PV
ns=2;s=industrial-sim.Plant01.Area-Utilities.Power.BusVoltageV
```

浏览导入建议从 `Objects/InduForgeSim/Plant01/Area-Water/Line-01` 开始，检查工作台是否能展示对象层级、变量 NodeId、数据类型、描述和单位属性。多数过程量带 `DataTypeName`、`EngineeringUnit`、`EURange` 属性，用于模拟真实设备常见元数据。

兼容点表：

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

树形浏览代表点：

| 路径 | 类型 | 读写 |
| --- | --- | --- |
| `Plant01/Area-Water/Line-01/Skid-Pump-01/Pump-P101/Status/State` | String | R |
| `Plant01/Area-Water/Line-01/Skid-Pump-01/Pump-P101/Telemetry/PressureBar` | Double | R |
| `Plant01/Area-Water/Line-01/Skid-Pump-01/Pump-P101/Telemetry/FlowM3H` | Double | R |
| `Plant01/Area-Water/Line-01/Skid-Pump-01/Pump-P101/Command/RunCommand` | Boolean | RW |
| `Plant01/Area-Water/Line-01/Skid-Pump-01/Pump-P101/Command/TargetPressureBar` | Double | RW |
| `Plant01/Area-Water/Line-01/Skid-Pump-01/Pump-P102/Status/Ready` | Boolean | R |
| `Plant01/Area-Water/Line-01/Tank-T101/Telemetry/LevelPercent` | Double | R |
| `Plant01/Area-Water/Line-01/Valve-XV101/Telemetry/PositionPercent` | Double | R |
| `Plant01/Area-Water/Line-01/Valve-XV101/Command/OpenCommand` | Boolean | RW |
| `Plant01/Area-Water/Line-01/Instruments/PIT-101/PV` | Double | R |
| `Plant01/Area-Utilities/Power/BusVoltageV` | Double | R |
| `Plant01/Area-Utilities/InstrumentAir/HeaderPressureBar` | Double | R |

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
2. OPC UA 从 `Objects/InduForgeSim/Plant01` 浏览节点树，检查层级、NodeId、描述、单位和量程元数据是否可见。
3. Modbus/S7 手工录入点表，检查地址、类型、缩放是否容易配置。
4. 读取预览值，观察质量码、时间戳、报警码是否能表达。
5. 写启动命令，确认状态从 `starting` 过渡到 `running`。
6. 切换 `alarm`、`noisy`、`intermittent` 场景，检查诊断和错误提示是否足够。

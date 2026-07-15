# 工业连接驱动目录简化设计

## 目标

新建工业连接弹窗按用户熟悉的厂商或标准协议直接查找驱动，移除 `PLC / Fieldbus / OPC` 可见分类层，减少无价值的展开操作和分类歧义。

## 展示结构

驱动目录使用两级树：

```text
厂商或协议族
└─ 具体驱动
```

示例：

```text
Siemens [西门子]
└─ Siemens S7 TCP [S7 以太网]

Modbus
├─ Modbus TCP [以太网]
└─ Modbus RTU [串口]

OPC UA
└─ OPC UA [标准客户端]
```

## 数据模型

- 连接执行和存储继续使用 `protocolFamily + driverId`，不修改后端接口与数据库模型。
- Manifest 中继续保留 `category`，供搜索、后台管理和未来筛选使用。
- `category` 不再生成弹窗中的可见树节点。
- 搜索仍匹配分类、厂商或协议族、中英文驱动名称及 `driverId`，因此输入“PLC”仍可定位相关驱动。

## 交互

- 厂商或协议族节点显示图标、中英文名称和驱动数量。
- 展开后显示可选择的具体驱动。
- 选择具体驱动后，右侧立即加载对应连接 Schema。
- 搜索结果仍保持两级结构，不把匹配项变成平铺列表。

## 验证标准

- 弹窗不再出现 `PLC / Fieldbus / OPC` 顶层节点。
- Siemens、Modbus、OPC UA 成为顶层节点。
- 搜索“PLC”“串口”“opcua.standard”均能命中对应驱动。
- 选择驱动和加载连接配置的现有行为不变。
- 单元测试、TypeScript 类型检查和定向 ESLint 通过。

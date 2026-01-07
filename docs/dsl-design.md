# InduForge 低代码平台 DSL 设计规范

> **版本**: 2.0.0  
> **更新日期**: 2025-12  
> **适用场景**: 工业互联网 (IIoT) / SCADA / 企业级应用

---

## ⚠️ 重构说明

**本文档为 DSL 2.0 设计规范。2026 年 1 月起，设计中心进入重构阶段，采用 Schema v2 规范。**

### 主要变化

| 变化点   | DSL 2.0                      | Schema v2（新）                         |
| -------- | ---------------------------- | --------------------------------------- |
| 数据结构 | `components[]` 数组          | `nodesById` + `pagesById` 规范化        |
| 数据绑定 | `dataSources` + 表达式       | `bindings` + `dataProviders` + 三态隔离 |
| 数据源   | `dataCenter`/`http`/`static` | 数据点绑定（`kind: datapoint`）         |
| 国际化   | `i18n.messages`              | `$i18n` 属性 + 工程级/页面级资源        |
| 主题     | `config.theme`               | `themes.definitions` + CSS 变量         |

### 新文档

- **[Schema 设计](./designer/refactor/schema-design.md)** - 规范化工程 Schema（v2）
- **[数据绑定 v2](./designer/refactor/data-binding-v2.md)** - 三态隔离、Binding 结构
- **[国际化与主题](./designer/refactor/i18n-theme.md)** - i18n、主题系统

> 新项目建议使用 Schema v2 规范。本文档保留作为参考。

---

## 目录

1. [DSL 架构概述](#1-dsl-架构概述)
2. [Page Schema 页面结构](#2-page-schema-页面结构)
3. [Component Schema 组件结构](#3-component-schema-组件结构)
4. [DataSource Schema 数据源](#4-datasource-schema-数据源)
5. [Action Schema 动作系统](#5-action-schema-动作系统)
6. [Expression 表达式系统](#6-expression-表达式系统)
7. [Permissions Schema 权限体系](#7-permissions-schema-权限体系)
8. [Animation Schema 动画系统](#8-animation-schema-动画系统)
9. [Validation Schema 验证规则](#9-validation-schema-验证规则)
10. [I18n Schema 国际化](#10-i18n-schema-国际化)
11. [运行时机制](#11-运行时机制)
12. [最佳实践](#12-最佳实践)

---

## 1. DSL 架构概述

### 1.1 设计原则

- **声明式**: 描述"是什么"而非"怎么做"
- **可序列化**: 纯 JSON 结构，便于存储和传输
- **可扩展**: 支持自定义组件和动作
- **运行时无关**: DSL 与渲染引擎解耦

### 1.2 DSL 模块划分

```
┌─────────────────────────────────────────────────────────────┐
│                      Page Schema                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │    meta     │  │   config    │  │     lifecycle       │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │  variables  │  │ dataSources │  │     permissions     │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                    components[]                       │   │
│  │  ┌────────────────────────────────────────────────┐  │   │
│  │  │  Component Schema                              │  │   │
│  │  │  ├─ style      (样式)                          │  │   │
│  │  │  ├─ props      (属性)                          │  │   │
│  │  │  ├─ bindings   (数据绑定)                      │  │   │
│  │  │  ├─ events     (事件 → Action[])               │  │   │
│  │  │  ├─ animations (动画)                          │  │   │
│  │  │  ├─ conditions (条件渲染)                      │  │   │
│  │  │  └─ children[] (子组件)                        │  │   │
│  │  └────────────────────────────────────────────────┘  │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### 1.3 版本控制

```json
{
  "$schema": "https://induforge.io/schemas/page/2.0.0.json",
  "version": "2.0.0"
}
```

---

## 2. Page Schema 页面结构

### 2.1 完整结构

```json
{
  "$schema": "https://induforge.io/schemas/page/2.0.0.json",
  "version": "2.0.0",

  "meta": {
    "id": "page_monitor_01",
    "name": "生产监控大屏",
    "description": "实时监控生产线状态",
    "tags": ["监控", "大屏"],
    "thumbnail": "assets://thumbnails/page_monitor_01.png"
  },

  "config": {
    "width": 1920,
    "height": 1080,
    "scaleMode": "fit",
    "backgroundColor": "#0d1117",
    "backgroundImage": "assets://images/bg_dark.png",
    "backgroundSize": "cover",
    "gridSize": 10,
    "snapToGrid": true,
    "theme": "dark",
    "responsive": {
      "enabled": false,
      "breakpoints": {
        "mobile": 768,
        "tablet": 1024,
        "desktop": 1920
      }
    }
  },

  "variables": {
    "isLoading": { "type": "boolean", "default": false },
    "currentTab": { "type": "string", "default": "overview" },
    "selectedDevice": { "type": "object", "default": null },
    "alarmCount": { "type": "number", "default": 0 }
  },

  "dataSources": [],

  "lifecycle": {
    "onMounted": [],
    "onUnmounted": [],
    "onActivated": [],
    "onDeactivated": []
  },

  "components": [],

  "permissions": {
    "pageAccess": ["admin", "operator", "viewer"],
    "componentAcl": []
  },

  "i18n": {
    "defaultLocale": "zh-CN",
    "messages": {}
  }
}
```

### 2.2 config 配置项详解

| 字段            | 类型    | 默认值    | 说明                                |
| --------------- | ------- | --------- | ----------------------------------- |
| width           | number  | 1920      | 画布宽度                            |
| height          | number  | 1080      | 画布高度                            |
| scaleMode       | enum    | "fit"     | 缩放模式: fit/fill/fixed/responsive |
| backgroundColor | string  | "#ffffff" | 背景色                              |
| backgroundImage | string  | -         | 背景图(支持 assets:// 协议)         |
| gridSize        | number  | 10        | 网格大小                            |
| snapToGrid      | boolean | true      | 是否吸附网格                        |
| theme           | string  | "light"   | 主题: light/dark/custom             |

### 2.3 variables 变量定义

```json
{
  "variables": {
    "simpleVar": "直接值",

    "typedVar": {
      "type": "number",
      "default": 0,
      "description": "计数器"
    },

    "computedVar": {
      "type": "computed",
      "expression": "{{ vars.count * 2 }}",
      "dependencies": ["count"]
    },

    "persistedVar": {
      "type": "string",
      "default": "",
      "persist": "localStorage",
      "key": "user_preference"
    }
  }
}
```

---

## 3. Component Schema 组件结构

### 3.1 基础结构

```json
{
  "id": "comp_motor_01",
  "type": "IndustrialMotor",
  "name": "1号电机",

  "style": {
    "position": "absolute",
    "left": 100,
    "top": 200,
    "width": 120,
    "height": 120,
    "zIndex": 10,
    "opacity": 1,
    "transform": "rotate(0deg)",
    "cursor": "pointer"
  },

  "props": {
    "status": "stopped",
    "rpm": 0,
    "modelSrc": "assets://models/motor.glb"
  },

  "bindings": {
    "props.rpm": "{{ data.ds_motor.rpm }}",
    "props.status": "{{ data.ds_motor.status === 1 ? 'running' : 'stopped' }}",
    "style.opacity": "{{ vars.isLoading ? 0.5 : 1 }}"
  },

  "events": {
    "click": [],
    "dblclick": [],
    "mouseenter": [],
    "mouseleave": [],
    "contextmenu": []
  },

  "animations": [],

  "conditions": {
    "visible": "{{ vars.showMotor }}",
    "enabled": "{{ !vars.isLoading }}"
  },

  "slots": {},

  "children": [],

  "locked": false,
  "hidden": false,
  "group": null
}
```

### 3.2 复合组件引用

```json
{
  "id": "comp_pump_01",
  "type": "CustomComponent",
  "refId": "cc_standard_pump_v1",
  "name": "1号水泵",

  "props": {
    "title": "循环水泵",
    "showLabel": true
  },

  "bindings": {
    "props.status": "{{ data.ds_pump.status }}",
    "props.flow": "{{ data.ds_pump.flowRate }}"
  },

  "overrides": {
    "comp_inner_label": {
      "style.color": "#ff0000"
    }
  }
}
```

### 3.3 容器组件

```json
{
  "id": "comp_panel_01",
  "type": "Panel",
  "name": "设备面板",

  "props": {
    "title": "设备状态",
    "collapsible": true,
    "defaultCollapsed": false
  },

  "layout": {
    "type": "flex",
    "direction": "column",
    "gap": 10,
    "padding": [16, 16, 16, 16],
    "align": "stretch",
    "justify": "start"
  },

  "children": [
    { "id": "child_1", "type": "Text", "props": { "content": "子组件1" } },
    { "id": "child_2", "type": "Text", "props": { "content": "子组件2" } }
  ]
}
```

### 3.4 循环渲染 (v-for)

```json
{
  "id": "comp_device_list",
  "type": "Container",
  "name": "设备列表",

  "loop": {
    "source": "{{ data.ds_devices.list }}",
    "itemVar": "device",
    "indexVar": "index",
    "keyField": "id"
  },

  "children": [
    {
      "id": "device_card_${index}",
      "type": "DeviceCard",
      "bindings": {
        "props.name": "{{ device.name }}",
        "props.status": "{{ device.status }}",
        "props.value": "{{ device.value }}"
      }
    }
  ]
}
```

### 3.5 插槽机制

```json
{
  "id": "comp_dialog_01",
  "type": "Dialog",
  "props": {
    "title": "设备详情",
    "width": 600
  },

  "slots": {
    "default": [
      { "id": "content_1", "type": "Text", "props": { "content": "主内容" } }
    ],
    "footer": [
      { "id": "btn_cancel", "type": "Button", "props": { "text": "取消" } },
      {
        "id": "btn_confirm",
        "type": "Button",
        "props": { "text": "确定", "type": "primary" }
      }
    ]
  }
}
```

---

## 4. DataSource Schema 数据源

### 4.1 数据源类型

| 类型       | 说明                  | 适用场景             |
| ---------- | --------------------- | -------------------- |
| dataCenter | 数据中心查询/点位订阅 | PLC 数据、数据库查询 |
| http       | REST API 调用         | 第三方接口           |
| websocket  | WebSocket 实时连接    | 实时推送             |
| static     | 静态数据              | 配置数据、字典       |
| computed   | 计算数据源            | 数据聚合、转换       |

### 4.2 dataCenter 类型

```json
{
  "id": "ds_motor_realtime",
  "type": "dataCenter",
  "name": "电机实时数据",

  "config": {
    "sourceType": "tags",
    "connectionId": "conn_plc_01",
    "tags": ["motor_rpm", "motor_status", "motor_temp"],
    "mode": "subscription",
    "interval": 0
  },

  "transformer": "(data) => ({ rpm: data.motor_rpm, status: data.motor_status, temp: data.motor_temp })",

  "errorHandler": "(error) => ({ rpm: 0, status: 'error', temp: 0 })",

  "options": {
    "autoStart": true,
    "retryOnError": true,
    "retryCount": 3,
    "retryInterval": 5000
  }
}
```

```json
{
  "id": "ds_history_data",
  "type": "dataCenter",
  "name": "历史数据查询",

  "config": {
    "sourceType": "query",
    "queryId": "query_device_history",
    "mode": "request",
    "parameters": {
      "deviceId": "{{ vars.selectedDeviceId }}",
      "startTime": "{{ vars.startTime }}",
      "endTime": "{{ vars.endTime }}"
    }
  }
}
```

### 4.3 http 类型

```json
{
  "id": "ds_weather",
  "type": "http",
  "name": "天气数据",

  "config": {
    "url": "https://api.weather.com/v1/current",
    "method": "GET",
    "headers": {
      "Authorization": "Bearer {{ env.WEATHER_API_KEY }}"
    },
    "params": {
      "city": "{{ vars.cityCode }}"
    },
    "timeout": 10000
  },

  "mode": "poll",
  "interval": 300000,

  "transformer": "(res) => ({ temp: res.data.temperature, humidity: res.data.humidity })",

  "errorHandler": "(error) => ({ temp: '--', humidity: '--' })"
}
```

### 4.4 websocket 类型

```json
{
  "id": "ds_realtime_alarm",
  "type": "websocket",
  "name": "实时告警",

  "config": {
    "url": "wss://api.example.com/ws/alarms",
    "protocols": [],
    "heartbeat": {
      "enabled": true,
      "interval": 30000,
      "message": { "type": "ping" }
    },
    "reconnect": {
      "enabled": true,
      "maxRetries": 5,
      "interval": 5000
    }
  },

  "events": {
    "onOpen": [
      {
        "action": "wsSend",
        "payload": { "type": "subscribe", "channel": "alarms" }
      }
    ],
    "onMessage": [
      {
        "action": "setVariable",
        "payload": { "key": "latestAlarm", "value": "{{ $event.data }}" }
      }
    ],
    "onError": [],
    "onClose": []
  }
}
```

### 4.5 static 类型

```json
{
  "id": "ds_status_options",
  "type": "static",
  "name": "状态选项",

  "data": [
    { "value": 0, "label": "停止", "color": "#999" },
    { "value": 1, "label": "运行", "color": "#52c41a" },
    { "value": 2, "label": "故障", "color": "#ff4d4f" },
    { "value": 3, "label": "维护", "color": "#faad14" }
  ]
}
```

### 4.6 computed 类型

```json
{
  "id": "ds_summary",
  "type": "computed",
  "name": "汇总数据",

  "dependencies": ["ds_device_list", "ds_alarm_list"],

  "compute": "(sources) => ({ totalDevices: sources.ds_device_list.length, activeAlarms: sources.ds_alarm_list.filter(a => a.status === 'active').length, runningRate: (sources.ds_device_list.filter(d => d.status === 1).length / sources.ds_device_list.length * 100).toFixed(1) })"
}
```

---

## 5. Action Schema 动作系统

### 5.1 动作类型一览

| 动作类型     | 说明         | 参数                           |
| ------------ | ------------ | ------------------------------ |
| setVariable  | 设置变量     | key, value, merge              |
| executeQuery | 执行数据源   | dataSourceId, params           |
| navigate     | 页面跳转     | path, params, target           |
| openDialog   | 打开弹窗     | dialogId, props                |
| closeDialog  | 关闭弹窗     | dialogId, result               |
| message      | 消息提示     | type, content, duration        |
| confirm      | 确认对话框   | title, content, onOk, onCancel |
| request      | HTTP 请求    | url, method, data              |
| script       | 自定义脚本   | code                           |
| emit         | 触发事件     | event, payload                 |
| delay        | 延迟执行     | duration                       |
| condition    | 条件分支     | if, then, else                 |
| loop         | 循环执行     | items, actions                 |
| parallel     | 并行执行     | actions                        |
| writeTag     | 写入点位     | tagId, value                   |
| refresh      | 刷新数据源   | dataSourceId                   |
| download     | 下载文件     | url, filename                  |
| copy         | 复制到剪贴板 | content                        |
| print        | 打印         | target                         |

### 5.2 基础动作示例

```json
{
  "events": {
    "click": [
      {
        "id": "act_1",
        "action": "setVariable",
        "payload": {
          "key": "isLoading",
          "value": true
        }
      },
      {
        "id": "act_2",
        "action": "executeQuery",
        "payload": {
          "dataSourceId": "ds_device_detail",
          "params": {
            "id": "{{ vars.selectedId }}"
          }
        }
      },
      {
        "id": "act_3",
        "action": "setVariable",
        "payload": {
          "key": "isLoading",
          "value": false
        }
      }
    ]
  }
}
```

### 5.3 条件分支

```json
{
  "id": "act_conditional",
  "action": "condition",
  "payload": {
    "if": "{{ vars.userRole === 'admin' }}",
    "then": [
      {
        "action": "navigate",
        "payload": { "path": "/admin/dashboard" }
      }
    ],
    "else": [
      {
        "action": "message",
        "payload": { "type": "warning", "content": "无权限访问" }
      }
    ]
  }
}
```

### 5.4 循环执行

```json
{
  "id": "act_batch_update",
  "action": "loop",
  "payload": {
    "items": "{{ vars.selectedIds }}",
    "itemVar": "id",
    "actions": [
      {
        "action": "request",
        "payload": {
          "url": "/api/devices/{{ id }}/status",
          "method": "PUT",
          "data": { "status": 1 }
        }
      }
    ]
  }
}
```

### 5.5 并行执行

```json
{
  "id": "act_parallel_load",
  "action": "parallel",
  "payload": {
    "actions": [
      { "action": "executeQuery", "payload": { "dataSourceId": "ds_devices" } },
      { "action": "executeQuery", "payload": { "dataSourceId": "ds_alarms" } },
      { "action": "executeQuery", "payload": { "dataSourceId": "ds_stats" } }
    ],
    "onAllComplete": [
      {
        "action": "setVariable",
        "payload": { "key": "isLoading", "value": false }
      }
    ]
  }
}
```

### 5.6 确认对话框

```json
{
  "id": "act_delete_confirm",
  "action": "confirm",
  "payload": {
    "title": "确认删除",
    "content": "确定要删除设备 {{ vars.selectedDevice.name }} 吗？",
    "type": "warning",
    "onOk": [
      {
        "action": "request",
        "payload": {
          "url": "/api/devices/{{ vars.selectedDevice.id }}",
          "method": "DELETE"
        }
      },
      {
        "action": "message",
        "payload": { "type": "success", "content": "删除成功" }
      },
      {
        "action": "executeQuery",
        "payload": { "dataSourceId": "ds_device_list" }
      }
    ],
    "onCancel": []
  }
}
```

### 5.7 写入点位 (工业场景)

```json
{
  "id": "act_start_motor",
  "action": "writeTag",
  "payload": {
    "connectionId": "conn_plc_01",
    "tagId": "motor_01_cmd",
    "value": 1,
    "confirm": true,
    "confirmMessage": "确定要启动电机吗？"
  }
}
```

---

## 6. Expression 表达式系统

### 6.1 表达式语法

使用 `{{ expression }}` 包裹，支持 JavaScript 表达式。

### 6.2 上下文变量

| 变量          | 说明         | 示例                        |
| ------------- | ------------ | --------------------------- |
| `vars`        | 页面变量     | `{{ vars.isLoading }}`      |
| `data`        | 数据源数据   | `{{ data.ds_motor.rpm }}`   |
| `props`       | 组件属性     | `{{ props.title }}`         |
| `$event`      | 事件对象     | `{{ $event.target.value }}` |
| `$item`       | 循环项       | `{{ $item.name }}`          |
| `$index`      | 循环索引     | `{{ $index }}`              |
| `$prevResult` | 上一动作结果 | `{{ $prevResult.data }}`    |
| `$global`     | 全局变量     | `{{ $global.theme }}`       |
| `$user`       | 当前用户     | `{{ $user.role }}`          |
| `$route`      | 路由信息     | `{{ $route.params.id }}`    |
| `$env`        | 环境变量     | `{{ $env.API_BASE }}`       |

### 6.3 内置函数

```javascript
// 格式化
{
  {
    $format.number(value, 2);
  }
} // 数字格式化
{
  {
    $format.date(date, "YYYY-MM-DD");
  }
} // 日期格式化
{
  {
    $format.currency(value, "CNY");
  }
} // 货币格式化
{
  {
    $format.percent(value);
  }
} // 百分比格式化
{
  {
    $format.fileSize(bytes);
  }
} // 文件大小格式化

// 数组操作
{
  {
    $array.sum(arr, "field");
  }
} // 求和
{
  {
    $array.avg(arr, "field");
  }
} // 平均值
{
  {
    $array.max(arr, "field");
  }
} // 最大值
{
  {
    $array.min(arr, "field");
  }
} // 最小值
{
  {
    $array.groupBy(arr, "field");
  }
} // 分组
{
  {
    $array.sortBy(arr, "field", "desc");
  }
} // 排序
{
  {
    $array.unique(arr, "field");
  }
} // 去重

// 字符串操作
{
  {
    $string.truncate(str, 20);
  }
} // 截断
{
  {
    $string.template(tpl, data);
  }
} // 模板替换

// 条件判断
{
  {
    $if(condition, trueVal, falseVal);
  }
} // 三元表达式
{
  {
    $switch(value, cases, defaultVal);
  }
} // Switch 表达式

// 工业计算
{
  {
    $scale(value, rawMin, rawMax, euMin, euMax);
  }
} // 线性变换
{
  {
    $clamp(value, min, max);
  }
} // 范围限制
{
  {
    $deadband(value, lastValue, threshold);
  }
} // 死区判断
```

### 6.4 表达式示例

```json
{
  "bindings": {
    "props.text": "{{ '温度: ' + $format.number(data.ds_temp.value, 1) + '℃' }}",

    "style.color": "{{ data.ds_temp.value > 80 ? '#ff4d4f' : data.ds_temp.value > 60 ? '#faad14' : '#52c41a' }}",

    "props.percent": "{{ Math.round(data.ds_progress.current / data.ds_progress.total * 100) }}",

    "props.options": "{{ data.ds_devices.list.map(d => ({ label: d.name, value: d.id })) }}",

    "props.disabled": "{{ !$user.permissions.includes('device:control') || vars.isLoading }}"
  }
}
```

---

## 7. Permissions Schema 权限体系

### 7.1 页面级权限

```json
{
  "permissions": {
    "pageAccess": ["admin", "operator"],
    "pageEdit": ["admin"],
    "requireAuth": true,
    "redirectOnDeny": "/403"
  }
}
```

### 7.2 组件级权限

```json
{
  "permissions": {
    "componentAcl": [
      {
        "componentId": "btn_start_motor",
        "visibleFor": ["admin", "operator"],
        "enabledFor": ["admin"],
        "rules": [
          {
            "condition": "{{ $user.department === 'production' }}",
            "permissions": ["visible", "enabled"]
          }
        ]
      },
      {
        "componentId": "panel_admin",
        "visibleFor": ["admin"],
        "enabledFor": ["admin"]
      }
    ]
  }
}
```

### 7.3 数据级权限

```json
{
  "dataSources": [
    {
      "id": "ds_devices",
      "type": "dataCenter",
      "config": {
        "queryId": "query_devices"
      },
      "permissions": {
        "filter": "{{ $user.role === 'admin' ? {} : { departmentId: $user.departmentId } }}"
      }
    }
  ]
}
```

---

## 8. Animation Schema 动画系统

### 8.1 动画定义

```json
{
  "animations": [
    {
      "id": "anim_alarm_flash",
      "trigger": "data_change",
      "condition": "{{ data.ds_device.status === 2 }}",
      "type": "flash",
      "config": {
        "duration": 500,
        "iterations": "infinite",
        "colors": ["#ff4d4f", "transparent"]
      }
    },
    {
      "id": "anim_rotate",
      "trigger": "data_change",
      "condition": "{{ data.ds_motor.status === 1 }}",
      "type": "rotate",
      "config": {
        "duration": 2000,
        "iterations": "infinite",
        "direction": "normal",
        "easing": "linear"
      }
    },
    {
      "id": "anim_enter",
      "trigger": "mount",
      "type": "fadeIn",
      "config": {
        "duration": 300,
        "delay": 0
      }
    }
  ]
}
```

### 8.2 动画类型

| 类型             | 说明     | 配置项                          |
| ---------------- | -------- | ------------------------------- |
| flash            | 闪烁     | colors, duration, iterations    |
| rotate           | 旋转     | duration, direction, iterations |
| scale            | 缩放     | from, to, duration              |
| fadeIn/fadeOut   | 淡入淡出 | duration, delay                 |
| slideIn/slideOut | 滑入滑出 | direction, duration             |
| shake            | 抖动     | intensity, duration             |
| pulse            | 脉冲     | scale, duration                 |
| custom           | 自定义   | keyframes, duration, easing     |

### 8.3 触发条件

| 触发器      | 说明           |
| ----------- | -------------- |
| mount       | 组件挂载时     |
| unmount     | 组件卸载时     |
| data_change | 数据变化时     |
| hover       | 鼠标悬停时     |
| click       | 点击时         |
| focus       | 获得焦点时     |
| visible     | 进入可视区域时 |

---

## 9. Validation Schema 验证规则

### 9.1 表单验证

```json
{
  "id": "input_temperature",
  "type": "InputNumber",
  "props": {
    "label": "目标温度",
    "placeholder": "请输入温度值"
  },

  "validation": {
    "rules": [
      { "required": true, "message": "温度不能为空" },
      { "type": "number", "message": "请输入有效数字" },
      { "min": 0, "max": 100, "message": "温度范围 0-100" },
      {
        "validator": "{{ (value) => value % 5 === 0 }}",
        "message": "温度必须是5的倍数"
      }
    ],
    "trigger": ["change", "blur"],
    "validateFirst": true
  }
}
```

### 9.2 验证规则类型

| 规则                | 说明       | 参数                         |
| ------------------- | ---------- | ---------------------------- |
| required            | 必填       | message                      |
| type                | 类型检查   | string/number/email/url/date |
| min/max             | 数值范围   | min, max, message            |
| minLength/maxLength | 长度范围   | minLength, maxLength         |
| pattern             | 正则匹配   | pattern, message             |
| enum                | 枚举值     | enum[], message              |
| validator           | 自定义函数 | validator, message           |
| asyncValidator      | 异步验证   | asyncValidator, message      |

---

## 10. I18n Schema 国际化

### 10.1 页面级国际化

```json
{
  "i18n": {
    "defaultLocale": "zh-CN",
    "fallbackLocale": "en-US",
    "messages": {
      "zh-CN": {
        "title": "生产监控",
        "status.running": "运行中",
        "status.stopped": "已停止",
        "btn.start": "启动",
        "btn.stop": "停止"
      },
      "en-US": {
        "title": "Production Monitor",
        "status.running": "Running",
        "status.stopped": "Stopped",
        "btn.start": "Start",
        "btn.stop": "Stop"
      }
    }
  }
}
```

### 10.2 使用国际化

```json
{
  "props": {
    "title": "{{ $t('title') }}",
    "statusText": "{{ $t('status.' + (data.ds_device.status === 1 ? 'running' : 'stopped')) }}"
  }
}
```

---

## 11. 运行时机制

### 11.1 生命周期

```
┌─────────────────────────────────────────────────────────────┐
│                      Page Lifecycle                          │
├─────────────────────────────────────────────────────────────┤
│  1. beforeCreate  - 页面实例创建前                           │
│  2. created       - 页面实例创建完成                         │
│  3. beforeMount   - 挂载前，DOM 未渲染                       │
│  4. mounted       - 挂载完成，DOM 已渲染                     │
│  5. activated     - 页面激活（从缓存恢复）                   │
│  6. deactivated   - 页面停用（进入缓存）                     │
│  7. beforeUnmount - 卸载前                                   │
│  8. unmounted     - 卸载完成                                 │
└─────────────────────────────────────────────────────────────┘
```

```json
{
  "lifecycle": {
    "onMounted": [
      {
        "action": "executeQuery",
        "payload": { "dataSourceId": "ds_init_data" }
      },
      {
        "action": "setVariable",
        "payload": { "key": "isReady", "value": true }
      }
    ],
    "onUnmounted": [
      {
        "action": "script",
        "payload": { "code": "console.log('Page unmounted')" }
      }
    ],
    "onActivated": [
      { "action": "refresh", "payload": { "dataSourceId": "ds_realtime" } }
    ]
  }
}
```

### 11.2 数据流

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  DataSource  │────▶│   Bindings   │────▶│  Component   │
│   (数据源)    │     │   (绑定)      │     │   (组件)     │
└──────────────┘     └──────────────┘     └──────────────┘
       ▲                                         │
       │                                         │
       │              ┌──────────────┐           │
       └──────────────│   Actions    │◀──────────┘
                      │   (动作)      │
                      └──────────────┘
```

### 11.3 实时数据订阅流程

```
1. 页面加载 → 解析 dataSources
2. 发现 mode: "subscription" → 建立 WebSocket 连接
3. 发送订阅指令 → { type: "subscribe", tags: [...] }
4. 后端推送数据 → { type: "data", payload: {...} }
5. 更新 data 上下文 → 触发 bindings 重新计算
6. 组件响应式更新
7. 页面卸载 → 发送取消订阅 → 关闭连接
```

### 11.4 并发编辑锁机制

```
┌─────────────────────────────────────────────────────────────┐
│                    Concurrent Edit Lock                      │
├─────────────────────────────────────────────────────────────┤
│  1. 用户A打开页面 → 发送 ACQUIRE_LOCK                        │
│  2. 服务端检查 lockedBy 字段                                 │
│     - NULL/超时 → 锁定成功，返回 LOCK_ACQUIRED               │
│     - 已被锁定 → 返回 LOCK_DENIED + 锁定者信息               │
│  3. 锁定成功 → 进入编辑模式                                  │
│  4. 锁定失败 → 进入只读模式，显示锁定者信息                  │
│  5. 心跳保持 → 每30秒发送 HEARTBEAT                          │
│  6. 保存/关闭 → 发送 RELEASE_LOCK                            │
│  7. 异常断开 → 服务端60秒超时自动释放                        │
└─────────────────────────────────────────────────────────────┘
```

### 11.5 资源引用协议

| 协议         | 说明        | 示例                              |
| ------------ | ----------- | --------------------------------- |
| `assets://`  | 项目资源库  | `assets://images/bg.png`          |
| `global://`  | 全局资源库  | `global://icons/motor.svg`        |
| `http(s)://` | 外部 URL    | `https://cdn.example.com/img.png` |
| `data:`      | Base64 内联 | `data:image/png;base64,...`       |

运行时解析：

```javascript
// assets://images/bg.png
// → /api/projects/{projectId}/assets/images/bg.png
// → https://cdn.example.com/projects/{projectId}/images/bg.png
```

---

## 12. 最佳实践

### 12.1 性能优化

1. **数据源优化**

   - 合理设置轮询间隔，避免过于频繁
   - 使用 subscription 模式替代高频轮询
   - 启用缓存减少重复请求

2. **组件优化**

   - 避免深层嵌套（建议不超过 5 层）
   - 大列表使用虚拟滚动
   - 复杂计算使用 computed 数据源

3. **表达式优化**
   - 避免在表达式中进行复杂计算
   - 使用 computed 变量缓存计算结果

### 12.2 安全建议

1. **脚本沙箱**

   - customScript 在沙箱环境执行
   - 禁止访问 window、document 等全局对象
   - 限制执行时间（默认 5 秒超时）

2. **数据验证**

   - 所有用户输入必须验证
   - 写入点位前进行权限检查
   - 敏感操作需要二次确认

3. **权限控制**
   - 最小权限原则
   - 组件级权限细粒度控制
   - 数据级权限过滤

### 12.3 命名规范

| 类型   | 前缀   | 示例                  |
| ------ | ------ | --------------------- |
| 页面   | page\_ | page_monitor_01       |
| 组件   | comp\_ | comp_motor_01         |
| 数据源 | ds\_   | ds_realtime_data      |
| 变量   | -      | isLoading, currentTab |
| 动作   | act\_  | act_submit_form       |
| 动画   | anim\_ | anim_flash_alarm      |

### 12.4 DSL 版本迁移

当 DSL 版本升级时，需要提供迁移脚本：

```javascript
// migrations/1.0.0_to_2.0.0.js
export function migrate(oldSchema) {
  const newSchema = { ...oldSchema };

  // 迁移 variables 格式
  if (oldSchema.variables) {
    newSchema.variables = Object.entries(oldSchema.variables).reduce(
      (acc, [key, value]) => {
        acc[key] =
          typeof value === "object"
            ? value
            : { type: typeof value, default: value };
        return acc;
      },
      {}
    );
  }

  // 迁移 dataSources 格式
  // ...

  newSchema.version = "2.0.0";
  return newSchema;
}
```

---

## 附录 A: 完整页面示例

```json
{
  "$schema": "https://induforge.io/schemas/page/2.0.0.json",
  "version": "2.0.0",

  "meta": {
    "id": "page_motor_monitor",
    "name": "电机监控",
    "description": "实时监控电机运行状态"
  },

  "config": {
    "width": 1920,
    "height": 1080,
    "scaleMode": "fit",
    "backgroundColor": "#0d1117",
    "theme": "dark"
  },

  "variables": {
    "selectedMotorId": { "type": "string", "default": null },
    "isControlling": { "type": "boolean", "default": false }
  },

  "dataSources": [
    {
      "id": "ds_motors",
      "type": "dataCenter",
      "config": {
        "sourceType": "tags",
        "connectionId": "conn_plc_main",
        "tags": [
          "motor_01_rpm",
          "motor_01_status",
          "motor_02_rpm",
          "motor_02_status"
        ],
        "mode": "subscription"
      }
    }
  ],

  "lifecycle": {
    "onMounted": [
      {
        "action": "setVariable",
        "payload": { "key": "selectedMotorId", "value": "motor_01" }
      }
    ]
  },

  "components": [
    {
      "id": "comp_title",
      "type": "Text",
      "style": { "position": "absolute", "left": 20, "top": 20 },
      "props": { "content": "电机监控系统", "fontSize": 24, "color": "#fff" }
    },
    {
      "id": "comp_motor_01",
      "type": "IndustrialMotor",
      "style": {
        "position": "absolute",
        "left": 100,
        "top": 150,
        "width": 200,
        "height": 200
      },
      "props": { "title": "1号电机" },
      "bindings": {
        "props.rpm": "{{ data.ds_motors.motor_01_rpm }}",
        "props.status": "{{ data.ds_motors.motor_01_status }}"
      },
      "events": {
        "click": [
          {
            "action": "setVariable",
            "payload": { "key": "selectedMotorId", "value": "motor_01" }
          }
        ]
      },
      "animations": [
        {
          "trigger": "data_change",
          "condition": "{{ data.ds_motors.motor_01_status === 1 }}",
          "type": "rotate",
          "config": { "duration": 2000, "iterations": "infinite" }
        }
      ]
    },
    {
      "id": "comp_btn_start",
      "type": "Button",
      "style": { "position": "absolute", "left": 100, "top": 400 },
      "props": { "text": "启动", "type": "primary" },
      "bindings": {
        "props.loading": "{{ vars.isControlling }}",
        "props.disabled": "{{ data.ds_motors[vars.selectedMotorId + '_status'] === 1 }}"
      },
      "events": {
        "click": [
          {
            "action": "setVariable",
            "payload": { "key": "isControlling", "value": true }
          },
          {
            "action": "writeTag",
            "payload": {
              "tagId": "{{ vars.selectedMotorId + '_cmd' }}",
              "value": 1,
              "confirm": true,
              "confirmMessage": "确定要启动电机吗？"
            }
          },
          { "action": "delay", "payload": { "duration": 1000 } },
          {
            "action": "setVariable",
            "payload": { "key": "isControlling", "value": false }
          }
        ]
      }
    }
  ],

  "permissions": {
    "pageAccess": ["admin", "operator", "viewer"],
    "componentAcl": [
      {
        "componentId": "comp_btn_start",
        "visibleFor": ["admin", "operator"],
        "enabledFor": ["admin"]
      }
    ]
  }
}
```

---

## 附录 B: 数据库表与 DSL 字段映射

| 数据库表                                | 存储内容     | DSL 对应                 |
| --------------------------------------- | ------------ | ------------------------ |
| design_pages.schemaContent              | 完整页面 DSL | 整个 JSON                |
| design_pages.pageConfig                 | 页面配置     | config 节点              |
| design_pages.variables                  | 变量定义     | variables 节点           |
| design_pages.dataSources                | 数据源配置   | dataSources 节点         |
| design_project_settings.globalVariables | 全局变量     | $global 上下文           |
| design_project_settings.globalStyles    | 全局样式     | 主题配置                 |
| design_custom_components.schemaContent  | 复合组件 DSL | CustomComponent 引用     |
| design_datasources.config               | 数据源模板   | 可被页面引用             |
| design_roles                            | 运行时角色   | permissions.roles        |
| design_permissions                      | 组件权限     | permissions.componentAcl |

---

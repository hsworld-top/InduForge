# 动作系统（Action System）

本文档定义 Designer 的动作系统，包括基础动作、控制流动作和工业场景专用动作。

## 1. 概述

动作是响应用户交互或系统事件时执行的操作单元。动作可以：

- 单独执行
- 串联执行（顺序）
- 条件执行（分支）
- 循环执行
- 并行执行

## 2. 动作结构

### 2.1 基础结构

```json
{
  "id": "act_xxx",
  "type": "actionType",
  "config": {
    /* 动作特定配置 */
  },
  "permissions": {
    "allowRoles": ["admin"]
  }
}
```

### 2.2 事件绑定

```json
{
  "events": {
    "click": [
      { "type": "setVar", "config": { "name": "isLoading", "value": true } },
      { "type": "callApi", "config": { "endpoint": "/api/data" } },
      { "type": "setVar", "config": { "name": "isLoading", "value": false } }
    ]
  }
}
```

## 3. 动作类型总览

### 3.1 导航与页面

| 类型          | 说明     | 配置                        |
| ------------- | -------- | --------------------------- |
| `navigate`    | 页面跳转 | pageId, params, replace     |
| `openUrl`     | 打开链接 | url, target(\_blank/\_self) |
| `openDialog`  | 打开弹窗 | dialogId, props             |
| `closeDialog` | 关闭弹窗 | dialogId, result            |
| `back`        | 返回上页 | -                           |

### 3.2 认证与授权

| 类型     | 说明 | 配置            |
| -------- | ---- | --------------- |
| `login`  | 登录 | successRedirect |
| `logout` | 登出 | redirectTo      |

### 3.3 变量操作

| 类型       | 说明     | 配置                      |
| ---------- | -------- | ------------------------- |
| `setVar`   | 设置变量 | scope, name, value, merge |
| `resetVar` | 重置变量 | scope, name               |

### 3.4 数据操作

| 类型       | 说明       | 配置                            |
| ---------- | ---------- | ------------------------------- |
| `callApi`  | 调用接口   | endpoint, method, body, headers |
| `refresh`  | 刷新数据源 | dataSourceId                    |
| `writeTag` | 写入标签   | path, value, confirm            |

### 3.5 用户反馈

| 类型      | 说明       | 配置                           |
| --------- | ---------- | ------------------------------ |
| `notify`  | 显示通知   | type, message, duration        |
| `message` | 消息提示   | type, content, duration        |
| `confirm` | 确认对话框 | title, content, onOk, onCancel |
| `loading` | 显示加载   | show, text                     |

### 3.6 工具类

| 类型       | 说明         | 配置             |
| ---------- | ------------ | ---------------- |
| `copy`     | 复制到剪贴板 | content          |
| `download` | 下载文件     | url, filename    |
| `print`    | 打印         | target, options  |
| `emit`     | 触发事件     | event, payload   |
| `script`   | 自定义脚本   | code（沙箱执行） |

### 3.7 控制流

| 类型        | 说明     | 配置                    |
| ----------- | -------- | ----------------------- |
| `condition` | 条件分支 | if, then, else          |
| `loop`      | 循环执行 | items, itemVar, actions |
| `parallel`  | 并行执行 | actions, onAllComplete  |
| `delay`     | 延迟执行 | duration                |
| `sequence`  | 顺序执行 | actions                 |

## 4. 基础动作详解

### 4.1 navigate（页面跳转）

```json
{
  "type": "navigate",
  "config": {
    "pageId": "page_detail",
    "params": {
      "id": "{{ $props.deviceId }}",
      "tab": "overview"
    },
    "replace": false
  }
}
```

### 4.2 setVar（设置变量）

```json
{
  "type": "setVar",
  "config": {
    "scope": "page",
    "name": "selectedDevice",
    "value": "{{ $event.row }}",
    "merge": false
  }
}
```

**scope 值**：

- `page` - 页面级变量
- `global` - 全局变量

**merge**：当值为对象时，`true` 表示合并，`false` 表示替换。

### 4.3 callApi（调用接口）

```json
{
  "type": "callApi",
  "config": {
    "endpoint": "/api/devices/{{ $vars.page.deviceId }}",
    "method": "PUT",
    "body": {
      "status": "{{ $vars.page.newStatus }}"
    },
    "headers": {
      "X-Custom-Header": "value"
    },
    "onSuccess": [
      {
        "type": "notify",
        "config": { "type": "success", "message": "操作成功" }
      },
      { "type": "refresh", "config": { "dataSourceId": "ds_devices" } }
    ],
    "onError": [
      {
        "type": "notify",
        "config": { "type": "error", "message": "{{ $error.message }}" }
      }
    ]
  }
}
```

### 4.4 writeTag（写入标签）- 工业场景

```json
{
  "type": "writeTag",
  "config": {
    "path": "mqtt.EMQX.控制组.motor_cmd",
    "value": 1,
    "confirm": true,
    "confirmConfig": {
      "title": "确认操作",
      "content": "确定要启动电机吗？",
      "type": "warning"
    }
  },
  "permissions": {
    "allowRoles": ["admin", "operator"]
  }
}
```

### 4.5 notify（显示通知）

```json
{
  "type": "notify",
  "config": {
    "type": "success",
    "message": "保存成功",
    "duration": 3000,
    "position": "top-right"
  }
}
```

**type 值**：`success` | `error` | `warning` | `info`

### 4.6 confirm（确认对话框）

```json
{
  "type": "confirm",
  "config": {
    "title": "确认删除",
    "content": "确定要删除设备 {{ $vars.page.selectedDevice.name }} 吗？",
    "type": "warning",
    "okText": "确定",
    "cancelText": "取消",
    "onOk": [
      {
        "type": "callApi",
        "config": {
          "endpoint": "/api/devices/{{ $vars.page.selectedDevice.id }}",
          "method": "DELETE"
        }
      },
      {
        "type": "notify",
        "config": { "type": "success", "message": "删除成功" }
      },
      { "type": "refresh", "config": { "dataSourceId": "ds_devices" } }
    ],
    "onCancel": []
  }
}
```

### 4.7 openDialog（打开弹窗）

```json
{
  "type": "openDialog",
  "config": {
    "dialogId": "dialog_device_edit",
    "props": {
      "deviceId": "{{ $vars.page.selectedDevice.id }}",
      "mode": "edit"
    },
    "onClose": [
      { "type": "refresh", "config": { "dataSourceId": "ds_devices" } }
    ]
  }
}
```

## 5. 控制流动作详解

### 5.1 condition（条件分支）

```json
{
  "type": "condition",
  "config": {
    "if": "{{ $vars.global.currentUser.role === 'admin' }}",
    "then": [
      { "type": "navigate", "config": { "pageId": "page_admin_dashboard" } }
    ],
    "else": [
      { "type": "navigate", "config": { "pageId": "page_user_dashboard" } }
    ]
  }
}
```

**多条件分支**：

```json
{
  "type": "condition",
  "config": {
    "branches": [
      {
        "if": "{{ $vars.page.status === 'running' }}",
        "then": [{ "type": "notify", "config": { "message": "设备运行中" } }]
      },
      {
        "if": "{{ $vars.page.status === 'stopped' }}",
        "then": [{ "type": "notify", "config": { "message": "设备已停止" } }]
      },
      {
        "if": "{{ $vars.page.status === 'error' }}",
        "then": [
          {
            "type": "notify",
            "config": { "type": "error", "message": "设备故障" }
          }
        ]
      }
    ],
    "default": [{ "type": "notify", "config": { "message": "状态未知" } }]
  }
}
```

### 5.2 loop（循环执行）

```json
{
  "type": "loop",
  "config": {
    "items": "{{ $vars.page.selectedIds }}",
    "itemVar": "id",
    "indexVar": "index",
    "actions": [
      {
        "type": "callApi",
        "config": {
          "endpoint": "/api/devices/{{ id }}/status",
          "method": "PUT",
          "body": { "status": 1 }
        }
      }
    ],
    "onComplete": [
      { "type": "notify", "config": { "message": "批量操作完成" } }
    ],
    "continueOnError": true
  }
}
```

**配置说明**：

| 字段            | 说明                           |
| --------------- | ------------------------------ |
| items           | 迭代数组（表达式）             |
| itemVar         | 当前项变量名（默认 `$item`）   |
| indexVar        | 索引变量名（默认 `$index`）    |
| actions         | 每次迭代执行的动作             |
| onComplete      | 循环完成后执行                 |
| continueOnError | 单项出错是否继续（默认 false） |

### 5.3 parallel（并行执行）

```json
{
  "type": "parallel",
  "config": {
    "actions": [
      { "type": "refresh", "config": { "dataSourceId": "ds_devices" } },
      { "type": "refresh", "config": { "dataSourceId": "ds_alarms" } },
      { "type": "refresh", "config": { "dataSourceId": "ds_stats" } }
    ],
    "onAllComplete": [
      { "type": "setVar", "config": { "name": "isLoading", "value": false } }
    ],
    "onAnyError": [
      {
        "type": "notify",
        "config": { "type": "error", "message": "部分数据加载失败" }
      }
    ],
    "timeout": 30000
  }
}
```

### 5.4 delay（延迟执行）

```json
{
  "type": "delay",
  "config": {
    "duration": 2000,
    "then": [
      { "type": "setVar", "config": { "name": "showTip", "value": false } }
    ]
  }
}
```

### 5.5 sequence（顺序执行）- 带错误处理

```json
{
  "type": "sequence",
  "config": {
    "actions": [
      { "type": "setVar", "config": { "name": "isLoading", "value": true } },
      { "type": "callApi", "config": { "endpoint": "/api/data" } },
      { "type": "setVar", "config": { "name": "isLoading", "value": false } }
    ],
    "onError": [
      { "type": "setVar", "config": { "name": "isLoading", "value": false } },
      {
        "type": "notify",
        "config": { "type": "error", "message": "{{ $error.message }}" }
      }
    ]
  }
}
```

## 6. 特殊上下文变量

动作执行时可访问的上下文：

| 变量           | 说明                 | 可用场景                |
| -------------- | -------------------- | ----------------------- |
| `$event`       | 触发事件对象         | 所有事件触发的动作      |
| `$props`       | 组件属性             | 所有动作                |
| `$vars.page`   | 页面变量             | 所有动作                |
| `$vars.global` | 全局变量             | 所有动作                |
| `$dp`          | 数据点值             | 所有动作                |
| `$item`        | 当前循环项           | loop 内的动作           |
| `$index`       | 当前循环索引         | loop 内的动作           |
| `$prevResult`  | 上一个动作的返回结果 | sequence/顺序执行的动作 |
| `$error`       | 错误信息             | onError 回调            |
| `$result`      | API/动作执行结果     | onSuccess 回调          |

## 7. 工业场景示例

### 7.1 设备启停控制

```json
{
  "events": {
    "click": [
      {
        "type": "confirm",
        "config": {
          "title": "启动确认",
          "content": "确定要启动 {{ $props.deviceName }} 吗？",
          "type": "warning",
          "onOk": [
            {
              "type": "setVar",
              "config": { "name": "isControlling", "value": true }
            },
            {
              "type": "writeTag",
              "config": {
                "path": "{{ $props.controlPath }}",
                "value": 1
              }
            },
            {
              "type": "delay",
              "config": {
                "duration": 3000,
                "then": [
                  {
                    "type": "condition",
                    "config": {
                      "if": "{{ $dp[$props.statusPath] === 1 }}",
                      "then": [
                        {
                          "type": "notify",
                          "config": { "type": "success", "message": "启动成功" }
                        }
                      ],
                      "else": [
                        {
                          "type": "notify",
                          "config": {
                            "type": "error",
                            "message": "启动超时，请检查设备"
                          }
                        }
                      ]
                    }
                  },
                  {
                    "type": "setVar",
                    "config": { "name": "isControlling", "value": false }
                  }
                ]
              }
            }
          ]
        }
      }
    ]
  }
}
```

### 7.2 批量设备操作

```json
{
  "events": {
    "click": [
      {
        "type": "confirm",
        "config": {
          "title": "批量操作",
          "content": "确定要停止选中的 {{ $vars.page.selectedDevices.length }} 台设备吗？",
          "onOk": [
            {
              "type": "setVar",
              "config": { "name": "batchProgress", "value": 0 }
            },
            {
              "type": "loop",
              "config": {
                "items": "{{ $vars.page.selectedDevices }}",
                "itemVar": "device",
                "actions": [
                  {
                    "type": "writeTag",
                    "config": {
                      "path": "{{ device.controlPath }}",
                      "value": 0
                    }
                  },
                  {
                    "type": "setVar",
                    "config": {
                      "name": "batchProgress",
                      "value": "{{ ($index + 1) / $vars.page.selectedDevices.length * 100 }}"
                    }
                  },
                  {
                    "type": "delay",
                    "config": { "duration": 500 }
                  }
                ],
                "onComplete": [
                  {
                    "type": "notify",
                    "config": { "type": "success", "message": "批量停止完成" }
                  },
                  {
                    "type": "setVar",
                    "config": { "name": "selectedDevices", "value": [] }
                  }
                ],
                "continueOnError": true
              }
            }
          ]
        }
      }
    ]
  }
}
```

### 7.3 参数设定

```json
{
  "events": {
    "click": [
      {
        "type": "condition",
        "config": {
          "if": "{{ $vars.page.temperature < 0 || $vars.page.temperature > 100 }}",
          "then": [
            {
              "type": "notify",
              "config": { "type": "error", "message": "温度范围 0-100" }
            }
          ],
          "else": [
            {
              "type": "confirm",
              "config": {
                "title": "参数确认",
                "content": "确定将目标温度设为 {{ $vars.page.temperature }}℃ 吗？",
                "onOk": [
                  {
                    "type": "writeTag",
                    "config": {
                      "path": "mqtt.EMQX.控制组.target_temp",
                      "value": "{{ $vars.page.temperature }}"
                    }
                  },
                  {
                    "type": "notify",
                    "config": { "type": "success", "message": "设定成功" }
                  }
                ]
              }
            }
          ]
        }
      }
    ]
  }
}
```

## 8. 权限控制

### 8.1 动作级权限

```json
{
  "type": "writeTag",
  "config": {
    "path": "mqtt.EMQX.控制组.emergency_stop",
    "value": 1
  },
  "permissions": {
    "allowRoles": ["admin", "operator"],
    "denyRoles": ["viewer"]
  }
}
```

### 8.2 条件权限

```json
{
  "type": "writeTag",
  "config": {
    "path": "{{ $props.controlPath }}",
    "value": 1
  },
  "permissions": {
    "condition": "{{ $vars.global.currentUser.department === $props.deviceDepartment }}"
  }
}
```

## 9. 动作执行器实现

### 9.1 ActionExecutor 接口

```typescript
interface ActionExecutor {
  execute(action: Action, context: ActionContext): Promise<any>;
  registerHandler(type: string, handler: ActionHandler): void;
}

interface ActionContext {
  $event: any;
  $props: Record<string, any>;
  $vars: { page: Record<string, any>; global: Record<string, any> };
  $dp: Record<string, any>;
  $item?: any;
  $index?: number;
  $prevResult?: any;
}

type ActionHandler = (config: any, context: ActionContext) => Promise<any>;
```

### 9.2 内置处理器

```typescript
// 注册内置动作处理器
executor.registerHandler("setVar", async (config, ctx) => {
  const value = evaluateExpression(config.value, ctx);
  if (config.scope === "global") {
    globalVarStore.set(config.name, value, config.merge);
  } else {
    pageVarStore.set(config.name, value, config.merge);
  }
});

executor.registerHandler("condition", async (config, ctx) => {
  const condition = evaluateExpression(config.if, ctx);
  const actions = condition ? config.then : config.else;
  for (const action of actions || []) {
    await executor.execute(action, ctx);
  }
});

executor.registerHandler("loop", async (config, ctx) => {
  const items = evaluateExpression(config.items, ctx);
  for (let i = 0; i < items.length; i++) {
    const loopCtx = {
      ...ctx,
      [config.itemVar || "$item"]: items[i],
      [config.indexVar || "$index"]: i,
    };
    try {
      for (const action of config.actions) {
        await executor.execute(action, loopCtx);
      }
    } catch (e) {
      if (!config.continueOnError) throw e;
    }
  }
  if (config.onComplete) {
    for (const action of config.onComplete) {
      await executor.execute(action, ctx);
    }
  }
});
```

## 10. 测试要点

- [ ] 基础动作执行正确性
- [ ] 条件分支逻辑
- [ ] 循环执行与错误处理
- [ ] 并行执行与超时
- [ ] 表达式求值
- [ ] 权限校验
- [ ] 错误回调触发
- [ ] 上下文变量传递

---

**相关文档**：

- [Schema 设计](./schema-design.md)
- [表达式引擎](./expression-engine.md)
- [数据绑定 v2](./data-binding-v2.md)

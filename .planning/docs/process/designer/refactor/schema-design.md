# Schema 设计（v2）

本文档定义 Designer 工程的规范化 Schema 结构。

## 1. 设计原则

1. **规范化存储**：采用 `pagesById + nodesById` 扁平结构，避免深层嵌套
2. **ID 引用**：父子关系通过 ID 数组表达，便于移动操作
3. **配置分离**：布局、绑定、权限、事件各自独立
4. **可扩展**：预留扩展点，支持版本迁移

## 2. 顶层结构

```json
{
  "schemaVersion": 2,

  "project": {
    "projectId": "proj_xxx",
    "name": "产线A看板",
    "uiTargets": ["pc", "bigscreen"],
    "createdAt": 1736123000,
    "updatedAt": 1736123999
  },

  "securityDecl": {
    "roles": ["admin", "operator", "viewer"],
    "mode": "nodeLocalAuth"
  },

  "entry": {
    "loginPageId": "page_login",
    "homePageId": "page_home"
  },

  "dataProviders": {
    "dc_main": {
      "type": "dataCenter",
      "description": "主数据中心",
      "capabilities": ["mqtt", "db", "calc"]
    }
  },

  "vars": {
    "global": {
      "currentUser": { "type": "object", "default": null },
      "theme": { "type": "string", "default": "dark" }
    },
    "pages": {
      "page_home": {
        "selectedDeviceId": { "type": "string", "default": "" }
      }
    }
  },

  "assetsById": {
    "img_logo": {
      "type": "image",
      "uri": "assets/logo.png",
      "storageMode": "packaged",
      "contentHash": "abc123"
    }
  },

  "pagesById": {
    /* 页面定义 */
  },
  "nodesById": {
    /* 节点定义 */
  }
}
```

## 3. 页面定义（pagesById）

```json
{
  "pagesById": {
    "page_home": {
      "id": "page_home",
      "name": "首页",
      "path": "/home",
      "target": "pc",
      "logicalId": "logic_home",
      "isDefaultTarget": true,
      "rootNodeId": "node_root_home",
      "config": {
        "width": 1920,
        "height": 1080,
        "fitMode": "contain",
        "background": {
          "kind": "color",
          "value": "#1a1a2e"
        }
      }
    },
    "page_home_bigscreen": {
      "id": "page_home_bigscreen",
      "name": "首页-大屏",
      "path": "/home",
      "target": "bigscreen",
      "logicalId": "logic_home",
      "isDefaultTarget": false,
      "rootNodeId": "node_root_home_bs",
      "config": {
        "width": 3840,
        "height": 2160,
        "fitMode": "contain"
      }
    }
  }
}
```

### 页面字段说明

| 字段            | 类型       | 说明                        |
| --------------- | ---------- | --------------------------- |
| id              | string     | 页面唯一 ID                 |
| name            | string     | 页面名称                    |
| path            | string     | 路由路径                    |
| target          | string     | 目标端：pc/bigscreen/mobile |
| logicalId       | string     | 逻辑页面 ID（同路由共享）   |
| isDefaultTarget | boolean    | 是否默认视图                |
| rootNodeId      | string     | 根节点 ID                   |
| config          | PageConfig | 页面配置                    |

## 4. 页面生命周期（Lifecycle）

页面支持生命周期钩子，用于初始化数据、清理资源等：

```json
{
  "pagesById": {
    "page_home": {
      "id": "page_home",
      "name": "首页",
      "path": "/home",
      "rootNodeId": "node_root_home",
      "config": {
        /* ... */
      },
      "lifecycle": {
        "onMounted": [
          { "type": "callApi", "config": { "endpoint": "/api/init" } },
          { "type": "setVar", "config": { "name": "isReady", "value": true } }
        ],
        "onUnmounted": [
          { "type": "setVar", "config": { "name": "isReady", "value": false } }
        ],
        "onActivated": [
          { "type": "refresh", "config": { "dataSourceId": "ds_realtime" } }
        ],
        "onDeactivated": []
      }
    }
  }
}
```

### 生命周期钩子说明

| 钩子          | 触发时机           | 使用场景             |
| ------------- | ------------------ | -------------------- |
| onMounted     | 页面挂载完成       | 初始化数据、启动订阅 |
| onUnmounted   | 页面卸载           | 清理资源、取消订阅   |
| onActivated   | 从缓存恢复（激活） | 刷新数据、恢复状态   |
| onDeactivated | 进入缓存（停用）   | 暂停定时器、保存状态 |

## 5. 节点定义（nodesById）

```json
{
  "nodesById": {
    "node_root_home": {
      "id": "node_root_home",
      "type": "FlexContainer",
      "props": {
        "direction": "column",
        "gap": 16
      },
      "style": {
        "padding": "16px",
        "background": { "kind": "color", "value": "#1a1a2e" }
      },
      "layoutItem": null,
      "bindings": {},
      "permissions": {},
      "events": {},
      "animations": [],
      "conditions": {},
      "children": ["node_header", "node_content"]
    },

    "node_header": {
      "id": "node_header",
      "type": "FreeContainer",
      "props": { "height": 80 },
      "style": {},
      "layoutItem": { "flex": { "grow": 0, "shrink": 0 } },
      "bindings": {},
      "permissions": {},
      "events": {},
      "children": ["node_logo", "node_title"]
    },

    "node_logo": {
      "id": "node_logo",
      "type": "Image",
      "props": {
        "src": { "kind": "asset", "assetId": "img_logo" }
      },
      "style": {},
      "layoutItem": {
        "free": {
          "mode": "constraints",
          "constraints": {
            "top": 12,
            "left": 16,
            "width": 120,
            "height": 40
          },
          "z": 1
        }
      },
      "bindings": {},
      "permissions": {},
      "events": {},
      "children": []
    },

    "node_temp_display": {
      "id": "node_temp_display",
      "type": "Text",
      "props": {
        "text": "--"
      },
      "style": {
        "fontSize": 48,
        "fontWeight": "bold",
        "color": "#fff"
      },
      "layoutItem": { "flex": { "grow": 0 } },
      "bindings": {
        "text": {
          "kind": "datapoint",
          "provider": "dc_main",
          "datapointId": "dp_temp_001",
          "path": "mqtt.EMQX.温度组.temperature",
          "transform": [
            { "op": "toFixed", "args": [1] },
            { "op": "suffix", "args": ["℃"] }
          ],
          "fallback": "--",
          "designMock": "25.5℃"
        }
      },
      "permissions": {
        "visible": { "allowRoles": ["admin", "operator", "viewer"] }
      },
      "events": {},
      "children": []
    }
  }
}
```

### 节点字段说明

| 字段        | 类型        | 说明                        |
| ----------- | ----------- | --------------------------- |
| id          | string      | 节点唯一 ID                 |
| type        | string      | 组件类型                    |
| props       | object      | 组件属性                    |
| style       | object      | 样式定义                    |
| layoutItem  | LayoutItem  | 布局配置（由父容器解释）    |
| bindings    | object      | 数据绑定                    |
| permissions | object      | 权限配置                    |
| events      | object      | 事件处理                    |
| animations  | Animation[] | 动画配置（见动画系统文档）  |
| conditions  | object      | 条件渲染（visible/enabled） |
| loop        | LoopConfig  | 循环渲染配置（可选）        |
| slots       | object      | 插槽内容（可选）            |
| refId       | string      | 自定义组件引用 ID（可选）   |
| overrides   | object      | 自定义组件属性覆盖（可选）  |
| children    | string[]    | 子节点 ID 数组              |

## 6. Canvas 图形节点

除了 DOM 组件节点，还支持 Canvas 图形节点，用于绘制工艺流程图：

### 6.1 图形类型

| type           | 说明      | 主要属性                            |
| -------------- | --------- | ----------------------------------- |
| Canvas.Line    | 直线/折线 | points, stroke, strokeWidth         |
| Canvas.Rect    | 矩形      | x, y, width, height, fill, stroke   |
| Canvas.Circle  | 圆形      | cx, cy, radius, fill, stroke        |
| Canvas.Ellipse | 椭圆      | cx, cy, rx, ry, fill, stroke        |
| Canvas.Polygon | 多边形    | points, fill, stroke                |
| Canvas.Path    | 路径      | d, fill, stroke                     |
| Canvas.Pipe    | 管道      | points, width, flowSpeed, flowColor |
| Canvas.Text    | 文字标注  | x, y, text, fontSize, fill          |
| Canvas.Symbol  | 符号引用  | symbolId, x, y, scale               |

### 6.2 图形节点示例

```json
{
  "graphicsById": {
    "gfx_pipe_main": {
      "id": "gfx_pipe_main",
      "type": "Canvas.Pipe",
      "props": {
        "points": [
          [100, 200],
          [300, 200],
          [300, 400],
          [500, 400]
        ],
        "width": 20,
        "strokeColor": "#3498db",
        "fillColor": "#2980b9",
        "flowDirection": "forward"
      },
      "bindings": {
        "flowSpeed": {
          "kind": "datapoint",
          "provider": "dc_main",
          "path": "mqtt.EMQX.流量组.flow_speed",
          "transform": [{ "op": "scale", "args": [0.1] }],
          "fallback": 0,
          "designMock": 5
        },
        "strokeColor": {
          "kind": "expr",
          "expr": "{{ $dp['mqtt.EMQX.状态组.alarm'] ? '#e74c3c' : '#3498db' }}"
        }
      },
      "events": {
        "click": [{ "type": "navigate", "config": { "path": "/pipe-detail" } }]
      },
      "animations": [],
      "z": 1
    },

    "gfx_rect_tank": {
      "id": "gfx_rect_tank",
      "type": "Canvas.Rect",
      "props": {
        "x": 200,
        "y": 100,
        "width": 100,
        "height": 150,
        "fill": "#34495e",
        "stroke": "#2c3e50",
        "strokeWidth": 2,
        "cornerRadius": 8
      },
      "bindings": {
        "fill": {
          "kind": "expr",
          "expr": "{{ $dp['mqtt.EMQX.液位组.level'] > 80 ? '#e74c3c' : '#34495e' }}"
        }
      },
      "events": {},
      "animations": [],
      "z": 2
    },

    "gfx_valve_1": {
      "id": "gfx_valve_1",
      "type": "Canvas.Symbol",
      "props": {
        "symbolId": "sym_valve_gate",
        "x": 250,
        "y": 350,
        "scale": 1.0,
        "rotation": 0
      },
      "bindings": {
        "props.rotation": {
          "kind": "expr",
          "expr": "{{ $dp['mqtt.EMQX.阀门组.valve_1_position'] * 0.9 }}"
        }
      },
      "events": {
        "click": [
          {
            "type": "writeTag",
            "config": {
              "path": "mqtt.EMQX.阀门组.valve_1_cmd",
              "value": "{{ $dp['mqtt.EMQX.阀门组.valve_1_position'] > 0 ? 0 : 100 }}"
            }
          }
        ]
      },
      "z": 3
    },

    "gfx_label_temp": {
      "id": "gfx_label_temp",
      "type": "Canvas.Text",
      "props": {
        "x": 300,
        "y": 50,
        "text": "温度: --",
        "fontSize": 14,
        "fill": "#ffffff",
        "fontWeight": "bold"
      },
      "bindings": {
        "text": {
          "kind": "expr",
          "expr": "温度: {{ $format($dp['mqtt.EMQX.温度组.temp'], '0.0') }}℃"
        }
      },
      "z": 10
    }
  }
}
```

### 6.3 图形字段说明

| 字段       | 类型        | 说明                         |
| ---------- | ----------- | ---------------------------- |
| id         | string      | 图形唯一 ID                  |
| type       | string      | 图形类型（Canvas.xxx）       |
| props      | object      | 图形属性（坐标、尺寸、颜色） |
| bindings   | object      | 数据绑定（同组件）           |
| events     | object      | 事件处理（click、hover 等）  |
| animations | Animation[] | 动画配置                     |
| z          | number      | 图层顺序                     |

### 6.4 管道（Pipe）详细属性

```json
{
  "props": {
    "points": [[x1, y1], [x2, y2], ...],  // 路径点
    "width": 20,                           // 管道宽度
    "strokeColor": "#3498db",              // 边框颜色
    "fillColor": "#2980b9",                // 填充颜色
    "flowDirection": "forward | backward | none",
    "flowSpeed": 5,                        // 流动速度 (px/s)
    "flowColor": "#ffffff",                // 流动指示颜色
    "flowDash": [10, 10],                  // 流动虚线样式
    "cornerRadius": 10,                    // 拐角圆角
    "startCap": "flat | round | arrow",    // 起点样式
    "endCap": "flat | round | arrow"       // 终点样式
  }
}
```

### 6.5 符号库定义

```json
{
  "symbolsById": {
    "sym_valve_gate": {
      "id": "sym_valve_gate",
      "name": "闸阀",
      "category": "valve",
      "graphics": [
        {
          "type": "rect",
          "x": -15,
          "y": -20,
          "width": 30,
          "height": 40,
          "fill": "#666"
        },
        {
          "type": "line",
          "points": [
            [0, -30],
            [0, 30]
          ],
          "stroke": "#333",
          "strokeWidth": 4
        },
        { "type": "circle", "cx": 0, "cy": 0, "radius": 8, "fill": "#f00" }
      ],
      "anchors": [
        { "name": "top", "x": 0, "y": -30 },
        { "name": "bottom", "x": 0, "y": 30 }
      ],
      "defaultSize": { "width": 40, "height": 60 }
    },
    "sym_pump": {
      "id": "sym_pump",
      "name": "泵",
      "category": "equipment",
      "graphics": [
        {
          "type": "circle",
          "cx": 0,
          "cy": 0,
          "radius": 25,
          "fill": "#3498db",
          "stroke": "#2980b9"
        },
        {
          "type": "polygon",
          "points": [
            [-10, -8],
            [10, 0],
            [-10, 8]
          ],
          "fill": "#fff"
        }
      ],
      "anchors": [
        { "name": "inlet", "x": -25, "y": 0 },
        { "name": "outlet", "x": 25, "y": 0 }
      ],
      "defaultSize": { "width": 50, "height": 50 }
    }
  }
}
```

### 6.6 页面结构（同时包含 DOM 和 Canvas）

```json
{
  "pagesById": {
    "page_flow": {
      "id": "page_flow",
      "path": "/flow",
      "title": "工艺流程图",
      "rootNodeId": "node_root_flow",
      "graphicsIds": [
        "gfx_pipe_main",
        "gfx_rect_tank",
        "gfx_valve_1",
        "gfx_label_temp"
      ],
      "config": {
        "canvasSize": { "width": 1920, "height": 1080 },
        "background": "#1a1a2e"
      }
    }
  }
}
```

## 7. 循环渲染（Loop）

用于根据数据源动态生成重复组件：

```json
{
  "node_device_list": {
    "id": "node_device_list",
    "type": "FlexContainer",
    "props": { "direction": "column", "gap": 8 },
    "loop": {
      "source": "{{ $dp['db.query.设备列表'] }}",
      "itemVar": "device",
      "indexVar": "index",
      "keyField": "id"
    },
    "children": ["node_device_card_template"]
  },
  "node_device_card_template": {
    "id": "node_device_card_template",
    "type": "DeviceCard",
    "bindings": {
      "name": { "kind": "expr", "expr": "{{ device.name }}" },
      "status": { "kind": "expr", "expr": "{{ device.status }}" },
      "value": { "kind": "expr", "expr": "{{ device.value }}" }
    },
    "events": {
      "click": [
        {
          "type": "setVar",
          "config": { "name": "selectedDeviceId", "value": "{{ device.id }}" }
        }
      ]
    }
  }
}
```

### Loop 配置说明

| 字段     | 类型   | 说明                         |
| -------- | ------ | ---------------------------- |
| source   | string | 数据源表达式（返回数组）     |
| itemVar  | string | 当前项变量名（默认 `$item`） |
| indexVar | string | 索引变量名（默认 `$index`）  |
| keyField | string | 唯一键字段（用于 diff 优化） |

### 循环上下文

在循环内可访问的变量：

```javascript
{
  {
    device;
  }
} // 当前项（由 itemVar 定义）
{
  {
    index;
  }
} // 当前索引（由 indexVar 定义）
{
  {
    $item;
  }
} // 默认当前项变量
{
  {
    $index;
  }
} // 默认索引变量
```

## 8. 插槽机制（Slots）

用于复合组件的内容分发：

```json
{
  "node_dialog": {
    "id": "node_dialog",
    "type": "Dialog",
    "props": {
      "title": "设备详情",
      "width": 600,
      "visible": "{{ $vars.page.dialogVisible }}"
    },
    "slots": {
      "default": ["node_dialog_content"],
      "header": ["node_dialog_header"],
      "footer": ["node_dialog_footer"]
    }
  },
  "node_dialog_content": {
    "id": "node_dialog_content",
    "type": "Text",
    "props": { "text": "对话框主内容" }
  },
  "node_dialog_footer": {
    "id": "node_dialog_footer",
    "type": "FlexContainer",
    "props": { "justify": "flex-end", "gap": 8 },
    "children": ["node_btn_cancel", "node_btn_confirm"]
  }
}
```

### Slots 结构

```typescript
interface SlotsConfig {
  default?: string[]; // 默认插槽
  [slotName: string]: string[]; // 具名插槽
}
```

## 9. 条件渲染（Conditions）

控制组件的可见性和可用性：

```json
{
  "node_admin_panel": {
    "id": "node_admin_panel",
    "type": "Panel",
    "conditions": {
      "visible": "{{ $vars.global.currentUser.role === 'admin' }}",
      "enabled": "{{ !$vars.page.isLoading }}"
    }
  }
}
```

### Conditions 字段

| 字段    | 类型           | 说明                       |
| ------- | -------------- | -------------------------- |
| visible | string/boolean | 是否渲染（false 时不渲染） |
| enabled | string/boolean | 是否可用（false 时禁用）   |

> **注意**：`visible: false` 不渲染组件（不占位），`enabled: false` 渲染但禁用交互。

### 与 Permissions 的区别

- **conditions**：基于数据/变量的动态条件
- **permissions**：基于用户角色的静态权限

## 10. 自定义组件引用

引用预定义的自定义组件，并支持属性覆盖：

```json
{
  "node_pump": {
    "id": "node_pump",
    "type": "CustomComponent",
    "refId": "cc_standard_pump_v1",
    "props": {
      "title": "循环水泵",
      "showLabel": true
    },
    "bindings": {
      "status": {
        "kind": "datapoint",
        "path": "mqtt.EMQX.泵组.pump_01_status"
      },
      "flow": { "kind": "datapoint", "path": "mqtt.EMQX.泵组.pump_01_flow" }
    },
    "overrides": {
      "node_inner_label": {
        "style": { "color": "#ff0000" }
      },
      "node_inner_icon": {
        "props": { "size": 32 }
      }
    }
  }
}
```

### 自定义组件字段

| 字段      | 类型   | 说明                              |
| --------- | ------ | --------------------------------- |
| refId     | string | 自定义组件定义 ID                 |
| overrides | object | 内部节点覆盖（key 为内部节点 ID） |

## 11. 资源引用协议

Schema 中引用资源的统一协议：

| 协议         | 说明        | 示例                              |
| ------------ | ----------- | --------------------------------- |
| `assets://`  | 项目资源库  | `assets://images/bg.png`          |
| `global://`  | 全局资源库  | `global://icons/motor.svg`        |
| `http(s)://` | 外部 URL    | `https://cdn.example.com/img.png` |
| `data:`      | Base64 内联 | `data:image/png;base64,...`       |

### 运行时解析

```javascript
// assets://images/bg.png
// Designer 预览: blob:...
// Runtime: /api/projects/{projectId}/assets/images/bg.png
//          或 https://cdn.example.com/projects/{projectId}/images/bg.png
```

## 12. Binding 结构

### 11.1 数据点绑定

```json
{
  "text": {
    "kind": "datapoint",
    "provider": "dc_main",
    "datapointId": "dp_xxx",
    "path": "mqtt.EMQX.温度组.temperature",
    "transform": [
      { "op": "toFixed", "args": [1] },
      { "op": "suffix", "args": ["℃"] }
    ],
    "fallback": "--",
    "designMock": "25.5℃"
  }
}
```

### 11.2 变量绑定

```json
{
  "disabled": {
    "kind": "var",
    "scope": "page",
    "name": "isLoading",
    "transform": [],
    "fallback": false
  }
}
```

### 11.3 表达式绑定

```json
{
  "visible": {
    "kind": "expr",
    "expr": "{{ $dp['mqtt.EMQX.温度组.temperature'] > 30 }}",
    "fallback": true
  }
}
```

## 13. LayoutItem 结构

### 12.1 Flex 布局项

```json
{
  "layoutItem": {
    "flex": {
      "grow": 1,
      "shrink": 0,
      "basis": "auto",
      "alignSelf": "stretch"
    }
  }
}
```

### 12.2 Free 布局项（绝对定位）

```json
{
  "layoutItem": {
    "free": {
      "mode": "abs",
      "abs": { "x": 10, "y": 10, "w": 120, "h": 40, "z": 1 }
    }
  }
}
```

### 12.3 Free 布局项（约束定位）

```json
{
  "layoutItem": {
    "free": {
      "mode": "constraints",
      "constraints": {
        "top": 12,
        "right": 12,
        "width": 160,
        "height": 48,
        "keepAspect": true
      },
      "z": 10
    }
  }
}
```

### 12.4 Grid 布局项

```json
{
  "layoutItem": {
    "grid": {
      "row": 1,
      "col": 2,
      "rowSpan": 1,
      "colSpan": 2
    }
  }
}
```

## 14. Permission 结构

### 13.1 组件权限

```json
{
  "permissions": {
    "visible": { "allowRoles": ["admin", "operator"] },
    "enable": { "allowRoles": ["admin"] },
    "readonly": { "denyRoles": ["viewer"] }
  }
}
```

### 13.2 动作权限

```json
{
  "events": {
    "click": [
      {
        "type": "writeTag",
        "config": { "path": "mqtt.EMQX.控制.start", "value": 1 },
        "permissions": { "allowRoles": ["admin"] }
      }
    ]
  }
}
```

## 15. Events 结构

```json
{
  "events": {
    "click": [
      {
        "type": "navigate",
        "config": {
          "pageId": "page_detail",
          "params": { "id": "{{ $props.deviceId }}" }
        }
      }
    ],
    "change": [
      {
        "type": "setVar",
        "config": {
          "scope": "page",
          "name": "selectedId",
          "value": "{{ $event.value }}"
        }
      }
    ]
  }
}
```

## 16. 内置动作类型

基础动作类型：

| 类型        | 说明       | 配置                    |
| ----------- | ---------- | ----------------------- |
| navigate    | 页面跳转   | pageId, params, replace |
| login       | 登录       | successRedirect         |
| logout      | 登出       | redirectTo              |
| setVar      | 设置变量   | scope, name, value      |
| callApi     | 调用接口   | endpoint, method, body  |
| writeTag    | 写入标签   | path, value, confirm    |
| notify      | 显示通知   | type, message, duration |
| openUrl     | 打开链接   | url, target             |
| openDialog  | 打开弹窗   | dialogId, props         |
| closeDialog | 关闭弹窗   | dialogId, result        |
| refresh     | 刷新数据源 | dataSourceId            |

控制流动作：

| 类型      | 说明     | 配置                    |
| --------- | -------- | ----------------------- |
| condition | 条件分支 | if, then, else          |
| loop      | 循环执行 | items, itemVar, actions |
| parallel  | 并行执行 | actions, onAllComplete  |
| delay     | 延迟执行 | duration, then          |

> 详细动作系统文档：[动作系统](./action-system.md)

## 17. 动画配置

节点支持动画配置，用于状态驱动动画和交互动画：

```json
{
  "animations": [
    {
      "id": "anim_rotate",
      "trigger": "dataChange",
      "condition": "{{ $dp['mqtt.EMQX.电机组.motor_running'] === true }}",
      "type": "rotate",
      "config": {
        "duration": 2000,
        "iterations": "infinite",
        "easing": "linear"
      }
    },
    {
      "id": "anim_alarm",
      "trigger": "dataChange",
      "condition": "{{ $dp['mqtt.EMQX.告警.level'] === 'critical' }}",
      "type": "flash",
      "config": {
        "duration": 500,
        "colors": ["#ff4d4f", "transparent"]
      }
    }
  ]
}
```

> 详细动画系统文档：[动画系统](./animation-system.md)

## 18. TypeScript 类型定义

```typescript
// schema/types.ts

export interface ProjectSchema {
  schemaVersion: number;
  project: ProjectMeta;
  securityDecl: SecurityDecl;
  entry: EntryConfig;
  dataProviders: Record<string, DataProvider>;
  vars: VarsConfig;
  assetsById: Record<string, AssetRef>;
  pagesById: Record<string, PageNode>;
  nodesById: Record<string, ComponentNode>;
}

export interface ProjectMeta {
  projectId: string;
  name: string;
  uiTargets: ("pc" | "bigscreen" | "mobile")[];
  createdAt: number;
  updatedAt: number;
}

export interface PageNode {
  id: string;
  name: string;
  path: string;
  target: "pc" | "bigscreen" | "mobile";
  logicalId: string;
  isDefaultTarget: boolean;
  rootNodeId: string;
  config: PageConfig;
}

export interface ComponentNode {
  id: string;
  type: string;
  props: Record<string, any>;
  style: StyleConfig;
  layoutItem: LayoutItem | null;
  bindings: Record<string, Binding>;
  permissions: PermissionConfig;
  events: Record<string, Action[]>;
  children: string[];
}

export type Binding = DatapointBinding | VarBinding | ExprBinding;

export interface DatapointBinding {
  kind: "datapoint";
  provider: string;
  datapointId: string;
  path: string;
  transform?: TransformOp[];
  fallback?: any;
  designMock?: any;
}

export interface VarBinding {
  kind: "var";
  scope: "page" | "global";
  name: string;
  transform?: TransformOp[];
  fallback?: any;
}

export interface ExprBinding {
  kind: "expr";
  expr: string;
  fallback?: any;
}

export type LayoutItem =
  | { flex: FlexLayoutItem }
  | { free: FreeLayoutItem }
  | { grid: GridLayoutItem };

export interface FreeLayoutItem {
  mode: "abs" | "constraints";
  abs?: { x: number; y: number; w: number; h: number; z: number };
  constraints?: {
    top?: number;
    right?: number;
    bottom?: number;
    left?: number;
    width?: number;
    height?: number;
    keepAspect?: boolean;
  };
  z?: number;
}
```

---

**相关文档**：

- [编辑器内核](./editor-core.md)
- [数据绑定 v2](./data-binding-v2.md)
- [布局系统](./layout-system.md)
- [动作系统](./action-system.md)
- [动画系统](./animation-system.md)
- [表达式引擎](./expression-engine.md)

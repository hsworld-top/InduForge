# 动画系统（Animation System）

本文档定义 Designer 的动画系统，支持组件动画、过渡效果和工业场景特有的状态动画。

## 1. 概述

动画系统提供：

- **状态驱动动画**：根据数据变化自动触发（如告警闪烁、电机旋转）
- **交互动画**：响应用户操作（如 hover、click）
- **生命周期动画**：组件挂载/卸载时的过渡效果
- **自定义动画**：基于 keyframes 的自定义动画

## 2. 动画结构

### 2.1 节点级动画定义

```json
{
  "nodesById": {
    "node_motor": {
      "id": "node_motor",
      "type": "IndustrialMotor",
      "props": { "name": "1号电机" },
      "animations": [
        {
          "id": "anim_rotate",
          "trigger": "dataChange",
          "condition": "{{ $dp['mqtt.EMQX.电机组.motor_status'] === 1 }}",
          "type": "rotate",
          "config": {
            "duration": 2000,
            "iterations": "infinite",
            "direction": "normal",
            "easing": "linear"
          }
        },
        {
          "id": "anim_alarm",
          "trigger": "dataChange",
          "condition": "{{ $dp['mqtt.EMQX.电机组.motor_alarm'] === true }}",
          "type": "flash",
          "config": {
            "duration": 500,
            "iterations": "infinite",
            "colors": ["#ff4d4f", "transparent"]
          }
        }
      ]
    }
  }
}
```

### 2.2 动画字段说明

| 字段      | 类型   | 说明                     |
| --------- | ------ | ------------------------ |
| id        | string | 动画唯一标识             |
| trigger   | string | 触发条件                 |
| condition | string | 触发表达式（可选）       |
| type      | string | 动画类型                 |
| config    | object | 动画配置                 |
| target    | string | 动画目标（默认为当前节点）|

## 3. 触发条件（Trigger）

| 触发器       | 说明                 | 使用场景             |
| ------------ | -------------------- | -------------------- |
| `mount`      | 组件挂载时           | 入场动画             |
| `unmount`    | 组件卸载时           | 离场动画             |
| `dataChange` | 数据点/变量变化时    | 状态动画（最常用）   |
| `hover`      | 鼠标悬停时           | 交互反馈             |
| `hoverEnd`   | 鼠标离开时           | 恢复状态             |
| `click`      | 点击时               | 点击反馈             |
| `focus`      | 获得焦点时           | 表单交互             |
| `blur`       | 失去焦点时           | 表单交互             |
| `visible`    | 进入可视区域时       | 懒加载动画           |
| `manual`     | 手动触发（通过动作） | 自定义控制           |

## 4. 动画类型

### 4.1 旋转（rotate）- 工业常用

```json
{
  "type": "rotate",
  "config": {
    "duration": 2000,
    "iterations": "infinite",
    "direction": "normal",
    "easing": "linear",
    "origin": "center center",
    "angle": 360
  }
}
```

| 配置项     | 类型           | 默认值          | 说明                                |
| ---------- | -------------- | --------------- | ----------------------------------- |
| duration   | number         | 1000            | 单次动画时长（ms）                  |
| iterations | number/string  | 1               | 迭代次数，`"infinite"` 为无限循环   |
| direction  | string         | "normal"        | 方向：normal/reverse/alternate      |
| easing     | string         | "linear"        | 缓动函数                            |
| origin     | string         | "center center" | 旋转中心                            |
| angle      | number         | 360             | 旋转角度                            |

### 4.2 闪烁（flash）- 告警常用

```json
{
  "type": "flash",
  "config": {
    "duration": 500,
    "iterations": "infinite",
    "colors": ["#ff4d4f", "transparent"],
    "property": "backgroundColor"
  }
}
```

| 配置项   | 类型     | 默认值                         | 说明                 |
| -------- | -------- | ------------------------------ | -------------------- |
| duration | number   | 500                            | 单次闪烁时长         |
| colors   | string[] | ["#ff4d4f", "transparent"]     | 闪烁颜色数组         |
| property | string   | "backgroundColor"              | 闪烁属性（背景/边框）|

### 4.3 缩放（scale）

```json
{
  "type": "scale",
  "config": {
    "from": 1,
    "to": 1.2,
    "duration": 300,
    "easing": "ease-out"
  }
}
```

### 4.4 淡入淡出（fadeIn / fadeOut）

```json
{
  "type": "fadeIn",
  "config": {
    "duration": 300,
    "delay": 0,
    "from": 0,
    "to": 1
  }
}
```

### 4.5 滑入滑出（slideIn / slideOut）

```json
{
  "type": "slideIn",
  "config": {
    "direction": "left",
    "distance": 100,
    "duration": 300,
    "easing": "ease-out"
  }
}
```

| direction | 说明     |
| --------- | -------- |
| left      | 从左滑入 |
| right     | 从右滑入 |
| top       | 从上滑入 |
| bottom    | 从下滑入 |

### 4.6 抖动（shake）

```json
{
  "type": "shake",
  "config": {
    "intensity": 10,
    "duration": 500,
    "direction": "horizontal"
  }
}
```

### 4.7 脉冲（pulse）

```json
{
  "type": "pulse",
  "config": {
    "scale": 1.05,
    "duration": 1000,
    "iterations": "infinite"
  }
}
```

### 4.8 跳动（bounce）

```json
{
  "type": "bounce",
  "config": {
    "height": 20,
    "duration": 500,
    "iterations": 3
  }
}
```

### 4.9 呼吸（breathe）- 工业状态指示

```json
{
  "type": "breathe",
  "config": {
    "minOpacity": 0.3,
    "maxOpacity": 1,
    "duration": 2000,
    "iterations": "infinite"
  }
}
```

### 4.10 自定义（custom）

```json
{
  "type": "custom",
  "config": {
    "keyframes": [
      { "offset": 0, "transform": "translateY(0)" },
      { "offset": 0.5, "transform": "translateY(-10px)" },
      { "offset": 1, "transform": "translateY(0)" }
    ],
    "duration": 1000,
    "iterations": "infinite",
    "easing": "ease-in-out"
  }
}
```

## 5. 工业场景示例

### 5.1 电机运转动画

```json
{
  "animations": [
    {
      "id": "anim_motor_rotate",
      "trigger": "dataChange",
      "condition": "{{ $dp['mqtt.EMQX.电机组.motor_running'] === true }}",
      "type": "rotate",
      "config": {
        "duration": "{{ 60000 / Math.max($dp['mqtt.EMQX.电机组.motor_rpm'], 1) }}",
        "iterations": "infinite",
        "direction": "normal",
        "easing": "linear"
      }
    }
  ]
}
```

> **说明**：根据实际转速动态计算动画周期，实现真实感。

### 5.2 阀门开关动画

```json
{
  "animations": [
    {
      "id": "anim_valve_open",
      "trigger": "dataChange",
      "condition": "{{ $dp['mqtt.EMQX.阀门组.valve_01_pos'] !== $prevValue }}",
      "type": "rotate",
      "config": {
        "from": "{{ $prevValue * 90 }}",
        "to": "{{ $dp['mqtt.EMQX.阀门组.valve_01_pos'] * 90 }}",
        "duration": 500,
        "easing": "ease-out"
      }
    }
  ]
}
```

### 5.3 告警闪烁（多级）

```json
{
  "animations": [
    {
      "id": "anim_alarm_critical",
      "trigger": "dataChange",
      "condition": "{{ $dp['mqtt.EMQX.告警.level'] === 'critical' }}",
      "type": "flash",
      "config": {
        "duration": 300,
        "iterations": "infinite",
        "colors": ["#ff4d4f", "#fff"]
      }
    },
    {
      "id": "anim_alarm_warning",
      "trigger": "dataChange",
      "condition": "{{ $dp['mqtt.EMQX.告警.level'] === 'warning' }}",
      "type": "flash",
      "config": {
        "duration": 800,
        "iterations": "infinite",
        "colors": ["#faad14", "#fff"]
      }
    },
    {
      "id": "anim_normal",
      "trigger": "dataChange",
      "condition": "{{ $dp['mqtt.EMQX.告警.level'] === 'normal' }}",
      "type": "none"
    }
  ]
}
```

### 5.4 液位动画（进度指示）

```json
{
  "animations": [
    {
      "id": "anim_level_wave",
      "trigger": "dataChange",
      "type": "custom",
      "config": {
        "keyframes": [
          { "offset": 0, "transform": "translateX(0)" },
          { "offset": 1, "transform": "translateX(-50%)" }
        ],
        "duration": 3000,
        "iterations": "infinite",
        "easing": "linear"
      },
      "target": ".wave-layer"
    }
  ]
}
```

### 5.5 传送带动画

```json
{
  "animations": [
    {
      "id": "anim_conveyor",
      "trigger": "dataChange",
      "condition": "{{ $dp['mqtt.EMQX.传送带.running'] === true }}",
      "type": "custom",
      "config": {
        "keyframes": [
          { "offset": 0, "backgroundPosition": "0 0" },
          { "offset": 1, "backgroundPosition": "100px 0" }
        ],
        "duration": "{{ 1000 / $dp['mqtt.EMQX.传送带.speed'] }}",
        "iterations": "infinite",
        "easing": "linear"
      }
    }
  ]
}
```

### 5.6 设备状态指示灯

```json
{
  "animations": [
    {
      "id": "anim_status_running",
      "trigger": "dataChange",
      "condition": "{{ $dp['device.status'] === 'running' }}",
      "type": "breathe",
      "config": {
        "minOpacity": 0.5,
        "maxOpacity": 1,
        "duration": 2000,
        "iterations": "infinite"
      }
    },
    {
      "id": "anim_status_fault",
      "trigger": "dataChange",
      "condition": "{{ $dp['device.status'] === 'fault' }}",
      "type": "flash",
      "config": {
        "duration": 400,
        "colors": ["#ff4d4f", "#330000"]
      }
    },
    {
      "id": "anim_status_stopped",
      "trigger": "dataChange",
      "condition": "{{ $dp['device.status'] === 'stopped' }}",
      "type": "none"
    }
  ]
}
```

## 6. 页面过渡动画

### 6.1 页面配置

```json
{
  "pagesById": {
    "page_home": {
      "config": {
        "transition": {
          "enter": {
            "type": "fadeIn",
            "duration": 300
          },
          "leave": {
            "type": "fadeOut",
            "duration": 200
          }
        }
      }
    }
  }
}
```

### 6.2 组件入场动画（交错）

```json
{
  "animations": [
    {
      "id": "anim_stagger_enter",
      "trigger": "mount",
      "type": "fadeIn",
      "config": {
        "duration": 300,
        "delay": "{{ $nodeIndex * 50 }}",
        "from": 0,
        "to": 1
      }
    }
  ]
}
```

## 7. 动画组合

### 7.1 同时播放多个动画

```json
{
  "animations": [
    {
      "id": "anim_highlight_scale",
      "trigger": "hover",
      "type": "scale",
      "config": { "to": 1.05, "duration": 200 }
    },
    {
      "id": "anim_highlight_shadow",
      "trigger": "hover",
      "type": "custom",
      "config": {
        "keyframes": [
          { "offset": 0, "boxShadow": "0 0 0 rgba(0,0,0,0)" },
          { "offset": 1, "boxShadow": "0 4px 12px rgba(0,0,0,0.15)" }
        ],
        "duration": 200
      }
    }
  ]
}
```

### 7.2 条件互斥动画

```json
{
  "animations": [
    {
      "id": "anim_status_1",
      "trigger": "dataChange",
      "condition": "{{ $dp['status'] === 1 }}",
      "type": "rotate",
      "config": { "iterations": "infinite" },
      "priority": 1
    },
    {
      "id": "anim_status_2",
      "trigger": "dataChange",
      "condition": "{{ $dp['status'] === 2 }}",
      "type": "flash",
      "config": { "iterations": "infinite" },
      "priority": 2
    }
  ]
}
```

> **priority**：数值越大优先级越高，同时满足条件时高优先级覆盖低优先级。

## 8. 动画控制（通过动作）

### 8.1 启动动画

```json
{
  "type": "startAnimation",
  "config": {
    "nodeId": "node_motor",
    "animationId": "anim_rotate"
  }
}
```

### 8.2 停止动画

```json
{
  "type": "stopAnimation",
  "config": {
    "nodeId": "node_motor",
    "animationId": "anim_rotate"
  }
}
```

### 8.3 播放一次性动画

```json
{
  "type": "playAnimation",
  "config": {
    "nodeId": "node_button",
    "animation": {
      "type": "shake",
      "config": { "intensity": 5, "duration": 300 }
    }
  }
}
```

## 9. 动画引擎实现

### 9.1 AnimationManager 接口

```typescript
interface AnimationManager {
  // 注册节点动画
  registerAnimations(nodeId: string, animations: Animation[]): void;

  // 启动动画
  start(nodeId: string, animationId: string): void;

  // 停止动画
  stop(nodeId: string, animationId?: string): void;

  // 播放一次性动画
  play(nodeId: string, animation: AnimationConfig): Promise<void>;

  // 更新条件（数据变化时调用）
  updateConditions(context: AnimationContext): void;
}

interface AnimationContext {
  $dp: Record<string, any>;
  $vars: { page: Record<string, any>; global: Record<string, any> };
  $prevValue?: any;
}
```

### 9.2 CSS 动画生成

```typescript
class CSSAnimationGenerator {
  generate(animation: Animation): string {
    switch (animation.type) {
      case "rotate":
        return this.generateRotate(animation.config);
      case "flash":
        return this.generateFlash(animation.config);
      case "custom":
        return this.generateCustom(animation.config);
      // ...
    }
  }

  private generateRotate(config: RotateConfig): string {
    const name = `rotate-${generateId()}`;
    const keyframes = `
      @keyframes ${name} {
        from { transform: rotate(0deg); }
        to { transform: rotate(${config.angle || 360}deg); }
      }
    `;
    return {
      keyframes,
      style: {
        animation: `${name} ${config.duration}ms ${config.easing} ${config.iterations}`,
        transformOrigin: config.origin,
      },
    };
  }

  private generateFlash(config: FlashConfig): string {
    const name = `flash-${generateId()}`;
    const colors = config.colors;
    const step = 100 / (colors.length - 1);
    const keyframes = `
      @keyframes ${name} {
        ${colors
          .map((color, i) => `${i * step}% { ${config.property}: ${color}; }`)
          .join("\n")}
      }
    `;
    return {
      keyframes,
      style: {
        animation: `${name} ${config.duration}ms linear ${config.iterations}`,
      },
    };
  }
}
```

### 9.3 GSAP 集成（可选）

对于复杂动画，可使用 GSAP：

```typescript
import gsap from "gsap";

class GSAPAnimationAdapter {
  private timelines: Map<string, gsap.core.Timeline> = new Map();

  play(element: HTMLElement, animation: Animation): gsap.core.Timeline {
    const tl = gsap.timeline({
      repeat: animation.config.iterations === "infinite" ? -1 : 0,
    });

    switch (animation.type) {
      case "rotate":
        tl.to(element, {
          rotation: animation.config.angle || 360,
          duration: animation.config.duration / 1000,
          ease: animation.config.easing,
        });
        break;
      // ...
    }

    return tl;
  }

  stop(nodeId: string): void {
    const tl = this.timelines.get(nodeId);
    tl?.kill();
    this.timelines.delete(nodeId);
  }
}
```

## 10. 性能优化

### 10.1 优化建议

1. **使用 CSS 动画优先**：比 JS 动画性能更好
2. **使用 transform 和 opacity**：触发 GPU 加速
3. **避免同时运行大量动画**：设置最大并发数
4. **离屏暂停**：不可见元素暂停动画
5. **合理的动画时长**：避免过长或过短

### 10.2 离屏暂停

```typescript
class AnimationManager {
  private observer: IntersectionObserver;

  constructor() {
    this.observer = new IntersectionObserver((entries) => {
      entries.forEach((entry) => {
        const nodeId = entry.target.dataset.nodeId;
        if (entry.isIntersecting) {
          this.resumeAnimations(nodeId);
        } else {
          this.pauseAnimations(nodeId);
        }
      });
    });
  }
}
```

### 10.3 动画池

```typescript
class AnimationPool {
  private maxConcurrent = 50;
  private running: Set<string> = new Set();
  private queue: Array<{ nodeId: string; animation: Animation }> = [];

  request(nodeId: string, animation: Animation): boolean {
    if (this.running.size < this.maxConcurrent) {
      this.running.add(`${nodeId}:${animation.id}`);
      return true;
    }
    this.queue.push({ nodeId, animation });
    return false;
  }

  release(nodeId: string, animationId: string): void {
    this.running.delete(`${nodeId}:${animationId}`);
    if (this.queue.length > 0) {
      const next = this.queue.shift();
      this.request(next.nodeId, next.animation);
    }
  }
}
```

## 11. 测试要点

- [ ] 各类型动画正确渲染
- [ ] 触发条件正确判断
- [ ] 条件表达式求值
- [ ] 动画启动/停止
- [ ] 优先级处理
- [ ] 离屏暂停/恢复
- [ ] 性能（大量动画场景）
- [ ] 内存泄漏检测

---

**相关文档**：

- [Schema 设计](./schema-design.md)
- [数据绑定 v2](./data-binding-v2.md)
- [表达式引擎](./expression-engine.md)


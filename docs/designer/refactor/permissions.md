# 权限系统（Permissions）

权限系统用于控制组件的可见性、可用性，以及动作的执行权限。

## 1. 概述

### 1.1 权限层次

```
┌─────────────────────────────────────────────────────────────────┐
│                        权限层次结构                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                   工程级权限声明                         │   │
│  │           (securityDecl.roles)                          │   │
│  └─────────────────────────────────────────────────────────┘   │
│                           │                                     │
│           ┌───────────────┼───────────────┐                    │
│           ▼               ▼               ▼                    │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐              │
│  │ 组件权限    │ │ 动作权限    │ │ 页面权限    │              │
│  │ (visible)   │ │ (actions)   │ │ (route)     │              │
│  │ (enable)    │ │             │ │             │              │
│  │ (readonly)  │ │             │ │             │              │
│  └─────────────┘ └─────────────┘ └─────────────┘              │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 权限类型

| 类型     | 控制范围             | 示例                   |
| -------- | -------------------- | ---------------------- |
| 组件权限 | 组件的显示/启用/只读 | 管理员才能看到删除按钮 |
| 动作权限 | 动作是否可执行       | 操作员才能执行启动操作 |
| 页面权限 | 页面是否可访问       | 管理员才能进入设置页面 |

## 2. 角色定义

### 2.1 工程级角色声明

```json
{
  "securityDecl": {
    "mode": "nodeLocalAuth",
    "roles": [
      {
        "name": "admin",
        "displayName": "管理员",
        "description": "系统管理员，拥有所有权限"
      },
      {
        "name": "operator",
        "displayName": "操作员",
        "description": "生产操作人员，可执行操作"
      },
      {
        "name": "viewer",
        "displayName": "观察员",
        "description": "只读用户，只能查看数据"
      }
    ],
    "defaultRole": "viewer"
  }
}
```

### 2.2 角色数据库表

`design_roles` 表：

| 字段        | 类型         | 说明             |
| ----------- | ------------ | ---------------- |
| id          | char(36)     | 主键             |
| projectId   | char(36)     | 所属工程         |
| name        | varchar(50)  | 角色标识（唯一） |
| displayName | varchar(100) | 显示名称         |
| description | varchar(255) | 描述             |
| permissions | json         | 预设权限配置     |
| isDefault   | tinyint(1)   | 是否默认角色     |

## 3. 组件权限

### 3.1 Schema 结构

```json
{
  "nodesById": {
    "node_delete_btn": {
      "id": "node_delete_btn",
      "type": "Button",
      "props": { "text": "删除" },
      "permissions": {
        "visible": {
          "mode": "roles",
          "roles": ["admin"]
        },
        "enable": {
          "mode": "roles",
          "roles": ["admin", "operator"]
        }
      }
    }
  }
}
```

### 3.2 权限配置类型

```typescript
interface PermissionConfig {
  // 可见性
  visible?: PermissionRule;
  // 可用性（禁用但可见）
  enable?: PermissionRule;
  // 只读（表单组件）
  readonly?: PermissionRule;
}

type PermissionRule =
  | { mode: "all" } // 所有人
  | { mode: "none" } // 无人
  | { mode: "roles"; roles: string[] } // 指定角色
  | { mode: "expr"; expr: string }; // 表达式
```

### 3.3 权限模式示例

**基于角色**：

```json
{
  "visible": {
    "mode": "roles",
    "roles": ["admin", "operator"]
  }
}
```

**基于表达式**：

```json
{
  "visible": {
    "mode": "expr",
    "expr": "{{ $user.department === 'production' }}"
  }
}
```

**组合条件**：

```json
{
  "enable": {
    "mode": "expr",
    "expr": "{{ $user.roles.includes('operator') && $vars.machineStatus === 'idle' }}"
  }
}
```

## 4. 动作权限

### 4.1 Schema 结构

```json
{
  "events": {
    "onClick": [
      {
        "type": "callApi",
        "api": "startMachine",
        "permission": {
          "mode": "roles",
          "roles": ["admin", "operator"],
          "denyMessage": "您没有启动设备的权限"
        }
      }
    ]
  }
}
```

### 4.2 权限校验流程

```typescript
async function executeAction(action: Action, context: ActionContext) {
  // 1. 前端权限校验
  if (action.permission) {
    const allowed = checkPermission(action.permission, context.user);
    if (!allowed) {
      if (action.permission.denyMessage) {
        showMessage(action.permission.denyMessage, "warning");
      }
      return;
    }
  }

  // 2. 执行动作（后端会再次校验）
  try {
    await actionRuntime.execute(action, context);
  } catch (error) {
    if (error.code === "PERMISSION_DENIED") {
      showMessage("权限不足", "error");
    }
  }
}
```

### 4.3 敏感动作

某些动作需要强制配置权限：

```typescript
const sensitiveActions = ["writeTag", "callApi", "navigate"];

function validateAction(action: Action) {
  if (sensitiveActions.includes(action.type) && !action.permission) {
    warnings.push(`动作 ${action.type} 未配置权限，建议添加权限控制`);
  }
}
```

## 5. 页面权限

### 5.1 Schema 结构

```json
{
  "pagesById": {
    "page_settings": {
      "path": "/settings",
      "permissions": {
        "access": {
          "mode": "roles",
          "roles": ["admin"]
        }
      }
    }
  }
}
```

### 5.2 路由守卫

```typescript
// RuntimeEngine 路由守卫
router.beforeEach((to, from, next) => {
  const page = getPageByPath(to.path);

  if (page?.permissions?.access) {
    const allowed = checkPermission(page.permissions.access, currentUser);
    if (!allowed) {
      next("/403");
      return;
    }
  }

  next();
});
```

## 6. 运行时实现

### 6.1 PermissionService

```typescript
class PermissionService {
  private currentUser: User | null = null;

  setUser(user: User): void {
    this.currentUser = user;
  }

  /**
   * 检查权限规则
   */
  check(rule: PermissionRule): boolean {
    if (!this.currentUser) {
      return rule.mode === "all";
    }

    switch (rule.mode) {
      case "all":
        return true;

      case "none":
        return false;

      case "roles":
        return rule.roles.some((role) =>
          this.currentUser!.roles.includes(role)
        );

      case "expr":
        return this.evaluateExpr(rule.expr);

      default:
        return false;
    }
  }

  /**
   * 获取组件的权限状态
   */
  getComponentPermissions(node: ComponentNode): ComponentPermissionState {
    return {
      visible: this.check(node.permissions?.visible ?? { mode: "all" }),
      enabled: this.check(node.permissions?.enable ?? { mode: "all" }),
      readonly: this.check(node.permissions?.readonly ?? { mode: "none" }),
    };
  }

  private evaluateExpr(expr: string): boolean {
    return expressionEngine.evaluate(expr, {
      $user: this.currentUser,
      $roles: this.currentUser?.roles ?? [],
    });
  }
}

interface ComponentPermissionState {
  visible: boolean;
  enabled: boolean;
  readonly: boolean;
}
```

### 6.2 与渲染集成

```typescript
// NodeRenderer.vue
<template>
  <component
    v-if="permissions.visible"
    :is="getComponent(node.type)"
    v-bind="resolvedProps"
    :disabled="!permissions.enabled"
    :readonly="permissions.readonly"
  />
</template>

<script setup>
const permissionService = inject('permissionService');

const permissions = computed(() => {
  return permissionService.getComponentPermissions(node);
});
</script>
```

## 7. Designer 支持

### 7.1 权限配置面板

```
┌─────────────────────────────────────────────────────────────────┐
│  权限配置                                                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  可见性:                                                        │
│  ○ 所有人可见                                                   │
│  ○ 指定角色可见                                                 │
│    ☑ admin (管理员)                                            │
│    ☑ operator (操作员)                                         │
│    ☐ viewer (观察员)                                           │
│  ○ 表达式控制                                                   │
│    [{{ $user.department === 'production' }}]                   │
│                                                                 │
│  ─────────────────────────────────────────────────────────────  │
│                                                                 │
│  可用性:                                                        │
│  ○ 所有人可用                                                   │
│  ● 指定角色可用                                                 │
│    ☑ admin                                                     │
│    ☑ operator                                                  │
│    ☐ viewer                                                    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 7.2 权限预览

设计态可以模拟不同角色预览：

```
┌─────────────────────────────────────────────────────────────────┐
│  预览角色: [操作员 ▼]                        [重置] [预览]      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  当前角色可见组件: 15/20                                        │
│  当前角色可用组件: 12/15                                        │
│                                                                 │
│  隐藏的组件:                                                    │
│  • 删除按钮 (仅 admin)                                         │
│  • 系统设置入口 (仅 admin)                                      │
│  • 用户管理按钮 (仅 admin)                                      │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## 8. 安全考虑

### 8.1 前后端双重校验

**前端**：

- 控制 UI 显示/启用状态
- 提升用户体验
- 不可信（可被绕过）

**后端**：

- API 层权限校验
- 数据库操作前校验
- 安全保障

```typescript
// 后端 API
app.post("/api/v1/machines/:id/start", authenticate, async (req, res) => {
  // 再次校验权限
  if (
    !req.user.roles.includes("operator") &&
    !req.user.roles.includes("admin")
  ) {
    return res.status(403).json({
      success: false,
      errorCode: "PERMISSION_DENIED",
      message: "无权执行此操作",
    });
  }

  // 执行操作
  await machineService.start(req.params.id);
  res.json({ success: true });
});
```

### 8.2 敏感数据脱敏

```json
{
  "bindings": {
    "text": {
      "kind": "datapoint",
      "path": "user.phone",
      "maskRule": {
        "roles": ["viewer"],
        "mask": "phone"
      }
    }
  }
}
```

### 8.3 审计日志

```typescript
// 记录敏感操作
async function executeAction(action: Action, context: ActionContext) {
  const result = await actionRuntime.execute(action, context);

  // 记录审计日志
  if (action.audit !== false) {
    await auditLog.record({
      action: action.type,
      userId: context.user.id,
      timestamp: Date.now(),
      params: action.params,
      result: result.success,
    });
  }

  return result;
}
```

## 9. 典型场景

### 9.1 操作员工作站

```json
{
  "permissions": {
    "visible": { "mode": "roles", "roles": ["operator", "admin"] },
    "enable": {
      "mode": "expr",
      "expr": "{{ $dp('machine.status') === 'idle' }}"
    }
  }
}
```

### 9.2 只读大屏

```json
{
  "pageConfig": {
    "permissions": {
      "defaultReadonly": true
    }
  }
}
```

### 9.3 管理后台

```json
{
  "pagesById": {
    "page_admin": {
      "permissions": {
        "access": { "mode": "roles", "roles": ["admin"] }
      }
    }
  }
}
```

---

**相关文档**：

- [Schema 设计](./schema-design.md)
- [动作系统](./action-system.md)
- [运行时引擎](./runtime-engine.md)

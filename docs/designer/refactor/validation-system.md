# 验证系统（Validation System）

本文档定义 Designer 的表单验证系统，用于用户输入校验和数据完整性保障。

## 1. 概述

验证系统提供：

- **声明式规则**：通过 Schema 配置验证规则
- **多种规则类型**：必填、类型、范围、格式、自定义等
- **灵活触发时机**：输入时、失焦时、提交时
- **异步验证**：支持远程校验（如唯一性检查）
- **国际化**：错误消息支持 i18n

## 2. 验证结构

### 2.1 节点级验证配置

```json
{
  "node_input_temp": {
    "id": "node_input_temp",
    "type": "InputNumber",
    "props": {
      "label": "目标温度",
      "placeholder": "请输入温度值"
    },
    "validation": {
      "rules": [
        { "required": true, "message": "温度不能为空" },
        { "type": "number", "message": "请输入有效数字" },
        { "min": 0, "max": 100, "message": "温度范围 0-100℃" },
        {
          "validator": "{{ (value) => value % 5 === 0 }}",
          "message": "温度必须是5的倍数"
        }
      ],
      "trigger": ["change", "blur"],
      "validateFirst": true
    }
  }
}
```

### 2.2 验证配置字段

| 字段          | 类型     | 说明                                |
| ------------- | -------- | ----------------------------------- |
| rules         | Rule[]   | 验证规则数组                        |
| trigger       | string[] | 触发时机：change/blur/submit        |
| validateFirst | boolean  | 是否遇到第一个错误即停止（默认 false）|

## 3. 规则类型

### 3.1 必填规则（required）

```json
{
  "required": true,
  "message": "此字段不能为空"
}
```

### 3.2 类型规则（type）

```json
{
  "type": "string",
  "message": "请输入字符串"
}
```

支持的类型：

| 类型     | 说明            |
| -------- | --------------- |
| string   | 字符串          |
| number   | 数字            |
| integer  | 整数            |
| boolean  | 布尔值          |
| array    | 数组            |
| object   | 对象            |
| email    | 邮箱格式        |
| url      | URL 格式        |
| date     | 日期格式        |
| phone    | 手机号格式      |
| idCard   | 身份证号格式    |

### 3.3 长度规则（len/minLength/maxLength）

```json
{
  "minLength": 6,
  "maxLength": 20,
  "message": "长度需在 6-20 个字符之间"
}
```

```json
{
  "len": 11,
  "message": "手机号必须是11位"
}
```

### 3.4 范围规则（min/max）

```json
{
  "min": 0,
  "max": 100,
  "message": "数值范围 0-100"
}
```

### 3.5 正则规则（pattern）

```json
{
  "pattern": "^[A-Za-z0-9]+$",
  "message": "只能包含字母和数字"
}
```

### 3.6 枚举规则（enum）

```json
{
  "enum": ["running", "stopped", "fault"],
  "message": "状态必须是 running、stopped 或 fault"
}
```

### 3.7 自定义验证器（validator）

```json
{
  "validator": "{{ (value, rule, callback) => { if (value % 5 !== 0) { callback('必须是5的倍数'); } else { callback(); } } }}",
  "message": "验证失败"
}
```

简化语法（返回布尔值）：

```json
{
  "validator": "{{ (value) => value % 5 === 0 }}",
  "message": "必须是5的倍数"
}
```

### 3.8 异步验证器（asyncValidator）

```json
{
  "asyncValidator": "checkDeviceNameUnique",
  "message": "设备名称已存在"
}
```

异步验证器定义：

```typescript
// 注册异步验证器
validationEngine.registerAsyncValidator(
  "checkDeviceNameUnique",
  async (value, rule, context) => {
    const response = await fetch(`/api/devices/check-name?name=${value}`);
    const result = await response.json();
    if (!result.available) {
      throw new Error("设备名称已存在");
    }
  }
);
```

### 3.9 条件验证（when）

```json
{
  "required": true,
  "message": "开启高级模式时此字段必填",
  "when": "{{ $vars.page.advancedMode === true }}"
}
```

### 3.10 依赖验证（dependencies）

```json
{
  "validator": "{{ (value) => value > $vars.page.minTemp }}",
  "message": "最高温度必须大于最低温度",
  "dependencies": ["minTemp"]
}
```

## 4. 触发时机

### 4.1 change（输入时）

每次值变化时触发验证：

```json
{
  "trigger": ["change"]
}
```

### 4.2 blur（失焦时）

输入框失去焦点时触发：

```json
{
  "trigger": ["blur"]
}
```

### 4.3 submit（提交时）

表单提交时触发：

```json
{
  "trigger": ["submit"]
}
```

### 4.4 组合触发

```json
{
  "trigger": ["change", "blur"]
}
```

## 5. 表单级验证

### 5.1 表单容器配置

```json
{
  "node_form": {
    "id": "node_form",
    "type": "Form",
    "props": {
      "model": "{{ $vars.page.formData }}",
      "labelWidth": 100
    },
    "formValidation": {
      "validateOnChange": true,
      "validateOnBlur": true,
      "scrollToError": true,
      "errorDisplayMode": "first"
    },
    "children": ["node_input_name", "node_input_temp", "node_btn_submit"]
  }
}
```

### 5.2 表单验证动作

```json
{
  "events": {
    "click": [
      {
        "type": "validateForm",
        "config": {
          "formId": "node_form",
          "onValid": [
            { "type": "callApi", "config": { "endpoint": "/api/save" } }
          ],
          "onInvalid": [
            {
              "type": "notify",
              "config": { "type": "error", "message": "请检查表单填写" }
            }
          ]
        }
      }
    ]
  }
}
```

### 5.3 重置验证状态

```json
{
  "type": "resetValidation",
  "config": {
    "formId": "node_form",
    "fields": ["fieldName"]
  }
}
```

## 6. 工业场景示例

### 6.1 设备参数设定

```json
{
  "node_input_setpoint": {
    "type": "InputNumber",
    "props": {
      "label": "目标设定值",
      "precision": 2
    },
    "validation": {
      "rules": [
        { "required": true, "message": "设定值不能为空" },
        { "type": "number", "message": "请输入有效数字" },
        {
          "min": "{{ $dp['device.paramConfig.minValue'] }}",
          "max": "{{ $dp['device.paramConfig.maxValue'] }}",
          "message": "{{ '设定值范围: ' + $dp['device.paramConfig.minValue'] + ' - ' + $dp['device.paramConfig.maxValue'] }}"
        },
        {
          "validator": "{{ (value) => { const step = $dp['device.paramConfig.step']; return value % step === 0; } }}",
          "message": "{{ '设定值必须是 ' + $dp['device.paramConfig.step'] + ' 的倍数' }}"
        }
      ],
      "trigger": ["change", "blur"]
    }
  }
}
```

### 6.2 用户登录表单

```json
{
  "validation": {
    "rules": [
      { "required": true, "message": "请输入用户名" },
      { "minLength": 4, "maxLength": 20, "message": "用户名长度 4-20 位" },
      { "pattern": "^[A-Za-z][A-Za-z0-9_]*$", "message": "用户名只能包含字母、数字和下划线，且以字母开头" }
    ]
  }
}
```

### 6.3 密码确认

```json
{
  "node_input_confirm_password": {
    "type": "Input",
    "props": { "label": "确认密码", "type": "password" },
    "validation": {
      "rules": [
        { "required": true, "message": "请确认密码" },
        {
          "validator": "{{ (value) => value === $vars.page.formData.password }}",
          "message": "两次输入的密码不一致",
          "dependencies": ["password"]
        }
      ],
      "trigger": ["blur"]
    }
  }
}
```

### 6.4 IP 地址验证

```json
{
  "validation": {
    "rules": [
      { "required": true, "message": "请输入 IP 地址" },
      {
        "pattern": "^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$",
        "message": "请输入有效的 IP 地址"
      }
    ]
  }
}
```

### 6.5 端口范围验证

```json
{
  "validation": {
    "rules": [
      { "required": true, "message": "请输入端口号" },
      { "type": "integer", "message": "端口号必须是整数" },
      { "min": 1, "max": 65535, "message": "端口号范围 1-65535" },
      {
        "validator": "{{ (value) => ![80, 443, 3306, 6379].includes(value) }}",
        "message": "该端口已被系统保留"
      }
    ]
  }
}
```

### 6.6 设备名称唯一性

```json
{
  "validation": {
    "rules": [
      { "required": true, "message": "请输入设备名称" },
      { "minLength": 2, "maxLength": 50, "message": "设备名称长度 2-50 个字符" },
      {
        "asyncValidator": "checkDeviceNameUnique",
        "message": "设备名称已存在"
      }
    ],
    "trigger": ["blur"]
  }
}
```

## 7. 验证状态与反馈

### 7.1 验证状态

```typescript
interface FieldValidationState {
  status: "pending" | "validating" | "success" | "error" | "warning";
  errors: string[];
  warnings: string[];
}
```

### 7.2 UI 反馈

**输入框状态样式**：

```json
{
  "style": {
    "borderColor": "{{ $validation.status === 'error' ? '#ff4d4f' : $validation.status === 'success' ? '#52c41a' : '#d9d9d9' }}"
  }
}
```

**错误消息显示**：

```json
{
  "node_error_message": {
    "type": "Text",
    "props": {
      "text": "{{ $validation.errors[0] }}"
    },
    "style": {
      "color": "#ff4d4f",
      "fontSize": 12
    },
    "conditions": {
      "visible": "{{ $validation.status === 'error' }}"
    }
  }
}
```

## 8. 验证引擎实现

### 8.1 核心接口

```typescript
interface ValidationEngine {
  // 验证单个字段
  validateField(
    value: any,
    rules: ValidationRule[],
    context: ValidationContext
  ): Promise<ValidationResult>;

  // 验证表单
  validateForm(
    formData: Record<string, any>,
    schema: FormSchema
  ): Promise<FormValidationResult>;

  // 注册异步验证器
  registerAsyncValidator(
    name: string,
    validator: AsyncValidator
  ): void;

  // 注册自定义规则类型
  registerRuleType(
    type: string,
    handler: RuleHandler
  ): void;
}

interface ValidationResult {
  valid: boolean;
  errors: string[];
  warnings: string[];
}

interface FormValidationResult {
  valid: boolean;
  fields: Record<string, ValidationResult>;
  firstError?: { field: string; message: string };
}
```

### 8.2 规则执行器

```typescript
class RuleExecutor {
  async execute(
    value: any,
    rule: ValidationRule,
    context: ValidationContext
  ): Promise<string | null> {
    // 条件验证
    if (rule.when) {
      const shouldValidate = evaluateExpression(rule.when, context);
      if (!shouldValidate) return null;
    }

    // 必填验证
    if (rule.required) {
      if (value === undefined || value === null || value === "") {
        return rule.message || "此字段必填";
      }
    }

    // 空值跳过后续验证
    if (value === undefined || value === null || value === "") {
      return null;
    }

    // 类型验证
    if (rule.type) {
      if (!this.checkType(value, rule.type)) {
        return rule.message || `类型错误，期望 ${rule.type}`;
      }
    }

    // 范围验证
    if (rule.min !== undefined || rule.max !== undefined) {
      const numValue = typeof value === "number" ? value : parseFloat(value);
      if (rule.min !== undefined && numValue < rule.min) {
        return rule.message || `不能小于 ${rule.min}`;
      }
      if (rule.max !== undefined && numValue > rule.max) {
        return rule.message || `不能大于 ${rule.max}`;
      }
    }

    // 长度验证
    if (rule.minLength !== undefined || rule.maxLength !== undefined) {
      const len = String(value).length;
      if (rule.minLength !== undefined && len < rule.minLength) {
        return rule.message || `长度不能少于 ${rule.minLength}`;
      }
      if (rule.maxLength !== undefined && len > rule.maxLength) {
        return rule.message || `长度不能超过 ${rule.maxLength}`;
      }
    }

    // 正则验证
    if (rule.pattern) {
      const regex = new RegExp(rule.pattern);
      if (!regex.test(String(value))) {
        return rule.message || "格式不正确";
      }
    }

    // 枚举验证
    if (rule.enum) {
      if (!rule.enum.includes(value)) {
        return rule.message || `必须是以下值之一: ${rule.enum.join(", ")}`;
      }
    }

    // 自定义验证器
    if (rule.validator) {
      const validatorFn = evaluateExpression(rule.validator, context);
      const result = await validatorFn(value, rule, context);
      if (result === false || typeof result === "string") {
        return typeof result === "string" ? result : rule.message;
      }
    }

    // 异步验证器
    if (rule.asyncValidator) {
      const validator = this.asyncValidators.get(rule.asyncValidator);
      if (validator) {
        try {
          await validator(value, rule, context);
        } catch (e) {
          return e.message || rule.message;
        }
      }
    }

    return null;
  }
}
```

## 9. 测试要点

- [ ] 各规则类型正确性
- [ ] 触发时机正确
- [ ] 异步验证执行
- [ ] 条件验证逻辑
- [ ] 依赖字段触发
- [ ] 表单整体验证
- [ ] 验证状态管理
- [ ] 错误消息国际化
- [ ] 性能（大量字段场景）

---

**相关文档**：

- [Schema 设计](./schema-design.md)
- [表达式引擎](./expression-engine.md)
- [动作系统](./action-system.md)


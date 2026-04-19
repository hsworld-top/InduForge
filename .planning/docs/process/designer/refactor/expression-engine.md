# 表达式引擎（Expression Engine）

本文档定义 Designer 的表达式系统，包括语法、上下文变量和内置函数。

## 1. 概述

表达式引擎用于：

- 数据绑定中的动态值计算
- 条件判断（visible/enabled/动画触发）
- 动作配置中的参数引用
- 样式动态计算

## 2. 语法

### 2.1 基本语法

使用 `{{ expression }}` 包裹 JavaScript 表达式：

```javascript
// 简单引用
{{ $dp['mqtt.EMQX.温度组.temperature'] }}

// 计算
{{ $dp['device.current'] * $dp['device.voltage'] }}

// 条件
{{ $dp['device.status'] === 1 ? '运行' : '停止' }}

// 字符串拼接
{{ '温度: ' + $dp['device.temp'] + '℃' }}
```

### 2.2 表达式限制

为安全考虑，表达式引擎有以下限制：

- ❌ 不支持函数声明 (`function`, 箭头函数)
- ❌ 不支持 `new` 操作符
- ❌ 不支持 `eval`, `Function`
- ❌ 不支持访问 `window`, `document`, `globalThis`
- ❌ 不支持 `import`, `require`
- ✅ 支持基本运算符
- ✅ 支持三元表达式
- ✅ 支持数组/对象字面量
- ✅ 支持内置函数调用

## 3. 上下文变量

### 3.1 数据点（$dp）

访问数据中心的数据点值：

```javascript
{{ $dp['mqtt.EMQX.温度组.temperature'] }}
{{ $dp['db.query.设备列表'][0].name }}
{{ $dp['calc.产量统计.日产量'] }}
```

### 3.2 变量（$vars）

访问页面变量和全局变量：

```javascript
// 页面变量
{{ $vars.page.selectedId }}
{{ $vars.page.isLoading }}

// 全局变量
{{ $vars.global.currentUser }}
{{ $vars.global.theme }}
```

### 3.3 组件属性（$props）

访问当前组件的属性：

```javascript
{{ $props.deviceId }}
{{ $props.title }}
{{ $props.config.maxValue }}
```

### 3.4 事件对象（$event）

在事件处理中访问事件信息：

```javascript
// 点击事件
{{ $event.target }}

// 输入事件
{{ $event.value }}

// 表格行点击
{{ $event.row }}
{{ $event.rowIndex }}

// 自定义事件数据
{{ $event.data }}
```

### 3.5 循环上下文（$item, $index）

在循环渲染中访问当前项：

```javascript
// 默认变量名
{{ $item.name }}
{{ $index }}

// 自定义变量名（由 loop.itemVar/indexVar 定义）
{{ device.name }}
{{ index }}
```

### 3.6 动作结果（$prevResult, $result）

访问动作执行结果：

```javascript
// 上一个动作的返回结果
{{ $prevResult.data }}

// API 调用成功结果
{{ $result.data.list }}
```

### 3.7 错误信息（$error）

在错误处理中访问错误信息：

```javascript
{{ $error.message }}
{{ $error.code }}
{{ $error.details }}
```

### 3.8 用户信息（$user）

访问当前登录用户：

```javascript
{{ $user.id }}
{{ $user.name }}
{{ $user.role }}
{{ $user.permissions }}
{{ $user.department }}
```

### 3.9 路由信息（$route）

访问当前路由：

```javascript
{{ $route.path }}
{{ $route.params.id }}
{{ $route.query.tab }}
```

### 3.10 环境变量（$env）

访问运行时环境变量：

```javascript
{{ $env.API_BASE }}
{{ $env.MODE }}
{{ $env.VERSION }}
```

### 3.11 节点信息（$node）

访问当前节点信息：

```javascript
{{ $node.id }}
{{ $node.type }}
```

## 4. 内置函数

### 4.1 格式化函数（$format）

```javascript
// 数字格式化
{{ $format.number(value, 2) }}                    // 保留2位小数
{{ $format.number(12345.6, 0) }}                  // → "12346"

// 日期格式化
{{ $format.date(timestamp, 'YYYY-MM-DD') }}       // → "2025-01-06"
{{ $format.date(timestamp, 'YYYY-MM-DD HH:mm:ss') }}

// 货币格式化
{{ $format.currency(value, 'CNY') }}              // → "¥1,234.56"
{{ $format.currency(value, 'USD') }}              // → "$1,234.56"

// 百分比格式化
{{ $format.percent(0.1234) }}                     // → "12.34%"
{{ $format.percent(0.1234, 1) }}                  // → "12.3%"

// 文件大小格式化
{{ $format.fileSize(1024) }}                      // → "1 KB"
{{ $format.fileSize(1048576) }}                   // → "1 MB"

// 时长格式化
{{ $format.duration(3661) }}                      // → "1:01:01"
{{ $format.duration(3661, 'verbose') }}           // → "1小时1分1秒"
```

### 4.2 数组函数（$array）

```javascript
// 求和
{{ $array.sum(list, 'value') }}
{{ $array.sum([1, 2, 3]) }}                       // → 6

// 平均值
{{ $array.avg(list, 'score') }}

// 最值
{{ $array.max(list, 'price') }}
{{ $array.min(list, 'price') }}

// 分组
{{ $array.groupBy(list, 'category') }}
// → { "A": [...], "B": [...] }

// 排序
{{ $array.sortBy(list, 'createdAt', 'desc') }}

// 去重
{{ $array.unique(list, 'id') }}

// 过滤
{{ $array.filter(list, item => item.status === 'active') }}

// 查找
{{ $array.find(list, item => item.id === selectedId) }}

// 计数
{{ $array.count(list) }}
{{ $array.count(list, item => item.status === 'active') }}

// 取前N个
{{ $array.take(list, 10) }}

// 扁平化
{{ $array.flatten(nestedList) }}
{{ $array.flatten(nestedList, 2) }}               // 指定深度
```

### 4.3 字符串函数（$string）

```javascript
// 截断
{{ $string.truncate(text, 20) }}                  // → "这是一段很长的文..."
{{ $string.truncate(text, 20, '...更多') }}

// 模板替换
{{ $string.template('Hello, {name}!', { name: 'World' }) }}

// 大小写转换
{{ $string.upper(text) }}
{{ $string.lower(text) }}
{{ $string.capitalize(text) }}

// 填充
{{ $string.padStart(num, 4, '0') }}               // → "0042"
{{ $string.padEnd(text, 10, '-') }}

// 包含判断
{{ $string.includes(text, 'keyword') }}
{{ $string.startsWith(text, 'prefix') }}
{{ $string.endsWith(text, 'suffix') }}
```

### 4.4 条件函数

```javascript
// 三元表达式简化
{{ $if(condition, trueValue, falseValue) }}

// 空值处理
{{ $ifNull(value, defaultValue) }}
{{ $ifEmpty(value, defaultValue) }}

// Switch 表达式
{{ $switch(status, {
     1: '运行',
     2: '停止',
     3: '故障'
   }, '未知') }}

// 范围映射
{{ $map(value, [
     [0, 30, '低'],
     [30, 70, '中'],
     [70, 100, '高']
   ], '超限') }}
```

### 4.5 数学函数

```javascript
// 基础数学
{{ Math.abs(value) }}
{{ Math.round(value) }}
{{ Math.floor(value) }}
{{ Math.ceil(value) }}
{{ Math.sqrt(value) }}
{{ Math.pow(base, exp) }}

// 范围限制
{{ $clamp(value, min, max) }}

// 四舍五入到指定小数位
{{ $round(value, 2) }}

// 随机数
{{ $random(1, 100) }}                             // 1-100 随机整数
{{ $randomFloat(0, 1) }}                          // 0-1 随机小数
```

### 4.6 工业计算函数

```javascript
// 线性变换（原始值到工程值）
{{ $scale(rawValue, rawMin, rawMax, euMin, euMax) }}
// 例: $scale(2048, 0, 4095, 0, 100) → 50

// 死区判断（避免数据频繁波动）
{{ $deadband(newValue, lastValue, threshold) }}
// 返回是否超出死区，用于判断是否需要更新

// 累计值（运行时间、产量等）
{{ $accumulate(value, 'key') }}

// 滑动平均
{{ $movingAvg(value, 'key', windowSize) }}

// 状态持续时间
{{ $stateDuration(status, 'key') }}               // 返回秒数

// 报警延迟（避免瞬时告警）
{{ $alarmDelay(condition, 'key', delaySeconds) }}

// 脉冲检测（边沿触发）
{{ $risingEdge(currentValue, 'key') }}
{{ $fallingEdge(currentValue, 'key') }}
```

### 4.7 日期时间函数

```javascript
// 当前时间
{{ $now() }}                                      // 时间戳
{{ $today() }}                                    // 今天 00:00:00

// 时间计算
{{ $dateAdd(date, 1, 'day') }}
{{ $dateAdd(date, -1, 'hour') }}
{{ $dateDiff(date1, date2, 'day') }}

// 时间判断
{{ $isToday(date) }}
{{ $isThisWeek(date) }}
{{ $isThisMonth(date) }}

// 时间范围
{{ $startOfDay(date) }}
{{ $endOfDay(date) }}
{{ $startOfMonth(date) }}
```

### 4.8 颜色函数

```javascript
// 颜色插值（用于渐变显示）
{{ $colorLerp('#00ff00', '#ff0000', ratio) }}

// 根据值映射颜色
{{ $colorScale(value, [
     [0, '#52c41a'],      // 绿色
     [50, '#faad14'],     // 黄色
     [100, '#ff4d4f']     // 红色
   ]) }}

// 透明度调整
{{ $alpha('#ff0000', 0.5) }}                      // → "rgba(255,0,0,0.5)"
```

## 5. 工业场景示例

### 5.1 温度显示与报警

```json
{
  "bindings": {
    "text": {
      "kind": "expr",
      "expr": "{{ $format.number($dp['mqtt.EMQX.温度组.temperature'], 1) + '℃' }}"
    },
    "style.color": {
      "kind": "expr",
      "expr": "{{ $colorScale($dp['mqtt.EMQX.温度组.temperature'], [[0, '#52c41a'], [60, '#faad14'], [80, '#ff4d4f']]) }}"
    }
  },
  "conditions": {
    "visible": "{{ $dp['mqtt.EMQX.温度组.temperature'] !== null }}"
  }
}
```

### 5.2 设备状态显示

```json
{
  "bindings": {
    "text": {
      "kind": "expr",
      "expr": "{{ $switch($dp['device.status'], { 0: '停止', 1: '运行', 2: '故障', 3: '维护' }, '未知') }}"
    },
    "type": {
      "kind": "expr",
      "expr": "{{ $switch($dp['device.status'], { 0: 'info', 1: 'success', 2: 'danger', 3: 'warning' }, 'default') }}"
    }
  }
}
```

### 5.3 生产效率计算

```json
{
  "bindings": {
    "value": {
      "kind": "expr",
      "expr": "{{ $round($dp['production.actual'] / $dp['production.target'] * 100, 1) }}"
    },
    "status": {
      "kind": "expr",
      "expr": "{{ $dp['production.actual'] / $dp['production.target'] >= 0.9 ? 'success' : $dp['production.actual'] / $dp['production.target'] >= 0.7 ? 'warning' : 'danger' }}"
    }
  }
}
```

### 5.4 运行时长格式化

```json
{
  "bindings": {
    "text": {
      "kind": "expr",
      "expr": "{{ $format.duration($dp['device.runningSeconds'], 'verbose') }}"
    }
  }
}
```

### 5.5 数据表格过滤

```json
{
  "bindings": {
    "data": {
      "kind": "expr",
      "expr": "{{ $array.filter($dp['db.query.设备列表'], d => $string.includes(d.name, $vars.page.searchKeyword) && ($vars.page.statusFilter === 'all' || d.status === $vars.page.statusFilter)) }}"
    }
  }
}
```

## 6. 表达式引擎实现

### 6.1 核心接口

```typescript
interface ExpressionEngine {
  // 求值
  evaluate(expr: string, context: ExpressionContext): any;

  // 批量求值
  evaluateBindings(
    bindings: Record<string, Binding>,
    context: ExpressionContext
  ): Record<string, any>;

  // 依赖分析
  analyzeDependencies(expr: string): ExpressionDependencies;

  // 注册自定义函数
  registerFunction(name: string, fn: Function): void;
}

interface ExpressionContext {
  $dp: Record<string, any>;
  $vars: {
    page: Record<string, any>;
    global: Record<string, any>;
  };
  $props: Record<string, any>;
  $event?: any;
  $item?: any;
  $index?: number;
  $prevResult?: any;
  $error?: any;
  $user?: UserInfo;
  $route?: RouteInfo;
  $env?: Record<string, any>;
  $node?: NodeInfo;
}

interface ExpressionDependencies {
  datapoints: string[];
  pageVars: string[];
  globalVars: string[];
  props: string[];
}
```

### 6.2 安全求值器

```typescript
class SafeExpressionEvaluator {
  private sandbox: Record<string, any>;

  constructor() {
    // 构建安全沙箱
    this.sandbox = {
      // 白名单：内置函数
      $format: formatFunctions,
      $array: arrayFunctions,
      $string: stringFunctions,
      $if: ifFn,
      $ifNull: ifNullFn,
      $switch: switchFn,
      $clamp: clampFn,
      $round: roundFn,
      $scale: scaleFn,
      // ... 其他内置函数

      // 白名单：安全的全局对象
      Math,
      JSON: { parse: JSON.parse, stringify: JSON.stringify },
      Date,
      Number,
      String,
      Boolean,
      Array,
      Object,
    };
  }

  evaluate(expr: string, context: ExpressionContext): any {
    // 1. 提取 {{ }} 内的表达式
    const code = expr.replace(/^\{\{|\}\}$/g, "").trim();

    // 2. 安全检查
    this.validateExpression(code);

    // 3. 构建执行上下文
    const evalContext = { ...this.sandbox, ...context };

    // 4. 安全执行
    return this.safeEval(code, evalContext);
  }

  private validateExpression(code: string): void {
    // 禁止危险关键字
    const forbidden = [
      "eval",
      "Function",
      "constructor",
      "prototype",
      "__proto__",
      "window",
      "document",
      "globalThis",
      "import",
      "require",
      "process",
    ];

    for (const keyword of forbidden) {
      if (code.includes(keyword)) {
        throw new Error(`Expression contains forbidden keyword: ${keyword}`);
      }
    }
  }
}
```

## 7. 测试要点

- [ ] 基本表达式求值
- [ ] 上下文变量访问
- [ ] 内置函数正确性
- [ ] 安全检查（禁止危险代码）
- [ ] 依赖分析准确性
- [ ] 错误处理与降级
- [ ] 性能（缓存编译结果）
- [ ] 循环上下文传递

---

**相关文档**：

- [Schema 设计](./schema-design.md)
- [数据绑定 v2](./data-binding-v2.md)
- [动作系统](./action-system.md)


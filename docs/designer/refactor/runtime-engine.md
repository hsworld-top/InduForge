# 运行时引擎（RuntimeEngine）

本文档描述工程发布后在节点侧运行的 RuntimeEngine，包括数据服务、资源生命周期管理、Watchdog 等。

## 0. 部署与启动

### 0.1 部署来源

RuntimeEngine 由 **dev_ide 运维管理界面** 部署到节点：

- 工程发布后产生 IFP 包（含 datacenter.json 数据连接配置）
- 运维人员在 dev_ide 选择节点并部署
- 部署时配置运行端口、默认语言、默认主题等
- **同一工程可部署到多个节点，同一节点只能运行一个工程**
- **数据连接配置在工程设置中预先配置，随 IFP 发布**

### 0.2 IFP 包内容

```
project-v1.0.0.ifp
├── manifest.json           # 清单文件（dataRequirements）
├── project.json            # 设计器 Schema
├── datacenter.json         # 数据中心配置（连接/查询/数据点定义）
├── assets/                 # 资源文件
└── assets-mapping.json     # 资源映射
```

### 0.3 datacenter.json（数据连接配置）

数据连接配置在**工程设置**中预先配置，随 IFP 发布：

```json
// datacenter.json（IFP 包内）
{
  "connections": [
    {
      "id": "conn_mysql_prod",
      "name": "生产库",
      "type": "relational",
      "config": {
        "dbType": "mysql",
        "host": "10.0.0.10",
        "port": 3306,
        "database": "production",
        "user": "runtime_user",
        "password": "encrypted:xxx"
      }
    },
    {
      "id": "conn_mqtt_emqx",
      "name": "EMQX",
      "type": "mqtt",
      "config": {
        "broker": "mqtt://10.0.0.20:1883",
        "username": "runtime",
        "password": "encrypted:yyy"
      }
    }
  ],
  "queries": [...],
  "datapoints": [...]
}
```

> **说明**：连接配置在工程设置中管理，支持加密存储敏感信息。部署时无需重新配置。

### 0.4 运行时配置（部署时设置）

部署时仅配置运行时参数：

```json
{
  "port": 8080,
  "uiTarget": "bigscreen",
  "defaultLocale": "zh-CN",
  "defaultTheme": "dark"
}
```

---

## 1. 架构概览

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            RuntimeEngine                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                          DataService（数据服务层）                    │   │
│  │                                                                       │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                  │   │
│  │  │ DataPoint   │  │ Subscription│  │ Cache       │                  │   │
│  │  │ Registry    │  │ Manager     │  │ Manager     │                  │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘                  │   │
│  │         │                │                │                          │   │
│  │         └────────────────┼────────────────┘                          │   │
│  │                          ▼                                           │   │
│  │  ┌───────────────────────────────────────────────────────────────┐  │   │
│  │  │                    Provider Adapters                           │  │   │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │  │   │
│  │  │  │ HTTP/REST   │  │ WebSocket   │  │ MQTT        │            │  │   │
│  │  │  │ Adapter     │  │ Adapter     │  │ Adapter     │            │  │   │
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘            │  │   │
│  │  └───────────────────────────────────────────────────────────────┘  │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                    │                                        │
│                                    ▼                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                     ConnectionProfile（节点侧配置）                   │   │
│  │  {                                                                   │   │
│  │    "dc_main": {                                                     │   │
│  │      "httpBase": "http://10.0.0.5:9092/api",                        │   │
│  │      "wsUrl": "ws://10.0.0.5:9092/socket.io"                        │   │
│  │    }                                                                 │   │
│  │  }                                                                   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                          Renderer Layer                               │   │
│  │  ┌─────────────────────────────┐  ┌─────────────────────────────┐   │   │
│  │  │     CanvasRenderer          │  │      DOMRenderer            │   │   │
│  │  │  (Konva - 图形/管道/符号)    │  │   (Vue - 组件渲染)          │   │   │
│  │  └─────────────────────────────┘  └─────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                          Other Services                               │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                  │   │
│  │  │ Auth        │  │ Action      │  │ Lifecycle   │                  │   │
│  │  │ Service     │  │ Executor    │  │ Manager     │                  │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘                  │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## 2. DataService（数据服务）

### 2.1 核心职责

- 解析工程 Schema 中的数据绑定
- 连接数据提供者
- 管理数据订阅
- 维护数据缓存
- 触发组件更新

### 2.2 DataPoint Registry

```typescript
class DataPointRegistry {
  private points: Map<string, DataPointInfo>;

  constructor(manifest: Manifest) {
    // 从 manifest 初始化数据点注册表
    for (const dp of manifest.dataRequirements.datapoints) {
      this.points.set(dp.path, {
        path: dp.path,
        sourceType: dp.sourceType,
        required: dp.required,
        value: undefined,
        status: "pending",
        lastUpdated: null,
      });
    }
  }

  getValue(path: string): any {
    return this.points.get(path)?.value;
  }

  setValue(path: string, value: any): void {
    const point = this.points.get(path);
    if (point) {
      point.value = value;
      point.status = "ready";
      point.lastUpdated = Date.now();
    }
  }

  getStatus(path: string): DataPointStatus {
    return this.points.get(path)?.status ?? "unknown";
  }
}
```

### 2.3 Subscription Manager

```typescript
class SubscriptionManager {
  private subscriptions: Map<string, SubscriptionInfo>;
  private refCounts: Map<string, number>;

  subscribe(path: string, callback: (value: any) => void): () => void {
    // 引用计数
    const count = (this.refCounts.get(path) ?? 0) + 1;
    this.refCounts.set(path, count);

    // 首次订阅
    if (count === 1) {
      this.createSubscription(path);
    }

    // 添加回调
    const callbacks = this.subscriptions.get(path)!.callbacks;
    callbacks.add(callback);

    // 返回取消订阅函数
    return () => {
      callbacks.delete(callback);
      const newCount = (this.refCounts.get(path) ?? 1) - 1;
      this.refCounts.set(path, newCount);

      // 最后一个订阅者
      if (newCount === 0) {
        this.destroySubscription(path);
      }
    };
  }

  private createSubscription(path: string): void {
    const sourceType = this.registry.getSourceType(path);
    const adapter = this.getAdapter(sourceType);

    const unsubscribe = adapter.subscribe(path, (value) => {
      this.registry.setValue(path, value);
      const sub = this.subscriptions.get(path);
      sub?.callbacks.forEach((cb) => cb(value));
    });

    this.subscriptions.set(path, {
      path,
      unsubscribe,
      callbacks: new Set(),
    });
  }

  private destroySubscription(path: string): void {
    const sub = this.subscriptions.get(path);
    sub?.unsubscribe();
    this.subscriptions.delete(path);
  }
}
```

### 2.4 Provider Adapters

```typescript
interface ProviderAdapter {
  connect(config: ProviderConfig): Promise<void>;
  disconnect(): void;
  subscribe(path: string, callback: (value: any) => void): () => void;
  request(path: string): Promise<any>;
}

// HTTP/REST Adapter
class HttpAdapter implements ProviderAdapter {
  private baseUrl: string;

  async connect(config: ProviderConfig): Promise<void> {
    this.baseUrl = config.httpBase;
  }

  async request(path: string): Promise<any> {
    const response = await fetch(
      `${this.baseUrl}/datapoints/${encodeURIComponent(path)}/value`
    );
    return response.json();
  }

  subscribe(path: string, callback: (value: any) => void): () => void {
    // HTTP 轮询实现
    const interval = setInterval(async () => {
      const value = await this.request(path);
      callback(value);
    }, 5000);

    return () => clearInterval(interval);
  }
}

// WebSocket Adapter
class WebSocketAdapter implements ProviderAdapter {
  private socket: Socket;
  private callbacks: Map<string, Set<(value: any) => void>>;

  async connect(config: ProviderConfig): Promise<void> {
    this.socket = io(config.wsUrl);

    this.socket.on("datapoint:value", (data) => {
      const cbs = this.callbacks.get(data.path);
      cbs?.forEach((cb) => cb(data.value));
    });
  }

  subscribe(path: string, callback: (value: any) => void): () => void {
    if (!this.callbacks.has(path)) {
      this.callbacks.set(path, new Set());
      this.socket.emit("datapoint:subscribe", { path });
    }
    this.callbacks.get(path)!.add(callback);

    return () => {
      this.callbacks.get(path)?.delete(callback);
      if (this.callbacks.get(path)?.size === 0) {
        this.socket.emit("datapoint:unsubscribe", { path });
        this.callbacks.delete(path);
      }
    };
  }
}
```

## 3. 资源生命周期管理

### 3.1 DisposableScope

```typescript
type Disposer = () => void;

class DisposableScope {
  private disposers: Disposer[] = [];
  private disposed = false;

  add(disposer: Disposer): void {
    if (this.disposed) {
      disposer();
      return;
    }
    this.disposers.push(disposer);
  }

  disposeAll(): void {
    if (this.disposed) return;
    this.disposed = true;

    // 逆序释放
    for (let i = this.disposers.length - 1; i >= 0; i--) {
      try {
        this.disposers[i]();
      } catch (e) {
        console.error("Dispose error:", e);
      }
    }
    this.disposers = [];
  }
}

// Vue Composable
function useDisposableScope(): DisposableScope {
  const scope = new DisposableScope();

  onUnmounted(() => {
    scope.disposeAll();
  });

  return scope;
}
```

### 3.2 组件使用示例

```typescript
// 组件内
setup(props) {
  const scope = useDisposableScope();
  const dataService = inject('dataService');
  const timerService = inject('timerService');

  // 订阅数据点
  const temperature = ref(null);
  scope.add(
    dataService.subscribe('mqtt.EMQX.温度组.temperature', (value) => {
      temperature.value = value;
    })
  );

  // 定时器
  scope.add(
    timerService.setInterval(() => {
      // 定时逻辑
    }, 5000)
  );

  // HTTP 请求
  const controller = new AbortController();
  scope.add(() => controller.abort());

  onMounted(async () => {
    const data = await fetch('/api/data', { signal: controller.signal });
    // ...
  });

  return { temperature };
}
```

### 3.3 平台 API 规范

所有可产生副作用的 API 必须返回 Disposer：

```typescript
interface DataService {
  subscribe(path: string, callback: (value: any) => void): Disposer;
}

interface TimerService {
  setTimeout(fn: () => void, delay: number): Disposer;
  setInterval(fn: () => void, interval: number): Disposer;
}

interface EventBus {
  on(event: string, handler: (...args: any[]) => void): Disposer;
}

interface ActionExecutor {
  // 请求使用 AbortController
  callApi(config: ApiConfig, signal?: AbortSignal): Promise<any>;
}
```

## 4. Watchdog（看门狗）

### 4.1 配置

```json
{
  "runtimeConfig": {
    "watchdog": {
      "enabled": true,
      "maxUptimeHours": 24,
      "maxErrorsPerMin": 10,
      "maxDisconnectedMinutes": 30,
      "action": "reload",
      "healthCheckInterval": 60000
    }
  }
}
```

### 4.2 实现

```typescript
class Watchdog {
  private startTime: number;
  private errorCounts: number[] = [];
  private disconnectedSince: number | null = null;
  private config: WatchdogConfig;

  constructor(config: WatchdogConfig) {
    this.config = config;
    this.startTime = Date.now();

    // 定时检查
    setInterval(() => this.check(), config.healthCheckInterval);
  }

  recordError(error: Error): void {
    this.errorCounts.push(Date.now());
    // 只保留最近一分钟的错误
    const oneMinAgo = Date.now() - 60000;
    this.errorCounts = this.errorCounts.filter((t) => t > oneMinAgo);
  }

  setConnectionStatus(connected: boolean): void {
    if (!connected && this.disconnectedSince === null) {
      this.disconnectedSince = Date.now();
    } else if (connected) {
      this.disconnectedSince = null;
    }
  }

  private check(): void {
    // 检查运行时间
    const uptimeHours = (Date.now() - this.startTime) / (1000 * 60 * 60);
    if (uptimeHours > this.config.maxUptimeHours) {
      this.triggerAction("uptime exceeded");
      return;
    }

    // 检查错误率
    if (this.errorCounts.length > this.config.maxErrorsPerMin) {
      this.triggerAction("error rate exceeded");
      return;
    }

    // 检查断连时间
    if (this.disconnectedSince !== null) {
      const disconnectedMinutes =
        (Date.now() - this.disconnectedSince) / (1000 * 60);
      if (disconnectedMinutes > this.config.maxDisconnectedMinutes) {
        this.triggerAction("disconnected too long");
        return;
      }
    }
  }

  private triggerAction(reason: string): void {
    console.warn(`Watchdog triggered: ${reason}`);

    switch (this.config.action) {
      case "reload":
        location.reload();
        break;
      case "notify":
        this.notifyAgent(reason);
        break;
      case "restart":
        this.requestRestart(reason);
        break;
    }
  }
}
```

### 4.3 NodeAgent 协同

```typescript
// RuntimeEngine 向 NodeAgent 上报健康状态
class HealthReporter {
  private interval: number;

  start(): void {
    this.interval = setInterval(() => {
      this.report();
    }, 30000);
  }

  private async report(): Promise<void> {
    const status = {
      uptime: Date.now() - this.startTime,
      memory: performance.memory?.usedJSHeapSize,
      connections: this.dataService.getActiveConnections(),
      errors: this.watchdog.getRecentErrors(),
      lastDataUpdate: this.dataService.getLastUpdateTime(),
    };

    try {
      await fetch("/agent/health", {
        method: "POST",
        body: JSON.stringify(status),
      });
    } catch (e) {
      // Agent 不可达
    }
  }
}
```

## 5. 启动流程

```typescript
class RuntimeEngine {
  private dataService: DataService;
  private watchdog: Watchdog;
  private authService: AuthService;
  private actionExecutor: ActionExecutor;

  async start(ifpUrl: string): Promise<void> {
    // 1. 加载工程制品
    const { manifest, project, assetMapping } = await this.loadIFP(ifpUrl);

    // 2. 加载连接配置
    const connectionProfile = await this.loadConnectionProfile();

    // 3. 验证 dataRequirements
    this.validateDataRequirements(manifest, connectionProfile);

    // 4. 初始化数据服务
    this.dataService = new DataService(manifest, connectionProfile);
    await this.dataService.connect();

    // 5. 初始化认证服务（如果需要）
    if (manifest.security.mode !== "none") {
      this.authService = new AuthService(manifest.security);
    }

    // 6. 初始化动作执行器
    this.actionExecutor = new ActionExecutor(
      this.dataService,
      this.authService
    );

    // 7. 启动 Watchdog
    this.watchdog = new Watchdog(project.runtimeConfig?.watchdog);

    // 8. 启动健康上报
    new HealthReporter(this).start();

    // 9. 渲染应用
    this.render(project, assetMapping);
  }
}
```

## 6. 多端选路

```typescript
class RuntimeEngine {
  resolvePageView(path: string): PageNode | null {
    // 从 ConnectionProfile 获取当前 target
    const currentTarget = this.connectionProfile.uiTarget ?? "pc";

    const pages = Object.values(this.project.pagesById).filter(
      (p) => p.path === path
    );

    // 1. 精确匹配当前 target
    let page = pages.find((p) => p.target === currentTarget);

    // 2. fallback 到默认视图
    if (!page) {
      page = pages.find((p) => p.isDefaultTarget);
    }

    // 3. 再 fallback 到 pc
    if (!page) {
      page = pages.find((p) => p.target === "pc");
    }

    return page ?? null;
  }
}
```

## 7. Canvas 图形运行时渲染

RuntimeEngine 需要同时渲染 DOM 组件和 Canvas 图形。

### 7.1 架构

```
┌─────────────────────────────────────────────────────────────────┐
│                      RuntimeRenderer                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │              CanvasRenderer (z-index: 1)                 │   │
│  │                                                         │   │
│  │  graphicsById → Konva Shapes                            │   │
│  │  - 绑定解析 → props 更新                                 │   │
│  │  - 动画驱动（管道流动等）                                │   │
│  │                                                         │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │              DOMRenderer (z-index: 10)                   │   │
│  │                                                         │   │
│  │  nodesById → Vue Components                             │   │
│  │  - 绑定解析 → props 更新                                 │   │
│  │  - 事件处理                                              │   │
│  │                                                         │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 7.2 图形绑定解析

Canvas 图形的绑定解析与 DOM 组件相同：

```typescript
class CanvasRenderer {
  private stage: Konva.Stage;
  private layer: Konva.Layer;
  private shapesMap: Map<string, Konva.Shape> = new Map();
  private disposableScope: DisposableScope;

  constructor(
    private container: HTMLElement,
    private graphicsById: Record<string, GraphicNode>,
    private dataService: DataService
  ) {
    this.initStage();
    this.renderGraphics();
  }

  private renderGraphics(): void {
    for (const [id, graphic] of Object.entries(this.graphicsById)) {
      const shape = this.createShape(graphic);
      this.shapesMap.set(id, shape);
      this.layer.add(shape);

      // 设置数据绑定
      this.setupBindings(id, graphic);
    }
    this.layer.draw();
  }

  private setupBindings(graphicId: string, graphic: GraphicNode): void {
    for (const [propKey, binding] of Object.entries(graphic.bindings)) {
      if (binding.kind === "datapoint") {
        // 订阅数据点
        const unsubscribe = this.dataService.subscribe(binding.path, (value) =>
          this.updateGraphicProp(graphicId, propKey, value, binding.transform)
        );
        this.disposableScope.add(unsubscribe);
      }
    }
  }

  private updateGraphicProp(
    graphicId: string,
    propKey: string,
    value: any,
    transform?: Transform[]
  ): void {
    const shape = this.shapesMap.get(graphicId);
    if (!shape) return;

    // 应用转换
    const transformedValue = applyTransform(value, transform);

    // 更新 Konva shape 属性
    shape.setAttr(this.mapPropToAttr(propKey), transformedValue);
    this.layer.batchDraw();
  }
}
```

### 7.3 管道流动动画

```typescript
class PipeRenderer {
  private animationController: AnimationController;

  renderPipe(graphic: GraphicNode): Konva.Group {
    const pipe = new Konva.Group();

    // 绘制管道主体
    const body = new Konva.Line({
      points: flattenPoints(graphic.props.points),
      stroke: graphic.props.strokeColor,
      strokeWidth: graphic.props.width,
      lineCap: "round",
      lineJoin: "round",
    });
    pipe.add(body);

    // 创建流动动画层
    const flowIndicator = new Konva.Line({
      points: flattenPoints(graphic.props.points),
      stroke: graphic.props.flowColor ?? "#ffffff",
      strokeWidth: graphic.props.width * 0.3,
      dash: graphic.props.flowDash ?? [10, 10],
      dashOffset: 0,
    });
    pipe.add(flowIndicator);

    // 启动流动动画（由绑定的 flowSpeed 驱动）
    if (graphic.bindings.flowSpeed) {
      this.animationController.register(graphic.id, (speed: number) => {
        if (speed > 0) {
          const direction = graphic.props.flowDirection === "backward" ? -1 : 1;
          flowIndicator.dashOffset(
            flowIndicator.dashOffset() + speed * direction
          );
        }
      });
    }

    return pipe;
  }
}
```

### 7.4 符号渲染

```typescript
class SymbolRenderer {
  private symbolsById: Record<string, SymbolDef>;

  renderSymbol(graphic: GraphicNode): Konva.Group {
    const symbolDef = this.symbolsById[graphic.props.symbolId];
    if (!symbolDef) {
      console.warn(`Symbol not found: ${graphic.props.symbolId}`);
      return new Konva.Group();
    }

    const group = new Konva.Group({
      x: graphic.props.x,
      y: graphic.props.y,
      scaleX: graphic.props.scale ?? 1,
      scaleY: graphic.props.scale ?? 1,
      rotation: graphic.props.rotation ?? 0,
    });

    // 渲染符号内的基础图形
    for (const primitive of symbolDef.graphics) {
      const shape = this.createPrimitive(primitive);
      group.add(shape);
    }

    return group;
  }
}
```

### 7.5 事件处理

```typescript
// Canvas 图形事件绑定
function bindGraphicEvents(
  shape: Konva.Shape,
  graphic: GraphicNode,
  actionExecutor: ActionExecutor
): void {
  for (const [eventName, actions] of Object.entries(graphic.events)) {
    shape.on(eventName, (e) => {
      const context = {
        $event: { target: graphic.id, type: eventName },
        $graphic: graphic,
      };
      actionExecutor.executeActions(actions, context);
    });
  }
}
```

## 8. 测试要点

- [ ] DataService 订阅/取消订阅正确性
- [ ] 引用计数订阅合并
- [ ] Provider Adapters 各类型实现
- [ ] DisposableScope 生命周期管理
- [ ] Watchdog 各触发条件
- [ ] 启动流程完整性
- [ ] 多端选路规则
- [ ] 健康状态上报
- [ ] **Canvas 图形渲染正确性**
- [ ] **Canvas 图形数据绑定更新**
- [ ] **管道流动动画正常**
- [ ] **符号渲染和旋转正确**
- [ ] **Canvas 图形事件触发正确**

---

**相关文档**：

- [发布流水线](./publish-pipeline.md)
- [数据绑定 v2](./data-binding-v2.md)
- [设计态交互](./design-interaction.md)
- [多端适配](./multi-view.md)

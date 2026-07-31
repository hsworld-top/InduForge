# 工业采集驱动目录全量加载实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 让工业采集连接向导加载并展示全部驱动，不受目录接口单页 100 条限制。

**架构：** 保留 Data Service 分页契约，在 Datacenter API 层先读取第一页，再按 `totalPages` 并行读取剩余页并顺序合并。连接向导消费全量方法，现有驱动树负责搜索、分组和滚动展示。

**技术栈：** Vue 3、TypeScript、Axios、Zod、Vitest。

---

## 文件结构

- 创建 `datacenter/tests/collector-api.test.ts`：验证多页驱动目录完整合并和请求参数。
- 修改 `datacenter/src/api/collector.api.ts`：提供全量驱动目录加载方法。
- 修改 `datacenter/src/components/collector-workbench/CollectorConnectionWizard.vue`：连接向导改用全量目录。

### 任务 1：全量加载驱动目录

**文件：**
- 创建：`datacenter/tests/collector-api.test.ts`
- 修改：`datacenter/src/api/collector.api.ts:39-45`
- 修改：`datacenter/src/components/collector-workbench/CollectorConnectionWizard.vue:133-137,203-215`

- [x] **步骤 1：编写失败的多页目录测试**

创建请求 Mock，第一页返回 100 条、`totalPages: 2`，第二页返回 15 条；调用 `listAllCollectorDrivers()` 后断言结果为 115 条、首尾 ID 正确，并断言两次请求分别使用页码 1 和 2、每页 100 条。

```ts
const { requestMock } = vi.hoisted(() => ({ requestMock: vi.fn() }))
vi.mock('@/utils/request', () => ({ default: requestMock }))

it('loads and merges every collector driver page in order', async () => {
  requestMock
    .mockResolvedValueOnce(apiPage(1, drivers.slice(0, 100), 115, 2))
    .mockResolvedValueOnce(apiPage(2, drivers.slice(100), 115, 2))

  const result = await listAllCollectorDrivers()

  expect(result).toHaveLength(115)
  expect(result[0]?.driverId).toBe('driver-001')
  expect(result[114]?.driverId).toBe('driver-115')
  expect(requestMock).toHaveBeenNthCalledWith(
    2,
    expect.objectContaining({ params: { page: 2, pageSize: 100 } }),
  )
})
```

- [x] **步骤 2：运行测试确认失败**

运行：`pnpm --dir datacenter test -- collector-api.test.ts`

预期：FAIL，提示 `listAllCollectorDrivers` 尚未导出。

- [x] **步骤 3：实现 API 全量加载方法**

在 `collector.api.ts` 中复用现有 `listCollectorDrivers`：

```ts
const collectorDriverCatalogPageSize = 100

export async function listAllCollectorDrivers(): Promise<CollectorDriverSummary[]> {
  const firstPage = await listCollectorDrivers({ page: 1, pageSize: collectorDriverCatalogPageSize })
  const remainingPages = await Promise.all(
    Array.from({ length: Math.max(0, firstPage.pagination.totalPages - 1) }, (_, index) =>
      listCollectorDrivers({ page: index + 2, pageSize: collectorDriverCatalogPageSize }),
    ),
  )
  return [firstPage, ...remainingPages].flatMap((page) => page.list)
}
```

`Promise.all` 保留输入顺序；任意请求失败时方法整体抛错，由向导现有错误分支清空列表并提示失败。

- [x] **步骤 4：连接向导改用完整目录**

将 `CollectorConnectionWizard.vue` 的导入和加载逻辑改为：

```ts
import {
  createCollectorConnection,
  getCollectorDriver,
  listAllCollectorDrivers,
} from '@/api/collector.api'

drivers.value = await listAllCollectorDrivers()
```

- [x] **步骤 5：运行定向测试和静态检查**

运行：

```powershell
pnpm --dir datacenter test -- collector-api.test.ts
pnpm --dir datacenter typecheck
```

预期：测试通过，TypeScript 零错误。

- [x] **步骤 6：运行 Datacenter 全量验证**

运行：

```powershell
pnpm --dir datacenter test
pnpm --dir datacenter build
git diff --check
```

预期：全部测试通过，生产构建成功，无空白格式错误。

# Roadmap: InduForge

## Overview

本路线图聚焦“在不改动总体架构前提下，完成契约统一、发布闭环、运行可观测与运维稳态化”。阶段顺序按依赖推进，优先降低跨模块联调不确定性。

## Phases

**Phase Numbering:**
- 整数阶段（1,2,3...）为常规里程碑工作
- 小数阶段（2.1,2.2...）用于紧急插入工作

- [ ] **Phase 1: API 契约统一** - 统一后端响应结构与错误语义。
- [ ] **Phase 2: 发布产物契约闭环** - 固化 Designer 到 IFP 的发布契约与校验。
- [ ] **Phase 3: 运行态可观测打通** - 落实 Runtime/NodeAgent/平台状态协议。
- [ ] **Phase 4: 鉴权基线收敛** - 收敛开发旁路并补关键权限回归。
- [ ] **Phase 5: 运维稳态与交付收尾** - 修复执行链路一致性并补 smoke 验证。

## Phase Details

### Phase 1: API 契约统一
**Goal**: 让前端面对统一的成功/失败语义，降低跨模块联调成本。  
**Depends on**: Nothing (first phase)  
**Requirements**: [APIC-01, APIC-02, APIC-03]  
**Success Criteria** (what must be TRUE):
1. `dev_core` 目标路由统一返回 `ApiResponse`。
2. 业务异常统一映射到 `AppError` + `ErrorCodes`。
3. 接口日志与返回体均可追踪 `requestId`。
**Plans**: 3 plans

Plans:
- [ ] 01-01: 盘点并改造非统一响应路由（v1 关键路径）
- [ ] 01-02: 建立统一错误映射与中间件收口
- [ ] 01-03: 增补接口契约与回归测试

### Phase 2: 发布产物契约闭环
**Goal**: 让发布制品可被节点侧稳定消费，避免“发布成功但运行失败”。  
**Depends on**: Phase 1  
**Requirements**: [PUBC-01, PUBC-02, PUBC-03]  
**Success Criteria** (what must be TRUE):
1. Designer 输出结构满足约定 schema。
2. IFP 产物包含 manifest/pages/assets/data 等运行必需内容。
3. 发布前校验失败会阻断发布并给出可定位信息。
**Plans**: 3 plans

Plans:
- [ ] 02-01: 对齐发布 schema 与导出字段
- [ ] 02-02: 完善 IFP 打包内容与 manifest 生成
- [ ] 02-03: 实现发布前契约校验与失败提示

### Phase 3: 运行态可观测打通
**Goal**: 平台可以持续看到节点运行状态并快速定位故障。  
**Depends on**: Phase 2  
**Requirements**: [RTOB-01, RTOB-02, RTOB-03]  
**Success Criteria** (what must be TRUE):
1. Runtime `GET /health` 稳定可用于探活。
2. Runtime `GET /status` 返回约定状态字段。
3. 平台侧状态展示与 NodeAgent 状态一致。
**UI hint**: yes  
**Plans**: 3 plans

Plans:
- [ ] 03-01: 完成 Runtime 健康/状态接口实现与契约测试
- [ ] 03-02: 对齐 NodeAgent 状态采集与上报逻辑
- [ ] 03-03: 平台侧状态展示与错误摘要联动

### Phase 4: 鉴权基线收敛
**Goal**: 保证开发与生产权限模型一致，减少越权与联调偏差。  
**Depends on**: Phase 3  
**Requirements**: [SECU-01, SECU-02, SECU-03]  
**Success Criteria** (what must be TRUE):
1. 开发旁路必须显式开启且默认关闭。
2. 关键路由完成能力与访问校验收敛。
3. 权限回归用例覆盖失效 token 与越权访问。
**Plans**: 2 plans

Plans:
- [ ] 04-01: 收敛鉴权旁路与关键路由守卫
- [ ] 04-02: 增加权限回归测试与异常场景验证

### Phase 5: 运维稳态与交付收尾
**Goal**: 提升部署执行可复盘性，形成可持续交付基线。  
**Depends on**: Phase 4  
**Requirements**: [OPSR-01, OPSR-02, OPSR-03]  
**Success Criteria** (what must be TRUE):
1. NodeAgent 日志不再覆盖，能追溯历史关键事件。
2. Makefile 目标和文档一致，常用命令一次可执行。
3. 发布-部署链路有最小 smoke 验证脚本与操作说明。
**Plans**: 2 plans

Plans:
- [ ] 05-01: 修复日志策略与构建脚本一致性
- [ ] 05-02: 建立发布-部署 smoke 验证并沉淀文档

## Progress

**Execution Order:**
Phases execute in numeric order: 1 -> 2 -> 3 -> 4 -> 5

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. API 契约统一 | 0/3 | Not started | - |
| 2. 发布产物契约闭环 | 0/3 | Not started | - |
| 3. 运行态可观测打通 | 0/3 | Not started | - |
| 4. 鉴权基线收敛 | 0/2 | Not started | - |
| 5. 运维稳态与交付收尾 | 0/2 | Not started | - |

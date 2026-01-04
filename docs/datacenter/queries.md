# 查询管理

本文档面向内部开发，描述当前查询管理能力与接口范围（高层）。

## 核心能力

- 查询定义（基于连接）
- 查询执行（参数化）
- 执行日志记录

## 当前接口范围

- 查询列表：`GET /api/v1/data/projects/:projectId/queries`
- 创建查询：`POST /api/v1/data/projects/:projectId/queries`
- 执行查询：`POST /api/v1/data/queries/:id/execute`

## 当前限制

- 查询更新/删除接口暂未在后端实现（前端 UI 已预留）
- 仅支持 SQL 类型查询执行


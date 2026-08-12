# data_service 协作规则

## 模块定位

- `data_service` 是平台侧与开发态的数据域服务，负责连接、查询、数据点、协议接入、计算、预览会话与数据域 API。

## 进入前先看

- `data_service/go.mod`
- `data_service/Makefile`
- `data_service/cmd/`
- `data_service/internal/`
- `docs/03-模块设计/data_service/README.md`

## 开发约束

- 运行时为 Go，优先沿用 `cmd/` 与 `internal/` 的现有分层，不把平台治理逻辑混入该模块。
- 数据连接、查询、数据点、协议接入和计算能力要保持同一数据域模型下的连续性。
- 预览会话与开发态数据链路改动要保持当前链路的一致性，不要只修单点接口而破坏整条链路。
- 数据库结构只维护 `internal/db/schema/schema.sql` 最终建库基线；已有开发库变更直接执行 SQL，不新增迁移目录或历史兼容逻辑。
- 当前验证入口以 `Makefile` 为准。

## 禁止事项

- 不要把 `data_service` 扩成平台治理或发布部署服务。
- 不要新增版本迁移、down 脚本、旧表搬迁、字段回填或启动时结构修复。
- 不要在当前任务中顺手扩展无关协议域或运行态职责。

## 验证命令

- `make -C data_service test`
- `make -C data_service build`

## 相关契约

- `docs/03-模块设计/data_service/README.md`
- `docs/03-模块设计/datacenter/README.md`
- `docs/01-产品与架构/平台系统架构.md`
- `docs/02-系统设计/README.md`

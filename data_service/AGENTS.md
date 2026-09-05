# data_service 协作规则

## 模块定位

- `data_service` 是平台侧与开发态的数据域服务，负责连接、查询、数据点、协议接入、计算、预览会话与数据域 API。

## 按任务查阅

以下路径和命令均从仓库根目录解析，只读取任务涉及的文件与章节。

- 调整模块职责或数据域边界：`docs/03-模块设计/data_service/README.md`；涉及 DataCenter 消费时，再查 `docs/03-模块设计/datacenter/README.md`。
- 修改接口、预览或数据逻辑：定位 `data_service/internal/` 中对应处理器、服务和测试；仅启动装配变更涉及 `data_service/cmd/`。
- 修改数据库结构：`data_service/internal/db/schema/schema.sql` 与 `docs/04-契约与规范/数据库设计.md`。
- 修改依赖或校验入口：`data_service/go.mod`、`data_service/Makefile`。

## 开发约束

- 运行时为 Go，优先沿用模块现有的入口、服务与数据访问分层，不把平台治理逻辑混入该模块。
- 数据连接、查询、数据点、协议接入和计算能力要保持同一数据域模型下的连续性。
- 预览会话与开发态数据链路改动要保持当前链路的一致性，不要只修单点接口而破坏整条链路。
- 数据库结构只维护 `data_service/internal/db/schema/schema.sql` 最终建库基线；已有开发库变更直接执行 SQL，不新增迁移目录或历史兼容逻辑。

## 禁止事项

- 不要把 `data_service` 扩成平台治理或发布部署服务。
- 不要新增版本迁移、down 脚本、旧表搬迁、字段回填或启动时结构修复。
- 不要在当前任务中顺手扩展无关协议域或运行态职责。

## 按影响面验证

- 局部 Go 逻辑优先在 `data_service` 模块内对受影响包执行 `go test`；跨包影响需要模块回归时使用 `make -C data_service test`。
- 入口、依赖或编译关系变更使用 `make -C data_service build`；数据库变更另按数据库设计验证最终基线。
- 以上是按需选择的入口，不要求每次全部执行；必要验证通过后即可结束，不为纯文档改动运行业务测试。

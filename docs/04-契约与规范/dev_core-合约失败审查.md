# dev_core 合约失败审查

2026-09-13 执行 `go test ./tests/contract` 有两项失败。OpenAPI 标记为 dev_ide 的 5 个操作中，前端接口清单缺少头像读写和节点指标接口，属于前端库存与契约不同步，不应通过弱化测试解决。另有节点审批的真实 HTTP 证据缺失：dev_ide 已有接口调用，但 Go 合约测试扫描不到对应请求样例，属于证据覆盖缺口。建议补齐 dev_ide API inventory 后，再为四个操作增加真实 httptest；在此之前不调整 x-client 标记或删除契约操作。

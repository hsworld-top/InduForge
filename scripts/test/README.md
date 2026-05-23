# 模拟数据测试脚本

本目录专门存放开发联调、演示验证时使用的模拟数据脚本。

这些脚本用于手动验证协议接入、消息流、实时数据等场景，不属于自动化单元测试；如果后续需要写单元测试或集成测试，请放到对应模块自己的测试目录。

## MQTT 模拟发布

```bash
node scripts/test/mqtt-publish-test.js
```

指定地址、主题和间隔：

```bash
node scripts/test/mqtt-publish-test.js --broker mqtt://127.0.0.1:18883 --topic induforge/mock-data --interval 500
```

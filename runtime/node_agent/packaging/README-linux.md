# InduForge NodeAgent（Linux）

安装包按 `amd64` 与 `arm64` 分别发布；安装器会校验当前机器架构与包内制品是否匹配。

## 安装并接入中心

```bash
tar -xzf induforge-node-*.tar.gz
cd induforge-node-agent-*
sudo ./install.sh \
  --server-url "https://你的中心地址" \
  --enrollment-code "中心生成的一次性接入码"
```

Linux 包会安装 `project-gateway`、`runtime-api`、`runtime-engine` 能力模板；如节点还需
采集能力，在安装时显式加入 `--enable-collector`：

```bash
sudo ./install.sh --enable-collector --server-url "https://你的中心地址" --enrollment-code "一次性接入码"
```

默认安装目录为 `/opt/induforge/node-agent`，配置文件为
`/etc/induforge/node-agent/config.yaml`。两个接入参数齐全时，安装器会创建并启动
`induforge-node-agent.service`；接入码领取成功后会从配置中清除。

## 接入、ReleaseStore 与工程部署的边界

安装后模板会声明本机已安装能力，因此即使尚未有工程 release，NodeAgent 也能完成领取并
显示为在线。模板默认 `enabled: false`，不会自动启动任何 capability 子进程，也不会把未
配置的服务伪报为运行中。

旧本机 Supervisor 配置需要运维人员在 `config.yaml` 中为服务补齐 `releaseRoot`、`releaseDigest`、
`executable`、`healthUrl`（以及可选 `arguments`、`environment`）；它不是正式 Release 部署的替代品。
正式命令中 Agent 会以自身 token 拉取节点专属 Binding 和同源 `application/zstd` Release 流，拒绝
重定向、外部 URL、中心下发路径/命令/环境变量/Secret；签名公钥只来自本机 trust store。当前正式
launcher 尚未交付，因此完成安全安装后会明确拒绝启动，不能通过手工补齐旧配置绕过门禁。

`ReleaseStore` 已由正式命令调和调用：它限额流式写入、验证 outer SHA-256、签名与所有摘要、
版本/能力兼容性，内容寻址保存并原子写入 `current.json`。它不负责 Secret、内层工件物化、
RuntimeEngine 的真实只读 mount、Runtime Foundation、启动或回滚服务。因此它不是“安装包已能
部署工程”的证明。

Release tar.zst 根目录的标准文件为：

```text
release-manifest.json
client-assets.tar.zst
runtime-artifact.tar.zst
health-contract.json
resource-recommendation.json
schema-plan.json
sbom.cdx.json
collector-artifact.tar.zst       # 可选
checksums.json
signature.sig
```

`checksums.json` 必须使用 `release-checksums.v1`，包含按路径严格升序的
`{path,sha256,size}` 文件清单；它覆盖以上固定 Release 文件（可选 Collector 仅在 Manifest 声明
时出现），不包含自身和 `signature.sig`。`signature.sig` 是对精确原始 checksums 字节的 64-byte
Ed25519 签名，且 Manifest 的 `supplyChain.signingKeyId` 必须等于本地受信 key ID。压缩包不能
含目录或其他路径。失败绝不激活 `current.json`。

即使 Center 同机安装，也不能把它放入本安装目录、复用服务账户或嵌入 NodeAgent 进程；Center 和
Agent 继续是独立服务、身份、目录、端口和网络连接。

常用命令：

```bash
sudo systemctl status induforge-node-agent
sudo journalctl -u induforge-node-agent -f
sudo systemctl restart induforge-node-agent
```

## 容器或手工验收

没有 systemd 时可只安装文件：

```bash
./install.sh --prefix /app --config-dir /config --no-service
NODE_AGENT_CONFIG=/config/config.yaml NODE_AGENT_WORKDIR=/app NODE_AGENT_DATA_DIR=/app/data \
  /app/bin/node-agent --daemon
```

## 卸载

`sudo ./uninstall.sh` 会停止并清理本节点受管 K3s、容器进程、CNI 网络、挂载点和
节点安装时指定的数据目录（包括本地 PVC），同时删除 NodeAgent 程序与服务；默认
保留节点身份、配置和日志，便于诊断与按原身份重装。

确实需要把服务器恢复为未接入状态时，再使用 `sudo ./uninstall.sh --purge` 清除
保留的节点身份、配置和日志。基础服务数据会在两种卸载方式下清理，因此执行前应
由运维管理员确认该节点已不再承载需要保留的数据。

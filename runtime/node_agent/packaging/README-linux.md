# InduForge NodeAgent（Linux）

本安装包同时包含 `amd64` 与 `arm64` 可执行文件，安装器会自动选择当前机器架构。

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

## 接入与工程 release 的边界

安装后模板会声明本机已安装能力，因此即使尚未有工程 release，NodeAgent 也能完成领取并
显示为在线。模板默认 `enabled: false`，不会自动启动任何 capability 子进程，也不会把未
配置的服务伪报为运行中。

收到工程部署命令前，本机运维人员必须在 `config.yaml` 中为要启用的服务补齐
`releaseRoot`、`releaseDigest`、`executable`、`healthUrl`（以及可选 `arguments`、`environment`）。
release 目录和清单必须已由受控物料链放入安装目录；NodeAgent 不下载也不接受中心下发的
路径、命令或密钥。`project-gateway` 的对外地址只能通过该本机组件的 `publicUrl` 配置。
缺少任一 release 字段时，服务保持不可启动并 fail-closed。

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

`sudo ./uninstall.sh` 仅删除程序和服务，默认保留配置、节点身份、日志与诊断数据。
确实需要重新注册时再使用 `sudo ./uninstall.sh --purge` 清除全部状态。

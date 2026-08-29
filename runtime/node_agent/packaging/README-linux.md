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

默认安装目录为 `/opt/induforge/node-agent`，配置文件为
`/etc/induforge/node-agent/config.yaml`。两个接入参数齐全时，安装器会创建并启动
`induforge-node-agent.service`；接入码领取成功后会从配置中清除。

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

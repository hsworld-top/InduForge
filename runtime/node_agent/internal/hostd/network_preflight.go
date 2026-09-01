package hostd

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"
)

// networkPreflight 在写入任何 K3s 文件前检查本机端口占用，并验证工作节点
// 能主动连接中心 API。UDP 检查只能发现本机占用；跨节点防火墙连通性仍需
// 安装验收从两端探测并给出明确的 UDP 8472 放行提示。
func networkPreflight(ctx context.Context, plan ClusterPlan) error {
	if err := checkLocalPort(ctx, "tcp", plan.NodeIP, plan.KubeletPort, "Kubelet"); err != nil {
		return err
	}
	if err := checkLocalPort(ctx, "udp", plan.NodeIP, plan.VXLANPort, "Flannel VXLAN"); err != nil {
		return err
	}
	if plan.Operation == OperationInitServer {
		return checkLocalPort(ctx, "tcp", plan.NodeIP, plan.APIPort, "K3s API")
	}
	parsed, err := url.Parse(plan.ServerURL)
	if err != nil {
		return fmt.Errorf("解析 K3s Server 地址失败: %w", err)
	}
	dialer := net.Dialer{Timeout: 3 * time.Second}
	connection, err := dialer.DialContext(ctx, "tcp", parsed.Host)
	if err != nil {
		return fmt.Errorf("无法连接中心 K3s API %s，请检查 TCP %d 防火墙和路由: %w", parsed.Host, plan.APIPort, err)
	}
	return connection.Close()
}

func checkLocalPort(ctx context.Context, network, host string, port int, purpose string) error {
	address := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	var (
		closer interface{ Close() error }
		err    error
	)
	listenConfig := net.ListenConfig{}
	if network == "udp" {
		closer, err = listenConfig.ListenPacket(ctx, network, address)
	} else {
		closer, err = listenConfig.Listen(ctx, network, address)
	}
	if err != nil {
		return fmt.Errorf("%s 所需的 %s %d 已被占用或不可绑定: %w", purpose, network, port, err)
	}
	return closer.Close()
}

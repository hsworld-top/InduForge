package ops

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// SetNodeConnectionURL 仅接受安装人员确认的入口，不从浏览器 Host 或容器网卡推断。
func (h *Handler) SetNodeConnectionURL(value string) error {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	if value == "" {
		h.nodeConnectionURL = ""
		return nil
	}
	u, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("NODE_CONNECTION_URL 地址无效")
	}
	ip := net.ParseIP(u.Hostname())
	if (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || strings.EqualFold(u.Hostname(), "localhost") || (ip != nil && (ip.IsLoopback() || ip.IsUnspecified())) {
		return fmt.Errorf("NODE_CONNECTION_URL 必须是节点可访问的 HTTP(S) 中心地址，不能使用本机回环地址、路径或凭据")
	}
	h.nodeConnectionURL = value
	return nil
}

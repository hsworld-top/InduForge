package hostd

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	K3sVersion = "v1.36.4+k3s1"

	OperationInitServer = "init-server"
	OperationJoinAgent  = "join-agent"
)

var (
	uuidPattern     = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	tokenPattern    = regexp.MustCompile(`^[A-Za-z0-9._:-]+$`)
	safePathPattern = regexp.MustCompile(`^/[A-Za-z0-9._/-]+$`)
)

// ClusterPlan 是 NodeAgent 能交给 root Hostd 的完整声明式输入。Hostd 不接受
// shell、额外参数、环境变量或中心提供的文件路径，避免控制面扩大为远程执行通道。
type ClusterPlan struct {
	SchemaVersion string `json:"schemaVersion"`
	Generation    int64  `json:"generation"`
	ClusterID     string `json:"clusterId"`
	NodeID        string `json:"nodeId"`
	Operation     string `json:"operation"`
	K3sVersion    string `json:"k3sVersion"`
	ServerURL     string `json:"serverUrl,omitempty"`
	Token         string `json:"token"`
	NodeIP        string `json:"nodeIp"`
	DataDir       string `json:"dataDir"`
	APIPort       int    `json:"apiPort"`
	VXLANPort     int    `json:"vxlanPort"`
	KubeletPort   int    `json:"kubeletPort"`
}

type ClusterState struct {
	SchemaVersion string `json:"schemaVersion"`
	Generation    int64  `json:"generation"`
	ClusterID     string `json:"clusterId"`
	NodeID        string `json:"nodeId"`
	Operation     string `json:"operation"`
	K3sVersion    string `json:"k3sVersion"`
	NodeName      string `json:"nodeName"`
	NodeIP        string `json:"nodeIp"`
	DataDir       string `json:"dataDir"`
	ServiceName   string `json:"serviceName"`
	ObservedState string `json:"observedState"`
	Message       string `json:"message,omitempty"`
}

type UninstallRequest struct {
	ClusterID string `json:"clusterId"`
	NodeID    string `json:"nodeId"`
	PurgeData bool   `json:"purgeData"`
}

func (plan ClusterPlan) Validate() error {
	if plan.SchemaVersion != "induforge.cluster-plan.v1" {
		return errors.New("不支持的集群计划版本")
	}
	if plan.Generation < 1 {
		return errors.New("集群计划 generation 必须大于 0")
	}
	if !uuidPattern.MatchString(strings.ToLower(plan.ClusterID)) || !uuidPattern.MatchString(strings.ToLower(plan.NodeID)) {
		return errors.New("运行集群或节点 ID 无效")
	}
	if plan.Operation != OperationInitServer && plan.Operation != OperationJoinAgent {
		return errors.New("不支持的 K3s 初始化操作")
	}
	if plan.K3sVersion != K3sVersion {
		return fmt.Errorf("K3s 版本必须固定为 %s", K3sVersion)
	}
	if len(plan.Token) < 32 || len(plan.Token) > 256 || !tokenPattern.MatchString(plan.Token) {
		return errors.New("K3s 集群令牌格式无效")
	}
	ip := net.ParseIP(strings.TrimSpace(plan.NodeIP))
	if ip == nil || ip.IsUnspecified() || ip.IsLoopback() || ip.IsMulticast() {
		return errors.New("节点 IP 必须是可用于节点间通信的单播地址")
	}
	if err := validateDataDir(plan.DataDir); err != nil {
		return err
	}
	if plan.APIPort < 1 || plan.APIPort > 65535 {
		return errors.New("K3s API 端口无效")
	}
	if plan.VXLANPort != 8472 || plan.KubeletPort != 10250 {
		return errors.New("当前 K3s 网络基线要求 UDP 8472 和 TCP 10250")
	}
	if plan.Operation == OperationInitServer {
		if strings.TrimSpace(plan.ServerURL) != "" {
			return errors.New("首节点初始化不能指定 serverUrl")
		}
		return nil
	}
	return validateServerURL(plan.ServerURL, plan.APIPort)
}

func validateDataDir(value string) error {
	if value == "" || !filepath.IsAbs(value) || filepath.Clean(value) != value || !safePathPattern.MatchString(value) {
		return errors.New("K3s 数据目录必须是规范化绝对路径")
	}
	for _, blocked := range []string{"/", "/bin", "/boot", "/dev", "/etc", "/home", "/opt", "/proc", "/root", "/run", "/sys", "/tmp", "/usr", "/var"} {
		if value == blocked {
			return errors.New("K3s 数据目录不能指向系统级目录")
		}
	}
	return nil
}

func validateServerURL(value string, apiPort int) error {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.Port() != fmt.Sprintf("%d", apiPort) {
		return fmt.Errorf("K3s serverUrl 必须使用计划指定的 TCP %d 端口", apiPort)
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return errors.New("K3s serverUrl 不能包含凭据、路径、查询或片段")
	}
	return nil
}

func (plan ClusterPlan) NodeName() string {
	compact := strings.ReplaceAll(strings.ToLower(plan.NodeID), "-", "")
	return "if-" + compact[:12]
}

func (plan ClusterPlan) ServiceName() string {
	if plan.Operation == OperationInitServer {
		return "induforge-k3s-server.service"
	}
	return "induforge-k3s-agent.service"
}

func (plan ClusterPlan) RenderK3sConfig() string {
	lines := []string{
		"data-dir: " + quoteYAML(plan.DataDir),
		"node-name: " + quoteYAML(plan.NodeName()),
		"node-ip: " + quoteYAML(plan.NodeIP),
		"token: " + quoteYAML(plan.Token),
	}
	if plan.Operation == OperationInitServer {
		lines = append(lines, fmt.Sprintf("https-listen-port: %d", plan.APIPort), "secrets-encryption: true", "write-kubeconfig-mode: \"0600\"", "cluster-init: true", "tls-san:", "  - "+quoteYAML(plan.NodeIP))
	} else {
		lines = append(lines, "server: "+quoteYAML(plan.ServerURL))
	}
	return strings.Join(lines, "\n") + "\n"
}

func quoteYAML(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
}

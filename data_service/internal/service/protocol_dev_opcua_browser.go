package service

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
	apperrors "github.com/indu-forge/data_service/internal/errors"
)

const (
	defaultOpcuaBrowseMaxDepth        = 8
	defaultOpcuaBrowseMaxNodes        = 2000
	defaultOpcuaBrowseConnectTimeout  = 5 * time.Second
	defaultOpcuaBrowseRequestTimeout  = 8 * time.Second
	defaultOpcuaBrowseSessionTimeout  = 30 * time.Second
	defaultOpcuaBrowseRootNodeID      = "i=85"
	defaultOpcuaBrowseApplicationName = "InduForge OPC UA Workbench"
)

// ProtocolDevOpcuaBrowser 表示开发态 OPC UA 地址空间浏览适配器。
type ProtocolDevOpcuaBrowser interface {
	Browse(ctx context.Context, session ProtocolDevSession) (*ProtocolDevOpcuaBrowseResult, error)
}

// ProtocolDevOpcuaRealBrowser 连接真实 OPC UA Server 并浏览地址空间。
type ProtocolDevOpcuaRealBrowser struct{}

// NewProtocolDevOpcuaRealBrowser 创建真实 OPC UA 浏览适配器。
func NewProtocolDevOpcuaRealBrowser() *ProtocolDevOpcuaRealBrowser {
	return &ProtocolDevOpcuaRealBrowser{}
}

type opcuaBrowseOptions struct {
	Endpoint        string
	SecurityPolicy  string
	SecurityMode    string
	AuthType        string
	Username        string
	Password        string
	CertificatePEM  string
	PrivateKeyPEM   string
	RootNodeID      string
	MaxDepth        int
	MaxNodes        int
	ConnectTimeout  time.Duration
	RequestTimeout  time.Duration
	SessionTimeout  time.Duration
	ApplicationName string
}

type opcuaBrowseState struct {
	result       ProtocolDevOpcuaBrowseResult
	visited      map[string]bool
	limitHit     bool
	depthHit     bool
	dataTypeWarn bool
}

// Browse 返回真实 OPC UA 地址空间节点。失败时直接返回错误，避免前端误以为看到的是现场地址空间。
func (b *ProtocolDevOpcuaRealBrowser) Browse(ctx context.Context, session ProtocolDevSession) (*ProtocolDevOpcuaBrowseResult, error) {
	if b == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "OPC UA 真实浏览适配器未初始化")
	}
	options, err := parseOpcuaBrowseOptions(session.Config)
	if err != nil {
		return nil, err
	}
	browseCtx, cancel := context.WithTimeout(ctx, options.RequestTimeout)
	defer cancel()
	clientOptions, err := options.clientOptions(browseCtx)
	if err != nil {
		return nil, err
	}
	client, err := opcua.NewClient(options.Endpoint, clientOptions...)
	if err != nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, fmt.Sprintf("OPC UA 客户端初始化失败: %v", err))
	}

	if err := client.Connect(browseCtx); err != nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, fmt.Sprintf("OPC UA 连接失败: %v", err))
	}
	defer client.Close(context.Background())

	rootID, err := ua.ParseNodeID(options.RootNodeID)
	if err != nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, fmt.Sprintf("OPC UA 浏览根节点无效: %v", err))
	}
	state := &opcuaBrowseState{
		result:  ProtocolDevOpcuaBrowseResult{Nodes: []ProtocolDevBrowseNode{}, Diagnostics: []string{}},
		visited: map[string]bool{},
	}
	if err := b.browseNode(browseCtx, client.Node(rootID), nil, 0, options, state); err != nil {
		return nil, err
	}
	if state.limitHit {
		state.result.Diagnostics = append(state.result.Diagnostics, fmt.Sprintf("OPC UA 地址空间节点数超过限制，已截断到 %d 个节点。", options.MaxNodes))
	}
	if state.depthHit {
		state.result.Diagnostics = append(state.result.Diagnostics, fmt.Sprintf("OPC UA 地址空间层级超过限制，已截断到 %d 层。", options.MaxDepth))
	}
	if state.dataTypeWarn {
		state.result.Diagnostics = append(state.result.Diagnostics, "部分变量 DataType 属性读取失败，已保留节点并留空数据类型。")
	}
	return &state.result, nil
}

func (b *ProtocolDevOpcuaRealBrowser) browseNode(ctx context.Context, node *opcua.Node, parentID *string, depth int, options opcuaBrowseOptions, state *opcuaBrowseState) error {
	if len(state.result.Nodes) >= options.MaxNodes {
		state.limitHit = true
		return nil
	}
	if depth > options.MaxDepth {
		state.depthHit = true
		return nil
	}
	if node == nil || node.ID == nil {
		return nil
	}
	nodeID := node.ID.String()
	if state.visited[nodeID] {
		return nil
	}
	state.visited[nodeID] = true

	attrs, err := node.Attributes(ctx, ua.AttributeIDNodeClass, ua.AttributeIDBrowseName, ua.AttributeIDDisplayName, ua.AttributeIDDataType)
	if err != nil {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, fmt.Sprintf("OPC UA 浏览节点属性失败 nodeId=%s: %v", nodeID, err))
	}
	nodeClass, err := opcuaNodeClassFromAttributes(attrs)
	if err != nil {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, fmt.Sprintf("OPC UA 读取节点类型失败 nodeId=%s: %v", nodeID, err))
	}
	if nodeClass != ua.NodeClassObject && nodeClass != ua.NodeClassVariable {
		return nil
	}

	browseName := opcuaBrowseNameFromAttributes(attrs)
	displayName := opcuaDisplayNameFromAttributes(attrs)
	name := firstNonEmpty(displayName, browseName, nodeID)
	dataType := ""
	if nodeClass == ua.NodeClassVariable {
		var dataTypeOK bool
		dataType, dataTypeOK = opcuaDataTypeFromAttributes(attrs)
		if !dataTypeOK {
			state.dataTypeWarn = true
		}
	}

	currentID := "opcua-" + nodeID
	nodeType := "folder"
	if nodeClass == ua.NodeClassVariable {
		nodeType = "variable"
	}
	state.result.Nodes = append(state.result.Nodes, ProtocolDevBrowseNode{
		ID:          currentID,
		ParentID:    parentID,
		Name:        name,
		NodeID:      nodeID,
		NodeType:    nodeType,
		DataType:    dataType,
		Modeled:     false,
		BrowseName:  browseName,
		DisplayName: displayName,
	})
	if len(state.result.Nodes) >= options.MaxNodes {
		state.limitHit = true
		return nil
	}

	// OPC UA 常见层级关系分散在 Organizes/HasComponent/HasProperty 中，逐类浏览可覆盖多数设备建模方式。
	for _, refType := range []uint32{id.Organizes, id.HasComponent, id.HasProperty} {
		children, err := node.ReferencedNodes(ctx, refType, ua.BrowseDirectionForward, ua.NodeClassObject|ua.NodeClassVariable, true)
		if err != nil {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, fmt.Sprintf("OPC UA 浏览子节点失败 nodeId=%s: %v", nodeID, err))
		}
		for _, child := range children {
			if err := b.browseNode(ctx, child, &currentID, depth+1, options, state); err != nil {
				return err
			}
			if state.limitHit {
				return nil
			}
		}
	}
	return nil
}

func parseOpcuaBrowseOptions(config map[string]any) (opcuaBrowseOptions, error) {
	optionsMap := mapFromAny(config["options"])
	sslConfig := mapFromAny(optionsMap["sslConfig"])
	if len(sslConfig) == 0 {
		sslConfig = mapFromAny(config["sslConfig"])
	}
	endpoint := strings.TrimSpace(firstProtocolString(config, "endpoint", "url"))
	if endpoint == "" {
		return opcuaBrowseOptions{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "OPC UA endpoint 不能为空")
	}
	authType := strings.TrimSpace(strings.ToLower(firstProtocolString(config, "authType")))
	if authType == "" {
		authType = "anonymous"
	}
	return opcuaBrowseOptions{
		Endpoint:        endpoint,
		SecurityPolicy:  firstNonEmpty(firstProtocolString(config, "securityPolicy"), "None"),
		SecurityMode:    normalizeOpcuaBrowseSecurityMode(firstProtocolString(config, "securityMode")),
		AuthType:        authType,
		Username:        firstProtocolString(config, "username"),
		Password:        firstProtocolString(config, "password"),
		CertificatePEM:  firstNonEmpty(textFromMap(sslConfig, "cert"), textFromMap(sslConfig, "certificate")),
		PrivateKeyPEM:   firstNonEmpty(textFromMap(sslConfig, "key"), textFromMap(sslConfig, "privateKey")),
		RootNodeID:      firstNonEmpty(textFromMap(optionsMap, "browseRootNodeId"), defaultOpcuaBrowseRootNodeID),
		MaxDepth:        positiveIntFromMap(optionsMap, "browseMaxDepth", defaultOpcuaBrowseMaxDepth),
		MaxNodes:        positiveIntFromMap(optionsMap, "browseMaxNodes", defaultOpcuaBrowseMaxNodes),
		ConnectTimeout:  durationFromMS(optionsMap, "connectTimeoutMs", defaultOpcuaBrowseConnectTimeout),
		RequestTimeout:  durationFromMS(optionsMap, "browseTimeoutMs", defaultOpcuaBrowseRequestTimeout),
		SessionTimeout:  durationFromMS(optionsMap, "sessionTimeoutMs", defaultOpcuaBrowseSessionTimeout),
		ApplicationName: firstNonEmpty(textFromMap(optionsMap, "applicationName"), defaultOpcuaBrowseApplicationName),
	}, nil
}

func (o opcuaBrowseOptions) clientOptions(ctx context.Context) ([]opcua.Option, error) {
	result := []opcua.Option{
		opcua.ApplicationName(o.ApplicationName),
		opcua.AutoReconnect(false),
		opcua.DialTimeout(o.ConnectTimeout),
		opcua.RequestTimeout(o.RequestTimeout),
		opcua.SessionTimeout(o.SessionTimeout),
	}
	authMode := ua.UserTokenTypeAnonymous
	if strings.EqualFold(o.AuthType, "username_password") {
		if strings.TrimSpace(o.Username) == "" {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "OPC UA username_password 认证需要 username")
		}
		authMode = ua.UserTokenTypeUserName
		result = append(result, opcua.AuthUsername(o.Username, o.Password))
	} else {
		result = append(result, opcua.AuthAnonymous())
	}
	if strings.TrimSpace(o.CertificatePEM) != "" {
		cert, err := parseOpcuaCertificate(o.CertificatePEM)
		if err != nil {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, fmt.Sprintf("OPC UA 客户端证书无效: %v", err))
		}
		result = append(result, opcua.Certificate(cert))
	}
	if strings.TrimSpace(o.PrivateKeyPEM) != "" {
		key, err := parseOpcuaPrivateKey(o.PrivateKeyPEM)
		if err != nil {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, fmt.Sprintf("OPC UA 客户端私钥无效: %v", err))
		}
		result = append(result, opcua.PrivateKey(key))
	}
	// 安全连接必须根据服务端 EndpointDescription 注入远端证书和用户 token policy，否则 Sign/SignAndEncrypt 容易在握手阶段失败。
	endpoints, err := opcua.GetEndpoints(ctx, o.Endpoint, opcua.DialTimeout(o.ConnectTimeout), opcua.RequestTimeout(o.RequestTimeout))
	if err != nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, fmt.Sprintf("OPC UA 读取服务端端点失败: %v", err))
	}
	securityMode := ua.MessageSecurityModeFromString(o.SecurityMode)
	endpoint, err := opcua.SelectEndpoint(endpoints, o.SecurityPolicy, securityMode)
	if err != nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, fmt.Sprintf("OPC UA 未找到匹配安全端点 policy=%s mode=%s: %v", o.SecurityPolicy, o.SecurityMode, err))
	}
	result = append(result, opcua.SecurityFromEndpoint(endpoint, authMode))
	return result, nil
}

func opcuaNodeClassFromAttributes(attrs []*ua.DataValue) (ua.NodeClass, error) {
	if len(attrs) == 0 || attrs[0] == nil {
		return ua.NodeClassUnspecified, fmt.Errorf("NodeClass 属性为空")
	}
	if attrs[0].Status != ua.StatusOK {
		return ua.NodeClassUnspecified, attrs[0].Status
	}
	return ua.NodeClass(attrs[0].Value.Int()), nil
}

func opcuaBrowseNameFromAttributes(attrs []*ua.DataValue) string {
	if len(attrs) < 2 || attrs[1] == nil || attrs[1].Status != ua.StatusOK || attrs[1].Value == nil {
		return ""
	}
	if value, ok := attrs[1].Value.Value().(*ua.QualifiedName); ok && value != nil {
		return strings.TrimSpace(value.Name)
	}
	return strings.TrimSpace(attrs[1].Value.String())
}

func opcuaDisplayNameFromAttributes(attrs []*ua.DataValue) string {
	if len(attrs) < 3 || attrs[2] == nil || attrs[2].Status != ua.StatusOK || attrs[2].Value == nil {
		return ""
	}
	if value, ok := attrs[2].Value.Value().(*ua.LocalizedText); ok && value != nil {
		return strings.TrimSpace(value.Text)
	}
	return strings.TrimSpace(attrs[2].Value.String())
}

func opcuaDataTypeFromAttributes(attrs []*ua.DataValue) (string, bool) {
	if len(attrs) < 4 || attrs[3] == nil || attrs[3].Status != ua.StatusOK || attrs[3].Value == nil {
		return "", false
	}
	dataTypeID := attrs[3].Value.NodeID()
	if dataTypeID == nil {
		return "", false
	}
	return opcuaDataTypeName(dataTypeID), true
}

func opcuaDataTypeName(nodeID *ua.NodeID) string {
	if nodeID == nil {
		return ""
	}
	if nodeID.Namespace() == 0 {
		switch nodeID.IntID() {
		case id.Boolean:
			return "Boolean"
		case id.SByte:
			return "SByte"
		case id.Byte:
			return "Byte"
		case id.Int16:
			return "Int16"
		case id.UInt16:
			return "UInt16"
		case id.Int32:
			return "Int32"
		case id.UInt32:
			return "UInt32"
		case id.Int64:
			return "Int64"
		case id.UInt64:
			return "UInt64"
		case id.Float:
			return "Float"
		case id.Double:
			return "Double"
		case id.String:
			return "String"
		case id.DateTime:
			return "DateTime"
		case id.BaseDataType:
			return "BaseDataType"
		}
	}
	return nodeID.String()
}

func normalizeOpcuaBrowseSecurityMode(value string) string {
	normalized := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(value)), "_", "")
	switch normalized {
	case "sign":
		return "Sign"
	case "signandencrypt":
		return "SignAndEncrypt"
	default:
		return "None"
	}
}

func parseOpcuaCertificate(input string) ([]byte, error) {
	text := strings.TrimSpace(input)
	if text == "" {
		return nil, nil
	}
	block, _ := pem.Decode([]byte(text))
	if block == nil {
		return []byte(text), nil
	}
	if block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("PEM 类型必须是 CERTIFICATE")
	}
	return block.Bytes, nil
}

func parseOpcuaPrivateKey(input string) (*rsa.PrivateKey, error) {
	text := strings.TrimSpace(input)
	if text == "" {
		return nil, nil
	}
	der := []byte(text)
	if block, _ := pem.Decode(der); block != nil {
		der = block.Bytes
	}
	if key, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, err
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("仅支持 RSA 私钥")
	}
	return key, nil
}

func textFromMap(input map[string]any, key string) string {
	value, ok := input[key]
	if !ok {
		return ""
	}
	return strings.TrimSpace(toProtocolString(value))
}

func positiveIntFromMap(input map[string]any, key string, fallback int) int {
	value, ok := input[key]
	if !ok {
		return fallback
	}
	switch typed := value.(type) {
	case int:
		if typed > 0 {
			return typed
		}
	case int32:
		if typed > 0 {
			return int(typed)
		}
	case int64:
		if typed > 0 {
			return int(typed)
		}
	case float64:
		if typed > 0 {
			return int(typed)
		}
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil && parsed > 0 {
			return parsed
		}
	}
	return fallback
}

func durationFromMS(input map[string]any, key string, fallback time.Duration) time.Duration {
	value := positiveIntFromMap(input, key, int(fallback/time.Millisecond))
	return time.Duration(value) * time.Millisecond
}

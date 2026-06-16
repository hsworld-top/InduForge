package service

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
	apperrors "github.com/indu-forge/data_service/internal/errors"
)

const (
	defaultOpcuaBrowseMaxNodes        = 2000
	defaultOpcuaBrowseConnectTimeout  = 5 * time.Second
	defaultOpcuaBrowseRequestTimeout  = 8 * time.Second
	defaultOpcuaBrowseSessionTimeout  = 30 * time.Second
	defaultOpcuaBrowseRootNodeID      = "i=85"
	defaultOpcuaBrowseApplicationName = "InduForge OPC UA Workbench"
)

// ProtocolDevOpcuaBrowser 表示开发态 OPC UA 地址空间浏览适配器。
type ProtocolDevOpcuaBrowser interface {
	Open(ctx context.Context, session ProtocolDevSession) error
	Close(sessionID string)
	Browse(ctx context.Context, session ProtocolDevSession, parentNodeID string) (*ProtocolDevOpcuaBrowseResult, error)
	BrowseSubtree(ctx context.Context, session ProtocolDevSession, parentNodeID string) (*ProtocolDevOpcuaBrowseResult, error)
}

type opcuaBrowseOperation func(context.Context, *opcua.Client, opcuaBrowseOptions) (*ProtocolDevOpcuaBrowseResult, error)

// ProtocolDevOpcuaRealBrowser 连接真实 OPC UA Server 并浏览地址空间。
type ProtocolDevOpcuaRealBrowser struct {
	mu      sync.Mutex
	clients map[string]*opcuaBrowseClient
}

// NewProtocolDevOpcuaRealBrowser 创建真实 OPC UA 浏览适配器。
func NewProtocolDevOpcuaRealBrowser() *ProtocolDevOpcuaRealBrowser {
	return &ProtocolDevOpcuaRealBrowser{clients: map[string]*opcuaBrowseClient{}}
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
	dataTypeWarn bool
}

type opcuaBrowseClient struct {
	mu      sync.Mutex
	client  *opcua.Client
	options opcuaBrowseOptions
}

// Open 在工作台连接阶段建立真实 OPC UA 客户端，后续浏览复用该连接，避免每次展开树节点都重新握手。
func (b *ProtocolDevOpcuaRealBrowser) Open(ctx context.Context, session ProtocolDevSession) error {
	if b == nil {
		return apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "OPC UA 真实浏览适配器未初始化")
	}
	options, err := parseOpcuaBrowseOptions(session.Config)
	if err != nil {
		return err
	}
	connectCtx, cancel := context.WithTimeout(ctx, options.ConnectTimeout+options.RequestTimeout)
	defer cancel()
	clientOptions, err := options.clientOptions(connectCtx)
	if err != nil {
		return err
	}
	client, err := opcua.NewClient(options.Endpoint, clientOptions...)
	if err != nil {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, fmt.Sprintf("OPC UA 客户端初始化失败: %v", err))
	}
	if err := client.Connect(connectCtx); err != nil {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, fmt.Sprintf("OPC UA 连接失败: %v", err))
	}
	b.mu.Lock()
	old := b.clients[session.SessionID]
	b.clients[session.SessionID] = &opcuaBrowseClient{client: client, options: options}
	b.mu.Unlock()
	if old != nil && old.client != nil {
		old.client.Close(context.Background())
	}
	return nil
}

// Close 释放工作台会话持有的 OPC UA 客户端。
func (b *ProtocolDevOpcuaRealBrowser) Close(sessionID string) {
	if b == nil {
		return
	}
	b.mu.Lock()
	entry := b.clients[sessionID]
	delete(b.clients, sessionID)
	b.mu.Unlock()
	if entry != nil && entry.client != nil {
		entry.client.Close(context.Background())
	}
}

// Browse 返回指定父节点的一层真实 OPC UA 地址空间子节点。失败时直接返回错误，避免前端误以为看到的是现场地址空间。
func (b *ProtocolDevOpcuaRealBrowser) Browse(ctx context.Context, session ProtocolDevSession, parentNodeID string) (*ProtocolDevOpcuaBrowseResult, error) {
	return b.runWithClientRetry(ctx, session, func(runCtx context.Context, client *opcua.Client, options opcuaBrowseOptions) (*ProtocolDevOpcuaBrowseResult, error) {
		return b.browseWithClient(runCtx, client, options, parentNodeID)
	})
}

// BrowseSubtree 递归收集指定节点子树下的变量。该操作只在用户明确选择“导入子树变量”时触发。
func (b *ProtocolDevOpcuaRealBrowser) BrowseSubtree(ctx context.Context, session ProtocolDevSession, parentNodeID string) (*ProtocolDevOpcuaBrowseResult, error) {
	return b.runWithClientRetry(ctx, session, func(runCtx context.Context, client *opcua.Client, options opcuaBrowseOptions) (*ProtocolDevOpcuaBrowseResult, error) {
		return b.browseSubtreeWithClient(runCtx, client, options, parentNodeID)
	})
}

func (b *ProtocolDevOpcuaRealBrowser) browseWithClient(ctx context.Context, client *opcua.Client, options opcuaBrowseOptions, parentNodeID string) (*ProtocolDevOpcuaBrowseResult, error) {
	rootIDText := strings.TrimSpace(parentNodeID)
	if rootIDText == "" {
		rootIDText = options.RootNodeID
	}
	rootID, err := ua.ParseNodeID(rootIDText)
	if err != nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, fmt.Sprintf("OPC UA 浏览父节点无效: %v", err))
	}
	state := &opcuaBrowseState{
		result:  ProtocolDevOpcuaBrowseResult{Nodes: []ProtocolDevBrowseNode{}, Diagnostics: []string{}},
		visited: map[string]bool{},
	}
	parentID := ""
	if strings.TrimSpace(parentNodeID) != "" {
		parentID = opcuaBrowseNodeID(rootID.String())
	}
	if err := b.browseChildren(ctx, client.Node(rootID), parentID, options, state); err != nil {
		return nil, err
	}
	if state.limitHit {
		state.result.Diagnostics = append(state.result.Diagnostics, fmt.Sprintf("OPC UA 当前节点子节点数超过限制，已截断到 %d 个节点。", options.MaxNodes))
	}
	if state.dataTypeWarn {
		state.result.Diagnostics = append(state.result.Diagnostics, "部分变量 DataType 属性读取失败，已保留节点并留空数据类型。")
	}
	return &state.result, nil
}

func (b *ProtocolDevOpcuaRealBrowser) browseSubtreeWithClient(ctx context.Context, client *opcua.Client, options opcuaBrowseOptions, parentNodeID string) (*ProtocolDevOpcuaBrowseResult, error) {
	rootIDText := strings.TrimSpace(parentNodeID)
	if rootIDText == "" {
		rootIDText = options.RootNodeID
	}
	rootID, err := ua.ParseNodeID(rootIDText)
	if err != nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, fmt.Sprintf("OPC UA 浏览父节点无效: %v", err))
	}
	state := &opcuaBrowseState{
		result:  ProtocolDevOpcuaBrowseResult{Nodes: []ProtocolDevBrowseNode{}, Diagnostics: []string{}},
		visited: map[string]bool{},
	}
	parentID := ""
	if strings.TrimSpace(parentNodeID) != "" {
		parentID = opcuaBrowseNodeID(rootID.String())
	}
	if err := b.collectVariableSubtree(ctx, client, client.Node(rootID), parentID, options, state); err != nil {
		return nil, err
	}
	if state.limitHit {
		state.result.Diagnostics = append(state.result.Diagnostics, fmt.Sprintf("OPC UA 子树变量数超过限制，已截断到 %d 个节点。", options.MaxNodes))
	}
	if state.dataTypeWarn {
		state.result.Diagnostics = append(state.result.Diagnostics, "部分变量 DataType 属性读取失败，已保留节点并留空数据类型。")
	}
	return &state.result, nil
}

func (b *ProtocolDevOpcuaRealBrowser) browseChildren(ctx context.Context, parent *opcua.Node, parentID string, options opcuaBrowseOptions, state *opcuaBrowseState) error {
	if parent == nil || parent.ID == nil {
		return nil
	}
	parentIDPtr := stringPtrIfNotEmpty(parentID)
	// OPC UA 常见层级关系分散在 Organizes/HasComponent/HasProperty 中，逐类浏览可覆盖多数设备建模方式。
	for _, refType := range []uint32{id.Organizes, id.HasComponent, id.HasProperty} {
		children, err := parent.ReferencedNodes(ctx, refType, ua.BrowseDirectionForward, ua.NodeClassObject|ua.NodeClassVariable, true)
		if err != nil {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, fmt.Sprintf("OPC UA 浏览子节点失败 nodeId=%s: %v", parent.ID.String(), err))
		}
		for _, child := range children {
			if len(state.result.Nodes) >= options.MaxNodes {
				state.limitHit = true
				return nil
			}
			if child == nil || child.ID == nil {
				continue
			}
			nodeID := child.ID.String()
			if state.visited[nodeID] {
				continue
			}
			state.visited[nodeID] = true
			node, err := b.projectBrowseNode(ctx, child, parentIDPtr, state)
			if err != nil {
				return err
			}
			if node != nil {
				state.result.Nodes = append(state.result.Nodes, *node)
			}
		}
	}
	return nil
}

func (b *ProtocolDevOpcuaRealBrowser) collectVariableSubtree(ctx context.Context, client *opcua.Client, parent *opcua.Node, parentID string, options opcuaBrowseOptions, state *opcuaBrowseState) error {
	if parent == nil || parent.ID == nil || state.limitHit {
		return nil
	}
	children, err := b.projectChildNodes(ctx, parent, parentID, options, state)
	if err != nil {
		return err
	}
	for _, child := range children {
		if state.limitHit {
			return nil
		}
		if child.NodeType == "variable" {
			state.result.Nodes = append(state.result.Nodes, child)
			continue
		}
		nodeID, err := ua.ParseNodeID(child.NodeID)
		if err != nil {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, fmt.Sprintf("OPC UA 子树节点无效 nodeId=%s: %v", child.NodeID, err))
		}
		if err := b.collectVariableSubtree(ctx, client, client.Node(nodeID), child.ID, options, state); err != nil {
			return err
		}
	}
	return nil
}

func (b *ProtocolDevOpcuaRealBrowser) projectChildNodes(ctx context.Context, parent *opcua.Node, parentID string, options opcuaBrowseOptions, state *opcuaBrowseState) ([]ProtocolDevBrowseNode, error) {
	if parent == nil || parent.ID == nil {
		return []ProtocolDevBrowseNode{}, nil
	}
	parentIDPtr := stringPtrIfNotEmpty(parentID)
	result := []ProtocolDevBrowseNode{}
	for _, refType := range []uint32{id.Organizes, id.HasComponent, id.HasProperty} {
		children, err := parent.ReferencedNodes(ctx, refType, ua.BrowseDirectionForward, ua.NodeClassObject|ua.NodeClassVariable, true)
		if err != nil {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, fmt.Sprintf("OPC UA 浏览子节点失败 nodeId=%s: %v", parent.ID.String(), err))
		}
		for _, child := range children {
			if len(state.result.Nodes)+len(result) >= options.MaxNodes {
				state.limitHit = true
				return result, nil
			}
			if child == nil || child.ID == nil {
				continue
			}
			nodeID := child.ID.String()
			if state.visited[nodeID] {
				continue
			}
			state.visited[nodeID] = true
			node, err := b.projectBrowseNode(ctx, child, parentIDPtr, state)
			if err != nil {
				return nil, err
			}
			if node != nil {
				result = append(result, *node)
			}
		}
	}
	return result, nil
}

func (b *ProtocolDevOpcuaRealBrowser) projectBrowseNode(ctx context.Context, node *opcua.Node, parentID *string, state *opcuaBrowseState) (*ProtocolDevBrowseNode, error) {
	nodeID := node.ID.String()
	attrs, err := node.Attributes(ctx, ua.AttributeIDNodeClass, ua.AttributeIDBrowseName, ua.AttributeIDDisplayName, ua.AttributeIDDataType)
	if err != nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, fmt.Sprintf("OPC UA 浏览节点属性失败 nodeId=%s: %v", nodeID, err))
	}
	nodeClass, err := opcuaNodeClassFromAttributes(attrs)
	if err != nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, fmt.Sprintf("OPC UA 读取节点类型失败 nodeId=%s: %v", nodeID, err))
	}
	if nodeClass != ua.NodeClassObject && nodeClass != ua.NodeClassVariable {
		return nil, nil
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

	nodeType := "folder"
	hasChildren := true
	if nodeClass == ua.NodeClassVariable {
		nodeType = "variable"
		hasChildren = false
	}
	return &ProtocolDevBrowseNode{
		ID:          opcuaBrowseNodeID(nodeID),
		ParentID:    parentID,
		Name:        name,
		NodeID:      nodeID,
		NodeType:    nodeType,
		DataType:    dataType,
		Modeled:     false,
		HasChildren: hasChildren,
		BrowseName:  browseName,
		DisplayName: displayName,
	}, nil
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
		MaxNodes:        positiveIntFromMap(optionsMap, "browseMaxNodes", defaultOpcuaBrowseMaxNodes),
		ConnectTimeout:  durationFromMS(optionsMap, "connectTimeoutMs", defaultOpcuaBrowseConnectTimeout),
		RequestTimeout:  durationFromMS(optionsMap, "browseTimeoutMs", defaultOpcuaBrowseRequestTimeout),
		SessionTimeout:  durationFromMS(optionsMap, "sessionTimeoutMs", defaultOpcuaBrowseSessionTimeout),
		ApplicationName: firstNonEmpty(textFromMap(optionsMap, "applicationName"), defaultOpcuaBrowseApplicationName),
	}, nil
}

func opcuaBrowseNodeID(nodeID string) string {
	return "opcua-" + nodeID
}

func stringPtrIfNotEmpty(value string) *string {
	text := strings.TrimSpace(value)
	if text == "" {
		return nil
	}
	return &text
}

func (b *ProtocolDevOpcuaRealBrowser) requireClient(sessionID string) (*opcuaBrowseClient, error) {
	if b == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "OPC UA 真实浏览适配器未初始化")
	}
	b.mu.Lock()
	entry := b.clients[sessionID]
	b.mu.Unlock()
	if entry == nil || entry.client == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "OPC UA 开发态连接未建立，请重新连接工作台")
	}
	return entry, nil
}

func (b *ProtocolDevOpcuaRealBrowser) runWithClientRetry(ctx context.Context, session ProtocolDevSession, operation opcuaBrowseOperation) (*ProtocolDevOpcuaBrowseResult, error) {
	result, err := b.runWithClient(ctx, session, operation)
	if err == nil || !isRecoverableOpcuaSessionError(err) {
		return result, err
	}
	b.Close(session.SessionID)
	if openErr := b.Open(ctx, session); openErr != nil {
		return nil, openErr
	}
	return b.runWithClient(ctx, session, operation)
}

func (b *ProtocolDevOpcuaRealBrowser) runWithClient(ctx context.Context, session ProtocolDevSession, operation opcuaBrowseOperation) (*ProtocolDevOpcuaBrowseResult, error) {
	entry, err := b.requireClient(session.SessionID)
	if err != nil {
		if openErr := b.Open(ctx, session); openErr != nil {
			return nil, openErr
		}
		entry, err = b.requireClient(session.SessionID)
		if err != nil {
			return nil, err
		}
	}
	entry.mu.Lock()
	defer entry.mu.Unlock()
	runCtx, cancel := context.WithTimeout(ctx, entry.options.RequestTimeout)
	defer cancel()
	return operation(runCtx, entry.client, entry.options)
}

func isRecoverableOpcuaSessionError(err error) bool {
	return errors.Is(err, io.EOF) ||
		errors.Is(err, ua.StatusBadSessionNotActivated) ||
		errors.Is(err, ua.StatusBadSessionIDInvalid) ||
		errors.Is(err, ua.StatusBadSecureChannelIDInvalid)
}

func (o opcuaBrowseOptions) clientOptions(ctx context.Context) ([]opcua.Option, error) {
	result := []opcua.Option{
		opcua.ApplicationName(o.ApplicationName),
		opcua.AutoReconnect(true),
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
	if o.usesClientCertificate() {
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

func (o opcuaBrowseOptions) usesClientCertificate() bool {
	return !strings.EqualFold(strings.TrimSpace(o.SecurityPolicy), "None") || !strings.EqualFold(strings.TrimSpace(o.SecurityMode), "None")
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
		case id.Number:
			return "Number"
		case id.Integer:
			return "Integer"
		case id.UInteger:
			return "UInteger"
		case id.Duration:
			return "Duration"
		case id.LocaleID:
			return "LocaleID"
		case id.LocalizedText:
			return "LocalizedText"
		case id.QualifiedName:
			return "QualifiedName"
		case id.NodeID:
			return "NodeID"
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

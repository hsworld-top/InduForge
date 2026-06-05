package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

const (
	defaultHTTPTimeoutMS = 5000
	maxHTTPTimeoutMS     = 30000
	maxHTTPResponseBytes = 2 * 1024 * 1024
)

var allowedHTTPRequestMethods = map[string]struct{}{
	http.MethodGet:    {},
	http.MethodPost:   {},
	http.MethodPut:    {},
	http.MethodPatch:  {},
	http.MethodDelete: {},
}

var allowedHTTPBodyTypes = map[string]struct{}{
	"none":                  {},
	"json":                  {},
	"raw":                   {},
	"form-data":             {},
	"x-www-form-urlencoded": {},
}

// HTTPRequestGroup 表示 HTTP 工作台左侧集合树分组。
type HTTPRequestGroup struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"projectId"`
	ConnectionID string    `json:"connectionId"`
	ParentID     *string   `json:"parentId"`
	Name         string    `json:"name"`
	SortOrder    int       `json:"sortOrder"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// HTTPRequest 表示一个 HTTP 接口请求配置。
type HTTPRequest struct {
	ID            string         `json:"id"`
	ProjectID     string         `json:"projectId"`
	ConnectionID  string         `json:"connectionId"`
	GroupID       *string        `json:"groupId"`
	Name          string         `json:"name"`
	Method        string         `json:"method"`
	URL           string         `json:"url"`
	Params        []any          `json:"params"`
	Headers       []any          `json:"headers"`
	Auth          map[string]any `json:"auth"`
	BodyType      string         `json:"bodyType"`
	Body          map[string]any `json:"body"`
	Settings      map[string]any `json:"settings"`
	Enabled       bool           `json:"enabled"`
	SortOrder     int            `json:"sortOrder"`
	SourceType    string         `json:"sourceType"`
	DataPointID   string         `json:"dataPointId"`
	DataPointPath string         `json:"dataPointPath"`
	Quality       string         `json:"quality"`
	LastSentAt    *time.Time     `json:"lastSentAt"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
}

// CreateHTTPRequestGroupInput 描述创建 HTTP 分组的输入。
type CreateHTTPRequestGroupInput struct {
	ParentID  *string `json:"parentId"`
	Name      string  `json:"name"`
	SortOrder int     `json:"sortOrder"`
}

// UpdateHTTPRequestGroupInput 描述更新 HTTP 分组的输入。
type UpdateHTTPRequestGroupInput struct {
	ParentID    *string `json:"parentId"`
	HasParentID bool    `json:"-"`
	Name        string  `json:"name"`
	SortOrder   int     `json:"sortOrder"`
}

// CreateHTTPRequestInput 描述创建 HTTP 请求的输入。
type CreateHTTPRequestInput struct {
	GroupID   *string        `json:"groupId"`
	Name      string         `json:"name"`
	Method    string         `json:"method"`
	URL       string         `json:"url"`
	Params    []any          `json:"params"`
	Headers   []any          `json:"headers"`
	Auth      map[string]any `json:"auth"`
	BodyType  string         `json:"bodyType"`
	Body      map[string]any `json:"body"`
	Settings  map[string]any `json:"settings"`
	Enabled   bool           `json:"enabled"`
	SortOrder int            `json:"sortOrder"`
}

// UpdateHTTPRequestInput 描述更新 HTTP 请求的输入。
type UpdateHTTPRequestInput struct {
	GroupID    *string        `json:"groupId"`
	HasGroupID bool           `json:"-"`
	Name       string         `json:"name"`
	Method     string         `json:"method"`
	URL        string         `json:"url"`
	Params     []any          `json:"params"`
	Headers    []any          `json:"headers"`
	Auth       map[string]any `json:"auth"`
	BodyType   string         `json:"bodyType"`
	Body       map[string]any `json:"body"`
	Settings   map[string]any `json:"settings"`
	Enabled    bool           `json:"enabled"`
	SortOrder  int            `json:"sortOrder"`
}

// HTTPRequestListResult 表示 HTTP 请求分页结果。
type HTTPRequestListResult struct {
	List       []HTTPRequest              `json:"list"`
	Pagination ProtocolModelingPagination `json:"pagination"`
}

// HTTPSendResponse 表示一次接口发送结果。
type HTTPSendResponse struct {
	Status     int               `json:"status"`
	StatusText string            `json:"statusText"`
	Headers    map[string]string `json:"headers"`
	Body       any               `json:"body"`
	RawBody    string            `json:"rawBody"`
	DurationMS int64             `json:"durationMs"`
	SizeBytes  int               `json:"sizeBytes"`
	ReceivedAt time.Time         `json:"receivedAt"`
}

// HTTPWorkbenchService 承载 HTTP 工作台开发态配置与发送逻辑。
type HTTPWorkbenchService struct {
	repository  *repository.HTTPWorkbenchRepository
	connections *repository.ConnectionRepository
	client      *http.Client
}

// NewHTTPWorkbenchService 创建 HTTP 工作台服务。
func NewHTTPWorkbenchService(repo *repository.HTTPWorkbenchRepository, connections *repository.ConnectionRepository) *HTTPWorkbenchService {
	return &HTTPWorkbenchService{repository: repo, connections: connections, client: http.DefaultClient}
}

// ListGroups 返回 HTTP 请求分组。
func (s *HTTPWorkbenchService) ListGroups(ctx context.Context, projectID, connectionID string) ([]HTTPRequestGroup, error) {
	if err := validateProjectAndConnection(projectID, connectionID); err != nil {
		return nil, err
	}
	records, err := s.repository.ListGroups(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	result := make([]HTTPRequestGroup, 0, len(records))
	for _, record := range records {
		result = append(result, toHTTPRequestGroup(record))
	}
	return result, nil
}

// CreateGroup 创建 HTTP 请求分组。
func (s *HTTPWorkbenchService) CreateGroup(ctx context.Context, projectID, connectionID, userID string, input CreateHTTPRequestGroupInput) (*HTTPRequestGroup, error) {
	if err := validateProjectConnectionAndUser(projectID, connectionID, userID); err != nil {
		return nil, err
	}
	name, err := normalizeHTTPRequiredText(input.Name, 100, "HTTP 请求分组名称不能为空")
	if err != nil {
		return nil, err
	}
	parentID, err := s.normalizeGroupParent(ctx, projectID, connectionID, input.ParentID, "")
	if err != nil {
		return nil, err
	}
	record, err := s.repository.CreateGroup(ctx, repository.CreateHTTPRequestGroupParams{
		ProjectID:    projectID,
		ConnectionID: connectionID,
		ParentID:     parentID,
		Name:         name,
		SortOrder:    input.SortOrder,
		UserID:       userID,
	})
	if err != nil {
		return nil, err
	}
	result := toHTTPRequestGroup(*record)
	return &result, nil
}

// UpdateGroup 更新 HTTP 请求分组。
func (s *HTTPWorkbenchService) UpdateGroup(ctx context.Context, projectID, groupID, userID string, input UpdateHTTPRequestGroupInput) (*HTTPRequestGroup, error) {
	if err := validateProjectAndUser(projectID, userID); err != nil {
		return nil, err
	}
	if err := validateUUIDText(groupID, "groupId 格式无效"); err != nil {
		return nil, err
	}
	current, err := s.repository.GetGroup(ctx, projectID, groupID)
	if err != nil {
		return nil, err
	}
	name, err := normalizeHTTPRequiredText(fallbackTrimmed(input.Name, current.Name), 100, "HTTP 请求分组名称不能为空")
	if err != nil {
		return nil, err
	}
	parentID := cloneOptionalString(current.ParentID)
	if input.HasParentID {
		parentID, err = s.normalizeGroupParent(ctx, projectID, current.ConnectionID, input.ParentID, current.ID)
		if err != nil {
			return nil, err
		}
	}
	sortOrder := input.SortOrder
	if sortOrder == 0 && current.SortOrder != 0 {
		sortOrder = current.SortOrder
	}
	record, err := s.repository.UpdateGroup(ctx, repository.UpdateHTTPRequestGroupParams{
		ProjectID: projectID,
		GroupID:   groupID,
		ParentID:  parentID,
		Name:      name,
		SortOrder: sortOrder,
		UserID:    userID,
	})
	if err != nil {
		return nil, err
	}
	result := toHTTPRequestGroup(*record)
	return &result, nil
}

// DeleteGroup 删除 HTTP 请求分组。
func (s *HTTPWorkbenchService) DeleteGroup(ctx context.Context, projectID, groupID, userID string) error {
	if err := validateProjectAndUser(projectID, userID); err != nil {
		return err
	}
	if err := validateUUIDText(groupID, "groupId 格式无效"); err != nil {
		return err
	}
	return s.repository.DeleteGroup(ctx, projectID, groupID, userID)
}

// ListRequestsPage 查询 HTTP 接口请求分页。
func (s *HTTPWorkbenchService) ListRequestsPage(ctx context.Context, projectID, connectionID string, groupID *string, search string, page, pageSize int) (*HTTPRequestListResult, error) {
	if err := validateProjectAndConnection(projectID, connectionID); err != nil {
		return nil, err
	}
	page, pageSize = normalizePageAndSize(page, pageSize, 20, 100)
	records, total, err := s.repository.ListRequestsPage(ctx, projectID, connectionID, groupID, search, page, pageSize)
	if err != nil {
		return nil, err
	}
	items := make([]HTTPRequest, 0, len(records))
	for _, record := range records {
		items = append(items, toHTTPRequest(record))
	}
	return &HTTPRequestListResult{
		List:       items,
		Pagination: newProtocolModelingPagination(page, pageSize, total),
	}, nil
}

// GetRequest 读取单个 HTTP 请求。
func (s *HTTPWorkbenchService) GetRequest(ctx context.Context, projectID, requestID string) (*HTTPRequest, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateUUIDText(requestID, "requestId 格式无效"); err != nil {
		return nil, err
	}
	record, err := s.repository.GetRequest(ctx, projectID, requestID)
	if err != nil {
		return nil, err
	}
	result := toHTTPRequest(*record)
	return &result, nil
}

// CreateRequest 创建 HTTP 请求并同步数据点。
func (s *HTTPWorkbenchService) CreateRequest(ctx context.Context, projectID, connectionID, userID string, input CreateHTTPRequestInput) (*HTTPRequest, error) {
	if err := validateProjectConnectionAndUser(projectID, connectionID, userID); err != nil {
		return nil, err
	}
	connection, err := s.loadHTTPConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	params, err := s.normalizeCreateRequest(ctx, projectID, userID, *connection, input)
	if err != nil {
		return nil, err
	}
	record, err := s.repository.CreateRequestWithDataPoint(ctx, params)
	if err != nil {
		return nil, err
	}
	result := toHTTPRequest(*record)
	return &result, nil
}

// UpdateRequest 更新 HTTP 请求并同步数据点。
func (s *HTTPWorkbenchService) UpdateRequest(ctx context.Context, projectID, requestID, userID string, input UpdateHTTPRequestInput) (*HTTPRequest, error) {
	if err := validateProjectAndUser(projectID, userID); err != nil {
		return nil, err
	}
	if err := validateUUIDText(requestID, "requestId 格式无效"); err != nil {
		return nil, err
	}
	current, err := s.repository.GetRequest(ctx, projectID, requestID)
	if err != nil {
		return nil, err
	}
	connection, err := s.loadHTTPConnection(ctx, projectID, current.ConnectionID)
	if err != nil {
		return nil, err
	}
	params, err := s.normalizeUpdateRequest(ctx, projectID, userID, *connection, *current, input)
	if err != nil {
		return nil, err
	}
	record, err := s.repository.UpdateRequestWithDataPoint(ctx, params)
	if err != nil {
		return nil, err
	}
	result := toHTTPRequest(*record)
	return &result, nil
}

// DeleteRequest 删除 HTTP 请求并标记数据点失效。
func (s *HTTPWorkbenchService) DeleteRequest(ctx context.Context, projectID, requestID string) error {
	if err := validateProjectID(projectID); err != nil {
		return err
	}
	if err := validateUUIDText(requestID, "requestId 格式无效"); err != nil {
		return err
	}
	return s.repository.DeleteRequestWithDataPoint(ctx, projectID, requestID)
}

// SendRequest 执行一次开发态 HTTP 请求并写回最后响应。
func (s *HTTPWorkbenchService) SendRequest(ctx context.Context, projectID, requestID, userID string) (*HTTPSendResponse, error) {
	if err := validateProjectAndUser(projectID, userID); err != nil {
		return nil, err
	}
	if err := validateUUIDText(requestID, "requestId 格式无效"); err != nil {
		return nil, err
	}
	record, err := s.repository.GetRequest(ctx, projectID, requestID)
	if err != nil {
		return nil, err
	}
	if !record.Enabled {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "HTTP 请求已停用")
	}
	connection, err := s.loadHTTPConnection(ctx, projectID, record.ConnectionID)
	if err != nil {
		return nil, err
	}

	startedAt := time.Now()
	response, sendErr := s.executeHTTPRequest(ctx, *connection, *record)
	quality := "good"
	var defaultValue *string
	if sendErr != nil {
		quality = "bad"
		response = &HTTPSendResponse{
			Status:     0,
			StatusText: "请求失败",
			Headers:    map[string]string{},
			Body: map[string]any{
				"error": sendErr.Error(),
			},
			RawBody:    sendErr.Error(),
			DurationMS: time.Since(startedAt).Milliseconds(),
			SizeBytes:  len([]byte(sendErr.Error())),
			ReceivedAt: time.Now().UTC(),
		}
	} else {
		value := stringifyJSONValue(response)
		defaultValue = &value
	}
	if saveErr := s.repository.SaveRequestSendSnapshot(ctx, projectID, repository.HTTPRequestSendSnapshot{
		RequestID:       record.ID,
		Quality:         quality,
		LastSentAt:      time.Now().UTC(),
		DefaultValue:    defaultValue,
		DataPointConfig: httpRequestSourceConfig(*connection, *record),
		UserID:          userID,
	}); saveErr != nil {
		return nil, saveErr
	}
	return response, nil
}

func (s *HTTPWorkbenchService) executeHTTPRequest(ctx context.Context, connection repository.ConnectionRecord, record repository.HTTPRequestRecord) (*HTTPSendResponse, error) {
	timeoutMS := httpTimeoutMS(record.Settings, connection.Config)
	requestCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMS)*time.Millisecond)
	defer cancel()

	targetURL, err := resolveHTTPRequestURL(connection.Config, record)
	if err != nil {
		return nil, err
	}
	body, contentType, err := buildHTTPRequestBody(record)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(requestCtx, record.Method, targetURL, body)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "创建 HTTP 请求失败", err)
	}
	for key, value := range enabledKeyValuePairs(record.Headers) {
		req.Header.Set(key, value)
	}
	if contentType != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", contentType)
	}
	if err := applyHTTPRequestAuth(req, record.Auth); err != nil {
		return nil, err
	}

	startedAt := time.Now()
	client := s.httpClient(record.Settings)
	resp, err := client.Do(req)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "HTTP 请求发送失败", err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(resp.Body, maxHTTPResponseBytes+1))
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "读取 HTTP 响应失败", err)
	}
	if len(payload) > maxHTTPResponseBytes {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "HTTP 响应超过 2MB，已停止保存")
	}
	bodyValue, rawBody := decodeHTTPResponseBody(payload, resp.Header.Get("Content-Type"))
	return &HTTPSendResponse{
		Status:     resp.StatusCode,
		StatusText: resp.Status,
		Headers:    sanitizePreviewHeaders(resp.Header),
		Body:       bodyValue,
		RawBody:    rawBody,
		DurationMS: time.Since(startedAt).Milliseconds(),
		SizeBytes:  len(payload),
		ReceivedAt: time.Now().UTC(),
	}, nil
}

func (s *HTTPWorkbenchService) httpClient(settings map[string]any) *http.Client {
	followRedirects := true
	if value, ok := settings["followRedirects"]; ok {
		followRedirects = boolFromAny(value)
	}
	tlsVerify := true
	if value, ok := settings["tlsVerify"]; ok {
		tlsVerify = boolFromAny(value)
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if !tlsVerify {
		// 开发态工作台允许关闭 TLS 校验，便于调试自签名测试环境。
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	}
	client := &http.Client{Transport: transport}
	if !followRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
	if s.client != nil && followRedirects && tlsVerify {
		return s.client
	}
	return client
}

func (s *HTTPWorkbenchService) normalizeCreateRequest(ctx context.Context, projectID, userID string, connection repository.ConnectionRecord, input CreateHTTPRequestInput) (repository.CreateHTTPRequestParams, error) {
	name, method, requestURL, bodyType, err := normalizeHTTPRequestCore(input.Name, input.Method, input.URL, input.BodyType)
	if err != nil {
		return repository.CreateHTTPRequestParams{}, err
	}
	groupID, err := s.normalizeRequestGroup(ctx, projectID, connection.ID, input.GroupID)
	if err != nil {
		return repository.CreateHTTPRequestParams{}, err
	}
	record := repository.HTTPRequestRecord{
		ProjectID:    projectID,
		ConnectionID: connection.ID,
		GroupID:      groupID,
		Name:         name,
		Method:       method,
		URL:          requestURL,
	}
	return repository.CreateHTTPRequestParams{
		ProjectID:       projectID,
		ConnectionID:    connection.ID,
		GroupID:         groupID,
		Name:            name,
		Method:          method,
		URL:             requestURL,
		Params:          normalizeKeyValueRows(input.Params),
		Headers:         normalizeKeyValueRows(input.Headers),
		Auth:            normalizeHTTPAuth(input.Auth),
		BodyType:        bodyType,
		Body:            normalizeHTTPBody(input.Body),
		Settings:        normalizeHTTPSettings(input.Settings),
		Enabled:         true,
		SortOrder:       input.SortOrder,
		DataPointPath:   buildHTTPDataPointPath(connection.Name, name),
		DataPointConfig: httpRequestSourceConfig(connection, record),
		UserID:          userID,
	}, nil
}

func (s *HTTPWorkbenchService) normalizeUpdateRequest(ctx context.Context, projectID, userID string, connection repository.ConnectionRecord, current repository.HTTPRequestRecord, input UpdateHTTPRequestInput) (repository.UpdateHTTPRequestParams, error) {
	name, method, requestURL, bodyType, err := normalizeHTTPRequestCore(fallbackTrimmed(input.Name, current.Name), fallbackTrimmed(input.Method, current.Method), fallbackTrimmed(input.URL, current.URL), fallbackTrimmed(input.BodyType, current.BodyType))
	if err != nil {
		return repository.UpdateHTTPRequestParams{}, err
	}
	groupID := current.GroupID
	if input.HasGroupID {
		groupID, err = s.normalizeRequestGroup(ctx, projectID, connection.ID, input.GroupID)
		if err != nil {
			return repository.UpdateHTTPRequestParams{}, err
		}
	}
	record := current
	record.Name = name
	record.Method = method
	record.URL = requestURL
	return repository.UpdateHTTPRequestParams{
		ID:              current.ID,
		ProjectID:       projectID,
		GroupID:         groupID,
		Name:            name,
		Method:          method,
		URL:             requestURL,
		Params:          normalizeKeyValueRowsOrDefault(input.Params, current.Params),
		Headers:         normalizeKeyValueRowsOrDefault(input.Headers, current.Headers),
		Auth:            normalizeMapOrDefault(input.Auth, current.Auth, normalizeHTTPAuth),
		BodyType:        bodyType,
		Body:            normalizeMapOrDefault(input.Body, current.Body, normalizeHTTPBody),
		Settings:        normalizeMapOrDefault(input.Settings, current.Settings, normalizeHTTPSettings),
		Enabled:         input.Enabled,
		SortOrder:       input.SortOrder,
		DataPointPath:   buildHTTPDataPointPath(connection.Name, name),
		DataPointConfig: httpRequestSourceConfig(connection, record),
		DefaultValue:    nil,
		UserID:          userID,
	}, nil
}

func (s *HTTPWorkbenchService) normalizeGroupParent(ctx context.Context, projectID, connectionID string, parentID *string, currentID string) (*string, error) {
	if parentID == nil || strings.TrimSpace(*parentID) == "" {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*parentID)
	if err := validateUUIDText(trimmed, "parentId 格式无效"); err != nil {
		return nil, err
	}
	parent, err := s.repository.GetGroup(ctx, projectID, trimmed)
	if err != nil {
		return nil, err
	}
	if parent.ConnectionID != connectionID {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "上级分组不属于当前 HTTP 接入源")
	}
	if currentID != "" && trimmed == currentID {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "上级分组不能指向自身")
	}
	return &trimmed, nil
}

func (s *HTTPWorkbenchService) normalizeRequestGroup(ctx context.Context, projectID, connectionID string, groupID *string) (*string, error) {
	if groupID == nil || strings.TrimSpace(*groupID) == "" {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*groupID)
	if err := validateUUIDText(trimmed, "groupId 格式无效"); err != nil {
		return nil, err
	}
	group, err := s.repository.GetGroup(ctx, projectID, trimmed)
	if err != nil {
		return nil, err
	}
	if group.ConnectionID != connectionID {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "请求分组不属于当前 HTTP 接入源")
	}
	return &trimmed, nil
}

func (s *HTTPWorkbenchService) loadHTTPConnection(ctx context.Context, projectID, connectionID string) (*repository.ConnectionRecord, error) {
	if s.connections == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "HTTP 工作台连接仓储未初始化")
	}
	connection, err := s.connections.GetByProjectAndID(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(connection.Type) != "http" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "接入源类型不是 HTTP")
	}
	return connection, nil
}

func resolveHTTPRequestURL(config map[string]any, record repository.HTTPRequestRecord) (string, error) {
	raw := strings.TrimSpace(record.URL)
	parsed, err := url.Parse(raw)
	if err == nil && parsed.IsAbs() {
		return applyQueryParams(parsed, record.Params).String(), nil
	}
	return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "HTTP 请求 URL 需填写完整地址，例如 https://api.example.com/data")
}

func applyQueryParams(target *url.URL, rows []any) *url.URL {
	query := target.Query()
	for key, value := range enabledKeyValuePairs(rows) {
		query.Set(key, value)
	}
	target.RawQuery = query.Encode()
	return target
}

func buildHTTPRequestBody(record repository.HTTPRequestRecord) (io.Reader, string, error) {
	switch record.BodyType {
	case "none":
		return nil, "", nil
	case "json":
		payload := record.Body["json"]
		if payload == nil {
			payload = record.Body
		}
		bytesPayload, err := json.Marshal(payload)
		if err != nil {
			return nil, "", apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "HTTP JSON Body 格式无效", err)
		}
		return bytes.NewReader(bytesPayload), "application/json", nil
	case "raw":
		return strings.NewReader(toString(record.Body["raw"])), "text/plain; charset=utf-8", nil
	case "x-www-form-urlencoded":
		form := url.Values{}
		for key, value := range enabledKeyValuePairs(anySliceFromMap(record.Body, "form")) {
			form.Set(key, value)
		}
		return strings.NewReader(form.Encode()), "application/x-www-form-urlencoded", nil
	case "form-data":
		buffer := &bytes.Buffer{}
		writer := multipart.NewWriter(buffer)
		for key, value := range enabledKeyValuePairs(anySliceFromMap(record.Body, "form")) {
			_ = writer.WriteField(key, value)
		}
		if err := writer.Close(); err != nil {
			return nil, "", apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "HTTP form-data Body 格式无效", err)
		}
		return buffer, writer.FormDataContentType(), nil
	default:
		return nil, "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "HTTP Body 类型不受支持")
	}
}

func applyHTTPRequestAuth(req *http.Request, auth map[string]any) error {
	authType := strings.ToLower(strings.TrimSpace(toString(auth["type"])))
	switch authType {
	case "", "none":
		return nil
	case "bearer":
		token := strings.TrimSpace(toString(auth["token"]))
		if token == "" {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Bearer Token 不能为空")
		}
		req.Header.Set("Authorization", "Bearer "+token)
	case "basic":
		username := toString(auth["username"])
		password := toString(auth["password"])
		credential := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
		req.Header.Set("Authorization", "Basic "+credential)
	default:
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "HTTP Auth 类型不受支持")
	}
	return nil
}

func decodeHTTPResponseBody(payload []byte, contentType string) (any, string) {
	raw := string(payload)
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", raw
	}
	var decoded any
	if strings.Contains(strings.ToLower(contentType), "json") || strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		if err := json.Unmarshal(payload, &decoded); err == nil {
			return decoded, raw
		}
	}
	return raw, raw
}

func enabledKeyValuePairs(rows []any) map[string]string {
	result := map[string]string{}
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		if enabled, exists := item["enabled"]; exists && !boolFromAny(enabled) {
			continue
		}
		key := strings.TrimSpace(toString(item["key"]))
		if key == "" {
			continue
		}
		result[key] = toString(item["value"])
	}
	return result
}

func anySliceFromMap(input map[string]any, key string) []any {
	raw, ok := input[key]
	if !ok {
		return []any{}
	}
	if typed, ok := raw.([]any); ok {
		return typed
	}
	return []any{}
}

func httpTimeoutMS(settings, config map[string]any) int {
	timeout := intFromAny(settings["timeoutMs"], 0)
	if timeout <= 0 {
		timeout = intFromAny(config["timeoutMs"], defaultHTTPTimeoutMS)
	}
	if timeout < 1000 {
		return 1000
	}
	if timeout > maxHTTPTimeoutMS {
		return maxHTTPTimeoutMS
	}
	return timeout
}

func normalizeHTTPRequestCore(name, method, requestURL, bodyType string) (string, string, string, string, error) {
	normalizedName, err := normalizeHTTPRequiredText(name, 100, "HTTP 请求名称不能为空")
	if err != nil {
		return "", "", "", "", err
	}
	normalizedMethod := strings.ToUpper(strings.TrimSpace(method))
	if normalizedMethod == "" {
		normalizedMethod = http.MethodGet
	}
	if _, ok := allowedHTTPRequestMethods[normalizedMethod]; !ok {
		return "", "", "", "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "HTTP Method 不受支持")
	}
	normalizedURL, err := normalizeHTTPRequiredText(requestURL, 2000, "HTTP 请求 URL 不能为空")
	if err != nil {
		return "", "", "", "", err
	}
	normalizedBodyType := strings.ToLower(strings.TrimSpace(bodyType))
	if normalizedBodyType == "" {
		normalizedBodyType = "none"
	}
	if _, ok := allowedHTTPBodyTypes[normalizedBodyType]; !ok {
		return "", "", "", "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "HTTP Body 类型不受支持")
	}
	return normalizedName, normalizedMethod, normalizedURL, normalizedBodyType, nil
}

func normalizeHTTPRequiredText(value string, maxLen int, message string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, message)
	}
	if len([]rune(trimmed)) > maxLen {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, fmt.Sprintf("字段长度不能超过 %d 个字符", maxLen))
	}
	return trimmed, nil
}

func normalizeKeyValueRows(rows []any) []any {
	if rows == nil {
		return []any{}
	}
	result := make([]any, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		if !ok {
			continue
		}
		result = append(result, map[string]any{
			"enabled":     boolFromAnyWithDefault(item["enabled"], true),
			"key":         strings.TrimSpace(toString(item["key"])),
			"value":       toString(item["value"]),
			"description": toString(item["description"]),
		})
	}
	return result
}

func normalizeKeyValueRowsOrDefault(rows, fallback []any) []any {
	if rows == nil {
		return fallback
	}
	return normalizeKeyValueRows(rows)
}

func normalizeHTTPAuth(auth map[string]any) map[string]any {
	if auth == nil {
		return map[string]any{"type": "none"}
	}
	result := cloneMap(auth)
	authType := strings.ToLower(strings.TrimSpace(toString(result["type"])))
	if authType == "" {
		authType = "none"
	}
	result["type"] = authType
	return result
}

func normalizeHTTPBody(body map[string]any) map[string]any {
	return cloneMap(body)
}

func normalizeHTTPSettings(settings map[string]any) map[string]any {
	result := cloneMap(settings)
	if _, ok := result["timeoutMs"]; !ok {
		result["timeoutMs"] = defaultHTTPTimeoutMS
	}
	if _, ok := result["followRedirects"]; !ok {
		result["followRedirects"] = true
	}
	if _, ok := result["tlsVerify"]; !ok {
		result["tlsVerify"] = true
	}
	return result
}

func normalizeMapOrDefault(input, fallback map[string]any, normalize func(map[string]any) map[string]any) map[string]any {
	if input == nil {
		return fallback
	}
	return normalize(input)
}

func httpRequestSourceConfig(connection repository.ConnectionRecord, record repository.HTTPRequestRecord) map[string]any {
	return map[string]any{
		"connectionId": connection.ID,
		"requestId":    record.ID,
		"method":       record.Method,
		"url":          record.URL,
	}
}

func buildHTTPDataPointPath(connectionName, requestName string) string {
	return "http." + normalizeDatapointSegment(connectionName) + "." + normalizeDatapointSegment(requestName)
}


func stringifyJSONValue(value any) string {
	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(payload)
}

// defaultValueFromSnapshot 把"上次抓到的消息"序列化为数据点默认值；
// HTTP/WS 工作台都可用作"未抓到时的初始值"推导。输入 nil 返回 nil。
func defaultValueFromSnapshot(value any) *string {
	if value == nil {
		return nil
	}
	encoded := stringifyJSONValue(value)
	return &encoded
}

func boolFromAnyWithDefault(value any, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return boolFromAny(value)
}

func toHTTPRequestGroup(record repository.HTTPRequestGroupRecord) HTTPRequestGroup {
	return HTTPRequestGroup{
		ID:           record.ID,
		ProjectID:    record.ProjectID,
		ConnectionID: record.ConnectionID,
		ParentID:     cloneOptionalString(record.ParentID),
		Name:         record.Name,
		SortOrder:    record.SortOrder,
		CreatedAt:    record.CreatedAt,
		UpdatedAt:    record.UpdatedAt,
	}
}

func toHTTPRequest(record repository.HTTPRequestRecord) HTTPRequest {
	return HTTPRequest{
		ID:            record.ID,
		ProjectID:     record.ProjectID,
		ConnectionID:  record.ConnectionID,
		GroupID:       cloneOptionalString(record.GroupID),
		Name:          record.Name,
		Method:        record.Method,
		URL:           record.URL,
		Params:        cloneJSONArray(record.Params),
		Headers:       cloneJSONArray(record.Headers),
		Auth:          cloneMap(record.Auth),
		BodyType:      record.BodyType,
		Body:          cloneMap(record.Body),
		Settings:      cloneMap(record.Settings),
		Enabled:       record.Enabled,
		SortOrder:     record.SortOrder,
		SourceType:    "http.request",
		DataPointID:   httpStringValue(record.DataPointID),
		DataPointPath: httpStringValue(record.DataPointPath),
		Quality:       fallbackTrimmed(record.Quality, "unknown"),
		LastSentAt:    record.LastSentAt,
		CreatedAt:     record.CreatedAt,
		UpdatedAt:     record.UpdatedAt,
	}
}

func httpStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

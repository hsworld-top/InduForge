package hostd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"
)

type Client struct {
	httpClient *http.Client
}

func NewUnixClient(socketPath string) (*Client, error) {
	if socketPath == "" {
		return nil, fmt.Errorf("hostd socket 路径不能为空")
	}
	dialer := &net.Dialer{Timeout: 3 * time.Second}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return dialer.DialContext(ctx, "unix", socketPath)
		},
		DisableKeepAlives: true,
	}
	// 集群卸载需要等待工作负载和命名空间退出，不能沿用普通探活请求的短超时。
	return &Client{httpClient: &http.Client{Transport: transport, Timeout: 3 * time.Minute}}, nil
}

func (client *Client) Status(ctx context.Context) (ClusterState, error) {
	return client.request(ctx, http.MethodGet, "/v1/cluster/status", nil)
}

func (client *Client) Apply(ctx context.Context, plan ClusterPlan) (ClusterState, error) {
	return client.request(ctx, http.MethodPost, "/v1/cluster/apply", plan)
}

func (client *Client) Uninstall(ctx context.Context, input UninstallRequest) (ClusterState, error) {
	return client.request(ctx, http.MethodPost, "/v1/cluster/uninstall", input)
}

func (client *Client) FoundationStatuses(ctx context.Context) ([]FoundationState, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://unix/v1/foundation/status", nil)
	if err != nil {
		return nil, err
	}
	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	var envelope struct {
		Code int               `json:"code"`
		Msg  string            `json:"msg"`
		Data []FoundationState `json:"data"`
	}
	decoder := json.NewDecoder(response.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&envelope); err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 || envelope.Code != 0 {
		return nil, fmt.Errorf("hostd 请求失败: status=%d code=%d msg=%s", response.StatusCode, envelope.Code, envelope.Msg)
	}
	return envelope.Data, nil
}

func (client *Client) ApplyFoundation(ctx context.Context, plan FoundationPlan) (FoundationState, error) {
	return client.foundationRequest(ctx, http.MethodPost, "/v1/foundation/apply", plan)
}

func (client *Client) DeleteFoundation(ctx context.Context, input FoundationDeleteRequest) (FoundationState, error) {
	return client.foundationRequest(ctx, http.MethodPost, "/v1/foundation/delete", input)
}

func (client *Client) TimeSyncStatus(ctx context.Context) (TimeSyncState, error) {
	return client.timeSyncRequest(ctx, http.MethodGet, "/v1/time-sync/status", nil)
}

func (client *Client) ApplyTimeSync(ctx context.Context, plan TimeSyncPlan) (TimeSyncState, error) {
	return client.timeSyncRequest(ctx, http.MethodPost, "/v1/time-sync/apply", plan)
}

func (client *Client) timeSyncRequest(ctx context.Context, method, path string, payload any) (TimeSyncState, error) {
	var body bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			return TimeSyncState{}, err
		}
	}
	request, err := http.NewRequestWithContext(ctx, method, "http://unix"+path, &body)
	if err != nil {
		return TimeSyncState{}, err
	}
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := client.httpClient.Do(request)
	if err != nil {
		return TimeSyncState{}, err
	}
	defer response.Body.Close()
	var envelope struct {
		Code int           `json:"code"`
		Msg  string        `json:"msg"`
		Data TimeSyncState `json:"data"`
	}
	decoder := json.NewDecoder(response.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&envelope); err != nil {
		return TimeSyncState{}, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 || envelope.Code != 0 {
		return TimeSyncState{}, fmt.Errorf("hostd 请求失败: status=%d code=%d msg=%s", response.StatusCode, envelope.Code, envelope.Msg)
	}
	return envelope.Data, nil
}

func (client *Client) foundationRequest(ctx context.Context, method, path string, payload any) (FoundationState, error) {
	var body bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			return FoundationState{}, err
		}
	}
	request, err := http.NewRequestWithContext(ctx, method, "http://unix"+path, &body)
	if err != nil {
		return FoundationState{}, err
	}
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := client.httpClient.Do(request)
	if err != nil {
		return FoundationState{}, err
	}
	defer response.Body.Close()
	var envelope struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data FoundationState `json:"data"`
	}
	decoder := json.NewDecoder(response.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&envelope); err != nil {
		return FoundationState{}, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 || envelope.Code != 0 {
		return FoundationState{}, fmt.Errorf("hostd 请求失败: status=%d code=%d msg=%s", response.StatusCode, envelope.Code, envelope.Msg)
	}
	return envelope.Data, nil
}

func (client *Client) request(ctx context.Context, method, path string, payload any) (ClusterState, error) {
	var body bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			return ClusterState{}, err
		}
	}
	request, err := http.NewRequestWithContext(ctx, method, "http://unix"+path, &body)
	if err != nil {
		return ClusterState{}, err
	}
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := client.httpClient.Do(request)
	if err != nil {
		return ClusterState{}, err
	}
	defer response.Body.Close()
	var envelope struct {
		Code int          `json:"code"`
		Msg  string       `json:"msg"`
		Data ClusterState `json:"data"`
	}
	decoder := json.NewDecoder(response.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&envelope); err != nil {
		return ClusterState{}, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 || envelope.Code != 0 {
		return ClusterState{}, fmt.Errorf("hostd 请求失败: status=%d code=%d msg=%s", response.StatusCode, envelope.Code, envelope.Msg)
	}
	return envelope.Data, nil
}

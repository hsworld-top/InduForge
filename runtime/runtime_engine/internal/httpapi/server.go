// Package httpapi 暴露 RuntimeEngine 的最小健康和状态 HTTP 接口。
package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/indu-forge/runtime-engine/internal/loader"
)

type EngineState struct {
	mu        sync.RWMutex
	startedAt time.Time
	version   string
	loaded    *loader.Loaded
	loadError error
}

func NewEngineState(version string, loaded *loader.Loaded, loadError error) *EngineState {
	return &EngineState{startedAt: time.Now().UTC(), version: version, loaded: loaded, loadError: loadError}
}

type envelope struct {
	Code  int    `json:"code"`
	Msg   string `json:"msg"`
	Data  any    `json:"data"`
	ReqID string `json:"reqId"`
}

type status struct {
	SchemaVersion     string    `json:"schemaVersion"`
	ComponentRole     string    `json:"componentRole"`
	SiteID            string    `json:"siteId,omitempty"`
	ProjectID         string    `json:"projectId,omitempty"`
	DeploymentID      string    `json:"deploymentId,omitempty"`
	AccountID         string    `json:"accountId,omitempty"`
	ExecutionForm     string    `json:"executionForm"`
	LifecycleState    string    `json:"lifecycleState"`
	HealthState       string    `json:"healthState"`
	Version           string    `json:"version"`
	StartedAt         string    `json:"startedAt"`
	UptimeSeconds     int64     `json:"uptimeSeconds"`
	ObservedAt        string    `json:"observedAt"`
	LastError         *string   `json:"lastError"`
	ReasonCode        *string   `json:"reasonCode"`
	BusinessFreshness freshness `json:"businessFreshness"`
}

type freshness struct {
	State               string  `json:"state"`
	LastBusinessEventAt *string `json:"lastBusinessEventAt"`
	EvaluatedAt         string  `json:"evaluatedAt"`
}

func (state *EngineState) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", state.handleStatus)
	mux.HandleFunc("/api/v1/runtime/status", state.handleStatus)
	return mux
}

func (state *EngineState) handleStatus(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", http.MethodGet)
		writeEnvelope(writer, http.StatusMethodNotAllowed, envelope{Code: 1, Msg: "仅支持 GET", Data: nil, ReqID: requestID()})
		return
	}
	current := state.snapshot()
	if state.loadError != nil {
		writeEnvelope(writer, http.StatusServiceUnavailable, envelope{Code: 1, Msg: "RuntimeEngine 配置或 Artifact 校验失败", Data: current, ReqID: requestID()})
		return
	}
	writeEnvelope(writer, http.StatusOK, envelope{Code: 0, Msg: "ok", Data: current, ReqID: requestID()})
}

func (state *EngineState) snapshot() status {
	state.mu.RLock()
	defer state.mu.RUnlock()
	now := time.Now().UTC()
	value := status{
		SchemaVersion: "runtime-health-status.v1",
		ComponentRole: "runtime-engine",
		// 校验失败前无法信任外部配置，仍提供稳定占位身份以保持 status data 合同完整。
		SiteID:            "unknown",
		DeploymentID:      "unknown",
		AccountID:         "unknown",
		ExecutionForm:     "k3s-workload",
		LifecycleState:    "RUNNING",
		HealthState:       "HEALTHY",
		Version:           state.version,
		StartedAt:         utcTimestamp(state.startedAt),
		UptimeSeconds:     int64(now.Sub(state.startedAt).Seconds()),
		ObservedAt:        utcTimestamp(now),
		BusinessFreshness: freshness{State: "UNKNOWN", LastBusinessEventAt: nil, EvaluatedAt: utcTimestamp(now)},
	}
	if state.loaded != nil {
		value.SiteID = state.loaded.Config.SiteID
		value.ProjectID = state.loaded.Config.ProjectID
		value.DeploymentID = state.loaded.Config.DeploymentID
		value.AccountID = state.loaded.Config.AccountID
	}
	if state.loadError != nil {
		value.LifecycleState, value.HealthState = "FAILED", "UNAVAILABLE"
		value.LastError, value.ReasonCode = stringPointer("运行配置或项目产物校验未通过"), stringPointer("ARTIFACT_VALIDATION_FAILED")
	}
	return value
}

func writeEnvelope(writer http.ResponseWriter, httpStatus int, body envelope) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(httpStatus)
	_ = json.NewEncoder(writer).Encode(body)
}

func requestID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "runtime-engine"
	}
	return hex.EncodeToString(bytes)
}

func utcTimestamp(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }
func stringPointer(value string) *string  { return &value }

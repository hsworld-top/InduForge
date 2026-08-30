// Package httpapi exposes only the frozen RuntimeEngine health contract.
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

type Lifecycle string

const (
	Starting Lifecycle = "STARTING"
	Running  Lifecycle = "RUNNING"
	Stopping Lifecycle = "STOPPING"
	Stopped  Lifecycle = "STOPPED"
	Failed   Lifecycle = "FAILED"
)

type Health string

const (
	Healthy     Health = "HEALTHY"
	Degraded    Health = "DEGRADED"
	Unavailable Health = "UNAVAILABLE"
)

type EngineState struct {
	mu           sync.RWMutex
	startedAt    time.Time
	version      string
	loaded       *loader.Loaded
	lifecycle    Lifecycle
	health       Health
	reason       *string
	lastBusiness *time.Time
}

func NewEngineState(version string, loaded *loader.Loaded, loadError error) *EngineState {
	s := &EngineState{startedAt: time.Now().UTC(), version: version, loaded: loaded, lifecycle: Starting, health: Unavailable}
	if loadError != nil {
		s.lifecycle = Failed
		s.reason = ptr("ARTIFACT_VALIDATION_FAILED")
	}
	return s
}
func (s *EngineState) SetLoaded(v *loader.Loaded) { s.mu.Lock(); s.loaded = v; s.mu.Unlock() }
func (s *EngineState) SetState(l Lifecycle, h Health, reasonCode string) {
	s.mu.Lock()
	if !validTransition(s.lifecycle, l) {
		s.mu.Unlock()
		return
	}
	s.lifecycle = l
	s.health = h
	if reasonCode == "" {
		s.reason = nil
	} else {
		s.reason = ptr(reasonCode)
	}
	s.mu.Unlock()
}
func validTransition(from, to Lifecycle) bool {
	if from == to || to == Failed {
		return from != Stopped
	}
	switch from {
	case Starting:
		return to == Running || to == Stopping || to == Stopped
	case Running:
		return to == Stopping
	case Stopping:
		return to == Stopped
	}
	return false
}
func (s *EngineState) RecordBusinessSuccess(at time.Time) {
	s.mu.Lock()
	v := at.UTC()
	s.lastBusiness = &v
	s.mu.Unlock()
}

// ReportDegraded never changes lifecycle: a late worker signal during drain
// must not turn a STOPPING/FAILED engine back into RUNNING.
func (s *EngineState) ReportDegraded(code string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lifecycle != Running {
		return
	}
	s.health = Degraded
	if code == "" {
		s.reason = nil
	} else {
		s.reason = ptr(code)
	}
}
func (s *EngineState) ReportIngressFailure(code string) { s.ReportDegraded(code) }

type envelope struct {
	Code  int    `json:"code"`
	Msg   string `json:"msg"`
	Data  any    `json:"data"`
	ReqID string `json:"reqId"`
}
type status struct {
	SchemaVersion     string    `json:"schemaVersion"`
	ComponentRole     string    `json:"componentRole"`
	SiteID            string    `json:"siteId"`
	ProjectID         string    `json:"projectId,omitempty"`
	DeploymentID      string    `json:"deploymentId"`
	AccountID         string    `json:"accountId"`
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

func (s *EngineState) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" && r.URL.Path != "/api/v1/status" {
			writeEnvelope(w, http.StatusNotFound, envelope{Code: 404, Msg: "not found", ReqID: requestID()})
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeEnvelope(w, http.StatusMethodNotAllowed, envelope{Code: 405, Msg: "method not allowed", ReqID: requestID()})
			return
		}
		value := s.snapshot()
		if r.URL.Path == "/health" {
			up := value.LifecycleState == string(Running) && (value.HealthState == string(Healthy) || value.HealthState == string(Degraded))
			code, statusCode := 1, http.StatusServiceUnavailable
			if up {
				code, statusCode = 0, http.StatusOK
			}
			writeEnvelope(w, statusCode, envelope{Code: code, Msg: map[bool]string{true: "ok", false: "unavailable"}[up], Data: struct {
				Status     string `json:"status"`
				ObservedAt string `json:"observedAt"`
			}{map[bool]string{true: "UP", false: "DOWN"}[up], value.ObservedAt}, ReqID: requestID()})
			return
		}
		writeEnvelope(w, http.StatusOK, envelope{Code: 0, Msg: "ok", Data: value, ReqID: requestID()})
	})
}
func (s *EngineState) snapshot() status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := time.Now().UTC()
	v := status{SchemaVersion: "runtime-health-status.v1", ComponentRole: "runtime-engine", SiteID: "unknown", DeploymentID: "unknown", AccountID: "unknown", ExecutionForm: "k3s-workload", LifecycleState: string(s.lifecycle), HealthState: string(s.health), Version: s.version, StartedAt: timestamp(s.startedAt), UptimeSeconds: int64(now.Sub(s.startedAt).Seconds()), ObservedAt: timestamp(now), ReasonCode: s.reason, BusinessFreshness: freshness{State: "UNKNOWN", EvaluatedAt: timestamp(now)}}
	if s.loaded != nil {
		v.SiteID = s.loaded.Config.SiteID
		v.ProjectID = s.loaded.Config.ProjectID
		v.DeploymentID = s.loaded.Config.DeploymentID
		v.AccountID = s.loaded.Config.AccountID
		v.ExecutionForm = s.loaded.Config.ExecutionForm
	}
	if s.lastBusiness != nil {
		x := timestamp(*s.lastBusiness)
		v.BusinessFreshness.LastBusinessEventAt = &x
		if now.Sub(*s.lastBusiness) <= 2*time.Minute {
			v.BusinessFreshness.State = "FRESH"
		} else {
			v.BusinessFreshness.State = "STALE"
		}
	}
	if s.reason != nil {
		x := "runtime-engine failure: " + *s.reason
		v.LastError = &x
	}
	return v
}
func writeEnvelope(w http.ResponseWriter, code int, body envelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}
func requestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "runtime-engine"
	}
	return hex.EncodeToString(b)
}
func timestamp(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }
func ptr(v string) *string         { return &v }

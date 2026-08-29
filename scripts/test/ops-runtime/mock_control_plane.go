package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
)

func respond(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "msg": "ok", "data": data, "reqId": "e2e"})
}
func main() {
	var mu sync.RWMutex
	phase := "start"
	http.HandleFunc("/advance", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		phase = r.URL.Query().Get("phase")
		mu.Unlock()
		respond(w, map[string]any{"phase": phase})
	})
	http.HandleFunc("/api/v1/ops/agent/enrollments/claim", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Code string `json:"code"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		role := "runtime_linux"
		if strings.Contains(body.Code, "collector") {
			role = "collector_linux"
		}
		respond(w, map[string]any{"enrollment": map[string]any{"id": "enroll-" + role, "role": role, "status": "claimed"}, "hostNode": map[string]any{"id": "host-" + role, "role": role, "observedStatus": "pending_approval"}, "agentToken": "token-" + role, "pendingApproval": false})
	})
	http.HandleFunc("/api/v1/ops/agent/host-nodes/", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer token-") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/commands") {
			id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/ops/agent/host-nodes/"), "/commands")
			mu.RLock()
			current := phase
			mu.RUnlock()
			commands := commandsFor(id, current)
			respond(w, map[string]any{"commands": commands})
			return
		}
		respond(w, map[string]any{"hostNode": map[string]any{"id": "ok", "observedStatus": "online"}, "commands": []any{}})
	})
	_ = http.ListenAndServe(":18080", nil)
}

func commandsFor(id, phase string) []map[string]any {
	role, workload, generation, desired, operation := "collector", "collector-e2e", 1, "running", "deploy"
	if id == "host-runtime_linux" {
		role, workload = "compute", "compute-e2e"
	}
	if phase == "stop" {
		generation, desired, operation = 2, "stopped", "stop"
	}
	if phase == "restart" {
		generation, desired, operation = 3, "running", "restart"
	}
	commands := []map[string]any{{"commandId": workload + "-cmd", "runId": "run-1", "workloadId": workload, "deploymentId": "deployment-e2e", "role": role, "operation": operation, "desiredStatus": desired, "generation": generation}}
	if id == "host-runtime_linux" {
		commands = append(commands, map[string]any{"commandId": "alert-e2e-cmd", "runId": "run-1", "workloadId": "alert-e2e", "deploymentId": "deployment-e2e", "role": "alert", "operation": operation, "desiredStatus": desired, "generation": generation})
	}
	return commands
}

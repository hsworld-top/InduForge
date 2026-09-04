package ops

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/indu-forge/dev_core/internal/auth"
)

func (h *Handler) listRuntimeEnvironments(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		filter := page(r)
		items, total, err := h.service.ListRuntimeEnvironments(r.Context(), u, filter)
		if err != nil {
			h.err(w, r, err)
			return
		}
		payload := make([]any, 0, len(items))
		for _, item := range items {
			payload = append(payload, runtimeEnvironmentPayload(item))
		}
		h.writeSuccess(w, r, pageData(payload, total, filter))
	})
}

func (h *Handler) createRuntimeEnvironment(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		var input CreateRuntimeEnvironmentInput
		if !decode(r, &input) {
			h.invalid(w, r)
			return
		}
		item, err := h.service.CreateRuntimeEnvironment(r.Context(), u, input)
		if err != nil {
			h.err(w, r, err)
			return
		}
		h.writeSuccess(w, r, runtimeEnvironmentPayload(item))
	})
}

func (h *Handler) getRuntimeEnvironment(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		item, err := h.service.GetRuntimeEnvironment(r.Context(), u, chi.URLParam(r, "id"))
		if err != nil {
			h.err(w, r, err)
			return
		}
		h.writeSuccess(w, r, runtimeEnvironmentPayload(item))
	})
}

func (h *Handler) updateRuntimeEnvironment(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		var input UpdateRuntimeEnvironmentInput
		if !decode(r, &input) {
			h.invalid(w, r)
			return
		}
		item, err := h.service.UpdateRuntimeEnvironment(r.Context(), u, chi.URLParam(r, "id"), input)
		if err != nil {
			h.err(w, r, err)
			return
		}
		h.writeSuccess(w, r, runtimeEnvironmentPayload(item))
	})
}

func (h *Handler) deleteRuntimeEnvironment(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		var input DeleteRuntimeEnvironmentInput
		if !decode(r, &input) {
			h.invalid(w, r)
			return
		}
		status, err := h.service.DeleteRuntimeEnvironment(r.Context(), u, chi.URLParam(r, "id"), input)
		if err != nil {
			h.err(w, r, err)
			return
		}
		h.writeSuccess(w, r, map[string]any{"status": status})
	})
}

func (h *Handler) listRuntimeEnvironmentNodes(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		filter := page(r)
		items, total, err := h.service.ListRuntimeEnvironmentNodes(r.Context(), u, chi.URLParam(r, "id"), filter)
		if err != nil {
			h.err(w, r, err)
			return
		}
		payload := make([]any, 0, len(items))
		for _, item := range items {
			payload = append(payload, nodePayload(item))
		}
		h.writeSuccess(w, r, pageData(payload, total, filter))
	})
}

func (h *Handler) addRuntimeEnvironmentNodes(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		var input AddRuntimeEnvironmentNodesInput
		if !decode(r, &input) {
			h.invalid(w, r)
			return
		}
		items, err := h.service.AddRuntimeEnvironmentNodes(r.Context(), u, chi.URLParam(r, "id"), input)
		if err != nil {
			h.err(w, r, err)
			return
		}
		payload := make([]any, 0, len(items))
		for _, item := range items {
			payload = append(payload, nodePayload(item))
		}
		h.writeSuccess(w, r, map[string]any{"items": payload, "total": len(payload)})
	})
}

func (h *Handler) removeRuntimeEnvironmentNode(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		if err := h.service.RemoveRuntimeEnvironmentNode(r.Context(), u, chi.URLParam(r, "id"), chi.URLParam(r, "nodeId")); err != nil {
			h.err(w, r, err)
			return
		}
		h.writeSuccess(w, r, map[string]any{"removed": true})
	})
}

func (h *Handler) listRuntimeEnvironmentEvents(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		filter := page(r)
		items, total, err := h.service.ListRuntimeEnvironmentEvents(r.Context(), u, chi.URLParam(r, "id"), filter)
		if err != nil {
			h.err(w, r, err)
			return
		}
		payload := make([]any, 0, len(items))
		for _, item := range items {
			payload = append(payload, runtimeEnvironmentEventPayload(item))
		}
		h.writeSuccess(w, r, pageData(payload, total, filter))
	})
}

func (h *Handler) listRuntimeEnvironmentServices(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		items, err := h.service.ListRuntimeEnvironmentServices(r.Context(), u, chi.URLParam(r, "id"))
		if err != nil {
			h.err(w, r, err)
			return
		}
		payload := make([]any, 0, len(items))
		for _, item := range items {
			payload = append(payload, runtimeEnvironmentServicePayload(item))
		}
		h.writeSuccess(w, r, map[string]any{"items": payload, "total": len(payload)})
	})
}

func (h *Handler) deployRuntimeEnvironmentFoundation(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		var input DeployFoundationInput
		if !decode(r, &input) {
			h.invalid(w, r)
			return
		}
		items, err := h.service.DeployRuntimeEnvironmentFoundation(r.Context(), u, chi.URLParam(r, "id"), input)
		if err != nil {
			h.err(w, r, err)
			return
		}
		payload := make([]any, 0, len(items))
		for _, item := range items {
			payload = append(payload, runtimeEnvironmentServicePayload(item))
		}
		h.writeSuccess(w, r, map[string]any{"items": payload, "total": len(payload)})
	})
}

func (h *Handler) migrateRuntimeEnvironmentFoundation(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(u auth.User) {
		var input MigrateFoundationInput
		if !decode(r, &input) {
			h.invalid(w, r)
			return
		}
		items, err := h.service.MigrateRuntimeEnvironmentFoundation(r.Context(), u, chi.URLParam(r, "id"), input)
		if err != nil {
			h.err(w, r, err)
			return
		}
		payload := make([]any, 0, len(items))
		for _, item := range items {
			payload = append(payload, runtimeEnvironmentServicePayload(item))
		}
		h.writeSuccess(w, r, map[string]any{"items": payload, "total": len(payload)})
	})
}

func runtimeEnvironmentServicePayload(item RuntimeEnvironmentService) map[string]any {
	return map[string]any{
		"id": item.ID, "environmentId": item.EnvironmentID, "nodeId": item.NodeID,
		"nodeName": item.NodeName, "serviceType": item.ServiceType,
		"desiredStatus": item.DesiredStatus, "observedStatus": item.ObservedStatus,
		"lastMessage": item.LastMessage, "desiredGeneration": item.DesiredGeneration,
		"observedGeneration": item.ObservedGeneration, "observedAt": item.ObservedAt,
		"observedStale": item.ObservedStale,
		"operation":     item.Operation,
		"createdAt":     item.CreatedAt, "updatedAt": item.UpdatedAt,
	}
}

func runtimeEnvironmentPayload(item RuntimeEnvironment) map[string]any {
	return map[string]any{
		"id": item.ID, "name": item.Name, "code": item.Code,
		"isDefault": item.IsDefault,
		"status":    item.Status, "desiredStatus": item.DesiredStatus,
		"nodeCount": item.NodeCount, "onlineNodeCount": item.OnlineNodeCount,
		"foundationTotal": item.FoundationTotal, "foundationHealthy": item.FoundationHealthy,
		"projectCount": item.ProjectCount, "recentChange": item.RecentChange,
		"runningDeploymentCount": item.RunningDeploymentCount,
		"recentAt":               item.RecentAt, "recentBy": item.RecentBy,
		"createdAt": item.CreatedAt, "updatedAt": item.UpdatedAt,
	}
}

func runtimeEnvironmentEventPayload(item RuntimeEnvironmentEvent) map[string]any {
	return map[string]any{
		"id": item.ID, "environmentId": item.EnvironmentID, "eventType": item.EventType,
		"name": item.Name, "target": item.Target, "result": item.Result,
		"message": item.Message, "operator": item.OperatorName, "createdAt": item.CreatedAt,
	}
}

package hostd

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

const maxRequestBytes = 64 << 10

type API struct {
	manager *Manager
}

func NewAPI(manager *Manager) (*API, error) {
	if manager == nil {
		return nil, errors.New("hostd manager 不能为空")
	}
	return &API{manager: manager}, nil
}

func (api *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/cluster/status", api.status)
	mux.HandleFunc("POST /v1/cluster/apply", api.apply)
	mux.HandleFunc("POST /v1/cluster/uninstall", api.uninstall)
	mux.HandleFunc("GET /v1/foundation/status", api.foundationStatus)
	mux.HandleFunc("POST /v1/foundation/apply", api.foundationApply)
	mux.HandleFunc("POST /v1/foundation/delete", api.foundationDelete)
	mux.HandleFunc("GET /v1/time-sync/status", api.timeSyncStatus)
	mux.HandleFunc("POST /v1/time-sync/apply", api.timeSyncApply)
	return mux
}

func (api *API) timeSyncStatus(writer http.ResponseWriter, request *http.Request) {
	state, err := api.manager.TimeSyncStatus(request.Context())
	api.writeTimeSync(writer, state, err)
}

func (api *API) timeSyncApply(writer http.ResponseWriter, request *http.Request) {
	var plan TimeSyncPlan
	if err := decodeStrict(writer, request, &plan); err != nil {
		api.writeError(writer, http.StatusBadRequest, err)
		return
	}
	state, err := api.manager.ApplyTimeSync(request.Context(), plan)
	api.writeTimeSync(writer, state, err)
}

func (api *API) writeTimeSync(writer http.ResponseWriter, state TimeSyncState, err error) {
	if err != nil {
		api.writeError(writer, http.StatusConflict, err)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode(map[string]any{"code": 0, "msg": "ok", "data": state})
}

func (api *API) foundationStatus(writer http.ResponseWriter, request *http.Request) {
	states, err := api.manager.FoundationStatuses(request.Context())
	if err != nil {
		api.writeError(writer, http.StatusConflict, err)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode(map[string]any{"code": 0, "msg": "ok", "data": states})
}

func (api *API) foundationDelete(writer http.ResponseWriter, request *http.Request) {
	var input FoundationDeleteRequest
	if err := decodeStrict(writer, request, &input); err != nil {
		api.writeError(writer, http.StatusBadRequest, err)
		return
	}
	state, err := api.manager.DeleteFoundation(request.Context(), input)
	api.writeFoundation(writer, state, err)
}

func (api *API) foundationApply(writer http.ResponseWriter, request *http.Request) {
	var plan FoundationPlan
	if err := decodeStrict(writer, request, &plan); err != nil {
		api.writeError(writer, http.StatusBadRequest, err)
		return
	}
	state, err := api.manager.ApplyFoundation(request.Context(), plan)
	api.writeFoundation(writer, state, err)
}

func (api *API) writeFoundation(writer http.ResponseWriter, state FoundationState, err error) {
	if err != nil {
		api.writeError(writer, http.StatusConflict, err)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode(map[string]any{"code": 0, "msg": "ok", "data": state})
}

func (api *API) status(writer http.ResponseWriter, request *http.Request) {
	state, err := api.manager.Status(request.Context())
	api.write(writer, state, err)
}

func (api *API) apply(writer http.ResponseWriter, request *http.Request) {
	var plan ClusterPlan
	if err := decodeStrict(writer, request, &plan); err != nil {
		api.writeError(writer, http.StatusBadRequest, err)
		return
	}
	state, err := api.manager.Apply(request.Context(), plan)
	api.write(writer, state, err)
}

func (api *API) uninstall(writer http.ResponseWriter, request *http.Request) {
	var input UninstallRequest
	if err := decodeStrict(writer, request, &input); err != nil {
		api.writeError(writer, http.StatusBadRequest, err)
		return
	}
	state, err := api.manager.Uninstall(request.Context(), input)
	api.write(writer, state, err)
}

func decodeStrict(writer http.ResponseWriter, request *http.Request, destination any) error {
	request.Body = http.MaxBytesReader(writer, request.Body, maxRequestBytes)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("请求包含多个 JSON 值")
		}
		return err
	}
	return nil
}

func (api *API) write(writer http.ResponseWriter, state ClusterState, err error) {
	if err != nil {
		api.writeError(writer, http.StatusConflict, err)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode(map[string]any{"code": 0, "msg": "ok", "data": state})
}

func (api *API) writeError(writer http.ResponseWriter, status int, err error) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(map[string]any{"code": status, "msg": err.Error(), "data": nil})
}

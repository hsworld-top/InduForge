package api

import (
	"encoding/json"
	"net/http"
)

const SuccessCode = 0

type Envelope struct {
	Code  int    `json:"code"`
	Msg   string `json:"msg"`
	Data  any    `json:"data"`
	ReqID string `json:"reqId"`
}

func WriteSuccess(w http.ResponseWriter, r *http.Request, data any) {
	WriteJSON(w, http.StatusOK, Envelope{
		Code:  SuccessCode,
		Msg:   "success",
		Data:  data,
		ReqID: RequestIDFromContext(r.Context()),
	})
}

func WriteError(w http.ResponseWriter, r *http.Request, statusCode, code int, message string) {
	WriteErrorData(w, r, statusCode, code, message, nil)
}

func WriteErrorData(w http.ResponseWriter, r *http.Request, statusCode, code int, message string, data any) {
	WriteJSON(w, statusCode, Envelope{
		Code:  code,
		Msg:   message,
		Data:  data,
		ReqID: RequestIDFromContext(r.Context()),
	})
}

func WriteJSON(w http.ResponseWriter, statusCode int, payload Envelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

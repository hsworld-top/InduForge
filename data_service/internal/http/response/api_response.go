package response

import (
	"encoding/json"
	"net/http"
)

// ApiResponse 是 data_service 统一的 HTTP 响应结构。
type ApiResponse struct {
	Success   bool   `json:"success"`
	ErrorCode string `json:"errorCode"`
	Message   string `json:"message"`
	RequestID string `json:"requestId"`
	Data      any    `json:"data"`
}

// WriteSuccess 写出统一成功响应。
func WriteSuccess(w http.ResponseWriter, requestID string, data any) {
	writeJSON(w, http.StatusOK, ApiResponse{
		Success:   true,
		RequestID: requestID,
		Data:      data,
	})
}

// WriteError 写出统一错误响应。
func WriteError(w http.ResponseWriter, statusCode int, requestID, errorCode, message string) {
	writeJSON(w, statusCode, ApiResponse{
		Success:   false,
		ErrorCode: errorCode,
		Message:   message,
		RequestID: requestID,
	})
}

// writeJSON 将响应结构序列化后写回客户端。
func writeJSON(w http.ResponseWriter, statusCode int, payload ApiResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

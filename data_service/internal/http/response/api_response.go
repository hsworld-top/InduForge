package response

import (
	"encoding/json"
	"net/http"
	"strings"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

// ApiResponse 是 data_service 统一的 HTTP 响应结构。
type ApiResponse struct {
	// Code 是业务码：0 表示成功，其余为错误码。
	Code int `json:"code"`
	// Msg 是响应提示信息。
	Msg string `json:"msg"`
	// Data 是业务数据。错误响应固定为 nil。
	Data any `json:"data"`
	// ReqID 是请求链路 ID，便于跨服务排障。
	ReqID string `json:"reqId"`
}

// WriteSuccess 写出统一成功响应。
func WriteSuccess(w http.ResponseWriter, requestID string, data any) {
	writeJSON(w, http.StatusOK, ApiResponse{
		Code:  apperrors.SuccessCode,
		Msg:   "success",
		Data:  data,
		ReqID: requestID,
	})
}

// WriteAppError 使用内部强类型错误码写出统一错误响应。
func WriteAppError(w http.ResponseWriter, statusCode int, requestID string, errorCode apperrors.ErrorCode, message string) {
	WriteErrorCode(w, statusCode, requestID, errorCode.PublicCode(), message)
}

// WriteAppErrorData 仅用于需要向客户端返回恢复动作上下文的结构化冲突。
func WriteAppErrorData(w http.ResponseWriter, statusCode int, requestID string, errorCode apperrors.ErrorCode, message string, data any) {
	msg := strings.TrimSpace(message)
	if msg == "" {
		msg = "系统内部错误"
	}
	writeJSON(w, statusCode, ApiResponse{Code: errorCode.PublicCode(), Msg: msg, Data: data, ReqID: requestID})
}

// WriteErrorCode 使用公开整数错误码写出统一错误响应。
func WriteErrorCode(w http.ResponseWriter, statusCode int, requestID string, code int, message string) {
	msg := strings.TrimSpace(message)
	if msg == "" {
		msg = "系统内部错误"
	}

	writeJSON(w, statusCode, ApiResponse{
		Code:  code,
		Msg:   msg,
		Data:  nil,
		ReqID: requestID,
	})
}

// WriteError 使用历史字符串错误码写出统一错误响应。
// 该函数保留用于兼容旧调用方，新代码优先使用 WriteAppError 或 WriteErrorCode。
func WriteError(w http.ResponseWriter, statusCode int, requestID, errorCode, message string) {
	WriteErrorCode(w, statusCode, requestID, apperrors.ResolvePublicCode(errorCode), message)
}

// writeJSON 将响应结构序列化后写回客户端。
func writeJSON(w http.ResponseWriter, statusCode int, payload ApiResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

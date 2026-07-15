package handler

import (
	"fmt"
	"io"
	"net/http"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

const maxCollectorImportFileSize = 20 << 20

type CollectorImportHandler struct {
	service *service.CollectorImportService
}

func NewCollectorImportHandler(importService *service.CollectorImportService) *CollectorImportHandler {
	return &CollectorImportHandler{service: importService}
}
func (h *CollectorImportHandler) Template(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	result, err := h.service.BuildTemplate(r.PathValue("driverId"), r.URL.Query().Get("format"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	w.Header().Set("Content-Type", result.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, result.FileName))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(result.Content)
	return nil
}
func (h *CollectorImportHandler) Preview(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxCollectorImportFileSize)
	if err := r.ParseMultipartForm(maxCollectorImportFileSize); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "导入文件超过 20MB 或表单格式无效", err)
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "缺少导入文件", err)
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "读取导入文件失败", err)
	}
	page, err := parseOptionalInt(r.FormValue("page"), 1, "page")
	if err != nil {
		return err
	}
	pageSize, err := parseOptionalInt(r.FormValue("pageSize"), 20, "pageSize")
	if err != nil {
		return err
	}
	result, err := h.service.Preview(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, header.Filename, content, page, pageSize)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}
func (h *CollectorImportHandler) Commit(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	var input struct {
		ImportID string `json:"importId"`
	}
	if err := decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.service.Commit(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), input.ImportID)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"list": result})
	return nil
}

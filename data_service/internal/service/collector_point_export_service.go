package service

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

type CollectorPointExportRequest struct {
	Format          string   `json:"format"`
	Scope           string   `json:"scope"`
	GroupID         *string  `json:"groupId"`
	IncludeChildren bool     `json:"includeChildren"`
	Search          string   `json:"search"`
	DataType        string   `json:"dataType"`
	Enabled         *bool    `json:"enabled"`
	SortBy          string   `json:"sortBy"`
	SortOrder       string   `json:"sortOrder"`
	Page            int      `json:"page"`
	PageSize        int      `json:"pageSize"`
	Pages           []int    `json:"pages"`
	PointIDs        []string `json:"pointIds"`
}

type CollectorPointExportPlan struct {
	FileName          string
	projectID         string
	connectionID      string
	addressProperties []string
	filter            repository.CollectorPointExportFilter
	store             CollectorPointStore
}

func (s *CollectorPointService) PreparePointExport(ctx context.Context, projectID, connectionID string, input CollectorPointExportRequest) (CollectorPointExportPlan, error) {
	format := strings.ToLower(strings.TrimSpace(input.Format))
	if format == "" {
		format = "csv"
	}
	if format != "csv" {
		return CollectorPointExportPlan{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "当前仅支持 CSV 导出")
	}
	scope := strings.TrimSpace(input.Scope)
	if scope != "group" && scope != "current_page" && scope != "selected" && scope != "pages" {
		return CollectorPointExportPlan{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "变量导出范围无效")
	}
	if input.PageSize <= 0 || input.PageSize > 500 {
		return CollectorPointExportPlan{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "导出分页大小必须为 1 到 500")
	}
	if scope == "current_page" && input.Page < 1 {
		return CollectorPointExportPlan{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "当前页码无效")
	}
	pages := make([]int64, 0, len(input.Pages))
	if scope == "pages" {
		if len(input.Pages) == 0 || len(input.Pages) > 200 {
			return CollectorPointExportPlan{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "指定页数必须为 1 到 200 页")
		}
		seen := make(map[int]struct{}, len(input.Pages))
		for _, page := range input.Pages {
			if page < 1 {
				return CollectorPointExportPlan{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "指定页码必须大于 0")
			}
			if _, exists := seen[page]; exists {
				continue
			}
			seen[page] = struct{}{}
			pages = append(pages, int64(page))
		}
		sort.Slice(pages, func(i, j int) bool { return pages[i] < pages[j] })
	}
	if scope == "selected" && (len(input.PointIDs) == 0 || len(input.PointIDs) > 10000) {
		return CollectorPointExportPlan{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "勾选导出变量数量必须为 1 到 10000")
	}
	connection, err := s.store.GetConnection(ctx, projectID, connectionID)
	if err != nil {
		return CollectorPointExportPlan{}, err
	}
	driver, ok := s.catalog.Driver(connection.DriverID)
	if !ok {
		return CollectorPointExportPlan{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "工业采集驱动不存在")
	}
	var addressSchema struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(driver.AddressSchema, &addressSchema); err != nil {
		return CollectorPointExportPlan{}, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取驱动地址字段失败", err)
	}
	addressProperties := make([]string, 0, len(addressSchema.Properties))
	for name := range addressSchema.Properties {
		addressProperties = append(addressProperties, name)
	}
	sort.Strings(addressProperties)
	fileName := fmt.Sprintf("%s-变量-%s.csv", sanitizeCollectorExportFileName(connection.Name), time.Now().Format("20060102150405"))
	return CollectorPointExportPlan{
		FileName: fileName, projectID: projectID, connectionID: connectionID,
		addressProperties: addressProperties, store: s.store,
		filter: repository.CollectorPointExportFilter{
			Scope: scope, GroupID: input.GroupID, IncludeChildren: input.IncludeChildren,
			Search: input.Search, DataType: input.DataType, Enabled: input.Enabled,
			SortBy: input.SortBy, SortOrder: input.SortOrder, Page: input.Page,
			PageSize: input.PageSize, Pages: pages, PointIDs: input.PointIDs,
		},
	}, nil
}

func (p CollectorPointExportPlan) WriteCSV(ctx context.Context, output io.Writer) error {
	if _, err := io.WriteString(output, "\ufeff"); err != nil {
		return err
	}
	writer := csv.NewWriter(output)
	headers := []string{"groupPath", "name", "description", "dataType", "elementCount", "enabled"}
	for _, name := range p.addressProperties {
		headers = append(headers, "address."+name)
	}
	headers = append(headers, "readOptions", "acquisition", "metadata")
	if err := writer.Write(headers); err != nil {
		return err
	}
	err := p.store.StreamPointsForExport(ctx, p.projectID, p.connectionID, p.filter, func(record repository.CollectorPointExportRecord) error {
		description := ""
		if record.Description != nil {
			description = *record.Description
		}
		row := []string{record.GroupPath, record.Name, description, record.DataType, strconv.Itoa(record.ElementCount), strconv.FormatBool(record.Enabled)}
		for _, name := range p.addressProperties {
			row = append(row, collectorExportCell(record.Address[name]))
		}
		row = append(row, collectorExportJSON(record.ReadOptions), collectorExportJSON(record.Acquisition), collectorExportJSON(record.Metadata))
		return writer.Write(row)
	})
	writer.Flush()
	if err != nil {
		return err
	}
	return writer.Error()
}

func collectorExportCell(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return collectorExportJSON(value)
}

func collectorExportJSON(value any) string {
	payload, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(payload)
}

func sanitizeCollectorExportFileName(value string) string {
	name := strings.TrimSpace(value)
	if name == "" {
		name = "工业连接"
	}
	replacer := strings.NewReplacer("\\", "_", "/", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	return replacer.Replace(name)
}

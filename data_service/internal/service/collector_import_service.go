package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/indu-forge/data_service/internal/collectorprotocol"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
	"github.com/xuri/excelize/v2"
)

const maxCollectorImportRows = 10000

type CollectorImportStore interface {
	CreateImportSession(context.Context, repository.CreateCollectorImportSessionParams) (*repository.CollectorImportSessionRecord, error)
	GetImportSession(context.Context, string, string, string) (*repository.CollectorImportSessionRecord, error)
	CommitImportSession(context.Context, string, string, string) ([]repository.CollectorPointRecord, error)
}
type CollectorImportTemplate struct {
	Content               []byte
	ContentType, FileName string
}
type CollectorImportPreview struct {
	ImportID   string                            `json:"importId"`
	TotalRows  int                               `json:"totalRows"`
	ValidRows  int                               `json:"validRows"`
	Candidates []CollectorPoint                  `json:"candidates"`
	Errors     []repository.CollectorImportError `json:"errors"`
	Pagination CollectorDriverPagination         `json:"pagination"`
	ExpiresAt  string                            `json:"expiresAt"`
}
type CollectorImportService struct {
	store             CollectorImportStore
	points            *CollectorPointService
	catalog           *collectorprotocol.Catalog
	addressProperties map[string]map[string]string
}

func NewCollectorImportService(store CollectorImportStore, points *CollectorPointService, catalog *collectorprotocol.Catalog) (*CollectorImportService, error) {
	if store == nil || points == nil || catalog == nil {
		return nil, fmt.Errorf("点位导入服务依赖不完整")
	}
	result := &CollectorImportService{store: store, points: points, catalog: catalog, addressProperties: map[string]map[string]string{}}
	for _, driver := range catalog.Drivers() {
		var document map[string]any
		if err := json.Unmarshal(driver.AddressSchema, &document); err != nil {
			return nil, err
		}
		properties, _ := document["properties"].(map[string]any)
		types := map[string]string{}
		for name, value := range properties {
			property, _ := value.(map[string]any)
			types[name], _ = property["type"].(string)
		}
		result.addressProperties[driver.Manifest.DriverID] = types
	}
	return result, nil
}

func (s *CollectorImportService) BuildTemplate(driverID, format string) (CollectorImportTemplate, error) {
	driver, ok := s.catalog.Driver(driverID)
	if !ok {
		return CollectorImportTemplate{}, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "工业采集驱动不存在")
	}
	headers := s.templateHeaders(driver.Manifest.DriverID)
	if strings.EqualFold(format, "xlsx") {
		file := excelize.NewFile()
		defer func() { _ = file.Close() }()
		sheet := file.GetSheetName(0)
		for index, header := range headers {
			cell, err := excelize.CoordinatesToCellName(index+1, 1)
			if err != nil {
				return CollectorImportTemplate{}, fmt.Errorf("生成工业点位模板列失败: %w", err)
			}
			if err := file.SetCellValue(sheet, cell, header); err != nil {
				return CollectorImportTemplate{}, fmt.Errorf("写入工业点位模板失败: %w", err)
			}
		}
		buffer, err := file.WriteToBuffer()
		if err != nil {
			return CollectorImportTemplate{}, err
		}
		return CollectorImportTemplate{Content: buffer.Bytes(), ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", FileName: driver.Manifest.DriverID + "-points.xlsx"}, nil
	}
	buffer := bytes.NewBuffer(nil)
	writer := csv.NewWriter(buffer)
	if err := writer.Write(headers); err != nil {
		return CollectorImportTemplate{}, fmt.Errorf("写入工业点位 CSV 模板失败: %w", err)
	}
	writer.Flush()
	return CollectorImportTemplate{Content: buffer.Bytes(), ContentType: "text/csv; charset=utf-8", FileName: driver.Manifest.DriverID + "-points.csv"}, writer.Error()
}

func (s *CollectorImportService) Preview(ctx context.Context, projectID, connectionID, userID, fileName string, content []byte, page, pageSize int) (CollectorImportPreview, error) {
	if page < 1 || pageSize < 1 || pageSize > 100 {
		return CollectorImportPreview{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "分页参数无效")
	}
	connection, err := s.storeConnection(ctx, projectID, connectionID)
	if err != nil {
		return CollectorImportPreview{}, err
	}
	rows, err := readCollectorImportRows(fileName, content)
	if err != nil {
		return CollectorImportPreview{}, err
	}
	if len(rows) < 2 {
		return CollectorImportPreview{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "导入文件没有数据行")
	}
	if len(rows)-1 > maxCollectorImportRows {
		return CollectorImportPreview{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "导入文件最多允许 10000 行")
	}
	headers := map[string]int{}
	for index, value := range rows[0] {
		headers[strings.TrimSpace(value)] = index
	}
	existingCodes, err := s.points.store.ListPointCodes(ctx, projectID, connectionID)
	if err != nil {
		return CollectorImportPreview{}, err
	}
	usedCodes := make(map[string]struct{}, len(existingCodes)+len(rows))
	for _, code := range existingCodes {
		usedCodes[code] = struct{}{}
	}
	candidates := make([]repository.CollectorImportCandidate, 0)
	issues := make([]repository.CollectorImportError, 0)
	for rowIndex, row := range rows[1:] {
		input, groupPath, parseErr := s.parseImportRow(connection.DriverID, headers, row)
		if parseErr != nil {
			issues = append(issues, repository.CollectorImportError{Row: rowIndex + 2, Code: "INVALID_ROW", Message: parseErr.Error()})
			continue
		}
		point, buildErr := s.points.buildPointParams(connection, userID, uuid.NewString(), allocateCollectorPointCode(input.Name, usedCodes), input)
		if buildErr != nil {
			issues = append(issues, repository.CollectorImportError{Row: rowIndex + 2, Code: "VALIDATION_FAILED", Message: buildErr.Error()})
			continue
		}
		candidates = append(candidates, repository.CollectorImportCandidate{Point: point, GroupPath: groupPath})
	}
	expiresAt := time.Now().Add(30 * time.Minute)
	session, err := s.store.CreateImportSession(ctx, repository.CreateCollectorImportSessionParams{ID: uuid.NewString(), ProjectID: projectID, ConnectionID: connectionID, DriverID: connection.DriverID, CreatedBy: userID, Candidates: candidates, Errors: issues, TotalRows: len(rows) - 1, ExpiresAt: expiresAt})
	if err != nil {
		return CollectorImportPreview{}, err
	}
	return buildCollectorImportPreview(session, page, pageSize), nil
}
func (s *CollectorImportService) GetPreview(ctx context.Context, projectID, connectionID, importID string, page, pageSize int) (CollectorImportPreview, error) {
	session, err := s.store.GetImportSession(ctx, projectID, connectionID, importID)
	if err != nil {
		return CollectorImportPreview{}, err
	}
	return buildCollectorImportPreview(session, page, pageSize), nil
}
func (s *CollectorImportService) Commit(ctx context.Context, projectID, connectionID, importID string) ([]CollectorPoint, error) {
	records, err := s.store.CommitImportSession(ctx, projectID, connectionID, importID)
	if err != nil {
		return nil, err
	}
	return mapCollectorPoints(records), nil
}
func (s *CollectorImportService) storeConnection(ctx context.Context, projectID, connectionID string) (*repository.CollectorConnectionRecord, error) {
	return s.points.store.GetConnection(ctx, projectID, connectionID)
}

func (s *CollectorImportService) templateHeaders(driverID string) []string {
	headers := []string{"groupPath", "name", "description", "dataType", "elementCount", "enabled"}
	properties := make([]string, 0, len(s.addressProperties[driverID]))
	for name := range s.addressProperties[driverID] {
		properties = append(properties, name)
	}
	sort.Strings(properties)
	for _, name := range properties {
		headers = append(headers, "address."+name)
	}
	return append(headers, "readOptions", "acquisitionMode", "acquisitionOverrides", "metadata")
}
func (s *CollectorImportService) parseImportRow(driverID string, headers map[string]int, row []string) (CreateCollectorPointInput, string, error) {
	get := func(name string) string {
		index, ok := headers[name]
		if !ok || index >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[index])
	}
	name, dataType := get("name"), get("dataType")
	if name == "" || dataType == "" {
		return CreateCollectorPointInput{}, "", fmt.Errorf("name、dataType 不能为空")
	}
	elementCount := 1
	if raw := get("elementCount"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			return CreateCollectorPointInput{}, "", fmt.Errorf("elementCount 必须是整数")
		}
		elementCount = value
	}
	enabled := true
	if raw := get("enabled"); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return CreateCollectorPointInput{}, "", fmt.Errorf("enabled 必须是布尔值")
		}
		enabled = value
	}
	address := map[string]any{}
	for key, valueType := range s.addressProperties[driverID] {
		raw := get("address." + key)
		if raw == "" {
			continue
		}
		value, err := parseCollectorImportScalar(raw, valueType)
		if err != nil {
			return CreateCollectorPointInput{}, "", fmt.Errorf("address.%s 格式无效", key)
		}
		address[key] = value
	}
	readOptions, err := parseCollectorImportObject(get("readOptions"))
	if err != nil {
		return CreateCollectorPointInput{}, "", fmt.Errorf("readOptions 必须是 JSON 对象")
	}
	acquisitionMode := get("acquisitionMode")
	if acquisitionMode == "" {
		acquisitionMode = "inherit"
	}
	acquisitionOverrides, err := parseCollectorImportObject(get("acquisitionOverrides"))
	if err != nil {
		return CreateCollectorPointInput{}, "", fmt.Errorf("acquisitionOverrides 必须是 JSON 对象")
	}
	metadata, err := parseCollectorImportObject(get("metadata"))
	if err != nil {
		return CreateCollectorPointInput{}, "", fmt.Errorf("metadata 必须是 JSON 对象")
	}
	var description *string
	if value := get("description"); value != "" {
		description = &value
	}
	return CreateCollectorPointInput{Name: name, Description: description, Address: address, DataType: dataType, ElementCount: elementCount, ReadOptions: readOptions, AcquisitionMode: acquisitionMode, AcquisitionOverrides: acquisitionOverrides, Enabled: &enabled, Metadata: metadata}, get("groupPath"), nil
}
func parseCollectorImportScalar(raw, valueType string) (any, error) {
	switch valueType {
	case "integer":
		return strconv.Atoi(raw)
	case "number":
		return strconv.ParseFloat(raw, 64)
	case "boolean":
		return strconv.ParseBool(raw)
	default:
		return raw, nil
	}
}
func parseCollectorImportObject(raw string) (map[string]any, error) {
	if raw == "" {
		return map[string]any{}, nil
	}
	var value map[string]any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return nil, err
	}
	return value, nil
}
func readCollectorImportRows(fileName string, content []byte) ([][]string, error) {
	switch strings.ToLower(filepath.Ext(fileName)) {
	case ".csv", ".tsv":
		reader := csv.NewReader(bytes.NewReader(content))
		if strings.EqualFold(filepath.Ext(fileName), ".tsv") {
			reader.Comma = '\t'
		}
		reader.FieldsPerRecord = -1
		rows, err := reader.ReadAll()
		if err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "解析导入文本失败", err)
		}
		return rows, nil
	case ".xlsx":
		file, err := excelize.OpenReader(bytes.NewReader(content))
		if err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "解析 XLSX 文件失败", err)
		}
		defer func() { _ = file.Close() }()
		sheets := file.GetSheetList()
		if len(sheets) == 0 {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "XLSX 没有工作表")
		}
		rows, err := file.GetRows(sheets[0])
		if err != nil {
			return nil, err
		}
		return rows, nil
	default:
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "仅支持 CSV、TSV 和 XLSX 文件")
	}
}
func buildCollectorImportPreview(session *repository.CollectorImportSessionRecord, page, pageSize int) CollectorImportPreview {
	start := (page - 1) * pageSize
	candidateEnd := min(start+pageSize, len(session.Candidates))
	errorEnd := min(start+pageSize, len(session.Errors))
	candidateList := []CollectorPoint{}
	if start < len(session.Candidates) {
		for _, candidate := range session.Candidates[start:candidateEnd] {
			candidateList = append(candidateList, toCollectorPointParams(candidate.Point))
		}
	}
	errorList := []repository.CollectorImportError{}
	if start < len(session.Errors) {
		errorList = append(errorList, session.Errors[start:errorEnd]...)
	}
	totalPages := 0
	if session.TotalRows > 0 {
		totalPages = (session.TotalRows + pageSize - 1) / pageSize
	}
	return CollectorImportPreview{ImportID: session.ID, TotalRows: session.TotalRows, ValidRows: session.ValidRows, Candidates: candidateList, Errors: errorList, Pagination: CollectorDriverPagination{Page: page, PageSize: pageSize, Total: session.TotalRows, TotalPages: totalPages}, ExpiresAt: session.ExpiresAt.Format("2006-01-02 15:04:05")}
}
func toCollectorPointParams(point repository.CreateCollectorPointParams) CollectorPoint {
	return CollectorPoint{ID: point.ID, GroupID: point.GroupID, Code: point.Code, Name: point.Name, Description: point.Description, Address: point.Address, AddressText: point.AddressText, AddressSchemaVersion: point.AddressSchemaVersion, DataType: point.DataType, ElementCount: point.ElementCount, ReadOptions: point.ReadOptions, Acquisition: point.Acquisition, AcquisitionMode: point.AcquisitionMode, AcquisitionOverrides: point.AcquisitionOverrides, Enabled: point.Enabled, SortOrder: point.SortOrder, Metadata: point.Metadata}
}

var _ io.Reader

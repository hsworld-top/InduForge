package service

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/indu-forge/data_service/internal/collectorprotocol"
	apperrors "github.com/indu-forge/data_service/internal/errors"
)

type ListCollectorDriversInput struct {
	Page           int
	PageSize       int
	Search         string
	ProtocolFamily string
	Category       string
	Transport      string
}

type CollectorDriverSummary struct {
	ProtocolFamily   string              `json:"protocolFamily"`
	DriverID         string              `json:"driverId"`
	DriverVersion    string              `json:"driverVersion"`
	SchemaVersion    int                 `json:"schemaVersion"`
	DisplayName      string              `json:"displayName"`
	Category         string              `json:"category"`
	Transports       []string            `json:"transports"`
	Operations       []string            `json:"operations"`
	DataTypes        []string            `json:"dataTypes"`
	AcquisitionModes []string            `json:"acquisitionModes"`
	Platforms        map[string][]string `json:"platforms"`
}

type CollectorDriverDetail struct {
	CollectorDriverSummary
	ConnectionSchema json.RawMessage `json:"connectionSchema"`
	AddressSchema    json.RawMessage `json:"addressSchema"`
	UISchema         json.RawMessage `json:"uiSchema"`
}

type CollectorDriverPage struct {
	List       []CollectorDriverSummary  `json:"list"`
	Pagination CollectorDriverPagination `json:"pagination"`
}

type CollectorDriverPagination struct {
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

type CollectorCatalogService struct {
	catalog *collectorprotocol.Catalog
}

func NewCollectorCatalogService(catalog *collectorprotocol.Catalog) *CollectorCatalogService {
	return &CollectorCatalogService{catalog: catalog}
}

func (s *CollectorCatalogService) List(input ListCollectorDriversInput) (CollectorDriverPage, error) {
	if input.Page < 1 || input.PageSize < 1 || input.PageSize > 100 {
		return CollectorDriverPage{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "分页参数无效")
	}
	filtered := make([]CollectorDriverSummary, 0)
	for _, driver := range s.catalog.Drivers() {
		if !matchesCollectorDriver(driver.Manifest, input) {
			continue
		}
		filtered = append(filtered, collectorDriverSummary(driver.Manifest))
	}
	total := len(filtered)
	start := (input.Page - 1) * input.PageSize
	if start > total {
		start = total
	}
	end := start + input.PageSize
	if end > total {
		end = total
	}
	totalPages := 0
	if total > 0 {
		totalPages = (total + input.PageSize - 1) / input.PageSize
	}
	return CollectorDriverPage{
		List:       append([]CollectorDriverSummary(nil), filtered[start:end]...),
		Pagination: CollectorDriverPagination{Page: input.Page, PageSize: input.PageSize, Total: total, TotalPages: totalPages},
	}, nil
}

func (s *CollectorCatalogService) Get(driverID string) (CollectorDriverDetail, error) {
	driverID = strings.TrimSpace(driverID)
	driver, ok := s.catalog.Driver(driverID)
	if !ok {
		return CollectorDriverDetail{}, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "工业采集驱动不存在")
	}
	return CollectorDriverDetail{
		CollectorDriverSummary: collectorDriverSummary(driver.Manifest),
		ConnectionSchema:       append(json.RawMessage(nil), driver.ConnectionSchema...),
		AddressSchema:          append(json.RawMessage(nil), driver.AddressSchema...),
		UISchema:               append(json.RawMessage(nil), driver.UISchema...),
	}, nil
}

func matchesCollectorDriver(manifest collectorprotocol.Manifest, input ListCollectorDriversInput) bool {
	if value := strings.TrimSpace(input.ProtocolFamily); value != "" && !strings.EqualFold(manifest.ProtocolFamily, value) {
		return false
	}
	if value := strings.TrimSpace(input.Category); value != "" && !strings.EqualFold(manifest.Category, value) {
		return false
	}
	if value := strings.TrimSpace(input.Transport); value != "" && !containsFold(manifest.Transports, value) {
		return false
	}
	if search := strings.ToLower(strings.TrimSpace(input.Search)); search != "" {
		haystack := strings.ToLower(strings.Join([]string{manifest.DriverID, manifest.DisplayName, manifest.ProtocolFamily, manifest.Category}, " "))
		return strings.Contains(haystack, search)
	}
	return true
}

func containsFold(values []string, expected string) bool {
	for _, value := range values {
		if strings.EqualFold(value, expected) {
			return true
		}
	}
	return false
}

func collectorDriverSummary(manifest collectorprotocol.Manifest) CollectorDriverSummary {
	return CollectorDriverSummary{
		ProtocolFamily: manifest.ProtocolFamily, DriverID: manifest.DriverID, DriverVersion: manifest.DriverVersion,
		SchemaVersion: manifest.SchemaVersion, DisplayName: manifest.DisplayName, Category: manifest.Category,
		Transports: append([]string(nil), manifest.Transports...), Operations: append([]string(nil), manifest.Operations...),
		DataTypes: append([]string(nil), manifest.DataTypes...), AcquisitionModes: append([]string(nil), manifest.AcquisitionModes...),
		Platforms: manifest.Platforms,
	}
}

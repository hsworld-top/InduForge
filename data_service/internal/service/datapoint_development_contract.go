package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/indu-forge/data_service/internal/repository"
)

// DataPointDevelopmentMethod 是代码工作区可读取的方法契约，不包含数据源私有配置。
type DataPointDevelopmentMethod struct {
	Name        string         `json:"name"`
	Parameters  map[string]any `json:"parameters"`
	Result      map[string]any `json:"result,omitempty"`
	Event       map[string]any `json:"event,omitempty"`
	Description string         `json:"description"`
}

// DataPointDevelopmentContract 是单个数据点的稳定开发视图。
// ID 使用工程内唯一的 path，运行时 SDK 也以该标识定位数据点。
type DataPointDevelopmentContract struct {
	ID          string                       `json:"id"`
	Name        string                       `json:"name"`
	Description string                       `json:"description,omitempty"`
	DataType    string                       `json:"dataType"`
	Status      string                       `json:"status"`
	Methods     []DataPointDevelopmentMethod `json:"methods"`
	UpdatedAt   time.Time                    `json:"updatedAt"`
}

// DataPointDevelopmentContractSnapshot 是工程全部数据点的只读开发快照。
type DataPointDevelopmentContractSnapshot struct {
	ProjectID       string                         `json:"projectId"`
	ContractVersion string                         `json:"contractVersion"`
	GeneratedAt     time.Time                      `json:"generatedAt"`
	DataPoints      []DataPointDevelopmentContract `json:"datapoints"`
}

// GetDevelopmentContract 返回完整工程数据点开发契约。
// 该接口专供控制面生成上下文包；它不返回 SQL、连接参数、实时值或内部主键。
func (s *DataPointService) GetDevelopmentContract(ctx context.Context, projectID string) (*DataPointDevelopmentContractSnapshot, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if s == nil || s.repository == nil {
		return nil, fmt.Errorf("数据点仓储未初始化")
	}

	records, err := s.repository.ListAllByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if err := s.refreshDataPointValidity(ctx, projectID, records); err != nil {
		return nil, err
	}

	contracts := make([]DataPointDevelopmentContract, 0, len(records))
	for _, record := range records {
		contracts = append(contracts, buildDataPointDevelopmentContract(record))
	}
	sort.Slice(contracts, func(left, right int) bool { return contracts[left].ID < contracts[right].ID })

	return &DataPointDevelopmentContractSnapshot{
		ProjectID:       projectID,
		ContractVersion: developmentContractVersion(contracts),
		GeneratedAt:     time.Now().UTC(),
		DataPoints:      contracts,
	}, nil
}

func buildDataPointDevelopmentContract(record repository.DataPointRecord) DataPointDevelopmentContract {
	description := ""
	if record.Description != nil {
		description = *record.Description
	}
	valueSchema := map[string]any{"type": record.DataType}
	methods := []DataPointDevelopmentMethod{
		{
			Name:        "get",
			Parameters:  map[string]any{"type": "object", "additionalProperties": true},
			Result:      valueSchema,
			Description: "读取数据点当前值或参数化查询结果。",
		},
		{
			Name:        "set",
			Parameters:  map[string]any{"type": "object", "required": []string{"value"}, "properties": map[string]any{"value": valueSchema}},
			Result:      valueSchema,
			Description: "写入数据点值；运行时会继续执行权限校验。",
		},
	}
	if record.RefreshMode == "subscription" {
		methods = append(methods, DataPointDevelopmentMethod{
			Name:        "sub",
			Parameters:  map[string]any{"type": "object", "additionalProperties": true},
			Event:       valueSchema,
			Description: "订阅数据点更新事件。",
		})
	}

	return DataPointDevelopmentContract{
		ID:          record.Path,
		Name:        record.Name,
		Description: description,
		DataType:    record.DataType,
		Status:      record.Status,
		Methods:     methods,
		UpdatedAt:   record.UpdatedAt.UTC(),
	}
}

func developmentContractVersion(contracts []DataPointDevelopmentContract) string {
	encoded, err := json.Marshal(contracts)
	if err != nil {
		// contracts 只由内部受控字段构造；保留兜底以满足后续字段扩展时的错误边界。
		encoded = []byte("[]")
	}
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
}

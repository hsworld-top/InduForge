package service

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/indu-forge/data_service/internal/collectorprotocol"
	"github.com/indu-forge/data_service/internal/repository"
)

func TestToCollectorPointMapsLatestDebugSnapshot(t *testing.T) {
	valueText := "12.5"
	dataType := "float64"
	quality := "Good"
	errorMessage := "上次读取失败"
	readAt := time.Date(2026, 7, 20, 10, 0, 0, 0, time.Local)
	attemptAt := readAt.Add(time.Minute)
	point := toCollectorPoint(repository.CollectorPointRecord{
		ID: "point-1", Address: map[string]any{}, ReadOptions: map[string]any{}, Acquisition: map[string]any{}, Metadata: map[string]any{},
		LatestDebugSnapshot: &repository.CollectorPointDebugSnapshotRecord{
			Value: 12.5, ValueText: &valueText, DataType: &dataType, Quality: &quality, ReadAt: &readAt,
			LastAttemptStatus: "failed", LastAttemptAt: attemptAt, LastErrorMessage: &errorMessage,
		},
	})
	if point.LatestDebugSnapshot == nil {
		t.Fatal("expected latest debug snapshot")
	}
	if point.LatestDebugSnapshot.Quality == nil || *point.LatestDebugSnapshot.Quality != "Good" {
		t.Fatalf("unexpected snapshot: %#v", point.LatestDebugSnapshot)
	}
	if point.LatestDebugSnapshot.LastAttemptAt != "2026-07-20 10:01:00" {
		t.Fatalf("lastAttemptAt = %s", point.LatestDebugSnapshot.LastAttemptAt)
	}
}

func TestCollectorPointServiceCreatesOpcUaAddressText(t *testing.T) {
	store := &fakeCollectorPointStore{connection: repository.CollectorConnectionRecord{ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000", DriverID: "opcua.standard", SchemaVersion: 1}}
	service := newRepositoryCollectorPointService(t, store)
	_, err := service.CreatePointsBatch(context.Background(), "550e8400-e29b-41d4-a716-446655440000", store.connection.ID, "550e8400-e29b-41d4-a716-446655440001", []CreateCollectorPointInput{{Code: "temperature", Name: "温度", Address: map[string]any{"nodeId": "ns=2;s=Temperature"}, DataType: "float32", ElementCount: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if len(store.created) != 1 || store.created[0].AddressText != "ns=2;s=Temperature" {
		t.Fatalf("unexpected params: %#v", store.created)
	}
}

func TestCollectorPointServiceValidatesModbusAddressAndDataTypeCombinations(t *testing.T) {
	tests := []struct {
		name         string
		driverID     string
		address      map[string]any
		dataType     string
		elementCount int
		wantFailure  string
	}{
		{name: "coil rejects numeric type", address: map[string]any{"station": 1, "area": "coil", "address": 10}, dataType: "int16", elementCount: 1, wantFailure: "线圈和离散输入只支持 bool 数据类型"},
		{name: "coil rejects bit index", address: map[string]any{"station": 1, "area": "coil", "address": 10, "bitIndex": 1}, dataType: "bool", elementCount: 1, wantFailure: "线圈和离散输入不能配置寄存器位索引"},
		{name: "register bool requires bit index", address: map[string]any{"station": 1, "area": "holdingRegister", "address": 10}, dataType: "bool", elementCount: 1, wantFailure: "寄存器 bool 变量必须配置位索引"},
		{name: "register numeric rejects bit index", address: map[string]any{"station": 1, "area": "holdingRegister", "address": 10, "bitIndex": 1}, dataType: "int16", elementCount: 1, wantFailure: "只有寄存器 bool 变量可以配置位索引"},
		{name: "register rejects oversized value", address: map[string]any{"station": 1, "area": "holdingRegister", "address": 10}, dataType: "int32", elementCount: 63, wantFailure: "Modbus 单变量读取长度超过协议上限"},
		{name: "register bit rejects oversized value", address: map[string]any{"station": 1, "area": "holdingRegister", "address": 10, "bitIndex": 15}, dataType: "bool", elementCount: 1986, wantFailure: "Modbus 单变量读取长度超过协议上限"},
		{name: "holding register accepts numeric type", address: map[string]any{"station": 1, "area": "holdingRegister", "address": 10}, dataType: "int16", elementCount: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeCollectorPointStore{connection: repository.CollectorConnectionRecord{
				ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000",
				DriverID: "modbus.tcp", DriverVersion: "1.0.0", SchemaVersion: 1,
			}}
			service := newRepositoryCollectorPointService(t, store)
			result, err := service.CreatePointsBatch(
				context.Background(),
				store.connection.ProjectID,
				store.connection.ID,
				"550e8400-e29b-41d4-a716-446655440001",
				[]CreateCollectorPointInput{{Name: "测试变量", Address: tt.address, DataType: tt.dataType, ElementCount: tt.elementCount}},
			)
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantFailure == "" {
				if len(result.List) != 1 || len(result.Failed) != 0 {
					t.Fatalf("合法变量未创建: %+v", result)
				}
				return
			}
			if len(result.List) != 0 || len(result.Failed) != 1 || !strings.Contains(result.Failed[0].Message, tt.wantFailure) {
				t.Fatalf("非法组合未返回预期错误 %q: %+v", tt.wantFailure, result)
			}
		})
	}
}

func TestCollectorPointServiceValidatesMitsubishiMcAddressAndReadLength(t *testing.T) {
	tests := []struct {
		name         string
		address      map[string]any
		dataType     string
		elementCount int
		wantText     string
		wantFailure  string
	}{
		{name: "word address", address: map[string]any{"address": "D100"}, dataType: "int16", elementCount: 1, wantText: "D100"},
		{name: "bit address", address: map[string]any{"address": "M0"}, dataType: "bool", elementCount: 1, wantText: "M0"},
		{name: "trim address", address: map[string]any{"address": "  D100  "}, dataType: "int16", elementCount: 1, wantText: "D100"},
		{name: "empty address", address: map[string]any{"address": "   "}, dataType: "int16", elementCount: 1, wantFailure: "Mitsubishi MC 设备地址无效"},
		{name: "address contains whitespace", address: map[string]any{"address": "D 100"}, dataType: "int16", elementCount: 1, wantFailure: "Mitsubishi MC 设备地址无效"},
		{name: "float64 exceeds word limit", address: map[string]any{"address": "D100"}, dataType: "float64", elementCount: 241, wantFailure: "Mitsubishi MC 单变量读取长度超过协议上限 960"},
		{name: "bool exceeds bit limit", address: map[string]any{"address": "M0"}, dataType: "bool", elementCount: 7169, wantFailure: "Mitsubishi MC 单变量读取长度超过协议上限 7168"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeCollectorPointStore{connection: repository.CollectorConnectionRecord{
				ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000",
				DriverID: "mitsubishi.mc-3e-tcp", DriverVersion: "1.0.0", SchemaVersion: 1,
			}}
			service := newRepositoryCollectorPointService(t, store)
			result, err := service.CreatePointsBatch(
				context.Background(),
				store.connection.ProjectID,
				store.connection.ID,
				"550e8400-e29b-41d4-a716-446655440001",
				[]CreateCollectorPointInput{{Name: "测试变量", Address: tt.address, DataType: tt.dataType, ElementCount: tt.elementCount}},
			)
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantFailure == "" {
				if len(result.List) != 1 || len(result.Failed) != 0 || result.List[0].AddressText != tt.wantText {
					t.Fatalf("合法变量未按预期创建: %+v", result)
				}
				return
			}
			if len(result.List) != 0 || len(result.Failed) != 1 || !strings.Contains(result.Failed[0].Message, tt.wantFailure) {
				t.Fatalf("非法地址未返回预期错误 %q: %+v", tt.wantFailure, result)
			}
		})
	}
}

func TestCollectorPointServiceValidatesMitsubishiNetworkVariantAddresses(t *testing.T) {
	drivers := []string{
		"mitsubishi.a1e-ascii-tcp",
		"mitsubishi.a1e-binary-tcp",
		"mitsubishi.mc-ascii-tcp",
		"mitsubishi.mc-ascii-udp",
		"mitsubishi.mc-binary-udp",
		"mitsubishi.mc-r-binary-tcp",
		"mitsubishi.a3c-serial",
		"mitsubishi.a3c-serial-over-tcp",
		"mitsubishi.fx-links-serial",
		"mitsubishi.fx-links-over-tcp",
		"mitsubishi.fx-serial",
		"mitsubishi.fx-serial-over-tcp",
	}

	for _, driverID := range drivers {
		t.Run(driverID, func(t *testing.T) {
			store := &fakeCollectorPointStore{connection: repository.CollectorConnectionRecord{
				ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000",
				DriverID: driverID, DriverVersion: "1.0.0", SchemaVersion: 1,
			}}
			service := newRepositoryCollectorPointService(t, store)
			result, err := service.CreatePointsBatch(
				context.Background(),
				store.connection.ProjectID,
				store.connection.ID,
				"550e8400-e29b-41d4-a716-446655440001",
				[]CreateCollectorPointInput{{Name: "测试变量", Address: map[string]any{"address": "  D100  "}, DataType: "int16", ElementCount: 1}},
			)
			if err != nil {
				t.Fatal(err)
			}
			if len(result.List) != 1 || len(result.Failed) != 0 || result.List[0].AddressText != "D100" {
				t.Fatalf("三菱网络原生地址未按预期创建: %+v", result)
			}
		})
	}
}

func TestCollectorPointServiceValidatesMitsubishiCipAddress(t *testing.T) {
	tests := []struct {
		name         string
		address      string
		dataType     string
		elementCount int
		wantText     string
		wantFailure  string
	}{
		{name: "preserve tag case", address: "  PumpA.Speed  ", dataType: "float32", elementCount: 1, wantText: "PumpA.Speed"},
		{name: "address contains whitespace", address: "PumpA. Speed", dataType: "float32", elementCount: 1, wantFailure: "Mitsubishi CIP 标签地址无效"},
		{name: "datetime array", address: "ClockTag", dataType: "datetime", elementCount: 2, wantFailure: "datetime 标签只支持单元素读取"},
		{name: "element count exceeds limit", address: "ArrayTag", dataType: "int16", elementCount: 65536, wantFailure: "元素数量必须在 1 到 65535 之间"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeCollectorPointStore{connection: repository.CollectorConnectionRecord{
				ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000",
				DriverID: "mitsubishi.cip", DriverVersion: "1.0.0", SchemaVersion: 1,
			}}
			service := newRepositoryCollectorPointService(t, store)
			result, err := service.CreatePointsBatch(
				context.Background(),
				store.connection.ProjectID,
				store.connection.ID,
				"550e8400-e29b-41d4-a716-446655440001",
				[]CreateCollectorPointInput{{Name: "测试变量", Address: map[string]any{"address": tt.address}, DataType: tt.dataType, ElementCount: tt.elementCount}},
			)
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantFailure == "" {
				if len(result.List) != 1 || len(result.Failed) != 0 || result.List[0].AddressText != tt.wantText {
					t.Fatalf("合法三菱 CIP 标签未按预期创建: %+v", result)
				}
				return
			}
			if len(result.List) != 0 || len(result.Failed) != 1 || !strings.Contains(result.Failed[0].Message, tt.wantFailure) {
				t.Fatalf("非法三菱 CIP 标签未返回预期错误 %q: %+v", tt.wantFailure, result)
			}
		})
	}
}

func TestCollectorPointServiceValidatesOmronFinsAddressAndReadLength(t *testing.T) {
	tests := []struct {
		name         string
		driverID     string
		address      map[string]any
		dataType     string
		elementCount int
		wantText     string
		wantFailure  string
	}{
		{name: "tcp word address", driverID: "omron.fins-tcp", address: map[string]any{"address": "D100"}, dataType: "int16", elementCount: 1, wantText: "D100"},
		{name: "udp bit address", driverID: "omron.fins-udp", address: map[string]any{"address": "D100.0"}, dataType: "bool", elementCount: 1, wantText: "D100.0"},
		{name: "trim address", driverID: "omron.fins-tcp", address: map[string]any{"address": "  E0.100  "}, dataType: "float32", elementCount: 1, wantText: "E0.100"},
		{name: "empty address", driverID: "omron.fins-tcp", address: map[string]any{"address": "   "}, dataType: "int16", elementCount: 1, wantFailure: "Omron FINS 设备地址无效"},
		{name: "address contains whitespace", driverID: "omron.fins-udp", address: map[string]any{"address": "D 100"}, dataType: "int16", elementCount: 1, wantFailure: "Omron FINS 设备地址无效"},
		{name: "float64 exceeds limit", driverID: "omron.fins-tcp", address: map[string]any{"address": "D100"}, dataType: "float64", elementCount: 16384, wantFailure: "Omron FINS 单变量读取长度超过平台上限 65535"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeCollectorPointStore{connection: repository.CollectorConnectionRecord{
				ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000",
				DriverID: tt.driverID, DriverVersion: "1.0.0", SchemaVersion: 1,
			}}
			service := newRepositoryCollectorPointService(t, store)
			result, err := service.CreatePointsBatch(
				context.Background(),
				store.connection.ProjectID,
				store.connection.ID,
				"550e8400-e29b-41d4-a716-446655440001",
				[]CreateCollectorPointInput{{Name: "测试变量", Address: tt.address, DataType: tt.dataType, ElementCount: tt.elementCount}},
			)
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantFailure == "" {
				if len(result.List) != 1 || len(result.Failed) != 0 || result.List[0].AddressText != tt.wantText {
					t.Fatalf("合法变量未按预期创建: %+v", result)
				}
				return
			}
			if len(result.List) != 0 || len(result.Failed) != 1 || !strings.Contains(result.Failed[0].Message, tt.wantFailure) {
				t.Fatalf("非法地址未返回预期错误 %q: %+v", tt.wantFailure, result)
			}
		})
	}
}

func TestCollectorPointServiceValidatesOmronVariantAddresses(t *testing.T) {
	tests := []struct {
		name         string
		driverID     string
		address      string
		dataType     string
		elementCount int
		wantText     string
		wantFailure  string
	}{
		{name: "cip tag", driverID: "omron.cip", address: "Program:MainProgram.A1", dataType: "int16", elementCount: 1, wantText: "Program:MainProgram.A1"},
		{name: "connected cip tag", driverID: "omron.connected-cip", address: "  C[0,1]  ", dataType: "float32", elementCount: 2, wantText: "C[0,1]"},
		{name: "hostlink serial", driverID: "omron.hostlink", address: "D100", dataType: "int16", elementCount: 1, wantText: "D100"},
		{name: "hostlink tcp", driverID: "omron.hostlink-over-tcp", address: "CIO100.1", dataType: "bool", elementCount: 1, wantText: "CIO100.1"},
		{name: "cmode serial", driverID: "omron.hostlink-cmode", address: "H5.2", dataType: "bool", elementCount: 1, wantText: "H5.2"},
		{name: "cmode tcp", driverID: "omron.hostlink-cmode-over-tcp", address: "E0.100.2", dataType: "int16", elementCount: 1, wantText: "E0.100.2"},
		{name: "cip whitespace", driverID: "omron.cip", address: "Bad Tag", dataType: "int16", elementCount: 1, wantFailure: "Omron CIP 标签地址无效"},
		{name: "invalid cmode", driverID: "omron.hostlink-cmode", address: "DB100", dataType: "int16", elementCount: 1, wantFailure: "Omron HostLink C-Mode 设备地址无效"},
		{name: "oversized array", driverID: "omron.connected-cip", address: "A1", dataType: "int16", elementCount: 65536, wantFailure: "Omron 元素数量不能超过 65535"},
		{name: "datetime array", driverID: "omron.cip", address: "Timestamp", dataType: "datetime", elementCount: 2, wantFailure: "Omron datetime 标签只支持单元素读取"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeCollectorPointStore{connection: repository.CollectorConnectionRecord{
				ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000",
				DriverID: tt.driverID, DriverVersion: "1.0.0", SchemaVersion: 1,
			}}
			service := newRepositoryCollectorPointService(t, store)
			result, err := service.CreatePointsBatch(
				context.Background(), store.connection.ProjectID, store.connection.ID,
				"550e8400-e29b-41d4-a716-446655440001",
				[]CreateCollectorPointInput{{Name: "测试变量", Address: map[string]any{"address": tt.address}, DataType: tt.dataType, ElementCount: tt.elementCount}},
			)
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantFailure == "" {
				if len(result.List) != 1 || len(result.Failed) != 0 || result.List[0].AddressText != tt.wantText {
					t.Fatalf("合法变量未按预期创建: %+v", result)
				}
				return
			}
			if len(result.List) != 0 || len(result.Failed) != 1 || !strings.Contains(result.Failed[0].Message, tt.wantFailure) {
				t.Fatalf("非法地址未返回预期错误 %q: %+v", tt.wantFailure, result)
			}
		})
	}
}

func TestCollectorPointServiceValidatesAllenBradleyTagAddress(t *testing.T) {
	tests := []struct {
		name         string
		address      map[string]any
		dataType     string
		elementCount int
		wantText     string
		wantFailure  string
	}{
		{name: "global tag", address: map[string]any{"address": "Temperature"}, dataType: "float32", elementCount: 1, wantText: "Temperature"},
		{name: "program tag", address: map[string]any{"address": "Program:MainProgram.LocalTag"}, dataType: "bool", elementCount: 1, wantText: "Program:MainProgram.LocalTag"},
		{name: "trim tag", address: map[string]any{"address": "  ArrayTag[10]  "}, dataType: "int32", elementCount: 1, wantText: "ArrayTag[10]"},
		{name: "tag contains whitespace", address: map[string]any{"address": "Bad Tag"}, dataType: "int16", elementCount: 1, wantFailure: "Allen-Bradley 标签地址无效"},
		{name: "array exceeds limit", address: map[string]any{"address": "ArrayTag"}, dataType: "int16", elementCount: 65536, wantFailure: "Allen-Bradley 元素数量不能超过 65535"},
		{name: "datetime array", address: map[string]any{"address": "Timestamp"}, dataType: "datetime", elementCount: 2, wantFailure: "Allen-Bradley datetime 标签只支持单元素读取"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeCollectorPointStore{connection: repository.CollectorConnectionRecord{
				ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000",
				DriverID: "allen-bradley.ethernet-ip", DriverVersion: "1.0.0", SchemaVersion: 1,
			}}
			service := newRepositoryCollectorPointService(t, store)
			result, err := service.CreatePointsBatch(
				context.Background(),
				store.connection.ProjectID,
				store.connection.ID,
				"550e8400-e29b-41d4-a716-446655440001",
				[]CreateCollectorPointInput{{Name: "测试变量", Address: tt.address, DataType: tt.dataType, ElementCount: tt.elementCount}},
			)
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantFailure == "" {
				if len(result.List) != 1 || len(result.Failed) != 0 || result.List[0].AddressText != tt.wantText {
					t.Fatalf("合法变量未按预期创建: %+v", result)
				}
				return
			}
			if len(result.List) != 0 || len(result.Failed) != 1 || !strings.Contains(result.Failed[0].Message, tt.wantFailure) {
				t.Fatalf("非法标签未返回预期错误 %q: %+v", tt.wantFailure, result)
			}
		})
	}
}

func TestCollectorPointServiceValidatesBeckhoffAdsAddress(t *testing.T) {
	tests := []struct {
		name         string
		address      map[string]any
		dataType     string
		elementCount int
		wantText     string
		wantFailure  string
	}{
		{name: "symbol tag", address: map[string]any{"address": "s=MAIN.temperature"}, dataType: "float32", elementCount: 1, wantText: "s=MAIN.temperature"},
		{name: "memory bit", address: map[string]any{"address": "  M100.0  "}, dataType: "bool", elementCount: 1, wantText: "M100.0"},
		{name: "index group", address: map[string]any{"address": "IG=0xF020;0"}, dataType: "uint16", elementCount: 2, wantText: "IG=0xF020;0"},
		{name: "address contains whitespace", address: map[string]any{"address": "s=MAIN. value"}, dataType: "float32", elementCount: 1, wantFailure: "倍福 ADS 地址无效"},
		{name: "array exceeds limit", address: map[string]any{"address": "M100"}, dataType: "int16", elementCount: 65536, wantFailure: "倍福 ADS 元素数量不能超过 65535"},
		{name: "datetime unsupported", address: map[string]any{"address": "s=MAIN.timestamp"}, dataType: "datetime", elementCount: 1, wantFailure: "驱动不支持该平台数据类型"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeCollectorPointStore{connection: repository.CollectorConnectionRecord{
				ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000",
				DriverID: "beckhoff.ads-tcp", DriverVersion: "1.0.0", SchemaVersion: 1,
			}}
			service := newRepositoryCollectorPointService(t, store)
			result, err := service.CreatePointsBatch(
				context.Background(),
				store.connection.ProjectID,
				store.connection.ID,
				"550e8400-e29b-41d4-a716-446655440001",
				[]CreateCollectorPointInput{{Name: "测试变量", Address: tt.address, DataType: tt.dataType, ElementCount: tt.elementCount}},
			)
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantFailure == "" {
				if len(result.List) != 1 || len(result.Failed) != 0 || result.List[0].AddressText != tt.wantText {
					t.Fatalf("合法变量未按预期创建: %+v", result)
				}
				return
			}
			if len(result.List) != 0 || len(result.Failed) != 1 || !strings.Contains(result.Failed[0].Message, tt.wantFailure) {
				t.Fatalf("非法地址未返回预期错误 %q: %+v", tt.wantFailure, result)
			}
		})
	}
}

func TestCollectorPointServiceValidatesIec104Address(t *testing.T) {
	tests := []struct {
		name         string
		address      map[string]any
		dataType     string
		elementCount int
		wantText     string
		wantFailure  string
	}{
		{name: "single point", address: map[string]any{"informationType": "singlePoint", "informationObjectAddress": 1}, dataType: "bool", elementCount: 1, wantText: "singlePoint:1"},
		{name: "short float", address: map[string]any{"informationType": "shortFloatMeasured", "informationObjectAddress": 100}, dataType: "float32", elementCount: 1, wantText: "shortFloatMeasured:100"},
		{name: "mismatched type", address: map[string]any{"informationType": "doublePoint", "informationObjectAddress": 2}, dataType: "bool", elementCount: 1, wantFailure: "doublePoint 信息类型必须使用 uint8 数据类型"},
		{name: "array unsupported", address: map[string]any{"informationType": "scaledMeasured", "informationObjectAddress": 3}, dataType: "int16", elementCount: 2, wantFailure: "元素数量必须为 1"},
		{name: "ioa exceeds limit", address: map[string]any{"informationType": "integratedTotal", "informationObjectAddress": 16777216}, dataType: "uint32", elementCount: 1, wantFailure: "采集点地址不符合驱动 Schema"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeCollectorPointStore{connection: repository.CollectorConnectionRecord{
				ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000",
				DriverID: "iec.60870-5-104", DriverVersion: "1.0.0", SchemaVersion: 1,
			}}
			service := newRepositoryCollectorPointService(t, store)
			result, err := service.CreatePointsBatch(
				context.Background(),
				store.connection.ProjectID,
				store.connection.ID,
				"550e8400-e29b-41d4-a716-446655440001",
				[]CreateCollectorPointInput{{Name: "测试变量", Address: tt.address, DataType: tt.dataType, ElementCount: tt.elementCount}},
			)
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantFailure == "" {
				if len(result.List) != 1 || len(result.Failed) != 0 || result.List[0].AddressText != tt.wantText {
					t.Fatalf("合法变量未按预期创建: %+v", result)
				}
				return
			}
			if len(result.List) != 0 || len(result.Failed) != 1 || !strings.Contains(result.Failed[0].Message, tt.wantFailure) {
				t.Fatalf("非法地址未返回预期错误 %q: %+v", tt.wantFailure, result)
			}
		})
	}
}

func TestCollectorPointServiceValidatesLsisFastEnetAddress(t *testing.T) {
	tests := []struct {
		name         string
		address      map[string]any
		elementCount int
		wantText     string
		wantFailure  string
	}{
		{name: "word address", address: map[string]any{"address": "  MB100  "}, elementCount: 1, wantText: "MB100"},
		{name: "bit address", address: map[string]any{"address": "IX0.0.0"}, elementCount: 1, wantText: "IX0.0.0"},
		{name: "address contains whitespace", address: map[string]any{"address": "MB 100"}, elementCount: 1, wantFailure: "LSIS 设备地址无效"},
		{name: "array exceeds limit", address: map[string]any{"address": "MB100"}, elementCount: 65536, wantFailure: "LSIS 元素数量不能超过 65535"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeCollectorPointStore{connection: repository.CollectorConnectionRecord{
				ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000",
				DriverID: "lsis.fast-enet", DriverVersion: "1.0.0", SchemaVersion: 1,
			}}
			service := newRepositoryCollectorPointService(t, store)
			result, err := service.CreatePointsBatch(
				context.Background(),
				store.connection.ProjectID,
				store.connection.ID,
				"550e8400-e29b-41d4-a716-446655440001",
				[]CreateCollectorPointInput{{Name: "测试变量", Address: tt.address, DataType: "int16", ElementCount: tt.elementCount}},
			)
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantFailure == "" {
				if len(result.List) != 1 || len(result.Failed) != 0 || result.List[0].AddressText != tt.wantText {
					t.Fatalf("合法变量未按预期创建: %+v", result)
				}
				return
			}
			if len(result.List) != 0 || len(result.Failed) != 1 || !strings.Contains(result.Failed[0].Message, tt.wantFailure) {
				t.Fatalf("非法地址未返回预期错误 %q: %+v", tt.wantFailure, result)
			}
		})
	}
}

func TestCollectorPointServiceValidatesLsisSerialVariantAddresses(t *testing.T) {
	for _, driverID := range []string{"lsis.cnet", "lsis.cnet-over-tcp", "lsis.cpu-serial"} {
		t.Run(driverID, func(t *testing.T) {
			store := &fakeCollectorPointStore{connection: repository.CollectorConnectionRecord{
				ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000",
				DriverID: driverID, DriverVersion: "1.0.0", SchemaVersion: 1,
			}}
			service := newRepositoryCollectorPointService(t, store)
			result, err := service.CreatePointsBatch(context.Background(), store.connection.ProjectID, store.connection.ID,
				"550e8400-e29b-41d4-a716-446655440001", []CreateCollectorPointInput{{Name: "测试变量", Address: map[string]any{"address": "  D100  "}, DataType: "int16", ElementCount: 1}})
			if err != nil {
				t.Fatal(err)
			}
			if len(result.List) != 1 || len(result.Failed) != 0 || result.List[0].AddressText != "D100" {
				t.Fatalf("LSIS 变量未按预期创建: %+v", result)
			}
		})
	}
}

func TestCollectorPointServiceValidatesGeSrtpAddress(t *testing.T) {
	tests := []struct {
		name         string
		address      map[string]any
		dataType     string
		elementCount int
		wantText     string
		wantFailure  string
	}{
		{name: "word register", address: map[string]any{"address": " r001 "}, dataType: "int16", elementCount: 1, wantText: "R1"},
		{name: "bit area", address: map[string]any{"address": "m1"}, dataType: "bool", elementCount: 1, wantText: "M1"},
		{name: "bool word area", address: map[string]any{"address": "AI1"}, dataType: "bool", elementCount: 1, wantFailure: "字寄存器不支持 bool"},
		{name: "unsupported area", address: map[string]any{"address": "D100"}, dataType: "int16", elementCount: 1, wantFailure: "GE SRTP 设备地址无效"},
		{name: "array exceeds limit", address: map[string]any{"address": "R1"}, dataType: "int16", elementCount: 65536, wantFailure: "GE SRTP 元素数量不能超过 65535"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeCollectorPointStore{connection: repository.CollectorConnectionRecord{
				ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000",
				DriverID: "ge.srtp-tcp", DriverVersion: "1.0.0", SchemaVersion: 1,
			}}
			service := newRepositoryCollectorPointService(t, store)
			result, err := service.CreatePointsBatch(
				context.Background(),
				store.connection.ProjectID,
				store.connection.ID,
				"550e8400-e29b-41d4-a716-446655440001",
				[]CreateCollectorPointInput{{Name: "测试变量", Address: tt.address, DataType: tt.dataType, ElementCount: tt.elementCount}},
			)
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantFailure == "" {
				if len(result.List) != 1 || len(result.Failed) != 0 || result.List[0].AddressText != tt.wantText {
					t.Fatalf("合法变量未按预期创建: %+v", result)
				}
				return
			}
			if len(result.List) != 0 || len(result.Failed) != 1 || !strings.Contains(result.Failed[0].Message, tt.wantFailure) {
				t.Fatalf("非法地址未返回预期错误 %q: %+v", tt.wantFailure, result)
			}
		})
	}
}

func TestCollectorPointServiceValidatesInovanceAddress(t *testing.T) {
	tests := []struct {
		name         string
		driverID     string
		address      map[string]any
		dataType     string
		elementCount int
		wantText     string
		wantFailure  string
	}{
		{name: "native address", driverID: "inovance.modbus-tcp", address: map[string]any{"address": " d0 "}, dataType: "int16", elementCount: 1, wantText: "D0"},
		{name: "serial address", driverID: "inovance.modbus-serial", address: map[string]any{"address": "m100"}, dataType: "bool", elementCount: 1, wantText: "M100"},
		{name: "serial over tcp address", driverID: "inovance.modbus-rtu-over-tcp", address: map[string]any{"address": "sd0.5"}, dataType: "bool", elementCount: 1, wantText: "SD0.5"},
		{name: "address contains whitespace", driverID: "inovance.modbus-serial", address: map[string]any{"address": "D 0"}, dataType: "int16", elementCount: 1, wantFailure: "汇川 PLC 设备地址无效"},
		{name: "read exceeds limit", driverID: "inovance.modbus-rtu-over-tcp", address: map[string]any{"address": "D0"}, dataType: "float64", elementCount: 32, wantFailure: "单变量读取长度超过协议上限"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeCollectorPointStore{connection: repository.CollectorConnectionRecord{
				ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000",
				DriverID: tt.driverID, DriverVersion: "1.0.0", SchemaVersion: 1,
			}}
			service := newRepositoryCollectorPointService(t, store)
			result, err := service.CreatePointsBatch(
				context.Background(),
				store.connection.ProjectID,
				store.connection.ID,
				"550e8400-e29b-41d4-a716-446655440001",
				[]CreateCollectorPointInput{{Name: "测试变量", Address: tt.address, DataType: tt.dataType, ElementCount: tt.elementCount}},
			)
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantFailure == "" {
				if len(result.List) != 1 || len(result.Failed) != 0 || result.List[0].AddressText != tt.wantText {
					t.Fatalf("合法变量未按预期创建: %+v", result)
				}
				return
			}
			if len(result.List) != 0 || len(result.Failed) != 1 || !strings.Contains(result.Failed[0].Message, tt.wantFailure) {
				t.Fatalf("非法地址未返回预期错误 %q: %+v", tt.wantFailure, result)
			}
		})
	}
}

func TestCollectorPointServiceValidatesInovanceSpecialAddresses(t *testing.T) {
	tests := []struct {
		name         string
		driverID     string
		address      string
		dataType     string
		elementCount int
		wantText     string
		wantFailure  string
	}{
		{name: "connected cip preserves tag", driverID: "inovance.connected-cip", address: "  Program:Main.A1  ", dataType: "int16", elementCount: 1, wantText: "Program:Main.A1"},
		{name: "easy net address", driverID: "inovance.easy-net", address: "w100", dataType: "int16", elementCount: 1, wantText: "W100"},
		{name: "computer link address", driverID: "inovance.computer-link", address: "d100", dataType: "float32", elementCount: 1, wantText: "D100"},
		{name: "invalid whitespace", driverID: "inovance.easy-net", address: "W 100", dataType: "int16", elementCount: 1, wantFailure: "汇川设备地址无效"},
		{name: "oversized count", driverID: "inovance.computer-link", address: "D100", dataType: "int16", elementCount: 65536, wantFailure: "汇川元素数量不能超过 65535"},
		{name: "datetime array", driverID: "inovance.connected-cip", address: "Timestamp", dataType: "datetime", elementCount: 2, wantFailure: "不支持该 datetime 配置"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeCollectorPointStore{connection: repository.CollectorConnectionRecord{
				ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000",
				DriverID: tt.driverID, DriverVersion: "1.0.0", SchemaVersion: 1,
			}}
			service := newRepositoryCollectorPointService(t, store)
			result, err := service.CreatePointsBatch(context.Background(), store.connection.ProjectID, store.connection.ID,
				"550e8400-e29b-41d4-a716-446655440001",
				[]CreateCollectorPointInput{{Name: "测试变量", Address: map[string]any{"address": tt.address}, DataType: tt.dataType, ElementCount: tt.elementCount}})
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantFailure == "" {
				if len(result.List) != 1 || len(result.Failed) != 0 || result.List[0].AddressText != tt.wantText {
					t.Fatalf("合法变量未按预期创建: %+v", result)
				}
				return
			}
			if len(result.List) != 0 || len(result.Failed) != 1 || !strings.Contains(result.Failed[0].Message, tt.wantFailure) {
				t.Fatalf("非法地址未返回预期错误 %q: %+v", tt.wantFailure, result)
			}
		})
	}
}

func TestCollectorPointServiceValidatesFatekProgramAddress(t *testing.T) {
	tests := []struct {
		name         string
		address      map[string]any
		dataType     string
		elementCount int
		wantText     string
		wantFailure  string
	}{
		{name: "word address", address: map[string]any{"address": " d001 "}, dataType: "int16", elementCount: 1, wantText: "D1"},
		{name: "station override", address: map[string]any{"address": "S=2;d100"}, dataType: "int16", elementCount: 1, wantText: "s=2;D100"},
		{name: "bit address", address: map[string]any{"address": "m0"}, dataType: "bool", elementCount: 1, wantText: "M0"},
		{name: "bool area mismatch", address: map[string]any{"address": "D0"}, dataType: "bool", elementCount: 1, wantFailure: "bool 变量只支持"},
		{name: "word area mismatch", address: map[string]any{"address": "M0"}, dataType: "int16", elementCount: 1, wantFailure: "数值变量只支持"},
		{name: "element count exceeds limit", address: map[string]any{"address": "D0"}, dataType: "int16", elementCount: 65536, wantFailure: "元素数量不能超过 65535"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeCollectorPointStore{connection: repository.CollectorConnectionRecord{
				ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000",
				DriverID: "fatek.program-tcp", DriverVersion: "1.0.0", SchemaVersion: 1,
			}}
			service := newRepositoryCollectorPointService(t, store)
			result, err := service.CreatePointsBatch(context.Background(), store.connection.ProjectID, store.connection.ID,
				"550e8400-e29b-41d4-a716-446655440001",
				[]CreateCollectorPointInput{{Name: "测试变量", Address: tt.address, DataType: tt.dataType, ElementCount: tt.elementCount}})
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantFailure == "" {
				if len(result.List) != 1 || len(result.Failed) != 0 || result.List[0].AddressText != tt.wantText {
					t.Fatalf("合法变量未按预期创建: %+v", result)
				}
				return
			}
			if len(result.List) != 0 || len(result.Failed) != 1 || !strings.Contains(result.Failed[0].Message, tt.wantFailure) {
				t.Fatalf("非法地址未返回预期错误 %q: %+v", tt.wantFailure, result)
			}
		})
	}
}

func TestCollectorPointServiceNormalizesFreedomAndCimonAddresses(t *testing.T) {
	tests := []struct {
		name        string
		driverID    string
		address     map[string]any
		wantText    string
		wantFailure string
	}{
		{name: "freedom request", driverID: "freedom.tcp", address: map[string]any{"requestHex": "00-00 00 00 00 06 01 03", "responseOffset": 6}, wantText: "offset=6;00 00 00 00 00 06 01 03"},
		{name: "freedom invalid request", driverID: "freedom.udp", address: map[string]any{"requestHex": "001"}, wantFailure: "偶数长度"},
		{name: "cimon address", driverID: "cimon.hmi-protocol", address: map[string]any{"address": " d100 "}, wantText: "D100"},
		{name: "cimon invalid whitespace", driverID: "cimon.hmi-protocol", address: map[string]any{"address": "D 100"}, wantFailure: "Cimon HMI 设备地址无效"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeCollectorPointStore{connection: repository.CollectorConnectionRecord{
				ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000",
				DriverID: tt.driverID, DriverVersion: "1.0.0", SchemaVersion: 1,
			}}
			service := newRepositoryCollectorPointService(t, store)
			result, err := service.CreatePointsBatch(context.Background(), store.connection.ProjectID, store.connection.ID,
				"550e8400-e29b-41d4-a716-446655440001",
				[]CreateCollectorPointInput{{Name: "测试变量", Address: tt.address, DataType: "int16", ElementCount: 1}})
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantFailure == "" {
				if len(result.List) != 1 || len(result.Failed) != 0 || result.List[0].AddressText != tt.wantText {
					t.Fatalf("合法变量未按预期创建: %+v", result)
				}
				return
			}
			if len(result.List) != 0 || len(result.Failed) != 1 || !strings.Contains(result.Failed[0].Message, tt.wantFailure) {
				t.Fatalf("非法地址未返回预期错误 %q: %+v", tt.wantFailure, result)
			}
		})
	}
}

func TestCollectorPointServiceNormalizesYamatakeAndYaskawaAddresses(t *testing.T) {
	tests := []struct {
		driverID    string
		address     string
		wantText    string
		wantFailure string
	}{
		{driverID: "yamatake.digitron-tcp", address: "S=02;0501", wantText: "s=2;501"},
		{driverID: "yamatake.digitron-serial", address: "s=256;501", wantFailure: "站号必须在 0 到 255"},
		{driverID: "yaskawa.memobus-tcp", address: " mfc=67;x=61;0 ", wantText: "mfc=67;x=61;0"},
		{driverID: "yaskawa.memobus-udp", address: "M 100", wantFailure: "安川 Memobus 地址无效"},
	}
	for _, tt := range tests {
		t.Run(tt.driverID+tt.address, func(t *testing.T) {
			store := &fakeCollectorPointStore{connection: repository.CollectorConnectionRecord{ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000", DriverID: tt.driverID, DriverVersion: "1.0.0", SchemaVersion: 1}}
			service := newRepositoryCollectorPointService(t, store)
			result, err := service.CreatePointsBatch(context.Background(), store.connection.ProjectID, store.connection.ID, "550e8400-e29b-41d4-a716-446655440001", []CreateCollectorPointInput{{Name: "测试变量", Address: map[string]any{"address": tt.address}, DataType: "int16", ElementCount: 1}})
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantFailure == "" {
				if len(result.List) != 1 || result.List[0].AddressText != tt.wantText {
					t.Fatalf("合法地址未规范化: %+v", result)
				}
				return
			}
			if len(result.Failed) != 1 || !strings.Contains(result.Failed[0].Message, tt.wantFailure) {
				t.Fatalf("非法地址未返回预期错误: %+v", result)
			}
		})
	}
}

func TestCollectorPointServiceNormalizesSpecializedNetworkAddresses(t *testing.T) {
	tests := []struct{ driverID, address, wantText, wantFailure string }{
		{driverID: "oriental-motor.eip", address: "INPUT[048]", wantText: "input[48]"},
		{driverID: "toyo.puc", address: "PRG=01;em0010", wantText: "prg=1;EM10"},
		{driverID: "turck.reader-tcp", address: "000100", wantText: "100"},
		{driverID: "turck.reader-tcp", address: "A100", wantFailure: "非负数字偏移"},
	}
	for _, tt := range tests {
		t.Run(tt.driverID+tt.address, func(t *testing.T) {
			store := &fakeCollectorPointStore{connection: repository.CollectorConnectionRecord{ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000", DriverID: tt.driverID, DriverVersion: "1.0.0", SchemaVersion: 1}}
			service := newRepositoryCollectorPointService(t, store)
			result, err := service.CreatePointsBatch(context.Background(), store.connection.ProjectID, store.connection.ID, "550e8400-e29b-41d4-a716-446655440001", []CreateCollectorPointInput{{Name: "测试变量", Address: map[string]any{"address": tt.address}, DataType: "int16", ElementCount: 1}})
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantFailure == "" {
				if len(result.List) != 1 || result.List[0].AddressText != tt.wantText {
					t.Fatalf("合法地址未规范化: %+v", result)
				}
				return
			}
			if len(result.Failed) != 1 || !strings.Contains(result.Failed[0].Message, tt.wantFailure) {
				t.Fatalf("非法地址未返回预期错误: %+v", result)
			}
		})
	}
}

func TestCollectorPointServiceNormalizesInstrumentMeterAddresses(t *testing.T) {
	tests := []struct {
		driverID    string
		address     string
		dataType    string
		wantText    string
		wantFailure string
	}{
		{driverID: "cjt188.serial", address: "901f", dataType: "float64", wantText: "90-1F"},
		{driverID: "cjt188.tcp", address: "S=123;d1-2a", dataType: "float64", wantText: "s=00000000000123;D1-2A"},
		{driverID: "rkc.temperature-controller-serial", address: "m1", dataType: "float64", wantText: "M1"},
		{driverID: "rkc.temperature-controller-tcp", address: "S=02;pb", dataType: "float64", wantText: "s=2;PB"},
		{driverID: "rkc.temperature-controller-tcp", address: "BAD", dataType: "float64", wantFailure: "两位代码"},
		{driverID: "dam3601.serial", address: "S=02;temperature[001]", dataType: "float32", wantText: "s=2;temperature[1]"},
		{driverID: "yudian.ai-bus", address: "S=02;010", dataType: "int16", wantText: "s=2;10"},
		{driverID: "delixi.dtsu6606", address: "x=3;0768", dataType: "float32", wantText: "x=3;0768"},
		{driverID: "dlt645.2007-serial", address: "S=2;00-00-00-01", dataType: "float64", wantText: "s=000000000002;00-00-00-01"},
		{driverID: "dlt645.1997-over-tcp", address: "b6-11", dataType: "string", wantText: "B6-11"},
		{driverID: "dlt698.tcp-net", address: "s=A1;20-00-02-00", dataType: "uint32", wantText: "s=0000000000A1;20-00-02-00"},
		{driverID: "dcs.nanjing-auto", address: "s=2;x=3;100", dataType: "int16", wantText: "s=2;x=3;100"},
		{driverID: "robot.estun-tcp", address: "51", dataType: "int16", wantText: "51"},
		{driverID: "robot.fanuc-interface", address: "SDO1", dataType: "bool", wantText: "SDO1"},
		{driverID: "siemens.s7-plus", address: "8A0E0001.A", dataType: "int16", wantText: "8A0E0001.A"},
		{driverID: "mqtt.rpc-device", address: "DB1.0", dataType: "int16", wantText: "DB1.0"},
		{driverID: "ec-fan.machine-serial", address: "speedMaximum", dataType: "int32", wantText: "speedMaximum"},
	}
	for _, tt := range tests {
		t.Run(tt.driverID+tt.address, func(t *testing.T) {
			store := &fakeCollectorPointStore{connection: repository.CollectorConnectionRecord{ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000", DriverID: tt.driverID, DriverVersion: "1.0.0", SchemaVersion: 1}}
			service := newRepositoryCollectorPointService(t, store)
			result, err := service.CreatePointsBatch(context.Background(), store.connection.ProjectID, store.connection.ID, "550e8400-e29b-41d4-a716-446655440001", []CreateCollectorPointInput{{Name: "测试变量", Address: map[string]any{"address": tt.address}, DataType: tt.dataType, ElementCount: 1}})
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantFailure == "" {
				if len(result.List) != 1 || result.List[0].AddressText != tt.wantText {
					t.Fatalf("合法地址未规范化: %+v", result)
				}
				return
			}
			if len(result.Failed) != 1 || !strings.Contains(result.Failed[0].Message, tt.wantFailure) {
				t.Fatalf("非法地址未返回预期错误: %+v", result)
			}
		})
	}
}

func TestCollectorPointServiceValidatesSiemensS7AddressAndDataTypeCombinations(t *testing.T) {
	tests := []struct {
		name         string
		address      map[string]any
		dataType     string
		elementCount int
		wantFailure  string
	}{
		{name: "bool requires bit offset", address: map[string]any{"area": "marker", "byteOffset": 10}, dataType: "bool", elementCount: 1, wantFailure: "Siemens S7 bool 变量必须配置可位寻址区域和位偏移"},
		{name: "numeric rejects bit offset", address: map[string]any{"area": "marker", "byteOffset": 10, "bitOffset": 1}, dataType: "int16", elementCount: 1, wantFailure: "只有 Siemens S7 bool 变量可以配置位偏移"},
		{name: "timer rejects float", address: map[string]any{"area": "timer", "byteOffset": 10}, dataType: "float32", elementCount: 1, wantFailure: "定时器和计数器只支持 int16 或 uint16 数据类型"},
		{name: "data block requires number", address: map[string]any{"area": "dataBlock", "byteOffset": 10}, dataType: "int16", elementCount: 1, wantFailure: "采集点地址不符合驱动 Schema"},
		{name: "marker rejects data block number", address: map[string]any{"area": "marker", "dbNumber": 1, "byteOffset": 10}, dataType: "int16", elementCount: 1, wantFailure: "采集点地址不符合驱动 Schema"},
		{name: "datetime rejects array", address: map[string]any{"area": "dataBlock", "dbNumber": 1, "byteOffset": 10}, dataType: "datetime", elementCount: 2, wantFailure: "Siemens S7 datetime 变量只支持单元素读取"},
		{name: "data block accepts numeric type", address: map[string]any{"area": "dataBlock", "dbNumber": 1, "byteOffset": 10}, dataType: "int16", elementCount: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeCollectorPointStore{connection: repository.CollectorConnectionRecord{
				ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000",
				DriverID: "siemens.s7-tcp", DriverVersion: "1.0.0", SchemaVersion: 1,
			}}
			service := newRepositoryCollectorPointService(t, store)
			result, err := service.CreatePointsBatch(
				context.Background(),
				store.connection.ProjectID,
				store.connection.ID,
				"550e8400-e29b-41d4-a716-446655440001",
				[]CreateCollectorPointInput{{Name: "测试变量", Address: tt.address, DataType: tt.dataType, ElementCount: tt.elementCount}},
			)
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantFailure == "" {
				if len(result.List) != 1 || len(result.Failed) != 0 {
					t.Fatalf("合法变量未创建: %+v", result)
				}
				if result.List[0].AddressText != "DB1.10" {
					t.Fatalf("S7 地址文本不正确: %s", result.List[0].AddressText)
				}
				return
			}
			if len(result.List) != 0 || len(result.Failed) != 1 || !strings.Contains(result.Failed[0].Message, tt.wantFailure) {
				t.Fatalf("非法组合未返回预期错误 %q: %+v", tt.wantFailure, result)
			}
		})
	}
}

func TestCollectorPointServiceFindsExistingAddressIndexes(t *testing.T) {
	store := &fakeCollectorPointStore{
		connection:           repository.CollectorConnectionRecord{ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000", DriverID: "opcua.standard", SchemaVersion: 1},
		existingAddressTexts: []string{"ns=2;s=Temperature"},
	}
	service := newRepositoryCollectorPointService(t, store)
	indexes, err := service.FindExistingPointAddressIndexes(context.Background(), store.connection.ProjectID, store.connection.ID, []map[string]any{
		{"nodeId": "ns=2;s=Temperature"},
		{"nodeId": "ns=2;s=Pressure"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(indexes) != 1 || indexes[0] != 0 {
		t.Fatalf("unexpected indexes: %#v", indexes)
	}
}

func TestCollectorPointServiceReturnsInvalidAddressAsBatchFailure(t *testing.T) {
	store := &fakeCollectorPointStore{connection: repository.CollectorConnectionRecord{ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000", DriverID: "opcua.standard", SchemaVersion: 1}}
	service := newRepositoryCollectorPointService(t, store)
	result, err := service.CreatePointsBatch(context.Background(), "550e8400-e29b-41d4-a716-446655440000", store.connection.ID, "550e8400-e29b-41d4-a716-446655440001", []CreateCollectorPointInput{{Code: "temperature", Name: "温度", Address: map[string]any{}, DataType: "float32", ElementCount: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.List) != 0 || len(result.Failed) != 1 || result.Failed[0].Code != "VALIDATION_FAILED" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestCollectorPointServiceRejectsMalformedOpcUaNodeID(t *testing.T) {
	store := &fakeCollectorPointStore{connection: repository.CollectorConnectionRecord{ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000", DriverID: "opcua.standard", SchemaVersion: 1}}
	service := newRepositoryCollectorPointService(t, store)

	result, err := service.CreatePointsBatch(context.Background(), store.connection.ProjectID, store.connection.ID, "550e8400-e29b-41d4-a716-446655440001", []CreateCollectorPointInput{{Name: "LastChange", Address: map[string]any{"nodeId": "111"}, DataType: "bool", ElementCount: 1}})

	if err != nil {
		t.Fatal(err)
	}
	if len(result.List) != 0 || len(result.Failed) != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if !strings.Contains(result.Failed[0].Message, "OPC UA NodeId 格式无效") {
		t.Fatalf("unexpected failure: %#v", result.Failed[0])
	}
	if len(store.created) != 0 {
		t.Fatalf("invalid point must not be persisted: %#v", store.created)
	}
}

func TestCollectorPointServiceNormalizesYokogawaAddress(t *testing.T) {
	store := &fakeCollectorPointStore{connection: repository.CollectorConnectionRecord{ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000", DriverID: "yokogawa.link-tcp", SchemaVersion: 1}}
	service := newRepositoryCollectorPointService(t, store)

	result, err := service.CreatePointsBatch(context.Background(), store.connection.ProjectID, store.connection.ID, "550e8400-e29b-41d4-a716-446655440001", []CreateCollectorPointInput{
		{Name: "横河数据", Address: map[string]any{"address": "cpu=02;d100"}, DataType: "int16", ElementCount: 1},
		{Name: "横河位点", Address: map[string]any{"address": "D101"}, DataType: "bool", ElementCount: 1},
	})

	if err != nil {
		t.Fatal(err)
	}
	if len(result.List) != 0 || len(store.created) != 0 {
		t.Fatalf("同批次存在非法行时不得写入合法行: %+v", result)
	}
	if len(result.Failed) != 1 || !strings.Contains(result.Failed[0].Message, "必须使用继电器地址") {
		t.Fatalf("非法横河类型组合未被拒绝: %+v", result)
	}
}

func TestCollectorPointServiceRejectsWholeBatchWhenAnyRowConflicts(t *testing.T) {
	store := &fakeCollectorPointStore{
		connection:            repository.CollectorConnectionRecord{ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000", DriverID: "opcua.standard", SchemaVersion: 1},
		existingConflictNames: []string{"existing"},
	}
	service := newRepositoryCollectorPointService(t, store)
	result, err := service.CreatePointsBatch(context.Background(), store.connection.ProjectID, store.connection.ID, "550e8400-e29b-41d4-a716-446655440001", []CreateCollectorPointInput{
		{Name: "Temperature", Address: map[string]any{"nodeId": "ns=2;s=Temperature"}, DataType: "float32", ElementCount: 1},
		{Name: "temperature", Address: map[string]any{"nodeId": "ns=2;s=Temperature2"}, DataType: "float32", ElementCount: 1},
		{Name: "Existing", Address: map[string]any{"nodeId": "ns=2;s=Existing"}, DataType: "float32", ElementCount: 1},
		{Name: "Pressure", Address: map[string]any{"nodeId": "ns=2;s=Pressure"}, DataType: "float32", ElementCount: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.List) != 0 || len(result.Failed) != 2 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.Failed[0].Index != 1 || result.Failed[0].Code != "DUPLICATE_NAME" || result.Failed[1].Index != 2 {
		t.Fatalf("unexpected failures: %#v", result.Failed)
	}
}

func TestCollectorPointServiceDoesNotCreateLaterRowsAfterConflict(t *testing.T) {
	store := &fakeCollectorPointStore{
		connection:                   repository.CollectorConnectionRecord{ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000", DriverID: "opcua.standard", SchemaVersion: 1},
		existingConflictAddressTexts: []string{"ns=2;s=ExistingAddress"},
	}
	pointService := newRepositoryCollectorPointService(t, store)
	result, err := pointService.CreatePointsBatch(context.Background(), store.connection.ProjectID, store.connection.ID, "550e8400-e29b-41d4-a716-446655440001", []CreateCollectorPointInput{
		{Name: "Temperature", Address: map[string]any{"nodeId": "ns=2;s=ExistingAddress"}, DataType: "float32", ElementCount: 1},
		{Name: "temperature", Address: map[string]any{"nodeId": "ns=2;s=ValidAddress"}, DataType: "float32", ElementCount: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.List) != 0 {
		t.Fatalf("unexpected created points: %#v", result.List)
	}
	if len(result.Failed) != 1 || result.Failed[0].Index != 0 || result.Failed[0].Code != "DUPLICATE_ADDRESS" {
		t.Fatalf("unexpected failures: %#v", result.Failed)
	}
}

func TestCollectorPointExportWritesReimportableCSV(t *testing.T) {
	description := "现场温度"
	store := &fakeCollectorPointStore{
		connection: repository.CollectorConnectionRecord{ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000", Name: "锅炉 OPC UA", DriverID: "opcua.standard", SchemaVersion: 1},
		exportRecords: []repository.CollectorPointExportRecord{{
			CollectorPointRecord: repository.CollectorPointRecord{
				ID: "550e8400-e29b-41d4-a716-446655440003", Name: "温度", Description: &description,
				Address: map[string]any{"nodeId": "ns=2;s=Temperature"}, DataType: "float32",
				ElementCount: 1, Enabled: true, ReadOptions: map[string]any{}, Acquisition: map[string]any{"intervalMs": 1000}, Metadata: map[string]any{},
			},
			GroupPath: "锅炉/温度",
		}},
	}
	pointService := newRepositoryCollectorPointService(t, store)
	plan, err := pointService.PreparePointExport(context.Background(), store.connection.ProjectID, store.connection.ID, CollectorPointExportRequest{Format: "csv", Scope: "current_page", Page: 1, PageSize: 50})
	if err != nil {
		t.Fatal(err)
	}
	buffer := bytes.NewBuffer(nil)
	if err := plan.WriteCSV(context.Background(), buffer); err != nil {
		t.Fatal(err)
	}
	payload := buffer.String()
	for _, expected := range []string{"\ufeffgroupPath,name,description,dataType,elementCount,enabled,address.nodeId", "锅炉/温度,温度,现场温度,float32,1,true,ns=2;s=Temperature"} {
		if !strings.Contains(payload, expected) {
			t.Fatalf("CSV 缺少内容 %q: %s", expected, payload)
		}
	}
}

func newRepositoryCollectorPointService(t *testing.T, store CollectorPointStore) *CollectorPointService {
	t.Helper()
	root := filepath.Clean(filepath.Join("..", "..", "..", "contracts", "collector-protocols"))
	catalog, err := collectorprotocol.LoadCatalog(os.DirFS(root))
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewCollectorPointService(store, catalog)
	if err != nil {
		t.Fatal(err)
	}
	return service
}

type fakeCollectorPointStore struct {
	connection                   repository.CollectorConnectionRecord
	created                      []repository.CreateCollectorPointParams
	existingAddressTexts         []string
	existingConflictNames        []string
	existingConflictAddressTexts []string
	exportRecords                []repository.CollectorPointExportRecord
}

func (f *fakeCollectorPointStore) GetConnection(context.Context, string, string) (*repository.CollectorConnectionRecord, error) {
	return &f.connection, nil
}
func (f *fakeCollectorPointStore) ListPointGroups(context.Context, string, string, *string) ([]repository.CollectorPointGroupRecord, error) {
	return nil, nil
}
func (f *fakeCollectorPointStore) CreatePointGroup(context.Context, repository.CreateCollectorPointGroupParams) (*repository.CollectorPointGroupRecord, error) {
	return nil, nil
}
func (f *fakeCollectorPointStore) UpdatePointGroup(context.Context, repository.UpdateCollectorPointGroupParams) (*repository.CollectorPointGroupRecord, error) {
	return nil, nil
}
func (f *fakeCollectorPointStore) DeletePointGroup(context.Context, string, string, string) error {
	return nil
}
func (f *fakeCollectorPointStore) ListPoints(context.Context, string, string, repository.CollectorPointListFilter) ([]repository.CollectorPointRecord, int, error) {
	return nil, 0, nil
}
func (f *fakeCollectorPointStore) ListPointCodes(context.Context, string, string) ([]string, error) {
	return nil, nil
}
func (f *fakeCollectorPointStore) ListExistingPointAddressTexts(context.Context, string, string, []string) ([]string, error) {
	return f.existingAddressTexts, nil
}
func (f *fakeCollectorPointStore) ListExistingPointConflicts(context.Context, string, string, []string, []string) (repository.CollectorPointExistingConflicts, error) {
	return repository.CollectorPointExistingConflicts{Names: f.existingConflictNames, AddressTexts: f.existingConflictAddressTexts}, nil
}
func (f *fakeCollectorPointStore) StreamPointsForExport(_ context.Context, _ string, _ string, _ repository.CollectorPointExportFilter, visit func(repository.CollectorPointExportRecord) error) error {
	for _, record := range f.exportRecords {
		if err := visit(record); err != nil {
			return err
		}
	}
	return nil
}
func (f *fakeCollectorPointStore) GetPointsByIDs(context.Context, string, string, []string) ([]repository.CollectorPointRecord, error) {
	return nil, nil
}
func (f *fakeCollectorPointStore) CreatePointsBatch(_ context.Context, params []repository.CreateCollectorPointParams) ([]repository.CollectorPointRecord, error) {
	f.created = params
	records := make([]repository.CollectorPointRecord, 0, len(params))
	for _, item := range params {
		records = append(records, repository.CollectorPointRecord{
			ID: item.ID, ProjectID: item.ProjectID, ConnectionID: item.ConnectionID, GroupID: item.GroupID,
			Code: item.Code, Name: item.Name, Description: item.Description, Address: item.Address,
			AddressText: item.AddressText, AddressSchemaVersion: item.AddressSchemaVersion, DataType: item.DataType,
			ElementCount: item.ElementCount, ReadOptions: item.ReadOptions, Acquisition: item.Acquisition,
			Enabled: item.Enabled, SortOrder: item.SortOrder, Metadata: item.Metadata,
		})
	}
	return records, nil
}
func (f *fakeCollectorPointStore) UpdatePointsBatch(context.Context, []repository.UpdateCollectorPointParams) ([]repository.CollectorPointRecord, error) {
	return nil, nil
}
func (f *fakeCollectorPointStore) DeletePointsBatch(context.Context, string, string, []string) error {
	return nil
}
func (f *fakeCollectorPointStore) MovePointsBatch(context.Context, string, string, *string, []string) error {
	return nil
}

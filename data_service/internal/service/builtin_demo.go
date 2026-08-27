package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/indu-forge/data_service/internal/auth"
)

const (
	// BuiltinDemoProjectID 与 dev_core 的空库初始化保持一致，不作为不可删除的系统工程。
	BuiltinDemoProjectID = "00000000-0000-4000-8000-000000000001"
	builtinDemoUserID    = "00000000-0000-4000-8000-000000000004"
	builtinDemoTenantID  = "550e8400-e29b-41d4-a716-446655440000"
)

type BuiltinDemoSeeder struct {
	connections *ConnectionService
	queries     *QueryService
	realtime    *RealtimeStoreService
	computes    *ComputeService
	alarms      *AlarmItemService
}

func NewBuiltinDemoSeeder(connections *ConnectionService, queries *QueryService, realtime *RealtimeStoreService, computes *ComputeService, alarms *AlarmItemService) *BuiltinDemoSeeder {
	return &BuiltinDemoSeeder{connections: connections, queries: queries, realtime: realtime, computes: computes, alarms: alarms}
}

// Ensure 为内置教程工程创建可直接运行的 IF 关系、时序和实时数据。
// 各步按名称幂等检查，用户编辑后不会在服务重启时被覆盖。
func (s *BuiltinDemoSeeder) Ensure(ctx context.Context) error {
	if s == nil || s.connections == nil || s.queries == nil || s.realtime == nil || s.computes == nil || s.alarms == nil {
		return fmt.Errorf("内置教程数据初始化器依赖不完整")
	}
	claims := &auth.Claims{UserID: builtinDemoUserID, Role: "SYSTEM_ADMIN", TenantID: builtinDemoTenantID, ProjectIDs: []string{BuiltinDemoProjectID}, Capabilities: []string{"*"}}

	connections, err := s.connections.ListConnections(ctx, BuiltinDemoProjectID, builtinDemoTenantID)
	if err != nil {
		return fmt.Errorf("读取教程接入源失败: %w", err)
	}
	relation, err := s.ensureConnection(ctx, connections, "IF关系库", "builtin.relation")
	if err != nil {
		return err
	}
	timeseries, err := s.ensureConnection(ctx, connections, "IF时序库", "builtin.timeseries")
	if err != nil {
		return err
	}
	realtime, err := s.ensureConnection(ctx, connections, "IF实时库", "builtin.realtime")
	if err != nil {
		return err
	}
	if err := s.ensureRelationData(ctx, relation.ID); err != nil {
		return err
	}
	if err := s.ensureTimeseriesData(ctx, timeseries.ID); err != nil {
		return err
	}

	temperaturePointID, err := s.ensureRelationQuery(ctx, relation.ID)
	if err != nil {
		return err
	}
	if err := s.ensureTimeseriesQuery(ctx, timeseries.ID); err != nil {
		return err
	}
	if err := s.ensureRealtimeSetpoint(ctx, realtime.ID); err != nil {
		return err
	}
	if err := s.ensureCompute(ctx, claims, temperaturePointID); err != nil {
		return err
	}
	if err := s.ensureAlarm(ctx, claims, temperaturePointID); err != nil {
		return err
	}
	return nil
}

func (s *BuiltinDemoSeeder) ensureConnection(ctx context.Context, existing []Connection, name, connectionType string) (*Connection, error) {
	for index := range existing {
		if existing[index].Name == name && existing[index].Type == connectionType {
			return &existing[index], nil
		}
	}
	enabled := true
	created, err := s.connections.CreateConnection(ctx, BuiltinDemoProjectID, builtinDemoTenantID, builtinDemoUserID, CreateConnectionInput{Name: name, Type: connectionType, Enabled: &enabled, Config: map[string]any{}})
	if err != nil {
		return nil, fmt.Errorf("创建教程接入源 %s 失败: %w", name, err)
	}
	return created, nil
}

func (s *BuiltinDemoSeeder) ensureRelationData(ctx context.Context, connectionID string) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS demo_line_status (
			id integer PRIMARY KEY,
			line_name text NOT NULL,
			temperature double precision NOT NULL,
			pressure double precision NOT NULL,
			setpoint double precision NOT NULL,
			running boolean NOT NULL,
			updated_at timestamptz NOT NULL DEFAULT now()
		)`,
		`INSERT INTO demo_line_status (id, line_name, temperature, pressure, setpoint, running)
		 SELECT 1, '一号线', 76.4, 0.68, 80.0, true
		 WHERE NOT EXISTS (SELECT 1 FROM demo_line_status WHERE id = 1)`,
	}
	for _, statement := range statements {
		if _, err := s.connections.ExecuteSQL(ctx, BuiltinDemoProjectID, connectionID, statement, nil); err != nil {
			return fmt.Errorf("初始化 IF 关系库教程数据失败: %w", err)
		}
	}
	return nil
}

func (s *BuiltinDemoSeeder) ensureTimeseriesData(ctx context.Context, connectionID string) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS demo_temperature_history (
			observed_at timestamptz NOT NULL,
			temperature double precision NOT NULL,
			pressure double precision NOT NULL,
			quality text NOT NULL DEFAULT 'good',
			PRIMARY KEY (observed_at)
		)`,
		`SELECT create_hypertable('demo_temperature_history', 'observed_at', if_not_exists => true)`,
		`INSERT INTO demo_temperature_history (observed_at, temperature, pressure, quality)
		 SELECT now() - make_interval(mins => (sample * 5)::int), 72.0 + (sample % 6) * 0.8, 0.62 + (sample % 4) * 0.02, 'good'
		 FROM generate_series(0, 23) AS sample
		 WHERE NOT EXISTS (SELECT 1 FROM demo_temperature_history)`,
	}
	for _, statement := range statements {
		if _, err := s.connections.ExecuteSQL(ctx, BuiltinDemoProjectID, connectionID, statement, nil); err != nil {
			return fmt.Errorf("初始化 IF 时序库教程数据失败: %w", err)
		}
	}
	return nil
}

func (s *BuiltinDemoSeeder) ensureRelationQuery(ctx context.Context, connectionID string) (string, error) {
	existing, err := s.queries.ListQueries(ctx, BuiltinDemoProjectID, QueryListFilter{ConnectionID: connectionID, Page: 1, PageSize: 100})
	if err != nil {
		return "", fmt.Errorf("读取关系库教程查询失败: %w", err)
	}
	for _, query := range existing.Queries {
		if query.Name == "demo_line_current" {
			for _, output := range query.Outputs {
				if output.Key == "temperature" {
					return output.DataPointID, nil
				}
			}
			return "", fmt.Errorf("教程查询 demo_line_current 缺少 temperature 输出")
		}
	}
	unit, precision := "℃", 1
	pressureUnit, pressurePrecision := "MPa", 2
	created, err := s.queries.CreateQuery(ctx, BuiltinDemoProjectID, builtinDemoUserID, CreateQueryInput{
		Name: "demo_line_current", ConnectionID: connectionID, QueryType: "sql",
		Config: map[string]any{"sql": `SELECT temperature, pressure FROM demo_line_status WHERE id = 1`},
		Outputs: []SourceOutputInput{
			{Key: "temperature", DisplayName: "设备温度", Selector: SourceOutputSelector{Kind: "column", Column: "temperature"}, DataType: "float64", Unit: &unit, PrecisionNum: &precision},
			{Key: "pressure", DisplayName: "管线压力", Selector: SourceOutputSelector{Kind: "column", Column: "pressure"}, DataType: "float64", Unit: &pressureUnit, PrecisionNum: &pressurePrecision},
		},
	})
	if err != nil {
		return "", fmt.Errorf("创建关系库教程查询失败: %w", err)
	}
	for _, output := range created.Outputs {
		if output.Key == "temperature" {
			return output.DataPointID, nil
		}
	}
	return "", fmt.Errorf("创建的教程查询缺少 temperature 输出")
}

func (s *BuiltinDemoSeeder) ensureTimeseriesQuery(ctx context.Context, connectionID string) error {
	existing, err := s.queries.ListQueries(ctx, BuiltinDemoProjectID, QueryListFilter{ConnectionID: connectionID, Page: 1, PageSize: 100})
	if err != nil {
		return fmt.Errorf("读取时序库教程查询失败: %w", err)
	}
	for _, query := range existing.Queries {
		if query.Name == "demo_temperature_history" {
			return nil
		}
	}
	_, err = s.queries.CreateQuery(ctx, BuiltinDemoProjectID, builtinDemoUserID, CreateQueryInput{
		Name: "demo_temperature_history", ConnectionID: connectionID, QueryType: "sql",
		Config:  map[string]any{"sql": `SELECT observed_at, temperature, pressure, quality FROM demo_temperature_history ORDER BY observed_at DESC LIMIT 24`},
		Outputs: []SourceOutputInput{{Key: "result", DisplayName: "温度历史", Selector: SourceOutputSelector{Kind: "whole"}, DataType: "object"}},
	})
	if err != nil {
		return fmt.Errorf("创建时序库教程查询失败: %w", err)
	}
	return nil
}

func (s *BuiltinDemoSeeder) ensureRealtimeSetpoint(ctx context.Context, connectionID string) error {
	const key = "demo:line1:setpoint"
	if _, err := s.realtime.GetKey(ctx, BuiltinDemoProjectID, connectionID, key); err == nil {
		return nil
	}
	_, err := s.realtime.SaveKey(ctx, BuiltinDemoProjectID, connectionID, builtinDemoUserID, SaveRealtimeStoreKeyInput{Key: key, Type: "string", ValueType: "number", Value: 80.0, TTLSeconds: 0, Description: "一号线温度设定值"})
	if err != nil {
		return fmt.Errorf("创建实时库教程设定值失败: %w", err)
	}
	return nil
}

func (s *BuiltinDemoSeeder) ensureCompute(ctx context.Context, claims *auth.Claims, temperaturePointID string) error {
	existing, err := s.computes.ListComputeUnits(ctx, claims, BuiltinDemoProjectID, ComputeUnitListFilter{Search: "temperatureConvert", Page: 1, PageSize: 100})
	if err != nil {
		return fmt.Errorf("读取教程计算单元失败: %w", err)
	}
	for _, unit := range existing.Units {
		if unit.Name == "temperatureConvert" {
			return nil
		}
	}
	description := "读取 IF 关系库温度数据点，输出摄氏度与华氏度。"
	_, err = s.computes.CreateComputeUnit(ctx, claims, BuiltinDemoProjectID, CreateComputeUnitInput{
		Name: "temperatureConvert", Description: &description, Language: "js", TriggerType: "manual",
		ScriptCode: `const reading = await temperature.get();
if (reading.code !== 0) return reading;
const celsius = Number(reading.data.value);
return { celsius, fahrenheit: celsius * 9 / 5 + 32 };`,
		TriggerConfig: map[string]any{},
		InputBindings: map[string]any{"datapointVariables": []any{map[string]any{"datapointId": temperaturePointID, "alias": "temperature"}}},
		Outputs:       []ComputeOutputInput{{Key: "result", Name: "温度换算结果", Path: "calc.temperatureConvert.result", DataType: "object", NullPolicy: "error"}},
	})
	if err != nil {
		return fmt.Errorf("创建教程计算单元失败: %w", err)
	}
	return nil
}

func (s *BuiltinDemoSeeder) ensureAlarm(ctx context.Context, claims *auth.Claims, temperaturePointID string) error {
	existing, err := s.alarms.List(ctx, claims, BuiltinDemoProjectID, AlarmItemListFilter{DatapointID: temperaturePointID, Page: 1, PageSize: 100})
	if err != nil {
		return fmt.Errorf("读取教程报警失败: %w", err)
	}
	for _, item := range existing.List {
		if strings.TrimSpace(item.DisplayName) == "设备温度_越限" {
			return nil
		}
	}
	preset, enabled := "limit", true
	_, err = s.alarms.Create(ctx, claims, BuiltinDemoProjectID, SaveAlarmItemInput{
		DatapointID: temperaturePointID, DisplayName: "设备温度_越限", PresetSlot: &preset,
		Mode: "point", EvaluationMode: "single", IsEnabled: &enabled,
		Conditions:   []AlarmCondition{{Kind: "threshold", Operator: "gt", Label: "高温", Severity: "warning", Params: map[string]any{"threshold": 80.0}, Deadband: 1}},
		Notification: AlarmNotificationSettings{Mode: "inherit", ChannelIDs: []string{}},
	})
	if err != nil {
		return fmt.Errorf("创建教程报警失败: %w", err)
	}
	return nil
}

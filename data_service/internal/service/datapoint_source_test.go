package service

import (
	"context"
	"testing"

	"github.com/indu-forge/data_service/internal/repository"
)

func TestToDataPointListItemKeepsSourceObjectID(t *testing.T) {
	sourceID := "00000000-0000-0000-0000-000000000001"
	service := &DataPointService{}
	item := service.toDataPointListItem(context.Background(), "project-1", repository.DataPointRecord{
		SourceType: "calc.output",
		SourceID:   &sourceID,
	})

	if item.SourceID == nil || *item.SourceID != sourceID {
		t.Fatalf("sourceId = %#v, want %q", item.SourceID, sourceID)
	}
}

func TestSourceTypeUsesSourceIDAsConnectionIncludesProtocolOutputs(t *testing.T) {
	for _, sourceType := range []string{"http.request", "websocket.session", "realtime.key", "kafka.field", "kafka.raw"} {
		if !sourceTypeUsesSourceIDAsConnection(sourceType) {
			t.Fatalf("source type %q should use sourceId as connection id", sourceType)
		}
	}

	for _, sourceType := range []string{"calc.output", "db.query", "mqtt.tag", "collector.point"} {
		if sourceTypeUsesSourceIDAsConnection(sourceType) {
			t.Fatalf("source type %q must not use sourceId as connection id", sourceType)
		}
	}
}

func TestDataPointQueryParametersAcceptsSavedDefinitions(t *testing.T) {
	parameters, err := dataPointQueryParameters(map[string]any{
		"parameters": []any{
			map[string]any{"name": "withoutDefault"},
			map[string]any{"name": "limit", "default": 10},
		},
	})
	if err != nil {
		t.Fatalf("dataPointQueryParameters returned error: %v", err)
	}
	if len(parameters) != 1 || parameters["limit"] != 10 {
		t.Fatalf("parameters = %#v, want only explicit default", parameters)
	}

	empty, err := dataPointQueryParameters(map[string]any{"parameters": []any{}})
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty definitions = %#v, %v; want empty map", empty, err)
	}
}

func TestQueryDatasetValueKeepsOnlyReturnedRowCount(t *testing.T) {
	dataset := queryDatasetValue(&QueryExecutionResult{
		Data:        []map[string]any{{"id": 1}},
		Columns:     []string{"id"},
		RowCount:    1,
		Truncated:   true,
		TruncatedBy: "rows",
	})
	if dataset["rowCount"] != 1 {
		t.Fatalf("dataset = %#v, want returned row count", dataset)
	}
	for _, omitted := range []string{"total", "truncated"} {
		if _, exists := dataset[omitted]; exists {
			t.Fatalf("dataset must omit %s: %#v", omitted, dataset)
		}
	}
}

func TestQueryOutputValidityUsesSavedOutputMapping(t *testing.T) {
	query := repository.QueryRecord{ID: "query-1", Name: "demo_line_current", Outputs: []repository.SourceOutputMappingRecord{
		{DataPointID: "temperature-point", DataPointPath: "db.IF关系库.demo_line_current.temperature", DisplayName: "设备温度"},
		{DataPointID: "history-point", DataPointPath: "db.IF时序库.demo_temperature_history", DisplayName: "温度历史"},
	}}

	for _, point := range []repository.DataPointRecord{
		{ID: "temperature-point", Path: "db.IF关系库.demo_line_current.temperature", Name: "设备温度"},
		{ID: "history-point", Path: "db.IF时序库.demo_temperature_history", Name: "温度历史"},
	} {
		if !queryOutputMappingMatchesDataPoint(query, point) {
			t.Fatalf("saved query output should remain valid: %#v", point)
		}
	}

	if queryOutputMappingMatchesDataPoint(query, repository.DataPointRecord{ID: "temperature-point", Path: "db.IF关系库.demo_line_current.wrong", Name: "设备温度"}) {
		t.Fatal("mismatched generated path must be rejected")
	}
}

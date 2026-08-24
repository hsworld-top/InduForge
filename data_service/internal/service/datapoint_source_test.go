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

package service

import (
	"context"
	"testing"
	"time"

	"github.com/indu-forge/data_service/internal/repository"
)

type fakeProtocolPreviewRepository struct {
	connection repository.ProtocolPreviewConnectionRecord
	record     repository.CreateAccessSourceRecordParams
}

func (r *fakeProtocolPreviewRepository) GetPreviewConnection(ctx context.Context, projectID, connectionID string) (*repository.ProtocolPreviewConnectionRecord, error) {
	return &r.connection, nil
}

func (r *fakeProtocolPreviewRepository) CreateAccessSourceRecord(ctx context.Context, params repository.CreateAccessSourceRecordParams) error {
	r.record = params
	return nil
}

type fakeProtocolPreviewAdapter struct {
	input  ProtocolPreviewAdapterInput
	result ProtocolPreviewResult
}

func (a *fakeProtocolPreviewAdapter) Preview(ctx context.Context, input ProtocolPreviewAdapterInput) (*ProtocolPreviewResult, error) {
	a.input = input
	return &a.result, nil
}

func TestProtocolPreviewService_ClampsDefaultsAndDispatchesAdapter(t *testing.T) {
	projectID := "269828b0-f634-4b42-85ff-3c6f2a67e8ad"
	repo := &fakeProtocolPreviewRepository{
		connection: repository.ProtocolPreviewConnectionRecord{
			ID:        "7a5d533b-9cad-4f0e-99e5-7c4a43b83c75",
			ProjectID: projectID,
			Type:      "http",
			Status:    "connected",
			Config: map[string]any{
				"baseUrl": "http://127.0.0.1/api",
				"method":  "GET",
			},
		},
	}
	adapter := &fakeProtocolPreviewAdapter{
		result: ProtocolPreviewResult{
			Protocol:      "http",
			ConnectionID:  repo.connection.ID,
			Status:        "ok",
			Samples:       []any{map[string]any{"temperature": 32}},
			Diagnostics:   map[string]any{"statusCode": 200},
			DurationMS:    12,
			Truncated:     false,
			EffectiveTime: time.Now(),
		},
	}
	service := NewProtocolPreviewService(repo, map[string]ProtocolPreviewAdapter{"http": adapter})

	result, err := service.Preview(context.Background(), projectID, repo.connection.ID, ProtocolPreviewInput{})
	if err != nil {
		t.Fatalf("preview failed: %v", err)
	}

	if adapter.input.Limit != 10 {
		t.Fatalf("expected default limit 10, got %d", adapter.input.Limit)
	}
	if adapter.input.Timeout != 5*time.Second {
		t.Fatalf("expected default timeout 5s, got %s", adapter.input.Timeout)
	}
	if result.Protocol != "http" || result.Status != "ok" {
		t.Fatalf("unexpected preview result: %#v", result)
	}
	if repo.record.SampleCount != 1 {
		t.Fatalf("expected summarized sample count 1, got %d", repo.record.SampleCount)
	}
	if _, ok := repo.record.Detail["samples"]; ok {
		t.Fatalf("access source record must not persist raw samples: %#v", repo.record.Detail)
	}
}

func TestProtocolPreviewService_ClampsExplicitBounds(t *testing.T) {
	projectID := "269828b0-f634-4b42-85ff-3c6f2a67e8ad"
	repo := &fakeProtocolPreviewRepository{
		connection: repository.ProtocolPreviewConnectionRecord{
			ID:        "7a5d533b-9cad-4f0e-99e5-7c4a43b83c75",
			ProjectID: projectID,
			Type:      "redis",
			Status:    "connected",
			Config:    map[string]any{"address": "127.0.0.1:6379"},
		},
	}
	adapter := &fakeProtocolPreviewAdapter{result: ProtocolPreviewResult{Protocol: "redis", ConnectionID: repo.connection.ID, Status: "ok"}}
	service := NewProtocolPreviewService(repo, map[string]ProtocolPreviewAdapter{"redis": adapter})

	_, err := service.Preview(context.Background(), projectID, repo.connection.ID, ProtocolPreviewInput{
		Limit:     1000,
		TimeoutMS: 90000,
	})
	if err != nil {
		t.Fatalf("preview failed: %v", err)
	}

	if adapter.input.Limit != 100 {
		t.Fatalf("expected limit clamp to 100, got %d", adapter.input.Limit)
	}
	if adapter.input.Timeout != 30*time.Second {
		t.Fatalf("expected timeout clamp to 30s, got %s", adapter.input.Timeout)
	}
}

package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/indu-forge/data_service/internal/repository"
)

func TestCollectorAgentKeepsSerialPortResources(t *testing.T) {
	now := time.Now()
	payload, err := json.Marshal([]CollectorProtocolCapability{{
		DriverID:       "modbus.rtu",
		DriverVersion:  "1.0.0",
		SchemaVersions: []int{1},
		Operations:     []string{"connection.test", "point.read"},
		Resources:      &CollectorProtocolResources{SerialPorts: []string{"COM2", "COM10"}},
	}})
	if err != nil {
		t.Fatal(err)
	}

	agent := toCollectorAgent(repository.CollectorDevAgentRecord{
		ID:           "agent-1",
		Name:         "测试代理",
		Capabilities: payload,
		LastSeenAt:   &now,
		CreatedAt:    now,
	}, now)

	if len(agent.Capabilities) != 1 || agent.Capabilities[0].Resources == nil {
		t.Fatalf("unexpected capabilities: %#v", agent.Capabilities)
	}
	ports := agent.Capabilities[0].Resources.SerialPorts
	if len(ports) != 2 || ports[0] != "COM2" || ports[1] != "COM10" {
		t.Fatalf("unexpected serial ports: %#v", ports)
	}
}

func TestValidateCollectorTaskInputUsesSavedObjectReferences(t *testing.T) {
	err := validateCollectorTaskInput(CollectorTaskInput{AgentID: "agent-1", ConnectionID: "550e8400-e29b-41d4-a716-446655440002", Operation: "point.read", Input: map[string]any{"pointIds": []any{"550e8400-e29b-41d4-a716-446655440003"}}, TimeoutSeconds: 30})
	if err != nil {
		t.Fatal(err)
	}
	if err := validateCollectorTaskInput(CollectorTaskInput{AgentID: "agent-1", ConnectionID: "550e8400-e29b-41d4-a716-446655440002", Operation: "opcua" + ".read", Input: map[string]any{}}); err == nil {
		t.Fatal("expected protocol-specific operation to be rejected")
	}
}

func TestCollectorAgentSupportsDriverVersionSchemaAndOperation(t *testing.T) {
	capabilities := []CollectorProtocolCapability{{DriverID: "opcua.standard", DriverVersion: "1.0.0", SchemaVersions: []int{1}, Operations: []string{"connection.test", "point.read"}}}
	if !collectorAgentSupports(capabilities, "opcua.standard", "1.0.0", 1, "point.read") {
		t.Fatal("expected capability match")
	}
	if collectorAgentSupports(capabilities, "opcua.standard", "1.0.0", 2, "point.read") {
		t.Fatal("unexpected schema version match")
	}
}

func TestValidateCollectorTaskInputAcceptsBrowseBatch(t *testing.T) {
	err := validateCollectorTaskInput(CollectorTaskInput{
		AgentID:        "agent-1",
		ConnectionID:   "550e8400-e29b-41d4-a716-446655440002",
		Operation:      "device.browse",
		Input:          map[string]any{"parentNodeIds": []any{"ns=0;i=85", "ns=2;s=Line1"}, "maxDepth": float64(1)},
		TimeoutSeconds: 30,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestWaitForCollectorTaskRetriesUntilTaskArrives(t *testing.T) {
	calls := 0
	signal := make(chan struct{}, 1)
	signal <- struct{}{}
	record, err := waitForCollectorTask(context.Background(), time.Second, signal, func() (*repository.CollectorDevTaskRecord, error) {
		calls++
		if calls < 2 {
			return nil, nil
		}
		return &repository.CollectorDevTaskRecord{ID: "task-1"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if record == nil || record.ID != "task-1" {
		t.Fatalf("unexpected task: %#v", record)
	}
	if calls != 2 {
		t.Fatalf("claim calls = %d, want 2", calls)
	}
}

func TestWaitForCollectorTaskStopsWhenContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := waitForCollectorTask(ctx, time.Second, make(chan struct{}), func() (*repository.CollectorDevTaskRecord, error) {
		return nil, nil
	})
	if err != context.Canceled {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

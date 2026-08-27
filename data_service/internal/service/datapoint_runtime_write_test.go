package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

func TestRuntimeWritePermissionDenyWins(t *testing.T) {
	grant := repository.RuntimePermissionGrant{AllowRoles: []string{"operator"}, DenyRoles: []string{"operator"}, Inherit: true}
	if runtimeWriteAllowed(grant, "operator") {
		t.Fatal("denyRoles 必须优先于 allowRoles")
	}
	if runtimeWriteAllowed(repository.RuntimePermissionGrant{Inherit: false}, "operator") {
		t.Fatal("inherit=false 且未明确允许时必须拒绝")
	}
	if !runtimeWriteAllowed(repository.RuntimePermissionGrant{AllowRoles: []string{"operator"}, Inherit: true}, "maintainer") {
		t.Fatal("inherit=true 时未列入 allowRoles 的角色仍应沿用工程写权限")
	}
}

func TestNormalizeRuntimeWriteValueChecksTypeAndRange(t *testing.T) {
	record := repository.DataPointRecord{DataType: "float64", MinValue: float64Pointer(0), MaxValue: float64Pointer(100)}
	value, err := normalizeRuntimeWriteValue(record, float64(42))
	if err != nil || value != float64(42) {
		t.Fatalf("合法数值被拒绝: value=%v err=%v", value, err)
	}
	_, err = normalizeRuntimeWriteValue(record, float64(101))
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("越界应返回业务错误: %v", err)
	}
	record.DataType = "bool"
	if _, err := normalizeRuntimeWriteValue(record, "true"); err == nil {
		t.Fatal("布尔数据点不应接受字符串")
	}
}

func TestNormalizeRuntimeWriteValueChecksStructuredTypes(t *testing.T) {
	if _, err := normalizeRuntimeWriteValue(repository.DataPointRecord{DataType: "object"}, []any{"invalid"}); err == nil {
		t.Fatal("object 数据点必须拒绝数组")
	}
	if _, err := normalizeRuntimeWriteValue(repository.DataPointRecord{DataType: "array"}, map[string]any{"invalid": true}); err == nil {
		t.Fatal("array 数据点必须拒绝对象")
	}
}

type fakeMqttDataPointPublisher struct {
	projectID      string
	subscriptionID string
	payload        any
}

func (f *fakeMqttDataPointPublisher) PublishSubscriptionMessage(_ context.Context, projectID, subscriptionID string, payload any) (*MqttPublishResult, error) {
	f.projectID = projectID
	f.subscriptionID = subscriptionID
	f.payload = payload
	return &MqttPublishResult{Topic: "demo/line/events", QOS: 1}, nil
}

func TestWriteMqttSubscriptionDataPointPublishesMessage(t *testing.T) {
	publisher := &fakeMqttDataPointPublisher{}
	service := &DataPointService{mqttPublisher: publisher}
	sourceID := "subscription-1"
	payload := map[string]any{"event": "operator_test", "running": true}
	result, err := service.writeDataPointRecord(context.Background(), repository.DataPointRecord{
		ID:         "point-1",
		ProjectID:  "project-1",
		Path:       "mqtt.IF消息库.demo_line_events",
		Status:     "active",
		SourceType: "mqtt.subscription",
		SourceID:   &sourceID,
		DataType:   "object",
		RuntimePermissions: repository.DataPointRuntimePermissions{
			Write: repository.RuntimePermissionGrant{Inherit: true},
		},
	}, "user-1", "operator", payload)
	if err != nil {
		t.Fatalf("writeDataPointRecord() error = %v", err)
	}
	if result.Path != "mqtt.IF消息库.demo_line_events" {
		t.Fatalf("unexpected result path %q", result.Path)
	}
	if publisher.projectID != "project-1" || publisher.subscriptionID != sourceID {
		t.Fatalf("unexpected publish target project=%q subscription=%q", publisher.projectID, publisher.subscriptionID)
	}
	if !reflect.DeepEqual(publisher.payload, payload) {
		t.Fatalf("unexpected publish payload %#v", publisher.payload)
	}
}

func TestStringifyDataPointValueUsesJSONForStructuredValues(t *testing.T) {
	actual := stringifyDataPointValue(map[string]any{"running": true, "value": float64(42)})
	if actual != `{"running":true,"value":42}` {
		t.Fatalf("stringifyDataPointValue() = %q", actual)
	}
}

func float64Pointer(value float64) *float64 { return &value }

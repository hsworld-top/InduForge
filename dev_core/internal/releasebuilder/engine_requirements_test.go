package releasebuilder

import (
	"reflect"
	"testing"
)

func TestDeriveEngineRequirements(t *testing.T) {
	tests := []struct {
		name     string
		document map[string]any
		want     []string
	}{
		{"empty and display text", map[string]any{"alarmItems": []any{}, "displayName": "alarm expression", "computeUnits": []any{}}, []string{"base"}},
		{"compute output", map[string]any{"dataPoints": []any{map[string]any{"sourceType": "calc.output"}}}, []string{"base", "compute"}},
		{"alarm", map[string]any{"alarmItems": []any{map[string]any{"rule": "x>1"}}}, []string{"base", "alarm"}},
		{"collector", map[string]any{"dataPoints": []any{map[string]any{"sourceType": "collector.opcua"}}}, []string{"base", "collector"}},
		{"all", map[string]any{"computeUnits": []any{map[string]any{"id": "c"}}, "alarmItems": []any{map[string]any{"id": "a"}}, "dataPoints": []any{map[string]any{"sourceType": "collector.modbus"}}}, []string{"base", "compute", "alarm", "collector"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := DeriveEngineRequirements(test.document); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("got %v want %v", got, test.want)
			}
		})
	}
}

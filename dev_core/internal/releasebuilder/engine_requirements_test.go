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
		{"no optional", map[string]any{"pages": []any{}}, []string{"base"}},
		{"compute", map[string]any{"datapoints": []any{map[string]any{"expression": "a+b"}}}, []string{"base", "compute"}},
		{"alarm", map[string]any{"alarms": []any{map[string]any{"rule": "x>1"}}}, []string{"base", "alarm"}},
		{"collector", map[string]any{"datapoints": []any{map[string]any{"acquisition": map[string]any{"source": "opc"}}}}, []string{"base", "collector"}},
		{"all", map[string]any{"computed": true, "alarms": []any{1}, "collector": map[string]any{"device": "x"}}, []string{"base", "compute", "alarm", "collector"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := DeriveEngineRequirements(test.document); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("got %v want %v", got, test.want)
			}
		})
	}
}

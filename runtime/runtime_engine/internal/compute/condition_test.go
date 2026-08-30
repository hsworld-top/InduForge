package compute

import (
	"encoding/json"
	"testing"
)

func TestEvaluateConditionUsesExactNumbers(t *testing.T) {
	values := map[string]json.RawMessage{"a": []byte(`9007199254740993`), "b": []byte(`9007199254740992`), "decimal": []byte(`1.0000000000000000000000000002`)}
	got, err := EvaluateCondition("a > b && decimal > 1.0000000000000000000000000001", values)
	if err != nil || !got {
		t.Fatalf("large comparison = %v, %v", got, err)
	}
	if _, err := EvaluateCondition("a > b", map[string]json.RawMessage{"a": []byte(`1 trailing`), "b": []byte(`0`)}); err == nil {
		t.Fatal("trailing input accepted")
	}
}

func TestOutputTypeBoundaries(t *testing.T) {
	if !matchesDataType([]byte(`"AQI="`), "bytes") || matchesDataType([]byte(`"not base64!"`), "bytes") {
		t.Fatal("bytes base64 boundary")
	}
	if !matchesDataType([]byte(`3.40282346638528859811704183484516925440e38`), "float32") || matchesDataType([]byte(`3.5e38`), "float32") {
		t.Fatal("float32 range")
	}
	if !matchesDataType([]byte(`1.7976931348623157e308`), "float64") || matchesDataType([]byte(`1e10000`), "float64") {
		t.Fatal("float64 range")
	}
	if matchesDataType([]byte(`9223372036854775808`), "int64") || !matchesDataType([]byte(`18446744073709551615`), "uint64") {
		t.Fatal("integer range")
	}
}

package repository

import "testing"

func TestJSONPayloadEqualKeepsAdjacentLargeIntegersDistinct(t *testing.T) {
	if jsonPayloadEqual([]byte(`{"value":9007199254740992}`), []byte(`{"value":9007199254740993}`)) {
		t.Fatal("相邻的大整数不能因 float64 精度而被视作相等")
	}
	if !jsonPayloadEqual([]byte(`{"value":1.0}`), []byte(`{"value":1}`)) {
		t.Fatal("数值等价 JSON 应被视作相等")
	}
}

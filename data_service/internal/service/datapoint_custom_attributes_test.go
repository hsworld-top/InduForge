package service

import "testing"

func TestNormalizeDataPointAttributeDefaults(t *testing.T) {
	t.Run("accepts string values and trims keys", func(t *testing.T) {
		result, err := normalizeDataPointAttributeDefaults(map[string]string{
			" asset_code ": " PUMP-001 ",
			"line.no":      "03",
		})
		if err != nil {
			t.Fatalf("normalize attributes: %v", err)
		}
		if result["asset_code"] != " PUMP-001 " || result["line.no"] != "03" {
			t.Fatalf("unexpected normalized attributes: %#v", result)
		}
	})

	t.Run("nil becomes empty object", func(t *testing.T) {
		result, err := normalizeDataPointAttributeDefaults(nil)
		if err != nil {
			t.Fatalf("normalize nil attributes: %v", err)
		}
		if result == nil || len(result) != 0 {
			t.Fatalf("expected empty map, got %#v", result)
		}
	})

	t.Run("rejects reserved key", func(t *testing.T) {
		_, err := normalizeDataPointAttributeDefaults(map[string]string{"quality": "custom"})
		if err == nil || err.Error() != "属性 Key“quality”已被内置属性占用" {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("rejects invalid key", func(t *testing.T) {
		_, err := normalizeDataPointAttributeDefaults(map[string]string{"Asset Code": "PUMP-001"})
		if err == nil {
			t.Fatal("expected invalid key error")
		}
	})
}

package repository

import "testing"

func TestNormalizeGeneratedDataPointPath(t *testing.T) {
	for input, want := range map[string]string{" /a/b/ ": "a.b", "a\\b": "a.b", "a..b": "a..b"} {
		if got := normalizeGeneratedDataPointPath(input); got != want {
			t.Fatalf("normalize(%q)=%q, want %q", input, got, want)
		}
	}
}

func TestGeneratedDataPointSourceConfigKey(t *testing.T) {
	if got := generatedDataPointSourceConfigKey("mqtt.tag"); got != "tagId" {
		t.Fatalf("got %q", got)
	}
	if got := generatedDataPointSourceConfigKey("unknown"); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestBuildV2GeneratedDataPointPath(t *testing.T) {
	got, err := buildV2GeneratedDataPointPath("mqtt.subscription", "sub1", "temperature")
	if err != nil || got != "mqtt.sub1.temperature" {
		t.Fatalf("got %q, err %v", got, err)
	}
	if _, err := buildV2GeneratedDataPointPath("bad", "x", "y"); err == nil {
		t.Fatal("unknown source should fail")
	}
}

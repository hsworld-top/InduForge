package repository

import "testing"

func TestNormalizeGeneratedDataPointPath(t *testing.T) {
	for input, want := range map[string]string{" /a/b/ ": "a.b", "a\\b": "a.b", "a..b": "a..b"} {
		if got := normalizeGeneratedDataPointPath(input); got != want { t.Fatalf("normalize(%q)=%q, want %q", input, got, want) }
	}
}

func TestGeneratedDataPointSourceConfigKey(t *testing.T) {
	if got := generatedDataPointSourceConfigKey("mqtt.tag"); got != "tagId" { t.Fatalf("got %q", got) }
	if got := generatedDataPointSourceConfigKey("unknown"); got != "" { t.Fatalf("got %q", got) }
}

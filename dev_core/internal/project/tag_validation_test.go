package project

import (
	"strings"
	"testing"
)

func TestTagNameValidation(t *testing.T) {
	for _, name := range []string{"产线 A", "温度/压力", strings.Repeat("标", 32)} {
		if err := validateTagName(name); err != nil {
			t.Fatalf("valid name %q: %v", name, err)
		}
	}
	for _, name := range []string{"", strings.Repeat("标", 33), "a\nb"} {
		if validateTagName(name) == nil {
			t.Fatalf("accepted invalid name %q", name)
		}
	}
}
func TestTagSelectionDeduplicatesBeforeLimit(t *testing.T) {
	ids, err := normalizeTagIDs([]string{" a ", "a", "", "b"})
	if err != nil || len(ids) != 2 || ids[0] != "a" {
		t.Fatalf("unexpected tags: %v %v", ids, err)
	}
	tooMany := []string{}
	for i := 0; i < 11; i++ {
		tooMany = append(tooMany, string(rune('a'+i)))
	}
	if _, err := normalizeTagIDs(tooMany); err == nil {
		t.Fatal("accepted more than ten tags")
	}
}

package config

import "testing"

func TestStrictSemVer(t *testing.T) {
	for _, value := range []string{"1.0.0", "0.1.2", "1.2.3-rc.1+build.7"} {
		if !strictSemVer(value) {
			t.Fatalf("合法版本被拒绝: %s", value)
		}
	}
	for _, value := range []string{"", "1.0", "v1.0.0", "01.0.0", "1.0.0-", "1.0.0-01"} {
		if strictSemVer(value) {
			t.Fatalf("非法版本被接受: %s", value)
		}
	}
}

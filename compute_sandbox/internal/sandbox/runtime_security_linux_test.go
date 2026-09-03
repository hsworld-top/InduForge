//go:build linux

package sandbox

import "testing"

func TestValidateExecutionSecurityRejectsPrivilegeLeak(t *testing.T) {
	valid := executionSecurity{UID: runtimeUID, GID: runtimeUID, NoNewPrivs: true}
	if err := validateExecutionSecurity(valid); err != nil {
		t.Fatalf("valid isolated child rejected: %v", err)
	}
	tests := []executionSecurity{
		{UID: 0, GID: runtimeUID, NoNewPrivs: true},
		{UID: runtimeUID, GID: 0, NoNewPrivs: true},
		{UID: runtimeUID, GID: runtimeUID, Effective: 1, NoNewPrivs: true},
		{UID: runtimeUID, GID: runtimeUID, Permitted: 1, NoNewPrivs: true},
		{UID: runtimeUID, GID: runtimeUID, Inheritable: 1, NoNewPrivs: true},
		{UID: runtimeUID, GID: runtimeUID, Bounding: 1, NoNewPrivs: true},
		{UID: runtimeUID, GID: runtimeUID, NoNewPrivs: false},
	}
	for index, value := range tests {
		if err := validateExecutionSecurity(value); err == nil {
			t.Fatalf("unsafe child %d accepted: %+v", index, value)
		}
	}
}

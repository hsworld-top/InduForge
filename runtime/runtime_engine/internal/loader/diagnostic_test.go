package loader

import (
	"errors"
	"testing"
)

func TestDiagnosticCodeDoesNotExposeInput(t *testing.T) {
	if got := DiagnosticCode(errors.New("项目 Artifact 原始字节 SHA-256 不匹配: 期望 secret-value")); got != "project-artifact" {
		t.Fatalf("DiagnosticCode()=%q", got)
	}
}

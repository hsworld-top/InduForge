package dataservice

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestInternalClientEnsureProjectTenantBindingUsesInternalToken(t *testing.T) {
	const token = "internal-test-token"
	projectID, tenantID := uuid.NewString(), uuid.NewString()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.Header.Get("X-InduForge-Internal-Token") != token || r.Header.Get("Authorization") != "" {
			t.Fatal("internal client did not use the expected authentication boundary")
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), tenantID) {
			t.Fatal("tenant binding request body is invalid")
		}
		_, _ = w.Write([]byte(`{"code":0,"msg":"success","data":{"created":true},"reqId":"data-request"}`))
	}))
	defer server.Close()
	client, err := NewInternalClient(server.URL, token)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.EnsureProjectTenantBinding(context.Background(), projectID, tenantID); err != nil {
		t.Fatal(err)
	}
}

func TestInternalClientDoesNotLeakTokenOnFailure(t *testing.T) {
	const token = "internal-secret-must-not-leak"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"code":10002,"msg":"内部认证失败","data":null,"reqId":"data-request"}`))
	}))
	defer server.Close()
	client, err := NewInternalClient(server.URL, token)
	if err != nil {
		t.Fatal(err)
	}
	err = client.EnsureProjectTenantBinding(context.Background(), uuid.NewString(), uuid.NewString())
	if err == nil || strings.Contains(err.Error(), token) {
		t.Fatalf("failure leaked token: %v", err)
	}
}

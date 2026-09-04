package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/indu-forge/data_service/internal/auth"
	"github.com/indu-forge/data_service/internal/repository"
)

type epochGateStub struct{ calls []string }

func (s *epochGateStub) ResolveQueryProject(_ context.Context, _, _ string) (string, error) {
	return "", context.Canceled
}

func (s *epochGateStub) GateNormalWrite(_ context.Context, projectID, _ string, expected string) (func(), error) {
	s.calls = append(s.calls, expected)
	if expected != "epoch-2" {
		return nil, &repository.AuthoringEpochConflict{ProjectID: projectID, Current: 2}
	}
	return func() {}, nil
}

func TestAuthoringFenceGuardRequiresCurrentEpochAndReturnsReloadContext(t *testing.T) {
	const secret = "epoch-middleware-secret"
	validator, err := auth.NewJWTValidator(secret)
	if err != nil {
		t.Fatal(err)
	}
	projectID := "11111111-1111-4111-8111-111111111111"
	gate := &epochGateStub{}
	nextCalls := 0
	handler := AuthoringFenceGuard(validator, gate)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalls++
		w.WriteHeader(http.StatusNoContent)
	}))
	token := signEpochTestJWT(t, secret, map[string]any{"userId": "user", "tenantId": "tenant", "projectIds": []string{projectID}})

	for _, epoch := range []string{"", "broken", "epoch-1"} {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/data/projects/"+projectID+"/connections", nil)
		request.Header.Set("Authorization", "Bearer "+token)
		request.Header.Set("X-InduForge-Authoring-Epoch", epoch)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusConflict {
			t.Fatalf("epoch %q status=%d", epoch, recorder.Code)
		}
		var envelope struct {
			Data map[string]any `json:"data"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil || envelope.Data["projectId"] != projectID || envelope.Data["currentAuthoringEpoch"] != "epoch-2" || envelope.Data["action"] != "reload" {
			t.Fatalf("epoch %q response=%s err=%v", epoch, recorder.Body.String(), err)
		}
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/data/projects/"+projectID+"/connections", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("X-InduForge-Authoring-Epoch", "epoch-2")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent || nextCalls != 1 {
		t.Fatalf("current epoch did not reach handler: status=%d calls=%d", recorder.Code, nextCalls)
	}
}

func signEpochTestJWT(t *testing.T, secret string, payload map[string]any) string {
	t.Helper()
	encode := func(value any) string {
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		return base64.RawURLEncoding.EncodeToString(raw)
	}
	signed := encode(map[string]any{"alg": "HS256", "typ": "JWT"}) + "." + encode(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(signed))
	return signed + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

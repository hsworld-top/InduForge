package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/indu-forge/dev_core/internal/objectstore"
)

type avatarObjects struct {
	files map[string][]byte
	fail  bool
}

func (s *avatarObjects) Put(_ context.Context, key string, reader io.Reader, size int64, contentType string) (objectstore.ObjectRef, error) {
	if s.fail {
		return objectstore.ObjectRef{}, errors.New("storage unavailable")
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return objectstore.ObjectRef{}, err
	}
	s.files[key] = data
	return objectstore.ObjectRef{Key: key, Size: size, ContentType: contentType}, nil
}
func (s *avatarObjects) Open(_ context.Context, key string) (objectstore.ObjectReader, error) {
	data, ok := s.files[key]
	if !ok {
		return objectstore.ObjectReader{}, errors.New("missing")
	}
	return objectstore.ObjectReader{Reader: io.NopCloser(bytes.NewReader(data)), Size: int64(len(data)), ContentType: "image/png"}, nil
}
func (s *avatarObjects) Delete(_ context.Context, key string) error { delete(s.files, key); return nil }

func avatarRequest(t *testing.T, server http.Handler, cookie *http.Cookie, data []byte, extra bool) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "avatar.png")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write(data)
	if extra {
		_ = writer.WriteField("userId", "another-user")
	}
	_ = writer.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/me/avatar", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if cookie != nil {
		req.AddCookie(cookie)
	}
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, req)
	return recorder
}
func TestAvatarUploadAndPrivateRead(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 400, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 400; x++ {
			source.SetNRGBA(x, y, color.NRGBA{20, 100, 160, 255})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, source); err != nil {
		t.Fatal(err)
	}
	for _, platform := range []bool{false, true} {
		store := &avatarObjects{files: map[string][]byte{}}
		server, repo := newAuthServer(t, store)
		body := loginBody("admin123")
		if platform {
			repo.user.Role = "SUPER_ADMIN"
			repo.user.TenantID = ""
			body["platform"] = true
			delete(body, "tenantCode")
		}
		_, login := performJSONResponse(t, server, http.MethodPost, "/api/v1/auth/login", body, "", nil)
		cookie := responseCookie(t, login, "if_access")
		if got := avatarRequest(t, server, nil, encoded.Bytes(), false); got.Code != http.StatusUnauthorized {
			t.Fatalf("anonymous upload: %d", got.Code)
		}
		for _, invalid := range [][]byte{[]byte("<svg><script/></svg>"), []byte("not an image"), encoded.Bytes()[:20], make([]byte, 5<<20+1)} {
			got := avatarRequest(t, server, cookie, invalid, false)
			if got.Code != http.StatusBadRequest {
				t.Fatalf("invalid image: %d %s", got.Code, got.Body.String())
			}
		}
		if got := avatarRequest(t, server, cookie, encoded.Bytes(), true); got.Code != http.StatusBadRequest {
			t.Fatal("must reject submitted user identity")
		}
		for n := 0; n < 2; n++ {
			got := avatarRequest(t, server, cookie, encoded.Bytes(), false)
			var payload envelope
			if err := json.Unmarshal(got.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			assertSuccess(t, payload)
			url := payload.Data["avatarUrl"].(string)
			if !strings.HasPrefix(url, "/api/v1/auth/me/avatar?v=") {
				t.Fatalf("private URL: %s", url)
			}
			if len(store.files) != 1 {
				t.Fatalf("old objects not cleaned: %d", len(store.files))
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()
			server.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Fatal("anonymous avatar read must fail")
			}
			req.AddCookie(cookie)
			rec = httptest.NewRecorder()
			server.ServeHTTP(rec, req)
			img, err := png.Decode(bytes.NewReader(rec.Body.Bytes()))
			if err != nil {
				t.Fatal(err)
			}
			if img.Bounds().Dx() != 256 || img.Bounds().Dy() != 256 || rec.Header().Get("Cache-Control") != "private, no-store" {
				t.Fatal("image size or cache policy invalid")
			}
			me, _ := performJSONResponse(t, server, http.MethodGet, "/api/v1/auth/me", nil, "", []*http.Cookie{cookie})
			if me.Data["avatarUrl"] != url {
				t.Fatal("reload must retain avatar")
			}
		}
		old := repo.user.Avatar
		store.fail = true
		if got := avatarRequest(t, server, cookie, encoded.Bytes(), false); got.Code != http.StatusInternalServerError {
			t.Fatal("storage failure must not succeed")
		}
		if repo.user.Avatar != old {
			t.Fatal("failed upload changed avatar")
		}
		cleared, _ := performJSONResponse(t, server, http.MethodDelete, "/api/v1/auth/me/avatar", nil, "", []*http.Cookie{cookie})
		assertSuccess(t, cleared)
		if cleared.Data["avatarUrl"] != "" || repo.user.Avatar != "" || len(store.files) != 0 {
			t.Fatal("reset must clear reference and object")
		}

	}
}

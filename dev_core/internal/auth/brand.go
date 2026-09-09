package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
	"golang.org/x/image/draw"
	"image"
	"io"
	"math/big"
	"net/http"
	"strings"
)

type brandRepository interface {
	GetTenantBranding(context.Context, string) (TenantBranding, error)
}

func (h *Handler) brandAsset(w http.ResponseWriter, r *http.Request) {
	repo, ok := h.service.repository.(brandRepository)
	if !ok || h.service.avatars == nil {
		platformapi.WriteError(w, r, 404, platformapi.ErrorCodeInvalidRequest, "图片不存在")
		return
	}
	brand, err := repo.GetTenantBranding(r.Context(), r.URL.Query().Get("tenantCode"))
	key := r.URL.Query().Get("asset")
	allowed := false
	for _, kind := range []string{"logo", "background", "captcha"} {
		allowed = allowed || strings.HasPrefix(key, "tenants/"+brand.ID+"/"+kind+"/")
	}
	if err != nil || !allowed || strings.Contains(key, "..") {
		platformapi.WriteError(w, r, 404, platformapi.ErrorCodeInvalidRequest, "图片不存在")
		return
	}
	object, err := h.service.avatars.Open(r.Context(), key)
	if err != nil {
		platformapi.WriteError(w, r, 404, platformapi.ErrorCodeInvalidRequest, "图片不存在")
		return
	}
	defer object.Reader.Close()
	w.Header().Set("Content-Type", object.ContentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
	w.Header().Set("Cache-Control", "private, max-age=300")
	_, _ = io.Copy(w, object.Reader)
}

// 自定义背景读取失败时使用内置素材；只允许当前组织的图片，限制解码体积。
func (s *Service) captchaBackground(ctx context.Context, code string) image.Image {
	repo, ok := s.repository.(brandRepository)
	if !ok || s.avatars == nil {
		return nil
	}
	brand, err := repo.GetTenantBranding(ctx, code)
	if err != nil || len(brand.CaptchaBackgrounds) == 0 {
		return nil
	}
	pick, err := rand.Int(rand.Reader, big.NewInt(int64(len(brand.CaptchaBackgrounds))))
	if err != nil {
		return nil
	}
	key := brand.CaptchaBackgrounds[pick.Int64()]
	if !strings.HasPrefix(key, "tenants/"+brand.ID+"/captcha/") {
		return nil
	}
	object, err := s.avatars.Open(ctx, key)
	if err != nil {
		return nil
	}
	defer object.Reader.Close()
	raw, err := io.ReadAll(io.LimitReader(object.Reader, 5<<20+1))
	if err != nil || len(raw) > 5<<20 {
		return nil
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > 16_000_000 {
		return nil
	}
	source, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil
	}
	target := image.NewNRGBA(image.Rect(0, 0, captchaWidth, captchaHeight))
	draw.CatmullRom.Scale(target, target.Bounds(), source, source.Bounds(), draw.Src, nil)
	return target
}

package auth

import (
	"bytes"
	"context"
	"fmt"
	"github.com/indu-forge/dev_core/internal/objectstore"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"testing"
)

func TestCustomCaptchaBackgroundProducesPuzzle(t *testing.T) {
	background := image.NewNRGBA(image.Rect(0, 0, 300, 180))
	for i := 0; i < 3; i++ {
		result, target, err := generatePuzzle(background)
		if err != nil || target <= 0 || result["image"] == "" || result["thumb"] == "" {
			t.Fatalf("invalid custom puzzle: %v %d", err, target)
		}
	}
}

type brandTestRepo struct {
	Repository
	brand TenantBranding
}

func (r brandTestRepo) GetTenantBranding(context.Context, string) (TenantBranding, error) {
	return r.brand, nil
}

type brandTestStore struct {
	AvatarStore
	files map[string][]byte
}

func (s brandTestStore) Open(_ context.Context, key string) (objectstore.ObjectReader, error) {
	raw, ok := s.files[key]
	if !ok {
		return objectstore.ObjectReader{}, fmt.Errorf("missing")
	}
	return objectstore.ObjectReader{Reader: io.NopCloser(bytes.NewReader(raw)), ContentType: "image/png"}, nil
}
func TestCaptchaUsesTenantGalleryAndFallsBack(t *testing.T) {
	files := map[string][]byte{}
	keys := []string{"tenants/one/captcha/red.png", "tenants/one/captcha/blue.png"}
	for i, key := range keys {
		img := image.NewNRGBA(image.Rect(0, 0, 300, 180))
		c := color.NRGBA{R: 255, A: 255}
		if i == 1 {
			c = color.NRGBA{B: 255, A: 255}
		}
		draw.Draw(img, img.Bounds(), &image.Uniform{C: c}, image.Point{}, draw.Src)
		var buf bytes.Buffer
		_ = png.Encode(&buf, img)
		files[key] = buf.Bytes()
	}
	service := Service{repository: brandTestRepo{brand: TenantBranding{ID: "one", CaptchaBackgrounds: keys}}, avatars: brandTestStore{files: files}}
	for i := 0; i < 10; i++ {
		img := service.captchaBackground(context.Background(), "one")
		if img == nil {
			t.Fatal("configured background missing")
		}
		r, g, b, _ := img.At(0, 0).RGBA()
		if g != 0 || (r == 0 && b == 0) {
			t.Fatal("custom background ignored")
		}
	}
	service.repository = brandTestRepo{brand: TenantBranding{ID: "two", CaptchaBackgrounds: keys}}
	if service.captchaBackground(context.Background(), "two") != nil {
		t.Fatal("cross tenant background accepted")
	}
}

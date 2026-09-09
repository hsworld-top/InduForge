package auth

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/indu-forge/dev_core/internal/objectstore"
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const maxAvatarBytes = 5 << 20

type AvatarStore interface {
	Put(context.Context, string, io.Reader, int64, string) (objectstore.ObjectRef, error)
	Open(context.Context, string) (objectstore.ObjectReader, error)
	Delete(context.Context, string) error
}

func (s *Service) SetAvatarStore(store AvatarStore) { s.avatars = store }

func avatarURL(user User) string {
	if user.Avatar == "" {
		return ""
	}
	digest := sha256.Sum256([]byte(user.ID + ":" + user.Avatar))
	return fmt.Sprintf("/api/v1/auth/me/avatar?v=%x", digest[:12])
}

// 先检查文件与像素上限，再解码并居中裁剪；重新编码移除元数据和非图片载荷。
func prepareAvatar(data []byte) ([]byte, error) {
	if len(data) == 0 || len(data) > maxAvatarBytes {
		return nil, errors.New("头像文件不能超过 5MB")
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "jpeg" && format != "png" && format != "webp") {
		return nil, errors.New("请选择有效的 PNG、JPG 或 WebP 图片")
	}
	if config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > 16_000_000 {
		return nil, errors.New("头像图片不能超过 1600 万像素")
	}
	source, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, errors.New("图片损坏或无法读取")
	}
	bounds := source.Bounds()
	size := min(bounds.Dx(), bounds.Dy())
	x, y := bounds.Min.X+(bounds.Dx()-size)/2, bounds.Min.Y+(bounds.Dy()-size)/2
	target := image.NewNRGBA(image.Rect(0, 0, 256, 256))
	draw.CatmullRom.Scale(target, target.Bounds(), source, image.Rect(x, y, x+size, y+size), draw.Src, nil)
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, target); err != nil {
		return nil, err
	}
	return encoded.Bytes(), nil
}

func (h *Handler) UploadCurrentUserAvatar(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAvatarBytes+(64<<10))
	if err := r.ParseMultipartForm(maxAvatarBytes + (64 << 10)); err != nil {
		platformapi.WriteError(w, r, http.StatusBadRequest, platformapi.ErrorCodeInvalidRequest, "请选择不超过 5MB 的图片")
		return
	}
	defer r.MultipartForm.RemoveAll()
	if len(r.MultipartForm.Value) != 0 || len(r.MultipartForm.File) != 1 || len(r.MultipartForm.File["file"]) != 1 {
		platformapi.WriteError(w, r, http.StatusBadRequest, platformapi.ErrorCodeInvalidRequest, "请仅上传一张头像图片")
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxAvatarBytes+1))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	encoded, err := prepareAvatar(data)
	if err != nil {
		platformapi.WriteError(w, r, http.StatusBadRequest, platformapi.ErrorCodeInvalidRequest, err.Error())
		return
	}
	if h.service.avatars == nil {
		h.writeError(w, r, errors.New("头像存储未配置"))
		return
	}
	key := "users/" + user.ID + "/avatars/" + uuid.NewString() + ".png"
	if _, err := h.service.avatars.Put(r.Context(), key, bytes.NewReader(encoded), int64(len(encoded)), "image/png"); err != nil {
		h.writeError(w, r, err)
		return
	}
	old, err := h.service.repository.UpdateAvatar(r.Context(), user.ID, key)
	if err != nil {
		h.service.deleteAvatar(r.Context(), user.ID, key)
		h.writeError(w, r, err)
		return
	}
	h.service.deleteAvatar(r.Context(), user.ID, old)
	user.Avatar = key
	platformapi.WriteSuccess(w, r, publicUser(user))
}

func (s *Service) deleteAvatar(ctx context.Context, userID, key string) {
	// 只清理本账号的受控目录，避免旧字段中的任意路径影响其他资产。
	if !strings.HasPrefix(key, "users/"+userID+"/avatars/") {
		return
	}
	if err := s.avatars.Delete(ctx, key); err != nil {
		slog.Warn("清理被替换的头像失败", "userId", userID, "error", err)
	}
}

func (h *Handler) GetCurrentUserAvatar(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	if user.Avatar == "" || !strings.HasPrefix(user.Avatar, "users/"+user.ID+"/avatars/") {
		platformapi.WriteError(w, r, http.StatusNotFound, platformapi.ErrorCodeNotFound, "尚未设置头像")
		return
	}
	if h.service.avatars == nil {
		h.writeError(w, r, errors.New("头像存储未配置"))
		return
	}
	object, err := h.service.avatars.Open(r.Context(), user.Avatar)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	defer object.Reader.Close()
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = io.Copy(w, object.Reader)
}

func (h *Handler) DeleteCurrentUserAvatar(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	if h.service.avatars == nil {
		h.writeError(w, r, errors.New("头像存储未配置"))
		return
	}
	old, err := h.service.repository.UpdateAvatar(r.Context(), user.ID, "")
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.service.deleteAvatar(r.Context(), user.ID, old)
	user.Avatar = ""
	platformapi.WriteSuccess(w, r, publicUser(user))
}

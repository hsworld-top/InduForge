package auth

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/indu-forge/dev_core/internal/objectstore"
	"io"
	"net"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"unicode/utf8"

	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetAuthCaptcha(w http.ResponseWriter, r *http.Request, params platformapi.GetAuthCaptchaParams) {
	tenantCode := ""
	if params.TenantCode != nil {
		tenantCode = *params.TenantCode
	}
	result, err := h.service.Captcha(r.Context(), strings.TrimSpace(params.Username), strings.TrimSpace(tenantCode), clientIP(r), r.URL.Query().Get("platform") == "true")
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, result)
}

func (h *Handler) GetAuthConfig(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("asset") != "" {
		h.brandAsset(w, r)
		return
	}
	tenantCode := strings.TrimSpace(r.URL.Query().Get("tenantCode"))
	if r.URL.Query().Get("platform") == "true" {
		tenantCode = ""
	}
	result := h.service.Config(r.Context(), tenantCode)
	if r.URL.Query().Get("platform") == "true" {
		result["name"], result["title"], result["appName"] = "InduFrame", "InduFrame", "InduFrame"
	}
	platformapi.WriteSuccess(w, r, result)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Platform          bool   `json:"platform"`
		RememberMe        bool   `json:"rememberMe"`
		Username          string `json:"username"`
		Password          string `json:"password"`
		TenantCode        string `json:"tenantCode"`
		SliderChallengeID string `json:"sliderChallengeId"`
		SliderOffset      int    `json:"sliderOffset"`
	}
	if err := decodeJSON(r, &body); err != nil {
		platformapi.WriteError(w, r, http.StatusBadRequest, platformapi.ErrorCodeInvalidRequest, "请求体格式错误")
		return
	}
	result, err := h.service.Login(r.Context(), LoginInput{
		Platform: body.Platform, RememberMe: body.RememberMe, Username: body.Username, Password: body.Password, TenantCode: body.TenantCode,
		SliderChallengeID: body.SliderChallengeID, SliderOffset: body.SliderOffset, LoginIP: clientIP(r),
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	if id, ok := result.User["id"].(string); ok {
		tenant, _ := result.User["tenantId"].(string)
		if tenant != "" {
			*r = *r.WithContext(WithUser(r.Context(), User{ID: id, TenantID: tenant, Username: body.Username}))
		}
	}
	h.setSessionCookies(w, r, result.TokenPair)
	platformapi.WriteSuccess(w, r, map[string]any{"user": result.User})
}

func (h *Handler) RefreshAuthToken(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		h.clearSessionCookies(w, r)
		platformapi.WriteError(w, r, http.StatusUnauthorized, platformapi.ErrorCodeTokenRequired, "缺少刷新会话")
		return
	}
	result, err := h.service.Refresh(r.Context(), strings.TrimSpace(cookie.Value))
	if err != nil {
		h.clearSessionCookies(w, r)
		h.writeError(w, r, err)
		return
	}
	h.setSessionCookies(w, r, result)
	platformapi.WriteSuccess(w, r, map[string]any{})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	refreshToken := ""
	if cookie, err := r.Cookie(refreshCookieName); err == nil {
		refreshToken = cookie.Value
	}
	if err := h.service.Logout(r.Context(), user.ID, refreshToken); err != nil {
		h.writeError(w, r, err)
		return
	}
	if err := h.service.RevokeAccessToken(r.Context(), AccessTokenFromRequest(r)); err != nil {
		h.writeError(w, r, err)
		return
	}
	h.clearSessionCookies(w, r)
	platformapi.WriteSuccess(w, r, map[string]any{})
}

func (h *Handler) sessionCookie(r *http.Request, name, value string, maxAge int) *http.Cookie {
	secure := r.TLS != nil || strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https")
	return &http.Cookie{Name: name, Value: value, Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, MaxAge: maxAge}
}

func (h *Handler) setSessionCookies(w http.ResponseWriter, r *http.Request, pair TokenPair) {
	accessAge, refreshAge := 0, 0
	if pair.RememberMe {
		accessAge, refreshAge = int(pair.ExpiresIn), int(pair.RefreshExpiresIn)
	}
	http.SetCookie(w, h.sessionCookie(r, accessCookieName, pair.AccessToken, accessAge))
	// 刷新令牌的过期时间由服务端存储校验；浏览器 Cookie 仅作为同源会话载体。
	http.SetCookie(w, h.sessionCookie(r, refreshCookieName, pair.RefreshToken, refreshAge))
}

func (h *Handler) clearSessionCookies(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, h.sessionCookie(r, accessCookieName, "", -1))
	http.SetCookie(w, h.sessionCookie(r, refreshCookieName, "", -1))
}

func (h *Handler) GetCurrentAuthUser(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	platformapi.WriteSuccess(w, r, publicUser(user))
}

func (h *Handler) ChangeAuthPassword(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}
	if err := decodeJSON(r, &body); err != nil {
		platformapi.WriteError(w, r, http.StatusBadRequest, platformapi.ErrorCodeInvalidRequest, "请求体格式错误")
		return
	}
	if err := h.service.ChangePassword(r.Context(), user, body.OldPassword, body.NewPassword); err != nil {
		h.writeError(w, r, err)
		return
	}
	h.clearSessionCookies(w, r)
	platformapi.WriteSuccess(w, r, map[string]any{})
}

func (h *Handler) requireUser(w http.ResponseWriter, r *http.Request) (User, bool) {
	if user, ok := UserFromContext(r.Context()); ok {
		return user, true
	}
	token := AccessTokenFromRequest(r)
	if token == "" {
		platformapi.WriteError(w, r, http.StatusUnauthorized, platformapi.ErrorCodeTokenRequired, "缺少访问令牌")
		return User{}, false
	}
	user, err := h.service.Authenticate(r.Context(), token)
	if err != nil {
		platformapi.WriteError(w, r, http.StatusUnauthorized, platformapi.ErrorCodeTokenInvalid, "访问令牌无效或已过期")
		return User{}, false
	}
	return user, true
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrInvalidCaptcha):
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeInvalidCaptcha, ErrInvalidCaptcha.Error())
	case errors.Is(err, ErrTenantRequired):
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeTenantRequired, ErrTenantRequired.Error())
	case errors.Is(err, ErrInvalidCredentials), errors.Is(err, ErrInvalidRefresh):
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeInvalidCredentials, err.Error())
	case errors.Is(err, ErrForbidden):
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeUserUnavailable, err.Error())
	case errors.Is(err, ErrUnauthorized):
		platformapi.WriteError(w, r, http.StatusUnauthorized, platformapi.ErrorCodeTokenInvalid, ErrUnauthorized.Error())
	default:
		platformapi.WriteError(w, r, http.StatusInternalServerError, platformapi.ErrorCodeInternal, "系统内部错误")
	}
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("请求体只能包含一个 JSON 值")
	}
	return nil
}

func bearerToken(header string) string {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

func clientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]); forwarded != "" {
		return forwarded
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func (h *Handler) ListAuthTenants(w http.ResponseWriter, r *http.Request, _ platformapi.ListAuthTenantsParams) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	repo, ok := h.service.repository.(interface {
		ListLoginTenants(context.Context, string, string, int, int) ([]TenantBranding, int64, error)
	})
	if !ok {
		h.writeError(w, r, ErrForbidden)
		return
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code != "" {
		limit = 1
	}
	items, total, err := repo.ListLoginTenants(r.Context(), strings.TrimSpace(r.URL.Query().Get("keyword")), code, page, limit)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	list := make([]map[string]any, 0, len(items))
	for _, item := range items {
		list = append(list, map[string]any{"id": item.ID, "code": item.Code, "name": item.Name, "logoUrl": objectstore.BrandURL(item.Code, item.LogoObjectKey), "loginBackgroundUrl": objectstore.BrandURL(item.Code, item.LoginBackgroundObjectKey)})
	}
	platformapi.WriteSuccess(w, r, map[string]any{"list": list, "pagination": map[string]any{"total": total, "page": page, "limit": limit, "pages": (total + int64(limit) - 1) / int64(limit)}})
}

// UpdateCurrentAuthUser 只接收个人资料；写入目标始终来自已认证会话，禁止提交身份或权限字段。
func (h *Handler) UpdateCurrentAuthUser(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var fields map[string]json.RawMessage
	if err := decodeJSON(r, &fields); err != nil || len(fields) != 2 {
		platformapi.WriteError(w, r, http.StatusBadRequest, platformapi.ErrorCodeInvalidRequest, "请仅提交姓名和联系邮箱")
		return
	}
	var fullName, email string
	for key, target := range map[string]*string{"fullName": &fullName, "email": &email} {
		raw, exists := fields[key]
		if !exists || json.Unmarshal(raw, target) != nil {
			platformapi.WriteError(w, r, http.StatusBadRequest, platformapi.ErrorCodeInvalidRequest, "姓名和联系邮箱必须为文本或空值")
			return
		}
		*target = strings.TrimSpace(*target)
	}
	if utf8.RuneCountInString(fullName) > 100 || utf8.RuneCountInString(email) > 254 {
		platformapi.WriteError(w, r, http.StatusBadRequest, platformapi.ErrorCodeInvalidRequest, "姓名最多 100 个字符，联系邮箱最多 254 个字符")
		return
	}
	if email != "" {
		address, err := mail.ParseAddress(email)
		if err != nil || address.Address != email || !strings.Contains(email, "@") {
			platformapi.WriteError(w, r, http.StatusBadRequest, platformapi.ErrorCodeInvalidRequest, "联系邮箱格式不正确")
			return
		}
	}
	if err := h.service.repository.UpdateProfile(r.Context(), user.ID, fullName, email); err != nil {
		h.writeError(w, r, err)
		return
	}
	user.FullName, user.Email = fullName, email
	platformapi.WriteSuccess(w, r, publicUser(user))
}

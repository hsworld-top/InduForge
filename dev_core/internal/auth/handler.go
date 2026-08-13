package auth

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"

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
	result, err := h.service.Captcha(r.Context(), strings.TrimSpace(params.Username), strings.TrimSpace(tenantCode), clientIP(r))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, result)
}

func (h *Handler) GetAuthConfig(w http.ResponseWriter, r *http.Request) {
	platformapi.WriteSuccess(w, r, h.service.Config(r.Context(), strings.TrimSpace(r.URL.Query().Get("tenantCode"))))
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
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
		Username: body.Username, Password: body.Password, TenantCode: body.TenantCode,
		SliderChallengeID: body.SliderChallengeID, SliderOffset: body.SliderOffset, LoginIP: clientIP(r),
	})
	if err != nil {
		h.writeError(w, r, err)
		return
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
	http.SetCookie(w, h.sessionCookie(r, accessCookieName, pair.AccessToken, int(pair.ExpiresIn)))
	// 刷新令牌的过期时间由服务端存储校验；浏览器 Cookie 仅作为同源会话载体。
	http.SetCookie(w, h.sessionCookie(r, refreshCookieName, pair.RefreshToken, 30*24*60*60))
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

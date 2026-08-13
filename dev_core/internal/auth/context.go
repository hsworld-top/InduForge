package auth

import (
	"context"
	"net/http"
	"strings"
)

type userContextKey struct{}

const (
	accessCookieName  = "if_access"
	refreshCookieName = "if_refresh"
)

// AccessTokenFromRequest 优先从同源 HttpOnly Cookie 读取浏览器会话。
// Authorization 仅保留给服务间调用和自动化测试，浏览器前端不再持有 JWT。
func AccessTokenFromRequest(r *http.Request) string {
	if r != nil {
		if cookie, err := r.Cookie(accessCookieName); err == nil && strings.TrimSpace(cookie.Value) != "" {
			return strings.TrimSpace(cookie.Value)
		}
	}
	return bearerToken(r.Header.Get("Authorization"))
}

// ForwardAuthorization 为 dev_core 到数据域的内部调用生成认证头，避免把 JWT 暴露给浏览器。
func ForwardAuthorization(r *http.Request) string {
	token := AccessTokenFromRequest(r)
	if token == "" {
		return ""
	}
	return "Bearer " + token
}

// ResolveUser 将有效访问令牌对应的用户放入请求上下文；缺失或无效令牌交由具体接口返回统一错误。
func ResolveUser(service *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := AccessTokenFromRequest(r)
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}
			user, err := service.Authenticate(r.Context(), token)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), user)))
		})
	}
}

func WithUser(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, userContextKey{}, user)
}

func UserFromContext(ctx context.Context) (User, bool) {
	user, ok := ctx.Value(userContextKey{}).(User)
	return user, ok
}

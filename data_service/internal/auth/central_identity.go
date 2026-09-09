package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

const identityLeaseInterval = 15 * time.Second

// ValidateCenterURL 只接受服务根地址；拒绝用户信息、查询串、路径和片段，避免发送凭据到错误入口。
func ValidateCenterURL(raw string) error {
	_, err := identityEndpoint(raw)
	return err
}

func identityEndpoint(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || (u.Path != "" && u.Path != "/") || u.ForceQuery || u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("DEV_CORE_URL 必须是 HTTP(S) 服务根地址")
	}
	u.Path = "/api/v1/auth/me"
	return u.String(), nil
}

func invalidIdentity(message string) error {
	return apperrors.NewAppError(apperrors.ErrorCodeAuthTokenInvalid, http.StatusUnauthorized, message)
}

func (v *JWTValidator) verifyIdentity(ctx context.Context, token string, claims *Claims) error {
	if v.client == nil || v.centerURL == "" {
		return invalidIdentity("中心身份复查未配置")
	}
	if claims.UserID == "" || claims.TenantID == "" || claims.Role == "SUPER_ADMIN" {
		return invalidIdentity("当前身份不能访问租户数据")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.centerURL, nil)
	if err != nil {
		return invalidIdentity("中心身份复查失败")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	res, err := v.client.Do(req)
	if err != nil {
		return invalidIdentity("中心身份复查不可用，请重新登录后重试")
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return invalidIdentity("登录状态已失效")
	}
	var result struct {
		Code *int `json:"code"`
		Data struct {
			ID                 string     `json:"id"`
			TenantID           string     `json:"tenantId"`
			Role               string     `json:"role"`
			Platform           bool       `json:"platform"`
			MustChangePassword *bool      `json:"mustChangePassword"`
			Status             string     `json:"status"`
			ExpiresAt          *time.Time `json:"expiresAt"`
			Tenant             *struct {
				ID        string     `json:"id"`
				Status    string     `json:"status"`
				ExpiresAt *time.Time `json:"expiresAt"`
			} `json:"tenant"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 64<<10)).Decode(&result); err != nil {
		return invalidIdentity("中心身份响应无效")
	}
	user := result.Data
	if result.Code == nil || *result.Code != 0 || user.ID != claims.UserID || user.TenantID != claims.TenantID || user.Role == "" || user.Role != claims.Role || user.Platform || user.Role == "SUPER_ADMIN" || user.MustChangePassword == nil || *user.MustChangePassword {
		return invalidIdentity("当前身份无权访问数据服务，请重新登录")
	}
	// 用户/租户状态与到期由中心鉴权负责；返回这些字段时仍做一致性检查，避免接受矛盾响应。
	now := time.Now()
	if (user.Status != "" && user.Status != "active") || (user.ExpiresAt != nil && !now.Before(*user.ExpiresAt)) {
		return invalidIdentity("用户已停用或过期")
	}
	if user.Tenant != nil && (user.Tenant.ID != claims.TenantID || (user.Tenant.Status != "" && user.Tenant.Status != "active") || (user.Tenant.ExpiresAt != nil && !now.Before(*user.Tenant.ExpiresAt))) {
		return invalidIdentity("租户状态已失效")
	}
	return nil
}

// WithIdentityLease 为已通过初次认证的长连接定期复查；调用方须在连接结束时取消。
// 不缓存授权结果：复查失败立即取消上下文，协议层负责关闭底层连接。
func (v *JWTValidator) WithIdentityLease(parent context.Context, token string) (context.Context, context.CancelFunc) {
	return v.withIdentityLease(parent, token, identityLeaseInterval)
}

func (v *JWTValidator) withIdentityLease(parent context.Context, token string, interval time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := v.ValidateContext(ctx, token); err != nil {
					cancel()
					return
				}
			}
		}
	}()
	return ctx, cancel
}

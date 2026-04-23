package auth

import "context"

type claimsContextKey struct{}

// Claims 表示 data_service 认证后解析出来的 JWT 业务信息。
type Claims struct {
	UserID       string   `json:"userId"`
	Role         string   `json:"role"`
	TenantID     string   `json:"tenantId"`
	ProjectIDs   []string `json:"projectIds"`
	Capabilities []string `json:"capabilities"`
}

func (c *Claims) hasGlobalAccessRole() bool {
	if c == nil {
		return false
	}

	switch c.Role {
	case "SYSTEM_ADMIN", "SUPER_ADMIN":
		return true
	default:
		return false
	}
}

// WithClaims 将解析后的 JWT 业务信息写入上下文。
func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey{}, claims)
}

// ClaimsFromContext 从上下文中读取 JWT 业务信息。
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	if ctx == nil {
		return nil, false
	}

	claims, ok := ctx.Value(claimsContextKey{}).(*Claims)
	return claims, ok && claims != nil
}

// HasCapability 判断当前 JWT 是否包含指定能力点。
func (c *Claims) HasCapability(required string) bool {
	if c == nil || required == "" {
		return false
	}
	if c.hasGlobalAccessRole() {
		return true
	}

	for _, capability := range c.Capabilities {
		if capability == "*" || capability == required {
			return true
		}
	}

	return false
}

// HasProjectAccess 判断当前 JWT 是否拥有指定项目的访问权。
func (c *Claims) HasProjectAccess(projectID string) bool {
	if c == nil || projectID == "" {
		return false
	}
	if c.hasGlobalAccessRole() {
		return true
	}

	for _, currentProjectID := range c.ProjectIDs {
		if currentProjectID == "*" || currentProjectID == projectID {
			return true
		}
	}

	return false
}

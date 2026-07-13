package auth

import "context"

type collectorAgentContextKey struct{}

// CollectorAgentIdentity 表示通过 Agent Token 认证后的租户级代理身份。
type CollectorAgentIdentity struct {
	AgentID  string
	TenantID string
}

func WithCollectorAgent(ctx context.Context, identity *CollectorAgentIdentity) context.Context {
	return context.WithValue(ctx, collectorAgentContextKey{}, identity)
}

func CollectorAgentFromContext(ctx context.Context) (*CollectorAgentIdentity, bool) {
	if ctx == nil {
		return nil, false
	}
	identity, ok := ctx.Value(collectorAgentContextKey{}).(*CollectorAgentIdentity)
	return identity, ok && identity != nil
}

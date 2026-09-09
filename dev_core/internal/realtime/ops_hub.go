package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/indu-forge/dev_core/internal/auth"
)

const (
	maxOpsClients       = 2048
	maxOpsSubscriptions = 8
	// 一个租户三秒内可能汇集上千节点的错峰心跳。ID 集合保持有界，但要避免
	// 千级部署频繁退化为 broad 通知，使只查看当前页的客户端全部回源。
	maxOpsEntityIDs      = 4096
	maxOpsPendingTenants = 1024
	maxOpsWatchRequests  = 12
	opsAuthExpired       = "AUTH_EXPIRED"
	opsAuthForbidden     = "AUTH_FORBIDDEN"
)

type opsWatch struct {
	SubscriptionID string   `json:"subscriptionId"`
	Topics         []string `json:"topics"`
	EntityIDs      []string `json:"entityIds,omitempty"`
}

type opsChange struct {
	Epoch     string   `json:"epoch"`
	Sequence  uint64   `json:"sequence"`
	Topics    []string `json:"topics"`
	EntityIDs []string `json:"entityIds,omitempty"`
	Timestamp string   `json:"timestamp"`
	Terminal  bool     `json:"terminal,omitempty"`
	Resync    bool     `json:"resync,omitempty"`
}

type opsPending struct {
	topics          map[string]bool
	ids             map[string]bool
	broad, terminal bool
	due             time.Time
}

type opsPacket struct {
	event   string
	payload any
}

type opsClient struct {
	token      string
	actor      auth.User
	expires    time.Time
	watches    map[string]opsWatch
	send       func(string, any) error
	disconnect func()
	window     time.Time
	requests   int
	writable   func() bool
	outbox     []opsPacket
	blockedAt  time.Time
	closing    bool
	authSentAt time.Time
}

type sessionValidator func(context.Context, string) (auth.User, time.Time, error)
type authCheck struct{ token string }

// opsHub 是单中心实例的有界失效通知总线。只携带ID，不携带节点消息、凭据或对象描述。
// 全部连接共用一个合并计时器和四个鉴权复核工作者；通知路径不查询数据库。
type opsHub struct {
	mu       sync.Mutex
	epoch    string
	sequence uint64
	clients  map[string]*opsClient
	pending  map[string]*opsPending
	resync   bool
	checks   map[string]time.Time
	checking map[string]bool
	validate sessionValidator
	wake     chan struct{}
	jobs     chan authCheck
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

func newOpsHub(validate sessionValidator) *opsHub {
	ctx, cancel := context.WithCancel(context.Background())
	h := &opsHub{epoch: uuid.NewString(), clients: map[string]*opsClient{}, pending: map[string]*opsPending{}, checks: map[string]time.Time{}, checking: map[string]bool{}, validate: validate, wake: make(chan struct{}, 1), jobs: make(chan authCheck, 128), ctx: ctx, cancel: cancel}
	h.wg.Add(5)
	go h.run()
	for i := 0; i < 4; i++ {
		go h.authWorker()
	}
	return h
}

func (h *opsHub) close() { h.cancel(); h.wg.Wait() }

func allowedOpsTopic(actor auth.User, topic string) bool {
	switch topic {
	case "nodes", "deployments":
		return auth.RequireCapability(actor, auth.CapabilityNodeRead) == nil
	case "environments":
		return auth.RequireCapability(actor, auth.CapabilityEnvironmentRead) == nil
	case "events":
		return auth.RequireCapability(actor, auth.CapabilityNodeRead) == nil && auth.RequireCapability(actor, auth.CapabilityEnvironmentRead) == nil
	}
	return false
}

func (h *opsHub) add(id, token string, actor auth.User, expires time.Time, send func(string, any) error, writable func() bool, disconnect func()) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.clients) >= maxOpsClients || !expires.After(time.Now()) || actor.TenantID == "" {
		return false
	}
	h.clients[id] = &opsClient{token: token, actor: actor, expires: expires, watches: map[string]opsWatch{}, send: send, writable: writable, disconnect: disconnect}
	if _, exists := h.checks[token]; !exists {
		h.checks[token] = time.Now()
	}
	return true
}

func (h *opsHub) remove(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	client := h.clients[id]
	delete(h.clients, id)
	if client != nil {
		for _, other := range h.clients {
			if other.token == client.token {
				return
			}
		}
		delete(h.checks, client.token)
		delete(h.checking, client.token)
	}
}

func decodeOpsWatch(value any) (opsWatch, bool) {
	raw, err := json.Marshal(value)
	if err != nil || len(raw) > 32768 {
		return opsWatch{}, false
	}
	var watch opsWatch
	if json.Unmarshal(raw, &watch) != nil || len(watch.SubscriptionID) == 0 || len(watch.SubscriptionID) > 80 || len(watch.Topics) > 4 || len(watch.EntityIDs) > maxOpsEntityIDs {
		return opsWatch{}, false
	}
	for _, id := range watch.EntityIDs {
		if len(id) > 80 || id == "" {
			return opsWatch{}, false
		}
	}
	return watch, true
}

func (h *opsHub) watch(id string, value any, remove bool) {
	h.mu.Lock()
	client := h.clients[id]
	if client == nil || client.closing {
		h.mu.Unlock()
		return
	}
	now := time.Now()
	if now.Sub(client.window) >= time.Second {
		client.window = now
		client.requests = 0
	}
	client.requests++
	if code := h.invalidSessionCode(client, now); code != "" {
		beginOpsAuthClose(client, code)
		h.mu.Unlock()
		h.signal()
		return
	}
	if client.requests > maxOpsWatchRequests {
		h.mu.Unlock()
		client.disconnect()
		h.remove(id)
		return
	}
	// 先限频再解析，畸形/超大订阅请求同样消耗配额，不能绕过分配成本保护。
	h.mu.Unlock()
	watch, valid := decodeOpsWatch(value)
	if !valid {
		return
	}
	h.mu.Lock()
	if h.clients[id] != client || client.closing {
		h.mu.Unlock()
		return
	}
	if remove {
		delete(client.watches, watch.SubscriptionID)
		h.mu.Unlock()
		return
	}
	if _, exists := client.watches[watch.SubscriptionID]; !exists && len(client.watches) >= maxOpsSubscriptions {
		h.mu.Unlock()
		return
	}
	topics := map[string]bool{}
	for _, topic := range watch.Topics {
		if allowedOpsTopic(client.actor, topic) {
			topics[topic] = true
		}
	}
	watch.Topics = sortedKeys(topics)
	client.watches[watch.SubscriptionID] = watch
	ready := map[string]any{"subscriptionId": watch.SubscriptionID, "epoch": h.epoch, "sequence": h.sequence, "topics": watch.Topics}
	overflow := !queueOpsPacket(client, opsPacket{"ops:ready", ready})
	h.mu.Unlock()
	// ready 与change共用同一有界出站队列，杜绝解锁后change先于ready抵达。
	if overflow {
		client.disconnect()
		h.remove(id)
	}
	h.signal()
}

// 相邻状态失效通知可合并；结构/溢出通知采用broad，终态标记不得丢失。
func queueOpsPacket(client *opsClient, packet opsPacket) bool {
	if client.closing {
		return true
	}
	if packet.event == "ops:change" && len(client.outbox) > 0 {
		last := &client.outbox[len(client.outbox)-1]
		if last.event == "ops:change" {
			old, next := last.payload.(opsChange), packet.payload.(opsChange)
			topics := map[string]bool{}
			for _, v := range append(old.Topics, next.Topics...) {
				topics[v] = true
			}
			next.Topics = sortedKeys(topics)
			next.EntityIDs = nil
			next.Terminal = next.Terminal || old.Terminal
			next.Resync = next.Resync || old.Resync
			last.payload = next
			return true
		}
	}
	if len(client.outbox) >= 8 {
		return false
	}
	client.outbox = append(client.outbox, packet)
	return true
}

// 失效会话先清除待发业务通知，再优先发送一次机器可读原因；下一tick关闭连接。
// AUTH_EXPIRED允许前端有限续租，撤权或复核失败必须AUTH_FORBIDDEN，不能无限刷新重试。
func beginOpsAuthClose(client *opsClient, code string) {
	if client.closing {
		return
	}
	client.closing = true
	client.watches = nil
	client.outbox = []opsPacket{{"ops:auth", map[string]any{"code": code}}}
}

func (h *opsHub) invalidSessionCode(client *opsClient, now time.Time) string {
	if !client.expires.After(now) {
		return opsAuthExpired
	}
	if now.Sub(h.checks[client.token]) > 60*time.Second {
		return opsAuthForbidden
	}
	return ""
}

func sortedKeys(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func (h *opsHub) publish(tenant string, topics, ids []string, terminal bool) {
	if tenant == "" {
		return
	}
	h.mu.Lock()
	item := h.pending[tenant]
	if item == nil {
		if len(h.pending) >= maxOpsPendingTenants {
			h.resync = true
			h.mu.Unlock()
			h.signal()
			return
		}
		item = &opsPending{topics: map[string]bool{}, ids: map[string]bool{}, due: time.Now().Add(3 * time.Second)}
		h.pending[tenant] = item
	}
	for _, topic := range topics {
		switch topic {
		case "nodes", "environments", "deployments", "events":
			item.topics[topic] = true
		}
	}
	if len(ids) == 0 {
		item.broad = true
	}
	for _, id := range ids {
		if len(item.ids) >= maxOpsEntityIDs {
			item.broad = true
			break
		}
		if id != "" && len(id) <= 80 {
			item.ids[id] = true
		}
	}
	item.terminal = item.terminal || terminal
	// 错峰节点指标按租户3秒合并；状态/API变更和终态走快通道。
	if terminal || len(topics) != 1 || topics[0] != "nodes" {
		item.due = time.Now()
	}
	h.mu.Unlock()
	if terminal {
		h.signal()
	}
}

func (h *opsHub) signal() {
	select {
	case h.wake <- struct{}{}:
	default:
	}
}

func watchMatches(w opsWatch, p *opsPending) bool {
	match := false
	for _, topic := range w.Topics {
		match = match || p.topics[topic]
	}
	if !match {
		return false
	}
	if p.broad || len(w.EntityIDs) == 0 {
		return true
	}
	for _, id := range w.EntityIDs {
		if p.ids[id] {
			return true
		}
	}
	return false
}

func (h *opsHub) flush() {
	h.mu.Lock()
	items, resync := h.pending, h.resync
	h.pending = map[string]*opsPending{}
	h.resync = false
	overflow := map[string]*opsClient{}
	for tenant, item := range items {
		if item.due.After(time.Now()) {
			h.pending[tenant] = item
			continue
		}
		h.sequence++
		for id, client := range h.clients {
			if client.actor.TenantID != tenant {
				continue
			}
			allowed := map[string]bool{}
			matchedIDs := map[string]bool{}
			unrestricted := false
			for _, watch := range client.watches {
				if watchMatches(watch, item) {
					unrestricted = unrestricted || len(watch.EntityIDs) == 0
					for _, entityID := range watch.EntityIDs {
						if item.ids[entityID] {
							matchedIDs[entityID] = true
						}
					}
					for _, topic := range watch.Topics {
						if item.topics[topic] {
							allowed[topic] = true
						}
					}
				}
			}
			if len(allowed) == 0 {
				continue
			}
			change := opsChange{Epoch: h.epoch, Sequence: h.sequence, Topics: sortedKeys(allowed), Timestamp: time.Now().UTC().Format(time.RFC3339Nano), Terminal: item.terminal}
			if !item.broad {
				if unrestricted {
					change.EntityIDs = sortedKeys(item.ids)
				} else {
					change.EntityIDs = sortedKeys(matchedIDs)
				}
			}
			if !queueOpsPacket(client, opsPacket{"ops:change", change}) {
				overflow[id] = client
			}
		}
	}
	if resync {
		h.sequence++
		for id, client := range h.clients {
			topics := map[string]bool{}
			for _, w := range client.watches {
				for _, topic := range w.Topics {
					topics[topic] = true
				}
			}
			if len(topics) > 0 {
				if !queueOpsPacket(client, opsPacket{"ops:change", opsChange{Epoch: h.epoch, Sequence: h.sequence, Topics: sortedKeys(topics), Timestamp: time.Now().UTC().Format(time.RFC3339Nano), Resync: true}}) {
					overflow[id] = client
				}
			}
		}
	}
	h.mu.Unlock()
	for id, client := range overflow {
		client.disconnect()
		h.remove(id)
	}
}

// 每轮每连接至多发送一个包。实际transport不可写时不调用Emit，避免engine.io无界writeBuffer。
// 不可写超过5秒则断开，重连ready触发HTTP对账；不能静默丢失终态。
func (h *opsHub) drain() {
	h.mu.Lock()
	clients := map[string]*opsClient{}
	for id, c := range h.clients {
		if len(c.outbox) > 0 || c.closing {
			clients[id] = c
		}
	}
	h.mu.Unlock()
	for id, c := range clients {
		now := time.Now()
		h.mu.Lock()
		if h.clients[id] != c {
			h.mu.Unlock()
			continue
		}
		if code := h.invalidSessionCode(c, now); code != "" {
			beginOpsAuthClose(c, code)
		}
		stale := c.closing && ((!c.authSentAt.IsZero() && now.Sub(c.authSentAt) >= 250*time.Millisecond) || (c.authSentAt.IsZero() && !c.writable()))
		if stale {
			h.mu.Unlock()
			c.disconnect()
			h.remove(id)
			continue
		}
		if len(c.outbox) == 0 {
			h.mu.Unlock()
			continue
		}
		if !c.writable() {
			if c.blockedAt.IsZero() {
				c.blockedAt = now
			}
			stale = now.Sub(c.blockedAt) > 5*time.Second
			if !stale {
				h.mu.Unlock()
				continue
			}
		}
		packet := c.outbox[0]
		c.outbox = c.outbox[1:]
		c.blockedAt = time.Time{}
		if packet.event == "ops:auth" {
			c.authSentAt = now
		}
		h.mu.Unlock()
		if stale || c.send(packet.event, packet.payload) != nil {
			c.disconnect()
			h.remove(id)
		}
	}
}

func (h *opsHub) maintain() {
	now := time.Now()
	h.mu.Lock()
	for _, client := range h.clients {
		if code := h.invalidSessionCode(client, now); code != "" {
			beginOpsAuthClose(client, code)
		}
	}
	for token, last := range h.checks {
		if now.Sub(last) < 30*time.Second || h.checking[token] {
			continue
		}
		select {
		case h.jobs <- authCheck{token: token}:
			h.checking[token] = true
		default:
		}
	}
	h.mu.Unlock()
	h.signal()
}

func (h *opsHub) authWorker() {
	defer h.wg.Done()
	for {
		select {
		case <-h.ctx.Done():
			return
		case job := <-h.jobs:
			ctx, cancel := context.WithTimeout(h.ctx, 3*time.Second)
			actor, _, err := h.validate(ctx, job.token)
			cancel()
			h.mu.Lock()
			delete(h.checking, job.token)
			if _, exists := h.checks[job.token]; exists && err == nil {
				h.checks[job.token] = time.Now()
			}
			for _, client := range h.clients {
				if client.token == job.token && (err != nil || actor.TenantID != client.actor.TenantID || actor.Role != client.actor.Role || actor.ID != client.actor.ID) {
					code := opsAuthForbidden
					if errors.Is(err, auth.ErrAccessTokenExpired) {
						code = opsAuthExpired
					}
					beginOpsAuthClose(client, code)
				}
			}
			h.mu.Unlock()
			h.signal()
		}
	}
}

func (h *opsHub) run() {
	defer h.wg.Done()
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	maintenance := time.NewTicker(time.Second)
	defer maintenance.Stop()
	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.flush()
			h.drain()
		case <-h.wake:
			h.flush()
			h.drain()
		case <-maintenance.C:
			h.maintain()
		}
	}
}

package realtime

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/indu-forge/dev_core/internal/auth"
)

func testOpsHub(t *testing.T) *opsHub {
	t.Helper()
	h := newOpsHub(func(context.Context, string) (auth.User, time.Time, error) {
		return auth.User{}, time.Time{}, errors.New("test validator unused")
	})
	// 停止后台时钟后手动推进flush/drain，使顺序、容量和超时边界可确定复现。
	h.close()
	return h
}

type hubProbe struct {
	packets  []opsPacket
	closed   bool
	writable bool
}

func addProbe(t *testing.T, h *opsHub, id, tenant, role string) *hubProbe {
	t.Helper()
	p := &hubProbe{writable: true}
	if !h.add(id, "token-"+id, auth.User{ID: id, TenantID: tenant, Role: role}, time.Now().Add(time.Hour), func(event string, value any) error {
		p.packets = append(p.packets, opsPacket{event, value})
		return nil
	}, func() bool { return p.writable }, func() { p.closed = true }) {
		t.Fatal("add failed")
	}
	return p
}

func subscribeProbe(h *opsHub, id string, topics ...string) {
	h.watch(id, opsWatch{SubscriptionID: "watch", Topics: topics}, false)
}

func TestOpsHubTenantTopicAndEntityIsolation(t *testing.T) {
	h := testOpsHub(t)
	a := addProbe(t, h, "a", "tenant-a", "OPS_ADMIN")
	b := addProbe(t, h, "b", "tenant-b", "OPS_ADMIN")
	viewer := addProbe(t, h, "viewer", "tenant-a", "VIEWER")
	node := addProbe(t, h, "node", "tenant-a", "OPS_ADMIN")
	subscribeProbe(h, "a", "deployments", "unknown")
	subscribeProbe(h, "b", "deployments")
	subscribeProbe(h, "viewer", "nodes", "deployments", "environments", "events")
	h.watch("node", opsWatch{SubscriptionID: "watch", Topics: []string{"nodes"}, EntityIDs: []string{"node-a"}}, false)
	h.drain()
	if topics := a.packets[0].payload.(map[string]any)["topics"]; !reflect.DeepEqual(topics, []string{"deployments"}) {
		t.Fatalf("allowed topics=%v", topics)
	}
	if topics := viewer.packets[0].payload.(map[string]any)["topics"].([]string); len(topics) != 0 {
		t.Fatalf("viewer got topics %v", topics)
	}
	h.publish("tenant-a", []string{"deployments"}, []string{"deployment-a"}, true)
	h.flush()
	h.drain()
	if len(a.packets) != 2 || len(b.packets) != 1 || len(viewer.packets) != 1 || len(node.packets) != 1 {
		t.Fatalf("unexpected delivery counts %d/%d/%d/%d", len(a.packets), len(b.packets), len(viewer.packets), len(node.packets))
	}
	h.publish("tenant-a", []string{"nodes"}, []string{"node-b"}, true)
	h.flush()
	h.drain()
	if len(node.packets) != 1 {
		t.Fatal("entity filter leaked unrelated change")
	}
	h.publish("tenant-a", []string{"nodes"}, []string{"node-a"}, true)
	h.flush()
	h.drain()
	if len(node.packets) != 2 {
		t.Fatal("matching node change missing")
	}
}

func TestOpsHubReadyPrecedesChangesAndSequenceIncreases(t *testing.T) {
	h := testOpsHub(t)
	p := addProbe(t, h, "a", "tenant", "OPS_ADMIN")
	subscribeProbe(h, "a", "deployments")
	h.publish("tenant", []string{"deployments"}, []string{"one"}, true)
	h.flush()
	h.drain()
	h.drain()
	if len(p.packets) != 2 || p.packets[0].event != "ops:ready" || p.packets[1].event != "ops:change" {
		t.Fatalf("packet order=%v", p.packets)
	}
	ready := p.packets[0].payload.(map[string]any)
	first := p.packets[1].payload.(opsChange)
	if first.Sequence <= ready["sequence"].(uint64) || first.Epoch != ready["epoch"] {
		t.Fatal("invalid ready/change cursor")
	}
	h.publish("tenant", []string{"deployments"}, nil, true)
	h.flush()
	h.drain()
	if p.packets[2].payload.(opsChange).Sequence <= first.Sequence {
		t.Fatal("sequence did not advance")
	}
}

func TestOpsHubMetricsAreMergedOnceForManyClients(t *testing.T) {
	h := testOpsHub(t)
	probes := make([]*hubProbe, 64)
	for i := range probes {
		id := fmt.Sprint(i)
		probes[i] = addProbe(t, h, id, "tenant", "OPS_ADMIN")
		subscribeProbe(h, id, "nodes")
	}
	h.drain()
	for i := 0; i < 1000; i++ {
		h.publish("tenant", []string{"nodes"}, []string{fmt.Sprint(i)}, false)
	}
	h.flush()
	h.drain()
	for _, p := range probes {
		if len(p.packets) != 1 {
			t.Fatal("metrics bypassed batch window")
		}
	}
	h.mu.Lock()
	h.pending["tenant"].due = time.Now().Add(-time.Second)
	h.mu.Unlock()
	h.flush()
	h.drain()
	for _, p := range probes {
		if len(p.packets) != 2 {
			t.Fatalf("expected one merged delivery, got %d", len(p.packets))
		}
		change := p.packets[1].payload.(opsChange)
		if change.Sequence != 1 || len(change.EntityIDs) != 0 {
			t.Fatalf("overflow must broaden IDs: %+v", change)
		}
	}
}

func TestOpsHubSlowClientBoundedAndDoesNotBlockOthers(t *testing.T) {
	h := testOpsHub(t)
	slow := addProbe(t, h, "slow", "tenant", "OPS_ADMIN")
	slow.writable = false
	fast := addProbe(t, h, "fast", "tenant", "OPS_ADMIN")
	subscribeProbe(h, "slow", "deployments")
	subscribeProbe(h, "fast", "deployments")
	h.drain()
	for i := 0; i < 1000; i++ {
		h.publish("tenant", []string{"deployments"}, nil, true)
		h.flush()
		h.drain()
	}
	if len(slow.packets) != 0 || len(h.clients["slow"].outbox) > 2 || len(fast.packets) != 1001 {
		t.Fatalf("backpressure failed slow=%d queue=%d fast=%d", len(slow.packets), len(h.clients["slow"].outbox), len(fast.packets))
	}
	if !h.clients["slow"].outbox[1].payload.(opsChange).Terminal {
		t.Fatal("merged terminal flag lost")
	}
	h.clients["slow"].blockedAt = time.Now().Add(-6 * time.Second)
	h.drain()
	if !slow.closed || h.clients["slow"] != nil || fast.closed {
		t.Fatal("slow client not isolated")
	}
}

func TestOpsHubQueueOverflowDisconnectsInsteadOfDroppingReady(t *testing.T) {
	h := testOpsHub(t)
	p := addProbe(t, h, "a", "tenant", "OPS_ADMIN")
	p.writable = false
	for i := 0; i < 9; i++ {
		subscribeProbe(h, "a", "nodes")
	}
	if !p.closed || h.clients["a"] != nil {
		t.Fatal("outbox overflow should disconnect")
	}
}

func TestOpsHubMalformedWatchIsRateLimited(t *testing.T) {
	h := testOpsHub(t)
	p := addProbe(t, h, "a", "tenant", "OPS_ADMIN")
	for i := 0; i <= maxOpsWatchRequests; i++ {
		h.watch("a", map[string]any{"topics": "invalid"}, false)
	}
	if !p.closed || h.clients["a"] != nil {
		t.Fatal("malformed watches bypassed request limiter")
	}
}

func TestOpsHubExpiredAndUnverifiedSessionsFailClosed(t *testing.T) {
	for _, kind := range []string{"expired", "unverified"} {
		t.Run(kind, func(t *testing.T) {
			h := testOpsHub(t)
			p := addProbe(t, h, "a", "tenant", "OPS_ADMIN")
			if kind == "expired" {
				h.clients["a"].expires = time.Now().Add(-time.Second)
			} else {
				h.checks["token-a"] = time.Now().Add(-61 * time.Second)
			}
			h.maintain()
			h.drain()
			code := opsAuthExpired
			if kind == "unverified" {
				code = opsAuthForbidden
			}
			if len(p.packets) != 1 || p.packets[0].event != "ops:auth" || p.packets[0].payload.(map[string]any)["code"] != code {
				t.Fatalf("missing auth close reason: %v", p.packets)
			}
			h.publish("tenant", []string{"nodes"}, nil, true)
			h.flush()
			if len(h.clients["a"].outbox) != 0 {
				t.Fatal("closing session accepted business delivery")
			}
			h.clients["a"].authSentAt = time.Now().Add(-time.Second)
			h.drain()
			if !p.closed || h.clients["a"] != nil {
				t.Fatal("invalid session remained live")
			}
		})
	}
}

func TestOpsHubRevokedSessionDisconnectsAllSharedTokenClients(t *testing.T) {
	var calls atomic.Int32
	h := newOpsHub(func(context.Context, string) (auth.User, time.Time, error) {
		calls.Add(1)
		return auth.User{}, time.Time{}, errors.New("revoked")
	})
	defer h.close()
	var disconnected atomic.Int32
	for i := 0; i < 20; i++ {
		if !h.add(fmt.Sprint(i), "shared-token", auth.User{ID: "user", TenantID: "tenant", Role: "OPS_ADMIN"}, time.Now().Add(time.Hour), func(string, any) error { return nil }, func() bool { return true }, func() { disconnected.Add(1) }) {
			t.Fatal("add failed")
		}
	}
	h.mu.Lock()
	h.checks["shared-token"] = time.Now().Add(-31 * time.Second)
	h.mu.Unlock()
	h.maintain()
	waitUntil(t, func() bool { return disconnected.Load() == 20 })
	if calls.Load() != 1 {
		t.Fatalf("same token caused %d auth calls", calls.Load())
	}
}

func TestOpsHubAuthNoticeGetsFlushGraceAndSuppressesBusinessPackets(t *testing.T) {
	h := testOpsHub(t)
	p := addProbe(t, h, "a", "tenant", "OPS_ADMIN")
	subscribeProbe(h, "a", "deployments")
	beginOpsAuthClose(h.clients["a"], opsAuthForbidden)
	h.drain()
	// transport发送auth期间短暂不可写不应立即强制close，否则提示帧可能尚未发出。
	p.writable = false
	h.drain()
	if p.closed || len(p.packets) != 1 || p.packets[0].event != "ops:auth" {
		t.Fatal("auth notice was closed before grace period")
	}
	h.watch("a", opsWatch{SubscriptionID: "late", Topics: []string{"deployments"}}, false)
	h.publish("tenant", []string{"deployments"}, nil, true)
	h.flush()
	if len(h.clients["a"].outbox) != 0 {
		t.Fatal("business notification survived auth close")
	}
	h.clients["a"].authSentAt = time.Now().Add(-time.Second)
	h.drain()
	if !p.closed {
		t.Fatal("auth closing exceeded grace period")
	}
}

func waitUntil(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(time.Millisecond * 5)
	}
	t.Fatal("condition did not become true")
}

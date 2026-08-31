package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/indu-forge/runtime-api/internal/artifact"
	"github.com/nats-io/nats.go"
)

type PointEvent struct {
	SchemaVersion   string          `json:"schemaVersion"`
	DeploymentID    string          `json:"deploymentId"`
	AccountID       string          `json:"accountId"`
	PointID         string          `json:"pointId"`
	EventID         string          `json:"eventId"`
	Value           json.RawMessage `json:"value"`
	Quality         string          `json:"quality"`
	SourceTimestamp time.Time       `json:"sourceTimestamp"`
	ServerTimestamp time.Time       `json:"serverTimestamp"`
	Sequence        int64           `json:"sequence"`
	Path            string          `json:"path"`
}

type subscription struct {
	paths  map[string]struct{}
	events chan PointEvent
}

type Hub struct {
	mu          sync.RWMutex
	deployment  string
	account     string
	catalog     *artifact.Catalog
	subscribers map[uint64]*subscription
	sequence    atomic.Uint64
	lastEventNS atomic.Int64
	connected   atomic.Bool
	dropped     atomic.Uint64
}

func NewHub(deploymentID, accountID string, catalog *artifact.Catalog) *Hub {
	return &Hub{deployment: deploymentID, account: accountID, catalog: catalog, subscribers: map[uint64]*subscription{}}
}

func (h *Hub) Subscribe(paths []string) (<-chan PointEvent, func(), error) {
	if len(paths) == 0 || len(paths) > 100 {
		return nil, nil, errors.New("订阅路径数量必须在 1 到 100 之间")
	}
	wanted := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if _, ok := h.catalog.PointByPath(path); !ok {
			return nil, nil, fmt.Errorf("数据点不存在: %s", path)
		}
		wanted[path] = struct{}{}
	}
	id := h.sequence.Add(1)
	sub := &subscription{paths: wanted, events: make(chan PointEvent, 64)}
	h.mu.Lock()
	h.subscribers[id] = sub
	h.mu.Unlock()
	var once sync.Once
	cancel := func() {
		once.Do(func() {
			h.mu.Lock()
			delete(h.subscribers, id)
			close(sub.events)
			h.mu.Unlock()
		})
	}
	return sub.events, cancel, nil
}

func (h *Hub) Accept(subject string, body []byte) error {
	if len(body) == 0 || len(body) > 1<<20 {
		return errors.New("实时事件为空或超过 1MiB")
	}
	var event PointEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return fmt.Errorf("解析实时事件: %w", err)
	}
	if event.SchemaVersion != "data.raw.v1" && event.SchemaVersion != "data.computed.v1" {
		return errors.New("实时事件 schemaVersion 不受支持")
	}
	if event.DeploymentID != h.deployment || event.AccountID != h.account || event.EventID == "" {
		return errors.New("实时事件 deployment/account/event 身份不匹配")
	}
	prefix := "data.raw."
	if event.SchemaVersion == "data.computed.v1" {
		prefix = "data.computed."
	}
	if subject != prefix+event.PointID {
		return errors.New("实时事件 subject 与 pointId 不匹配")
	}
	point, ok := h.catalog.PointByID(event.PointID)
	if !ok {
		return errors.New("实时事件 pointId 不在 Release Artifact")
	}
	event.Path = point.Path
	h.lastEventNS.Store(time.Now().UTC().UnixNano())
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, subscriber := range h.subscribers {
		if _, interested := subscriber.paths[event.Path]; !interested {
			continue
		}
		select {
		case subscriber.events <- event:
		default:
			h.dropped.Add(1)
		}
	}
	return nil
}

func (h *Hub) Connected() bool { return h.connected.Load() }

func (h *Hub) SetConnected(value bool) { h.connected.Store(value) }

func (h *Hub) LastEventAt() *time.Time {
	value := h.lastEventNS.Load()
	if value == 0 {
		return nil
	}
	timestamp := time.Unix(0, value).UTC()
	return &timestamp
}

func (h *Hub) DroppedEvents() uint64 { return h.dropped.Load() }

type NATSSubscriber struct {
	connection *nats.Conn
	subs       []*nats.Subscription
}

type NATSOptions struct {
	URL             string
	CredentialsFile string
	Name            string
}

func ConnectNATS(ctx context.Context, options NATSOptions, hub *Hub) (*NATSSubscriber, error) {
	if strings.TrimSpace(options.URL) == "" || hub == nil {
		return nil, errors.New("NATS URL 和 realtime hub 不能为空")
	}
	natsOptions := []nats.Option{
		nats.Name(options.Name),
		nats.Timeout(5 * time.Second),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(time.Second),
		nats.DisconnectErrHandler(func(_ *nats.Conn, _ error) { hub.SetConnected(false) }),
		nats.ReconnectHandler(func(_ *nats.Conn) { hub.SetConnected(true) }),
		nats.ClosedHandler(func(_ *nats.Conn) { hub.SetConnected(false) }),
	}
	if strings.TrimSpace(options.CredentialsFile) != "" {
		natsOptions = append(natsOptions, nats.UserCredentials(options.CredentialsFile))
	}
	connection, err := nats.Connect(options.URL, natsOptions...)
	if err != nil {
		return nil, fmt.Errorf("连接 NATS: %w", err)
	}
	subscriber := &NATSSubscriber{connection: connection}
	for _, subject := range []string{"data.raw.>", "data.computed.>"} {
		sub, subscribeErr := connection.Subscribe(subject, func(message *nats.Msg) {
			_ = hub.Accept(message.Subject, message.Data)
		})
		if subscribeErr != nil {
			subscriber.Close()
			return nil, fmt.Errorf("订阅 NATS %s: %w", subject, subscribeErr)
		}
		subscriber.subs = append(subscriber.subs, sub)
	}
	if err := connection.FlushWithContext(ctx); err != nil {
		subscriber.Close()
		return nil, fmt.Errorf("激活 NATS 订阅: %w", err)
	}
	hub.SetConnected(connection.IsConnected())
	return subscriber, nil
}

func (s *NATSSubscriber) Close() {
	if s == nil {
		return
	}
	for _, sub := range s.subs {
		_ = sub.Unsubscribe()
	}
	if s.connection != nil {
		s.connection.Drain()
		s.connection.Close()
	}
}

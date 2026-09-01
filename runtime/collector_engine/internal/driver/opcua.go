package driver

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"github.com/indu-forge/collector-engine/internal/loader"
	"github.com/indu-forge/collector-engine/internal/resolver"
	"sync"
	"time"
)

type uaClient interface {
	Connect(context.Context) error
	Close(context.Context) error
	Read(context.Context, *ua.ReadRequest) (*ua.ReadResponse, error)
}
type OPCUA struct {
	resolver  *resolver.Resolver
	mu        sync.Mutex
	clients   map[string]uaClient
	newClient func(string) (uaClient, error)
}

func NewOPCUA(r *resolver.Resolver) *OPCUA {
	return &OPCUA{resolver: r, clients: map[string]uaClient{}, newClient: func(endpoint string) (uaClient, error) {
		return opcua.NewClient(endpoint, opcua.SecurityMode(ua.MessageSecurityModeNone), opcua.SecurityPolicy(ua.SecurityPolicyURINone), opcua.AuthAnonymous())
	}}
}
func (*OPCUA) ID() string { return "opcua.standard" }
func (d *OPCUA) Ready(ctx context.Context, c loader.Connection, b loader.BindingConnection) error {
	_, e := d.client(ctx, c, b)
	return e
}
func (*OPCUA) Read(context.Context, loader.Connection, loader.Mapping) (Result, error) {
	return Result{}, errors.New("OPC UA binding 未提供")
}
func (d *OPCUA) ReadBatch(ctx context.Context, c loader.Connection, b loader.BindingConnection, mappings []loader.Mapping) (map[string]Result, error) {
	cl, e := d.client(ctx, c, b)
	if e != nil {
		return nil, e
	}
	nodes := make([]*ua.ReadValueID, 0, len(mappings))
	for _, m := range mappings {
		var a struct {
			NodeID string `json:"nodeId"`
		}
		if !strictJSON(m.Address, &a) {
			return nil, errors.New("OPC UA nodeId 非法")
		}
		id, e := ua.ParseNodeID(a.NodeID)
		if e != nil {
			return nil, errors.New("OPC UA nodeId 非法")
		}
		nodes = append(nodes, &ua.ReadValueID{NodeID: id, AttributeID: ua.AttributeIDValue})
	}
	res, e := cl.Read(ctx, &ua.ReadRequest{NodesToRead: nodes, TimestampsToReturn: ua.TimestampsToReturnSource})
	if e != nil || res == nil || len(res.Results) != len(mappings) {
		d.invalidate(c.ConnectionID)
		return nil, errors.New("OPC UA 读取失败")
	}
	out := map[string]Result{}
	for i, v := range res.Results {
		if v == nil || v.Status != ua.StatusOK {
			out[mappings[i].DatapointID] = Result{Quality: "bad", SourceTimestamp: time.Now().UTC()}
			continue
		}
		value, e := json.Marshal(v.Value.Value())
		if e != nil {
			return nil, errors.New("OPC UA 值无法编码")
		}
		at := v.SourceTimestamp
		if at.IsZero() {
			at = time.Now().UTC()
		}
		out[mappings[i].DatapointID] = Result{Value: value, Quality: "good", SourceTimestamp: at.UTC()}
	}
	return out, nil
}
func (d *OPCUA) client(ctx context.Context, c loader.Connection, b loader.BindingConnection) (uaClient, error) {
	if d == nil || d.resolver == nil {
		return nil, errors.New("OPC UA resolver 不可用")
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if cl := d.clients[c.ConnectionID]; cl != nil {
		return cl, nil
	}
	endpoint, e := d.resolver.ResolveOPCUA(ctx, b.ResourceRef)
	if e != nil {
		return nil, e
	}
	cl, e := d.newClient(endpoint.URL)
	if e != nil {
		return nil, errors.New("OPC UA 配置非法")
	}
	if e = cl.Connect(ctx); e != nil {
		return nil, errors.New("OPC UA 连接失败")
	}
	d.clients[c.ConnectionID] = cl
	return cl, nil
}
func (d *OPCUA) invalidate(id string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if c := d.clients[id]; c != nil {
		_ = c.Close(context.Background())
		delete(d.clients, id)
	}
}

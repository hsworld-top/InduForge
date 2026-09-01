package driver

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/goburrow/modbus"
	"github.com/indu-forge/collector-engine/internal/loader"
	"github.com/indu-forge/collector-engine/internal/resolver"
	"math"
	"net"
	"sort"
	"sync"
	"time"
)

const maxRegisters uint16 = 125

type ModbusTCP struct {
	resolver *resolver.Resolver
	mu       sync.Mutex
	clients  map[string]*modbus.TCPClientHandler
}

func NewModbusTCP(r *resolver.Resolver) *ModbusTCP {
	return &ModbusTCP{resolver: r, clients: map[string]*modbus.TCPClientHandler{}}
}
func (*ModbusTCP) ID() string { return "modbus.tcp" }
func (d *ModbusTCP) Ready(ctx context.Context, c loader.Connection, b loader.BindingConnection) error {
	_, err := d.handler(ctx, c, b)
	return err
}
func (d *ModbusTCP) Read(context.Context, loader.Connection, loader.Mapping) (Result, error) {
	return Result{}, errors.New("Modbus binding 未提供")
}

// ReadWithBinding is used by the scheduler because Driver's legacy Read method has no deployment binding argument.
func (d *ModbusTCP) ReadWithBinding(ctx context.Context, c loader.Connection, b loader.BindingConnection, m loader.Mapping) (Result, error) {
	h, err := d.handler(ctx, c, b)
	if err != nil {
		return Result{}, err
	}
	var a struct {
		Station  uint8  `json:"station"`
		Area     string `json:"area"`
		Address  uint16 `json:"address"`
		BitIndex *uint8 `json:"bitIndex"`
	}
	if !strictJSON(m.Address, &a) {
		return Result{}, errors.New("Modbus 地址非法")
	}
	h.SlaveId = a.Station
	client := modbus.NewClient(h)
	qty := uint16(m.ElementCount)
	if qty == 0 {
		qty = 1
	}
	if qty > maxRegisters {
		return Result{}, errors.New("Modbus 读取长度超限")
	}
	var raw []byte
	switch a.Area {
	case "coil":
		raw, err = client.ReadCoils(a.Address, qty)
	case "discreteInput":
		raw, err = client.ReadDiscreteInputs(a.Address, qty)
	case "inputRegister":
		raw, err = client.ReadInputRegisters(a.Address, qty)
	case "holdingRegister":
		raw, err = client.ReadHoldingRegisters(a.Address, qty)
	default:
		return Result{}, errors.New("Modbus 区域非法")
	}
	if err != nil {
		return Result{}, errors.New("Modbus 读取失败")
	}
	value, err := decodeModbus(raw, m.DataType, a.BitIndex)
	if err != nil {
		return Result{}, err
	}
	return Result{Value: value, Quality: "good", SourceTimestamp: time.Now().UTC()}, nil
}

// ReadBatch keeps all reads on one verified TCP connection. V1 mappings do not
// carry a batch policy; each area is bounded by the Modbus protocol limit and
// future schema versions may widen this to contiguous range coalescing.
func (d *ModbusTCP) ReadBatch(ctx context.Context, c loader.Connection, b loader.BindingConnection, mappings []loader.Mapping) (map[string]Result, error) {
	h, err := d.handler(ctx, c, b)
	if err != nil {
		return nil, err
	}
	type item struct {
		mapping loader.Mapping
		station byte
		area    string
		address uint16
		bit     *uint8
		width   uint16
	}
	groups := map[string][]item{}
	for _, mapping := range mappings {
		var a struct {
			Station  byte   `json:"station"`
			Area     string `json:"area"`
			Address  uint16 `json:"address"`
			BitIndex *uint8 `json:"bitIndex"`
		}
		if !strictJSON(mapping.Address, &a) {
			return nil, errors.New("Modbus 地址非法")
		}
		width := registerWidth(mapping)
		if a.Area == "coil" || a.Area == "discreteInput" {
			width = 1
		}
		if width == 0 {
			return nil, errors.New("Modbus 数据类型非法")
		}
		key := fmt.Sprintf("%d/%s", a.Station, a.Area)
		groups[key] = append(groups[key], item{mapping, a.Station, a.Area, a.Address, a.BitIndex, width})
	}
	results := map[string]Result{}
	for _, items := range groups {
		sort.Slice(items, func(i, j int) bool { return items[i].address < items[j].address })
		for start := 0; start < len(items); {
			end := start + 1
			windowEnd := uint32(items[start].address) + uint32(items[start].width)
			limit := uint32(125)
			if items[start].area == "coil" || items[start].area == "discreteInput" {
				limit = 2000
			}
			for end < len(items) {
				nextEnd := uint32(items[end].address) + uint32(items[end].width)
				if items[end].address > uint16(windowEnd+2) || nextEnd-uint32(items[start].address) > limit {
					break
				}
				if nextEnd > windowEnd {
					windowEnd = nextEnd
				}
				end++
			}
			raw, readErr := readWindow(h, items[start].station, items[start].area, items[start].address, uint16(windowEnd-uint32(items[start].address)))
			if readErr != nil {
				d.invalidate(c.ConnectionID)
				return nil, errors.New("Modbus 批量读取失败")
			}
			for _, it := range items[start:end] {
				offset := int(it.address - items[start].address)
				var chunk []byte
				if it.area == "coil" || it.area == "discreteInput" {
					chunk = []byte{raw[offset/8]}
					if offset%8 > 0 {
						chunk[0] >>= uint(offset % 8)
					}
				} else {
					from := offset * 2
					chunk = raw[from : from+int(it.width)*2]
				}
				v, e := decodeModbus(chunk, it.mapping.DataType, it.bit)
				if e != nil {
					return nil, e
				}
				results[it.mapping.DatapointID] = Result{Value: v, Quality: "good", SourceTimestamp: time.Now().UTC()}
			}
			start = end
		}
	}
	return results, nil
}
func registerWidth(m loader.Mapping) uint16 {
	bytes := map[string]uint16{"int8": 2, "uint8": 2, "int16": 2, "uint16": 2, "int32": 4, "uint32": 4, "float32": 4, "int64": 8, "uint64": 8, "float64": 8}[m.DataType]
	if bytes == 0 {
		return 0
	}
	return (bytes / 2) * uint16(m.ElementCount)
}
func readWindow(h *modbus.TCPClientHandler, station byte, area string, address, quantity uint16) ([]byte, error) {
	h.SlaveId = station
	c := modbus.NewClient(h)
	switch area {
	case "coil":
		return c.ReadCoils(address, quantity)
	case "discreteInput":
		return c.ReadDiscreteInputs(address, quantity)
	case "inputRegister":
		return c.ReadInputRegisters(address, quantity)
	case "holdingRegister":
		return c.ReadHoldingRegisters(address, quantity)
	}
	return nil, errors.New("Modbus 区域非法")
}
func (d *ModbusTCP) invalidate(id string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if h := d.clients[id]; h != nil {
		_ = h.Close()
		delete(d.clients, id)
	}
}
func (d *ModbusTCP) handler(ctx context.Context, c loader.Connection, b loader.BindingConnection) (*modbus.TCPClientHandler, error) {
	if d == nil || d.resolver == nil {
		return nil, errors.New("Modbus resolver 不可用")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if h := d.clients[c.ConnectionID]; h != nil {
		return h, nil
	}
	endpoint, err := d.resolver.ResolveModbus(ctx, b.ResourceRef)
	if err != nil {
		return nil, err
	}
	address := net.JoinHostPort(endpoint.Host, fmt.Sprint(endpoint.Port))
	h := modbus.NewTCPClientHandler(address)
	h.Timeout = 10 * time.Second
	if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) < h.Timeout {
		h.Timeout = time.Until(deadline)
	}
	if h.Timeout <= 0 {
		return nil, context.DeadlineExceeded
	}
	if err := h.Connect(); err != nil {
		return nil, errors.New("Modbus 连接失败")
	}
	d.clients[c.ConnectionID] = h
	return h, nil
}
func strictJSON(raw []byte, target any) bool {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	return d.Decode(target) == nil
}
func decodeModbus(raw []byte, typ string, bit *uint8) (json.RawMessage, error) {
	if typ == "bool" {
		if len(raw) == 0 {
			return nil, errors.New("Modbus 响应过短")
		}
		v := raw[0]&1 != 0
		if bit != nil {
			v = raw[0]&(1<<*bit) != 0
		}
		return json.Marshal(v)
	}
	need := map[string]int{"int8": 1, "uint8": 1, "int16": 2, "uint16": 2, "int32": 4, "uint32": 4, "float32": 4, "int64": 8, "uint64": 8, "float64": 8}[typ]
	if need == 0 || len(raw) < need {
		return nil, errors.New("Modbus 数据类型或响应长度非法")
	}
	switch typ {
	case "int8":
		return json.Marshal(int8(raw[0]))
	case "uint8":
		return json.Marshal(raw[0])
	case "int16":
		return json.Marshal(int16(binary.BigEndian.Uint16(raw)))
	case "uint16":
		return json.Marshal(binary.BigEndian.Uint16(raw))
	case "int32":
		return json.Marshal(int32(binary.BigEndian.Uint32(raw)))
	case "uint32":
		return json.Marshal(binary.BigEndian.Uint32(raw))
	case "int64":
		return json.Marshal(int64(binary.BigEndian.Uint64(raw)))
	case "uint64":
		return json.Marshal(binary.BigEndian.Uint64(raw))
	case "float32":
		return json.Marshal(math.Float32frombits(binary.BigEndian.Uint32(raw)))
	case "float64":
		return json.Marshal(math.Float64frombits(binary.BigEndian.Uint64(raw)))
	}
	return nil, errors.New("Modbus 数据类型非法")
}

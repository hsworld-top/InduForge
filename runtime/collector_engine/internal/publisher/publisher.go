package publisher

import (
	"context"
	"errors"
	"github.com/nats-io/nats.go"
	"time"
)

var ErrBackpressure = errors.New("发布背压")

type Publisher interface {
	Ready(context.Context) error
	Publish(context.Context, string, []byte) error
	Close()
}
type NATS struct {
	nc      *nats.Conn
	timeout time.Duration
}

func Connect(url string, opts []nats.Option) (*NATS, error) {
	nc, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, errors.New("NATS 连接失败")
	}
	return &NATS{nc: nc, timeout: 5 * time.Second}, nil
}
func (p *NATS) Ready(ctx context.Context) error {
	if p == nil || p.nc == nil || p.nc.Status() != nats.CONNECTED {
		return errors.New("NATS 未就绪")
	}
	return nil
}
func (p *NATS) Publish(ctx context.Context, subject string, body []byte) error {
	if err := p.Ready(ctx); err != nil {
		return err
	}
	js, err := p.nc.JetStream()
	if err != nil {
		return errors.New("NATS JetStream 不可用")
	}
	t := p.timeout
	if deadline, ok := ctx.Deadline(); ok {
		t = time.Until(deadline)
	}
	if t <= 0 {
		return context.DeadlineExceeded
	}
	pctx, cancel := context.WithTimeout(ctx, t)
	defer cancel()
	_, err = js.Publish(subject, body, nats.Context(pctx))
	if errors.Is(err, nats.ErrTimeout) {
		return ErrBackpressure
	}
	if err != nil {
		return errors.New("NATS 发布失败")
	}
	return nil
}
func (p *NATS) Close() {
	if p != nil && p.nc != nil {
		p.nc.Close()
	}
}

package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// PahoBuiltinMessagePublisher 使用开发态 message-hub 发布 IF消息库测试消息。
type PahoBuiltinMessagePublisher struct {
	client mqtt.Client
}

func NewPahoBuiltinMessagePublisher(addr, username, password string) (*PahoBuiltinMessagePublisher, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return nil, fmt.Errorf("消息中心地址为空")
	}
	if !strings.Contains(addr, "://") {
		addr = "tcp://" + addr
	}
	clientID := "data-service-builtin-" + randomHex(6)
	options := mqtt.NewClientOptions().
		AddBroker(addr).
		SetClientID(clientID).
		SetCleanSession(true).
		SetConnectTimeout(5 * time.Second)
	if strings.TrimSpace(username) != "" {
		options.SetUsername(strings.TrimSpace(username))
		options.SetPassword(password)
	}
	client := mqtt.NewClient(options)
	token := client.Connect()
	if !token.WaitTimeout(5 * time.Second) {
		return nil, fmt.Errorf("连接消息中心超时")
	}
	if err := token.Error(); err != nil {
		return nil, err
	}
	return &PahoBuiltinMessagePublisher{client: client}, nil
}

func (p *PahoBuiltinMessagePublisher) Publish(ctx context.Context, topic string, payload []byte, qos byte) error {
	if p == nil || p.client == nil {
		return fmt.Errorf("消息发布器未初始化")
	}
	token := p.client.Publish(topic, qos, false, payload)
	done := make(chan struct{})
	go func() {
		token.Wait()
		close(done)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return token.Error()
	}
}

func (p *PahoBuiltinMessagePublisher) Close() {
	if p != nil && p.client != nil && p.client.IsConnected() {
		p.client.Disconnect(250)
	}
}

func randomHex(size int) string {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

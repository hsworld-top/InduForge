package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/indu-forge/data_service/internal/repository"
)

// MqttConnectionRuntimeManager 维护工作台层面的真实 MQTT 连接。
// 连接按钮只负责 broker client 生命周期，具体 topic 订阅仍由 preview socket 按页面引用计数管理。
type MqttConnectionRuntimeManager struct {
	mu      sync.Mutex
	clients map[string]mqtt.Client

	builtinAddr     string
	builtinUsername string
	builtinPassword string
}

func NewMqttConnectionRuntimeManager() *MqttConnectionRuntimeManager {
	return &MqttConnectionRuntimeManager{clients: make(map[string]mqtt.Client)}
}

func (m *MqttConnectionRuntimeManager) ConfigureBuiltinMessageHub(addr, username, password string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.builtinAddr = strings.TrimSpace(addr)
	m.builtinUsername = strings.TrimSpace(username)
	m.builtinPassword = password
}

func (m *MqttConnectionRuntimeManager) ConnectExternal(ctx context.Context, connection repository.MqttConnectionDetailRecord) error {
	brokerURL, err := mqttBrokerURLFromConnection(connection)
	if err != nil {
		return err
	}
	options := mqtt.NewClientOptions().
		AddBroker(brokerURL).
		SetClientID(buildWorkbenchMqttClientID("external", connection.ID, connection.ClientID)).
		SetCleanSession(connection.CleanSession).
		SetKeepAlive(time.Duration(connection.Keepalive) * time.Second).
		SetConnectRetry(false).
		SetAutoReconnect(true).
		SetConnectTimeout(time.Duration(connection.ConnectTimeoutMS) * time.Millisecond)
	if connection.Username != nil {
		options.SetUsername(strings.TrimSpace(*connection.Username))
	}
	if connection.Password != nil {
		options.SetPassword(strings.TrimSpace(*connection.Password))
	}
	return m.connect(ctx, runtimeConnectionKey(connection.ProjectID, connection.ID), options, time.Duration(connection.ConnectTimeoutMS)*time.Millisecond)
}

func (m *MqttConnectionRuntimeManager) ConnectBuiltin(ctx context.Context, summary repository.MqttConnectionSummaryRecord) error {
	m.mu.Lock()
	addr := m.builtinAddr
	username := m.builtinUsername
	password := m.builtinPassword
	m.mu.Unlock()

	addr = strings.TrimSpace(addr)
	if addr == "" {
		return fmt.Errorf("IF消息库 message-hub 地址为空")
	}
	if !strings.Contains(addr, "://") {
		addr = "tcp://" + addr
	}

	options := mqtt.NewClientOptions().
		AddBroker(addr).
		SetClientID(buildWorkbenchMqttClientID("builtin", summary.ID, nil)).
		SetCleanSession(true).
		SetConnectRetry(false).
		SetAutoReconnect(true).
		SetConnectTimeout(5 * time.Second)
	if username != "" {
		options.SetUsername(username)
		options.SetPassword(password)
	}
	return m.connect(ctx, runtimeConnectionKey(summary.ProjectID, summary.ID), options, 5*time.Second)
}

func (m *MqttConnectionRuntimeManager) Disconnect(projectID, connectionID string) {
	key := runtimeConnectionKey(projectID, connectionID)
	m.mu.Lock()
	client := m.clients[key]
	delete(m.clients, key)
	m.mu.Unlock()
	if client != nil && client.IsConnected() {
		client.Disconnect(250)
	}
}

// IsConnected 只读取当前 data_service 进程内的开发态临时会话，不映射为持久化配置状态。
func (m *MqttConnectionRuntimeManager) IsConnected(projectID, connectionID string) bool {
	key := runtimeConnectionKey(projectID, connectionID)
	m.mu.Lock()
	defer m.mu.Unlock()
	client := m.clients[key]
	return client != nil && client.IsConnected()
}

func (m *MqttConnectionRuntimeManager) connect(ctx context.Context, key string, options *mqtt.ClientOptions, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	m.mu.Lock()
	if existing := m.clients[key]; existing != nil {
		if existing.IsConnected() {
			m.mu.Unlock()
			return nil
		}
		delete(m.clients, key)
		m.mu.Unlock()
		existing.Disconnect(250)
	} else {
		m.mu.Unlock()
	}

	client := mqtt.NewClient(options)
	token := client.Connect()
	done := make(chan struct{})
	go func() {
		token.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		client.Disconnect(250)
		return ctx.Err()
	case <-time.After(timeout):
		client.Disconnect(250)
		return fmt.Errorf("MQTT connect timeout")
	case <-done:
		if err := token.Error(); err != nil {
			client.Disconnect(250)
			return err
		}
	}

	m.mu.Lock()
	if previous := m.clients[key]; previous != nil {
		previous.Disconnect(250)
	}
	m.clients[key] = client
	m.mu.Unlock()
	return nil
}

func runtimeConnectionKey(projectID, connectionID string) string {
	return strings.TrimSpace(projectID) + ":" + strings.TrimSpace(connectionID)
}

func buildWorkbenchMqttClientID(kind, connectionID string, configured *string) string {
	prefix := "data-service-workbench"
	if configured != nil && strings.TrimSpace(*configured) != "" {
		prefix = strings.TrimSpace(*configured)
	}
	return normalizeMqttClientID(prefix + "-" + kind + "-" + connectionID)
}

func normalizeMqttClientID(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 64 {
		return value
	}
	return value[:64]
}

func mqttBrokerURLFromConnection(connection repository.MqttConnectionDetailRecord) (string, error) {
	raw := strings.TrimSpace(connection.BrokerURL)
	if raw == "" {
		return "", fmt.Errorf("MQTT broker 地址为空")
	}
	if strings.Contains(raw, "://") {
		return raw, nil
	}
	scheme := strings.TrimSpace(strings.ToLower(connection.Protocol))
	if scheme == "" || scheme == "mqtt" {
		scheme = "tcp"
	} else if scheme == "mqtts" {
		scheme = "ssl"
	}
	if connection.Port > 0 && !strings.Contains(raw, ":") {
		return fmt.Sprintf("%s://%s:%d", scheme, raw, connection.Port), nil
	}
	return fmt.Sprintf("%s://%s", scheme, raw), nil
}

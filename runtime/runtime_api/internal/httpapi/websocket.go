package httpapi

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type subscribeRequest struct {
	Action string   `json:"action"`
	Paths  []string `json:"paths"`
}

var pointUpgrader = websocket.Upgrader{
	HandshakeTimeout: 5 * time.Second,
	ReadBufferSize:   4096,
	WriteBufferSize:  4096,
	CheckOrigin: func(request *http.Request) bool {
		origin := strings.TrimSpace(request.Header.Get("Origin"))
		if origin == "" {
			return true
		}
		parsed, err := url.Parse(origin)
		if err != nil || parsed.Host == "" {
			return false
		}
		expected := request.Header.Get("X-Forwarded-Host")
		if expected == "" {
			expected = request.Host
		}
		return strings.EqualFold(parsed.Host, expected)
	},
}

func (s *Server) pointSocket(writer http.ResponseWriter, request *http.Request) {
	connection, err := pointUpgrader.Upgrade(writer, request, nil)
	if err != nil {
		return
	}
	defer connection.Close()
	connection.SetReadLimit(16 << 10)
	_ = connection.SetReadDeadline(time.Now().Add(10 * time.Second))
	var subscribe subscribeRequest
	if err := connection.ReadJSON(&subscribe); err != nil || subscribe.Action != "subscribe" {
		_ = connection.WriteJSON(map[string]any{"type": "error", "code": 40001, "msg": "首条消息必须是 subscribe"})
		return
	}
	events, cancel, err := s.config.Realtime.Subscribe(subscribe.Paths)
	if err != nil {
		_ = connection.WriteJSON(map[string]any{"type": "error", "code": 40001, "msg": err.Error()})
		return
	}
	defer cancel()
	_ = connection.SetReadDeadline(time.Time{})
	connection.SetPongHandler(func(string) error {
		return connection.SetReadDeadline(time.Now().Add(60 * time.Second))
	})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := connection.ReadMessage(); err != nil {
				return
			}
		}
	}()
	if err := connection.WriteJSON(map[string]any{"type": "subscribed", "paths": subscribe.Paths}); err != nil {
		return
	}
	ping := time.NewTicker(20 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-request.Context().Done():
			return
		case <-done:
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			if err := connection.WriteJSON(map[string]any{"type": "point", "data": event}); err != nil {
				return
			}
		case <-ping.C:
			if err := connection.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)); err != nil {
				return
			}
		}
	}
}

func (s *Server) alarmSocket(writer http.ResponseWriter, request *http.Request) {
	connection, err := pointUpgrader.Upgrade(writer, request, nil)
	if err != nil {
		return
	}
	defer connection.Close()
	connection.SetReadLimit(16 << 10)
	_ = connection.SetReadDeadline(time.Now().Add(10 * time.Second))
	var subscribe struct {
		Action string `json:"action"`
	}
	if err := connection.ReadJSON(&subscribe); err != nil || subscribe.Action != "subscribe" {
		_ = connection.WriteJSON(map[string]any{"type": "error", "code": 40001, "msg": "首条消息必须是 subscribe"})
		return
	}
	events, cancel := s.config.Realtime.SubscribeAlarms()
	defer cancel()
	_ = connection.SetReadDeadline(time.Time{})
	connection.SetPongHandler(func(string) error { return connection.SetReadDeadline(time.Now().Add(60 * time.Second)) })
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := connection.ReadMessage(); err != nil {
				return
			}
		}
	}()
	if err := connection.WriteJSON(map[string]any{"type": "subscribed"}); err != nil {
		return
	}
	ping := time.NewTicker(20 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-request.Context().Done():
			return
		case <-done:
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			if err := connection.WriteJSON(map[string]any{"type": "alarm", "data": event}); err != nil {
				return
			}
		case <-ping.C:
			if err := connection.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)); err != nil {
				return
			}
		}
	}
}

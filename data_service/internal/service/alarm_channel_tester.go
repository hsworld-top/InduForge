package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/indu-forge/data_service/internal/repository"
)

const alarmChannelTestBodyLimit = 64 * 1024

func sendAlarmChannelTest(ctx context.Context, channel repository.AlarmNotificationChannelRecord) error {
	rawURL := strings.TrimSpace(toString(channel.Config["webhookUrl"]))
	target, err := url.Parse(rawURL)
	if err != nil || target.Hostname() == "" {
		return errors.New("通知渠道地址无效")
	}
	if target.Scheme != "https" {
		return errors.New("通知渠道测试默认只允许 HTTPS")
	}
	validator := newAlarmTestTargetValidator(os.Getenv("DATA_SERVICE_ALARM_TEST_ALLOWLIST"))
	if err := validator.validateHost(ctx, target.Hostname()); err != nil {
		return err
	}

	payload := alarmChannelTestPayload(channel.ChannelType)
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化通知测试消息失败: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), bytes.NewReader(body))
	if err != nil {
		return errors.New("创建通知测试请求失败")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "InduForge-Alarm-Channel-Test/1.0")

	dialer := &net.Dialer{Timeout: 5 * time.Second}
	transport := &http.Transport{
		Proxy:           nil,
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		DialContext: func(dialCtx context.Context, network, address string) (net.Conn, error) {
			host, port, splitErr := net.SplitHostPort(address)
			if splitErr != nil {
				return nil, splitErr
			}
			ips, resolveErr := validator.allowedIPs(dialCtx, host)
			if resolveErr != nil {
				return nil, resolveErr
			}
			return dialer.DialContext(dialCtx, network, net.JoinHostPort(ips[0].String(), port))
		},
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
		CheckRedirect: func(next *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errors.New("通知渠道重定向次数过多")
			}
			if next.URL.Scheme != "https" {
				return errors.New("通知渠道重定向必须使用 HTTPS")
			}
			return validator.validateHost(next.Context(), next.URL.Hostname())
		},
	}
	defer transport.CloseIdleConnections()
	response, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("通知测试请求失败: %w", err)
	}
	defer response.Body.Close()
	if _, err := io.Copy(io.Discard, io.LimitReader(response.Body, alarmChannelTestBodyLimit)); err != nil {
		return fmt.Errorf("读取通知测试响应失败: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("目标服务器返回 HTTP %d", response.StatusCode)
	}
	return nil
}

func alarmChannelTestPayload(channelType string) map[string]any {
	content := "InduForge 报警通知渠道测试：这是一条固定脱敏测试消息，不属于运行态报警。"
	switch channelType {
	case "dingtalk":
		return map[string]any{"msgtype": "text", "text": map[string]string{"content": content}}
	case "wecom":
		return map[string]any{"msgtype": "text", "text": map[string]string{"content": content}}
	default:
		return map[string]any{"type": "alarm_channel_test", "message": content, "sentAt": time.Now().UTC().Format(time.RFC3339)}
	}
}

type alarmTestTargetValidator struct {
	domains  map[string]struct{}
	prefixes []netip.Prefix
}

func newAlarmTestTargetValidator(raw string) alarmTestTargetValidator {
	result := alarmTestTargetValidator{domains: map[string]struct{}{}, prefixes: []netip.Prefix{}}
	for _, item := range strings.Split(raw, ",") {
		item = strings.ToLower(strings.TrimSpace(item))
		if item == "" {
			continue
		}
		if prefix, err := netip.ParsePrefix(item); err == nil {
			result.prefixes = append(result.prefixes, prefix)
		} else {
			result.domains[strings.TrimSuffix(item, ".")] = struct{}{}
		}
	}
	return result
}

func (v alarmTestTargetValidator) validateHost(ctx context.Context, host string) error {
	_, err := v.allowedIPs(ctx, host)
	return err
}

func (v alarmTestTargetValidator) allowedIPs(ctx context.Context, host string) ([]net.IP, error) {
	normalizedHost := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
	if normalizedHost == "" {
		return nil, errors.New("通知渠道目标主机为空")
	}
	_, domainAllowed := v.domains[normalizedHost]
	addresses, err := net.DefaultResolver.LookupIP(ctx, "ip", normalizedHost)
	if err != nil || len(addresses) == 0 {
		return nil, errors.New("通知渠道目标 DNS 解析失败")
	}
	allowed := make([]net.IP, 0, len(addresses))
	for _, address := range addresses {
		parsed, ok := netip.AddrFromSlice(address)
		if !ok {
			continue
		}
		parsed = parsed.Unmap()
		if isAlarmMetadataAddress(parsed) {
			return nil, errors.New("通知渠道目标不能为云元数据地址")
		}
		if domainAllowed || v.prefixAllowed(parsed) || isPublicAlarmTestAddress(parsed) {
			allowed = append(allowed, address)
			continue
		}
		return nil, errors.New("通知渠道目标解析到回环、链路本地、云元数据或未授权内网地址")
	}
	if len(allowed) == 0 {
		return nil, errors.New("通知渠道目标没有可用地址")
	}
	return allowed, nil
}

func (v alarmTestTargetValidator) prefixAllowed(address netip.Addr) bool {
	for _, prefix := range v.prefixes {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}

func isPublicAlarmTestAddress(address netip.Addr) bool {
	if !address.IsValid() || address.IsLoopback() || address.IsPrivate() || address.IsLinkLocalUnicast() || address.IsLinkLocalMulticast() || address.IsUnspecified() || address.IsMulticast() {
		return false
	}
	// 云厂商元数据地址必须始终显式阻断，即使系统路由把它视为普通地址。
	if isAlarmMetadataAddress(address) {
		return false
	}
	return true
}

func isAlarmMetadataAddress(address netip.Addr) bool {
	return address == netip.MustParseAddr("169.254.169.254") || address == netip.MustParseAddr("100.100.100.200")
}

func sanitizeAlarmTestMessage(value string) string {
	value = strings.TrimSpace(value)
	if len([]rune(value)) > 480 {
		return string([]rune(value)[:480]) + "…"
	}
	return value
}

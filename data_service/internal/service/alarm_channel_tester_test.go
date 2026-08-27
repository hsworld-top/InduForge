package service

import (
	"net/netip"
	"testing"
)

func TestAlarmChannelTestAddressPolicy(t *testing.T) {
	for _, value := range []string{"127.0.0.1", "10.0.0.1", "169.254.169.254", "100.100.100.200", "::1"} {
		if isPublicAlarmTestAddress(netip.MustParseAddr(value)) {
			t.Fatalf("危险地址不应允许通知测试: %s", value)
		}
	}
	for _, value := range []string{"1.1.1.1", "8.8.8.8", "2606:4700:4700::1111"} {
		if !isPublicAlarmTestAddress(netip.MustParseAddr(value)) {
			t.Fatalf("公共地址应允许通知测试: %s", value)
		}
	}
}

func TestAlarmChannelTestAllowlistCIDR(t *testing.T) {
	validator := newAlarmTestTargetValidator("10.20.0.0/16,alarm.internal.example")
	if !validator.prefixAllowed(netip.MustParseAddr("10.20.3.4")) {
		t.Fatal("显式 CIDR 应允许内网地址")
	}
	if validator.prefixAllowed(netip.MustParseAddr("10.21.3.4")) {
		t.Fatal("CIDR 外地址不应被误允许")
	}
	if _, ok := validator.domains["alarm.internal.example"]; !ok {
		t.Fatal("显式域名应进入允许列表")
	}
}

func TestAlarmChannelMetadataAddressCannotBeAllowlisted(t *testing.T) {
	validator := newAlarmTestTargetValidator("169.254.0.0/16,100.64.0.0/10")
	for _, value := range []string{"169.254.169.254", "100.100.100.200"} {
		address := netip.MustParseAddr(value)
		if !validator.prefixAllowed(address) {
			t.Fatalf("测试前提失败：%s 应被 CIDR 覆盖", value)
		}
		if !isAlarmMetadataAddress(address) {
			t.Fatalf("云元数据地址必须始终识别: %s", value)
		}
	}
}

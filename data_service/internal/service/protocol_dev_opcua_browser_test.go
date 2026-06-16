package service

import (
	"context"
	"testing"
	"time"

	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
)

func TestParseOpcuaBrowseOptions(t *testing.T) {
	options, err := parseOpcuaBrowseOptions(map[string]any{
		"endpoint":       "opc.tcp://127.0.0.1:18540/induforge/sim",
		"securityPolicy": "Basic256Sha256",
		"securityMode":   "Sign",
		"authType":       "username_password",
		"username":       "tester",
		"password":       "secret",
		"options": map[string]any{
			"browseRootNodeId": "ns=2;s=Root",
			"browseMaxDepth":   float64(4),
			"browseMaxNodes":   "120",
			"connectTimeoutMs": 2500,
			"browseTimeoutMs":  float64(3500),
			"sslConfig": map[string]any{
				"cert": "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----",
				"key":  "-----BEGIN RSA PRIVATE KEY-----\nMIIB\n-----END RSA PRIVATE KEY-----",
			},
		},
	})
	if err != nil {
		t.Fatalf("parse options failed: %v", err)
	}
	if options.Endpoint != "opc.tcp://127.0.0.1:18540/induforge/sim" || options.SecurityPolicy != "Basic256Sha256" || options.SecurityMode != "Sign" {
		t.Fatalf("unexpected security options: %#v", options)
	}
	if options.AuthType != "username_password" || options.Username != "tester" || options.Password != "secret" {
		t.Fatalf("unexpected auth options: %#v", options)
	}
	if options.RootNodeID != "ns=2;s=Root" || options.MaxDepth != 4 || options.MaxNodes != 120 {
		t.Fatalf("unexpected browse limits: %#v", options)
	}
	if options.ConnectTimeout != 2500*time.Millisecond || options.RequestTimeout != 3500*time.Millisecond {
		t.Fatalf("unexpected timeouts: %#v", options)
	}
	if options.CertificatePEM == "" || options.PrivateKeyPEM == "" {
		t.Fatalf("expected ssl config copied: %#v", options)
	}
}

func TestParseOpcuaBrowseOptionsMissingEndpoint(t *testing.T) {
	if _, err := parseOpcuaBrowseOptions(map[string]any{}); err == nil {
		t.Fatal("expected missing endpoint error")
	}
}

func TestOpcuaBrowseOptionsClientOptionsRequiresUsername(t *testing.T) {
	options := opcuaBrowseOptions{
		Endpoint:        "opc.tcp://127.0.0.1:4840",
		SecurityPolicy:  "None",
		SecurityMode:    "None",
		AuthType:        "username_password",
		ConnectTimeout:  time.Second,
		RequestTimeout:  time.Second,
		SessionTimeout:  time.Second,
		ApplicationName: "test",
	}
	if _, err := options.clientOptions(context.Background()); err == nil {
		t.Fatal("expected username validation error")
	}
}

func TestOpcuaBrowseOptionsUsesClientCertificateOnlyForSecureMode(t *testing.T) {
	noneOptions := opcuaBrowseOptions{SecurityPolicy: "None", SecurityMode: "None", CertificatePEM: "bad-cert", PrivateKeyPEM: "bad-key"}
	if noneOptions.usesClientCertificate() {
		t.Fatal("None/None should ignore configured certificate material")
	}
	secureOptions := opcuaBrowseOptions{SecurityPolicy: "Basic256Sha256", SecurityMode: "Sign"}
	if !secureOptions.usesClientCertificate() {
		t.Fatal("secure endpoint should use configured certificate material")
	}
}

func TestOpcuaDataTypeName(t *testing.T) {
	cases := map[*ua.NodeID]string{
		ua.NewNumericNodeID(0, id.Boolean):  "Boolean",
		ua.NewNumericNodeID(0, id.Double):   "Double",
		ua.NewNumericNodeID(0, id.String):   "String",
		ua.NewNumericNodeID(0, id.DateTime): "DateTime",
		ua.NewStringNodeID(2, "CustomType"): "ns=2;s=CustomType",
	}
	for nodeID, expected := range cases {
		if actual := opcuaDataTypeName(nodeID); actual != expected {
			t.Fatalf("expected %s for %s, got %s", expected, nodeID.String(), actual)
		}
	}
}

func TestNormalizeOpcuaBrowseSecurityMode(t *testing.T) {
	cases := map[string]string{
		"":                 "None",
		"none":             "None",
		"Sign":             "Sign",
		"sign":             "Sign",
		"SignAndEncrypt":   "SignAndEncrypt",
		"sign_and_encrypt": "SignAndEncrypt",
	}
	for input, expected := range cases {
		if actual := normalizeOpcuaBrowseSecurityMode(input); actual != expected {
			t.Fatalf("expected %s for %q, got %s", expected, input, actual)
		}
	}
}

func TestParseOpcuaCertificatePEM(t *testing.T) {
	der, err := parseOpcuaCertificate("-----BEGIN CERTIFICATE-----\nAQID\n-----END CERTIFICATE-----")
	if err != nil {
		t.Fatalf("parse certificate failed: %v", err)
	}
	if len(der) != 3 || der[0] != 1 || der[1] != 2 || der[2] != 3 {
		t.Fatalf("unexpected der: %#v", der)
	}
}

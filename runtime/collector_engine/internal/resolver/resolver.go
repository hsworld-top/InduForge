// Package resolver is the only boundary that reads site resources and secret files.
package resolver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/indu-forge/collector-engine/internal/loader"
	"github.com/nats-io/nats.go"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Resolver struct {
	index     loader.Index
	directory string
}
type NATSConnection struct {
	URL     string
	Options []nats.Option
}
type ModbusEndpoint struct {
	Host string
	Port int
}
type OPCUAEndpoint struct{ URL string }

func New(loaded *loader.Loaded) (*Resolver, error) {
	if loaded == nil || !filepath.IsAbs(loaded.IndexDir) || filepath.Clean(loaded.IndexDir) != loaded.IndexDir {
		return nil, errors.New("resolver 配置非法")
	}
	return &Resolver{index: loaded.Index, directory: loaded.IndexDir}, nil
}
func (r *Resolver) ResolveNATS(_ context.Context, resourceRef, secretRef, accountID string) (NATSConnection, error) {
	var resource struct {
		URL       string `json:"url"`
		AccountID string `json:"accountId"`
	}
	if !r.resource(resourceRef, &resource, "url", "accountId") || resource.AccountID != accountID || !validNATSURL(resource.URL) {
		return NATSConnection{}, errors.New("NATS resource 不可用")
	}
	secret, err := r.secret(secretRef)
	if err != nil {
		return NATSConnection{}, errors.New("NATS credential 不可用")
	}
	var credential struct {
		SchemaVersion string `json:"schemaVersion"`
		AuthType      string `json:"authType"`
		Token         string `json:"token"`
		Username      string `json:"username"`
		Password      string `json:"password"`
	}
	if !strictNATSCredential(secret, &credential) || credential.SchemaVersion != "nats-credential.v1" {
		return NATSConnection{}, errors.New("NATS credential 不可用")
	}
	var opts []nats.Option
	switch credential.AuthType {
	case "token":
		if credential.Token == "" || credential.Username != "" || credential.Password != "" {
			return NATSConnection{}, errors.New("NATS credential 不可用")
		}
		opts = []nats.Option{nats.Token(credential.Token)}
	case "username-password":
		if credential.Token != "" || credential.Username == "" || credential.Password == "" {
			return NATSConnection{}, errors.New("NATS credential 不可用")
		}
		opts = []nats.Option{nats.UserInfo(credential.Username, credential.Password)}
	default:
		return NATSConnection{}, errors.New("NATS credential 不可用")
	}
	return NATSConnection{URL: resource.URL, Options: opts}, nil
}
func strictNATSCredential(raw []byte, target any) bool {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(target) != nil || d.Decode(&struct{}{}) != io.EOF {
		return false
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) != nil {
		return false
	}
	if _, ok := obj["schemaVersion"]; !ok {
		return false
	}
	if _, ok := obj["authType"]; !ok {
		return false
	}
	return len(obj) == 3 || len(obj) == 5
}
func (r *Resolver) ResolveModbus(_ context.Context, ref string) (ModbusEndpoint, error) {
	var v struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}
	if !r.resource(ref, &v, "host", "port") || v.Host == "" || v.Port < 1 || v.Port > 65535 {
		return ModbusEndpoint{}, errors.New("Modbus resource 不可用")
	}
	return ModbusEndpoint{Host: v.Host, Port: v.Port}, nil
}

// ResolveOPCUA only accepts an opc.tcp endpoint from the trusted resource index.
// V1 artifact fixes securityMode/securityPolicy to None and authenticationType to anonymous.
func (r *Resolver) ResolveOPCUA(_ context.Context, ref string) (OPCUAEndpoint, error) {
	var v struct {
		URL string `json:"url"`
	}
	if !r.resource(ref, &v, "url") {
		return OPCUAEndpoint{}, errors.New("OPC UA resource 不可用")
	}
	u, e := url.Parse(v.URL)
	if e != nil || u.Scheme != "opc.tcp" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return OPCUAEndpoint{}, errors.New("OPC UA resource 不可用")
	}
	return OPCUAEndpoint{URL: v.URL}, nil
}
func (r *Resolver) resource(ref string, target any, keys ...string) bool {
	raw, ok := r.index.Resources[ref]
	return ok && strict(raw, target, keys...)
}
func (r *Resolver) secret(ref string) ([]byte, error) {
	relative, ok := r.index.Secrets[ref]
	if !ok || !safeRelative(relative) {
		return nil, errors.New("secret 不可用")
	}
	path := filepath.Join(r.directory, relative)
	if filepath.Dir(path) != r.directory && !strings.HasPrefix(path, r.directory+string(filepath.Separator)) {
		return nil, errors.New("secret 不可用")
	}
	// Kubernetes Secret 投影使用 ..data 软链接原子切换。只接受解析后仍位于
	// resolver 索引目录下的普通文件，拒绝逃逸、循环和设备文件。
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, errors.New("secret 不可用")
	}
	resolvedRoot, err := filepath.EvalSymlinks(r.directory)
	if err != nil || !within(resolvedRoot, resolved) {
		return nil, errors.New("secret 不可用")
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.Mode().IsRegular() {
		return nil, errors.New("secret 不可用")
	}
	b, err := os.ReadFile(resolved)
	if err != nil || len(b) > 1<<20 {
		return nil, errors.New("secret 不可用")
	}
	return b, nil
}
func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
func strict(raw []byte, target any, keys ...string) bool {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(target) != nil || d.Decode(&struct{}{}) != io.EOF {
		return false
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) != nil || len(obj) != len(keys) {
		return false
	}
	for _, k := range keys {
		if _, ok := obj[k]; !ok {
			return false
		}
	}
	return true
}
func safeRelative(s string) bool {
	return s != "" && !filepath.IsAbs(s) && filepath.Clean(s) == s && !strings.HasPrefix(s, "../") && s != "." && s != ".." && !strings.ContainsAny(s, "\\\x00")
}
func validNATSURL(value string) bool {
	u, e := url.Parse(value)
	return e == nil && (u.Scheme == "nats" || u.Scheme == "tls") && u.Host != "" && u.User == nil && u.RawQuery == "" && u.Fragment == ""
}

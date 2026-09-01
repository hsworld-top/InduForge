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
	if !strict(secret, &credential, "schemaVersion", "authType", "token", "username", "password") || credential.SchemaVersion != "nats-credential.v1" {
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
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, errors.New("secret 不可用")
	}
	b, err := os.ReadFile(path)
	if err != nil || len(b) > 1<<20 {
		return nil, errors.New("secret 不可用")
	}
	return b, nil
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

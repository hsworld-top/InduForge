// Package resolver implements the shared, fail-closed site index boundary.
package resolver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	pathpkg "path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/indu-forge/runtime-engine/internal/compute"
	"github.com/indu-forge/runtime-engine/internal/transport/jetstream"
	"github.com/nats-io/nats.go"
	"golang.org/x/sys/unix"
)

const maxIndexBytes = 1 << 20

// The two readers are variables only to make the Linux mount boundary
// deterministic in unit tests.  Production keeps their fail-closed defaults:
// an unavailable procfs proof is never treated as a read-only mount.
var (
	mountInfoReader = func() ([]byte, error) { return os.ReadFile("/proc/self/mountinfo") }
	fdLinkReader    = func(f *os.File) (string, error) {
		return os.Readlink("/proc/self/fd/" + strconv.FormatUint(uint64(f.Fd()), 10))
	}
)

// Index has no exported secret values.  Its methods return only constrained
// connection objects needed by the respective adapters.
type Index struct {
	directory  string
	resources  map[string]json.RawMessage
	secrets    map[string]string
	production bool
}

func Open(path string, production bool) (*Index, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, errors.New("站点索引路径必须为规范绝对路径")
	}
	b, info, err := secureRead(path, production)
	if err != nil {
		return nil, errors.New("读取站点索引失败")
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("站点索引必须是普通文件")
	}
	if err := rejectDuplicateJSON(b); err != nil {
		return nil, errors.New("站点索引 JSON 非法")
	}
	var doc struct {
		SchemaVersion string                     `json:"schemaVersion"`
		Resources     map[string]json.RawMessage `json:"resources"`
		Secrets       map[string]string          `json:"secrets"`
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&doc); err != nil || dec.Decode(&struct{}{}) != io.EOF || doc.SchemaVersion != "collector-runtime-index.v1" || doc.Resources == nil || doc.Secrets == nil {
		return nil, errors.New("站点索引格式非法")
	}
	for ref, raw := range doc.Resources {
		if !validRef(ref, "site-resource://") || len(raw) == 0 || len(raw) > maxIndexBytes || !validJSON(raw) {
			return nil, errors.New("站点资源非法")
		}
	}
	dir := filepath.Dir(path)
	for ref, rel := range doc.Secrets {
		if !validRef(ref, "secret://") || !validRelative(rel) {
			return nil, errors.New("站点 secret 引用非法")
		}
	}
	return &Index{directory: dir, resources: doc.Resources, secrets: doc.Secrets, production: production}, nil
}

func (i *Index) ResolveNATS(_ context.Context, resourceRef, secretRef, accountID string) (jetstream.ConnectionOptions, error) {
	resource, ok := i.resources[resourceRef]
	if !ok {
		return jetstream.ConnectionOptions{}, errors.New("NATS resource 不存在")
	}
	var value struct {
		URL       string `json:"url"`
		AccountID string `json:"accountId"`
	}
	if !strictObject(resource, &value, "url", "accountId") || value.AccountID != accountID {
		return jetstream.ConnectionOptions{}, errors.New("NATS resource 非法")
	}
	u, err := url.Parse(value.URL)
	if err != nil || (u.Scheme != "nats" && u.Scheme != "tls") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return jetstream.ConnectionOptions{}, errors.New("NATS URL 非法")
	}
	secret, err := i.secret(secretRef)
	if err != nil {
		return jetstream.ConnectionOptions{}, err
	}
	var credential struct {
		SchemaVersion string `json:"schemaVersion"`
		AuthType      string `json:"authType"`
		Token         string `json:"token"`
		Username      string `json:"username"`
		Password      string `json:"password"`
	}
	if !allowedObject(secret, &credential, "schemaVersion", "authType", "token", "username", "password") || credential.SchemaVersion != "nats-credential.v1" {
		return jetstream.ConnectionOptions{}, errors.New("NATS credential 非法")
	}
	var opts []nats.Option
	switch credential.AuthType {
	case "none":
		if i.production || credential.Token != "" || credential.Username != "" || credential.Password != "" {
			return jetstream.ConnectionOptions{}, errors.New("NATS credential 非法")
		}
	case "token":
		if credential.Token == "" || credential.Username != "" || credential.Password != "" {
			return jetstream.ConnectionOptions{}, errors.New("NATS credential 非法")
		}
		opts = append(opts, nats.Token(credential.Token))
	case "username-password":
		if credential.Token != "" || credential.Username == "" || credential.Password == "" {
			return jetstream.ConnectionOptions{}, errors.New("NATS credential 非法")
		}
		opts = append(opts, nats.UserInfo(credential.Username, credential.Password))
	default:
		return jetstream.ConnectionOptions{}, errors.New("NATS credential 非法")
	}
	identity, err := jetstream.NewAccountIdentity(value.AccountID)
	if err != nil {
		return jetstream.ConnectionOptions{}, err
	}
	return jetstream.NewConnectionOptions(value.URL, identity, opts...)
}

func (i *Index) ResolvePostgres(_ context.Context, secretRef string) (string, error) {
	secret, err := i.secret(secretRef)
	if err != nil {
		return "", err
	}
	var value struct {
		SchemaVersion string `json:"schemaVersion"`
		DSN           string `json:"dsn"`
	}
	if !strictObject(secret, &value, "schemaVersion", "dsn") || value.SchemaVersion != "postgres-dsn.v1" || value.DSN == "" {
		return "", errors.New("PostgreSQL secret 非法")
	}
	return value.DSN, nil
}

func (i *Index) ResolveComputeSandbox(ctx context.Context, resourceRef, secretRef string) (compute.SandboxEndpoint, error) {
	if err := ctx.Err(); err != nil {
		return compute.SandboxEndpoint{}, err
	}
	resource, ok := i.resources[resourceRef]
	if !ok {
		return compute.SandboxEndpoint{}, errors.New("sandbox resource 不存在")
	}
	var r struct {
		URL string `json:"url"`
	}
	if !strictObject(resource, &r, "url") {
		return compute.SandboxEndpoint{}, errors.New("sandbox resource 非法")
	}
	secret, err := i.secret(secretRef)
	if err != nil {
		return compute.SandboxEndpoint{}, err
	}
	var s struct {
		SchemaVersion string `json:"schemaVersion"`
		Token         string `json:"token"`
	}
	if !strictObject(secret, &s, "schemaVersion", "token") || s.SchemaVersion != "bearer-token.v1" {
		return compute.SandboxEndpoint{}, errors.New("sandbox secret 非法")
	}
	return compute.NewSandboxEndpoint(r.URL, s.Token)
}

func (i *Index) secret(ref string) ([]byte, error) {
	if i == nil {
		return nil, errors.New("resolver 为空")
	}
	rel, ok := i.secrets[ref]
	if !ok || !validRelative(rel) {
		return nil, errors.New("secret 不存在")
	}
	path, err := containedRelativePath(i.directory, rel)
	if err != nil {
		return nil, errors.New("secret 路径逃逸")
	}
	b, _, err := secureRead(path, i.production)
	if err != nil {
		return nil, errors.New("读取 secret 失败")
	}
	if !validJSON(b) || rejectDuplicateJSON(b) != nil {
		return nil, errors.New("secret JSON 非法")
	}
	return b, nil
}
func secureRead(path string, readOnly bool) ([]byte, os.FileInfo, error) {
	fd, err := openNoFollow(path)
	if err != nil {
		return nil, nil, err
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || (readOnly && info.Mode().Perm()&022 != 0) {
		return nil, nil, errors.New("不安全文件")
	}
	// Permissions alone do not make a mounted configuration immutable: an
	// administrator or compromised sidecar could atomically rename another
	// 0644 file into the same directory.  In production, prove that this exact
	// already-open inode is covered by the most-specific Linux read-only mount.
	// If procfs cannot prove that fact (including non-Linux hosts), reject.
	if readOnly {
		if err := verifyReadOnlyMount(f, info); err != nil {
			return nil, nil, errors.New("文件未由只读挂载保护")
		}
	}
	b, err := io.ReadAll(io.LimitReader(f, maxIndexBytes+1))
	if err != nil || len(b) > maxIndexBytes {
		return nil, nil, errors.New("文件过大")
	}
	after, err := f.Stat()
	if err != nil || !os.SameFile(info, after) || info.Size() != after.Size() {
		return nil, nil, errors.New("文件变化")
	}
	return b, info, nil
}

func verifyReadOnlyMount(f *os.File, info os.FileInfo) error {
	if f == nil || info == nil {
		return errors.New("missing file")
	}
	target, err := fdLinkReader(f)
	if err != nil || target == "" || strings.HasSuffix(target, " (deleted)") || !filepath.IsAbs(target) || filepath.Clean(target) != target {
		return errors.New("cannot resolve descriptor")
	}
	// Resolve through procfs first, then re-stat the resulting path to ensure
	// mount matching refers to the same inode that will actually be consumed.
	// A later same-directory rename cannot change this FD's content.
	current, err := os.Stat(target)
	if err != nil || !os.SameFile(info, current) {
		return errors.New("descriptor identity changed")
	}
	mountInfo, err := mountInfoReader()
	if err != nil {
		return errors.New("mountinfo unavailable")
	}
	readOnly, err := mostSpecificReadOnlyMount(target, mountInfo)
	if err != nil || !readOnly {
		return errors.New("not read-only")
	}
	return nil
}

// mostSpecificReadOnlyMount returns the mount option for the deepest mount
// point covering path.  Checking only a parent mount is unsafe when a nested
// bind mount turns a subdirectory back to rw.
func mostSpecificReadOnlyMount(path string, mountInfo []byte) (bool, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return false, errors.New("unsafe path")
	}
	bestLength, bestReadOnly := -1, false
	for _, line := range strings.Split(strings.TrimSpace(string(mountInfo)), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 7 {
			continue
		}
		separator := -1
		for n, field := range fields {
			if field == "-" {
				separator = n
				break
			}
		}
		if separator < 6 || len(fields) <= separator+3 {
			continue
		}
		mountPoint, err := unescapeMountInfoPath(fields[4])
		if err != nil || !filepath.IsAbs(mountPoint) || filepath.Clean(mountPoint) != mountPoint || !pathCoveredByMount(path, mountPoint) {
			continue
		}
		if len(mountPoint) <= bestLength {
			continue
		}
		options := strings.Split(fields[5], ",")
		readOnly, readWrite := false, false
		for _, option := range options {
			readOnly = readOnly || option == "ro"
			readWrite = readWrite || option == "rw"
		}
		if readOnly == readWrite { // malformed or ambiguous mount flags fail closed.
			return false, errors.New("ambiguous mount flags")
		}
		bestLength, bestReadOnly = len(mountPoint), readOnly
	}
	if bestLength < 0 {
		return false, errors.New("covering mount not found")
	}
	return bestReadOnly, nil
}

func pathCoveredByMount(path, mountPoint string) bool {
	if mountPoint == string(filepath.Separator) {
		return true
	}
	return path == mountPoint || strings.HasPrefix(path, mountPoint+string(filepath.Separator))
}

func unescapeMountInfoPath(value string) (string, error) {
	var output strings.Builder
	for index := 0; index < len(value); {
		if value[index] != '\\' {
			output.WriteByte(value[index])
			index++
			continue
		}
		if index+3 >= len(value) || value[index+1] < '0' || value[index+1] > '7' || value[index+2] < '0' || value[index+2] > '7' || value[index+3] < '0' || value[index+3] > '7' {
			return "", errors.New("invalid mount escape")
		}
		n, err := strconv.ParseUint(value[index+1:index+4], 8, 8)
		if err != nil {
			return "", err
		}
		output.WriteByte(byte(n))
		index += 4
	}
	return output.String(), nil
}

// openNoFollow resolves every absolute path component through directory file
// descriptors.  A prior Lstat check is insufficient because an attacker can
// swap a parent directory between checking it and opening its child.
func openNoFollow(path string) (int, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return -1, errors.New("unsafe path")
	}
	segments := strings.Split(strings.TrimPrefix(path, string(filepath.Separator)), string(filepath.Separator))
	fd, err := unix.Open(string(filepath.Separator), unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return -1, err
	}
	for index, segment := range segments {
		if segment == "" || segment == "." || segment == ".." {
			unix.Close(fd)
			return -1, errors.New("unsafe path component")
		}
		flags := unix.O_RDONLY | unix.O_NOFOLLOW | unix.O_CLOEXEC
		if index < len(segments)-1 {
			flags |= unix.O_DIRECTORY
		}
		next, openErr := unix.Openat(fd, segment, flags, 0)
		unix.Close(fd)
		if openErr != nil {
			return -1, openErr
		}
		fd = next
	}
	return fd, nil
}
func validRef(v, prefix string) bool {
	if (prefix != "site-resource://" && prefix != "secret://") || !strings.HasPrefix(v, prefix) {
		return false
	}
	tail := v[len(prefix):]
	// Mirrors ^(site-resource|secret)://[A-Za-z0-9][A-Za-z0-9._:/@-]{0,255}$
	// exactly: the identifier tail is mandatory and is bounded independently
	// from the scheme prefix.
	if len(tail) == 0 || len(tail) > 256 || !asciiAlphaNumeric(rune(tail[0])) {
		return false
	}
	for _, char := range tail[1:] {
		if !(asciiAlphaNumeric(char) || char == '.' || char == '_' || char == ':' || char == '/' || char == '@' || char == '-') {
			return false
		}
	}
	return true
}

func asciiAlphaNumeric(value rune) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value >= '0' && value <= '9'
}
func validRelative(v string) bool {
	if v == "" || filepath.IsAbs(v) || filepath.VolumeName(v) != "" || strings.HasPrefix(v, "/") || pathpkg.Clean(v) != v || strings.Contains(v, "\\") || strings.Contains(v, ":") || strings.ContainsRune(v, '\x00') || v == "." || v == ".." || strings.HasPrefix(v, "../") {
		return false
	}
	for _, segment := range strings.Split(v, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

// containedRelativePath accepts a normalized multi-segment site secret such
// as secrets/pg.json while proving its lexical result cannot escape directory.
// openNoFollow subsequently resolves every component from /, so a parent
// replacement with a symlink is still rejected at the actual use point.
func containedRelativePath(directory, relative string) (string, error) {
	if !filepath.IsAbs(directory) || filepath.Clean(directory) != directory || !validRelative(relative) {
		return "", errors.New("unsafe relative path")
	}
	path := filepath.Join(directory, filepath.FromSlash(relative))
	contained, err := filepath.Rel(directory, path)
	contained = filepath.ToSlash(contained)
	if err != nil || contained != relative || !validRelative(contained) {
		return "", errors.New("path escapes directory")
	}
	return path, nil
}
func validJSON(b []byte) bool {
	var v any
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	return d.Decode(&v) == nil && d.Decode(&struct{}{}) == io.EOF
}
func strictObject(raw []byte, dest any, allowed ...string) bool {
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil || m == nil {
		return false
	}
	if len(m) != len(allowed) {
		return false
	}
	for _, k := range allowed {
		if _, ok := m[k]; !ok {
			return false
		}
	}
	return json.Unmarshal(raw, dest) == nil
}
func allowedObject(raw []byte, dest any, allowed ...string) bool {
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil || m == nil {
		return false
	}
	permitted := map[string]bool{}
	for _, key := range allowed {
		permitted[key] = true
	}
	for key := range m {
		if !permitted[key] {
			return false
		}
	}
	return json.Unmarshal(raw, dest) == nil
}

// encoding/json otherwise accepts duplicate object members.  Scan every
// object token first, including nested inline resources, before decoding.
func rejectDuplicateJSON(raw []byte) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	var walk func() error
	walk = func() error {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch x := tok.(type) {
		case json.Delim:
			if x == '{' {
				seen := map[string]bool{}
				for d.More() {
					k, e := d.Token()
					if e != nil {
						return e
					}
					key, ok := k.(string)
					if !ok || seen[key] {
						return errors.New("duplicate")
					}
					seen[key] = true
					if e = walk(); e != nil {
						return e
					}
				}
				_, err = d.Token()
				return err
			}
			if x == '[' {
				for d.More() {
					if err := walk(); err != nil {
						return err
					}
				}
				_, err = d.Token()
				return err
			}
		}
		return nil
	}
	if err := walk(); err != nil {
		return err
	}
	if d.More() {
		return errors.New("trailing")
	}
	return nil
}

var _ compute.SandboxResolver = (*Index)(nil)

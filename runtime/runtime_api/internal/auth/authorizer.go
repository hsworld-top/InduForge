package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"

	"github.com/indu-forge/runtime-api/internal/securefile"
)

var sha256Hex = regexp.MustCompile(`^[0-9a-f]{64}$`)

type Principal struct {
	SubjectID    string     `json:"subjectId"`
	Username     string     `json:"username,omitempty"`
	DisplayName  string     `json:"displayName,omitempty"`
	Roles        []string   `json:"roles"`
	Capabilities []string   `json:"capabilities,omitempty"`
	ExpiresAt    *time.Time `json:"-"`
}

type tokenRecord struct {
	TokenSHA256  string     `json:"tokenSha256"`
	SubjectID    string     `json:"subjectId"`
	Roles        []string   `json:"roles"`
	Capabilities []string   `json:"capabilities,omitempty"`
	ExpiresAt    *time.Time `json:"expiresAt"`
}

type userRecord struct {
	Username     string     `json:"username"`
	PasswordHash string     `json:"passwordHash"`
	SubjectID    string     `json:"subjectId"`
	DisplayName  string     `json:"displayName,omitempty"`
	Status       string     `json:"status"`
	Roles        []string   `json:"roles"`
	Capabilities []string   `json:"capabilities,omitempty"`
	ExpiresAt    *time.Time `json:"expiresAt"`
}

type tokenSecret struct {
	SchemaVersion string        `json:"schemaVersion"`
	Tokens        []tokenRecord `json:"tokens"`
	Users         []userRecord  `json:"users,omitempty"`
}

type Authorizer struct {
	tokens []tokenRecord
	users  []userRecord
}

func Load(path string) (*Authorizer, error) {
	payload, err := securefile.ReadSecret(path, 1<<20)
	if err != nil {
		return nil, fmt.Errorf("读取 Runtime API token secret: %w", err)
	}
	var secret tokenSecret
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&secret); err != nil {
		return nil, fmt.Errorf("解析 Runtime API token secret: %w", err)
	}
	if secret.SchemaVersion != "runtime-api-tokens.v1" || len(secret.Tokens) == 0 || len(secret.Tokens) > 1024 {
		return nil, errors.New("Runtime API token secret 版本或数量非法")
	}
	seen := map[string]struct{}{}
	for _, token := range secret.Tokens {
		if !sha256Hex.MatchString(token.TokenSHA256) || strings.TrimSpace(token.SubjectID) == "" || len(token.Roles) == 0 {
			return nil, errors.New("Runtime API token 记录非法")
		}
		if _, exists := seen[token.TokenSHA256]; exists {
			return nil, errors.New("Runtime API token digest 重复")
		}
		seen[token.TokenSHA256] = struct{}{}
	}
	for _, user := range secret.Users {
		if strings.TrimSpace(user.Username) == "" || strings.TrimSpace(user.SubjectID) == "" || user.Status == "disabled" || len(user.Roles) == 0 || !strings.HasPrefix(user.PasswordHash, "$argon2id$") {
			return nil, errors.New("Runtime API 运行用户记录非法")
		}
	}
	return &Authorizer{tokens: append([]tokenRecord(nil), secret.Tokens...), users: append([]userRecord(nil), secret.Users...)}, nil
}

func (a *Authorizer) Authenticate(request *http.Request, now time.Time) (Principal, error) {
	header := strings.TrimSpace(request.Header.Get("Authorization"))
	if !strings.HasPrefix(header, "Bearer ") || len(header) <= len("Bearer ") {
		return Principal{}, errors.New("missing bearer token")
	}
	raw := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	if raw == "" || len(raw) > 4096 {
		return Principal{}, errors.New("invalid bearer token")
	}
	digest := sha256.Sum256([]byte(raw))
	encoded := hex.EncodeToString(digest[:])
	for _, record := range a.tokens {
		if subtle.ConstantTimeCompare([]byte(encoded), []byte(record.TokenSHA256)) != 1 {
			continue
		}
		if record.ExpiresAt != nil && !now.Before(record.ExpiresAt.UTC()) {
			return Principal{}, errors.New("expired bearer token")
		}
		return Principal{SubjectID: record.SubjectID, Roles: append([]string(nil), record.Roles...), Capabilities: append([]string(nil), record.Capabilities...), ExpiresAt: record.ExpiresAt}, nil
	}
	return Principal{}, errors.New("invalid bearer token")
}

func (a *Authorizer) AuthenticateCredentials(username, password string, now time.Time) (Principal, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" || len(password) > 1024 {
		return Principal{}, errors.New("invalid credentials")
	}
	for _, user := range a.users {
		if user.Username != username {
			continue
		}
		if user.Status == "disabled" {
			return Principal{}, errors.New("disabled runtime user")
		}
		if user.ExpiresAt != nil && !now.Before(user.ExpiresAt.UTC()) {
			return Principal{}, errors.New("expired runtime user")
		}
		if !verifyArgon2id(password, user.PasswordHash) {
			return Principal{}, errors.New("invalid credentials")
		}
		return Principal{SubjectID: user.SubjectID, Username: user.Username, DisplayName: user.DisplayName, Roles: append([]string(nil), user.Roles...), Capabilities: append([]string(nil), user.Capabilities...), ExpiresAt: user.ExpiresAt}, nil
	}
	return Principal{}, errors.New("invalid credentials")
}

func verifyArgon2id(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}
	var version int
	var memory, iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false
	}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(expected) == 0 {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func DigestToken(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

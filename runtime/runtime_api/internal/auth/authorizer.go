package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/indu-forge/runtime-api/internal/securefile"
)

var sha256Hex = regexp.MustCompile(`^[0-9a-f]{64}$`)

type Principal struct {
	SubjectID    string     `json:"subjectId"`
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

type tokenSecret struct {
	SchemaVersion string        `json:"schemaVersion"`
	Tokens        []tokenRecord `json:"tokens"`
}

type Authorizer struct{ tokens []tokenRecord }

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
	return &Authorizer{tokens: append([]tokenRecord(nil), secret.Tokens...)}, nil
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

func DigestToken(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

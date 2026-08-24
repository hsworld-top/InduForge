package service

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func workbenchHeaderSecretSuffix(name string) string {
	digest := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(name))))
	return "header." + hex.EncodeToString(digest[:8])
}

func isSensitiveWorkbenchHeader(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, marker := range []string{"authorization", "cookie", "token", "secret", "api-key", "apikey", "password", "private-key"} {
		if strings.Contains(name, marker) {
			return true
		}
	}
	return false
}

func splitWorkbenchSecrets(headers []any, auth map[string]any) ([]any, map[string]any, map[string]string) {
	cleanHeaders := make([]any, 0, len(headers))
	for _, raw := range headers {
		if row, ok := raw.(map[string]any); ok {
			cleanHeaders = append(cleanHeaders, cloneMap(row))
		} else {
			cleanHeaders = append(cleanHeaders, raw)
		}
	}
	cleanAuth := cloneMap(auth)
	secrets := map[string]string{}
	for _, raw := range cleanHeaders {
		row, ok := raw.(map[string]any)
		if !ok || !isSensitiveWorkbenchHeader(toString(row["key"])) {
			continue
		}
		if value := toString(row["value"]); value != "" {
			secrets[workbenchHeaderSecretSuffix(toString(row["key"]))] = value
		}
		row["value"] = ""
		row["secretSet"] = secrets[workbenchHeaderSecretSuffix(toString(row["key"]))] != ""
	}
	for _, key := range []string{"password", "token"} {
		if value := toString(cleanAuth[key]); value != "" {
			secrets["auth."+key] = value
		}
		delete(cleanAuth, key)
		cleanAuth[key+"Set"] = secrets["auth."+key] != ""
	}
	return cleanHeaders, cleanAuth, secrets
}

func hydrateWorkbenchSecrets(headers []any, auth map[string]any, secrets map[string]string) ([]any, map[string]any) {
	resultHeaders := make([]any, 0, len(headers))
	for _, raw := range headers {
		if row, ok := raw.(map[string]any); ok {
			resultHeaders = append(resultHeaders, cloneMap(row))
		} else {
			resultHeaders = append(resultHeaders, raw)
		}
	}
	resultAuth := cloneMap(auth)
	for _, raw := range resultHeaders {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if value, exists := secrets[workbenchHeaderSecretSuffix(toString(row["key"]))]; exists {
			row["value"] = value
		}
	}
	for _, key := range []string{"password", "token"} {
		if value, exists := secrets["auth."+key]; exists {
			resultAuth[key] = value
		}
	}
	return resultHeaders, resultAuth
}

func workbenchScopedSecrets(all map[string]string, prefix string) map[string]string {
	result := map[string]string{}
	for key, value := range all {
		if strings.HasPrefix(key, prefix) {
			result[strings.TrimPrefix(key, prefix)] = value
		}
	}
	return result
}

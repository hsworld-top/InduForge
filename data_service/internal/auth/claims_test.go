package auth

import "testing"

func TestClaimsHasCapabilitySupportsWildcard(t *testing.T) {
	claims := &Claims{
		Capabilities: []string{"*"},
	}

	if !claims.HasCapability("project:read") {
		t.Fatalf("expected wildcard capability to grant project:read")
	}
}

func TestClaimsHasProjectAccessSupportsWildcard(t *testing.T) {
	claims := &Claims{
		ProjectIDs: []string{"*"},
	}

	if !claims.HasProjectAccess("project-1") {
		t.Fatalf("expected wildcard project id to grant project access")
	}
}

func TestClaimsGlobalRoleCompatibilitySupportsLegacyToken(t *testing.T) {
	claims := &Claims{
		Role: "SYSTEM_ADMIN",
	}

	if !claims.HasCapability("project:read") {
		t.Fatalf("expected SYSTEM_ADMIN legacy token to grant project:read")
	}
	if !claims.HasProjectAccess("project-1") {
		t.Fatalf("expected SYSTEM_ADMIN legacy token to grant project access")
	}
}

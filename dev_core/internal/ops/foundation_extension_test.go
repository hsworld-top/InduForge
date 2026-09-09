package ops

import "testing"

func TestFoundationExtension(t *testing.T) {
	existing := map[string]string{"traefik": "center"}
	if err := validateFoundationExtension(existing, []FoundationAssignment{{ServiceType: "traefik", NodeID: "center"}, {ServiceType: "if_realtime", NodeID: "new-node"}}); err != nil {
		t.Fatal(err)
	}
	if err := validateFoundationExtension(existing, []FoundationAssignment{{ServiceType: "traefik", NodeID: "new-node"}}); err != ErrFoundationMoveUnsupported {
		t.Fatalf("migration allowed: %v", err)
	}
}

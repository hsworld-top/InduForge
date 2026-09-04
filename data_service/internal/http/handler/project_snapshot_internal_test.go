package handler

import "testing"

func TestValidateInternalSnapshotIdentity(t *testing.T) {
	projectID := "11111111-1111-4111-8111-111111111111"
	tenantID := "22222222-2222-4222-8222-222222222222"
	actorID := "33333333-3333-4333-8333-333333333333"
	if err := validateInternalSnapshotIdentity(projectID, tenantID, actorID, true); err != nil {
		t.Fatal(err)
	}
	for _, input := range [][3]string{{"bad", tenantID, actorID}, {projectID, "bad", actorID}, {projectID, tenantID, "bad"}} {
		if err := validateInternalSnapshotIdentity(input[0], input[1], input[2], true); err == nil {
			t.Fatalf("invalid identity accepted: %v", input)
		}
	}
}

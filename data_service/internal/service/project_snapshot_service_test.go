package service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/indu-forge/data_service/internal/repository"
)

func TestNormalizeProjectSnapshotDefaultsDataPointRuntimePermissionsWhenOmitted(t *testing.T) {
	projectID := uuid.NewString()

	var snapshot repository.ProjectSnapshot
	if err := json.Unmarshal([]byte(`{
		"datapoints": [
			{
				"id": "`+uuid.NewString()+`",
				"projectId": "`+projectID+`",
				"path": "metrics.snapshot.default",
				"name": "metrics.snapshot.default",
				"sourceType": "manual",
				"sourceConfig": {},
				"dataType": "string",
				"tags": [],
				"refreshMode": "auto",
				"status": "active"
			}
		]
	}`), &snapshot); err != nil {
		t.Fatalf("unmarshal snapshot failed: %v", err)
	}

	normalized := normalizeProjectSnapshot(snapshot)
	writeGrant := normalized.DataPoints[0].RuntimePermissions.Write
	if !writeGrant.Inherit {
		t.Fatal("expected snapshot datapoint write permission to default inherit=true")
	}
	if len(writeGrant.AllowRoles) != 0 {
		t.Fatalf("expected empty allowRoles, got %v", writeGrant.AllowRoles)
	}
	if len(writeGrant.DenyRoles) != 0 {
		t.Fatalf("expected empty denyRoles, got %v", writeGrant.DenyRoles)
	}

	artifact := repository.BuildProjectArtifactV1(projectID, &normalized, time.Now().UTC())
	artifactWriteGrant := artifact.DataPoints[0].RuntimePermissions.Write
	if !artifactWriteGrant.Inherit {
		t.Fatal("expected artifact datapoint write permission to default inherit=true")
	}
}

func TestNormalizeProjectSnapshotRejectsMalformedRuntimePermissions(t *testing.T) {
	projectID := uuid.NewString()

	testCases := []struct {
		name    string
		payload string
	}{
		{
			name: "runtimePermissions null",
			payload: `{
				"datapoints": [{
					"id": "` + uuid.NewString() + `",
					"projectId": "` + projectID + `",
					"path": "metrics.snapshot.invalid.null",
					"name": "metrics.snapshot.invalid.null",
					"sourceType": "manual",
					"sourceConfig": {},
					"dataType": "string",
					"tags": [],
					"refreshMode": "auto",
					"status": "active",
					"runtimePermissions": null
				}]
			}`,
		},
		{
			name: "runtimePermissions empty object",
			payload: `{
				"datapoints": [{
					"id": "` + uuid.NewString() + `",
					"projectId": "` + projectID + `",
					"path": "metrics.snapshot.invalid.empty",
					"name": "metrics.snapshot.invalid.empty",
					"sourceType": "manual",
					"sourceConfig": {},
					"dataType": "string",
					"tags": [],
					"refreshMode": "auto",
					"status": "active",
					"runtimePermissions": {}
				}]
			}`,
		},
		{
			name: "runtimePermissions write null",
			payload: `{
				"datapoints": [{
					"id": "` + uuid.NewString() + `",
					"projectId": "` + projectID + `",
					"path": "metrics.snapshot.invalid.write-null",
					"name": "metrics.snapshot.invalid.write-null",
					"sourceType": "manual",
					"sourceConfig": {},
					"dataType": "string",
					"tags": [],
					"refreshMode": "auto",
					"status": "active",
					"runtimePermissions": { "write": null }
				}]
			}`,
		},
		{
			name: "runtimePermissions write inherit null",
			payload: `{
				"datapoints": [{
					"id": "` + uuid.NewString() + `",
					"projectId": "` + projectID + `",
					"path": "metrics.snapshot.invalid.inherit-null",
					"name": "metrics.snapshot.invalid.inherit-null",
					"sourceType": "manual",
					"sourceConfig": {},
					"dataType": "string",
					"tags": [],
					"refreshMode": "auto",
					"status": "active",
					"runtimePermissions": { "write": { "inherit": null } }
				}]
			}`,
		},
		{
			name: "runtimePermissions write allowRoles null",
			payload: `{
				"datapoints": [{
					"id": "` + uuid.NewString() + `",
					"projectId": "` + projectID + `",
					"path": "metrics.snapshot.invalid.allowRoles-null",
					"name": "metrics.snapshot.invalid.allowRoles-null",
					"sourceType": "manual",
					"sourceConfig": {},
					"dataType": "string",
					"tags": [],
					"refreshMode": "auto",
					"status": "active",
					"runtimePermissions": { "write": { "allowRoles": null, "inherit": true } }
				}]
			}`,
		},
		{
			name: "runtimePermissions write denyRoles null",
			payload: `{
				"datapoints": [{
					"id": "` + uuid.NewString() + `",
					"projectId": "` + projectID + `",
					"path": "metrics.snapshot.invalid.denyRoles-null",
					"name": "metrics.snapshot.invalid.denyRoles-null",
					"sourceType": "manual",
					"sourceConfig": {},
					"dataType": "string",
					"tags": [],
					"refreshMode": "auto",
					"status": "active",
					"runtimePermissions": { "write": { "denyRoles": null, "inherit": true } }
				}]
			}`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var snapshot repository.ProjectSnapshot
			if err := json.Unmarshal([]byte(testCase.payload), &snapshot); err == nil {
				t.Fatal("expected malformed runtimePermissions to be rejected")
			}
		})
	}
}

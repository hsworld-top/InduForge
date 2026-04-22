package repository

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNormalizeDataPointRuntimePermissionsDefaults(t *testing.T) {
	permissions, err := unmarshalDataPointRuntimePermissions(nil)
	if err != nil {
		t.Fatalf("expected default runtime permissions, got %v", err)
	}

	if !permissions.Write.Inherit {
		t.Fatal("expected default write permission to inherit")
	}
	if len(permissions.Write.AllowRoles) != 0 {
		t.Fatalf("expected empty allowRoles, got %v", permissions.Write.AllowRoles)
	}
	if len(permissions.Write.DenyRoles) != 0 {
		t.Fatalf("expected empty denyRoles, got %v", permissions.Write.DenyRoles)
	}
}

func TestUnmarshalDataPointRuntimePermissionsTreatsLegacyEmptyObjectAsDefault(t *testing.T) {
	permissions, err := unmarshalDataPointRuntimePermissions([]byte(`{}`))
	if err != nil {
		t.Fatalf("expected legacy empty object to fall back to default runtime permissions, got %v", err)
	}

	if !permissions.Write.Inherit {
		t.Fatal("expected legacy empty object write permission to default inherit=true")
	}
	if len(permissions.Write.AllowRoles) != 0 {
		t.Fatalf("expected empty allowRoles, got %v", permissions.Write.AllowRoles)
	}
	if len(permissions.Write.DenyRoles) != 0 {
		t.Fatalf("expected empty denyRoles, got %v", permissions.Write.DenyRoles)
	}
}

func TestDataPointRuntimePermissionsRejectsExplicitNullFields(t *testing.T) {
	testCases := []struct {
		name    string
		payload string
	}{
		{
			name:    "write.inherit null",
			payload: `{"write":{"inherit":null}}`,
		},
		{
			name:    "write.allowRoles null",
			payload: `{"write":{"allowRoles":null,"inherit":true}}`,
		},
		{
			name:    "write.denyRoles null",
			payload: `{"write":{"denyRoles":null,"inherit":true}}`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var permissions DataPointRuntimePermissions
			if err := json.Unmarshal([]byte(testCase.payload), &permissions); err == nil {
				t.Fatal("expected explicit null runtime permission field to be rejected")
			}
		})
	}
}

func TestBuildProjectArtifactV1IncludesDataPointRuntimePermissions(t *testing.T) {
	artifact := BuildProjectArtifactV1("project-1", &ProjectSnapshot{
		DataPoints: []DataPointRecord{
			{
				ID:         "dp-1",
				ProjectID:  "project-1",
				Path:       "metrics.temp",
				Name:       "metrics.temp",
				SourceType: "manual",
				SourceConfig: map[string]any{
					"static": true,
				},
				DataType:    "number",
				RefreshMode: "auto",
				Status:      "active",
				RuntimePermissions: DataPointRuntimePermissions{
					Write: RuntimePermissionGrant{
						AllowRoles: []string{"operator"},
						DenyRoles:  []string{"guest"},
						Inherit:    false,
					},
				},
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			},
		},
	}, time.Now().UTC())

	if len(artifact.DataPoints) != 1 {
		t.Fatalf("expected 1 datapoint in artifact, got %d", len(artifact.DataPoints))
	}

	writeGrant := artifact.DataPoints[0].RuntimePermissions.Write
	if writeGrant.Inherit {
		t.Fatal("expected artifact write permission inherit=false")
	}
	if len(writeGrant.AllowRoles) != 1 || writeGrant.AllowRoles[0] != "operator" {
		t.Fatalf("expected allowRoles to contain operator, got %v", writeGrant.AllowRoles)
	}
	if len(writeGrant.DenyRoles) != 1 || writeGrant.DenyRoles[0] != "guest" {
		t.Fatalf("expected denyRoles to contain guest, got %v", writeGrant.DenyRoles)
	}
}

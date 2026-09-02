package ops

import (
	"context"
	"reflect"
	"testing"

	"github.com/indu-forge/dev_core/internal/auth"
)

func TestDeploymentEngineRequirementsAlwaysIncludesBase(t *testing.T) {
	got, err := deploymentEngineRequirements([]byte(`{"capabilities":["alarm","collector"]}`))
	if err != nil || !reflect.DeepEqual(got, []string{ServiceBase, ServiceAlarm, ServiceCollector}) {
		t.Fatalf("unexpected requirements: %#v, %v", got, err)
	}
}

func TestDevelopmentEngineRequirementsUsesConfiguredAuthoritativeBuilder(t *testing.T) {
	service := NewService(nil, nil)
	called := false
	service.SetDevelopmentRequirementsBuilder(func(_ context.Context, actor auth.User, projectID, authorization string) ([]string, error) {
		called = true
		if actor.TenantID != "tenant" || projectID != testProjectID || authorization != "Bearer token" {
			t.Fatalf("unexpected preflight input: actor=%#v project=%s authorization=%q", actor, projectID, authorization)
		}
		return []string{ServiceBase, ServiceCompute, ServiceCollector}, nil
	})
	requirements, err := service.DevelopmentEngineRequirements(context.Background(), auth.User{TenantID: "tenant", Role: "OPS_ADMIN"}, testProjectID, "Bearer token")
	if err != nil || !reflect.DeepEqual(requirements, []string{ServiceBase, ServiceCompute, ServiceCollector}) || !called {
		t.Fatalf("requirements=%v called=%v err=%v", requirements, called, err)
	}
}

func TestValidateEnginePlacementsRejectsManualOptionalEngine(t *testing.T) {
	if err := validateEnginePlacements([]string{ServiceBase}, map[string]string{ServiceBase: "node-1", ServiceCompute: "node-1"}); err == nil {
		t.Fatal("unused engine was accepted")
	}
}

func TestValidateEnginePlacementsUsesBaseEngineDisplayName(t *testing.T) {
	err := validateEnginePlacements([]string{ServiceBase}, nil)
	if err == nil || err.Error() != ServiceBaseName+"必须选择部署节点" {
		t.Fatalf("基础引擎文案不统一: %v", err)
	}
}

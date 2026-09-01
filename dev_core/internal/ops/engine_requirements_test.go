package ops

import (
	"reflect"
	"testing"
)

func TestDeploymentEngineRequirementsAlwaysIncludesBase(t *testing.T) {
	got, err := deploymentEngineRequirements([]byte(`{"capabilities":["alarm","collector"]}`))
	if err != nil || !reflect.DeepEqual(got, []string{ServiceBase, ServiceAlarm, ServiceCollector}) {
		t.Fatalf("unexpected requirements: %#v, %v", got, err)
	}
}

func TestValidateEnginePlacementsRejectsManualOptionalEngine(t *testing.T) {
	if err := validateEnginePlacements([]string{ServiceBase}, map[string]string{ServiceBase: "node-1", ServiceCompute: "node-1"}); err == nil {
		t.Fatal("unused engine was accepted")
	}
}

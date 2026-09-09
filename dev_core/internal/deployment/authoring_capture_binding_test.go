package deployment

import (
	"context"
	"errors"
	"testing"
)

type recordingProjectBindingEnsurer struct {
	projectID, tenantID, epoch string
	err                        error
}

func (e *recordingProjectBindingEnsurer) EnsureProjectTenantBindingAtEpoch(_ context.Context, projectID, tenantID, epoch string) error {
	e.projectID, e.tenantID, e.epoch = projectID, tenantID, epoch
	return e.err
}

func TestEnsureDataProjectBindingUsesCurrentProjectIdentityAndEpoch(t *testing.T) {
	ensurer := &recordingProjectBindingEnsurer{}
	project := Project{ID: "60cae81f-6aad-484d-8778-da28e03051e2", TenantID: "550e8400-e29b-41d4-a716-446655440000", AuthoringEpoch: 3}
	if err := ensureDataProjectBinding(context.Background(), ensurer, project); err != nil {
		t.Fatal(err)
	}
	if ensurer.projectID != project.ID || ensurer.tenantID != project.TenantID || ensurer.epoch != "epoch-3" {
		t.Fatalf("数据域工程绑定身份错误: %#v", ensurer)
	}
}

func TestEnsureDataProjectBindingStopsCaptureWhenSynchronizationFails(t *testing.T) {
	want := errors.New("binding unavailable")
	err := ensureDataProjectBinding(context.Background(), &recordingProjectBindingEnsurer{err: want}, Project{ID: "project", TenantID: "tenant", AuthoringEpoch: 1})
	if !errors.Is(err, want) {
		t.Fatalf("绑定失败应保留根因: %v", err)
	}
}

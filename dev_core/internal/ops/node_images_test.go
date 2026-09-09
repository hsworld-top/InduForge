package ops

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type blockedImages struct {
	err  error
	refs []string
}

func (p *blockedImages) EnsureNodeImages(_ context.Context, _ string, refs []string) error {
	p.refs = refs
	return p.err
}
func TestImageGatePreservesWorkloadBeforeAnyKubernetesWrite(t *testing.T) {
	requests := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests++; w.WriteHeader(200) }))
	defer s.Close()
	for _, gateErr := range []error{&imagePreparationPending{message: "正在下载运行资源"}, errors.New("镜像校验失败")} {
		p := &blockedImages{err: gateErr}
		r := &KubernetesProjectReconciler{client: s.Client(), endpoint: s.URL, imagePreparer: p}
		if err := r.Reconcile(context.Background(), ProjectWorkload{NodeID: "node", Engine: ServiceBase}); err != gateErr {
			t.Fatal(err)
		}
		if len(p.refs) != 3 || requests != 0 {
			t.Fatalf("refs=%v requests=%d", p.refs, requests)
		}
	}
}
func TestImageReferenceCoverage(t *testing.T) {
	if len(projectImageReferences(ServiceCollector)) != 0 {
		t.Fatal("native collector should not need image")
	}
	if len(projectImageReferences(ServiceCompute)) != 2 {
		t.Fatal("compute missing sandbox")
	}
	if len(foundationImageReferences("if_message")) == 0 {
		t.Fatal("MQTT image missing")
	}
}

package ops

import (
	"strings"
	"testing"
)

func TestFoundationResourceRefsAreInternalAndDeterministic(t *testing.T) {
	environmentID := "66666666-6666-4666-8666-666666666666"
	nats, err := foundationResourceRefs(environmentID, "nats_jetstream")
	if err != nil || nats["service"] != "nats://nats.if-env-666666666666.svc.cluster.local:4222" || nats["secretKey"] != "nats-token" || nats["authMode"] != "environment-token+project-subject-isolation" {
		t.Fatalf("nats refs invalid: %v %#v", err, nats)
	}
	postgres, err := foundationResourceRefs(environmentID, "if_history")
	if err != nil || postgres["database"] != "induforge_runtime" || postgres["adminUser"] != "postgres" || postgres["secretKey"] != "postgres-password" || !strings.HasPrefix(postgres["schemaPrefix"], "runtime_") {
		t.Fatalf("postgres refs invalid: %v %#v", err, postgres)
	}
	if strings.Contains(strings.Join([]string{nats["service"], nats["secretKey"], postgres["service"], postgres["secretKey"]}, " "), "password-value") {
		t.Fatal("resource refs leaked a value")
	}
	if refs, err := foundationResourceRefs(environmentID, "if_realtime"); err != nil || refs != nil {
		t.Fatalf("unrelated service got refs: %v %#v", err, refs)
	}
}

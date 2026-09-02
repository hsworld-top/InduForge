package hostd

import (
	"fmt"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestFoundationPlanRendersFixedValidManifest(t *testing.T) {
	plan := FoundationPlan{
		SchemaVersion: FoundationSchemaVersion,
		Generation:    1,
		EnvironmentID: "11111111-1111-4111-8111-111111111111",
		NodeID:        "22222222-2222-4222-8222-222222222222",
		Assignments: map[string]string{
			"postgres": "33333333-3333-4333-8333-333333333333",
			"redis":    "44444444-4444-4444-8444-444444444444",
			"emqx":     "55555555-5555-4555-8555-555555555555",
			"nats":     "66666666-6666-4666-8666-666666666666",
			"object":   "77777777-7777-4777-8777-777777777777",
			"nginx":    "88888888-8888-4888-8888-888888888888",
		},
	}
	if err := plan.Validate(); err != nil {
		t.Fatal(err)
	}
	manifest := plan.RenderManifest("private-secret-value-0123456789")
	if strings.Contains(manifest, "kubernetes.io/hostname") || !strings.Contains(manifest, "induforge.io/host-node-id: 33333333-3333-4333-8333-333333333333") {
		t.Fatal("基础服务必须使用完整平台节点 ID 的稳定标签调度")
	}
	for _, expected := range []string{"kind: Role\nmetadata:\n  name: induforge-project-reconciler", "kind: RoleBinding", "name: center-control\n    namespace: induforge-system", "resources: [\"deployments\"]"} {
		if !strings.Contains(manifest, expected) {
			t.Fatalf("环境级 project reconciler RBAC 缺失 %q", expected)
		}
	}
	for _, image := range []string{"timescale/timescaledb:2.26.4-pg16", "redis:7.2-alpine", "emqx/emqx:5.6.1", "nats:2.12.8-alpine", "chrislusf/seaweedfs:3.85", "nginx:1.28-alpine"} {
		if !strings.Contains(manifest, "image: "+image) || !strings.Contains(manifest, "imagePullPolicy: Never") {
			t.Fatalf("fixed offline image missing: %s", image)
		}
	}
	if !strings.Contains(manifest, "-s3.config=/etc/seaweedfs/s3.json") || !strings.Contains(manifest, "object-access-key:") || strings.Contains(manifest, `\"name\":\"anonymous\"`) {
		t.Fatal("object storage must require generated S3 credentials")
	}
	if !strings.Contains(manifest, "EMQX_AUTHENTICATION__1__BACKEND") || !strings.Contains(manifest, "kind: Job") || !strings.Contains(manifest, "emqx-runtime-credential") {
		t.Fatal("message broker must require the generated runtime credential")
	}
	if foundationNATSMaxPayloadBytes < minimumDLQPayloadBytes {
		t.Fatalf("NATS MaxPayload=%d must cover the frozen DLQ payload=%d", foundationNATSMaxPayloadBytes, minimumDLQPayloadBytes)
	}
	if !strings.Contains(manifest, "nats-token:") || !strings.Contains(manifest, "nats.conf:") || !strings.Contains(manifest, fmt.Sprintf("max_payload: %d", foundationNATSMaxPayloadBytes)) {
		t.Fatal("foundation credentials must expose the NATS token and bounded MaxPayload config")
	}
	for _, document := range strings.Split(manifest, "\n---\n") {
		var value any
		if err := yaml.Unmarshal([]byte(document), &value); err != nil {
			t.Fatalf("invalid YAML document: %v\n%s", err, document)
		}
	}
}

func TestFoundationNATSConfigHashTriggersOnlyNATSConfigRollout(t *testing.T) {
	secret := "private-secret-value-0123456789"
	config := foundationNATSConfig(secret)
	hash := foundationConfigHash(config)
	if hash != foundationConfigHash(config) {
		t.Fatal("相同 NATS 配置必须产生稳定 Pod 模板摘要")
	}
	if hash == foundationConfigHash(config+"# changed\n") {
		t.Fatal("NATS 配置变更必须改变 Pod 模板摘要")
	}
	if hash == foundationConfigHash(foundationNATSConfig("other-private-secret-value-0123456789")) {
		t.Fatal("NATS 凭据变更必须改变 Pod 模板摘要")
	}

	plan := FoundationPlan{
		SchemaVersion: FoundationSchemaVersion, Generation: 1,
		EnvironmentID: "11111111-1111-4111-8111-111111111111", NodeID: "22222222-2222-4222-8222-222222222222",
		Assignments: map[string]string{
			"postgres": "33333333-3333-4333-8333-333333333333", "redis": "44444444-4444-4444-8444-444444444444",
			"emqx": "55555555-5555-4555-8555-855555555555", "nats": "66666666-6666-4666-8666-666666666666",
			"object": "77777777-7777-4777-8777-777777777777", "nginx": "88888888-8888-4888-8888-888888888888",
		},
	}
	manifest := plan.RenderManifest(secret)
	if strings.Count(manifest, "induforge.io/config-sha256: "+hash) != 1 {
		t.Fatal("NATS StatefulSet 必须且只能有一个当前配置摘要注解")
	}
	for _, document := range strings.Split(manifest, "\n---\n") {
		if !strings.Contains(document, "kind: StatefulSet\nmetadata:\n  name: nats") {
			continue
		}
		if strings.Contains(document, secret) {
			t.Fatal("NATS Pod 模板不得泄露 Secret 原文")
		}
		return
	}
	t.Fatal("NATS StatefulSet 缺失")
}

func TestFoundationPlanRejectsArbitraryOrIncompleteAssignments(t *testing.T) {
	plan := FoundationPlan{SchemaVersion: FoundationSchemaVersion, Generation: 1, EnvironmentID: "11111111-1111-4111-8111-111111111111", NodeID: "22222222-2222-4222-8222-222222222222", Assignments: map[string]string{"shell": "33333333-3333-4333-8333-333333333333"}}
	if err := plan.Validate(); err == nil {
		t.Fatal("arbitrary workload must be rejected")
	}
}

func TestFoundationMigrationUsesExplicitTargetClaim(t *testing.T) {
	plan := FoundationPlan{
		SchemaVersion: FoundationSchemaVersion, Generation: 2, Operation: "migrate",
		EnvironmentID: "11111111-1111-4111-8111-111111111111", NodeID: "22222222-2222-4222-8222-222222222222",
		Assignments:       map[string]string{"postgres": "33333333-3333-4333-8333-333333333333", "redis": "33333333-3333-4333-8333-333333333333", "emqx": "33333333-3333-4333-8333-333333333333", "nats": "33333333-3333-4333-8333-333333333333", "object": "33333333-3333-4333-8333-333333333333", "nginx": "33333333-3333-4333-8333-333333333333"},
		SourceAssignments: map[string]string{"postgres": "44444444-4444-4444-8444-444444444444"},
		Claims:            map[string]string{"postgres": "data-postgres-m12345678"},
	}
	if err := plan.Validate(); err != nil {
		t.Fatal(err)
	}
	manifest := plan.RenderManifest("private-secret-value-0123456789")
	if !strings.Contains(manifest, "claimName: data-postgres-m12345678") {
		t.Fatal("migrated stateful service must mount the explicit target PVC")
	}
	helper := renderFoundationMigration(foundationNamespace(plan.EnvironmentID), "postgres", plan.Generation, plan.SourceAssignments["postgres"], plan.Assignments["postgres"], "data-postgres-0", plan.Claims["postgres"])
	for _, document := range strings.Split(helper, "\n---\n") {
		var value any
		if err := yaml.Unmarshal([]byte(document), &value); err != nil {
			t.Fatalf("invalid migration YAML: %v\n%s", err, document)
		}
	}
}

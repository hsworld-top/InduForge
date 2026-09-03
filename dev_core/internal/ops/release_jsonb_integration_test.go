package ops

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// TestDeployableReleaseSurvivesPostgreSQLJSONBRoundTrip 覆盖生产实际存储边界。
// 默认单元测试环境没有 PostgreSQL 时跳过；发布验证通过显式 DSN 运行该用例。
func TestDeployableReleaseSurvivesPostgreSQLJSONBRoundTrip(t *testing.T) {
	dsn := os.Getenv("INDUFORGE_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("未配置 PostgreSQL 集成测试 DSN")
	}
	ctx := context.Background()
	connection, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close(ctx)
	if _, err = connection.Exec(ctx, `CREATE TEMP TABLE release_manifest_jsonb_test (manifest jsonb NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	manifest := validReleaseManifest(testProjectID)
	if _, err = connection.Exec(ctx, `INSERT INTO release_manifest_jsonb_test(manifest) VALUES($1::jsonb)`, manifest); err != nil {
		t.Fatal(err)
	}
	var stored []byte
	if err = connection.QueryRow(ctx, `SELECT manifest FROM release_manifest_jsonb_test`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(stored)) == strings.TrimSpace(manifest) {
		t.Fatal("测试未经过 PostgreSQL JSONB 表示转换")
	}
	if err = validateDeployableRelease(testProjectID, testVersionID, "releases/tenant/project/release.tar.zst", strings.Repeat("a", 64), strings.Repeat("e", 64), strings.Repeat("f", 64), "induforge-release-2026-01", stored); err != nil {
		t.Fatalf("PostgreSQL JSONB 往返后的正式 Release 被拒绝: %v", err)
	}
}

// TestStoredProductionReleaseValidates 只读验证一个真实版本记录，确保构建、JSONB
// 持久化和生产部署校验三段契约闭环。
func TestStoredProductionReleaseValidates(t *testing.T) {
	dsn, releaseID := os.Getenv("INDUFORGE_TEST_POSTGRES_DSN"), os.Getenv("INDUFORGE_TEST_RELEASE_ID")
	if dsn == "" || releaseID == "" {
		t.Skip("未配置 PostgreSQL DSN 或待验证 Release ID")
	}
	ctx := context.Background()
	connection, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close(ctx)
	var status, projectID, artifactKey, artifactHash, manifestHash, checksumsHash, signingKeyID string
	var manifest []byte
	err = connection.QueryRow(ctx, `SELECT status,project_id::text,COALESCE(artifact_key,''),COALESCE(artifact_hash,''),COALESCE(manifest_hash,''),COALESCE(checksums_hash,''),COALESCE(signing_key_id,''),manifest FROM application_versions WHERE id=$1 AND deleted_at IS NULL`, releaseID).Scan(&status, &projectID, &artifactKey, &artifactHash, &manifestHash, &checksumsHash, &signingKeyID, &manifest)
	if err != nil {
		t.Fatal(err)
	}
	if status != "ready" {
		t.Fatalf("Release 尚未构建成功: %s", status)
	}
	if err = validateDeployableReleaseForDeployment(projectID, releaseID, artifactKey, artifactHash, manifestHash, checksumsHash, signingKeyID, true, manifest); err != nil {
		t.Fatalf("真实 JSONB Release 不可部署: %v", err)
	}
}

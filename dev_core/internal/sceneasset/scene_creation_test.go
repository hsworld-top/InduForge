package sceneasset

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestSceneInputIgnoresClientManagedFields(t *testing.T) {
	var input SceneInput
	if err := json.Unmarshal([]byte(`{"sceneId":"client-id","entryPath":"displays/client.json","kind":"2d","name":"产线总览"}`), &input); err != nil {
		t.Fatalf("解析创建请求失败: %v", err)
	}
	if input.SceneID != "" || input.EntryPath != "" {
		t.Fatalf("客户端不得指定场景 ID 或入口: %+v", input)
	}
}

func TestValidateSceneName(t *testing.T) {
	if !errors.Is(validateSceneName(""), ErrInvalidSceneName) {
		t.Fatal("空场景名称应被拒绝")
	}
	if !errors.Is(validateSceneName(strings.Repeat("场", 201)), ErrInvalidSceneName) {
		t.Fatal("超过 200 个字符的场景名称应被拒绝")
	}
	if err := validateSceneName("产线总览"); err != nil {
		t.Fatalf("合法场景名称被拒绝: %v", err)
	}
}

func TestMapSceneNameUniqueConstraint(t *testing.T) {
	err := mapDatabaseError(&pgconn.PgError{Code: "23505", ConstraintName: "scene_documents_project_name_active_uidx"})
	if !errors.Is(err, ErrSceneNameConflict) {
		t.Fatalf("场景名称唯一约束应转换为业务冲突: %v", err)
	}
}

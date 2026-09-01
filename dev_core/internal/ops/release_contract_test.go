package ops

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestFormalReleaseAndBindingContractFixtures(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("无法定位契约测试目录")
	}
	contractRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "../../../contracts/runtime"))

	tests := []struct {
		name, schema, fixture string
		valid                 bool
	}{
		{"正式 Release Manifest", "release-manifest-v2.schema.json", "release-manifest-v2.valid.json", true},
		{"Release 禁止夹带 Secret", "release-manifest-v2.schema.json", "release-manifest-v2.invalid-secret.json", false},
		{"Release 摘要清单", "release-checksums-v1.schema.json", "release-checksums-v1.valid.json", true},
		{"摘要清单禁止签名自引用", "release-checksums-v1.schema.json", "release-checksums-v1.invalid-signature-entry.json", false},
		{"单节点 DeploymentBinding", "deployment-binding-v1.schema.json", "deployment-binding-v1.valid.json", true},
		{"Binding 禁止任意命令", "deployment-binding-v1.schema.json", "deployment-binding-v1.invalid-command.json", false},
		{"内部运行激活契约", "runtime-activation-v1.schema.json", "runtime-activation-v1.valid.json", true},
		{"激活契约禁止任意命令", "runtime-activation-v1.schema.json", "runtime-activation-v1.invalid-command.json", false},
		{"激活契约要求运行 Secret", "runtime-activation-v1.schema.json", "runtime-activation-v1.invalid-missing-secret.json", false},
		{"激活契约要求只读工件", "runtime-activation-v1.schema.json", "runtime-activation-v1.invalid-writable-mount.json", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			schema := compileRuntimeContract(t, contractRoot, test.schema)
			fixture := readContractJSON(t, filepath.Join(contractRoot, "fixtures", test.fixture))
			err := schema.Validate(fixture)
			if test.valid && err != nil {
				t.Fatalf("合法样例未通过 %s: %v", test.schema, err)
			}
			if !test.valid && err == nil {
				t.Fatalf("非法样例被 %s 接受", test.schema)
			}
		})
	}
}

func compileRuntimeContract(t *testing.T, contractRoot, schemaFile string) *jsonschema.Schema {
	t.Helper()
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	const contractBase = "https://induforge.dev/contracts/runtime/"
	for _, name := range []string{"common.schema.json", schemaFile} {
		if err := compiler.AddResource(contractBase+name, readContractJSON(t, filepath.Join(contractRoot, name))); err != nil {
			t.Fatalf("注册契约 %s: %v", name, err)
		}
	}
	schema, err := compiler.Compile(contractBase + schemaFile)
	if err != nil {
		t.Fatalf("编译契约 %s: %v", schemaFile, err)
	}
	return schema
}

func readContractJSON(t *testing.T, path string) any {
	t.Helper()
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取契约 %s: %v", path, err)
	}
	var document any
	if err := json.Unmarshal(payload, &document); err != nil {
		t.Fatalf("解析契约 %s: %v", path, err)
	}
	return document
}

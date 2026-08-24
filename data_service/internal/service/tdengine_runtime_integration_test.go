package service

import (
	"context"
	"os"
	"testing"
)

// TestTDengineRuntimeIntegration 使用显式环境变量连接临时真实实例，默认不占用普通单元测试环境。
func TestTDengineRuntimeIntegration(t *testing.T) {
	if os.Getenv("TDENGINE_INTEGRATION") != "1" {
		t.Skip("set TDENGINE_INTEGRATION=1 to run")
	}
	config := map[string]any{
		"protocol": "ws", "host": "127.0.0.1", "port": 16041,
		"username": "root", "password": "taosdata", "databaseName": "stage_a",
		"timezone": "Asia/Shanghai", "tlsSkipVerify": false,
	}
	ctx := context.Background()
	runtime, err := connectTDengineRuntime(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()

	for _, statement := range []string{"SELECT * FROM ordinary", "SHOW TABLES", "DESCRIBE meters", "EXPLAIN SELECT * FROM ordinary"} {
		if err := validateTDengineReadOnlySQL(statement); err != nil {
			t.Fatalf("readonly validation %q: %v", statement, err)
		}
		if _, _, err := runtime.query(ctx, statement); err != nil {
			t.Fatalf("execute %q: %v", statement, err)
		}
	}

	tables, err := listTDengineTables(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"meters": "supertable", "meter_1": "child_table", "ordinary": "table"}
	for _, table := range tables {
		if expected, exists := want[table.Name]; exists {
			if table.Kind != expected {
				t.Fatalf("table %s kind=%s, want %s", table.Name, table.Kind, expected)
			}
			delete(want, table.Name)
		}
	}
	if len(want) != 0 {
		t.Fatalf("missing table kinds: %#v", want)
	}
	structure, err := getTDengineTableStructure(ctx, config, "meters")
	if err != nil || len(structure.Columns) < 5 {
		t.Fatalf("describe stable: columns=%d err=%v", len(structure.Columns), err)
	}
	data, err := getTDengineTableData(ctx, config, "ordinary", 1, 20)
	if err != nil || data.Pagination.Total != 1 || len(data.Rows) != 1 {
		t.Fatalf("preview table: data=%#v err=%v", data, err)
	}

	bad := cloneMap(config)
	bad["password"] = "wrong-password"
	if runtime, err := connectTDengineRuntime(ctx, bad); err == nil {
		runtime.Close()
		t.Fatal("wrong credentials must fail")
	}
}

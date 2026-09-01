package contract

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestDevIDEEndpointsMatchOpenAPI(t *testing.T) {
	t.Parallel()

	frontend, err := devIDEEndpoints()
	if err != nil {
		t.Fatalf("扫描 dev_ide 接口失败: %v", err)
	}
	contract, err := openAPIEndpoints()
	if err != nil {
		t.Fatalf("读取 OpenAPI 接口失败: %v", err)
	}

	frontendKeys := endpointKeys(frontend)
	contract = endpointsForClient(contract, "dev_ide")
	contractKeys := endpointKeys(contract)
	if !reflect.DeepEqual(frontendKeys, contractKeys) {
		t.Fatalf("dev_ide 与 OpenAPI 接口不一致\ndev_ide 独有: %v\nOpenAPI 独有: %v", difference(frontendKeys, contractKeys), difference(contractKeys, frontendKeys))
	}
	if len(contract) != 94 {
		t.Fatalf("OpenAPI 操作数量错误: got %d, want 94", len(contract))
	}
}

func endpointsForClient(items []endpoint, client string) []endpoint {
	result := make([]endpoint, 0, len(items))
	for _, item := range items {
		if item.Client == client {
			result = append(result, item)
		}
	}
	return result
}

func TestEveryOperationHasRealHTTPTestEvidence(t *testing.T) {
	t.Parallel()

	contract, err := openAPIEndpoints()
	if err != nil {
		t.Fatalf("读取 OpenAPI 接口失败: %v", err)
	}
	tested, err := httpTestEndpoints()
	if err != nil {
		t.Fatalf("扫描 Go HTTP 测试失败: %v", err)
	}

	missing := make([]string, 0)
	for _, item := range contract {
		if !hasHTTPTestEvidence(item, tested) {
			missing = append(missing, fmt.Sprintf("%s (%s %s)", item.OperationID, item.Method, item.Path))
		}
	}
	if len(missing) > 0 {
		t.Fatalf("以下 operationId 缺少真实 HTTP 请求测试证据: %v", missing)
	}
}

func hasHTTPTestEvidence(operation endpoint, tested []endpoint) bool {
	for _, item := range tested {
		if item.Method != operation.Method {
			continue
		}
		operationSegments := strings.Split(strings.Trim(operation.Path, "/"), "/")
		testSegments := strings.Split(strings.Trim(item.Path, "/"), "/")
		if len(operationSegments) != len(testSegments) {
			continue
		}
		matched := true
		for index := range operationSegments {
			if operationSegments[index] != "{param}" && operationSegments[index] != testSegments[index] {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

func endpointKeys(items []endpoint) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, fmt.Sprintf("%s %s", item.Method, item.Path))
	}
	return result
}

func difference(left, right []string) []string {
	rightSet := make(map[string]struct{}, len(right))
	for _, item := range right {
		rightSet[item] = struct{}{}
	}

	result := make([]string, 0)
	for _, item := range left {
		if _, exists := rightSet[item]; !exists {
			result = append(result, item)
		}
	}
	return result
}

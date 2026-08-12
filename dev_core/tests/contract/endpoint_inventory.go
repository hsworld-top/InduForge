package contract

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
)

type endpoint struct {
	Method      string
	Path        string
	OperationID string
	Client      string
}

var (
	requestCallPattern   = regexp.MustCompile(`request\.(get|post|put|patch|delete)[^\r\n(]*\(\s*([[:punct:]])(/[^\r\n]+)`)
	interpolationPattern = regexp.MustCompile(`\$\{[^}]+\}`)
	pathParameterPattern = regexp.MustCompile(`\{[^}]+\}`)
)

func devIDEEndpoints() ([]endpoint, error) {
	sourceRoot, err := devIDESourceRoot()
	if err != nil {
		return nil, err
	}

	unique := make(map[string]endpoint)
	err = filepath.WalkDir(sourceRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		extension := strings.ToLower(filepath.Ext(path))
		if extension != ".ts" && extension != ".js" && extension != ".vue" {
			return nil
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, match := range requestCallPattern.FindAllStringSubmatch(string(content), -1) {
			if len(match) < 4 {
				continue
			}
			quote := match[2]
			rawPath := strings.Split(match[3], quote)[0]
			rawPath = strings.Split(rawPath, "?")[0]
			if strings.HasPrefix(rawPath, "/data/") {
				continue
			}
			normalizedPath := normalizePath(interpolationPattern.ReplaceAllString(rawPath, "{param}"))
			item := endpoint{Method: strings.ToUpper(match[1]), Path: normalizedPath}
			unique[item.Method+" "+item.Path] = item
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return sortedEndpoints(unique), nil
}

func openAPIEndpoints() ([]endpoint, error) {
	spec, err := platformapi.GetSwagger()
	if err != nil {
		return nil, fmt.Errorf("加载 OpenAPI: %w", err)
	}

	unique := make(map[string]endpoint)
	for path, pathItem := range spec.Paths.Map() {
		operations := map[string]string{
			"DELETE": operationID(pathItem.Delete),
			"GET":    operationID(pathItem.Get),
			"PATCH":  operationID(pathItem.Patch),
			"POST":   operationID(pathItem.Post),
			"PUT":    operationID(pathItem.Put),
		}
		for method, operationID := range operations {
			if operationID == "" {
				continue
			}
			normalizedPath := normalizePath(path)
			operation := operationByMethod(pathItem, method)
			item := endpoint{Method: method, Path: normalizedPath, OperationID: operationID, Client: operationClient(operation)}
			unique[item.Method+" "+item.Path] = item
		}
	}

	return sortedEndpoints(unique), nil
}

func operationByMethod(pathItem *openapi3.PathItem, method string) *openapi3.Operation {
	switch method {
	case "DELETE":
		return pathItem.Delete
	case "GET":
		return pathItem.Get
	case "PATCH":
		return pathItem.Patch
	case "POST":
		return pathItem.Post
	case "PUT":
		return pathItem.Put
	default:
		return nil
	}
}

func operationClient(operation *openapi3.Operation) string {
	if operation == nil {
		return "dev_ide"
	}
	value, exists := operation.Extensions["x-client"]
	if !exists {
		return "dev_ide"
	}
	client, ok := value.(string)
	if !ok || strings.TrimSpace(client) == "" {
		return "dev_ide"
	}
	return strings.TrimSpace(client)
}

func httpTestEndpoints() ([]endpoint, error) {
	testRoot, err := internalSourceRoot()
	if err != nil {
		return nil, err
	}

	unique := make(map[string]endpoint)
	fileSet := token.NewFileSet()
	err = filepath.WalkDir(testRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}

		parsed, parseErr := parser.ParseFile(fileSet, path, nil, 0)
		if parseErr != nil {
			return parseErr
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			for index := 0; index+1 < len(call.Args); index++ {
				method, ok := httpMethod(call.Args[index])
				if !ok {
					continue
				}
				requestPath, ok := testRequestPath(call.Args[index+1])
				if !ok || !strings.HasPrefix(requestPath, "/api/v1/") {
					continue
				}
				requestPath = strings.Split(requestPath, "?")[0]
				requestPath = strings.TrimPrefix(requestPath, "/api/v1")
				item := endpoint{Method: method, Path: normalizePath(requestPath)}
				unique[item.Method+" "+item.Path] = item
			}
			return true
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	return sortedEndpoints(unique), nil
}

func httpMethod(expression ast.Expr) (string, bool) {
	selector, ok := expression.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	packageName, ok := selector.X.(*ast.Ident)
	if !ok || packageName.Name != "http" {
		return "", false
	}
	methods := map[string]string{
		"MethodDelete": "DELETE",
		"MethodGet":    "GET",
		"MethodPatch":  "PATCH",
		"MethodPost":   "POST",
		"MethodPut":    "PUT",
	}
	method, ok := methods[selector.Sel.Name]
	return method, ok
}

func testRequestPath(expression ast.Expr) (string, bool) {
	switch value := expression.(type) {
	case *ast.BasicLit:
		if value.Kind != token.STRING {
			return "", false
		}
		path, err := strconv.Unquote(value.Value)
		return path, err == nil
	case *ast.BinaryExpr:
		if value.Op != token.ADD {
			return "", false
		}
		left, leftOK := testRequestPath(value.X)
		right, rightOK := testRequestPath(value.Y)
		if !leftOK || !rightOK {
			return "", false
		}
		return left + right, true
	case *ast.ParenExpr:
		return testRequestPath(value.X)
	case *ast.Ident, *ast.CallExpr, *ast.IndexExpr, *ast.SelectorExpr:
		return "{param}", true
	default:
		return "", false
	}
}

func operationID(operation *openapi3.Operation) string {
	if operation == nil {
		return ""
	}
	if operation.OperationID == "" {
		return ""
	}
	return strings.ToLower(operation.OperationID[:1]) + operation.OperationID[1:]
}

func normalizePath(path string) string {
	return pathParameterPattern.ReplaceAllString(path, "{param}")
}

func sortedEndpoints(items map[string]endpoint) []endpoint {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make([]endpoint, 0, len(keys))
	for _, key := range keys {
		result = append(result, items[key])
	}
	return result
}

func devIDESourceRoot() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("无法定位契约测试文件")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "dev_ide", "src"))
	if _, err := os.Stat(root); err != nil {
		return "", fmt.Errorf("定位 dev_ide 源码目录: %w", err)
	}
	return root, nil
}

func internalSourceRoot() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("无法定位契约测试文件")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "internal"))
	if _, err := os.Stat(root); err != nil {
		return "", fmt.Errorf("定位 Go 模块测试目录: %w", err)
	}
	return root, nil
}

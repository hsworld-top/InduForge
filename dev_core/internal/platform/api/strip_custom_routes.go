//go:build ignore

// 场景平台使用独立 Handler 挂载路由；本工具保留 OpenAPI 生成类型和内嵌规范，移除通用控制面的重复路由注册。
package main

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
)

var customRoutePrefixes = []string{
	"/scene-provider",
	"/scene-editor-sessions/",
	"/scene-asset-editor-sessions/",
	"/projects/{projectId}/scenes",
	"/projects/{projectId}/scene-assets",
	"/ops",
}

func main() {
	if len(os.Args) < 2 {
		panic("usage: go run strip_custom_routes.go -- generated.go")
	}
	filename := os.Args[len(os.Args)-1]
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, filename, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != "HandlerWithOptions" || function.Body == nil {
			continue
		}
		statements := function.Body.List[:0]
		for _, statement := range function.Body.List {
			if !registersCustomRoute(statement) {
				statements = append(statements, statement)
			}
		}
		function.Body.List = statements
	}
	output, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer output.Close()
	if err := format.Node(output, set, file); err != nil {
		panic(fmt.Errorf("格式化生成文件: %w", err))
	}
}

func registersCustomRoute(statement ast.Stmt) bool {
	matched := false
	ast.Inspect(statement, func(node ast.Node) bool {
		literal, ok := node.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		value, err := strconv.Unquote(literal.Value)
		if err != nil {
			return true
		}
		for _, prefix := range customRoutePrefixes {
			if strings.HasPrefix(value, prefix) {
				matched = true
				return false
			}
		}
		return true
	})
	return matched
}

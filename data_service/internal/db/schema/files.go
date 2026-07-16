package schema

import "embed"

// Files 内嵌最终数据库结构基线，运行时无需依赖源码目录。
//
//go:embed schema.sql
var Files embed.FS

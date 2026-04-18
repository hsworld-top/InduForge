package migrations

import "embed"

// Files 内嵌全部 migration SQL，避免运行时依赖源码目录结构。
//
//go:embed *.sql
var Files embed.FS

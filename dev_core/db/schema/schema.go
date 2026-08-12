package schema

import _ "embed"

// CoreSQL 是控制面空库初始化的唯一结构基线。
//
//go:embed core-schema.sql
var CoreSQL string

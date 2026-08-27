package service

import "strings"

// canonicalSQLColumnType 将不同数据库驱动返回的列类型统一为平台数据点类型。
// SQL 表达式无法提供精确类型时返回 string，避免用某一行的 null 样本误判字段类型。
func canonicalSQLColumnType(databaseType string) string {
	typeName := strings.ToLower(strings.TrimSpace(databaseType))
	if typeName == "" {
		return "string"
	}
	if strings.HasPrefix(typeName, "_") || strings.Contains(typeName, "array") || strings.HasSuffix(typeName, "[]") {
		return "array"
	}
	if strings.Contains(typeName, "unsigned") {
		switch {
		case strings.Contains(typeName, "tinyint"):
			return "uint8"
		case strings.Contains(typeName, "smallint"):
			return "uint16"
		case strings.Contains(typeName, "bigint"):
			return "uint64"
		default:
			return "uint32"
		}
	}
	switch typeName {
	case "bool", "boolean", "bit":
		return "bool"
	case "int1", "tinyint":
		return "int8"
	case "int2", "smallint", "smallserial":
		return "int16"
	case "int", "int4", "integer", "mediumint", "serial":
		return "int32"
	case "int8", "bigint", "bigserial":
		return "int64"
	case "float4", "real":
		return "float32"
	case "float", "float8", "double", "double precision":
		return "float64"
	case "decimal", "numeric", "money", "smallmoney":
		return "decimal"
	case "bytea", "blob", "tinyblob", "mediumblob", "longblob", "binary", "varbinary", "image":
		return "bytes"
	case "json", "jsonb", "object", "map", "struct":
		return "object"
	case "date", "time", "timetz", "timestamp", "timestamptz", "datetime", "datetime2", "smalldatetime":
		return "datetime"
	default:
		return "string"
	}
}

func canonicalSQLColumnTypes(columns, databaseTypes []string) map[string]string {
	result := make(map[string]string, len(columns))
	for index, column := range columns {
		databaseType := ""
		if index < len(databaseTypes) {
			databaseType = databaseTypes[index]
		}
		result[column] = canonicalSQLColumnType(databaseType)
	}
	return result
}

// sqlWorkbenchStatementReturnsRows 判断工作台 SQL 是否应走结果集读取路径。
// 除只读查询外，DML 的 RETURNING/OUTPUT 也会返回结果集，必须允许用户预览。
func sqlWorkbenchStatementReturnsRows(sqlText string) bool {
	tokens, _, err := tdengineSQLTokens(sqlText)
	if err != nil || len(tokens) == 0 {
		return false
	}
	switch tokens[0] {
	case "select", "with", "show", "describe", "desc", "explain", "values", "table", "call", "exec", "execute":
		return true
	case "insert", "update", "delete", "merge":
		for _, token := range tokens[1:] {
			if token == "returning" || token == "output" {
				return true
			}
		}
	}
	return false
}

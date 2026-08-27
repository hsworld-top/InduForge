package service

import "strings"

var canonicalDataPointTypes = map[string]struct{}{
	"bool": {}, "int8": {}, "uint8": {}, "int16": {}, "uint16": {}, "int32": {}, "uint32": {},
	"int64": {}, "uint64": {}, "float32": {}, "float64": {}, "decimal": {}, "string": {}, "bytes": {},
	"datetime": {}, "object": {}, "array": {},
}

// canonicalDataPointType 只接受最终公开契约中的规范类型。
func canonicalDataPointType(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	_, ok := canonicalDataPointTypes[value]
	return value, ok
}

// externalDataPointType 仅用于导入、协议样本等外部边界，将常见别名收敛为规范类型。
func externalDataPointType(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "boolean":
		value = "bool"
	case "number", "float", "double":
		value = "float64"
	case "int", "integer":
		value = "int32"
	case "long":
		value = "int64"
	case "json":
		value = "object"
	case "byte[]":
		value = "bytes"
	case "date", "timestamp":
		value = "datetime"
	}
	_, ok := canonicalDataPointTypes[value]
	return value, ok
}

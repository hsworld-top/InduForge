package service

import (
	"encoding/base64"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// normalizeSQLValue 将驱动内部值转换为稳定、可 JSON 序列化且不丢精度的工作台值。
func normalizeSQLValue(value any, databaseType string) any {
	switch typed := value.(type) {
	case pgtype.Numeric:
		encoded, err := typed.Value()
		if err == nil {
			return encoded
		}
		return nil
	case pgtype.UUID:
		if !typed.Valid {
			return nil
		}
		return typed.String()
	case [16]byte:
		if isSQLUUIDType(databaseType) {
			return uuid.UUID(typed).String()
		}
		return typed
	case pgtype.Time:
		encoded, err := typed.Value()
		if err == nil {
			if text, ok := encoded.(string); ok {
				return strings.TrimSuffix(text, ".000000")
			}
			return encoded
		}
		return nil
	case []byte:
		if isSQLUUIDType(databaseType) && len(typed) == 16 {
			parsed, err := uuid.FromBytes(typed)
			if err == nil {
				return parsed.String()
			}
		}
		if canonicalSQLColumnType(databaseType) == "bytes" {
			return base64.StdEncoding.EncodeToString(typed)
		}
		return string(typed)
	default:
		return value
	}
}

func isSQLUUIDType(databaseType string) bool {
	typeName := strings.TrimSpace(databaseType)
	return strings.EqualFold(typeName, "uuid") || strings.EqualFold(typeName, "uniqueidentifier")
}

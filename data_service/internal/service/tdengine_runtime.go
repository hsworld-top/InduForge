package service

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/taosdata/driver-go/v3/taosWS"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

// tdengineRuntime 与关系数据库运行时分离，避免把 TDengine 的超级表/标签语义伪装成普通关系表。
type tdengineRuntime struct{ db *sql.DB }

func connectTDengineRuntime(ctx context.Context, config map[string]any) (*tdengineRuntime, error) {
	dsn, err := buildTDengineDSN(config)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("taosWS", dsn)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "TDengine 连接串格式无效", err)
	}
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(2 * time.Minute)
	pingCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "连接 TDengine 失败", err)
	}
	return &tdengineRuntime{db: db}, nil
}

func buildTDengineDSN(config map[string]any) (string, error) {
	protocol := strings.ToLower(strings.TrimSpace(toString(config["protocol"])))
	if protocol != "ws" && protocol != "wss" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "TDengine protocol 仅支持 ws/wss")
	}
	host := strings.TrimSpace(toString(config["host"]))
	username := strings.TrimSpace(toString(config["username"]))
	password := toString(config["password"])
	database := strings.TrimSpace(toString(config["databaseName"]))
	port, ok := positiveInteger(config["port"])
	if host == "" || username == "" || password == "" || database == "" || !ok || port > 65535 {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "TDengine 结构化连接配置不完整")
	}
	query := url.Values{}
	if timezone := strings.TrimSpace(toString(config["timezone"])); timezone != "" {
		query.Set("timezone", timezone)
	}
	if skip, _ := config["tlsSkipVerify"].(bool); skip {
		query.Set("skipVerify", "true")
	}
	dsn := fmt.Sprintf("%s:%s@%s(%s:%d)/%s", url.QueryEscape(username), url.QueryEscape(password), protocol, host, port, url.PathEscape(database))
	if encoded := query.Encode(); encoded != "" {
		dsn += "?" + encoded
	}
	return dsn, nil
}

func (r *tdengineRuntime) Close() {
	if r != nil && r.db != nil {
		_ = r.db.Close()
	}
}

func (r *tdengineRuntime) query(ctx context.Context, sqlText string, args ...any) ([]string, [][]any, error) {
	columns, _, rows, _, err := r.queryBounded(ctx, sqlText, developmentSQLMaxRows, developmentSQLMaxBytes, args...)
	return columns, rows, err
}

func (r *tdengineRuntime) queryBounded(ctx context.Context, sqlText string, maxRows, maxBytes int, args ...any) ([]string, map[string]string, [][]any, string, error) {
	queryCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	boundSQL, err := bindTDengineQuery(sqlText, args)
	if err != nil {
		return nil, nil, nil, "", err
	}
	rows, err := r.db.QueryContext(queryCtx, boundSQL)
	if err != nil {
		return nil, nil, nil, "", apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, sqlExecutionErrorMessage("执行 TDengine SQL 失败", err), err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, nil, "", err
	}
	columnTypeInfo, err := rows.ColumnTypes()
	if err != nil {
		return nil, nil, nil, "", err
	}
	databaseTypes := make([]string, 0, len(columnTypeInfo))
	for _, columnType := range columnTypeInfo {
		databaseTypes = append(databaseTypes, columnType.DatabaseTypeName())
	}
	result := make([][]any, 0)
	resultBytes := 0
	truncatedBy := ""
	for rows.Next() {
		values := make([]any, len(columns))
		scanTargets := make([]any, len(columns))
		for index := range values {
			scanTargets[index] = &values[index]
		}
		if err := rows.Scan(scanTargets...); err != nil {
			return nil, nil, nil, "", err
		}
		for index, value := range values {
			values[index] = normalizeQueryValue(value)
		}
		accepted, reason, nextBytes := admitSQLResultRow(len(result), resultBytes, maxRows, maxBytes, values)
		if !accepted {
			truncatedBy = reason
			break
		}
		result = append(result, values)
		resultBytes = nextBytes
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, "", err
	}
	return columns, canonicalSQLColumnTypes(columns, databaseTypes), result, truncatedBy, nil
}

func (r *tdengineRuntime) execAffected(ctx context.Context, sqlText string, args ...any) (int64, error) {
	boundSQL, err := bindTDengineQuery(sqlText, args)
	if err != nil {
		return 0, err
	}
	result, err := r.db.ExecContext(ctx, boundSQL)
	if err != nil {
		return 0, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, sqlExecutionErrorMessage("执行 TDengine SQL 失败", err), err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, nil
	}
	return affected, nil
}

// bindTDengineQuery 将工作台值参数编码为 TDengine SQL 字面量。
// taosWS 的查询接口不支持 database/sql 的 QueryContext 参数绑定，因此这里只替换
// 注释、字符串和引用标识符之外的 ?，并对字符串做单引号转义。
func bindTDengineQuery(sqlText string, args []any) (string, error) {
	if err := validateSQLParameterCount("tdengine", sqlText, args); err != nil {
		return "", err
	}
	if len(args) == 0 {
		return sqlText, nil
	}

	var result strings.Builder
	result.Grow(len(sqlText) + len(args)*8)
	argIndex := 0
	for index := 0; index < len(sqlText); {
		ch := sqlText[index]
		if ch == '-' && index+1 < len(sqlText) && sqlText[index+1] == '-' {
			end := index + 2
			for end < len(sqlText) && sqlText[end] != '\n' {
				end++
			}
			result.WriteString(sqlText[index:end])
			index = end
			continue
		}
		if ch == '/' && index+1 < len(sqlText) && sqlText[index+1] == '*' {
			end := index + 2
			for end+1 < len(sqlText) && !(sqlText[end] == '*' && sqlText[end+1] == '/') {
				end++
			}
			if end+1 < len(sqlText) {
				end += 2
			} else {
				end = len(sqlText)
			}
			result.WriteString(sqlText[index:end])
			index = end
			continue
		}
		if ch == '\'' || ch == '"' || ch == '`' {
			quote := ch
			end := index + 1
			for end < len(sqlText) {
				if sqlText[end] == quote {
					if end+1 < len(sqlText) && sqlText[end+1] == quote {
						end += 2
						continue
					}
					end++
					break
				}
				end++
			}
			result.WriteString(sqlText[index:end])
			index = end
			continue
		}
		if ch == '?' {
			result.WriteString(tdengineSQLLiteral(args[argIndex]))
			argIndex++
			index++
			continue
		}
		result.WriteByte(ch)
		index++
	}
	return result.String(), nil
}

func tdengineSQLLiteral(value any) string {
	switch typed := value.(type) {
	case nil:
		return "NULL"
	case bool:
		if typed {
			return "TRUE"
		}
		return "FALSE"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return fmt.Sprint(typed)
	case time.Time:
		return "'" + typed.Format(time.RFC3339Nano) + "'"
	case []byte:
		return "'" + hex.EncodeToString(typed) + "'"
	default:
		return "'" + strings.ReplaceAll(fmt.Sprint(typed), "'", "''") + "'"
	}
}

func validateTDengineReadOnlySQL(sqlText string) error {
	tokens, statementCount, err := tdengineSQLTokens(sqlText)
	if err != nil || statementCount != 1 || len(tokens) == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "TDengine 工作台只允许单条只读 SQL")
	}
	allowedFirst := map[string]bool{"select": true, "with": true, "show": true, "describe": true, "desc": true, "explain": true}
	if !allowedFirst[tokens[0]] {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "TDengine 工作台只允许单条只读 SQL")
	}
	blocked := map[string]bool{"insert": true, "update": true, "delete": true, "create": true, "alter": true, "drop": true, "grant": true, "revoke": true, "use": true, "flush": true, "compact": true, "reset": true}
	for _, token := range tokens {
		if blocked[token] {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "TDengine 工作台只允许单条只读 SQL")
		}
	}
	return nil
}

// validateTDengineSingleSQL 保持工作台与数据点回放只执行一条语句；
// DML/DDL 是否允许由连接账号权限和统一数据库边界校验决定。
func validateTDengineSingleSQL(sqlText string) error {
	_, statementCount, err := tdengineSQLTokens(sqlText)
	if err != nil || statementCount != 1 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "TDengine 工作台只允许单条 SQL")
	}
	return ensureSQLWorkbenchDatabaseBoundary(sqlText)
}

// tdengineSQLTokens 只提取注释、字符串和引用标识符之外的关键字，并统计真实语句数。
func tdengineSQLTokens(sqlText string) ([]string, int, error) {
	tokens := make([]string, 0, 16)
	statements := 0
	hasContent := false
	for index := 0; index < len(sqlText); {
		ch := sqlText[index]
		if ch == '-' && index+1 < len(sqlText) && sqlText[index+1] == '-' {
			index += 2
			for index < len(sqlText) && sqlText[index] != '\n' {
				index++
			}
			continue
		}
		if ch == '/' && index+1 < len(sqlText) && sqlText[index+1] == '*' {
			index += 2
			closed := false
			for index+1 < len(sqlText) {
				if sqlText[index] == '*' && sqlText[index+1] == '/' {
					index += 2
					closed = true
					break
				}
				index++
			}
			if !closed {
				return nil, 0, fmt.Errorf("SQL 注释未闭合")
			}
			continue
		}
		if ch == '\'' || ch == '"' || ch == '`' {
			quote := ch
			index++
			closed := false
			for index < len(sqlText) {
				if sqlText[index] == quote {
					if index+1 < len(sqlText) && sqlText[index+1] == quote {
						index += 2
						continue
					}
					index++
					closed = true
					break
				}
				index++
			}
			if !closed {
				return nil, 0, fmt.Errorf("SQL 引号未闭合")
			}
			hasContent = true
			continue
		}
		if ch == ';' {
			if hasContent {
				statements++
				hasContent = false
			}
			index++
			continue
		}
		if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || ch == '_' {
			start := index
			index++
			for index < len(sqlText) {
				current := sqlText[index]
				if !((current >= 'A' && current <= 'Z') || (current >= 'a' && current <= 'z') || (current >= '0' && current <= '9') || current == '_') {
					break
				}
				index++
			}
			tokens = append(tokens, strings.ToLower(sqlText[start:index]))
			hasContent = true
			continue
		}
		if ch > ' ' {
			hasContent = true
		}
		index++
	}
	if hasContent {
		statements++
	}
	return tokens, statements, nil
}

func tdengineTableName(value string) (string, error) {
	if !regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`).MatchString(value) {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "TDengine 表名格式无效")
	}
	return value, nil
}

func tdengineStringCell(values []any, names []string, wanted string) string {
	for index, name := range names {
		if strings.EqualFold(name, wanted) && index < len(values) {
			if values[index] == nil {
				return ""
			}
			return fmt.Sprint(values[index])
		}
	}
	return ""
}

func listTDengineTables(ctx context.Context, config map[string]any) ([]RelationalTable, error) {
	runtime, err := connectTDengineRuntime(ctx, config)
	if err != nil {
		return nil, err
	}
	defer runtime.Close()
	result := make([]RelationalTable, 0)
	seen := map[string]struct{}{}
	database := strings.ReplaceAll(strings.TrimSpace(toString(config["databaseName"])), "'", "''")
	for _, query := range []struct{ SQL, Kind string }{
		{fmt.Sprintf("SELECT stable_name AS name, '' AS stable_name FROM information_schema.ins_stables WHERE db_name = '%s'", database), "supertable"},
		{fmt.Sprintf("SELECT table_name AS name, stable_name FROM information_schema.ins_tables WHERE db_name = '%s'", database), "table"},
	} {
		columns, rows, queryErr := runtime.query(ctx, query.SQL)
		if queryErr != nil {
			return nil, queryErr
		}
		for _, values := range rows {
			name := tdengineStringCell(values, columns, "name")
			if name == "" && len(values) > 0 {
				name = fmt.Sprint(values[0])
			}
			if name == "" {
				continue
			}
			key := strings.ToLower(name)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			kind := query.Kind
			// SHOW TABLES 会返回所属超级表；有 stable_name 的记录是子表，不应与普通表混在一起。
			if query.Kind == "table" && strings.TrimSpace(tdengineStringCell(values, columns, "stable_name")) != "" {
				kind = "child_table"
			}
			result = append(result, RelationalTable{Schema: "", Name: name, Type: kind, Kind: kind})
		}
	}
	return result, nil
}

func getTDengineTableStructure(ctx context.Context, config map[string]any, tableName string) (*RelationalTableStructure, error) {
	name, err := tdengineTableName(tableName)
	if err != nil {
		return nil, err
	}
	runtime, err := connectTDengineRuntime(ctx, config)
	if err != nil {
		return nil, err
	}
	defer runtime.Close()
	columns, rows, err := runtime.query(ctx, "DESCRIBE "+name)
	if err != nil {
		return nil, err
	}
	structure := &RelationalTableStructure{Columns: make([]RelationalTableColumn, 0, len(rows)), Indexes: []RelationalIndex{}, ForeignKeys: []RelationalForeignKey{}}
	for _, values := range rows {
		field := tdengineStringCell(values, columns, "Field")
		if field == "" && len(values) > 0 {
			field = fmt.Sprint(values[0])
		}
		typeName := tdengineStringCell(values, columns, "Type")
		if typeName == "" && len(values) > 1 {
			typeName = fmt.Sprint(values[1])
		}
		note := tdengineStringCell(values, columns, "Note")
		if field == "" {
			continue
		}
		column := RelationalTableColumn{Name: field, Type: typeName, Nullable: true}
		if strings.EqualFold(note, "TAG") {
			column.Comment = stringPointer("TAG")
		}
		structure.Columns = append(structure.Columns, column)
	}
	return structure, nil
}

func getTDengineTableData(ctx context.Context, config map[string]any, tableName string, page, limit int) (*RelationalTableData, error) {
	name, err := tdengineTableName(tableName)
	if err != nil {
		return nil, err
	}
	runtime, err := connectTDengineRuntime(ctx, config)
	if err != nil {
		return nil, err
	}
	defer runtime.Close()
	_, countRows, err := runtime.query(ctx, fmt.Sprintf("SELECT COUNT(*) AS total FROM %s", name))
	if err != nil {
		return nil, err
	}
	total := 0
	if len(countRows) > 0 && len(countRows[0]) > 0 {
		total, err = strconv.Atoi(fmt.Sprint(countRows[0][0]))
		if err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析 TDengine 数据总数失败", err)
		}
	}
	columns, rows, err := runtime.query(ctx, fmt.Sprintf("SELECT * FROM %s LIMIT %d OFFSET %d", name, limit, (page-1)*limit))
	if err != nil {
		return nil, err
	}
	totalPages := 0
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}
	return &RelationalTableData{Columns: columns, Rows: rows, Pagination: RelationalPagination{Page: page, Limit: limit, Total: total, TotalPages: totalPages}}, nil
}

package service

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

var tableDesignIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func normalizeCreateTableInput(connectionType string, input CreateRelationalTableInput) (CreateRelationalTableInput, error) {
	tableName, err := normalizeTableDesignIdentifier(input.Name, "表名")
	if err != nil {
		return CreateRelationalTableInput{}, err
	}
	kind := strings.TrimSpace(strings.ToLower(input.Kind))
	if kind == "" {
		kind = "table"
	}
	if kind != "table" && kind != "hypertable" {
		return CreateRelationalTableInput{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "表类型仅支持普通表或时序超表")
	}
	if kind == "hypertable" && connectionType != "builtin.timeseries" {
		return CreateRelationalTableInput{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "仅 IF 时序库支持时序超表")
	}
	if kind == "hypertable" && input.Timeseries == nil {
		return CreateRelationalTableInput{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "时序超表必须配置 TimescaleDB 参数")
	}
	if kind == "table" && input.Timeseries != nil {
		return CreateRelationalTableInput{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "普通表不能配置 TimescaleDB 参数")
	}
	if len(input.Columns) == 0 {
		return CreateRelationalTableInput{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "至少需要一个字段")
	}
	columns := make([]CreateRelationalTableColumnInput, 0, len(input.Columns))
	columnNames := map[string]struct{}{}
	for _, column := range input.Columns {
		normalizedColumn, err := normalizeTableColumnInput(column)
		if err != nil {
			return CreateRelationalTableInput{}, err
		}
		if _, exists := columnNames[normalizedColumn.Name]; exists {
			return CreateRelationalTableInput{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "字段名重复: "+normalizedColumn.Name)
		}
		columnNames[normalizedColumn.Name] = struct{}{}
		columns = append(columns, normalizedColumn)
	}
	var timeseries *CreateRelationalTimeseriesInput
	if kind == "hypertable" && input.Timeseries != nil {
		dimensions := make([]CreateRelationalTableColumnInput, 0, len(input.Timeseries.DimensionColumns))
		dimensionNames := map[string]struct{}{}
		for _, dimension := range input.Timeseries.DimensionColumns {
			normalizedDimension, err := normalizeTableColumnInput(dimension)
			if err != nil {
				return CreateRelationalTableInput{}, err
			}
			if _, exists := dimensionNames[normalizedDimension.Name]; exists {
				return CreateRelationalTableInput{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "维度字段名重复: "+normalizedDimension.Name)
			}
			if _, exists := columnNames[normalizedDimension.Name]; exists {
				return CreateRelationalTableInput{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "维度字段不能与数据字段重名: "+normalizedDimension.Name)
			}
			dimensionNames[normalizedDimension.Name] = struct{}{}
			dimensions = append(dimensions, normalizedDimension)
			columnNames[normalizedDimension.Name] = struct{}{}
			columns = append(columns, normalizedDimension)
		}
		timeColumn := strings.TrimSpace(input.Timeseries.TimeColumn)
		if kind == "hypertable" {
			if timeColumn == "" {
				return CreateRelationalTableInput{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "时序超表必须选择时间分区字段")
			}
			timeColumnExists := false
			for _, column := range columns {
				if column.Name == timeColumn {
					timeColumnExists = true
					if column.Type != "timestamp" && column.Type != "timestamptz" && column.Type != "datetime" {
						return CreateRelationalTableInput{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "时间分区字段必须是 TIMESTAMP 或 TIMESTAMPTZ")
					}
					break
				}
			}
			if !timeColumnExists {
				return CreateRelationalTableInput{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "时间分区字段不存在: "+timeColumn)
			}
		}
		timeseries = &CreateRelationalTimeseriesInput{
			TimeColumn:       timeColumn,
			DimensionColumns: dimensions,
			ChunkInterval:    normalizeTimeseriesInterval(input.Timeseries.ChunkInterval, "1 day"),
			RetentionDays:    normalizeRetentionDays(input.Timeseries.RetentionDays),
		}
	}
	indexes := make([]CreateRelationalTableIndexInput, 0, len(input.Indexes))
	indexNames := map[string]struct{}{}
	for _, index := range input.Indexes {
		normalizedIndex, err := normalizeTableIndexInput(index, columnNames)
		if err != nil {
			return CreateRelationalTableInput{}, err
		}
		if _, exists := indexNames[normalizedIndex.Name]; exists {
			return CreateRelationalTableInput{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "索引名重复: "+normalizedIndex.Name)
		}
		indexNames[normalizedIndex.Name] = struct{}{}
		indexes = append(indexes, normalizedIndex)
	}
	return CreateRelationalTableInput{
		Name:       tableName,
		Kind:       kind,
		Columns:    columns,
		Indexes:    indexes,
		Timeseries: timeseries,
	}, nil
}

func normalizeUpdateTableInput(input UpdateRelationalTableInput) (UpdateRelationalTableInput, error) {
	if len(input.Columns) == 0 {
		return UpdateRelationalTableInput{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "至少需要一个字段")
	}
	columns := make([]CreateRelationalTableColumnInput, 0, len(input.Columns))
	columnNames := map[string]struct{}{}
	for _, column := range input.Columns {
		normalizedColumn, err := normalizeTableColumnInput(column)
		if err != nil {
			return UpdateRelationalTableInput{}, err
		}
		if _, exists := columnNames[normalizedColumn.Name]; exists {
			return UpdateRelationalTableInput{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "字段名重复: "+normalizedColumn.Name)
		}
		columnNames[normalizedColumn.Name] = struct{}{}
		columns = append(columns, normalizedColumn)
	}
	indexes := make([]CreateRelationalTableIndexInput, 0, len(input.Indexes))
	indexNames := map[string]struct{}{}
	for _, index := range input.Indexes {
		normalizedIndex, err := normalizeTableIndexInput(index, columnNames)
		if err != nil {
			return UpdateRelationalTableInput{}, err
		}
		if _, exists := indexNames[normalizedIndex.Name]; exists {
			return UpdateRelationalTableInput{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "索引名重复: "+normalizedIndex.Name)
		}
		indexNames[normalizedIndex.Name] = struct{}{}
		indexes = append(indexes, normalizedIndex)
	}
	return UpdateRelationalTableInput{Columns: columns, Indexes: indexes}, nil
}

func normalizeTimeseriesInterval(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func normalizeRetentionDays(value *int) *int {
	if value == nil || *value <= 0 {
		return nil
	}
	next := *value
	return &next
}

func normalizeTableColumnInput(input CreateRelationalTableColumnInput) (CreateRelationalTableColumnInput, error) {
	name, err := normalizeTableDesignIdentifier(input.Name, "字段名")
	if err != nil {
		return CreateRelationalTableColumnInput{}, err
	}
	columnType := strings.TrimSpace(strings.ToLower(input.Type))
	if columnType == "" {
		return CreateRelationalTableColumnInput{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "字段类型不能为空")
	}
	return CreateRelationalTableColumnInput{
		Name:          name,
		Type:          columnType,
		Length:        normalizePositivePointer(input.Length),
		Precision:     normalizePositivePointer(input.Precision),
		Scale:         normalizeNonNegativePointer(input.Scale),
		Nullable:      input.Nullable,
		Primary:       input.Primary,
		AutoIncrement: input.AutoIncrement,
		DefaultValue:  strings.TrimSpace(input.DefaultValue),
		Comment:       strings.TrimSpace(input.Comment),
	}, nil
}

func normalizeTableIndexInput(input CreateRelationalTableIndexInput, columnNames map[string]struct{}) (CreateRelationalTableIndexInput, error) {
	name, err := normalizeTableDesignIdentifier(input.Name, "索引名")
	if err != nil {
		return CreateRelationalTableIndexInput{}, err
	}
	indexType := strings.TrimSpace(strings.ToLower(input.Type))
	if indexType == "" {
		indexType = "index"
	}
	if indexType != "index" && indexType != "unique" {
		return CreateRelationalTableIndexInput{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "索引类型仅支持普通索引或唯一索引")
	}
	columns := make([]string, 0, len(input.Columns))
	for _, column := range input.Columns {
		columnName, err := normalizeTableDesignIdentifier(column, "索引字段")
		if err != nil {
			return CreateRelationalTableIndexInput{}, err
		}
		if _, exists := columnNames[columnName]; !exists {
			return CreateRelationalTableIndexInput{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "索引字段不存在: "+columnName)
		}
		columns = append(columns, columnName)
	}
	if len(columns) == 0 {
		return CreateRelationalTableIndexInput{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "索引至少需要一个字段")
	}
	return CreateRelationalTableIndexInput{Name: name, Type: indexType, Columns: columns}, nil
}

func normalizeTableDesignIdentifier(value, label string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, label+"不能为空")
	}
	if !tableDesignIdentifierPattern.MatchString(value) {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, label+"只能包含字母、数字、下划线，且不能以数字开头")
	}
	if len(value) > 63 {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, label+"长度不能超过 63 个字符")
	}
	return value, nil
}

func normalizePositivePointer(value *int) *int {
	if value == nil || *value <= 0 {
		return nil
	}
	next := *value
	return &next
}

func normalizeNonNegativePointer(value *int) *int {
	if value == nil || *value < 0 {
		return nil
	}
	next := *value
	return &next
}

func buildCreateTableDDL(dbType, schema string, input CreateRelationalTableInput) (string, error) {
	switch dbType {
	case "mysql":
		return buildMySQLCreateTableDDL(schema, input)
	case "sqlserver":
		return buildSQLServerCreateTableDDL(schema, input)
	default:
		return buildPostgresCreateTableDDL(schema, input)
	}
}

func buildPostgresCreateTableDDL(schema string, input CreateRelationalTableInput) (string, error) {
	definitions := make([]string, 0, len(input.Columns)+1)
	primaryColumns := make([]string, 0)
	for _, column := range input.Columns {
		columnType, err := postgresColumnType(column)
		if err != nil {
			return "", err
		}
		definition := fmt.Sprintf("%s %s", pgx.Identifier{column.Name}.Sanitize(), columnType)
		if column.Primary || !column.Nullable {
			definition += " NOT NULL"
		}
		if column.DefaultValue != "" && !column.AutoIncrement {
			definition += " DEFAULT " + column.DefaultValue
		}
		definitions = append(definitions, definition)
		if column.Primary {
			primaryColumns = append(primaryColumns, pgx.Identifier{column.Name}.Sanitize())
		}
	}
	if len(primaryColumns) > 0 {
		definitions = append(definitions, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(primaryColumns, ", ")))
	}
	tableName := pgx.Identifier{schema, input.Name}.Sanitize()
	statements := []string{fmt.Sprintf("CREATE TABLE %s (\n  %s\n)", tableName, strings.Join(definitions, ",\n  "))}
	for _, index := range input.Indexes {
		columns := quotePostgresColumns(index.Columns)
		prefix := "CREATE INDEX"
		if index.Type == "unique" {
			prefix = "CREATE UNIQUE INDEX"
		}
		statements = append(statements, fmt.Sprintf("%s %s ON %s (%s)", prefix, pgx.Identifier{index.Name}.Sanitize(), tableName, strings.Join(columns, ", ")))
	}
	return strings.Join(statements, ";\n") + ";", nil
}

func buildPostgresCreateHypertableStatements(schema string, input CreateRelationalTableInput) ([]string, error) {
	if input.Timeseries == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "时序超表缺少 TimescaleDB 配置")
	}
	createTableDDL, err := buildPostgresCreateTableDDL(schema, input)
	if err != nil {
		return nil, err
	}
	statements := []string{createTableDDL}
	qualifiedName := pgx.Identifier{schema, input.Name}.Sanitize()
	chunkInterval := normalizeTimeseriesInterval(input.Timeseries.ChunkInterval, "1 day")
	statements = append(statements, fmt.Sprintf(
		"SELECT create_hypertable(%s::regclass, %s, if_not_exists => TRUE, chunk_time_interval => INTERVAL %s)",
		quotePostgresLiteral(qualifiedName),
		quotePostgresLiteral(input.Timeseries.TimeColumn),
		quotePostgresLiteral(chunkInterval),
	))
	if input.Timeseries.RetentionDays != nil {
		statements = append(statements, fmt.Sprintf(
			"SELECT add_retention_policy(%s::regclass, INTERVAL %s, if_not_exists => TRUE)",
			quotePostgresLiteral(qualifiedName),
			quotePostgresLiteral(strconv.Itoa(*input.Timeseries.RetentionDays)+" days"),
		))
	}
	return statements, nil
}

func buildPostgresUpdateTableDDL(schema, tableName string, current *RelationalTableStructure, input UpdateRelationalTableInput) ([]string, error) {
	if current == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "当前表结构不能为空")
	}
	qualifiedTable := pgx.Identifier{schema, tableName}.Sanitize()
	currentColumns := make(map[string]RelationalTableColumn, len(current.Columns))
	for _, column := range current.Columns {
		currentColumns[column.Name] = column
	}
	nextColumns := make(map[string]CreateRelationalTableColumnInput, len(input.Columns))
	for _, column := range input.Columns {
		nextColumns[column.Name] = column
	}

	statements := make([]string, 0)
	for _, column := range input.Columns {
		currentColumn, exists := currentColumns[column.Name]
		if !exists {
			columnType, err := postgresColumnType(column)
			if err != nil {
				return nil, err
			}
			definition := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", qualifiedTable, pgx.Identifier{column.Name}.Sanitize(), columnType)
			if !column.Nullable {
				definition += " NOT NULL"
			}
			if column.DefaultValue != "" && !column.AutoIncrement {
				definition += " DEFAULT " + column.DefaultValue
			}
			statements = append(statements, definition)
			if column.Comment != "" {
				statements = append(statements, postgresColumnCommentDDL(schema, tableName, column.Name, column.Comment))
			}
			continue
		}
		if currentColumn.IsPrimary != column.Primary || currentColumn.AutoIncrement != column.AutoIncrement {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "暂不支持修改主键或自增字段: "+column.Name)
		}
		if !postgresColumnTypeMatches(currentColumn, column) {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "暂不支持修改字段类型: "+column.Name)
		}
		if !postgresColumnDefaultMatches(currentColumn, column) {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "暂不支持修改字段默认值: "+column.Name)
		}
		if currentColumn.Nullable != column.Nullable {
			if column.Nullable {
				statements = append(statements, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL", qualifiedTable, pgx.Identifier{column.Name}.Sanitize()))
			} else {
				statements = append(statements, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL", qualifiedTable, pgx.Identifier{column.Name}.Sanitize()))
			}
		}
		currentComment := ""
		if currentColumn.Comment != nil {
			currentComment = *currentColumn.Comment
		}
		if currentComment != column.Comment {
			statements = append(statements, postgresColumnCommentDDL(schema, tableName, column.Name, column.Comment))
		}
	}
	for _, column := range current.Columns {
		if _, exists := nextColumns[column.Name]; exists {
			continue
		}
		if column.IsPrimary {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "暂不支持删除主键字段: "+column.Name)
		}
		statements = append(statements, fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", qualifiedTable, pgx.Identifier{column.Name}.Sanitize()))
	}

	indexStatements, err := buildPostgresUpdateIndexDDL(schema, qualifiedTable, current.Indexes, input.Indexes)
	if err != nil {
		return nil, err
	}
	statements = append(statements, indexStatements...)
	return statements, nil
}

func buildPostgresUpdateIndexDDL(schema, qualifiedTable string, current []RelationalIndex, next []CreateRelationalTableIndexInput) ([]string, error) {
	currentIndexes := make(map[string]RelationalIndex, len(current))
	for _, index := range current {
		if strings.EqualFold(index.Type, "PRIMARY") {
			continue
		}
		currentIndexes[index.Name] = index
	}
	nextIndexes := make(map[string]CreateRelationalTableIndexInput, len(next))
	for _, index := range next {
		nextIndexes[index.Name] = index
	}
	statements := make([]string, 0)
	for name, index := range currentIndexes {
		nextIndex, exists := nextIndexes[name]
		if !exists {
			statements = append(statements, fmt.Sprintf("DROP INDEX %s", pgx.Identifier{schema, name}.Sanitize()))
			continue
		}
		if !postgresIndexMatches(index, nextIndex) {
			statements = append(statements, fmt.Sprintf("DROP INDEX %s", pgx.Identifier{schema, name}.Sanitize()))
			statements = append(statements, postgresCreateIndexDDL(qualifiedTable, nextIndex))
		}
	}
	for name, index := range nextIndexes {
		if _, exists := currentIndexes[name]; !exists {
			statements = append(statements, postgresCreateIndexDDL(qualifiedTable, index))
		}
	}
	return statements, nil
}

func postgresColumnCommentDDL(schema, tableName, columnName, comment string) string {
	value := "NULL"
	if comment != "" {
		value = "'" + strings.ReplaceAll(comment, "'", "''") + "'"
	}
	return fmt.Sprintf(
		"COMMENT ON COLUMN %s IS %s",
		pgx.Identifier{schema, tableName, columnName}.Sanitize(),
		value,
	)
}

func postgresCreateIndexDDL(qualifiedTable string, index CreateRelationalTableIndexInput) string {
	prefix := "CREATE INDEX"
	if index.Type == "unique" {
		prefix = "CREATE UNIQUE INDEX"
	}
	return fmt.Sprintf("%s %s ON %s (%s)", prefix, pgx.Identifier{index.Name}.Sanitize(), qualifiedTable, strings.Join(quotePostgresColumns(index.Columns), ", "))
}

func postgresIndexMatches(current RelationalIndex, next CreateRelationalTableIndexInput) bool {
	currentType := "index"
	if strings.EqualFold(current.Type, "UNIQUE") {
		currentType = "unique"
	}
	if currentType != next.Type || len(current.Columns) != len(next.Columns) {
		return false
	}
	for index, column := range current.Columns {
		if column != next.Columns[index] {
			return false
		}
	}
	return true
}

func postgresColumnTypeMatches(current RelationalTableColumn, next CreateRelationalTableColumnInput) bool {
	if next.AutoIncrement && current.AutoIncrement {
		return true
	}
	normalized := strings.ToLower(strings.TrimSpace(current.Type))
	switch next.Type {
	case "varchar", "string":
		return normalized == "character varying" || normalized == "varchar"
	case "int", "integer":
		return normalized == "integer"
	case "double", "float":
		return normalized == "double precision"
	case "json", "jsonb":
		return normalized == "jsonb"
	case "datetime", "timestamp":
		return normalized == "timestamp without time zone" || normalized == "timestamp"
	case "timestamptz":
		return normalized == "timestamp with time zone" || normalized == "timestamptz"
	default:
		return normalized == next.Type
	}
}

func postgresColumnDefaultMatches(current RelationalTableColumn, next CreateRelationalTableColumnInput) bool {
	if current.AutoIncrement {
		return true
	}
	currentDefault := ""
	if current.DefaultValue != nil {
		currentDefault = strings.TrimSpace(*current.DefaultValue)
	}
	return currentDefault == strings.TrimSpace(next.DefaultValue)
}

func postgresColumnType(column CreateRelationalTableColumnInput) (string, error) {
	if column.AutoIncrement {
		if column.Type == "bigint" {
			return "bigserial", nil
		}
		return "serial", nil
	}
	switch column.Type {
	case "string", "varchar":
		length := 255
		if column.Length != nil {
			length = *column.Length
		}
		return fmt.Sprintf("varchar(%d)", length), nil
	case "text":
		return "text", nil
	case "int", "integer":
		return "integer", nil
	case "bigint":
		return "bigint", nil
	case "float", "double":
		return "double precision", nil
	case "decimal":
		return decimalType("numeric", column), nil
	case "boolean":
		return "boolean", nil
	case "datetime", "timestamp":
		return "timestamp", nil
	case "timestamptz":
		return "timestamptz", nil
	case "json", "jsonb":
		return "jsonb", nil
	default:
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "字段类型不受支持: "+column.Type)
	}
}

func buildMySQLCreateTableDDL(schema string, input CreateRelationalTableInput) (string, error) {
	definitions := make([]string, 0, len(input.Columns)+len(input.Indexes)+1)
	primaryColumns := make([]string, 0)
	for _, column := range input.Columns {
		columnType, err := mysqlColumnType(column)
		if err != nil {
			return "", err
		}
		definition := fmt.Sprintf("`%s` %s", escapeMySQLIdentifier(column.Name), columnType)
		if column.Primary || !column.Nullable {
			definition += " NOT NULL"
		}
		if column.AutoIncrement {
			definition += " AUTO_INCREMENT"
		} else if column.DefaultValue != "" {
			definition += " DEFAULT " + column.DefaultValue
		}
		definitions = append(definitions, definition)
		if column.Primary {
			primaryColumns = append(primaryColumns, fmt.Sprintf("`%s`", escapeMySQLIdentifier(column.Name)))
		}
	}
	if len(primaryColumns) > 0 {
		definitions = append(definitions, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(primaryColumns, ", ")))
	}
	for _, index := range input.Indexes {
		columns := quoteMySQLColumns(index.Columns)
		prefix := "KEY"
		if index.Type == "unique" {
			prefix = "UNIQUE KEY"
		}
		definitions = append(definitions, fmt.Sprintf("%s `%s` (%s)", prefix, escapeMySQLIdentifier(index.Name), strings.Join(columns, ", ")))
	}
	return fmt.Sprintf("CREATE TABLE `%s`.`%s` (\n  %s\n)", escapeMySQLIdentifier(schema), escapeMySQLIdentifier(input.Name), strings.Join(definitions, ",\n  ")), nil
}

func mysqlColumnType(column CreateRelationalTableColumnInput) (string, error) {
	switch column.Type {
	case "string", "varchar":
		length := 255
		if column.Length != nil {
			length = *column.Length
		}
		return fmt.Sprintf("varchar(%d)", length), nil
	case "text":
		return "text", nil
	case "int", "integer":
		return "int", nil
	case "bigint":
		return "bigint", nil
	case "float":
		return "float", nil
	case "double":
		return "double", nil
	case "decimal":
		return decimalType("decimal", column), nil
	case "boolean":
		return "tinyint(1)", nil
	case "datetime", "timestamp", "timestamptz":
		return "datetime", nil
	case "json", "jsonb":
		return "json", nil
	default:
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "字段类型不受支持: "+column.Type)
	}
}

func buildSQLServerCreateTableDDL(schema string, input CreateRelationalTableInput) (string, error) {
	tableName := fmt.Sprintf("[%s].[%s]", escapeSQLServerIdentifier(schema), escapeSQLServerIdentifier(input.Name))
	definitions := make([]string, 0, len(input.Columns)+1)
	primaryColumns := make([]string, 0)
	for _, column := range input.Columns {
		columnType, err := sqlServerColumnType(column)
		if err != nil {
			return "", err
		}
		definition := fmt.Sprintf("[%s] %s", escapeSQLServerIdentifier(column.Name), columnType)
		if column.AutoIncrement {
			definition += " IDENTITY(1,1)"
		}
		if column.Primary || !column.Nullable {
			definition += " NOT NULL"
		} else {
			definition += " NULL"
		}
		if column.DefaultValue != "" && !column.AutoIncrement {
			definition += " DEFAULT " + column.DefaultValue
		}
		definitions = append(definitions, definition)
		if column.Primary {
			primaryColumns = append(primaryColumns, fmt.Sprintf("[%s]", escapeSQLServerIdentifier(column.Name)))
		}
	}
	if len(primaryColumns) > 0 {
		definitions = append(definitions, fmt.Sprintf("CONSTRAINT [PK_%s] PRIMARY KEY (%s)", escapeSQLServerIdentifier(input.Name), strings.Join(primaryColumns, ", ")))
	}
	statements := []string{fmt.Sprintf("CREATE TABLE %s (\n  %s\n)", tableName, strings.Join(definitions, ",\n  "))}
	for _, index := range input.Indexes {
		columns := quoteSQLServerColumns(index.Columns)
		prefix := "CREATE INDEX"
		if index.Type == "unique" {
			prefix = "CREATE UNIQUE INDEX"
		}
		statements = append(statements, fmt.Sprintf("%s [%s] ON %s (%s)", prefix, escapeSQLServerIdentifier(index.Name), tableName, strings.Join(columns, ", ")))
	}
	return strings.Join(statements, ";\n") + ";", nil
}

func sqlServerColumnType(column CreateRelationalTableColumnInput) (string, error) {
	switch column.Type {
	case "string", "varchar":
		length := 255
		if column.Length != nil {
			length = *column.Length
		}
		return fmt.Sprintf("nvarchar(%d)", length), nil
	case "text":
		return "nvarchar(max)", nil
	case "int", "integer":
		return "int", nil
	case "bigint":
		return "bigint", nil
	case "float", "double":
		return "float", nil
	case "decimal":
		return decimalType("decimal", column), nil
	case "boolean":
		return "bit", nil
	case "datetime", "timestamp", "timestamptz":
		return "datetime2", nil
	case "json", "jsonb":
		return "nvarchar(max)", nil
	default:
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "字段类型不受支持: "+column.Type)
	}
}

func decimalType(name string, column CreateRelationalTableColumnInput) string {
	precision := 18
	scale := 2
	if column.Precision != nil {
		precision = *column.Precision
	}
	if column.Scale != nil {
		scale = *column.Scale
	}
	return fmt.Sprintf("%s(%d,%d)", name, precision, scale)
}

func quotePostgresColumns(columns []string) []string {
	result := make([]string, 0, len(columns))
	for _, column := range columns {
		result = append(result, pgx.Identifier{column}.Sanitize())
	}
	return result
}

func quotePostgresLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func quoteMySQLColumns(columns []string) []string {
	result := make([]string, 0, len(columns))
	for _, column := range columns {
		result = append(result, fmt.Sprintf("`%s`", escapeMySQLIdentifier(column)))
	}
	return result
}

func quoteSQLServerColumns(columns []string) []string {
	result := make([]string, 0, len(columns))
	for _, column := range columns {
		result = append(result, fmt.Sprintf("[%s]", escapeSQLServerIdentifier(column)))
	}
	return result
}

func relationalTableKind(connectionType, dbType, tableType, tableName string) string {
	normalizedType := strings.ToUpper(strings.TrimSpace(tableType))
	if strings.Contains(normalizedType, "VIEW") {
		return "view"
	}
	return "table"
}

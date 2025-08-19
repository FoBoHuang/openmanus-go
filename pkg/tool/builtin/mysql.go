package builtin

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"openmanus-go/pkg/tool"
)

// MySQLTool MySQL 数据库工具
type MySQLTool struct {
	*tool.BaseTool
	db *sql.DB
}

// NewMySQLTool 创建 MySQL 工具
func NewMySQLTool(dsn string) (*MySQLTool, error) {
	inputSchema := tool.CreateJSONSchema("object", map[string]any{
		"operation": tool.StringProperty("操作类型：query, execute, insert, update, delete, describe, show_tables"),
		"sql":       tool.StringProperty("SQL 语句"),
		"table":     tool.StringProperty("表名（用于 describe 操作）"),
		"params":    tool.ArrayProperty("参数列表", tool.StringProperty("")),
	}, []string{"operation"})

	outputSchema := tool.CreateJSONSchema("object", map[string]any{
		"success":       tool.BooleanProperty("操作是否成功"),
		"result":        tool.StringProperty("操作结果描述"),
		"rows":          tool.ArrayProperty("查询结果行", tool.ObjectProperty("行数据", nil)),
		"affected_rows": tool.NumberProperty("影响的行数"),
		"columns":       tool.ArrayProperty("列信息", tool.StringProperty("")),
		"error":         tool.StringProperty("错误信息"),
	}, []string{"success"})

	baseTool := tool.NewBaseTool(
		"mysql",
		"MySQL 数据库操作工具，支持查询、插入、更新、删除等操作",
		inputSchema,
		outputSchema,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping MySQL: %w", err)
	}

	return &MySQLTool{
		BaseTool: baseTool,
		db:       db,
	}, nil
}

// Invoke 执行 MySQL 操作
func (m *MySQLTool) Invoke(ctx context.Context, args map[string]any) (map[string]any, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return m.errorResult("operation is required"), nil
	}

	switch strings.ToLower(operation) {
	case "query":
		sqlStr, _ := args["sql"].(string)
		params := m.parseParams(args["params"])
		return m.query(ctx, sqlStr, params)
	case "execute":
		sqlStr, _ := args["sql"].(string)
		params := m.parseParams(args["params"])
		return m.execute(ctx, sqlStr, params)
	case "insert":
		sqlStr, _ := args["sql"].(string)
		params := m.parseParams(args["params"])
		return m.insert(ctx, sqlStr, params)
	case "update":
		sqlStr, _ := args["sql"].(string)
		params := m.parseParams(args["params"])
		return m.update(ctx, sqlStr, params)
	case "delete":
		sqlStr, _ := args["sql"].(string)
		params := m.parseParams(args["params"])
		return m.delete(ctx, sqlStr, params)
	case "describe":
		table, _ := args["table"].(string)
		return m.describe(ctx, table)
	case "show_tables":
		return m.showTables(ctx)
	default:
		return m.errorResult(fmt.Sprintf("unsupported operation: %s", operation)), nil
	}
}

// parseParams 解析参数
func (m *MySQLTool) parseParams(paramsRaw any) []any {
	if paramsRaw == nil {
		return nil
	}

	if params, ok := paramsRaw.([]any); ok {
		return params
	}

	return nil
}

// query 执行查询操作
func (m *MySQLTool) query(ctx context.Context, sqlStr string, params []any) (map[string]any, error) {
	if sqlStr == "" {
		return m.errorResult("sql is required for query operation"), nil
	}

	rows, err := m.db.QueryContext(ctx, sqlStr, params...)
	if err != nil {
		return m.errorResult(fmt.Sprintf("query failed: %v", err)), nil
	}
	defer rows.Close()

	// 获取列信息
	columns, err := rows.Columns()
	if err != nil {
		return m.errorResult(fmt.Sprintf("failed to get columns: %v", err)), nil
	}

	// 读取数据
	var results []map[string]any
	for rows.Next() {
		// 创建扫描目标
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return m.errorResult(fmt.Sprintf("failed to scan row: %v", err)), nil
		}

		// 构建行数据
		row := make(map[string]any)
		for i, col := range columns {
			if values[i] != nil {
				// 处理字节数组
				if b, ok := values[i].([]byte); ok {
					row[col] = string(b)
				} else {
					row[col] = values[i]
				}
			} else {
				row[col] = nil
			}
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return m.errorResult(fmt.Sprintf("row iteration error: %v", err)), nil
	}

	return map[string]any{
		"success": true,
		"result":  fmt.Sprintf("Query executed successfully, returned %d rows", len(results)),
		"rows":    results,
		"columns": columns,
	}, nil
}

// execute 执行通用 SQL
func (m *MySQLTool) execute(ctx context.Context, sqlStr string, params []any) (map[string]any, error) {
	if sqlStr == "" {
		return m.errorResult("sql is required for execute operation"), nil
	}

	result, err := m.db.ExecContext(ctx, sqlStr, params...)
	if err != nil {
		return m.errorResult(fmt.Sprintf("execute failed: %v", err)), nil
	}

	affectedRows, _ := result.RowsAffected()
	lastInsertId, _ := result.LastInsertId()

	return map[string]any{
		"success":        true,
		"result":         "SQL executed successfully",
		"affected_rows":  affectedRows,
		"last_insert_id": lastInsertId,
	}, nil
}

// insert 执行插入操作
func (m *MySQLTool) insert(ctx context.Context, sqlStr string, params []any) (map[string]any, error) {
	if sqlStr == "" {
		return m.errorResult("sql is required for insert operation"), nil
	}

	result, err := m.db.ExecContext(ctx, sqlStr, params...)
	if err != nil {
		return m.errorResult(fmt.Sprintf("insert failed: %v", err)), nil
	}

	affectedRows, _ := result.RowsAffected()
	lastInsertId, _ := result.LastInsertId()

	return map[string]any{
		"success":        true,
		"result":         fmt.Sprintf("Inserted %d row(s)", affectedRows),
		"affected_rows":  affectedRows,
		"last_insert_id": lastInsertId,
	}, nil
}

// update 执行更新操作
func (m *MySQLTool) update(ctx context.Context, sqlStr string, params []any) (map[string]any, error) {
	if sqlStr == "" {
		return m.errorResult("sql is required for update operation"), nil
	}

	result, err := m.db.ExecContext(ctx, sqlStr, params...)
	if err != nil {
		return m.errorResult(fmt.Sprintf("update failed: %v", err)), nil
	}

	affectedRows, _ := result.RowsAffected()

	return map[string]any{
		"success":       true,
		"result":        fmt.Sprintf("Updated %d row(s)", affectedRows),
		"affected_rows": affectedRows,
	}, nil
}

// delete 执行删除操作
func (m *MySQLTool) delete(ctx context.Context, sqlStr string, params []any) (map[string]any, error) {
	if sqlStr == "" {
		return m.errorResult("sql is required for delete operation"), nil
	}

	result, err := m.db.ExecContext(ctx, sqlStr, params...)
	if err != nil {
		return m.errorResult(fmt.Sprintf("delete failed: %v", err)), nil
	}

	affectedRows, _ := result.RowsAffected()

	return map[string]any{
		"success":       true,
		"result":        fmt.Sprintf("Deleted %d row(s)", affectedRows),
		"affected_rows": affectedRows,
	}, nil
}

// describe 描述表结构
func (m *MySQLTool) describe(ctx context.Context, table string) (map[string]any, error) {
	if table == "" {
		return m.errorResult("table is required for describe operation"), nil
	}

	sqlStr := fmt.Sprintf("DESCRIBE `%s`", table)
	rows, err := m.db.QueryContext(ctx, sqlStr)
	if err != nil {
		return m.errorResult(fmt.Sprintf("describe failed: %v", err)), nil
	}
	defer rows.Close()

	var results []map[string]any
	for rows.Next() {
		var field, fieldType, null, key, defaultVal, extra sql.NullString

		if err := rows.Scan(&field, &fieldType, &null, &key, &defaultVal, &extra); err != nil {
			return m.errorResult(fmt.Sprintf("failed to scan describe result: %v", err)), nil
		}

		row := map[string]any{
			"Field":   field.String,
			"Type":    fieldType.String,
			"Null":    null.String,
			"Key":     key.String,
			"Default": defaultVal.String,
			"Extra":   extra.String,
		}
		results = append(results, row)
	}

	return map[string]any{
		"success": true,
		"result":  fmt.Sprintf("Table '%s' structure retrieved", table),
		"rows":    results,
	}, nil
}

// showTables 显示所有表
func (m *MySQLTool) showTables(ctx context.Context) (map[string]any, error) {
	rows, err := m.db.QueryContext(ctx, "SHOW TABLES")
	if err != nil {
		return m.errorResult(fmt.Sprintf("show tables failed: %v", err)), nil
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return m.errorResult(fmt.Sprintf("failed to scan table name: %v", err)), nil
		}
		tables = append(tables, table)
	}

	return map[string]any{
		"success": true,
		"result":  fmt.Sprintf("Found %d tables", len(tables)),
		"tables":  tables,
	}, nil
}

// errorResult 创建错误结果
func (m *MySQLTool) errorResult(message string) map[string]any {
	return map[string]any{
		"success": false,
		"error":   message,
	}
}

// Close 关闭数据库连接
func (m *MySQLTool) Close() error {
	return m.db.Close()
}

// Ping 测试数据库连接
func (m *MySQLTool) Ping(ctx context.Context) error {
	return m.db.PingContext(ctx)
}

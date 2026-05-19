package schema

import (
	"context"
	"fmt"
	"strings"

	"github.com/gogf/gf/v2/database/gdb"

	"ksogit.kingsoft.net/wpsee/itabbase/server/internal/model"
)

// SyncAll creates/alters tables for all collections.
func SyncAll(ctx context.Context, db gdb.DB, cols []model.Collection) error {
	dialect := Dialect(db)
	for _, c := range cols {
		if err := syncOne(ctx, db, dialect, c); err != nil {
			return fmt.Errorf("sync %s: %w", c.Name, err)
		}
	}
	return nil
}

// SyncNonBuiltin syncs only non-builtin collections.
func SyncNonBuiltin(ctx context.Context, db gdb.DB, cols []model.Collection) error {
	dialect := Dialect(db)
	for _, c := range cols {
		if c.Source == model.SourceBuiltin {
			continue
		}
		if err := syncOne(ctx, db, dialect, c); err != nil {
			return fmt.Errorf("sync %s: %w", c.Name, err)
		}
	}
	return nil
}

func syncOne(ctx context.Context, db gdb.DB, dialect string, c model.Collection) error {
	tbl := c.DBTable()
	exists, err := tableExists(ctx, db, dialect, tbl)
	if err != nil {
		return err
	}
	if !exists {
		stmt := BuildCreateTable(dialect, c)
		if _, err := db.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("create table: %w", err)
		}
		return nil
	}
	cols, err := tableColumns(ctx, db, tbl)
	if err != nil {
		return err
	}
	for _, f := range c.Fields {
		if f.IsVirtual() {
			continue
		}
		if _, ok := cols[f.Name]; ok {
			continue
		}
		stmt := BuildAddColumn(dialect, tbl, f)
		if _, err := db.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("add column %s.%s: %w", tbl, f.Name, err)
		}
	}
	return nil
}

// Dialect returns the DB driver type in lowercase.
func Dialect(db gdb.DB) string {
	cfg := db.GetConfig()
	if cfg == nil {
		return ""
	}
	return strings.ToLower(cfg.Type)
}

func tableExists(ctx context.Context, db gdb.DB, dialect, name string) (bool, error) {
	var sql string
	switch dialect {
	case "sqlite":
		sql = `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`
	case "mysql":
		sql = `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name=?`
	default:
		return false, fmt.Errorf("itab: unsupported dialect %q", dialect)
	}
	v, err := db.GetValue(ctx, sql, name)
	if err != nil {
		return false, err
	}
	return v.Int() > 0, nil
}

func tableColumns(ctx context.Context, db gdb.DB, name string) (map[string]gdb.TableField, error) {
	fields, err := db.TableFields(ctx, name)
	if err != nil {
		return nil, err
	}
	out := make(map[string]gdb.TableField, len(fields))
	for k, v := range fields {
		if v != nil {
			out[k] = *v
		}
	}
	return out, nil
}

// BuildCreateTable returns a CREATE TABLE statement for the given collection.
func BuildCreateTable(dialect string, c model.Collection) string {
	var b strings.Builder
	b.WriteString("CREATE TABLE ")
	b.WriteString(QuoteIdent(dialect, c.DBTable()))
	b.WriteString(" (\n")
	b.WriteString("  ")
	b.WriteString(idColumnDef(dialect))
	for _, f := range c.Fields {
		if f.IsVirtual() {
			continue
		}
		b.WriteString(",\n  ")
		b.WriteString(columnDef(dialect, f))
	}
	b.WriteString("\n)")
	if dialect == "mysql" {
		b.WriteString(" ENGINE=InnoDB DEFAULT CHARSET=utf8mb4")
	}
	return b.String()
}

// BuildAddColumn returns an ALTER TABLE ADD COLUMN statement.
func BuildAddColumn(dialect, table string, f model.Field) string {
	return fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s",
		QuoteIdent(dialect, table),
		columnDef(dialect, f),
	)
}

func idColumnDef(dialect string) string {
	switch dialect {
	case "sqlite":
		return `"id" INTEGER PRIMARY KEY AUTOINCREMENT`
	case "mysql":
		return "`id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY"
	}
	return `"id" INTEGER PRIMARY KEY`
}

func columnDef(dialect string, f model.Field) string {
	parts := []string{QuoteIdent(dialect, f.Name), sqlType(dialect, f)}
	if f.Required {
		parts = append(parts, "NOT NULL")
	}
	if f.Default != nil {
		parts = append(parts, "DEFAULT "+formatDefault(f))
	}
	return strings.Join(parts, " ")
}

func sqlType(dialect string, f model.Field) string {
	switch dialect {
	case "sqlite":
		switch f.Type {
		case model.TString, model.TText:
			return "TEXT"
		case model.TInt, model.TBool, model.TBelongsTo:
			return "INTEGER"
		case model.TFloat:
			return "REAL"
		case model.TDateTime:
			return "DATETIME"
		}
	case "mysql":
		switch f.Type {
		case model.TString:
			n := f.MaxLen
			if n <= 0 {
				n = 255
			}
			return fmt.Sprintf("VARCHAR(%d)", n)
		case model.TText:
			return "TEXT"
		case model.TInt, model.TBelongsTo:
			return "BIGINT"
		case model.TBool:
			return "TINYINT(1)"
		case model.TFloat:
			return "DOUBLE"
		case model.TDateTime:
			return "DATETIME"
		}
	}
	return "TEXT"
}

func formatDefault(f model.Field) string {
	switch v := f.Default.(type) {
	case bool:
		if v {
			return "1"
		}
		return "0"
	case string:
		return "'" + strings.ReplaceAll(v, "'", "''") + "'"
	case int, int32, int64, float32, float64:
		return fmt.Sprintf("%v", v)
	}
	return "NULL"
}

// QuoteIdent quotes an identifier for the given dialect.
func QuoteIdent(dialect, name string) string {
	if dialect == "mysql" {
		return "`" + name + "`"
	}
	return `"` + name + `"`
}

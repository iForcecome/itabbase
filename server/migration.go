package itab

import (
	"context"
	"fmt"
	"strings"

	"github.com/gogf/gf/v2/database/gdb"
)

func (k *Kernel) syncCollections(ctx context.Context) error {
	dialect := k.dialect()
	for _, c := range k.collections {
		if err := k.syncOne(ctx, dialect, c); err != nil {
			return fmt.Errorf("sync %s: %w", c.Name, err)
		}
	}
	return nil
}

func (k *Kernel) syncOne(ctx context.Context, dialect string, c Collection) error {
	exists, err := k.tableExists(ctx, dialect, c.Name)
	if err != nil {
		return err
	}
	if !exists {
		stmt := buildCreateTable(dialect, c)
		if _, err := k.db.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("create table: %w", err)
		}
		return nil
	}
	cols, err := k.tableColumns(ctx, c.Name)
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
		stmt := buildAddColumn(dialect, c.Name, f)
		if _, err := k.db.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("add column %s.%s: %w", c.Name, f.Name, err)
		}
	}
	return nil
}

func (k *Kernel) dialect() string {
	cfg := k.db.GetConfig()
	if cfg == nil {
		return ""
	}
	return strings.ToLower(cfg.Type)
}

func (k *Kernel) tableExists(ctx context.Context, dialect, name string) (bool, error) {
	var sql string
	switch dialect {
	case "sqlite":
		sql = `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`
	case "mysql":
		sql = `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name=?`
	default:
		return false, fmt.Errorf("itab: unsupported dialect %q", dialect)
	}
	v, err := k.db.GetValue(ctx, sql, name)
	if err != nil {
		return false, err
	}
	return v.Int() > 0, nil
}

func (k *Kernel) tableColumns(ctx context.Context, name string) (map[string]gdb.TableField, error) {
	fields, err := k.db.TableFields(ctx, name)
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

func buildCreateTable(dialect string, c Collection) string {
	var b strings.Builder
	b.WriteString("CREATE TABLE ")
	b.WriteString(quoteIdent(dialect, c.Name))
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

func buildAddColumn(dialect, table string, f Field) string {
	return fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s",
		quoteIdent(dialect, table),
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

func columnDef(dialect string, f Field) string {
	parts := []string{quoteIdent(dialect, f.Name), sqlType(dialect, f)}
	if f.Required {
		parts = append(parts, "NOT NULL")
	}
	if f.Default != nil {
		parts = append(parts, "DEFAULT "+formatDefault(f))
	}
	return strings.Join(parts, " ")
}

func sqlType(dialect string, f Field) string {
	switch dialect {
	case "sqlite":
		switch f.Type {
		case TString, TText:
			return "TEXT"
		case TInt, TBool, TBelongsTo:
			return "INTEGER"
		case TFloat:
			return "REAL"
		case TDateTime:
			return "DATETIME"
		}
	case "mysql":
		switch f.Type {
		case TString:
			n := f.MaxLen
			if n <= 0 {
				n = 255
			}
			return fmt.Sprintf("VARCHAR(%d)", n)
		case TText:
			return "TEXT"
		case TInt, TBelongsTo:
			return "BIGINT"
		case TBool:
			return "TINYINT(1)"
		case TFloat:
			return "DOUBLE"
		case TDateTime:
			return "DATETIME"
		}
	}
	return "TEXT"
}

func formatDefault(f Field) string {
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

func quoteIdent(dialect, name string) string {
	if dialect == "mysql" {
		return "`" + name + "`"
	}
	return `"` + name + `"`
}

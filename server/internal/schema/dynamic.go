package schema

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"

	"ksogit.kingsoft.net/wpsee/itabbase/server/internal/model"
)

// FindCollection returns the collection with the given name from the slice.
func FindCollection(cols []model.Collection, name string) (model.Collection, bool) {
	for _, c := range cols {
		if c.Name == name {
			return c, true
		}
	}
	return model.Collection{}, false
}

// LoadDynamic reads _collections + _fields from the DB and returns
// the resulting Collection structs. Existing collections (by name) are skipped.
func LoadDynamic(ctx context.Context, db gdb.DB, existing []model.Collection) ([]model.Collection, error) {
	colRows, err := db.Model(model.BuiltinMetaCollections).Ctx(ctx).OrderAsc("sort").OrderAsc("id").All()
	if err != nil {
		return nil, fmt.Errorf("load _collections: %w", err)
	}
	if colRows.IsEmpty() {
		return nil, nil
	}

	fieldRows, err := db.Model(model.BuiltinMetaFields).Ctx(ctx).OrderAsc("sort").OrderAsc("id").All()
	if err != nil {
		return nil, fmt.Errorf("load _fields: %w", err)
	}

	fieldsByCol := map[string][]model.Field{}
	for _, row := range fieldRows.List() {
		colName, _ := row["collection_name"].(string)
		f := FieldFromRow(row)
		fieldsByCol[colName] = append(fieldsByCol[colName], f)
	}

	var result []model.Collection
	for _, row := range colRows.List() {
		name, _ := row["name"].(string)
		display, _ := row["display"].(string)
		if _, exists := FindCollection(existing, name); exists {
			g.Log().Warningf(ctx, "[itab] skip dynamic collection %q: name conflicts with existing", name)
			continue
		}
		fields := fieldsByCol[name]
		if len(fields) == 0 {
			continue
		}
		c := model.Collection{
			Name:    name,
			Display: display,
			Fields:  fields,
			Source:   model.SourceDynamic,
		}
		if err := c.Validate(); err != nil {
			g.Log().Warningf(ctx, "[itab] skip invalid dynamic collection %q: %v", name, err)
			continue
		}
		result = append(result, c)
	}
	return result, nil
}

// FieldFromRow converts a DB row map into a model.Field.
func FieldFromRow(row map[string]any) model.Field {
	getString := func(key string) string {
		v, _ := row[key].(string)
		return v
	}
	getInt := func(key string) int {
		switch v := row[key].(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
		return 0
	}
	getBool := func(key string) bool {
		switch v := row[key].(type) {
		case bool:
			return v
		case int:
			return v != 0
		case int64:
			return v != 0
		case float64:
			return v != 0
		}
		return false
	}

	f := model.Field{
		Name:     getString("name"),
		Type:     model.FieldType(getString("type")),
		Required: getBool("required"),
		MaxLen:   getInt("max_len"),
		Target:   getString("target"),
		Through:  getString("through"),
	}
	if dv := getString("default_value"); dv != "" {
		var parsed any
		if json.Unmarshal([]byte(dv), &parsed) == nil {
			f.Default = parsed
		}
	}
	return f
}

// ParseFieldsFromBody parses a JSON array of field definitions.
func ParseFieldsFromBody(raw []any) []model.Field {
	fields := make([]model.Field, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		f := ParseOneField(m)
		if f.Name != "" && f.Type != "" {
			fields = append(fields, f)
		}
	}
	return fields
}

// ParseOneField converts a JSON map into a model.Field.
func ParseOneField(m map[string]any) model.Field {
	getString := func(key string) string {
		v, _ := m[key].(string)
		return v
	}
	getInt := func(key string) int {
		switch v := m[key].(type) {
		case float64:
			return int(v)
		case int:
			return v
		case int64:
			return int(v)
		}
		return 0
	}
	getBool := func(key string) bool {
		v, _ := m[key].(bool)
		return v
	}

	f := model.Field{
		Name:     getString("name"),
		Type:     model.FieldType(getString("type")),
		Required: getBool("required"),
		MaxLen:   getInt("max_len"),
		Target:   getString("target"),
		Through:  getString("through"),
	}
	if dv := getString("default_value"); dv != "" {
		var parsed any
		if json.Unmarshal([]byte(dv), &parsed) == nil {
			f.Default = parsed
		}
	} else if dv, ok := m["default"]; ok {
		f.Default = dv
	}
	return f
}

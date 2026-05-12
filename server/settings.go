package itab

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
)

// GetSetting reads a single value from the `system_settings` collection.
// Returns "" if the key doesn't exist. No caching — the table is tiny and
// callers (e.g. auth callback) hit it once per request.
func GetSetting(ctx context.Context, db gdb.DB, key string) string {
	row, err := db.Model(BuiltinSettings).Ctx(ctx).Where("key", key).One()
	if err != nil || row.IsEmpty() {
		return ""
	}
	return row["value"].String()
}

// SetSetting upserts a setting key/value pair. Returns error from DB layer.
func SetSetting(ctx context.Context, db gdb.DB, key, value string) error {
	n, err := db.Model(BuiltinSettings).Ctx(ctx).Where("key", key).Count()
	if err != nil {
		return err
	}
	if n == 0 {
		_, err = db.Model(BuiltinSettings).Ctx(ctx).Insert(map[string]any{
			"key":   key,
			"value": value,
		})
		return err
	}
	_, err = db.Model(BuiltinSettings).Ctx(ctx).
		Where("key", key).Data(map[string]any{"value": value}).Update()
	return err
}

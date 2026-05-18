package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"

	"ksogit.kingsoft.net/wpsee/itabbase/server/internal/auth"
	"ksogit.kingsoft.net/wpsee/itabbase/server/internal/model"
)

const (
	defaultSuperAdminUsername = "itabbase"
	defaultSuperAdminPassword = "admin123"
	defaultAdminRoleName      = "admin"
	defaultUserRoleName       = "user"
)

// Seed creates default roles, super admin account, and core system_settings
// on first launch. Idempotent.
func Seed(ctx context.Context, db gdb.DB) error {
	if err := ensureRole(ctx, db, defaultAdminRoleName, "管理员"); err != nil {
		return fmt.Errorf("seed role %q: %w", defaultAdminRoleName, err)
	}
	if err := ensureRole(ctx, db, defaultUserRoleName, "普通用户"); err != nil {
		return fmt.Errorf("seed role %q: %w", defaultUserRoleName, err)
	}
	if err := ensureSuperAdmin(ctx, db); err != nil {
		return fmt.Errorf("seed super admin: %w", err)
	}
	if err := ensureDefaultSettings(ctx, db); err != nil {
		return fmt.Errorf("seed settings: %w", err)
	}
	return nil
}

func ensureDefaultSettings(ctx context.Context, db gdb.DB) error {
	defaults := []struct{ Key, Value string }{
		{"require_approval", "true"},
	}
	for _, d := range defaults {
		n, err := db.Model(model.BuiltinSettings).Ctx(ctx).Where("key", d.Key).Count()
		if err != nil {
			return err
		}
		if n > 0 {
			continue
		}
		if _, err := db.Model(model.BuiltinSettings).Ctx(ctx).Insert(g.Map{
			"key":   d.Key,
			"value": d.Value,
		}); err != nil {
			return err
		}
	}
	return nil
}

func ensureRole(ctx context.Context, db gdb.DB, name, display string) error {
	n, err := db.Model(model.BuiltinRoles).Ctx(ctx).Where("name", name).Count()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	_, err = db.Model(model.BuiltinRoles).Ctx(ctx).Insert(g.Map{
		"name":    name,
		"display": display,
	})
	return err
}

func ensureSuperAdmin(ctx context.Context, db gdb.DB) error {
	n, err := db.Model(model.BuiltinUserRoles+" ur").Ctx(ctx).
		LeftJoin(model.BuiltinRoles+" r", "r.id = ur.role_id").
		Where("r.name", defaultAdminRoleName).
		Count()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	hash, err := auth.HashPassword(defaultSuperAdminPassword)
	if err != nil {
		return err
	}
	now := time.Now()
	result, err := db.Model(model.BuiltinUsers).Ctx(ctx).Insert(g.Map{
		"username":      defaultSuperAdminUsername,
		"display_name":  "Super Admin",
		"password_hash": hash,
		"status":        model.UserStatusActive,
		"disabled":      false,
		"first_seen_at": now,
		"last_seen_at":  now,
	})
	if err != nil {
		return err
	}
	userID, err := result.LastInsertId()
	if err != nil {
		return err
	}
	roleRow, err := db.Model(model.BuiltinRoles).Ctx(ctx).Where("name", defaultAdminRoleName).One()
	if err != nil || roleRow.IsEmpty() {
		return fmt.Errorf("admin role missing after ensureRole")
	}
	roleID := roleRow["id"].Int64()
	if _, err := db.Model(model.BuiltinUserRoles).Ctx(ctx).Insert(g.Map{
		"user_id": userID,
		"role_id": roleID,
	}); err != nil {
		return err
	}
	g.Log().Warningf(ctx,
		"[itab] seeded super admin (username=%q password=%q) — CHANGE THIS BEFORE EXPOSING TO ANY UNTRUSTED NETWORK",
		defaultSuperAdminUsername, defaultSuperAdminPassword)
	return nil
}

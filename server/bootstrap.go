package itab

import (
	"context"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

const (
	defaultSuperAdminUsername = "itabbase"
	defaultSuperAdminPassword = "admin123"
	defaultAdminRoleName      = "admin"
	defaultUserRoleName       = "user"
)

// ensureBootstrap seeds default roles ("admin"/"user"), a default super
// admin account, and core system_settings on first launch. Idempotent.
func (k *Kernel) ensureBootstrap(ctx context.Context) error {
	if err := k.ensureRole(ctx, defaultAdminRoleName, "管理员"); err != nil {
		return fmt.Errorf("seed role %q: %w", defaultAdminRoleName, err)
	}
	if err := k.ensureRole(ctx, defaultUserRoleName, "普通用户"); err != nil {
		return fmt.Errorf("seed role %q: %w", defaultUserRoleName, err)
	}
	if err := k.ensureSuperAdmin(ctx); err != nil {
		return fmt.Errorf("seed super admin: %w", err)
	}
	if err := k.ensureDefaultSettings(ctx); err != nil {
		return fmt.Errorf("seed settings: %w", err)
	}
	return nil
}

// ensureDefaultSettings inserts default system_settings rows if missing.
func (k *Kernel) ensureDefaultSettings(ctx context.Context) error {
	defaults := []struct{ Key, Value string }{
		{"require_approval", "true"},
	}
	for _, d := range defaults {
		n, err := k.db.Model(BuiltinSettings).Ctx(ctx).Where("key", d.Key).Count()
		if err != nil {
			return err
		}
		if n > 0 {
			continue
		}
		if _, err := k.db.Model(BuiltinSettings).Ctx(ctx).Insert(g.Map{
			"key":   d.Key,
			"value": d.Value,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (k *Kernel) ensureRole(ctx context.Context, name, display string) error {
	n, err := k.db.Model(BuiltinRoles).Ctx(ctx).Where("name", name).Count()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	_, err = k.db.Model(BuiltinRoles).Ctx(ctx).Insert(g.Map{
		"name":    name,
		"display": display,
	})
	return err
}

// ensureSuperAdmin creates the default itabbase/admin123 account and grants
// it the admin role, only if no user currently holds the admin role.
func (k *Kernel) ensureSuperAdmin(ctx context.Context) error {
	n, err := k.db.Model(BuiltinUserRoles+" ur").Ctx(ctx).
		LeftJoin(BuiltinRoles+" r", "r.id = ur.role_id").
		Where("r.name", defaultAdminRoleName).
		Count()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	hash, err := HashPassword(defaultSuperAdminPassword)
	if err != nil {
		return err
	}
	now := time.Now()
	result, err := k.db.Model(BuiltinUsers).Ctx(ctx).Insert(g.Map{
		"username":      defaultSuperAdminUsername,
		"display_name":  "Super Admin",
		"password_hash": hash,
		"status":        UserStatusActive,
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
	roleRow, err := k.db.Model(BuiltinRoles).Ctx(ctx).Where("name", defaultAdminRoleName).One()
	if err != nil || roleRow.IsEmpty() {
		return fmt.Errorf("admin role missing after ensureRole")
	}
	roleID := roleRow["id"].Int64()
	if _, err := k.db.Model(BuiltinUserRoles).Ctx(ctx).Insert(g.Map{
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

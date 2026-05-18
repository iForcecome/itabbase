package itab

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
)

// Builtin collection names. Kernel auto-registers these on New(); business
// code calling RegisterCollection with these names will panic (duplicate).
const (
	BuiltinUsers          = "users"
	BuiltinRoles          = "roles"
	BuiltinUserRoles      = "user_roles"
	BuiltinSettings       = "system_settings"
	BuiltinMetaCollections = "_collections"
	BuiltinMetaFields      = "_fields"

	SourceBuiltin = "builtin"
	SourceCode    = "code"
	SourceDynamic = "dynamic"
)

// User lifecycle status values stored in `users.status`.
const (
	UserStatusActive   = "active"
	UserStatusPending  = "pending"
	UserStatusRejected = "rejected"
)

func builtinUsersCollection() Collection {
	return Collection{
		Name:     BuiltinUsers,
		Display:  "用户",
		Source:   SourceBuiltin,
		Internal: true,
		Fields: []Field{
			{Name: "username", Type: TString, MaxLen: 64},
			{Name: "external_id", Type: TString, MaxLen: 128},
			{Name: "display_name", Type: TString, MaxLen: 200},
			{Name: "avatar", Type: TString, MaxLen: 500},
			{Name: "password_hash", Type: TString, MaxLen: 200},
			{Name: "status", Type: TString, MaxLen: 20, Default: UserStatusActive},
			{Name: "disabled", Type: TBool, Default: false},
			{Name: "first_seen_at", Type: TDateTime},
			{Name: "last_seen_at", Type: TDateTime},
			{Name: "user_roles", Type: THasMany, Target: BuiltinUserRoles, Through: "user_id"},
		},
		ACL: ACL{
			"admin": {ActionAll},
			"user":  {ActionList, ActionGet},
		},
	}
}

func builtinSettingsCollection() Collection {
	return Collection{
		Name:     BuiltinSettings,
		Display:  "系统设置",
		Source:   SourceBuiltin,
		Internal: true,
		Fields: []Field{
			{Name: "key", Type: TString, Required: true, MaxLen: 64},
			{Name: "value", Type: TString, MaxLen: 500},
		},
		ACL: ACL{
			"admin": {ActionAll},
		},
	}
}

func builtinRolesCollection() Collection {
	return Collection{
		Name:     BuiltinRoles,
		Display:  "角色",
		Source:   SourceBuiltin,
		Internal: true,
		Fields: []Field{
			{Name: "name", Type: TString, Required: true, MaxLen: 64},
			{Name: "display", Type: TString, MaxLen: 200},
		},
		ACL: ACL{
			"admin": {ActionAll},
			"user":  {ActionList, ActionGet},
		},
	}
}

func builtinUserRolesCollection() Collection {
	return Collection{
		Name:     BuiltinUserRoles,
		Display:  "用户角色",
		Source:   SourceBuiltin,
		Internal: true,
		Fields: []Field{
			{Name: "user_id", Type: TBelongsTo, Target: BuiltinUsers, Required: true},
			{Name: "role_id", Type: TBelongsTo, Target: BuiltinRoles, Required: true},
		},
		ACL: ACL{
			"admin": {ActionAll},
			"user":  {ActionList, ActionGet},
		},
	}
}

func builtinMetaCollectionsCollection() Collection {
	return Collection{
		Name:     BuiltinMetaCollections,
		Display:  "集合定义",
		Source:   SourceBuiltin,
		Internal: true,
		Fields: []Field{
			{Name: "name", Type: TString, Required: true, MaxLen: 64},
			{Name: "display", Type: TString, MaxLen: 200},
			{Name: "icon", Type: TString, MaxLen: 64},
			{Name: "sort", Type: TInt, Default: 0},
		},
		ACL: ACL{"admin": {ActionAll}},
	}
}

func builtinMetaFieldsCollection() Collection {
	return Collection{
		Name:     BuiltinMetaFields,
		Display:  "字段定义",
		Source:   SourceBuiltin,
		Internal: true,
		Fields: []Field{
			{Name: "collection_name", Type: TString, Required: true, MaxLen: 64},
			{Name: "name", Type: TString, Required: true, MaxLen: 64},
			{Name: "type", Type: TString, Required: true, MaxLen: 32},
			{Name: "display", Type: TString, MaxLen: 200},
			{Name: "required", Type: TBool, Default: false},
			{Name: "default_value", Type: TString, MaxLen: 500},
			{Name: "max_len", Type: TInt, Default: 0},
			{Name: "target", Type: TString, MaxLen: 64},
			{Name: "through", Type: TString, MaxLen: 64},
			{Name: "sort", Type: TInt, Default: 0},
		},
		ACL: ACL{"admin": {ActionAll}},
	}
}

// registerBuiltins appends kernel's core collections to the registry.
// Hooks that need access to k.db are attached here as closures.
func (k *Kernel) registerBuiltins() {
	usersCol := builtinUsersCollection()
	usersCol.Hooks.AfterUpdate = k.bindUserRoleOnActivate

	cols := []Collection{
		usersCol,
		builtinRolesCollection(),
		builtinUserRolesCollection(),
		builtinSettingsCollection(),
		builtinMetaCollectionsCollection(),
		builtinMetaFieldsCollection(),
	}
	for _, c := range cols {
		if c.Source == "" {
			c.Source = SourceBuiltin
			c.Internal = true
		}
		if err := c.Validate(); err != nil {
			panic("itab builtin: " + err.Error())
		}
		k.collections = append(k.collections, c)
	}
}

// bindUserRoleOnActivate is the users.AfterUpdate hook: when a user's status
// transitions to "active" (typically admin approval) and that user has no
// user_roles row yet, insert one binding to the "user" role. Idempotent.
func (k *Kernel) bindUserRoleOnActivate(ctx context.Context, rec *Record) error {
	status, _ := rec.Get("status").(string)
	if status != UserStatusActive {
		return nil
	}
	var userID int64
	switch v := rec.Get("id").(type) {
	case int64:
		userID = v
	case int:
		userID = int64(v)
	case float64:
		userID = int64(v)
	default:
		return nil
	}
	if userID == 0 {
		return nil
	}
	n, err := k.db.Model(BuiltinUserRoles).Ctx(ctx).Where("user_id", userID).Count()
	if err != nil || n > 0 {
		return err
	}
	roleRow, err := k.db.Model(BuiltinRoles).Ctx(ctx).Where("name", "user").One()
	if err != nil || roleRow.IsEmpty() {
		return err
	}
	_, err = k.db.Model(BuiltinUserRoles).Ctx(ctx).Insert(g.Map{
		"user_id": userID,
		"role_id": roleRow["id"].Int64(),
	})
	return err
}

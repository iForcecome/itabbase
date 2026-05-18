package schema

import "ksogit.kingsoft.net/wpsee/itabbase/server/internal/model"

// BuiltinCollections returns all kernel-managed collection definitions.
// The caller (Kernel) may attach hooks before registering them.
func BuiltinCollections() []model.Collection {
	return []model.Collection{
		builtinUsersCollection(),
		builtinRolesCollection(),
		builtinUserRolesCollection(),
		builtinSettingsCollection(),
		builtinMetaCollectionsCollection(),
		builtinMetaFieldsCollection(),
	}
}

func builtinUsersCollection() model.Collection {
	return model.Collection{
		Name:     model.BuiltinUsers,
		Display:  "用户",
		Source:   model.SourceBuiltin,
		Internal: true,
		Fields: []model.Field{
			{Name: "username", Type: model.TString, MaxLen: 64},
			{Name: "external_id", Type: model.TString, MaxLen: 128},
			{Name: "login_name", Type: model.TString, MaxLen: 128},
			{Name: "display_name", Type: model.TString, MaxLen: 200},
			{Name: "avatar", Type: model.TString, MaxLen: 500},
			{Name: "password_hash", Type: model.TString, MaxLen: 200},
			{Name: "email", Type: model.TString, MaxLen: 200},
			{Name: "phone", Type: model.TString, MaxLen: 32},
			{Name: "gender", Type: model.TString, MaxLen: 16},
			{Name: "employee_id", Type: model.TString, MaxLen: 64},
			{Name: "title", Type: model.TString, MaxLen: 128},
			{Name: "department", Type: model.TString, MaxLen: 256},
			{Name: "department_path", Type: model.TString, MaxLen: 500},
			{Name: "company_id", Type: model.TString, MaxLen: 128},
			{Name: "status", Type: model.TString, MaxLen: 20, Default: model.UserStatusActive},
			{Name: "disabled", Type: model.TBool, Default: false},
			{Name: "first_seen_at", Type: model.TDateTime},
			{Name: "last_seen_at", Type: model.TDateTime},
			{Name: "user_roles", Type: model.THasMany, Target: model.BuiltinUserRoles, Through: "user_id"},
		},
		ACL: model.ACL{
			"admin": {model.ActionAll},
			"user":  {model.ActionList, model.ActionGet},
		},
	}
}

func builtinSettingsCollection() model.Collection {
	return model.Collection{
		Name:     model.BuiltinSettings,
		Display:  "系统设置",
		Source:   model.SourceBuiltin,
		Internal: true,
		Fields: []model.Field{
			{Name: "key", Type: model.TString, Required: true, MaxLen: 64},
			{Name: "value", Type: model.TString, MaxLen: 500},
		},
		ACL: model.ACL{
			"admin": {model.ActionAll},
		},
	}
}

func builtinRolesCollection() model.Collection {
	return model.Collection{
		Name:     model.BuiltinRoles,
		Display:  "角色",
		Source:   model.SourceBuiltin,
		Internal: true,
		Fields: []model.Field{
			{Name: "name", Type: model.TString, Required: true, MaxLen: 64},
			{Name: "display", Type: model.TString, MaxLen: 200},
		},
		ACL: model.ACL{
			"admin": {model.ActionAll},
			"user":  {model.ActionList, model.ActionGet},
		},
	}
}

func builtinUserRolesCollection() model.Collection {
	return model.Collection{
		Name:     model.BuiltinUserRoles,
		Display:  "用户角色",
		Source:   model.SourceBuiltin,
		Internal: true,
		Fields: []model.Field{
			{Name: "user_id", Type: model.TBelongsTo, Target: model.BuiltinUsers, Required: true},
			{Name: "role_id", Type: model.TBelongsTo, Target: model.BuiltinRoles, Required: true},
		},
		ACL: model.ACL{
			"admin": {model.ActionAll},
			"user":  {model.ActionList, model.ActionGet},
		},
	}
}

func builtinMetaCollectionsCollection() model.Collection {
	return model.Collection{
		Name:     model.BuiltinMetaCollections,
		Display:  "集合定义",
		Source:   model.SourceBuiltin,
		Internal: true,
		Fields: []model.Field{
			{Name: "name", Type: model.TString, Required: true, MaxLen: 64},
			{Name: "display", Type: model.TString, MaxLen: 200},
			{Name: "icon", Type: model.TString, MaxLen: 64},
			{Name: "sort", Type: model.TInt, Default: 0},
		},
		ACL: model.ACL{"admin": {model.ActionAll}},
	}
}

func builtinMetaFieldsCollection() model.Collection {
	return model.Collection{
		Name:     model.BuiltinMetaFields,
		Display:  "字段定义",
		Source:   model.SourceBuiltin,
		Internal: true,
		Fields: []model.Field{
			{Name: "collection_name", Type: model.TString, Required: true, MaxLen: 64},
			{Name: "name", Type: model.TString, Required: true, MaxLen: 64},
			{Name: "type", Type: model.TString, Required: true, MaxLen: 32},
			{Name: "display", Type: model.TString, MaxLen: 200},
			{Name: "required", Type: model.TBool, Default: false},
			{Name: "default_value", Type: model.TString, MaxLen: 500},
			{Name: "max_len", Type: model.TInt, Default: 0},
			{Name: "target", Type: model.TString, MaxLen: 64},
			{Name: "through", Type: model.TString, MaxLen: 64},
			{Name: "sort", Type: model.TInt, Default: 0},
		},
		ACL: model.ACL{"admin": {model.ActionAll}},
	}
}

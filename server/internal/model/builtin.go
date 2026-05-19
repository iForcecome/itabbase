package model

// Builtin collection names.
const (
	BuiltinUsers           = "users"
	BuiltinRoles           = "roles"
	BuiltinUserRoles       = "user_roles"
	BuiltinSettings        = "system_settings"
	BuiltinMetaCollections = "collections"
	BuiltinMetaFields      = "fields"

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

const SessionKeyUserID = "itab_uid"

// Package itab is the v0 minimal kernel: collection registry + auto CRUD + schema sync.
//
// Usage:
//
//	k := itab.New(itab.WithDB(g.DB()))
//	k.RegisterCollection(itab.Collection{Name: "todos", Fields: []itab.Field{...}})
//	if err := k.Mount(group); err != nil { ... }
package itab

import (
	"ksogit.kingsoft.net/wpsee/itabbase/server/internal/auth"
	"ksogit.kingsoft.net/wpsee/itabbase/server/internal/handler"
	"ksogit.kingsoft.net/wpsee/itabbase/server/internal/model"
	"ksogit.kingsoft.net/wpsee/itabbase/server/internal/provider/wps365"
)

// --- Type aliases (public API unchanged) ---

type Collection = model.Collection
type Field = model.Field
type FieldType = model.FieldType
type ACL = model.ACL
type Hooks = model.Hooks
type HookFunc = model.HookFunc
type Record = model.Record
type User = model.User
type AuthAdapter = model.AuthAdapter
type OAuthProvider = model.OAuthProvider
type SSOConfig = model.SSOConfig
type OAuthToken = model.OAuthToken
type OAuthUserInfo = model.OAuthUserInfo
type Route = model.Route
type RouteACL = model.RouteACL
type Hook = model.HookFunc

// --- FieldType constants ---

const (
	TString    = model.TString
	TText      = model.TText
	TInt       = model.TInt
	TFloat     = model.TFloat
	TBool      = model.TBool
	TDateTime  = model.TDateTime
	TBelongsTo = model.TBelongsTo
	THasMany   = model.THasMany
)

// --- Action constants ---

const (
	ActionList   = model.ActionList
	ActionGet    = model.ActionGet
	ActionCreate = model.ActionCreate
	ActionUpdate = model.ActionUpdate
	ActionDelete = model.ActionDelete
	ActionAll    = model.ActionAll
)

// --- Builtin collection name constants ---

const (
	BuiltinUsers           = model.BuiltinUsers
	BuiltinRoles           = model.BuiltinRoles
	BuiltinUserRoles       = model.BuiltinUserRoles
	BuiltinSettings        = model.BuiltinSettings
	BuiltinMetaCollections = model.BuiltinMetaCollections
	BuiltinMetaFields      = model.BuiltinMetaFields

	SourceBuiltin = model.SourceBuiltin
	SourceCode    = model.SourceCode
	SourceDynamic = model.SourceDynamic
)

// --- User status constants ---

const (
	UserStatusActive   = model.UserStatusActive
	UserStatusPending  = model.UserStatusPending
	UserStatusRejected = model.UserStatusRejected
)

// --- Auth constants ---

const RoleAnonymous = model.RoleAnonymous

var ErrUnauthenticated = model.ErrUnauthenticated
var ErrBadCredentials = model.ErrBadCredentials

// --- Re-exported functions ---

var (
	NewRecord      = model.NewRecord
	RequireAuthed  = model.RequireAuthed
	RequireRole    = model.RequireRole
	AllowAnonymous = model.AllowAnonymous
	UserFromCtx    = model.UserFromCtx
	TxFromCtx      = model.TxFromCtx
	WithTxCtx      = model.WithTxCtx
	HashPassword   = auth.HashPassword
	VerifyPassword = auth.VerifyPassword
	CollectionMeta = handler.CollectionMeta
)

// WPS365 is the built-in OAuthProvider for WPS 365.
var WPS365 OAuthProvider = wps365.Provider

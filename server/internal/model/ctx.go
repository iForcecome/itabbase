package model

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
)

type ctxKey int

const (
	userCtxKey  ctxKey = 1
	rolesCtxKey ctxKey = 2
	txCtxKey    ctxKey = 3
)

func CtxWithUser(ctx context.Context, u User) context.Context {
	return context.WithValue(ctx, userCtxKey, u)
}

// UserFromCtx returns the authenticated user attached by the kernel's
// auth middleware. ok is false for anonymous requests.
func UserFromCtx(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(userCtxKey).(User)
	return u, ok
}

type CtxRoles struct {
	Roles  []string
	Authed bool
}

func CtxWithRoles(ctx context.Context, roles []string, authed bool) context.Context {
	return context.WithValue(ctx, rolesCtxKey, CtxRoles{Roles: roles, Authed: authed})
}

func RolesFromCtx(ctx context.Context) (roles []string, authed bool, ok bool) {
	cr, present := ctx.Value(rolesCtxKey).(CtxRoles)
	if !present {
		return nil, false, false
	}
	return cr.Roles, cr.Authed, true
}

// WithTxCtx attaches a transaction to ctx so kernel hooks can use it.
func WithTxCtx(ctx context.Context, tx gdb.TX) context.Context {
	return context.WithValue(ctx, txCtxKey, tx)
}

// TxFromCtx returns the transaction stashed by the kernel for the current
// Before* hook. Hooks that read or write the DB MUST use this so their
// operations participate in the same transaction as the surrounding write.
func TxFromCtx(ctx context.Context) (gdb.TX, bool) {
	tx, ok := ctx.Value(txCtxKey).(gdb.TX)
	return tx, ok
}

// OpError carries an HTTP status alongside a message back through gdb.Transaction.
type OpError struct {
	Status int
	Msg    string
}

func (e *OpError) Error() string { return e.Msg }

func UserErr(status int, msg string) *OpError {
	return &OpError{Status: status, Msg: msg}
}

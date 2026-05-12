package itab

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
)

const (
	rolesCtxKey ctxKey = 2
	txCtxKey    ctxKey = 3
)

type ctxRoles struct {
	roles  []string
	authed bool
}

func ctxWithRoles(ctx context.Context, roles []string, authed bool) context.Context {
	return context.WithValue(ctx, rolesCtxKey, ctxRoles{roles: roles, authed: authed})
}

func rolesFromCtx(ctx context.Context) (roles []string, authed bool, ok bool) {
	cr, present := ctx.Value(rolesCtxKey).(ctxRoles)
	if !present {
		return nil, false, false
	}
	return cr.roles, cr.authed, true
}

// WithTxCtx attaches a transaction to ctx so kernel hooks can use it.
func WithTxCtx(ctx context.Context, tx gdb.TX) context.Context {
	return context.WithValue(ctx, txCtxKey, tx)
}

// TxFromCtx returns the transaction stashed by the kernel for the current
// Before* hook. Hooks that read or write the DB MUST use this so their
// operations participate in the same transaction as the surrounding write.
//
// Outside of Before* hooks (e.g. After*, custom routes, list/get handlers),
// ok will be false.
func TxFromCtx(ctx context.Context) (gdb.TX, bool) {
	tx, ok := ctx.Value(txCtxKey).(gdb.TX)
	return tx, ok
}

// opError carries an HTTP status alongside a message back through gdb.Transaction.
// We use a pointer-receiver type so it survives `errors.As` if it ever gets wrapped.
type opError struct {
	Status int
	Msg    string
}

func (e *opError) Error() string { return e.Msg }

func userErr(status int, msg string) *opError {
	return &opError{Status: status, Msg: msg}
}

package itab

import (
	"context"
	"errors"
	"strconv"

	"github.com/gogf/gf/v2/database/gdb"
	"golang.org/x/crypto/bcrypt"
)

// ErrBadCredentials is returned by VerifyPassword for any auth failure —
// unknown username, wrong password, or disabled account. Callers must not
// expose which (avoid user enumeration).
var ErrBadCredentials = errors.New("itab: bad credentials")

const passwordBcryptCost = 12

// HashPassword bcrypts a plaintext password. Use when seeding the default
// super admin or rotating a stored hash.
func HashPassword(plain string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(plain), passwordBcryptCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

// VerifyPassword looks up a non-disabled local user by username and checks
// the bcrypt password hash. Returns the authenticated User on success or
// ErrBadCredentials otherwise.
func VerifyPassword(ctx context.Context, db gdb.DB, username, password string) (User, error) {
	if username == "" || password == "" {
		return User{}, ErrBadCredentials
	}
	row, err := db.Model(BuiltinUsers).Ctx(ctx).
		Where("username", username).
		Where("disabled", false).
		One()
	if err != nil || row.IsEmpty() {
		return User{}, ErrBadCredentials
	}
	hash := row["password_hash"].String()
	if hash == "" {
		return User{}, ErrBadCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return User{}, ErrBadCredentials
	}
	id := row["id"].Int64()
	return User{
		ID:      strconv.FormatInt(id, 10),
		LocalID: id,
		Name:    row["display_name"].String(),
	}, nil
}

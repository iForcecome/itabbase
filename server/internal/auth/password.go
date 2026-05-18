package auth

import (
	"context"
	"strconv"

	"github.com/gogf/gf/v2/database/gdb"
	"golang.org/x/crypto/bcrypt"

	"ksogit.kingsoft.net/wpsee/itabbase/server/internal/model"
)

const passwordBcryptCost = 12

// HashPassword bcrypts a plaintext password.
func HashPassword(plain string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(plain), passwordBcryptCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

// VerifyPassword looks up a non-disabled local user by username and checks
// the bcrypt password hash.
func VerifyPassword(ctx context.Context, db gdb.DB, username, password string) (model.User, error) {
	if username == "" || password == "" {
		return model.User{}, model.ErrBadCredentials
	}
	row, err := db.Model(model.BuiltinUsers).Ctx(ctx).
		Where("username", username).
		Where("disabled", false).
		One()
	if err != nil || row.IsEmpty() {
		return model.User{}, model.ErrBadCredentials
	}
	hash := row["password_hash"].String()
	if hash == "" {
		return model.User{}, model.ErrBadCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return model.User{}, model.ErrBadCredentials
	}
	id := row["id"].Int64()
	return model.User{
		ID:      strconv.FormatInt(id, 10),
		LocalID: id,
		Name:    row["display_name"].String(),
	}, nil
}

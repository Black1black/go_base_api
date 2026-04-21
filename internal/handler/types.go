package handler

import "ex_proj_go/internal/entity"

type (
	Users interface {
		GetByID(id int64) (*entity.User, error)
	}

	Authorization interface {
		RegistrationUser(name, email, hashedPassword string) error
		Login(userId int64, token string) error
		GetIDByEmail(email string, hashedPassword string) (int64, error)
		DeleteRefreshToken(id int64, token string) error
		ReplaceOldPassword(id int64, oldPassword, newPassword string) error
	}
)

package auth

type (
	Authorization interface {
		AddNewUser(name, email, hashedPassword string) error
		AddToken(userId int64, token string) error
		GetIDByEmail(email string, hashedPassword string) (int64, error)
		DeleteToken(id int64, token string) error
		ReplacePassword(id int64, oldPassword, newPassword string) error
	}
)

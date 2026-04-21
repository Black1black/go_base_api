package auth

import (
	"errors"
	"ex_proj_go/internal/models"

	"fmt"

	"github.com/ybru-tech/georm"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) AddNewUser(name, email, hashedPassword string) error {
	var user *models.Users

	// Проверка существует ли уже пользователь с данным email
	if err := r.db.Where("email = ?", email).First(&user).Error; err == nil {
		return errors.New("пользователь с таким email уже существует")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// Начало транзакции
	tx := r.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	// Создание нового пользователя
	user = &models.Users{Name: name, Email: &email, HashedPassword: hashedPassword, Status: "new"}
	if err := tx.Create(&user).Error; err != nil {
		return err
	}

	// Создание записи в таблице users_location
	userLocation := models.UsersLocation{
		UserID:   user.ID,
		Location: georm.Point{}, //georm.Point{srid: 4326}, // Добавьте логику установления координат
	}
	if err := tx.Create(&userLocation).Error; err != nil {
		return err
	}

	// Коммит транзакции
	return tx.Commit().Error
}

func (r *Repository) AddToken(userId int64, token string) error {
	authToken := models.AuthToken{
		UserID: int64(userId),
		Token:  token,
	}
	if err := r.db.Create(&authToken).Error; err != nil {
		return err
	}

	return nil
}

func (r *Repository) GetIDByEmail(email string, hashedPassword string) (int64, error) {
	var user *models.Users
	// Поиск пользователя по адресу электронной почты и хешированному паролю
	result := r.db.Where("email = ? AND hashed_password = ?", email, hashedPassword).First(&user)

	// Проверка на ошибки при выполнении запроса
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return 0, errors.New("пользователь не найден")
		}
		return 0, result.Error
	}

	return user.ID, nil
}

func (r *Repository) DeleteToken(userID int64, token string) error {
	result := r.db.Where("user_id = ? AND token = ?", userID, token).Delete(&models.AuthToken{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("no records found to delete")
	}

	return nil
}

func (r *Repository) ReplacePassword(id int64, oldPassword, newPassword string) error {
	result := r.db.Transaction(func(tx *gorm.DB) error {
		var user *models.Users

		// Проверяем пользователя и старый пароль
		if err := tx.Where("id = ? AND hashed_password = ?", id, oldPassword).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("user not found or wrong password")
			}
			return err
		}

		user.HashedPassword = newPassword
		if err := tx.Save(&user).Error; err != nil {
			return err
		}

		return nil
	})

	if result != nil {
		return result
	}

	return nil
}

package entity

import (
	"errors"
	"regexp"
)

type Tokens struct {
	accessToken  string
	refreshToken string
}

// SUsersAuth структура для данных авторизации
type SUsersAuth struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate проверяет, что хотя бы одно из полей (телефон или email) задано
func (s *SUsersAuth) Validate() error {
	if s.Email == "" {
		return errors.New("email must be defined")
	}

	if !isValidEmail(s.Email) {
		return errors.New("invalid email format")
	}

	return nil
}

// Пример проверки корректности email
func isValidEmail(email string) bool {
	// Простейшая регулярка для валидации email
	var re = regexp.MustCompile(`^[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,4}$`)
	return re.MatchString(email)
}

func Login(loginData *SUsersAuth) *Tokens {

}

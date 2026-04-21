package auth

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
)

type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func GetNewAuthTools(salt, signingKey string, accessTokenTTL, refreshTokenTTL time.Duration) *AuthTools {
	return &AuthTools{
		salt:            salt,
		signingKey:      signingKey,
		refreshTokenTTL: refreshTokenTTL,
		accessTokenTTL:  accessTokenTTL,
	}
}

type AuthTools struct {
	salt            string        // "dfkgjfdgeorigjei43435sfdfsd" Добавляем к паролю перед хэшированием
	signingKey      string        // "token_key"
	accessTokenTTL  time.Duration // 12 * time.Hour // Token Time-To-Live
	refreshTokenTTL time.Duration
}

type tokenClaims struct {
	jwt.StandardClaims
	UserId int64 `json:"user_id"`
}

func (a *AuthTools) GenerateToken(userId int64, tokenType string) (string, error) {
	var tokenTTL time.Duration

	switch tokenType {
	case "refresh":
		tokenTTL = a.refreshTokenTTL
	case "access":
		tokenTTL = a.accessTokenTTL
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &tokenClaims{
		jwt.StandardClaims{
			ExpiresAt: time.Now().Add(tokenTTL).Unix(), // На x часов больше текущего времени
			IssuedAt:  time.Now().Unix(),               // time of generate token
		},
		userId,
	})

	return token.SignedString([]byte(a.signingKey))

}

func (a *AuthTools) ParseToken(accessOrRefreshToken string) (int64, error) {
	token, err := jwt.ParseWithClaims(accessOrRefreshToken, &tokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(a.signingKey), nil
	})
	if err != nil {
		return 0, err
	}
	claims, ok := token.Claims.(*tokenClaims)
	if !ok || !token.Valid {
		return 0, errors.New("invalid or expired token")
	}
	return claims.UserId, nil
}

func (a *AuthTools) GeneratePasswordHash(password string) string {
	hash := sha1.New()
	hash.Write([]byte(password))

	return fmt.Sprintf("%x", hash.Sum([]byte(a.salt)))
}

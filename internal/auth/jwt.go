package auth

import (
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func MakeJWT(userId uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	// the second argument is the what our token will contain
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn).UTC()),
		Subject:   userId.String(),
	})
	tokenString, err := token.SignedString(tokenSecret)
	if err != nil {
		return "", err
	}
	log.Println(tokenString)
	return tokenString, nil
}

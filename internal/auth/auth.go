package users

import (
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func GetJWTToken(duration int, userId int, secret string) string {
	default24hours := 24 * 60 * 60
	if duration == 0 || duration > default24hours {
		duration = default24hours
	}
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.RegisteredClaims{
			Issuer:    "chirpy",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Second * time.Duration(duration))),
			Subject:   fmt.Sprintf("%d", userId),
		})
	stringToken, err := token.SignedString([]byte(secret))
	if err != nil {
		log.Fatalf("Fatal error when constructing JWT token: %v", err)
	}
	return stringToken
}

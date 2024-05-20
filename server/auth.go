package server

import (
	"os"

	"github.com/golang-jwt/jwt"
)

func getSecretKey() string {
	return os.Getenv("SERVER_SECRET_KEY")
}

type authJWTClaims struct {
	jwt.StandardClaims
	Email string `json:"email"`
}

func generateAccessToken(claims authJWTClaims) (string, error) {
	t := jwt.New(jwt.SigningMethodHS256)
	t.Claims = claims
	return t.SignedString([]byte(getSecretKey()))
}

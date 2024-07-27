package server

import (
	"net/http"
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

// Return the default headers to use to make queries against the server.
// This is a convenience function for worker clients that upload data.
func GetDefaultServerHeaders(authToken string) http.Header {
	h := http.Header{}
	h.Add("Authorization", authToken)
	h.Add("Content-Type", "application/json")
	return h
}

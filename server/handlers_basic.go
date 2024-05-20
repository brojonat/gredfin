package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/jackc/pgx/v5/pgxpool"
)

type defaultJSONResponse struct {
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

func writeInternalError(l *slog.Logger, w http.ResponseWriter, e error) {
	l.Error(e.Error())
	json.NewEncoder(w).Encode(defaultJSONResponse{Error: "internal error"})
}

// handlePing pings the database
func handlePing(l *slog.Logger, p *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := p.Ping(r.Context())
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		json.NewEncoder(w).Encode(defaultJSONResponse{Message: "PONG"})
	}
}

// handleGetToken returns a token
func handleIssueToken(l *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t := r.Header.Get("Authorization")
		if t == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(defaultJSONResponse{Error: "must supply authorization header"})
		}
		email := r.URL.Query().Get("email")
		if email == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(defaultJSONResponse{Error: "must supply email"})
		}
		if t != getSecretKey() {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(defaultJSONResponse{Error: "not authorized"})
			return
		}
		sc := jwt.StandardClaims{
			ExpiresAt: time.Now().Add(2 * 7 * 24 * time.Hour).Unix(),
		}
		c := authJWTClaims{
			StandardClaims: sc,
			Email:          email,
		}
		token, _ := generateAccessToken(c)
		l.Warn("issuing sudo token", "token", token)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(defaultJSONResponse{Message: token})
	}
}

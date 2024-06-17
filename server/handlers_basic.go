package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DefaultJSONResponse struct {
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

func writeInternalError(l *slog.Logger, w http.ResponseWriter, e error) {
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:]) // skip [Callers, Infof]
	r := slog.NewRecord(time.Now(), slog.LevelError, e.Error(), pcs[0])
	_ = l.Handler().Handle(context.Background(), r)
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(DefaultJSONResponse{Error: "internal error"})
}

func writeBadRequestError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusBadRequest)
	resp := DefaultJSONResponse{Error: err.Error()}
	json.NewEncoder(w).Encode(resp)
}

func writeEmptyResultError(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	resp := DefaultJSONResponse{Error: "empty result set"}
	json.NewEncoder(w).Encode(resp)
}

// handlePing pings the database
func handlePing(l *slog.Logger, p *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := p.Ping(r.Context())
		if err != nil {
			writeInternalError(l, w, err)
			return
		}
		json.NewEncoder(w).Encode(DefaultJSONResponse{Message: "PONG"})
	}
}

// handleGetToken returns a token
func handleIssueToken(l *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t := r.Header.Get("Authorization")
		if t == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(DefaultJSONResponse{Error: "must supply authorization header"})
			return
		}
		email := r.URL.Query().Get("email")
		if email == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(DefaultJSONResponse{Error: "must supply email"})
			return
		}
		if t != getSecretKey() {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(DefaultJSONResponse{Error: "not authorized"})
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
		json.NewEncoder(w).Encode(DefaultJSONResponse{Message: token})
	}
}

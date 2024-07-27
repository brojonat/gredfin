package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"firebase.google.com/go/auth"
	"github.com/golang-jwt/jwt"
	"github.com/gorilla/handlers"
)

const FirebaseJWTHeader = "Firebase-JWT"

type contextKey int

var jwtCtxKey contextKey = 1

type handlerAdapter func(http.HandlerFunc) http.HandlerFunc

// AdaptHandler will wrap h with the supplied middleware; note that the
// middleware will be evaluated in the order they are supplied
func adaptHandler(h http.HandlerFunc, opts ...handlerAdapter) http.HandlerFunc {
	for i := range opts {
		opt := opts[len(opts)-1-i]
		h = opt(h)
	}
	return h
}

// Convenience middleware that applies commonly used middleware to the wrapped
// handler. This will make the handler gracefully handle panics, sets the
// content type to application/json, limits the body size that clients can send,
// wraps the handler with the usual CORS settings.
func apiMode(l *slog.Logger, maxBytes int64, headers, methods, origins []string) handlerAdapter {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			next = makeGraceful(l)(next)
			next = setMaxBytesReader(maxBytes)(next)
			next = setContentType("application/json")(next)
			handlers.CORS(
				handlers.AllowedHeaders(headers),
				handlers.AllowedMethods(methods),
				handlers.AllowedOrigins(origins),
			)(next).ServeHTTP(w, r)
		}
	}
}

func setContentType(content string) handlerAdapter {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", content)
			next(w, r)
		}
	}
}

func makeGraceful(l *slog.Logger) handlerAdapter {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				err := recover()
				if err != nil {
					l.Error("recovered from panic")
					switch v := err.(type) {
					case error:
						writeInternalError(l, w, v)
					case string:
						writeInternalError(l, w, fmt.Errorf(v))
					default:
						writeInternalError(l, w, fmt.Errorf("recovered but unexpected type from recover()"))
					}
				}
			}()
			next.ServeHTTP(w, r)
		}
	}
}

func setMaxBytesReader(mb int64) handlerAdapter {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, mb)
			next(w, r)
		}
	}
}

func bearerAuthorizer() func(*http.Request) bool {
	return func(r *http.Request) bool {
		var claims authJWTClaims
		ts := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if ts == "" {
			return false
		}
		kf := func(token *jwt.Token) (interface{}, error) {
			return []byte(getSecretKey()), nil
		}
		token, err := jwt.ParseWithClaims(ts, &claims, kf)
		if err != nil || !token.Valid {
			return false
		}
		// FIXME: verify this actually works and then do something similar in
		// the firebase authorizer since right now they don't have parity
		ctx := context.WithValue(r.Context(), jwtCtxKey, token.Claims)
		*r = *r.WithContext(ctx)
		return true
	}
}

// Uses Firebase-JWT header and firebase client to auth
func firebaseAuthorizer(hname string, fbc *auth.Client) func(*http.Request) bool {
	return func(r *http.Request) bool {
		if _, err := fbc.VerifyIDToken(r.Context(), r.Header.Get(hname)); err != nil {
			return false
		}
		return true
	}
}

// Iterates over the supplied authorizers and if at least one passes, then the
// next handler is called, otherwise an unauthorized response is written.
func atLeastOneAuth(authorizers ...func(*http.Request) bool) handlerAdapter {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			for _, a := range authorizers {
				if !a(r) {
					continue
				}
				next(w, r)
				return
			}
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(DefaultJSONResponse{Error: "unauthorized"})
		}
	}
}

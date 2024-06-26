package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt"
	"github.com/gorilla/handlers"
)

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

func mustAuth() handlerAdapter {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var claims authJWTClaims
			ts := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if ts == "" {
				resp := DefaultJSONResponse{Error: "missing authorization header"}
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(resp)
				return
			}
			kf := func(token *jwt.Token) (interface{}, error) {
				return []byte(getSecretKey()), nil
			}
			token, err := jwt.ParseWithClaims(ts, &claims, kf)
			if err != nil || !token.Valid {
				// this can happen for all sorts of typical reasons (expired tokens, etc.)
				// so nothing is logged and the user just gets a generic unauthorized message
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(DefaultJSONResponse{Error: "bad token value"})
				return
			}
			ctx := context.WithValue(r.Context(), jwtCtxKey, token.Claims)
			r = r.WithContext(ctx)
			next(w, r)
		}
	}
}

package handlers

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	PasswordHash string `json:"password_hash"`
	jwt.RegisteredClaims
}

func HandleSignIn(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Некорректный запрос"}`, http.StatusBadRequest)
		return
	}

	storedPassword := os.Getenv("TODO_PASSWORD")
	if storedPassword == "" {
		http.Error(w, `{"error":"Аутентификация отключена"}`, http.StatusUnauthorized)
		return
	}

	if req.Password != storedPassword {
		http.Error(w, `{"error":"Неверный пароль"}`, http.StatusUnauthorized)
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		PasswordHash: hashPassword(storedPassword),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(8 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	})
	secretKey := []byte("your_secret_key")
	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		http.Error(w, `{"error":"Ошибка генерации токена"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token": tokenString,
	})
}

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		storedPassword := os.Getenv("TODO_PASSWORD")
		if storedPassword == "" {
			next(w, r)
			return
		}

		cookie, err := r.Cookie("token")
		if err != nil {
			http.Error(w, `{"error":"Необходима аутентификация"}`, http.StatusUnauthorized)
			return
		}

		tokenString := cookie.Value
		secretKey := []byte("your_secret_key")

		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return secretKey, nil
		})
		if err != nil || !token.Valid {
			http.Error(w, `{"error":"Неверный токен"}`, http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(*Claims)
		if !ok || claims.PasswordHash != hashPassword(storedPassword) {
			http.Error(w, `{"error":"Неверный токен"}`, http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

func hashPassword(password string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(password)))
}

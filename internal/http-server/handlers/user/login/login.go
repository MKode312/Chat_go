package login_handler

import (
	val "chat_go/internal/lib/api/validation"
	"chat_go/internal/lib/logger/sl"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-playground/validator/v10"
)

type Request struct {
	Username string `json:"Username" validate:"required"`
	Password string `json:"Password" validate:"required"`
}

type UserLoginer interface {
	LoginUser(username, password string) (string, error)
}

func NewLoginHandler(log *slog.Logger, userInteractor UserLoginer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			log.Error("error decoding request", sl.Err(err))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			validateErr := err.(validator.ValidationErrors)
			log.Error("invalid request", sl.Err(err))
			http.Error(w, val.ValidationError(validateErr), http.StatusBadRequest)
			return
		}

		token, err := userInteractor.LoginUser(req.Username, req.Password)
		if err != nil {
			http.Error(w, "Invalid login or password", http.StatusUnauthorized)
			log.Error("invalid login or password", sl.Err(err))
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "auth_token",
			Value:    token,
			Path:     "/",
			SameSite: http.SameSiteNoneMode,
			Secure:   true,
		})

		http.SetCookie(w, &http.Cookie{
			Name:     "your_username",
			Value:    req.Username,
			Path:     "/",
			SameSite: http.SameSiteNoneMode,
			Secure:   true,
		})

		response := map[string]string{"token": token}
		json.NewEncoder(w).Encode(response)

		log.Info("success Login")
	}
}

package profile_handler

import (
	"chat_go/internal/lib/api/models"
	"chat_go/internal/lib/logger/sl"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
)

type Response struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type UserGetter interface {
	GetUser(username string) (models.User, error)
}

func NewGetUserHandler(log *slog.Logger, userGetter UserGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pathValues := strings.Split(r.URL.Path, "/")
		if len(pathValues) < 3 || pathValues[2] == "" {
			http.Error(w, "Invalid URL format", http.StatusBadRequest)
			return
		}
		username := pathValues[2]
		firstLetter := []rune(username)[0]
		letter := string(firstLetter)
		if letter != "@" {
			http.Error(w, "Username must start with @", http.StatusBadRequest)
		}

		user, err := userGetter.GetUser(username)
		if err != nil {
			http.Error(w, "Invalid username", http.StatusBadRequest)
			log.Error("invalid username", sl.Err(err))
			return
		}

		response := map[string]interface{}{"user": user}
		json.NewEncoder(w).Encode(response)

		log.Info("success GetUser")
	}
}

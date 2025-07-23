package save_handler

import (
	val "chat_go/internal/lib/api/validation"
	"chat_go/internal/lib/logger/sl"
	"chat_go/internal/storage"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type Request struct {
	Password string `json:"Password" validate:"required"`
	Nickname string `json:"Nickname" validate:"required"`
	Username string `json:"Username" validate:"required"`
	Bio      string `json:"Bio" validate:"required"`
}

type UserSaver interface {
	SaveUser(bio string, password string, nickname string, username string) (int64, error)
}

func NewSaveHandler(log *slog.Logger, userSaver UserSaver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.user.save.New"

		if r.Method != http.MethodPost {
			http.Error(w, "Wrong method", http.StatusMethodNotAllowed)
		}

		log = log.With(slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req Request

		err := render.DecodeJSON(r.Body, &req)
		if err != nil {
			log.Error("failed to decode request body", sl.Err(err))
			http.Error(w, "Failed to decode request body", http.StatusInternalServerError)
			return
		}

		log.Info("request bosy decoded", slog.Any("request", req))

		if err := validator.New().Struct(req); err != nil {
			validateErr := err.(validator.ValidationErrors)
			log.Error("invalid request", sl.Err(err))
			http.Error(w, val.ValidationError(validateErr), http.StatusBadRequest)
			return
		}

		username := req.Username
		firstLetter := []rune(username)[0]
		letter := string(firstLetter)

		if letter != "@" {
			http.Error(w, "Username must start with @", http.StatusBadRequest)
			return
		}

		id, err := userSaver.SaveUser(req.Bio, req.Password, req.Nickname, username)
		if errors.Is(err, storage.ErrUserAlreadyExists) {
			log.Info("user already exists", slog.String("user", req.Nickname))
			http.Error(w, "User already exists", http.StatusBadRequest)
			return
		}
		if err != nil {
			log.Error("failed to add user", sl.Err(err))
			http.Error(w, "Failed to add user", http.StatusInternalServerError)
			return
		}

		log.Info("user added", slog.Int64("id", id))
		json.NewEncoder(w).Encode("You have successfully created a profile!")
		w.WriteHeader(http.StatusCreated)
	}
}

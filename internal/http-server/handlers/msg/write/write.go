package write

import (
	val "chat_go/internal/lib/api/validation"
	"chat_go/internal/lib/logger/sl"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type Request struct {
	ChatName string `json:"ChatName" validate:"required"`
	ID       int64  `json:"ID" validate:"required"`
	Text     string `json:"Text" validate:"required"`
}

type MessagesInteractor interface {
	SaveMessage(sender string, chatName string, chatID int64, text string) (int64, error)
	GetParticipantsByChatNameAndID(chatName string, id int64) (string, error)
}

func NewWriteMessagesHandler(log *slog.Logger, messageInteractor MessagesInteractor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.msg.Write"

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

		cookie, err := r.Cookie("your_username")
		if err != nil {
			log.Error("failed checking cookie", sl.Err(err))
			http.Error(w, "Failed checking your cookie", http.StatusInternalServerError)
			return
		}

		sender := cookie.Value

		participants, err := messageInteractor.GetParticipantsByChatNameAndID(req.ChatName, req.ID)
		if err != nil {
			log.Error("failed validating your participation in this chat", sl.Err(err))
			http.Error(w, "Failed validating your participation in this chat", http.StatusBadRequest)
			return
		}

		if !strings.Contains(participants, sender) {
			log.Warn("You are not in this chat")
			http.Error(w, "You are not in this chat", http.StatusForbidden)
			return
		}

		id, err := messageInteractor.SaveMessage(sender, req.ChatName, req.ID, req.Text)
		if err != nil {
			log.Error("failed to write a message", sl.Err(err))
			http.Error(w, "Failed to write a message", http.StatusInternalServerError)
			return
		}
		log.Info("message added", slog.Int64("id", id))
		w.Write([]byte("You have successfully written a message!"))
		w.WriteHeader(http.StatusOK)

	}
}

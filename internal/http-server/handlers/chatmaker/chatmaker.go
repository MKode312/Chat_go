package chatmaker_handler

import (
	"chat_go/internal/lib/api/models"
	val "chat_go/internal/lib/api/validation"
	"chat_go/internal/lib/logger/sl"
	"chat_go/internal/storage"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type Request struct {
	Name         string `json:"Name" validate:"required"`
	Participants string `json:"Participants(usernames)" validate:"required"`
}

type ResponseMessages struct {
	ID     int64  `json:"id"`
	Sender string `json:"sender"`
	Text   string `json:"text"`
}

type ResponseData struct {
	Name         string             `json:"name"`
	Participants string             `json:"participants"`
	RespMsg      []ResponseMessages `json:"messages"`
}

type ChatInteractor interface {
	MakeChat(name, username string) (int64, error)
	GetParticipantsByChatNameAndID(chatName string, id int64) (string, error)
	GetSenderOfMessageByChatName(chatName string) (string, error)
	GetAllMessagesByChatnameAndID(chatName string, id int64) ([]models.Message, error)
	GetNicknameByUsername(username string) (string, error)
}

func NewChatmakerHandler(log *slog.Logger, ChatInteractor ChatInteractor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.chatmaker.Chatmaker"

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
			return
		}

		log.Info("request bosy decoded", slog.Any("request", req))

		if err := validator.New().Struct(req); err != nil {
			validateErr := err.(validator.ValidationErrors)
			log.Error("invalid request", sl.Err(err))
			http.Error(w, val.ValidationError(validateErr), http.StatusBadRequest)
			return
		}

		users := strings.ReplaceAll(req.Participants, " ", "")
		listOfUsers := strings.Split(users, ",")
		for _, user := range listOfUsers {
			req, err := http.NewRequest("GET", "http://localhost:8083/chat/"+user, nil)
			if err != nil {
				log.Error("wrong url", sl.Err(err))
				http.Error(w, "invalid url", http.StatusBadRequest)
				return
			}
			cookie, err := r.Cookie("auth_token")
			if err != nil {
				log.Error("failed checking cookie", sl.Err(err))
				http.Error(w, "failed checking your cookie", http.StatusInternalServerError)
				return
			}
			tokenString := cookie.Value
			req.AddCookie(&http.Cookie{
				Name:     "auth_token",
				Value:    tokenString,
				Path:     "/",
				SameSite: http.SameSiteNoneMode,
				Secure:   true,
			})
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				log.Error("failed to create a response while checking users", sl.Err(err))
				http.Error(w, "failed to create a response while checking users", http.StatusInternalServerError)
				return
			}
			if resp.StatusCode != http.StatusOK {
				http.Error(w, "there are some non-existing users", http.StatusBadRequest)
				return
			}
		}
		if len(users) == 0 {
			log.Error("no users in the chat")
			http.Error(w, "no users in the chat", http.StatusConflict)
			return
		} else {
			id, err := ChatInteractor.MakeChat(req.Name, req.Participants)
			if errors.Is(err, storage.ErrChatAlreadyExists) {
				log.Info("chat already exists", slog.String("chat", req.Name))
				http.Error(w, "chat already exists", http.StatusBadRequest)
				return
			}
			if err != nil {
				log.Error("failed to make chat", sl.Err(err))
				http.Error(w, "Failed to make chat", http.StatusInternalServerError)
				return
			}
			log.Info("chat added", slog.Int64("id", id))

			response1 := map[string]string{"You have successfully created a chat with this name:": req.Name}
			json.NewEncoder(w).Encode(response1)

			response2 := map[string]int64{"Here is your chat`s ID:": id}
			json.NewEncoder(w).Encode(response2)

			w.WriteHeader(http.StatusCreated)
		}
	}
}

func NewGetChatHandler(log *slog.Logger, chatInteractor ChatInteractor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.chatmaker.GetChat"

		if r.Method != http.MethodGet {
			http.Error(w, "Wrong method", http.StatusMethodNotAllowed)
		}

		log = log.With(slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		pathValues := strings.Split(r.URL.Path, "/")
		if len(pathValues) < 3 || pathValues[2] == "" {
			http.Error(w, "Invalid URL format", http.StatusBadRequest)
			return
		}
		ChatName := pathValues[2]
		ID := pathValues[3]

		id, err := strconv.Atoi(ID)
		if err != nil {
			log.Error("failed to convert ID", sl.Err(err))
			http.Error(w, "Failed to convert ID", http.StatusInternalServerError)
			return
		}

		participants, err := chatInteractor.GetParticipantsByChatNameAndID(ChatName, int64(id))
		if err != nil {
			log.Error("failed to get participants of this chat", sl.Err(err))
			http.Error(w, "Failed to get participants of this chat", http.StatusBadRequest)
			return
		}

		cookie, err := r.Cookie("your_username")
		if err != nil {
			log.Error("failed checking cookie", sl.Err(err))
			http.Error(w, "Failed checking your cookie", http.StatusInternalServerError)
			return
		}

		sender := cookie.Value

		if !strings.Contains(participants, sender) {
			log.Warn("You are not in this chat")
			http.Error(w, "You are not in this chat", http.StatusForbidden)
			return
		}

		participants = strings.ReplaceAll(participants, ", ", " ")

		usernames := strings.Fields(participants)
		if len(usernames) <= 0 {
			http.Error(w, "Invalid chat", http.StatusBadRequest)
			return
		}

		var ParticipantsNicknames []string

		for _, username := range usernames {
			nickname, err := chatInteractor.GetNicknameByUsername(username)
			if err != nil {
				log.Error("failed to get nickname", sl.Err(err))
				return
			}
			ParticipantsNicknames = append(ParticipantsNicknames, nickname)
		}

		Participants := strings.Join(ParticipantsNicknames, ", ")

		messages, err := chatInteractor.GetAllMessagesByChatnameAndID(ChatName, int64(id))
		if err != nil {
			log.Error("failed to get the list of messages in this chat", sl.Err(err))
			http.Error(w, "Failed to get the list of messages in this chat", http.StatusBadRequest)
			return
		}

		var respMsg []ResponseMessages

		for _, msg := range messages {
			resp := ResponseMessages{
				ID:     msg.ID,
				Sender: msg.Sender,
				Text:   msg.Text,
			}
			respMsg = append(respMsg, resp)
		}
		respData := ResponseData{
			Name:         ChatName,
			Participants: Participants,
			RespMsg:      respMsg,
		}
		json.NewEncoder(w).Encode(respData)
		log.Info("successful GetChatOperation")
	}
}

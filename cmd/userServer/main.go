package main

import (
	user_config "chat_go/internal/config/user"
	login_handler "chat_go/internal/http-server/handlers/user/login"
	profile_handler "chat_go/internal/http-server/handlers/user/profile"
	"chat_go/internal/http-server/handlers/user/save"
	authorization_middleware "chat_go/internal/http-server/middlewares/authorization"
	mwLogger "chat_go/internal/http-server/middlewares/logger"
	"chat_go/internal/lib/logger/handlers/slogpretty"
	"chat_go/internal/lib/logger/sl"
	"chat_go/internal/storage/sqlite"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {

	cfg := user_config.MustLoad()

	log := setupLogger(cfg.Env)

	log.Info("user server enabled on: " + cfg.Address)

	storage, err := sqlite.New(cfg.StoragePath)
	if err != nil {
		log.Error("failed to init storage", sl.Err(err))
		os.Exit(1)
	}


	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(mwLogger.New(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Post("/chat/register", save_handler.NewSaveHandler(log, storage))
	router.Post("/chat/login", login_handler.NewLoginHandler(log, storage))

	router.Group(func(r chi.Router) {
		r.Use(authorization_middleware.AuthorizeJWTToken)

		r.Get("/chat/{username}", profile_handler.NewGetUserHandler(log, storage))
	})

	srv := &http.Server{
		Addr: cfg.Address,
		Handler: router,
		ReadTimeout: cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout: cfg.HTTPServer.IdleTimeout,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Error("failed to start server")
	}
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = setupPrettySlog()
	case envDev:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envProd:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	return log
}

func setupPrettySlog() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	handler := opts.NewPrettyHandler(os.Stdout)

	return slog.New(handler)
}

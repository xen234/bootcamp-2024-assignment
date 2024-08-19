package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"

	"github.com/xen234/bootcamp-2024-assignment/api"
	"github.com/xen234/bootcamp-2024-assignment/internal/config"
	"github.com/xen234/bootcamp-2024-assignment/internal/handlers"
	"github.com/xen234/bootcamp-2024-assignment/internal/storage/sqlite"
	"github.com/xen234/bootcamp-2024-assignment/logger/sl"
)

const (
	envLocal = "local"
	envDev   = "dev"
)

func setupLogging(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}

func main() {
	conf := config.MustLoad()

	log := setupLogging(conf.Env)

	slog.SetDefault(log)

	log.Info("Starting server", slog.String("env", conf.Env))
	log.Debug("Debugging info enabled")

	cwd, err := os.Getwd()
	if err != nil {
		log.Error("Ошибка при получении текущей рабочей директории", sl.Err(err))
	}

	log.Info("Storage path", slog.String("path", conf.StoragePath))
	log.Info("Текущая рабочая директория:", slog.String("cwd", cwd))

	storage, err := sqlite.New("./storage.db")
	if err != nil {
		log.Error("Failed to initialize storage", sl.Err(err))
		os.Exit(1)
	}

	_ = storage

	log.Info("Storage initialized", slog.String("env", "dev"))

	r := chi.NewRouter()

	myServer := &handlers.MyServer{
		Storage: storage,
	}

	apiHandler := api.HandlerFromMux(myServer, r)

	log.Info("Starting server on :8081")
	if err := http.ListenAndServe(":8081", apiHandler); err != nil {
		log.Error("Failed to start server: %v", sl.Err(err))
		os.Exit(1)
	}

}

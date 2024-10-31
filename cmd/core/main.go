package main

import (
	"net/http"

	"github.com/boskuv/goreminder/internal/api"
	"github.com/boskuv/goreminder/internal/config"
	"github.com/boskuv/goreminder/internal/logger"
	"github.com/boskuv/goreminder/internal/metrics"
	"github.com/boskuv/goreminder/internal/repository"
	"github.com/boskuv/goreminder/internal/service"
	"github.com/gorilla/mux"
)

func main() {
	// Инициализация конфигурации
	cfg := config.LoadConfig()

	// Инициализация логгера
	log := logger.New(cfg)

	// Инициализация базы данных
	db, err := repository.NewDB(cfg)
	if err != nil {
		log.Fatal("Ошибка при подключении к БД:", err)
	}

	// Инициализация Prometheus метрик
	metrics.InitMetrics()

	// Инициализация сервисов
	taskRepo := repository.NewTaskRepository(db)
	userRepo := repository.NewUserRepository(db)
	taskService := service.NewTaskService(taskRepo)
	userService := service.NewUserService(userRepo)

	// Настройка маршрутов
	r := mux.NewRouter()
	api.SetupRoutes(r, taskService, userService)

	// Запуск сервера
	log.Info("Запуск Core Service на порту:", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatal("Ошибка запуска сервера:", err)
	}
}

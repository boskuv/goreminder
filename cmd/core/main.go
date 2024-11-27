package main

import (
	"log"
	"os"

	"github.com/boskuv/goreminder/internal/repository"
	"github.com/boskuv/goreminder/internal/service"
	"github.com/boskuv/goreminder/pkg/args"
	"github.com/boskuv/goreminder/pkg/config"
	"github.com/boskuv/goreminder/pkg/logger"
	"github.com/rs/zerolog"
)

func main() {
	// parse command-line arguments
	parsedArgs := args.ParseArgs()

	// setup configuration
	config.Setup(parsedArgs.ConfigPath)

	cfg := config.GetConfig()

	// determine minimum logging level based on flag input
	var minlvl zerolog.Level
	minlvl, _ = zerolog.ParseLevel("debug") // TODO: from cfg
	//if err != nil {
	//return errs.E(op, err)
	//}

	// setup logger with appropriate defaults
	logger := logger.New(os.Stdout, minlvl, true)

	logger.Info().Msg("Graceful startup")

	// Инициализация Prometheus метрик
	// metrics.InitMetrics()

	// Инициализация базы данных
	dbConfig := &repository.DBConfig{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.Username,
		Password: cfg.Database.Password,
		DbName:   cfg.Database.Dbname,
		SSLMode:  "disable", // For local development
	}

	db, err := repository.NewDB(dbConfig)
	if err != nil {
		log.Fatal("Ошибка при подключении к БД:", err)
	}

	// setup repositories
	taskRepo := repository.NewTaskRepository(db)
	userRepo := repository.NewUserRepository(db)

	service.NewTaskService(*taskRepo, *userRepo) // pointers?
	// userService := service.NewUserService(userRepo)

	// // Настройка маршрутов
	// r := mux.NewRouter()
	// api.SetupRoutes(r, taskService, userService)

	// // Запуск сервера
	// log.Info("Запуск Core Service на порту:", cfg.Port)
	// if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
	// 	log.Fatal("Ошибка запуска сервера:", err)
	// }
}

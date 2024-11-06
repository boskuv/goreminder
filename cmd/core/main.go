package main

import (
	"os"

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

	// determine minimum logging level based on flag input
	var minlvl zerolog.Level
	minlvl, _ = zerolog.ParseLevel("debug") // TODO: from cfg
	//if err != nil {
	//return errs.E(op, err)
	//}

	// setup logger with appropriate defaults
	logger := logger.New(os.Stdout, minlvl, true)

	logger.Info().Msg("Graceful startup")

	// // Инициализация базы данных
	// db, err := repository.NewDB(cfg)
	// if err != nil {
	// 	log.Fatal("Ошибка при подключении к БД:", err)
	// }

	// Инициализация Prometheus метрик
	// metrics.InitMetrics()

	// // Инициализация сервисов
	// taskRepo := repository.NewTaskRepository(db)
	// userRepo := repository.NewUserRepository(db)
	// taskService := service.NewTaskService(taskRepo)
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

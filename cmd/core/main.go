package main

import (
	"github.com/boskuv/goreminder/pkg/args"
	"github.com/boskuv/goreminder/pkg/config"
)

func main() {
	// Parse command-line arguments
	parsedArgs := args.ParseArgs()

	// Setup configuration
	config.Setup(parsedArgs.ConfigPath)

	// Инициализация логгера
	//log := logger.New(cfg)

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

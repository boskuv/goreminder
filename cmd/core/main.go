package main

import (
	"log"
	"os"

	"github.com/boskuv/goreminder/internal/repository"
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
		Name:     cfg.Database.Dbname,
		SSLMode:  "disable", // For local development
	}

	db, err := repository.NewDB(dbConfig)
	if err != nil {
		log.Fatal("Ошибка при подключении к БД:", err)
	}

	// setup repositories
	taskRepo := repository.NewTaskRepository(db)
	userRepo := repository.NewUserRepository(db)

	// testing repos

	// 1. ensure the user exists
	user := &repository.User{
		Name:         "John Doe",
		Email:        "johndoe@example.com",
		PasswordHash: "hashed_password",
	}

	userID, err := userRepo.CreateUser(user)
	if err != nil {
		log.Fatal("Ошибка при создании пользователя:", err)
	}

	// 2. create task
	task := &repository.Task{
		Title:       "Learn Go",
		Description: "Read and practice Go programming",
		UserID:      userID,
		DueDate:     "2024-11-01",
		Status:      "pending",
	}

	taskID, err := taskRepo.CreateTask(task)
	if err != nil {
		log.Fatal("Ошибка при создании задачи:", err)
	}
	log.Printf("Задача создана с ID: %d\n", taskID)

	// 3. fetch task
	fetchedTask, err := taskRepo.GetTaskByID(taskID)
	if err != nil {
		log.Fatal("Ошибка при получении задачи:", err)
	}
	log.Printf("Полученная задача: %+v\n", fetchedTask)

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

package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	swaggerFiles "github.com/swaggo/files" // Swagger embedded files
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/boskuv/goreminder/docs"
	_ "github.com/boskuv/goreminder/docs" // Import generated Swagger docs
	"github.com/boskuv/goreminder/internal/api/handlers"
	_ "github.com/boskuv/goreminder/internal/api/handlers"
	"github.com/boskuv/goreminder/internal/api/routes"
	"github.com/boskuv/goreminder/internal/repository"
	"github.com/boskuv/goreminder/internal/service"
	"github.com/boskuv/goreminder/pkg/args"
	"github.com/boskuv/goreminder/pkg/config"
	"github.com/boskuv/goreminder/pkg/logger"
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

	taskService := service.NewTaskService(*taskRepo, *userRepo) // pointers?
	userService := service.NewUserService(*userRepo)

	// Initialize handlers
	taskHandler := handlers.NewTaskHandler(taskService)
	userHandler := handlers.NewUserHandler(userService)

	// Setup Swagger info
	docs.SwaggerInfo.Title = "Task Management API"
	docs.SwaggerInfo.Description = "API documentation for the Task Management system"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = "localhost:8080" // TODO: remove hardcode
	docs.SwaggerInfo.Schemes = []string{"http"}

	// Setup router
	router := gin.Default()

	// Register Swagger handler
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Register application routes
	routes.RegisterRoutes(router, taskHandler, userHandler)

	// Start server
	port := cfg.Server.Port
	if port == "" {
		port = "8080"
	}
	log.Printf("Starting server on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}

}

package main

import (
	"context"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	swaggerFiles "github.com/swaggo/files" // Swagger embedded files
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/boskuv/goreminder/docs"
	_ "github.com/boskuv/goreminder/docs" // Import generated Swagger docs
	"github.com/boskuv/goreminder/internal/api/handlers"
	_ "github.com/boskuv/goreminder/internal/api/handlers"
	"github.com/boskuv/goreminder/internal/api/middleware"
	"github.com/boskuv/goreminder/internal/api/routes"
	"github.com/boskuv/goreminder/internal/repository"
	"github.com/boskuv/goreminder/internal/service"
	"github.com/boskuv/goreminder/pkg/args"
	"github.com/boskuv/goreminder/pkg/config"
	"github.com/boskuv/goreminder/pkg/logger"
	"github.com/boskuv/goreminder/pkg/observability"
	"github.com/boskuv/goreminder/pkg/queue"
)

func main() {
	// parse command-line arguments
	parsedArgs := args.ParseArgs()

	// setup configuration
	err := config.Setup(parsedArgs.ConfigPath)

	if err != nil {
		panic("unable to setup configuration")
	}

	cfg := config.GetConfig()

	// determine minimum logging level based on flag input
	var minlvl zerolog.Level
	minlvl, _ = zerolog.ParseLevel("debug") // TODO: from cfg
	//if err != nil {
	//return errs.E(op, err)
	//}

	// setup logger with appropriate defaults
	log := logger.New(os.Stdout, minlvl, true)
	logger.LogErrorStackViaPkgErrors(true)

	if cfg.Metrics.Enabled {
		observability.StartMetricsServer(cfg.Metrics.Addr)
	}

	if cfg.Tracing.Enabled {
		tracer, _ := observability.InitTracer(cfg.Tracing.ServiceName, cfg.Tracing.Endpoint, cfg.Tracing.Insecure)
		defer tracer.Shutdown(context.Background())
	}

	// DB init
	// TODO: move to config validation
	maxLifetime, err := time.ParseDuration(cfg.Database.ConnMaxLifetime)
	if err != nil {
		//return fmt.Errorf("error while parsing db max_lifetime: %w", err)
	}

	dbConfig := &repository.DBConfig{
		Host:         cfg.Database.Host,
		Port:         cfg.Database.Port,
		User:         cfg.Database.Username,
		Password:     cfg.Database.Password,
		DbName:       cfg.Database.Dbname,
		SSLMode:      "disable", // For local development
		MaxOpenConns: cfg.Database.MaxOpenConns,
		MaxIdleConns: cfg.Database.MaxIdleConns,
		MaxLifetime:  maxLifetime,
		MaxRetries:   cfg.Database.MaxRetries,
	}

	db, err := repository.NewDB(dbConfig)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("error while connecting to DB")
	}

	// producer init
	producerConfig := &queue.ProducerConfig{
		Host:      cfg.Producer.Host,
		Port:      cfg.Producer.Port,
		User:      cfg.Producer.User,
		Password:  cfg.Producer.Password,
		QueueName: cfg.Producer.QueueName,
		Exchange:  cfg.Producer.Exchange,
	}

	producer, err := queue.NewProducer(producerConfig)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("error while connecting to producer")
	}
	defer func() {
		if err := producer.Close(); err != nil {
			log.Error().Stack().Err(err).Msg("failed to close producer")
		} else {
			log.Info().Msg("producer is closed gracefully")
		}
	}()
	defer producer.Close()

	// setup repositories
	taskRepo := repository.NewTaskRepository(db)
	userRepo := repository.NewUserRepository(db)
	messengerRepo := repository.NewMessengerRepository(db)

	// setup services
	taskService := service.NewTaskService(taskRepo, userRepo, messengerRepo, producer)
	userService := service.NewUserService(userRepo)
	messengerService := service.NewMessengerService(messengerRepo, userRepo)

	// initialize handlers
	taskHandler := handlers.NewTaskHandler(log, taskService)
	userHandler := handlers.NewUserHandler(log, userService)
	messengerHandler := handlers.NewMessengerHandler(log, messengerService)

	// setup swagger info
	docs.SwaggerInfo.Title = "Task Management API"
	docs.SwaggerInfo.Description = "API documentation for the Task Management system"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = "localhost:8080" // TODO: remove hardcode
	docs.SwaggerInfo.Schemes = []string{"http"}

	// setup router
	router := gin.Default()

	// Register Swagger handler
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// add middlewares
	if cfg.Metrics.Enabled {
		middleware.InitMetrics()
		router.Use(middleware.MetricsMiddleware())
	}

	if cfg.Tracing.Enabled {
		router.Use(middleware.TracingMiddleware(cfg.Tracing.ServiceName))
	}

	// register application routes
	routes.RegisterRoutes(router, taskHandler, userHandler, messengerHandler)
	routes.RegisterSystemRoutes(router, docs.SwaggerInfo.Version)

	log.Info().Msg("graceful startup")

	// start server
	port := cfg.Server.Port
	// TODO: add default port if not provided
	if port == "" {
		port = "8080"
	}
	log.Printf("Starting server on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal().Err(err).Msg("failed to run server")
	}

}

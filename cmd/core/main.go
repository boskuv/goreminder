package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
	swaggerFiles "github.com/swaggo/files" // Swagger embedded files
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/boskuv/goreminder/docs"
	_ "github.com/boskuv/goreminder/docs" // Import generated Swagger docs
	"github.com/boskuv/goreminder/internal/api/handlers"
	_ "github.com/boskuv/goreminder/internal/api/handlers"
	"github.com/boskuv/goreminder/internal/api/middleware"
	"github.com/boskuv/goreminder/internal/api/routes"
	"github.com/boskuv/goreminder/internal/api/validation"
	"github.com/boskuv/goreminder/internal/repository"
	"github.com/boskuv/goreminder/internal/service"
	"github.com/boskuv/goreminder/pkg/args"
	"github.com/boskuv/goreminder/pkg/config"
	"github.com/boskuv/goreminder/pkg/database"
	"github.com/boskuv/goreminder/pkg/logger"
	"github.com/boskuv/goreminder/pkg/observability"
	"github.com/boskuv/goreminder/pkg/queue"
	"github.com/boskuv/goreminder/pkg/version"
	"github.com/go-playground/validator/v10"
)

func main() {
	// root context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// parse command-line arguments
	parsedArgs := args.ParseArgs()

	// setup configuration
	err := config.Setup(parsedArgs.ConfigPath)
	if err != nil {
		errMsg := fmt.Sprintf("Unable to setup configuration: %s", err)
		panic(errMsg)
	}

	cfg := config.GetConfig()

	// determine minimum logging level based on flag input
	var minlvl zerolog.Level
	var debugMode bool
	if os.Getenv("DEBUG") == "true" {
		minlvl, _ = zerolog.ParseLevel("debug")
	}
	minlvl, _ = zerolog.ParseLevel("info")

	// setup logger with appropriate defaults
	log := logger.New(os.Stdout, minlvl, true)
	logger.LogErrorStackViaPkgErrors(true)

	if cfg.Metrics.Enabled {
		observability.StartMetricsServer(cfg.Metrics.Addr)
	}

	var tracer interface{ Shutdown(context.Context) error }
	if cfg.Tracing.Enabled {
		tp, err := observability.InitTracer(cfg.Tracing.ServiceName, cfg.Tracing.Endpoint, cfg.Tracing.Insecure)
		if err != nil {
			log.Warn().Err(err).Msg("failed to initialize tracer, tracing will be disabled")
		} else {
			tracer = tp
		}
	}

	// DB init
	maxLifetime, err := time.ParseDuration(cfg.Database.ConnMaxLifetime)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("error while setting max max_lifetime for DB connection")
	}

	dbConfig := &database.DBConfig{
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
		RetryDelay:   3 * time.Second, // Default retry delay
	}

	// Create connection manager with automatic health checking and reconnection
	// Check interval: 30 seconds (can be made configurable)
	checkInterval := 30 * time.Second

	// Create connection manager with automatic health checking and reconnection
	dbManager, err := database.NewConnectionManager(ctx, dbConfig, checkInterval, func(db *sqlx.DB) {
		log.Info().Msg("database reconnected successfully - new connection pool created")
	})
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("error while connecting to DB")
	}
	defer func() {
		if err := dbManager.Close(); err != nil {
			log.Error().Stack().Err(err).Msg("failed to close database manager")
		} else {
			log.Info().Msg("database manager is closed gracefully")
		}
	}()

	// Get the database connection from the manager
	db := dbManager.GetDB()

	// Run database migrations automatically on startup
	// Can be disabled by setting SKIP_MIGRATIONS=true
	if os.Getenv("SKIP_MIGRATIONS") != "true" {
		// Allow migrations directory to be overridden via environment variable for CI/CD
		migrationsDir := os.Getenv("MIGRATIONS_DIR")
		if migrationsDir == "" {
			migrationsDir = "migrations"
		}
		if err := database.RunMigrations(db, migrationsDir, log); err != nil {
			log.Fatal().Stack().Err(err).Msg("failed to run database migrations")
		}
	} else {
		log.Info().Msg("database migrations skipped (SKIP_MIGRATIONS=true)")
	}

	// setup repositories
	taskRepo := repository.NewTaskRepository(db, log)
	userRepo := repository.NewUserRepository(db, log)
	messengerRepo := repository.NewMessengerRepository(db, log)
	taskHistoryRepo := repository.NewTaskHistoryRepository(db, log)
	backlogRepo := repository.NewBacklogRepository(db, log)
	targetRepo := repository.NewTargetRepository(db, log)
	digestSettingsRepo := repository.NewDigestSettingsRepository(db, log)

	// producer init (can be disabled via configuration for DB-only mode)
	var publisher queue.Publisher
	if cfg.Producer.Enabled {
		producerConfig := queue.NewProducerConfig(
			cfg.Producer.Host,
			cfg.Producer.Port,
			cfg.Producer.User,
			cfg.Producer.Password,
			cfg.Producer.QueueName,
			cfg.Producer.Exchange,
			cfg.Producer.ConnectionRetries,
			time.Duration(cfg.Producer.ConnectionRetryDelay),
		)

		producer, err := queue.NewProducer(producerConfig, log)
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

		publisher = producer
	} else {
		log.Warn().Msg("queue producer disabled via config, running in DB-only mode")
		publisher = queue.NoopPublisher{}
	}

	// setup services
	taskService := service.NewTaskService(taskRepo, userRepo, messengerRepo, taskHistoryRepo, publisher, log)
	userService := service.NewUserService(userRepo, taskRepo, messengerRepo, publisher, log)
	messengerService := service.NewMessengerService(messengerRepo, userRepo, log)
	backlogService := service.NewBacklogService(backlogRepo, userRepo, messengerRepo, log)
	targetService := service.NewTargetService(targetRepo, userRepo, messengerRepo, log)
	digestService := service.NewDigestService(digestSettingsRepo, backlogRepo, targetRepo, taskRepo, userRepo, messengerRepo, publisher, log)

	// setup scheduler
	taskScheduler := service.NewTaskScheduler(taskRepo, taskService, log)

	// initialize handlers
	taskHandler := handlers.NewTaskHandler(taskService, log)
	userHandler := handlers.NewUserHandler(userService, log)
	messengerHandler := handlers.NewMessengerHandler(messengerService, log)
	backlogHandler := handlers.NewBacklogHandler(backlogService, log)
	targetHandler := handlers.NewTargetHandler(targetService, log)
	digestHandler := handlers.NewDigestHandler(digestService, log)

	// setup swagger info
	appVersion := version.GetVersion()
	log.Info().Str("version", appVersion).Msg("printing current app version...")

	docs.SwaggerInfo.Title = "GoReminder API"
	docs.SwaggerInfo.Description = "API documentation for the GoReminder system"
	docs.SwaggerInfo.Version = appVersion
	docs.SwaggerInfo.Schemes = []string{"http"}

	// setup router
	if !debugMode {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Register custom validators
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		if err := validation.RegisterCustomValidators(v); err != nil {
			log.Warn().Err(err).Msg("failed to register custom validators")
		}
	}

	// Register Swagger handler
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Add base middlewares (order matters)

	// 1. Request ID middleware - should be first to add ID to all requests
	router.Use(middleware.RequestIDMiddleware())

	// 2. Logger middleware - logs all requests with request ID
	router.Use(middleware.LoggerMiddleware(log))

	// 3. CORS middleware (if enabled)
	if cfg.Cors.Enabled {
		router.Use(middleware.CorsMiddleware(middleware.CorsConfig{
			Enabled:          cfg.Cors.Enabled,
			AllowOrigins:     cfg.Cors.AllowOrigins,
			AllowMethods:     cfg.Cors.AllowMethods,
			AllowHeaders:     cfg.Cors.AllowHeaders,
			ExposeHeaders:    cfg.Cors.ExposeHeaders,
			AllowCredentials: cfg.Cors.AllowCredentials,
			MaxAge:           cfg.Cors.MaxAge,
		}))
	}

	// 4. Rate limiting middleware (if enabled)
	if cfg.RateLimit.Enabled {
		rateLimitWindow, err := time.ParseDuration(cfg.RateLimit.Window)
		if err != nil {
			log.Warn().Err(err).Msg("failed to parse rate limit window, using default 1m")
			rateLimitWindow = 1 * time.Minute
		}
		router.Use(middleware.RateLimitMiddleware(middleware.RateLimitConfig{
			Enabled:  cfg.RateLimit.Enabled,
			Requests: cfg.RateLimit.Requests,
			Window:   rateLimitWindow,
		}))
	}

	// 5. Metrics middleware (if enabled)
	if cfg.Metrics.Enabled {
		middleware.InitMetrics()
		router.Use(middleware.MetricsMiddleware())
	}

	// 6. Tracing middleware (if enabled)
	if cfg.Tracing.Enabled {
		router.Use(middleware.TracingMiddleware(cfg.Tracing.ServiceName))
	}

	// register application routes
	routes.RegisterRoutes(router, taskHandler, userHandler, messengerHandler, backlogHandler, targetHandler, digestHandler)
	routes.RegisterSystemRoutes(router, appVersion)

	log.Info().Msg("graceful startup")

	// start scheduler in background (only if autoreschedule is enabled)
	var schedulerCancel context.CancelFunc
	if cfg.Autoreschedule.Enabled {
		schedulerCtx, cancel := context.WithCancel(ctx)
		schedulerCancel = cancel
		defer schedulerCancel()
		scheduleTime := cfg.Autoreschedule.Time
		if scheduleTime == "" {
			scheduleTime = "00:00" // Default to 00:00 if not specified
		}
		go taskScheduler.StartScheduler(schedulerCtx, scheduleTime)
		log.Info().
			Str("schedule_time", scheduleTime).
			Msg("task scheduler started")
	} else {
		log.Info().Msg("task scheduler disabled (autoreschedule.enabled = false)")
	}

	// start server
	port := cfg.Server.Port
	if port == "" {
		port = "8080"
	}
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// run server
	go func() {
		log.Printf("starting server on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("failed to run server")
		}
	}()

	// wait for signal
	<-sigCh
	log.Info().Msg("shutdown signal received")

	// stop scheduler
	schedulerCancel()
	log.Info().Msg("task scheduler stopped")

	// begin graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 10*time.Second)
	defer shutdownCancel()

	// stop accepting new requests and wait for inflight to finish
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Stack().Err(err).Msg("server shutdown error")
	} else {
		log.Info().Msg("server is closed gracefully")
	}

	// Database manager is closed via defer above

	// tracer shutdown (if set)
	if tracer != nil {
		if err := tracer.Shutdown(shutdownCtx); err != nil {
			log.Error().Stack().Err(err).Msg("failed to close tracer")
		} else {
			log.Info().Msg("tracer is closed gracefully")
		}
	}

}

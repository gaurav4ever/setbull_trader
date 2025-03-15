package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"setbull_trader/cmd/trading/transport"
	"setbull_trader/internal/core/adapters/client/dhan"
	"setbull_trader/internal/core/service/orders"
	"setbull_trader/internal/trading/config"
	"setbull_trader/pkg/database"
	"setbull_trader/pkg/log"

	"github.com/gin-gonic/gin"
)

// App represents the application
type App struct {
	config       *config.Config
	router       *gin.Engine
	httpServer   *http.Server
	orderService *orders.Service
	dhanClient   *dhan.Client
}

// NewApp creates a new application
func NewApp() *App {
	ctx := context.Background()
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load configuration: %v", err)
	}
	log.Info("Application configuration loaded successfully.") // New log statement

	// Set up Gin router in release mode for production
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// Add middleware
	router.Use(gin.Recovery())
	router.Use(requestLoggerMiddleware())

	// Initialize Dhan client
	dhanClient := dhan.NewClient(&cfg.Dhan)

	// Initialize services
	orderService := orders.NewService(dhanClient)

	// Get database config
	dbConfig, err := config.LoadDatabase(*cfg)
	if err != nil {
		log.Fatalf("failed to load database config: %v", err)
	}

	connectionMaster, cleanup, err := database.OpenMaster(ctx, dbConfig)
	if err != nil {
		cleanup()
		log.Fatalf("Unable to connect to Database: %v", err)
	}

	// Schema migration handling
	migrationHandler := database.NewMigrationHandler(connectionMaster, dbConfig)
	log.Info("####### STARTING SCHEMA MIGRAION #######")
	if err := migrationHandler.ApplyMigrations(); err != nil {
		log.Fatalf("failed to apply database migrations: %v", err)
	}
	log.Info("####### SCHEMA MIGRAION DONE #######")

	// Set up HTTP handlers
	httpHandler := transport.NewHTTPHandler(orderService)
	httpHandler.RegisterRoutes(router)

	return &App{
		config:       cfg,
		router:       router,
		orderService: orderService,
		dhanClient:   dhanClient,
	}
}

// Run starts the application
func (a *App) Run() error {
	// Set up HTTP server
	a.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%s", a.config.Server.Port),
		Handler:      a.router,
		ReadTimeout:  time.Duration(a.config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(a.config.Server.WriteTimeout) * time.Second,
	}

	// Channel to listen for errors coming from the listener
	serverErrors := make(chan error, 1)

	// Start the HTTP server
	go func() {
		log.Info("Starting HTTP server on port %s", a.config.Server.Port)
		serverErrors <- a.httpServer.ListenAndServe()
	}()

	// Channel to listen for an interrupt or terminate signal
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Blocking main and waiting for shutdown or server errors
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case <-shutdown:
		log.Info("Shutting down server gracefully...")

		// Give outstanding requests 5 seconds to complete
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Shutdown the server
		if err := a.httpServer.Shutdown(ctx); err != nil {
			a.httpServer.Close()
			return fmt.Errorf("could not stop server gracefully: %w", err)
		}
	}

	return nil
}

// requestLoggerMiddleware logs each request
func requestLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path

		// Process request
		c.Next()

		// Log request
		end := time.Now()
		latency := end.Sub(start)

		log.Info("Request: %s %s | Status: %d | Latency: %v",
			c.Request.Method,
			path,
			c.Writer.Status(),
			latency,
		)
	}
}

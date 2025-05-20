package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"setbull_trader/cmd/trading/transport/rest"
	"setbull_trader/internal/core/adapters/client/dhan"
	"setbull_trader/internal/core/adapters/client/upstox"
	"setbull_trader/internal/core/service/orders"
	"setbull_trader/internal/repository"
	"setbull_trader/internal/repository/postgres"
	"setbull_trader/internal/service"
	"setbull_trader/internal/service/normalizer"
	"setbull_trader/internal/service/parser"
	"setbull_trader/internal/trading/config"
	"setbull_trader/pkg/cache"
	"setbull_trader/pkg/database"
	"setbull_trader/pkg/log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// App represents the application
type App struct {
	config                  *config.Config
	router                  *gin.Engine
	httpServer              *http.Server
	orderService            *orders.Service
	dhanClient              *dhan.Client
	db                      *gorm.DB
	stockRepo               repository.StockRepository
	tradeParamsRepo         repository.TradeParametersRepository
	executionPlanRepo       repository.ExecutionPlanRepository
	levelEntryRepo          repository.LevelEntryRepository
	orderExecutionRepo      repository.OrderExecutionRepository
	fibCalculator           *service.FibonacciCalculator
	stockService            *service.StockService
	tradeParamsService      *service.TradeParametersService
	executionPlanService    *service.ExecutionPlanService
	orderExecutionService   *service.OrderExecutionService
	utilityService          *service.UtilityService
	candleProcessingService *service.CandleProcessingService
	candleAggService        *service.CandleAggregationService
	batchFetchService       *service.BatchFetchService
	restServer              *rest.Server
	stockUniverseRepo       repository.StockUniverseRepository
	stockUniverseService    *service.StockUniverseService
	upstoxParser            *parser.UpstoxParser
	stockNormalizer         *normalizer.StockNormalizer
	tradingCalendarService  *service.TradingCalendarService
	stockFilterPipeline     *service.StockFilterPipeline
	marketQuoteService      *service.MarketQuoteService
	stockGroupService       *service.StockGroupService
	groupExecutionService   *service.GroupExecutionService
}

// NewApp creates a new application
func NewApp() *App {
	ctx := context.Background()
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load configuration: %v", err)
	}
	log.Info("Application configuration loaded successfully.")

	// Set up Gin router in release mode for production
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// Add middleware
	router.Use(gin.Recovery())
	router.Use(requestLoggerMiddleware())

	// Initialize Dhan client
	dhanClient := dhan.NewClient(&cfg.Dhan)

	// Initialize legacy services
	orderService := orders.NewService(dhanClient)

	// Initialize cache configurations
	inMemConfig, err := config.LoadInMemoryCache(*cfg)
	if err != nil {
		log.Fatal("Failed to load in-memory cache config: %v", err)
	}

	redisConfig, err := config.LoadRedis(*cfg)
	if err != nil {
		log.Fatal("Failed to load Redis config: %v", err)
	}

	// Initialize cache instances
	cacheInMem := cache.NewInMemoryCache(inMemConfig)
	redisClient := cache.NewRedisStore(redisConfig)

	// Initialize cache
	cacheManager := cache.NewCacheManager(cacheInMem, redisClient)

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
	log.Info("####### STARTING SCHEMA MIGRATION #######")
	if err := migrationHandler.ApplyMigrations(); err != nil {
		log.Fatalf("failed to apply database migrations: %v", err)
	}
	log.Info("####### SCHEMA MIGRATION DONE #######")

	// Extract SQL DB from the connection
	db := connectionMaster.DB

	// UPSTOX configurations
	// Initialize Upstox configuration
	upstoxConfig := &upstox.AuthConfig{
		ClientID:     cfg.Upstox.ClientID,
		ClientSecret: cfg.Upstox.ClientSecret,
		RedirectURI:  cfg.Upstox.RedirectURI,
		BasePath:     cfg.Upstox.BasePath,
	}

	// Initialize repositories
	stockRepo := postgres.NewStockRepository(db)
	tradeParamsRepo := postgres.NewTradeParametersRepository(db)
	executionPlanRepo := postgres.NewExecutionPlanRepository(db)
	levelEntryRepo := postgres.NewLevelEntryRepository(db)
	orderExecutionRepo := postgres.NewOrderExecutionRepository(db)
	tokenRepo := upstox.NewTokenRepository(cacheManager)
	candleRepo := postgres.NewCandleRepository(db)
	stockUniverseRepo := postgres.NewStockUniverseRepository(db)
	filteredStockRepo := postgres.NewFilteredStockRepository(db)
	stockGroupRepo := postgres.NewStockGroupRepository(db)

	upstoxParser := parser.NewUpstoxParser(cfg.StockUniverse.FilePath)
	stockNormalizer := normalizer.NewStockNormalizer()

	// Initialize services
	fibCalculator := service.NewFibonacciCalculator()
	stockService := service.NewStockService(stockRepo, tradeParamsRepo, executionPlanRepo, levelEntryRepo)
	tradeParamsService := service.NewTradeParametersService(tradeParamsRepo, stockRepo)
	executionPlanService := service.NewExecutionPlanService(executionPlanRepo, levelEntryRepo, stockRepo, tradeParamsRepo)
	orderExecutionService := service.NewOrderExecutionService(orderExecutionRepo, executionPlanRepo, stockRepo, levelEntryRepo, *orderService, *stockService)
	utilityService := service.NewUtilityService(fibCalculator)
	tradingCalendarService := service.NewTradingCalendarService(cfg.Trading.Market.ExcludeWeekends)
	upstoxAuthService := upstox.NewAuthService(upstoxConfig, tokenRepo, cacheManager)
	candleProcessingService := service.NewCandleProcessingService(upstoxAuthService, candleRepo, cfg.HistoricalData.BatchSize, "upstox_session")
	batchFetchService := service.NewBatchFetchService(candleProcessingService, cfg.HistoricalData.MaxConcurrentRequests)
	candleAggService := service.NewCandleAggregationService(candleRepo, batchFetchService, tradingCalendarService)
	stockUniverseService := service.NewStockUniverseService(stockUniverseRepo, upstoxParser, stockNormalizer, cfg.StockUniverse.FilePath)
	technicalIndicatorService := service.NewTechnicalIndicatorService(candleRepo)
	stockFilterPipeline := service.NewStockFilterPipeline(stockUniverseService, candleRepo, technicalIndicatorService, tradingCalendarService, filteredStockRepo, cfg)
	marketQuoteService := service.NewMarketQuoteService(upstoxAuthService)
	stockGroupService := service.NewStockGroupService(stockGroupRepo, orderExecutionService, stockService)
	groupExecutionService := service.NewGroupExecutionService(stockGroupService, marketQuoteService, tradeParamsService, executionPlanService, orderExecutionService, cfg, stockUniverseService, technicalIndicatorService)
	stockGroupHandler := rest.NewStockGroupHandler(stockGroupService, stockUniverseService, groupExecutionService)

	restServer := rest.NewServer(
		orderService,
		stockService,
		tradeParamsService,
		executionPlanService,
		orderExecutionService,
		utilityService,
		upstoxAuthService,
		candleAggService,
		batchFetchService,
		stockUniverseService,
		candleProcessingService,
		stockFilterPipeline,
		marketQuoteService,
		groupExecutionService,
		stockGroupService,
		stockGroupHandler,
	)

	// Wire up the group execution scheduler
	_ = service.NewGroupExecutionScheduler(candleAggService, groupExecutionService, stockGroupService, stockUniverseService)

	return &App{
		config:                  cfg,
		router:                  router,
		orderService:            orderService,
		dhanClient:              dhanClient,
		db:                      db,
		stockRepo:               stockRepo,
		tradeParamsRepo:         tradeParamsRepo,
		executionPlanRepo:       executionPlanRepo,
		levelEntryRepo:          levelEntryRepo,
		orderExecutionRepo:      orderExecutionRepo,
		fibCalculator:           fibCalculator,
		stockService:            stockService,
		tradeParamsService:      tradeParamsService,
		executionPlanService:    executionPlanService,
		orderExecutionService:   orderExecutionService,
		utilityService:          utilityService,
		candleProcessingService: candleProcessingService,
		candleAggService:        candleAggService,
		batchFetchService:       batchFetchService,
		restServer:              restServer,
		stockUniverseRepo:       stockUniverseRepo,
		stockUniverseService:    stockUniverseService,
		upstoxParser:            upstoxParser,
		stockNormalizer:         stockNormalizer,
		tradingCalendarService:  tradingCalendarService,
		stockFilterPipeline:     stockFilterPipeline,
		marketQuoteService:      marketQuoteService,
		stockGroupService:       stockGroupService,
		groupExecutionService:   groupExecutionService,
	}
}

// Run starts the application
func (a *App) Run() error {
	// Set up HTTP server
	a.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%s", a.config.Server.Port),
		Handler:      a.restServer,
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

	// Set up context for background goroutines
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var enable1MinCandleIngestion = false

	if enable1MinCandleIngestion {
		// Start precise 1-min ingestion and aggregation loop
		go func() {
			offsetSeconds := a.config.OneMinCandleIngestionOffsetSeconds
			if offsetSeconds < 0 || offsetSeconds > 59 {
				offsetSeconds = 2 // fallback default
			}
			log.Info("Starting precise 1-min candle ingestion and aggregation loop with offset %ds", offsetSeconds)
			for {
				now := time.Now()
				nextMinute := now.Truncate(time.Minute).Add(time.Minute)
				nextTrigger := nextMinute.Add(time.Duration(offsetSeconds) * time.Second)
				sleepDuration := nextTrigger.Sub(now)
				if sleepDuration < 0 {
					sleepDuration = time.Second // fallback minimal sleep
				}
				select {
				case <-ctx.Done():
					log.Info("Stopping precise 1-min ingestion loop")
					return
				case <-time.After(sleepDuration):
				}

				// Fetch all selected stocks
				stocks, err := a.stockGroupService.FetchAllStocksFromAllGroups(ctx, a.stockUniverseService)
				log.Info("Fetched %d stocks", len(stocks))
				if err != nil {
					log.Error("Failed to fetch selected stocks: %v", err)
					continue
				}
				for _, stock := range stocks {
					if stock.InstrumentKey == "" {
						continue
					}
					log.Info("Ingesting 1-min candle for %s", stock.InstrumentKey)
					_, err := a.candleProcessingService.ProcessIntraDayCandles(ctx, stock.InstrumentKey, "1minute")
					if err != nil {
						log.Error("Failed to ingest 1-min candle for %s: %v", stock.InstrumentKey, err)
					}
				}

				// Only aggregate and fire 5-min candle at correct 5-min boundaries since market open
				if isFiveMinBoundarySinceMarketOpen(nextMinute) {
					for _, stock := range stocks {
						if stock.InstrumentKey == "" {
							continue
						}
						end := nextMinute
						start := end.Add(-5 * time.Minute)
						log.Info("[5min AGG] Aggregating 5-min candle for %s for time range %s to %s", stock.InstrumentKey, start.Format(time.RFC3339Nano), end.Format(time.RFC3339Nano))
						err := a.candleAggService.NotifyOnNew5MinCandles(ctx, stock.InstrumentKey, start, end)
						if err != nil {
							log.Error("[5min AGG] Failed to aggregate/fire 5-min candle for %s: %v", stock.InstrumentKey, err)
						}
					}
				}

				// Log timing accuracy
				actualTrigger := time.Now()
				intendedTrigger := nextTrigger
				drift := actualTrigger.Sub(intendedTrigger)
				log.Info("[1min Ingestion Timing] Actual: %s | Intended: %s | Drift: %dms", actualTrigger.Format(time.RFC3339Nano), intendedTrigger.Format(time.RFC3339Nano), drift.Milliseconds())
				if drift > 500*time.Millisecond || drift < -500*time.Millisecond {
					log.Warn("[1min Ingestion Timing] Drift exceeds 500ms: %dms", drift.Milliseconds())
				}
			}
		}()
	}

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

// Helper function to check if a given time is a 5-min boundary since market open (9:15)
func isFiveMinBoundarySinceMarketOpen(t time.Time) bool {
	marketOpenHour := 9
	marketOpenMinute := 15
	if t.Hour() < marketOpenHour || (t.Hour() == marketOpenHour && t.Minute() < marketOpenMinute) {
		return false
	}
	minutesSinceOpen := (t.Hour()-marketOpenHour)*60 + (t.Minute() - marketOpenMinute)
	return minutesSinceOpen >= 0 && minutesSinceOpen%5 == 0
}

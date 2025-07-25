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
	alertService            *service.AlertService
	bbWidthMonitorService   *service.BBWidthMonitorService
	groupExecutionScheduler *service.GroupExecutionScheduler
	masterDataService       service.MasterDataService
	masterDataHandler       *rest.MasterDataHandler
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
	candle5MinRepo := postgres.NewCandle5MinRepository(db)
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
	candleProcessingService := service.NewCandleProcessingService(upstoxAuthService, candleRepo, candle5MinRepo, cfg.HistoricalData.BatchSize, "upstox_session")
	stockUniverseService := service.NewStockUniverseService(stockUniverseRepo, upstoxParser, stockNormalizer, cfg.StockUniverse.FilePath)
	batchFetchService := service.NewBatchFetchService(candleProcessingService, stockUniverseService, cfg.HistoricalData.MaxConcurrentRequests)
	candleAggService := service.NewCandleAggregationService(candleRepo, candle5MinRepo, batchFetchService, tradingCalendarService, utilityService)
	technicalIndicatorService := service.NewTechnicalIndicatorService(candleRepo)
	stockFilterPipeline := service.NewStockFilterPipeline(stockUniverseService, candleRepo, technicalIndicatorService, tradingCalendarService, filteredStockRepo, cfg)
	marketQuoteService := service.NewMarketQuoteService(upstoxAuthService)
	stockGroupService := service.NewStockGroupService(stockGroupRepo, orderExecutionService, stockService)
	groupExecutionService := service.NewGroupExecutionService(stockGroupService, marketQuoteService, tradeParamsService, executionPlanService, orderExecutionService, cfg, stockUniverseService, technicalIndicatorService, candleAggService)
	stockGroupHandler := rest.NewStockGroupHandler(stockGroupService, stockUniverseService, groupExecutionService)

	// Initialize BB width monitoring services
	alertService := service.NewAlertService(&cfg.BBWidthMonitoring)
	bbWidthMonitorService := service.NewBBWidthMonitorService(
		stockGroupService,
		technicalIndicatorService,
		alertService,
		stockUniverseService,
		&cfg.BBWidthMonitoring,
		candleAggService,
	)

	// Initialize master data service
	masterDataProcessRepo := postgres.NewMasterDataProcessRepository(db)

	// Create service adapters
	dailyDataAdapter := service.NewDailyDataServiceAdapter(candleAggService, stockUniverseService)
	filterPipelineAdapter := service.NewFilterPipelineServiceAdapter(stockFilterPipeline)
	minuteDataAdapter := service.NewMinuteDataServiceAdapter(batchFetchService)

	masterDataService := service.NewMasterDataService(
		masterDataProcessRepo,
		tradingCalendarService,
		dailyDataAdapter,
		filterPipelineAdapter,
		minuteDataAdapter,
	)

	// Create master data handler
	masterDataHandler := rest.NewMasterDataHandler(masterDataService)

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
		masterDataHandler,
	)

	// Wire up the group execution scheduler with BB width monitoring
	groupExecutionScheduler := service.NewGroupExecutionScheduler(groupExecutionService, stockGroupService, stockUniverseService, bbWidthMonitorService)

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
		alertService:            alertService,
		bbWidthMonitorService:   bbWidthMonitorService,
		groupExecutionScheduler: groupExecutionScheduler,
		masterDataService:       masterDataService,
		masterDataHandler:       masterDataHandler,
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

	var enable1MinCandleIngestion = true

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
				log.Info("[LIVE] Fetched %d stocks for 1-minute ingestion", len(stocks))
				if err != nil {
					log.Error("[LIVE] Failed to fetch selected stocks: %v", err)
					continue
				}

				// Track which stocks need 5-minute aggregation
				stocksNeeding5MinAgg := make([]string, 0)

				for _, stock := range stocks {
					if stock.InstrumentKey == "" {
						log.Debug("[LIVE] Skipping stock with empty instrument key: %s", stock.Symbol)
						continue
					}

					log.Debug("[LIVE] Ingesting 1-min candle for %s (%s)", stock.InstrumentKey, stock.Symbol)
					recordCount, err := a.candleProcessingService.ProcessIntraDayCandles(ctx, stock.InstrumentKey, "1minute")
					if err != nil {
						log.Error("[LIVE] Failed to ingest 1-min candle for %s: %v", stock.InstrumentKey, err)
						continue
					}

					if recordCount > 0 {
						log.Info("[LIVE] Successfully ingested %d 1-min candles for %s", recordCount, stock.InstrumentKey)

						// Check if this stock needs 5-minute aggregation
						latestCandle, err := a.candleProcessingService.GetLatestCandle(ctx, stock.InstrumentKey, "1minute")
						if err != nil {
							log.Error("[LIVE] Failed to get latest candle for %s: %v", stock.InstrumentKey, err)
							continue
						}

						if latestCandle != nil && a.candleProcessingService.IsFiveMinBoundarySinceMarketOpen(latestCandle.Timestamp) {
							log.Info("[LIVE] Stock %s needs 5-minute aggregation at %s", stock.InstrumentKey, latestCandle.Timestamp.Format("15:04"))
							stocksNeeding5MinAgg = append(stocksNeeding5MinAgg, stock.InstrumentKey)
						}
					} else {
						log.Debug("[LIVE] No new 1-min candles for %s", stock.InstrumentKey)
					}
				}

				// Trigger 5-minute aggregation for stocks that need it
				if len(stocksNeeding5MinAgg) > 0 {
					log.Info("[LIVE] Triggering 5-minute aggregation for %d stocks: %v", len(stocksNeeding5MinAgg), stocksNeeding5MinAgg)

					for _, instrumentKey := range stocksNeeding5MinAgg {
						latestCandle, err := a.candleProcessingService.GetLatestCandle(ctx, instrumentKey, "1minute")
						if err != nil {
							log.Error("[LIVE] Failed to get latest candle for 5-min aggregation for %s: %v", instrumentKey, err)
							continue
						}

						if latestCandle != nil {
							log.Info("[LIVE] Aggregating 5-min candles for %s at %s", instrumentKey, latestCandle.Timestamp.Format("15:04"))
							if err := a.candleProcessingService.AggregateAndStore5MinCandles(ctx, instrumentKey, latestCandle.Timestamp); err != nil {
								log.Error("[LIVE] Failed to aggregate 5-min candles for %s: %v", instrumentKey, err)
							} else {
								log.Info("[LIVE] Successfully aggregated and stored 5-min candles for %s", instrumentKey)
							}
						}
					}
				}

				// Legacy 5-minute aggregation for backward compatibility
				if isFiveMinBoundarySinceMarketOpen(nextMinute) {
					end := nextMinute
					start := end.Add(-5 * time.Minute)
					log.Info("[LIVE] Legacy 5-min aggregation for time range %s to %s", start.Format(time.RFC3339Nano), end.Format(time.RFC3339Nano))
					err := a.stockGroupService.NotifyOnNew5Min(ctx, start, end)
					if err != nil {
						log.Error("[LIVE] Failed to execute legacy 5-min aggregation: %v", err)
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

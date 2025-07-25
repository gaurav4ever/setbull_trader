---
description: "GitHub Copilot Rules for Setbull Trader Project"
globs: ["**/*"]
alwaysApply: true
---

# Setbull Trader - GitHub Copilot Configuration Rules

## Project Context & Identity

This is the **Setbull Trader** - an algorithmic trading platform with:
- **Backend**: Go-based trading engine with broker API integrations
- **Frontend**: Svelte/TypeScript user interface
- **Focus**: Automated trading strategies (BB Width, VWAP, EMA-VWAP-BB)
- **Brokers**: Dhan API, Upstox API
- **Architecture**: Layered (domain/repository/service/transport)

## Core Development Principles

### 1. DESIGN-FIRST APPROACH
- When implementing significant features: HLD → LLD → Phase Plan → Implementation
- Break complex features into 1-2 file phases
- Request confirmation before proceeding with implementation
- Thoroughly explain reasoning and methodology

### 2. ARCHITECTURE COMPLIANCE
```yaml
Layer Separation:
  - Domain: Pure business models (no transport annotations)
  - Repository: Data access only (no business logic)
  - Service: Business logic and orchestration
  - Transport: HTTP handlers and API contracts

Database Standards:
  - Migration-based schema evolution
  - Soft deletes with 'active' flags
  - Audit fields: created_at, updated_at
  - Security IDs over symbols for orders

Trading Rules:
  - Max 3 concurrent selected stocks
  - Always validate order parameters
  - Implement stop-loss mechanisms
  - Use circuit breakers for risk management
```

### 3. CODE QUALITY STANDARDS

#### Go Code Patterns
```go
// Custom error types with context
type TradingError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Cause   error  `json:"-"`
}

// Repository interface pattern
type StockRepository interface {
    GetBySecurityID(ctx context.Context, securityID string) (*domain.Stock, error)
    Save(ctx context.Context, stock *domain.Stock) error
    GetActiveStocks(ctx context.Context) ([]*domain.Stock, error)
}

// Service dependency injection
type TradingService struct {
    stockRepo      StockRepository
    orderRepo      OrderRepository
    dhanClient     broker.DhanClient
    logger         *log.Logger
    riskManager    *RiskManager
}

// Proper error handling with context
func (s *TradingService) ExecuteStrategy(ctx context.Context, strategy Strategy) error {
    if err := strategy.Validate(); err != nil {
        return fmt.Errorf("strategy validation failed: %w", err)
    }
    
    // Implementation with proper logging
    s.logger.Info("executing strategy", 
        "strategy", strategy.Name(),
        "timestamp", time.Now(),
    )
    
    return nil
}
```

#### Database Entity Pattern
```go
type Stock struct {
    ID          int64     `json:"id" db:"id"`
    SecurityID  string    `json:"security_id" db:"security_id"`
    Symbol      string    `json:"symbol" db:"symbol"`
    Exchange    string    `json:"exchange" db:"exchange"`
    Price       float64   `json:"price" db:"price"`
    Active      bool      `json:"active" db:"active"`
    CreatedAt   time.Time `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}
```

#### Svelte/TypeScript Patterns
```typescript
// Store pattern for state management
import { writable, type Writable } from 'svelte/store';

interface TradingState {
    selectedStocks: Stock[];
    activeOrders: Order[];
    isLoading: boolean;
    error: string | null;
}

export const tradingStore: Writable<TradingState> = writable({
    selectedStocks: [],
    activeOrders: [],
    isLoading: false,
    error: null
});

// API service with proper error handling
class TradingAPI {
    private baseUrl = '/api';
    
    async submitOrder(order: OrderRequest): Promise<OrderResponse> {
        try {
            const response = await fetch(`${this.baseUrl}/orders`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(order)
            });
            
            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.message || 'Order submission failed');
            }
            
            return await response.json();
        } catch (error) {
            console.error('API Error:', error);
            throw error;
        }
    }
}
```

## Broker API Integration Rules

### 1. Dhan API Patterns
```go
type DhanClient struct {
    baseURL    string
    apiKey     string
    httpClient *http.Client
    logger     *log.Logger
}

func (c *DhanClient) PlaceOrder(ctx context.Context, req *DhanOrderRequest) (*DhanOrderResponse, error) {
    // 1. Validate input parameters
    if req.SecurityID == "" {
        return nil, &TradingError{Code: "INVALID_SECURITY_ID", Message: "security ID is required"}
    }
    
    // 2. Log request (without sensitive data)
    c.logger.Info("placing order", 
        "security_id", req.SecurityID,
        "quantity", req.Quantity,
        "order_type", req.OrderType,
    )
    
    // 3. Implement retry logic for transient errors
    // 4. Handle rate limiting
    // 5. Map response to internal domain models
}
```

### 2. Order Management Standards
```go
type OrderRequest struct {
    SecurityID   string  `json:"security_id" validate:"required"`
    Symbol       string  `json:"symbol" validate:"required"`
    Quantity     int     `json:"quantity" validate:"min=1"`
    Price        float64 `json:"price" validate:"min=0"`
    OrderType    string  `json:"order_type" validate:"oneof=BUY SELL"`
    ProductType  string  `json:"product_type"`
    Validity     string  `json:"validity"`
}

// Idempotent order submission
func (s *OrderService) SubmitOrder(ctx context.Context, req *OrderRequest) (*Order, error) {
    // Check for duplicate orders
    existing, err := s.orderRepo.GetByClientOrderID(ctx, req.ClientOrderID)
    if err != nil && !errors.Is(err, ErrNotFound) {
        return nil, fmt.Errorf("failed to check existing order: %w", err)
    }
    
    if existing != nil {
        return existing, nil // Return existing order
    }
    
    // Proceed with new order submission
}
```

## Trading Strategy Implementation

### 1. Strategy Interface
```go
type Strategy interface {
    Name() string
    Execute(ctx context.Context, data *MarketData) (*Signal, error)
    Validate() error
    GetParameters() map[string]interface{}
}

type Signal struct {
    Action     string    `json:"action"` // BUY, SELL, HOLD
    SecurityID string    `json:"security_id"`
    Confidence float64   `json:"confidence"`
    Timestamp  time.Time `json:"timestamp"`
    Metadata   map[string]interface{} `json:"metadata"`
}
```

### 2. Bollinger Bands Width Strategy
```go
type BBWidthStrategy struct {
    period         int     `json:"period"`
    stdDevMultiple float64 `json:"std_dev_multiple"`
    widthThreshold float64 `json:"width_threshold"`
}

func (s *BBWidthStrategy) Execute(ctx context.Context, data *MarketData) (*Signal, error) {
    // Calculate Bollinger Bands
    sma := calculateSMA(data.ClosePrices, s.period)
    stdDev := calculateStdDev(data.ClosePrices, s.period)
    
    upperBand := sma + (s.stdDevMultiple * stdDev)
    lowerBand := sma - (s.stdDevMultiple * stdDev)
    bbWidth := (upperBand - lowerBand) / sma
    
    // Generate signal based on width
    if bbWidth < s.widthThreshold {
        return &Signal{
            Action:     "MONITOR", // Squeeze detected
            SecurityID: data.SecurityID,
            Confidence: 0.8,
            Timestamp:  time.Now(),
            Metadata: map[string]interface{}{
                "bb_width": bbWidth,
                "squeeze":  true,
            },
        }, nil
    }
    
    return &Signal{Action: "HOLD"}, nil
}
```

### 3. VWAP Strategy
```go
type VWAPStrategy struct {
    lookbackPeriod int `json:"lookback_period"`
}

func (s *VWAPStrategy) Execute(ctx context.Context, data *MarketData) (*Signal, error) {
    vwap := calculateVWAP(data.Prices, data.Volumes, s.lookbackPeriod)
    currentPrice := data.ClosePrices[len(data.ClosePrices)-1]
    
    // Signal logic based on price vs VWAP
    if currentPrice > vwap*1.002 { // 0.2% above VWAP
        return &Signal{
            Action:     "BUY",
            SecurityID: data.SecurityID,
            Confidence: 0.7,
            Timestamp:  time.Now(),
            Metadata: map[string]interface{}{
                "vwap":          vwap,
                "current_price": currentPrice,
                "deviation":     (currentPrice - vwap) / vwap,
            },
        }, nil
    }
    
    return &Signal{Action: "HOLD"}, nil
}
```

## Risk Management Patterns

### 1. Position Size Calculator
```go
type RiskManager struct {
    maxPositionSize   float64
    maxDailyLoss      float64
    maxConcurrentTrades int
}

func (rm *RiskManager) CalculatePositionSize(
    accountBalance float64,
    stockPrice float64,
    riskPercentage float64,
) (int, error) {
    maxRiskAmount := accountBalance * (riskPercentage / 100)
    maxShares := int(maxRiskAmount / stockPrice)
    
    if maxShares > rm.maxPositionSize {
        maxShares = int(rm.maxPositionSize)
    }
    
    return maxShares, nil
}
```

### 2. Stop Loss Implementation
```go
type StopLossManager struct {
    defaultStopLoss float64 // percentage
}

func (slm *StopLossManager) CreateStopLossOrder(
    originalOrder *Order,
    stopLossPercentage float64,
) *OrderRequest {
    var stopPrice float64
    
    if originalOrder.OrderType == "BUY" {
        stopPrice = originalOrder.Price * (1 - stopLossPercentage/100)
    } else {
        stopPrice = originalOrder.Price * (1 + stopLossPercentage/100)
    }
    
    return &OrderRequest{
        SecurityID:  originalOrder.SecurityID,
        Symbol:      originalOrder.Symbol,
        Quantity:    originalOrder.Quantity,
        Price:       stopPrice,
        OrderType:   getOppositeOrderType(originalOrder.OrderType),
        ProductType: "STOP_LOSS",
        Validity:    "DAY",
    }
}
```

## Testing Patterns

### 1. Unit Testing
```go
func TestBBWidthStrategy_Execute(t *testing.T) {
    strategy := &BBWidthStrategy{
        period:         20,
        stdDevMultiple: 2.0,
        widthThreshold: 0.02,
    }
    
    testData := &MarketData{
        SecurityID:   "NSE_EQ|INE002A01018",
        ClosePrices:  []float64{100, 101, 99, 102, 98},
        Volumes:      []int64{1000, 1100, 900, 1200, 800},
        Timestamps:   generateTimestamps(5),
    }
    
    signal, err := strategy.Execute(context.Background(), testData)
    
    assert.NoError(t, err)
    assert.NotNil(t, signal)
    assert.Contains(t, []string{"BUY", "SELL", "HOLD", "MONITOR"}, signal.Action)
}
```

### 2. Integration Testing
```go
func TestDhanClient_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    client := NewDhanClient(testConfig.DhanAPIKey, testConfig.DhanBaseURL)
    
    // Test with sandbox/demo account
    order := &DhanOrderRequest{
        SecurityID:  "NSE_EQ|INE002A01018",
        Quantity:    1,
        Price:       100.0,
        OrderType:   "BUY",
        ProductType: "CNC",
        Validity:    "DAY",
    }
    
    response, err := client.PlaceOrder(context.Background(), order)
    assert.NoError(t, err)
    assert.NotEmpty(t, response.OrderID)
}
```

## Logging and Monitoring

### 1. Structured Logging
```go
// Use structured logging throughout
logger.Info("strategy executed",
    "strategy", strategy.Name(),
    "security_id", securityID,
    "signal", signal.Action,
    "confidence", signal.Confidence,
    "execution_time_ms", time.Since(start).Milliseconds(),
)

logger.Error("order submission failed",
    "broker", "dhan",
    "security_id", order.SecurityID,
    "error", err,
    "retry_attempt", retryCount,
)
```

### 2. Performance Monitoring
```go
func (s *TradingService) ExecuteStrategyWithMetrics(ctx context.Context, strategy Strategy) error {
    start := time.Now()
    defer func() {
        duration := time.Since(start)
        s.metrics.RecordStrategyExecution(strategy.Name(), duration)
    }()
    
    return s.ExecuteStrategy(ctx, strategy)
}
```

## Security Best Practices

### 1. Credential Management
```go
type Config struct {
    DhanAPIKey     string `env:"DHAN_API_KEY,required"`
    UpstoxAPIKey   string `env:"UPSTOX_API_KEY,required"`
    DatabaseURL    string `env:"DATABASE_URL,required"`
    JWTSecret      string `env:"JWT_SECRET,required"`
    Environment    string `env:"ENVIRONMENT" envDefault:"development"`
}

// Never log sensitive data
func (c *DhanClient) logRequest(req *DhanOrderRequest) {
    c.logger.Info("api request",
        "endpoint", "/orders",
        "security_id", req.SecurityID,
        "quantity", req.Quantity,
        // Never log API keys or sensitive data
    )
}
```

### 2. Input Validation
```go
func ValidateOrderRequest(req *OrderRequest) error {
    if req.SecurityID == "" {
        return errors.New("security_id is required")
    }
    
    if req.Quantity <= 0 {
        return errors.New("quantity must be positive")
    }
    
    if req.Price < 0 {
        return errors.New("price cannot be negative")
    }
    
    validOrderTypes := []string{"BUY", "SELL"}
    if !contains(validOrderTypes, req.OrderType) {
        return fmt.Errorf("invalid order_type: %s", req.OrderType)
    }
    
    return nil
}
```

## Performance Optimization

### 1. Database Optimization
```go
// Use prepared statements for repeated queries
const getActiveStocksQuery = `
    SELECT id, security_id, symbol, exchange, price, created_at, updated_at 
    FROM stocks 
    WHERE active = true 
    ORDER BY symbol
`

// Implement connection pooling
func NewDatabase(databaseURL string) (*sql.DB, error) {
    db, err := sql.Open("postgres", databaseURL)
    if err != nil {
        return nil, err
    }
    
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(time.Hour)
    
    return db, nil
}
```

### 2. Caching Strategy
```go
type MarketDataCache struct {
    cache map[string]*CacheEntry
    mutex sync.RWMutex
    ttl   time.Duration
}

type CacheEntry struct {
    Data      *MarketData
    ExpiresAt time.Time
}

func (c *MarketDataCache) Get(securityID string) (*MarketData, bool) {
    c.mutex.RLock()
    defer c.mutex.RUnlock()
    
    entry, exists := c.cache[securityID]
    if !exists || time.Now().After(entry.ExpiresAt) {
        return nil, false
    }
    
    return entry.Data, true
}
```

## File Organization Rules

### Backend Structure
```
internal/
├── core/           # Core business logic and interfaces
├── domain/         # Domain models and business rules
├── repository/     # Data access layer implementations
├── service/        # Business logic services
├── trading/        # Trading-specific logic and strategies
└── transport/      # HTTP handlers and API contracts

strategies/         # Trading strategy implementations
├── bb_width.go
├── vwap.go
├── ema_vwap_bb.go
└── base.go

pkg/               # Shared utilities and helpers
├── logger/
├── metrics/
└── utils/
```

### Frontend Structure
```
frontend/src/
├── components/     # Reusable UI components
├── routes/         # Page components (Svelte routing)
├── stores/         # State management (Svelte stores)
├── lib/           # Utilities and API services
├── types/         # TypeScript type definitions
└── assets/        # Static assets
```

## Common Anti-Patterns to Avoid

1. **Mixing Layers**: Never put HTTP concerns in domain models
2. **Ignoring Errors**: Always handle and log errors appropriately
3. **Symbol vs Security ID**: Always prefer security IDs for broker APIs
4. **Blocking Operations**: Use proper async patterns and timeouts
5. **Memory Leaks**: Always close resources (DB connections, HTTP clients)
6. **Hardcoded Values**: Use configuration for all environment-specific values
7. **Missing Validation**: Validate all inputs at API boundaries
8. **Race Conditions**: Protect shared state with proper synchronization

Remember: This is a financial trading system where data accuracy, security, and performance are mission-critical. Always prioritize correctness and proper error handling over convenience.

# Setbull Trader - GitHub Copilot Instructions

## Project Overview
Setbull Trader is a sophisticated algorithmic trading platform built with Go backend and Svelte frontend, focusing on automated trading strategies with broker API integrations (Dhan, Upstox).

## Core Architecture Context

### Tech Stack
- **Backend**: Go (main.go entry point)
- **Frontend**: Svelte with TypeScript
- **Database**: SQL with migration-based schema
- **Brokers**: Dhan API, Upstox API integration
- **Architecture**: Layered (domain, repository, service, transport)

### Key Directories
- `internal/`: Core business logic (domain, repository, service, trading)
- `frontend/`: Svelte application
- `strategies/`: Trading algorithms and strategies
- `dhan/`, `upstox/`: Broker API integrations
- `kb/`: Knowledge base and documentation

## Code Generation Guidelines

### 1. Design-First Approach
When generating significant features:
1. Provide high-level design first
2. Present low-level design with reasoning
3. Break implementation into phases (1-2 files per phase)
4. Request confirmation before proceeding

### 2. Architecture Adherence
- **Layered Architecture**: Maintain strict separation between layers
- **Domain Models**: Never include transport-specific annotations
- **Repository Pattern**: Only handle data access, no business logic
- **Service Layer**: Contains all business logic and orchestration

### 3. Go-Specific Patterns
```go
// Error handling - use custom error types
type TradingError struct {
    Code    string
    Message string
    Cause   error
}

// Repository interface pattern
type StockRepository interface {
    GetBySecurityID(ctx context.Context, securityID string) (*domain.Stock, error)
    Save(ctx context.Context, stock *domain.Stock) error
}

// Service structure with dependencies
type TradingService struct {
    stockRepo StockRepository
    logger    *log.Logger
}
```

### 4. Database Patterns
- Always include audit fields: `created_at`, `updated_at`
- Use soft deletes with `active` boolean flags
- Security IDs over symbols for all order operations
- Proper indexing on WHERE clause fields

### 5. API Integration Standards
```go
// Broker API client structure
type DhanClient struct {
    baseURL    string
    apiKey     string
    httpClient *http.Client
    logger     *log.Logger
}

// Error handling with retries
func (c *DhanClient) PlaceOrder(ctx context.Context, order *OrderRequest) (*OrderResponse, error) {
    // Validate order parameters
    // Log request
    // Handle retries for transient errors
    // Return structured errors
}
```

## Trading-Specific Code Patterns

### 1. Order Management
```go
type Order struct {
    SecurityID   string    `json:"security_id" db:"security_id"`
    Symbol       string    `json:"symbol" db:"symbol"`
    Quantity     int       `json:"quantity" db:"quantity"`
    Price        float64   `json:"price" db:"price"`
    OrderType    string    `json:"order_type" db:"order_type"`
    CreatedAt    time.Time `json:"created_at" db:"created_at"`
    UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
    Active       bool      `json:"active" db:"active"`
}
```

### 2. Risk Management
- Enforce position size limits
- Implement stop-loss mechanisms
- Never exceed 3 concurrent selected stocks
- Include circuit breakers

### 3. Strategy Implementation
```go
type Strategy interface {
    Name() string
    Execute(ctx context.Context, data *MarketData) (*Signal, error)
    Validate() error
}

type BBWidthStrategy struct {
    period    int
    threshold float64
}
```

## Frontend (Svelte) Patterns

### 1. Component Structure
```typescript
// Store pattern for state management
import { writable } from 'svelte/store';

export const tradingState = writable({
    selectedStocks: [],
    activeOrders: [],
    isLoading: false
});

// Error handling
export const errorStore = writable({
    message: '',
    type: 'info' | 'warning' | 'error'
});
```

### 2. API Integration
```typescript
// Consistent error handling
async function submitOrder(orderData: OrderRequest): Promise<OrderResponse> {
    try {
        const response = await fetch('/api/orders', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(orderData)
        });
        
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        
        return await response.json();
    } catch (error) {
        logger.error('Order submission failed:', error);
        throw error;
    }
}
```

## Testing Patterns

### 1. Unit Tests
```go
func TestTradingService_ExecuteStrategy(t *testing.T) {
    // Arrange
    mockRepo := &MockStockRepository{}
    service := NewTradingService(mockRepo, logger)
    
    // Act
    result, err := service.ExecuteStrategy(ctx, strategyName, stockData)
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

### 2. Integration Tests
```go
func TestDhanAPI_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    // Test with actual API
}
```

## Security & Performance

### 1. Security Patterns
```go
// Secure credential management
type Config struct {
    DhanAPIKey    string `env:"DHAN_API_KEY,required"`
    UpstoxAPIKey  string `env:"UPSTOX_API_KEY,required"`
    DatabaseURL   string `env:"DATABASE_URL,required"`
}

// Input validation
func ValidateOrderRequest(req *OrderRequest) error {
    if req.SecurityID == "" {
        return errors.New("security_id is required")
    }
    // Additional validations
}
```

### 2. Performance Patterns
```go
// Connection pooling
func NewDBPool(databaseURL string) (*sql.DB, error) {
    db, err := sql.Open("postgres", databaseURL)
    if err != nil {
        return nil, err
    }
    
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(time.Hour)
    
    return db, nil
}

// Caching for market data
type CacheManager struct {
    cache map[string]*CacheEntry
    mutex sync.RWMutex
    ttl   time.Duration
}
```

## Logging & Monitoring

```go
// Structured logging
logger.Info("Order submitted",
    "order_id", orderID,
    "security_id", securityID,
    "quantity", quantity,
    "price", price,
)

// Error logging with context
logger.Error("API call failed",
    "broker", "dhan",
    "endpoint", "/orders",
    "error", err,
    "retry_count", retryCount,
)
```

## File Organization Preferences

### 1. Backend Structure
```
internal/
├── core/           # Core business logic
├── domain/         # Domain models and interfaces
├── repository/     # Data access layer
├── service/        # Business logic services
└── trading/        # Trading-specific logic

strategies/         # Trading strategies
├── bb_width.go
├── vwap.go
└── ema_vwap_bb.go
```

### 2. Frontend Structure
```
frontend/src/
├── components/     # Reusable UI components
├── routes/         # Page components
├── stores/         # State management
├── lib/           # Utilities and helpers
└── types/         # TypeScript type definitions
```

## Common Pitfalls to Avoid

1. **Never mix layers**: Domain models shouldn't contain HTTP annotations
2. **Security ID priority**: Always use security IDs for broker API calls
3. **Proper error handling**: Don't ignore errors, wrap with context
4. **Race conditions**: Be careful with concurrent map access
5. **Memory leaks**: Properly close database connections and HTTP clients
6. **API rate limits**: Implement proper retry mechanisms with backoff

## Specific Business Logic

### 1. Fibonacci-Based Execution Levels
- Understand the mathematical basis for trade entry/exit points
- Implement proper level calculations with precision

### 2. Bollinger Bands Width Strategy
- Monitor BB width for squeeze/expansion signals
- Implement proper volatility calculations

### 3. VWAP Strategy
- Calculate volume-weighted average price correctly
- Consider intraday vs. daily VWAP calculations

## Development Workflow

1. **Feature Branches**: Always create feature branches
2. **Testing**: Write tests before or alongside code
3. **Documentation**: Comment complex trading logic thoroughly
4. **Performance**: Profile CPU/memory for trading algorithms
5. **Security**: Never commit API keys or credentials

Remember: This is a financial trading system where accuracy, security, and performance are critical. Always prioritize data integrity and proper error handling.

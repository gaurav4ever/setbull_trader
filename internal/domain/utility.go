package domain

// LevelWithQuantity extends ExecutionLevel with quantity information
type LevelWithQuantity struct {
	Level       float64 `json:"level"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
	Quantity    int     `json:"quantity"`
}

// FibonacciLevelsResponse represents the response for Fibonacci calculations
type FibonacciLevelsResponse struct {
	TotalQuantity int                 `json:"totalQuantity"`
	Levels        []LevelWithQuantity `json:"levels"`
}

package postgres

import (
	"context"
	"database/sql"
	"errors"

	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// StockRepository implements repository.StockRepository interface using PostgreSQL
type StockRepository struct {
	db *sqlx.DB
}

// NewStockRepository creates a new StockRepository
func NewStockRepository(db *sqlx.DB) repository.StockRepository {
	return &StockRepository{db: db}
}

// Create creates a new stock
func (r *StockRepository) Create(ctx context.Context, stock *domain.Stock) error {
	if stock.ID == "" {
		stock.ID = uuid.New().String()
	}

	query := `
		INSERT INTO stocks (id, symbol, name, current_price, is_selected)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.ExecContext(ctx, query,
		stock.ID,
		stock.Symbol,
		stock.Name,
		stock.CurrentPrice,
		stock.IsSelected,
	)

	return err
}

// GetByID retrieves a stock by its ID
func (r *StockRepository) GetByID(ctx context.Context, id string) (*domain.Stock, error) {
	var stock domain.Stock

	query := `
		SELECT id, symbol, name, current_price, is_selected
		FROM stocks
		WHERE id = $1
	`

	err := r.db.GetContext(ctx, &stock, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Return nil if stock not found
		}
		return nil, err
	}

	return &stock, nil
}

// GetBySymbol retrieves a stock by its symbol
func (r *StockRepository) GetBySymbol(ctx context.Context, symbol string) (*domain.Stock, error) {
	var stock domain.Stock

	query := `
		SELECT id, symbol, name, current_price, is_selected
		FROM stocks
		WHERE symbol = $1
	`

	err := r.db.GetContext(ctx, &stock, query, symbol)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Return nil if stock not found
		}
		return nil, err
	}

	return &stock, nil
}

// GetAll retrieves all stocks
func (r *StockRepository) GetAll(ctx context.Context) ([]*domain.Stock, error) {
	var stocks []*domain.Stock

	query := `
		SELECT id, symbol, name, current_price, is_selected
		FROM stocks
		ORDER BY symbol
	`

	err := r.db.SelectContext(ctx, &stocks, query)
	if err != nil {
		return nil, err
	}

	return stocks, nil
}

// GetSelected retrieves all selected stocks
func (r *StockRepository) GetSelected(ctx context.Context) ([]*domain.Stock, error) {
	var stocks []*domain.Stock

	query := `
		SELECT id, symbol, name, current_price, is_selected
		FROM stocks
		WHERE is_selected = true
		ORDER BY symbol
	`

	err := r.db.SelectContext(ctx, &stocks, query)
	if err != nil {
		return nil, err
	}

	return stocks, nil
}

// Update updates a stock
func (r *StockRepository) Update(ctx context.Context, stock *domain.Stock) error {
	query := `
		UPDATE stocks
		SET symbol = $1, name = $2, current_price = $3, is_selected = $4
		WHERE id = $5
	`

	_, err := r.db.ExecContext(ctx, query,
		stock.Symbol,
		stock.Name,
		stock.CurrentPrice,
		stock.IsSelected,
		stock.ID,
	)

	return err
}

// ToggleSelection toggles the selection status of a stock
func (r *StockRepository) ToggleSelection(ctx context.Context, id string, isSelected bool) error {
	query := `
		UPDATE stocks
		SET is_selected = $1
		WHERE id = $2
	`

	_, err := r.db.ExecContext(ctx, query, isSelected, id)
	return err
}

// Delete deletes a stock
func (r *StockRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM stocks WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

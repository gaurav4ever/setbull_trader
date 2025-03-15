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

// TradeParametersRepository implements repository.TradeParametersRepository using PostgreSQL
type TradeParametersRepository struct {
	db *sqlx.DB
}

// NewTradeParametersRepository creates a new TradeParametersRepository
func NewTradeParametersRepository(db *sqlx.DB) repository.TradeParametersRepository {
	return &TradeParametersRepository{db: db}
}

// Create creates new trade parameters
func (r *TradeParametersRepository) Create(ctx context.Context, params *domain.TradeParameters) error {
	if params.ID == "" {
		params.ID = uuid.New().String()
	}

	query := `
		INSERT INTO trade_parameters (id, stock_id, starting_price, sl_percentage, risk_amount, trade_side)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(ctx, query,
		params.ID,
		params.StockID,
		params.StartingPrice,
		params.StopLossPercentage,
		params.RiskAmount,
		params.TradeSide,
	)

	return err
}

// GetByID retrieves trade parameters by their ID
func (r *TradeParametersRepository) GetByID(ctx context.Context, id string) (*domain.TradeParameters, error) {
	var params domain.TradeParameters

	query := `
		SELECT id, stock_id, starting_price, sl_percentage, risk_amount, trade_side
		FROM trade_parameters
		WHERE id = $1
	`

	err := r.db.GetContext(ctx, &params, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Return nil if not found
		}
		return nil, err
	}

	return &params, nil
}

// GetByStockID retrieves trade parameters for a specific stock
func (r *TradeParametersRepository) GetByStockID(ctx context.Context, stockID string) (*domain.TradeParameters, error) {
	var params domain.TradeParameters

	query := `
		SELECT id, stock_id, starting_price, sl_percentage, risk_amount, trade_side
		FROM trade_parameters
		WHERE stock_id = $1
	`

	err := r.db.GetContext(ctx, &params, query, stockID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Return nil if not found
		}
		return nil, err
	}

	return &params, nil
}

// Update updates trade parameters
func (r *TradeParametersRepository) Update(ctx context.Context, params *domain.TradeParameters) error {
	query := `
		UPDATE trade_parameters
		SET starting_price = $1, sl_percentage = $2, risk_amount = $3, trade_side = $4
		WHERE id = $5
	`

	_, err := r.db.ExecContext(ctx, query,
		params.StartingPrice,
		params.StopLossPercentage,
		params.RiskAmount,
		params.TradeSide,
		params.ID,
	)

	return err
}

// Delete deletes trade parameters
func (r *TradeParametersRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM trade_parameters WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

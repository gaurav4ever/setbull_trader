package postgres

import (
	"context"

	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// LevelEntryRepository implements repository.LevelEntryRepository using PostgreSQL
type LevelEntryRepository struct {
	db *sqlx.DB
}

// NewLevelEntryRepository creates a new LevelEntryRepository
func NewLevelEntryRepository(db *sqlx.DB) repository.LevelEntryRepository {
	return &LevelEntryRepository{db: db}
}

// CreateMany creates multiple level entries for an execution plan
func (r *LevelEntryRepository) CreateMany(ctx context.Context, entries []domain.LevelEntry) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO level_entries (id, execution_plan_id, fib_level, price, quantity, description)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i := range entries {
		if entries[i].ID == "" {
			entries[i].ID = uuid.New().String()
		}

		_, err = stmt.ExecContext(ctx,
			entries[i].ID,
			entries[i].ExecutionPlanID,
			entries[i].FibLevel,
			entries[i].Price,
			entries[i].Quantity,
			entries[i].Description,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetByExecutionPlanID retrieves all level entries for an execution plan
func (r *LevelEntryRepository) GetByExecutionPlanID(ctx context.Context, planID string) ([]domain.LevelEntry, error) {
	var entries []domain.LevelEntry

	query := `
		SELECT id, execution_plan_id, fib_level, price, quantity, description
		FROM level_entries
		WHERE execution_plan_id = $1
		ORDER BY fib_level
	`

	err := r.db.SelectContext(ctx, &entries, query, planID)
	if err != nil {
		return nil, err
	}

	return entries, nil
}

// DeleteByExecutionPlanID deletes all level entries for an execution plan
func (r *LevelEntryRepository) DeleteByExecutionPlanID(ctx context.Context, planID string) error {
	query := `DELETE FROM level_entries WHERE execution_plan_id = $1`
	_, err := r.db.ExecContext(ctx, query, planID)
	return err
}

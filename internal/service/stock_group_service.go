package service

import (
	"context"
	"errors"
	"fmt"
	dto "setbull_trader/internal/core/dto/response"
	"setbull_trader/internal/domain"
	"setbull_trader/internal/repository/postgres"
	"setbull_trader/pkg/log"

	"github.com/google/uuid"
)

var (
	ErrMaxStocksPerGroup      = errors.New("cannot have more than 5 stocks in a group")
	ErrDuplicateGroup         = errors.New("duplicate group for entry type and stocks")
	ErrGroupExecutionConflict = errors.New("another group is already executing or pending")
)

type StockGroupService struct {
	repo             *postgres.StockGroupRepository
	orderExecService *OrderExecutionService
	stockService     *StockService
}

func NewStockGroupService(
	repo *postgres.StockGroupRepository,
	orderExecService *OrderExecutionService,
	stockService *StockService,
) *StockGroupService {
	return &StockGroupService{repo: repo, orderExecService: orderExecService, stockService: stockService}
}

func (s *StockGroupService) CreateGroup(ctx context.Context, entryType string, stockIDs []string) (*domain.StockGroup, error) {
	if len(stockIDs) == 0 || len(stockIDs) > 5 {
		return nil, ErrMaxStocksPerGroup
	}
	// Check for duplicate group (same entryType and stocks)
	groups, err := s.repo.List(ctx, entryType, "")
	if err != nil {
		return nil, err
	}
	for _, g := range groups {
		if sameStocks(g.Stocks, stockIDs) {
			return nil, ErrDuplicateGroup
		}
	}
	groupID := uuid.NewString()
	stocks := make([]domain.StockGroupStock, len(stockIDs))
	for i, sid := range stockIDs {
		stocks[i] = domain.StockGroupStock{
			ID:      uuid.NewString(),
			GroupID: groupID,
			StockID: sid,
		}
	}
	group := &domain.StockGroup{
		ID:        groupID,
		EntryType: entryType,
		Status:    domain.GroupPending,
		Stocks:    stocks,
	}
	if err := s.repo.Create(ctx, group); err != nil {
		return nil, err
	}
	return group, nil
}

func (s *StockGroupService) EditGroup(ctx context.Context, groupID string, stockIDs []string) error {
	if len(stockIDs) == 0 || len(stockIDs) > 5 {
		return ErrMaxStocksPerGroup
	}
	group, err := s.repo.GetByID(ctx, groupID)
	if err != nil || group == nil {
		return fmt.Errorf("group not found: %w", err)
	}
	// Replace stocks
	stocks := make([]domain.StockGroupStock, len(stockIDs))
	for i, sid := range stockIDs {
		stocks[i] = domain.StockGroupStock{
			ID:      uuid.NewString(),
			GroupID: groupID,
			StockID: sid,
		}
	}
	group.Stocks = stocks
	return s.repo.Update(ctx, group)
}

func (s *StockGroupService) DeleteGroup(ctx context.Context, groupID string) error {
	return s.repo.Delete(ctx, groupID)
}

func (s *StockGroupService) ListGroups(ctx context.Context, entryType string, status domain.StockGroupStatus) ([]domain.StockGroup, error) {
	return s.repo.List(ctx, entryType, status)
}

func (s *StockGroupService) ExecuteGroup(ctx context.Context, groupID string) error {
	// Only one group can be executing or pending
	active, err := s.repo.GetActiveOrExecutingGroup(ctx)
	if err != nil {
		return err
	}
	if active != nil && active.ID != groupID {
		return ErrGroupExecutionConflict
	}
	group, err := s.repo.GetByID(ctx, groupID)
	if err != nil || group == nil {
		return fmt.Errorf("group not found: %w", err)
	}
	group.Status = domain.GroupExecuting
	if err := s.repo.Update(ctx, group); err != nil {
		return err
	}
	// Atomic order placement for all stocks in the group
	var failed bool
	var failReason string
	for _, stockRef := range group.Stocks {
		stock, err := s.stockService.GetStockByID(ctx, stockRef.StockID)
		if err != nil || stock == nil {
			failed = true
			failReason = "stock not found or error"
			break
		}
		_, _, err = s.orderExecService.ExecuteOrdersForStock(ctx, stock.ID)
		if err != nil {
			failed = true
			failReason = err.Error()
			break
		}
	}
	if failed {
		group.Status = domain.GroupFailed
		s.repo.Update(ctx, group)
		return fmt.Errorf("group execution failed: %s", failReason)
	}
	group.Status = domain.GroupCompleted
	return s.repo.Update(ctx, group)
}

// Helper: check if two stock lists are the same (order-insensitive)
func sameStocks(existing []domain.StockGroupStock, ids []string) bool {
	if len(existing) != len(ids) {
		return false
	}
	m := make(map[string]struct{}, len(existing))
	for _, s := range existing {
		m[s.StockID] = struct{}{}
	}
	for _, id := range ids {
		if _, ok := m[id]; !ok {
			return false
		}
	}
	return true
}

func (s *StockGroupService) ListGroupsEnriched(
	ctx context.Context,
	entryType string,
	status domain.StockGroupStatus,
	universeService *StockUniverseService,
) ([]dto.StockGroupResponse, error) {
	groups, err := s.ListGroups(ctx, entryType, status)
	if err != nil {
		return nil, err
	}

	var result []dto.StockGroupResponse
	for _, group := range groups {
		var stocks []dto.StockGroupStockDTO
		for _, stockRef := range group.Stocks {
			stockDTO := dto.StockGroupStockDTO{
				StockID: stockRef.StockID,
			}
			// Get symbol
			stock, err := s.stockService.GetStockByID(ctx, stockRef.StockID)
			if err == nil && stock != nil {
				stockDTO.Symbol = stock.Symbol
				// Get instrument key and exchange token from universe
				if universeService != nil {
					univ, err := universeService.GetStockBySymbol(ctx, stock.Symbol)
					if err == nil && univ != nil {
						stockDTO.InstrumentKey = univ.InstrumentKey
						stockDTO.ExchangeToken = univ.ExchangeToken
					}
				}
			}
			stocks = append(stocks, stockDTO)
		}
		result = append(result, dto.StockGroupResponse{
			ID:        group.ID,
			EntryType: group.EntryType,
			Status:    string(group.Status),
			CreatedAt: group.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: group.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Stocks:    stocks,
		})
	}
	return result, nil
}

func (s *StockGroupService) FetchAllStocksFromAllGroups(
	ctx context.Context,
	universeService *StockUniverseService,
) ([]dto.StockGroupStockDTO, error) {
	groups, err := s.ListGroups(ctx, "", "")
	if err != nil {
		return nil, err
	}
	var stocks []dto.StockGroupStockDTO
	log.Info("Fetching %d groups", len(groups))
	for _, group := range groups {
		for _, stockRef := range group.Stocks {
			stockDTO := dto.StockGroupStockDTO{
				StockID: stockRef.StockID,
			}
			stock, err := s.stockService.GetOnlyStockByID(ctx, stockRef.StockID)
			if err == nil && stock != nil {
				stockDTO.Symbol = stock.Symbol
				if universeService != nil {
					univ, err := universeService.GetStockBySymbol(ctx, stock.Symbol)
					if err == nil && univ != nil {
						stockDTO.InstrumentKey = univ.InstrumentKey
						stockDTO.ExchangeToken = univ.ExchangeToken
					}
				}
			}
			stocks = append(stocks, stockDTO)
		}

	}
	return stocks, nil
}

func (s *StockGroupService) GetGroupByID(
	ctx context.Context,
	groupID string,
	universeService *StockUniverseService,
) (dto.StockGroupResponse, error) {
	group, err := s.repo.GetByID(ctx, groupID)
	if err != nil {
		return dto.StockGroupResponse{}, err
	}
	var stocks []dto.StockGroupStockDTO
	for _, stockRef := range group.Stocks {
		stockDTO := dto.StockGroupStockDTO{
			StockID: stockRef.StockID,
		}
		stock, err := s.stockService.GetStockByID(ctx, stockRef.StockID)
		if err == nil && stock != nil {
			stockDTO.Symbol = stock.Symbol
			if universeService != nil {
				univ, err := universeService.GetStockBySymbol(ctx, stock.Symbol)
				if err == nil && univ != nil {
					stockDTO.InstrumentKey = univ.InstrumentKey
					stockDTO.ExchangeToken = univ.ExchangeToken
				}
			}
			stocks = append(stocks, stockDTO)
		}
	}
	return dto.StockGroupResponse{
		ID:        group.ID,
		EntryType: group.EntryType,
		Status:    string(group.Status),
		CreatedAt: group.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: group.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Stocks:    stocks,
	}, nil
}

// GetGroupsByEntryType fetches all groups with the given entry type
func (s *StockGroupService) GetGroupsByEntryType(
	ctx context.Context,
	entryType string,
	universeService *StockUniverseService,
) ([]dto.StockGroupResponse, error) {
	groups, err := s.repo.List(ctx, entryType, "")
	if err != nil {
		return nil, err
	}

	var result []dto.StockGroupResponse
	for _, group := range groups {
		var stocks []dto.StockGroupStockDTO
		for _, stockRef := range group.Stocks {
			stockDTO := dto.StockGroupStockDTO{
				StockID: stockRef.StockID,
			}
			stock, err := s.stockService.GetStockByID(ctx, stockRef.StockID)
			if err == nil && stock != nil {
				stockDTO.Symbol = stock.Symbol
				if universeService != nil {
					univ, err := universeService.GetStockBySymbol(ctx, stock.Symbol)
					if err == nil && univ != nil {
						stockDTO.InstrumentKey = univ.InstrumentKey
						stockDTO.ExchangeToken = univ.ExchangeToken
					}
				}
			}
			stocks = append(stocks, stockDTO)
		}
		result = append(result, dto.StockGroupResponse{
			ID:        group.ID,
			EntryType: group.EntryType,
			Status:    string(group.Status),
			Stocks:    stocks,
		})
	}
	return result, nil
}

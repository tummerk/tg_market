package worker

import (
	"context"
	"time"

	"tg_market/internal/domain/entity"
	service "tg_market/internal/domain/service/gift"
)

type GiftTypeRepository interface {
	List(ctx context.Context, limit, offset int) ([]entity.GiftType, error)
	GetByID(ctx context.Context, id int64) (*entity.GiftType, error)
}

type MarketScanner struct {
	giftService  *service.GiftService
	giftTypeRepo GiftTypeRepository
	deals        chan<- service.GoodDeal
	interval     time.Duration
	giftTypeIDs  []int64
}

func NewMarketScanner(
	giftService *service.GiftService,
	giftTypeRepo GiftTypeRepository,
	deals chan<- service.GoodDeal,
	interval time.Duration,
) *MarketScanner {
	return &MarketScanner{
		giftService:  giftService,
		giftTypeRepo: giftTypeRepo,
		deals:        deals,
		interval:     interval,
	}
}

// WithGiftTypes ограничивает сканирование конкретными типами.
func (w *MarketScanner) WithGiftTypes(ids ...int64) *MarketScanner {
	w.giftTypeIDs = ids
	return w
}

func (w *MarketScanner) Run(ctx context.Context) error {
	logger(ctx).Info("market scanner started", "interval", w.interval, "types", w.giftTypeIDs)

	w.scanAll(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger(ctx).Info("market scanner stopped")
			return ctx.Err()
		case <-ticker.C:
			w.scanAll(ctx)
		}
	}
}

func (w *MarketScanner) scanAll(ctx context.Context) {
	giftTypes, err := w.getGiftTypes(ctx)
	if err != nil {
		logger(ctx).Error("failed to get gift types", "error", err)
		return
	}

	logger(ctx).Debug("scanning market", "types_count", len(giftTypes))

	var dealsFound int

	for _, gt := range giftTypes {
		select {
		case <-ctx.Done():
			return
		default:
		}

		count, err := w.scanOne(ctx, &gt)
		if err != nil {
			logger(ctx).Error("scan failed", "id", gt.ID, "name", gt.Name, "error", err)
			continue
		}

		dealsFound += count
		time.Sleep(500 * time.Millisecond)
	}

	if dealsFound > 0 {
		logger(ctx).Info("scan completed", "deals_found", dealsFound)
	}
}

func (w *MarketScanner) getGiftTypes(ctx context.Context) ([]entity.GiftType, error) {
	// Если указаны конкретные типы — загружаем только их
	if len(w.giftTypeIDs) > 0 {
		var result []entity.GiftType
		for _, id := range w.giftTypeIDs {
			gt, err := w.giftTypeRepo.GetByID(ctx, id)
			if err != nil {
				return nil, err
			}
			result = append(result, *gt)
		}
		return result, nil
	}

	// Иначе — все
	return w.giftTypeRepo.List(ctx, 100, 0)
}

func (w *MarketScanner) scanOne(ctx context.Context, giftType *entity.GiftType) (int, error) {

	avgPrice, err := w.giftService.GetGiftAveragePrice(ctx, giftType.ID)
	if err != nil {
		return 0, err
	}

	giftType.AveragePrice = avgPrice

	deals, err := w.giftService.CheckMarketForType(ctx, giftType)
	if err != nil {
		return 0, err
	}

	for _, deal := range deals {
		select {
		case w.deals <- deal:
		case <-ctx.Done():
			return len(deals), ctx.Err()
		}
	}

	return len(deals), nil
}

package worker

import (
	"context"
	"fmt"
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
	giftTypeIDs  []int64

	// Rate control
	requestInterval time.Duration // интервал между запросами (gap)
	lastRequest     time.Time
}

func NewMarketScanner(
	giftService *service.GiftService,
	giftTypeRepo GiftTypeRepository,
	deals chan<- service.GoodDeal,
) *MarketScanner {
	return &MarketScanner{
		giftService:     giftService,
		giftTypeRepo:    giftTypeRepo,
		deals:           deals,
		requestInterval: 750 * time.Millisecond, // дефолт
	}
}

func (w *MarketScanner) WithGiftTypes(ids ...int64) *MarketScanner {
	w.giftTypeIDs = ids
	return w
}

// WithRateControl устанавливает интервал между запросами.
// Для 2 клиентов и ratePerClient=1500ms → interval=750ms.
func (w *MarketScanner) WithRateControl(ratePerClient time.Duration, clientCount int) *MarketScanner {
	if clientCount > 0 {
		w.requestInterval = ratePerClient / time.Duration(clientCount)
	}
	return w
}

func (w *MarketScanner) Run(ctx context.Context) error {
	logger(ctx).Info("market scanner started",
		"types", w.giftTypeIDs,
		"request_interval", w.requestInterval,
	)

	for {
		select {
		case <-ctx.Done():
			logger(ctx).Info("market scanner stopped")
			return ctx.Err()
		default:
			w.scanAll(ctx)
		}
	}
}

// waitForNextSlot ждёт до следующего слота для запроса.
func (w *MarketScanner) waitForNextSlot(ctx context.Context) error {
	if w.lastRequest.IsZero() {
		w.lastRequest = time.Now()
		return nil
	}

	elapsed := time.Since(w.lastRequest)
	if elapsed >= w.requestInterval {
		w.lastRequest = time.Now()
		return nil
	}

	wait := w.requestInterval - elapsed

	select {
	case <-time.After(wait):
		w.lastRequest = time.Now()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *MarketScanner) scanAll(ctx context.Context) {
	giftTypes, err := w.getGiftTypes(ctx)
	if err != nil {
		logger(ctx).Error("failed to get gift types", "error", err)
		return
	}

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
	}

	if dealsFound > 0 {
		logger(ctx).Info("scan cycle completed", "deals_found", dealsFound)
	}
}

func (w *MarketScanner) getGiftTypes(ctx context.Context) ([]entity.GiftType, error) {
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

	return w.giftTypeRepo.List(ctx, 100, 0)
}

func (w *MarketScanner) scanOne(ctx context.Context, giftType *entity.GiftType) (int, error) {
	// Ждём слот перед GetGiftAveragePrice
	if err := w.waitForNextSlot(ctx); err != nil {
		return 0, err
	}

	now := time.Now()
	fmt.Printf("[%s] 🔍 Scan %s\n", now.Format("15:04:05.000"), giftType.Name)

	avgPrice, err := w.giftService.GetGiftAveragePrice(ctx, giftType.ID)
	if err != nil {
		return 0, err
	}

	giftType.AveragePrice = avgPrice

	// Ждём слот перед CheckMarketForType
	if err := w.waitForNextSlot(ctx); err != nil {
		return 0, err
	}

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

package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"tg_market/internal/domain/entity"
	service "tg_market/internal/domain/service/gift"
)

type GiftTypeRepository interface {
	List(ctx context.Context, limit, offset int) ([]entity.GiftType, error)
	GetByID(ctx context.Context, id int64) (*entity.GiftType, error)
}

type MarketScanner struct {
	giftService *service.GiftService
	deals       chan<- entity.Deal
	giftTypeIDs []int64

	requestInterval time.Duration
	lastRequest     time.Time

	// Control fields
	mu         sync.Mutex
	cancelFunc context.CancelFunc
	isRunning  bool
	wg         sync.WaitGroup
}

func NewMarketScanner(
	giftService *service.GiftService,
	giftTypeRepo GiftTypeRepository,
	deals chan<- entity.Deal,
) *MarketScanner {
	return &MarketScanner{
		giftService:     giftService,
		deals:           deals,
		requestInterval: 750 * time.Millisecond,
	}
}

func (w *MarketScanner) WithGiftTypes(ids ...int64) *MarketScanner {
	w.giftTypeIDs = ids
	return w
}

func (w *MarketScanner) WithRateControl(ratePerClient time.Duration, clientCount int) *MarketScanner {
	if clientCount > 0 {
		w.requestInterval = ratePerClient / time.Duration(clientCount)
	}
	return w
}

func (w *MarketScanner) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.isRunning {
		return errors.New("scanner is already running")
	}

	scanCtx, cancel := context.WithCancel(ctx)
	w.cancelFunc = cancel
	w.isRunning = true

	w.wg.Add(1) // ✅ Увеличиваем счётчик
	go func() {
		defer w.wg.Done() // ✅ Уменьшаем при завершении
		defer func() {
			w.mu.Lock()
			w.isRunning = false
			w.cancelFunc = nil
			w.mu.Unlock()
		}()

		if err := w.Run(scanCtx); err != nil && !errors.Is(err, context.Canceled) {
			fmt.Printf("Scanner stopped with error: %v\n", err)
		}
	}()

	return nil
}

func (w *MarketScanner) Stop() {
	w.mu.Lock()

	if !w.isRunning {
		w.mu.Unlock()
		return
	}

	if w.cancelFunc != nil {
		w.cancelFunc()
	}
	w.mu.Unlock()

	w.wg.Wait()
}

// IsRunning возвращает текущий статус
func (w *MarketScanner) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.isRunning
}

func (w *MarketScanner) Run(ctx context.Context) error {
	// logger(ctx).Info...
	fmt.Println("🚀 Market Scanner STARTED")

	for {
		select {
		case <-ctx.Done():
			fmt.Println("🛑 Market Scanner STOPPED")
			return ctx.Err()
		default:
			w.scanAll(ctx)
		}
	}
}

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

		count, err := w.scanOne(ctx, gt)
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
		result := make([]entity.GiftType, 0, len(w.giftTypeIDs))
		for _, id := range w.giftTypeIDs {
			gt, err := w.giftService.GetGiftType(ctx, id)
			if err != nil {
				return nil, err
			}
			result = append(result, *gt)
		}
		return result, nil
	}

	return w.giftService.ListGiftTypes(ctx, 100, 0)
}

func (w *MarketScanner) scanOne(ctx context.Context, giftType entity.GiftType) (int, error) { // Изменено: значение вместо указателя
	if err := w.waitForNextSlot(ctx); err != nil {
		return 0, err
	}

	now := time.Now()
	fmt.Printf("[%s] 🔍 Scan %s  discount=%.2f\n", now.Format("15:04:05.000"), giftType.Name, w.giftService.GetDiscount())

	avgPrice, err := w.giftService.GetGiftAveragePrice(ctx, giftType.ID)
	if err != nil {
		return 0, err
	}

	giftType.AveragePrice = avgPrice

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

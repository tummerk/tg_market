package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/patrickmn/go-cache"
	"tg_market/internal/domain"
	"tg_market/internal/domain/entity"
	"tg_market/pkg/errcodes"
	"time"
)

const (
	priceCacheTTL       = 5 * time.Minute
	defaultPricesSample = 10
)

type TgClient interface {
	GetGiftTypes(ctx context.Context, hash int) ([]entity.GiftType, error)
	GetLastPrices(ctx context.Context, giftTypeID int, limit int) ([]int, error)
	GetMarketGifts(ctx context.Context, giftTypeID int64, limit int) ([]entity.Gift, error)
}

type GiftTypeRepository interface {
	Create(ctx context.Context, gift *entity.GiftType) error
	GetByID(ctx context.Context, id int64) (*entity.GiftType, error)
	Update(ctx context.Context, gift *entity.GiftType) error
	UpdateStats(ctx context.Context, id int64, floorPrice, avgPrice int64, quantity int) error
	UpdatePriceStats(ctx context.Context, id int64, avgPrice int64) error
	DecreaseSupply(ctx context.Context, id int64) error
	List(ctx context.Context, limit, offset int) ([]entity.GiftType, error)
}

type GiftRepository interface {
	Create(ctx context.Context, gift *entity.Gift) error
	CreateBatch(ctx context.Context, gifts []*entity.Gift) error
	GetByID(ctx context.Context, id int64) (*entity.Gift, error)
	GetByIDs(ctx context.Context, ids []int64) ([]*entity.Gift, error)
	UpdateOwner(ctx context.Context, giftID, newOwnerID int64) error
	UpdatePrice(ctx context.Context, giftID int64, price *int64) error
	TransferGift(ctx context.Context, giftID, fromUserID, toUserID int64) error
	Exists(ctx context.Context, id int64) (bool, error)
}

type GiftService struct {
	giftTypeRepo       GiftTypeRepository
	giftRepo           GiftRepository
	tgClient           TgClient
	minDiscountPercent float64
	maxOffersToCheck   int
	processedCache     *cache.Cache
}

func NewGiftService(
	giftTypeRepo GiftTypeRepository,
	giftRepo GiftRepository,
	tgClient TgClient,
) *GiftService {
	return &GiftService{
		giftTypeRepo:       giftTypeRepo,
		giftRepo:           giftRepo,
		tgClient:           tgClient,
		minDiscountPercent: 20.0,
		maxOffersToCheck:   20,
		processedCache:     cache.New(time.Hour, priceCacheTTL),
	}
}

func (s *GiftService) WithDiscountThreshold(percent float64) *GiftService {
	s.minDiscountPercent = percent
	return s
}

type SyncResult struct {
	Created int
	Updated int
	Errors  int
}

type GoodDeal struct {
	Gift       entity.Gift
	GiftType   entity.GiftType
	FloorPrice int64
	AvgPrice   int64
	Discount   float64
}

func (s *GiftService) SyncCatalog(ctx context.Context) (SyncResult, error) {
	logger(ctx).Info("syncing catalog started")

	remoteGifts, err := s.tgClient.GetGiftTypes(ctx, 0)
	if err != nil {
		return SyncResult{}, fmt.Errorf("fetch gift types: %w", err)
	}

	logger(ctx).Info("fetched gifts from TG", "count", len(remoteGifts))

	var result SyncResult

	for _, remote := range remoteGifts {
		created, err := s.syncGiftType(ctx, remote)
		if err != nil {
			logger(ctx).Error("failed to sync gift", "id", remote.ID, "error", err)
			result.Errors++
			continue
		}

		if created {
			result.Created++
		} else {
			result.Updated++
		}
	}

	logger(ctx).Info("syncing catalog finished",
		"created", result.Created,
		"updated", result.Updated,
		"errors", result.Errors,
	)

	return result, nil
}

func (s *GiftService) syncGiftType(ctx context.Context, remote entity.GiftType) (created bool, err error) {
	existing, err := s.giftTypeRepo.GetByID(ctx, remote.ID)
	if err != nil {
		var appErr *domain.AppError
		if errors.As(err, &appErr) && appErr.Code == errcodes.GiftNotFound {
			if err := s.giftTypeRepo.Create(ctx, &remote); err != nil {
				return false, fmt.Errorf("create: %w", err)
			}
			return true, nil
		}
		return false, fmt.Errorf("get existing: %w", err)
	}

	remote.MarketFloorPrice = existing.MarketFloorPrice
	remote.AveragePrice = existing.AveragePrice
	remote.MarketQuantity = existing.MarketQuantity

	if remote.Name == "" {
		remote.Name = existing.Name
	}

	if err := s.giftTypeRepo.Update(ctx, &remote); err != nil {
		return false, fmt.Errorf("update: %w", err)
	}

	return false, nil
}

// CheckMarketForType сканирует рынок, сохраняет выгодные подарки в БД и отправляет в канал.
func (s *GiftService) CheckMarketForType(ctx context.Context, giftType *entity.GiftType) ([]GoodDeal, error) {
	if giftType.AveragePrice <= 0 {
		return nil, nil
	}

	gifts, err := s.tgClient.GetMarketGifts(ctx, giftType.ID, s.maxOffersToCheck)
	if err != nil {
		return nil, fmt.Errorf("get market gifts: %w", err)
	}

	var deals []GoodDeal

	for _, gift := range gifts {
		giftIDStr := fmt.Sprint(gift.ID)

		if _, found := s.processedCache.Get(giftIDStr); found {
			continue
		}

		deal, isGood := s.evaluateGift(giftType, gift)
		if !isGood {
			s.processedCache.Set(giftIDStr, true, cache.DefaultExpiration)
			continue
		}

		exists, err := s.giftRepo.Exists(ctx, gift.ID)
		if err != nil {
			logger(ctx).Error("db check failed", "err", err)
			continue
		}

		if exists {
			s.processedCache.Set(giftIDStr, true, cache.DefaultExpiration)
			continue
		}

		if err := s.giftRepo.Create(ctx, &gift); err != nil {
			logger(ctx).Error("failed to save gift", "err", err)
			continue
		}

		s.processedCache.Set(giftIDStr, true, cache.DefaultExpiration)

		deals = append(deals, deal)
	}

	return deals, nil
}

func (s *GiftService) evaluateGift(giftType *entity.GiftType, gift entity.Gift) (GoodDeal, bool) {
	// 1. База для сравнения — СРЕДНЯЯ цена (как ты просил)
	benchmarkPrice := giftType.AveragePrice

	// Если средней цены нет (не собрали статистику), мы не можем оценить выгоду.
	if benchmarkPrice <= 0 || gift.Price <= 0 {
		return GoodDeal{}, false
	}

	if gift.Price >= benchmarkPrice {
		return GoodDeal{}, false
	}

	profit := benchmarkPrice - gift.Price
	discountPercent := float64(profit) / float64(benchmarkPrice) * 100

	// 3. Сравниваем с порогом (например, 20.0)
	if discountPercent < s.minDiscountPercent {
		return GoodDeal{}, false
	}

	return GoodDeal{
		Gift:       gift,
		GiftType:   *giftType,
		FloorPrice: giftType.MarketFloorPrice, // Оставляем просто для информации
		AvgPrice:   benchmarkPrice,
		Discount:   discountPercent, // Возвращаем 25.0, а не 0.25
	}, true
}

func (s *GiftService) GetGiftAveragePrice(ctx context.Context, giftTypeID int64) (int64, error) {
	giftType, err := s.giftTypeRepo.GetByID(ctx, giftTypeID)
	if err != nil {
		return 0, fmt.Errorf("get gift type: %w", err)
	}

	if s.isPriceCacheValid(giftType) {
		return giftType.AveragePrice, nil
	}

	// Запрашиваем из TG
	avgPrice, err := s.fetchAndCalcAverage(ctx, giftTypeID)
	if err != nil {
		if giftType.AveragePrice > 0 {
			logger(ctx).Warn("failed to fetch prices, using cached",
				"gift_type_id", giftTypeID,
				"cached_price", giftType.AveragePrice,
				"error", err,
			)
			return giftType.AveragePrice, nil
		}
		return 0, fmt.Errorf("fetch prices: %w", err)
	}

	// Сохраняем в БД
	if err := s.giftTypeRepo.UpdatePriceStats(ctx, giftTypeID, avgPrice); err != nil {
		logger(ctx).Error("failed to update price stats", "error", err)
	}

	return avgPrice, nil
}

func (s *GiftService) isPriceCacheValid(giftType *entity.GiftType) bool {
	if giftType.AveragePrice <= 0 {
		return false
	}
	return time.Since(giftType.PriceUpdatedAt) < priceCacheTTL
}

func (s *GiftService) fetchAndCalcAverage(ctx context.Context, giftTypeID int64) (int64, error) {
	prices, err := s.tgClient.GetLastPrices(ctx, int(giftTypeID), defaultPricesSample)
	if err != nil {
		return 0, err
	}

	if len(prices) == 0 {
		return 0, nil
	}

	return calcAverage(prices), nil
}

func calcAverage(prices []int) int64 {
	if len(prices) == 0 {
		return 0
	}

	var sum int
	for _, p := range prices {
		sum += p
	}

	return int64(sum / len(prices))
}

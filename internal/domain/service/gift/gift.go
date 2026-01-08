package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
	"tg_market/internal/domain"
	"tg_market/internal/domain/entity"
	"tg_market/internal/domain/service/numRating"
	"tg_market/pkg/errcodes"
)

const (
	priceCacheTTL   = 5 * time.Minute
	countToAvgPrice = 10
)

type TgClient interface {
	GetGiftTypes(ctx context.Context, hash int) ([]entity.GiftType, error)
	GetLastPrices(ctx context.Context, giftTypeID int, limit int) ([]int, error)
	GetMarketDeals(ctx context.Context, giftTypeID int64, limit int) ([]entity.Deal, error)
	GetGiftsPage(ctx context.Context, giftID int64, offset string, limit int) ([]entity.Gift, string, error)
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
		maxOffersToCheck:   5,
		processedCache:     cache.New(time.Hour, priceCacheTTL),
	}
}

func (s *GiftService) WithDiscountThreshold(percent float64) *GiftService {
	s.minDiscountPercent = percent
	return s
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

// CheckMarketForType сканирует рынок и возвращает выгодные сделки.
func (s *GiftService) CheckMarketForType(ctx context.Context, giftType entity.GiftType) ([]entity.Deal, error) {
	if giftType.AveragePrice <= 0 {
		return nil, nil
	}

	// Клиент возвращает Deal с заполненным Gift и SellerAccessHash
	deals, err := s.tgClient.GetMarketDeals(ctx, giftType.ID, s.maxOffersToCheck)
	if err != nil {
		return nil, fmt.Errorf("get market deals: %w", err)
	}

	var goodDeals []entity.Deal

	for i := range deals {
		deal := &deals[i]
		giftIDStr := fmt.Sprint(deal.Gift.ID)

		// Уже обрабатывали?
		if _, found := s.processedCache.Get(giftIDStr); found {
			continue
		}
		// Обогащаем Deal бизнес-данными и проверяем выгодность
		if !s.enrichAndEvaluate(deal, giftType) {
			s.processedCache.Set(giftIDStr, true, cache.DefaultExpiration)
			continue
		}

		// Проверяем, есть ли уже в БД
		exists, err := s.giftRepo.Exists(ctx, deal.Gift.ID)
		if err != nil {
			logger(ctx).Error("db check failed", "err", err)
			continue
		}

		if exists {
			s.processedCache.Set(giftIDStr, true, cache.DefaultExpiration)
			continue
		}

		// Сохраняем подарок
		if err := s.giftRepo.Create(ctx, deal.Gift); err != nil {
			logger(ctx).Error("failed to save gift", "err", err)
			continue
		}

		s.processedCache.Set(giftIDStr, true, cache.DefaultExpiration)

		goodDeals = append(goodDeals, *deal)
	}

	return goodDeals, nil
}

// enrichAndEvaluate обогащает Deal бизнес-данными и возвращает true, если сделка выгодная.
func (s *GiftService) enrichAndEvaluate(deal *entity.Deal, giftType entity.GiftType) bool {
	benchmarkPrice := giftType.AveragePrice

	// Нет данных для оценки
	if benchmarkPrice <= 0 || deal.Gift.StarPrice <= 0 {
		return false
	}

	// Цена не ниже средней — не интересно
	if deal.Gift.StarPrice >= benchmarkPrice {
		return false
	}

	// Считаем профит
	profit := benchmarkPrice - deal.Gift.StarPrice
	discountPercent := float64(profit) / float64(benchmarkPrice) * 100

	// Не проходит порог
	if discountPercent < s.minDiscountPercent {
		return false
	}

	// Обогащаем Deal
	deal.GiftType = &giftType
	deal.AvgPrice = benchmarkPrice
	deal.Profit = discountPercent

	return true
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
	prices, err := s.tgClient.GetLastPrices(ctx, int(giftTypeID), countToAvgPrice)
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

func (s *GiftService) UpdateAllAveragePrices(ctx context.Context) (int, error) {
	const batchSize = 50
	const requestDelay = 1500 * time.Millisecond // Пауза, чтобы не душить API

	offset := 0
	updatedCount := 0

	logger(ctx).Info("starting bulk price update")

	for {
		// 1. Получаем пачку подарков из БД
		giftTypes, err := s.giftTypeRepo.List(ctx, batchSize, offset)
		if err != nil {
			return updatedCount, fmt.Errorf("failed to list gift types: %w", err)
		}

		if len(giftTypes) == 0 {
			break // Всё обработали
		}

		for _, gift := range giftTypes {
			// 2. Получаем новую среднюю цену (используем существующий приватный метод)
			newAvgPrice, err := s.fetchAndCalcAverage(ctx, gift.ID)
			if err != nil {
				// Логируем ошибку, но не прерываем весь процесс
				logger(ctx).Error("failed to fetch price for gift",
					"id", gift.ID,
					"name", gift.Name,
					"error", err,
				)
				continue
			}

			// Если цена = 0 (нет продаж), пропускаем или обновляем (зависит от логики, тут пропускаем)
			if newAvgPrice == 0 {
				continue
			}

			// 3. Сохраняем в БД
			if err := s.giftTypeRepo.UpdatePriceStats(ctx, gift.ID, newAvgPrice); err != nil {
				logger(ctx).Error("failed to update price stats in db", "id", gift.ID, "error", err)
				continue
			}

			updatedCount++

			// Анти-флуд пауза
			time.Sleep(requestDelay)
		}

		offset += batchSize
	}

	logger(ctx).Info("bulk price update finished", "updated_total", updatedCount)
	return updatedCount, nil
}

// ProcessGiftsByRating полностью проходит по всем подаркам одного типа и сохраняет в БД
// только те, что имеют рейтинг выше заданного процента
func (s *GiftService) ProcessGiftsByRating(ctx context.Context, giftTypeID int64, minRatingPercent float64) (int, error) {
	logger(ctx).Info("starting to process gifts by rating",
		"gift_type_id", giftTypeID,
		"min_rating_percent", minRatingPercent)

	// Используем пагинацию для получения всех подарков этого типа
	const batchSize = 500
	processedCount := 0
	var offset string // строковое смещение для новой функции
	countGoodNum := 0

	for {
		// Получаем подарки с помощью нового метода из Telegram клиента
		// Нам нужно получить доступ к Telegram клиенту через интерфейс
		gifts, nextOffset, err := s.tgClient.GetGiftsPage(ctx, giftTypeID, offset, batchSize)
		if err != nil {
			return processedCount, fmt.Errorf("failed to get gifts batch: %w", err)
		}

		// Если нет подарков, выходим из цикла
		if len(gifts) == 0 {
			break
		}

		// Обрабатываем каждый подарок
		for _, gift := range gifts {
			// Вычисляем рейтинг для номера подарка
			rating := numRating.CalculateValue(gift.Num)

			// Проверяем, удовлетворяет ли рейтинг минимальному порогу
			if rating.Score < minRatingPercent {
				continue
			}
			countGoodNum++
			// Логируем хорошие номера на уровне debug
			logger(ctx).Debug("found high-rated gift",
				"gift_id", gift.ID,
				"gift_num", gift.Num,
				"rating", rating.Score,
				"description", rating.Description)

			// Устанавливаем рейтинг в поле NumRating (округляем до целого)
			gift.NumRating = int(rating.Score)

			// Проверяем, существует ли уже такой подарок в БД
			exists, err := s.giftRepo.Exists(ctx, gift.ID)
			if err != nil {
				logger(ctx).Error("failed to check if gift exists", "gift_id", gift.ID, "error", err)
				continue
			}

			if !exists {
				// Сохраняем подарок в БД
				if err := s.giftRepo.Create(ctx, &gift); err != nil {
					logger(ctx).Error("failed to save gift to DB", "gift_id", gift.ID, "error", err)
					continue
				}
				logger(ctx).Debug("saved high-rated gift to DB",
					"gift_id", gift.ID,
					"gift_num", gift.Num,
					"rating", rating.Score)
			} else {
				logger(ctx).Debug("high-rated gift already exists in DB",
					"gift_id", gift.ID,
					"gift_num", gift.Num,
					"rating", rating.Score)
			}
		}

		// Обновляем счетчики
		processedCount += len(gifts)

		// Если nextOffset пустой, значит это была последняя страница
		if nextOffset == "" {
			break
		}

		// Обновляем смещение для следующей итерации
		offset = nextOffset

		// Делаем паузу, чтобы не перегружать API
		time.Sleep(100 * time.Millisecond)
	}

	logger(ctx).Info("finished processing gifts by rating",
		"gift_type_id", giftTypeID,
		"total_processed", processedCount,
		"min_rating_percent", minRatingPercent)

	return processedCount, nil
}

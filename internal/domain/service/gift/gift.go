package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"tg_market/internal/domain"
	"tg_market/internal/domain/entity"
	"tg_market/internal/domain/service/numRating"
	"tg_market/pkg/errcodes"

	"github.com/patrickmn/go-cache"
)

const (
	priceCacheTTL             = 5 * time.Minute
	countToAvgPrice           = 10
	defaultMaxOffersToCheck   = 20
	defaultMinDiscountPercent = 20.0
)

type TgClient interface {
	GetGiftTypes(ctx context.Context, hash int) ([]entity.GiftType, error)
	GetLastPrices(ctx context.Context, giftTypeID int, limit int) ([]int, error)
	GetMarketDeals(ctx context.Context, giftTypeID int64, limit int) ([]entity.Deal, error)
	GetGiftsPage(ctx context.Context, giftID int64, offset string, limit int) ([]entity.Gift, string, error)
	BuyDeal(ctx context.Context, deal entity.Deal) error
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
	giftTypeRepo GiftTypeRepository
	giftRepo     GiftRepository
	tgClient     TgClient

	autoBuyEnabled     bool
	balance            float64
	minDiscountPercent float64
	maxOffersToCheck   int
	mu                 sync.RWMutex
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
		minDiscountPercent: defaultMinDiscountPercent,
		maxOffersToCheck:   defaultMaxOffersToCheck,
		processedCache:     cache.New(time.Hour, priceCacheTTL),
		autoBuyEnabled:     true,
	}
}

func (s *GiftService) WithDiscountThreshold(percent float64) *GiftService {
	s.minDiscountPercent = percent
	return s
}

func (s *GiftService) SetDiscount(percent float64) {
	s.minDiscountPercent = percent
}

func (s *GiftService) SyncCatalog(ctx context.Context) (domain.SyncResult, error) {
	logger(ctx).Info("syncing catalog started")

	remoteGifts, err := s.tgClient.GetGiftTypes(ctx, 0)
	if err != nil {
		return domain.SyncResult{}, fmt.Errorf("fetch gift types: %w", err)
	}

	logger(ctx).Info("fetched gifts from TG", "count", len(remoteGifts))

	var result domain.SyncResult

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

	// 1. Получаем сделки с рынка
	deals, err := s.tgClient.GetMarketDeals(ctx, giftType.ID, s.maxOffersToCheck)
	if err != nil {
		return nil, fmt.Errorf("get market deals: %w", err)
	}

	var goodDeals []entity.Deal
	var newDealsCount int

	for i := range deals {
		deal := &deals[i]
		giftIDStr := fmt.Sprint(deal.Gift.ID)

		// Кэш
		if _, found := s.processedCache.Get(giftIDStr); found {
			continue
		}

		// 2. ОБЩИЙ АНАЛИЗ (Фильтрация мусора)
		// Эта функция решает, стоит ли вообще обращать внимание на лот (добавлять в список/базу)
		isGem, ratingScore := s.analyzeDeal(deal, giftType)

		if !isGem {
			s.processedCache.Set(giftIDStr, true, cache.DefaultExpiration)
			continue
		}

		// 3. Проверка БД
		exists, err := s.giftRepo.Exists(ctx, deal.Gift.ID)
		if err != nil {
			logger(ctx).Error("db check failed", "err", err)
			continue
		}
		if exists {
			s.processedCache.Set(giftIDStr, true, cache.DefaultExpiration)
			continue
		}

		newDealsCount++
		deal.Gift.NumRating = int(ratingScore)

		isBlack := deal.Gift.Attributes.Backdrop == "Black"
		isSuperCheap := deal.Profit > 15.0

		if s.autoBuyEnabled && (isBlack || isSuperCheap) {
			// Запускаем покупку
			go s.AutoBuy(ctx, *deal)

			// Логируем причину покупки
			logger(ctx).Info("🚀 Triggering AutoBuy",
				"id", deal.Gift.ID,
				"reason_black", isBlack,
				"reason_cheap", isSuperCheap,
				"profit", deal.Profit)
		}

		// 5. Сохраняем в историю БД (все интересные лоты, не только купленные)
		if err := s.giftRepo.Create(ctx, deal.Gift); err != nil {
			logger(ctx).Error("failed to save gift", "err", err)
		}

		s.processedCache.Set(giftIDStr, true, cache.DefaultExpiration)

		// Добавляем в возвращаемый слайс, чтобы пришло уведомление/лог
		goodDeals = append(goodDeals, *deal)
	}

	if newDealsCount > 0 {
		logger(ctx).Info("scan cycle stats", "type", giftType.Name, "new_items", newDealsCount, "found_gems", len(goodDeals))
	}

	return goodDeals, nil
}

// analyzeDeal проверяет "мягкие" критерии для уведомлений.
// Сюда попадают: обычные скидки (minDiscountPercent), красивые номера и редкие атрибуты.
func (s *GiftService) analyzeDeal(deal *entity.Deal, giftType entity.GiftType) (bool, float64) {
	deal.GiftType = &giftType
	deal.AvgPrice = giftType.AveragePrice

	// --- КРИТЕРИЙ 1: ЦЕНА (Мягкий фильтр) ---
	isGoodPrice := false
	if giftType.AveragePrice > 0 && deal.Gift.StarPrice > 0 {
		profit := giftType.AveragePrice - deal.Gift.StarPrice
		deal.Profit = float64(profit) / float64(giftType.AveragePrice) * 100

		// Используем стандартный конфиг (например, > 10% или 5%), чтобы просто уведомить
		if deal.Profit >= s.minDiscountPercent {
			isGoodPrice = true
		}
	}

	// --- КРИТЕРИЙ 2: НОМЕР ---
	rating := numRating.CalculateValue(deal.Gift.Num)
	isGoodNumber := rating.Score > 60

	// --- КРИТЕРИЙ 3: АТРИБУТЫ ---
	isRareAttribute := deal.Gift.Attributes.Backdrop == "Black"

	// Если хотя бы одно условие верно — возвращаем true (лот попадет в список и БД)
	if isGoodPrice || isGoodNumber || isRareAttribute {
		return true, rating.Score
	}

	return false, rating.Score
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

// --- Обновленный метод AutoBuy ---
func (s *GiftService) AutoBuy(ctx context.Context, deal entity.Deal) {
	s.mu.RLock()
	limit := s.balance
	s.mu.RUnlock()

	if deal.Gift.TonPrice > limit {
		return
	}

	// Попытка покупки
	err := s.tgClient.BuyDeal(ctx, deal)
	if err != nil {

		return
	}

	s.mu.Lock()
	s.balance -= deal.Gift.TonPrice
	s.mu.Unlock()
}

func (s *GiftService) SetAutoBuy() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.autoBuyEnabled = !s.autoBuyEnabled

	return s.autoBuyEnabled
}

func (s *GiftService) IsAutoBuyEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.autoBuyEnabled
}

func (s *GiftService) SetBalance(amount float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.balance = amount
}

// ListGiftTypes возвращает список типов подарков
func (s *GiftService) ListGiftTypes(ctx context.Context, limit, offset int) ([]entity.GiftType, error) {
	return s.giftTypeRepo.List(ctx, limit, offset)
}

// GetGiftType возвращает тип подарка по ID
func (s *GiftService) GetGiftType(ctx context.Context, id int64) (*entity.GiftType, error) {
	return s.giftTypeRepo.GetByID(ctx, id)
}

// GetBalance возвращает текущий лимит баланса
func (s *GiftService) GetBalance() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.balance
}

// GetDiscount возвращает текущий порог скидки
func (s *GiftService) GetDiscount() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.minDiscountPercent
}

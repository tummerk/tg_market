package telegram

import (
	"context"
	"fmt"
	"github.com/gotd/td/tg"
	"tg_market/internal/domain/entity"
	"tg_market/internal/domain/value"
	"time"
)

func (c *Client) GetLastPrices(ctx context.Context, giftTypeID int, limit int) ([]int, error) {
	req := &tg.PaymentsGetResaleStarGiftsRequest{
		GiftID:      int64(giftTypeID),
		Limit:       limit,
		SortByPrice: true,
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	resRaw, err := c.api.PaymentsGetResaleStarGifts(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get resale gifts: %w", err)
	}

	var gifts []tg.StarGiftClass
	gifts = resRaw.GetGifts()

	prices := make([]int, 0, len(gifts))

	for _, g := range gifts {
		uniqueGift, ok := g.(*tg.StarGiftUnique)
		if !ok {
			continue
		}
		resellOptions, ok := uniqueGift.GetResellAmount()
		if !ok || len(resellOptions) == 0 {
			continue
		}

		switch amount := resellOptions[0].(type) {
		case *tg.StarsAmount:
			prices = append(prices, int(amount.Amount))
		}
	}

	return prices, nil
}

// GetGiftTypes - получение всех типов gifts
func (c *Client) GetGiftTypes(ctx context.Context, hash int) ([]entity.GiftType, error) {
	resRaw, err := c.api.PaymentsGetStarGifts(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch gift types: %w", err)
	}

	var giftsInterfaces []tg.StarGiftClass
	switch res := resRaw.(type) {
	case *tg.PaymentsStarGifts:
		giftsInterfaces = res.Gifts
	case *tg.PaymentsStarGiftsNotModified:
		return []entity.GiftType{}, nil
	default:
		return nil, fmt.Errorf("unexpected response type: %T", resRaw)
	}

	result := make([]entity.GiftType, 0, len(giftsInterfaces))

	for _, giftRaw := range giftsInterfaces {
		gift, ok := giftRaw.(*tg.StarGift)
		if !ok {
			continue
		}

		totalSupply := 0
		remainingSupply := -1
		if total, ok := gift.GetAvailabilityTotal(); ok {
			totalSupply = total
		}

		if remains, ok := gift.GetAvailabilityRemains(); ok {
			remainingSupply = remains
		}

		item := entity.GiftType{
			ID:               gift.ID,
			Name:             gift.Title,
			StorePrice:       gift.Stars,
			TotalSupply:      totalSupply,
			RemainingSupply:  remainingSupply,
			MarketFloorPrice: 0,
			AveragePrice:     0,
			MarketQuantity:   0,
			UpdatedAt:        time.Now(),
		}

		result = append(result, item)
	}

	return result, nil
}

func (c *Client) GetMarketDeals(ctx context.Context, giftTypeID int64, limit int) ([]entity.Deal, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	req := &tg.PaymentsGetResaleStarGiftsRequest{
		GiftID:      giftTypeID,
		Limit:       limit,
		SortByPrice: true,
	}

	resRaw, err := c.api.PaymentsGetResaleStarGifts(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("tg api call failed: %w", err)
	}

	rawGifts := resRaw.GetGifts()
	rawUsers := resRaw.GetUsers()

	// Map для быстрого поиска accessHash
	users := make(map[int64]int64, len(rawUsers))
	for _, u := range rawUsers {
		if user, ok := u.(*tg.User); ok {
			users[user.ID] = user.AccessHash
		}
	}

	deals := make([]entity.Deal, 0, len(rawGifts))

	for _, g := range rawGifts {
		u, ok := g.(*tg.StarGiftUnique)
		if !ok {
			continue
		}

		var starPrice int64
		var tonPrice float64
		if opts, ok := u.GetResellAmount(); ok && len(opts) >= 2 {
			starPrice = opts[0].GetAmount()
			tonPrice = float64(opts[1].GetAmount()) / 1_000_000_000
		}

		if starPrice <= 0 {
			continue
		}

		var ownerID int64
		if ownerPeer, ok := u.GetOwnerID(); ok {
			if p, ok := ownerPeer.(*tg.PeerUser); ok {
				ownerID = p.UserID
			}
		}

		// --- НАЧАЛО: Парсинг атрибутов ---
		var attrs value.GiftAttributes
		var totalRarity int

		// Проходимся по всем атрибутам подарка
		for _, attr := range u.GetAttributes() {
			switch a := attr.(type) {
			case *tg.StarGiftAttributeModel:
				attrs.Model = a.Name
				totalRarity += a.RarityPermille
			case *tg.StarGiftAttributePattern:
				attrs.Pattern = a.Name
				totalRarity += a.RarityPermille
			case *tg.StarGiftAttributeBackdrop:
				attrs.Backdrop = a.Name
				totalRarity += a.RarityPermille
			}
		}
		attrs.RarityPerMille = totalRarity

		slug := u.GetSlug()
		link := fmt.Sprintf("https://t.me/nft/%s-%d", slug, u.Num)

		deals = append(deals, entity.Deal{
			Gift: &entity.Gift{
				ID:         u.ID,
				StarPrice:  starPrice,
				TonPrice:   tonPrice,
				Num:        u.Num,
				NumRating:  0,
				Slug:       slug,
				OwnerID:    ownerID,
				TypeID:     giftTypeID,
				Address:    link,
				Attributes: attrs, // <--- Вставляем распаршенные атрибуты
			},
			SellerAccessHash: users[ownerID],
		})
	}

	return deals, nil
}

// BuyDeal - покупает сделку с маркета
func (c *Client) BuyDeal(ctx context.Context, deal entity.Deal) error {
	gift := deal.Gift
	// giftType := deal.GiftType // (Для логов, если нужно)

	logger(ctx).Info("⚡️ BUYING DEAL START",
		"slug", gift.Slug,
		"num", gift.Num,
		"ton_price", gift.TonPrice,
	)

	// 1. Формируем InputPeer владельца
	ownerPeer := &tg.InputPeerUser{
		UserID:     gift.OwnerID,
		AccessHash: deal.SellerAccessHash,
	}

	peers := []struct {
		name string
		peer tg.InputPeerClass
	}{
		{"Self", &tg.InputPeerSelf{}}, // Приоритет (сработало в тесте)
		{"Owner", ownerPeer},          // Резерв
	}

	// Варианты слага
	rawSlug := gift.Slug
	slugs := []string{
		fmt.Sprintf("%s-%d", rawSlug, gift.Num), // Double Num: PreciousPeach-1561-1561 (если raw уже с номером)
		rawSlug,                                 // Single
		fmt.Sprintf("nft/%s", rawSlug),
	}

	// Если в gift.Slug нет номера (редко, но бывает), то нужно добавить его
	// Но обычно парсер возвращает уже Slug-Num.

	// 3. Перебор (Brute Force)
	for _, p := range peers {
		for _, s := range slugs {
			invoice := &tg.InputInvoiceStarGiftResale{
				Ton:  true,
				Slug: s,
				ToID: p.peer,
			}

			// Запрос формы
			formRaw, err := c.api.PaymentsGetPaymentForm(ctx, &tg.PaymentsGetPaymentFormRequest{
				Invoice: invoice,
			})

			if err != nil {
				// Логируем только для дебага, чтобы не спамить
				// logger(ctx).Debug("try failed", "slug", s, "peer", p.name, "err", err)
				continue
			}

			logger(ctx).Info("✅ FORM RECEIVED", "slug", s, "peer", p.name)

			// 4. Оплата
			return c.processPayment(ctx, formRaw, invoice)
		}
	}

	return fmt.Errorf("failed to buy: all slug/peer combinations failed")
}

// processPayment обрабатывает форму и шлет деньги
func (c *Client) processPayment(ctx context.Context, formRaw tg.PaymentsPaymentFormClass, invoice tg.InputInvoiceClass) error {
	formID, err := c.extractFormID(formRaw)
	if err != nil {
		return err
	}

	logger(ctx).Info("🚀 SENDING PAYMENT...", "form_id", formID)

	result, err := c.api.PaymentsSendStarsForm(ctx, &tg.PaymentsSendStarsFormRequest{
		FormID:  formID,
		Invoice: invoice,
	})
	if err != nil {
		return fmt.Errorf("send payment failed: %w", err)
	}

	// Проверяем результат
	switch r := result.(type) {
	case *tg.PaymentsPaymentResult:
		logger(ctx).Info("🏆 PAYMENT SUCCESS!")
		return nil
	case *tg.PaymentsPaymentVerificationNeeded:
		return fmt.Errorf("verification needed: %s", r.URL)
	default:
		return fmt.Errorf("unknown payment result: %T", result)
	}
}

// extractFormID извлекает FormID (с поддержкой нового типа)
func (c *Client) extractFormID(formRaw tg.PaymentsPaymentFormClass) (int64, error) {
	switch f := formRaw.(type) {
	case *tg.PaymentsPaymentForm:
		return f.FormID, nil
	case *tg.PaymentsPaymentFormStars:
		return f.FormID, nil
	case *tg.PaymentsPaymentFormStarGift: // <--- ВАЖНО: Добавлен новый тип
		return f.FormID, nil
	default:
		return 0, fmt.Errorf("unknown payment form type: %T", formRaw)
	}
}

// GetGiftsPage - Функция получает ОДНУ страницу подарков
func (c *Client) GetGiftsPage(ctx context.Context, giftID int64, offset string, limit int) ([]entity.Gift, string, error) {
	if limit <= 0 {
		limit = 50
	}

	req := &tg.PaymentsGetResaleStarGiftsRequest{
		GiftID:      giftID,
		Limit:       limit,
		SortByPrice: true,
		Offset:      offset,
	}

	resRaw, err := c.api.PaymentsGetResaleStarGifts(ctx, req)
	if err != nil {
		return nil, "", fmt.Errorf("ошибка API: %w", err)
	}

	rawGifts := resRaw.GetGifts()
	nextOffset := resRaw.NextOffset

	gifts := make([]entity.Gift, 0, len(rawGifts))

	for _, g := range rawGifts {
		u, ok := g.(*tg.StarGiftUnique)
		if !ok {
			continue
		}

		var starPrice int64
		var tonPrice float64
		if opts, ok := u.GetResellAmount(); ok && len(opts) >= 2 {
			starPrice = opts[0].GetAmount()
			tonPrice = float64(opts[1].GetAmount()) / 1_000_000_000
		}

		var ownerID int64
		if ownerPeer, ok := u.GetOwnerID(); ok {
			if p, ok := ownerPeer.(*tg.PeerUser); ok {
				ownerID = p.UserID
			}
		}

		slug := u.GetSlug()
		link := fmt.Sprintf("https://t.me/nft/%s-%d", slug, u.Num)

		gifts = append(gifts, entity.Gift{
			ID:        u.ID,
			StarPrice: starPrice,
			TonPrice:  tonPrice,
			Num:       u.Num,
			NumRating: 0, // Will be set by the service when processing ratings
			Slug:      slug,
			OwnerID:   ownerID,
			TypeID:    giftID,
			Address:   link,
			UpdatedAt: time.Now(),
		})
	}

	return gifts, nextOffset, nil
}

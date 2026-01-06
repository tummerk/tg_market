package telegram

import (
	"context"
	"fmt"
	"github.com/gotd/td/tg"
	"tg_market/internal/domain/entity"
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

	for _, gRaw := range giftsInterfaces {
		g, ok := gRaw.(*tg.StarGift)
		if !ok {
			continue
		}

		var stickerID int64
		if doc, ok := g.Sticker.(*tg.Document); ok {
			stickerID = doc.ID
		}

		totalSupply := 0
		remainingSupply := -1
		if total, ok := g.GetAvailabilityTotal(); ok {
			totalSupply = total
		}

		if remains, ok := g.GetAvailabilityRemains(); ok {
			remainingSupply = remains
		}

		item := entity.GiftType{
			ID:               g.ID,
			Name:             g.Title,
			StickerID:        stickerID,
			StorePrice:       g.Stars,
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

func (c *Client) GetMarketGifts(ctx context.Context, giftTypeID int64, limit int) ([]entity.Gift, error) {
	// 1. Обязательный таймаут, чтобы не зависнуть
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// 2. Формируем запрос
	req := &tg.PaymentsGetResaleStarGiftsRequest{
		GiftID:      giftTypeID,
		Limit:       limit,
		SortByPrice: true,
	}

	resRaw, err := c.api.PaymentsGetResaleStarGifts(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("tg api call failed: %w", err)
	}

	// 4. Парсим результат
	var result []entity.Gift
	rawGifts := resRaw.GetGifts()

	for _, g := range rawGifts {
		u, ok := g.(*tg.StarGiftUnique)
		if !ok {
			continue
		}

		// Достаем цену (логика как в твоем GetLastPrices)
		var price int64
		opts, ok := u.GetResellAmount()
		if ok && len(opts) > 0 {
			switch v := opts[0].(type) {
			case *tg.StarsAmount:
				price = v.Amount
			}
		}

		// Если цены нет или она 0 — пропускаем
		if price <= 0 {
			continue
		}

		// Достаем ID владельца (полезно знать, кто продает)
		var ownerID int64
		if ownerPeer, ok := u.GetOwnerID(); ok {
			switch p := ownerPeer.(type) {
			case *tg.PeerUser:
				ownerID = p.UserID
			}
		}

		var link string
		slug := u.GetSlug()
		link = fmt.Sprintf("https://t.me/nft/%s-%d", slug, u.Num)

		result = append(result, entity.Gift{
			ID:      u.ID,
			Price:   price,
			Num:     u.Num,
			OwnerID: ownerID,
			TypeID:  giftTypeID,
			Address: link,
		})
	}
	return result, nil
}

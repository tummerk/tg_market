package persistence

import (
	"encoding/json"
	"tg_market/internal/domain/entity"
	"tg_market/internal/domain/value"
	"time"
)

// giftSchema — внутренняя структура для маппинга строки БД.
type giftSchema struct {
	ID         int64     `db:"id"`
	TypeID     int64     `db:"type_id"`
	Address    string    `db:"address"`
	Num        int       `db:"num"`
	OwnerID    int64     `db:"owner_id"`
	Price      int64     `db:"price"`
	Attributes []byte    `db:"attributes"`
	UpdatedAt  time.Time `db:"updated_at"`
}

func (s *giftSchema) toDomain() (*entity.Gift, error) {
	attrs, err := s.parseAttributes()
	if err != nil {
		return nil, err
	}

	return &entity.Gift{
		ID:         s.ID,
		TypeID:     s.TypeID,
		Address:    s.Address,
		Num:        s.Num,
		OwnerID:    s.OwnerID,
		Price:      s.Price,
		Attributes: attrs,
		UpdatedAt:  s.UpdatedAt,
	}, nil
}

func (s *giftSchema) parseAttributes() (value.GiftAttributes, error) {
	var attrs value.GiftAttributes
	if len(s.Attributes) > 0 {
		if err := json.Unmarshal(s.Attributes, &attrs); err != nil {
			return attrs, err
		}
	}
	return attrs, nil
}

// GiftTypeSchema — представление таблицы gift_types в БД.
type GiftTypeSchema struct {
	ID               int64     `db:"id"`
	Name             string    `db:"name"`
	StickerID        int64     `db:"sticker_id"`
	StorePrice       int64     `db:"store_price"`
	TotalSupply      int       `db:"total_supply"`
	RemainingSupply  int       `db:"remaining_supply"`
	MarketFloorPrice int64     `db:"market_floor_price"`
	AveragePrice     int64     `db:"average_price"`
	PriceUpdatedAt   time.Time `db:"price_updated_at"`
	MarketQuantity   int       `db:"market_quantity"`
	UpdatedAt        time.Time `db:"updated_at"`
}

func FromGiftType(e *entity.GiftType) *GiftTypeSchema {
	return &GiftTypeSchema{
		ID:               e.ID,
		Name:             e.Name,
		StickerID:        e.StickerID,
		StorePrice:       e.StorePrice,
		TotalSupply:      e.TotalSupply,
		RemainingSupply:  e.RemainingSupply,
		MarketFloorPrice: e.MarketFloorPrice,
		AveragePrice:     e.AveragePrice,
		PriceUpdatedAt:   e.PriceUpdatedAt,
		MarketQuantity:   e.MarketQuantity,
		UpdatedAt:        e.UpdatedAt,
	}
}

func (s *GiftTypeSchema) ToDomain() *entity.GiftType {
	return &entity.GiftType{
		ID:               s.ID,
		Name:             s.Name,
		StickerID:        s.StickerID,
		StorePrice:       s.StorePrice,
		TotalSupply:      s.TotalSupply,
		RemainingSupply:  s.RemainingSupply,
		MarketFloorPrice: s.MarketFloorPrice,
		AveragePrice:     s.AveragePrice,
		PriceUpdatedAt:   s.PriceUpdatedAt,
		MarketQuantity:   s.MarketQuantity,
		UpdatedAt:        s.UpdatedAt,
	}
}

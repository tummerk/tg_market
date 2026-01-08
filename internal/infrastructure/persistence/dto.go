package persistence

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"tg_market/internal/domain/entity"
	"tg_market/internal/domain/value"
	"time"
)

// giftSchema — внутренняя структура для маппинга строки БД.
type giftSchema struct {
	ID         int64           `db:"id"`
	TypeID     int64           `db:"type_id"`
	Num        int             `db:"num"`
	NumRating  int             `db:"numRating"`
	OwnerID    int64           `db:"owner_id"`
	Address    sql.NullString  `db:"address"`    // Может быть NULL
	StarPrice  sql.NullInt64   `db:"star_price"` // Может быть NULL (не продается)
	TonPrice   sql.NullFloat64 `db:"ton_price"`  // Может быть NULL
	Attributes []byte          `db:"attributes"` // JSONB
	UpdatedAt  time.Time       `db:"updated_at"`
}

// toDomain конвертирует строку БД в доменную сущность
func (s *giftSchema) toDomain() (*entity.Gift, error) {
	var attrs value.GiftAttributes
	if len(s.Attributes) > 0 {
		if err := json.Unmarshal(s.Attributes, &attrs); err != nil {
			return nil, fmt.Errorf("unmarshal attrs: %w", err)
		}
	}

	gift := &entity.Gift{
		ID:         s.ID,
		TypeID:     s.TypeID,
		Num:        s.Num,
		NumRating:  s.NumRating,
		OwnerID:    s.OwnerID,
		Address:    s.Address.String,
		Attributes: attrs,
		UpdatedAt:  s.UpdatedAt,
	}

	if s.StarPrice.Valid {
		gift.StarPrice = s.StarPrice.Int64
	}
	if s.TonPrice.Valid {
		gift.TonPrice = s.TonPrice.Float64
	}

	return gift, nil
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
	Slug             string    `db:"slug"`
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
		Slug:             e.Slug,
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
		Slug:             s.Slug,
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

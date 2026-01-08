package entity

import "time"

type GiftType struct {
	ID               int64     `json:"id"`
	Name             string    `json:"name"`
	Slug             string    `json:"slug"`
	StorePrice       int64     `json:"store_price"`
	TotalSupply      int       `json:"total_supply"`
	RemainingSupply  int       `json:"remaining_supply"`
	MarketFloorPrice int64     `json:"floor_price"`
	AveragePrice     int64     `json:"average_price"`
	PriceUpdatedAt   time.Time `json:"price_updated_at"`
	MarketQuantity   int       `json:"market_quantity"`
	UpdatedAt        time.Time `json:"updated_at"`
}

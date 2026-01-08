package entity

import (
	"tg_market/internal/domain/value"
	"time"
)

type Gift struct {
	ID         int64                `json:"id" db:"id"`
	TypeID     int64                `json:"type_id" db:"type_id"`
	Num        int                  `json:"num" db:"num"`
	NumRating  int                  `json:"num_rating" db:"numRating"`
	Address    string               `json:"address" db:"address"`
	Slug       string               `json:"slug" db:"slug"`
	OwnerID    int64                `json:"owner_id" db:"owner_id"`
	StarPrice  int64                `json:"star_price,omitempty" db:"star_price"`
	TonPrice   float64              `json:"ton_price,omitempty" db:"ton_price"`
	Attributes value.GiftAttributes `json:"attributes" db:"attributes"`
	UpdatedAt  time.Time            `json:"updated_at" db:"updated_at"`
}

package entity

import (
	"tg_market/internal/domain/value"
	"time"
)

type Gift struct {
	ID         int64                `json:"id" db:"id"`
	TypeID     int64                `json:"type_id" db:"type_id"`
	Num        int                  `json:"num" db:"num"`
	Address    string               `json:"address" db:"address"`
	OwnerID    int64                `json:"owner_id" db:"owner_id"`
	Price      int64                `json:"price,omitempty" db:"price"` // sqlx может записать *int64, но читать лучше аккуратно
	Attributes value.GiftAttributes `json:"attributes" db:"attributes"`
	UpdatedAt  time.Time            `json:"updated_at" db:"updated_at"`
}

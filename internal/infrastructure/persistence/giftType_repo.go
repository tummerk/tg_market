package persistence

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"

	"tg_market/internal/domain"
	"tg_market/internal/domain/entity"
	"tg_market/pkg/errcodes"
)

type GiftTypeRepository struct {
	db *sqlx.DB
}

func NewGiftTypeRepository(db *sqlx.DB) *GiftTypeRepository {
	return &GiftTypeRepository{db: db}
}

func (r *GiftTypeRepository) withTx(ctx context.Context, fn func(tx *sqlx.Tx) error) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return domain.WrapError(err, errcodes.InternalServerError, "failed to begin transaction")
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return domain.WrapError(err, errcodes.InternalServerError, "failed to commit")
	}
	return nil
}

// Create — добавляем поле average_price в INSERT
func (r *GiftTypeRepository) Create(ctx context.Context, gift *entity.GiftType) error {
	return r.withTx(ctx, func(tx *sqlx.Tx) error {
		schema := FromGiftType(gift)
		if schema.UpdatedAt.IsZero() {
			schema.UpdatedAt = time.Now()
		}

		query := `
			INSERT INTO gift_types (
				id, name, sticker_id, store_price, total_supply, 
				remaining_supply, market_floor_price, average_price, 
				market_quantity, updated_at
			) VALUES (
				:id, :name, :sticker_id, :store_price, :total_supply, 
				:remaining_supply, :market_floor_price, :average_price, 
				:market_quantity, :updated_at
			)
			ON CONFLICT (id) DO NOTHING` // Можно добавить ON CONFLICT для идемпотентности

		_, err := tx.NamedExecContext(ctx, query, schema)
		if err != nil {
			return domain.WrapError(err, errcodes.InternalServerError, "failed to create gift type")
		}
		return nil
	})
}

// GetByID — без изменений (структура Schema сама подтянет поле)
func (r *GiftTypeRepository) GetByID(ctx context.Context, id int64) (*entity.GiftType, error) {
	query := `SELECT * FROM gift_types WHERE id = $1`
	var schema GiftTypeSchema
	if err := r.db.GetContext(ctx, &schema, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewError(errcodes.GiftNotFound, "gift type not found")
		}
		return nil, domain.WrapError(err, errcodes.InternalServerError, "failed to get gift type")
	}
	return schema.ToDomain(), nil
}

// Update — Полное обновление (включая average_price)
func (r *GiftTypeRepository) Update(ctx context.Context, gift *entity.GiftType) error {
	return r.withTx(ctx, func(tx *sqlx.Tx) error {
		schema := FromGiftType(gift)
		schema.UpdatedAt = time.Now()

		query := `
			UPDATE gift_types SET
				name = :name,
				sticker_id = :sticker_id,
				store_price = :store_price,
				total_supply = :total_supply,
				remaining_supply = :remaining_supply,
				market_floor_price = :market_floor_price,
				average_price = :average_price,
				market_quantity = :market_quantity,
				updated_at = :updated_at
			WHERE id = :id`

		res, err := tx.NamedExecContext(ctx, query, schema)
		if err != nil {
			return domain.WrapError(err, errcodes.InternalServerError, "failed to update gift type")
		}

		rows, _ := res.RowsAffected()
		if rows == 0 {
			return domain.NewError(errcodes.GiftNotFound, "gift type not found")
		}
		return nil
	})
}

// UpdateStats — обновление ТОЛЬКО рыночных данных (эффективно для воркера)
func (r *GiftTypeRepository) UpdateStats(ctx context.Context, id int64, floorPrice, avgPrice int64, quantity int) error {
	return r.withTx(ctx, func(tx *sqlx.Tx) error {
		query := `
			UPDATE gift_types 
			SET market_floor_price = $1,
			    average_price = $2,
			    market_quantity = $3,
			    updated_at = $4
			WHERE id = $5`

		res, err := tx.ExecContext(ctx, query, floorPrice, avgPrice, quantity, time.Now(), id)
		if err != nil {
			return domain.WrapError(err, errcodes.InternalServerError, "failed to update stats")
		}

		rows, _ := res.RowsAffected()
		if rows == 0 {
			return domain.NewError(errcodes.GiftNotFound, "gift type not found for stats update")
		}
		return nil
	})
}

// DecreaseSupply — уменьшение остатка
func (r *GiftTypeRepository) DecreaseSupply(ctx context.Context, id int64) error {
	return r.withTx(ctx, func(tx *sqlx.Tx) error {
		query := `
			UPDATE gift_types 
			SET remaining_supply = remaining_supply - 1,
			    updated_at = $1
			WHERE id = $2 AND remaining_supply > 0`

		res, err := tx.ExecContext(ctx, query, time.Now(), id)
		if err != nil {
			return domain.WrapError(err, errcodes.InternalServerError, "failed to decrease supply")
		}

		rows, _ := res.RowsAffected()
		if rows == 0 {
			// Проверяем существование, чтобы отдать точную ошибку
			var exists bool
			_ = tx.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM gift_types WHERE id = $1)`, id)
			if !exists {
				return domain.NewError(errcodes.GiftNotFound, "gift type not found")
			}
			return domain.NewError(errcodes.GiftOutOfStock, "gift out of stock")
		}
		return nil
	})
}

// List — получение списка
func (r *GiftTypeRepository) List(ctx context.Context, limit, offset int) ([]entity.GiftType, error) {
	query := `SELECT * FROM gift_types ORDER BY id ASC LIMIT $1 OFFSET $2`

	var schemas []GiftTypeSchema
	if err := r.db.SelectContext(ctx, &schemas, query, limit, offset); err != nil {
		return nil, domain.WrapError(err, errcodes.InternalServerError, "failed to list gift types")
	}

	result := make([]entity.GiftType, 0, len(schemas))
	for _, s := range schemas {
		result = append(result, *s.ToDomain())
	}
	return result, nil
}

func (r *GiftTypeRepository) UpdatePriceStats(ctx context.Context, id int64, avgPrice int64) error {
	query := `
		UPDATE gift_types 
		SET average_price = $1, price_updated_at = $2 
		WHERE id = $3`

	res, err := r.db.ExecContext(ctx, query, avgPrice, time.Now(), id)
	if err != nil {
		return domain.WrapError(err, errcodes.InternalServerError, "failed to update price stats")
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return domain.WrapError(err, errcodes.InternalServerError, "failed to check rows")
	}

	if rows == 0 {
		return domain.NewError(errcodes.GiftNotFound, "gift type not found")
	}

	return nil
}

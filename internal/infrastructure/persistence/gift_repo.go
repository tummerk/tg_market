package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"

	"tg_market/internal/domain"
	"tg_market/internal/domain/entity"
	"tg_market/pkg/errcodes"
)

type GiftItemRepository struct {
	db *sqlx.DB
}

func NewGiftRepository(db *sqlx.DB) *GiftItemRepository {
	return &GiftItemRepository{db: db}
}

func (r *GiftItemRepository) withTx(ctx context.Context, fn func(tx *sqlx.Tx) error) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return domain.WrapError(err, errcodes.InternalServerError, "begin tx failed")
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
		return domain.WrapError(err, errcodes.InternalServerError, "commit failed")
	}
	return nil
}

// -----------------------------------------------------------------------------
// Создание
// -----------------------------------------------------------------------------

func (r *GiftItemRepository) Create(ctx context.Context, gift *entity.Gift) error {
	return r.withTx(ctx, func(tx *sqlx.Tx) error {
		return r.createTx(ctx, tx, gift)
	})
}

// UpsertBatch сохраняет пачку подарков (или обновляет цены, если существуют)
func (r *GiftItemRepository) UpsertBatch(ctx context.Context, gifts []*entity.Gift) error {
	if len(gifts) == 0 {
		return nil
	}

	return r.withTx(ctx, func(tx *sqlx.Tx) error {
		query := `
			INSERT INTO gifts (id, type_id, num, numRating, owner_id, star_price, ton_price, address, attributes, updated_at)
			VALUES (:id, :type_id, :num, :num_rating, :owner_id, :star_price, :ton_price, :address, :attributes, :updated_at)
			ON CONFLICT (id) DO UPDATE SET
				num = EXCLUDED.num,
				numRating = EXCLUDED.numRating,
				owner_id   = EXCLUDED.owner_id,
				star_price = EXCLUDED.star_price,
				ton_price  = EXCLUDED.ton_price,
				updated_at = EXCLUDED.updated_at;
		`

		payloads := make([]map[string]any, 0, len(gifts))

		for _, gift := range gifts {
			attrsBytes, _ := json.Marshal(gift.Attributes)
			if gift.UpdatedAt.IsZero() {
				gift.UpdatedAt = time.Now()
			}

			var starPrice any = nil
			if gift.StarPrice > 0 {
				starPrice = gift.StarPrice
			}
			var tonPrice any = nil
			if gift.TonPrice > 0 {
				tonPrice = gift.TonPrice
			}

			payloads = append(payloads, map[string]any{
				"id":         gift.ID,
				"type_id":    gift.TypeID,
				"num":        gift.Num,
				"num_rating": gift.NumRating,
				"owner_id":   gift.OwnerID,
				"star_price": starPrice,
				"ton_price":  tonPrice,
				"address":    gift.Address,
				"attributes": attrsBytes,
				"updated_at": gift.UpdatedAt,
			})
		}

		if _, err := tx.NamedExecContext(ctx, query, payloads); err != nil {
			return domain.WrapError(err, errcodes.InternalServerError, "upsert batch failed")
		}

		return nil
	})
}

// внутренний insert
func (r *GiftItemRepository) createTx(ctx context.Context, tx *sqlx.Tx, gift *entity.Gift) error {
	attrsBytes, err := json.Marshal(gift.Attributes)
	if err != nil {
		return domain.WrapError(err, errcodes.InternalServerError, "marshal failed")
	}

	if gift.UpdatedAt.IsZero() {
		gift.UpdatedAt = time.Now()
	}

	var starPrice any = nil
	if gift.StarPrice > 0 {
		starPrice = gift.StarPrice
	}
	var tonPrice any = nil
	if gift.TonPrice > 0 {
		tonPrice = gift.TonPrice
	}

	query := `
		INSERT INTO gifts (id, type_id, num, numRating, owner_id, star_price, ton_price, address, attributes, updated_at)
		VALUES (:id, :type_id, :num, :num_rating, :owner_id, :star_price, :ton_price, :address, :attributes, :updated_at)`

	params := map[string]any{
		"id":         gift.ID,
		"type_id":    gift.TypeID,
		"num":        gift.Num,
		"num_rating": gift.NumRating,
		"owner_id":   gift.OwnerID,
		"star_price": starPrice,
		"ton_price":  tonPrice,
		"address":    gift.Address,
		"attributes": attrsBytes,
		"updated_at": gift.UpdatedAt,
	}

	if _, err := tx.NamedExecContext(ctx, query, params); err != nil {
		return domain.WrapError(err, errcodes.InternalServerError, "insert failed")
	}

	return nil
}

// -----------------------------------------------------------------------------
// Чтение
// -----------------------------------------------------------------------------

func (r *GiftItemRepository) GetByID(ctx context.Context, id int64) (*entity.Gift, error) {
	query := `SELECT * FROM gifts WHERE id = $1`

	var schema giftSchema
	if err := r.db.GetContext(ctx, &schema, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewError(errcodes.GiftNotFound, "gift not found")
		}
		return nil, domain.WrapError(err, errcodes.InternalServerError, "db error")
	}

	return schema.toDomain()
}

func (r *GiftItemRepository) GetByIDs(ctx context.Context, ids []int64) ([]*entity.Gift, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query, args, err := sqlx.In(`
		SELECT * 
		FROM gifts 
		WHERE id IN (?)`, ids)
	if err != nil {
		return nil, domain.WrapError(err, errcodes.InternalServerError, "failed to build query")
	}

	query = r.db.Rebind(query)

	var schemas []giftSchema
	if err := r.db.SelectContext(ctx, &schemas, query, args...); err != nil {
		return nil, domain.WrapError(err, errcodes.InternalServerError, "failed to select gifts")
	}

	gifts := make([]*entity.Gift, 0, len(schemas))
	for _, s := range schemas {
		gift, err := s.toDomain()
		if err != nil {
			return nil, domain.WrapError(err, errcodes.InternalServerError, "data corruption")
		}
		gifts = append(gifts, gift)
	}

	return gifts, nil
}

// -----------------------------------------------------------------------------
// UpdateOwner / UpdatePrice / TransferGift
// -----------------------------------------------------------------------------

// UpdateOwner передаёт подарок новому владельцу и снимает с продажи.
func (r *GiftItemRepository) UpdateOwner(ctx context.Context, giftID, newOwnerID int64) error {
	return r.withTx(ctx, func(tx *sqlx.Tx) error {
		query := `
			UPDATE gifts 
			SET owner_id = $1,
			    star_price = NULL,
			    ton_price  = NULL,
			    updated_at = $2
			WHERE id = $3`

		return r.execUpdateTx(ctx, tx, query, newOwnerID, time.Now(), giftID)
	})
}

// UpdatePrice — обновляет только star_price (ton_price оставляем как есть).
func (r *GiftItemRepository) UpdatePrice(ctx context.Context, giftID int64, price *int64) error {
	return r.withTx(ctx, func(tx *sqlx.Tx) error {
		query := `
			UPDATE gifts 
			SET star_price = $1,
			    updated_at = $2
			WHERE id = $3`

		return r.execUpdateTx(ctx, tx, query, price, time.Now(), giftID)
	})
}

// TransferGift атомарно передаёт подарок с проверкой владельца и снимает с продажи.
func (r *GiftItemRepository) TransferGift(ctx context.Context, giftID, fromUserID, toUserID int64) error {
	return r.withTx(ctx, func(tx *sqlx.Tx) error {
		// Лочим строку
		query := `
			SELECT * 
			FROM gifts 
			WHERE id = $1
			FOR UPDATE`

		var schema giftSchema
		if err := tx.GetContext(ctx, &schema, query, giftID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.NewError(errcodes.GiftNotFound, "gift not found")
			}
			return domain.WrapError(err, errcodes.InternalServerError, "failed to lock gift")
		}

		if schema.OwnerID != fromUserID {
			return domain.NewError(errcodes.Forbidden, "you don't own this gift")
		}

		updateQuery := `
			UPDATE gifts 
			SET owner_id   = $1,
			    star_price = NULL,
			    ton_price  = NULL,
			    updated_at = $2
			WHERE id = $3`

		return r.execUpdateTx(ctx, tx, updateQuery, toUserID, time.Now(), giftID)
	})
}

// -----------------------------------------------------------------------------
// Exists / execUpdateTx
// -----------------------------------------------------------------------------

func (r *GiftItemRepository) Exists(ctx context.Context, id int64) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM gifts WHERE id = $1)`
	var exists bool
	if err := r.db.GetContext(ctx, &exists, query, id); err != nil {
		return false, domain.WrapError(err, errcodes.InternalServerError, "failed to check gift existence")
	}
	return exists, nil
}

func (r *GiftItemRepository) execUpdateTx(ctx context.Context, tx *sqlx.Tx, query string, args ...any) error {
	res, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return domain.WrapError(err, errcodes.InternalServerError, "failed to execute update")
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return domain.WrapError(err, errcodes.InternalServerError, "failed to check rows")
	}

	if rows == 0 {
		return domain.NewError(errcodes.GiftNotFound, "gift not found")
	}

	return nil
}

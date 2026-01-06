package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"tg_market/internal/domain"
	"tg_market/internal/domain/entity"
	"tg_market/pkg/errcodes"
	"time"

	"github.com/jmoiron/sqlx"
)

type GiftItemRepository struct {
	db *sqlx.DB
}

// NewGiftRepository создаёт новый экземпляр репозитория.
func NewGiftRepository(db *sqlx.DB) *GiftItemRepository {
	return &GiftItemRepository{db: db}
}

// withTx выполняет функцию в транзакции.
func (r *GiftItemRepository) withTx(ctx context.Context, fn func(tx *sqlx.Tx) error) error {
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
		if rbErr := tx.Rollback(); rbErr != nil {
			return domain.WrapError(
				fmt.Errorf("%w; rollback: %v", err, rbErr),
				errcodes.InternalServerError,
				"transaction failed",
			)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return domain.WrapError(err, errcodes.InternalServerError, "failed to commit")
	}

	return nil
}

// Create сохраняет новый подарок.
func (r *GiftItemRepository) Create(ctx context.Context, gift *entity.Gift) error {
	return r.withTx(ctx, func(tx *sqlx.Tx) error {
		return r.createTx(ctx, tx, gift)
	})
}

// CreateBatch сохраняет массив подарков атомарно.
func (r *GiftItemRepository) CreateBatch(ctx context.Context, gifts []*entity.Gift) error {
	if len(gifts) == 0 {
		return nil
	}

	return r.withTx(ctx, func(tx *sqlx.Tx) error {
		for i, gift := range gifts {
			if err := r.createTx(ctx, tx, gift); err != nil {
				return domain.WrapError(err, errcodes.InternalServerError,
					fmt.Sprintf("failed at index %d", i))
			}
		}
		return nil
	})
}

// GetByID возвращает подарок по идентификатору.
func (r *GiftItemRepository) GetByID(ctx context.Context, id int64) (*entity.Gift, error) {
	query := `
		SELECT id, type_id, num, owner_id, price, attributes, updated_at 
		FROM gifts 
		WHERE id = $1`

	var schema giftSchema
	if err := r.db.GetContext(ctx, &schema, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewError(errcodes.GiftNotFound, "gift not found")
		}
		return nil, domain.WrapError(err, errcodes.InternalServerError, "failed to get gift")
	}

	return schema.toDomain()
}

// GetByIDs возвращает подарки по списку идентификаторов.
func (r *GiftItemRepository) GetByIDs(ctx context.Context, ids []int64) ([]*entity.Gift, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query, args, err := sqlx.In(`
		SELECT id, type_id, num, owner_id, price, attributes, updated_at 
		FROM gifts 
		WHERE id IN (?)`, ids)
	if err != nil {
		return nil, domain.WrapError(err, errcodes.InternalServerError, "failed to build query")
	}

	var schemas []giftSchema
	if err := r.db.SelectContext(ctx, &schemas, r.db.Rebind(query), args...); err != nil {
		return nil, domain.WrapError(err, errcodes.InternalServerError, "failed to get gifts")
	}

	gifts := make([]*entity.Gift, 0, len(schemas))
	for _, s := range schemas {
		gift, err := s.toDomain()
		if err != nil {
			return nil, domain.WrapError(err, errcodes.InternalServerError, "failed to convert gift")
		}
		gifts = append(gifts, gift)
	}

	return gifts, nil
}

// UpdateOwner передаёт подарок новому владельцу.
func (r *GiftItemRepository) UpdateOwner(ctx context.Context, giftID, newOwnerID int64) error {
	return r.withTx(ctx, func(tx *sqlx.Tx) error {
		query := `
			UPDATE gifts 
			SET owner_id = $1, updated_at = $2, price = NULL
			WHERE id = $3`

		return r.execUpdateTx(ctx, tx, query, newOwnerID, time.Now(), giftID)
	})
}

// UpdatePrice выставляет или снимает подарок с продажи.
func (r *GiftItemRepository) UpdatePrice(ctx context.Context, giftID int64, price *int64) error {
	return r.withTx(ctx, func(tx *sqlx.Tx) error {
		query := `
			UPDATE gifts 
			SET price = $1, updated_at = $2 
			WHERE id = $3`

		return r.execUpdateTx(ctx, tx, query, price, time.Now(), giftID)
	})
}

// TransferGift атомарно передаёт подарок с проверкой владельца.
func (r *GiftItemRepository) TransferGift(ctx context.Context, giftID, fromUserID, toUserID int64) error {
	return r.withTx(ctx, func(tx *sqlx.Tx) error {
		// Блокируем строку и проверяем владельца
		query := `
			SELECT id, type_id, num, owner_id, price, attributes, updated_at 
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

		// Обновляем владельца
		updateQuery := `
			UPDATE gifts 
			SET owner_id = $1, updated_at = $2, price = NULL
			WHERE id = $3`

		return r.execUpdateTx(ctx, tx, updateQuery, toUserID, time.Now(), giftID)
	})
}

// createTx — внутренний метод вставки в рамках транзакции.
func (r *GiftItemRepository) createTx(ctx context.Context, tx *sqlx.Tx, gift *entity.Gift) error {
	attrsBytes, err := json.Marshal(gift.Attributes)
	if err != nil {
		return domain.WrapError(err, errcodes.InternalServerError, "failed to marshal attributes")
	}

	updatedAt := gift.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}

	query := `
		INSERT INTO gifts (id, type_id, num, owner_id, price, attributes, updated_at, address)
		VALUES (:id, :type_id, :num, :owner_id, :price, :attributes, :updated_at, :address)`

	params := map[string]any{
		"id":         gift.ID,
		"type_id":    gift.TypeID,
		"num":        gift.Num,
		"owner_id":   gift.OwnerID,
		"price":      gift.Price,
		"attributes": attrsBytes,
		"updated_at": updatedAt,
		"address":    gift.Address,
	}

	if _, err := tx.NamedExecContext(ctx, query, params); err != nil {
		return domain.WrapError(err, errcodes.InternalServerError, "failed to insert gift")
	}

	return nil
}

// execUpdateTx — внутренний метод обновления в рамках транзакции.
func (r *GiftItemRepository) execUpdateTx(ctx context.Context, tx *sqlx.Tx, query string, args ...any) error {
	res, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return domain.WrapError(err, errcodes.InternalServerError, "failed to execute update")
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return domain.WrapError(err, errcodes.InternalServerError, "failed to check affected rows")
	}

	if rows == 0 {
		return domain.NewError(errcodes.GiftNotFound, "gift not found")
	}

	return nil
}

func (r *GiftItemRepository) Exists(ctx context.Context, id int64) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM gifts WHERE id = $1)`

	var exists bool
	if err := r.db.GetContext(ctx, &exists, query, id); err != nil {
		return false, domain.WrapError(err, errcodes.InternalServerError, "failed to check gift existence")
	}

	return exists, nil
}

-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS gift_types (
                                          id BIGINT PRIMARY KEY,
                                          name VARCHAR(255),
                                          slug   VARCHAR(255),
                                          store_price BIGINT NOT NULL DEFAULT 0,  -- Цена в звездах
                                          total_supply INT NOT NULL DEFAULT 0,    -- Общий тираж
                                          remaining_supply INT NOT NULL DEFAULT 0,-- Остаток в магазине
                                          market_floor_price BIGINT DEFAULT 0,    -- Минимальная цена сейчас
                                          market_quantity INT DEFAULT 0,          -- Количество лотов на продаже
                                          average_price   INT,
                                          price_updated_at TIMESTAMP,
                                          updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP

);

-- Здесь хранятся уникальные NFT (Unique/UserStarGift)
CREATE TABLE IF NOT EXISTS gifts (
                                     id BIGINT PRIMARY KEY,
                                     type_id BIGINT NOT NULL REFERENCES gift_types(id) ON DELETE CASCADE,
                                     address VARCHAR,
                                     num INT NOT NULL,
                                     numRating INT,
                                     owner_id BIGINT,
                                     star_price BIGINT DEFAULT NULL,
                                     ton_price BIGINT DEFAULT NULL,
                                     attributes JSONB DEFAULT '{}',
                                     updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS gifts;
DROP TABLE IF EXISTS gift_types;
-- +goose StatementEnd
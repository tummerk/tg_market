package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"tg_market/internal/config"
	service "tg_market/internal/domain/service/gift"
	"tg_market/internal/domain/service/numRating"
	"tg_market/internal/infrastructure/persistence"
	"tg_market/internal/infrastructure/telegram"
	"tg_market/pkg/application/connectors"
)

//1 go run cmd/findGoodNum/main.go <gift_type_id> [min_rating_percent]
//
//Например:
//
//1 go run cmd/findGoodNum/main.go 123456789 80

func main() {
	// Создаем контекст с возможностью отмены
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Настройка обработки сигналов для корректного завершения
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		cancel()
	}()

	// Настройка логгера
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug, // Установим уровень DEBUG для просмотра хороших номеров
	}))

	if err := run(ctx, log, cancel); err != nil {
		log.Error("application error", "error", err)
		os.Exit(1)
	}

	log.Info("application finished")
}

func run(ctx context.Context, log *slog.Logger, cancel context.CancelFunc) error {
	// 1. Загрузка конфигурации
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config load: %w", err)
	}

	// 2. Подключение к базе данных
	pg := &connectors.Postgres{
		DSN:             cfg.Postgres.DSN,
		MaxOpenConns:    cfg.Postgres.MaxOpenConns,
		MaxIdleConns:    cfg.Postgres.MaxIdleConns,
		ConnMaxLifetime: cfg.Postgres.ConnMaxLifetime,
	}
	db := pg.Client(ctx)
	defer pg.Close(ctx)

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("db ping: %w", err)
	}
	log.Info("database connection OK")

	// 3. Репозитории
	giftTypeRepo := persistence.NewGiftTypeRepository(db)
	giftRepo := persistence.NewGiftRepository(db)

	// 4. Telegram Pool
	accounts, err := telegram.LoadAccounts("accounts.json")
	if err != nil {
		return fmt.Errorf("load accounts: %w", err)
	}
	log.Info("loaded accounts", "count", len(accounts))

	pool, err := telegram.NewPool(cfg.Telegram, accounts)
	if err != nil {
		return fmt.Errorf("create pool: %w", err)
	}

	go func() {
		log.Info("starting telegram pool...")
		if err := pool.Start(ctx); err != nil && ctx.Err() == nil {
			log.Error("telegram pool stopped", "error", err)
			cancel()
		}
	}()

	if err := pool.WaitReady(ctx); err != nil {
		return fmt.Errorf("wait pool ready: %w", err)
	}
	log.Info("✅ Telegram Pool Ready", "clients", pool.Size())

	// Создаем сервис подарков
	svc := service.NewGiftService(giftTypeRepo, giftRepo, pool)

	// Получаем минимальный рейтинг, если он указан
	minRatingPercent := 75.0 // значение по умолчанию
	if len(os.Args) >= 2 {
		_, err = fmt.Sscanf(os.Args[1], "%f", &minRatingPercent)
		if err != nil {
			log.Error("invalid min rating percent format", "error", err)
			return fmt.Errorf("invalid min rating percent format: %w", err)
		}
	}

	types := []int64{5773668482394620318}
	for _, t := range types {
		processedCount, err := svc.ProcessGiftsByRating(ctx, t, minRatingPercent)
		if err != nil {
			return fmt.Errorf("process gifts by rating: %w", err)
		}
		log.Info("completed processing gifts by rating",
			"gift_type_id", t,
			"min_rating_percent", minRatingPercent,
			"processed_count", processedCount)
	}

	testNum := 777
	rating := numRating.CalculateValue(testNum)
	log.Info("test rating calculation",
		"number", testNum,
		"rating", rating.Score,
		"description", rating.Description)

	return nil
}

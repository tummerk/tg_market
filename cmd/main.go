package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"tg_market/internal/infrastructure/notifier"
	"time"

	"tg_market/internal/config"
	service "tg_market/internal/domain/service/gift"
	"tg_market/internal/infrastructure/persistence"
	"tg_market/internal/infrastructure/telegram"
	"tg_market/internal/worker"
	"tg_market/pkg/application/connectors"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(log)

	if err := run(ctx, log, cancel); err != nil {
		log.Error("application failed", "error", err)
		os.Exit(1)
	}

	log.Info("application stopped")
}

func run(ctx context.Context, log *slog.Logger, cancel context.CancelFunc) error {
	// 1. Config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config load: %w", err)
	}

	// 2. Database
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

	// 3. Repositories
	giftTypeRepo := persistence.NewGiftTypeRepository(db)
	giftRepo := persistence.NewGiftRepository(db)

	// 4. Telegram MTProto Client
	tgClient, err := telegram.NewClient(cfg.Telegram)
	if err != nil {
		return fmt.Errorf("tg client create: %w", err)
	}

	tgReady := make(chan struct{})
	go func() {
		log.Info("starting telegram client...")
		err := tgClient.Start(ctx, func() error {
			log.Info("✅ Telegram Authorized & Ready")
			close(tgReady)
			return nil
		})
		if err != nil {
			log.Error("telegram client stopped", "error", err)
			cancel()
		}
	}()

	select {
	case <-tgReady:
	case <-ctx.Done():
		return ctx.Err()
	}

	dealsCh := make(chan service.GoodDeal, 100)

	const (
		MyChatID = 1217838677
	)
	alertBot, err := notifier.NewTelegramBot(cfg.Bot.Token, MyChatID)
	if err != nil {
		return fmt.Errorf("notifier bot: %w", err)
	}
	log.Info("Testing bot notification...")
	if err := alertBot.SendText(ctx, "🚀 Bot is starting! Test message."); err != nil {
		log.Error("❌ Bot test failed! Check Token and ChatID", "err", err)
		// Можно вернуть ошибку, если бот критичен
		// return err
	} else {
		log.Info("✅ Bot test passed! Message sent.")
	}
	go func() {
		log.Info("notifier bot started listening")
		if err := alertBot.Run(ctx, dealsCh); err != nil {
			if ctx.Err() == nil {
				log.Error("notifier bot stopped", "error", err)
			}
		}
	}()

	svc := service.NewGiftService(giftTypeRepo, giftRepo, tgClient).
		WithDiscountThreshold(10)

	targetTypes := []int64{
		6003767644426076664,
		6003373314888696650,
		6014591077976114307,
	}

	scanner := worker.NewMarketScanner(svc, giftTypeRepo, dealsCh, 5*time.Second).
		WithGiftTypes(targetTypes...)

	go func() {
		defer close(dealsCh)

		if err := scanner.Run(ctx); err != nil {
			if ctx.Err() == nil {
				log.Error("scanner died", "err", err)
				cancel()
			}
		}
	}()

	log.Info("scanner started", "targets", targetTypes)

	<-ctx.Done()

	log.Info("application stopping...")
	return nil
}
